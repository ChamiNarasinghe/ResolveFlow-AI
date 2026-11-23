package users

import (
	"context"
	"errors"
	"fmt"

	"github.com/chami2002/resolveflow-api/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrEmailTaken is returned by Create when the email address is already registered.
var ErrEmailTaken = errors.New("email address is already registered")

// ErrAdminExists is returned by BootstrapAdmin when an administrator already exists
// or when a concurrent bootstrap attempt is detected.
var ErrAdminExists = errors.New("administrator account already exists")

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new agent account (self-registration). must_change_password = FALSE.
func (r *UserRepository) Create(
	ctx context.Context,
	fullName string,
	email string,
	passwordHash string,
) (*models.User, error) {
	return r.createWithRole(ctx, fullName, email, passwordHash, "agent", false)
}

// CreateWithRole creates a new user with an explicit role. must_change_password = FALSE.
// Used by Bootstrap path only — provisioned users go through CreateProvisioned.
func (r *UserRepository) CreateWithRole(
	ctx context.Context,
	fullName string,
	email string,
	passwordHash string,
	role string,
) (*models.User, error) {
	return r.createWithRole(ctx, fullName, email, passwordHash, role, false)
}

// CreateProvisioned creates a new user with an explicit role and sets
// must_change_password = TRUE. Used exclusively by the admin provisioning path.
func (r *UserRepository) CreateProvisioned(
	ctx context.Context,
	fullName string,
	email string,
	passwordHash string,
	role string,
) (*models.User, error) {
	return r.createWithRole(ctx, fullName, email, passwordHash, role, true)
}

func (r *UserRepository) createWithRole(
	ctx context.Context,
	fullName, email, passwordHash, role string,
	mustChangePassword bool,
) (*models.User, error) {
	query := `
		INSERT INTO users (full_name, email, password_hash, role, must_change_password)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, full_name, email, role, is_active, must_change_password, created_at, updated_at
	`

	var user models.User

	err := r.db.QueryRow(
		ctx,
		query,
		fullName,
		email,
		passwordHash,
		role,
		mustChangePassword,
	).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.Role,
		&user.IsActive,
		&user.MustChangePassword,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrEmailTaken
		}

		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

// CountAdmins returns the number of users with the admin role.
func (r *UserRepository) CountAdmins(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role = 'admin'`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count admins: %w", err)
	}
	return count, nil
}

// bootstrapLockID is a stable application-level advisory lock key used
// exclusively by BootstrapAdmin to serialise concurrent bootstrap attempts.
// The value is arbitrary but must be unique within the application.
const bootstrapLockID int64 = 7583920461

// BootstrapAdmin creates the first administrator account inside a single
// transaction protected by a PostgreSQL transaction-scoped advisory lock.
//
// The lock (pg_try_advisory_xact_lock) is acquired at the start of the
// transaction and released automatically when the transaction ends, whether
// by commit or rollback. If a concurrent caller already holds the lock the
// function returns ErrAdminExists so the caller retries or skips gracefully.
//
// The sequence inside the transaction is:
//  1. Acquire the advisory lock — blocks concurrent bootstrap attempts.
//  2. COUNT admins — if > 0, roll back and return ErrAdminExists.
//  3. INSERT the new admin row.
//  4. Commit.
//
// passwordHash must already be a bcrypt hash; plaintext is never accepted.
// must_change_password is always FALSE for the bootstrap admin.
func (r *UserRepository) BootstrapAdmin(
	ctx context.Context,
	fullName, email, passwordHash string,
) (*models.User, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Acquire a transaction-scoped exclusive advisory lock.
	// pg_try_advisory_xact_lock returns false immediately if another session
	// holds the lock, avoiding a deadlock or indefinite wait.
	var locked bool
	if err := tx.QueryRow(ctx,
		`SELECT pg_try_advisory_xact_lock($1)`, bootstrapLockID,
	).Scan(&locked); err != nil {
		return nil, fmt.Errorf("bootstrap: acquire lock: %w", err)
	}
	if !locked {
		// Another instance is bootstrapping right now; treat as already exists.
		return nil, ErrAdminExists
	}

	// Count existing admins while holding the lock.
	var count int
	if err := tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM users WHERE role = 'admin'`,
	).Scan(&count); err != nil {
		return nil, fmt.Errorf("bootstrap: count admins: %w", err)
	}
	if count > 0 {
		return nil, ErrAdminExists
	}

	// Insert the admin row. must_change_password = FALSE (operator-configured account).
	const q = `
		INSERT INTO users (full_name, email, password_hash, role, must_change_password)
		VALUES ($1, $2, $3, 'admin', FALSE)
		RETURNING id, full_name, email, role, is_active, must_change_password, created_at, updated_at
	`
	var user models.User
	if err := tx.QueryRow(ctx, q, fullName, email, passwordHash).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.Role,
		&user.IsActive,
		&user.MustChangePassword,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("bootstrap: insert admin: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("bootstrap: commit: %w", err)
	}

	return &user, nil
}

// GetByEmail retrieves a user and their password hash for authentication.
func (r *UserRepository) GetByEmail(
	ctx context.Context,
	email string,
) (*models.User, string, error) {
	query := `
		SELECT id, full_name, email, password_hash,
		       role, is_active, must_change_password, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user models.User
	var passwordHash string

	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&passwordHash,
		&user.Role,
		&user.IsActive,
		&user.MustChangePassword,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", pgx.ErrNoRows
		}

		return nil, "", fmt.Errorf("failed to retrieve user: %w", err)
	}

	return &user, passwordHash, nil
}

// GetByID retrieves a user using their ID.
func (r *UserRepository) GetByID(
	ctx context.Context,
	userID string,
) (*models.User, error) {
	query := `
		SELECT id, full_name, email, role, is_active, must_change_password, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user models.User

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.Role,
		&user.IsActive,
		&user.MustChangePassword,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}

		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	return &user, nil
}

// GetPasswordHash retrieves only the stored bcrypt hash for a user.
// The hash is never included in the User model and is only used for
// current-password verification during a password change.
func (r *UserRepository) GetPasswordHash(
	ctx context.Context,
	userID string,
) (string, error) {
	var hash string
	err := r.db.QueryRow(ctx,
		`SELECT password_hash FROM users WHERE id = $1`,
		userID,
	).Scan(&hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", pgx.ErrNoRows
		}
		return "", fmt.Errorf("failed to retrieve password hash: %w", err)
	}
	return hash, nil
}

// UpdatePassword atomically replaces the password hash and clears
// must_change_password for the given user.
func (r *UserRepository) UpdatePassword(
	ctx context.Context,
	userID string,
	newPasswordHash string,
) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE users
		SET password_hash        = $2,
		    must_change_password = FALSE,
		    updated_at           = NOW()
		WHERE id = $1
	`, userID, newPasswordHash)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

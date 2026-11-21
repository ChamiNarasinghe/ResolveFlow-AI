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

// Create creates a new user with the agent role and returns the public user details.
func (r *UserRepository) Create(
	ctx context.Context,
	fullName string,
	email string,
	passwordHash string,
) (*models.User, error) {
	return r.createWithRole(ctx, fullName, email, passwordHash, "agent")
}

// CreateWithRole creates a new user with an explicit role and returns the public user details.
func (r *UserRepository) CreateWithRole(
	ctx context.Context,
	fullName string,
	email string,
	passwordHash string,
	role string,
) (*models.User, error) {
	return r.createWithRole(ctx, fullName, email, passwordHash, role)
}

func (r *UserRepository) createWithRole(
	ctx context.Context,
	fullName, email, passwordHash, role string,
) (*models.User, error) {
	query := `
		INSERT INTO users (full_name, email, password_hash, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, full_name, email, role, is_active, created_at, updated_at
	`

	var user models.User

	err := r.db.QueryRow(
		ctx,
		query,
		fullName,
		email,
		passwordHash,
		role,
	).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.Role,
		&user.IsActive,
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

	// Insert the admin row.
	const q = `
		INSERT INTO users (full_name, email, password_hash, role)
		VALUES ($1, $2, $3, 'admin')
		RETURNING id, full_name, email, role, is_active, created_at, updated_at
	`
	var user models.User
	if err := tx.QueryRow(ctx, q, fullName, email, passwordHash).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.Role,
		&user.IsActive,
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
		       role, is_active, created_at, updated_at
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
		SELECT id, full_name, email, role, is_active, created_at, updated_at
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
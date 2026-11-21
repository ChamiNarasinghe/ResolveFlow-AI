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

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user and returns the public user details.
func (r *UserRepository) Create(
	ctx context.Context,
	fullName string,
	email string,
	passwordHash string,
) (*models.User, error) {
	query := `
		INSERT INTO users (full_name, email, password_hash, role)
		VALUES ($1, $2, $3, 'agent')
		RETURNING id, full_name, email, role, is_active, created_at, updated_at
	`

	var user models.User

	err := r.db.QueryRow(
		ctx,
		query,
		fullName,
		email,
		passwordHash,
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
			return nil, errors.New("email address is already registered")
		}

		return nil, fmt.Errorf("failed to create user: %w", err)
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
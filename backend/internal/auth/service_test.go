package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chami2002/resolveflow-api/internal/models"
	"github.com/jackc/pgx/v5"
)

type mockAuthUserRepository struct {
	user         *models.User
	passwordHash string
	createErr    error
	getErr       error
}

func (m *mockAuthUserRepository) Create(
	ctx context.Context,
	fullName, email, passwordHash string,
) (*models.User, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}

	m.passwordHash = passwordHash

	m.user = &models.User{
		ID:        "user-123",
		FullName:  fullName,
		Email:     email,
		Role:      "agent",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return m.user, nil
}

func (m *mockAuthUserRepository) GetByEmail(
	ctx context.Context,
	email string,
) (*models.User, string, error) {
	if m.getErr != nil {
		return nil, "", m.getErr
	}

	return m.user, m.passwordHash, nil
}

func (m *mockAuthUserRepository) GetByID(
	ctx context.Context,
	userID string,
) (*models.User, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}

	if m.user == nil || m.user.ID != userID {
		return nil, pgx.ErrNoRows
	}

	return m.user, nil
}

func newTestAuthService(t *testing.T, repo *mockAuthUserRepository) *AuthService {
	t.Helper()

	jwtService, err := NewJWTService(
		"this-is-a-test-secret-with-more-than-32-characters",
		15,
	)
	if err != nil {
		t.Fatalf("NewJWTService() error = %v", err)
	}

	return NewAuthService(repo, jwtService)
}

func TestAuthServiceRegister(t *testing.T) {
	repo := &mockAuthUserRepository{}
	service := newTestAuthService(t, repo)

	response, err := service.Register(
		context.Background(),
		RegisterRequest{
			FullName: "Test Agent",
			Email:    "agent@example.com",
			Password: "Password123",
		},
	)

	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if response.User.Role != "agent" {
		t.Fatalf("expected agent role, got %s", response.User.Role)
	}

	if response.AccessToken == "" {
		t.Fatal("expected access token")
	}

	if repo.passwordHash == "Password123" || repo.passwordHash == "" {
		t.Fatal("password was not securely hashed")
	}
}

func TestAuthServiceLogin(t *testing.T) {
	passwordHash, err := HashPassword("Password123")
	if err != nil {
		t.Fatal(err)
	}

	repo := &mockAuthUserRepository{
		user: &models.User{
			ID:       "user-123",
			FullName: "Test Agent",
			Email:    "agent@example.com",
			Role:     "agent",
			IsActive: true,
		},
		passwordHash: passwordHash,
	}

	service := newTestAuthService(t, repo)

	response, err := service.Login(
		context.Background(),
		LoginRequest{
			Email:    "agent@example.com",
			Password: "Password123",
		},
	)

	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if response.AccessToken == "" {
		t.Fatal("expected access token")
	}
}

func TestAuthServiceLoginInvalidPassword(t *testing.T) {
	passwordHash, _ := HashPassword("Password123")

	repo := &mockAuthUserRepository{
		user: &models.User{
			ID:       "user-123",
			Email:    "agent@example.com",
			Role:     "agent",
			IsActive: true,
		},
		passwordHash: passwordHash,
	}

	service := newTestAuthService(t, repo)

	_, err := service.Login(
		context.Background(),
		LoginRequest{
			Email:    "agent@example.com",
			Password: "WrongPassword123",
		},
	)

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials error, got %v", err)
	}
}

func TestAuthServiceLoginUserNotFound(t *testing.T) {
	repo := &mockAuthUserRepository{
		getErr: pgx.ErrNoRows,
	}

	service := newTestAuthService(t, repo)

	_, err := service.Login(
		context.Background(),
		LoginRequest{
			Email:    "unknown@example.com",
			Password: "Password123",
		},
	)

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials error, got %v", err)
	}
}

func TestAuthServiceLoginInactiveUser(t *testing.T) {
	passwordHash, _ := HashPassword("Password123")

	repo := &mockAuthUserRepository{
		user: &models.User{
			ID:       "user-123",
			Email:    "agent@example.com",
			Role:     "agent",
			IsActive: false,
		},
		passwordHash: passwordHash,
	}

	service := newTestAuthService(t, repo)

	_, err := service.Login(
		context.Background(),
		LoginRequest{
			Email:    "agent@example.com",
			Password: "Password123",
		},
	)

	if !errors.Is(err, ErrInactiveUser) {
		t.Fatalf("expected inactive user error, got %v", err)
	}
}

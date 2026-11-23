package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chami2002/resolveflow-api/internal/models"
	"github.com/chami2002/resolveflow-api/internal/users"
	"github.com/jackc/pgx/v5"
)

type mockAuthUserRepository struct {
	user         *models.User
	passwordHash string
	createErr    error
	getErr       error
	adminCount   int
	countErr     error
	createdRole  string
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

func (m *mockAuthUserRepository) CreateWithRole(
	ctx context.Context,
	fullName, email, passwordHash, role string,
) (*models.User, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}

	m.passwordHash = passwordHash
	m.createdRole = role

	m.user = &models.User{
		ID:        "user-456",
		FullName:  fullName,
		Email:     email,
		Role:      role,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return m.user, nil
}

func (m *mockAuthUserRepository) CountAdmins(ctx context.Context) (int, error) {
	return m.adminCount, m.countErr
}

func (m *mockAuthUserRepository) BootstrapAdmin(
	ctx context.Context,
	fullName, email, passwordHash string,
) (*models.User, error) {
	// Simulate the atomic lock+count+insert behaviour.
	if m.countErr != nil {
		return nil, m.countErr
	}
	if m.adminCount > 0 {
		return nil, users.ErrAdminExists
	}
	if m.createErr != nil {
		return nil, m.createErr
	}
	m.passwordHash = passwordHash
	m.createdRole = "admin"
	m.user = &models.User{
		ID:        "boot-admin-id",
		FullName:  fullName,
		Email:     email,
		Role:      "admin",
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

func (m *mockAuthUserRepository) CreateProvisioned(
	ctx context.Context,
	fullName, email, passwordHash, role string,
) (*models.User, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	m.passwordHash = passwordHash
	m.createdRole = role
	m.user = &models.User{
		ID:                 "user-456",
		FullName:           fullName,
		Email:              email,
		Role:               role,
		IsActive:           true,
		MustChangePassword: true,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	return m.user, nil
}

func (m *mockAuthUserRepository) GetPasswordHash(ctx context.Context, userID string) (string, error) {
	if m.getErr != nil {
		return "", m.getErr
	}
	return m.passwordHash, nil
}

func (m *mockAuthUserRepository) UpdatePassword(ctx context.Context, userID, newPasswordHash string) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.passwordHash = newPasswordHash
	return nil
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

func TestAuthServiceRegisterDuplicateEmail(t *testing.T) {
	repo := &mockAuthUserRepository{
		createErr: users.ErrEmailTaken,
	}
	service := newTestAuthService(t, repo)

	_, err := service.Register(
		context.Background(),
		RegisterRequest{
			FullName: "Test Agent",
			Email:    "agent@example.com",
			Password: "Password123",
		},
	)

	if !errors.Is(err, ErrEmailAlreadyRegistered) {
		t.Fatalf("expected ErrEmailAlreadyRegistered, got %v", err)
	}
}

func TestAuthServiceProvisionUserAgent(t *testing.T) {
	repo := &mockAuthUserRepository{}
	service := newTestAuthService(t, repo)

	user, err := service.ProvisionUser(context.Background(), ProvisionRequest{
		FullName: "Alice Agent",
		Email:    "alice@example.com",
		Password: "Password123",
		Role:     "agent",
	})

	if err != nil {
		t.Fatalf("ProvisionUser() error = %v", err)
	}
	if user.Role != "agent" {
		t.Errorf("expected agent role, got %s", user.Role)
	}
	if !user.MustChangePassword {
		t.Error("expected MustChangePassword = true for provisioned agent")
	}
	if repo.passwordHash == "Password123" || repo.passwordHash == "" {
		t.Error("password was not securely hashed")
	}
}

func TestAuthServiceProvisionUserAdmin(t *testing.T) {
	repo := &mockAuthUserRepository{}
	service := newTestAuthService(t, repo)

	user, err := service.ProvisionUser(context.Background(), ProvisionRequest{
		FullName: "Bob Admin",
		Email:    "bob@example.com",
		Password: "Password123",
		Role:     "admin",
	})

	if err != nil {
		t.Fatalf("ProvisionUser() error = %v", err)
	}
	if user.Role != "admin" {
		t.Errorf("expected admin role, got %s", user.Role)
	}
	if !user.MustChangePassword {
		t.Error("expected MustChangePassword = true for provisioned admin")
	}
}

func TestAuthServiceProvisionUserInvalidRole(t *testing.T) {
	repo := &mockAuthUserRepository{}
	service := newTestAuthService(t, repo)

	_, err := service.ProvisionUser(context.Background(), ProvisionRequest{
		FullName: "Eve",
		Email:    "eve@example.com",
		Password: "Password123",
		Role:     "superuser",
	})

	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestAuthServiceProvisionUserDuplicateEmail(t *testing.T) {
	repo := &mockAuthUserRepository{createErr: users.ErrEmailTaken}
	service := newTestAuthService(t, repo)

	_, err := service.ProvisionUser(context.Background(), ProvisionRequest{
		FullName: "Alice",
		Email:    "alice@example.com",
		Password: "Password123",
		Role:     "agent",
	})

	if !errors.Is(err, ErrEmailAlreadyRegistered) {
		t.Fatalf("expected ErrEmailAlreadyRegistered, got %v", err)
	}
}

func TestAuthServiceProvisionUserPasswordIsHashed(t *testing.T) {
	repo := &mockAuthUserRepository{}
	service := newTestAuthService(t, repo)

	_, err := service.ProvisionUser(context.Background(), ProvisionRequest{
		FullName: "Alice",
		Email:    "alice@example.com",
		Password: "Password123",
		Role:     "agent",
	})
	if err != nil {
		t.Fatal(err)
	}

	if repo.passwordHash == "Password123" {
		t.Error("plaintext password must not be stored")
	}
	if err := CheckPassword("Password123", repo.passwordHash); err != nil {
		t.Error("stored hash does not match original password")
	}
}

func TestAuthServiceBootstrapCreatesFirstAdmin(t *testing.T) {
	repo := &mockAuthUserRepository{adminCount: 0}
	service := newTestAuthService(t, repo)

	user, err := service.Bootstrap(context.Background(), "Root Admin", "root@example.com", "Password123")
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	if user.Role != "admin" {
		t.Errorf("expected admin role, got %s", user.Role)
	}
}

func TestAuthServiceBootstrapRefusesSecondAdmin(t *testing.T) {
	repo := &mockAuthUserRepository{adminCount: 1}
	service := newTestAuthService(t, repo)

	_, err := service.Bootstrap(context.Background(), "Root Admin", "root@example.com", "Password123")
	if !errors.Is(err, ErrAdminExists) {
		t.Fatalf("expected ErrAdminExists, got %v", err)
	}
}

func TestAuthServiceBootstrapRefusesOverwriteExistingAdmin(t *testing.T) {
	// adminCount > 0 means at least one admin exists — bootstrap must refuse.
	repo := &mockAuthUserRepository{adminCount: 2}
	service := newTestAuthService(t, repo)

	_, err := service.Bootstrap(context.Background(), "Another Admin", "another@example.com", "Password123")
	if !errors.Is(err, ErrAdminExists) {
		t.Fatalf("expected ErrAdminExists, got %v", err)
	}
}

func TestAuthServiceBootstrapDisabledWhenEnvEmpty(t *testing.T) {
	repo := &mockAuthUserRepository{}
	service := newTestAuthService(t, repo)

	_, err := service.Bootstrap(context.Background(), "", "", "")
	if !errors.Is(err, ErrBootstrapDisabled) {
		t.Fatalf("expected ErrBootstrapDisabled, got %v", err)
	}
}

func TestAuthServiceRegisterAlwaysAssignsAgentRole(t *testing.T) {
	// Public registration must never produce an admin, regardless of input.
	// Register() calls Create() which hard-codes 'agent' in the repository.
	repo := &mockAuthUserRepository{}
	service := newTestAuthService(t, repo)

	resp, err := service.Register(context.Background(), RegisterRequest{
		FullName: "Hacker",
		Email:    "hacker@example.com",
		Password: "Password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.User.Role != "agent" {
		t.Errorf("public registration must produce agent role, got %s", resp.User.Role)
	}
}

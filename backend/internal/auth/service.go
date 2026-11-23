package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/chami2002/resolveflow-api/internal/models"
	"github.com/chami2002/resolveflow-api/internal/users"
	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidCredentials     = errors.New("invalid email or password")
	ErrInactiveUser           = errors.New("user account is inactive")
	ErrUserNotFound           = errors.New("user not found")
	ErrEmailAlreadyRegistered = errors.New("email address is already registered")
)

var (
	ErrInvalidRole       = errors.New("invalid role")
	ErrAdminExists       = errors.New("an administrator account already exists")
	ErrBootstrapDisabled = errors.New("bootstrap is not configured")
)

// ErrIncorrectPassword is returned when the supplied current password does not
// match the stored hash during a password-change request.
var ErrIncorrectPassword = errors.New("current password is incorrect")

type AuthUserRepository interface {
	Create(
		ctx context.Context,
		fullName string,
		email string,
		passwordHash string,
	) (*models.User, error)

	CreateWithRole(
		ctx context.Context,
		fullName string,
		email string,
		passwordHash string,
		role string,
	) (*models.User, error)

	// CreateProvisioned creates a user with must_change_password = TRUE.
	// Used exclusively by the admin provisioning path.
	CreateProvisioned(
		ctx context.Context,
		fullName string,
		email string,
		passwordHash string,
		role string,
	) (*models.User, error)

	CountAdmins(ctx context.Context) (int, error)

	// BootstrapAdmin atomically acquires a database-level lock, verifies no
	// admin exists, and inserts the first administrator in one transaction.
	BootstrapAdmin(
		ctx context.Context,
		fullName string,
		email string,
		passwordHash string,
	) (*models.User, error)

	GetByEmail(
		ctx context.Context,
		email string,
	) (*models.User, string, error)

	GetByID(
		ctx context.Context,
		userID string,
	) (*models.User, error)

	// GetPasswordHash retrieves only the bcrypt hash for a user.
	// Never exposed through the User model.
	GetPasswordHash(ctx context.Context, userID string) (string, error)

	// UpdatePassword atomically replaces the hash and clears must_change_password.
	UpdatePassword(ctx context.Context, userID string, newPasswordHash string) error
}

type AuthResponse struct {
	User        *models.User `json:"user"`
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
}

type AuthService struct {
	users      AuthUserRepository
	jwtService *JWTService
}

func NewAuthService(
	users AuthUserRepository,
	jwtService *JWTService,
) *AuthService {
	return &AuthService{
		users:      users,
		jwtService: jwtService,
	}
}

// Register creates a new agent account (development mode only).
// must_change_password is FALSE — self-registered users choose their own password.
func (s *AuthService) Register(
	ctx context.Context,
	request RegisterRequest,
) (*AuthResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	passwordHash, err := HashPassword(request.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user, err := s.users.Create(
		ctx,
		request.FullName,
		request.Email,
		passwordHash,
	)
	if err != nil {
		if errors.Is(err, users.ErrEmailTaken) {
			return nil, ErrEmailAlreadyRegistered
		}
		return nil, err
	}

	token, err := s.jwtService.GenerateToken(
		user.ID,
		user.Email,
		user.Role,
		user.MustChangePassword,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	return &AuthResponse{
		User:        user,
		AccessToken: token,
		TokenType:   "Bearer",
	}, nil
}

// Login authenticates an existing user.
// The MustChangePassword flag is read from the database and embedded in the JWT.
func (s *AuthService) Login(
	ctx context.Context,
	request LoginRequest,
) (*AuthResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	user, passwordHash, err := s.users.GetByEmail(
		ctx,
		request.Email,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}

		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	if !user.IsActive {
		return nil, ErrInactiveUser
	}

	if err := CheckPassword(request.Password, passwordHash); err != nil {
		return nil, ErrInvalidCredentials
	}

	switch user.Role {
	case "admin", "manager", "agent":
	default:
		return nil, errors.New("user has an invalid role")
	}

	token, err := s.jwtService.GenerateToken(
		user.ID,
		user.Email,
		user.Role,
		user.MustChangePassword,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	return &AuthResponse{
		User:        user,
		AccessToken: token,
		TokenType:   "Bearer",
	}, nil
}

// ProvisionUser creates a new user with an explicit role and sets
// must_change_password = TRUE so the user is forced to change their
// administrator-supplied initial password on first login.
func (s *AuthService) ProvisionUser(
	ctx context.Context,
	request ProvisionRequest,
) (*models.User, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	passwordHash, err := HashPassword(request.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user, err := s.users.CreateProvisioned(
		ctx,
		request.FullName,
		request.Email,
		passwordHash,
		request.Role,
	)
	if err != nil {
		if errors.Is(err, users.ErrEmailTaken) {
			return nil, ErrEmailAlreadyRegistered
		}
		return nil, err
	}

	return user, nil
}

// Bootstrap creates the very first administrator account.
// It returns ErrAdminExists if any admin already exists or a concurrent
// bootstrap attempt is in progress.
// It returns ErrBootstrapDisabled if name/email/password are all empty.
// The atomic lock-count-insert is performed by the repository layer.
// must_change_password is FALSE — the bootstrap admin is operator-configured.
func (s *AuthService) Bootstrap(
	ctx context.Context,
	fullName, email, password string,
) (*models.User, error) {
	if fullName == "" && email == "" && password == "" {
		return nil, ErrBootstrapDisabled
	}

	req := ProvisionRequest{FullName: fullName, Email: email, Password: password, Role: "admin"}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	passwordHash, err := HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user, err := s.users.BootstrapAdmin(ctx, fullName, email, passwordHash)
	if err != nil {
		if errors.Is(err, users.ErrAdminExists) {
			return nil, ErrAdminExists
		}
		if errors.Is(err, users.ErrEmailTaken) {
			return nil, ErrEmailAlreadyRegistered
		}
		return nil, fmt.Errorf("bootstrap: %w", err)
	}

	return user, nil
}

// GetProfile retrieves the authenticated user's profile.
func (s *AuthService) GetProfile(
	ctx context.Context,
	userID string,
) (*models.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}

		return nil, fmt.Errorf("failed to retrieve user profile: %w", err)
	}

	if !user.IsActive {
		return nil, ErrInactiveUser
	}

	return user, nil
}

// ChangePassword verifies the current password, validates the new password,
// updates the hash atomically, and returns a fresh JWT with
// MustChangePassword = false.
func (s *AuthService) ChangePassword(
	ctx context.Context,
	userID string,
	request ChangePasswordRequest,
) (*AuthResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	currentHash, err := s.users.GetPasswordHash(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to retrieve password: %w", err)
	}

	if err := CheckPassword(request.CurrentPassword, currentHash); err != nil {
		return nil, ErrIncorrectPassword
	}

	newHash, err := HashPassword(request.NewPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	if err := s.users.UpdatePassword(ctx, userID, newHash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to update password: %w", err)
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve updated user: %w", err)
	}

	token, err := s.jwtService.GenerateToken(
		user.ID,
		user.Email,
		user.Role,
		false, // must_change_password is now cleared
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	return &AuthResponse{
		User:        user,
		AccessToken: token,
		TokenType:   "Bearer",
	}, nil
}

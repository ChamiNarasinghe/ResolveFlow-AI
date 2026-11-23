package auth

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"unicode"
)

// ErrValidation is the sentinel wrapped by all RegisterRequest.Validate errors.
// Handlers can use errors.Is(err, ErrValidation) or IsValidationError(err)
// to distinguish safe user-facing messages from internal failures.
var ErrValidation = errors.New("validation error")

// IsValidationError reports whether err originated from input validation
// (i.e. it is safe to return the message verbatim to the client).
func IsValidationError(err error) bool {
	return errors.Is(err, ErrValidation)
}

type RegisterRequest struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *RegisterRequest) Validate() error {
	r.FullName = strings.TrimSpace(r.FullName)
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))

	if r.FullName == "" {
		return fmt.Errorf("%w: full name is required", ErrValidation)
	}

	if len(r.FullName) > 150 {
		return fmt.Errorf("%w: full name must not exceed 150 characters", ErrValidation)
	}

	address, err := mail.ParseAddress(r.Email)
	if err != nil || address.Address != r.Email {
		return fmt.Errorf("%w: valid email address is required", ErrValidation)
	}

	if len(r.Password) < 8 {
		return fmt.Errorf("%w: password must contain at least 8 characters", ErrValidation)
	}

	if len(r.Password) > 72 {
		return fmt.Errorf("%w: password must not exceed 72 bytes", ErrValidation)
	}

	var hasLetter, hasNumber bool

	for _, character := range r.Password {
		if unicode.IsLetter(character) {
			hasLetter = true
		}
		if unicode.IsNumber(character) {
			hasNumber = true
		}
	}

	if !hasLetter || !hasNumber {
		return fmt.Errorf("%w: password must contain at least one letter and one number", ErrValidation)
	}

	return nil
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *LoginRequest) Validate() error {
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))

	address, err := mail.ParseAddress(r.Email)
	if err != nil || address.Address != r.Email {
		return fmt.Errorf("%w: valid email address is required", ErrValidation)
	}

	if r.Password == "" {
		return fmt.Errorf("%w: password is required", ErrValidation)
	}

	return nil
}

// ProvisionRequest is used by admins to create new users with an explicit role.
type ProvisionRequest struct {
	FullName string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (r *ProvisionRequest) Validate() error {
	r.FullName = strings.TrimSpace(r.FullName)
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))
	r.Role = strings.TrimSpace(r.Role)

	if r.FullName == "" {
		return fmt.Errorf("%w: name is required", ErrValidation)
	}
	if len(r.FullName) > 150 {
		return fmt.Errorf("%w: name must not exceed 150 characters", ErrValidation)
	}

	address, err := mail.ParseAddress(r.Email)
	if err != nil || address.Address != r.Email {
		return fmt.Errorf("%w: valid email address is required", ErrValidation)
	}

	if len(r.Password) < 8 {
		return fmt.Errorf("%w: password must contain at least 8 characters", ErrValidation)
	}
	if len(r.Password) > 72 {
		return fmt.Errorf("%w: password must not exceed 72 bytes", ErrValidation)
	}

	var hasLetter, hasNumber bool
	for _, ch := range r.Password {
		if unicode.IsLetter(ch) {
			hasLetter = true
		}
		if unicode.IsNumber(ch) {
			hasNumber = true
		}
	}
	if !hasLetter || !hasNumber {
		return fmt.Errorf("%w: password must contain at least one letter and one number", ErrValidation)
	}

	if r.Role != "agent" && r.Role != "admin" {
		return fmt.Errorf("%w: role must be agent or admin", ErrValidation)
	}

	return nil
}

// ChangePasswordRequest is used by authenticated users to change their password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

func (r *ChangePasswordRequest) Validate() error {
	if r.CurrentPassword == "" {
		return fmt.Errorf("%w: current password is required", ErrValidation)
	}

	if len(r.NewPassword) < 8 {
		return fmt.Errorf("%w: password must contain at least 8 characters", ErrValidation)
	}
	if len(r.NewPassword) > 72 {
		return fmt.Errorf("%w: password must not exceed 72 bytes", ErrValidation)
	}

	var hasLetter, hasNumber bool
	for _, ch := range r.NewPassword {
		if unicode.IsLetter(ch) {
			hasLetter = true
		}
		if unicode.IsNumber(ch) {
			hasNumber = true
		}
	}
	if !hasLetter || !hasNumber {
		return fmt.Errorf("%w: password must contain at least one letter and one number", ErrValidation)
	}

	if r.ConfirmPassword != r.NewPassword {
		return fmt.Errorf("%w: passwords do not match", ErrValidation)
	}

	if r.NewPassword == r.CurrentPassword {
		return fmt.Errorf("%w: new password must differ from current password", ErrValidation)
	}

	return nil
}

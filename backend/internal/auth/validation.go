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

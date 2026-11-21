package auth

import (
	"errors"
	"net/mail"
	"strings"
	"unicode"
)

type RegisterRequest struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *RegisterRequest) Validate() error {
	r.FullName = strings.TrimSpace(r.FullName)
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))

	if r.FullName == "" {
		return errors.New("full name is required")
	}

	if len(r.FullName) > 150 {
		return errors.New("full name must not exceed 150 characters")
	}

	address, err := mail.ParseAddress(r.Email)
	if err != nil || address.Address != r.Email {
		return errors.New("valid email address is required")
	}

	if len(r.Password) < 8 {
		return errors.New("password must contain at least 8 characters")
	}

	if len(r.Password) > 72 {
		return errors.New("password must not exceed 72 bytes")
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
		return errors.New("password must contain at least one letter and one number")
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
		return errors.New("valid email address is required")
	}

	if r.Password == "" {
		return errors.New("password is required")
	}

	return nil
}

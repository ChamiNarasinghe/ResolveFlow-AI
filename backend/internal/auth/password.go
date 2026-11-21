package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword securely hashes a password.
func HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", errors.New("password must contain at least 8 characters")
	}

	if len(password) > 72 {
		return "", errors.New("password must not exceed 72 bytes")
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

// CheckPassword verifies a password against its stored hash.
func CheckPassword(password, passwordHash string) error {
	return bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(password),
	)
}

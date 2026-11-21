package auth

import "testing"

func TestHashAndCheckPassword(t *testing.T) {
	password := "Password123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash == password {
		t.Fatal("password should not be stored as plain text")
	}

	if err := CheckPassword(password, hash); err != nil {
		t.Fatalf("CheckPassword() rejected correct password: %v", err)
	}

	if err := CheckPassword("WrongPassword123", hash); err == nil {
		t.Fatal("CheckPassword() accepted an incorrect password")
	}
}

func TestHashPasswordRejectsInvalidLength(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{"too short", "Pass12"},
		{"too long", string(make([]byte, 73))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := HashPassword(tt.password)

			if err == nil {
				t.Fatal("expected password length validation error")
			}
		})
	}
}

package auth

import (
	"strings"
	"testing"
)

func TestRegisterRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		request RegisterRequest
		wantErr bool
	}{
		{
			name: "valid registration",
			request: RegisterRequest{
				FullName: "Chami Narasinghe",
				Email:    "chami@example.com",
				Password: "Password123",
			},
		},
		{
			name: "missing full name",
			request: RegisterRequest{
				Email:    "chami@example.com",
				Password: "Password123",
			},
			wantErr: true,
		},
		{
			name: "invalid email",
			request: RegisterRequest{
				FullName: "Chami",
				Email:    "invalid-email",
				Password: "Password123",
			},
			wantErr: true,
		},
		{
			name: "short password",
			request: RegisterRequest{
				FullName: "Chami",
				Email:    "chami@example.com",
				Password: "Pass12",
			},
			wantErr: true,
		},
		{
			name: "password without number",
			request: RegisterRequest{
				FullName: "Chami",
				Email:    "chami@example.com",
				Password: "PasswordOnly",
			},
			wantErr: true,
		},
		{
			name: "password exceeding bcrypt limit",
			request: RegisterRequest{
				FullName: "Chami",
				Email:    "chami@example.com",
				Password: strings.Repeat("a", 73) + "1",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()

			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoginRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		request LoginRequest
		wantErr bool
	}{
		{
			name: "valid login",
			request: LoginRequest{
				Email:    "chami@example.com",
				Password: "Password123",
			},
		},
		{
			name: "invalid email",
			request: LoginRequest{
				Email:    "invalid-email",
				Password: "Password123",
			},
			wantErr: true,
		},
		{
			name: "missing password",
			request: LoginRequest{
				Email: "chami@example.com",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()

			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

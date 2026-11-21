package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/chami2002/resolveflow-api/internal/auth"
)

type contextKey string

const userClaimsKey contextKey = "userClaims"

func AuthMiddleware(
	jwtService *auth.JWTService,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			parts := strings.Fields(authHeader)

			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				http.Error(w, "Missing or invalid authorization header", http.StatusUnauthorized)
				return
			}

			claims, err := jwtService.ValidateToken(parts[1])
			if err != nil {
				http.Error(w, "Invalid or expired access token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserClaims retrieves authenticated user information from the request.
func GetUserClaims(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(userClaimsKey).(*auth.Claims)
	return claims, ok
}

// SetUserClaims stores claims in the context. Used in tests and middleware.
func SetUserClaims(ctx context.Context, claims *auth.Claims) context.Context {
	return context.WithValue(ctx, userClaimsKey, claims)
}

// RequireRoles allows access only to users with one of the specified roles.
func RequireRoles(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := GetUserClaims(r.Context())
			if !ok {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			for _, role := range allowedRoles {
				if claims.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}

			http.Error(w, "You do not have permission to access this resource", http.StatusForbidden)
		})
	}
}

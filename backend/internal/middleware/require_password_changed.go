package middleware

import (
	"net/http"
)

// RequirePasswordChanged blocks any request whose JWT claims carry
// MustChangePassword = true. It must be applied after AuthMiddleware so that
// claims are already present in the context.
//
// Fails closed: if claims cannot be retrieved the request is rejected with 401.
func RequirePasswordChanged(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := GetUserClaims(r.Context())
		if !ok {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		if claims.MustChangePassword {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"password_change_required","message":"You must change your password before accessing this resource."}`)) //nolint:errcheck
			return
		}

		next.ServeHTTP(w, r)
	})
}

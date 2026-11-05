package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"aws-local-test/internal/auth"
)

// AuthMiddleware inspects and validates JWT Bearer tokens in incoming HTTP requests.
type AuthMiddleware struct{}

func NewAuthMiddleware() *AuthMiddleware {
	return &AuthMiddleware{}
}

func (m *AuthMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeUnauthorized(w, "Missing Authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeUnauthorized(w, "Invalid Authorization header format. Expected 'Bearer <token>'")
			return
		}

		tokenString := strings.TrimSpace(parts[1])
		if _, err := auth.ParseToken(tokenString); err != nil {
			writeUnauthorized(w, "Invalid or expired token")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": msg,
	})
}

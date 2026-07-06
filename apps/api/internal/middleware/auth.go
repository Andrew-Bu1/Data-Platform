package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Andrew-Bu1/api/internal/utils"
)

type contextKey string

const userIDKey contextKey = "user_id"

// GetUserID extracts the authenticated user's ID from the request context.
// Returns the user ID string and true if present, or an empty string and false if not.
func GetUserID(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userIDKey).(string)
	return v, ok
}

type AuthMiddleware struct {
	log       *slog.Logger
	secretKey string
}

func NewAuthMiddleware(logger *slog.Logger, secretKey string) *AuthMiddleware {
	return &AuthMiddleware{
		log:       logger,
		secretKey: secretKey,
	}
}

func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization := r.Header.Get("Authorization")
		if authorization == "" {
			http.Error(w, "authorization header is required", http.StatusUnauthorized)
			return
		}

		token, ok := strings.CutPrefix(authorization, "Bearer ")
		if !ok || token == "" {
			http.Error(w, "invalid authorization header", http.StatusUnauthorized)
			return
		}

		claims, err := utils.ValidateToken(token, m.secretKey)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

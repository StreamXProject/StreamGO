package middleware

import (
	"context"
	"net/http"
	"strings"

	"streamgo/internal/api"
	"streamgo/internal/services"
)

type contextKey string

const (
	userIDKey contextKey = "userID"
)

// WithUserID injects a numeric user ID into context.
func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetUserID extracts the user ID from context, returning ok=false if not found.
func GetUserID(ctx context.Context) (int64, bool) {
	val := ctx.Value(userIDKey)
	if val == nil {
		return 0, false
	}
	uid, ok := val.(int64)
	return uid, ok && uid > 0
}

func extractToken(r *http.Request) string {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if authHeader != "" {
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			return strings.TrimSpace(authHeader[7:])
		}
		return authHeader
	}

	if qToken := strings.TrimSpace(r.URL.Query().Get("token")); qToken != "" {
		return qToken
	}
	if qAuth := strings.TrimSpace(r.URL.Query().Get("auth")); qAuth != "" {
		return qAuth
	}

	return ""
}

// RequireAuth enforces a valid authentication token.
func RequireAuth(authSvc *services.AuthService) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				api.RespondError(w, http.StatusUnauthorized, "Authentication required")
				return
			}

			userID, err := authSvc.VerifyToken(token)
			if err != nil || userID <= 0 {
				api.RespondError(w, http.StatusUnauthorized, "Invalid or expired authentication token")
				return
			}

			ctx := WithUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth parses authentication token if present without rejecting unauthenticated requests.
func OptionalAuth(authSvc *services.AuthService) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token != "" {
				if userID, err := authSvc.VerifyToken(token); err == nil && userID > 0 {
					r = r.WithContext(WithUserID(r.Context(), userID))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

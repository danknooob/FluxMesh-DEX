package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const (
	CtxUserID ctxKey = "user_id"
	CtxRole   ctxKey = "role"
)

type Claims struct {
	UserID string `json:"sub"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// JWTAuth returns middleware that validates the Authorization Bearer token.
// On success it injects X-User-ID and X-Role headers into the proxied request
// and stores them in context for downstream middleware (e.g. rate limiter).
func JWTAuth(secret string, requireAdmin bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeJSONError := func(code int, msg string) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(code)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
			}

			h := r.Header.Get("Authorization")
			if h == "" || !strings.HasPrefix(h, "Bearer ") {
				writeJSONError(http.StatusUnauthorized, "missing or invalid authorization header")
				return
			}

			raw := strings.TrimPrefix(h, "Bearer ")
			token, err := jwt.ParseWithClaims(raw, &Claims{}, func(t *jwt.Token) (interface{}, error) {
				return []byte(secret), nil
			})
			if err != nil {
				writeJSONError(http.StatusUnauthorized, "invalid or expired token")
				return
			}

			claims, ok := token.Claims.(*Claims)
			if !ok || !token.Valid {
				writeJSONError(http.StatusUnauthorized, "invalid token claims")
				return
			}

			if requireAdmin && claims.Role != "admin" {
				writeJSONError(http.StatusForbidden, "admin access required")
				return
			}

			r.Header.Set("X-User-ID", claims.UserID)
			r.Header.Set("X-Role", claims.Role)

			ctx := context.WithValue(r.Context(), CtxUserID, claims.UserID)
			ctx = context.WithValue(ctx, CtxRole, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

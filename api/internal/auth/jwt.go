package auth

import (
	"net/http"
	"strings"
	"time"

	"github.com/danknooob/fluxmesh-dex/api/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

// Claims represents JWT claims used by the API.
type Claims struct {
	UserID string `json:"sub"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// NewToken issues a new JWT for a given user and role.
func NewToken(userID, role string, cfg *config.Config) (string, error) {
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(cfg.JWT.ExpireMins) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(cfg.JWT.Secret))
}

func parseToken(tokenStr string, cfg *config.Config) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(cfg.JWT.Secret), nil
	})
	if err != nil {
		return nil, err
	}
	if c, ok := token.Claims.(*Claims); ok && token.Valid {
		return c, nil
	}
	return nil, jwt.ErrTokenInvalidClaims
}

// AuthMiddleware validates a Bearer token and injects user/role into context.
// If requireAdmin is true, only admin tokens are allowed.
func AuthMiddleware(cfg *config.Config, requireAdmin bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if h == "" || !strings.HasPrefix(h, "Bearer ") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		raw := strings.TrimPrefix(h, "Bearer ")

		claims, err := parseToken(raw, cfg)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if requireAdmin && claims.Role != "admin" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		ctx := WithUser(r.Context(), claims.UserID, claims.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}


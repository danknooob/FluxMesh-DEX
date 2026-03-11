package auth

import "context"

type ctxKey string

const (
	ctxUserIDKey ctxKey = "user_id"
	ctxRoleKey   ctxKey = "role"
)

// WithUser attaches user identity and role to the context.
func WithUser(ctx context.Context, userID, role string) context.Context {
	ctx = context.WithValue(ctx, ctxUserIDKey, userID)
	ctx = context.WithValue(ctx, ctxRoleKey, role)
	return ctx
}

// UserIDFrom extracts the user id from context, if present.
func UserIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(ctxUserIDKey).(string); ok {
		return v
	}
	return ""
}

// RoleFrom extracts the role from context, if present.
func RoleFrom(ctx context.Context) string {
	if v, ok := ctx.Value(ctxRoleKey).(string); ok {
		return v
	}
	return ""
}


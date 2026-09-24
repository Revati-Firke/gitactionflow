package auth

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type contextKey string

const userContextKey contextKey = "auth_user"

// ContextWithUser stores the authenticated user on a context.
func ContextWithUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// UserFromContext returns the authenticated user if present.
func UserFromContext(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(userContextKey).(User)
	return u, ok
}

// SetUserOnGin attaches the user to Gin context and request context.
func SetUserOnGin(c *gin.Context, user User) {
	c.Set(string(userContextKey), user)
	c.Request = c.Request.WithContext(ContextWithUser(c.Request.Context(), user))
}

// UserFromGin returns the user previously attached by middleware.
func UserFromGin(c *gin.Context) (User, bool) {
	v, ok := c.Get(string(userContextKey))
	if !ok {
		return User{}, false
	}
	u, ok := v.(User)
	return u, ok
}

// MustUserID is a helper for handlers that already passed RequireAuth.
func MustUserID(c *gin.Context) uuid.UUID {
	u, ok := UserFromGin(c)
	if !ok {
		return uuid.Nil
	}
	return u.ID
}

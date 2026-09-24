package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Revati-Firke/gitactionflow/backend/internal/auth"
	"github.com/Revati-Firke/gitactionflow/backend/internal/http/response"
	"github.com/Revati-Firke/gitactionflow/backend/internal/store"
)

// SessionReader loads valid sessions.
type SessionReader interface {
	GetValidByTokenHash(ctx context.Context, tokenHash []byte, now time.Time) (auth.Session, error)
}

// UserReader loads users by ID.
type UserReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (auth.User, error)
}

// AuthDeps wires session validation for protected routes.
type AuthDeps struct {
	Sessions SessionReader
	Users    UserReader
	Log      *slog.Logger
}

// RequireAuth rejects requests without a valid session cookie.
func RequireAuth(deps AuthDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, err := auth.ReadSessionCookie(c)
		if err != nil || raw == "" {
			response.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
			return
		}

		hash := auth.HashToken(raw)
		sess, err := deps.Sessions.GetValidByTokenHash(c.Request.Context(), hash, time.Now().UTC())
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				response.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
				return
			}
			if deps.Log != nil {
				deps.Log.Error("session lookup failed", "err", err)
			}
			response.Internal(c)
			return
		}

		user, err := deps.Users.GetByID(c.Request.Context(), sess.UserID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				response.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
				return
			}
			if deps.Log != nil {
				deps.Log.Error("user lookup failed", "err", err)
			}
			response.Internal(c)
			return
		}

		auth.SetUserOnGin(c, user)
		c.Next()
	}
}

package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Revati-Firke/gitactionflow/backend/internal/http/response"
)

// Pinger checks a dependency's health (typically PostgreSQL).
type Pinger interface {
	Ping(ctx context.Context) error
}

// Health handles GET /health (liveness).
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Ready returns a handler for GET /ready (readiness).
func Ready(db Pinger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if db == nil {
			response.JSONError(c, http.StatusServiceUnavailable, "NOT_READY", "database unavailable")
			return
		}
		if err := db.Ping(c.Request.Context()); err != nil {
			// Do not expose the underlying database error to clients.
			response.JSONError(c, http.StatusServiceUnavailable, "NOT_READY", "database unavailable")
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}

package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS allows the configured frontend origin with credentials (for SPA calls).
func CORS(frontendURL string) gin.HandlerFunc {
	allowed := normalizeOrigin(frontendURL)
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if allowed != "" && origin != "" && normalizeOrigin(origin) == allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Content-Type, X-Request-ID")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Expose-Headers", RequestIDHeader)
			c.Header("Vary", "Origin")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// TrimFrontendURL normalizes FRONTEND_URL for comparisons (exported for config).
func TrimFrontendURL(u string) string {
	return strings.TrimRight(strings.TrimSpace(u), "/")
}

package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders sets conservative browser security headers on all responses.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	}
}

// CSRFOrigin rejects cookie-authenticated mutating requests whose Origin does not
// match the configured frontend. GitHub webhooks and OAuth GETs are unaffected
// when this middleware is applied only to the /api group (and logout).
func CSRFOrigin(frontendURL string) gin.HandlerFunc {
	allowed := normalizeOrigin(frontendURL)
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}
		origin := c.GetHeader("Origin")
		if origin == "" {
			// Same-origin navigations sometimes omit Origin; Referer is a fallback.
			ref := c.GetHeader("Referer")
			if ref == "" {
				// Non-browser clients (curl) without Origin — allow for API tooling
				// when SameSite=Lax local; in production Prefer Origin present.
				c.Next()
				return
			}
			origin = ref
		}
		if normalizeOrigin(origin) != allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{"code": "CSRF_ORIGIN", "message": "origin not allowed"},
			})
			return
		}
		c.Next()
	}
}

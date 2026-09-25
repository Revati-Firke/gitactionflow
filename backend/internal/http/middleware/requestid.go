package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"

	"github.com/gin-gonic/gin"
)

const RequestIDHeader = "X-Request-ID"
const requestIDKey = "request_id"

// RequestID assigns or propagates a request id for log correlation.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			var b [8]byte
			_, _ = rand.Read(b[:])
			id = hex.EncodeToString(b[:])
		}
		c.Set(requestIDKey, id)
		c.Header(RequestIDHeader, id)
		c.Next()
	}
}

// RequestIDFromGin returns the request id if present.
func RequestIDFromGin(c *gin.Context) string {
	v, _ := c.Get(requestIDKey)
	s, _ := v.(string)
	return s
}

// LogWithRequest returns a logger with request_id when available.
func LogWithRequest(base *slog.Logger, c *gin.Context) *slog.Logger {
	if base == nil {
		base = slog.Default()
	}
	if id := RequestIDFromGin(c); id != "" {
		return base.With("request_id", id)
	}
	return base
}

package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorBody is the consistent client-facing error envelope.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail describes a single API error without leaking internals.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSONError writes a structured error response.
func JSONError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, ErrorBody{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

// Internal writes a generic internal error (safe for clients).
func Internal(c *gin.Context) {
	JSONError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}

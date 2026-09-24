package auth

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const SessionCookieName = "gaf_session"

// CookieOptions controls how the session cookie is written.
type CookieOptions struct {
	Secure   bool
	SameSite string
	TTL      time.Duration
	Path     string
}

func (o CookieOptions) sameSiteMode() http.SameSite {
	switch strings.ToLower(o.SameSite) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func (o CookieOptions) path() string {
	if o.Path == "" {
		return "/"
	}
	return o.Path
}

// SetSessionCookie writes the HttpOnly session cookie.
func SetSessionCookie(c *gin.Context, token string, opts CookieOptions) {
	c.SetSameSite(opts.sameSiteMode())
	c.SetCookie(
		SessionCookieName,
		token,
		int(opts.TTL.Seconds()),
		opts.path(),
		"",
		opts.Secure,
		true, // HttpOnly
	)
}

// ClearSessionCookie removes the session cookie.
func ClearSessionCookie(c *gin.Context, opts CookieOptions) {
	c.SetSameSite(opts.sameSiteMode())
	c.SetCookie(
		SessionCookieName,
		"",
		-1,
		opts.path(),
		"",
		opts.Secure,
		true,
	)
}

// ReadSessionCookie returns the raw session token from the request cookie.
func ReadSessionCookie(c *gin.Context) (string, error) {
	return c.Cookie(SessionCookieName)
}

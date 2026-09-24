package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Revati-Firke/gitactionflow/backend/internal/auth"
	"github.com/Revati-Firke/gitactionflow/backend/internal/githuboauth"
	"github.com/Revati-Firke/gitactionflow/backend/internal/http/response"
	"github.com/Revati-Firke/gitactionflow/backend/internal/store"
)

// OAuthStateStore persists one-time OAuth CSRF states.
type OAuthStateStore interface {
	Create(ctx context.Context, stateHash []byte, expiresAt time.Time) error
	Consume(ctx context.Context, stateHash []byte, now time.Time) error
}

// UserStore persists users.
type UserStore interface {
	UpsertFromGitHub(ctx context.Context, githubUserID int64, login, name, avatarURL string, encryptedToken []byte) (auth.User, error)
}

// SessionStore persists sessions.
type SessionStore interface {
	Create(ctx context.Context, userID uuid.UUID, tokenHash []byte, expiresAt time.Time) (auth.Session, error)
	DeleteByTokenHash(ctx context.Context, tokenHash []byte) error
}

// GitHubClient exchanges codes and fetches the GitHub user.
type GitHubClient interface {
	AuthorizeURL(state string) string
	ExchangeCode(ctx context.Context, code string) (githuboauth.TokenResponse, error)
	FetchUser(ctx context.Context, accessToken string) (githuboauth.User, error)
}

// AuthHandler serves OAuth and session endpoints.
type AuthHandler struct {
	GitHub      GitHubClient
	States      OAuthStateStore
	Users       UserStore
	Sessions    SessionStore
	TokenKey    []byte
	Cookie      auth.CookieOptions
	StateTTL    time.Duration
	FrontendURL string
	Log         *slog.Logger
}

// StartGitHub begins the OAuth flow.
func (h *AuthHandler) StartGitHub(c *gin.Context) {
	state, err := auth.RandomURLToken(32)
	if err != nil {
		h.logErr("generate oauth state", err)
		response.Internal(c)
		return
	}

	expires := time.Now().UTC().Add(h.StateTTL)
	if err := h.States.Create(c.Request.Context(), auth.HashToken(state), expires); err != nil {
		h.logErr("store oauth state", err)
		response.Internal(c)
		return
	}

	c.Redirect(http.StatusFound, h.GitHub.AuthorizeURL(state))
}

// CallbackGitHub completes the OAuth flow.
func (h *AuthHandler) CallbackGitHub(c *gin.Context) {
	if errParam := c.Query("error"); errParam != "" {
		h.redirectFrontend(c, "/login", "oauth_denied")
		return
	}

	code := c.Query("code")
	state := c.Query("state")
	if code == "" || state == "" {
		h.redirectFrontend(c, "/login", "invalid_callback")
		return
	}

	if err := h.States.Consume(c.Request.Context(), auth.HashToken(state), time.Now().UTC()); err != nil {
		h.redirectFrontend(c, "/login", "invalid_state")
		return
	}

	token, err := h.GitHub.ExchangeCode(c.Request.Context(), code)
	if err != nil {
		h.logErr("github token exchange", err)
		h.redirectFrontend(c, "/login", "oauth_failed")
		return
	}

	ghUser, err := h.GitHub.FetchUser(c.Request.Context(), token.AccessToken)
	if err != nil {
		h.logErr("github fetch user", err)
		h.redirectFrontend(c, "/login", "oauth_failed")
		return
	}

	encrypted, err := auth.Encrypt(h.TokenKey, []byte(token.AccessToken))
	if err != nil {
		h.logErr("encrypt access token", err)
		response.Internal(c)
		return
	}

	display := ghUser.Name
	if display == "" {
		display = ghUser.Login
	}

	user, err := h.Users.UpsertFromGitHub(
		c.Request.Context(),
		ghUser.ID,
		ghUser.Login,
		display,
		ghUser.AvatarURL,
		encrypted,
	)
	if err != nil {
		h.logErr("upsert user", err)
		response.Internal(c)
		return
	}

	sessionToken, err := auth.RandomURLToken(32)
	if err != nil {
		h.logErr("generate session token", err)
		response.Internal(c)
		return
	}

	expires := time.Now().UTC().Add(h.Cookie.TTL)
	if _, err := h.Sessions.Create(c.Request.Context(), user.ID, auth.HashToken(sessionToken), expires); err != nil {
		h.logErr("create session", err)
		response.Internal(c)
		return
	}

	auth.SetSessionCookie(c, sessionToken, h.Cookie)
	h.redirectFrontend(c, "/", "")
}

// Logout invalidates the current session (idempotent).
func (h *AuthHandler) Logout(c *gin.Context) {
	raw, err := auth.ReadSessionCookie(c)
	if err == nil && raw != "" {
		if delErr := h.Sessions.DeleteByTokenHash(c.Request.Context(), auth.HashToken(raw)); delErr != nil {
			h.logErr("delete session", delErr)
			response.Internal(c)
			return
		}
	}
	auth.ClearSessionCookie(c, h.Cookie)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Me returns the authenticated user profile.
func Me(c *gin.Context) {
	user, ok := auth.UserFromGin(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user.ToPublic()})
}

func (h *AuthHandler) redirectFrontend(c *gin.Context, path, errCode string) {
	u, err := url.Parse(h.FrontendURL)
	if err != nil {
		response.Internal(c)
		return
	}
	u.Path = path
	if errCode != "" {
		q := u.Query()
		q.Set("error", errCode)
		u.RawQuery = q.Encode()
	}
	c.Redirect(http.StatusFound, u.String())
}

func (h *AuthHandler) logErr(msg string, err error) {
	if h.Log == nil {
		return
	}
	h.Log.Error(msg, "err", err)
}

// Compile-time checks against concrete stores.
var (
	_ OAuthStateStore = (*store.OAuthStates)(nil)
	_ UserStore       = (*store.Users)(nil)
	_ SessionStore    = (*store.Sessions)(nil)
	_ GitHubClient    = (*githuboauth.Client)(nil)
)

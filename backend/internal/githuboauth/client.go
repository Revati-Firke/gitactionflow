package githuboauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	authorizeURL = "https://github.com/login/oauth/authorize"
	tokenURL     = "https://github.com/login/oauth/access_token"
	userURL      = "https://api.github.com/user"
	// Scopes: identify the user + repo access needed later for webhooks/labels/comments.
	// Kept minimal for the assignment workflow (no admin, no org, no gist).
	DefaultScopes = "read:user repo"
)

// Config for the GitHub OAuth HTTP client.
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	HTTPClient   *http.Client
	Scopes       string
}

// Client talks to GitHub's OAuth and REST APIs.
type Client struct {
	cfg Config
}

// New creates a GitHub OAuth client.
func New(cfg Config) *Client {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	if cfg.Scopes == "" {
		cfg.Scopes = DefaultScopes
	}
	return &Client{cfg: cfg}
}

// AuthorizeURL builds the GitHub authorize redirect URL.
func (c *Client) AuthorizeURL(state string) string {
	q := url.Values{}
	q.Set("client_id", c.cfg.ClientID)
	q.Set("redirect_uri", c.cfg.RedirectURL)
	q.Set("scope", c.cfg.Scopes)
	q.Set("state", state)
	return authorizeURL + "?" + q.Encode()
}

// TokenResponse is the OAuth token exchange result.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// User is the subset of GitHub /user we persist.
type User struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

// ExchangeCode swaps an authorization code for an access token.
func (c *Client) ExchangeCode(ctx context.Context, code string) (TokenResponse, error) {
	form := url.Values{}
	form.Set("client_id", c.cfg.ClientID)
	form.Set("client_secret", c.cfg.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", c.cfg.RedirectURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return TokenResponse{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("token exchange request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return TokenResponse{}, fmt.Errorf("read token response: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return TokenResponse{}, fmt.Errorf("token exchange status %d", res.StatusCode)
	}

	var tr TokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return TokenResponse{}, fmt.Errorf("decode token response: %w", err)
	}
	if tr.AccessToken == "" {
		// GitHub may return {"error":"..."} with 200
		return TokenResponse{}, fmt.Errorf("token exchange failed")
	}
	return tr, nil
}

// FetchUser loads the authenticated GitHub user profile.
func (c *Client) FetchUser(ctx context.Context, accessToken string) (User, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userURL, nil)
	if err != nil {
		return User{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	res, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return User{}, fmt.Errorf("github user request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return User{}, fmt.Errorf("read github user: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return User{}, fmt.Errorf("github user status %d", res.StatusCode)
	}

	var u User
	if err := json.Unmarshal(body, &u); err != nil {
		return User{}, fmt.Errorf("decode github user: %w", err)
	}
	if u.ID == 0 || u.Login == "" {
		return User{}, fmt.Errorf("invalid github user payload")
	}
	return u, nil
}

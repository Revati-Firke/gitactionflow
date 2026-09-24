package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds application configuration loaded from the environment.
type Config struct {
	AppEnv      string
	AppPort     int
	DatabaseURL string
	LogLevel    string
	AutoMigrate bool

	FrontendURL            string
	GitHubClientID         string
	GitHubClientSecret     string
	GitHubOAuthRedirectURL string
	GitHubWebhookSecret    string
	SlackWebhookURL        string
	SessionSecret          string
	SessionTTL             time.Duration
	OAuthStateTTL          time.Duration
	CookieSecure           bool
	CookieSameSite         string
	AIAPIKey               string
}

// Load reads configuration from environment variables, applies defaults, and validates.
func Load() (Config, error) {
	cfg := Config{
		AppEnv:      getEnv("APP_ENV", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")),

		FrontendURL:            firstNonEmpty(os.Getenv("FRONTEND_URL"), os.Getenv("FRONTEND_ORIGIN")),
		GitHubClientID:         strings.TrimSpace(os.Getenv("GITHUB_CLIENT_ID")),
		GitHubClientSecret:     os.Getenv("GITHUB_CLIENT_SECRET"),
		GitHubOAuthRedirectURL: getEnv("GITHUB_OAUTH_REDIRECT_URL", "http://localhost:8080/auth/github/callback"),
		GitHubWebhookSecret:    os.Getenv("GITHUB_WEBHOOK_SECRET"),
		SlackWebhookURL:        os.Getenv("SLACK_WEBHOOK_URL"),
		SessionSecret:          os.Getenv("SESSION_SECRET"),
		CookieSameSite:         getEnv("COOKIE_SAMESITE", "Lax"),
		AIAPIKey:               firstNonEmpty(os.Getenv("AI_API_KEY"), os.Getenv("GEMINI_API_KEY"), os.Getenv("GROQ_API_KEY")),
	}

	if cfg.FrontendURL == "" {
		cfg.FrontendURL = "http://localhost:5173"
	}

	port, err := parsePort(getEnv("APP_PORT", "8080"))
	if err != nil {
		return Config{}, err
	}
	cfg.AppPort = port

	autoMigrate, err := parseBool(getEnv("AUTO_MIGRATE", defaultAutoMigrate(cfg.AppEnv)))
	if err != nil {
		return Config{}, fmt.Errorf("AUTO_MIGRATE: %w", err)
	}
	cfg.AutoMigrate = autoMigrate

	secureDefault := "false"
	if strings.EqualFold(cfg.AppEnv, "production") {
		secureDefault = "true"
	}
	cookieSecure, err := parseBool(getEnv("COOKIE_SECURE", secureDefault))
	if err != nil {
		return Config{}, fmt.Errorf("COOKIE_SECURE: %w", err)
	}
	cfg.CookieSecure = cookieSecure

	sessionTTL, err := parseDuration(getEnv("SESSION_TTL", "168h")) // 7 days
	if err != nil {
		return Config{}, fmt.Errorf("SESSION_TTL: %w", err)
	}
	cfg.SessionTTL = sessionTTL

	stateTTL, err := parseDuration(getEnv("OAUTH_STATE_TTL", "10m"))
	if err != nil {
		return Config{}, fmt.Errorf("OAUTH_STATE_TTL: %w", err)
	}
	cfg.OAuthStateTTL = stateTTL

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate checks required configuration.
func (c Config) Validate() error {
	if strings.TrimSpace(c.AppEnv) == "" {
		return fmt.Errorf("APP_ENV is required")
	}
	if c.AppPort < 1 || c.AppPort > 65535 {
		return fmt.Errorf("APP_PORT must be between 1 and 65535")
	}
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if !validLogLevel(c.LogLevel) {
		return fmt.Errorf("LOG_LEVEL must be one of: debug, info, warn, error")
	}
	if strings.TrimSpace(c.FrontendURL) == "" {
		return fmt.Errorf("FRONTEND_URL is required")
	}
	if strings.TrimSpace(c.GitHubClientID) == "" {
		return fmt.Errorf("GITHUB_CLIENT_ID is required")
	}
	if strings.TrimSpace(c.GitHubClientSecret) == "" {
		return fmt.Errorf("GITHUB_CLIENT_SECRET is required")
	}
	if strings.TrimSpace(c.GitHubOAuthRedirectURL) == "" {
		return fmt.Errorf("GITHUB_OAUTH_REDIRECT_URL is required")
	}
	if len(strings.TrimSpace(c.SessionSecret)) < 32 {
		return fmt.Errorf("SESSION_SECRET is required and must be at least 32 characters")
	}
	if c.SessionTTL <= 0 {
		return fmt.Errorf("SESSION_TTL must be positive")
	}
	if c.OAuthStateTTL <= 0 {
		return fmt.Errorf("OAUTH_STATE_TTL must be positive")
	}
	switch strings.ToLower(c.CookieSameSite) {
	case "lax", "strict", "none":
	default:
		return fmt.Errorf("COOKIE_SAMESITE must be one of: Lax, Strict, None")
	}
	if strings.EqualFold(c.CookieSameSite, "none") && !c.CookieSecure {
		return fmt.Errorf("COOKIE_SAMESITE=None requires COOKIE_SECURE=true")
	}
	return nil
}

// Addr returns the HTTP listen address (e.g. ":8080").
func (c Config) Addr() string {
	return fmt.Sprintf(":%d", c.AppPort)
}

// Redacted returns a copy safe for logging (secrets cleared).
func (c Config) Redacted() Config {
	out := c
	out.DatabaseURL = redactURL(c.DatabaseURL)
	out.GitHubClientSecret = redactIfSet(c.GitHubClientSecret)
	out.GitHubWebhookSecret = redactIfSet(c.GitHubWebhookSecret)
	out.SlackWebhookURL = redactIfSet(c.SlackWebhookURL)
	out.SessionSecret = redactIfSet(c.SessionSecret)
	out.AIAPIKey = redactIfSet(c.AIAPIKey)
	return out
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}

func parsePort(raw string) (int, error) {
	port, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("APP_PORT must be an integer: %w", err)
	}
	return port, nil
}

func parseBool(raw string) (bool, error) {
	v, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return false, fmt.Errorf("invalid boolean %q", raw)
	}
	return v, nil
}

func parseDuration(raw string) (time.Duration, error) {
	d, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil {
		return 0, err
	}
	return d, nil
}

func defaultAutoMigrate(appEnv string) string {
	if strings.EqualFold(appEnv, "development") || strings.EqualFold(appEnv, "test") {
		return "true"
	}
	return "false"
}

func validLogLevel(level string) bool {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug", "info", "warn", "error":
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func redactIfSet(v string) string {
	if strings.TrimSpace(v) == "" {
		return ""
	}
	return "[redacted]"
}

func redactURL(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	if i := strings.Index(raw, "://"); i >= 0 {
		rest := raw[i+3:]
		if at := strings.Index(rest, "@"); at >= 0 {
			return raw[:i+3] + "[redacted]@" + rest[at+1:]
		}
	}
	return "[redacted]"
}

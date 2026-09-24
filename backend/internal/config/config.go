package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds application configuration loaded from the environment.
// Future integration fields are optional placeholders and are not used in Phase 2.
type Config struct {
	AppEnv      string
	AppPort     int
	DatabaseURL string
	LogLevel    string
	AutoMigrate bool

	// Future placeholders (optional; not required for Phase 2).
	GitHubClientID      string
	GitHubClientSecret  string
	GitHubWebhookSecret string
	SlackWebhookURL     string
	SessionSecret       string
	AIAPIKey            string
}

// Load reads configuration from environment variables, applies defaults, and validates.
func Load() (Config, error) {
	cfg := Config{
		AppEnv:      getEnv("APP_ENV", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")),

		GitHubClientID:      os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret:  os.Getenv("GITHUB_CLIENT_SECRET"),
		GitHubWebhookSecret: os.Getenv("GITHUB_WEBHOOK_SECRET"),
		SlackWebhookURL:     os.Getenv("SLACK_WEBHOOK_URL"),
		SessionSecret:       os.Getenv("SESSION_SECRET"),
		AIAPIKey:            firstNonEmpty(os.Getenv("AI_API_KEY"), os.Getenv("GEMINI_API_KEY"), os.Getenv("GROQ_API_KEY")),
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
			return v
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
	// Avoid logging credentials embedded in postgres://user:pass@host/db
	if i := strings.Index(raw, "://"); i >= 0 {
		rest := raw[i+3:]
		if at := strings.Index(rest, "@"); at >= 0 {
			return raw[:i+3] + "[redacted]@" + rest[at+1:]
		}
	}
	return "[redacted]"
}

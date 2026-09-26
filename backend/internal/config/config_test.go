package config_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Revati-Firke/gitactionflow/backend/internal/config"
)

func setAuthEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://gitactionflow:gitactionflow@localhost:5432/gitactionflow?sslmode=disable")
	t.Setenv("GITHUB_CLIENT_ID", "test-client-id")
	t.Setenv("GITHUB_CLIENT_SECRET", "test-client-secret")
	t.Setenv("GITHUB_OAUTH_REDIRECT_URL", "http://localhost:8080/auth/github/callback")
	t.Setenv("SESSION_SECRET", "0123456789abcdef0123456789abcdef") // 32 chars
	t.Setenv("GITHUB_WEBHOOK_SECRET", "webhook-secret-16+")
	t.Setenv("FRONTEND_URL", "http://localhost:5173")
}

func TestLoad_Defaults(t *testing.T) {
	setAuthEnv(t)
	t.Setenv("APP_ENV", "")
	t.Setenv("APP_PORT", "")
	t.Setenv("PORT", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("AUTO_MIGRATE", "")
	t.Setenv("EVENT_WORKER_ENABLED", "")
	t.Setenv("EVENT_WORKER_POLL_INTERVAL", "")
	t.Setenv("EVENT_MAX_RETRIES", "")
	t.Setenv("EVENT_PROCESSING_LEASE", "")
	t.Setenv("ACTION_MAX_RETRIES", "")
	t.Setenv("COOKIE_SECURE", "")
	t.Setenv("COOKIE_SAMESITE", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.AppEnv != "development" {
		t.Fatalf("AppEnv = %q, want development", cfg.AppEnv)
	}
	if cfg.AppPort != 8080 {
		t.Fatalf("AppPort = %d, want 8080", cfg.AppPort)
	}
	if !cfg.AutoMigrate {
		t.Fatal("AutoMigrate should default true in development")
	}
	if cfg.Addr() != ":8080" {
		t.Fatalf("Addr() = %q, want :8080", cfg.Addr())
	}
	if cfg.CookieSecure {
		t.Fatal("CookieSecure should default false in development")
	}
	if !strings.EqualFold(cfg.CookieSameSite, "Lax") {
		t.Fatalf("CookieSameSite = %q, want Lax", cfg.CookieSameSite)
	}
	if !cfg.EventWorkerEnabled {
		t.Fatal("EventWorkerEnabled should default true")
	}
	if cfg.EventWorkerPollInterval != 2*time.Second {
		t.Fatalf("poll = %v", cfg.EventWorkerPollInterval)
	}
	if cfg.EventMaxRetries != 3 {
		t.Fatalf("max retries = %d", cfg.EventMaxRetries)
	}
	if cfg.EventProcessingLease != time.Minute {
		t.Fatalf("lease = %v", cfg.EventProcessingLease)
	}
	if cfg.ActionMaxRetries != 3 {
		t.Fatalf("action max retries = %d", cfg.ActionMaxRetries)
	}
}

func TestLoad_PrefersPORTOverAPP_PORT(t *testing.T) {
	setAuthEnv(t)
	t.Setenv("APP_PORT", "8080")
	t.Setenv("PORT", "10000")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AppPort != 10000 {
		t.Fatalf("AppPort = %d, want 10000 from PORT", cfg.AppPort)
	}
}

func TestLoad_ProductionCookieDefaults(t *testing.T) {
	setAuthEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("COOKIE_SECURE", "")
	t.Setenv("COOKIE_SAMESITE", "")
	t.Setenv("AUTO_MIGRATE", "true") // override default false for prod
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.CookieSecure {
		t.Fatal("CookieSecure should default true in production")
	}
	if !strings.EqualFold(cfg.CookieSameSite, "None") {
		t.Fatalf("CookieSameSite = %q, want None in production", cfg.CookieSameSite)
	}
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	setAuthEnv(t)
	t.Setenv("DATABASE_URL", "")
	_, err := config.Load()
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("expected DATABASE_URL error, got %v", err)
	}
}

func TestLoad_MissingSessionSecret(t *testing.T) {
	setAuthEnv(t)
	t.Setenv("SESSION_SECRET", "short")
	_, err := config.Load()
	if err == nil || !strings.Contains(err.Error(), "SESSION_SECRET") {
		t.Fatalf("expected SESSION_SECRET error, got %v", err)
	}
}

func TestLoad_MissingGitHubClientID(t *testing.T) {
	setAuthEnv(t)
	t.Setenv("GITHUB_CLIENT_ID", "")
	_, err := config.Load()
	if err == nil || !strings.Contains(err.Error(), "GITHUB_CLIENT_ID") {
		t.Fatalf("expected GITHUB_CLIENT_ID error, got %v", err)
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	setAuthEnv(t)
	t.Setenv("PORT", "")
	t.Setenv("APP_PORT", "not-a-number")
	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for invalid APP_PORT")
	}
}

func TestLoad_InvalidLogLevel(t *testing.T) {
	setAuthEnv(t)
	t.Setenv("LOG_LEVEL", "verbose")
	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for invalid LOG_LEVEL")
	}
}

func TestLoad_SameSiteNoneRequiresSecure(t *testing.T) {
	setAuthEnv(t)
	t.Setenv("COOKIE_SAMESITE", "None")
	t.Setenv("COOKIE_SECURE", "false")
	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for SameSite=None without Secure")
	}
}

func TestRedacted_HidesSecrets(t *testing.T) {
	cfg := config.Config{
		DatabaseURL:        "postgres://user:secret@localhost:5432/db",
		GitHubClientSecret: "gh-secret",
		SessionSecret:      "session-secret-value-here-32chars!!",
		SlackWebhookURL:    "https://hooks.slack.com/services/T/B/xxx",
	}
	redacted := cfg.Redacted()
	if strings.Contains(redacted.DatabaseURL, "secret") {
		t.Fatalf("DatabaseURL still contains secret: %s", redacted.DatabaseURL)
	}
	if redacted.GitHubClientSecret != "[redacted]" || redacted.SessionSecret != "[redacted]" {
		t.Fatalf("secrets not redacted: %+v", redacted)
	}
}

package config_test

import (
	"strings"
	"testing"

	"github.com/Revati-Firke/gitactionflow/backend/internal/config"
)

func TestLoad_DefaultsAndRequiredDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://gitactionflow:gitactionflow@localhost:5432/gitactionflow?sslmode=disable")
	t.Setenv("APP_ENV", "")
	t.Setenv("APP_PORT", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("AUTO_MIGRATE", "")

	// Clear then set only DATABASE_URL — Load uses LookupEnv with defaults.
	t.Setenv("DATABASE_URL", "postgres://gitactionflow:gitactionflow@localhost:5432/gitactionflow?sslmode=disable")

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
	if cfg.LogLevel != "info" {
		t.Fatalf("LogLevel = %q, want info", cfg.LogLevel)
	}
	if !cfg.AutoMigrate {
		t.Fatal("AutoMigrate should default true in development")
	}
	if cfg.Addr() != ":8080" {
		t.Fatalf("Addr() = %q, want :8080", cfg.Addr())
	}
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("error %q should mention DATABASE_URL", err)
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("APP_PORT", "not-a-number")
	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for invalid APP_PORT")
	}
}

func TestLoad_InvalidLogLevel(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("APP_PORT", "8080")
	t.Setenv("LOG_LEVEL", "verbose")
	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for invalid LOG_LEVEL")
	}
}

func TestValidate_PortRange(t *testing.T) {
	cfg := config.Config{
		AppEnv:      "development",
		AppPort:     0,
		DatabaseURL: "postgres://u:p@localhost:5432/db",
		LogLevel:    "info",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for port 0")
	}
}

func TestRedacted_HidesSecrets(t *testing.T) {
	cfg := config.Config{
		DatabaseURL:         "postgres://user:secret@localhost:5432/db",
		GitHubClientSecret:  "gh-secret",
		GitHubWebhookSecret: "wh-secret",
		SlackWebhookURL:     "https://hooks.slack.com/services/T/B/xxx",
		SessionSecret:       "session",
		AIAPIKey:            "ai-key",
	}
	redacted := cfg.Redacted()
	if strings.Contains(redacted.DatabaseURL, "secret") {
		t.Fatalf("DatabaseURL still contains secret: %s", redacted.DatabaseURL)
	}
	if redacted.GitHubClientSecret != "[redacted]" {
		t.Fatalf("GitHubClientSecret = %q", redacted.GitHubClientSecret)
	}
	if redacted.SlackWebhookURL != "[redacted]" {
		t.Fatalf("SlackWebhookURL = %q", redacted.SlackWebhookURL)
	}
}

package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Revati-Firke/gitactionflow/backend/internal/config"
	"github.com/Revati-Firke/gitactionflow/backend/internal/database"
	"github.com/Revati-Firke/gitactionflow/backend/internal/http/router"
	"github.com/Revati-Firke/gitactionflow/backend/internal/logging"
)

// Run loads configuration, wires dependencies, serves HTTP, and shuts down gracefully.
func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	log := logging.New(cfg.LogLevel, cfg.AppEnv)
	redacted := cfg.Redacted()
	log.Info("starting gitactionflow backend",
		"app_env", redacted.AppEnv,
		"app_port", redacted.AppPort,
		"log_level", redacted.LogLevel,
		"auto_migrate", redacted.AutoMigrate,
		"database_url", redacted.DatabaseURL,
	)

	ctx := context.Background()

	if cfg.AutoMigrate {
		log.Info("applying database migrations")
		if err := database.MigrateUp(cfg.DatabaseURL); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
		log.Info("database migrations applied")
	}

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer pool.Close()
	log.Info("database pool ready")

	engine := router.New(router.Dependencies{
		DB:     poolPinger{pool: pool},
		AppEnv: cfg.AppEnv,
	})

	srv := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("http server listening", "addr", cfg.Addr())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Info("shutdown signal received", "signal", sig.String())
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("http shutdown error", "err", err)
		return fmt.Errorf("shutdown: %w", err)
	}

	log.Info("shutdown complete")
	return nil
}

// poolPinger adapts *pgxpool.Pool to handlers.Pinger.
type poolPinger struct {
	pool *pgxpool.Pool
}

func (p poolPinger) Ping(ctx context.Context) error {
	return database.Ping(ctx, p.pool)
}

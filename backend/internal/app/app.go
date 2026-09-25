package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Revati-Firke/gitactionflow/backend/internal/actions"
	"github.com/Revati-Firke/gitactionflow/backend/internal/ai"
	"github.com/Revati-Firke/gitactionflow/backend/internal/auth"
	"github.com/Revati-Firke/gitactionflow/backend/internal/config"
	"github.com/Revati-Firke/gitactionflow/backend/internal/database"
	"github.com/Revati-Firke/gitactionflow/backend/internal/events"
	"github.com/Revati-Firke/gitactionflow/backend/internal/githubapi"
	"github.com/Revati-Firke/gitactionflow/backend/internal/githuboauth"
	"github.com/Revati-Firke/gitactionflow/backend/internal/http/handlers"
	"github.com/Revati-Firke/gitactionflow/backend/internal/http/middleware"
	"github.com/Revati-Firke/gitactionflow/backend/internal/http/router"
	"github.com/Revati-Firke/gitactionflow/backend/internal/logging"
	"github.com/Revati-Firke/gitactionflow/backend/internal/reposervice"
	rulespkg "github.com/Revati-Firke/gitactionflow/backend/internal/rules"
	"github.com/Revati-Firke/gitactionflow/backend/internal/slack"
	"github.com/Revati-Firke/gitactionflow/backend/internal/store"
	"github.com/Revati-Firke/gitactionflow/backend/internal/webhook"
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
		"frontend_url", redacted.FrontendURL,
		"oauth_redirect", redacted.GitHubOAuthRedirectURL,
		"cookie_secure", redacted.CookieSecure,
		"cookie_samesite", redacted.CookieSameSite,
		"event_worker_enabled", cfg.EventWorkerEnabled,
		"event_max_retries", cfg.EventMaxRetries,
		"action_max_retries", cfg.ActionMaxRetries,
		"slack_configured", cfg.SlackWebhookURL != "",
		"ai_enabled", cfg.AIEnabled,
		"ai_provider", cfg.AIProvider,
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

	users := store.NewUsers(pool)
	sessions := store.NewSessions(pool)
	states := store.NewOAuthStates(pool)
	repos := store.NewRepositories(pool)
	webhookEvents := store.NewWebhookEvents(pool)
	ruleStore := store.NewRules(pool)
	actionStore := store.NewActions(pool)
	tokenKey := auth.DeriveKey(cfg.SessionSecret)
	ghAPI := githubapi.New(nil)
	slackClient := slack.New(cfg.SlackWebhookURL, nil)

	gh := githuboauth.New(githuboauth.Config{
		ClientID:     cfg.GitHubClientID,
		ClientSecret: cfg.GitHubClientSecret,
		RedirectURL:  cfg.GitHubOAuthRedirectURL,
	})

	authHandler := &handlers.AuthHandler{
		GitHub:   gh,
		States:   states,
		Users:    users,
		Sessions: sessions,
		TokenKey: tokenKey,
		Cookie: auth.CookieOptions{
			Secure:   cfg.CookieSecure,
			SameSite: cfg.CookieSameSite,
			TTL:      cfg.SessionTTL,
		},
		StateTTL:    cfg.OAuthStateTTL,
		FrontendURL: cfg.FrontendURL,
		Log:         log,
	}

	repoService := &reposervice.Service{
		GitHub:   ghAPI,
		Users:    users,
		Repos:    repos,
		TokenKey: tokenKey,
	}
	repoHandler := &handlers.RepositoryHandler{
		Service: repoService,
		Log:     log,
	}

	ruleService := &rulespkg.Service{
		Rules: ruleStore,
		Repos:  repos,
	}
	rulesHandler := &handlers.RulesHandler{
		Service: ruleService,
		Log:     log,
	}
	activityHandler := &handlers.ActivityHandler{
		Repos:   repos,
		Events:  webhookEvents,
		Actions: actionStore,
		Log:     log,
	}

	webhookHandler := &handlers.GitHubWebhookHandler{
		Service: &webhook.Service{
			Repos:       repos,
			Events:      webhookEvents,
			MaxRetries:  cfg.EventMaxRetries,
		},
		Secret:  cfg.GitHubWebhookSecret,
		MaxBody: cfg.WebhookMaxBodyBytes,
		Log:     log,
	}

	engine := router.New(router.Dependencies{
		DB:          poolPinger{pool: pool},
		AppEnv:      cfg.AppEnv,
		FrontendURL: cfg.FrontendURL,
		Auth:        authHandler,
		Repos:       repoHandler,
		Rules:       rulesHandler,
		Activity:    activityHandler,
		Webhooks:    webhookHandler,
		AuthMW: middleware.AuthDeps{
			Sessions: sessions,
			Users:    users,
			Log:      log,
		},
		Log: log,
	})

	srv := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	aiClient := ai.New(ai.Config{
		Enabled:  cfg.AIEnabled,
		Provider: cfg.AIProvider,
		APIKey:   cfg.AIAPIKey,
	})

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	var workerWG sync.WaitGroup
	if cfg.EventWorkerEnabled {
		executor := &actions.Executor{
			Actions:     actionStore,
			Repos:        repos,
			Users:       users,
			TokenKey:    tokenKey,
			GitHub:      ghAPI,
			Slack:       slackClient,
			AI:          aiClient,
			MaxAttempts: cfg.ActionMaxRetries,
			Log:         log,
		}
		worker := &events.Worker{
			Queue: webhookEvents,
			Pipeline: &events.Processor{
				Repos:    repos,
				Rules:   ruleStore,
				Actions: executor,
				Log:     log,
			},
			PollInterval: cfg.EventWorkerPollInterval,
			Lease:        cfg.EventProcessingLease,
			Log:          log,
		}
		workerWG.Add(1)
		go func() {
			defer workerWG.Done()
			worker.Run(workerCtx)
		}()
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
			workerCancel()
			workerWG.Wait()
			return fmt.Errorf("http server: %w", err)
		}
	}

	workerCancel()
	workerWG.Wait()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("http shutdown error", "err", err)
		return fmt.Errorf("shutdown: %w", err)
	}

	log.Info("shutdown complete")
	return nil
}

type poolPinger struct {
	pool *pgxpool.Pool
}

func (p poolPinger) Ping(ctx context.Context) error {
	return database.Ping(ctx, p.pool)
}

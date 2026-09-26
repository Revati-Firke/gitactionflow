package router

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Revati-Firke/gitactionflow/backend/internal/http/handlers"
	"github.com/Revati-Firke/gitactionflow/backend/internal/http/middleware"
)

// Dependencies are injected into the HTTP router.
type Dependencies struct {
	DB          handlers.Pinger
	AppEnv      string
	FrontendURL string
	Auth        *handlers.AuthHandler
	Repos       *handlers.RepositoryHandler
	Rules       *handlers.RulesHandler
	Activity    *handlers.ActivityHandler
	Webhooks    *handlers.GitHubWebhookHandler
	AuthMW      middleware.AuthDeps
	Log         *slog.Logger
}

// New builds the Gin engine with foundation, auth, repository, and webhook routes.
func New(deps Dependencies) *gin.Engine {
	if deps.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.SecurityHeaders())
	r.Use(gin.Logger())
	r.Use(middleware.CORS(deps.FrontendURL))

	authLimit := middleware.NewRateLimiter(2*time.Second, 20)
	apiLimit := middleware.NewRateLimiter(time.Second, 60)

	r.GET("/health", handlers.Health)
	r.GET("/ready", handlers.Ready(deps.DB))

	if deps.Webhooks != nil {
		// Public endpoint — authenticated by X-Hub-Signature-256, not session cookies.
		// Intentionally not rate-limited: GitHub retries must not be dropped by abuse controls.
		r.POST("/webhooks/github", deps.Webhooks.HandlePOST)
	}

	if deps.Auth != nil {
		r.GET("/auth/github", authLimit.Limit(), deps.Auth.StartGitHub)
		r.GET("/auth/github/callback", authLimit.Limit(), deps.Auth.CallbackGitHub)
		r.POST("/auth/logout", middleware.CSRFOrigin(deps.FrontendURL), apiLimit.Limit(), deps.Auth.Logout)

		api := r.Group("/api")
		api.Use(middleware.CSRFOrigin(deps.FrontendURL))
		api.Use(apiLimit.Limit())
		api.Use(middleware.RequireAuth(deps.AuthMW))
		api.GET("/me", handlers.Me)

		if deps.Repos != nil {
			api.GET("/github/repositories", deps.Repos.ListGitHubRepositories)
			api.GET("/repository", deps.Repos.GetConnected)
			api.POST("/repository", deps.Repos.Connect)
			api.DELETE("/repository", deps.Repos.Disconnect)
		}
		if deps.Rules != nil {
			api.GET("/rules", deps.Rules.List)
			api.POST("/rules", deps.Rules.Create)
			api.GET("/rules/:id", deps.Rules.Get)
			api.PUT("/rules/:id", deps.Rules.Update)
			api.DELETE("/rules/:id", deps.Rules.Delete)
		}
		if deps.Activity != nil {
			api.GET("/events", deps.Activity.ListEvents)
			api.GET("/actions", deps.Activity.ListActions)
		}
	}

	return r
}

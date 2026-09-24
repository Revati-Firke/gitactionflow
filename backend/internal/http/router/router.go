package router

import (
	"log/slog"

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
	AuthMW      middleware.AuthDeps
	Log         *slog.Logger
}

// New builds the Gin engine with foundation, auth, and repository routes.
func New(deps Dependencies) *gin.Engine {
	if deps.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(middleware.CORS(deps.FrontendURL))

	r.GET("/health", handlers.Health)
	r.GET("/ready", handlers.Ready(deps.DB))

	if deps.Auth != nil {
		r.GET("/auth/github", deps.Auth.StartGitHub)
		r.GET("/auth/github/callback", deps.Auth.CallbackGitHub)
		r.POST("/auth/logout", deps.Auth.Logout)

		api := r.Group("/api")
		api.Use(middleware.RequireAuth(deps.AuthMW))
		api.GET("/me", handlers.Me)

		if deps.Repos != nil {
			api.GET("/github/repositories", deps.Repos.ListGitHubRepositories)
			api.GET("/repository", deps.Repos.GetConnected)
			api.POST("/repository", deps.Repos.Connect)
			api.DELETE("/repository", deps.Repos.Disconnect)
		}
	}

	return r
}

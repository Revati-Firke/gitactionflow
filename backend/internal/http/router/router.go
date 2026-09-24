package router

import (
	"github.com/gin-gonic/gin"

	"github.com/Revati-Firke/gitactionflow/backend/internal/http/handlers"
)

// Dependencies are injected into the HTTP router.
type Dependencies struct {
	DB     handlers.Pinger
	AppEnv string
}

// New builds the Gin engine with foundation routes.
func New(deps Dependencies) *gin.Engine {
	if deps.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	r.GET("/health", handlers.Health)
	r.GET("/ready", handlers.Ready(deps.DB))

	return r
}

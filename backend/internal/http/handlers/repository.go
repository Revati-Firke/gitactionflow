package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Revati-Firke/gitactionflow/backend/internal/auth"
	"github.com/Revati-Firke/gitactionflow/backend/internal/http/response"
	"github.com/Revati-Firke/gitactionflow/backend/internal/reposervice"
	"github.com/Revati-Firke/gitactionflow/backend/internal/store"
)

// RepositoryHandler exposes repository management APIs.
type RepositoryHandler struct {
	Service *reposervice.Service
	Log     *slog.Logger
}

type connectRequest struct {
	GitHubRepositoryID int64 `json:"github_repository_id"`
}

// ListGitHubRepositories handles GET /api/github/repositories.
func (h *RepositoryHandler) ListGitHubRepositories(c *gin.Context) {
	user, ok := auth.UserFromGin(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	repos, err := h.Service.ListGitHubRepositories(c.Request.Context(), user.ID)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"repositories": repos})
}

// GetConnected handles GET /api/repository.
func (h *RepositoryHandler) GetConnected(c *gin.Context) {
	user, ok := auth.UserFromGin(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	repo, err := h.Service.GetConnected(c.Request.Context(), user.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.JSONError(c, http.StatusNotFound, "REPOSITORY_NOT_CONNECTED", "no repository connected")
			return
		}
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"repository": repo})
}

// Connect handles POST /api/repository.
func (h *RepositoryHandler) Connect(c *gin.Context) {
	user, ok := auth.UserFromGin(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	var req connectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	repo, err := h.Service.Connect(c.Request.Context(), user.ID, req.GitHubRepositoryID)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"repository": repo})
}

// Disconnect handles DELETE /api/repository.
func (h *RepositoryHandler) Disconnect(c *gin.Context) {
	user, ok := auth.UserFromGin(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	if err := h.Service.Disconnect(c.Request.Context(), user.ID); err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *RepositoryHandler) writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, reposervice.ErrInvalidInput):
		response.JSONError(c, http.StatusBadRequest, "INVALID_REPOSITORY_ID", "invalid repository id")
	case errors.Is(err, reposervice.ErrAlreadyConnected):
		response.JSONError(c, http.StatusConflict, "REPOSITORY_ALREADY_CONNECTED", "disconnect the current repository before connecting another")
	case errors.Is(err, reposervice.ErrAccessDenied):
		response.JSONError(c, http.StatusForbidden, "REPOSITORY_ACCESS_DENIED", "admin access to the repository is required")
	case errors.Is(err, reposervice.ErrRepositoryNotFound):
		response.JSONError(c, http.StatusNotFound, "REPOSITORY_NOT_FOUND", "repository could not be found")
	case errors.Is(err, reposervice.ErrGitHubUnauthorized), errors.Is(err, reposervice.ErrTokenUnavailable):
		response.JSONError(c, http.StatusUnauthorized, "GITHUB_UNAUTHORIZED", "github authentication failed; sign in again")
	case errors.Is(err, reposervice.ErrGitHubRateLimited):
		response.JSONError(c, http.StatusTooManyRequests, "GITHUB_RATE_LIMITED", "github rate limit exceeded")
	default:
		if h.Log != nil {
			h.Log.Error("repository handler error", "err", err)
		}
		response.Internal(c)
	}
}

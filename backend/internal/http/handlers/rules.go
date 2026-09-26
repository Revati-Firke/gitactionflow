package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Revati-Firke/gitactionflow/backend/internal/auth"
	"github.com/Revati-Firke/gitactionflow/backend/internal/http/response"
	"github.com/Revati-Firke/gitactionflow/backend/internal/rules"
	"github.com/Revati-Firke/gitactionflow/backend/internal/store"
)

// RulesHandler exposes authenticated rule CRUD APIs.
type RulesHandler struct {
	Service *rules.Service
	Log     *slog.Logger
}

type ruleRequest struct {
	Name           string          `json:"name"`
	Enabled        *bool           `json:"enabled"`
	EventType      string          `json:"event_type"`
	Keyword        *string         `json:"keyword"`
	Author         *string         `json:"author"`
	RequiredLabels []string        `json:"required_labels"`
	ActionType     string          `json:"action_type"`
	ActionConfig   json.RawMessage `json:"action_config"`
}

type ruleResponse struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Enabled        bool            `json:"enabled"`
	EventType      string          `json:"event_type"`
	Keyword        *string         `json:"keyword"`
	Author         *string         `json:"author"`
	RequiredLabels []string        `json:"required_labels"`
	ActionType     string          `json:"action_type"`
	ActionConfig   json.RawMessage `json:"action_config"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func toRuleResponse(r store.Rule) ruleResponse {
	labels := r.RequiredLabels
	if labels == nil {
		labels = []string{}
	}
	return ruleResponse{
		ID:             r.ID.String(),
		Name:           r.Name,
		Enabled:        r.Enabled,
		EventType:      r.EventType,
		Keyword:        r.Keyword,
		Author:         r.Author,
		RequiredLabels: labels,
		ActionType:     r.ActionType,
		ActionConfig:   r.ActionConfig,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

func (h *RulesHandler) toInput(req ruleRequest) rules.Input {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	return rules.Input{
		Name:           req.Name,
		Enabled:        enabled,
		EventType:      req.EventType,
		Keyword:        req.Keyword,
		Author:         req.Author,
		RequiredLabels: req.RequiredLabels,
		ActionType:     req.ActionType,
		ActionConfig:   req.ActionConfig,
	}
}

// List handles GET /api/rules.
func (h *RulesHandler) List(c *gin.Context) {
	user, ok := auth.UserFromGin(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	list, err := h.Service.List(c.Request.Context(), user.ID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	out := make([]ruleResponse, 0, len(list))
	for _, r := range list {
		out = append(out, toRuleResponse(r))
	}
	c.JSON(http.StatusOK, gin.H{"rules": out})
}

// Create handles POST /api/rules.
func (h *RulesHandler) Create(c *gin.Context) {
	user, ok := auth.UserFromGin(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	var req ruleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	created, err := h.Service.Create(c.Request.Context(), user.ID, h.toInput(req))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"rule": toRuleResponse(created)})
}

// Get handles GET /api/rules/:id.
func (h *RulesHandler) Get(c *gin.Context) {
	user, ok := auth.UserFromGin(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.JSONError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid rule id")
		return
	}
	r, err := h.Service.Get(c.Request.Context(), user.ID, id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"rule": toRuleResponse(r)})
}

// Update handles PUT /api/rules/:id.
func (h *RulesHandler) Update(c *gin.Context) {
	user, ok := auth.UserFromGin(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.JSONError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid rule id")
		return
	}
	var req ruleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	updated, err := h.Service.Update(c.Request.Context(), user.ID, id, h.toInput(req))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"rule": toRuleResponse(updated)})
}

// Delete handles DELETE /api/rules/:id.
func (h *RulesHandler) Delete(c *gin.Context) {
	user, ok := auth.UserFromGin(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.JSONError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid rule id")
		return
	}
	if err := h.Service.Delete(c.Request.Context(), user.ID, id); err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *RulesHandler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, rules.ErrInvalidInput):
		response.JSONError(c, http.StatusBadRequest, "INVALID_RULE", err.Error())
	case errors.Is(err, rules.ErrNoRepository):
		response.JSONError(c, http.StatusNotFound, "REPOSITORY_NOT_CONNECTED", "connect a repository before managing rules")
	case errors.Is(err, rules.ErrNotFound):
		response.JSONError(c, http.StatusNotFound, "RULE_NOT_FOUND", "rule not found")
	default:
		if h.Log != nil {
			h.Log.Error("rules handler error", "err", err)
		}
		response.Internal(c)
	}
}

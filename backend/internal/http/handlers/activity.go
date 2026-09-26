package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Revati-Firke/gitactionflow/backend/internal/auth"
	"github.com/Revati-Firke/gitactionflow/backend/internal/http/response"
	"github.com/Revati-Firke/gitactionflow/backend/internal/store"
)

// ActivityHandler serves authenticated event/action history for the dashboard.
type ActivityHandler struct {
	Repos   *store.Repositories
	Events  *store.WebhookEvents
	Actions *store.Actions
	Log     *slog.Logger
}

type eventResponse struct {
	ID           string     `json:"id"`
	EventType    string     `json:"event_type"`
	Action       string     `json:"action"`
	Status       string     `json:"status"`
	DeliveryID   string     `json:"delivery_id"`
	RetryCount   int        `json:"retry_count"`
	LastError    *string    `json:"last_error,omitempty"`
	ReceivedAt   time.Time  `json:"received_at"`
	ProcessedAt  *time.Time `json:"processed_at,omitempty"`
	FailedAt     *time.Time `json:"failed_at,omitempty"`
}

type actionResponse struct {
	ID           string     `json:"id"`
	EventID      string     `json:"event_id"`
	RuleID       *string    `json:"rule_id,omitempty"`
	RuleName     *string    `json:"rule_name,omitempty"`
	ActionType   string     `json:"action_type"`
	Status       string     `json:"status"`
	AttemptCount int        `json:"attempt_count"`
	MaxAttempts  int        `json:"max_attempts"`
	LastError    *string    `json:"last_error,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	FailedAt     *time.Time `json:"failed_at,omitempty"`
}

func parsePageLimit(c *gin.Context) (limit, offset int) {
	limit = 20
	offset = 0
	if raw := c.Query("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			limit = v
		}
	}
	if raw := c.Query("page"); raw != "" {
		if page, err := strconv.Atoi(raw); err == nil && page > 1 {
			offset = (page - 1) * limit
		}
	}
	if raw := c.Query("offset"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v >= 0 {
			offset = v
		}
	}
	return limit, offset
}

func (h *ActivityHandler) connectedRepo(c *gin.Context, userID uuid.UUID) (store.Repository, bool) {
	repo, err := h.Repos.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.JSONError(c, http.StatusNotFound, "REPOSITORY_NOT_CONNECTED", "no repository connected")
			return store.Repository{}, false
		}
		if h.Log != nil {
			h.Log.Error("load repository for activity", "err", err)
		}
		response.JSONError(c, http.StatusInternalServerError, "INTERNAL", "failed to load repository")
		return store.Repository{}, false
	}
	return repo, true
}

// ListEvents handles GET /api/events — scoped to the connected repository (no raw payloads).
func (h *ActivityHandler) ListEvents(c *gin.Context) {
	user, ok := auth.UserFromGin(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	repo, ok := h.connectedRepo(c, user.ID)
	if !ok {
		return
	}
	limit, offset := parsePageLimit(c)
	list, err := h.Events.ListRecentByRepository(c.Request.Context(), repo.ID, limit, offset)
	if err != nil {
		if h.Log != nil {
			h.Log.Error("list events", "err", err)
		}
		response.JSONError(c, http.StatusInternalServerError, "INTERNAL", "failed to list events")
		return
	}
	out := make([]eventResponse, 0, len(list))
	for _, e := range list {
		out = append(out, eventResponse{
			ID:          e.ID.String(),
			EventType:   e.EventType,
			Action:      e.Action,
			Status:      e.Status,
			DeliveryID:  e.DeliveryID,
			RetryCount:  e.RetryCount,
			LastError:   e.LastError,
			ReceivedAt:  e.ReceivedAt,
			ProcessedAt: e.ProcessedAt,
			FailedAt:    e.FailedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"events": out,
		"page": gin.H{
			"limit":  limit,
			"offset": offset,
		},
	})
}

// ListActions handles GET /api/actions — scoped to the connected repository.
func (h *ActivityHandler) ListActions(c *gin.Context) {
	user, ok := auth.UserFromGin(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	repo, ok := h.connectedRepo(c, user.ID)
	if !ok {
		return
	}
	limit, offset := parsePageLimit(c)
	list, err := h.Actions.ListRecentByRepository(c.Request.Context(), repo.ID, limit, offset)
	if err != nil {
		if h.Log != nil {
			h.Log.Error("list actions", "err", err)
		}
		response.JSONError(c, http.StatusInternalServerError, "INTERNAL", "failed to list actions")
		return
	}
	out := make([]actionResponse, 0, len(list))
	for _, row := range list {
		a := row.Action
		var ruleID *string
		if a.RuleID != nil {
			s := a.RuleID.String()
			ruleID = &s
		}
		out = append(out, actionResponse{
			ID:           a.ID.String(),
			EventID:      a.EventID.String(),
			RuleID:       ruleID,
			RuleName:     row.RuleName,
			ActionType:   a.ActionType,
			Status:       a.Status,
			AttemptCount: a.AttemptCount,
			MaxAttempts:  a.MaxAttempts,
			LastError:    a.LastError,
			CreatedAt:    a.CreatedAt,
			CompletedAt:  a.CompletedAt,
			FailedAt:     a.FailedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"actions": out,
		"page": gin.H{
			"limit":  limit,
			"offset": offset,
		},
	})
}

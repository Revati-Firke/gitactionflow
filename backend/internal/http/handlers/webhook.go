package handlers

import (
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Revati-Firke/gitactionflow/backend/internal/http/response"
	"github.com/Revati-Firke/gitactionflow/backend/internal/webhook"
)

// GitHubWebhookHandler receives GitHub webhook deliveries.
type GitHubWebhookHandler struct {
	Service *webhook.Service
	Secret  string
	MaxBody int64
	Log     *slog.Logger
}

// HandlePOST processes POST /webhooks/github.
func (h *GitHubWebhookHandler) HandlePOST(c *gin.Context) {
	maxBody := h.MaxBody
	if maxBody <= 0 {
		maxBody = 1 << 20 // 1 MiB default
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBody)
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			response.JSONError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body too large")
			return
		}
		response.JSONError(c, http.StatusBadRequest, "BAD_REQUEST", "unable to read request body")
		return
	}

	sig := c.GetHeader("X-Hub-Signature-256")
	if err := webhook.VerifySignature256(h.Secret, body, sig); err != nil {
		// Do not reveal verification details.
		response.JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid webhook signature")
		return
	}

	eventType := c.GetHeader("X-GitHub-Event")
	deliveryID := c.GetHeader("X-GitHub-Delivery")
	if eventType == "" || deliveryID == "" {
		response.JSONError(c, http.StatusBadRequest, "BAD_REQUEST", "missing required github headers")
		return
	}

	result, err := h.Service.Ingest(c.Request.Context(), webhook.IngestInput{
		DeliveryID: deliveryID,
		EventType:  eventType,
		RawBody:    body,
	})
	if err != nil {
		switch {
		case errors.Is(err, webhook.ErrInvalidPayload):
			response.JSONError(c, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid webhook payload")
		case errors.Is(err, webhook.ErrMissingDeliveryID), errors.Is(err, webhook.ErrMissingEvent):
			response.JSONError(c, http.StatusBadRequest, "BAD_REQUEST", "missing required github headers")
		case errors.Is(err, webhook.ErrPersistFailed):
			if h.Log != nil {
				h.Log.Error("webhook persist failed", "err", err, "delivery_id", deliveryID, "event", eventType)
			}
			response.JSONError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		default:
			if h.Log != nil {
				h.Log.Error("webhook ingest failed", "err", err, "delivery_id", deliveryID, "event", eventType)
			}
			response.Internal(c)
		}
		return
	}

	switch result.Status {
	case "accepted":
		c.JSON(http.StatusOK, gin.H{"status": "accepted"})
	case "already_received":
		c.JSON(http.StatusOK, gin.H{"status": "already_received"})
	case "ignored":
		c.JSON(http.StatusOK, gin.H{"status": "ignored", "reason": result.Reason})
	default:
		c.JSON(http.StatusOK, gin.H{"status": result.Status})
	}
}

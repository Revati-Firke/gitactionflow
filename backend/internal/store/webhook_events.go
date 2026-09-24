package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	WebhookStatusPending    = "pending"
	WebhookStatusProcessing = "processing"
	WebhookStatusProcessed  = "processed"
	WebhookStatusFailed     = "failed"
)

// WebhookEvent is a durably stored GitHub delivery.
type WebhookEvent struct {
	ID           uuid.UUID
	RepositoryID uuid.UUID
	DeliveryID   string
	EventType    string
	Action       string
	Payload      json.RawMessage
	Status       string
	ReceivedAt   time.Time
	ProcessedAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// WebhookEvents persists webhook deliveries.
type WebhookEvents struct {
	pool *pgxpool.Pool
}

func NewWebhookEvents(pool *pgxpool.Pool) *WebhookEvents {
	return &WebhookEvents{pool: pool}
}

// InsertPending stores a new delivery as pending.
// Returns ErrConflict when delivery_id already exists.
func (s *WebhookEvents) InsertPending(ctx context.Context, repositoryID uuid.UUID, deliveryID, eventType, action string, payload json.RawMessage) (WebhookEvent, error) {
	const q = `
INSERT INTO webhook_events (
    repository_id, delivery_id, event_type, action, payload, status, received_at
) VALUES ($1, $2, $3, $4, $5, $6, NOW())
RETURNING id, repository_id, delivery_id, event_type, action, payload, status,
          received_at, processed_at, created_at, updated_at
`
	var e WebhookEvent
	err := s.pool.QueryRow(ctx, q, repositoryID, deliveryID, eventType, action, payload, WebhookStatusPending).Scan(
		&e.ID, &e.RepositoryID, &e.DeliveryID, &e.EventType, &e.Action, &e.Payload, &e.Status,
		&e.ReceivedAt, &e.ProcessedAt, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return WebhookEvent{}, ErrConflict
		}
		return WebhookEvent{}, fmt.Errorf("insert webhook event: %w", err)
	}
	return e, nil
}

// GetByDeliveryID loads an event by GitHub delivery id.
func (s *WebhookEvents) GetByDeliveryID(ctx context.Context, deliveryID string) (WebhookEvent, error) {
	const q = `
SELECT id, repository_id, delivery_id, event_type, action, payload, status,
       received_at, processed_at, created_at, updated_at
FROM webhook_events WHERE delivery_id = $1
`
	var e WebhookEvent
	err := s.pool.QueryRow(ctx, q, deliveryID).Scan(
		&e.ID, &e.RepositoryID, &e.DeliveryID, &e.EventType, &e.Action, &e.Payload, &e.Status,
		&e.ReceivedAt, &e.ProcessedAt, &e.CreatedAt, &e.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return WebhookEvent{}, ErrNotFound
	}
	if err != nil {
		return WebhookEvent{}, fmt.Errorf("get webhook event: %w", err)
	}
	return e, nil
}

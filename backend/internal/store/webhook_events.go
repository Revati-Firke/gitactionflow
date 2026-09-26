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

// WebhookEvent is a durably stored GitHub delivery with processing state.
type WebhookEvent struct {
	ID           uuid.UUID
	RepositoryID uuid.UUID
	DeliveryID   string
	EventType    string
	Action       string
	Payload      json.RawMessage
	Status       string
	RetryCount   int
	MaxRetries   int
	NextRetryAt  *time.Time
	LastError    *string
	LockedAt     *time.Time
	ProcessedAt  *time.Time
	FailedAt     *time.Time
	ReceivedAt   time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// WebhookEvents persists webhook deliveries and processing state.
type WebhookEvents struct {
	pool *pgxpool.Pool
}

func NewWebhookEvents(pool *pgxpool.Pool) *WebhookEvents {
	return &WebhookEvents{pool: pool}
}

const webhookEventColumns = `
id, repository_id, delivery_id, event_type, action, payload, status,
retry_count, max_retries, next_retry_at, last_error, locked_at,
processed_at, failed_at, received_at, created_at, updated_at`

func scanWebhookEvent(row pgx.Row) (WebhookEvent, error) {
	var e WebhookEvent
	err := row.Scan(
		&e.ID, &e.RepositoryID, &e.DeliveryID, &e.EventType, &e.Action, &e.Payload, &e.Status,
		&e.RetryCount, &e.MaxRetries, &e.NextRetryAt, &e.LastError, &e.LockedAt,
		&e.ProcessedAt, &e.FailedAt, &e.ReceivedAt, &e.CreatedAt, &e.UpdatedAt,
	)
	return e, err
}

// InsertPending stores a new delivery as pending.
// Returns ErrConflict when delivery_id already exists.
func (s *WebhookEvents) InsertPending(ctx context.Context, repositoryID uuid.UUID, deliveryID, eventType, action string, payload json.RawMessage) (WebhookEvent, error) {
	return s.InsertPendingWithMaxRetries(ctx, repositoryID, deliveryID, eventType, action, payload, 3)
}

// InsertPendingWithMaxRetries stores a pending delivery with an explicit max_retries.
func (s *WebhookEvents) InsertPendingWithMaxRetries(ctx context.Context, repositoryID uuid.UUID, deliveryID, eventType, action string, payload json.RawMessage, maxRetries int) (WebhookEvent, error) {
	if maxRetries < 0 {
		maxRetries = 0
	}
	const q = `
INSERT INTO webhook_events (
    repository_id, delivery_id, event_type, action, payload, status, max_retries, received_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
RETURNING ` + webhookEventColumns
	e, err := scanWebhookEvent(s.pool.QueryRow(ctx, q, repositoryID, deliveryID, eventType, action, payload, WebhookStatusPending, maxRetries))
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
	const q = `SELECT ` + webhookEventColumns + ` FROM webhook_events WHERE delivery_id = $1`
	e, err := scanWebhookEvent(s.pool.QueryRow(ctx, q, deliveryID))
	if errors.Is(err, pgx.ErrNoRows) {
		return WebhookEvent{}, ErrNotFound
	}
	if err != nil {
		return WebhookEvent{}, fmt.Errorf("get webhook event: %w", err)
	}
	return e, nil
}

// GetByID loads an event by primary key.
func (s *WebhookEvents) GetByID(ctx context.Context, id uuid.UUID) (WebhookEvent, error) {
	const q = `SELECT ` + webhookEventColumns + ` FROM webhook_events WHERE id = $1`
	e, err := scanWebhookEvent(s.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return WebhookEvent{}, ErrNotFound
	}
	if err != nil {
		return WebhookEvent{}, fmt.Errorf("get webhook event by id: %w", err)
	}
	return e, nil
}

// ClaimNext atomically claims one eligible event for processing using
// FOR UPDATE SKIP LOCKED. Eligible rows are:
//   - pending with next_retry_at NULL or <= now
//   - processing with locked_at older than lease (stale reclaim)
//
// Stale reclaim increments retry_count. If retries are exhausted, the row is
// marked failed and ClaimNext continues looking for another event.
func (s *WebhookEvents) ClaimNext(ctx context.Context, lease time.Duration) (WebhookEvent, error) {
	if lease <= 0 {
		lease = time.Minute
	}
	leaseInterval := fmt.Sprintf("%f seconds", lease.Seconds())

	for i := 0; i < 32; i++ {
		claimed, exhausted, err := s.claimOne(ctx, leaseInterval)
		if err != nil {
			return WebhookEvent{}, err
		}
		if exhausted {
			continue
		}
		return claimed, nil
	}
	return WebhookEvent{}, ErrNotFound
}

// claimOne attempts a single claim. exhausted=true means a stale event was
// marked failed and the caller should try again.
func (s *WebhookEvents) claimOne(ctx context.Context, leaseInterval string) (WebhookEvent, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return WebhookEvent{}, false, fmt.Errorf("begin claim tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const selectQ = `
SELECT ` + webhookEventColumns + `
FROM webhook_events
WHERE
    (
        status = 'pending'
        AND (next_retry_at IS NULL OR next_retry_at <= NOW())
    )
    OR
    (
        status = 'processing'
        AND locked_at IS NOT NULL
        AND locked_at < NOW() - $1::interval
    )
ORDER BY created_at ASC
FOR UPDATE SKIP LOCKED
LIMIT 1
`
	e, err := scanWebhookEvent(tx.QueryRow(ctx, selectQ, leaseInterval))
	if errors.Is(err, pgx.ErrNoRows) {
		return WebhookEvent{}, false, ErrNotFound
	}
	if err != nil {
		return WebhookEvent{}, false, fmt.Errorf("select claimable event: %w", err)
	}

	wasStale := e.Status == WebhookStatusProcessing
	retryCount := e.RetryCount
	lastError := e.LastError
	if wasStale {
		retryCount++
		msg := "reclaimed stale processing lease"
		lastError = &msg
		if retryCount >= e.MaxRetries {
			const failQ = `
UPDATE webhook_events SET
    status = 'failed',
    retry_count = $2,
    last_error = $3,
    failed_at = NOW(),
    locked_at = NULL,
    next_retry_at = NULL,
    updated_at = NOW()
WHERE id = $1
`
			if _, err := tx.Exec(ctx, failQ, e.ID, retryCount, lastError); err != nil {
				return WebhookEvent{}, false, fmt.Errorf("fail stale event: %w", err)
			}
			if err := tx.Commit(ctx); err != nil {
				return WebhookEvent{}, false, fmt.Errorf("commit stale fail: %w", err)
			}
			return WebhookEvent{}, true, nil
		}
	}

	const claimQ = `
UPDATE webhook_events SET
    status = 'processing',
    locked_at = NOW(),
    retry_count = $2,
    last_error = $3,
    next_retry_at = NULL,
    updated_at = NOW()
WHERE id = $1
RETURNING ` + webhookEventColumns
	claimed, err := scanWebhookEvent(tx.QueryRow(ctx, claimQ, e.ID, retryCount, lastError))
	if err != nil {
		return WebhookEvent{}, false, fmt.Errorf("claim event: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return WebhookEvent{}, false, fmt.Errorf("commit claim: %w", err)
	}
	return claimed, false, nil
}

// MarkProcessed sets status=processed after successful pipeline validation.
func (s *WebhookEvents) MarkProcessed(ctx context.Context, id uuid.UUID) (WebhookEvent, error) {
	const q = `
UPDATE webhook_events SET
    status = 'processed',
    processed_at = NOW(),
    locked_at = NULL,
    next_retry_at = NULL,
    last_error = NULL,
    updated_at = NOW()
WHERE id = $1 AND status = 'processing'
RETURNING ` + webhookEventColumns
	e, err := scanWebhookEvent(s.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return WebhookEvent{}, ErrNotFound
	}
	if err != nil {
		return WebhookEvent{}, fmt.Errorf("mark processed: %w", err)
	}
	return e, nil
}

// ScheduleRetry increments retry_count, stores last_error, and either returns
// the event to pending with next_retry_at or marks it failed when exhausted.
func (s *WebhookEvents) ScheduleRetry(ctx context.Context, id uuid.UUID, errMsg string, backoff time.Duration) (WebhookEvent, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return WebhookEvent{}, fmt.Errorf("begin retry tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const selectQ = `SELECT ` + webhookEventColumns + ` FROM webhook_events WHERE id = $1 FOR UPDATE`
	e, err := scanWebhookEvent(tx.QueryRow(ctx, selectQ, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return WebhookEvent{}, ErrNotFound
	}
	if err != nil {
		return WebhookEvent{}, fmt.Errorf("load for retry: %w", err)
	}

	retryCount := e.RetryCount + 1
	if retryCount >= e.MaxRetries {
		const failQ = `
UPDATE webhook_events SET
    status = 'failed',
    retry_count = $2,
    last_error = $3,
    failed_at = NOW(),
    locked_at = NULL,
    next_retry_at = NULL,
    updated_at = NOW()
WHERE id = $1
RETURNING ` + webhookEventColumns
		failed, err := scanWebhookEvent(tx.QueryRow(ctx, failQ, id, retryCount, errMsg))
		if err != nil {
			return WebhookEvent{}, fmt.Errorf("mark failed: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return WebhookEvent{}, fmt.Errorf("commit failed: %w", err)
		}
		return failed, nil
	}

	if backoff < 0 {
		backoff = 0
	}
	next := time.Now().UTC().Add(backoff)
	const retryQ = `
UPDATE webhook_events SET
    status = 'pending',
    retry_count = $2,
    last_error = $3,
    next_retry_at = $4,
    locked_at = NULL,
    updated_at = NOW()
WHERE id = $1
RETURNING ` + webhookEventColumns
	pending, err := scanWebhookEvent(tx.QueryRow(ctx, retryQ, id, retryCount, errMsg, next))
	if err != nil {
		return WebhookEvent{}, fmt.Errorf("schedule retry: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return WebhookEvent{}, fmt.Errorf("commit retry: %w", err)
	}
	return pending, nil
}

// MarkFailedImmediately marks an event failed without further retries (permanent errors).
func (s *WebhookEvents) MarkFailedImmediately(ctx context.Context, id uuid.UUID, errMsg string) (WebhookEvent, error) {
	const q = `
UPDATE webhook_events SET
    status = 'failed',
    last_error = $2,
    failed_at = NOW(),
    locked_at = NULL,
    next_retry_at = NULL,
    updated_at = NOW()
WHERE id = $1 AND status = 'processing'
RETURNING ` + webhookEventColumns
	e, err := scanWebhookEvent(s.pool.QueryRow(ctx, q, id, errMsg))
	if errors.Is(err, pgx.ErrNoRows) {
		return WebhookEvent{}, ErrNotFound
	}
	if err != nil {
		return WebhookEvent{}, fmt.Errorf("mark failed immediately: %w", err)
	}
	return e, nil
}

// ListRecentByRepository returns newest events for a connected repository (bounded).
func (s *WebhookEvents) ListRecentByRepository(ctx context.Context, repositoryID uuid.UUID, limit, offset int) ([]WebhookEvent, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	const q = `
SELECT ` + webhookEventColumns + `
FROM webhook_events
WHERE repository_id = $1
ORDER BY received_at DESC, id DESC
LIMIT $2 OFFSET $3`
	rows, err := s.pool.Query(ctx, q, repositoryID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list webhook events: %w", err)
	}
	defer rows.Close()
	var out []WebhookEvent
	for rows.Next() {
		e, err := scanWebhookEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

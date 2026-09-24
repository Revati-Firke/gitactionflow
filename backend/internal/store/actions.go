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
	ActionStatusPending    = "pending"
	ActionStatusProcessing = "processing"
	ActionStatusCompleted  = "completed"
	ActionStatusFailed     = "failed"
)

// Action is a durable record of an intended external side effect.
type Action struct {
	ID             uuid.UUID
	EventID        uuid.UUID
	RuleID         *uuid.UUID
	ActionType     string
	ActionConfig   json.RawMessage
	IdempotencyKey string
	Status         string
	AttemptCount   int
	MaxAttempts    int
	NextRetryAt    *time.Time
	LastError      *string
	StartedAt      *time.Time
	CompletedAt    *time.Time
	FailedAt       *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Actions persists action execution state.
type Actions struct {
	pool *pgxpool.Pool
}

func NewActions(pool *pgxpool.Pool) *Actions {
	return &Actions{pool: pool}
}

const actionColumns = `
id, event_id, rule_id, action_type, action_config, idempotency_key, status,
attempt_count, max_attempts, next_retry_at, last_error, started_at, completed_at,
failed_at, created_at, updated_at`

func scanAction(row pgx.Row) (Action, error) {
	var a Action
	err := row.Scan(
		&a.ID, &a.EventID, &a.RuleID, &a.ActionType, &a.ActionConfig, &a.IdempotencyKey, &a.Status,
		&a.AttemptCount, &a.MaxAttempts, &a.NextRetryAt, &a.LastError, &a.StartedAt, &a.CompletedAt,
		&a.FailedAt, &a.CreatedAt, &a.UpdatedAt,
	)
	return a, err
}

// CreateOrGet inserts a pending action or returns the existing row for the idempotency key.
func (s *Actions) CreateOrGet(ctx context.Context, eventID uuid.UUID, ruleID uuid.UUID, actionType string, config json.RawMessage, idempotencyKey string, maxAttempts int) (Action, bool, error) {
	if maxAttempts < 0 {
		maxAttempts = 0
	}
	if len(config) == 0 {
		config = json.RawMessage(`{}`)
	}
	const insertQ = `
INSERT INTO actions (
    event_id, rule_id, action_type, action_config, idempotency_key, status, max_attempts
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING ` + actionColumns
	a, err := scanAction(s.pool.QueryRow(ctx, insertQ, eventID, ruleID, actionType, config, idempotencyKey, ActionStatusPending, maxAttempts))
	if err == nil {
		return a, true, nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		existing, gerr := s.GetByIdempotencyKey(ctx, idempotencyKey)
		if gerr != nil {
			return Action{}, false, gerr
		}
		return existing, false, nil
	}
	return Action{}, false, fmt.Errorf("create action: %w", err)
}

// GetByIdempotencyKey loads an action by unique key.
func (s *Actions) GetByIdempotencyKey(ctx context.Context, key string) (Action, error) {
	const q = `SELECT ` + actionColumns + ` FROM actions WHERE idempotency_key = $1`
	a, err := scanAction(s.pool.QueryRow(ctx, q, key))
	if errors.Is(err, pgx.ErrNoRows) {
		return Action{}, ErrNotFound
	}
	if err != nil {
		return Action{}, fmt.Errorf("get action: %w", err)
	}
	return a, nil
}

// ListByEventID returns all actions for an event, oldest first.
func (s *Actions) ListByEventID(ctx context.Context, eventID uuid.UUID) ([]Action, error) {
	const q = `SELECT ` + actionColumns + ` FROM actions WHERE event_id = $1 ORDER BY created_at ASC, id ASC`
	rows, err := s.pool.Query(ctx, q, eventID)
	if err != nil {
		return nil, fmt.Errorf("list actions: %w", err)
	}
	defer rows.Close()
	var out []Action
	for rows.Next() {
		a, err := scanAction(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// MarkProcessing claims a pending action for execution when ready.
func (s *Actions) MarkProcessing(ctx context.Context, id uuid.UUID) (Action, error) {
	const q = `
UPDATE actions SET
    status = 'processing',
    started_at = COALESCE(started_at, NOW()),
    next_retry_at = NULL,
    updated_at = NOW()
WHERE id = $1
  AND status = 'pending'
  AND (next_retry_at IS NULL OR next_retry_at <= NOW())
RETURNING ` + actionColumns
	a, err := scanAction(s.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Action{}, ErrNotFound
	}
	if err != nil {
		return Action{}, fmt.Errorf("mark processing: %w", err)
	}
	return a, nil
}

// MarkCompleted records successful external execution.
func (s *Actions) MarkCompleted(ctx context.Context, id uuid.UUID) (Action, error) {
	const q = `
UPDATE actions SET
    status = 'completed',
    completed_at = NOW(),
    last_error = NULL,
    next_retry_at = NULL,
    updated_at = NOW()
WHERE id = $1 AND status = 'processing'
RETURNING ` + actionColumns
	a, err := scanAction(s.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Action{}, ErrNotFound
	}
	if err != nil {
		return Action{}, fmt.Errorf("mark completed: %w", err)
	}
	return a, nil
}

// MarkFailedImmediately marks permanent failure without further retries.
func (s *Actions) MarkFailedImmediately(ctx context.Context, id uuid.UUID, errMsg string) (Action, error) {
	const q = `
UPDATE actions SET
    status = 'failed',
    attempt_count = attempt_count + 1,
    last_error = $2,
    failed_at = NOW(),
    next_retry_at = NULL,
    updated_at = NOW()
WHERE id = $1 AND status = 'processing'
RETURNING ` + actionColumns
	a, err := scanAction(s.pool.QueryRow(ctx, q, id, errMsg))
	if errors.Is(err, pgx.ErrNoRows) {
		return Action{}, ErrNotFound
	}
	if err != nil {
		return Action{}, fmt.Errorf("mark failed: %w", err)
	}
	return a, nil
}

// ScheduleRetry increments attempts and either requeues pending or marks failed.
func (s *Actions) ScheduleRetry(ctx context.Context, id uuid.UUID, errMsg string, backoff time.Duration) (Action, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Action{}, err
	}
	defer tx.Rollback(ctx)

	const sel = `SELECT ` + actionColumns + ` FROM actions WHERE id = $1 FOR UPDATE`
	a, err := scanAction(tx.QueryRow(ctx, sel, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Action{}, ErrNotFound
	}
	if err != nil {
		return Action{}, err
	}

	attempts := a.AttemptCount + 1
	if attempts >= a.MaxAttempts {
		const failQ = `
UPDATE actions SET
    status = 'failed',
    attempt_count = $2,
    last_error = $3,
    failed_at = NOW(),
    next_retry_at = NULL,
    updated_at = NOW()
WHERE id = $1
RETURNING ` + actionColumns
		failed, err := scanAction(tx.QueryRow(ctx, failQ, id, attempts, errMsg))
		if err != nil {
			return Action{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return Action{}, err
		}
		return failed, nil
	}

	if backoff < 0 {
		backoff = 0
	}
	next := time.Now().UTC().Add(backoff)
	const retryQ = `
UPDATE actions SET
    status = 'pending',
    attempt_count = $2,
    last_error = $3,
    next_retry_at = $4,
    updated_at = NOW()
WHERE id = $1
RETURNING ` + actionColumns
	pending, err := scanAction(tx.QueryRow(ctx, retryQ, id, attempts, errMsg, next))
	if err != nil {
		return Action{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Action{}, err
	}
	return pending, nil
}

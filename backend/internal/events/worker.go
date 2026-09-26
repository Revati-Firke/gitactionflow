package events

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/Revati-Firke/gitactionflow/backend/internal/store"
)

// EventQueue claims and updates processing state for webhook events.
type EventQueue interface {
	ClaimNext(ctx context.Context, lease time.Duration) (store.WebhookEvent, error)
	MarkProcessed(ctx context.Context, id uuid.UUID) (store.WebhookEvent, error)
	ScheduleRetry(ctx context.Context, id uuid.UUID, errMsg string, backoff time.Duration) (store.WebhookEvent, error)
	MarkFailedImmediately(ctx context.Context, id uuid.UUID, errMsg string) (store.WebhookEvent, error)
}

// Pipeline processes a claimed event (validation + future rules/actions).
type Pipeline interface {
	Process(ctx context.Context, e store.WebhookEvent) error
}

// Worker polls PostgreSQL for pending/stale events and runs the pipeline.
type Worker struct {
	Queue         EventQueue
	Pipeline      Pipeline
	PollInterval  time.Duration
	Lease         time.Duration
	Log           *slog.Logger
	MaxPerTick    int // drain up to N events per poll; 0 → 10
}

// Run blocks until ctx is cancelled. Safe to call from a goroutine.
func (w *Worker) Run(ctx context.Context) {
	interval := w.PollInterval
	if interval <= 0 {
		interval = 2 * time.Second
	}
	lease := w.Lease
	if lease <= 0 {
		lease = time.Minute
	}
	log := w.Log
	if log == nil {
		log = slog.Default()
	}

	log.Info("worker started", "poll_interval", interval.String(), "lease", lease.String())
	defer log.Info("worker stopped")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Opportunistic first tick.
	w.drain(ctx, lease, log)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.drain(ctx, lease, log)
		}
	}
}

func (w *Worker) drain(ctx context.Context, lease time.Duration, log *slog.Logger) {
	limit := w.MaxPerTick
	if limit <= 0 {
		limit = 10
	}
	for i := 0; i < limit; i++ {
		if ctx.Err() != nil {
			return
		}
		ok := w.processOne(ctx, lease, log)
		if !ok {
			return
		}
	}
}

// ProcessOneForTest runs a single claim+process cycle (tests only).
func (w *Worker) ProcessOneForTest(ctx context.Context, lease time.Duration) bool {
	log := w.Log
	if log == nil {
		log = slog.Default()
	}
	if lease <= 0 {
		lease = time.Minute
	}
	return w.processOne(ctx, lease, log)
}

// processOne claims and processes a single event. Returns false when idle.
func (w *Worker) processOne(ctx context.Context, lease time.Duration, log *slog.Logger) bool {
	ev, err := w.Queue.ClaimNext(ctx, lease)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return false
		}
		log.Error("event claim failed", "err", err)
		return false
	}

	attrs := EventLogAttrs(ev)
	log.Info("event claimed", attrs...)
	log.Info("event processing started", attrs...)

	if err := w.Pipeline.Process(ctx, ev); err != nil {
		w.handleFailure(ctx, ev, err, log)
		return true
	}

	if _, err := w.Queue.MarkProcessed(ctx, ev.ID); err != nil {
		log.Error("event mark processed failed", append(attrs, "err", err)...)
		return true
	}
	log.Info("event processed successfully", attrs...)
	return true
}

func (w *Worker) handleFailure(ctx context.Context, ev store.WebhookEvent, procErr error, log *slog.Logger) {
	attrs := EventLogAttrs(ev)
	msg := procErr.Error()
	log.Error("event processing failed", append(attrs, "err", msg)...)

	if IsPermanent(procErr) {
		updated, err := w.Queue.MarkFailedImmediately(ctx, ev.ID, msg)
		if err != nil {
			log.Error("event mark failed immediately error", append(attrs, "err", err)...)
			return
		}
		log.Info("event marked failed", append(EventLogAttrs(updated), "reason", "permanent")...)
		return
	}

	// Preview backoff for the upcoming retry_count (current + 1).
	nextCount := ev.RetryCount + 1
	backoff := RetryBackoff(nextCount)
	updated, err := w.Queue.ScheduleRetry(ctx, ev.ID, msg, backoff)
	if err != nil {
		log.Error("event schedule retry failed", append(attrs, "err", err)...)
		return
	}
	if updated.Status == store.WebhookStatusFailed {
		log.Info("event marked failed", append(EventLogAttrs(updated), "reason", "retries_exhausted")...)
		return
	}
	log.Info("event scheduled for retry", append(EventLogAttrs(updated), "next_retry_at", updated.NextRetryAt, "backoff", backoff.String())...)
}

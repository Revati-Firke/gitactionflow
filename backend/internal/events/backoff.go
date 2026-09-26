package events

import (
	"time"

	"github.com/Revati-Firke/gitactionflow/backend/internal/store"
)

// RetryBackoff returns delays after a failure for the new retry_count:
// 1 → 1m, 2 → 5m, 3+ → 15m.
func RetryBackoff(retryCountAfterFailure int) time.Duration {
	switch {
	case retryCountAfterFailure <= 1:
		return time.Minute
	case retryCountAfterFailure == 2:
		return 5 * time.Minute
	default:
		return 15 * time.Minute
	}
}

// EventLogAttrs are structured log fields for an event (no payload/secrets).
func EventLogAttrs(e store.WebhookEvent) []any {
	return []any{
		"event_id", e.ID.String(),
		"delivery_id", e.DeliveryID,
		"event_type", e.EventType,
		"repository_id", e.RepositoryID.String(),
		"retry_count", e.RetryCount,
	}
}

package events

import "errors"

// Permanent validation / data errors should not be retried.
var (
	ErrInvalidEvent     = errors.New("invalid persisted event")
	ErrUnsupportedEvent = errors.New("unsupported event type")
	ErrRepoMismatch     = errors.New("payload repository does not match connected repository")

	// ErrActionsIncomplete means matched actions still pending/retryable.
	ErrActionsIncomplete = errors.New("actions incomplete")
	// ErrActionsFailed means required actions permanently failed.
	ErrActionsFailed = errors.New("required actions failed")
)

// IsPermanent reports whether err should skip retries.
func IsPermanent(err error) bool {
	return errors.Is(err, ErrInvalidEvent) ||
		errors.Is(err, ErrUnsupportedEvent) ||
		errors.Is(err, ErrRepoMismatch) ||
		errors.Is(err, ErrActionsFailed)
}

package events

import "errors"

// Permanent validation / data errors should not be retried.
var (
	ErrInvalidEvent     = errors.New("invalid persisted event")
	ErrUnsupportedEvent = errors.New("unsupported event type")
	ErrRepoMismatch     = errors.New("payload repository does not match connected repository")
)

// IsPermanent reports whether err should skip retries.
func IsPermanent(err error) bool {
	return errors.Is(err, ErrInvalidEvent) ||
		errors.Is(err, ErrUnsupportedEvent) ||
		errors.Is(err, ErrRepoMismatch)
}

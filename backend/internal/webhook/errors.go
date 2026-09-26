package webhook

import "errors"

var (
	ErrMissingSignature  = errors.New("missing signature")
	ErrInvalidSignature  = errors.New("invalid signature")
	ErrMissingEvent      = errors.New("missing event type")
	ErrMissingDeliveryID = errors.New("missing delivery id")
	ErrUnsupportedEvent  = errors.New("unsupported event")
	ErrUnknownRepository = errors.New("unknown repository")
	ErrInvalidPayload    = errors.New("invalid payload")
	ErrBodyTooLarge      = errors.New("body too large")
	ErrAlreadyReceived   = errors.New("already received")
	ErrPersistFailed     = errors.New("persist failed")
)

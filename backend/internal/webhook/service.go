package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/Revati-Firke/gitactionflow/backend/internal/store"
)

// SupportedEvents are the GitHub X-GitHub-Event values we persist.
var SupportedEvents = map[string]struct{}{
	"issues":       {},
	"pull_request": {},
}

// RepoFinder looks up a connected repository by GitHub id.
type RepoFinder interface {
	GetByGitHubID(ctx context.Context, githubRepoID int64) (store.Repository, error)
}

// EventStore persists webhook deliveries.
type EventStore interface {
	InsertPendingWithMaxRetries(ctx context.Context, repositoryID uuid.UUID, deliveryID, eventType, action string, payload json.RawMessage, maxRetries int) (store.WebhookEvent, error)
}

// Service ingests verified GitHub webhook deliveries.
type Service struct {
	Repos      RepoFinder
	Events     EventStore
	MaxRetries int
}

// IngestInput is a verified webhook delivery ready for business validation.
type IngestInput struct {
	DeliveryID string
	EventType  string
	RawBody    []byte
}

// IngestResult describes how the delivery was handled.
type IngestResult struct {
	Status string // accepted | already_received | ignored
	Reason string
}

type envelope struct {
	Action string `json:"action"`
	Repo   struct {
		ID int64 `json:"id"`
	} `json:"repository"`
}

// Ingest validates event type/repo and persists a pending webhook_events row.
// Signature verification must already have succeeded.
func (s *Service) Ingest(ctx context.Context, in IngestInput) (IngestResult, error) {
	if in.DeliveryID == "" {
		return IngestResult{}, ErrMissingDeliveryID
	}
	if in.EventType == "" {
		return IngestResult{}, ErrMissingEvent
	}
	if _, ok := SupportedEvents[in.EventType]; !ok {
		return IngestResult{Status: "ignored", Reason: "unsupported_event"}, nil
	}

	var env envelope
	if err := json.Unmarshal(in.RawBody, &env); err != nil {
		return IngestResult{}, ErrInvalidPayload
	}
	if env.Repo.ID <= 0 {
		return IngestResult{}, ErrInvalidPayload
	}

	repo, err := s.Repos.GetByGitHubID(ctx, env.Repo.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			// Avoid GitHub retry storms for repos we do not manage.
			return IngestResult{Status: "ignored", Reason: "unknown_repository"}, nil
		}
		return IngestResult{}, fmt.Errorf("%w: %v", ErrPersistFailed, err)
	}

	payload := json.RawMessage(append([]byte(nil), in.RawBody...))
	maxRetries := s.MaxRetries
	if maxRetries < 0 {
		maxRetries = 3
	}
	_, err = s.Events.InsertPendingWithMaxRetries(ctx, repo.ID, in.DeliveryID, in.EventType, env.Action, payload, maxRetries)
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			return IngestResult{Status: "already_received"}, nil
		}
		return IngestResult{}, fmt.Errorf("%w: %v", ErrPersistFailed, err)
	}
	return IngestResult{Status: "accepted"}, nil
}

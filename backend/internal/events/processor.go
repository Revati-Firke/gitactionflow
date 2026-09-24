package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/Revati-Firke/gitactionflow/backend/internal/store"
	"github.com/Revati-Firke/gitactionflow/backend/internal/webhook"
)

// RepoLookup resolves a connected repository by internal id and GitHub id.
type RepoLookup interface {
	GetByID(ctx context.Context, id uuid.UUID) (store.Repository, error)
	GetByGitHubID(ctx context.Context, githubRepoID int64) (store.Repository, error)
}

// Processor validates a claimed webhook event for the processing pipeline.
// Future rule/action steps plug in after Validate succeeds.
type Processor struct {
	Repos RepoLookup
}

type payloadEnvelope struct {
	Action string `json:"action"`
	Repo   struct {
		ID int64 `json:"id"`
	} `json:"repository"`
}

// Process runs the Phase 6 pipeline: load/validate only (no rules or actions).
func (p *Processor) Process(ctx context.Context, e store.WebhookEvent) error {
	if err := p.Validate(ctx, e); err != nil {
		return err
	}
	// Future: rule engine + actions go here.
	return nil
}

// Validate checks persisted event integrity before marking processed.
func (p *Processor) Validate(ctx context.Context, e store.WebhookEvent) error {
	if e.ID == uuid.Nil || e.RepositoryID == uuid.Nil {
		return fmt.Errorf("%w: missing ids", ErrInvalidEvent)
	}
	if strings.TrimSpace(e.DeliveryID) == "" {
		return fmt.Errorf("%w: empty delivery_id", ErrInvalidEvent)
	}
	if _, ok := webhook.SupportedEvents[e.EventType]; !ok {
		return fmt.Errorf("%w: %s", ErrUnsupportedEvent, e.EventType)
	}
	if len(e.Payload) == 0 || !json.Valid(e.Payload) {
		return fmt.Errorf("%w: payload is not valid JSON", ErrInvalidEvent)
	}

	var env payloadEnvelope
	if err := json.Unmarshal(e.Payload, &env); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidEvent, err)
	}
	if env.Repo.ID <= 0 {
		return fmt.Errorf("%w: missing repository.id", ErrInvalidEvent)
	}

	connected, err := p.Repos.GetByID(ctx, e.RepositoryID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("%w: connected repository missing", ErrInvalidEvent)
		}
		return fmt.Errorf("load connected repository: %w", err)
	}
	if connected.GitHubRepositoryID != env.Repo.ID {
		return fmt.Errorf("%w: payload id %d != connected %d", ErrRepoMismatch, env.Repo.ID, connected.GitHubRepositoryID)
	}

	// Cross-check GitHub id still maps to the same row (defence in depth).
	byGH, err := p.Repos.GetByGitHubID(ctx, env.Repo.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("%w: repository no longer connected", ErrInvalidEvent)
		}
		return fmt.Errorf("lookup repository by github id: %w", err)
	}
	if byGH.ID != e.RepositoryID {
		return fmt.Errorf("%w: repository_id mismatch", ErrRepoMismatch)
	}

	return nil
}

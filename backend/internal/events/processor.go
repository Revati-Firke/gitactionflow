package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/Revati-Firke/gitactionflow/backend/internal/rules"
	"github.com/Revati-Firke/gitactionflow/backend/internal/store"
	"github.com/Revati-Firke/gitactionflow/backend/internal/webhook"
)

// RepoLookup resolves a connected repository by internal id and GitHub id.
type RepoLookup interface {
	GetByID(ctx context.Context, id uuid.UUID) (store.Repository, error)
	GetByGitHubID(ctx context.Context, githubRepoID int64) (store.Repository, error)
}

// RuleLoader loads enabled rules for a repository during processing.
type RuleLoader interface {
	ListEnabledByRepository(ctx context.Context, repositoryID uuid.UUID) ([]store.Rule, error)
}

// ActionRunner persists and executes action intents.
type ActionRunner interface {
	EnsureAndExecute(ctx context.Context, ev store.WebhookEvent, evCtx rules.EventContext, intents []rules.ActionIntent) error
}

// Processor validates events, evaluates rules, and runs actions.
type Processor struct {
	Repos    RepoLookup
	Rules   RuleLoader
	Actions ActionRunner
	Log     *slog.Logger
}

type payloadEnvelope struct {
	Action string `json:"action"`
	Repo   struct {
		ID int64 `json:"id"`
	} `json:"repository"`
}

// Process runs: validate → extract → rules → create/execute actions → success only if actions complete.
func (p *Processor) Process(ctx context.Context, e store.WebhookEvent) error {
	if err := p.Validate(ctx, e); err != nil {
		return err
	}

	evCtx, err := ExtractEventContext(e)
	if err != nil {
		return err
	}

	log := p.Log
	if log == nil {
		log = slog.Default()
	}

	log.Info("rule evaluation started",
		"event_id", e.ID.String(),
		"repository_id", e.RepositoryID.String(),
		"event_type", e.EventType,
	)

	var ruleList []store.Rule
	if p.Rules != nil {
		ruleList, err = p.Rules.ListEnabledByRepository(ctx, e.RepositoryID)
		if err != nil {
			return fmt.Errorf("load rules: %w", err)
		}
	}

	intents := rules.Evaluate(ruleList, evCtx)
	for _, intent := range intents {
		log.Info("rule matched",
			"event_id", e.ID.String(),
			"repository_id", e.RepositoryID.String(),
			"rule_id", intent.RuleID.String(),
			"event_type", e.EventType,
			"action_type", intent.ActionType,
			"rule_name", intent.RuleName,
		)
	}
	log.Info("rule evaluation completed",
		"event_id", e.ID.String(),
		"repository_id", e.RepositoryID.String(),
		"event_type", e.EventType,
		"matched", len(intents),
	)

	if p.Actions == nil {
		return nil
	}
	if err := p.Actions.EnsureAndExecute(ctx, e, evCtx, intents); err != nil {
		return err
	}
	return nil
}

// Validate checks persisted event integrity before rule evaluation.
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

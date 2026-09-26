package events_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/google/uuid"

	"github.com/Revati-Firke/gitactionflow/backend/internal/events"
	"github.com/Revati-Firke/gitactionflow/backend/internal/store"
)

type procRepos struct {
	repo store.Repository
}

func (p procRepos) GetByID(context.Context, uuid.UUID) (store.Repository, error) { return p.repo, nil }
func (p procRepos) GetByGitHubID(context.Context, int64) (store.Repository, error) {
	return p.repo, nil
}

type procRules struct {
	list []store.Rule
}

func (p procRules) ListEnabledByRepository(context.Context, uuid.UUID) ([]store.Rule, error) {
	return p.list, nil
}

func TestProcessor_EvaluatesRulesWithoutSideEffects(t *testing.T) {
	repoID := uuid.New()
	repo := store.Repository{ID: repoID, GitHubRepositoryID: 42}
	kw := "bug"
	ruleList := []store.Rule{{
		ID: uuid.New(), RepositoryID: repoID, Name: "Bug Issues", Enabled: true,
		EventType: "issues", Keyword: &kw, ActionType: "github_label",
		ActionConfig: json.RawMessage(`{"label":"automation"}`),
	}}

	payload, _ := json.Marshal(map[string]any{
		"action":     "opened",
		"repository": map[string]any{"id": 42},
		"issue": map[string]any{
			"title":  "Bug found in login",
			"body":   "details",
			"user":   map[string]any{"login": "octocat"},
			"labels": []map[string]any{},
		},
	})
	ev := store.WebhookEvent{
		ID: uuid.New(), RepositoryID: repoID, DeliveryID: uuid.NewString(),
		EventType: "issues", Action: "opened", Payload: payload, Status: store.WebhookStatusProcessing,
	}

	p := &events.Processor{
		Repos:  procRepos{repo: repo},
		Rules: procRules{list: ruleList},
		Log:   slog.Default(),
	}
	if err := p.Process(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
}

func TestExtractEventContext(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{
		"action": "opened",
		"issue": map[string]any{
			"number": 42, "title": "Hello", "body": "World",
			"user":   map[string]any{"login": "alice"},
			"labels": []map[string]any{{"name": "bug"}},
		},
	})
	ctx, err := events.ExtractEventContext(store.WebhookEvent{
		ID: uuid.New(), EventType: "issues", Action: "opened", Payload: payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Title != "Hello" || ctx.Author != "alice" || len(ctx.Labels) != 1 || ctx.IssueNumber != 42 {
		t.Fatalf("%+v", ctx)
	}
}

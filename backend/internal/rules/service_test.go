package rules_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/Revati-Firke/gitactionflow/backend/internal/rules"
	"github.com/Revati-Firke/gitactionflow/backend/internal/store"
)

type memRules struct {
	mu    sync.Mutex
	byID  map[uuid.UUID]store.Rule
	order []uuid.UUID
}

func newMemRules() *memRules {
	return &memRules{byID: map[uuid.UUID]store.Rule{}}
}

func (m *memRules) Create(_ context.Context, r store.Rule) (store.Rule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r.ID = uuid.New()
	m.byID[r.ID] = r
	m.order = append(m.order, r.ID)
	return r, nil
}

func (m *memRules) ListByRepository(_ context.Context, repositoryID uuid.UUID) ([]store.Rule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []store.Rule
	for _, id := range m.order {
		r := m.byID[id]
		if r.RepositoryID == repositoryID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *memRules) ListEnabledByRepository(ctx context.Context, repositoryID uuid.UUID) ([]store.Rule, error) {
	all, err := m.ListByRepository(ctx, repositoryID)
	if err != nil {
		return nil, err
	}
	var out []store.Rule
	for _, r := range all {
		if r.Enabled {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *memRules) GetByID(_ context.Context, id uuid.UUID) (store.Rule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.byID[id]
	if !ok {
		return store.Rule{}, store.ErrNotFound
	}
	return r, nil
}

func (m *memRules) Update(_ context.Context, repositoryID, id uuid.UUID, r store.Rule) (store.Rule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cur, ok := m.byID[id]
	if !ok || cur.RepositoryID != repositoryID {
		return store.Rule{}, store.ErrNotFound
	}
	r.ID = id
	r.RepositoryID = repositoryID
	m.byID[id] = r
	return r, nil
}

func (m *memRules) Delete(_ context.Context, repositoryID, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cur, ok := m.byID[id]
	if !ok || cur.RepositoryID != repositoryID {
		return store.ErrNotFound
	}
	delete(m.byID, id)
	return nil
}

type memRepos struct {
	byUser map[uuid.UUID]store.Repository
}

func (m *memRepos) GetByUserID(_ context.Context, userID uuid.UUID) (store.Repository, error) {
	r, ok := m.byUser[userID]
	if !ok {
		return store.Repository{}, store.ErrNotFound
	}
	return r, nil
}

func TestService_CRUDScopedToConnectedRepo(t *testing.T) {
	userA, userB := uuid.New(), uuid.New()
	repoA, repoB := uuid.New(), uuid.New()
	svc := &rules.Service{
		Rules: newMemRules(),
		Repos: &memRepos{byUser: map[uuid.UUID]store.Repository{
			userA: {ID: repoA},
			userB: {ID: repoB},
		}},
	}
	en := true
	created, err := svc.Create(context.Background(), userA, rules.Input{
		Name: "Bug", Enabled: en, EventType: "issues", ActionType: "github_label",
		ActionConfig: json.RawMessage(`{"label":"automation"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.RepositoryID != repoA {
		t.Fatal(created.RepositoryID)
	}

	_, err = svc.Get(context.Background(), userB, created.ID)
	if !errors.Is(err, rules.ErrNotFound) {
		t.Fatalf("want not found for other user, got %v", err)
	}

	_, err = svc.Create(context.Background(), uuid.New(), rules.Input{
		Name: "x", Enabled: true, EventType: "issues", ActionType: "github_label",
		ActionConfig: json.RawMessage(`{"label":"a"}`),
	})
	if !errors.Is(err, rules.ErrNoRepository) {
		t.Fatalf("got %v", err)
	}
}

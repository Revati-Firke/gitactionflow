package webhook_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/Revati-Firke/gitactionflow/backend/internal/store"
	"github.com/Revati-Firke/gitactionflow/backend/internal/webhook"
)

type memRepos struct {
	byGH map[int64]store.Repository
}

func (m *memRepos) GetByGitHubID(_ context.Context, githubRepoID int64) (store.Repository, error) {
	r, ok := m.byGH[githubRepoID]
	if !ok {
		return store.Repository{}, store.ErrNotFound
	}
	return r, nil
}

type memEvents struct {
	mu   sync.Mutex
	byID map[string]store.WebhookEvent
}

func newMemEvents() *memEvents {
	return &memEvents{byID: map[string]store.WebhookEvent{}}
}

func (m *memEvents) InsertPending(_ context.Context, repositoryID uuid.UUID, deliveryID, eventType, action string, payload json.RawMessage) (store.WebhookEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byID[deliveryID]; ok {
		return store.WebhookEvent{}, store.ErrConflict
	}
	e := store.WebhookEvent{
		ID: uuid.New(), RepositoryID: repositoryID, DeliveryID: deliveryID,
		EventType: eventType, Action: action, Payload: payload, Status: store.WebhookStatusPending,
	}
	m.byID[deliveryID] = e
	return e, nil
}

func TestIngest_AcceptsSupportedEvent(t *testing.T) {
	repoID := uuid.New()
	svc := &webhook.Service{
		Repos:  &memRepos{byGH: map[int64]store.Repository{42: {ID: repoID, GitHubRepositoryID: 42}}},
		Events: newMemEvents(),
	}
	body := []byte(`{"action":"opened","repository":{"id":42}}`)
	res, err := svc.Ingest(context.Background(), webhook.IngestInput{
		DeliveryID: "del-1", EventType: "issues", RawBody: body,
	})
	if err != nil || res.Status != "accepted" {
		t.Fatalf("%+v %v", res, err)
	}
}

func TestIngest_IgnoresUnsupportedEvent(t *testing.T) {
	svc := &webhook.Service{Repos: &memRepos{byGH: map[int64]store.Repository{}}, Events: newMemEvents()}
	res, err := svc.Ingest(context.Background(), webhook.IngestInput{
		DeliveryID: "del-2", EventType: "push", RawBody: []byte(`{"repository":{"id":1}}`),
	})
	if err != nil || res.Status != "ignored" || res.Reason != "unsupported_event" {
		t.Fatalf("%+v %v", res, err)
	}
}

func TestIngest_IgnoresUnknownRepository(t *testing.T) {
	svc := &webhook.Service{Repos: &memRepos{byGH: map[int64]store.Repository{}}, Events: newMemEvents()}
	res, err := svc.Ingest(context.Background(), webhook.IngestInput{
		DeliveryID: "del-3", EventType: "pull_request",
		RawBody: []byte(`{"action":"opened","repository":{"id":99}}`),
	})
	if err != nil || res.Status != "ignored" || res.Reason != "unknown_repository" {
		t.Fatalf("%+v %v", res, err)
	}
}

func TestIngest_DuplicateDelivery(t *testing.T) {
	repoID := uuid.New()
	events := newMemEvents()
	svc := &webhook.Service{
		Repos:  &memRepos{byGH: map[int64]store.Repository{7: {ID: repoID, GitHubRepositoryID: 7}}},
		Events: events,
	}
	in := webhook.IngestInput{
		DeliveryID: "same-del", EventType: "issues",
		RawBody: []byte(`{"action":"opened","repository":{"id":7}}`),
	}
	if _, err := svc.Ingest(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	res, err := svc.Ingest(context.Background(), in)
	if err != nil || res.Status != "already_received" {
		t.Fatalf("%+v %v", res, err)
	}
	if len(events.byID) != 1 {
		t.Fatalf("want 1 row, got %d", len(events.byID))
	}
}

func TestIngest_ConcurrentDuplicates(t *testing.T) {
	repoID := uuid.New()
	events := newMemEvents()
	svc := &webhook.Service{
		Repos:  &memRepos{byGH: map[int64]store.Repository{7: {ID: repoID, GitHubRepositoryID: 7}}},
		Events: events,
	}
	in := webhook.IngestInput{
		DeliveryID: "race-del", EventType: "issues",
		RawBody: []byte(`{"action":"opened","repository":{"id":7}}`),
	}
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.Ingest(context.Background(), in)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(events.byID) != 1 {
		t.Fatalf("want 1 row after race, got %d", len(events.byID))
	}
}

func TestIngest_InvalidPayload(t *testing.T) {
	svc := &webhook.Service{Repos: &memRepos{byGH: map[int64]store.Repository{}}, Events: newMemEvents()}
	_, err := svc.Ingest(context.Background(), webhook.IngestInput{
		DeliveryID: "x", EventType: "issues", RawBody: []byte(`not-json`),
	})
	if !errors.Is(err, webhook.ErrInvalidPayload) {
		t.Fatalf("got %v", err)
	}
}

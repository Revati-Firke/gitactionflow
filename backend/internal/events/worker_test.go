package events_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Revati-Firke/gitactionflow/backend/internal/events"
	"github.com/Revati-Firke/gitactionflow/backend/internal/store"
)

type memRepos struct {
	mu   sync.Mutex
	byID map[uuid.UUID]store.Repository
	byGH map[int64]store.Repository
}

func newMemRepos(r store.Repository) *memRepos {
	return &memRepos{
		byID: map[uuid.UUID]store.Repository{r.ID: r},
		byGH: map[int64]store.Repository{r.GitHubRepositoryID: r},
	}
}

func (m *memRepos) GetByID(_ context.Context, id uuid.UUID) (store.Repository, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.byID[id]
	if !ok {
		return store.Repository{}, store.ErrNotFound
	}
	return r, nil
}

func (m *memRepos) GetByGitHubID(_ context.Context, githubRepoID int64) (store.Repository, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.byGH[githubRepoID]
	if !ok {
		return store.Repository{}, store.ErrNotFound
	}
	return r, nil
}

type memQueue struct {
	mu     sync.Mutex
	events map[uuid.UUID]*store.WebhookEvent
	order  []uuid.UUID
}

func newMemQueue() *memQueue {
	return &memQueue{events: map[uuid.UUID]*store.WebhookEvent{}}
}

func (q *memQueue) put(e store.WebhookEvent) {
	q.mu.Lock()
	defer q.mu.Unlock()
	cp := e
	q.events[e.ID] = &cp
	q.order = append(q.order, e.ID)
}

func (q *memQueue) get(id uuid.UUID) store.WebhookEvent {
	q.mu.Lock()
	defer q.mu.Unlock()
	return *q.events[id]
}

func (q *memQueue) ClaimNext(_ context.Context, lease time.Duration) (store.WebhookEvent, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	now := time.Now().UTC()
	for _, id := range q.order {
		e := q.events[id]
		eligible := false
		if e.Status == store.WebhookStatusPending {
			if e.NextRetryAt == nil || !e.NextRetryAt.After(now) {
				eligible = true
			}
		}
		if e.Status == store.WebhookStatusProcessing && e.LockedAt != nil && e.LockedAt.Before(now.Add(-lease)) {
			eligible = true
			e.RetryCount++
			msg := "reclaimed stale processing lease"
			e.LastError = &msg
			if e.RetryCount >= e.MaxRetries {
				e.Status = store.WebhookStatusFailed
				t := now
				e.FailedAt = &t
				e.LockedAt = nil
				continue
			}
		}
		if !eligible {
			continue
		}
		e.Status = store.WebhookStatusProcessing
		t := now
		e.LockedAt = &t
		e.NextRetryAt = nil
		return *e, nil
	}
	return store.WebhookEvent{}, store.ErrNotFound
}

func (q *memQueue) MarkProcessed(_ context.Context, id uuid.UUID) (store.WebhookEvent, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	e, ok := q.events[id]
	if !ok || e.Status != store.WebhookStatusProcessing {
		return store.WebhookEvent{}, store.ErrNotFound
	}
	e.Status = store.WebhookStatusProcessed
	t := time.Now().UTC()
	e.ProcessedAt = &t
	e.LockedAt = nil
	e.LastError = nil
	return *e, nil
}

func (q *memQueue) ScheduleRetry(_ context.Context, id uuid.UUID, errMsg string, backoff time.Duration) (store.WebhookEvent, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	e, ok := q.events[id]
	if !ok {
		return store.WebhookEvent{}, store.ErrNotFound
	}
	e.RetryCount++
	e.LastError = &errMsg
	e.LockedAt = nil
	if e.RetryCount >= e.MaxRetries {
		e.Status = store.WebhookStatusFailed
		t := time.Now().UTC()
		e.FailedAt = &t
		e.NextRetryAt = nil
		return *e, nil
	}
	e.Status = store.WebhookStatusPending
	next := time.Now().UTC().Add(backoff)
	e.NextRetryAt = &next
	return *e, nil
}

func (q *memQueue) MarkFailedImmediately(_ context.Context, id uuid.UUID, errMsg string) (store.WebhookEvent, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	e, ok := q.events[id]
	if !ok || e.Status != store.WebhookStatusProcessing {
		return store.WebhookEvent{}, store.ErrNotFound
	}
	e.Status = store.WebhookStatusFailed
	e.LastError = &errMsg
	t := time.Now().UTC()
	e.FailedAt = &t
	e.LockedAt = nil
	e.NextRetryAt = nil
	return *e, nil
}

func validEvent(repo store.Repository) store.WebhookEvent {
	payload, _ := json.Marshal(map[string]any{
		"action":     "opened",
		"repository": map[string]any{"id": repo.GitHubRepositoryID},
	})
	return store.WebhookEvent{
		ID:           uuid.New(),
		RepositoryID: repo.ID,
		DeliveryID:   uuid.NewString(),
		EventType:    "issues",
		Action:       "opened",
		Payload:      payload,
		Status:       store.WebhookStatusPending,
		MaxRetries:   3,
	}
}

func TestProcessor_Success(t *testing.T) {
	repo := store.Repository{ID: uuid.New(), GitHubRepositoryID: 99}
	p := &events.Processor{Repos: newMemRepos(repo)}
	e := validEvent(repo)
	if err := p.Process(context.Background(), e); err != nil {
		t.Fatal(err)
	}
}

func TestProcessor_InvalidPayload(t *testing.T) {
	repo := store.Repository{ID: uuid.New(), GitHubRepositoryID: 99}
	p := &events.Processor{Repos: newMemRepos(repo)}
	e := validEvent(repo)
	e.Payload = json.RawMessage(`not-json`)
	err := p.Process(context.Background(), e)
	if !events.IsPermanent(err) {
		t.Fatalf("want permanent, got %v", err)
	}
}

func TestProcessor_RepoMismatch(t *testing.T) {
	repo := store.Repository{ID: uuid.New(), GitHubRepositoryID: 99}
	p := &events.Processor{Repos: newMemRepos(repo)}
	e := validEvent(repo)
	e.Payload, _ = json.Marshal(map[string]any{
		"action": "opened", "repository": map[string]any{"id": 1},
	})
	err := p.Process(context.Background(), e)
	if !errors.Is(err, events.ErrRepoMismatch) {
		t.Fatalf("got %v", err)
	}
}

func TestWorker_PendingToProcessed(t *testing.T) {
	repo := store.Repository{ID: uuid.New(), GitHubRepositoryID: 42}
	q := newMemQueue()
	e := validEvent(repo)
	q.put(e)

	w := &events.Worker{
		Queue: q, Pipeline: &events.Processor{Repos: newMemRepos(repo)},
		PollInterval: time.Hour, Lease: time.Minute, MaxPerTick: 5,
	}
	if !w.ProcessOneForTest(context.Background(), time.Minute) {
		t.Fatal("expected work")
	}
	got := q.get(e.ID)
	if got.Status != store.WebhookStatusProcessed || got.ProcessedAt == nil {
		t.Fatalf("got %+v", got)
	}
}

type failOncePipeline struct {
	n int
}

func (f *failOncePipeline) Process(context.Context, store.WebhookEvent) error {
	f.n++
	if f.n == 1 {
		return errors.New("transient boom")
	}
	return nil
}

func TestWorker_RetryThenSuccess(t *testing.T) {
	repo := store.Repository{ID: uuid.New(), GitHubRepositoryID: 42}
	q := newMemQueue()
	e := validEvent(repo)
	e.MaxRetries = 3
	q.put(e)

	pipe := &failOncePipeline{}
	w := &events.Worker{Queue: q, Pipeline: pipe, MaxPerTick: 5, Lease: time.Minute}

	if !w.ProcessOneForTest(context.Background(), time.Minute) {
		t.Fatal("expected first claim")
	}
	got := q.get(e.ID)
	if got.Status != store.WebhookStatusPending || got.RetryCount != 1 || got.LastError == nil || got.NextRetryAt == nil {
		t.Fatalf("after fail: %+v", got)
	}
	// Make retry eligible now.
	past := time.Now().UTC().Add(-time.Second)
	q.mu.Lock()
	q.events[e.ID].NextRetryAt = &past
	q.mu.Unlock()

	if !w.ProcessOneForTest(context.Background(), time.Minute) {
		t.Fatal("expected retry claim")
	}
	got = q.get(e.ID)
	if got.Status != store.WebhookStatusProcessed {
		t.Fatalf("want processed, got %+v", got)
	}
}

func TestWorker_RetryExhaustion(t *testing.T) {
	repo := store.Repository{ID: uuid.New(), GitHubRepositoryID: 42}
	q := newMemQueue()
	e := validEvent(repo)
	e.MaxRetries = 2
	q.put(e)

	w := &events.Worker{
		Queue: q,
		Pipeline: processFunc(func(context.Context, store.WebhookEvent) error {
			return errors.New("always fail")
		}),
		Lease: time.Minute,
	}

	for i := 0; i < 2; i++ {
		if !w.ProcessOneForTest(context.Background(), time.Minute) {
			t.Fatalf("tick %d: no work", i)
		}
		past := time.Now().UTC().Add(-time.Second)
		q.mu.Lock()
		if q.events[e.ID].NextRetryAt != nil {
			q.events[e.ID].NextRetryAt = &past
		}
		q.mu.Unlock()
	}
	got := q.get(e.ID)
	if got.Status != store.WebhookStatusFailed || got.FailedAt == nil || got.LastError == nil || got.RetryCount != 2 {
		t.Fatalf("got %+v", got)
	}
}

func TestWorker_PermanentFailure(t *testing.T) {
	repo := store.Repository{ID: uuid.New(), GitHubRepositoryID: 42}
	q := newMemQueue()
	e := validEvent(repo)
	e.Payload = json.RawMessage(`{}`)
	q.put(e)

	w := &events.Worker{
		Queue: q, Pipeline: &events.Processor{Repos: newMemRepos(repo)}, Lease: time.Minute,
	}
	w.ProcessOneForTest(context.Background(), time.Minute)
	got := q.get(e.ID)
	if got.Status != store.WebhookStatusFailed || got.RetryCount != 0 {
		t.Fatalf("want immediate failed, got %+v", got)
	}
}

func TestWorker_StaleReclaim(t *testing.T) {
	repo := store.Repository{ID: uuid.New(), GitHubRepositoryID: 42}
	q := newMemQueue()
	e := validEvent(repo)
	e.Status = store.WebhookStatusProcessing
	old := time.Now().UTC().Add(-2 * time.Minute)
	e.LockedAt = &old
	q.put(e)

	w := &events.Worker{
		Queue: q, Pipeline: &events.Processor{Repos: newMemRepos(repo)}, Lease: time.Minute,
	}
	if !w.ProcessOneForTest(context.Background(), time.Minute) {
		t.Fatal("expected reclaim")
	}
	got := q.get(e.ID)
	if got.Status != store.WebhookStatusProcessed {
		t.Fatalf("got %+v", got)
	}
	if got.RetryCount != 1 {
		t.Fatalf("retry_count=%d want 1 after stale reclaim", got.RetryCount)
	}
}

func TestClaim_ConcurrentExclusive(t *testing.T) {
	repo := store.Repository{ID: uuid.New(), GitHubRepositoryID: 7}
	q := newMemQueue()
	e := validEvent(repo)
	q.put(e)

	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := q.ClaimNext(context.Background(), time.Minute)
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	ok, notFound := 0, 0
	for err := range results {
		if err == nil {
			ok++
		} else if errors.Is(err, store.ErrNotFound) {
			notFound++
		} else {
			t.Fatalf("unexpected err %v", err)
		}
	}
	if ok != 1 || notFound != 1 {
		t.Fatalf("ok=%d notFound=%d", ok, notFound)
	}
}

func TestRetryBackoff(t *testing.T) {
	if events.RetryBackoff(1) != time.Minute {
		t.Fatal(events.RetryBackoff(1))
	}
	if events.RetryBackoff(2) != 5*time.Minute {
		t.Fatal(events.RetryBackoff(2))
	}
	if events.RetryBackoff(3) != 15*time.Minute {
		t.Fatal(events.RetryBackoff(3))
	}
}

type processFunc func(context.Context, store.WebhookEvent) error

func (f processFunc) Process(ctx context.Context, e store.WebhookEvent) error {
	return f(ctx, e)
}

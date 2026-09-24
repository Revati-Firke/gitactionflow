package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Revati-Firke/gitactionflow/backend/internal/http/handlers"
	"github.com/Revati-Firke/gitactionflow/backend/internal/http/response"
	"github.com/Revati-Firke/gitactionflow/backend/internal/store"
	"github.com/Revati-Firke/gitactionflow/backend/internal/webhook"
)

type whRepos struct {
	repo store.Repository
	err  error
}

func (m *whRepos) GetByGitHubID(context.Context, int64) (store.Repository, error) {
	if m.err != nil {
		return store.Repository{}, m.err
	}
	return m.repo, nil
}

type whEvents struct {
	mu     sync.Mutex
	byID   map[string]store.WebhookEvent
	fail   bool
	insert int
}

func (m *whEvents) InsertPending(_ context.Context, repositoryID uuid.UUID, deliveryID, eventType, action string, payload json.RawMessage) (store.WebhookEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.insert++
	if m.fail {
		return store.WebhookEvent{}, store.ErrConflict // will map oddly; use generic error via wrapping in service
	}
	if m.byID == nil {
		m.byID = map[string]store.WebhookEvent{}
	}
	if _, ok := m.byID[deliveryID]; ok {
		return store.WebhookEvent{}, store.ErrConflict
	}
	e := store.WebhookEvent{ID: uuid.New(), RepositoryID: repositoryID, DeliveryID: deliveryID, EventType: eventType, Action: action, Payload: payload, Status: store.WebhookStatusPending}
	m.byID[deliveryID] = e
	return e, nil
}

type failEvents struct{}

func (failEvents) InsertPending(context.Context, uuid.UUID, string, string, string, json.RawMessage) (store.WebhookEvent, error) {
	return store.WebhookEvent{}, errorsNew("db down")
}

func errorsNew(s string) error { return &simpleErr{s} }

type simpleErr struct{ s string }

func (e *simpleErr) Error() string { return e.s }

func setupWebhookRouter(secret string, svc *webhook.Service, maxBody int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := &handlers.GitHubWebhookHandler{Service: svc, Secret: secret, MaxBody: maxBody}
	r := gin.New()
	r.POST("/webhooks/github", h.HandlePOST)
	return r
}

func signedRequest(t *testing.T, secret, delivery, event string, body []byte) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Delivery", delivery)
	req.Header.Set("X-GitHub-Event", event)
	req.Header.Set("X-Hub-Signature-256", webhook.SignBody(secret, body))
	return req
}

func TestWebhook_ValidIssueAccepted(t *testing.T) {
	secret := "super-secret-webhook"
	repoID := uuid.New()
	events := &whEvents{byID: map[string]store.WebhookEvent{}}
	svc := &webhook.Service{
		Repos:  &whRepos{repo: store.Repository{ID: repoID, GitHubRepositoryID: 55}},
		Events: events,
	}
	r := setupWebhookRouter(secret, svc, 1<<20)
	body := []byte(`{"action":"opened","repository":{"id":55}}`)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, signedRequest(t, secret, "d-1", "issues", body))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var out map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if out["status"] != "accepted" {
		t.Fatalf("%v", out)
	}
	if ev := events.byID["d-1"]; ev.Status != store.WebhookStatusPending || ev.EventType != "issues" {
		t.Fatalf("%+v", ev)
	}
}

func TestWebhook_InvalidSignature(t *testing.T) {
	secret := "super-secret-webhook"
	events := &whEvents{byID: map[string]store.WebhookEvent{}}
	svc := &webhook.Service{Repos: &whRepos{repo: store.Repository{ID: uuid.New(), GitHubRepositoryID: 1}}, Events: events}
	r := setupWebhookRouter(secret, svc, 1<<20)
	body := []byte(`{"action":"opened","repository":{"id":1}}`)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Delivery", "d")
	req.Header.Set("X-GitHub-Event", "issues")
	req.Header.Set("X-Hub-Signature-256", webhook.SignBody("wrong", body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", w.Code)
	}
	if len(events.byID) != 0 {
		t.Fatal("must not persist on bad signature")
	}
	var errBody response.ErrorBody
	_ = json.Unmarshal(w.Body.Bytes(), &errBody)
	if strings.Contains(w.Body.String(), "sha256=") {
		t.Fatal("must not leak signature material")
	}
}

func TestWebhook_MissingSignature(t *testing.T) {
	r := setupWebhookRouter("secret", &webhook.Service{Repos: &whRepos{}, Events: &whEvents{}}, 1<<20)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("X-GitHub-Delivery", "d")
	req.Header.Set("X-GitHub-Event", "issues")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestWebhook_DuplicateReturns200(t *testing.T) {
	secret := "super-secret-webhook"
	repoID := uuid.New()
	events := &whEvents{byID: map[string]store.WebhookEvent{}}
	svc := &webhook.Service{Repos: &whRepos{repo: store.Repository{ID: repoID, GitHubRepositoryID: 9}}, Events: events}
	r := setupWebhookRouter(secret, svc, 1<<20)
	body := []byte(`{"action":"opened","repository":{"id":9}}`)
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, signedRequest(t, secret, "dup", "pull_request", body))
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d", w.Code)
		}
	}
	if len(events.byID) != 1 {
		t.Fatalf("rows=%d", len(events.byID))
	}
}

func TestWebhook_UnsupportedIgnored(t *testing.T) {
	secret := "super-secret-webhook"
	events := &whEvents{byID: map[string]store.WebhookEvent{}}
	r := setupWebhookRouter(secret, &webhook.Service{Repos: &whRepos{}, Events: events}, 1<<20)
	body := []byte(`{"repository":{"id":1}}`)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, signedRequest(t, secret, "p1", "push", body))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "unsupported_event") {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	if len(events.byID) != 0 {
		t.Fatal("should not persist unsupported")
	}
}

func TestWebhook_UnknownRepoIgnored(t *testing.T) {
	secret := "super-secret-webhook"
	events := &whEvents{byID: map[string]store.WebhookEvent{}}
	r := setupWebhookRouter(secret, &webhook.Service{Repos: &whRepos{err: store.ErrNotFound}, Events: events}, 1<<20)
	body := []byte(`{"action":"opened","repository":{"id":123}}`)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, signedRequest(t, secret, "u1", "issues", body))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "unknown_repository") {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}

func TestWebhook_DBFailureReturns500(t *testing.T) {
	secret := "super-secret-webhook"
	svc := &webhook.Service{
		Repos:  &whRepos{repo: store.Repository{ID: uuid.New(), GitHubRepositoryID: 1}},
		Events: failEvents{},
	}
	r := setupWebhookRouter(secret, svc, 1<<20)
	body := []byte(`{"action":"opened","repository":{"id":1}}`)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, signedRequest(t, secret, "db1", "issues", body))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestWebhook_BodyTooLarge(t *testing.T) {
	secret := "super-secret-webhook"
	r := setupWebhookRouter(secret, &webhook.Service{Repos: &whRepos{}, Events: &whEvents{}}, 64)
	body := []byte(strings.Repeat("a", 128))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, signedRequest(t, secret, "big", "issues", body))
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestWebhook_MissingHeaders(t *testing.T) {
	secret := "super-secret-webhook"
	r := setupWebhookRouter(secret, &webhook.Service{Repos: &whRepos{}, Events: &whEvents{}}, 1<<20)
	body := []byte(`{"repository":{"id":1}}`)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", webhook.SignBody(secret, body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", w.Code)
	}
}

package actions_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Revati-Firke/gitactionflow/backend/internal/actions"
	"github.com/Revati-Firke/gitactionflow/backend/internal/auth"
	"github.com/Revati-Firke/gitactionflow/backend/internal/events"
	"github.com/Revati-Firke/gitactionflow/backend/internal/githubapi"
	"github.com/Revati-Firke/gitactionflow/backend/internal/rules"
	"github.com/Revati-Firke/gitactionflow/backend/internal/slack"
	"github.com/Revati-Firke/gitactionflow/backend/internal/store"
)

type memActions struct {
	mu   sync.Mutex
	byID map[uuid.UUID]*store.Action
	byKey map[string]uuid.UUID
}

func newMemActions() *memActions {
	return &memActions{byID: map[uuid.UUID]*store.Action{}, byKey: map[string]uuid.UUID{}}
}

func (m *memActions) CreateOrGet(_ context.Context, eventID, ruleID uuid.UUID, actionType string, config json.RawMessage, key string, maxAttempts int) (store.Action, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id, ok := m.byKey[key]; ok {
		return *m.byID[id], false, nil
	}
	id := uuid.New()
	rid := ruleID
	a := store.Action{
		ID: id, EventID: eventID, RuleID: &rid, ActionType: actionType,
		ActionConfig: append(json.RawMessage(nil), config...), IdempotencyKey: key,
		Status: store.ActionStatusPending, MaxAttempts: maxAttempts,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	m.byID[id] = &a
	m.byKey[key] = id
	return a, true, nil
}

func (m *memActions) ListByEventID(_ context.Context, eventID uuid.UUID) ([]store.Action, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []store.Action
	for _, a := range m.byID {
		if a.EventID == eventID {
			cp := *a
			out = append(out, cp)
		}
	}
	return out, nil
}

func (m *memActions) MarkProcessing(_ context.Context, id uuid.UUID) (store.Action, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.byID[id]
	if !ok || a.Status != store.ActionStatusPending {
		return store.Action{}, store.ErrNotFound
	}
	if a.NextRetryAt != nil && a.NextRetryAt.After(time.Now().UTC()) {
		return store.Action{}, store.ErrNotFound
	}
	a.Status = store.ActionStatusProcessing
	now := time.Now().UTC()
	a.StartedAt = &now
	a.NextRetryAt = nil
	return *a, nil
}

func (m *memActions) MarkCompleted(_ context.Context, id uuid.UUID) (store.Action, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.byID[id]
	if !ok || a.Status != store.ActionStatusProcessing {
		return store.Action{}, store.ErrNotFound
	}
	a.Status = store.ActionStatusCompleted
	now := time.Now().UTC()
	a.CompletedAt = &now
	a.LastError = nil
	return *a, nil
}

func (m *memActions) MarkFailedImmediately(_ context.Context, id uuid.UUID, errMsg string) (store.Action, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.byID[id]
	if !ok || a.Status != store.ActionStatusProcessing {
		return store.Action{}, store.ErrNotFound
	}
	a.Status = store.ActionStatusFailed
	a.AttemptCount++
	a.LastError = &errMsg
	now := time.Now().UTC()
	a.FailedAt = &now
	return *a, nil
}

func (m *memActions) ScheduleRetry(_ context.Context, id uuid.UUID, errMsg string, backoff time.Duration) (store.Action, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.byID[id]
	if !ok {
		return store.Action{}, store.ErrNotFound
	}
	a.AttemptCount++
	a.LastError = &errMsg
	if a.AttemptCount >= a.MaxAttempts {
		a.Status = store.ActionStatusFailed
		now := time.Now().UTC()
		a.FailedAt = &now
		a.NextRetryAt = nil
		return *a, nil
	}
	a.Status = store.ActionStatusPending
	next := time.Now().UTC().Add(backoff)
	a.NextRetryAt = &next
	return *a, nil
}

type memRepos struct {
	repo store.Repository
}

func (m memRepos) GetByID(context.Context, uuid.UUID) (store.Repository, error) { return m.repo, nil }

type memTokens struct {
	enc []byte
}

func (m memTokens) GetEncryptedAccessToken(context.Context, uuid.UUID) ([]byte, error) {
	return m.enc, nil
}

type fakeGitHub struct {
	labels  []labelCall
	comments []commentCall
	err     error
}

type labelCall struct {
	owner, repo string
	number      int
	labels      []string
	token       string
}

type commentCall struct {
	owner, repo, comment string
	number               int
	token                string
}

func (f *fakeGitHub) AddIssueLabels(_ context.Context, token, owner, repo string, issueNumber int, labels []string) error {
	f.labels = append(f.labels, labelCall{owner: owner, repo: repo, number: issueNumber, labels: labels, token: token})
	return f.err
}

func (f *fakeGitHub) CreateIssueComment(_ context.Context, token, owner, repo string, issueNumber int, comment string) error {
	f.comments = append(f.comments, commentCall{owner: owner, repo: repo, number: issueNumber, comment: comment, token: token})
	return f.err
}

type fakeSlack struct {
	msgs []string
	err  error
}

func (f *fakeSlack) SendText(_ context.Context, text string) error {
	f.msgs = append(f.msgs, text)
	return f.err
}

func testKey() []byte {
	return auth.DeriveKey("0123456789abcdef0123456789abcdef")
}

func encryptToken(t *testing.T, plain string) []byte {
	t.Helper()
	enc, err := auth.Encrypt(testKey(), []byte(plain))
	if err != nil {
		t.Fatal(err)
	}
	return enc
}

func baseRepo(userID, repoID uuid.UUID) store.Repository {
	return store.Repository{
		ID: repoID, UserID: userID, GitHubRepositoryID: 99,
		Name: "demo", FullName: "acme/demo", OwnerLogin: "acme",
	}
}

func baseEvent(eventID, repoID uuid.UUID) store.WebhookEvent {
	return store.WebhookEvent{
		ID: eventID, RepositoryID: repoID, EventType: "issues", Action: "opened",
		DeliveryID: uuid.NewString(), Status: store.WebhookStatusProcessing,
	}
}

func TestExecutor_GitHubLabel(t *testing.T) {
	userID, repoID, eventID, ruleID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	gh := &fakeGitHub{}
	ex := &actions.Executor{
		Actions: newMemActions(), Repos: memRepos{repo: baseRepo(userID, repoID)},
		Users: memTokens{enc: encryptToken(t, "gh-token")}, TokenKey: testKey(),
		GitHub: gh, Slack: &fakeSlack{}, MaxAttempts: 3,
	}
	err := ex.EnsureAndExecute(context.Background(), baseEvent(eventID, repoID), rules.EventContext{
		EventID: eventID, IssueNumber: 7,
	}, []rules.ActionIntent{{
		RuleID: ruleID, EventID: eventID, ActionType: rules.ActionGitHubLabel,
		Config: json.RawMessage(`{"label":"automation"}`), RuleName: "Bug",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(gh.labels) != 1 || gh.labels[0].number != 7 || gh.labels[0].labels[0] != "automation" {
		t.Fatalf("%+v", gh.labels)
	}
	if gh.labels[0].owner != "acme" || gh.labels[0].repo != "demo" || gh.labels[0].token != "gh-token" {
		t.Fatalf("%+v", gh.labels[0])
	}
}

func TestExecutor_GitHubComment(t *testing.T) {
	userID, repoID, eventID, ruleID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	gh := &fakeGitHub{}
	ex := &actions.Executor{
		Actions: newMemActions(), Repos: memRepos{repo: baseRepo(userID, repoID)},
		Users: memTokens{enc: encryptToken(t, "tok")}, TokenKey: testKey(),
		GitHub: gh, Slack: &fakeSlack{}, MaxAttempts: 3,
	}
	err := ex.EnsureAndExecute(context.Background(), baseEvent(eventID, repoID), rules.EventContext{
		EventID: eventID, IssueNumber: 12,
	}, []rules.ActionIntent{{
		RuleID: ruleID, EventID: eventID, ActionType: rules.ActionGitHubComment,
		Config: json.RawMessage(`{"comment":"Matched rule."}`),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(gh.comments) != 1 || gh.comments[0].number != 12 || gh.comments[0].comment != "Matched rule." {
		t.Fatalf("%+v", gh.comments)
	}
}

func TestExecutor_SlackNotification(t *testing.T) {
	userID, repoID, eventID, ruleID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	sl := &fakeSlack{}
	ex := &actions.Executor{
		Actions: newMemActions(), Repos: memRepos{repo: baseRepo(userID, repoID)},
		Users: memTokens{enc: encryptToken(t, "tok")}, TokenKey: testKey(),
		GitHub: &fakeGitHub{}, Slack: sl, MaxAttempts: 3,
	}
	err := ex.EnsureAndExecute(context.Background(), baseEvent(eventID, repoID), rules.EventContext{EventID: eventID}, []rules.ActionIntent{{
		RuleID: ruleID, EventID: eventID, ActionType: rules.ActionSlackNotification,
		Config: json.RawMessage(`{"message":"A matching automation rule was triggered."}`), RuleName: "Bug Issues",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(sl.msgs) != 1 {
		t.Fatal(sl.msgs)
	}
	msg := sl.msgs[0]
	for _, want := range []string{"acme/demo", "issues", "opened", "Bug Issues", "A matching automation rule was triggered."} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing %q in %q", want, msg)
		}
	}
}

func TestExecutor_IdempotencySameTriple(t *testing.T) {
	userID, repoID, eventID, ruleID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	storeMem := newMemActions()
	gh := &fakeGitHub{}
	ex := &actions.Executor{
		Actions: storeMem, Repos: memRepos{repo: baseRepo(userID, repoID)},
		Users: memTokens{enc: encryptToken(t, "tok")}, TokenKey: testKey(),
		GitHub: gh, Slack: &fakeSlack{}, MaxAttempts: 3,
	}
	intent := rules.ActionIntent{
		RuleID: ruleID, EventID: eventID, ActionType: rules.ActionGitHubLabel,
		Config: json.RawMessage(`{"label":"automation"}`),
	}
	evCtx := rules.EventContext{EventID: eventID, IssueNumber: 1}
	ev := baseEvent(eventID, repoID)
	if err := ex.EnsureAndExecute(context.Background(), ev, evCtx, []rules.ActionIntent{intent}); err != nil {
		t.Fatal(err)
	}
	if err := ex.EnsureAndExecute(context.Background(), ev, evCtx, []rules.ActionIntent{intent}); err != nil {
		t.Fatal(err)
	}
	list, _ := storeMem.ListByEventID(context.Background(), eventID)
	if len(list) != 1 {
		t.Fatalf("want 1 action, got %d", len(list))
	}
	if len(gh.labels) != 1 {
		t.Fatalf("duplicate side effect: %d label calls", len(gh.labels))
	}
}

func TestExecutor_MultipleRules(t *testing.T) {
	userID, repoID, eventID := uuid.New(), uuid.New(), uuid.New()
	r1, r2 := uuid.New(), uuid.New()
	storeMem := newMemActions()
	ex := &actions.Executor{
		Actions: storeMem, Repos: memRepos{repo: baseRepo(userID, repoID)},
		Users: memTokens{enc: encryptToken(t, "tok")}, TokenKey: testKey(),
		GitHub: &fakeGitHub{}, Slack: &fakeSlack{}, MaxAttempts: 3,
	}
	err := ex.EnsureAndExecute(context.Background(), baseEvent(eventID, repoID), rules.EventContext{
		EventID: eventID, IssueNumber: 3,
	}, []rules.ActionIntent{
		{RuleID: r1, EventID: eventID, ActionType: rules.ActionGitHubLabel, Config: json.RawMessage(`{"label":"a"}`)},
		{RuleID: r2, EventID: eventID, ActionType: rules.ActionGitHubComment, Config: json.RawMessage(`{"comment":"hi"}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	list, _ := storeMem.ListByEventID(context.Background(), eventID)
	if len(list) != 2 {
		t.Fatalf("got %d", len(list))
	}
}

func TestExecutor_RetryThenSuccess(t *testing.T) {
	userID, repoID, eventID, ruleID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	gh := &fakeGitHub{err: githubapi.ErrRetryable}
	storeMem := newMemActions()
	ex := &actions.Executor{
		Actions: storeMem, Repos: memRepos{repo: baseRepo(userID, repoID)},
		Users: memTokens{enc: encryptToken(t, "tok")}, TokenKey: testKey(),
		GitHub: gh, Slack: &fakeSlack{}, MaxAttempts: 3,
	}
	intent := []rules.ActionIntent{{
		RuleID: ruleID, EventID: eventID, ActionType: rules.ActionGitHubLabel,
		Config: json.RawMessage(`{"label":"automation"}`),
	}}
	evCtx := rules.EventContext{EventID: eventID, IssueNumber: 1}
	ev := baseEvent(eventID, repoID)

	err := ex.EnsureAndExecute(context.Background(), ev, evCtx, intent)
	if !errors.Is(err, events.ErrActionsIncomplete) {
		t.Fatalf("got %v", err)
	}
	list, _ := storeMem.ListByEventID(context.Background(), eventID)
	if list[0].Status != store.ActionStatusPending || list[0].AttemptCount != 1 {
		t.Fatalf("%+v", list[0])
	}

	// Clear backoff so retry can claim.
	storeMem.mu.Lock()
	storeMem.byID[list[0].ID].NextRetryAt = nil
	storeMem.mu.Unlock()
	gh.err = nil

	if err := ex.EnsureAndExecute(context.Background(), ev, evCtx, intent); err != nil {
		t.Fatal(err)
	}
	list, _ = storeMem.ListByEventID(context.Background(), eventID)
	if list[0].Status != store.ActionStatusCompleted {
		t.Fatalf("%+v", list[0])
	}
}

func TestExecutor_RetryExhaustion(t *testing.T) {
	userID, repoID, eventID, ruleID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	gh := &fakeGitHub{err: githubapi.ErrRetryable}
	storeMem := newMemActions()
	ex := &actions.Executor{
		Actions: storeMem, Repos: memRepos{repo: baseRepo(userID, repoID)},
		Users: memTokens{enc: encryptToken(t, "tok")}, TokenKey: testKey(),
		GitHub: gh, Slack: &fakeSlack{}, MaxAttempts: 2,
	}
	intent := []rules.ActionIntent{{
		RuleID: ruleID, EventID: eventID, ActionType: rules.ActionGitHubLabel,
		Config: json.RawMessage(`{"label":"automation"}`),
	}}
	evCtx := rules.EventContext{EventID: eventID, IssueNumber: 1}
	ev := baseEvent(eventID, repoID)

	_ = ex.EnsureAndExecute(context.Background(), ev, evCtx, intent)
	list, _ := storeMem.ListByEventID(context.Background(), eventID)
	storeMem.mu.Lock()
	storeMem.byID[list[0].ID].NextRetryAt = nil
	storeMem.mu.Unlock()

	err := ex.EnsureAndExecute(context.Background(), ev, evCtx, intent)
	if !errors.Is(err, events.ErrActionsFailed) {
		t.Fatalf("got %v", err)
	}
	list, _ = storeMem.ListByEventID(context.Background(), eventID)
	if list[0].Status != store.ActionStatusFailed {
		t.Fatalf("%+v", list[0])
	}
}

func TestExecutor_PermanentError(t *testing.T) {
	userID, repoID, eventID, ruleID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	gh := &fakeGitHub{err: githubapi.ErrNotFound}
	storeMem := newMemActions()
	ex := &actions.Executor{
		Actions: storeMem, Repos: memRepos{repo: baseRepo(userID, repoID)},
		Users: memTokens{enc: encryptToken(t, "tok")}, TokenKey: testKey(),
		GitHub: gh, Slack: &fakeSlack{}, MaxAttempts: 5,
	}
	err := ex.EnsureAndExecute(context.Background(), baseEvent(eventID, repoID), rules.EventContext{
		EventID: eventID, IssueNumber: 1,
	}, []rules.ActionIntent{{
		RuleID: ruleID, EventID: eventID, ActionType: rules.ActionGitHubLabel,
		Config: json.RawMessage(`{"label":"automation"}`),
	}})
	if !errors.Is(err, events.ErrActionsFailed) {
		t.Fatalf("got %v", err)
	}
	list, _ := storeMem.ListByEventID(context.Background(), eventID)
	if list[0].Status != store.ActionStatusFailed || list[0].AttemptCount != 1 {
		t.Fatalf("%+v", list[0])
	}
}

func TestExecutor_SlackPermanentNotRetried(t *testing.T) {
	userID, repoID, eventID, ruleID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	sl := &fakeSlack{err: slack.ErrPermanent}
	ex := &actions.Executor{
		Actions: newMemActions(), Repos: memRepos{repo: baseRepo(userID, repoID)},
		Users: memTokens{enc: encryptToken(t, "tok")}, TokenKey: testKey(),
		GitHub: &fakeGitHub{}, Slack: sl, MaxAttempts: 5,
	}
	err := ex.EnsureAndExecute(context.Background(), baseEvent(eventID, repoID), rules.EventContext{EventID: eventID}, []rules.ActionIntent{{
		RuleID: ruleID, EventID: eventID, ActionType: rules.ActionSlackNotification,
		Config: json.RawMessage(`{"message":"hi"}`),
	}})
	if !errors.Is(err, events.ErrActionsFailed) {
		t.Fatalf("got %v", err)
	}
}

func TestExecutor_EventFailurePropagation(t *testing.T) {
	userID, repoID, eventID, ruleID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	ex := &actions.Executor{
		Actions: newMemActions(), Repos: memRepos{repo: baseRepo(userID, repoID)},
		Users: memTokens{enc: encryptToken(t, "tok")}, TokenKey: testKey(),
		GitHub: &fakeGitHub{err: githubapi.ErrForbidden}, Slack: &fakeSlack{}, MaxAttempts: 3,
	}
	err := ex.EnsureAndExecute(context.Background(), baseEvent(eventID, repoID), rules.EventContext{
		EventID: eventID, IssueNumber: 1,
	}, []rules.ActionIntent{{
		RuleID: ruleID, EventID: eventID, ActionType: rules.ActionGitHubLabel,
		Config: json.RawMessage(`{"label":"x"}`),
	}})
	if !errors.Is(err, events.ErrActionsFailed) {
		t.Fatalf("got %v", err)
	}
	if !events.IsPermanent(err) {
		t.Fatal("expected permanent for parent event")
	}
}

func TestIdempotencyKey(t *testing.T) {
	e, r := uuid.New(), uuid.New()
	k1 := actions.IdempotencyKey(e, r, "github_label")
	k2 := actions.IdempotencyKey(e, r, "github_label")
	k3 := actions.IdempotencyKey(e, r, "github_comment")
	if k1 != k2 || k1 == k3 {
		t.Fatalf("%s %s %s", k1, k2, k3)
	}
}

func TestProcessor_WithActionsCompletes(t *testing.T) {
	repoID := uuid.New()
	userID := uuid.New()
	repo := store.Repository{ID: repoID, UserID: userID, GitHubRepositoryID: 42, Name: "r", FullName: "o/r", OwnerLogin: "o"}
	kw := "bug"
	ruleID := uuid.New()
	ruleList := []store.Rule{{
		ID: ruleID, RepositoryID: repoID, Name: "Bug Issues", Enabled: true,
		EventType: "issues", Keyword: &kw, ActionType: "github_label",
		ActionConfig: json.RawMessage(`{"label":"automation"}`),
	}}
	payload, _ := json.Marshal(map[string]any{
		"action": "opened", "repository": map[string]any{"id": 42},
		"issue": map[string]any{
			"number": 9, "title": "Bug found", "body": "x",
			"user": map[string]any{"login": "u"}, "labels": []any{},
		},
	})
	ev := store.WebhookEvent{
		ID: uuid.New(), RepositoryID: repoID, DeliveryID: uuid.NewString(),
		EventType: "issues", Action: "opened", Payload: payload, Status: store.WebhookStatusProcessing,
	}
	gh := &fakeGitHub{}
	ex := &actions.Executor{
		Actions: newMemActions(), Repos: memRepos{repo: repo},
		Users: memTokens{enc: encryptToken(t, "tok")}, TokenKey: testKey(),
		GitHub: gh, Slack: &fakeSlack{}, MaxAttempts: 3,
	}
	p := &events.Processor{
		Repos: procRepos{repo: repo},
		Rules: procRules{list: ruleList},
		Actions: ex,
	}
	if err := p.Process(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	if len(gh.labels) != 1 || gh.labels[0].number != 9 {
		t.Fatalf("%+v", gh.labels)
	}
}

type procRepos struct{ repo store.Repository }

func (p procRepos) GetByID(context.Context, uuid.UUID) (store.Repository, error) { return p.repo, nil }
func (p procRepos) GetByGitHubID(context.Context, int64) (store.Repository, error) {
	return p.repo, nil
}

type procRules struct{ list []store.Rule }

func (p procRules) ListEnabledByRepository(context.Context, uuid.UUID) ([]store.Rule, error) {
	return p.list, nil
}

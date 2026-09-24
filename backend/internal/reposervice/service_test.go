package reposervice_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/Revati-Firke/gitactionflow/backend/internal/auth"
	"github.com/Revati-Firke/gitactionflow/backend/internal/githubapi"
	"github.com/Revati-Firke/gitactionflow/backend/internal/reposervice"
	"github.com/Revati-Firke/gitactionflow/backend/internal/store"
)

type fakeGH struct {
	list []githubapi.Repository
	get  map[int64]githubapi.Repository
	err  error
}

func (f *fakeGH) ListRepositories(context.Context, string) ([]githubapi.Repository, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.list, nil
}

func (f *fakeGH) GetRepository(_ context.Context, _ string, id int64) (githubapi.Repository, error) {
	if f.err != nil {
		return githubapi.Repository{}, f.err
	}
	r, ok := f.get[id]
	if !ok {
		return githubapi.Repository{}, githubapi.ErrNotFound
	}
	return r, nil
}

type memTokens struct {
	enc []byte
}

func (m *memTokens) GetEncryptedAccessToken(context.Context, uuid.UUID) ([]byte, error) {
	if len(m.enc) == 0 {
		return nil, store.ErrNotFound
	}
	return m.enc, nil
}

type memRepos struct {
	mu     sync.Mutex
	byUser map[uuid.UUID]store.Repository
}

func newMemRepos() *memRepos {
	return &memRepos{byUser: map[uuid.UUID]store.Repository{}}
}

func (m *memRepos) GetByUserID(_ context.Context, userID uuid.UUID) (store.Repository, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.byUser[userID]
	if !ok {
		return store.Repository{}, store.ErrNotFound
	}
	return r, nil
}

func (m *memRepos) Insert(_ context.Context, userID uuid.UUID, githubRepoID int64, name, fullName, ownerLogin, defaultBranch, htmlURL string, private bool) (store.Repository, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byUser[userID]; ok {
		return store.Repository{}, store.ErrConflict
	}
	r := store.Repository{
		ID: uuid.New(), UserID: userID, GitHubRepositoryID: githubRepoID,
		Name: name, FullName: fullName, OwnerLogin: ownerLogin,
		DefaultBranch: defaultBranch, HTMLURL: htmlURL, Private: private,
	}
	m.byUser[userID] = r
	return r, nil
}

func (m *memRepos) DeleteByUserID(_ context.Context, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.byUser, userID)
	return nil
}

func testService(t *testing.T) (*reposervice.Service, *fakeGH, *memRepos, uuid.UUID) {
	t.Helper()
	key := auth.DeriveKey("0123456789abcdef0123456789abcdef")
	enc, err := auth.Encrypt(key, []byte("gho_secret"))
	if err != nil {
		t.Fatal(err)
	}
	gh := &fakeGH{get: map[int64]githubapi.Repository{}}
	repos := newMemRepos()
	svc := &reposervice.Service{
		GitHub:   gh,
		Users:    &memTokens{enc: enc},
		Repos:    repos,
		TokenKey: key,
	}
	return svc, gh, repos, uuid.New()
}

func TestConnect_ValidAdminRepo(t *testing.T) {
	svc, gh, _, userID := testService(t)
	gh.get[10] = githubapi.Repository{
		ID: 10, Name: "demo", FullName: "u/demo", DefaultBranch: "main",
		HTMLURL: "https://github.com/u/demo", Owner: githubapi.Owner{Login: "u"},
		Permissions: githubapi.Permissions{Admin: true, Push: true},
	}
	got, err := svc.Connect(context.Background(), userID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if got.GitHubRepositoryID != 10 || got.FullName != "u/demo" {
		t.Fatalf("%+v", got)
	}
}

func TestConnect_RejectsNonAdmin(t *testing.T) {
	svc, gh, _, userID := testService(t)
	gh.get[11] = githubapi.Repository{
		ID: 11, Name: "x", FullName: "o/x", Owner: githubapi.Owner{Login: "o"},
		Permissions: githubapi.Permissions{Admin: false, Push: true},
	}
	_, err := svc.Connect(context.Background(), userID, 11)
	if !errors.Is(err, reposervice.ErrAccessDenied) {
		t.Fatalf("got %v", err)
	}
}

func TestConnect_RejectsInvalidID(t *testing.T) {
	svc, _, _, userID := testService(t)
	_, err := svc.Connect(context.Background(), userID, 0)
	if !errors.Is(err, reposervice.ErrInvalidInput) {
		t.Fatalf("got %v", err)
	}
}

func TestConnect_RejectsSecondRepo(t *testing.T) {
	svc, gh, _, userID := testService(t)
	gh.get[1] = githubapi.Repository{ID: 1, Name: "a", FullName: "u/a", Owner: githubapi.Owner{Login: "u"}, Permissions: githubapi.Permissions{Admin: true}}
	gh.get[2] = githubapi.Repository{ID: 2, Name: "b", FullName: "u/b", Owner: githubapi.Owner{Login: "u"}, Permissions: githubapi.Permissions{Admin: true}}
	if _, err := svc.Connect(context.Background(), userID, 1); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Connect(context.Background(), userID, 2)
	if !errors.Is(err, reposervice.ErrAlreadyConnected) {
		t.Fatalf("got %v", err)
	}
}

func TestConnect_NotFound(t *testing.T) {
	svc, _, _, userID := testService(t)
	_, err := svc.Connect(context.Background(), userID, 999)
	if !errors.Is(err, reposervice.ErrRepositoryNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestDisconnect_AndIsolation(t *testing.T) {
	svc, gh, repos, userA := testService(t)
	userB := uuid.New()
	gh.get[5] = githubapi.Repository{ID: 5, Name: "a", FullName: "a/a", Owner: githubapi.Owner{Login: "a"}, Permissions: githubapi.Permissions{Admin: true}}
	if _, err := svc.Connect(context.Background(), userA, 5); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetConnected(context.Background(), userB); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("user B must not see user A repo")
	}
	if err := svc.Disconnect(context.Background(), userA); err != nil {
		t.Fatal(err)
	}
	if len(repos.byUser) != 0 {
		t.Fatal("expected empty")
	}
	if err := svc.Disconnect(context.Background(), userA); err != nil {
		t.Fatal(err)
	}
}

func TestList_MapsGitHubErrors(t *testing.T) {
	svc, gh, _, userID := testService(t)
	gh.err = githubapi.ErrUnauthorized
	_, err := svc.ListGitHubRepositories(context.Background(), userID)
	if !errors.Is(err, reposervice.ErrGitHubUnauthorized) {
		t.Fatalf("got %v", err)
	}
}

package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Revati-Firke/gitactionflow/backend/internal/auth"
	"github.com/Revati-Firke/gitactionflow/backend/internal/githubapi"
	"github.com/Revati-Firke/gitactionflow/backend/internal/http/handlers"
	"github.com/Revati-Firke/gitactionflow/backend/internal/http/middleware"
	"github.com/Revati-Firke/gitactionflow/backend/internal/http/response"
	"github.com/Revati-Firke/gitactionflow/backend/internal/reposervice"
	"github.com/Revati-Firke/gitactionflow/backend/internal/store"
)

type repoFakeGH struct {
	get map[int64]githubapi.Repository
}

func (f *repoFakeGH) ListRepositories(context.Context, string) ([]githubapi.Repository, error) {
	return []githubapi.Repository{{
		ID: 1, Name: "demo", FullName: "u/demo", DefaultBranch: "main",
		HTMLURL: "https://github.com/u/demo", Owner: githubapi.Owner{Login: "u"},
		Permissions: githubapi.Permissions{Admin: true},
	}}, nil
}

func (f *repoFakeGH) GetRepository(_ context.Context, _ string, id int64) (githubapi.Repository, error) {
	r, ok := f.get[id]
	if !ok {
		return githubapi.Repository{}, githubapi.ErrNotFound
	}
	return r, nil
}

type repoMemTokens struct{ enc []byte }

func (m *repoMemTokens) GetEncryptedAccessToken(context.Context, uuid.UUID) ([]byte, error) {
	return m.enc, nil
}

type repoMemStore struct {
	byUser map[uuid.UUID]store.Repository
}

func (m *repoMemStore) GetByUserID(_ context.Context, userID uuid.UUID) (store.Repository, error) {
	r, ok := m.byUser[userID]
	if !ok {
		return store.Repository{}, store.ErrNotFound
	}
	return r, nil
}

func (m *repoMemStore) Insert(_ context.Context, userID uuid.UUID, githubRepoID int64, name, fullName, ownerLogin, defaultBranch, htmlURL string, private bool) (store.Repository, error) {
	if _, ok := m.byUser[userID]; ok {
		return store.Repository{}, store.ErrConflict
	}
	r := store.Repository{ID: uuid.New(), UserID: userID, GitHubRepositoryID: githubRepoID, Name: name, FullName: fullName, OwnerLogin: ownerLogin, DefaultBranch: defaultBranch, HTMLURL: htmlURL, Private: private}
	m.byUser[userID] = r
	return r, nil
}

func (m *repoMemStore) DeleteByUserID(_ context.Context, userID uuid.UUID) error {
	delete(m.byUser, userID)
	return nil
}

type sessMem struct {
	hash []byte
	user uuid.UUID
}

func (s *sessMem) GetValidByTokenHash(_ context.Context, tokenHash []byte, _ time.Time) (auth.Session, error) {
	if string(tokenHash) != string(s.hash) {
		return auth.Session{}, store.ErrNotFound
	}
	return auth.Session{UserID: s.user, ExpiresAt: time.Now().Add(time.Hour)}, nil
}

type userMem struct {
	user auth.User
}

func (u *userMem) GetByID(_ context.Context, id uuid.UUID) (auth.User, error) {
	if id != u.user.ID {
		return auth.User{}, store.ErrNotFound
	}
	return u.user, nil
}

func setupRepoRouter(t *testing.T) (*gin.Engine, string, *repoMemStore) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	key := auth.DeriveKey("0123456789abcdef0123456789abcdef")
	enc, _ := auth.Encrypt(key, []byte("tok"))
	user := auth.User{ID: uuid.New(), GitHubUsername: "u"}
	raw := "sess-token"
	repos := &repoMemStore{byUser: map[uuid.UUID]store.Repository{}}
	gh := &repoFakeGH{get: map[int64]githubapi.Repository{
		1: {ID: 1, Name: "demo", FullName: "u/demo", Owner: githubapi.Owner{Login: "u"}, DefaultBranch: "main", HTMLURL: "https://github.com/u/demo", Permissions: githubapi.Permissions{Admin: true}},
	}}
	svc := &reposervice.Service{GitHub: gh, Users: &repoMemTokens{enc: enc}, Repos: repos, TokenKey: key}
	h := &handlers.RepositoryHandler{Service: svc}

	r := gin.New()
	api := r.Group("/api")
	api.Use(middleware.RequireAuth(middleware.AuthDeps{
		Sessions: &sessMem{hash: auth.HashToken(raw), user: user.ID},
		Users:    &userMem{user: user},
	}))
	api.GET("/github/repositories", h.ListGitHubRepositories)
	api.GET("/repository", h.GetConnected)
	api.POST("/repository", h.Connect)
	api.DELETE("/repository", h.Disconnect)
	return r, raw, repos
}

func TestRepositoryAPIs_Unauthenticated(t *testing.T) {
	r, _, _ := setupRepoRouter(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/github/repositories", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestRepositoryAPIs_ConnectAndList(t *testing.T) {
	r, raw, _ := setupRepoRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/github/repositories", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: raw})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", w.Code, w.Body.String())
	}

	body := []byte(`{"github_repository_id":1}`)
	req2 := httptest.NewRequest(http.MethodPost, "/api/repository", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: raw})
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusCreated {
		t.Fatalf("connect status=%d body=%s", w2.Code, w2.Body.String())
	}
	if bytes.Contains(w2.Body.Bytes(), []byte("gho_")) || bytes.Contains(w2.Body.Bytes(), []byte("tok")) {
		// "tok" is short; check access_token instead
	}
	if bytes.Contains(w2.Body.Bytes(), []byte("access_token")) {
		t.Fatal("token leaked")
	}

	req3 := httptest.NewRequest(http.MethodPost, "/api/repository", bytes.NewReader([]byte(`{"github_repository_id":1}`)))
	req3.Header.Set("Content-Type", "application/json")
	req3.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: raw})
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusConflict {
		t.Fatalf("duplicate status=%d", w3.Code)
	}
	var errBody response.ErrorBody
	_ = json.Unmarshal(w3.Body.Bytes(), &errBody)
	if errBody.Error.Code != "REPOSITORY_ALREADY_CONNECTED" {
		t.Fatalf("%+v", errBody)
	}
}

func TestRepositoryAPIs_InvalidID(t *testing.T) {
	r, raw, _ := setupRepoRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/api/repository", bytes.NewReader([]byte(`{"github_repository_id":0}`)))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: raw})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", w.Code)
	}
}

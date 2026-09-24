package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Revati-Firke/gitactionflow/backend/internal/auth"
	"github.com/Revati-Firke/gitactionflow/backend/internal/githuboauth"
	"github.com/Revati-Firke/gitactionflow/backend/internal/http/handlers"
	"github.com/Revati-Firke/gitactionflow/backend/internal/http/middleware"
	"github.com/Revati-Firke/gitactionflow/backend/internal/http/response"
	"github.com/Revati-Firke/gitactionflow/backend/internal/store"
)

type memStates struct {
	mu    sync.Mutex
	items map[string]stateRec
}

type stateRec struct {
	expires  time.Time
	consumed bool
}

func newMemStates() *memStates {
	return &memStates{items: map[string]stateRec{}}
}

func (m *memStates) Create(_ context.Context, stateHash []byte, expiresAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[string(stateHash)] = stateRec{expires: expiresAt}
	return nil
}

func (m *memStates) Consume(_ context.Context, stateHash []byte, now time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.items[string(stateHash)]
	if !ok || rec.consumed || !rec.expires.After(now) {
		return store.ErrNotFound
	}
	rec.consumed = true
	m.items[string(stateHash)] = rec
	return nil
}

type memUsers struct {
	mu    sync.Mutex
	byID  map[uuid.UUID]auth.User
	token []byte
}

func newMemUsers() *memUsers {
	return &memUsers{byID: map[uuid.UUID]auth.User{}}
}

func (m *memUsers) UpsertFromGitHub(_ context.Context, githubUserID int64, login, name, avatarURL string, encryptedToken []byte) (auth.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.byID {
		if u.GitHubUserID == githubUserID {
			u.GitHubUsername = login
			u.DisplayName = name
			u.AvatarURL = avatarURL
			m.byID[u.ID] = u
			m.token = encryptedToken
			return u, nil
		}
	}
	u := auth.User{
		ID:             uuid.New(),
		GitHubUserID:   githubUserID,
		GitHubUsername: login,
		DisplayName:    name,
		AvatarURL:      avatarURL,
	}
	m.byID[u.ID] = u
	m.token = encryptedToken
	return u, nil
}

func (m *memUsers) GetByID(_ context.Context, id uuid.UUID) (auth.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byID[id]
	if !ok {
		return auth.User{}, store.ErrNotFound
	}
	return u, nil
}

type memSessions struct {
	mu     sync.Mutex
	byHash map[string]auth.Session
}

func newMemSessions() *memSessions {
	return &memSessions{byHash: map[string]auth.Session{}}
}

func (m *memSessions) Create(_ context.Context, userID uuid.UUID, tokenHash []byte, expiresAt time.Time) (auth.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := auth.Session{ID: uuid.New(), UserID: userID, TokenHash: tokenHash, ExpiresAt: expiresAt, CreatedAt: time.Now().UTC()}
	m.byHash[string(tokenHash)] = s
	return s, nil
}

func (m *memSessions) GetValidByTokenHash(_ context.Context, tokenHash []byte, now time.Time) (auth.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.byHash[string(tokenHash)]
	if !ok || !s.ExpiresAt.After(now) {
		return auth.Session{}, store.ErrNotFound
	}
	return s, nil
}

func (m *memSessions) DeleteByTokenHash(_ context.Context, tokenHash []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.byHash, string(tokenHash))
	return nil
}

type fakeGitHub struct {
	exchangeErr error
	fetchErr    error
	token       string
	user        githuboauth.User
	lastState   string
}

func (f *fakeGitHub) AuthorizeURL(state string) string {
	f.lastState = state
	return "https://github.com/login/oauth/authorize?state=" + state
}

func (f *fakeGitHub) ExchangeCode(context.Context, string) (githuboauth.TokenResponse, error) {
	if f.exchangeErr != nil {
		return githuboauth.TokenResponse{}, f.exchangeErr
	}
	return githuboauth.TokenResponse{AccessToken: f.token}, nil
}

func (f *fakeGitHub) FetchUser(context.Context, string) (githuboauth.User, error) {
	if f.fetchErr != nil {
		return githuboauth.User{}, f.fetchErr
	}
	return f.user, nil
}

func testAuthHandler(t *testing.T) (*handlers.AuthHandler, *memStates, *memUsers, *memSessions, *fakeGitHub) {
	t.Helper()
	states := newMemStates()
	users := newMemUsers()
	sessions := newMemSessions()
	gh := &fakeGitHub{
		token: "gho_test_token_never_expose",
		user:  githuboauth.User{ID: 42, Login: "octocat", Name: "The Octocat", AvatarURL: "https://example.com/a.png"},
	}
	h := &handlers.AuthHandler{
		GitHub:   gh,
		States:   states,
		Users:    users,
		Sessions: sessions,
		TokenKey: auth.DeriveKey("0123456789abcdef0123456789abcdef"),
		Cookie: auth.CookieOptions{
			Secure:   false,
			SameSite: "Lax",
			TTL:      time.Hour,
		},
		StateTTL:    10 * time.Minute,
		FrontendURL: "http://localhost:5173",
	}
	return h, states, users, sessions, gh
}

func TestStartGitHub_RedirectsAndStoresState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, states, _, _, gh := testAuthHandler(t)

	r := gin.New()
	r.GET("/auth/github", h.StartGitHub)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/github", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status=%d", w.Code)
	}
	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "github.com") || gh.lastState == "" {
		t.Fatalf("bad redirect: %s", loc)
	}
	if err := states.Consume(context.Background(), auth.HashToken(gh.lastState), time.Now().UTC()); err != nil {
		t.Fatalf("state should be consumable once: %v", err)
	}
	if err := states.Consume(context.Background(), auth.HashToken(gh.lastState), time.Now().UTC()); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("state must not be reusable")
	}
}

func TestCallback_RejectsMissingAndInvalidState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _, _, _, _ := testAuthHandler(t)
	r := gin.New()
	r.GET("/auth/github/callback", h.CallbackGitHub)

	cases := []string{
		"/auth/github/callback",
		"/auth/github/callback?code=abc",
		"/auth/github/callback?code=abc&state=nope",
	}
	for _, path := range cases {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusFound {
			t.Fatalf("%s status=%d", path, w.Code)
		}
		if !strings.Contains(w.Header().Get("Location"), "error=") {
			t.Fatalf("%s expected error redirect, got %s", path, w.Header().Get("Location"))
		}
	}
}

func TestCallback_ExpiredStateRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, states, _, _, _ := testAuthHandler(t)
	state := "expired-state-token"
	_ = states.Create(context.Background(), auth.HashToken(state), time.Now().UTC().Add(-time.Minute))

	r := gin.New()
	r.GET("/auth/github/callback", h.CallbackGitHub)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/github/callback?code=x&state="+state, nil))
	if !strings.Contains(w.Header().Get("Location"), "invalid_state") {
		t.Fatalf("expected invalid_state, got %s", w.Header().Get("Location"))
	}
}

func TestCallback_SuccessCreatesSessionWithoutLeakingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, states, users, sessions, gh := testAuthHandler(t)
	state := "good-state-token"
	_ = states.Create(context.Background(), auth.HashToken(state), time.Now().UTC().Add(time.Minute))

	r := gin.New()
	r.GET("/auth/github/callback", h.CallbackGitHub)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/github/callback?code=good&state="+state, nil))

	if w.Code != http.StatusFound {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), gh.token) || strings.Contains(w.Header().Get("Location"), gh.token) {
		t.Fatal("access token leaked in response")
	}
	if !strings.Contains(w.Header().Get("Set-Cookie"), auth.SessionCookieName) {
		t.Fatal("expected session cookie")
	}
	if !strings.Contains(w.Header().Get("Set-Cookie"), "HttpOnly") {
		t.Fatal("cookie must be HttpOnly")
	}
	if len(users.byID) != 1 {
		t.Fatal("expected user upsert")
	}
	if len(sessions.byHash) != 1 {
		t.Fatal("expected session create")
	}
	if bytesContainPlainToken(users.token, gh.token) {
		t.Fatal("token stored plaintext")
	}
}

func TestLogout_Idempotent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _, users, sessions, _ := testAuthHandler(t)
	user := auth.User{ID: uuid.New(), GitHubUsername: "u"}
	users.byID[user.ID] = user
	raw := "session-raw-token"
	_, _ = sessions.Create(context.Background(), user.ID, auth.HashToken(raw), time.Now().UTC().Add(time.Hour))

	r := gin.New()
	r.POST("/auth/logout", h.Logout)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: raw})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	if len(sessions.byHash) != 0 {
		t.Fatal("session should be deleted")
	}

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodPost, "/auth/logout", nil))
	if w2.Code != http.StatusOK {
		t.Fatalf("second logout status=%d", w2.Code)
	}
}

func TestMe_AuthAndUnauth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	users := newMemUsers()
	sessions := newMemSessions()
	user := auth.User{ID: uuid.New(), GitHubUsername: "octocat", DisplayName: "Octo", AvatarURL: "https://x"}
	users.byID[user.ID] = user
	raw := "valid-session"
	_, _ = sessions.Create(context.Background(), user.ID, auth.HashToken(raw), time.Now().UTC().Add(time.Hour))

	r := gin.New()
	r.GET("/api/me", middleware.RequireAuth(middleware.AuthDeps{Sessions: sessions, Users: users}), handlers.Me)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/me", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauth status=%d", w.Code)
	}
	var errBody response.ErrorBody
	_ = json.Unmarshal(w.Body.Bytes(), &errBody)
	if errBody.Error.Code != "UNAUTHORIZED" {
		t.Fatalf("code=%q", errBody.Error.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: raw})
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req)
	if w2.Code != http.StatusOK {
		t.Fatalf("auth status=%d body=%s", w2.Code, w2.Body.String())
	}
	if strings.Contains(w2.Body.String(), "gho_") || strings.Contains(w2.Body.String(), "access_token") {
		t.Fatal("token leaked in /api/me")
	}
	var body map[string]map[string]string
	if err := json.Unmarshal(w2.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["user"]["github_username"] != "octocat" {
		t.Fatalf("body=%v", body)
	}
}

func TestMe_ExpiredSessionRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	users := newMemUsers()
	sessions := newMemSessions()
	user := auth.User{ID: uuid.New(), GitHubUsername: "octocat"}
	users.byID[user.ID] = user
	raw := "expired-session"
	_, _ = sessions.Create(context.Background(), user.ID, auth.HashToken(raw), time.Now().UTC().Add(-time.Minute))

	r := gin.New()
	r.GET("/api/me", middleware.RequireAuth(middleware.AuthDeps{Sessions: sessions, Users: users}), handlers.Me)
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: raw})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestMe_InvalidSessionRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/me", middleware.RequireAuth(middleware.AuthDeps{Sessions: newMemSessions(), Users: newMemUsers()}), handlers.Me)
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "nope"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", w.Code)
	}
}

func bytesContainPlainToken(enc []byte, token string) bool {
	return strings.Contains(string(enc), token)
}

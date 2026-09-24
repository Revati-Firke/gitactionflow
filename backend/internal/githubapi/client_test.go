package githubapi_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Revati-Firke/gitactionflow/backend/internal/githubapi"
)

func TestListRepositories_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Fatalf("missing auth header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":1,"name":"a","full_name":"u/a","private":false,"default_branch":"main","html_url":"https://github.com/u/a","owner":{"login":"u"},"permissions":{"admin":true,"push":true,"pull":true}}]`))
	}))
	defer srv.Close()

	c := githubapi.New(srv.Client()).WithBaseURL(srv.URL)
	repos, err := c.ListRepositories(context.Background(), "tok")
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 || repos[0].ID != 1 {
		t.Fatalf("%+v", repos)
	}
}

func TestGetRepository_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	c := githubapi.New(srv.Client()).WithBaseURL(srv.URL)
	_, err := c.GetRepository(context.Background(), "tok", 99)
	if !errors.Is(err, githubapi.ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestGetRepository_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	c := githubapi.New(srv.Client()).WithBaseURL(srv.URL)
	_, err := c.GetRepository(context.Background(), "bad", 1)
	if !errors.Is(err, githubapi.ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
}

func TestListRepositories_RateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	c := githubapi.New(srv.Client()).WithBaseURL(srv.URL)
	_, err := c.ListRepositories(context.Background(), "tok")
	if !errors.Is(err, githubapi.ErrRateLimited) {
		t.Fatalf("got %v", err)
	}
}

func TestGetRepository_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":7,"name":"x","full_name":"o/x","private":true,"default_branch":"dev","html_url":"https://github.com/o/x","owner":{"login":"o"},"permissions":{"admin":true}}`))
	}))
	defer srv.Close()
	c := githubapi.New(srv.Client()).WithBaseURL(srv.URL)
	repo, err := c.GetRepository(context.Background(), "tok", 7)
	if err != nil || repo.ID != 7 || !repo.Permissions.Admin {
		t.Fatalf("%+v %v", repo, err)
	}
}

func TestAddIssueLabels_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/repos/o/r/issues/3/labels" {
			t.Fatalf("%s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()
	c := githubapi.New(srv.Client()).WithBaseURL(srv.URL)
	if err := c.AddIssueLabels(context.Background(), "tok", "o", "r", 3, []string{"automation"}); err != nil {
		t.Fatal(err)
	}
}

func TestCreateIssueComment_Retryable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	c := githubapi.New(srv.Client()).WithBaseURL(srv.URL)
	err := c.CreateIssueComment(context.Background(), "tok", "o", "r", 1, "hi")
	if !errors.Is(err, githubapi.ErrRetryable) {
		t.Fatalf("got %v", err)
	}
}

func TestCreateIssueComment_Permanent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
	}))
	defer srv.Close()
	c := githubapi.New(srv.Client()).WithBaseURL(srv.URL)
	err := c.CreateIssueComment(context.Background(), "tok", "o", "r", 1, "hi")
	if !errors.Is(err, githubapi.ErrPermanent) {
		t.Fatalf("got %v", err)
	}
}

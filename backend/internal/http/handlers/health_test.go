package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/Revati-Firke/gitactionflow/backend/internal/http/response"
	"github.com/Revati-Firke/gitactionflow/backend/internal/http/router"
)

func TestHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := router.New(router.Dependencies{AppEnv: "test"})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status = %q, want ok", body["status"])
	}
}

type stubPinger struct {
	err error
}

func (s stubPinger) Ping(context.Context) error {
	return s.err
}

func TestReady_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := router.New(router.Dependencies{
		AppEnv: "test",
		DB:     stubPinger{},
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body["status"] != "ready" {
		t.Fatalf("status = %q, want ready", body["status"])
	}
}

func TestReady_DBFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := router.New(router.Dependencies{
		AppEnv: "test",
		DB:     stubPinger{err: errors.New("connection refused")},
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}

	var body response.ErrorBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body.Error.Code != "NOT_READY" {
		t.Fatalf("code = %q, want NOT_READY", body.Error.Code)
	}
	if body.Error.Message != "database unavailable" {
		t.Fatalf("message = %q", body.Error.Message)
	}
	if strings.Contains(w.Body.String(), "connection refused") {
		t.Fatalf("response leaked internal error: %s", w.Body.String())
	}
}

func TestReady_NilDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := router.New(router.Dependencies{AppEnv: "test", DB: nil})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}

func TestJSONError_Shape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/err", func(c *gin.Context) {
		response.JSONError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid input")
	})

	req := httptest.NewRequest(http.MethodGet, "/err", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var body response.ErrorBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body.Error.Code != "BAD_REQUEST" || body.Error.Message != "invalid input" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

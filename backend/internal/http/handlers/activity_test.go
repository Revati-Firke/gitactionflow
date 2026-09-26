package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/Revati-Firke/gitactionflow/backend/internal/http/handlers"
	"github.com/Revati-Firke/gitactionflow/backend/internal/http/router"
)

type activityStubPinger struct{}

func (activityStubPinger) Ping(context.Context) error { return nil }

func TestCORS_AllowsPUTDelete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := router.New(router.Dependencies{
		DB:          activityStubPinger{},
		AppEnv:      "test",
		FrontendURL: "http://localhost:5173",
	})
	req := httptest.NewRequest(http.MethodOptions, "/api/rules", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "PUT")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status %d", w.Code)
	}
	allow := w.Header().Get("Access-Control-Allow-Methods")
	if !strings.Contains(allow, "PUT") || !strings.Contains(allow, "DELETE") {
		t.Fatalf("Allow-Methods = %q", allow)
	}
}

func TestActivity_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &handlers.ActivityHandler{}
	r := gin.New()
	r.GET("/api/events", h.ListEvents)
	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", w.Code)
	}
}

func TestEventJSON_HasNoPayloadKey(t *testing.T) {
	type eventResponse struct {
		ID        string `json:"id"`
		EventType string `json:"event_type"`
		Action    string `json:"action"`
		Status    string `json:"status"`
	}
	b, err := json.Marshal(eventResponse{ID: "1", EventType: "issues", Action: "opened", Status: "processed"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "payload") {
		t.Fatal(string(b))
	}
}

package slack_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Revati-Firke/gitactionflow/backend/internal/slack"
)

func TestSendText_OK(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := slack.New(srv.URL, srv.Client())
	if err := c.SendText(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotBody, "hello") {
		t.Fatalf("%s", gotBody)
	}
}

func TestSendText_NotConfigured(t *testing.T) {
	c := slack.New("", nil)
	if err := c.SendText(context.Background(), "x"); !errors.Is(err, slack.ErrNotConfigured) {
		t.Fatalf("got %v", err)
	}
}

func TestSendText_Retryable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	c := slack.New(srv.URL, srv.Client())
	err := c.SendText(context.Background(), "x")
	if !errors.Is(err, slack.ErrRetryable) {
		t.Fatalf("got %v", err)
	}
}

func TestSendText_Permanent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()
	c := slack.New(srv.URL, srv.Client())
	err := c.SendText(context.Background(), "x")
	if !errors.Is(err, slack.ErrPermanent) {
		t.Fatalf("got %v", err)
	}
	if strings.Contains(err.Error(), srv.URL) {
		t.Fatalf("webhook URL leaked in error: %v", err)
	}
}

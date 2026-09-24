package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var (
	ErrNotConfigured = errors.New("slack webhook not configured")
	ErrRetryable     = errors.New("slack retryable error")
	ErrPermanent     = errors.New("slack permanent error")
)

// Client posts messages to a Slack Incoming Webhook.
type Client struct {
	httpClient *http.Client
	webhookURL string
}

// New creates a Slack webhook client. Empty webhookURL means not configured.
func New(webhookURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{httpClient: httpClient, webhookURL: strings.TrimSpace(webhookURL)}
}

// SendText posts a simple text message. Never logs the webhook URL.
func (c *Client) SendText(ctx context.Context, text string) error {
	if c.webhookURL == "" {
		return ErrNotConfigured
	}
	body, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRetryable, err)
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 1<<20))

	switch {
	case res.StatusCode >= 200 && res.StatusCode < 300:
		return nil
	case res.StatusCode == http.StatusTooManyRequests || res.StatusCode >= 500:
		return fmt.Errorf("%w: status %d", ErrRetryable, res.StatusCode)
	default:
		return fmt.Errorf("%w: status %d", ErrPermanent, res.StatusCode)
	}
}

package ai

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

// Suggestion is validated, untrusted model output after schema checks.
type Suggestion struct {
	Summary         string   `json:"summary"`
	SuggestedLabels []string `json:"suggested_labels"`
	Priority        string   `json:"priority"`
}

// Suggester produces structured issue/PR assistance. Implementations must not
// execute actions — callers decide how to use validated output.
type Suggester interface {
	Suggest(ctx context.Context, title, body string) (Suggestion, error)
}

var (
	ErrDisabled     = errors.New("ai disabled")
	ErrInvalidOutput = errors.New("ai invalid output")
)

var allowedPriorities = map[string]struct{}{
	"low": {}, "medium": {}, "high": {}, "unknown": {},
}

var allowedLabels = map[string]struct{}{
	"bug": {}, "enhancement": {}, "documentation": {}, "question": {},
	"help wanted": {}, "good first issue": {}, "priority": {}, "wontfix": {},
}

// ValidateSuggestion enforces a strict allowlist schema on model output.
func ValidateSuggestion(s Suggestion) (Suggestion, error) {
	out := Suggestion{
		Summary:  strings.TrimSpace(s.Summary),
		Priority: strings.ToLower(strings.TrimSpace(s.Priority)),
	}
	if out.Summary == "" {
		return Suggestion{}, fmt.Errorf("%w: summary required", ErrInvalidOutput)
	}
	if len(out.Summary) > 1000 {
		out.Summary = out.Summary[:1000]
	}
	if out.Priority == "" {
		out.Priority = "unknown"
	}
	if _, ok := allowedPriorities[out.Priority]; !ok {
		out.Priority = "unknown"
	}
	seen := map[string]struct{}{}
	for _, l := range s.SuggestedLabels {
		l = strings.ToLower(strings.TrimSpace(l))
		if l == "" {
			continue
		}
		if _, ok := allowedLabels[l]; !ok {
			continue
		}
		if _, dup := seen[l]; dup {
			continue
		}
		seen[l] = struct{}{}
		out.SuggestedLabels = append(out.SuggestedLabels, l)
		if len(out.SuggestedLabels) >= 5 {
			break
		}
	}
	return out, nil
}

// Noop always returns ErrDisabled.
type Noop struct{}

func (Noop) Suggest(context.Context, string, string) (Suggestion, error) {
	return Suggestion{}, ErrDisabled
}

// Config selects an optional provider. Empty provider or missing key → Noop.
type Config struct {
	Enabled  bool
	Provider string // gemini | groq
	APIKey   string
	HTTP     *http.Client
}

// New builds a Suggester from config. Never fails; returns Noop when disabled.
func New(cfg Config) Suggester {
	if !cfg.Enabled || strings.TrimSpace(cfg.APIKey) == "" {
		return Noop{}
	}
	client := cfg.HTTP
	if client == nil {
		client = &http.Client{Timeout: 12 * time.Second}
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "groq":
		return &groqProvider{key: cfg.APIKey, http: client}
	case "gemini", "":
		return &geminiProvider{key: cfg.APIKey, http: client}
	default:
		return Noop{}
	}
}

const systemPrompt = `You assist a GitHub automation bot. Reply with ONLY compact JSON (no markdown):
{"summary":"one short sentence","suggested_labels":["bug"],"priority":"low|medium|high|unknown"}
Labels must be from: bug, enhancement, documentation, question, help wanted, good first issue, priority, wontfix.
Do not invent other keys. Do not include secrets or URLs to call.`

func parseModelJSON(raw string) (Suggestion, error) {
	raw = strings.TrimSpace(raw)
	if i := strings.Index(raw, "{"); i >= 0 {
		if j := strings.LastIndex(raw, "}"); j > i {
			raw = raw[i : j+1]
		}
	}
	var s Suggestion
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return Suggestion{}, fmt.Errorf("%w: %v", ErrInvalidOutput, err)
	}
	return ValidateSuggestion(s)
}

type geminiProvider struct {
	key  string
	http *http.Client
}

func (g *geminiProvider) Suggest(ctx context.Context, title, body string) (Suggestion, error) {
	endpoint := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=" + g.key
	payload := map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]string{{"text": systemPrompt + "\n\nTitle: " + title + "\n\nBody:\n" + truncate(body, 4000)}}},
		},
		"generationConfig": map[string]any{"temperature": 0.2, "maxOutputTokens": 256},
	}
	b, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return Suggestion{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := g.http.Do(req)
	if err != nil {
		return Suggestion{}, err
	}
	defer res.Body.Close()
	bodyBytes, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		return Suggestion{}, fmt.Errorf("gemini status %d", res.StatusCode)
	}
	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
		return Suggestion{}, err
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return Suggestion{}, ErrInvalidOutput
	}
	return parseModelJSON(parsed.Candidates[0].Content.Parts[0].Text)
}

type groqProvider struct {
	key  string
	http *http.Client
}

func (g *groqProvider) Suggest(ctx context.Context, title, body string) (Suggestion, error) {
	payload := map[string]any{
		"model": "llama-3.1-8b-instant",
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": "Title: " + title + "\n\nBody:\n" + truncate(body, 4000)},
		},
		"temperature": 0.2,
		"max_tokens":  256,
	}
	b, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.groq.com/openai/v1/chat/completions", bytes.NewReader(b))
	if err != nil {
		return Suggestion{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.key)
	res, err := g.http.Do(req)
	if err != nil {
		return Suggestion{}, err
	}
	defer res.Body.Close()
	bodyBytes, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		return Suggestion{}, fmt.Errorf("groq status %d", res.StatusCode)
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
		return Suggestion{}, err
	}
	if len(parsed.Choices) == 0 {
		return Suggestion{}, ErrInvalidOutput
	}
	return parseModelJSON(parsed.Choices[0].Message.Content)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

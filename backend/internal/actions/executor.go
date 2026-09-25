package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Revati-Firke/gitactionflow/backend/internal/ai"
	"github.com/Revati-Firke/gitactionflow/backend/internal/auth"
	"github.com/Revati-Firke/gitactionflow/backend/internal/events"
	"github.com/Revati-Firke/gitactionflow/backend/internal/githubapi"
	"github.com/Revati-Firke/gitactionflow/backend/internal/rules"
	"github.com/Revati-Firke/gitactionflow/backend/internal/slack"
	"github.com/Revati-Firke/gitactionflow/backend/internal/store"
)

// GitHubWriter performs GitHub write operations.
type GitHubWriter interface {
	AddIssueLabels(ctx context.Context, accessToken, owner, repo string, issueNumber int, labels []string) error
	CreateIssueComment(ctx context.Context, accessToken, owner, repo string, issueNumber int, comment string) error
}

// SlackSender sends Slack notifications.
type SlackSender interface {
	SendText(ctx context.Context, text string) error
}

// ActionStore persists actions.
type ActionStore interface {
	CreateOrGet(ctx context.Context, eventID uuid.UUID, ruleID uuid.UUID, actionType string, config json.RawMessage, idempotencyKey string, maxAttempts int) (store.Action, bool, error)
	ListByEventID(ctx context.Context, eventID uuid.UUID) ([]store.Action, error)
	MarkProcessing(ctx context.Context, id uuid.UUID) (store.Action, error)
	MarkCompleted(ctx context.Context, id uuid.UUID) (store.Action, error)
	MarkFailedImmediately(ctx context.Context, id uuid.UUID, errMsg string) (store.Action, error)
	ScheduleRetry(ctx context.Context, id uuid.UUID, errMsg string, backoff time.Duration) (store.Action, error)
}

// TokenProvider decrypts the connected user's GitHub token.
type TokenProvider interface {
	GetEncryptedAccessToken(ctx context.Context, userID uuid.UUID) ([]byte, error)
}

// RepoProvider loads repository metadata.
type RepoProvider interface {
	GetByID(ctx context.Context, id uuid.UUID) (store.Repository, error)
}

// Executor creates durable actions from intents and executes them.
type Executor struct {
	Actions     ActionStore
	Repos        RepoProvider
	Users       TokenProvider
	TokenKey    []byte
	GitHub      GitHubWriter
	Slack       SlackSender
	AI          ai.Suggester // optional; nil or Noop skips AI
	MaxAttempts int
	Log         *slog.Logger
}

// IdempotencyKey builds the unique key for an event/rule/action triple.
func IdempotencyKey(eventID, ruleID uuid.UUID, actionType string) string {
	return eventID.String() + ":" + ruleID.String() + ":" + actionType
}

// EnsureAndExecute persists intents then runs ready actions for the event.
// Returns nil only when every action for the event is completed (or there were none).
func (e *Executor) EnsureAndExecute(ctx context.Context, ev store.WebhookEvent, evCtx rules.EventContext, intents []rules.ActionIntent) error {
	log := e.Log
	if log == nil {
		log = slog.Default()
	}
	maxAttempts := e.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	for _, intent := range intents {
		key := IdempotencyKey(intent.EventID, intent.RuleID, intent.ActionType)
		a, created, err := e.Actions.CreateOrGet(ctx, intent.EventID, intent.RuleID, intent.ActionType, intent.Config, key, maxAttempts)
		if err != nil {
			return fmt.Errorf("persist action: %w", err)
		}
		if created {
			log.Info("action created",
				"event_id", ev.ID.String(),
				"rule_id", intent.RuleID.String(),
				"action_id", a.ID.String(),
				"action_type", intent.ActionType,
			)
		}
	}

	list, err := e.Actions.ListByEventID(ctx, ev.ID)
	if err != nil {
		return err
	}
	if len(list) == 0 {
		return nil
	}

	repo, err := e.Repos.GetByID(ctx, ev.RepositoryID)
	if err != nil {
		return fmt.Errorf("load repository: %w", err)
	}

	ruleNames := map[uuid.UUID]string{}
	for _, intent := range intents {
		ruleNames[intent.RuleID] = intent.RuleName
	}

	var suggestion *ai.Suggestion
	if needsAI(list) {
		if s, ok := e.fetchSuggestion(ctx, evCtx); ok {
			suggestion = &s
		}
	}

	for _, a := range list {
		if a.Status == store.ActionStatusCompleted {
			continue
		}
		if a.Status == store.ActionStatusFailed {
			continue
		}
		if a.Status == store.ActionStatusPending {
			if a.NextRetryAt != nil && a.NextRetryAt.After(time.Now().UTC()) {
				continue
			}
			claimed, err := e.Actions.MarkProcessing(ctx, a.ID)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					continue
				}
				return err
			}
			a = claimed
		}
		if a.Status != store.ActionStatusProcessing {
			continue
		}

		ruleName := ""
		if a.RuleID != nil {
			ruleName = ruleNames[*a.RuleID]
		}
		execErr := e.executeOne(ctx, repo, ev, evCtx, a, ruleName, suggestion)
		if execErr == nil {
			if _, err := e.Actions.MarkCompleted(ctx, a.ID); err != nil {
				return err
			}
			log.Info("action completed", "event_id", ev.ID.String(), "action_id", a.ID.String(), "action_type", a.ActionType)
			continue
		}

		msg := sanitizeErr(execErr)
		if isPermanent(execErr) {
			if _, err := e.Actions.MarkFailedImmediately(ctx, a.ID, msg); err != nil {
				return err
			}
			log.Error("action failed permanently", "event_id", ev.ID.String(), "action_id", a.ID.String(), "action_type", a.ActionType, "err", msg)
			continue
		}

		backoff := events.RetryBackoff(a.AttemptCount + 1)
		updated, err := e.Actions.ScheduleRetry(ctx, a.ID, msg, backoff)
		if err != nil {
			return err
		}
		if updated.Status == store.ActionStatusFailed {
			log.Error("action failed exhausted", "event_id", ev.ID.String(), "action_id", a.ID.String(), "action_type", a.ActionType, "err", msg)
		} else {
			log.Info("action scheduled for retry", "event_id", ev.ID.String(), "action_id", a.ID.String(), "action_type", a.ActionType, "backoff", backoff.String())
		}
	}

	final, err := e.Actions.ListByEventID(ctx, ev.ID)
	if err != nil {
		return err
	}
	var pending, failed, completed int
	for _, a := range final {
		switch a.Status {
		case store.ActionStatusCompleted:
			completed++
		case store.ActionStatusFailed:
			failed++
		default:
			pending++
		}
	}
	if failed > 0 && pending == 0 {
		return fmt.Errorf("%w: %d failed", events.ErrActionsFailed, failed)
	}
	if pending > 0 {
		return fmt.Errorf("%w: %d pending", events.ErrActionsIncomplete, pending)
	}
	_ = completed
	return nil
}

func (e *Executor) executeOne(ctx context.Context, repo store.Repository, ev store.WebhookEvent, evCtx rules.EventContext, a store.Action, ruleName string, suggestion *ai.Suggestion) error {
	switch a.ActionType {
	case rules.ActionGitHubLabel:
		return e.execGitHubLabel(ctx, repo, evCtx, a, suggestion)
	case rules.ActionGitHubComment:
		return e.execGitHubComment(ctx, repo, evCtx, a, suggestion)
	case rules.ActionSlackNotification:
		return e.execSlack(ctx, repo, ev, a, ruleName, suggestion)
	default:
		return fmt.Errorf("%w: unknown action_type", events.ErrInvalidEvent)
	}
}

func needsAI(list []store.Action) bool {
	for _, a := range list {
		var m map[string]any
		if json.Unmarshal(a.ActionConfig, &m) != nil {
			continue
		}
		if v, ok := m["use_ai"].(bool); ok && v {
			return true
		}
		if v, ok := m["append_ai_summary"].(bool); ok && v {
			return true
		}
	}
	return false
}

func (e *Executor) fetchSuggestion(ctx context.Context, evCtx rules.EventContext) (ai.Suggestion, bool) {
	log := e.Log
	if log == nil {
		log = slog.Default()
	}
	if e.AI == nil {
		return ai.Suggestion{}, false
	}
	s, err := e.AI.Suggest(ctx, evCtx.Title, evCtx.Body)
	if err != nil {
		log.Info("ai suggestion skipped", "err", err.Error(), "event_type", evCtx.EventType)
		return ai.Suggestion{}, false
	}
	log.Info("ai suggestion ready", "priority", s.Priority, "labels", len(s.SuggestedLabels))
	return s, true
}

func (e *Executor) accessToken(ctx context.Context, userID uuid.UUID) (string, error) {
	enc, err := e.Users.GetEncryptedAccessToken(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("%w: token unavailable", githubapi.ErrPermanent)
	}
	plain, err := auth.Decrypt(e.TokenKey, enc)
	if err != nil {
		return "", fmt.Errorf("%w: token decrypt", githubapi.ErrPermanent)
	}
	return string(plain), nil
}

func (e *Executor) execGitHubLabel(ctx context.Context, repo store.Repository, evCtx rules.EventContext, a store.Action, suggestion *ai.Suggestion) error {
	if evCtx.IssueNumber <= 0 {
		return fmt.Errorf("%w: missing issue number", githubapi.ErrPermanent)
	}
	var cfg struct {
		Label string `json:"label"`
		UseAI bool   `json:"use_ai"`
	}
	if err := json.Unmarshal(a.ActionConfig, &cfg); err != nil {
		return fmt.Errorf("%w: invalid label config", githubapi.ErrPermanent)
	}
	label := strings.TrimSpace(cfg.Label)
	if cfg.UseAI && suggestion != nil && len(suggestion.SuggestedLabels) > 0 {
		label = suggestion.SuggestedLabels[0]
	}
	if label == "" {
		return fmt.Errorf("%w: invalid label config", githubapi.ErrPermanent)
	}
	token, err := e.accessToken(ctx, repo.UserID)
	if err != nil {
		return err
	}
	return e.GitHub.AddIssueLabels(ctx, token, repo.OwnerLogin, repo.Name, evCtx.IssueNumber, []string{label})
}

func (e *Executor) execGitHubComment(ctx context.Context, repo store.Repository, evCtx rules.EventContext, a store.Action, suggestion *ai.Suggestion) error {
	if evCtx.IssueNumber <= 0 {
		return fmt.Errorf("%w: missing issue number", githubapi.ErrPermanent)
	}
	var cfg struct {
		Comment         string `json:"comment"`
		UseAI           bool   `json:"use_ai"`
		AppendAISummary bool   `json:"append_ai_summary"`
	}
	if err := json.Unmarshal(a.ActionConfig, &cfg); err != nil {
		return fmt.Errorf("%w: invalid comment config", githubapi.ErrPermanent)
	}
	comment := strings.TrimSpace(cfg.Comment)
	if (cfg.UseAI || cfg.AppendAISummary) && suggestion != nil && suggestion.Summary != "" {
		if cfg.UseAI && comment == "" {
			comment = suggestion.Summary
		} else if cfg.AppendAISummary {
			if comment == "" {
				comment = suggestion.Summary
			} else {
				comment = comment + "\n\n---\nAI summary: " + suggestion.Summary
			}
		}
	}
	if comment == "" {
		return fmt.Errorf("%w: invalid comment config", githubapi.ErrPermanent)
	}
	token, err := e.accessToken(ctx, repo.UserID)
	if err != nil {
		return err
	}
	return e.GitHub.CreateIssueComment(ctx, token, repo.OwnerLogin, repo.Name, evCtx.IssueNumber, comment)
}

func (e *Executor) execSlack(ctx context.Context, repo store.Repository, ev store.WebhookEvent, a store.Action, ruleName string, suggestion *ai.Suggestion) error {
	var cfg struct {
		Message         string `json:"message"`
		AppendAISummary bool   `json:"append_ai_summary"`
	}
	if err := json.Unmarshal(a.ActionConfig, &cfg); err != nil || strings.TrimSpace(cfg.Message) == "" {
		return fmt.Errorf("%w: invalid slack config", slack.ErrPermanent)
	}
	if ruleName == "" {
		ruleName = "(unnamed)"
	}
	msg := strings.TrimSpace(cfg.Message)
	if cfg.AppendAISummary && suggestion != nil && suggestion.Summary != "" {
		msg = msg + "\nAI: " + suggestion.Summary + " (priority: " + suggestion.Priority + ")"
	}
	text := fmt.Sprintf(
		"GitActionFlow automation triggered\n\nRepository: %s\nEvent: %s\nAction: %s\nRule: %s\n\n%s",
		repo.FullName, ev.EventType, ev.Action, ruleName, msg,
	)
	return e.Slack.SendText(ctx, text)
}

func isPermanent(err error) bool {
	return errors.Is(err, githubapi.ErrPermanent) ||
		errors.Is(err, githubapi.ErrUnauthorized) ||
		errors.Is(err, githubapi.ErrForbidden) ||
		errors.Is(err, githubapi.ErrNotFound) ||
		errors.Is(err, slack.ErrPermanent) ||
		errors.Is(err, slack.ErrNotConfigured) ||
		errors.Is(err, events.ErrInvalidEvent)
}

func sanitizeErr(err error) string {
	msg := err.Error()
	msg = strings.ReplaceAll(msg, "hooks.slack.com", "[redacted-host]")
	if len(msg) > 500 {
		msg = msg[:500]
	}
	return msg
}

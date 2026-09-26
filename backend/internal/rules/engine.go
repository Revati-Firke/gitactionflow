package rules

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"github.com/Revati-Firke/gitactionflow/backend/internal/store"
)

const (
	EventTypeIssues      = "issues"
	EventTypePullRequest = "pull_request"

	ActionGitHubLabel       = "github_label"
	ActionGitHubComment     = "github_comment"
	ActionSlackNotification = "slack_notification"
)

// EventContext is the normalized view of a webhook payload used for matching and actions.
type EventContext struct {
	EventID      uuid.UUID
	RepositoryID uuid.UUID
	EventType    string
	Action       string
	Author       string
	Title        string
	Body         string
	Labels       []string
	IssueNumber  int
}

// ActionIntent is a side-effect-free request produced by a matching rule.
// The action executor persists and runs these against GitHub/Slack.
type ActionIntent struct {
	RuleID     uuid.UUID
	EventID    uuid.UUID
	ActionType string
	Config     json.RawMessage
	RuleName   string
}

// Match reports whether an enabled rule matches the event context (AND of set conditions).
func Match(rule store.Rule, ev EventContext) bool {
	if !rule.Enabled {
		return false
	}
	if rule.EventType != ev.EventType {
		return false
	}
	if rule.Keyword != nil && strings.TrimSpace(*rule.Keyword) != "" {
		kw := strings.ToLower(strings.TrimSpace(*rule.Keyword))
		hay := strings.ToLower(ev.Title + "\n" + ev.Body)
		if !strings.Contains(hay, kw) {
			return false
		}
	}
	if rule.Author != nil && strings.TrimSpace(*rule.Author) != "" {
		if !strings.EqualFold(strings.TrimSpace(*rule.Author), strings.TrimSpace(ev.Author)) {
			return false
		}
	}
	if len(rule.RequiredLabels) > 0 {
		have := map[string]struct{}{}
		for _, l := range ev.Labels {
			have[strings.ToLower(strings.TrimSpace(l))] = struct{}{}
		}
		for _, need := range rule.RequiredLabels {
			n := strings.ToLower(strings.TrimSpace(need))
			if n == "" {
				continue
			}
			if _, ok := have[n]; !ok {
				return false
			}
		}
	}
	return true
}

// Evaluate returns action intents for every matching enabled rule in order.
// Disabled rules in the slice are skipped. Ordering of input should already be deterministic.
func Evaluate(ruleList []store.Rule, ev EventContext) []ActionIntent {
	var out []ActionIntent
	for _, r := range ruleList {
		if !Match(r, ev) {
			continue
		}
		cfg := append(json.RawMessage(nil), r.ActionConfig...)
		out = append(out, ActionIntent{
			RuleID:     r.ID,
			EventID:    ev.EventID,
			ActionType: r.ActionType,
			Config:     cfg,
			RuleName:   r.Name,
		})
	}
	return out
}

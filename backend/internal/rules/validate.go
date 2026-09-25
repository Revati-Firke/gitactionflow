package rules

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidInput = errors.New("invalid rule input")
)

// Input is the validated create/update payload (API-facing fields only).
type Input struct {
	Name           string
	Enabled        bool
	EventType      string
	Keyword        *string
	Author         *string
	RequiredLabels []string
	ActionType     string
	ActionConfig   json.RawMessage
}

// ValidateNormalizes checks and normalizes a rule input.
func ValidateNormalize(in Input) (Input, error) {
	out := in
	out.Name = strings.TrimSpace(in.Name)
	if out.Name == "" {
		return Input{}, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	out.EventType = strings.TrimSpace(in.EventType)
	switch out.EventType {
	case EventTypeIssues, EventTypePullRequest:
	default:
		return Input{}, fmt.Errorf("%w: event_type must be issues or pull_request", ErrInvalidInput)
	}
	out.ActionType = strings.TrimSpace(in.ActionType)
	switch out.ActionType {
	case ActionGitHubLabel, ActionGitHubComment, ActionSlackNotification:
	default:
		return Input{}, fmt.Errorf("%w: invalid action_type", ErrInvalidInput)
	}

	if in.Keyword != nil {
		kw := strings.TrimSpace(*in.Keyword)
		if kw == "" {
			out.Keyword = nil
		} else {
			out.Keyword = &kw
		}
	}
	if in.Author != nil {
		a := strings.TrimSpace(*in.Author)
		if a == "" {
			out.Author = nil
		} else {
			out.Author = &a
		}
	}

	var labels []string
	seen := map[string]struct{}{}
	for _, l := range in.RequiredLabels {
		t := strings.TrimSpace(l)
		if t == "" {
			continue
		}
		key := strings.ToLower(t)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		labels = append(labels, t)
	}
	out.RequiredLabels = labels

	cfg, err := normalizeActionConfig(out.ActionType, in.ActionConfig)
	if err != nil {
		return Input{}, err
	}
	out.ActionConfig = cfg
	return out, nil
}

func normalizeActionConfig(actionType string, raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		raw = json.RawMessage(`{}`)
	}
	if !json.Valid(raw) {
		return nil, fmt.Errorf("%w: action_config must be valid JSON", ErrInvalidInput)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("%w: action_config must be a JSON object", ErrInvalidInput)
	}
	useAI, _ := m["use_ai"].(bool)
	appendAI, _ := m["append_ai_summary"].(bool)

	switch actionType {
	case ActionGitHubLabel:
		label, _ := m["label"].(string)
		label = strings.TrimSpace(label)
		if label == "" && !useAI {
			return nil, fmt.Errorf("%w: action_config.label is required (or set use_ai)", ErrInvalidInput)
		}
		out := map[string]any{"label": label}
		if useAI {
			out["use_ai"] = true
		}
		return json.Marshal(out)
	case ActionGitHubComment:
		comment, _ := m["comment"].(string)
		comment = strings.TrimSpace(comment)
		if comment == "" && !useAI && !appendAI {
			return nil, fmt.Errorf("%w: action_config.comment is required (or set use_ai / append_ai_summary)", ErrInvalidInput)
		}
		out := map[string]any{"comment": comment}
		if useAI {
			out["use_ai"] = true
		}
		if appendAI {
			out["append_ai_summary"] = true
		}
		return json.Marshal(out)
	case ActionSlackNotification:
		msg, _ := m["message"].(string)
		msg = strings.TrimSpace(msg)
		if msg == "" {
			return nil, fmt.Errorf("%w: action_config.message is required", ErrInvalidInput)
		}
		out := map[string]any{"message": msg}
		if appendAI {
			out["append_ai_summary"] = true
		}
		return json.Marshal(out)
	default:
		return nil, fmt.Errorf("%w: invalid action_type", ErrInvalidInput)
	}
}

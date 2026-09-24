package events

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Revati-Firke/gitactionflow/backend/internal/rules"
	"github.com/Revati-Firke/gitactionflow/backend/internal/store"
)

// ExtractEventContext builds a matching/action context from a persisted webhook event.
func ExtractEventContext(e store.WebhookEvent) (rules.EventContext, error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return rules.EventContext{}, fmt.Errorf("%w: payload object", ErrInvalidEvent)
	}

	ctx := rules.EventContext{
		EventID:      e.ID,
		RepositoryID: e.RepositoryID,
		EventType:    e.EventType,
		Action:       e.Action,
		Labels:       []string{},
	}

	switch e.EventType {
	case "issues":
		item, ok := payload["issue"]
		if !ok {
			return ctx, nil
		}
		title, body, author, labels, number, err := extractItem(item)
		if err != nil {
			return rules.EventContext{}, err
		}
		ctx.Title, ctx.Body, ctx.Author, ctx.Labels, ctx.IssueNumber = title, body, author, labels, number
	case "pull_request":
		item, ok := payload["pull_request"]
		if !ok {
			return ctx, nil
		}
		title, body, author, labels, number, err := extractItem(item)
		if err != nil {
			return rules.EventContext{}, err
		}
		ctx.Title, ctx.Body, ctx.Author, ctx.Labels, ctx.IssueNumber = title, body, author, labels, number
	default:
		return rules.EventContext{}, fmt.Errorf("%w: %s", ErrUnsupportedEvent, e.EventType)
	}
	return ctx, nil
}

func extractItem(raw json.RawMessage) (title, body, author string, labels []string, number int, err error) {
	var obj struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		User   struct {
			Login string `json:"login"`
		} `json:"user"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "", "", "", nil, 0, fmt.Errorf("%w: item fields", ErrInvalidEvent)
	}
	for _, l := range obj.Labels {
		n := strings.TrimSpace(l.Name)
		if n != "" {
			labels = append(labels, n)
		}
	}
	if labels == nil {
		labels = []string{}
	}
	return obj.Title, obj.Body, obj.User.Login, labels, obj.Number, nil
}

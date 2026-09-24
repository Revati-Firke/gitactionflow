package rules_test

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/Revati-Firke/gitactionflow/backend/internal/rules"
	"github.com/Revati-Firke/gitactionflow/backend/internal/store"
)

func ptr(s string) *string { return &s }

func baseRule() store.Rule {
	return store.Rule{
		ID: uuid.New(), Name: "r", Enabled: true,
		EventType: rules.EventTypeIssues, ActionType: rules.ActionGitHubLabel,
		ActionConfig: json.RawMessage(`{"label":"automation"}`),
		RequiredLabels: []string{},
	}
}

func TestMatch_EventType(t *testing.T) {
	r := baseRule()
	ev := rules.EventContext{EventType: rules.EventTypeIssues, Title: "x"}
	if !rules.Match(r, ev) {
		t.Fatal("expected match")
	}
	ev.EventType = rules.EventTypePullRequest
	if rules.Match(r, ev) {
		t.Fatal("expected no match")
	}
}

func TestMatch_KeywordCaseInsensitiveTitleAndBody(t *testing.T) {
	r := baseRule()
	r.Keyword = ptr("bug")
	if !rules.Match(r, rules.EventContext{EventType: "issues", Title: "Bug found in login"}) {
		t.Fatal("title")
	}
	if !rules.Match(r, rules.EventContext{EventType: "issues", Body: "there is a BUG here"}) {
		t.Fatal("body")
	}
	if rules.Match(r, rules.EventContext{EventType: "issues", Title: "feature request"}) {
		t.Fatal("should not match")
	}
}

func TestMatch_AuthorCaseInsensitive(t *testing.T) {
	r := baseRule()
	r.Author = ptr("OctoCat")
	if !rules.Match(r, rules.EventContext{EventType: "issues", Author: "octocat"}) {
		t.Fatal("expected match")
	}
	if rules.Match(r, rules.EventContext{EventType: "issues", Author: "someone"}) {
		t.Fatal("expected miss")
	}
}

func TestMatch_RequiredLabels(t *testing.T) {
	r := baseRule()
	r.RequiredLabels = []string{"bug", "priority"}
	ev := rules.EventContext{EventType: "issues", Labels: []string{"Bug", "priority", "backend"}}
	if !rules.Match(r, ev) {
		t.Fatal("expected match")
	}
	ev.Labels = []string{"bug"}
	if rules.Match(r, ev) {
		t.Fatal("missing priority")
	}
}

func TestMatch_ANDSemantics(t *testing.T) {
	r := baseRule()
	r.Keyword = ptr("bug")
	r.Author = ptr("alice")
	ev := rules.EventContext{EventType: "issues", Title: "bug", Author: "bob"}
	if rules.Match(r, ev) {
		t.Fatal("author mismatch should fail")
	}
	ev.Author = "alice"
	if !rules.Match(r, ev) {
		t.Fatal("expected match")
	}
}

func TestMatch_UnspecifiedConditionsIgnored(t *testing.T) {
	r := baseRule()
	// no keyword/author/labels
	if !rules.Match(r, rules.EventContext{EventType: "issues", Author: "x", Title: "y"}) {
		t.Fatal("expected match")
	}
}

func TestMatch_DisabledIgnored(t *testing.T) {
	r := baseRule()
	r.Enabled = false
	if rules.Match(r, rules.EventContext{EventType: "issues"}) {
		t.Fatal("disabled")
	}
}

func TestEvaluate_MultipleAndOrder(t *testing.T) {
	a := baseRule()
	a.ID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	a.Name = "A"
	a.Keyword = ptr("bug")
	b := baseRule()
	b.ID = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	b.Name = "B"
	b.Keyword = ptr("login")
	c := baseRule()
	c.ID = uuid.MustParse("00000000-0000-0000-0000-000000000003")
	c.Name = "C"
	c.Keyword = ptr("bug")

	ev := rules.EventContext{EventID: uuid.New(), EventType: "issues", Title: "bug in login"}
	intents := rules.Evaluate([]store.Rule{a, b, c}, ev)
	if len(intents) != 3 {
		t.Fatalf("got %d", len(intents))
	}
	if intents[0].RuleName != "A" || intents[1].RuleName != "B" || intents[2].RuleName != "C" {
		t.Fatalf("%+v", intents)
	}
}

func TestValidateNormalize(t *testing.T) {
	_, err := rules.ValidateNormalize(rules.Input{
		Name: " ", EventType: "issues", ActionType: "github_label",
		ActionConfig: json.RawMessage(`{"label":"x"}`),
	})
	if err == nil {
		t.Fatal("empty name")
	}
	in, err := rules.ValidateNormalize(rules.Input{
		Name: "Bug", EventType: "issues", ActionType: "github_label",
		Keyword: ptr("  bug  "), Author: ptr(""), RequiredLabels: []string{"", " Bug ", "bug"},
		ActionConfig: json.RawMessage(`{"label":" automation "}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if in.Keyword == nil || *in.Keyword != "bug" || in.Author != nil {
		t.Fatalf("%+v", in)
	}
	if len(in.RequiredLabels) != 1 || in.RequiredLabels[0] != "Bug" {
		t.Fatalf("%v", in.RequiredLabels)
	}
}

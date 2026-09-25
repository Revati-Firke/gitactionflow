package ai

import "testing"

func TestValidateSuggestion_Allowlist(t *testing.T) {
	s, err := ValidateSuggestion(Suggestion{
		Summary:         "  Fixes crash  ",
		SuggestedLabels: []string{"Bug", "evil", "enhancement", "bug"},
		Priority:        "HIGH",
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.Summary != "Fixes crash" {
		t.Fatalf("summary %q", s.Summary)
	}
	if s.Priority != "high" {
		t.Fatalf("priority %q", s.Priority)
	}
	if len(s.SuggestedLabels) != 2 || s.SuggestedLabels[0] != "bug" || s.SuggestedLabels[1] != "enhancement" {
		t.Fatalf("labels %#v", s.SuggestedLabels)
	}
}

func TestValidateSuggestion_RejectEmpty(t *testing.T) {
	if _, err := ValidateSuggestion(Suggestion{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseModelJSON_StripsFence(t *testing.T) {
	s, err := parseModelJSON("```json\n{\"summary\":\"ok\",\"suggested_labels\":[\"bug\"],\"priority\":\"low\"}\n```")
	if err != nil {
		t.Fatal(err)
	}
	if s.Summary != "ok" || s.Priority != "low" {
		t.Fatalf("%+v", s)
	}
}

func TestNoop(t *testing.T) {
	_, err := Noop{}.Suggest(nil, "t", "b")
	if err != ErrDisabled {
		t.Fatalf("got %v", err)
	}
}

package screens

import (
	"testing"

	domainenv "heartless/internal/domain/environment"
)

// TestEnvironmentMenuItems checks the right-click menu lists the workspace's
// environments, marks the active one, and ends with the settings link.
func TestEnvironmentMenuItems(t *testing.T) {
	st := shellState{
		workspaceID:   "ws-1",
		environmentID: "env-b",
		environments: []domainenv.Environment{
			{ID: "env-a", Name: "Dev"},
			{ID: "env-b", Name: "Staging"},
			{ID: "env-c", Name: "Prod"},
		},
	}

	items := environmentMenuItems(&st)
	var got []string
	for _, it := range items {
		got = append(got, it.Key)
	}
	want := []string{"env:env-a", "env:env-b", "env:env-c", "", "env-settings"}
	if len(got) != len(want) {
		t.Fatalf("menu keys = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("menu key %d = %q, want %q (all: %q)", i, got[i], want[i], got)
		}
	}
	if items[1].Icon == nil {
		t.Fatal("active environment not marked with a check")
	}
	if items[0].Icon != nil || items[2].Icon != nil {
		t.Fatal("inactive environments marked as active")
	}
}

// TestParseEnvKey checks the environment menu key parser.
func TestParseEnvKey(t *testing.T) {
	if id, ok := parseEnvKey("env:env-123"); !ok || id != "env-123" {
		t.Fatalf("parseEnvKey = %q, %v", id, ok)
	}
	if _, ok := parseEnvKey("env:"); ok {
		t.Fatal("empty environment ID accepted")
	}
	if _, ok := parseEnvKey("env-settings"); ok {
		t.Fatal("settings key parsed as an environment")
	}
}

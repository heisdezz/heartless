package screens

import (
	"testing"

	"github.com/lobehub/lobe-mygo/pkg/theme"

	domainvar "heartless/internal/domain/variable"
)

// TestVariableSpansMarksKnownAndUnknown checks placeholders are highlighted by
// whether the active environment defines them.
func TestVariableSpansMarksKnownAndUnknown(t *testing.T) {
	tk := theme.Default().Tokens(false)
	known := map[string]string{"baseUrl": "https://x.test"}

	spans := variableSpans("{{baseUrl}}/api/{{missing}}/x", tk, func(n string) bool {
		_, ok := known[n]
		return ok
	})
	if len(spans) != 4 {
		t.Fatalf("spans = %d (%+v), want 4", len(spans), spans)
	}

	// The defined variable is a filled chip in the primary color.
	if spans[0].Text != "{{baseUrl}}" || spans[0].Color != tk.Primary || spans[0].Background != tk.PrimaryFillSecondary {
		t.Fatalf("known span = %+v", spans[0])
	}
	// The undefined one is warning-colored and wavy-underlined.
	if spans[2].Text != "{{missing}}" || spans[2].Color != tk.Warning || !spans[2].WavyUnderline {
		t.Fatalf("unknown span = %+v", spans[2])
	}
	// The plain text between them stays plain.
	if spans[1].Text != "/api/" || spans[1].Color != tk.TextSecondary {
		t.Fatalf("plain span = %+v", spans[1])
	}
}

// TestVariableSpansUnclosed checks an unclosed placeholder is marked.
func TestVariableSpansUnclosed(t *testing.T) {
	tk := theme.Default().Tokens(false)
	spans := variableSpans("https://x/{{oops", tk, func(string) bool { return false })
	if len(spans) == 0 {
		t.Fatal("no spans")
	}
	last := spans[len(spans)-1]
	if last.Text != "{{oops" || !last.WavyUnderline {
		t.Fatalf("unclosed span = %+v", last)
	}
}

// TestKnowsVariableUsesActiveScope checks the workspace's active environment
// decides which names count as defined.
func TestKnowsVariableUsesActiveScope(t *testing.T) {
	st := shellState{
		environmentID: "env-dev",
		variables: []domainvar.Variable{
			{Name: "baseUrl", Value: "https://dev.test", EnvironmentID: "env-dev"},
			{Name: "token", Value: "t", EnvironmentID: "env-prod"},
			{Name: "globalToken", Value: "g"},
		},
	}
	if !st.knowsVariable("baseUrl") {
		t.Error("active env variable not known")
	}
	if !st.knowsVariable("globalToken") {
		t.Error("workspace variable not known")
	}
	if st.knowsVariable("token") {
		t.Error("variable from another environment counted as known")
	}
	if st.knowsVariable("nope") {
		t.Error("unknown name reported known")
	}
}

// TestVariableSpansNoPlaceholders checks plain text gives no spans.
func TestVariableSpansNoPlaceholders(t *testing.T) {
	tk := theme.Default().Tokens(false)
	if spans := variableSpans("https://x.test/a", tk, nil); spans != nil {
		t.Fatalf("spans = %+v, want nil", spans)
	}
}

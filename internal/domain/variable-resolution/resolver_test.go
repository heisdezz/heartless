package variable_resolution

import (
	"context"
	"testing"

	"heartless/internal/domain/variable"
)

func TestResolvePrecedenceAndInterpolation(t *testing.T) {
	req := NewMemoryVariableSet()
	req.Add(*variable.NewVariable("name", "req", "text", false))
	env := NewMemoryVariableSet()
	env.Add(*variable.NewVariable("name", "env", "text", false))
	env.Add(*variable.NewVariable("host", "api.test", "text", false))
	ws := NewMemoryVariableSet()
	ws.Add(*variable.NewVariable("name", "ws", "text", false))

	r := NewResolver(req, NewMemoryVariableSet(), NewMemoryVariableSet(), env, ws, NewMemoryVariableSet())
	got := r.Resolve(context.Background(), "https://{{host}}/{{name}}/{{missing}}")
	want := "https://api.test/req/{{missing}}"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveCycleAndNested(t *testing.T) {
	env := NewMemoryVariableSet()
	env.Add(*variable.NewVariable("a", "{{b}}", "text", false))
	env.Add(*variable.NewVariable("b", "{{a}}", "text", false))
	env.Add(*variable.NewVariable("base", "https://{{host}}", "text", false))
	env.Add(*variable.NewVariable("host", "example.com", "text", false))
	r := NewResolver(NewMemoryVariableSet(), NewMemoryVariableSet(), NewMemoryVariableSet(), env, NewMemoryVariableSet(), NewMemoryVariableSet())

	if got := r.Resolve(context.Background(), "{{a}}"); got != "{{a}}" {
		t.Fatalf("cycle got %q", got)
	}
	if err := r.ValidateCircularReferences(); err == nil {
		t.Fatal("expected cycle error")
	}
	if got := r.Resolve(context.Background(), "{{base}}/x"); got != "https://example.com/x" {
		t.Fatalf("nested got %q", got)
	}
}

func TestSecretMetadata(t *testing.T) {
	env := NewMemoryVariableSet()
	env.Add(*variable.NewVariable("token", "secret", "text", true))
	env.Add(*variable.NewVariable("url", "https://x?token={{token}}", "text", false))
	r := NewResolver(NewMemoryVariableSet(), NewMemoryVariableSet(), NewMemoryVariableSet(), env, NewMemoryVariableSet(), NewMemoryVariableSet())

	meta := r.ResolveWithMeta(context.Background(), "{{url}}")
	if meta.Value != "https://x?token=secret" || !meta.Secret {
		t.Fatalf("unexpected meta: %+v", meta)
	}
	val, sec := r.ResolveWithSecrets("token")
	if val != "secret" || !sec {
		t.Fatalf("unexpected secret lookup: %q %v", val, sec)
	}
}

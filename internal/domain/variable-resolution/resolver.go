package variable_resolution

import (
	"context"
	"fmt"
	"strings"

	"heartless/internal/domain/variable"
)

const (
	openDelim  = "{{"
	closeDelim = "}}"
)

// Scope precedence (highest to lowest): request, folder, collection, environment, workspace, app defaults.
type Resolver struct {
	scopes []VariableSet
}

// NewResolver creates a resolver with deterministic scope precedence.
func NewResolver(requestVars, folderVars, collectionVars, envVars, workspaceVars, appDefaults VariableSet) *Resolver {
	return &Resolver{scopes: []VariableSet{requestVars, folderVars, collectionVars, envVars, workspaceVars, appDefaults}}
}

// Result captures a resolved value and whether it came from a secret variable.
type Result struct {
	Value  string
	Secret bool
	Found  bool
}

// Lookup returns the first matching variable by precedence.
func (r *Resolver) Lookup(name string) (variable.Variable, bool) {
	for _, s := range r.scopes {
		if s == nil {
			continue
		}
		if v, ok := s.Get(name); ok {
			return v, true
		}
	}
	return variable.Variable{}, false
}

// Resolve interpolates {{var}} placeholders in source.
// Missing variables and cycles leave the original placeholder unchanged.
// Context cancellation stops resolution early and returns best-effort output.
func (r *Resolver) Resolve(ctx context.Context, source string) string {
	if source == "" || !strings.Contains(source, openDelim) {
		return source
	}
	out, _ := r.resolveString(ctx, source, map[string]bool{}, 0)
	return out
}

// ResolveWithMeta resolves source and reports whether any resolved value was secret.
func (r *Resolver) ResolveWithMeta(ctx context.Context, source string) Result {
	if source == "" || !strings.Contains(source, openDelim) {
		return Result{Value: source}
	}
	out, secret := r.resolveString(ctx, source, map[string]bool{}, 0)
	return Result{Value: out, Secret: secret, Found: out != source}
}

const maxDepth = 10

func (r *Resolver) resolveString(ctx context.Context, source string, stack map[string]bool, depth int) (string, bool) {
	if depth > maxDepth {
		return source, false
	}
	if ctx != nil {
		select {
		case <-ctx.Done():
			return source, false
		default:
		}
	}

	var b strings.Builder
	secret := false
	i := 0
	for i < len(source) {
		start := strings.Index(source[i:], openDelim)
		if start < 0 {
			b.WriteString(source[i:])
			break
		}
		start += i
		b.WriteString(source[i:start])
		end := strings.Index(source[start+len(openDelim):], closeDelim)
		if end < 0 {
			b.WriteString(source[start:])
			return b.String(), secret
		}
		end += start + len(openDelim)
		name := strings.TrimSpace(source[start+len(openDelim) : end])
		placeholder := source[start : end+len(closeDelim)]
		val, sec, ok := r.resolveName(ctx, name, stack, depth)
		if ok {
			b.WriteString(val)
			if sec {
				secret = true
			}
		} else {
			b.WriteString(placeholder)
		}
		i = end + len(closeDelim)
	}
	return b.String(), secret
}

func (r *Resolver) resolveName(ctx context.Context, name string, stack map[string]bool, depth int) (string, bool, bool) {
	if name == "" {
		return "", false, false
	}
	if stack[name] {
		return "", false, false
	}
	v, ok := r.Lookup(name)
	if !ok {
		return "", false, false
	}
	stack[name] = true
	defer delete(stack, name)
	val, sec := r.resolveString(ctx, v.Value, stack, depth+1)
	if v.Secret {
		sec = true
	}
	return val, sec, true
}

// ResolveWithSecrets resolves a single variable name and returns value and secret flag.
func (r *Resolver) ResolveWithSecrets(name string) (string, bool) {
	v, ok := r.Lookup(name)
	if !ok {
		return "", false
	}
	val, sec := r.resolveString(context.Background(), v.Value, map[string]bool{name: true}, 0)
	if v.Secret {
		sec = true
	}
	return val, sec
}

// ValidateCircularReferences reports variables involved in cycles.
func (r *Resolver) ValidateCircularReferences() error {
	seen := map[string]bool{}
	for _, s := range r.scopes {
		if s == nil {
			continue
		}
		for _, v := range s.List() {
			if seen[v.Name] {
				continue
			}
			seen[v.Name] = true
			if hasCycle(r, v.Name, map[string]bool{}) {
				return fmt.Errorf("circular variable reference: %s", v.Name)
			}
		}
	}
	return nil
}

func hasCycle(r *Resolver, name string, stack map[string]bool) bool {
	if stack[name] {
		return true
	}
	v, ok := r.Lookup(name)
	if !ok || !strings.Contains(v.Value, openDelim) {
		return false
	}
	stack[name] = true
	defer delete(stack, name)
	i := 0
	for i < len(v.Value) {
		start := strings.Index(v.Value[i:], openDelim)
		if start < 0 {
			break
		}
		start += i
		end := strings.Index(v.Value[start+len(openDelim):], closeDelim)
		if end < 0 {
			break
		}
		end += start + len(openDelim)
		ref := strings.TrimSpace(v.Value[start+len(openDelim) : end])
		if ref != "" && hasCycle(r, ref, stack) {
			return true
		}
		i = end + len(closeDelim)
	}
	return false
}

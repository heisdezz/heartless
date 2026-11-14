package variable_resolution

import (
	"context"
	"strings"
	"sync"

	"heartless/internal/domain/variable"
)

// VariableSet is an interface for looking up variables by name.
// Implementations provide variable resolution from a specific scope.
type VariableSet interface {
	// Get returns the variable with the given name, or false if not found.
	Get(name string) (variable.Variable, bool)
	// Add adds a variable to the set.
	Add(variable.Variable)
	// Remove removes a variable from the set.
	Remove(name string)
	// List returns all variables in the set.
	List() []variable.Variable
}

// Resolver resolves variable references in request fields according to the precedence chain.
// The resolution order (highest to lowest priority):
//   1. Request-level variables
//   2. Folder-level variables
//   3. Collection-level variables
//   4. Active environment variables
//   5. Workspace-level variables
//   6. Application defaults
//
// Interpolation syntax: {{variable_name}}
//
// Secrets are marked but not exposed in resolved output logs.
type Resolver struct {
	// requestVars are variables scoped to the current request.
	requestVars VariableSet

	// folderVars are variables scoped to the request's folder.
	folderVars VariableSet

	// collectionVars are variables scoped to the request's collection.
	collectionVars VariableSet

	// envVars are variables from the active environment.
	envVars VariableSet

	// workspaceVars are variables from the active workspace.
	workspaceVars VariableSet

	// appDefaults are application-level default variables.
	appDefaults VariableSet

	// mu protects concurrent access to resolver state.
	mu sync.Mutex
}

// NewResolver creates a new variable resolver with the given variable sets.
func NewResolver(requestVars, folderVars, collectionVars VariableSet,
	envVars, workspaceVars, appDefaults VariableSet) *Resolver {

	return &Resolver{
		requestVars:      requestVars,
		folderVars:       folderVars,
		collectionVars:   collectionVars,
		envVars:          envVars,
		workspaceVars:    workspaceVars,
		appDefaults:      appDefaults,
	}
}

// Resolve resolves all variable references in the given source string.
// It searches through the precedence chain and returns the first match.
// If no variable is found, the original placeholder is returned unchanged.
func (r *Resolver) Resolve(ctx context.Context, source string) string {
	if source == "" {
		return source
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	result := source

	// Find all variable references {{...}} and resolve them.
	var buf strings.Builder
	i := 0
	for i < len(result) {
		// Find the next {{ placeholder
		idx := strings.Index(result[i:], "{{")
		if idx < 0 {
			// No more placeholders; append the rest and stop
			buf.WriteString(result[i:])
			break
		}
		// Append text before the placeholder
		buf.WriteString(result[i : i+idx])

		// Find the closing }}
		closeIdx := strings.Index(result[i+idx+2:], "}}")
		if closeIdx < 0 {
			// Malformed placeholder; append remaining and stop
			buf.WriteString(result[i:])
			break
		}
		varName := strings.TrimSpace(result[i+idx+2 : i+idx+2+closeIdx])

		// Resolve the variable according to precedence
		resolved := r.resolveVariable(varName)
		buf.WriteString(resolved)

		// Advance past the placeholder
		i = i + idx + 2 + closeIdx + 2
	}

	return buf.String()
}

// resolveVariable resolves a single variable name according to the precedence chain.
// Returns the effective value, or the original placeholder if not found.
func (r *Resolver) resolveVariable(varName string) string {
	// 1. Request-level variables
	if v, ok := r.requestVars.Get(varName); ok && !v.Secret {
		return v.Value
	}

	// 2. Folder-level variables
	if v, ok := r.folderVars.Get(varName); ok && !v.Secret {
		return v.Value
	}

	// 3. Collection-level variables
	if v, ok := r.collectionVars.Get(varName); ok && !v.Secret {
		return v.Value
	}

	// 4. Active environment variables
	if v, ok := r.envVars.Get(varName); ok && !v.Secret {
		return v.Value
	}

	// 5. Workspace-level variables
	if v, ok := r.workspaceVars.Get(varName); ok && !v.Secret {
		return v.Value
	}

	// 6. Application defaults
	if v, ok := r.appDefaults.Get(varName); ok && !v.Secret {
		return v.Value
	}

	// Variable not found in any scope - return original placeholder
	return "{{" + varName + "}}"
}

// ResolveWithSecrets resolves a single variable returning the value and secret flag.
// It checks scopes in precedence order and returns the first match.
func (r *Resolver) ResolveWithSecrets(varName string) (string, bool) {
	// Check each scope in precedence order
	scopes := []VariableSet{
		r.requestVars,
		r.folderVars,
		r.collectionVars,
		r.envVars,
		r.workspaceVars,
		r.appDefaults,
	}

	for _, scope := range scopes {
		if v, ok := scope.Get(varName); ok {
			return v.Value, v.Secret
		}
	}

	return "", false
}

// ValidateCircularReferences checks for circular variable references.
// This is a placeholder; a full implementation would need to track
// variable dependencies across scopes and detect cycles in the dependency graph.
func (r *Resolver) ValidateCircularReferences() error {
	// TODO: implement circular reference detection
	// This would require analyzing variable references across all scopes
	// and detecting cycles in the dependency graph.
	return nil
}

// MemoryVariableSet is a concrete implementation of VariableSet backed by an in-memory map.
type MemoryVariableSet struct {
	vars map[string]variable.Variable
	m    sync.RWMutex
}

// NewMemoryVariableSet creates a new in-memory variable set and returns VariableSet interface.
func NewMemoryVariableSet() VariableSet {
	return &MemoryVariableSet{
		vars: make(map[string]variable.Variable),
	}
}

// Add adds a variable to the set.
func (s *MemoryVariableSet) Add(v variable.Variable) {
	s.m.Lock()
	defer s.m.Unlock()
	s.vars[v.Name] = v
}

// Get returns the variable with the given name, or false if not found.
func (s MemoryVariableSet) Get(name string) (variable.Variable, bool) {
	s.m.RLock()
	defer s.m.RUnlock()
	v, ok := s.vars[name]
	return v, ok
}

// Remove removes a variable from the set.
func (s *MemoryVariableSet) Remove(name string) {
	s.m.Lock()
	defer s.m.Unlock()
	delete(s.vars, name)
}

// List returns all variables in the set.
func (s MemoryVariableSet) List() []variable.Variable {
	s.m.RLock()
	defer s.m.RUnlock()
	result := make([]variable.Variable, 0, len(s.vars))
	for _, v := range s.vars {
		result = append(result, v)
	}
	return result
}

// NewVariableSetFromVariables creates a VariableSet from a slice of variables.
func NewVariableSetFromVariables(vars []variable.Variable) VariableSet {
	m := make(map[string]variable.Variable, len(vars))
	for _, v := range vars {
		m[v.Name] = v
	}
	return &MemoryVariableSet{vars: m}
}
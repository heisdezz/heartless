package variable_resolution

import (
	"sync"

	"heartless/internal/domain/variable"
)

// VariableSet provides variable lookup for a specific scope.
type VariableSet interface {
	Get(name string) (variable.Variable, bool)
	Add(v variable.Variable)
	Remove(name string)
	List() []variable.Variable
}

// MemoryVariableSet is an in-memory VariableSet implementation.
type MemoryVariableSet struct {
	mu   sync.RWMutex
	vars map[string]variable.Variable
}

// NewMemoryVariableSet creates an empty in-memory variable set.
func NewMemoryVariableSet() VariableSet {
	return &MemoryVariableSet{vars: map[string]variable.Variable{}}
}

// NewVariableSetFromVariables creates a variable set populated from vars.
func NewVariableSetFromVariables(vars []variable.Variable) VariableSet {
	m := make(map[string]variable.Variable, len(vars))
	for _, v := range vars {
		m[v.Name] = v
	}
	return &MemoryVariableSet{vars: m}
}

func (s *MemoryVariableSet) Get(name string) (variable.Variable, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.vars[name]
	return v, ok
}

func (s *MemoryVariableSet) Add(v variable.Variable) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vars[v.Name] = v
}

func (s *MemoryVariableSet) Remove(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.vars, name)
}

func (s *MemoryVariableSet) List() []variable.Variable {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]variable.Variable, 0, len(s.vars))
	for _, v := range s.vars {
		out = append(out, v)
	}
	return out
}

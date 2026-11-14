package variable_service

import (
	"context"
	"time"

	"heartless/internal/domain/variable"
	"heartless/internal/domain/variable-resolution"
	variable_repo "heartless/internal/storage/sqlite/variable"
)

// VariableService handles variable-related business logic.
type VariableService interface {
	// Create creates a new variable.
	Create(ctx context.Context, name, typ string, secret bool) (*variable.Variable, error)
	// Get retrieves a variable by name.
	Get(ctx context.Context, name string) (*variable.Variable, error)
	// Update updates a variable's value or properties.
	Update(ctx context.Context, varReq *variable.Variable) error
	// Delete removes a variable by name.
	Delete(ctx context.Context, name string) error
	// List returns all variables for an environment/workspace.
	List(ctx context.Context, scope, scopeID string) ([]*variable.Variable, error)
	// Resolve resolves variable references in a string value.
	Resolve(ctx context.Context, source string) string
	// ResolveWithSecrets resolves variables returning value and secret flag.
	ResolveWithSecrets(ctx context.Context, varName string) (string, bool)
}

// variableService is the concrete implementation of VariableService using SQLite.
type variableService struct {
	variables       map[string]*variable.Variable
	resolver        *variable_resolution.Resolver
	variableSets    map[string]variable_resolution.VariableSet
	variableRepo    *variable_repo.Repository
}

// NewVariableService creates a new variable service with SQLite backend.
func NewVariableService(repo *variable_repo.Repository) *variableService {
	vs := &variableService{
		variables: make(map[string]*variable.Variable),
		variableRepo: repo,
		variableSets: make(map[string]variable_resolution.VariableSet),
	}

	// Initialize variable sets for different scopes, loaded from SQLite
	vs.variableSets["request"] = variable_resolution.NewMemoryVariableSet()
	vs.variableSets["folder"] = variable_resolution.NewMemoryVariableSet()
	vs.variableSets["collection"] = variable_resolution.NewMemoryVariableSet()
	vs.variableSets["environment"] = variable_resolution.NewMemoryVariableSet()
	vs.variableSets["workspace"] = variable_resolution.NewMemoryVariableSet()
	vs.variableSets["app-default"] = variable_resolution.NewMemoryVariableSet()

	// Load variables from SQLite and populate the sets
	vs.reloadVariableSets()

	// Create resolver linking all variable sets
	vs.resolver = variable_resolution.NewResolver(
		vs.variableSets["request"],
		vs.variableSets["folder"],
		vs.variableSets["collection"],
		vs.variableSets["environment"],
		vs.variableSets["workspace"],
		vs.variableSets["app-default"],
	)

	return vs
}

// reloadVariableSets reloads all variables from SQLite into the in-memory sets.
func (s *variableService) reloadVariableSets() {
	// Load all variables from SQLite
	vars, err := s.variableRepo.List(context.Background(), "", "")
	if err == nil {
		for _, v := range vars {
			if sv, ok := s.variableSets["environment"]; ok {
				sv.Add(*v)
			}
		}
	}
}

// Create creates a new variable.
func (s *variableService) Create(ctx context.Context, name, typ string, secret bool) (*variable.Variable, error) {
	v := variable.NewVariable(name, "", typ, secret)
	s.variables[name] = v

	// Save to SQLite
	if err := s.variableRepo.Save(ctx, name, "", "", v); err != nil {
		return nil, err
	}

	// Reload sets to include the new variable
	s.reloadVariableSets()

	return v, nil
}

// Get retrieves a variable by name.
func (s *variableService) Get(ctx context.Context, name string) (*variable.Variable, error) {
	v, ok := s.variables[name]
	if !ok {
		// Try to get from SQLite
		v, err := s.variableRepo.Get(ctx, name)
		if err != nil {
			return nil, variable.ErrNotFound
		}
		s.variables[name] = v
	}
	return v, nil
}

// Update updates a variable's value or properties.
func (s *variableService) Update(ctx context.Context, varReq *variable.Variable) error {
	if _, ok := s.variables[varReq.Name]; !ok {
		return variable.ErrNotFound
	}
	varReq.UpdatedAt = time.Now().UTC()
	s.variables[varReq.Name] = varReq

	// Update in SQLite
	if err := s.variableRepo.Save(ctx, varReq.Name, "", "", varReq); err != nil {
		return err
	}

	return nil
}

// Delete removes a variable by name.
func (s *variableService) Delete(ctx context.Context, name string) error {
	if _, ok := s.variables[name]; !ok {
		return variable.ErrNotFound
	}
	delete(s.variables, name)

	// Delete from SQLite
	if err := s.variableRepo.Delete(ctx, name); err != nil {
		return err
	}

	// Reload sets
	s.reloadVariableSets()

	return nil
}

// List returns all variables for an environment/workspace.
func (s *variableService) List(ctx context.Context, scope, scopeID string) ([]*variable.Variable, error) {
	return s.variableRepo.List(ctx, scopeID, "")
}

// Resolve resolves variable references in a string value.
func (s *variableService) Resolve(ctx context.Context, source string) string {
	return s.resolver.Resolve(ctx, source)
}

// ResolveWithSecrets resolves variables returning value and secret flag.
func (s *variableService) ResolveWithSecrets(ctx context.Context, varName string) (string, bool) {
	return s.resolver.ResolveWithSecrets(varName)
}
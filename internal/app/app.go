package app

import (
	"context"
	"database/sql"
	"fmt"

	"heartless/internal/config"
	domainreq "heartless/internal/domain/request"
	domainvar "heartless/internal/domain/variable"
	varres "heartless/internal/domain/variable-resolution"
	reqservice "heartless/internal/services/request"
	sqlitestore "heartless/internal/storage/sqlite"
	colrepo "heartless/internal/storage/sqlite/collection"
	envrepo "heartless/internal/storage/sqlite/environment"
	histrepo "heartless/internal/storage/sqlite/history"
	varrepo "heartless/internal/storage/sqlite/variable"
	wsrepo "heartless/internal/storage/sqlite/workspace"
	transport "heartless/internal/transport/http"
)

// Services holds application dependencies exposed to the GUI.
type Services struct {
	DB           *sql.DB
	Workspaces   *wsrepo.Repository
	Environments *envrepo.Repository
	Variables    *varrepo.Repository
	Collections  *colrepo.Repository
	History      *histrepo.Repository

	// Requests executes saved requests against HTTP. Each call is independent,
	// so concurrent executions never share state.
	Requests *reqservice.Executor
}

// ExecutorFor returns a request executor whose variable resolver reads the
// given environment and workspace variables. A new one per send keeps sends
// independent of one another and of later variable edits.
func (s *Services) ExecutorFor(envVars, workspaceVars []*domainvar.Variable) *reqservice.Executor {
	exec := reqservice.NewExecutor(transport.NewClient(), NewResolver(envVars, workspaceVars), domainreq.DefaultHTTPSettings())
	if s.History != nil {
		// Record each send, with its response payload, to the database.
		exec = exec.WithHistory(s.History)
	}
	return exec
}

// NewResolver builds a variable resolver from environment and workspace
// variables, with environment values taking precedence over workspace ones.
func NewResolver(envVars, workspaceVars []*domainvar.Variable) *varres.Resolver {
	env := varres.NewMemoryVariableSet()
	for _, v := range envVars {
		if v != nil {
			env.Add(*v)
		}
	}
	ws := varres.NewMemoryVariableSet()
	for _, v := range workspaceVars {
		if v != nil {
			ws.Add(*v)
		}
	}
	empty := func() varres.VariableSet { return varres.NewMemoryVariableSet() }
	return varres.NewResolver(empty(), empty(), empty(), env, ws, empty())
}

var defaultDBPath string

// SetDefaultDBPath sets the default SQLite database path used by Open.
func SetDefaultDBPath(path string) {
	defaultDBPath = path
}

// Open constructs application dependencies, opening SQLite and running migrations.
func Open(ctx context.Context) (*Services, error) {
	config.SetupLogging(config.DefaultLogging())
	if defaultDBPath == "" {
		return nil, fmt.Errorf("database path not set")
	}
	db, err := sqlitestore.Open(ctx, sqlitestore.Config{Path: defaultDBPath})
	if err != nil {
		return nil, err
	}
	history := histrepo.New(db)
	return &Services{
		DB:           db,
		Workspaces:   wsrepo.New(db),
		Environments: envrepo.New(db),
		Variables:    varrepo.New(db),
		Collections:  colrepo.New(db),
		History:      history,
		Requests:     reqservice.NewExecutor(transport.NewClient(), nil, domainreq.DefaultHTTPSettings()).WithHistory(history),
	}, nil
}

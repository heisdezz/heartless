package app

import (
	"context"
	"database/sql"
	"fmt"

	sqlitestore "heartless/internal/storage/sqlite"
	envrepo "heartless/internal/storage/sqlite/environment"
	varrepo "heartless/internal/storage/sqlite/variable"
	wsrepo "heartless/internal/storage/sqlite/workspace"
)

// Services holds application dependencies exposed to the GUI.
type Services struct {
	DB           *sql.DB
	Workspaces   *wsrepo.Repository
	Environments *envrepo.Repository
	Variables    *varrepo.Repository
}

var defaultDBPath string

// SetDefaultDBPath sets the default SQLite database path used by Open.
func SetDefaultDBPath(path string) {
	defaultDBPath = path
}

// Open constructs application dependencies, opening SQLite and running migrations.
func Open(ctx context.Context) (*Services, error) {
	if defaultDBPath == "" {
		return nil, fmt.Errorf("database path not set")
	}
	db, err := sqlitestore.Open(ctx, sqlitestore.Config{Path: defaultDBPath})
	if err != nil {
		return nil, err
	}
	return &Services{
		DB:           db,
		Workspaces:   wsrepo.New(db),
		Environments: envrepo.New(db),
		Variables:    varrepo.New(db),
	}, nil
}

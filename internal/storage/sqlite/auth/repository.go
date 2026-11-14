package auth

import (
	"context"
	"database/sql"
	"errors"

	domain "heartless/internal/domain/auth"
)

// Repository provides SQLite persistence for auth providers.
type Repository struct {
	db *sql.DB
}

// New creates an auth repository backed by SQLite.
func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Save saves an auth provider to the database.
func (r *Repository) Save(ctx context.Context, provider domain.AuthProvider) error {
	// SQLite doesn't persist providers between runs for this MVP;
	// providers are registered in-memory at startup.
	_ = ctx
	_ = provider
	return nil
}

// Get returns an auth provider by type and scope.
// This is a placeholder; in a full implementation, this would query SQLite.
func (r *Repository) Get(providerType, scope string) (domain.AuthProvider, error) {
	_ = providerType
	_ = scope
	return nil, errors.New("not implemented: use in-memory manager at startup")
}

// List returns all providers.
// This is a placeholder.
func (r *Repository) List(scope string) (map[string]domain.AuthProvider, error) {
	_ = scope
	return make(map[string]domain.AuthProvider), nil
}
package history

import (
	"context"

	domain "heartless/internal/domain/history"
)

// Store is the persistence contract required by the history service.
type Store interface {
	Repository
	Prune(ctx context.Context, workspaceID string, limit int) error
}

// Config controls history retention behavior.
type Config struct {
	RetentionLimit int
}

// ServiceImpl records and queries execution history with retention hooks.
type ServiceImpl struct {
	store Store
	cfg   Config
}

func New(store Store, cfg Config) *ServiceImpl {
	return &ServiceImpl{store: store, cfg: cfg}
}

func (s *ServiceImpl) Record(ctx context.Context, entry *domain.Entry) error {
	if err := s.store.Save(ctx, entry); err != nil {
		return err
	}
	if s.cfg.RetentionLimit > 0 && entry != nil {
		return s.store.Prune(ctx, entry.WorkspaceID, s.cfg.RetentionLimit)
	}
	return nil
}

func (s *ServiceImpl) Get(ctx context.Context, id string) (*domain.Entry, error) {
	return s.store.Get(ctx, id)
}

func (s *ServiceImpl) List(ctx context.Context, workspaceID string, limit int) ([]*domain.Entry, error) {
	return s.store.List(ctx, workspaceID, limit)
}

func (s *ServiceImpl) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

func (s *ServiceImpl) Clear(ctx context.Context, workspaceID string) error {
	return s.store.Clear(ctx, workspaceID)
}

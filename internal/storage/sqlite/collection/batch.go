package collection

import (
	"context"
	"database/sql"
	"fmt"

	domain "heartless/internal/domain/collection"
	"heartless/internal/domain/request"
	sqlitestore "heartless/internal/storage/sqlite"
)

// ImportBatch persists a set of collections, folders, and requests in a
// single transaction: either every entity is written or none are.
//
// Entities must be internally consistent: folders and requests must reference
// IDs of collections included in the same batch, and folders must be ordered
// parents-before-children.
func (r *Repository) ImportBatch(
	ctx context.Context,
	collections []*domain.Collection,
	folders []*domain.Folder,
	requests []*request.Request,
) error {
	return r.SyncBatch(ctx, collections, folders, requests, nil, nil)
}

// SyncBatch persists collections, folders, and requests and deletes removed
// folder and request IDs in a single transaction: either every change is
// applied or none are.
func (r *Repository) SyncBatch(
	ctx context.Context,
	collections []*domain.Collection,
	folders []*domain.Folder,
	requests []*request.Request,
	deleteFolderIDs []string,
	deleteRequestIDs []string,
) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("collection repository is not initialized")
	}
	return sqlitestore.Tx(ctx, r.db, func(tx *sql.Tx) error {
		for _, id := range deleteRequestIDs {
			if _, err := tx.ExecContext(ctx, `DELETE FROM requests WHERE id = ?`, id); err != nil {
				return fmt.Errorf("delete removed request %q: %w", id, err)
			}
		}
		for _, id := range deleteFolderIDs {
			if _, err := tx.ExecContext(ctx, `DELETE FROM folders WHERE id = ?`, id); err != nil {
				return fmt.Errorf("delete removed folder %q: %w", id, err)
			}
		}
		for _, col := range collections {
			if err := saveCollection(ctx, tx, col); err != nil {
				return fmt.Errorf("import collection %q: %w", nameOf(col), err)
			}
		}
		for _, folder := range folders {
			if err := saveFolder(ctx, tx, folder); err != nil {
				return fmt.Errorf("import folder %q: %w", nameOfFolder(folder), err)
			}
		}
		for _, req := range requests {
			if err := saveRequest(ctx, tx, req); err != nil {
				return fmt.Errorf("import request %q: %w", nameOfRequest(req), err)
			}
		}
		return nil
	})
}

func nameOf(col *domain.Collection) string {
	if col == nil {
		return ""
	}
	return col.Name
}

func nameOfFolder(folder *domain.Folder) string {
	if folder == nil {
		return ""
	}
	return folder.Name
}

func nameOfRequest(req *request.Request) string {
	if req == nil {
		return ""
	}
	return req.Name
}

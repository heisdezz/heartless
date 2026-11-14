package history

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	domain "heartless/internal/domain/history"
	sqlitestore "heartless/internal/storage/sqlite"
)

func TestHistoryRepository(t *testing.T) {
	db, err := sqlitestore.Open(context.Background(), sqlitestore.Config{Path: filepath.Join(t.TempDir(), "h.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := New(db)
	ctx := context.Background()

	e1 := domain.New("ws1")
	e1.RequestID = "r1"
	e1.Method = "GET"
	e1.URL = "https://a"
	e1.StatusCode = 200
	e1.CreatedAt = time.Now().UTC().Add(-2 * time.Second)
	if err := repo.Save(ctx, e1); err != nil {
		t.Fatal(err)
	}
	e2 := domain.New("ws1")
	e2.RequestID = "r2"
	e2.CreatedAt = time.Now().UTC()
	if err := repo.Save(ctx, e2); err != nil {
		t.Fatal(err)
	}

	got, err := repo.Get(ctx, e1.ID)
	if err != nil || got.URL != "https://a" {
		t.Fatalf("get: %v %+v", err, got)
	}
	list, err := repo.List(ctx, "ws1", 10)
	if err != nil || len(list) != 2 || list[0].ID != e2.ID {
		t.Fatalf("list: %v %#v", err, list)
	}
	if err := repo.Prune(ctx, "ws1", 1); err != nil {
		t.Fatal(err)
	}
	list, _ = repo.List(ctx, "ws1", 10)
	if len(list) != 1 || list[0].ID != e2.ID {
		t.Fatalf("prune: %#v", list)
	}
	if err := repo.Clear(ctx, "ws1"); err != nil {
		t.Fatal(err)
	}
	list, _ = repo.List(ctx, "ws1", 10)
	if len(list) != 0 {
		t.Fatalf("clear: %#v", list)
	}
}

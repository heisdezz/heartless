package history

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	domain "heartless/internal/domain/history"
	sqlitestore "heartless/internal/storage/sqlite"
	histrepo "heartless/internal/storage/sqlite/history"
)

func TestServiceImplRecordRetention(t *testing.T) {
	db, err := sqlitestore.Open(context.Background(), sqlitestore.Config{Path: filepath.Join(t.TempDir(), "h.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := histrepo.New(db)
	svc := New(store, Config{RetentionLimit: 2})
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		e := domain.New("ws1")
		e.RequestID = "r"
		e.CreatedAt = time.Now().UTC().Add(time.Duration(i) * time.Second)
		if err := svc.Record(ctx, e); err != nil {
			t.Fatal(err)
		}
	}
	list, err := svc.List(ctx, "ws1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 entries after retention, got %d", len(list))
	}
}

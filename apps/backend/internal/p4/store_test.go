package p4

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func newTestSQLLockStore(t *testing.T) *sqlLockStore {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`
		CREATE TABLE p4_file_locks (
			file_path TEXT PRIMARY KEY,
			task_id TEXT NOT NULL,
			changelist TEXT DEFAULT '',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`); err != nil {
		t.Fatalf("schema: %v", err)
	}
	return NewSQLLockStore(db)
}

func TestSQLLockStore_SetGetRelease(t *testing.T) {
	store := newTestSQLLockStore(t)
	ctx := context.Background()
	if err := store.SetLocks(ctx, "task-a", "1001", []string{"//depot/a.cs"}); err != nil {
		t.Fatalf("SetLocks: %v", err)
	}
	owner, ok, err := store.GetOwner(ctx, "//depot/a.cs")
	if err != nil || !ok || owner != "task-a" {
		t.Fatalf("GetOwner = (%q, %v, %v)", owner, ok, err)
	}
	files, err := store.ListByTask(ctx, "task-a")
	if err != nil || len(files) != 1 {
		t.Fatalf("ListByTask = %v, %v", files, err)
	}
	if err := store.ReleaseByTask(ctx, "task-a"); err != nil {
		t.Fatalf("ReleaseByTask: %v", err)
	}
	_, ok, err = store.GetOwner(ctx, "//depot/a.cs")
	if err != nil || ok {
		t.Fatalf("expected lock released, ok=%v err=%v", ok, err)
	}
}

package p4

import (
	"context"
	"database/sql"
	"time"
)

// LockStore persists task↔file lock mappings.
type LockStore interface {
	GetOwner(ctx context.Context, filePath string) (taskID string, ok bool, err error)
	SetLocks(ctx context.Context, taskID, changelist string, files []string) error
	ReleaseByTask(ctx context.Context, taskID string) error
	ListByTask(ctx context.Context, taskID string) ([]string, error)
}

type memoryLockStore struct {
	fileLocks map[string]string
}

func NewMemoryLockStore() *memoryLockStore {
	return &memoryLockStore{fileLocks: map[string]string{}}
}

func (s *memoryLockStore) GetOwner(_ context.Context, filePath string) (string, bool, error) {
	owner, ok := s.fileLocks[filePath]
	return owner, ok, nil
}

func (s *memoryLockStore) SetLocks(_ context.Context, taskID, _ string, files []string) error {
	for _, file := range files {
		s.fileLocks[file] = taskID
	}
	return nil
}

func (s *memoryLockStore) ReleaseByTask(_ context.Context, taskID string) error {
	for file, owner := range s.fileLocks {
		if owner == taskID {
			delete(s.fileLocks, file)
		}
	}
	return nil
}

func (s *memoryLockStore) ListByTask(_ context.Context, taskID string) ([]string, error) {
	files := make([]string, 0)
	for file, owner := range s.fileLocks {
		if owner == taskID {
			files = append(files, file)
		}
	}
	return files, nil
}

type sqlLockStore struct {
	db *sql.DB
}

func NewSQLLockStore(db *sql.DB) *sqlLockStore {
	return &sqlLockStore{db: db}
}

func (s *sqlLockStore) GetOwner(ctx context.Context, filePath string) (string, bool, error) {
	var taskID string
	err := s.db.QueryRowContext(ctx, `SELECT task_id FROM p4_file_locks WHERE file_path = ?`, filePath).Scan(&taskID)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return taskID, true, nil
}

func (s *sqlLockStore) SetLocks(ctx context.Context, taskID, changelist string, files []string) error {
	now := time.Now().UTC()
	for _, file := range files {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO p4_file_locks (file_path, task_id, changelist, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(file_path) DO UPDATE SET task_id = excluded.task_id, changelist = excluded.changelist, updated_at = excluded.updated_at
		`, file, taskID, changelist, now, now)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *sqlLockStore) ReleaseByTask(ctx context.Context, taskID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM p4_file_locks WHERE task_id = ?`, taskID)
	return err
}

func (s *sqlLockStore) ListByTask(ctx context.Context, taskID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT file_path FROM p4_file_locks WHERE task_id = ? ORDER BY file_path`, taskID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var files []string
	for rows.Next() {
		var file string
		if err := rows.Scan(&file); err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, rows.Err()
}

package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const (
	defaultBusyTimeout = 5 * time.Second

	// defaultSQLiteReaderConns is the number of concurrent read connections.
	// SQLite WAL mode allows many readers alongside a single writer; 4 is a
	// reasonable default for a desktop/server workload.
	defaultSQLiteReaderConns = 4
)

// OpenSQLite opens a SQLite database configured for writes (single connection).
func OpenSQLite(dbPath string) (*sql.DB, error) {
	normalizedPath := normalizeSQLitePath(dbPath)
	if err := ensureSQLiteDir(normalizedPath); err != nil {
		return nil, fmt.Errorf("failed to prepare database path: %w", err)
	}
	if err := ensureSQLiteFile(normalizedPath); err != nil {
		return nil, fmt.Errorf("failed to create database file: %w", err)
	}

	dsn := normalizedPath
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Apply PRAGMAs for WAL mode, FK, and busy timeout (modernc does not use DSN params).
	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA foreign_keys = ON",
		fmt.Sprintf("PRAGMA busy_timeout = %d", int(defaultBusyTimeout/time.Millisecond)),
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to set pragma %q: %w", p, err)
		}
	}

	// Single writer connection: serializes writes and avoids SQLITE_BUSY.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	return db, nil
}

// OpenSQLiteReader opens a read-only SQLite connection pool with multiple
// concurrent connections. Combined with WAL mode, this allows readers to
// proceed without blocking on (or being blocked by) writes.
func OpenSQLiteReader(dbPath string) (*sql.DB, error) {
	normalizedPath := normalizeSQLitePath(dbPath)

	dsn := "file:" + normalizedPath + "?mode=ro"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open read-only database: %w", err)
	}

	// Apply PRAGMAs for read connections too (FK, busy timeout).
	if _, err := db.Exec(fmt.Sprintf("PRAGMA busy_timeout = %d", int(defaultBusyTimeout/time.Millisecond))); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set busy_timeout: %w", err)
	}

	db.SetMaxOpenConns(defaultSQLiteReaderConns)
	db.SetMaxIdleConns(defaultSQLiteReaderConns)

	return db, nil
}

func ensureSQLiteDir(dbPath string) error {
	dir := filepath.Dir(dbPath)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func ensureSQLiteFile(dbPath string) error {
	file, err := os.OpenFile(dbPath, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	return file.Close()
}

func normalizeSQLitePath(dbPath string) string {
	if dbPath == "" {
		return dbPath
	}
	abs, err := filepath.Abs(dbPath)
	if err != nil {
		return dbPath
	}
	return abs
}

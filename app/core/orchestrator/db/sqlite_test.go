package db

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestNewSQLiteDBReturnsLockErrorWhenSchemaLocked(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "alter0.db")

	lockedConn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open lock connection: %v", err)
	}
	defer lockedConn.Close()

	if _, err := lockedConn.Exec(`CREATE TABLE IF NOT EXISTS lock_probe(id INTEGER PRIMARY KEY, value TEXT)`); err != nil {
		t.Fatalf("create lock table: %v", err)
	}

	if _, err := lockedConn.Exec(`BEGIN EXCLUSIVE`); err != nil {
		t.Fatalf("acquire exclusive lock: %v", err)
	}
	defer func() {
		_, _ = lockedConn.Exec(`ROLLBACK`)
	}()

	if _, err := lockedConn.Exec(`INSERT INTO lock_probe(value) VALUES('hold')`); err != nil {
		t.Fatalf("hold write lock: %v", err)
	}

	_, err = NewSQLiteDB(tempDir)
	if err == nil {
		t.Fatal("expected lock error, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "locked") {
		t.Fatalf("expected lock error, got: %v", err)
	}
}

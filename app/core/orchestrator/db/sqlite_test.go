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

func TestNewSQLiteDBMigratesV2ToV3WithoutDataLoss(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "alter0.db")

	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open legacy db: %v", err)
	}
	if err := seedSchemaV2(conn); err != nil {
		_ = conn.Close()
		t.Fatalf("seed schema v2: %v", err)
	}
	if _, err := conn.Exec(`
INSERT INTO tasks(id, user_id, title, status, created_at, updated_at)
VALUES('task-1', 'user-1', 'legacy task', 'open', 100, 200)`); err != nil {
		_ = conn.Close()
		t.Fatalf("insert legacy task: %v", err)
	}
	_ = conn.Close()

	database, err := NewSQLiteDB(tempDir)
	if err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	defer database.Close()

	var schemaVersion string
	if err := database.Conn().QueryRow(`SELECT value FROM schema_meta WHERE key = 'schema_version'`).Scan(&schemaVersion); err != nil {
		t.Fatalf("read schema version: %v", err)
	}
	if schemaVersion != "3" {
		t.Fatalf("expected schema version 3, got %s", schemaVersion)
	}

	var title string
	if err := database.Conn().QueryRow(`SELECT title FROM tasks WHERE id = 'task-1'`).Scan(&title); err != nil {
		t.Fatalf("read migrated task: %v", err)
	}
	if title != "legacy task" {
		t.Fatalf("unexpected migrated title: %s", title)
	}

	if !tableExists(t, database.Conn(), "task_memory") {
		t.Fatal("expected task_memory table to exist after migration")
	}

	backupPaths, err := filepath.Glob(dbPath + ".migration-*.bak")
	if err != nil {
		t.Fatalf("glob backup path: %v", err)
	}
	if len(backupPaths) != 1 {
		t.Fatalf("expected exactly one migration backup, got %d", len(backupPaths))
	}
}

func TestNewSQLiteDBRollsBackFromBackupWhenMigrationFails(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "alter0.db")

	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open legacy db: %v", err)
	}
	if err := seedSchemaV2(conn); err != nil {
		_ = conn.Close()
		t.Fatalf("seed schema v2: %v", err)
	}
	if _, err := conn.Exec(`
CREATE TABLE task_memory (
	task_id TEXT PRIMARY KEY,
	summary TEXT NOT NULL
)`); err != nil {
		_ = conn.Close()
		t.Fatalf("create malformed task_memory table: %v", err)
	}
	_ = conn.Close()

	_, err = NewSQLiteDB(tempDir)
	if err == nil {
		t.Fatal("expected migration failure, got nil")
	}
	if !strings.Contains(err.Error(), "rolled back from") {
		t.Fatalf("expected rollback hint in error, got: %v", err)
	}

	checkConn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("re-open db after rollback: %v", err)
	}
	defer checkConn.Close()

	var schemaVersion string
	if err := checkConn.QueryRow(`SELECT value FROM schema_meta WHERE key = 'schema_version'`).Scan(&schemaVersion); err != nil {
		t.Fatalf("read schema version after rollback: %v", err)
	}
	if schemaVersion != "2" {
		t.Fatalf("expected rolled back schema version 2, got %s", schemaVersion)
	}

	backupPaths, err := filepath.Glob(dbPath + ".migration-*.bak")
	if err != nil {
		t.Fatalf("glob backup path: %v", err)
	}
	if len(backupPaths) != 1 {
		t.Fatalf("expected exactly one migration backup, got %d", len(backupPaths))
	}
}

func seedSchemaV2(conn *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS schema_meta (key TEXT PRIMARY KEY, value TEXT NOT NULL)`,
		`INSERT INTO schema_meta (key, value) VALUES ('schema_version', '2')
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			title TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			closed_at INTEGER,
			last_channel_id TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			channel_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			meta JSON
		)`,
		`CREATE TABLE IF NOT EXISTS user_state (
			user_id TEXT PRIMARY KEY,
			forced_task_id TEXT,
			updated_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_user_status_updated ON tasks(user_id, status, updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_task_created ON messages(task_id, created_at ASC)`,
	}

	for _, query := range queries {
		if _, err := conn.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func tableExists(t *testing.T, conn *sql.DB, table string) bool {
	t.Helper()

	var name string
	err := conn.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name)
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		t.Fatalf("query sqlite_master for %s: %v", table, err)
	}
	return name == table
}

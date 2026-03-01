package db

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	_ "modernc.org/sqlite"
)

const currentSchemaVersion = 3

type DB struct {
	conn *sql.DB
	path string
}

type migrationError struct {
	backupPath string
	cause      error
}

func (e *migrationError) Error() string {
	return e.cause.Error()
}

func (e *migrationError) Unwrap() error {
	return e.cause
}

func NewSQLiteDB(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "alter0.db")
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	database := &DB{conn: conn, path: dbPath}
	if err := database.initSchema(); err != nil {
		_ = conn.Close()

		var migrateErr *migrationError
		if errors.As(err, &migrateErr) && migrateErr.backupPath != "" {
			if rollbackErr := restoreFromBackup(migrateErr.backupPath, dbPath); rollbackErr != nil {
				return nil, fmt.Errorf("failed to init schema: %w; rollback from %s also failed: %v", migrateErr.cause, migrateErr.backupPath, rollbackErr)
			}
			return nil, fmt.Errorf("failed to init schema (rolled back from %s): %w", migrateErr.backupPath, migrateErr.cause)
		}
		return nil, fmt.Errorf("failed to init schema: %w", err)
	}
	return database, nil
}

func (d *DB) initSchema() error {
	tx, err := d.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`CREATE TABLE IF NOT EXISTS schema_meta (key TEXT PRIMARY KEY, value TEXT NOT NULL)`); err != nil {
		return err
	}

	version, err := readSchemaVersion(tx)
	if err != nil {
		return err
	}

	if version > currentSchemaVersion {
		return fmt.Errorf("db schema version %d is newer than runtime version %d", version, currentSchemaVersion)
	}

	var backupPath string
	if version > 0 && version < currentSchemaVersion {
		backupPath, err = d.createMigrationBackup()
		if err != nil {
			return fmt.Errorf("create migration backup: %w", err)
		}
	}

	if err := applyMigrations(tx, version); err != nil {
		if backupPath != "" {
			return &migrationError{backupPath: backupPath, cause: err}
		}
		return err
	}

	return tx.Commit()
}

func readSchemaVersion(tx *sql.Tx) (int, error) {
	var versionText string
	err := tx.QueryRow(`SELECT value FROM schema_meta WHERE key = 'schema_version'`).Scan(&versionText)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	version, parseErr := strconv.Atoi(versionText)
	if parseErr != nil {
		return 0, fmt.Errorf("parse schema version %q: %w", versionText, parseErr)
	}
	if version < 0 {
		return 0, fmt.Errorf("invalid schema version %d", version)
	}
	return version, nil
}

func applyMigrations(tx *sql.Tx, version int) error {
	for version < currentSchemaVersion {
		nextVersion, err := applyNextMigration(tx, version)
		if err != nil {
			return err
		}
		if err := writeSchemaVersion(tx, nextVersion); err != nil {
			return err
		}
		version = nextVersion
	}
	return nil
}

func applyNextMigration(tx *sql.Tx, version int) (int, error) {
	switch version {
	case 0, 1:
		if err := migrateToTaskCoreSchema(tx); err != nil {
			return version, fmt.Errorf("migrate schema %d -> 2: %w", version, err)
		}
		return 2, nil
	case 2:
		if err := migrateToTaskMemory(tx); err != nil {
			return version, fmt.Errorf("migrate schema 2 -> 3: %w", err)
		}
		return 3, nil
	default:
		return version, fmt.Errorf("unsupported schema migration source version %d", version)
	}
}

func migrateToTaskCoreSchema(tx *sql.Tx) error {
	createTasks := `
CREATE TABLE IF NOT EXISTS tasks (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	title TEXT NOT NULL,
	status TEXT NOT NULL,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL,
	closed_at INTEGER,
	last_channel_id TEXT
);`
	if _, err := tx.Exec(createTasks); err != nil {
		return err
	}

	createMessages := `
CREATE TABLE IF NOT EXISTS messages (
	id TEXT PRIMARY KEY,
	task_id TEXT NOT NULL,
	user_id TEXT NOT NULL,
	channel_id TEXT NOT NULL,
	role TEXT NOT NULL,
	content TEXT NOT NULL,
	created_at INTEGER NOT NULL,
	meta JSON
);`
	if _, err := tx.Exec(createMessages); err != nil {
		return err
	}

	createUserState := `
CREATE TABLE IF NOT EXISTS user_state (
	user_id TEXT PRIMARY KEY,
	forced_task_id TEXT,
	updated_at INTEGER NOT NULL
);`
	if _, err := tx.Exec(createUserState); err != nil {
		return err
	}

	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_tasks_user_status_updated ON tasks(user_id, status, updated_at DESC)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_messages_task_created ON messages(task_id, created_at ASC)`); err != nil {
		return err
	}

	return nil
}

func migrateToTaskMemory(tx *sql.Tx) error {
	createTaskMemory := `
CREATE TABLE IF NOT EXISTS task_memory (
	task_id TEXT PRIMARY KEY,
	summary TEXT NOT NULL,
	updated_at INTEGER NOT NULL
);`
	if _, err := tx.Exec(createTaskMemory); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_task_memory_updated ON task_memory(updated_at DESC)`); err != nil {
		return err
	}
	return nil
}

func writeSchemaVersion(tx *sql.Tx, version int) error {
	if _, err := tx.Exec(`
INSERT INTO schema_meta (key, value) VALUES ('schema_version', ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value`, strconv.Itoa(version)); err != nil {
		return err
	}
	return nil
}

func (d *DB) createMigrationBackup() (string, error) {
	if _, err := d.conn.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return "", fmt.Errorf("checkpoint wal: %w", err)
	}

	backupPath := fmt.Sprintf("%s.migration-%d.bak", d.path, time.Now().Unix())
	if err := copyFile(d.path, backupPath); err != nil {
		return "", err
	}
	return backupPath, nil
}

func restoreFromBackup(backupPath, dbPath string) error {
	if err := copyFile(backupPath, dbPath); err != nil {
		return err
	}
	_ = os.Remove(dbPath + "-wal")
	_ = os.Remove(dbPath + "-shm")
	return nil
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	target, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer target.Close()

	if _, err := io.Copy(target, source); err != nil {
		return err
	}
	return target.Sync()
}

func (d *DB) Conn() *sql.DB {
	return d.conn
}

func (d *DB) Close() error {
	return d.conn.Close()
}

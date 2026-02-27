package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const currentSchemaVersion = "3"

type DB struct {
	conn *sql.DB
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

	database := &DB{conn: conn}
	if err := database.initSchema(); err != nil {
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

	var version string
	err = tx.QueryRow(`SELECT value FROM schema_meta WHERE key = 'schema_version'`).Scan(&version)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if version != currentSchemaVersion {
		// destructive reset for migration to task-centric schema
		if _, err := tx.Exec(`DROP TABLE IF EXISTS messages`); err != nil {
			return err
		}
		if _, err := tx.Exec(`DROP TABLE IF EXISTS tasks`); err != nil {
			return err
		}
		if _, err := tx.Exec(`DROP TABLE IF EXISTS user_state`); err != nil {
			return err
		}
		if _, err := tx.Exec(`DROP TABLE IF EXISTS task_memory`); err != nil {
			return err
		}
	}

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

	createTaskMemory := `
CREATE TABLE IF NOT EXISTS task_memory (
	task_id TEXT PRIMARY KEY,
	summary TEXT NOT NULL,
	updated_at INTEGER NOT NULL
);`
	if _, err := tx.Exec(createTaskMemory); err != nil {
		return err
	}

	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_tasks_user_status_updated ON tasks(user_id, status, updated_at DESC)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_messages_task_created ON messages(task_id, created_at ASC)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_task_memory_updated ON task_memory(updated_at DESC)`); err != nil {
		return err
	}

	if _, err := tx.Exec(`
INSERT INTO schema_meta (key, value) VALUES ('schema_version', ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value`, currentSchemaVersion); err != nil {
		return err
	}

	return tx.Commit()
}

func (d *DB) Conn() *sql.DB {
	return d.conn
}

func (d *DB) Close() error {
	return d.conn.Close()
}

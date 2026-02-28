package task

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"alter0/app/core/orchestrator/db"
)

type Store struct {
	db      *db.DB
	counter uint64
}

type Stats struct {
	Open       int `json:"open"`
	Closed     int `json:"closed"`
	Total      int `json:"total"`
	WithMemory int `json:"with_memory"`
}

func NewStore(database *db.DB) *Store {
	return &Store{db: database}
}

func (s *Store) CreateTask(ctx context.Context, userID string, title string, channelID string) (Task, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return Task{}, fmt.Errorf("user_id is required")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Untitled Task"
	}
	now := time.Now().Unix()
	id := s.newID("task")
	query := `INSERT INTO tasks (id, user_id, title, status, created_at, updated_at, closed_at, last_channel_id) VALUES (?, ?, ?, 'open', ?, ?, NULL, ?)`
	if _, err := s.db.Conn().ExecContext(ctx, query, id, userID, title, now, now, channelID); err != nil {
		return Task{}, err
	}
	return Task{
		ID:            id,
		UserID:        userID,
		Title:         title,
		Status:        "open",
		CreatedAt:     now,
		UpdatedAt:     now,
		LastChannelID: channelID,
	}, nil
}

func (s *Store) GetTask(ctx context.Context, taskID string) (Task, error) {
	query := `SELECT id, user_id, title, status, created_at, updated_at, COALESCE(closed_at, 0), COALESCE(last_channel_id, '') FROM tasks WHERE id = ?`
	var t Task
	err := s.db.Conn().QueryRowContext(ctx, query, taskID).Scan(
		&t.ID,
		&t.UserID,
		&t.Title,
		&t.Status,
		&t.CreatedAt,
		&t.UpdatedAt,
		&t.ClosedAt,
		&t.LastChannelID,
	)
	if err != nil {
		return Task{}, err
	}
	return t, nil
}

func (s *Store) ListOpenTasks(ctx context.Context, userID string, limit int) ([]Task, error) {
	return s.ListTasks(ctx, userID, "open", limit)
}

func (s *Store) ListTasks(ctx context.Context, userID string, status string, limit int) ([]Task, error) {
	if limit <= 0 {
		limit = 20
	}
	var (
		query string
		args  []interface{}
	)
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "all":
		query = `SELECT id, user_id, title, status, created_at, updated_at, COALESCE(closed_at, 0), COALESCE(last_channel_id, '') FROM tasks WHERE user_id = ? ORDER BY updated_at DESC LIMIT ?`
		args = []interface{}{userID, limit}
	case "open", "closed":
		query = `SELECT id, user_id, title, status, created_at, updated_at, COALESCE(closed_at, 0), COALESCE(last_channel_id, '') FROM tasks WHERE user_id = ? AND status = ? ORDER BY updated_at DESC LIMIT ?`
		args = []interface{}{userID, status, limit}
	default:
		return nil, fmt.Errorf("invalid status: %s", status)
	}

	rows, err := s.db.Conn().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Task, 0, limit)
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.UserID, &t.Title, &t.Status, &t.CreatedAt, &t.UpdatedAt, &t.ClosedAt, &t.LastChannelID); err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	return items, rows.Err()
}

func (s *Store) GetLatestOpenTask(ctx context.Context, userID string) (Task, error) {
	items, err := s.ListOpenTasks(ctx, userID, 1)
	if err != nil {
		return Task{}, err
	}
	if len(items) == 0 {
		return Task{}, sql.ErrNoRows
	}
	return items[0], nil
}

func (s *Store) AppendMessage(ctx context.Context, taskID string, userID string, channelID string, role string, content string, meta map[string]interface{}) error {
	if strings.TrimSpace(taskID) == "" {
		return fmt.Errorf("task_id is required")
	}
	now := time.Now().Unix()
	msgID := s.newID("msg")
	if meta == nil {
		meta = map[string]interface{}{}
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	tx, err := s.db.Conn().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	insertMsg := `INSERT INTO messages (id, task_id, user_id, channel_id, role, content, created_at, meta) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	if _, err := tx.ExecContext(ctx, insertMsg, msgID, taskID, userID, channelID, role, content, now, metaJSON); err != nil {
		return err
	}

	updateTask := `UPDATE tasks SET updated_at = ?, last_channel_id = ? WHERE id = ?`
	if _, err := tx.ExecContext(ctx, updateTask, now, channelID, taskID); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) GetTaskHistory(ctx context.Context, taskID string, limit int) ([]TaskMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `SELECT id, task_id, user_id, channel_id, role, content, created_at, meta FROM messages WHERE task_id = ? ORDER BY created_at DESC LIMIT ?`
	rows, err := s.db.Conn().QueryContext(ctx, query, taskID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]TaskMessage, 0, limit)
	for rows.Next() {
		var (
			m        TaskMessage
			metaJSON []byte
		)
		if err := rows.Scan(&m.ID, &m.TaskID, &m.UserID, &m.ChannelID, &m.Role, &m.Content, &m.CreatedAt, &metaJSON); err != nil {
			return nil, err
		}
		if len(metaJSON) > 0 {
			_ = json.Unmarshal(metaJSON, &m.Meta)
		}
		items = append(items, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// reverse to chronological order
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
	return items, nil
}

func (s *Store) UpsertTaskMemory(ctx context.Context, taskID string, summary string) error {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return fmt.Errorf("task_id is required")
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return fmt.Errorf("summary is required")
	}
	now := time.Now().Unix()
	query := `
INSERT INTO task_memory (task_id, summary, updated_at) VALUES (?, ?, ?)
ON CONFLICT(task_id) DO UPDATE SET summary = excluded.summary, updated_at = excluded.updated_at`
	_, err := s.db.Conn().ExecContext(ctx, query, taskID, summary, now)
	return err
}

func (s *Store) GetTaskMemory(ctx context.Context, taskID string) (TaskMemory, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return TaskMemory{}, fmt.Errorf("task_id is required")
	}
	query := `SELECT task_id, summary, updated_at FROM task_memory WHERE task_id = ?`
	var memory TaskMemory
	if err := s.db.Conn().QueryRowContext(ctx, query, taskID).Scan(&memory.TaskID, &memory.Summary, &memory.UpdatedAt); err != nil {
		return TaskMemory{}, err
	}
	return memory, nil
}

func (s *Store) DeleteTaskMemory(ctx context.Context, taskID string) error {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return fmt.Errorf("task_id is required")
	}
	_, err := s.db.Conn().ExecContext(ctx, `DELETE FROM task_memory WHERE task_id = ?`, taskID)
	return err
}

func (s *Store) PruneTaskMemoryByClosedAt(ctx context.Context, closedBeforeUnix int64) (int64, error) {
	if closedBeforeUnix <= 0 {
		return 0, fmt.Errorf("closed_before_unix must be greater than zero")
	}
	query := `
DELETE FROM task_memory
WHERE task_id IN (
	SELECT id FROM tasks
	WHERE status = 'closed' AND COALESCE(closed_at, 0) > 0 AND closed_at < ?
)`
	result, err := s.db.Conn().ExecContext(ctx, query, closedBeforeUnix)
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return affected, nil
}

func (s *Store) PruneTaskMemoryByOpenUpdatedAt(ctx context.Context, updatedBeforeUnix int64) (int64, error) {
	if updatedBeforeUnix <= 0 {
		return 0, fmt.Errorf("updated_before_unix must be greater than zero")
	}
	query := `
DELETE FROM task_memory
WHERE updated_at < ?
AND task_id IN (
	SELECT id FROM tasks
	WHERE status = 'open'
)`
	result, err := s.db.Conn().ExecContext(ctx, query, updatedBeforeUnix)
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return affected, nil
}

func (s *Store) ExportTaskMemorySnapshot(ctx context.Context, userID string, limit int) ([]TaskMemorySnapshot, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	if limit <= 0 {
		limit = 50
	}

	query := `
SELECT t.user_id, m.task_id, m.summary, m.updated_at
FROM task_memory m
JOIN tasks t ON t.id = m.task_id
WHERE t.user_id = ?
ORDER BY m.updated_at DESC
LIMIT ?`
	rows, err := s.db.Conn().QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]TaskMemorySnapshot, 0, limit)
	for rows.Next() {
		var item TaskMemorySnapshot
		if err := rows.Scan(&item.UserID, &item.TaskID, &item.Summary, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) RestoreTaskMemorySnapshot(ctx context.Context, userID string, snapshots []TaskMemorySnapshot) (int, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return 0, fmt.Errorf("user_id is required")
	}

	tx, err := s.db.Conn().BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	applied := 0
	for _, item := range snapshots {
		taskID := strings.TrimSpace(item.TaskID)
		summary := strings.TrimSpace(item.Summary)
		if taskID == "" || summary == "" {
			continue
		}

		taskOwner, err := s.lookupTaskOwnerTx(ctx, tx, taskID)
		if err != nil {
			if err == sql.ErrNoRows {
				return 0, fmt.Errorf("task not found for snapshot: %s", taskID)
			}
			return 0, err
		}
		if taskOwner != userID {
			return 0, fmt.Errorf("task does not belong to user: %s", taskID)
		}

		if err := s.upsertTaskMemoryTx(ctx, tx, taskID, summary, item.UpdatedAt); err != nil {
			return 0, err
		}
		applied++
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return applied, nil
}

func (s *Store) lookupTaskOwnerTx(ctx context.Context, tx *sql.Tx, taskID string) (string, error) {
	var owner string
	err := tx.QueryRowContext(ctx, `SELECT user_id FROM tasks WHERE id = ?`, taskID).Scan(&owner)
	if err != nil {
		return "", err
	}
	return owner, nil
}

func (s *Store) upsertTaskMemoryTx(ctx context.Context, tx *sql.Tx, taskID string, summary string, updatedAt int64) error {
	if updatedAt <= 0 {
		updatedAt = time.Now().Unix()
	}
	query := `
INSERT INTO task_memory (task_id, summary, updated_at) VALUES (?, ?, ?)
ON CONFLICT(task_id) DO UPDATE SET summary = excluded.summary, updated_at = excluded.updated_at`
	_, err := tx.ExecContext(ctx, query, taskID, summary, updatedAt)
	return err
}

func (s *Store) CloseTask(ctx context.Context, taskID string) error {
	now := time.Now().Unix()
	query := `UPDATE tasks SET status = 'closed', closed_at = ?, updated_at = ? WHERE id = ? AND status != 'closed'`
	_, err := s.db.Conn().ExecContext(ctx, query, now, now, taskID)
	return err
}

func (s *Store) GlobalStats(ctx context.Context) (Stats, error) {
	stats := Stats{}
	rows, err := s.db.Conn().QueryContext(ctx, `SELECT status, COUNT(1) FROM tasks GROUP BY status`)
	if err != nil {
		return Stats{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return Stats{}, err
		}
		switch strings.ToLower(strings.TrimSpace(status)) {
		case "open":
			stats.Open = count
		case "closed":
			stats.Closed = count
		}
		stats.Total += count
	}
	if err := rows.Err(); err != nil {
		return Stats{}, err
	}

	if err := s.db.Conn().QueryRowContext(ctx, `SELECT COUNT(1) FROM task_memory`).Scan(&stats.WithMemory); err != nil {
		return Stats{}, err
	}
	return stats, nil
}

func (s *Store) SetForcedTask(ctx context.Context, userID string, taskID string) error {
	now := time.Now().Unix()
	query := `
INSERT INTO user_state (user_id, forced_task_id, updated_at) VALUES (?, ?, ?)
ON CONFLICT(user_id) DO UPDATE SET forced_task_id = excluded.forced_task_id, updated_at = excluded.updated_at`
	_, err := s.db.Conn().ExecContext(ctx, query, userID, taskID, now)
	return err
}

func (s *Store) ConsumeForcedTask(ctx context.Context, userID string) (string, error) {
	tx, err := s.db.Conn().BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	var forced sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT forced_task_id FROM user_state WHERE user_id = ?`, userID).Scan(&forced); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE user_state SET forced_task_id = NULL, updated_at = ? WHERE user_id = ?`, time.Now().Unix(), userID); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	if !forced.Valid {
		return "", nil
	}
	return forced.String, nil
}

func (s *Store) PeekForcedTask(ctx context.Context, userID string) (string, error) {
	var forced sql.NullString
	err := s.db.Conn().QueryRowContext(ctx, `SELECT forced_task_id FROM user_state WHERE user_id = ?`, userID).Scan(&forced)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	if !forced.Valid {
		return "", nil
	}
	return forced.String, nil
}

func (s *Store) newID(prefix string) string {
	seq := atomic.AddUint64(&s.counter, 1)
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), seq)
}

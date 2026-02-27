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

func (s *Store) CloseTask(ctx context.Context, taskID string) error {
	now := time.Now().Unix()
	query := `UPDATE tasks SET status = 'closed', closed_at = ?, updated_at = ? WHERE id = ? AND status != 'closed'`
	_, err := s.db.Conn().ExecContext(ctx, query, now, now, taskID)
	return err
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

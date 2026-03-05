package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"

	shareddomain "alter0/internal/shared/domain"
)

type TaskStatus string

const (
	TaskStatusQueued   TaskStatus = "queued"
	TaskStatusRunning  TaskStatus = "running"
	TaskStatusSuccess  TaskStatus = "success"
	TaskStatusFailed   TaskStatus = "failed"
	TaskStatusCanceled TaskStatus = "canceled"
)

type TaskLogLevel string

const (
	TaskLogLevelInfo  TaskLogLevel = "info"
	TaskLogLevelWarn  TaskLogLevel = "warn"
	TaskLogLevelError TaskLogLevel = "error"
)

type TaskLog struct {
	Seq       int          `json:"seq"`
	Stage     string       `json:"stage,omitempty"`
	CreatedAt time.Time    `json:"created_at,omitempty"`
	Timestamp time.Time    `json:"timestamp,omitempty"`
	Level     TaskLogLevel `json:"level"`
	Message   string       `json:"message"`
}

type TaskArtifact struct {
	ArtifactID   string    `json:"artifact_id"`
	ArtifactType string    `json:"artifact_type,omitempty"`
	Name         string    `json:"name"`
	ContentType  string    `json:"content_type"`
	Size         int64     `json:"size"`
	Content      string    `json:"content"`
	URI          string    `json:"uri,omitempty"`
	Summary      string    `json:"summary,omitempty"`
	DownloadURL  string    `json:"download_url,omitempty"`
	PreviewURL   string    `json:"preview_url,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type TaskResult struct {
	Route     shareddomain.Route `json:"route,omitempty"`
	Output    string             `json:"output,omitempty"`
	ErrorCode string             `json:"error_code,omitempty"`
	Metadata  map[string]string  `json:"metadata,omitempty"`
}

type TaskSummary struct {
	TaskID     string     `json:"task_id"`
	TaskType   string     `json:"task_type"`
	Goal       string     `json:"goal"`
	Result     string     `json:"result"`
	Status     TaskStatus `json:"status"`
	FinishedAt time.Time  `json:"finished_at"`
	Tags       []string   `json:"tags,omitempty"`
}

type TaskMessageLink struct {
	TaskID           string `json:"task_id,omitempty"`
	SessionID        string `json:"session_id,omitempty"`
	RequestMessageID string `json:"request_message_id,omitempty"`
	ResultMessageID  string `json:"result_message_id,omitempty"`
}

type Task struct {
	ID              string            `json:"id"`
	SessionID       string            `json:"session_id"`
	SourceMessageID string            `json:"source_message_id"`
	MessageID       string            `json:"message_id,omitempty"`
	TaskType        string            `json:"task_type,omitempty"`
	Status          TaskStatus        `json:"status"`
	Phase           string            `json:"phase,omitempty"`
	Progress        int               `json:"progress"`
	QueuePosition   int               `json:"queue_position,omitempty"`
	QueueWaitMS     int64             `json:"queue_wait_ms,omitempty"`
	RetryCount      int               `json:"retry_count"`
	MaxRetries      int               `json:"max_retries"`
	TimeoutMS       int64             `json:"timeout_ms"`
	TimeoutAt       time.Time         `json:"timeout_at,omitempty"`
	AcceptedAt      time.Time         `json:"accepted_at,omitempty"`
	PhaseUpdatedAt  time.Time         `json:"phase_updated_at,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	StartedAt       time.Time         `json:"started_at,omitempty"`
	FinishedAt      time.Time         `json:"finished_at,omitempty"`
	RequestContent  string            `json:"request_content"`
	RequestMetadata map[string]string `json:"request_metadata,omitempty"`
	Summary         string            `json:"summary,omitempty"`
	TaskSummary     TaskSummary       `json:"task_summary,omitempty"`
	ErrorCode       string            `json:"error_code,omitempty"`
	ErrorMessage    string            `json:"error_message,omitempty"`
	Result          TaskResult        `json:"result,omitempty"`
	Logs            []TaskLog         `json:"logs,omitempty"`
	Artifacts       []TaskArtifact    `json:"artifacts,omitempty"`
	MessageLink     TaskMessageLink   `json:"message_link,omitempty"`
}

func (s TaskStatus) IsValid() bool {
	switch s {
	case TaskStatusQueued, TaskStatusRunning, TaskStatusSuccess, TaskStatusFailed, TaskStatusCanceled:
		return true
	default:
		return false
	}
}

func (s TaskStatus) IsTerminal() bool {
	return s == TaskStatusSuccess || s == TaskStatusFailed || s == TaskStatusCanceled
}

func (l TaskLogLevel) IsValid() bool {
	switch l {
	case TaskLogLevelInfo, TaskLogLevelWarn, TaskLogLevelError:
		return true
	default:
		return false
	}
}

func (s TaskSummary) IsZero() bool {
	return strings.TrimSpace(s.TaskID) == "" &&
		strings.TrimSpace(s.TaskType) == "" &&
		strings.TrimSpace(s.Goal) == "" &&
		strings.TrimSpace(s.Result) == "" &&
		strings.TrimSpace(string(s.Status)) == "" &&
		s.FinishedAt.IsZero() &&
		len(s.Tags) == 0
}

func (s TaskSummary) Validate() error {
	if strings.TrimSpace(s.TaskID) == "" {
		return errors.New("task_summary.task_id is required")
	}
	if strings.TrimSpace(s.TaskType) == "" {
		return errors.New("task_summary.task_type is required")
	}
	if strings.TrimSpace(s.Goal) == "" {
		return errors.New("task_summary.goal is required")
	}
	if strings.TrimSpace(s.Result) == "" {
		return errors.New("task_summary.result is required")
	}
	if !s.Status.IsValid() {
		return errors.New("task_summary.status is invalid")
	}
	if s.FinishedAt.IsZero() {
		return errors.New("task_summary.finished_at is required")
	}
	return nil
}

func (t Task) Validate() error {
	if strings.TrimSpace(t.ID) == "" {
		return errors.New("task id is required")
	}
	if strings.TrimSpace(t.SessionID) == "" {
		return errors.New("session_id is required")
	}
	sourceMessageID := strings.TrimSpace(t.SourceMessageID)
	if sourceMessageID == "" {
		sourceMessageID = strings.TrimSpace(t.MessageID)
	}
	if sourceMessageID == "" {
		return errors.New("source_message_id is required")
	}
	if !t.Status.IsValid() {
		return errors.New("task status is invalid")
	}
	if strings.TrimSpace(t.Phase) == "" {
		return errors.New("phase is required")
	}
	if t.Progress < 0 || t.Progress > 100 {
		return errors.New("progress must be within 0..100")
	}
	if t.QueuePosition < 0 {
		return errors.New("queue_position must be non-negative")
	}
	if t.QueueWaitMS < 0 {
		return errors.New("queue_wait_ms must be non-negative")
	}
	if t.Status != TaskStatusQueued && t.QueuePosition > 0 {
		return errors.New("queue_position is only valid for queued status")
	}
	if t.RetryCount < 0 {
		return errors.New("retry_count must be non-negative")
	}
	if t.MaxRetries < 0 {
		return errors.New("max_retries must be non-negative")
	}
	if t.TimeoutMS < 0 {
		return errors.New("timeout_ms must be non-negative")
	}
	if t.CreatedAt.IsZero() {
		return errors.New("created_at is required")
	}
	if t.UpdatedAt.IsZero() {
		return errors.New("updated_at is required")
	}
	if strings.TrimSpace(t.RequestContent) == "" {
		return errors.New("request_content is required")
	}

	for _, item := range t.Logs {
		if item.Seq < 0 {
			return errors.New("task log seq must be non-negative")
		}
		if item.CreatedAt.IsZero() && item.Timestamp.IsZero() {
			return errors.New("task log created_at is required")
		}
		if !item.Level.IsValid() {
			return errors.New("task log level is invalid")
		}
		if strings.TrimSpace(item.Message) == "" {
			return errors.New("task log message is required")
		}
	}
	for _, item := range t.Artifacts {
		if strings.TrimSpace(item.ArtifactID) == "" && strings.TrimSpace(item.URI) == "" {
			return errors.New("artifact_id is required")
		}
		if strings.TrimSpace(item.Name) == "" {
			return errors.New("artifact name is required")
		}
		if strings.TrimSpace(item.ContentType) == "" {
			return errors.New("artifact content_type is required")
		}
		if item.Size < 0 {
			return errors.New("artifact size must be non-negative")
		}
		if item.CreatedAt.IsZero() {
			return errors.New("artifact created_at is required")
		}
	}
	if !t.TaskSummary.IsZero() {
		if err := t.TaskSummary.Validate(); err != nil {
			return fmt.Errorf("task summary invalid: %w", err)
		}
		if strings.TrimSpace(t.TaskSummary.TaskID) != strings.TrimSpace(t.ID) {
			return errors.New("task_summary.task_id must match task id")
		}
	}
	if strings.TrimSpace(t.MessageLink.TaskID) != "" && strings.TrimSpace(t.MessageLink.TaskID) != strings.TrimSpace(t.ID) {
		return errors.New("message_link.task_id must match task id")
	}
	if strings.TrimSpace(t.MessageLink.SessionID) != "" && strings.TrimSpace(t.MessageLink.SessionID) != strings.TrimSpace(t.SessionID) {
		return errors.New("message_link.session_id must match session id")
	}
	return nil
}

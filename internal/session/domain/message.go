package domain

import (
	"errors"
	"strings"
	"time"

	shareddomain "alter0/internal/shared/domain"
)

type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
)

type RouteResult struct {
	Route     shareddomain.Route `json:"route,omitempty"`
	ErrorCode string             `json:"error_code,omitempty"`
	TaskID    string             `json:"task_id,omitempty"`
}

type MessageRecord struct {
	MessageID   string      `json:"message_id"`
	SessionID   string      `json:"session_id"`
	Role        MessageRole `json:"role"`
	Content     string      `json:"content"`
	Timestamp   time.Time   `json:"timestamp"`
	RouteResult RouteResult `json:"route_result,omitempty"`
}

type SessionSummary struct {
	SessionID     string    `json:"session_id"`
	MessageCount  int       `json:"message_count"`
	StartedAt     time.Time `json:"started_at"`
	LastMessageAt time.Time `json:"last_message_at"`
	LastMessageID string    `json:"last_message_id"`
	LastRoute     string    `json:"last_route,omitempty"`
	LastErrorCode string    `json:"last_error_code,omitempty"`
	LastPreview   string    `json:"last_preview"`
}

func (r MessageRole) IsValid() bool {
	return r == MessageRoleUser || r == MessageRoleAssistant
}

func (r MessageRecord) Validate() error {
	if strings.TrimSpace(r.MessageID) == "" {
		return errors.New("message_id is required")
	}
	if strings.TrimSpace(r.SessionID) == "" {
		return errors.New("session_id is required")
	}
	if !r.Role.IsValid() {
		return errors.New("role must be user or assistant")
	}
	if strings.TrimSpace(r.Content) == "" {
		return errors.New("content is required")
	}
	if r.Timestamp.IsZero() {
		return errors.New("timestamp is required")
	}
	return nil
}

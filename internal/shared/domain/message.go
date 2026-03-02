package domain

import (
	"errors"
	"strings"
	"time"
)

type Source string

const (
	SourceCLI Source = "cli"
	SourceWeb Source = "web"
)

type UnifiedMessage struct {
	MessageID  string            `json:"message_id"`
	SessionID  string            `json:"session_id"`
	UserID     string            `json:"user_id,omitempty"`
	Source     Source            `json:"source"`
	Content    string            `json:"content"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	TraceID    string            `json:"trace_id"`
	ReceivedAt time.Time         `json:"received_at"`
}

func (m UnifiedMessage) Validate() error {
	if strings.TrimSpace(m.MessageID) == "" {
		return errors.New("message_id is required")
	}
	if strings.TrimSpace(m.SessionID) == "" {
		return errors.New("session_id is required")
	}
	if strings.TrimSpace(m.Content) == "" {
		return errors.New("content is required")
	}
	if m.Source != SourceCLI && m.Source != SourceWeb {
		return errors.New("source must be cli or web")
	}
	if m.ReceivedAt.IsZero() {
		return errors.New("received_at is required")
	}
	return nil
}

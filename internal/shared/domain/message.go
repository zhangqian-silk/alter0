package domain

import (
	"errors"
	"strings"
	"time"
)

type ChannelType string

const (
	ChannelTypeCLI       ChannelType = "cli"
	ChannelTypeWeb       ChannelType = "web"
	ChannelTypeScheduler ChannelType = "scheduler"
)

type TriggerType string

const (
	TriggerTypeUser   TriggerType = "user"
	TriggerTypeCron   TriggerType = "cron"
	TriggerTypeSystem TriggerType = "system"
)

type UnifiedMessage struct {
	MessageID     string            `json:"message_id"`
	SessionID     string            `json:"session_id"`
	UserID        string            `json:"user_id,omitempty"`
	ChannelID     string            `json:"channel_id"`
	ChannelType   ChannelType       `json:"channel_type"`
	TriggerType   TriggerType       `json:"trigger_type"`
	Content       string            `json:"content"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	TraceID       string            `json:"trace_id"`
	CorrelationID string            `json:"correlation_id,omitempty"`
	ReceivedAt    time.Time         `json:"received_at"`
}

func (m UnifiedMessage) Validate() error {
	if strings.TrimSpace(m.MessageID) == "" {
		return errors.New("message_id is required")
	}
	if strings.TrimSpace(m.SessionID) == "" {
		return errors.New("session_id is required")
	}
	if strings.TrimSpace(m.ChannelID) == "" {
		return errors.New("channel_id is required")
	}
	if strings.TrimSpace(string(m.ChannelType)) == "" {
		return errors.New("channel_type is required")
	}
	switch m.TriggerType {
	case TriggerTypeUser, TriggerTypeCron, TriggerTypeSystem:
	default:
		return errors.New("trigger_type must be user, cron, or system")
	}
	if strings.TrimSpace(m.Content) == "" {
		return errors.New("content is required")
	}
	if strings.TrimSpace(m.TraceID) == "" {
		return errors.New("trace_id is required")
	}
	if m.ReceivedAt.IsZero() {
		return errors.New("received_at is required")
	}
	return nil
}

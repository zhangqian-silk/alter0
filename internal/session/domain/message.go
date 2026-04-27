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
	Route        shareddomain.Route         `json:"route,omitempty"`
	ErrorCode    string                     `json:"error_code,omitempty"`
	TaskID       string                     `json:"task_id,omitempty"`
	ProcessSteps []shareddomain.ProcessStep `json:"process_steps,omitempty"`
}

type MessageSource struct {
	TriggerType   shareddomain.TriggerType `json:"trigger_type,omitempty"`
	ChannelType   shareddomain.ChannelType `json:"channel_type,omitempty"`
	ChannelID     string                   `json:"channel_id,omitempty"`
	CorrelationID string                   `json:"correlation_id,omitempty"`
	AgentID       string                   `json:"agent_id,omitempty"`
	AgentName     string                   `json:"agent_name,omitempty"`
	JobID         string                   `json:"job_id,omitempty"`
	JobName       string                   `json:"job_name,omitempty"`
	FiredAt       time.Time                `json:"fired_at,omitempty"`
}

type MessageRecord struct {
	MessageID   string            `json:"message_id"`
	SessionID   string            `json:"session_id"`
	Role        MessageRole       `json:"role"`
	Content     string            `json:"content"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
	RouteResult RouteResult       `json:"route_result,omitempty"`
	Source      MessageSource     `json:"source,omitempty"`
}

type SessionSummary struct {
	SessionID     string                   `json:"session_id"`
	MessageCount  int                      `json:"message_count"`
	StartedAt     time.Time                `json:"started_at"`
	LastMessageAt time.Time                `json:"last_message_at"`
	LastMessageID string                   `json:"last_message_id"`
	LastRoute     string                   `json:"last_route,omitempty"`
	LastErrorCode string                   `json:"last_error_code,omitempty"`
	LastPreview   string                   `json:"last_preview"`
	TriggerType   shareddomain.TriggerType `json:"trigger_type,omitempty"`
	ChannelType   shareddomain.ChannelType `json:"channel_type,omitempty"`
	ChannelID     string                   `json:"channel_id,omitempty"`
	CorrelationID string                   `json:"correlation_id,omitempty"`
	AgentID       string                   `json:"agent_id,omitempty"`
	AgentName     string                   `json:"agent_name,omitempty"`
	JobID         string                   `json:"job_id,omitempty"`
	JobName       string                   `json:"job_name,omitempty"`
	FiredAt       time.Time                `json:"fired_at,omitempty"`
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
	if err := r.Source.Validate(); err != nil {
		return err
	}
	return nil
}

func (s MessageSource) Validate() error {
	trigger := strings.TrimSpace(string(s.TriggerType))
	switch trigger {
	case "":
	case string(shareddomain.TriggerTypeUser), string(shareddomain.TriggerTypeCron), string(shareddomain.TriggerTypeSystem):
	default:
		return errors.New("source.trigger_type must be user/cron/system")
	}

	channel := strings.TrimSpace(string(s.ChannelType))
	switch channel {
	case "":
	case string(shareddomain.ChannelTypeCLI), string(shareddomain.ChannelTypeWeb), string(shareddomain.ChannelTypeScheduler):
	default:
		return errors.New("source.channel_type must be cli/web/scheduler")
	}
	return nil
}

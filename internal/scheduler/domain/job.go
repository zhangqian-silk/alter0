package domain

import (
	"errors"
	"strings"
	"time"
)

type Job struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Interval  time.Duration     `json:"-"`
	Enabled   bool              `json:"enabled"`
	SessionID string            `json:"session_id"`
	UserID    string            `json:"user_id,omitempty"`
	ChannelID string            `json:"channel_id,omitempty"`
	Content   string            `json:"content"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

func (j Job) Validate() error {
	if strings.TrimSpace(j.ID) == "" {
		return errors.New("job id is required")
	}
	if j.Interval <= 0 {
		return errors.New("interval must be greater than zero")
	}
	if strings.TrimSpace(j.SessionID) == "" {
		return errors.New("session_id is required")
	}
	if strings.TrimSpace(j.Content) == "" {
		return errors.New("content is required")
	}
	return nil
}

package domain

import (
	"errors"
	"strings"

	shareddomain "alter0/internal/shared/domain"
)

type Channel struct {
	ID          string                   `json:"id"`
	Type        shareddomain.ChannelType `json:"type"`
	Enabled     bool                     `json:"enabled"`
	Description string                   `json:"description,omitempty"`
	Metadata    map[string]string        `json:"metadata,omitempty"`
}

func (c Channel) Validate() error {
	if strings.TrimSpace(c.ID) == "" {
		return errors.New("channel id is required")
	}
	if strings.TrimSpace(string(c.Type)) == "" {
		return errors.New("channel type is required")
	}
	return nil
}

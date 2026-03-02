package domain

import (
	"errors"
	"strings"
)

type Skill struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Enabled     bool              `json:"enabled"`
	Description string            `json:"description,omitempty"`
	Version     string            `json:"version,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

func (s Skill) Validate() error {
	if strings.TrimSpace(s.ID) == "" {
		return errors.New("skill id is required")
	}
	if strings.TrimSpace(s.Name) == "" {
		return errors.New("skill name is required")
	}
	return nil
}

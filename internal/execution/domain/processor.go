package domain

import (
	"context"

	shareddomain "alter0/internal/shared/domain"
)

type NLProcessor interface {
	Process(ctx context.Context, content string, metadata map[string]string) (string, error)
}

type StreamEventType string

const (
	StreamEventTypeOutput  StreamEventType = "output"
	StreamEventTypeProcess StreamEventType = "process"
)

type StreamEvent struct {
	Type        StreamEventType           `json:"type"`
	Text        string                    `json:"text"`
	ProcessStep *shareddomain.ProcessStep `json:"process_step,omitempty"`
}

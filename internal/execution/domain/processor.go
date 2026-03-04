package domain

import "context"

type NLProcessor interface {
	Process(ctx context.Context, content string, metadata map[string]string) (string, error)
}

type StreamEventType string

const (
	StreamEventTypeOutput StreamEventType = "output"
)

type StreamEvent struct {
	Type StreamEventType `json:"type"`
	Text string          `json:"text"`
}

package gateway

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type TraceEvent struct {
	Timestamp string `json:"timestamp"`
	RequestID string `json:"request_id,omitempty"`
	MessageID string `json:"message_id,omitempty"`
	ChannelID string `json:"channel_id,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	AgentID   string `json:"agent_id,omitempty"`
	Event     string `json:"event"`
	Status    string `json:"status"`
	Detail    string `json:"detail,omitempty"`
}

type TraceRecorder interface {
	Record(TraceEvent) error
}

type JSONLTraceRecorder struct {
	basePath string
	mu       sync.Mutex
}

func NewTraceRecorder(basePath string) (*JSONLTraceRecorder, error) {
	path := strings.TrimSpace(basePath)
	if path == "" {
		return nil, fmt.Errorf("trace base path is required")
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, err
	}
	return &JSONLTraceRecorder{basePath: path}, nil
}

func (r *JSONLTraceRecorder) Record(event TraceEvent) error {
	if r == nil {
		return nil
	}
	ts := time.Now().UTC()
	if strings.TrimSpace(event.Timestamp) == "" {
		event.Timestamp = ts.Format(time.RFC3339Nano)
	}
	if strings.TrimSpace(event.Status) == "" {
		event.Status = "ok"
	}
	if strings.TrimSpace(event.Event) == "" {
		event.Event = "unknown"
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	dayDir := filepath.Join(r.basePath, ts.Format("2006-01-02"))
	if err := os.MkdirAll(dayDir, 0755); err != nil {
		return err
	}
	path := filepath.Join(dayDir, "gateway_events.jsonl")

	r.mu.Lock()
	defer r.mu.Unlock()

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(payload, '\n'))
	return err
}

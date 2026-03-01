package execlog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	coreexecutor "alter0/app/core/executor"
)

const (
	StageRouter = "router"
	StageGen    = "gen"
	StageCloser = "closer"
)

type Meta struct {
	SessionID string
	Stage     string
	TaskID    string
	TaskHint  string
	UserID    string
	ChannelID string
}

type metaKey struct{}

type entry struct {
	Timestamp     string `json:"timestamp"`
	SessionID     string `json:"session_id"`
	Stage         string `json:"stage"`
	Executor      string `json:"executor"`
	TaskID        string `json:"task_id,omitempty"`
	UserID        string `json:"user_id,omitempty"`
	ChannelID     string `json:"channel_id,omitempty"`
	Status        string `json:"status"`
	DurationMs    int64  `json:"duration_ms"`
	PromptChars   int    `json:"prompt_chars"`
	OutputChars   int    `json:"output_chars,omitempty"`
	PromptPreview string `json:"prompt_preview,omitempty"`
	OutputPreview string `json:"output_preview,omitempty"`
	Error         string `json:"error,omitempty"`
}

var mu sync.Mutex

func WithMeta(ctx context.Context, meta Meta) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	current := GetMeta(ctx)
	merged := mergeMeta(current, meta)
	return context.WithValue(ctx, metaKey{}, merged)
}

func GetMeta(ctx context.Context) Meta {
	if ctx == nil {
		return Meta{}
	}
	meta, _ := ctx.Value(metaKey{}).(Meta)
	return meta
}

func RunExecutorWithTimeout(ctx context.Context, executor string, prompt string, timeout time.Duration, stage string) (string, error) {
	start := time.Now()
	meta := GetMeta(ctx)
	if strings.TrimSpace(stage) != "" {
		meta.Stage = stage
	}
	if strings.TrimSpace(meta.Stage) == "" {
		meta.Stage = "unknown"
	}

	execCtx := coreexecutor.WithExecTrace(ctx, coreexecutor.ExecTrace{
		SessionID: meta.SessionID,
		Stage:     meta.Stage,
		TaskID:    meta.TaskID,
		TaskHint:  meta.TaskHint,
		UserID:    meta.UserID,
		ChannelID: meta.ChannelID,
	})

	output, err := coreexecutor.RunExecutorWithTimeout(execCtx, executor, prompt, timeout)
	_ = appendHourlyLog(start, strings.TrimSpace(executor), meta, prompt, output, err, time.Since(start))
	return output, err
}

func mergeMeta(base Meta, override Meta) Meta {
	out := base
	if strings.TrimSpace(override.SessionID) != "" {
		out.SessionID = strings.TrimSpace(override.SessionID)
	}
	if strings.TrimSpace(override.Stage) != "" {
		out.Stage = strings.TrimSpace(override.Stage)
	}
	if strings.TrimSpace(override.TaskID) != "" {
		out.TaskID = strings.TrimSpace(override.TaskID)
	}
	if strings.TrimSpace(override.TaskHint) != "" {
		out.TaskHint = strings.TrimSpace(override.TaskHint)
	}
	if strings.TrimSpace(override.UserID) != "" {
		out.UserID = strings.TrimSpace(override.UserID)
	}
	if strings.TrimSpace(override.ChannelID) != "" {
		out.ChannelID = strings.TrimSpace(override.ChannelID)
	}
	return out
}

func appendHourlyLog(ts time.Time, executor string, meta Meta, prompt string, output string, runErr error, duration time.Duration) error {
	if executor == "" {
		executor = "unknown"
	}
	sessionID := strings.TrimSpace(meta.SessionID)
	if sessionID == "" {
		sessionID = "unknown"
	}
	stage := strings.TrimSpace(meta.Stage)
	if stage == "" {
		stage = "unknown"
	}

	record := entry{
		Timestamp:     ts.Format(time.RFC3339Nano),
		SessionID:     sessionID,
		Stage:         stage,
		Executor:      executor,
		TaskID:        strings.TrimSpace(meta.TaskID),
		UserID:        strings.TrimSpace(meta.UserID),
		ChannelID:     strings.TrimSpace(meta.ChannelID),
		Status:        "ok",
		DurationMs:    duration.Milliseconds(),
		PromptChars:   len(prompt),
		OutputChars:   len(output),
		PromptPreview: previewText(prompt, 240),
		OutputPreview: previewText(output, 240),
	}
	if runErr != nil {
		record.Status = "error"
		record.Error = runErr.Error()
	}

	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}

	dayDir := filepath.Join("output", "orchestrator", ts.Format("2006-01-02"))
	if err := os.MkdirAll(dayDir, 0755); err != nil {
		return fmt.Errorf("failed to create orchestrator log dir: %w", err)
	}
	logPath := filepath.Join(dayDir, fmt.Sprintf("executor_%s.jsonl", ts.Format("20060102-15")))

	mu.Lock()
	defer mu.Unlock()

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(append(payload, '\n')); err != nil {
		return err
	}
	return nil
}

func previewText(s string, limit int) string {
	clean := strings.TrimSpace(s)
	if clean == "" || limit <= 0 {
		return ""
	}
	clean = strings.ReplaceAll(clean, "\r\n", "\n")
	clean = strings.ReplaceAll(clean, "\n", "\\n")
	runes := []rune(clean)
	if len(runes) <= limit {
		return clean
	}
	return string(runes[:limit]) + "..."
}

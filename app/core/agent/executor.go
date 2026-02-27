package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"alter0/app/pkg/cmdutil"
)

const DefaultCLITimeout = 120 * time.Second

const (
	ExecStageRouter = "router"
	ExecStageGen    = "gen"
	ExecStageCloser = "closer"
)

type ExecTrace struct {
	SessionID string
	Stage     string
	TaskID    string
	TaskHint  string
	UserID    string
	ChannelID string
}

type execTraceKey struct{}

var codexExecMu sync.Mutex

func WithExecTrace(ctx context.Context, trace ExecTrace) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, execTraceKey{}, trace)
}

func getExecTrace(ctx context.Context) ExecTrace {
	if ctx == nil {
		return ExecTrace{}
	}
	trace, _ := ctx.Value(execTraceKey{}).(ExecTrace)
	return trace
}

func NormalizeExecutorName(name string) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, " ", "_")
	switch normalized {
	case "claude", "claude_code":
		return "claude_code"
	case "codex":
		return "codex"
	default:
		return ""
	}
}

func ResolveExecutorCommand(name string) (string, []string, error) {
	switch NormalizeExecutorName(name) {
	case "claude_code":
		if err := cmdutil.RequireExecutable("claude"); err != nil {
			return "", nil, fmt.Errorf("executor not installed: claude_code (missing executable: claude)")
		}
		return "claude", nil, nil
	case "codex":
		if err := cmdutil.RequireExecutable("codex"); err != nil {
			return "", nil, fmt.Errorf("executor not installed: codex (missing executable: codex)")
		}
		// Use non-interactive mode so stdin prompts work in CLI pipelines.
		return "codex", []string{"exec"}, nil
	default:
		return "", nil, fmt.Errorf("unknown executor: %s", name)
	}
}

func EnsureExecutorInstalled(ctx context.Context, executorName string) (string, error) {
	_ = ctx
	normalized := NormalizeExecutorName(executorName)
	if normalized == "" {
		return "", fmt.Errorf("unknown executor: %s", executorName)
	}
	switch normalized {
	case "codex":
		if err := cmdutil.RequireExecutable("codex"); err != nil {
			return "", fmt.Errorf("executor not installed: codex (missing executable: codex)")
		}
	case "claude_code":
		if err := cmdutil.RequireExecutable("claude"); err != nil {
			return "", fmt.Errorf("executor not installed: claude_code (missing executable: claude)")
		}
	default:
		return "", fmt.Errorf("unknown executor: %s", executorName)
	}
	return "", nil
}

func RunExecutor(ctx context.Context, executorName string, prompt string) (string, error) {
	return RunExecutorWithTimeout(ctx, executorName, prompt, DefaultCLITimeout)
}

func RunExecutorWithTimeout(ctx context.Context, executorName string, prompt string, timeout time.Duration) (string, error) {
	normalized := strings.TrimSpace(executorName)
	if normalized == "" {
		return "", fmt.Errorf("Executor 未配置，请使用 /executor 选择")
	}
	if NormalizeExecutorName(normalized) == "codex" {
		return runCodexWithTimeout(ctx, prompt, timeout)
	}
	cmdName, cmdArgs, err := ResolveExecutorCommand(normalized)
	if err != nil {
		return "", err
	}
	output, err := cmdutil.RunWithInput(ctx, cmdName, cmdArgs, prompt, timeout)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(output) == "" {
		return "", fmt.Errorf("empty response")
	}
	return output, nil
}

func runCodexWithTimeout(ctx context.Context, prompt string, timeout time.Duration) (string, error) {
	if err := cmdutil.RequireExecutable("codex"); err != nil {
		return "", fmt.Errorf("executor not installed: codex (missing executable: codex)")
	}

	trace := getExecTrace(ctx)
	tmpDir := filepath.Join("output", "tmp", "codex")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create codex temp dir: %w", err)
	}

	stageTag := shortStageTag(trace.Stage)
	sessionTag := shortIDTag(trace.SessionID, "nosid", 8)
	pattern := fmt.Sprintf("cx_%s_%s_*.tmp", stageTag, sessionTag)

	tmpFile, err := os.CreateTemp(tmpDir, pattern)
	if err != nil {
		return "", fmt.Errorf("failed to create codex temp output file: %w", err)
	}
	outputPath := tmpFile.Name()
	if closeErr := tmpFile.Close(); closeErr != nil {
		return "", fmt.Errorf("failed to close codex temp output file: %w", closeErr)
	}
	defer func() { _ = os.Remove(outputPath) }()

	cmdCtx := cmdutil.WithCommandLogContext(ctx, cmdutil.CommandLogContext{
		SessionID: trace.SessionID,
		Stage:     trace.Stage,
	})
	codexExecMu.Lock()
	_, err = cmdutil.RunWithInput(cmdCtx, "codex", []string{"exec", "-o", outputPath}, prompt, timeout)
	if err != nil {
		codexExecMu.Unlock()
		return "", err
	}

	data, err := os.ReadFile(outputPath)
	codexExecMu.Unlock()
	if err != nil {
		return "", fmt.Errorf("failed to read codex output file: %w", err)
	}
	output := strings.TrimSpace(string(data))
	if output == "" {
		return "", fmt.Errorf("empty response")
	}
	return output, nil
}

func shortStageTag(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case ExecStageRouter:
		return "r"
	case ExecStageGen:
		return "g"
	case ExecStageCloser:
		return "c"
	default:
		return "u"
	}
}

func safeFileTag(raw string, fallback string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return fallback
	}
	s = strings.ReplaceAll(s, " ", "_")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return fallback
	}
	return b.String()
}

func shortIDTag(raw string, fallback string, maxLen int) string {
	tag := safeFileTag(raw, "")
	if tag == "" {
		return fallback
	}
	if maxLen <= 0 {
		return tag
	}
	runes := []rune(tag)
	if len(runes) <= maxLen {
		return tag
	}
	return string(runes[len(runes)-maxLen:])
}

package infrastructure

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	execdomain "alter0/internal/execution/domain"
)

const (
	defaultCodexCommand              = "codex"
	defaultCodexSandboxMode          = "danger-full-access"
	codexSandboxEnvKey               = "ALTER0_CODEX_SANDBOX"
	codexSandboxMetadataKey          = "codex_sandbox"
	codexWorkspaceModeEnvKey         = "ALTER0_CODEX_WORKSPACE_MODE"
	codexWorkspaceModeMetadataKey    = "codex_workspace_mode"
	codexWorkspaceRootDirEnvKey      = "ALTER0_CODEX_WORKSPACE_ROOT"
	codexWorkspaceRootDirMetadataKey = "codex_workspace_root"
	codexWorkspaceModeSession        = "session"
	codexWorkspaceModeRepoRoot       = "repo-root"
	defaultWorkspaceRootDir          = ".alter0"
	workspaceDirectoryName           = "workspaces"
	workspaceSessionsDirName         = "sessions"
	workspaceTasksDirName            = "tasks"
	taskIDMetadataKey                = "task_id"
	sessionIDMetadataFallback        = "session_id"
)

type commandRunner func(ctx context.Context, name string, args ...string) *exec.Cmd

type CodexCLIProcessor struct {
	command string
	runner  commandRunner
}

type codexExecutionPayload struct {
	Protocol    string                   `json:"protocol"`
	UserPrompt  string                   `json:"user_prompt"`
	SkillPolicy *execdomain.SkillContext `json:"skill_context,omitempty"`
	MCPPolicy   *execdomain.MCPContext   `json:"mcp_context,omitempty"`
}

type codexJSONEvent struct {
	Type string          `json:"type"`
	Item *codexEventItem `json:"item,omitempty"`
}

type codexEventItem struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	Delta string `json:"delta,omitempty"`
}

func NewCodexCLIProcessor() *CodexCLIProcessor {
	return &CodexCLIProcessor{
		command: defaultCodexCommand,
		runner:  exec.CommandContext,
	}
}

func (p *CodexCLIProcessor) Process(ctx context.Context, content string, metadata map[string]string) (string, error) {
	prompt := strings.TrimSpace(content)
	if prompt == "" {
		return "", errors.New("content is required")
	}
	renderedPrompt, err := buildCodexPrompt(prompt, metadata)
	if err != nil {
		return "", err
	}
	workspaceDir, err := resolveCodexWorkspace(metadata)
	if err != nil {
		return "", err
	}

	outputFile, err := os.CreateTemp("", "alter0-codex-output-*.txt")
	if err != nil {
		return "", fmt.Errorf("create codex output file: %w", err)
	}
	outputPath := outputFile.Name()
	if err := outputFile.Close(); err != nil {
		_ = os.Remove(outputPath)
		return "", fmt.Errorf("prepare codex output file: %w", err)
	}
	defer func() {
		_ = os.Remove(outputPath)
	}()

	commandName := strings.TrimSpace(p.command)
	if commandName == "" {
		commandName = defaultCodexCommand
	}
	runner := p.runner
	if runner == nil {
		runner = exec.CommandContext
	}

	args := []string{
		"exec",
		"--color", "never",
		"--skip-git-repo-check",
		"--sandbox", resolveCodexSandboxMode(metadata),
		"-o", outputPath,
		renderedPrompt,
	}
	cmd := runner(ctx, commandName, args...)
	if workspaceDir != "" {
		cmd.Dir = workspaceDir
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		details := strings.TrimSpace(stderr.String())
		if details == "" {
			details = strings.TrimSpace(stdout.String())
		}
		if details == "" {
			return "", fmt.Errorf("codex command failed: %w", err)
		}
		return "", fmt.Errorf("codex command failed: %w: %s", err, details)
	}

	rawOutput, err := os.ReadFile(outputPath)
	if err != nil {
		return "", fmt.Errorf("read codex output file: %w", err)
	}
	result := strings.TrimSpace(string(rawOutput))
	if result == "" {
		return "", errors.New("codex returned empty output")
	}
	return result, nil
}

func (p *CodexCLIProcessor) ProcessStream(
	ctx context.Context,
	content string,
	metadata map[string]string,
	emit func(event execdomain.StreamEvent) error,
) (string, error) {
	prompt := strings.TrimSpace(content)
	if prompt == "" {
		return "", errors.New("content is required")
	}
	renderedPrompt, err := buildCodexPrompt(prompt, metadata)
	if err != nil {
		return "", err
	}
	workspaceDir, err := resolveCodexWorkspace(metadata)
	if err != nil {
		return "", err
	}

	commandName := strings.TrimSpace(p.command)
	if commandName == "" {
		commandName = defaultCodexCommand
	}
	runner := p.runner
	if runner == nil {
		runner = exec.CommandContext
	}

	args := []string{
		"exec",
		"--color", "never",
		"--skip-git-repo-check",
		"--sandbox", resolveCodexSandboxMode(metadata),
		"--json",
		"--progress-cursor",
		renderedPrompt,
	}
	cmd := runner(ctx, commandName, args...)
	if workspaceDir != "" {
		cmd.Dir = workspaceDir
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("create codex stdout pipe: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		details := strings.TrimSpace(stderr.String())
		if details == "" {
			return "", fmt.Errorf("codex command failed: %w", err)
		}
		return "", fmt.Errorf("codex command failed: %w: %s", err, details)
	}

	output, scanErr := collectStreamOutput(stdoutPipe, emit)
	waitErr := cmd.Wait()
	if scanErr != nil {
		return "", scanErr
	}
	if waitErr != nil {
		details := strings.TrimSpace(stderr.String())
		if details == "" {
			return "", fmt.Errorf("codex command failed: %w", waitErr)
		}
		return "", fmt.Errorf("codex command failed: %w: %s", waitErr, details)
	}
	result := strings.TrimSpace(output)
	if result == "" {
		return "", errors.New("codex returned empty output")
	}
	return result, nil
}

func collectStreamOutput(
	reader io.Reader,
	emit func(event execdomain.StreamEvent) error,
) (string, error) {
	scanner := bufio.NewScanner(reader)
	// Allow larger JSONL events for long model outputs.
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	emittedOutput := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		event := codexJSONEvent{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		if event.Item == nil || event.Item.Type != "agent_message" {
			continue
		}

		text := event.Item.Text
		delta := event.Item.Delta

		switch event.Type {
		case "item.delta":
			if strings.TrimSpace(delta) == "" {
				continue
			}
			if err := emitStreamDelta(emit, delta); err != nil {
				return "", err
			}
			emittedOutput += delta
		case "item.updated", "item.completed":
			if strings.TrimSpace(text) == "" {
				continue
			}
			nextDelta := resolveStreamDelta(emittedOutput, text)
			if nextDelta == "" {
				continue
			}
			if err := emitStreamDelta(emit, nextDelta); err != nil {
				return "", err
			}
			emittedOutput += nextDelta
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read codex stream output: %w", err)
	}
	return emittedOutput, nil
}

func emitStreamDelta(emit func(event execdomain.StreamEvent) error, delta string) error {
	if emit == nil {
		return nil
	}
	if strings.TrimSpace(delta) == "" {
		return nil
	}
	return emit(execdomain.StreamEvent{
		Type: execdomain.StreamEventTypeOutput,
		Text: delta,
	})
}

func resolveStreamDelta(previous string, next string) string {
	if previous == "" {
		return next
	}
	if strings.HasPrefix(next, previous) {
		return next[len(previous):]
	}
	return next
}

func buildCodexPrompt(prompt string, metadata map[string]string) (string, error) {
	rawSkillContext := strings.TrimSpace(metadataValue(metadata, execdomain.SkillContextMetadataKey))
	rawMCPContext := strings.TrimSpace(metadataValue(metadata, execdomain.MCPContextMetadataKey))

	var skillContext *execdomain.SkillContext
	if rawSkillContext != "" {
		parsedSkillContext := execdomain.SkillContext{}
		if err := json.Unmarshal([]byte(rawSkillContext), &parsedSkillContext); err != nil {
			return "", fmt.Errorf("invalid skill context metadata: %w", err)
		}
		if len(parsedSkillContext.Skills) > 0 {
			skillContext = &parsedSkillContext
		}
	}

	var mcpContext *execdomain.MCPContext
	if rawMCPContext != "" {
		parsedMCPContext := execdomain.MCPContext{}
		if err := json.Unmarshal([]byte(rawMCPContext), &parsedMCPContext); err != nil {
			return "", fmt.Errorf("invalid mcp context metadata: %w", err)
		}
		if len(parsedMCPContext.Servers) > 0 {
			mcpContext = &parsedMCPContext
		}
	}
	if skillContext == nil && mcpContext == nil {
		return prompt, nil
	}

	payload := codexExecutionPayload{
		Protocol:    "alter0.codex-exec/v1",
		UserPrompt:  prompt,
		SkillPolicy: skillContext,
		MCPPolicy:   mcpContext,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal codex prompt payload: %w", err)
	}
	return string(encoded), nil
}

func metadataValue(metadata map[string]string, key string) string {
	if len(metadata) == 0 {
		return ""
	}
	return metadata[key]
}

func resolveCodexSandboxMode(metadata map[string]string) string {
	candidate := strings.TrimSpace(firstNonEmpty(
		metadataValue(metadata, codexSandboxMetadataKey),
		os.Getenv(codexSandboxEnvKey),
		defaultCodexSandboxMode,
	))
	switch candidate {
	case "read-only", "workspace-write", "danger-full-access":
		return candidate
	default:
		return defaultCodexSandboxMode
	}
}

func resolveCodexWorkspaceMode(metadata map[string]string) string {
	candidate := strings.ToLower(strings.TrimSpace(firstNonEmpty(
		metadataValue(metadata, codexWorkspaceModeMetadataKey),
		os.Getenv(codexWorkspaceModeEnvKey),
		codexWorkspaceModeSession,
	)))
	switch candidate {
	case codexWorkspaceModeSession, codexWorkspaceModeRepoRoot:
		return candidate
	default:
		return codexWorkspaceModeSession
	}
}

func resolveCodexWorkspace(metadata map[string]string) (string, error) {
	workspaceMode := resolveCodexWorkspaceMode(metadata)
	if workspaceMode == codexWorkspaceModeRepoRoot {
		repoRoot := strings.TrimSpace(firstNonEmpty(
			metadataValue(metadata, codexWorkspaceRootDirMetadataKey),
			os.Getenv(codexWorkspaceRootDirEnvKey),
			".",
		))
		absolute, err := filepath.Abs(repoRoot)
		if err != nil {
			return "", fmt.Errorf("resolve codex workspace path: %w", err)
		}
		info, statErr := os.Stat(absolute)
		if statErr != nil {
			return "", fmt.Errorf("resolve codex workspace path: %w", statErr)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("resolve codex workspace path: %s is not a directory", absolute)
		}
		return absolute, nil
	}

	sessionID := sanitizeWorkspaceSegment(firstNonEmpty(
		metadataValue(metadata, execdomain.RuntimeSessionIDMetadataKey),
		metadataValue(metadata, sessionIDMetadataFallback),
	))
	if sessionID == "" {
		return "", errors.New("workspace session context is required")
	}

	parts := []string{
		defaultWorkspaceRootDir,
		workspaceDirectoryName,
		workspaceSessionsDirName,
		sessionID,
	}
	taskID := sanitizeWorkspaceSegment(metadataValue(metadata, taskIDMetadataKey))
	if taskID != "" {
		parts = append(parts, workspaceTasksDirName, taskID)
	}
	workspaceDir := filepath.Join(parts...)
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		return "", fmt.Errorf("prepare codex workspace: %w", err)
	}
	absolute, err := filepath.Abs(workspaceDir)
	if err != nil {
		return "", fmt.Errorf("resolve codex workspace path: %w", err)
	}
	return absolute, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		return trimmed
	}
	return ""
}

func sanitizeWorkspaceSegment(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	var builder strings.Builder
	builder.Grow(len(trimmed))
	hyphenPending := false
	for _, ch := range trimmed {
		if (ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '_' || ch == '.' {
			builder.WriteRune(ch)
			hyphenPending = false
			continue
		}
		if hyphenPending {
			continue
		}
		builder.WriteByte('-')
		hyphenPending = true
	}
	sanitized := strings.Trim(builder.String(), "-._")
	if sanitized == "" {
		return ""
	}
	if len(sanitized) > 96 {
		sanitized = sanitized[:96]
	}
	return strings.ToLower(sanitized)
}

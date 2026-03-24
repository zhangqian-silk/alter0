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
	"time"

	execdomain "alter0/internal/execution/domain"
)

const (
	defaultCodexCommand              = "codex"
	defaultCodexSandboxMode          = "danger-full-access"
	defaultCodexWaitDelay            = 1500 * time.Millisecond
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
	codexAddDirsMetadataKey          = "codex_add_dirs"
	codexAddDirsEnvKey               = "ALTER0_CODEX_ADD_DIRS"
	codexAddDirRulesMetadataKey      = "codex_add_dir_rules"
	codexAddDirRulesEnvKey           = "ALTER0_CODEX_ADD_DIR_RULES"
)

type codexAddDirRule struct {
	Path    string `json:"path"`
	Enabled *bool  `json:"enabled,omitempty"`
}

var defaultAddDirRules = []codexAddDirRule{
	{Path: "/bin", Enabled: boolPtr(true)},
	{Path: "/usr/bin", Enabled: boolPtr(true)},
	{Path: "/usr/local/bin", Enabled: boolPtr(true)},
	{Path: "/usr/sbin", Enabled: boolPtr(true)},
	{Path: "/sbin", Enabled: boolPtr(true)},
	{Path: "/tmp", Enabled: boolPtr(true)},
}

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
	Type    string          `json:"type"`
	Message string          `json:"message,omitempty"`
	Item    *codexEventItem `json:"item,omitempty"`
}

type codexEventItem struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	Delta string `json:"delta,omitempty"`
}

type codexPipeResult struct {
	output string
	fatal  string
	err    error
}

type codexStreamResult struct {
	output string
	err    error
}

func NewCodexCLIProcessor() *CodexCLIProcessor {
	return NewCodexCLIProcessorWithCommand(defaultCodexCommand)
}

func NewCodexCLIProcessorWithCommand(command string) *CodexCLIProcessor {
	return &CodexCLIProcessor{
		command: strings.TrimSpace(command),
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
	}
	for _, dir := range resolveCodexAddDirs(metadata) {
		args = append(args, "--add-dir", dir)
	}
	args = append(args, "-o", outputPath, renderedPrompt)
	procCtx, procCancel := context.WithCancel(ctx)
	defer procCancel()

	cmd := runner(procCtx, commandName, args...)
	configureCodexCommand(cmd)
	cmd.WaitDelay = defaultCodexWaitDelay
	if workspaceDir != "" {
		cmd.Dir = workspaceDir
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if fatalMessage := fatalCodexAuthMessage(stderr.String()); fatalMessage != "" {
			return "", fmt.Errorf("codex authentication failed: %s", fatalMessage)
		}
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
	}
	for _, dir := range resolveCodexAddDirs(metadata) {
		args = append(args, "--add-dir", dir)
	}
	args = append(args, "--json", "--progress-cursor", renderedPrompt)
	procCtx, procCancel := context.WithCancel(ctx)
	defer procCancel()

	cmd := runner(procCtx, commandName, args...)
	configureCodexCommand(cmd)
	cmd.WaitDelay = defaultCodexWaitDelay
	if workspaceDir != "" {
		cmd.Dir = workspaceDir
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("create codex stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("create codex stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("codex command failed: %w", err)
	}

	stdoutCh := make(chan codexStreamResult, 1)
	go func() {
		output, err := collectStreamOutput(stdoutPipe, emit)
		stdoutCh <- codexStreamResult{output: output, err: err}
	}()
	stderrFatalCh, stderrCh := watchCodexStderr(stderrPipe)

	stdoutResult := codexStreamResult{}
	fatalErr := error(nil)
	stdoutDone := false
	for !stdoutDone && fatalErr == nil {
		select {
		case stdoutResult = <-stdoutCh:
			stdoutDone = true
			if stdoutResult.err != nil {
				procCancel()
				_ = stderrPipe.Close()
			}
		case fatalMessage, ok := <-stderrFatalCh:
			if !ok {
				stderrFatalCh = nil
				continue
			}
			if strings.TrimSpace(fatalMessage) == "" {
				continue
			}
			fatalErr = fmt.Errorf("codex authentication failed: %s", fatalMessage)
			procCancel()
			_ = stdoutPipe.Close()
			_ = stderrPipe.Close()
			stdoutResult = <-stdoutCh
		}
	}
	waitErr := cmd.Wait()
	stderrResult := <-stderrCh
	if fatalErr != nil {
		return "", fatalErr
	}
	if stderrResult.fatal != "" {
		return "", fmt.Errorf("codex authentication failed: %s", stderrResult.fatal)
	}
	if stderrResult.err != nil {
		return "", fmt.Errorf("read codex stderr: %w", stderrResult.err)
	}
	if stdoutResult.err != nil {
		return "", stdoutResult.err
	}
	if waitErr != nil {
		details := strings.TrimSpace(stderrResult.output)
		if details == "" {
			return "", fmt.Errorf("codex command failed: %w", waitErr)
		}
		return "", fmt.Errorf("codex command failed: %w: %s", waitErr, details)
	}
	result := strings.TrimSpace(stdoutResult.output)
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
		if fatalMessage := fatalCodexAuthMessage(event.Message); fatalMessage != "" {
			return "", fmt.Errorf("codex authentication failed: %s", fatalMessage)
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

func collectCodexPipe(reader io.Reader) <-chan codexPipeResult {
	resultCh := make(chan codexPipeResult, 1)
	go func() {
		data, err := io.ReadAll(reader)
		if isIgnorablePipeReadError(err) {
			err = nil
		}
		resultCh <- codexPipeResult{
			output: strings.TrimSpace(string(data)),
			err:    err,
		}
	}()
	return resultCh
}

func watchCodexStderr(reader io.Reader) (<-chan string, <-chan codexPipeResult) {
	fatalCh := make(chan string, 1)
	resultCh := make(chan codexPipeResult, 1)
	go func() {
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

		lines := make([]string, 0, 8)
		fatal := ""
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			lines = append(lines, line)
			if fatal == "" {
				if message := fatalCodexAuthMessage(line); message != "" {
					fatal = message
					fatalCh <- fatal
				}
			}
		}
		close(fatalCh)
		resultCh <- codexPipeResult{
			output: strings.TrimSpace(strings.Join(lines, "\n")),
			fatal:  fatal,
			err:    normalizePipeReadError(scanner.Err()),
		}
	}()
	return fatalCh, resultCh
}

func normalizePipeReadError(err error) error {
	if isIgnorablePipeReadError(err) {
		return nil
	}
	return err
}

func isIgnorablePipeReadError(err error) bool {
	if err == nil {
		return false
	}
	normalized := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(normalized, "file already closed")
}

func fatalCodexAuthMessage(message string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return ""
	}
	normalized := strings.ToLower(trimmed)
	switch {
	case strings.Contains(normalized, "401 unauthorized"):
		return trimmed
	case strings.Contains(normalized, "403 forbidden"):
		return trimmed
	case strings.Contains(normalized, "missing bearer"):
		return trimmed
	case strings.Contains(normalized, "missing basic authentication"):
		return trimmed
	case strings.Contains(normalized, "missing bearer or basic authentication"):
		return trimmed
	case strings.Contains(normalized, "invalid api key"):
		return trimmed
	case strings.Contains(normalized, "incorrect api key"):
		return trimmed
	case strings.Contains(normalized, "failed to refresh token"):
		return trimmed
	case strings.Contains(normalized, "refresh_token_reused"):
		return trimmed
	case strings.Contains(normalized, "provided authentication token is expired"):
		return trimmed
	case strings.Contains(normalized, "please try signing in again"):
		return trimmed
	case strings.Contains(normalized, "please log out and sign in again"):
		return trimmed
	default:
		return ""
	}
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

func resolveCodexAddDirs(metadata map[string]string) []string {
	enabledByPath := make(map[string]bool)
	order := make([]string, 0)

	addRule := func(path string, enabled bool) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		if _, ok := enabledByPath[path]; !ok {
			order = append(order, path)
		}
		enabledByPath[path] = enabled
	}

	for _, rule := range defaultAddDirRules {
		enabled := true
		if rule.Enabled != nil {
			enabled = *rule.Enabled
		}
		addRule(rule.Path, enabled)
	}

	for _, rule := range resolveConfiguredAddDirRules(metadata) {
		enabled := true
		if rule.Enabled != nil {
			enabled = *rule.Enabled
		}
		addRule(rule.Path, enabled)
	}

	raw := strings.TrimSpace(firstNonEmpty(
		metadataValue(metadata, codexAddDirsMetadataKey),
		os.Getenv(codexAddDirsEnvKey),
	))
	if raw != "" {
		for _, part := range strings.Split(raw, ",") {
			addRule(part, true)
		}
	}

	result := make([]string, 0, len(order))
	for _, path := range order {
		if !enabledByPath[path] {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			result = append(result, path)
		}
	}
	return result
}

func resolveConfiguredAddDirRules(metadata map[string]string) []codexAddDirRule {
	raw := strings.TrimSpace(firstNonEmpty(
		metadataValue(metadata, codexAddDirRulesMetadataKey),
		os.Getenv(codexAddDirRulesEnvKey),
	))
	if raw == "" {
		return nil
	}
	var rules []codexAddDirRule
	if err := json.Unmarshal([]byte(raw), &rules); err != nil {
		return nil
	}
	return rules
}

func boolPtr(value bool) *bool {
	v := value
	return &v
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

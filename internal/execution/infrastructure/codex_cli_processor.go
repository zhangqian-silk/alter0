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
	"sync"
	"time"

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
	codexWorktreeSourceRootKey       = "codex_worktree_source_root"
	codexWorkspaceModeSession        = "session"
	codexWorkspaceModeRepoRoot       = "repo-root"
	codexWorkspaceModeSessionRepo    = "session-repo-clone"
	defaultWorkspaceRootDir          = ".alter0"
	workspaceDirectoryName           = "workspaces"
	workspaceSessionsDirName         = "sessions"
	workspaceTasksDirName            = "tasks"
	workspaceRepoDirName             = "repo"
	taskIDMetadataKey                = "task_id"
	sessionIDMetadataFallback        = "session_id"
	codexAddDirsMetadataKey          = "codex_add_dirs"
	codexAddDirsEnvKey               = "ALTER0_CODEX_ADD_DIRS"
	codexAddDirRulesMetadataKey      = "codex_add_dir_rules"
	codexAddDirRulesEnvKey           = "ALTER0_CODEX_ADD_DIR_RULES"
	defaultCodexHeartbeatInterval    = time.Minute
	codexHeartbeatSource             = "codex_cli"
	codexGitPrepareTimeout           = 5 * time.Second
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
	command           string
	runner            commandRunner
	heartbeatInterval time.Duration
}

type codexThreadState struct {
	Enabled bool
	Path    string
}

type persistedCodexThreadState struct {
	ThreadID  string    `json:"thread_id"`
	UpdatedAt time.Time `json:"updated_at"`
}

type codexExecutionPayload struct {
	Protocol     string                     `json:"protocol"`
	UserPrompt   string                     `json:"user_prompt"`
	AgentContext *codexAgentContext         `json:"agent_context,omitempty"`
	Runtime      *execdomain.RuntimeContext `json:"runtime_context,omitempty"`
	SkillPolicy  *execdomain.SkillContext   `json:"skill_context,omitempty"`
	MCPPolicy    *execdomain.MCPContext     `json:"mcp_context,omitempty"`
	MemoryPolicy *execdomain.MemoryContext  `json:"memory_context,omitempty"`
}

type codexAgentContext struct {
	Protocol    string `json:"protocol"`
	AgentID     string `json:"agent_id,omitempty"`
	AgentName   string `json:"agent_name,omitempty"`
	DelegatedBy string `json:"delegated_by,omitempty"`
}

type codexJSONEvent struct {
	Type     string          `json:"type"`
	Message  string          `json:"message,omitempty"`
	ThreadID string          `json:"thread_id,omitempty"`
	Item     *codexEventItem `json:"item,omitempty"`
}

type codexEventItem struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	Delta string `json:"delta,omitempty"`
}

func NewCodexCLIProcessor() *CodexCLIProcessor {
	return NewCodexCLIProcessorWithCommand(defaultCodexCommand)
}

func NewCodexCLIProcessorWithCommand(command string) *CodexCLIProcessor {
	return &CodexCLIProcessor{
		command:           strings.TrimSpace(command),
		runner:            exec.CommandContext,
		heartbeatInterval: defaultCodexHeartbeatInterval,
	}
}

func (p *CodexCLIProcessor) Process(ctx context.Context, content string, metadata map[string]string) (string, error) {
	prompt := strings.TrimSpace(content)
	if prompt == "" {
		return "", errors.New("content is required")
	}
	workspaceDir, err := resolveCodexWorkspace(metadata)
	if err != nil {
		return "", err
	}
	prepared, err := prepareCodexInvocation(prompt, metadata, workspaceDir)
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

	threadState := resolveCodexThreadState(workspaceDir, metadata)
	threadID := loadCodexThreadID(threadState)
	args := buildCodexExecArgs(metadata, threadID, outputPath, false)
	procCtx, procCancel := context.WithCancel(ctx)
	defer procCancel()

	cmd := runner(procCtx, commandName, args...)
	if workspaceDir != "" {
		cmd.Dir = workspaceDir
	}
	cmd.Stdin = strings.NewReader(prepared.Prompt)
	if len(prepared.Env) > 0 {
		baseEnv := cmd.Env
		if len(baseEnv) == 0 {
			baseEnv = os.Environ()
		}
		cmd.Env = append(baseEnv, prepared.Env...)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		details := strings.TrimSpace(stderr.String())
		if details == "" {
			details = strings.TrimSpace(stdout.String())
		}
		if details == "" {
			return "", fmt.Errorf("codex command failed: %w", err)
		}
		return "", fmt.Errorf("codex command failed: %w: %s", err, details)
	}
	stopHeartbeat := p.startHeartbeatReporter(procCtx, cmd)
	waitErr := cmd.Wait()
	stopHeartbeat()
	if waitErr != nil {
		details := strings.TrimSpace(stderr.String())
		if details == "" {
			details = strings.TrimSpace(stdout.String())
		}
		if details == "" {
			return "", fmt.Errorf("codex command failed: %w", waitErr)
		}
		return "", fmt.Errorf("codex command failed: %w: %s", waitErr, details)
	}
	if threadState.Enabled {
		persistCodexThreadID(threadState, collectThreadIDFromOutput(stdout.String()))
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
	workspaceDir, err := resolveCodexWorkspace(metadata)
	if err != nil {
		return "", err
	}
	prepared, err := prepareCodexInvocation(prompt, metadata, workspaceDir)
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

	threadState := resolveCodexThreadState(workspaceDir, metadata)
	threadID := loadCodexThreadID(threadState)
	args := buildCodexExecArgs(metadata, threadID, "", true)
	procCtx, procCancel := context.WithCancel(ctx)
	defer procCancel()

	cmd := runner(procCtx, commandName, args...)
	if workspaceDir != "" {
		cmd.Dir = workspaceDir
	}
	cmd.Stdin = strings.NewReader(prepared.Prompt)
	if len(prepared.Env) > 0 {
		baseEnv := cmd.Env
		if len(baseEnv) == 0 {
			baseEnv = os.Environ()
		}
		cmd.Env = append(baseEnv, prepared.Env...)
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
	stopHeartbeat := p.startHeartbeatReporter(procCtx, cmd)

	output, threadID, scanErr := collectStreamOutput(stdoutPipe, emit)
	if scanErr != nil {
		procCancel()
		_ = cmd.Wait()
		stopHeartbeat()
		return "", scanErr
	}
	waitErr := cmd.Wait()
	stopHeartbeat()
	if waitErr != nil {
		details := strings.TrimSpace(stderr.String())
		if details == "" {
			return "", fmt.Errorf("codex command failed: %w", waitErr)
		}
		return "", fmt.Errorf("codex command failed: %w: %s", waitErr, details)
	}
	if threadState.Enabled {
		persistCodexThreadID(threadState, threadID)
	}
	result := strings.TrimSpace(output)
	if result == "" {
		return "", errors.New("codex returned empty output")
	}
	return result, nil
}

func (p *CodexCLIProcessor) startHeartbeatReporter(ctx context.Context, cmd *exec.Cmd) func() {
	if ctx == nil || cmd == nil || cmd.Process == nil {
		return func() {}
	}
	interval := p.heartbeatInterval
	if interval <= 0 {
		interval = defaultCodexHeartbeatInterval
	}

	stopCh := make(chan struct{})
	var once sync.Once
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-stopCh:
				return
			case <-ticker.C:
				if !isCodexProcessRunning(cmd) {
					return
				}
				execdomain.EmitRuntimeHeartbeat(ctx, execdomain.RuntimeHeartbeat{
					Source:    codexHeartbeatSource,
					Message:   fmt.Sprintf("codex cli session heartbeat ok (pid=%d)", cmd.Process.Pid),
					Timestamp: time.Now().UTC(),
				})
			}
		}
	}()

	return func() {
		once.Do(func() {
			close(stopCh)
		})
	}
}

func buildCodexExecArgs(metadata map[string]string, threadID string, outputPath string, stream bool) []string {
	args := []string{
		"exec",
		"--color", "never",
		"--skip-git-repo-check",
		"--sandbox", resolveCodexSandboxMode(metadata),
	}
	for _, dir := range resolveCodexAddDirs(metadata) {
		args = append(args, "--add-dir", dir)
	}
	if strings.TrimSpace(outputPath) != "" {
		args = append(args, "-o", outputPath)
	}
	if strings.TrimSpace(threadID) != "" {
		args = append(args, "resume")
		if stream {
			args = append(args, "--json")
		}
		args = append(args, strings.TrimSpace(threadID), "-")
		return args
	}
	if stream {
		args = append(args, "--json")
	}
	args = append(args, "-")
	return args
}

func resolveCodexThreadState(workspaceDir string, metadata map[string]string) codexThreadState {
	if resolveCodexRuntimeStrategy(metadata) == execdomain.CodexRuntimeStrategyPlain {
		return codexThreadState{}
	}
	trimmedWorkspace := strings.TrimSpace(workspaceDir)
	if trimmedWorkspace == "" {
		return codexThreadState{}
	}
	return codexThreadState{
		Enabled: true,
		Path:    filepath.Join(trimmedWorkspace, defaultWorkspaceRootDir, codexRuntimeDirName, "thread.json"),
	}
}

func loadCodexThreadID(state codexThreadState) string {
	if !state.Enabled || strings.TrimSpace(state.Path) == "" {
		return ""
	}
	data, err := os.ReadFile(state.Path)
	if err != nil {
		return ""
	}
	record := persistedCodexThreadState{}
	if err := json.Unmarshal(data, &record); err != nil {
		return ""
	}
	return strings.TrimSpace(record.ThreadID)
}

func persistCodexThreadID(state codexThreadState, threadID string) {
	threadID = strings.TrimSpace(threadID)
	if !state.Enabled || strings.TrimSpace(state.Path) == "" || threadID == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(state.Path), 0o755); err != nil {
		return
	}
	data, err := json.MarshalIndent(persistedCodexThreadState{
		ThreadID:  threadID,
		UpdatedAt: time.Now().UTC(),
	}, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(state.Path, append(data, '\n'), 0o600)
}

func isCodexProcessRunning(cmd *exec.Cmd) bool {
	if cmd == nil || cmd.Process == nil {
		return false
	}
	return cmd.ProcessState == nil
}

func collectStreamOutput(
	reader io.Reader,
	emit func(event execdomain.StreamEvent) error,
) (string, string, error) {
	scanner := bufio.NewScanner(reader)
	// Allow larger JSONL events for long model outputs.
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	emittedOutput := ""
	threadID := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		event := codexJSONEvent{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		if strings.TrimSpace(event.ThreadID) != "" {
			threadID = strings.TrimSpace(event.ThreadID)
		}
		if fatalMessage := fatalCodexEventMessage(event.Message); fatalMessage != "" {
			return "", threadID, fmt.Errorf("codex authentication failed: %s", fatalMessage)
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
				return "", threadID, err
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
				return "", threadID, err
			}
			emittedOutput += nextDelta
		}
	}

	if err := scanner.Err(); err != nil {
		return "", threadID, fmt.Errorf("read codex stream output: %w", err)
	}
	return emittedOutput, threadID, nil
}

func collectThreadIDFromOutput(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}
		event := codexJSONEvent{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		if threadID := strings.TrimSpace(event.ThreadID); threadID != "" {
			return threadID
		}
	}
	return ""
}

func fatalCodexEventMessage(message string) string {
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
	return strings.TrimSpace(prompt), nil
}

func buildCodexAgentContext(metadata map[string]string) *codexAgentContext {
	agentID := strings.TrimSpace(metadataValue(metadata, execdomain.AgentIDMetadataKey))
	agentName := strings.TrimSpace(metadataValue(metadata, execdomain.AgentNameMetadataKey))
	delegatedBy := strings.TrimSpace(metadataValue(metadata, execdomain.AgentDelegatedByMetadataKey))
	if agentID == "" && agentName == "" && delegatedBy == "" {
		return nil
	}
	return &codexAgentContext{
		Protocol:    "alter0.agent-context/v1",
		AgentID:     agentID,
		AgentName:   agentName,
		DelegatedBy: delegatedBy,
	}
}

func buildCodexRuntimeContext(metadata map[string]string) *execdomain.RuntimeContext {
	sessionID := strings.TrimSpace(metadataValue(metadata, execdomain.RuntimeSessionIDMetadataKey))
	messageID := strings.TrimSpace(metadataValue(metadata, execdomain.RuntimeMessageIDMetadataKey))
	traceID := strings.TrimSpace(metadataValue(metadata, execdomain.RuntimeTraceIDMetadataKey))

	context := execdomain.RuntimeContext{
		Protocol:  execdomain.RuntimeContextProtocolVersion,
		SessionID: sessionID,
		MessageID: messageID,
		TraceID:   traceID,
	}
	hasContent := sessionID != "" || messageID != "" || traceID != ""

	if workspace := buildCodexRuntimeWorkspace(metadata, sessionID); workspace != nil {
		context.Workspace = workspace
		hasContent = true
	}
	if repository := buildCodexRuntimeRepository(metadata); repository != nil {
		context.Repository = repository
		hasContent = true
	}
	if preview := buildCodexRuntimePreview(metadata, sessionID); preview != nil {
		context.Preview = preview
		hasContent = true
	}
	if !hasContent {
		return nil
	}
	return &context
}

func buildCodexRuntimeWorkspace(metadata map[string]string, sessionID string) *execdomain.RuntimeWorkspace {
	sessionID = sanitizeWorkspaceSegment(sessionID)
	taskID := sanitizeWorkspaceSegment(metadataValue(metadata, taskIDMetadataKey))
	mode := resolveCodexWorkspaceMode(metadata)
	if sessionID == "" && taskID == "" {
		return nil
	}

	workspace := &execdomain.RuntimeWorkspace{Mode: mode}
	sourceRepoRoot := strings.TrimSpace(metadataValue(metadata, codexWorktreeSourceRootKey))
	if sourceRepoRoot == "" {
		if repoRoot, err := resolveToolRepoRoot(); err == nil {
			sourceRepoRoot = repoRoot
		}
	}
	if sourceRepoRoot != "" && sessionID != "" {
		workspace.SessionPath = buildCodingSessionWorkspacePath(sourceRepoRoot, sessionID)
		if mode == codexWorkspaceModeSessionRepo {
			workspace.RepositoryPath = buildCodingSessionRepoWorkspacePath(sourceRepoRoot, sessionID)
		}
	}
	if workspace.SessionPath != "" && taskID != "" {
		workspace.TaskPath = filepath.ToSlash(filepath.Join(workspace.SessionPath, workspaceTasksDirName, taskID))
	}
	if workspace.Mode == "" && workspace.SessionPath == "" && workspace.TaskPath == "" && workspace.RepositoryPath == "" {
		return nil
	}
	return workspace
}

func buildCodexRuntimeRepository(metadata map[string]string) *execdomain.RuntimeRepository {
	sourceRepoRoot := strings.TrimSpace(metadataValue(metadata, codexWorktreeSourceRootKey))
	if sourceRepoRoot == "" && !isCodingAgent(metadata) {
		return nil
	}
	if sourceRepoRoot == "" {
		repoRoot, err := resolveToolRepoRoot()
		if err != nil {
			return nil
		}
		sourceRepoRoot = repoRoot
	}

	repository := &execdomain.RuntimeRepository{
		SourcePath: filepath.ToSlash(sourceRepoRoot),
	}
	repository.RemoteURL = resolveGitCommandOutput(sourceRepoRoot, "remote", "get-url", "origin")
	repository.ActiveBranch = resolveGitCommandOutput(sourceRepoRoot, "branch", "--show-current")
	if repository.ActiveBranch == "" {
		repository.ActiveBranch = resolveGitCommandOutput(sourceRepoRoot, "symbolic-ref", "--short", "HEAD")
	}
	repository.DefaultBranch = resolveGitDefaultBranch(sourceRepoRoot)
	if repository.SourcePath == "" && repository.RemoteURL == "" && repository.ActiveBranch == "" && repository.DefaultBranch == "" {
		return nil
	}
	return repository
}

func buildCodexRuntimePreview(metadata map[string]string, sessionID string) *execdomain.RuntimePreviewRule {
	if !isCodingAgent(metadata) {
		return nil
	}
	previewURL := buildPreviewURLForSession(sessionID)
	if previewURL == "" {
		return nil
	}
	return &execdomain.RuntimePreviewRule{
		URL:                  previewURL,
		RequiredOnCompletion: true,
	}
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
	case codexWorkspaceModeSession, codexWorkspaceModeRepoRoot, codexWorkspaceModeSessionRepo:
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
	if workspaceMode == codexWorkspaceModeSessionRepo {
		return resolveCodexSessionRepoWorkspace(metadata)
	}

	sessionWorkspace, err := resolveCodexSessionWorkspaceBase(metadata)
	if err != nil {
		return "", err
	}
	taskID := sanitizeWorkspaceSegment(metadataValue(metadata, taskIDMetadataKey))
	if taskID != "" {
		sessionWorkspace = filepath.Join(sessionWorkspace, workspaceTasksDirName, taskID)
	}
	if err := os.MkdirAll(sessionWorkspace, 0o755); err != nil {
		return "", fmt.Errorf("prepare codex workspace: %w", err)
	}
	return sessionWorkspace, nil
}

func resolveCodexSessionWorkspaceBase(metadata map[string]string) (string, error) {
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
	workspaceDir := filepath.Join(parts...)
	absolute, err := filepath.Abs(workspaceDir)
	if err != nil {
		return "", fmt.Errorf("resolve codex workspace path: %w", err)
	}
	return absolute, nil
}

func resolveCodexSessionRepoWorkspace(metadata map[string]string) (string, error) {
	sessionWorkspace, err := resolveCodexSessionWorkspaceBase(metadata)
	if err != nil {
		return "", err
	}
	sourceRepoRoot, err := resolveCodexWorktreeSourceRoot(metadata)
	if err != nil {
		return "", err
	}
	worktreeDir := filepath.Join(sessionWorkspace, workspaceRepoDirName)
	if err := ensureSessionRepoClone(sourceRepoRoot, worktreeDir); err != nil {
		return "", err
	}
	return worktreeDir, nil
}

func resolveCodexWorktreeSourceRoot(metadata map[string]string) (string, error) {
	repoRoot := strings.TrimSpace(metadataValue(metadata, codexWorktreeSourceRootKey))
	if repoRoot == "" {
		var err error
		repoRoot, err = resolveToolRepoRoot()
		if err != nil {
			return "", fmt.Errorf("resolve worktree source repo: %w", err)
		}
	}
	absolute, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", fmt.Errorf("resolve worktree source repo: %w", err)
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve worktree source repo: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("resolve worktree source repo: %s is not a directory", absolute)
	}
	topLevel, err := resolveGitTopLevel(absolute)
	if err != nil {
		return "", err
	}
	return topLevel, nil
}

func ensureSessionRepoClone(sourceRepoRoot string, worktreeDir string) error {
	if topLevel, err := resolveGitTopLevel(worktreeDir); err == nil && sameCleanPath(topLevel, worktreeDir) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(worktreeDir), 0o755); err != nil {
		return fmt.Errorf("prepare session repo workspace: %w", err)
	}
	if entries, err := os.ReadDir(worktreeDir); err == nil && len(entries) > 0 {
		return fmt.Errorf("prepare session repo workspace: %s already exists and is not a git repository", worktreeDir)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("prepare session repo workspace: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), codexGitPrepareTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "clone", "--no-hardlinks", sourceRepoRoot, worktreeDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		details := strings.TrimSpace(string(output))
		if details == "" {
			return fmt.Errorf("prepare session repo workspace: %w", err)
		}
		return fmt.Errorf("prepare session repo workspace: %w: %s", err, details)
	}
	if remoteURL := resolveGitCommandOutput(sourceRepoRoot, "remote", "get-url", "origin"); remoteURL != "" {
		cmd = exec.CommandContext(ctx, "git", "-C", worktreeDir, "remote", "set-url", "origin", remoteURL)
		output, err = cmd.CombinedOutput()
		if err != nil {
			details := strings.TrimSpace(string(output))
			if details == "" {
				return fmt.Errorf("prepare session repo workspace remote: %w", err)
			}
			return fmt.Errorf("prepare session repo workspace remote: %w: %s", err, details)
		}
	}
	return nil
}

func resolveGitTopLevel(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", errors.New("git path is required")
	}
	info, err := os.Stat(trimmed)
	if err != nil {
		return "", fmt.Errorf("resolve git top-level: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("resolve git top-level: %s is not a directory", trimmed)
	}
	ctx, cancel := context.WithTimeout(context.Background(), codexGitPrepareTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	cmd.Dir = trimmed
	output, err := cmd.CombinedOutput()
	if err != nil {
		details := strings.TrimSpace(string(output))
		if details == "" {
			return "", fmt.Errorf("resolve git top-level: %w", err)
		}
		return "", fmt.Errorf("resolve git top-level: %w: %s", err, details)
	}
	topLevel := strings.TrimSpace(string(output))
	if topLevel == "" {
		return "", errors.New("resolve git top-level: empty output")
	}
	return filepath.Clean(topLevel), nil
}

func sameCleanPath(left string, right string) bool {
	return filepath.Clean(strings.TrimSpace(left)) == filepath.Clean(strings.TrimSpace(right))
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

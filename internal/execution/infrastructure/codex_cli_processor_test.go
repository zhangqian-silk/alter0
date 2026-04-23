package infrastructure

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	execdomain "alter0/internal/execution/domain"
)

func TestCodexCLIProcessorProcessSuccess(t *testing.T) {
	processor := newTestProcessor("success", mustBuildTestPrompt(t, "reply: hello", testRuntimeMetadata()))

	output, err := processor.Process(context.Background(), "reply: hello", testRuntimeMetadata())
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if output != "mock response" {
		t.Fatalf("Process() output = %q, want %q", output, "mock response")
	}
}

func TestCodexCLIProcessorProcessEmitsHeartbeat(t *testing.T) {
	processor := newTestProcessor("slow-success", mustBuildTestPrompt(t, "reply: hello", testRuntimeMetadata()))
	processor.heartbeatInterval = 20 * time.Millisecond
	heartbeats := make(chan execdomain.RuntimeHeartbeat, 8)
	ctx := execdomain.WithRuntimeHeartbeatReporter(context.Background(), func(heartbeat execdomain.RuntimeHeartbeat) {
		heartbeats <- heartbeat
	})

	output, err := processor.Process(ctx, "reply: hello", testRuntimeMetadata())
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if output != "mock response" {
		t.Fatalf("Process() output = %q, want %q", output, "mock response")
	}
	if len(heartbeats) == 0 {
		t.Fatal("expected heartbeat during process execution")
	}
	heartbeat := <-heartbeats
	if heartbeat.Source != codexHeartbeatSource {
		t.Fatalf("heartbeat source = %q, want %q", heartbeat.Source, codexHeartbeatSource)
	}
	if !strings.Contains(heartbeat.Message, "heartbeat ok") {
		t.Fatalf("heartbeat message = %q, want heartbeat marker", heartbeat.Message)
	}
}

func TestCodexCLIProcessorProcessCommandFailure(t *testing.T) {
	processor := newTestProcessor("failure", mustBuildTestPrompt(t, "reply: hello", testRuntimeMetadata()))

	_, err := processor.Process(context.Background(), "reply: hello", testRuntimeMetadata())
	if err == nil {
		t.Fatal("Process() error = nil, want failure")
	}
	if !strings.Contains(err.Error(), "codex command failed") {
		t.Fatalf("Process() error = %q, want command failure marker", err.Error())
	}
	if !strings.Contains(err.Error(), "mock failure") {
		t.Fatalf("Process() error = %q, want helper stderr", err.Error())
	}
}

func TestCodexCLIProcessorProcessEmptyOutput(t *testing.T) {
	processor := newTestProcessor("empty", mustBuildTestPrompt(t, "reply: hello", testRuntimeMetadata()))

	_, err := processor.Process(context.Background(), "reply: hello", testRuntimeMetadata())
	if err == nil {
		t.Fatal("Process() error = nil, want empty output error")
	}
	if !strings.Contains(err.Error(), "empty output") {
		t.Fatalf("Process() error = %q, want empty output marker", err.Error())
	}
}

func TestCodexCLIProcessorProcessEmptyContent(t *testing.T) {
	processor := NewCodexCLIProcessor()

	_, err := processor.Process(context.Background(), "   ", nil)
	if err == nil {
		t.Fatal("Process() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "content is required") {
		t.Fatalf("Process() error = %q, want content validation", err.Error())
	}
}

func TestCodexCLIProcessorProcessWithNativeRuntimeAssets(t *testing.T) {
	rootDir := t.TempDir()
	activeHome := filepath.Join(t.TempDir(), "active-codex-home")
	if err := os.MkdirAll(activeHome, 0o755); err != nil {
		t.Fatalf("mkdir active home: %v", err)
	}
	if err := os.WriteFile(filepath.Join(activeHome, "auth.json"), []byte(`{"auth_mode":"apikey","OPENAI_API_KEY":"sk-test"}`), 0o600); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	t.Setenv("CODEX_HOME", activeHome)

	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("chdir root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	skillContext := execdomain.SkillContext{
		Protocol: execdomain.SkillContextProtocolVersion,
		Skills: []execdomain.SkillSpec{
			{
				ID:          "summary",
				Name:        "Summary",
				Description: "summary docs",
				Guide:       "review the memory files before editing",
				Priority:    200,
				ParameterTemplate: map[string]string{
					"lang": "zh-CN",
				},
				Constraints: []string{"max:300"},
			},
		},
	}
	rawSkillContext, err := json.Marshal(skillContext)
	if err != nil {
		t.Fatalf("marshal skill context: %v", err)
	}

	rawMemoryContext, err := json.Marshal(execdomain.MemoryContext{
		Protocol: execdomain.MemoryContextProtocolVersion,
		Files: []execdomain.MemoryFileSpec{
			{
				ID:        "user_md",
				Selection: "user_md",
				Title:     "USER.md",
				Path:      "/repo/USER.md",
				Exists:    true,
				Writable:  true,
				Content:   "name: alter0\nresponse_style: concise\n",
			},
		},
		Recall: []execdomain.MemoryRecallHit{
			{MemoryID: "user_md", Title: "USER.md", Snippet: "response_style: concise"},
		},
	})
	if err != nil {
		t.Fatalf("marshal memory context: %v", err)
	}
	rawMCPContext, err := json.Marshal(execdomain.MCPContext{
		Protocol: execdomain.MCPContextProtocolVersion,
		Servers: []execdomain.MCPServer{
			{
				ID:               "filesystem",
				Name:             "Filesystem",
				Scope:            "session",
				Transport:        "stdio",
				Command:          "npx",
				Args:             []string{"-y", "@modelcontextprotocol/server-filesystem"},
				TimeoutMS:        10000,
				FailureIsolation: true,
				ToolWhitelist:    []string{"read_file", "list_dir"},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal mcp context: %v", err)
	}
	processor := newTestProcessor("success", "reply: hello")

	output, err := processor.Process(context.Background(), "reply: hello", map[string]string{
		execdomain.RuntimeSessionIDMetadataKey: "session-default",
		execdomain.SkillContextMetadataKey:     string(rawSkillContext),
		execdomain.MemoryContextMetadataKey:    string(rawMemoryContext),
		execdomain.MCPContextMetadataKey:       string(rawMCPContext),
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if output != "mock response" {
		t.Fatalf("Process() output = %q, want %q", output, "mock response")
	}

	sessionWorkspace := filepath.Join(rootDir, ".alter0", "workspaces", "sessions", "session-default")
	configText, err := os.ReadFile(filepath.Join(sessionWorkspace, "codex-home", "config.toml"))
	if err != nil {
		t.Fatalf("read codex runtime config: %v", err)
	}
	for _, expected := range []string{
		`[mcp_servers.filesystem]`,
		`command = "npx"`,
		`enabled_tools = ["read_file", "list_dir"]`,
	} {
		if !strings.Contains(string(configText), expected) {
			t.Fatalf("expected config to contain %q, got:\n%s", expected, string(configText))
		}
	}
	agentsText, err := os.ReadFile(filepath.Join(sessionWorkspace, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read runtime AGENTS: %v", err)
	}
	for _, expected := range []string{
		".alter0/codex-runtime/runtime.md",
		".alter0/codex-runtime/skills.md",
		".alter0/codex-runtime/memory/",
	} {
		if !strings.Contains(string(agentsText), expected) {
			t.Fatalf("expected AGENTS to contain %q, got:\n%s", expected, string(agentsText))
		}
	}
	skillText, err := os.ReadFile(filepath.Join(sessionWorkspace, ".alter0", "codex-runtime", "skills.md"))
	if err != nil {
		t.Fatalf("read runtime skills: %v", err)
	}
	if !strings.Contains(string(skillText), "Summary") || !strings.Contains(string(skillText), "review the memory files before editing") {
		t.Fatalf("unexpected runtime skills:\n%s", string(skillText))
	}
	memoryText, err := os.ReadFile(filepath.Join(sessionWorkspace, ".alter0", "codex-runtime", "memory", "user_md.md"))
	if err != nil {
		t.Fatalf("read runtime memory: %v", err)
	}
	if !strings.Contains(string(memoryText), "response_style: concise") {
		t.Fatalf("unexpected runtime memory:\n%s", string(memoryText))
	}
}

func TestCodexCLIProcessorPlainStrategySkipsNativeRuntimeAssets(t *testing.T) {
	rootDir := t.TempDir()
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("chdir root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})
	rawSkillContext, err := json.Marshal(execdomain.SkillContext{
		Protocol: execdomain.SkillContextProtocolVersion,
		Skills: []execdomain.SkillSpec{
			{ID: "summary", Name: "Summary", Description: "summary docs"},
		},
	})
	if err != nil {
		t.Fatalf("marshal skill context: %v", err)
	}
	processor := newTestProcessor("success", "reply: hello")

	output, err := processor.Process(context.Background(), "reply: hello", map[string]string{
		execdomain.RuntimeSessionIDMetadataKey:   "session-default",
		execdomain.SkillContextMetadataKey:      string(rawSkillContext),
		execdomain.CodexRuntimeStrategyMetadataKey: execdomain.CodexRuntimeStrategyPlain,
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if output != "mock response" {
		t.Fatalf("Process() output = %q, want %q", output, "mock response")
	}
	sessionWorkspace := filepath.Join(rootDir, ".alter0", "workspaces", "sessions", "session-default")
	if _, err := os.Stat(filepath.Join(sessionWorkspace, "codex-home", "config.toml")); !os.IsNotExist(err) {
		t.Fatalf("expected plain strategy to skip codex runtime config, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(sessionWorkspace, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("expected plain strategy to skip runtime AGENTS, got err=%v", err)
	}
}

func TestCodexCLIProcessorProcessUsesSessionTaskWorkspace(t *testing.T) {
	expectedWorkspace := filepath.Join(".alter0", "workspaces", "sessions", "session-a", "tasks", "task-a")
	processor := newTestProcessor("success", mustBuildTestPrompt(t, "reply: hello", map[string]string{
		execdomain.RuntimeSessionIDMetadataKey: "session-a",
		"task_id":                              "task-a",
	}), expectedWorkspace)

	output, err := processor.Process(context.Background(), "reply: hello", map[string]string{
		execdomain.RuntimeSessionIDMetadataKey: "session-a",
		"task_id":                              "task-a",
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if output != "mock response" {
		t.Fatalf("Process() output = %q, want %q", output, "mock response")
	}
}

func TestResolveCodexWorkspaceSupportsRepoRootMode(t *testing.T) {
	workspace, err := resolveCodexWorkspace(map[string]string{
		codexWorkspaceModeMetadataKey: codexWorkspaceModeRepoRoot,
	})
	if err != nil {
		t.Fatalf("resolveCodexWorkspace() error = %v", err)
	}
	expected, absErr := filepath.Abs(".")
	if absErr != nil {
		t.Fatalf("resolve expected workspace: %v", absErr)
	}
	if workspace != expected {
		t.Fatalf("resolveCodexWorkspace() = %q, want %q", workspace, expected)
	}
}

func TestResolveCodexWorkspaceSupportsSessionRepoCloneMode(t *testing.T) {
	sourceRepoRoot := t.TempDir()
	initGitRepoWithCommit(t, sourceRepoRoot)

	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(sourceRepoRoot); err != nil {
		t.Fatalf("chdir source repo root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	workspace, err := resolveCodexWorkspace(map[string]string{
		execdomain.RuntimeSessionIDMetadataKey: "coding-session",
		codexWorkspaceModeMetadataKey:          codexWorkspaceModeSessionRepo,
		codexWorktreeSourceRootKey:             sourceRepoRoot,
	})
	if err != nil {
		t.Fatalf("resolveCodexWorkspace() error = %v", err)
	}
	expected, absErr := filepath.Abs(filepath.Join(".alter0", "workspaces", "sessions", "coding-session", "repo"))
	if absErr != nil {
		t.Fatalf("resolve expected workspace: %v", absErr)
	}
	if workspace != expected {
		t.Fatalf("resolveCodexWorkspace() = %q, want %q", workspace, expected)
	}
	topLevel, err := resolveGitTopLevel(workspace)
	if err != nil {
		t.Fatalf("resolveGitTopLevel() error = %v", err)
	}
	if topLevel != expected {
		t.Fatalf("resolveGitTopLevel() = %q, want %q", topLevel, expected)
	}
	gitDir, statErr := os.Stat(filepath.Join(workspace, ".git"))
	if statErr != nil {
		t.Fatalf("stat clone git dir: %v", statErr)
	}
	if !gitDir.IsDir() {
		t.Fatalf("expected full clone .git directory, got file at %q", filepath.Join(workspace, ".git"))
	}
}

func TestCodexCLIProcessorProcessAllowsRepoRootModeWithoutSessionContext(t *testing.T) {
	processor := newTestProcessor("success", "reply: hello")

	output, err := processor.Process(context.Background(), "reply: hello", map[string]string{
		codexWorkspaceModeMetadataKey: codexWorkspaceModeRepoRoot,
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if output != "mock response" {
		t.Fatalf("Process() output = %q, want %q", output, "mock response")
	}
}

func TestBuildCodexExecMetadataUsesSessionRepoCloneForCodingAgent(t *testing.T) {
	sourceRepoRoot := t.TempDir()
	initGitRepoWithCommit(t, sourceRepoRoot)

	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(sourceRepoRoot); err != nil {
		t.Fatalf("chdir source repo root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	metadata := buildCodexExecMetadata(map[string]string{
		execdomain.AgentIDMetadataKey:           "coding",
		execdomain.RuntimeSessionIDMetadataKey:  "coding-session",
		execdomain.AgentNameMetadataKey:         "Coding Agent",
		execdomain.AgentSystemPromptMetadataKey: "Own coding delivery.",
	})
	if got := metadata[codexWorkspaceModeMetadataKey]; got != codexWorkspaceModeSessionRepo {
		t.Fatalf("codex workspace mode = %q, want %q", got, codexWorkspaceModeSessionRepo)
	}
	if got := metadata[codexWorktreeSourceRootKey]; got != sourceRepoRoot {
		t.Fatalf("codex repo source root = %q, want %q", got, sourceRepoRoot)
	}
}

func TestCodexCLIProcessorProcessStreamSuccess(t *testing.T) {
	processor := newTestProcessor("stream-success", mustBuildTestPrompt(t, "reply: hello", testRuntimeMetadata()))
	deltas := make([]string, 0, 2)

	output, err := processor.ProcessStream(context.Background(), "reply: hello", testRuntimeMetadata(), func(event execdomain.StreamEvent) error {
		deltas = append(deltas, event.Text)
		return nil
	})
	if err != nil {
		t.Fatalf("ProcessStream() error = %v", err)
	}
	if output != "mock streamed response" {
		t.Fatalf("ProcessStream() output = %q, want %q", output, "mock streamed response")
	}
	if len(deltas) == 0 {
		t.Fatalf("expected stream deltas, got none")
	}
}

func TestCodexCLIProcessorProcessStreamEmitsHeartbeat(t *testing.T) {
	processor := newTestProcessor("stream-slow-success", mustBuildTestPrompt(t, "reply: hello", testRuntimeMetadata()))
	processor.heartbeatInterval = 20 * time.Millisecond
	heartbeats := make(chan execdomain.RuntimeHeartbeat, 8)
	ctx := execdomain.WithRuntimeHeartbeatReporter(context.Background(), func(heartbeat execdomain.RuntimeHeartbeat) {
		heartbeats <- heartbeat
	})

	output, err := processor.ProcessStream(ctx, "reply: hello", testRuntimeMetadata(), nil)
	if err != nil {
		t.Fatalf("ProcessStream() error = %v", err)
	}
	if output != "mock streamed response" {
		t.Fatalf("ProcessStream() output = %q, want %q", output, "mock streamed response")
	}
	if len(heartbeats) == 0 {
		t.Fatal("expected heartbeat during stream execution")
	}
}

func TestCodexCLIProcessorProcessStreamCommandFailure(t *testing.T) {
	processor := newTestProcessor("stream-failure", mustBuildTestPrompt(t, "reply: hello", testRuntimeMetadata()))

	_, err := processor.ProcessStream(context.Background(), "reply: hello", testRuntimeMetadata(), nil)
	if err == nil {
		t.Fatal("ProcessStream() error = nil, want failure")
	}
	if !strings.Contains(err.Error(), "codex command failed") {
		t.Fatalf("ProcessStream() error = %q, want command failure marker", err.Error())
	}
}

func TestCodexCLIProcessorProcessStreamFailsFastOnAuthError(t *testing.T) {
	processor := newTestProcessor("stream-auth-failure", mustBuildTestPrompt(t, "reply: hello", testRuntimeMetadata()))

	startedAt := time.Now()
	_, err := processor.ProcessStream(context.Background(), "reply: hello", testRuntimeMetadata(), nil)
	if err == nil {
		t.Fatal("ProcessStream() error = nil, want auth failure")
	}
	if !strings.Contains(err.Error(), "codex authentication failed") {
		t.Fatalf("ProcessStream() error = %q, want auth failure marker", err.Error())
	}
	if !strings.Contains(err.Error(), "Missing bearer or basic authentication") {
		t.Fatalf("ProcessStream() error = %q, want auth detail", err.Error())
	}
	if elapsed := time.Since(startedAt); elapsed > 2*time.Second {
		t.Fatalf("ProcessStream() elapsed = %s, want fast auth failure", elapsed)
	}
}

func TestCodexCLIProcessorProcessStreamUsesSessionWorkspace(t *testing.T) {
	expectedWorkspace := filepath.Join(".alter0", "workspaces", "sessions", "stream-session")
	processor := newTestProcessor("stream-success", mustBuildTestPrompt(t, "reply: hello", map[string]string{
		execdomain.RuntimeSessionIDMetadataKey: "stream-session",
	}), expectedWorkspace)

	output, err := processor.ProcessStream(context.Background(), "reply: hello", map[string]string{
		execdomain.RuntimeSessionIDMetadataKey: "stream-session",
	}, nil)
	if err != nil {
		t.Fatalf("ProcessStream() error = %v", err)
	}
	if output != "mock streamed response" {
		t.Fatalf("ProcessStream() output = %q, want %q", output, "mock streamed response")
	}
}

func TestCodexCLIProcessorProcessRequiresWorkspaceContext(t *testing.T) {
	processor := NewCodexCLIProcessor()

	_, err := processor.Process(context.Background(), "reply: hello", nil)
	if err == nil {
		t.Fatal("Process() error = nil, want workspace validation error")
	}
	if !strings.Contains(err.Error(), "workspace session context is required") {
		t.Fatalf("Process() error = %q, want workspace validation", err.Error())
	}
}

func testRuntimeMetadata() map[string]string {
	return map[string]string{
		execdomain.RuntimeSessionIDMetadataKey: "session-default",
	}
}

func mustBuildTestPrompt(t *testing.T, prompt string, metadata map[string]string) string {
	t.Helper()
	rendered, err := buildCodexPrompt(prompt, metadata)
	if err != nil {
		t.Fatalf("buildCodexPrompt() error = %v", err)
	}
	return rendered
}

func initGitRepoWithCommit(t *testing.T, dir string) {
	t.Helper()
	runGitCommand(t, dir, "init")
	runGitCommand(t, dir, "config", "user.name", "Alter0 Test")
	runGitCommand(t, dir, "config", "user.email", "alter0@example.com")
	runGitCommand(t, dir, "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("test repo\n"), 0o644); err != nil {
		t.Fatalf("write repo seed: %v", err)
	}
	runGitCommand(t, dir, "add", "README.md")
	runGitCommand(t, dir, "commit", "-m", "init repo")
}

func runGitCommand(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Alter0 Test",
		"GIT_AUTHOR_EMAIL=alter0@example.com",
		"GIT_COMMITTER_NAME=Alter0 Test",
		"GIT_COMMITTER_EMAIL=alter0@example.com",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v (%s)", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
}

func newTestProcessor(mode, expectedPrompt string, expectedWorkspaceSuffix ...string) *CodexCLIProcessor {
	expectedWorkspace := ""
	if len(expectedWorkspaceSuffix) > 0 {
		expectedWorkspace = strings.TrimSpace(expectedWorkspaceSuffix[0])
	}
	return &CodexCLIProcessor{
		command: "codex",
		runner: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			cmdArgs := append([]string{"-test.run=TestCodexCLIProcessorHelperProcess", "--", name}, args...)
			cmd := exec.CommandContext(ctx, os.Args[0], cmdArgs...)
			cmd.Env = append(
				os.Environ(),
				"GO_WANT_CODEX_HELPER_PROCESS=1",
				"CODEX_HELPER_MODE="+mode,
				"CODEX_HELPER_EXPECT_PROMPT="+expectedPrompt,
				"CODEX_HELPER_EXPECT_WORKSPACE_SUFFIX="+expectedWorkspace,
			)
			return cmd
		},
	}
}

func TestCodexCLIProcessorHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_CODEX_HELPER_PROCESS") != "1" {
		return
	}

	separatorIndex := -1
	for i, arg := range os.Args {
		if arg == "--" {
			separatorIndex = i
			break
		}
	}
	if separatorIndex < 0 || separatorIndex+1 >= len(os.Args) {
		os.Exit(2)
	}

	forwarded := os.Args[separatorIndex+1:]
	if len(forwarded) < 2 {
		os.Exit(2)
	}
	if forwarded[0] != "codex" || forwarded[1] != "exec" {
		os.Exit(2)
	}
	for _, arg := range forwarded {
		if arg == "-a" || arg == "--ask-for-approval" {
			os.Exit(2)
		}
	}
	if !containsArgPair(forwarded, "--sandbox", defaultCodexSandboxMode) {
		os.Exit(2)
	}

	promptArg := forwarded[len(forwarded)-1]
	prompt := promptArg
	if promptArg == "-" {
		rawPrompt, err := io.ReadAll(os.Stdin)
		if err != nil {
			os.Exit(2)
		}
		prompt = string(rawPrompt)
	}
	expectedPrompt := os.Getenv("CODEX_HELPER_EXPECT_PROMPT")
	if expectedPrompt != "" && promptArg != "-" {
		os.Exit(2)
	}
	if expectedPrompt != "" && prompt != expectedPrompt {
		os.Exit(2)
	}
	expectedWorkspace := strings.TrimSpace(os.Getenv("CODEX_HELPER_EXPECT_WORKSPACE_SUFFIX"))
	if expectedWorkspace != "" {
		workingDir, err := os.Getwd()
		if err != nil {
			os.Exit(2)
		}
		actual := strings.ToLower(strings.ReplaceAll(filepath.Clean(workingDir), "\\", "/"))
		expected := strings.ToLower(strings.ReplaceAll(filepath.Clean(expectedWorkspace), "\\", "/"))
		if !strings.HasSuffix(actual, expected) {
			os.Exit(2)
		}
	}

	mode := os.Getenv("CODEX_HELPER_MODE")

	outputPath := ""
	for i := 0; i < len(forwarded)-1; i++ {
		if forwarded[i] == "-o" && i+1 < len(forwarded) {
			outputPath = forwarded[i+1]
			break
		}
	}
	if !strings.HasPrefix(mode, "stream-") && strings.TrimSpace(outputPath) == "" {
		os.Exit(2)
	}

	switch mode {
	case "success":
		_ = os.WriteFile(outputPath, []byte("mock response\n"), 0o600)
		os.Exit(0)
	case "slow-success":
		time.Sleep(80 * time.Millisecond)
		_ = os.WriteFile(outputPath, []byte("mock response\n"), 0o600)
		os.Exit(0)
	case "empty":
		_ = os.WriteFile(outputPath, []byte(" \n"), 0o600)
		os.Exit(0)
	case "failure":
		_, _ = os.Stderr.WriteString("mock failure")
		os.Exit(19)
	case "stream-success":
		_, _ = os.Stdout.WriteString("{\"type\":\"thread.started\"}\n")
		_, _ = os.Stdout.WriteString("{\"type\":\"item.delta\",\"item\":{\"type\":\"agent_message\",\"delta\":\"mock \"}}\n")
		_, _ = os.Stdout.WriteString("{\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":\"mock streamed response\"}}\n")
		os.Exit(0)
	case "stream-slow-success":
		_, _ = os.Stdout.WriteString("{\"type\":\"thread.started\"}\n")
		time.Sleep(80 * time.Millisecond)
		_, _ = os.Stdout.WriteString("{\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":\"mock streamed response\"}}\n")
		os.Exit(0)
	case "stream-failure":
		_, _ = os.Stderr.WriteString("mock stream failure")
		os.Exit(19)
	case "stream-auth-failure":
		_, _ = os.Stdout.WriteString("{\"type\":\"thread.started\"}\n")
		_, _ = os.Stdout.WriteString("{\"type\":\"error\",\"message\":\"Reconnecting... 1/5 (unexpected status 401 Unauthorized: Missing bearer or basic authentication in header)\"}\n")
		time.Sleep(5 * time.Second)
		os.Exit(19)
	default:
		os.Exit(2)
	}
}

func containsArgPair(args []string, key string, value string) bool {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == key && args[i+1] == value {
			return true
		}
	}
	return false
}

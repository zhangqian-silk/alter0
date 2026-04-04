package infrastructure

import (
	"context"
	"encoding/json"
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

func TestCodexCLIProcessorProcessWithSkillContextPayload(t *testing.T) {
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

	expectedPrompt, err := buildCodexPrompt("reply: hello", map[string]string{
		execdomain.RuntimeSessionIDMetadataKey: "session-default",
		execdomain.SkillContextMetadataKey:     string(rawSkillContext),
	})
	if err != nil {
		t.Fatalf("buildCodexPrompt() error = %v", err)
	}
	processor := newTestProcessor("success", expectedPrompt)

	output, err := processor.Process(context.Background(), "reply: hello", map[string]string{
		execdomain.RuntimeSessionIDMetadataKey: "session-default",
		execdomain.SkillContextMetadataKey:     string(rawSkillContext),
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if output != "mock response" {
		t.Fatalf("Process() output = %q, want %q", output, "mock response")
	}
}

func TestCodexCLIProcessorProcessWithMCPContextPayload(t *testing.T) {
	mcpContext := execdomain.MCPContext{
		Protocol: execdomain.MCPContextProtocolVersion,
		Servers: []execdomain.MCPServer{
			{
				ID:               "github-mcp",
				Name:             "GitHub MCP",
				Scope:            "request",
				Transport:        "http",
				URL:              "https://mcp.example.com",
				TimeoutMS:        9000,
				FailureIsolation: true,
			},
		},
	}
	rawMCPContext, err := json.Marshal(mcpContext)
	if err != nil {
		t.Fatalf("marshal mcp context: %v", err)
	}

	expectedPrompt, err := buildCodexPrompt("reply: hello", map[string]string{
		execdomain.RuntimeSessionIDMetadataKey: "session-default",
		execdomain.MCPContextMetadataKey:       string(rawMCPContext),
	})
	if err != nil {
		t.Fatalf("buildCodexPrompt() error = %v", err)
	}
	processor := newTestProcessor("success", expectedPrompt)

	output, err := processor.Process(context.Background(), "reply: hello", map[string]string{
		execdomain.RuntimeSessionIDMetadataKey: "session-default",
		execdomain.MCPContextMetadataKey:       string(rawMCPContext),
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if output != "mock response" {
		t.Fatalf("Process() output = %q, want %q", output, "mock response")
	}
}

func TestCodexCLIProcessorProcessWithSkillAndMCPContextPayload(t *testing.T) {
	skillContext := execdomain.SkillContext{
		Protocol: execdomain.SkillContextProtocolVersion,
		Skills: []execdomain.SkillSpec{
			{
				ID:          "summary",
				Name:        "Summary",
				Description: "summary docs",
				Priority:    200,
			},
		},
	}
	rawSkillContext, err := json.Marshal(skillContext)
	if err != nil {
		t.Fatalf("marshal skill context: %v", err)
	}

	mcpContext := execdomain.MCPContext{
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
			},
		},
	}
	rawMCPContext, err := json.Marshal(mcpContext)
	if err != nil {
		t.Fatalf("marshal mcp context: %v", err)
	}

	expectedPrompt, err := buildCodexPrompt("reply: hello", map[string]string{
		execdomain.RuntimeSessionIDMetadataKey: "session-default",
		execdomain.SkillContextMetadataKey:     string(rawSkillContext),
		execdomain.MCPContextMetadataKey:       string(rawMCPContext),
	})
	if err != nil {
		t.Fatalf("buildCodexPrompt() error = %v", err)
	}
	processor := newTestProcessor("success", expectedPrompt)

	output, err := processor.Process(context.Background(), "reply: hello", map[string]string{
		execdomain.RuntimeSessionIDMetadataKey: "session-default",
		execdomain.SkillContextMetadataKey:     string(rawSkillContext),
		execdomain.MCPContextMetadataKey:       string(rawMCPContext),
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if output != "mock response" {
		t.Fatalf("Process() output = %q, want %q", output, "mock response")
	}
}

func TestCodexCLIProcessorProcessWithMemoryContextPayload(t *testing.T) {
	memoryContext := execdomain.MemoryContext{
		Protocol: execdomain.MemoryContextProtocolVersion,
		Files: []execdomain.MemoryFileSpec{
			{
				ID:        "user_md",
				Selection: "user_md",
				Title:     "USER.md",
				Path:      "/repo/USER.md",
				Exists:    true,
				Writable:  true,
				Content:   "name: alter0",
			},
		},
	}
	rawMemoryContext, err := json.Marshal(memoryContext)
	if err != nil {
		t.Fatalf("marshal memory context: %v", err)
	}

	expectedPrompt, err := buildCodexPrompt("reply: hello", map[string]string{
		execdomain.RuntimeSessionIDMetadataKey: "session-default",
		execdomain.MemoryContextMetadataKey:    string(rawMemoryContext),
	})
	if err != nil {
		t.Fatalf("buildCodexPrompt() error = %v", err)
	}
	processor := newTestProcessor("success", expectedPrompt)

	output, err := processor.Process(context.Background(), "reply: hello", map[string]string{
		execdomain.RuntimeSessionIDMetadataKey: "session-default",
		execdomain.MemoryContextMetadataKey:    string(rawMemoryContext),
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if output != "mock response" {
		t.Fatalf("Process() output = %q, want %q", output, "mock response")
	}
}

func TestCodexCLIProcessorProcessWithAgentContextPayload(t *testing.T) {
	expectedPrompt, err := buildCodexPrompt("reply: hello", map[string]string{
		execdomain.RuntimeSessionIDMetadataKey:  "session-default",
		execdomain.AgentIDMetadataKey:           "coding",
		execdomain.AgentNameMetadataKey:         "Coding Agent",
		execdomain.AgentSystemPromptMetadataKey: "Drive implementation through Codex.",
		execdomain.AgentDelegatedByMetadataKey:  "main",
	})
	if err != nil {
		t.Fatalf("buildCodexPrompt() error = %v", err)
	}
	processor := newTestProcessor("success", expectedPrompt)

	output, err := processor.Process(context.Background(), "reply: hello", map[string]string{
		execdomain.RuntimeSessionIDMetadataKey:  "session-default",
		execdomain.AgentIDMetadataKey:           "coding",
		execdomain.AgentNameMetadataKey:         "Coding Agent",
		execdomain.AgentSystemPromptMetadataKey: "Drive implementation through Codex.",
		execdomain.AgentDelegatedByMetadataKey:  "main",
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if output != "mock response" {
		t.Fatalf("Process() output = %q, want %q", output, "mock response")
	}
}

func TestBuildCodexPromptIncludesRuntimeContextForCodingAgent(t *testing.T) {
	sourceRepoRoot := t.TempDir()
	initGitRepoWithCommit(t, sourceRepoRoot)
	runGitCommand(t, sourceRepoRoot, "checkout", "-b", "feat/runtime-context")
	runGitCommand(t, sourceRepoRoot, "remote", "add", "origin", "https://example.com/demo/repo.git")

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

	rawPrompt, err := buildCodexPrompt("reply: hello", buildCodexExecMetadata(map[string]string{
		execdomain.RuntimeSessionIDMetadataKey: "coding-session",
		execdomain.RuntimeMessageIDMetadataKey: "message-1",
		execdomain.RuntimeTraceIDMetadataKey:   "trace-1",
		execdomain.AgentIDMetadataKey:          "coding",
	}))
	if err != nil {
		t.Fatalf("buildCodexPrompt() error = %v", err)
	}

	payload := codexExecutionPayload{}
	if err := json.Unmarshal([]byte(rawPrompt), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Runtime == nil {
		t.Fatal("expected runtime context in codex payload")
	}
	if payload.Runtime.Protocol != execdomain.RuntimeContextProtocolVersion {
		t.Fatalf("runtime protocol = %q, want %q", payload.Runtime.Protocol, execdomain.RuntimeContextProtocolVersion)
	}
	if payload.Runtime.SessionID != "coding-session" || payload.Runtime.MessageID != "message-1" || payload.Runtime.TraceID != "trace-1" {
		t.Fatalf("unexpected runtime ids: %+v", payload.Runtime)
	}
	if payload.Runtime.Workspace == nil || payload.Runtime.Workspace.RepositoryPath == "" {
		t.Fatalf("expected runtime workspace repository path, got %+v", payload.Runtime.Workspace)
	}
	if !strings.HasSuffix(payload.Runtime.Workspace.RepositoryPath, "/.alter0/workspaces/sessions/coding-session/repo") {
		t.Fatalf("unexpected repository workspace path: %q", payload.Runtime.Workspace.RepositoryPath)
	}
	if payload.Runtime.Repository == nil || payload.Runtime.Repository.SourcePath != filepath.ToSlash(sourceRepoRoot) {
		t.Fatalf("unexpected runtime repository context: %+v", payload.Runtime.Repository)
	}
	if payload.Runtime.Repository.RemoteURL != "https://example.com/demo/repo.git" {
		t.Fatalf("runtime repository remote = %q", payload.Runtime.Repository.RemoteURL)
	}
	if payload.Runtime.Repository.ActiveBranch != "feat/runtime-context" {
		t.Fatalf("runtime repository branch = %q", payload.Runtime.Repository.ActiveBranch)
	}
	if payload.Runtime.Preview == nil || !payload.Runtime.Preview.RequiredOnCompletion {
		t.Fatalf("unexpected runtime preview rule: %+v", payload.Runtime.Preview)
	}
	if !strings.HasPrefix(payload.Runtime.Preview.URL, "https://") || !strings.HasSuffix(payload.Runtime.Preview.URL, ".alter0.cn") {
		t.Fatalf("unexpected runtime preview url: %q", payload.Runtime.Preview.URL)
	}
}

func TestBuildCodexPromptIncludesProductContexts(t *testing.T) {
	productContext := execdomain.ProductContext{
		Protocol:      execdomain.ProductContextProtocolVersion,
		ProductID:     "travel",
		Name:          "Travel",
		MasterAgentID: "travel-master",
	}
	rawProductContext, err := json.Marshal(productContext)
	if err != nil {
		t.Fatalf("marshal product context: %v", err)
	}
	productDiscovery := execdomain.ProductDiscoveryContext{
		Protocol:        execdomain.ProductDiscoveryProtocolVersion,
		SelectedProduct: "travel",
		SelectionReason: "matched by route",
		MatchedProducts: []execdomain.ProductContext{productContext},
	}
	rawProductDiscovery, err := json.Marshal(productDiscovery)
	if err != nil {
		t.Fatalf("marshal product discovery: %v", err)
	}

	rawPrompt, err := buildCodexPrompt("reply: hello", map[string]string{
		execdomain.RuntimeSessionIDMetadataKey: "product-session",
		execdomain.ProductContextMetadataKey:   string(rawProductContext),
		execdomain.ProductDiscoveryMetadataKey: string(rawProductDiscovery),
	})
	if err != nil {
		t.Fatalf("buildCodexPrompt() error = %v", err)
	}

	payload := codexExecutionPayload{}
	if err := json.Unmarshal([]byte(rawPrompt), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Product == nil || payload.Product.ProductID != "travel" {
		t.Fatalf("unexpected product context: %+v", payload.Product)
	}
	if payload.Discovery == nil || payload.Discovery.SelectedProduct != "travel" {
		t.Fatalf("unexpected product discovery context: %+v", payload.Discovery)
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

func TestResolveCodexWorkspaceSupportsSessionRepoWorktreeMode(t *testing.T) {
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

func TestBuildCodexExecMetadataUsesSessionRepoWorktreeForCodingAgent(t *testing.T) {
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
		t.Fatalf("codex worktree source root = %q, want %q", got, sourceRepoRoot)
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

	prompt := forwarded[len(forwarded)-1]
	expectedPrompt := os.Getenv("CODEX_HELPER_EXPECT_PROMPT")
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

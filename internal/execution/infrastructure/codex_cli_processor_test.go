package infrastructure

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	execdomain "alter0/internal/execution/domain"
)

func TestCodexCLIProcessorProcessSuccess(t *testing.T) {
	processor := newTestProcessor("success", "reply: hello")

	output, err := processor.Process(context.Background(), "reply: hello", testRuntimeMetadata())
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if output != "mock response" {
		t.Fatalf("Process() output = %q, want %q", output, "mock response")
	}
}

func TestCodexCLIProcessorProcessCommandFailure(t *testing.T) {
	processor := newTestProcessor("failure", "reply: hello")

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
	processor := newTestProcessor("empty", "reply: hello")

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

	expectedPrompt := `{"protocol":"alter0.codex-exec/v1","user_prompt":"reply: hello","skill_context":{"protocol":"alter0.skill-context/v1","skills":[{"id":"summary","name":"Summary","description":"summary docs","priority":200,"parameter_template":{"lang":"zh-CN"},"constraints":["max:300"]}]}}`
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

	expectedPrompt := `{"protocol":"alter0.codex-exec/v1","user_prompt":"reply: hello","mcp_context":{"protocol":"alter0.mcp-context/v1","servers":[{"id":"github-mcp","name":"GitHub MCP","scope":"request","transport":"http","url":"https://mcp.example.com","timeout_ms":9000,"failure_isolation":true}]}}`
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

	expectedPrompt := `{"protocol":"alter0.codex-exec/v1","user_prompt":"reply: hello","skill_context":{"protocol":"alter0.skill-context/v1","skills":[{"id":"summary","name":"Summary","description":"summary docs","priority":200}]},"mcp_context":{"protocol":"alter0.mcp-context/v1","servers":[{"id":"filesystem","name":"Filesystem","scope":"session","transport":"stdio","command":"npx","args":["-y","@modelcontextprotocol/server-filesystem"],"timeout_ms":10000,"failure_isolation":true}]}}`
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

func TestCodexCLIProcessorProcessUsesSessionTaskWorkspace(t *testing.T) {
	expectedWorkspace := filepath.Join(".alter0", "workspaces", "sessions", "session-a", "tasks", "task-a")
	processor := newTestProcessor("success", "reply: hello", expectedWorkspace)

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

func TestCodexCLIProcessorProcessStreamSuccess(t *testing.T) {
	processor := newTestProcessor("stream-success", "reply: hello")
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

func TestCodexCLIProcessorProcessStreamCommandFailure(t *testing.T) {
	processor := newTestProcessor("stream-failure", "reply: hello")

	_, err := processor.ProcessStream(context.Background(), "reply: hello", testRuntimeMetadata(), nil)
	if err == nil {
		t.Fatal("ProcessStream() error = nil, want failure")
	}
	if !strings.Contains(err.Error(), "codex command failed") {
		t.Fatalf("ProcessStream() error = %q, want command failure marker", err.Error())
	}
}

func TestCodexCLIProcessorProcessStreamUsesSessionWorkspace(t *testing.T) {
	expectedWorkspace := filepath.Join(".alter0", "workspaces", "sessions", "stream-session")
	processor := newTestProcessor("stream-success", "reply: hello", expectedWorkspace)

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
	case "stream-failure":
		_, _ = os.Stderr.WriteString("mock stream failure")
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

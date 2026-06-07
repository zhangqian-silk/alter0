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

func TestClaudeCodeProcessorProcessPreparesRuntimeAndProviderEnv(t *testing.T) {
	workspace := t.TempDir()
	metadata := testRuntimeMetadata()
	metadata[execdomain.ClaudeAPIKeyMetadataKey] = "sk-test"
	metadata[execdomain.ClaudeBaseURLMetadataKey] = "https://claude-gateway.example/v1"
	metadata[execdomain.LLMModelMetadataKey] = "claude-sonnet-4-6"
	metadata[codexWorkspaceRootDirMetadataKey] = workspace
	metadata[codexWorkspaceModeMetadataKey] = codexWorkspaceModeSession
	metadata[execdomain.SkillContextMetadataKey] = mustMarshalClaudeTestSkillContext(t, workspace)

	processor := newTestClaudeProcessor("success", "整理方案", filepath.Join(".alter0", "workspaces", "sessions", "session-default"))

	output, err := processor.Process(context.Background(), "整理方案", metadata)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if output != "mock claude response" {
		t.Fatalf("Process() output = %q, want mock claude response", output)
	}
	sessionWorkspace := filepath.Join(workspace, ".alter0", "workspaces", "sessions", "session-default")
	assertClaudeFileContains(t, filepath.Join(sessionWorkspace, "CLAUDE.md"), "Read `.alter0/claude-runtime/runtime.md`")
	assertClaudeFileContains(t, filepath.Join(sessionWorkspace, ".alter0", "claude-runtime", "runtime.md"), "session-default")
	assertClaudeFileContains(t, filepath.Join(sessionWorkspace, ".alter0", "claude-runtime", "skills.md"), "- file_path: .alter0/claude-runtime/skills/summary/SKILL.md")
	assertClaudeFileContains(t, filepath.Join(sessionWorkspace, ".alter0", "claude-runtime", "skills", "summary", "references", "style.md"), "brief")
}

func mustMarshalClaudeTestSkillContext(t *testing.T, rootDir string) string {
	t.Helper()
	skillDir := filepath.Join(rootDir, "docs", "skills", "summary", "references")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	skillPath := filepath.Join(rootDir, "docs", "skills", "summary", "SKILL.md")
	if err := os.WriteFile(skillPath, []byte("# Summary\n"), 0o644); err != nil {
		t.Fatalf("write skill file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "style.md"), []byte("brief\n"), 0o644); err != nil {
		t.Fatalf("write skill reference: %v", err)
	}
	raw, err := json.Marshal(execdomain.SkillContext{
		Protocol: execdomain.SkillContextProtocolVersion,
		Skills: []execdomain.SkillSpec{
			{
				ID:          "summary",
				Name:        "Summary",
				Description: "summary docs",
				FilePath:    skillPath,
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal skill context: %v", err)
	}
	return string(raw)
}

func newTestClaudeProcessor(mode, expectedPrompt string, expectedWorkspaceSuffix string) *ClaudeCodeProcessor {
	return &ClaudeCodeProcessor{
		command: "claude",
		runner: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			cmdArgs := append([]string{"-test.run=TestClaudeCodeProcessorHelperProcess", "--", name}, args...)
			cmd := exec.CommandContext(ctx, os.Args[0], cmdArgs...)
			cmd.Env = append(
				os.Environ(),
				"GO_WANT_CLAUDE_HELPER_PROCESS=1",
				"CLAUDE_HELPER_MODE="+mode,
				"CLAUDE_HELPER_EXPECT_PROMPT="+expectedPrompt,
				"CLAUDE_HELPER_EXPECT_WORKSPACE_SUFFIX="+expectedWorkspaceSuffix,
			)
			return cmd
		},
	}
}

func TestClaudeCodeProcessorHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_CLAUDE_HELPER_PROCESS") != "1" {
		return
	}

	separatorIndex := -1
	for i, arg := range os.Args {
		if arg == "--" {
			separatorIndex = i
			break
		}
	}
	if separatorIndex < 0 {
		os.Exit(2)
	}
	args := os.Args[separatorIndex+1:]
	if len(args) == 0 || args[0] != "claude" {
		os.Exit(3)
	}
	if os.Getenv("ANTHROPIC_API_KEY") != "sk-test" {
		os.Exit(4)
	}
	if os.Getenv("ANTHROPIC_BASE_URL") != "https://claude-gateway.example/v1" {
		os.Exit(5)
	}
	if strings.TrimSpace(os.Getenv("CLAUDE_CONFIG_DIR")) == "" {
		os.Exit(6)
	}
	if !containsArgSequence(args, "--print") || !containsArgSequence(args, "--output-format", "text") || !containsArgSequence(args, "--model", "claude-sonnet-4-6") {
		os.Exit(7)
	}
	expectedPrompt := os.Getenv("CLAUDE_HELPER_EXPECT_PROMPT")
	if len(args) == 0 || args[len(args)-1] != expectedPrompt {
		os.Exit(8)
	}
	expectedWorkspace := filepath.FromSlash(os.Getenv("CLAUDE_HELPER_EXPECT_WORKSPACE_SUFFIX"))
	if expectedWorkspace != "" && !strings.HasSuffix(filepath.Clean(mustGetwdForClaudeHelper()), expectedWorkspace) {
		os.Exit(9)
	}

	switch os.Getenv("CLAUDE_HELPER_MODE") {
	case "success":
		_, _ = os.Stdout.WriteString("mock claude response\n")
	default:
		_, _ = os.Stderr.WriteString("unexpected claude helper mode")
		os.Exit(10)
	}
	os.Exit(0)
}

func mustGetwdForClaudeHelper() string {
	wd, err := os.Getwd()
	if err != nil {
		os.Exit(11)
	}
	return wd
}

func assertClaudeFileContains(t *testing.T, path string, expected string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}
	if !strings.Contains(string(content), expected) {
		t.Fatalf("file %s does not contain %q:\n%s", path, expected, string(content))
	}
}

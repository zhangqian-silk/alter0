package infrastructure

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	execdomain "alter0/internal/execution/domain"
)

func TestCodexCLIProcessorProcessSuccess(t *testing.T) {
	processor := newTestProcessor("success", "reply: hello")

	output, err := processor.Process(context.Background(), "reply: hello", nil)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if output != "mock response" {
		t.Fatalf("Process() output = %q, want %q", output, "mock response")
	}
}

func TestCodexCLIProcessorProcessCommandFailure(t *testing.T) {
	processor := newTestProcessor("failure", "reply: hello")

	_, err := processor.Process(context.Background(), "reply: hello", nil)
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

	_, err := processor.Process(context.Background(), "reply: hello", nil)
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
		execdomain.SkillContextMetadataKey: string(rawSkillContext),
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if output != "mock response" {
		t.Fatalf("Process() output = %q, want %q", output, "mock response")
	}
}

func newTestProcessor(mode, expectedPrompt string) *CodexCLIProcessor {
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

	outputPath := ""
	for i := 0; i < len(forwarded)-1; i++ {
		if forwarded[i] == "-o" && i+1 < len(forwarded) {
			outputPath = forwarded[i+1]
			break
		}
	}
	if strings.TrimSpace(outputPath) == "" {
		os.Exit(2)
	}

	prompt := forwarded[len(forwarded)-1]
	expectedPrompt := os.Getenv("CODEX_HELPER_EXPECT_PROMPT")
	if expectedPrompt != "" && prompt != expectedPrompt {
		os.Exit(2)
	}

	switch os.Getenv("CODEX_HELPER_MODE") {
	case "success":
		_ = os.WriteFile(outputPath, []byte("mock response\n"), 0o600)
		os.Exit(0)
	case "empty":
		_ = os.WriteFile(outputPath, []byte(" \n"), 0o600)
		os.Exit(0)
	case "failure":
		_, _ = os.Stderr.WriteString("mock failure")
		os.Exit(19)
	default:
		os.Exit(2)
	}
}

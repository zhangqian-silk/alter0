package infrastructure

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const defaultCodexCommand = "codex"

type commandRunner func(ctx context.Context, name string, args ...string) *exec.Cmd

type CodexCLIProcessor struct {
	command string
	runner  commandRunner
}

func NewCodexCLIProcessor() *CodexCLIProcessor {
	return &CodexCLIProcessor{
		command: defaultCodexCommand,
		runner:  exec.CommandContext,
	}
}

func (p *CodexCLIProcessor) Process(ctx context.Context, content string, _ map[string]string) (string, error) {
	prompt := strings.TrimSpace(content)
	if prompt == "" {
		return "", errors.New("content is required")
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
		"-o", outputPath,
		prompt,
	}
	cmd := runner(ctx, commandName, args...)

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

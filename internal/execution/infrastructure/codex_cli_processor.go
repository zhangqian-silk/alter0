package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	execdomain "alter0/internal/execution/domain"
)

const defaultCodexCommand = "codex"

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
		renderedPrompt,
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

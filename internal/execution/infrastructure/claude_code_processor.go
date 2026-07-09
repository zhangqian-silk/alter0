package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"alter0/internal/codex/infrastructure/runtimeconfig"
	execdomain "alter0/internal/execution/domain"
)

const (
	defaultClaudeCommand      = "claude"
	claudeRuntimeDirName      = "claude-runtime"
	claudeRuntimeHomeDirName  = "claude-home"
	claudeRuntimeRuntimePath  = ".alter0/claude-runtime/runtime.md"
	claudeRuntimeSkillsPath   = ".alter0/claude-runtime/skills.md"
	claudeRuntimeMemoryDir    = ".alter0/claude-runtime/memory"
	claudeRuntimeRootFileName = "CLAUDE.md"
)

type ClaudeCodeProcessor struct {
	command string
	runner  commandRunner
}

func NewClaudeCodeProcessor() *ClaudeCodeProcessor {
	return NewClaudeCodeProcessorWithCommand(defaultClaudeCommand)
}

func NewClaudeCodeProcessorWithCommand(command string) *ClaudeCodeProcessor {
	return &ClaudeCodeProcessor{
		command: strings.TrimSpace(command),
		runner:  exec.CommandContext,
	}
}

func (p *ClaudeCodeProcessor) Process(ctx context.Context, content string, metadata map[string]string) (string, error) {
	prompt := strings.TrimSpace(content)
	if prompt == "" {
		return "", errors.New("content is required")
	}
	apiKey := strings.TrimSpace(metadataValue(metadata, execdomain.ClaudeAPIKeyMetadataKey))
	if apiKey == "" {
		return "", errors.New("claude api key is required")
	}
	workspaceDir, err := resolveCodexWorkspace(metadata)
	if err != nil {
		return "", err
	}
	prepared, err := prepareClaudeCodeInvocation(metadata, workspaceDir)
	if err != nil {
		return "", err
	}

	commandName := strings.TrimSpace(p.command)
	if commandName == "" {
		commandName = defaultClaudeCommand
	}
	runner := p.runner
	if runner == nil {
		runner = exec.CommandContext
	}

	args := buildClaudeCodeArgs(prompt, metadata)
	cmd := runner(ctx, commandName, args...)
	if workspaceDir != "" {
		cmd.Dir = workspaceDir
	}
	baseEnv := cmd.Env
	if len(baseEnv) == 0 {
		baseEnv = os.Environ()
	}
	cmd.Env = append(baseEnv, prepared.Env...)

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
			return "", fmt.Errorf("claude command failed: %w", err)
		}
		return "", fmt.Errorf("claude command failed: %w: %s", err, details)
	}
	result := strings.TrimSpace(stdout.String())
	threadTitle, result := normalizeClaudeCodeOutput(result)
	storeRuntimeThreadTitle(metadata, threadTitle)
	if result == "" {
		return "", errors.New("claude returned empty output")
	}
	return result, nil
}

type claudeStructuredOutputEvent struct {
	Type              string                  `json:"type"`
	Title             string                  `json:"title,omitempty"`
	Name              string                  `json:"name,omitempty"`
	ThreadTitle       string                  `json:"thread_title,omitempty"`
	ConversationTitle string                  `json:"conversation_title,omitempty"`
	Thread            *claudeStructuredThread `json:"thread,omitempty"`
	Session           *claudeStructuredThread `json:"session,omitempty"`
	Conversation      *claudeStructuredThread `json:"conversation,omitempty"`
}

type claudeStructuredThread struct {
	Title string `json:"title,omitempty"`
	Name  string `json:"name,omitempty"`
}

func normalizeClaudeCodeOutput(output string) (string, string) {
	threadTitle := ""
	visibleLines := []string{}
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "{") {
			event := claudeStructuredOutputEvent{}
			if err := json.Unmarshal([]byte(trimmed), &event); err == nil {
				if title := externalThreadTitleFromClaudeStructuredEvent(event); title != "" {
					threadTitle = title
					continue
				}
			}
		}
		visibleLines = append(visibleLines, line)
	}
	return threadTitle, strings.TrimSpace(strings.Join(visibleLines, "\n"))
}

func externalThreadTitleFromClaudeStructuredEvent(event claudeStructuredOutputEvent) string {
	for _, candidate := range []string{
		event.Title,
		event.ThreadTitle,
		event.ConversationTitle,
		event.Name,
		nestedClaudeStructuredThreadTitle(event.Thread),
		nestedClaudeStructuredThreadTitle(event.Session),
		nestedClaudeStructuredThreadTitle(event.Conversation),
	} {
		if title := strings.TrimSpace(candidate); title != "" {
			return title
		}
	}
	return ""
}

func nestedClaudeStructuredThreadTitle(thread *claudeStructuredThread) string {
	if thread == nil {
		return ""
	}
	if title := strings.TrimSpace(thread.Title); title != "" {
		return title
	}
	return strings.TrimSpace(thread.Name)
}

func buildClaudeCodeArgs(prompt string, metadata map[string]string) []string {
	args := []string{"--print", "--output-format", "text"}
	if model := strings.TrimSpace(metadataValue(metadata, execdomain.LLMModelMetadataKey)); model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, prompt)
	return args
}

type preparedClaudeCodeInvocation struct {
	RuntimeHome string
	Env         []string
}

func prepareClaudeCodeInvocation(metadata map[string]string, workspaceDir string) (preparedClaudeCodeInvocation, error) {
	runtimeHome, err := resolveClaudeRuntimeHome(metadata, workspaceDir)
	if err != nil {
		return preparedClaudeCodeInvocation{}, err
	}
	if strings.TrimSpace(runtimeHome) == "" {
		return preparedClaudeCodeInvocation{}, errors.New("claude runtime home is required")
	}
	if err := os.MkdirAll(runtimeHome, 0o755); err != nil {
		return preparedClaudeCodeInvocation{}, fmt.Errorf("prepare claude runtime home: %w", err)
	}
	if err := prepareClaudeWorkspaceFiles(metadata, workspaceDir); err != nil {
		return preparedClaudeCodeInvocation{}, err
	}

	env := []string{
		"ANTHROPIC_API_KEY=" + strings.TrimSpace(metadataValue(metadata, execdomain.ClaudeAPIKeyMetadataKey)),
		"CLAUDE_CONFIG_DIR=" + runtimeHome,
	}
	if baseURL := strings.TrimSpace(metadataValue(metadata, execdomain.ClaudeBaseURLMetadataKey)); baseURL != "" {
		env = append(env, "ANTHROPIC_BASE_URL="+baseURL)
	}
	if model := strings.TrimSpace(metadataValue(metadata, execdomain.LLMModelMetadataKey)); model != "" {
		env = append(env, "ANTHROPIC_MODEL="+model)
	}
	return preparedClaudeCodeInvocation{RuntimeHome: runtimeHome, Env: env}, nil
}

func resolveClaudeRuntimeHome(metadata map[string]string, workspaceDir string) (string, error) {
	if value := strings.TrimSpace(metadataValue(metadata, execdomain.ClaudeConfigDirMetadataKey)); value != "" {
		return value, nil
	}
	if base, err := resolveCodexSessionWorkspaceBase(metadata); err == nil && strings.TrimSpace(base) != "" {
		return filepath.Join(base, claudeRuntimeHomeDirName), nil
	}
	trimmedWorkspace := strings.TrimSpace(workspaceDir)
	if trimmedWorkspace == "" {
		return "", nil
	}
	return filepath.Join(trimmedWorkspace, defaultWorkspaceRootDir, claudeRuntimeDirName, claudeRuntimeHomeDirName), nil
}

func prepareClaudeWorkspaceFiles(metadata map[string]string, workspaceDir string) error {
	workspaceDir = strings.TrimSpace(workspaceDir)
	if workspaceDir == "" {
		return errors.New("workspace dir is required")
	}
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		return fmt.Errorf("prepare claude workspace: %w", err)
	}

	rootInstructions := []string{}
	if runtimeContext := buildCodexRuntimeContext(metadata); runtimeContext != nil {
		if err := writeClaudeManagedFile(workspaceDir, claudeRuntimeRuntimePath, renderRuntimeContextMarkdown(runtimeContext)); err != nil {
			return err
		}
		rootInstructions = append(rootInstructions, "- Read `.alter0/claude-runtime/runtime.md` for session workspace, repository, and preview scope.")
	}
	if skillContext, err := parseSkillContext(metadata); err != nil {
		return err
	} else if skillContext != nil {
		materializedSkillContext, skillFiles, err := materializeClaudeSkillContextFiles(*skillContext)
		if err != nil {
			return err
		}
		skillContext = &materializedSkillContext
		for _, file := range skillFiles {
			if err := writeClaudeManagedRuntimeFile(workspaceDir, file); err != nil {
				return err
			}
		}
		if err := writeClaudeManagedFile(workspaceDir, claudeRuntimeSkillsPath, renderSkillContextMarkdown(*skillContext)); err != nil {
			return err
		}
		rootInstructions = append(rootInstructions, "- Read `.alter0/claude-runtime/skills.md` for resolved skill rules, reusable guides, parameters, and constraints.")
	}
	if memoryContext, err := parseMemoryContext(metadata); err != nil {
		return err
	} else if memoryContext != nil {
		for _, file := range renderMemoryContextFiles(*memoryContext) {
			relativePath := strings.Replace(file.RelativePath, codexRuntimeMemoryDir, claudeRuntimeMemoryDir, 1)
			if err := writeClaudeManagedFile(workspaceDir, relativePath, file.Content); err != nil {
				return err
			}
		}
		rootInstructions = append(rootInstructions, "- Read the files under `.alter0/claude-runtime/memory/` for injected memory content and recall snippets before acting.")
	}
	if len(rootInstructions) == 0 {
		rootInstructions = append(rootInstructions, "- Work only inside the current session workspace unless the user explicitly scopes another target.")
	}
	return writeClaudeManagedFile(workspaceDir, claudeRuntimeRootFileName, strings.Join(rootInstructions, "\n")+"\n")
}

func materializeClaudeSkillContextFiles(context execdomain.SkillContext) (execdomain.SkillContext, []runtimeconfig.ManagedFile, error) {
	updated, files, err := materializeCodexSkillContextFiles(context)
	if err != nil {
		return context, nil, err
	}
	for i := range updated.Skills {
		updated.Skills[i].FilePath = strings.Replace(updated.Skills[i].FilePath, ".alter0/codex-runtime/skills", ".alter0/claude-runtime/skills", 1)
	}
	for i := range files {
		files[i].RelativePath = strings.Replace(files[i].RelativePath, ".alter0/codex-runtime/skills", ".alter0/claude-runtime/skills", 1)
	}
	return updated, files, nil
}

func writeClaudeManagedRuntimeFile(workspaceDir string, file runtimeconfig.ManagedFile) error {
	mode := file.Mode
	if mode == 0 {
		mode = 0o644
	}
	return writeClaudeManagedFileWithMode(workspaceDir, file.RelativePath, file.Content, mode)
}

func writeClaudeManagedFile(workspaceDir string, relativePath string, content string) error {
	return writeClaudeManagedFileWithMode(workspaceDir, relativePath, content, 0o644)
}

func writeClaudeManagedFileWithMode(workspaceDir string, relativePath string, content string, mode os.FileMode) error {
	relativePath = strings.TrimSpace(relativePath)
	if relativePath == "" {
		return nil
	}
	if mode == 0 {
		mode = 0o644
	}
	path := filepath.Join(workspaceDir, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("prepare claude managed file dir %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		return fmt.Errorf("write claude managed file %s: %w", path, err)
	}
	return nil
}

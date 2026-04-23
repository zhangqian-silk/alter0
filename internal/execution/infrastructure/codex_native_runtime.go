package infrastructure

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"alter0/internal/codex/infrastructure/runtimeconfig"
	execdomain "alter0/internal/execution/domain"
)

const (
	codexRuntimeDirName      = "codex-runtime"
	codexRuntimeHomeDirName  = "codex-home"
	codexRuntimeSkillsPath   = ".alter0/codex-runtime/skills.md"
	codexRuntimeRuntimePath  = ".alter0/codex-runtime/runtime.md"
	codexRuntimeMemoryDir    = ".alter0/codex-runtime/memory"
	codexRuntimeRecallPath   = ".alter0/codex-runtime/memory/recall.md"
)

type preparedCodexInvocation struct {
	Prompt string
	Env    []string
}

func prepareCodexInvocation(prompt string, metadata map[string]string, workspaceDir string) (preparedCodexInvocation, error) {
	trimmedPrompt := strings.TrimSpace(prompt)
	if trimmedPrompt == "" {
		return preparedCodexInvocation{}, fmt.Errorf("content is required")
	}
	strategy := resolveCodexRuntimeStrategy(metadata)
	if strategy == execdomain.CodexRuntimeStrategyPlain {
		return preparedCodexInvocation{Prompt: trimmedPrompt}, nil
	}
	env, err := prepareCodexNativeRuntime(metadata, workspaceDir)
	if err != nil {
		return preparedCodexInvocation{}, err
	}
	return preparedCodexInvocation{
		Prompt: trimmedPrompt,
		Env:    env,
	}, nil
}

func resolveCodexRuntimeStrategy(metadata map[string]string) string {
	value := strings.ToLower(strings.TrimSpace(metadataValue(metadata, execdomain.CodexRuntimeStrategyMetadataKey)))
	switch value {
	case execdomain.CodexRuntimeStrategyPlain:
		return execdomain.CodexRuntimeStrategyPlain
	default:
		return execdomain.CodexRuntimeStrategyNative
	}
}

func prepareCodexNativeRuntime(metadata map[string]string, workspaceDir string) ([]string, error) {
	spec, err := buildCodexRuntimeSpec(metadata, workspaceDir)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(spec.RuntimeHome) == "" {
		return nil, nil
	}
	prepared, err := runtimeconfig.Prepare(spec)
	if err != nil {
		return nil, err
	}
	return prepared.Env, nil
}

func buildCodexRuntimeSpec(metadata map[string]string, workspaceDir string) (runtimeconfig.Spec, error) {
	runtimeHome, err := resolveCodexRuntimeHome(metadata, workspaceDir)
	if err != nil {
		return runtimeconfig.Spec{}, err
	}
	runtimeContext := buildCodexRuntimeContext(metadata)
	skillContext, err := parseSkillContext(metadata)
	if err != nil {
		return runtimeconfig.Spec{}, err
	}
	mcpContext, err := parseMCPContext(metadata)
	if err != nil {
		return runtimeconfig.Spec{}, err
	}
	memoryContext, err := parseMemoryContext(metadata)
	if err != nil {
		return runtimeconfig.Spec{}, err
	}
	files := make([]runtimeconfig.ManagedFile, 0, 4)
	instructions := make([]string, 0, 4)
	if runtimeContext != nil {
		files = append(files, runtimeconfig.ManagedFile{
			RelativePath: codexRuntimeRuntimePath,
			Content:      renderRuntimeContextMarkdown(runtimeContext),
		})
		instructions = append(instructions, "- Read `.alter0/codex-runtime/runtime.md` for session workspace, repository, and preview scope.")
	}
	if skillContext != nil {
		files = append(files, runtimeconfig.ManagedFile{
			RelativePath: codexRuntimeSkillsPath,
			Content:      renderSkillContextMarkdown(*skillContext),
		})
		instructions = append(instructions, "- Read `.alter0/codex-runtime/skills.md` for resolved skill rules, reusable guides, parameters, and constraints.")
	}
	if memoryContext != nil {
		files = append(files, renderMemoryContextFiles(*memoryContext)...)
		instructions = append(instructions, "- Read the files under `.alter0/codex-runtime/memory/` for injected memory content and recall snippets before acting.")
	}

	mcpServers := buildManagedMCPServers(mcpContext)
	if len(mcpServers) > 0 {
		instructions = append(instructions, "- Native MCP servers for this run are configured in Codex runtime config. Use only the tools exposed there.")
	}

	rootInstructions := strings.Join(instructions, "\n")
	return runtimeconfig.Spec{
		RuntimeHome:      runtimeHome,
		WorkspaceDir:     workspaceDir,
		MCPServers:       mcpServers,
		ManagedFiles:     files,
		RootInstructions: rootInstructions,
	}, nil
}

func resolveCodexRuntimeHome(metadata map[string]string, workspaceDir string) (string, error) {
	if base, err := resolveCodexSessionWorkspaceBase(metadata); err == nil && strings.TrimSpace(base) != "" {
		return filepath.Join(base, codexRuntimeHomeDirName), nil
	}
	trimmedWorkspace := strings.TrimSpace(workspaceDir)
	if trimmedWorkspace == "" {
		return "", nil
	}
	return filepath.Join(trimmedWorkspace, defaultWorkspaceRootDir, codexRuntimeDirName, codexRuntimeHomeDirName), nil
}

func parseSkillContext(metadata map[string]string) (*execdomain.SkillContext, error) {
	raw := strings.TrimSpace(metadataValue(metadata, execdomain.SkillContextMetadataKey))
	if raw == "" {
		return nil, nil
	}
	var context execdomain.SkillContext
	if err := json.Unmarshal([]byte(raw), &context); err != nil {
		return nil, fmt.Errorf("invalid skill context metadata: %w", err)
	}
	if len(context.Skills) == 0 && len(context.ResolvedParameters) == 0 && len(context.Conflicts) == 0 {
		return nil, nil
	}
	return &context, nil
}

func parseMCPContext(metadata map[string]string) (*execdomain.MCPContext, error) {
	raw := strings.TrimSpace(metadataValue(metadata, execdomain.MCPContextMetadataKey))
	if raw == "" {
		return nil, nil
	}
	var context execdomain.MCPContext
	if err := json.Unmarshal([]byte(raw), &context); err != nil {
		return nil, fmt.Errorf("invalid mcp context metadata: %w", err)
	}
	if len(context.Servers) == 0 {
		return nil, nil
	}
	return &context, nil
}

func parseMemoryContext(metadata map[string]string) (*execdomain.MemoryContext, error) {
	raw := strings.TrimSpace(metadataValue(metadata, execdomain.MemoryContextMetadataKey))
	if raw == "" {
		return nil, nil
	}
	var context execdomain.MemoryContext
	if err := json.Unmarshal([]byte(raw), &context); err != nil {
		return nil, fmt.Errorf("invalid memory context metadata: %w", err)
	}
	if len(context.Files) == 0 && len(context.Recall) == 0 {
		return nil, nil
	}
	return &context, nil
}

func buildManagedMCPServers(context *execdomain.MCPContext) []runtimeconfig.MCPServer {
	if context == nil || len(context.Servers) == 0 {
		return nil
	}
	servers := make([]runtimeconfig.MCPServer, 0, len(context.Servers))
	for _, server := range context.Servers {
		item := runtimeconfig.MCPServer{
			Name:              server.ID,
			Transport:         server.Transport,
			Command:           server.Command,
			Args:              append([]string(nil), server.Args...),
			Env:               cloneStringMap(server.Env),
			URL:               server.URL,
			Headers:           cloneStringMap(server.Headers),
			EnabledTools:      append([]string(nil), server.ToolWhitelist...),
			StartupTimeoutSec: server.TimeoutMS / 1000,
			ToolTimeoutSec:    server.TimeoutMS / 1000,
			Required:          !server.FailureIsolation,
		}
		servers = append(servers, item)
	}
	return servers
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func renderRuntimeContextMarkdown(context *execdomain.RuntimeContext) string {
	lines := []string{"# Runtime Context", ""}
	if strings.TrimSpace(context.SessionID) != "" {
		lines = append(lines, "- session_id: "+context.SessionID)
	}
	if strings.TrimSpace(context.MessageID) != "" {
		lines = append(lines, "- message_id: "+context.MessageID)
	}
	if strings.TrimSpace(context.TraceID) != "" {
		lines = append(lines, "- trace_id: "+context.TraceID)
	}
	if context.Workspace != nil {
		lines = append(lines, "", "## Workspace")
		if strings.TrimSpace(context.Workspace.Mode) != "" {
			lines = append(lines, "- mode: "+context.Workspace.Mode)
		}
		if strings.TrimSpace(context.Workspace.SessionPath) != "" {
			lines = append(lines, "- session_path: "+context.Workspace.SessionPath)
		}
		if strings.TrimSpace(context.Workspace.TaskPath) != "" {
			lines = append(lines, "- task_path: "+context.Workspace.TaskPath)
		}
		if strings.TrimSpace(context.Workspace.RepositoryPath) != "" {
			lines = append(lines, "- repository_path: "+context.Workspace.RepositoryPath)
		}
	}
	if context.Repository != nil {
		lines = append(lines, "", "## Repository")
		if strings.TrimSpace(context.Repository.SourcePath) != "" {
			lines = append(lines, "- source_path: "+context.Repository.SourcePath)
		}
		if strings.TrimSpace(context.Repository.RemoteURL) != "" {
			lines = append(lines, "- remote_url: "+context.Repository.RemoteURL)
		}
		if strings.TrimSpace(context.Repository.ActiveBranch) != "" {
			lines = append(lines, "- active_branch: "+context.Repository.ActiveBranch)
		}
		if strings.TrimSpace(context.Repository.DefaultBranch) != "" {
			lines = append(lines, "- default_branch: "+context.Repository.DefaultBranch)
		}
	}
	if context.Preview != nil {
		lines = append(lines, "", "## Preview")
		if strings.TrimSpace(context.Preview.URL) != "" {
			lines = append(lines, "- url: "+context.Preview.URL)
		}
		if context.Preview.RequiredOnCompletion {
			lines = append(lines, "- required_on_completion: true")
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func renderSkillContextMarkdown(context execdomain.SkillContext) string {
	lines := []string{"# Skill Context", ""}
	if len(context.ResolvedParameters) > 0 {
		lines = append(lines, "## Resolved Parameters")
		keys := make([]string, 0, len(context.ResolvedParameters))
		for key := range context.ResolvedParameters {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			lines = append(lines, "- "+key+": "+context.ResolvedParameters[key])
		}
		lines = append(lines, "")
	}
	for _, skill := range context.Skills {
		lines = append(lines, "## "+skill.Name, "")
		lines = append(lines, "- id: "+skill.ID)
		if strings.TrimSpace(skill.Description) != "" {
			lines = append(lines, "- description: "+skill.Description)
		}
		if strings.TrimSpace(skill.FilePath) != "" {
			lines = append(lines, "- file_path: "+skill.FilePath)
		}
		if strings.TrimSpace(skill.Guide) != "" {
			lines = append(lines, "", "### Guide", "", strings.TrimSpace(skill.Guide), "")
		}
		if len(skill.Constraints) > 0 {
			lines = append(lines, "### Constraints")
			for _, constraint := range skill.Constraints {
				lines = append(lines, "- "+constraint)
			}
			lines = append(lines, "")
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func renderMemoryContextFiles(context execdomain.MemoryContext) []runtimeconfig.ManagedFile {
	files := make([]runtimeconfig.ManagedFile, 0, len(context.Files)+1)
	for _, file := range context.Files {
		name := strings.TrimSpace(file.Selection)
		if name == "" {
			name = strings.TrimSpace(file.ID)
		}
		if name == "" {
			continue
		}
		relativePath := filepath.ToSlash(filepath.Join(codexRuntimeMemoryDir, sanitizeWorkspaceSegment(name)+".md"))
		content := file.Content
		if strings.TrimSpace(content) == "" {
			content = "# " + fallbackMemoryTitle(file.Title, file.Selection, file.ID) + "\n\n(no inline content captured)\n"
		}
		files = append(files, runtimeconfig.ManagedFile{
			RelativePath: relativePath,
			Content:      content,
		})
	}
	if len(context.Recall) > 0 {
		lines := []string{"# Memory Recall", ""}
		for _, item := range context.Recall {
			lines = append(lines, "- "+strings.TrimSpace(item.Title)+": "+strings.TrimSpace(item.Snippet))
		}
		files = append(files, runtimeconfig.ManagedFile{
			RelativePath: codexRuntimeRecallPath,
			Content:      strings.Join(lines, "\n") + "\n",
		})
	}
	return files
}

func fallbackMemoryTitle(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return "Memory"
}

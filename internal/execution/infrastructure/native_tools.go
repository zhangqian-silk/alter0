package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	llmdomain "alter0/internal/llm/domain"
)

const (
	toolListDir   = "list_dir"
	toolRead      = "read"
	toolWrite     = "write"
	toolEdit      = "edit"
	toolBash      = "bash"
	toolCodexExec = "codex_exec"
	toolComplete  = "complete"

	toolBaseRepo      = "repo"
	toolBaseWorkspace = "workspace"

	defaultReadLimitBytes     = 32 * 1024
	maxReadLimitBytes         = 128 * 1024
	defaultListDirEntryLimit  = 200
	maxListDirEntryLimit      = 1000
	defaultCommandTimeoutSecs = 20
	maxCommandTimeoutSecs     = 120
	maxCommandOutputBytes     = 32 * 1024
)

var defaultNativeToolIDs = []string{
	toolListDir,
	toolRead,
	toolWrite,
	toolEdit,
	toolBash,
}

var defaultAgentToolIDs = []string{
	toolListDir,
	toolRead,
	toolWrite,
	toolEdit,
	toolBash,
	toolCodexExec,
}

type nativeToolRuntime struct {
	repoRoot     string
	workspaceDir string
}

type nativeToolExecutor struct {
	runtime nativeToolRuntime
}

func newNativeToolExecutor(metadata map[string]string) (*nativeToolExecutor, error) {
	repoRoot, err := resolveToolRepoRoot()
	if err != nil {
		return nil, err
	}
	workspaceDir, err := resolveCodexWorkspace(metadata)
	if err != nil {
		return nil, err
	}
	return &nativeToolExecutor{
		runtime: nativeToolRuntime{
			repoRoot:     repoRoot,
			workspaceDir: workspaceDir,
		},
	}, nil
}

func buildNativeTools(metadata map[string]string, defaults []string) []llmdomain.Tool {
	allowed := parseSelectedToolIDs(metadata, defaults)
	items := make([]llmdomain.Tool, 0, len(allowed))
	addTool := func(tool llmdomain.Tool) {
		if _, ok := allowed[tool.Name]; !ok {
			return
		}
		items = append(items, tool)
	}

	addTool(llmdomain.Tool{
		Name:        toolListDir,
		Description: "List files and directories under the repo root or the session workspace.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Directory path. Relative paths are resolved from repo root unless base=workspace.",
				},
				"base": map[string]interface{}{
					"type":        "string",
					"enum":        []string{toolBaseRepo, toolBaseWorkspace},
					"description": "Path base. Defaults to repo.",
				},
				"recursive": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether to recurse into subdirectories.",
				},
				"max_entries": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of entries to return. Defaults to 200.",
				},
			},
		},
	})
	addTool(llmdomain.Tool{
		Name:        toolRead,
		Description: "Read a text file from the repo root or the session workspace.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "File path. Relative paths are resolved from repo root unless base=workspace.",
				},
				"base": map[string]interface{}{
					"type":        "string",
					"enum":        []string{toolBaseRepo, toolBaseWorkspace},
					"description": "Path base. Defaults to repo.",
				},
				"offset": map[string]interface{}{
					"type":        "integer",
					"description": "Byte offset to start reading from. Defaults to 0.",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum bytes to return. Defaults to 32768.",
				},
			},
			"required": []string{"path"},
		},
	})
	addTool(llmdomain.Tool{
		Name:        toolWrite,
		Description: "Write or append text content to a file under the repo root or the session workspace.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Target file path. Relative paths are resolved from repo root unless base=workspace.",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Text content to write.",
				},
				"base": map[string]interface{}{
					"type":        "string",
					"enum":        []string{toolBaseRepo, toolBaseWorkspace},
					"description": "Path base. Defaults to repo.",
				},
				"mode": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"overwrite", "append"},
					"description": "Write mode. Defaults to overwrite.",
				},
				"create_dirs": map[string]interface{}{
					"type":        "boolean",
					"description": "Create parent directories when missing. Defaults to true.",
				},
			},
			"required": []string{"path", "content"},
		},
	})
	addTool(llmdomain.Tool{
		Name:        toolEdit,
		Description: "Replace existing text in a file under the repo root or the session workspace.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Target file path. Relative paths are resolved from repo root unless base=workspace.",
				},
				"old_text": map[string]interface{}{
					"type":        "string",
					"description": "Exact text to replace.",
				},
				"new_text": map[string]interface{}{
					"type":        "string",
					"description": "Replacement text.",
				},
				"base": map[string]interface{}{
					"type":        "string",
					"enum":        []string{toolBaseRepo, toolBaseWorkspace},
					"description": "Path base. Defaults to repo.",
				},
				"replace_all": map[string]interface{}{
					"type":        "boolean",
					"description": "Replace all matches instead of requiring a single exact match.",
				},
			},
			"required": []string{"path", "old_text", "new_text"},
		},
	})
	addTool(llmdomain.Tool{
		Name:        toolBash,
		Description: "Run a shell command in the repo root or a selected working directory and return stdout, stderr, and exit code.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "Shell command to execute.",
				},
				"working_dir": map[string]interface{}{
					"type":        "string",
					"description": "Optional working directory. Relative paths are resolved from repo root unless base=workspace.",
				},
				"base": map[string]interface{}{
					"type":        "string",
					"enum":        []string{toolBaseRepo, toolBaseWorkspace},
					"description": "Path base for working_dir. Defaults to repo.",
				},
				"timeout_seconds": map[string]interface{}{
					"type":        "integer",
					"description": "Command timeout in seconds. Defaults to 20.",
				},
			},
			"required": []string{"command"},
		},
	})
	return items
}

func parseSelectedToolIDs(metadata map[string]string, defaults []string) map[string]struct{} {
	raw, exists := lookupMetadata(metadata, "alter0.agent.tools")
	if !exists {
		return normalizeToolIDSet(defaults)
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]struct{}{}
	}

	items := []string{}
	if strings.HasPrefix(raw, "[") {
		if err := json.Unmarshal([]byte(raw), &items); err != nil {
			return map[string]struct{}{}
		}
	} else {
		items = strings.Split(raw, ",")
	}
	return normalizeToolIDSet(items)
}

func normalizeToolIDSet(values []string) map[string]struct{} {
	allowed := map[string]struct{}{}
	for _, value := range values {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case toolListDir, toolRead, toolWrite, toolEdit, toolBash, toolCodexExec:
			allowed[strings.ToLower(strings.TrimSpace(value))] = struct{}{}
		}
	}
	return allowed
}

func lookupMetadata(metadata map[string]string, key string) (string, bool) {
	if len(metadata) == 0 {
		return "", false
	}
	value, ok := metadata[key]
	return value, ok
}

func (e *nativeToolExecutor) Execute(ctx context.Context, toolCall llmdomain.ToolCall) (*llmdomain.ToolResult, error) {
	if e == nil {
		return nil, errors.New("native tool executor unavailable")
	}
	name := strings.ToLower(strings.TrimSpace(toolCall.Name))
	switch name {
	case toolListDir:
		return e.listDir(toolCall)
	case toolRead:
		return e.readFile(toolCall)
	case toolWrite:
		return e.writeFile(toolCall)
	case toolEdit:
		return e.editFile(toolCall)
	case toolBash:
		return e.runCommand(ctx, toolCall)
	default:
		return nil, fmt.Errorf("unsupported native tool: %s", toolCall.Name)
	}
}

func (e *nativeToolExecutor) listDir(toolCall llmdomain.ToolCall) (*llmdomain.ToolResult, error) {
	payload := struct {
		Path       string `json:"path"`
		Base       string `json:"base"`
		Recursive  bool   `json:"recursive"`
		MaxEntries int    `json:"max_entries"`
	}{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(toolCall.Arguments)), &payload); err != nil {
		return nil, fmt.Errorf("parse %s arguments: %w", toolListDir, err)
	}
	targetPath := payload.Path
	if strings.TrimSpace(targetPath) == "" {
		targetPath = "."
	}
	resolvedPath, err := e.runtime.resolvePath(targetPath, payload.Base)
	if err != nil {
		return toolErrorResult(toolCall, err), nil
	}
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return toolErrorResult(toolCall, err), nil
	}
	if !info.IsDir() {
		return toolErrorResult(toolCall, errors.New("path is not a directory")), nil
	}

	limit := payload.MaxEntries
	if limit <= 0 {
		limit = defaultListDirEntryLimit
	}
	if limit > maxListDirEntryLimit {
		limit = maxListDirEntryLimit
	}

	type entry struct {
		Path  string `json:"path"`
		Type  string `json:"type"`
		Size  int64  `json:"size,omitempty"`
		Depth int    `json:"depth"`
	}

	entries := make([]entry, 0, limit)
	if payload.Recursive {
		walkErr := filepath.WalkDir(resolvedPath, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if path == resolvedPath {
				return nil
			}
			if len(entries) >= limit {
				return fs.SkipAll
			}
			kind := "file"
			size := int64(0)
			if d.IsDir() {
				kind = "dir"
			} else if info, err := d.Info(); err == nil {
				size = info.Size()
			}
			entries = append(entries, entry{
				Path:  e.runtime.displayPath(path),
				Type:  kind,
				Size:  size,
				Depth: strings.Count(filepath.ToSlash(strings.TrimPrefix(path, resolvedPath)), "/"),
			})
			return nil
		})
		if walkErr != nil && !errors.Is(walkErr, fs.SkipAll) {
			return toolErrorResult(toolCall, walkErr), nil
		}
	} else {
		children, err := os.ReadDir(resolvedPath)
		if err != nil {
			return toolErrorResult(toolCall, err), nil
		}
		sort.SliceStable(children, func(i, j int) bool {
			return strings.ToLower(children[i].Name()) < strings.ToLower(children[j].Name())
		})
		for _, child := range children {
			if len(entries) >= limit {
				break
			}
			path := filepath.Join(resolvedPath, child.Name())
			kind := "file"
			size := int64(0)
			if child.IsDir() {
				kind = "dir"
			} else if info, err := child.Info(); err == nil {
				size = info.Size()
			}
			entries = append(entries, entry{
				Path:  e.runtime.displayPath(path),
				Type:  kind,
				Size:  size,
				Depth: 0,
			})
		}
	}

	return toolSuccessResult(toolCall, map[string]any{
		"path":        e.runtime.displayPath(resolvedPath),
		"base":        normalizeToolBase(payload.Base),
		"recursive":   payload.Recursive,
		"entry_count": len(entries),
		"entries":     entries,
	}), nil
}

func (e *nativeToolExecutor) readFile(toolCall llmdomain.ToolCall) (*llmdomain.ToolResult, error) {
	payload := struct {
		Path   string `json:"path"`
		Base   string `json:"base"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
	}{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(toolCall.Arguments)), &payload); err != nil {
		return nil, fmt.Errorf("parse %s arguments: %w", toolRead, err)
	}
	resolvedPath, err := e.runtime.resolvePath(payload.Path, payload.Base)
	if err != nil {
		return toolErrorResult(toolCall, err), nil
	}
	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return toolErrorResult(toolCall, err), nil
	}
	if payload.Offset < 0 {
		payload.Offset = 0
	}
	if payload.Offset > len(data) {
		payload.Offset = len(data)
	}
	limit := payload.Limit
	if limit <= 0 {
		limit = defaultReadLimitBytes
	}
	if limit > maxReadLimitBytes {
		limit = maxReadLimitBytes
	}
	end := payload.Offset + limit
	if end > len(data) {
		end = len(data)
	}
	content := string(data[payload.Offset:end])
	return toolSuccessResult(toolCall, map[string]any{
		"path":           e.runtime.displayPath(resolvedPath),
		"base":           normalizeToolBase(payload.Base),
		"size_bytes":     len(data),
		"offset":         payload.Offset,
		"returned_bytes": len(content),
		"truncated":      end < len(data),
		"content":        content,
	}), nil
}

func (e *nativeToolExecutor) writeFile(toolCall llmdomain.ToolCall) (*llmdomain.ToolResult, error) {
	payload := struct {
		Path       string `json:"path"`
		Content    string `json:"content"`
		Base       string `json:"base"`
		Mode       string `json:"mode"`
		CreateDirs *bool  `json:"create_dirs"`
	}{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(toolCall.Arguments)), &payload); err != nil {
		return nil, fmt.Errorf("parse %s arguments: %w", toolWrite, err)
	}
	resolvedPath, err := e.runtime.resolvePath(payload.Path, payload.Base)
	if err != nil {
		return toolErrorResult(toolCall, err), nil
	}
	createDirs := true
	if payload.CreateDirs != nil {
		createDirs = *payload.CreateDirs
	}
	if createDirs {
		if err := os.MkdirAll(filepath.Dir(resolvedPath), 0o755); err != nil {
			return toolErrorResult(toolCall, err), nil
		}
	}

	mode := strings.ToLower(strings.TrimSpace(payload.Mode))
	if mode == "" {
		mode = "overwrite"
	}
	switch mode {
	case "overwrite":
		if err := os.WriteFile(resolvedPath, []byte(payload.Content), 0o644); err != nil {
			return toolErrorResult(toolCall, err), nil
		}
	case "append":
		file, err := os.OpenFile(resolvedPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return toolErrorResult(toolCall, err), nil
		}
		defer file.Close()
		if _, err := file.WriteString(payload.Content); err != nil {
			return toolErrorResult(toolCall, err), nil
		}
	default:
		return toolErrorResult(toolCall, fmt.Errorf("unsupported write mode: %s", payload.Mode)), nil
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		return toolErrorResult(toolCall, err), nil
	}
	return toolSuccessResult(toolCall, map[string]any{
		"path":       e.runtime.displayPath(resolvedPath),
		"base":       normalizeToolBase(payload.Base),
		"mode":       mode,
		"size_bytes": info.Size(),
	}), nil
}

func (e *nativeToolExecutor) editFile(toolCall llmdomain.ToolCall) (*llmdomain.ToolResult, error) {
	payload := struct {
		Path       string `json:"path"`
		OldText    string `json:"old_text"`
		NewText    string `json:"new_text"`
		Base       string `json:"base"`
		ReplaceAll bool   `json:"replace_all"`
	}{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(toolCall.Arguments)), &payload); err != nil {
		return nil, fmt.Errorf("parse %s arguments: %w", toolEdit, err)
	}
	if payload.OldText == "" {
		return toolErrorResult(toolCall, errors.New("old_text is required")), nil
	}
	resolvedPath, err := e.runtime.resolvePath(payload.Path, payload.Base)
	if err != nil {
		return toolErrorResult(toolCall, err), nil
	}
	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return toolErrorResult(toolCall, err), nil
	}
	content := string(data)
	matchCount := strings.Count(content, payload.OldText)
	if matchCount == 0 {
		return toolErrorResult(toolCall, errors.New("old_text not found")), nil
	}
	if !payload.ReplaceAll && matchCount != 1 {
		return toolErrorResult(toolCall, fmt.Errorf("old_text matched %d times; set replace_all=true to replace all matches", matchCount)), nil
	}
	nextContent := strings.Replace(content, payload.OldText, payload.NewText, conditionalReplaceCount(payload.ReplaceAll))
	if err := os.WriteFile(resolvedPath, []byte(nextContent), 0o644); err != nil {
		return toolErrorResult(toolCall, err), nil
	}
	return toolSuccessResult(toolCall, map[string]any{
		"path":         e.runtime.displayPath(resolvedPath),
		"base":         normalizeToolBase(payload.Base),
		"replacements": conditionalReplacementReportCount(payload.ReplaceAll, matchCount),
		"size_bytes":   len(nextContent),
	}), nil
}

func (e *nativeToolExecutor) runCommand(ctx context.Context, toolCall llmdomain.ToolCall) (*llmdomain.ToolResult, error) {
	payload := struct {
		Command        string `json:"command"`
		WorkingDir     string `json:"working_dir"`
		Base           string `json:"base"`
		TimeoutSeconds int    `json:"timeout_seconds"`
	}{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(toolCall.Arguments)), &payload); err != nil {
		return nil, fmt.Errorf("parse %s arguments: %w", toolBash, err)
	}
	commandText := strings.TrimSpace(payload.Command)
	if commandText == "" {
		return toolErrorResult(toolCall, errors.New("command is required")), nil
	}

	workingDir := e.runtime.repoRoot
	if strings.TrimSpace(payload.WorkingDir) != "" {
		resolvedPath, err := e.runtime.resolvePath(payload.WorkingDir, payload.Base)
		if err != nil {
			return toolErrorResult(toolCall, err), nil
		}
		workingDir = resolvedPath
	}
	timeoutSeconds := payload.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = defaultCommandTimeoutSecs
	}
	if timeoutSeconds > maxCommandTimeoutSecs {
		timeoutSeconds = maxCommandTimeoutSecs
	}
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	cmd := buildShellCommand(runCtx, commandText)
	cmd.Dir = workingDir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	if err != nil && exitCode == 0 {
		exitCode = 1
	}

	success := err == nil && exitCode == 0
	result := map[string]any{
		"command":         commandText,
		"working_dir":     e.runtime.displayPath(workingDir),
		"timeout_seconds": timeoutSeconds,
		"success":         success,
		"exit_code":       exitCode,
		"stdout":          truncateToolText(stdout.String(), maxCommandOutputBytes),
		"stderr":          truncateToolText(stderr.String(), maxCommandOutputBytes),
	}
	if err != nil {
		result["error"] = err.Error()
	}
	return toolResult(toolCall, result, !success), nil
}

func buildShellCommand(ctx context.Context, command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-Command", command)
	}
	return exec.CommandContext(ctx, "/bin/sh", "-lc", command)
}

func resolveToolRepoRoot() (string, error) {
	root, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve tool repo root: %w", err)
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve tool repo root: %w", err)
	}
	return absolute, nil
}

func (r nativeToolRuntime) resolvePath(rawPath string, base string) (string, error) {
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		return "", errors.New("path is required")
	}
	var candidate string
	if filepath.IsAbs(trimmed) {
		candidate = trimmed
	} else {
		root := r.baseRoot(base)
		candidate = filepath.Join(root, filepath.FromSlash(trimmed))
	}
	absolute, err := filepath.Abs(candidate)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	if !r.isAllowedPath(absolute) {
		return "", errors.New("path is outside allowed roots")
	}
	return absolute, nil
}

func (r nativeToolRuntime) baseRoot(base string) string {
	if normalizeToolBase(base) == toolBaseWorkspace && strings.TrimSpace(r.workspaceDir) != "" {
		return r.workspaceDir
	}
	return r.repoRoot
}

func (r nativeToolRuntime) isAllowedPath(path string) bool {
	if isWithinRoot(path, r.repoRoot) {
		return true
	}
	if strings.TrimSpace(r.workspaceDir) != "" && isWithinRoot(path, r.workspaceDir) {
		return true
	}
	return false
}

func (r nativeToolRuntime) displayPath(path string) string {
	if rel, ok := relativeWithin(path, r.repoRoot); ok {
		return filepath.ToSlash(rel)
	}
	if rel, ok := relativeWithin(path, r.workspaceDir); ok {
		return "workspace/" + filepath.ToSlash(rel)
	}
	return filepath.ToSlash(path)
}

func isWithinRoot(path string, root string) bool {
	_, ok := relativeWithin(path, root)
	return ok
}

func relativeWithin(path string, root string) (string, bool) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", false
	}
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return "", false
	}
	if relative == "." {
		return ".", true
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", false
	}
	return relative, true
}

func normalizeToolBase(base string) string {
	if strings.EqualFold(strings.TrimSpace(base), toolBaseWorkspace) {
		return toolBaseWorkspace
	}
	return toolBaseRepo
}

func conditionalReplaceCount(replaceAll bool) int {
	if replaceAll {
		return -1
	}
	return 1
}

func conditionalReplacementReportCount(replaceAll bool, matchCount int) int {
	if replaceAll {
		return matchCount
	}
	return 1
}

func truncateToolText(value string, limit int) string {
	normalized := strings.ReplaceAll(value, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	if limit <= 0 || len(normalized) <= limit {
		return normalized
	}
	if limit <= 3 {
		return normalized[:limit]
	}
	return normalized[:limit-3] + "..."
}

func toolSuccessResult(toolCall llmdomain.ToolCall, payload map[string]any) *llmdomain.ToolResult {
	return toolResult(toolCall, payload, false)
}

func toolErrorResult(toolCall llmdomain.ToolCall, err error) *llmdomain.ToolResult {
	return toolResult(toolCall, map[string]any{
		"success": false,
		"error":   err.Error(),
	}, true)
}

func toolResult(toolCall llmdomain.ToolCall, payload map[string]any, isError bool) *llmdomain.ToolResult {
	encoded, err := json.Marshal(payload)
	if err != nil {
		encoded = []byte(`{"success":false,"error":"failed to encode tool result"}`)
		isError = true
	}
	return &llmdomain.ToolResult{
		ToolCallID: toolCall.ID,
		Name:       toolCall.Name,
		Result:     string(encoded),
		IsError:    isError,
	}
}

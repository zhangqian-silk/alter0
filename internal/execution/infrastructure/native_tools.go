package infrastructure

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	llmdomain "alter0/internal/llm/domain"
)

const (
	toolCodexExec         = "codex_exec"
	toolSearchMemory      = "search_memory"
	toolReadMemory        = "read_memory"
	toolWriteMemory       = "write_memory"
	toolDelegateAgent     = "delegate_agent"
	toolDeployTestService = "deploy_test_service"
	toolComplete          = "complete"

	defaultReadLimitBytes = 32 * 1024
	maxReadLimitBytes     = 128 * 1024
)

var defaultAgentToolIDs = []string{
	toolCodexExec,
	toolSearchMemory,
	toolReadMemory,
	toolWriteMemory,
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
		case toolCodexExec, toolSearchMemory, toolReadMemory, toolWriteMemory, toolDelegateAgent, toolDeployTestService:
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

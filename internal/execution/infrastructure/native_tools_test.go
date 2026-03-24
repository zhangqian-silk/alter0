package infrastructure

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	llmdomain "alter0/internal/llm/domain"
)

func TestNativeToolExecutorWorkspaceFlow(t *testing.T) {
	metadata := testRuntimeMetadata()
	metadata["task_id"] = "native-tools-smoke"
	workspaceDir, err := resolveCodexWorkspace(metadata)
	if err != nil {
		t.Fatalf("resolve workspace: %v", err)
	}
	defer os.RemoveAll(workspaceDir)

	executor, err := newNativeToolExecutor(metadata)
	if err != nil {
		t.Fatalf("newNativeToolExecutor() error = %v", err)
	}

	writeResult, err := executor.Execute(context.Background(), llmdomain.ToolCall{
		ID:   "write-1",
		Name: toolWrite,
		Arguments: `{
			"path":"notes.txt",
			"base":"workspace",
			"content":"hello native tools"
		}`,
	})
	if err != nil {
		t.Fatalf("write tool error = %v", err)
	}
	if writeResult.IsError {
		t.Fatalf("write tool returned error result: %s", writeResult.Result)
	}

	readResult, err := executor.Execute(context.Background(), llmdomain.ToolCall{
		ID:   "read-1",
		Name: toolRead,
		Arguments: `{
			"path":"notes.txt",
			"base":"workspace"
		}`,
	})
	if err != nil {
		t.Fatalf("read tool error = %v", err)
	}
	readPayload := decodeToolPayload(t, readResult)
	if !strings.Contains(stringValue(readPayload["content"]), "hello native tools") {
		t.Fatalf("unexpected read payload: %v", readPayload)
	}

	editResult, err := executor.Execute(context.Background(), llmdomain.ToolCall{
		ID:   "edit-1",
		Name: toolEdit,
		Arguments: `{
			"path":"notes.txt",
			"base":"workspace",
			"old_text":"hello native tools",
			"new_text":"hello edited tools"
		}`,
	})
	if err != nil {
		t.Fatalf("edit tool error = %v", err)
	}
	if editResult.IsError {
		t.Fatalf("edit tool returned error result: %s", editResult.Result)
	}

	listResult, err := executor.Execute(context.Background(), llmdomain.ToolCall{
		ID:   "list-1",
		Name: toolListDir,
		Arguments: `{
			"path":".",
			"base":"workspace"
		}`,
	})
	if err != nil {
		t.Fatalf("list_dir tool error = %v", err)
	}
	listPayload := decodeToolPayload(t, listResult)
	if !strings.Contains(listResult.Result, "notes.txt") {
		t.Fatalf("expected notes.txt in list payload: %v", listPayload)
	}

	commandResult, err := executor.Execute(context.Background(), llmdomain.ToolCall{
		ID:   "bash-1",
		Name: toolBash,
		Arguments: `{
			"command":"echo NATIVE_TOOL_OK",
			"working_dir":".",
			"base":"workspace",
			"timeout_seconds":10
		}`,
	})
	if err != nil {
		t.Fatalf("bash tool error = %v", err)
	}
	commandPayload := decodeToolPayload(t, commandResult)
	if !boolValue(commandPayload["success"]) {
		t.Fatalf("expected command success: %v", commandPayload)
	}
	if !strings.Contains(stringValue(commandPayload["stdout"]), "NATIVE_TOOL_OK") {
		t.Fatalf("unexpected command stdout: %v", commandPayload)
	}

	updatedData, err := os.ReadFile(workspaceDir + string(os.PathSeparator) + "notes.txt")
	if err != nil {
		t.Fatalf("read updated file from workspace: %v", err)
	}
	if !strings.Contains(string(updatedData), "hello edited tools") {
		t.Fatalf("unexpected file content: %q", string(updatedData))
	}
}

func decodeToolPayload(t *testing.T, result *llmdomain.ToolResult) map[string]any {
	t.Helper()
	if result == nil {
		t.Fatal("tool result is nil")
	}
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(result.Result), &payload); err != nil {
		t.Fatalf("decode tool payload: %v", err)
	}
	return payload
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func boolValue(value any) bool {
	ok, _ := value.(bool)
	return ok
}

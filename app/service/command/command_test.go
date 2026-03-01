package command

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"alter0/app/pkg/types"
	servicestore "alter0/app/service/store"
	servicetask "alter0/app/service/task"
)

func TestAuthorizeCommand_AdminOnlyCommands(t *testing.T) {
	exec := NewExecutor(nil, nil, []string{"admin_user"})

	if err := exec.authorizeCommand("guest", []string{"executor"}); err == nil {
		t.Fatalf("expected /executor to require admin")
	}
	if err := exec.authorizeCommand("guest", []string{"config", "set", "executor.name", "codex"}); err == nil {
		t.Fatalf("expected /config set to require admin")
	}
	if err := exec.authorizeCommand("guest", []string{"config", "get"}); err != nil {
		t.Fatalf("expected /config get to be allowed: %v", err)
	}
	if err := exec.authorizeCommand("admin_user", []string{"executor"}); err != nil {
		t.Fatalf("expected admin to pass /executor check: %v", err)
	}
}

func TestSetAdminUsers_TrimsAndDedupes(t *testing.T) {
	exec := NewExecutor(nil, nil, nil)
	exec.SetAdminUsers([]string{"  alice ", "alice", "", "bob"})

	if !exec.isAdminUser("alice") {
		t.Fatalf("alice should be admin")
	}
	if !exec.isAdminUser("bob") {
		t.Fatalf("bob should be admin")
	}
	if exec.isAdminUser("carol") {
		t.Fatalf("carol should not be admin")
	}
}

func TestFormatAuditCommandLine_DefaultsAndReason(t *testing.T) {
	line := formatAuditCommandLine("", "", "", "config set executor.name codex", "deny", "permission denied")
	expected := "[AUDIT] user=anonymous channel=unknown request=n/a decision=deny command=\"config set executor.name codex\" reason=\"permission denied\""
	if line != expected {
		t.Fatalf("unexpected audit line:\n got: %s\nwant: %s", line, expected)
	}
}

func TestFormatAuditCommandLine_WithoutReason(t *testing.T) {
	line := formatAuditCommandLine("u1", "cli", "req-1", "help", "allow", "")
	expected := "[AUDIT] user=u1 channel=cli request=req-1 decision=allow command=\"help\""
	if line != expected {
		t.Fatalf("unexpected audit line:\n got: %s\nwant: %s", line, expected)
	}
}

func TestAppendCommandAuditEntry_WritesJSONL(t *testing.T) {
	baseDir := t.TempDir()
	original := commandAuditBasePath
	commandAuditBasePath = baseDir
	t.Cleanup(func() {
		commandAuditBasePath = original
	})

	ts := time.Date(2026, 2, 27, 23, 52, 0, 0, time.UTC)
	if err := appendCommandAuditEntry(ts, "", "", "", " config set executor.name codex ", "deny", " permission denied "); err != nil {
		t.Fatalf("appendCommandAuditEntry failed: %v", err)
	}

	logPath := filepath.Join(baseDir, "2026-02-27", "command_permission.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}

	var record commandAuditEntry
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatalf("failed to decode audit log: %v", err)
	}
	if record.UserID != "anonymous" || record.ChannelID != "unknown" || record.RequestID != "n/a" {
		t.Fatalf("unexpected defaults: %+v", record)
	}
	if record.Command != "config set executor.name codex" {
		t.Fatalf("unexpected command: %q", record.Command)
	}
	if record.Decision != "deny" {
		t.Fatalf("unexpected decision: %q", record.Decision)
	}
	if record.Reason != "permission denied" {
		t.Fatalf("unexpected reason: %q", record.Reason)
	}
}

func TestExecuteSlash_Status(t *testing.T) {
	exec := NewExecutor(nil, nil, nil)
	exec.SetStatusProvider(func(ctx context.Context) map[string]interface{} {
		return map[string]interface{}{
			"gateway": map[string]interface{}{"healthy": true},
		}
	})

	out, handled, err := exec.ExecuteSlash(context.Background(), types.Message{
		Content:   "/status",
		UserID:    "u-1",
		ChannelID: "cli",
		RequestID: "req-1",
	})
	if err != nil {
		t.Fatalf("execute slash status failed: %v", err)
	}
	if !handled {
		t.Fatal("expected /status to be handled")
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("status output is not json: %v", err)
	}
	gateway, ok := payload["gateway"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected gateway payload, got %+v", payload)
	}
	healthy, ok := gateway["healthy"].(bool)
	if !ok || !healthy {
		t.Fatalf("expected gateway healthy=true, got %+v", gateway)
	}
}

func TestExecuteSlash_StatusWithoutProvider(t *testing.T) {
	exec := NewExecutor(nil, nil, nil)
	_, handled, err := exec.ExecuteSlash(context.Background(), types.Message{
		Content:   "/status",
		UserID:    "u-1",
		ChannelID: "cli",
		RequestID: "req-1",
	})
	if !handled {
		t.Fatal("expected /status to be handled")
	}
	if err == nil {
		t.Fatal("expected missing provider error")
	}
	if !strings.Contains(err.Error(), "status provider") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecuteTaskCommand_Stats(t *testing.T) {
	tempDir := t.TempDir()
	database, err := servicestore.NewSQLiteDB(filepath.Join(tempDir, "db"))
	if err != nil {
		t.Fatalf("init sqlite failed: %v", err)
	}
	defer database.Close()

	store := servicetask.NewStore(database)
	exec := NewExecutor(nil, store, nil)
	ctx := context.Background()

	task1, err := store.CreateTask(ctx, "u-1", "task one", "cli")
	if err != nil {
		t.Fatalf("create task1 failed: %v", err)
	}
	_, err = store.CreateTask(ctx, "u-1", "task two", "cli")
	if err != nil {
		t.Fatalf("create task2 failed: %v", err)
	}
	if err := store.CloseTask(ctx, task1.ID); err != nil {
		t.Fatalf("close task failed: %v", err)
	}
	if err := store.SetForcedTask(ctx, "u-1", task1.ID); err != nil {
		t.Fatalf("set forced task failed: %v", err)
	}

	out, err := exec.executeTaskCommand(ctx, "u-1", []string{"stats"})
	if err != nil {
		t.Fatalf("execute task stats failed: %v", err)
	}
	if !strings.Contains(out, "open: 1") {
		t.Fatalf("expected open count in output: %s", out)
	}
	if !strings.Contains(out, "closed: 1") {
		t.Fatalf("expected closed count in output: %s", out)
	}
	if !strings.Contains(out, task1.ID) {
		t.Fatalf("expected forced task id in output: %s", out)
	}
}

package command

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
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

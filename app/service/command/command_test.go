package command

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"alter0/app/pkg/types"
)

func TestExecuteSlash_CustomHandler(t *testing.T) {
	exec := NewExecutor()
	exec.Register("ping", func(ctx context.Context, msg types.Message, args []string) (string, error) {
		if len(args) == 0 {
			return "pong", nil
		}
		return "pong:" + strings.Join(args, ","), nil
	})

	out, handled, err := exec.ExecuteSlash(context.Background(), types.Message{
		Content:   "/ping a b",
		UserID:    "u-1",
		ChannelID: "cli",
		RequestID: "req-1",
	})
	if err != nil {
		t.Fatalf("execute slash failed: %v", err)
	}
	if !handled {
		t.Fatal("expected command to be handled")
	}
	if out != "pong:a,b" {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestExecuteSlash_Authorizer(t *testing.T) {
	exec := NewExecutor()
	exec.Register("admin", func(ctx context.Context, msg types.Message, args []string) (string, error) {
		return "ok", nil
	})
	exec.SetAuthorizer(func(userID string, parts []string) error {
		if len(parts) > 0 && parts[0] == "admin" && userID != "root" {
			return fmt.Errorf("permission denied")
		}
		return nil
	})

	_, handled, err := exec.ExecuteSlash(context.Background(), types.Message{Content: "/admin", UserID: "guest"})
	if !handled {
		t.Fatal("expected /admin to be handled")
	}
	if err == nil {
		t.Fatal("expected permission error")
	}

	out, handled, err := exec.ExecuteSlash(context.Background(), types.Message{Content: "/admin", UserID: "root"})
	if err != nil {
		t.Fatalf("expected root to pass: %v", err)
	}
	if !handled || out != "ok" {
		t.Fatalf("unexpected result handled=%t out=%q", handled, out)
	}
}

func TestHelpText_DefaultAndCustom(t *testing.T) {
	exec := NewExecutor()
	exec.Register("ping", func(ctx context.Context, msg types.Message, args []string) (string, error) {
		return "pong", nil
	})

	out, handled, err := exec.ExecuteSlash(context.Background(), types.Message{Content: "/help"})
	if err != nil || !handled {
		t.Fatalf("default help failed handled=%t err=%v", handled, err)
	}
	if !strings.Contains(out, "/ping") {
		t.Fatalf("expected default help to include /ping, got: %s", out)
	}

	exec.SetHelpProvider(func() string { return "custom help" })
	out, handled, err = exec.ExecuteSlash(context.Background(), types.Message{Content: "/help"})
	if err != nil || !handled {
		t.Fatalf("custom help failed handled=%t err=%v", handled, err)
	}
	if out != "custom help" {
		t.Fatalf("expected custom help, got: %s", out)
	}
}

func TestExecuteSlash_Status(t *testing.T) {
	exec := NewExecutor()
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
	exec := NewExecutor()
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

func TestExecuteSlash_UnknownCommand(t *testing.T) {
	exec := NewExecutor()
	_, handled, err := exec.ExecuteSlash(context.Background(), types.Message{Content: "/unknown"})
	if !handled {
		t.Fatal("expected unknown command to be handled as slash")
	}
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("expected unknown command error, got: %v", err)
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

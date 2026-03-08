package application

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	terminaldomain "alter0/internal/terminal/domain"
)

func TestResolveShellCommandForOSUsesUTF8PowerShellOnWindows(t *testing.T) {
	command := resolveShellCommandForOS("windows", Options{}, func(file string) (string, error) {
		if file == "powershell.exe" {
			return filepath.Join(`C:\Windows\System32`, "WindowsPowerShell", "v1.0", "powershell.exe"), nil
		}
		return "", filepath.ErrBadPattern
	})

	if !strings.EqualFold(filepath.Base(command.path), "powershell.exe") {
		t.Fatalf("expected powershell.exe, got %q", command.path)
	}
	if len(command.args) < 4 {
		t.Fatalf("expected powershell bootstrap args, got %v", command.args)
	}
	if !containsArg(command.args, "-NoExit") {
		t.Fatalf("expected -NoExit in args, got %v", command.args)
	}
	if !containsJoined(command.args, "UTF8Encoding") {
		t.Fatalf("expected UTF-8 bootstrap command, got %v", command.args)
	}
	if !containsJoined(command.args, "chcp.com 65001") {
		t.Fatalf("expected code page bootstrap, got %v", command.args)
	}
}

func TestResolveShellCommandForOSIgnoresPwshAsDefaultOnWindows(t *testing.T) {
	command := resolveShellCommandForOS("windows", Options{}, func(file string) (string, error) {
		if file == "powershell.exe" {
			return filepath.Join(`C:\Windows\System32`, "WindowsPowerShell", "v1.0", "powershell.exe"), nil
		}
		if file == "pwsh.exe" {
			return filepath.Join(`C:\Program Files\PowerShell`, "7", "pwsh.exe"), nil
		}
		return "", filepath.ErrBadPattern
	})

	if !strings.EqualFold(filepath.Base(command.path), "powershell.exe") {
		t.Fatalf("expected powershell.exe as default, got %q", command.path)
	}
}

func TestResolveShellCommandForOSPreservesExplicitShell(t *testing.T) {
	command := resolveShellCommandForOS("linux", Options{
		Shell:     "cmd.exe",
		ShellArgs: []string{"/Q"},
	}, nil)

	if command.path != "cmd.exe" {
		t.Fatalf("expected explicit shell path, got %q", command.path)
	}
	if len(command.args) != 1 || command.args[0] != "/Q" {
		t.Fatalf("expected explicit shell args, got %v", command.args)
	}
	if command.label != "cmd.exe /Q" {
		t.Fatalf("expected explicit shell label, got %q", command.label)
	}
}

func TestResolveShellCommandForOSBootstrapsCmdEncodingOnWindows(t *testing.T) {
	command := resolveShellCommandForOS("windows", Options{
		Shell: "cmd.exe",
	}, nil)

	if command.path != "cmd.exe" {
		t.Fatalf("expected explicit shell path, got %q", command.path)
	}
	if len(command.args) != 3 {
		t.Fatalf("expected cmd bootstrap args, got %v", command.args)
	}
	if command.args[0] != "/D" || command.args[1] != "/K" {
		t.Fatalf("expected cmd bootstrap switches, got %v", command.args)
	}
	if !containsJoined(command.args, "65001") {
		t.Fatalf("expected UTF-8 codepage bootstrap, got %v", command.args)
	}
	if command.label != "cmd.exe" {
		t.Fatalf("expected compact label, got %q", command.label)
	}
}

func TestResolveShellCommandForOSPreservesExplicitWindowsArgs(t *testing.T) {
	command := resolveShellCommandForOS("windows", Options{
		Shell:     "powershell.exe",
		ShellArgs: []string{"-NoLogo", "-NoExit"},
	}, nil)

	if command.path != "powershell.exe" {
		t.Fatalf("expected explicit shell path, got %q", command.path)
	}
	if len(command.args) != 2 || command.args[0] != "-NoLogo" || command.args[1] != "-NoExit" {
		t.Fatalf("expected explicit shell args, got %v", command.args)
	}
	if command.label != "powershell.exe -NoLogo -NoExit" {
		t.Fatalf("expected explicit shell label, got %q", command.label)
	}
}

func TestServiceListPrefersLastOutputAtOverUpdatedAt(t *testing.T) {
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	service := &Service{
		sessions: map[string]*runtimeSession{
			"terminal-output-newer": {
				summary: terminaldomain.Session{
					ID:           "terminal-output-newer",
					OwnerID:      "owner-a",
					CreatedAt:    now.Add(-10 * time.Minute),
					LastOutputAt: now.Add(-2 * time.Minute),
					UpdatedAt:    now.Add(-4 * time.Minute),
				},
			},
			"terminal-updated-newer": {
				summary: terminaldomain.Session{
					ID:           "terminal-updated-newer",
					OwnerID:      "owner-a",
					CreatedAt:    now.Add(-9 * time.Minute),
					LastOutputAt: now.Add(-3 * time.Minute),
					UpdatedAt:    now.Add(-1 * time.Minute),
				},
			},
		},
	}

	items := service.List("owner-a")
	if len(items) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(items))
	}
	if items[0].ID != "terminal-output-newer" {
		t.Fatalf("expected last output ordering, got first session %q", items[0].ID)
	}
}

func TestRuntimeSessionAppendEntryLockedUpdatesLastOutputAtOnlyForRealOutput(t *testing.T) {
	session := &runtimeSession{
		summary: terminaldomain.Session{
			ID:        "terminal-output-flags",
			OwnerID:   "owner-a",
			CreatedAt: time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC),
		},
	}

	session.appendEntryLocked("system", "shell started")
	if !session.summary.LastOutputAt.IsZero() {
		t.Fatalf("expected system entry to keep last_output_at empty, got %s", session.summary.LastOutputAt)
	}

	session.appendEntryLocked("input", "pwd")
	if !session.summary.LastOutputAt.IsZero() {
		t.Fatalf("expected input entry to keep last_output_at empty, got %s", session.summary.LastOutputAt)
	}

	session.appendEntryLocked("stdout", "workspace")
	if session.summary.LastOutputAt.IsZero() {
		t.Fatalf("expected stdout entry to update last_output_at")
	}

	lastOutputAt := session.summary.LastOutputAt
	session.appendEntryLocked("stderr", "warning")
	if session.summary.LastOutputAt.IsZero() || session.summary.LastOutputAt.Before(lastOutputAt) {
		t.Fatalf("expected stderr entry to preserve or advance last_output_at")
	}
}

func containsArg(args []string, target string) bool {
	for _, item := range args {
		if item == target {
			return true
		}
	}
	return false
}

func containsJoined(args []string, fragment string) bool {
	for _, item := range args {
		if strings.Contains(item, fragment) {
			return true
		}
	}
	return false
}

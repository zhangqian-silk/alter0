package application

import (
	"path/filepath"
	"strings"
	"testing"
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

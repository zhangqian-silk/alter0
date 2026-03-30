package main

import (
	"os"
	"testing"
)

func TestEnsureDefaultCodexWorkspaceModeSetsSessionDefault(t *testing.T) {
	t.Setenv(defaultCodexWorkspaceModeEnvKey, "")

	ensureDefaultCodexWorkspaceMode()

	if got := os.Getenv(defaultCodexWorkspaceModeEnvKey); got != defaultCodexWorkspaceMode {
		t.Fatalf("expected %s=%q, got %q", defaultCodexWorkspaceModeEnvKey, defaultCodexWorkspaceMode, got)
	}
}

func TestEnsureDefaultCodexWorkspaceModePreservesExistingValue(t *testing.T) {
	t.Setenv(defaultCodexWorkspaceModeEnvKey, "session")

	ensureDefaultCodexWorkspaceMode()

	if got := os.Getenv(defaultCodexWorkspaceModeEnvKey); got != "session" {
		t.Fatalf("expected existing workspace mode preserved, got %q", got)
	}
}

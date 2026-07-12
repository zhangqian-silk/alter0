package application

import (
	"os"
	"path/filepath"
	"strings"
)

func RetireLegacySessionRuntimeArtifacts(workspaceDir string) error {
	workspaceDir = strings.TrimSpace(workspaceDir)
	if workspaceDir == "" {
		return nil
	}
	for _, path := range []string{
		filepath.Join(workspaceDir, "codex-home"),
		filepath.Join(workspaceDir, ".alter0", "codex-runtime", "skills"),
		filepath.Join(workspaceDir, ".alter0", "codex-runtime", "memory"),
	} {
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}
	for _, path := range []string{
		filepath.Join(workspaceDir, ".alter0", "codex-runtime", "skills.md"),
		filepath.Join(workspaceDir, ".alter0", "codex-runtime", "runtime.md"),
	} {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if err := removeLegacyManagedInstructions(filepath.Join(workspaceDir, "AGENTS.md")); err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(workspaceDir, ".alter0", "codex-runtime"))
	return nil
}

func removeLegacyManagedInstructions(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	content := string(raw)
	const startMarker = "<!-- alter0:codex-runtime:start -->"
	const endMarker = "<!-- alter0:codex-runtime:end -->"
	start := strings.Index(content, startMarker)
	end := strings.Index(content, endMarker)
	if start < 0 || end < start {
		return nil
	}
	end += len(endMarker)
	cleaned := strings.TrimSpace(content[:start] + content[end:])
	if cleaned == "" {
		return os.Remove(path)
	}
	return os.WriteFile(path, []byte(cleaned+"\n"), 0o644)
}

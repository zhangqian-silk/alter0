package infrastructure

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const gitMetadataTimeout = 1500 * time.Millisecond

func resolveGitDefaultBranch(repoRoot string) string {
	value := resolveGitCommandOutput(repoRoot, "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	value = strings.TrimSpace(strings.TrimPrefix(value, "origin/"))
	if value != "" {
		return value
	}
	return ""
}

func resolveGitCommandOutput(repoRoot string, args ...string) string {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" || len(args) == 0 {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), gitMetadataTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func buildSessionWorkspacePath(repoRoot string, sessionID string) string {
	repoRoot = strings.TrimSpace(repoRoot)
	sessionID = sanitizeWorkspaceSegment(sessionID)
	if repoRoot == "" || sessionID == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Join(repoRoot, defaultWorkspaceRootDir, workspaceDirectoryName, workspaceSessionsDirName, sessionID))
}

func buildSessionRepoWorkspacePath(repoRoot string, sessionID string) string {
	sessionWorkspace := buildSessionWorkspacePath(repoRoot, sessionID)
	if sessionWorkspace == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Join(sessionWorkspace, workspaceRepoDirName))
}

package infrastructure

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	execdomain "alter0/internal/execution/domain"
)

const (
	codingAgentGitTimeout        = 1500 * time.Millisecond
	codingAgentPreviewBaseDomain = "alter0.cn"
)

func renderCodingAgentExecutionContext(metadata map[string]string) string {
	repoRoot, err := resolveToolRepoRoot()
	if err != nil {
		return ""
	}

	currentBranch := resolveGitCommandOutput(repoRoot, "branch", "--show-current")
	if currentBranch == "" {
		currentBranch = resolveGitCommandOutput(repoRoot, "symbolic-ref", "--short", "HEAD")
	}
	remoteRepo := resolveGitCommandOutput(repoRoot, "remote", "get-url", "origin")
	defaultBranch := resolveGitDefaultBranch(repoRoot)
	sessionID := strings.TrimSpace(firstNonEmpty(
		metadataValue(metadata, execdomain.RuntimeSessionIDMetadataKey),
		metadataValue(metadata, sessionIDMetadataFallback),
	))
	sessionWorkspaceDir := buildCodingSessionWorkspacePath(repoRoot, sessionID)
	repoWorkspaceDir := buildCodingSessionRepoWorkspacePath(repoRoot, sessionID)
	sessionShortHash := shortSessionHash(sessionID)
	previewHost := ""
	previewURL := ""
	if sessionShortHash != "" {
		previewHost = sessionShortHash + "." + codingAgentPreviewBaseDomain
		previewURL = "https://" + previewHost
	}

	lines := []string{
		"Current coding workspace context:",
		"- Remote repository: " + fallbackCodingContextValue(remoteRepo),
		"- Source repository path: " + fallbackCodingContextValue(repoRoot),
		"- Active local branch: " + fallbackCodingContextValue(currentBranch),
		"- Dedicated repository workspace path: " + fallbackCodingContextValue(repoWorkspaceDir),
		"- Session workspace path: " + fallbackCodingContextValue(sessionWorkspaceDir),
	}
	if defaultBranch != "" {
		lines = append(lines, "- PR base branch: "+defaultBranch)
	} else {
		lines = append(lines, "- PR base branch: detect from the current git remote before final PR handoff")
	}
	if sessionShortHash != "" {
		lines = append(lines,
			"- Session short hash: "+sessionShortHash,
			"- Preview host: "+previewHost,
			"- Preview URL: "+previewURL,
			"- Preview rule: when a test page is needed, deploy or update it on the session preview host above. The subdomain must use the current session short hash.",
		)
	} else {
		lines = append(lines, "- Preview rule: when a test page is needed, derive an 8-character hex short hash from the current session id and use https://<short-hash>."+codingAgentPreviewBaseDomain)
	}
	lines = append(lines,
		"Delivery requirements:",
		"- When repository code needs to be pulled, inspected, edited, built, or tested, do it in the dedicated repository workspace above as a full git clone with its own .git metadata rather than directly in the source repository path or via git worktree.",
		"- Treat repository state, branch hygiene, verification evidence, preview deployment, and PR readiness as part of the coding task rather than optional follow-up work.",
		"- For user-visible web changes, do not claim success without validating the changed page and, when a preview page is expected, deploying or updating it at the preview URL based on the current session short hash before reporting completion.",
		"- Keep the branch and PR handoff explicit: report the working branch, the intended PR base branch, test results, preview URL when applicable, and any remaining blockers.",
	)
	return strings.Join(lines, "\n")
}

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
	ctx, cancel := context.WithTimeout(context.Background(), codingAgentGitTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func shortSessionHash(sessionID string) string {
	trimmed := strings.TrimSpace(sessionID)
	if trimmed == "" {
		return ""
	}
	sum := sha1.Sum([]byte(trimmed))
	return hex.EncodeToString(sum[:])[:8]
}

func fallbackCodingContextValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "(not detected)"
	}
	return value
}

func cloneExecutionMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(metadata))
	for key, value := range metadata {
		out[key] = value
	}
	return out
}

func buildCodexExecMetadata(metadata map[string]string) map[string]string {
	out := cloneExecutionMetadata(metadata)
	if !isCodingAgent(metadata) {
		return out
	}
	out[codexWorkspaceModeMetadataKey] = codexWorkspaceModeSessionRepo
	if _, exists := out[codexWorktreeSourceRootKey]; exists {
		return out
	}
	repoRoot, err := resolveToolRepoRoot()
	if err != nil {
		return out
	}
	out[codexWorktreeSourceRootKey] = repoRoot
	return out
}

func buildPreviewURLForSession(sessionID string) string {
	short := shortSessionHash(sessionID)
	if short == "" {
		return ""
	}
	return fmt.Sprintf("https://%s.%s", short, codingAgentPreviewBaseDomain)
}

func buildCodingSessionWorkspacePath(repoRoot string, sessionID string) string {
	repoRoot = strings.TrimSpace(repoRoot)
	sessionID = sanitizeWorkspaceSegment(sessionID)
	if repoRoot == "" || sessionID == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Join(repoRoot, defaultWorkspaceRootDir, workspaceDirectoryName, workspaceSessionsDirName, sessionID))
}

func buildCodingSessionRepoWorkspacePath(repoRoot string, sessionID string) string {
	sessionWorkspace := buildCodingSessionWorkspacePath(repoRoot, sessionID)
	if sessionWorkspace == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Join(sessionWorkspace, workspaceRepoDirName))
}

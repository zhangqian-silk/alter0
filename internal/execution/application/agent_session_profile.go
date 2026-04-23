package application

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	execdomain "alter0/internal/execution/domain"
	shareddomain "alter0/internal/shared/domain"
)

const (
	agentSessionProfileAutoStartMarker  = "<!-- alter0:agent-session:auto:start -->"
	agentSessionProfileAutoEndMarker    = "<!-- alter0:agent-session:auto:end -->"
	agentSessionProfileNotesStartMarker = "<!-- alter0:agent-session:notes:start -->"
	agentSessionProfileNotesEndMarker   = "<!-- alter0:agent-session:notes:end -->"
	agentSessionPreviewBaseDomain       = "alter0.cn"
	agentSessionGitTimeout              = 1200 * time.Millisecond
)

func ensureAgentSessionProfileFile(repoRoot string, msg shareddomain.UnifiedMessage, now time.Time) (string, error) {
	relativePath := agentSessionProfileRelativePath(msg)
	if relativePath == "" {
		return "", nil
	}
	absolutePath := filepath.Join(repoRoot, filepath.FromSlash(relativePath))
	if now.IsZero() {
		now = time.Now().UTC()
	}
	existingNotes := ""
	if raw, err := os.ReadFile(absolutePath); err == nil {
		existingNotes = extractAgentSessionProfileNotes(string(raw))
	}
	rendered := renderAgentSessionProfile(repoRoot, msg, now, existingNotes)
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(absolutePath, []byte(rendered), 0o644); err != nil {
		return "", err
	}
	return absolutePath, nil
}

func agentSessionProfileRelativePath(msg shareddomain.UnifiedMessage) string {
	agentID := normalizeMemoryAgentID(metadataValue(msg.Metadata, execdomain.AgentIDMetadataKey))
	sessionID := normalizeMemorySessionID(msg.SessionID)
	if agentID == "" || sessionID == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Join(".alter0", "agents", agentID, "sessions", sessionID+".md"))
}

func normalizeMemorySessionID(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	normalized = memoryAgentIDSanitizer.ReplaceAllString(normalized, "-")
	normalized = strings.Trim(normalized, "-.")
	return normalized
}

func extractAgentSessionProfileNotes(content string) string {
	start := strings.Index(content, agentSessionProfileNotesStartMarker)
	end := strings.Index(content, agentSessionProfileNotesEndMarker)
	if start < 0 || end < 0 || end <= start {
		return ""
	}
	start += len(agentSessionProfileNotesStartMarker)
	return strings.TrimSpace(content[start:end])
}

func renderAgentSessionProfile(
	repoRoot string,
	msg shareddomain.UnifiedMessage,
	now time.Time,
	existingNotes string,
) string {
	lines := []string{
		"# Agent Session Profile",
		"",
		agentSessionProfileAutoStartMarker,
		"## Session Identity",
		"- agent_id: " + fallbackProfileValue(metadataValue(msg.Metadata, execdomain.AgentIDMetadataKey)),
		"- agent_name: " + fallbackProfileValue(metadataValue(msg.Metadata, execdomain.AgentNameMetadataKey)),
		"- session_id: " + fallbackProfileValue(msg.SessionID),
		"- updated_at: " + now.UTC().Format(time.RFC3339),
		"- channel_type: " + fallbackProfileValue(string(msg.ChannelType)),
		"- channel_id: " + fallbackProfileValue(msg.ChannelID),
		"- trigger_type: " + fallbackProfileValue(string(msg.TriggerType)),
	}
	if userID := strings.TrimSpace(msg.UserID); userID != "" {
		lines = append(lines, "- user_id: "+userID)
	}
	workspacePath := resolveAgentSessionWorkspacePath(repoRoot, msg)
	if workspacePath != "" {
		lines = append(lines, "", "## Session Scope", "- workspace_path: "+workspacePath)
	}
	if strings.EqualFold(strings.TrimSpace(metadataValue(msg.Metadata, execdomain.AgentIDMetadataKey)), "coding") {
		if codingLines := renderAgentSessionCodingContext(repoRoot, workspacePath, msg.SessionID); len(codingLines) > 0 {
			lines = append(lines, "")
			lines = append(lines, codingLines...)
		}
	}
	lines = append(lines, agentSessionProfileAutoEndMarker, "", "## Notes", agentSessionProfileNotesStartMarker)
	if strings.TrimSpace(existingNotes) != "" {
		lines = append(lines, existingNotes)
	}
	lines = append(lines, agentSessionProfileNotesEndMarker)
	return strings.Join(lines, "\n")
}

func renderAgentSessionCodingContext(repoRoot string, workspacePath string, sessionID string) []string {
	repositoryPath := resolveAgentSessionRepoWorkspacePath(workspacePath)
	gitInspectionPath := repositoryPath
	if topLevel := resolveGitRepoRoot(gitInspectionPath); topLevel == "" {
		gitInspectionPath = resolvePreferredSessionRepoPath(repoRoot, repositoryPath)
	}
	lines := []string{"## Coding Context"}
	if repositoryPath != "" {
		lines = append(lines, "- local_repository_path: "+repositoryPath)
	}
	if repoRoot != "" {
		lines = append(lines, "- source_repository_path: "+filepath.ToSlash(repoRoot))
	}
	if remote := resolveGitCommandOutput(gitInspectionPath, "remote", "get-url", "origin"); remote != "" {
		lines = append(lines, "- remote_repository: "+remote)
	}
	if branch := resolveGitCurrentBranch(gitInspectionPath); branch != "" {
		lines = append(lines, "- active_branch: "+branch)
	}
	if base := resolveGitDefaultBranch(gitInspectionPath); base != "" {
		lines = append(lines, "- pr_base_branch: "+base)
	}
	if workspacePath != "" {
		lines = append(lines, "- session_workspace_path: "+workspacePath)
	}
	if previewURL := buildAgentSessionPreviewURL(sessionID); previewURL != "" {
		lines = append(lines, "- preview_url: "+previewURL)
	}
	if len(lines) == 1 {
		return nil
	}
	return lines
}

func resolveAgentSessionWorkspacePath(repoRoot string, msg shareddomain.UnifiedMessage) string {
	sessionID := strings.TrimSpace(msg.SessionID)
	if repoRoot == "" || sessionID == "" {
		return ""
	}
	parts := []string{repoRoot, ".alter0", "workspaces", "sessions", sessionID}
	if taskID := strings.TrimSpace(msg.Metadata["task_id"]); taskID != "" {
		parts = append(parts, "tasks", taskID)
	}
	return filepath.ToSlash(filepath.Join(parts...))
}

func resolvePreferredSessionRepoPath(repoRoot string, workspacePath string) string {
	if topLevel := resolveGitRepoRoot(workspacePath); topLevel != "" {
		return topLevel
	}
	if topLevel := resolveGitRepoRoot(repoRoot); topLevel != "" {
		return topLevel
	}
	return ""
}

func resolveAgentSessionRepoWorkspacePath(workspacePath string) string {
	workspacePath = strings.TrimSpace(workspacePath)
	if workspacePath == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Join(workspacePath, "repo"))
}

func resolveGitRepoRoot(path string) string {
	return resolveGitCommandOutput(path, "rev-parse", "--show-toplevel")
}

func resolveGitCurrentBranch(repoRoot string) string {
	currentBranch := resolveGitCommandOutput(repoRoot, "branch", "--show-current")
	if currentBranch != "" {
		return currentBranch
	}
	return resolveGitCommandOutput(repoRoot, "symbolic-ref", "--short", "HEAD")
}

func resolveGitDefaultBranch(repoRoot string) string {
	value := resolveGitCommandOutput(repoRoot, "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	return strings.TrimSpace(strings.TrimPrefix(value, "origin/"))
}

func resolveGitCommandOutput(repoRoot string, args ...string) string {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" || len(args) == 0 {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), agentSessionGitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func buildAgentSessionPreviewURL(sessionID string) string {
	shortHash := buildAgentSessionShortHash(sessionID)
	if shortHash == "" {
		return ""
	}
	return fmt.Sprintf("https://%s.%s", shortHash, agentSessionPreviewBaseDomain)
}

func buildAgentSessionShortHash(sessionID string) string {
	trimmed := strings.TrimSpace(sessionID)
	if trimmed == "" {
		return ""
	}
	sum := sha1.Sum([]byte(trimmed))
	return hex.EncodeToString(sum[:])[:8]
}

func fallbackProfileValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "(not set)"
	}
	return trimmed
}

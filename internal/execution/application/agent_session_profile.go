package application

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
	existingAttributes := map[string]string{}
	for _, candidatePath := range agentSessionProfileRelativePaths(msg) {
		raw, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(candidatePath)))
		if err != nil {
			continue
		}
		existingNotes = extractAgentSessionProfileNotes(string(raw))
		existingAttributes = extractAgentSessionProfileAttributes(string(raw))
		break
	}
	rendered := renderAgentSessionProfile(repoRoot, msg, now, existingNotes, existingAttributes)
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(absolutePath, []byte(rendered), 0o644); err != nil {
		return "", err
	}
	return absolutePath, nil
}

func agentSessionProfileRelativePath(msg shareddomain.UnifiedMessage) string {
	paths := agentSessionProfileRelativePaths(msg)
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

func agentSessionProfileRelativePaths(msg shareddomain.UnifiedMessage) []string {
	agentID := normalizeMemoryAgentID(metadataValue(msg.Metadata, execdomain.AgentIDMetadataKey))
	sessionID := normalizeMemorySessionID(msg.SessionID)
	if agentID == "" || sessionID == "" {
		return nil
	}
	return agentPrivateRelativePaths(agentID, "sessions", sessionID+".md")
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
	existingAttributes map[string]string,
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
	instanceAttributes := resolveAgentSessionInstanceAttributes(repoRoot, workspacePath, msg, existingAttributes)
	if len(instanceAttributes) > 0 {
		lines = append(lines, "", "## Instance Attributes")
		lines = append(lines, renderAgentSessionAttributeLines(instanceAttributes)...)
	}
	if strings.EqualFold(strings.TrimSpace(metadataValue(msg.Metadata, execdomain.AgentIDMetadataKey)), "coding") {
		if codingLines := renderAgentSessionCodingContext(instanceAttributes); len(codingLines) > 0 {
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

func renderAgentSessionCodingContext(instanceAttributes map[string]string) []string {
	lines := []string{"## Coding Context"}
	if repositoryPath := strings.TrimSpace(instanceAttributes["repository_path"]); repositoryPath != "" {
		lines = append(lines, "- local_repository_path: "+repositoryPath)
	}
	if sourceRepositoryPath := strings.TrimSpace(instanceAttributes["source_repository_path"]); sourceRepositoryPath != "" {
		lines = append(lines, "- source_repository_path: "+sourceRepositoryPath)
	}
	if remote := strings.TrimSpace(instanceAttributes["remote_repository"]); remote != "" {
		lines = append(lines, "- remote_repository: "+remote)
	}
	if branch := strings.TrimSpace(instanceAttributes["branch"]); branch != "" {
		lines = append(lines, "- active_branch: "+branch)
	}
	if base := strings.TrimSpace(instanceAttributes["base_branch"]); base != "" {
		lines = append(lines, "- pr_base_branch: "+base)
	}
	if workspacePath := strings.TrimSpace(instanceAttributes["session_workspace_path"]); workspacePath != "" {
		lines = append(lines, "- session_workspace_path: "+workspacePath)
	}
	if previewURL := strings.TrimSpace(instanceAttributes["preview_url"]); previewURL != "" {
		lines = append(lines, "- preview_url: "+previewURL)
	}
	if len(lines) == 1 {
		return nil
	}
	return lines
}

func extractAgentSessionProfileAttributes(content string) map[string]string {
	section := extractAgentSessionProfileSection(content, "## Instance Attributes")
	if section == "" {
		return map[string]string{}
	}
	attributes := map[string]string{}
	for _, line := range strings.Split(section, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		keyValue := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
		separator := strings.Index(keyValue, ":")
		if separator <= 0 {
			continue
		}
		key := normalizeAgentInstanceAttributeKey(keyValue[:separator])
		value := strings.TrimSpace(keyValue[separator+1:])
		if key == "" || value == "" || value == "(not set)" {
			continue
		}
		attributes[key] = value
	}
	return attributes
}

func extractAgentSessionProfileSection(content string, header string) string {
	autoStart := strings.Index(content, agentSessionProfileAutoStartMarker)
	autoEnd := strings.Index(content, agentSessionProfileAutoEndMarker)
	if autoStart < 0 || autoEnd < 0 || autoEnd <= autoStart {
		return ""
	}
	autoStart += len(agentSessionProfileAutoStartMarker)
	autoBlock := content[autoStart:autoEnd]
	headerIndex := strings.Index(autoBlock, header)
	if headerIndex < 0 {
		return ""
	}
	sectionBody := autoBlock[headerIndex+len(header):]
	nextHeader := strings.Index(sectionBody, "\n## ")
	if nextHeader >= 0 {
		sectionBody = sectionBody[:nextHeader]
	}
	return strings.TrimSpace(sectionBody)
}

func resolveAgentSessionInstanceAttributes(
	repoRoot string,
	workspacePath string,
	msg shareddomain.UnifiedMessage,
	existingAttributes map[string]string,
) map[string]string {
	attributes := cloneAgentSessionAttributes(existingAttributes)
	for key, value := range parseAgentSessionMetadataAttributes(msg.Metadata) {
		attributes[key] = value
	}
	if strings.EqualFold(strings.TrimSpace(metadataValue(msg.Metadata, execdomain.AgentIDMetadataKey)), "coding") {
		for key, value := range resolveAgentSessionCodingAttributes(repoRoot, workspacePath, msg.SessionID) {
			attributes[key] = value
		}
	}
	if len(attributes) == 0 {
		return nil
	}
	return attributes
}

func parseAgentSessionMetadataAttributes(metadata map[string]string) map[string]string {
	attributes := map[string]string{}
	if len(metadata) == 0 {
		return attributes
	}
	if raw := strings.TrimSpace(metadataValue(metadata, execdomain.AgentInstanceAttributesMetadataKey)); raw != "" {
		var payload map[string]string
		if err := json.Unmarshal([]byte(raw), &payload); err == nil {
			for key, value := range payload {
				normalizedKey := normalizeAgentInstanceAttributeKey(key)
				normalizedValue := strings.TrimSpace(value)
				if normalizedKey == "" || normalizedValue == "" {
					continue
				}
				attributes[normalizedKey] = normalizedValue
			}
		}
	}
	for key, value := range metadata {
		if !strings.HasPrefix(strings.TrimSpace(key), execdomain.AgentInstanceAttributeMetadataPrefix) {
			continue
		}
		normalizedKey := normalizeAgentInstanceAttributeKey(strings.TrimPrefix(strings.TrimSpace(key), execdomain.AgentInstanceAttributeMetadataPrefix))
		normalizedValue := strings.TrimSpace(value)
		if normalizedKey == "" || normalizedValue == "" {
			continue
		}
		attributes[normalizedKey] = normalizedValue
	}
	return attributes
}

func resolveAgentSessionCodingAttributes(repoRoot string, workspacePath string, sessionID string) map[string]string {
	attributes := map[string]string{}
	repositoryPath := resolveAgentSessionRepoWorkspacePath(workspacePath)
	gitInspectionPath := repositoryPath
	if topLevel := resolveGitRepoRoot(gitInspectionPath); topLevel == "" {
		gitInspectionPath = resolvePreferredSessionRepoPath(repoRoot, repositoryPath)
	}
	if repositoryPath != "" {
		attributes["repository_path"] = repositoryPath
	}
	if repoRoot != "" {
		attributes["source_repository_path"] = filepath.ToSlash(repoRoot)
	}
	if remote := resolveGitCommandOutput(gitInspectionPath, "remote", "get-url", "origin"); remote != "" {
		attributes["remote_repository"] = remote
	}
	if branch := resolveGitCurrentBranch(gitInspectionPath); branch != "" {
		attributes["branch"] = branch
	}
	if base := resolveGitDefaultBranch(gitInspectionPath); base != "" {
		attributes["base_branch"] = base
	}
	if workspacePath != "" {
		attributes["session_workspace_path"] = workspacePath
	}
	if previewURL := buildAgentSessionPreviewURL(sessionID); previewURL != "" {
		attributes["preview_url"] = previewURL
	}
	if previewSubdomain := buildAgentSessionShortHash(sessionID); previewSubdomain != "" {
		attributes["preview_subdomain"] = previewSubdomain
	}
	return attributes
}

func renderAgentSessionAttributeLines(attributes map[string]string) []string {
	keys := make([]string, 0, len(attributes))
	for key := range attributes {
		normalizedKey := normalizeAgentInstanceAttributeKey(key)
		if normalizedKey == "" {
			continue
		}
		keys = append(keys, normalizedKey)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		value := strings.TrimSpace(attributes[key])
		if value == "" {
			continue
		}
		lines = append(lines, "- "+key+": "+value)
	}
	return lines
}

func cloneAgentSessionAttributes(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		normalizedKey := normalizeAgentInstanceAttributeKey(key)
		normalizedValue := strings.TrimSpace(value)
		if normalizedKey == "" || normalizedValue == "" {
			continue
		}
		cloned[normalizedKey] = normalizedValue
	}
	return cloned
}

func normalizeAgentInstanceAttributeKey(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	normalized = strings.ReplaceAll(normalized, " ", "_")
	normalized = memoryAgentIDSanitizer.ReplaceAllString(normalized, "_")
	return strings.Trim(normalized, "_.-")
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

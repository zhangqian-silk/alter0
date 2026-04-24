package web

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	controldomain "alter0/internal/control/domain"
)

var agentSessionProfilePathSanitizer = regexp.MustCompile(`[^a-z0-9._-]+`)

type agentSessionProfileResponse struct {
	AgentID    string                                   `json:"agent_id"`
	SessionID  string                                   `json:"session_id"`
	Path       string                                   `json:"path"`
	Exists     bool                                     `json:"exists"`
	Fields     []controldomain.AgentSessionProfileField `json:"fields,omitempty"`
	Attributes map[string]string                        `json:"attributes,omitempty"`
}

func (s *Server) agentSessionProfileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if s.agents == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "agent catalog unavailable"})
		return
	}
	agentID := normalizeAgentSessionProfilePathID(r.URL.Query().Get("agent_id"))
	sessionID := normalizeAgentSessionProfilePathID(r.URL.Query().Get("session_id"))
	if agentID == "" || sessionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "agent_id and session_id are required"})
		return
	}
	agent, ok := s.agents.ResolveAgent(agentID)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "agent not found"})
		return
	}
	relativePath := filepath.ToSlash(filepath.Join(".alter0", "agents", agent.ID, "sessions", sessionID+".md"))
	response := agentSessionProfileResponse{
		AgentID:   agent.ID,
		SessionID: sessionID,
		Path:      relativePath,
		Fields:    agent.SessionProfileFields,
	}
	if strings.TrimSpace(s.workspaceRoot) == "" {
		writeJSON(w, http.StatusOK, response)
		return
	}
	candidatePath := filepath.Join(s.workspaceRoot, filepath.FromSlash(relativePath))
	content, err := os.ReadFile(candidatePath)
	if err == nil {
		response.Exists = true
		response.Attributes = parseAgentSessionProfileAttributes(string(content))
	}
	writeJSON(w, http.StatusOK, response)
}

func normalizeAgentSessionProfilePathID(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	normalized = agentSessionProfilePathSanitizer.ReplaceAllString(normalized, "-")
	return strings.Trim(normalized, "-.")
}

func parseAgentSessionProfileAttributes(content string) map[string]string {
	section := extractAgentSessionProfileSection(content, "## Instance Attributes")
	if section == "" {
		return nil
	}
	attributes := make(map[string]string)
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
		key := strings.TrimSpace(keyValue[:separator])
		value := strings.TrimSpace(keyValue[separator+1:])
		if key == "" || value == "" || value == "(not set)" {
			continue
		}
		attributes[key] = value
	}
	if len(attributes) == 0 {
		return nil
	}
	return attributes
}

func extractAgentSessionProfileSection(content string, header string) string {
	const (
		autoStartMarker = "<!-- alter0:agent-session:auto:start -->"
		autoEndMarker   = "<!-- alter0:agent-session:auto:end -->"
	)
	autoStart := strings.Index(content, autoStartMarker)
	autoEnd := strings.Index(content, autoEndMarker)
	if autoStart < 0 || autoEnd < 0 || autoEnd <= autoStart {
		return ""
	}
	autoStart += len(autoStartMarker)
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

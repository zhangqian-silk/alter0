package application

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	controldomain "alter0/internal/control/domain"
	execdomain "alter0/internal/execution/domain"
	shareddomain "alter0/internal/shared/domain"
)

type SessionProfileExtractor interface {
	ExtractPatch(ctx context.Context, request SessionProfileExtractionRequest) (map[string]string, error)
}

type SessionProfileExtractionRequest struct {
	Message            shareddomain.UnifiedMessage
	Agent              controldomain.Agent
	ExistingAttributes map[string]string
}

type agentProfileResolver interface {
	ResolveAgent(id string) (controldomain.Agent, bool)
}

func (s *Service) resolveSessionProfileAttributes(
	ctx context.Context,
	msg shareddomain.UnifiedMessage,
	metadata map[string]string,
) {
	if s == nil || s.sessionProfileExtractor == nil || s.agentSource == nil {
		return
	}
	agentID := strings.TrimSpace(metadataValue(metadata, execdomain.AgentIDMetadataKey))
	if agentID == "" {
		return
	}
	agent, ok := s.agentSource.ResolveAgent(agentID)
	if !ok || len(agent.SessionProfileFields) == 0 {
		return
	}
	existingAttributes := readExistingSessionProfileAttributes(msg)
	patch, err := s.sessionProfileExtractor.ExtractPatch(ctx, SessionProfileExtractionRequest{
		Message:            msg,
		Agent:              agent,
		ExistingAttributes: existingAttributes,
	})
	if err != nil || len(patch) == 0 {
		return
	}
	allowed := map[string]controldomain.AgentSessionProfileField{}
	for _, field := range agent.SessionProfileFields {
		key := normalizeAgentInstanceAttributeKey(field.Key)
		if key == "" || field.ReadOnly {
			continue
		}
		allowed[key] = field
	}
	for key, value := range patch {
		normalizedKey := normalizeAgentInstanceAttributeKey(key)
		normalizedValue := strings.TrimSpace(value)
		if normalizedKey == "" || normalizedValue == "" {
			continue
		}
		if _, ok := allowed[normalizedKey]; !ok {
			continue
		}
		metadata[execdomain.AgentInstanceAttributeMetadataPrefix+normalizedKey] = normalizedValue
	}
}

func readExistingSessionProfileAttributes(msg shareddomain.UnifiedMessage) map[string]string {
	repoRoot, err := resolveMemoryRepoRoot()
	if err != nil {
		return nil
	}
	for _, relativePath := range agentSessionProfileRelativePaths(msg) {
		content, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(relativePath)))
		if err != nil {
			continue
		}
		return extractAgentSessionProfileAttributes(string(content))
	}
	return nil
}

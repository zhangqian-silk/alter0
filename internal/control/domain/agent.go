package domain

import (
	"encoding/json"
	"strconv"
	"strings"
)

const (
	agentProviderIDMetadataKey    = "agent.provider_id"
	agentModelMetadataKey         = "agent.model"
	agentSystemPromptMetadataKey  = "agent.system_prompt"
	agentMaxIterationsMetadataKey = "agent.max_iterations"
	agentToolsMetadataKey         = "agent.tools"
	agentSkillsMetadataKey        = "agent.skills"
	agentMCPsMetadataKey          = "agent.mcps"
)

type Agent struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Type          CapabilityType    `json:"type"`
	Enabled       bool              `json:"enabled"`
	Scope         CapabilityScope   `json:"scope"`
	Version       string            `json:"version"`
	ProviderID    string            `json:"provider_id,omitempty"`
	Model         string            `json:"model,omitempty"`
	SystemPrompt  string            `json:"system_prompt,omitempty"`
	MaxIterations int               `json:"max_iterations,omitempty"`
	Tools         []string          `json:"tools,omitempty"`
	Skills        []string          `json:"skills,omitempty"`
	MCPs          []string          `json:"mcps,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

func (a Agent) AsCapability() Capability {
	capabilityType := a.Type
	if capabilityType == "" {
		capabilityType = CapabilityTypeAgent
	}
	metadata := cloneAgentMetadata(a.Metadata)
	setAgentMetadata(metadata, agentProviderIDMetadataKey, a.ProviderID)
	setAgentMetadata(metadata, agentModelMetadataKey, a.Model)
	setAgentMetadata(metadata, agentSystemPromptMetadataKey, a.SystemPrompt)
	if a.MaxIterations > 0 {
		metadata[agentMaxIterationsMetadataKey] = strconv.Itoa(a.MaxIterations)
	} else {
		delete(metadata, agentMaxIterationsMetadataKey)
	}
	setAgentListMetadata(metadata, agentToolsMetadataKey, a.Tools)
	setAgentListMetadata(metadata, agentSkillsMetadataKey, a.Skills)
	setAgentListMetadata(metadata, agentMCPsMetadataKey, a.MCPs)
	return Capability{
		ID:       a.ID,
		Name:     a.Name,
		Type:     capabilityType,
		Enabled:  a.Enabled,
		Scope:    a.Scope,
		Version:  a.Version,
		Metadata: metadata,
	}
}

func (a Agent) Validate() error {
	return a.AsCapability().Validate()
}

func AgentFromCapability(capability Capability) Agent {
	normalized := capability.Normalized()
	metadata := cloneAgentMetadata(normalized.Metadata)
	maxIterations := 0
	if raw := strings.TrimSpace(metadata[agentMaxIterationsMetadataKey]); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			maxIterations = value
		}
	}
	agent := Agent{
		ID:            normalized.ID,
		Name:          normalized.Name,
		Type:          normalized.Type,
		Enabled:       normalized.Enabled,
		Scope:         normalized.Scope,
		Version:       normalized.Version,
		ProviderID:    strings.TrimSpace(metadata[agentProviderIDMetadataKey]),
		Model:         strings.TrimSpace(metadata[agentModelMetadataKey]),
		SystemPrompt:  metadata[agentSystemPromptMetadataKey],
		MaxIterations: maxIterations,
		Tools:         parseAgentListMetadata(metadata[agentToolsMetadataKey]),
		Skills:        parseAgentListMetadata(metadata[agentSkillsMetadataKey]),
		MCPs:          parseAgentListMetadata(metadata[agentMCPsMetadataKey]),
		Metadata:      metadata,
	}
	delete(agent.Metadata, agentProviderIDMetadataKey)
	delete(agent.Metadata, agentModelMetadataKey)
	delete(agent.Metadata, agentSystemPromptMetadataKey)
	delete(agent.Metadata, agentMaxIterationsMetadataKey)
	delete(agent.Metadata, agentToolsMetadataKey)
	delete(agent.Metadata, agentSkillsMetadataKey)
	delete(agent.Metadata, agentMCPsMetadataKey)
	if len(agent.Metadata) == 0 {
		agent.Metadata = nil
	}
	return agent
}

func cloneAgentMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(metadata))
	for key, value := range metadata {
		cloned[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return cloned
}

func setAgentMetadata(metadata map[string]string, key string, value string) {
	if metadata == nil {
		return
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		delete(metadata, key)
		return
	}
	metadata[key] = trimmed
}

func setAgentListMetadata(metadata map[string]string, key string, values []string) {
	if metadata == nil {
		return
	}
	items := normalizeAgentList(values)
	if len(items) == 0 {
		delete(metadata, key)
		return
	}
	encoded, err := json.Marshal(items)
	if err != nil {
		return
	}
	metadata[key] = string(encoded)
}

func parseAgentListMetadata(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	var items []string
	if strings.HasPrefix(trimmed, "[") {
		if err := json.Unmarshal([]byte(trimmed), &items); err == nil {
			return normalizeAgentList(items)
		}
	}
	return normalizeAgentList(strings.Split(trimmed, ","))
}

func normalizeAgentList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	items := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		lookup := strings.ToLower(trimmed)
		if _, ok := seen[lookup]; ok {
			continue
		}
		seen[lookup] = struct{}{}
		items = append(items, trimmed)
	}
	if len(items) == 0 {
		return nil
	}
	return items
}

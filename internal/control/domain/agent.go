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
	agentMemoryFilesMetadataKey   = "agent.memory_files"
	agentSourceMetadataKey        = "agent.source"
	agentKindMetadataKey          = "agent.kind"
	agentDescriptionMetadataKey   = "agent.description"
	agentEntryPointMetadataKey    = "agent.entrypoint"
	agentDelegatableMetadataKey   = "agent.delegatable"
	agentUIRouteMetadataKey       = "agent.ui_route"
	agentCapabilitiesMetadataKey  = "agent.capabilities"
	agentSessionProfileFieldsKey  = "agent.session_profile_fields"
	agentDeliverablesMetadataKey  = "agent.deliverables"
)

const (
	AgentSourceBuiltin = "builtin"
	AgentSourceManaged = "managed"

	AgentKindMain       = "main"
	AgentKindSpecialist = "specialist"
	AgentKindCustom     = "custom"
)

type Agent struct {
	ID                   string                     `json:"id"`
	Name                 string                     `json:"name"`
	Type                 CapabilityType             `json:"type"`
	Enabled              bool                       `json:"enabled"`
	Scope                CapabilityScope            `json:"scope"`
	Version              string                     `json:"version"`
	ProviderID           string                     `json:"provider_id,omitempty"`
	Model                string                     `json:"model,omitempty"`
	SystemPrompt         string                     `json:"system_prompt,omitempty"`
	MaxIterations        int                        `json:"max_iterations,omitempty"`
	Tools                []string                   `json:"tools,omitempty"`
	Skills               []string                   `json:"skills,omitempty"`
	MCPs                 []string                   `json:"mcps,omitempty"`
	MemoryFiles          []string                   `json:"memory_files,omitempty"`
	SessionProfileFields []AgentSessionProfileField `json:"session_profile_fields,omitempty"`
	Source               string                     `json:"source,omitempty"`
	Kind                 string                     `json:"kind,omitempty"`
	Description          string                     `json:"description,omitempty"`
	EntryPoint           bool                       `json:"entrypoint,omitempty"`
	Delegatable          bool                       `json:"delegatable,omitempty"`
	UIRoute              string                     `json:"ui_route,omitempty"`
	Capabilities         []string                   `json:"capabilities,omitempty"`
	Deliverables         []AgentDeliverable         `json:"deliverables,omitempty"`
	Metadata             map[string]string          `json:"metadata,omitempty"`
}

type AgentSessionProfileField struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	ReadOnly    bool   `json:"readonly,omitempty"`
}

type AgentDeliverable struct {
	ID                  string `json:"id"`
	Label               string `json:"label"`
	Description         string `json:"description,omitempty"`
	Format              string `json:"format,omitempty"`
	Required            bool   `json:"required,omitempty"`
	SessionAttributeKey string `json:"session_attribute_key,omitempty"`
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
	setAgentListMetadata(metadata, agentMemoryFilesMetadataKey, a.MemoryFiles)
	setAgentSessionProfileFieldsMetadata(metadata, a.SessionProfileFields)
	setAgentMetadata(metadata, agentSourceMetadataKey, a.Source)
	setAgentMetadata(metadata, agentKindMetadataKey, a.Kind)
	setAgentMetadata(metadata, agentDescriptionMetadataKey, a.Description)
	setAgentBoolMetadata(metadata, agentEntryPointMetadataKey, a.EntryPoint)
	setAgentBoolMetadata(metadata, agentDelegatableMetadataKey, a.Delegatable)
	setAgentMetadata(metadata, agentUIRouteMetadataKey, a.UIRoute)
	setAgentListMetadata(metadata, agentCapabilitiesMetadataKey, a.Capabilities)
	setAgentDeliverablesMetadata(metadata, a.Deliverables)
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
		ID:                   normalized.ID,
		Name:                 normalized.Name,
		Type:                 normalized.Type,
		Enabled:              normalized.Enabled,
		Scope:                normalized.Scope,
		Version:              normalized.Version,
		ProviderID:           strings.TrimSpace(metadata[agentProviderIDMetadataKey]),
		Model:                strings.TrimSpace(metadata[agentModelMetadataKey]),
		SystemPrompt:         metadata[agentSystemPromptMetadataKey],
		MaxIterations:        maxIterations,
		Tools:                parseAgentListMetadata(metadata[agentToolsMetadataKey]),
		Skills:               parseAgentListMetadata(metadata[agentSkillsMetadataKey]),
		MCPs:                 parseAgentListMetadata(metadata[agentMCPsMetadataKey]),
		MemoryFiles:          parseAgentListMetadata(metadata[agentMemoryFilesMetadataKey]),
		SessionProfileFields: parseAgentSessionProfileFieldsMetadata(metadata[agentSessionProfileFieldsKey]),
		Source:               normalizeAgentAttribute(metadata[agentSourceMetadataKey]),
		Kind:                 normalizeAgentAttribute(metadata[agentKindMetadataKey]),
		Description:          strings.TrimSpace(metadata[agentDescriptionMetadataKey]),
		EntryPoint:           parseAgentBoolMetadata(metadata[agentEntryPointMetadataKey]),
		Delegatable:          parseAgentBoolMetadata(metadata[agentDelegatableMetadataKey]),
		UIRoute:              strings.TrimSpace(metadata[agentUIRouteMetadataKey]),
		Capabilities:         parseAgentListMetadata(metadata[agentCapabilitiesMetadataKey]),
		Deliverables:         parseAgentDeliverablesMetadata(metadata[agentDeliverablesMetadataKey]),
		Metadata:             metadata,
	}
	delete(agent.Metadata, agentProviderIDMetadataKey)
	delete(agent.Metadata, agentModelMetadataKey)
	delete(agent.Metadata, agentSystemPromptMetadataKey)
	delete(agent.Metadata, agentMaxIterationsMetadataKey)
	delete(agent.Metadata, agentToolsMetadataKey)
	delete(agent.Metadata, agentSkillsMetadataKey)
	delete(agent.Metadata, agentMCPsMetadataKey)
	delete(agent.Metadata, agentMemoryFilesMetadataKey)
	delete(agent.Metadata, agentSessionProfileFieldsKey)
	delete(agent.Metadata, agentSourceMetadataKey)
	delete(agent.Metadata, agentKindMetadataKey)
	delete(agent.Metadata, agentDescriptionMetadataKey)
	delete(agent.Metadata, agentEntryPointMetadataKey)
	delete(agent.Metadata, agentDelegatableMetadataKey)
	delete(agent.Metadata, agentUIRouteMetadataKey)
	delete(agent.Metadata, agentCapabilitiesMetadataKey)
	delete(agent.Metadata, agentDeliverablesMetadataKey)
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

func setAgentBoolMetadata(metadata map[string]string, key string, value bool) {
	if metadata == nil {
		return
	}
	if !value {
		delete(metadata, key)
		return
	}
	metadata[key] = "true"
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

func setAgentSessionProfileFieldsMetadata(metadata map[string]string, fields []AgentSessionProfileField) {
	if metadata == nil {
		return
	}
	normalized := normalizeAgentSessionProfileFields(fields)
	if len(normalized) == 0 {
		delete(metadata, agentSessionProfileFieldsKey)
		return
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return
	}
	metadata[agentSessionProfileFieldsKey] = string(encoded)
}

func setAgentDeliverablesMetadata(metadata map[string]string, deliverables []AgentDeliverable) {
	if metadata == nil {
		return
	}
	normalized := normalizeAgentDeliverables(deliverables)
	if len(normalized) == 0 {
		delete(metadata, agentDeliverablesMetadataKey)
		return
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return
	}
	metadata[agentDeliverablesMetadataKey] = string(encoded)
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

func parseAgentBoolMetadata(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parseAgentSessionProfileFieldsMetadata(raw string) []AgentSessionProfileField {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	var fields []AgentSessionProfileField
	if err := json.Unmarshal([]byte(trimmed), &fields); err != nil {
		return nil
	}
	return normalizeAgentSessionProfileFields(fields)
}

func parseAgentDeliverablesMetadata(raw string) []AgentDeliverable {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	var deliverables []AgentDeliverable
	if err := json.Unmarshal([]byte(trimmed), &deliverables); err != nil {
		return nil
	}
	return normalizeAgentDeliverables(deliverables)
}

func normalizeAgentAttribute(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
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

func normalizeAgentSessionProfileFields(fields []AgentSessionProfileField) []AgentSessionProfileField {
	if len(fields) == 0 {
		return nil
	}
	items := make([]AgentSessionProfileField, 0, len(fields))
	seen := map[string]struct{}{}
	for _, field := range fields {
		key := strings.TrimSpace(field.Key)
		label := strings.TrimSpace(field.Label)
		if key == "" || label == "" {
			continue
		}
		lookup := strings.ToLower(key)
		if _, ok := seen[lookup]; ok {
			continue
		}
		seen[lookup] = struct{}{}
		items = append(items, AgentSessionProfileField{
			Key:         key,
			Label:       label,
			Description: strings.TrimSpace(field.Description),
			ReadOnly:    field.ReadOnly,
		})
	}
	if len(items) == 0 {
		return nil
	}
	return items
}

func normalizeAgentDeliverables(items []AgentDeliverable) []AgentDeliverable {
	if len(items) == 0 {
		return nil
	}
	normalized := make([]AgentDeliverable, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		label := strings.TrimSpace(item.Label)
		id := normalizeAgentDeliverableID(item.ID)
		if id == "" {
			id = normalizeAgentDeliverableID(label)
		}
		if id == "" || label == "" {
			continue
		}
		lookup := strings.ToLower(id)
		if _, ok := seen[lookup]; ok {
			continue
		}
		seen[lookup] = struct{}{}
		normalized = append(normalized, AgentDeliverable{
			ID:                  id,
			Label:               label,
			Description:         strings.TrimSpace(item.Description),
			Format:              strings.TrimSpace(item.Format),
			Required:            item.Required,
			SessionAttributeKey: strings.TrimSpace(item.SessionAttributeKey),
		})
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizeAgentDeliverableID(raw string) string {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return ""
	}
	var builder strings.Builder
	lastHyphen := false
	for _, r := range trimmed {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastHyphen = false
			continue
		}
		if lastHyphen {
			continue
		}
		builder.WriteRune('-')
		lastHyphen = true
	}
	return strings.Trim(builder.String(), "-")
}

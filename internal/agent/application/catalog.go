package application

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	controldomain "alter0/internal/control/domain"
	execdomain "alter0/internal/execution/domain"
)

type ManagedAgentSource interface {
	ResolveAgent(id string) (controldomain.Agent, bool)
	ListAgents() []controldomain.Agent
}

type Catalog struct {
	managed ManagedAgentSource
	builtin map[string]controldomain.Agent
	order   []string
}

func NewCatalog(managed ManagedAgentSource) *Catalog {
	builtins := builtinAgents()
	index := make(map[string]controldomain.Agent, len(builtins))
	order := make([]string, 0, len(builtins))
	for _, agent := range builtins {
		normalized := normalizeRuntimeAgent(agent)
		index[strings.ToLower(strings.TrimSpace(normalized.ID))] = normalized
		order = append(order, normalized.ID)
	}
	return &Catalog{
		managed: managed,
		builtin: index,
		order:   order,
	}
}

func (c *Catalog) ResolveAgent(id string) (controldomain.Agent, bool) {
	key := strings.ToLower(strings.TrimSpace(id))
	if key == "" {
		return controldomain.Agent{}, false
	}
	if item, ok := c.builtin[key]; ok {
		return cloneAgent(item), true
	}
	if c == nil || c.managed == nil {
		return controldomain.Agent{}, false
	}
	item, ok := c.managed.ResolveAgent(key)
	if !ok {
		return controldomain.Agent{}, false
	}
	return normalizeRuntimeAgent(item), true
}

func (c *Catalog) IsBuiltinID(id string) bool {
	_, ok := c.builtin[strings.ToLower(strings.TrimSpace(id))]
	return ok
}

func (c *Catalog) ListAgents() []controldomain.Agent {
	items := make([]controldomain.Agent, 0, len(c.order)+8)
	for _, id := range c.order {
		if builtin, ok := c.ResolveAgent(id); ok {
			items = append(items, builtin)
		}
	}
	if c == nil || c.managed == nil {
		return items
	}
	managed := c.managed.ListAgents()
	sort.Slice(managed, func(i, j int) bool {
		left := normalizeRuntimeAgent(managed[i])
		right := normalizeRuntimeAgent(managed[j])
		if left.Name == right.Name {
			return left.ID < right.ID
		}
		return left.Name < right.Name
	})
	for _, item := range managed {
		items = append(items, normalizeRuntimeAgent(item))
	}
	return items
}

func (c *Catalog) ListEntrypointAgents() []controldomain.Agent {
	items := c.ListAgents()
	filtered := make([]controldomain.Agent, 0, len(items))
	for _, item := range items {
		if !item.Enabled || !item.EntryPoint {
			continue
		}
		if item.Kind == controldomain.AgentKindMain || strings.EqualFold(strings.TrimSpace(item.ID), "main") {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func (c *Catalog) ListDelegatableAgents(excludeID string) []controldomain.Agent {
	excluded := strings.ToLower(strings.TrimSpace(excludeID))
	items := c.ListAgents()
	filtered := make([]controldomain.Agent, 0, len(items))
	for _, item := range items {
		if !item.Enabled || !item.Delegatable {
			continue
		}
		if strings.ToLower(strings.TrimSpace(item.ID)) == excluded {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func ApplyProfileMetadata(metadata map[string]string, agent controldomain.Agent) map[string]string {
	items := cloneStringMap(metadata)
	if strings.TrimSpace(items[execdomain.ExecutionEngineMetadataKey]) == "" {
		items[execdomain.ExecutionEngineMetadataKey] = execdomain.ExecutionEngineAgent
	}
	items[execdomain.AgentIDMetadataKey] = strings.TrimSpace(agent.ID)
	items[execdomain.AgentNameMetadataKey] = strings.TrimSpace(agent.Name)
	providerID := strings.TrimSpace(items[execdomain.LLMProviderIDMetadataKey])
	if providerID == "" {
		providerID = strings.TrimSpace(agent.ProviderID)
	}
	if providerID != "" {
		items[execdomain.AgentProviderIDMetadataKey] = providerID
	}
	modelID := strings.TrimSpace(items[execdomain.LLMModelMetadataKey])
	if modelID == "" {
		modelID = strings.TrimSpace(agent.Model)
	}
	if modelID != "" {
		items[execdomain.AgentModelMetadataKey] = modelID
	}
	if strings.TrimSpace(agent.SystemPrompt) != "" {
		items[execdomain.AgentSystemPromptMetadataKey] = strings.TrimSpace(agent.SystemPrompt)
	}
	if agent.MaxIterations > 0 {
		items[execdomain.AgentMaxIterationsMetadataKey] = strconv.Itoa(agent.MaxIterations)
	}
	applyDefaultListMetadata(items, execdomain.AgentToolsMetadataKey, agent.Tools)
	applyDefaultListMetadata(items, execdomain.AgentCapabilitiesMetadataKey, agent.Capabilities)
	applyDefaultListMetadata(items, "alter0.skills.include", agent.Skills)
	applyDefaultListMetadata(items, "alter0.mcp.request.enable", agent.MCPs)
	applyDefaultListMetadata(items, "alter0.memory.include", agent.MemoryFiles)
	for key, value := range agent.Metadata {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		if trimmedKey == "" || trimmedValue == "" {
			continue
		}
		items[trimmedKey] = trimmedValue
	}
	return items
}

func applyDefaultListMetadata(items map[string]string, key string, values []string) {
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" || len(values) == 0 {
		return
	}
	if _, exists := items[trimmedKey]; exists {
		return
	}
	if raw, err := json.Marshal(values); err == nil {
		items[trimmedKey] = string(raw)
	}
}

func normalizeRuntimeAgent(agent controldomain.Agent) controldomain.Agent {
	normalized := agent
	normalized.ID = strings.TrimSpace(normalized.ID)
	normalized.Name = strings.TrimSpace(normalized.Name)
	normalized.Source = strings.TrimSpace(normalized.Source)
	if normalized.Source == "" {
		normalized.Source = controldomain.AgentSourceManaged
	}
	normalized.Kind = strings.TrimSpace(normalized.Kind)
	if normalized.Kind == "" {
		normalized.Kind = controldomain.AgentKindCustom
	}
	normalized.Description = strings.TrimSpace(normalized.Description)
	normalized.UIRoute = strings.TrimSpace(normalized.UIRoute)
	normalized.Tools = cloneStringList(normalized.Tools)
	normalized.Skills = cloneStringList(normalized.Skills)
	normalized.MCPs = cloneStringList(normalized.MCPs)
	normalized.MemoryFiles = cloneStringList(normalized.MemoryFiles)
	normalized.Capabilities = cloneStringList(normalized.Capabilities)
	normalized.Metadata = cloneStringMap(normalized.Metadata)
	return normalized
}

func cloneAgent(agent controldomain.Agent) controldomain.Agent {
	return normalizeRuntimeAgent(agent)
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func cloneStringList(input []string) []string {
	if len(input) == 0 {
		return nil
	}
	out := make([]string, 0, len(input))
	seen := map[string]struct{}{}
	for _, item := range input {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

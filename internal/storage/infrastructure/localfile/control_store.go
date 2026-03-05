package localfile

import (
	"context"
	"path/filepath"
	"strings"
	"sync"

	controlapp "alter0/internal/control/application"
	controldomain "alter0/internal/control/domain"
)

type controlState struct {
	Channels          []controldomain.Channel          `json:"channels"`
	Capabilities      []controldomain.Capability       `json:"capabilities,omitempty"`
	CapabilityAudits  []controldomain.CapabilityAudit  `json:"capability_audits,omitempty"`
	Environments      map[string]string                `json:"environments,omitempty"`
	EnvironmentAudits []controldomain.EnvironmentAudit `json:"environment_audits,omitempty"`
	Skills            []controldomain.Skill            `json:"skills,omitempty"`
	MCPs              []controldomain.Capability       `json:"mcps,omitempty"`
}

type ControlStore struct {
	path   string
	format Format
	mu     sync.Mutex
}

func NewControlStore(baseDir string, format Format) *ControlStore {
	return &ControlStore{
		path:   filepath.Join(baseDir, "control."+extension(format)),
		format: format,
	}
}

var _ controlapp.Store = (*ControlStore)(nil)

func (s *ControlStore) Load(_ context.Context) ([]controldomain.Channel, []controldomain.Capability, []controldomain.CapabilityAudit, map[string]string, []controldomain.EnvironmentAudit, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	raw, ok, err := readIfExists(s.path)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if !ok {
		return []controldomain.Channel{}, []controldomain.Capability{}, []controldomain.CapabilityAudit{}, map[string]string{}, []controldomain.EnvironmentAudit{}, nil
	}

	state := controlState{}
	if err := unmarshalPayload(s.format, raw, &state); err != nil {
		return nil, nil, nil, nil, nil, err
	}

	mergedCapabilities := mergeCapabilities(state.Capabilities, state.Skills, state.MCPs)
	return state.Channels, mergedCapabilities, state.CapabilityAudits, cloneEnvironmentValues(state.Environments), cloneEnvironmentAudits(state.EnvironmentAudits), nil
}

func (s *ControlStore) Save(
	_ context.Context,
	channels []controldomain.Channel,
	capabilities []controldomain.Capability,
	audits []controldomain.CapabilityAudit,
	environments map[string]string,
	environmentAudits []controldomain.EnvironmentAudit,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	raw, err := marshalPayload(s.format, "alter0 control state", controlState{
		Channels:          channels,
		Capabilities:      capabilities,
		CapabilityAudits:  audits,
		Environments:      cloneEnvironmentValues(environments),
		EnvironmentAudits: cloneEnvironmentAudits(environmentAudits),
	})
	if err != nil {
		return err
	}
	return writeFile(s.path, raw)
}

func mergeCapabilities(primary []controldomain.Capability, skills []controldomain.Skill, mcps []controldomain.Capability) []controldomain.Capability {
	if len(primary) == 0 && len(skills) == 0 && len(mcps) == 0 {
		return []controldomain.Capability{}
	}

	merged := make(map[string]controldomain.Capability, len(primary)+len(skills)+len(mcps))
	for _, capability := range primary {
		normalized := capability.Normalized()
		merged[capabilityMapKey(normalized.Type, normalized.ID)] = normalized
	}
	for _, skill := range skills {
		capability := skill.AsCapability().Normalized()
		key := capabilityMapKey(capability.Type, capability.ID)
		if _, exists := merged[key]; exists {
			continue
		}
		merged[key] = capability
	}
	for _, mcp := range mcps {
		capability := mcp.Normalized()
		if capability.Type == "" {
			capability.Type = controldomain.CapabilityTypeMCP
		}
		key := capabilityMapKey(capability.Type, capability.ID)
		if _, exists := merged[key]; exists {
			continue
		}
		merged[key] = capability
	}

	out := make([]controldomain.Capability, 0, len(merged))
	for _, capability := range merged {
		out = append(out, capability)
	}
	return out
}

func capabilityMapKey(capabilityType controldomain.CapabilityType, id string) string {
	return strings.ToLower(strings.TrimSpace(string(capabilityType))) + ":" + strings.ToLower(strings.TrimSpace(id))
}

func cloneEnvironmentValues(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	copied := make(map[string]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}

func cloneEnvironmentAudits(items []controldomain.EnvironmentAudit) []controldomain.EnvironmentAudit {
	if len(items) == 0 {
		return []controldomain.EnvironmentAudit{}
	}
	copied := make([]controldomain.EnvironmentAudit, 0, len(items))
	for _, item := range items {
		cloned := item
		if len(item.Changes) > 0 {
			cloned.Changes = append([]controldomain.EnvironmentAuditChange(nil), item.Changes...)
		} else {
			cloned.Changes = []controldomain.EnvironmentAuditChange{}
		}
		copied = append(copied, cloned)
	}
	return copied
}

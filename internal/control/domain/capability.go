package domain

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

type CapabilityType string

const (
	CapabilityTypeSkill CapabilityType = "skill"
	CapabilityTypeMCP   CapabilityType = "mcp"
	CapabilityTypeAgent CapabilityType = "agent"
)

type CapabilityScope string

const (
	CapabilityScopeGlobal  CapabilityScope = "global"
	CapabilityScopeSession CapabilityScope = "session"
	CapabilityScopeRequest CapabilityScope = "request"
)

const DefaultCapabilityVersion = "v1.0.0"

var (
	capabilityIDPattern      = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
	capabilityVersionPattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)
)

type Capability struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Type     CapabilityType    `json:"type"`
	Enabled  bool              `json:"enabled"`
	Scope    CapabilityScope   `json:"scope"`
	Version  string            `json:"version"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

func (c Capability) Normalized() Capability {
	out := c
	out.ID = strings.TrimSpace(out.ID)
	out.Name = strings.TrimSpace(out.Name)
	out.Type = CapabilityType(strings.ToLower(strings.TrimSpace(string(out.Type))))
	out.Scope = CapabilityScope(strings.ToLower(strings.TrimSpace(string(out.Scope))))
	out.Version = strings.TrimSpace(out.Version)
	if out.Scope == "" {
		out.Scope = CapabilityScopeGlobal
	}
	if out.Version == "" {
		out.Version = DefaultCapabilityVersion
	}
	if len(out.Metadata) == 0 {
		out.Metadata = nil
	} else {
		out.Metadata = normalizeMetadata(out.Metadata)
	}
	return out
}

func (c Capability) Validate() error {
	normalized := c.Normalized()
	if normalized.ID == "" {
		return errors.New("capability id is required")
	}
	if !capabilityIDPattern.MatchString(normalized.ID) {
		return errors.New("capability id must match ^[a-z0-9][a-z0-9._-]*$")
	}
	if normalized.Name == "" {
		return errors.New("capability name is required")
	}
	if !normalized.Type.IsSupported() {
		return errors.New("capability type must be one of: skill, mcp")
	}
	if !normalized.Scope.IsSupported() {
		return errors.New("capability scope must be one of: global, session, request")
	}
	if !capabilityVersionPattern.MatchString(normalized.Version) {
		return fmt.Errorf("capability version %q must match vMAJOR.MINOR.PATCH", normalized.Version)
	}
	for key := range normalized.Metadata {
		if key == "" {
			return errors.New("capability metadata key cannot be empty")
		}
	}
	return nil
}

func (t CapabilityType) IsSupported() bool {
	switch t {
	case CapabilityTypeSkill, CapabilityTypeMCP, CapabilityTypeAgent:
		return true
	default:
		return false
	}
}

func (s CapabilityScope) IsSupported() bool {
	switch s {
	case CapabilityScopeGlobal, CapabilityScopeSession, CapabilityScopeRequest:
		return true
	default:
		return false
	}
}

func normalizeMetadata(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}

	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	normalized := make(map[string]string, len(input))
	for _, key := range keys {
		trimmedKey := strings.TrimSpace(key)
		normalized[trimmedKey] = strings.TrimSpace(input[key])
	}
	return normalized
}

type CapabilityLifecycleAction string

const (
	CapabilityLifecycleEnable  CapabilityLifecycleAction = "enable"
	CapabilityLifecycleDisable CapabilityLifecycleAction = "disable"
	CapabilityLifecycleUpdate  CapabilityLifecycleAction = "update"
	CapabilityLifecycleDelete  CapabilityLifecycleAction = "delete"
)

type CapabilityAudit struct {
	CapabilityID   string                    `json:"capability_id"`
	CapabilityType CapabilityType            `json:"capability_type"`
	Action         CapabilityLifecycleAction `json:"action"`
	Version        string                    `json:"version"`
	Enabled        bool                      `json:"enabled"`
	Scope          CapabilityScope           `json:"scope"`
	Metadata       map[string]string         `json:"metadata,omitempty"`
	OccurredAt     time.Time                 `json:"occurred_at"`
}

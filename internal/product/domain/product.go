package domain

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type Status string

const (
	StatusDraft    Status = "draft"
	StatusActive   Status = "active"
	StatusDisabled Status = "disabled"
	StatusArchived Status = "archived"
)

type Visibility string

const (
	VisibilityPublic  Visibility = "public"
	VisibilityPrivate Visibility = "private"
)

type OwnerType string

const (
	OwnerTypeBuiltin OwnerType = "builtin"
	OwnerTypeManaged OwnerType = "managed"
)

const DefaultVersion = "v1.0.0"

var (
	productIDPattern      = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
	productVersionPattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)
)

type WorkerAgent struct {
	AgentID        string   `json:"agent_id"`
	Role           string   `json:"role,omitempty"`
	Responsibility string   `json:"responsibility,omitempty"`
	Capabilities   []string `json:"capabilities,omitempty"`
	Enabled        bool     `json:"enabled"`
}

type Product struct {
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	Slug             string        `json:"slug"`
	Summary          string        `json:"summary,omitempty"`
	Status           Status        `json:"status"`
	Visibility       Visibility    `json:"visibility"`
	OwnerType        OwnerType     `json:"owner_type"`
	MasterAgentID    string        `json:"master_agent_id,omitempty"`
	EntryRoute       string        `json:"entry_route,omitempty"`
	Tags             []string      `json:"tags,omitempty"`
	Version          string        `json:"version"`
	WorkerAgents     []WorkerAgent `json:"worker_agents,omitempty"`
	ArtifactTypes    []string      `json:"artifact_types,omitempty"`
	KnowledgeSources []string      `json:"knowledge_sources,omitempty"`
}

func (p Product) Normalized() Product {
	out := p
	out.ID = normalizeID(out.ID)
	out.Name = strings.TrimSpace(out.Name)
	out.Slug = normalizeID(out.Slug)
	out.Summary = strings.TrimSpace(out.Summary)
	out.Status = Status(strings.ToLower(strings.TrimSpace(string(out.Status))))
	if out.Status == "" {
		out.Status = StatusDraft
	}
	out.Visibility = Visibility(strings.ToLower(strings.TrimSpace(string(out.Visibility))))
	if out.Visibility == "" {
		out.Visibility = VisibilityPrivate
	}
	out.OwnerType = OwnerType(strings.ToLower(strings.TrimSpace(string(out.OwnerType))))
	if out.OwnerType == "" {
		out.OwnerType = OwnerTypeManaged
	}
	out.MasterAgentID = normalizeID(out.MasterAgentID)
	out.EntryRoute = strings.TrimSpace(out.EntryRoute)
	out.Tags = normalizeStringList(out.Tags)
	out.Version = strings.TrimSpace(out.Version)
	if out.Version == "" {
		out.Version = DefaultVersion
	}
	out.ArtifactTypes = normalizeStringList(out.ArtifactTypes)
	out.KnowledgeSources = normalizeStringList(out.KnowledgeSources)
	out.WorkerAgents = normalizeWorkerAgents(out.WorkerAgents)
	return out
}

func (p Product) Validate() error {
	normalized := p.Normalized()
	if normalized.ID == "" {
		return errors.New("product id is required")
	}
	if !productIDPattern.MatchString(normalized.ID) {
		return errors.New("product id must match ^[a-z0-9][a-z0-9._-]*$")
	}
	if normalized.Name == "" {
		return errors.New("product name is required")
	}
	if normalized.Slug != "" && !productIDPattern.MatchString(normalized.Slug) {
		return errors.New("product slug must match ^[a-z0-9][a-z0-9._-]*$")
	}
	if !normalized.Status.IsSupported() {
		return errors.New("product status must be one of: draft, active, disabled, archived")
	}
	if !normalized.Visibility.IsSupported() {
		return errors.New("product visibility must be one of: public, private")
	}
	if !normalized.OwnerType.IsSupported() {
		return errors.New("product owner_type must be one of: builtin, managed")
	}
	if !productVersionPattern.MatchString(normalized.Version) {
		return fmt.Errorf("product version %q must match vMAJOR.MINOR.PATCH", normalized.Version)
	}
	if normalized.Status == StatusActive && normalized.MasterAgentID == "" {
		return errors.New("product master_agent_id is required when status is active")
	}
	for _, worker := range normalized.WorkerAgents {
		if worker.AgentID == "" {
			return errors.New("product worker agent_id is required")
		}
	}
	return nil
}

func (s Status) IsSupported() bool {
	switch s {
	case StatusDraft, StatusActive, StatusDisabled, StatusArchived:
		return true
	default:
		return false
	}
}

func (v Visibility) IsSupported() bool {
	switch v {
	case VisibilityPublic, VisibilityPrivate:
		return true
	default:
		return false
	}
}

func (o OwnerType) IsSupported() bool {
	switch o {
	case OwnerTypeBuiltin, OwnerTypeManaged:
		return true
	default:
		return false
	}
}

func normalizeWorkerAgents(items []WorkerAgent) []WorkerAgent {
	if len(items) == 0 {
		return nil
	}
	out := make([]WorkerAgent, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		normalized := WorkerAgent{
			AgentID:        normalizeID(item.AgentID),
			Role:           strings.TrimSpace(item.Role),
			Responsibility: strings.TrimSpace(item.Responsibility),
			Capabilities:   normalizeStringList(item.Capabilities),
			Enabled:        item.Enabled,
		}
		if normalized.AgentID == "" {
			continue
		}
		if _, ok := seen[normalized.AgentID]; ok {
			continue
		}
		seen[normalized.AgentID] = struct{}{}
		out = append(out, normalized)
	}
	if len(out) == 0 {
		return nil
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].AgentID < out[j].AgentID
	})
	return out
}

func normalizeStringList(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
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
	sort.Strings(out)
	return out
}

func normalizeID(raw string) string {
	return strings.TrimSpace(strings.ToLower(raw))
}

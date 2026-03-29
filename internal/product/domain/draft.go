package domain

import (
	"errors"
	"sort"
	"strings"
	"time"
)

type GenerationMode string

const (
	GenerationModeBootstrap GenerationMode = "bootstrap"
	GenerationModeExpand    GenerationMode = "expand"
)

type ReviewStatus string

const (
	ReviewStatusDraft     ReviewStatus = "draft"
	ReviewStatusReviewed  ReviewStatus = "reviewed"
	ReviewStatusPublished ReviewStatus = "published"
)

type ProductDraftGenerateInput struct {
	Name                    string         `json:"name"`
	Goal                    string         `json:"goal"`
	TargetUsers             []string       `json:"target_users,omitempty"`
	CoreCapabilities        []string       `json:"core_capabilities,omitempty"`
	Constraints             []string       `json:"constraints,omitempty"`
	ExpectedArtifacts       []string       `json:"expected_artifacts,omitempty"`
	IntegrationRequirements []string       `json:"integration_requirements,omitempty"`
	Mode                    GenerationMode `json:"mode,omitempty"`
}

type ProductAgentDraft struct {
	AgentID                string   `json:"agent_id"`
	Name                   string   `json:"name"`
	Description            string   `json:"description,omitempty"`
	SystemPrompt           string   `json:"system_prompt,omitempty"`
	MaxIterations          int      `json:"max_iterations,omitempty"`
	Tools                  []string `json:"tools,omitempty"`
	Skills                 []string `json:"skills,omitempty"`
	MCPs                   []string `json:"mcps,omitempty"`
	MemoryFiles            []string `json:"memory_files,omitempty"`
	Capabilities           []string `json:"capabilities,omitempty"`
	AllowedDelegateTargets []string `json:"allowed_delegate_targets,omitempty"`
	Enabled                bool     `json:"enabled"`
	Delegatable            bool     `json:"delegatable"`
}

type ProductWorkerDraft struct {
	AgentID                string   `json:"agent_id"`
	Name                   string   `json:"name"`
	Role                   string   `json:"role,omitempty"`
	Responsibility         string   `json:"responsibility,omitempty"`
	Description            string   `json:"description,omitempty"`
	SystemPrompt           string   `json:"system_prompt,omitempty"`
	InputContract          string   `json:"input_contract,omitempty"`
	OutputContract         string   `json:"output_contract,omitempty"`
	AllowedTools           []string `json:"allowed_tools,omitempty"`
	AllowedDelegateTargets []string `json:"allowed_delegate_targets,omitempty"`
	Dependencies           []string `json:"dependencies,omitempty"`
	Skills                 []string `json:"skills,omitempty"`
	MCPs                   []string `json:"mcps,omitempty"`
	MemoryFiles            []string `json:"memory_files,omitempty"`
	Capabilities           []string `json:"capabilities,omitempty"`
	Priority               int      `json:"priority,omitempty"`
	MaxIterations          int      `json:"max_iterations,omitempty"`
	Enabled                bool     `json:"enabled"`
}

type ProductDraft struct {
	DraftID                 string               `json:"draft_id"`
	Product                 Product              `json:"product"`
	Mode                    GenerationMode       `json:"mode"`
	ReviewStatus            ReviewStatus         `json:"review_status"`
	GeneratedBy             string               `json:"generated_by"`
	GeneratedAt             time.Time            `json:"generated_at"`
	UpdatedAt               time.Time            `json:"updated_at"`
	Goal                    string               `json:"goal,omitempty"`
	TargetUsers             []string             `json:"target_users,omitempty"`
	CoreCapabilities        []string             `json:"core_capabilities,omitempty"`
	Constraints             []string             `json:"constraints,omitempty"`
	ExpectedArtifacts       []string             `json:"expected_artifacts,omitempty"`
	IntegrationRequirements []string             `json:"integration_requirements,omitempty"`
	ConflictSuggestions     []string             `json:"conflict_suggestions,omitempty"`
	MasterAgent             ProductAgentDraft    `json:"master_agent"`
	WorkerMatrix            []ProductWorkerDraft `json:"worker_matrix,omitempty"`
	PublishedProductID      string               `json:"published_product_id,omitempty"`
}

func (in ProductDraftGenerateInput) Normalized() ProductDraftGenerateInput {
	out := in
	out.Name = strings.TrimSpace(out.Name)
	out.Goal = strings.TrimSpace(out.Goal)
	out.TargetUsers = normalizeStringList(out.TargetUsers)
	out.CoreCapabilities = normalizeStringList(out.CoreCapabilities)
	out.Constraints = normalizeStringList(out.Constraints)
	out.ExpectedArtifacts = normalizeStringList(out.ExpectedArtifacts)
	out.IntegrationRequirements = normalizeStringList(out.IntegrationRequirements)
	out.Mode = GenerationMode(strings.ToLower(strings.TrimSpace(string(out.Mode))))
	if out.Mode == "" {
		out.Mode = GenerationModeBootstrap
	}
	return out
}

func (in ProductDraftGenerateInput) Validate() error {
	normalized := in.Normalized()
	if normalized.Name == "" {
		return errors.New("draft name is required")
	}
	if normalized.Goal == "" {
		return errors.New("draft goal is required")
	}
	if !normalized.Mode.IsSupported() {
		return errors.New("draft mode must be one of: bootstrap, expand")
	}
	return nil
}

func (d ProductDraft) Normalized() ProductDraft {
	out := d
	out.DraftID = normalizeID(out.DraftID)
	out.Product = out.Product.Normalized()
	out.Mode = GenerationMode(strings.ToLower(strings.TrimSpace(string(out.Mode))))
	if out.Mode == "" {
		out.Mode = GenerationModeBootstrap
	}
	out.ReviewStatus = ReviewStatus(strings.ToLower(strings.TrimSpace(string(out.ReviewStatus))))
	if out.ReviewStatus == "" {
		out.ReviewStatus = ReviewStatusDraft
	}
	out.GeneratedBy = normalizeID(out.GeneratedBy)
	out.Goal = strings.TrimSpace(out.Goal)
	out.TargetUsers = normalizeStringList(out.TargetUsers)
	out.CoreCapabilities = normalizeStringList(out.CoreCapabilities)
	out.Constraints = normalizeStringList(out.Constraints)
	out.ExpectedArtifacts = normalizeStringList(out.ExpectedArtifacts)
	out.IntegrationRequirements = normalizeStringList(out.IntegrationRequirements)
	out.ConflictSuggestions = normalizeStringList(out.ConflictSuggestions)
	out.MasterAgent = out.MasterAgent.Normalized()
	out.WorkerMatrix = normalizeProductWorkerDrafts(out.WorkerMatrix)
	out.PublishedProductID = normalizeID(out.PublishedProductID)
	return out
}

func (d ProductDraft) Validate() error {
	normalized := d.Normalized()
	if normalized.DraftID == "" {
		return errors.New("draft_id is required")
	}
	if !normalized.Mode.IsSupported() {
		return errors.New("draft mode must be one of: bootstrap, expand")
	}
	if !normalized.ReviewStatus.IsSupported() {
		return errors.New("review_status must be one of: draft, reviewed, published")
	}
	if err := normalized.Product.Validate(); err != nil {
		return err
	}
	if err := normalized.MasterAgent.Validate(); err != nil {
		return err
	}
	for _, worker := range normalized.WorkerMatrix {
		if err := worker.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (a ProductAgentDraft) Normalized() ProductAgentDraft {
	out := a
	out.AgentID = normalizeID(out.AgentID)
	out.Name = strings.TrimSpace(out.Name)
	out.Description = strings.TrimSpace(out.Description)
	out.SystemPrompt = strings.TrimSpace(out.SystemPrompt)
	if out.MaxIterations <= 0 {
		out.MaxIterations = 6
	}
	out.Tools = normalizeStringList(out.Tools)
	out.Skills = normalizeStringList(out.Skills)
	out.MCPs = normalizeStringList(out.MCPs)
	out.MemoryFiles = normalizeStringList(out.MemoryFiles)
	out.Capabilities = normalizeStringList(out.Capabilities)
	out.AllowedDelegateTargets = normalizeStringList(out.AllowedDelegateTargets)
	return out
}

func (a ProductAgentDraft) Validate() error {
	normalized := a.Normalized()
	if normalized.AgentID == "" {
		return errors.New("draft agent_id is required")
	}
	if normalized.Name == "" {
		return errors.New("draft agent name is required")
	}
	return nil
}

func (w ProductWorkerDraft) Normalized() ProductWorkerDraft {
	out := w
	out.AgentID = normalizeID(out.AgentID)
	out.Name = strings.TrimSpace(out.Name)
	out.Role = strings.TrimSpace(out.Role)
	out.Responsibility = strings.TrimSpace(out.Responsibility)
	out.Description = strings.TrimSpace(out.Description)
	out.SystemPrompt = strings.TrimSpace(out.SystemPrompt)
	out.InputContract = strings.TrimSpace(out.InputContract)
	out.OutputContract = strings.TrimSpace(out.OutputContract)
	out.AllowedTools = normalizeStringList(out.AllowedTools)
	out.AllowedDelegateTargets = normalizeStringList(out.AllowedDelegateTargets)
	out.Dependencies = normalizeStringList(out.Dependencies)
	out.Skills = normalizeStringList(out.Skills)
	out.MCPs = normalizeStringList(out.MCPs)
	out.MemoryFiles = normalizeStringList(out.MemoryFiles)
	out.Capabilities = normalizeStringList(out.Capabilities)
	if out.Priority < 0 {
		out.Priority = 0
	}
	if out.MaxIterations <= 0 {
		out.MaxIterations = 4
	}
	return out
}

func (w ProductWorkerDraft) Validate() error {
	normalized := w.Normalized()
	if normalized.AgentID == "" {
		return errors.New("draft worker agent_id is required")
	}
	if normalized.Name == "" {
		return errors.New("draft worker name is required")
	}
	if normalized.Role == "" {
		return errors.New("draft worker role is required")
	}
	return nil
}

func (m GenerationMode) IsSupported() bool {
	switch m {
	case GenerationModeBootstrap, GenerationModeExpand:
		return true
	default:
		return false
	}
}

func (s ReviewStatus) IsSupported() bool {
	switch s {
	case ReviewStatusDraft, ReviewStatusReviewed, ReviewStatusPublished:
		return true
	default:
		return false
	}
}

func normalizeProductWorkerDrafts(items []ProductWorkerDraft) []ProductWorkerDraft {
	if len(items) == 0 {
		return nil
	}
	out := make([]ProductWorkerDraft, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		normalized := item.Normalized()
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
		if out[i].Priority == out[j].Priority {
			return out[i].AgentID < out[j].AgentID
		}
		return out[i].Priority < out[j].Priority
	})
	return out
}

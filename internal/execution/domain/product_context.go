package domain

const (
	ProductContextMetadataKey     = "alter0.product_context"
	ProductContextProtocolVersion = "alter0.product-context/v1"
)

type ProductContext struct {
	Protocol         string                   `json:"protocol"`
	ProductID        string                   `json:"product_id"`
	Name             string                   `json:"name"`
	Slug             string                   `json:"slug,omitempty"`
	Summary          string                   `json:"summary,omitempty"`
	Status           string                   `json:"status,omitempty"`
	Visibility       string                   `json:"visibility,omitempty"`
	OwnerType        string                   `json:"owner_type,omitempty"`
	MasterAgentID    string                   `json:"master_agent_id,omitempty"`
	EntryRoute       string                   `json:"entry_route,omitempty"`
	Tags             []string                 `json:"tags,omitempty"`
	ArtifactTypes    []string                 `json:"artifact_types,omitempty"`
	KnowledgeSources []string                 `json:"knowledge_sources,omitempty"`
	WorkerAgents     []ProductWorkerAgentSpec `json:"worker_agents,omitempty"`
}

type ProductWorkerAgentSpec struct {
	AgentID        string   `json:"agent_id"`
	Role           string   `json:"role,omitempty"`
	Responsibility string   `json:"responsibility,omitempty"`
	Capabilities   []string `json:"capabilities,omitempty"`
	Enabled        bool     `json:"enabled"`
}

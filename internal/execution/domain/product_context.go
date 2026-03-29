package domain

const (
	ProductContextMetadataKey         = "alter0.product_context"
	ProductContextProtocolVersion     = "alter0.product-context/v1"
	ProductDiscoveryMetadataKey       = "alter0.product.discovery"
	ProductDiscoveryProtocolVersion   = "alter0.product-discovery/v1"
	ProductMatchedIDsMetadataKey      = "matched_product_ids"
	ProductSelectedIDMetadataKey      = "selected_product_id"
	ProductSelectionReasonMetadataKey = "selection_reason"
	ProductMasterAgentMetadataKey     = "master_agent_id"
	ProductExecutionModeMetadataKey   = "product_execution_mode"
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

type ProductDiscoveryContext struct {
	Protocol        string           `json:"protocol"`
	MatchedProducts []ProductContext `json:"matched_products,omitempty"`
	SelectedProduct string           `json:"selected_product_id,omitempty"`
	SelectionReason string           `json:"selection_reason,omitempty"`
	ExecutionMode   string           `json:"product_execution_mode,omitempty"`
}

type ProductWorkerAgentSpec struct {
	AgentID        string   `json:"agent_id"`
	Role           string   `json:"role,omitempty"`
	Responsibility string   `json:"responsibility,omitempty"`
	Capabilities   []string `json:"capabilities,omitempty"`
	Enabled        bool     `json:"enabled"`
}

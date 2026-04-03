package application

import (
	"strings"
	"testing"

	productdomain "alter0/internal/product/domain"
)

func TestDraftServiceGenerateDraftBuildsSingleTravelMaster(t *testing.T) {
	products := NewService()
	drafts := NewDraftService(products)

	draft, err := drafts.GenerateDraft(productdomain.ProductDraftGenerateInput{
		Name:              "Travel Premium",
		Goal:              "Generate city travel guides with metro, food, and route planning.",
		CoreCapabilities:  []string{"city guide", "itinerary", "metro", "food", "map"},
		ExpectedArtifacts: []string{"city_guide", "itinerary", "map_layers"},
	})
	if err != nil {
		t.Fatalf("generate draft failed: %v", err)
	}
	if draft.Mode != productdomain.GenerationModeBootstrap {
		t.Fatalf("expected bootstrap mode, got %s", draft.Mode)
	}
	if draft.Product.ID != "travel-premium" {
		t.Fatalf("expected generated product id travel-premium, got %s", draft.Product.ID)
	}
	if draft.MasterAgent.AgentID != "travel-premium-master" {
		t.Fatalf("expected master agent id, got %+v", draft.MasterAgent)
	}
	if got := draft.MasterAgent.Tools; len(got) != 4 || got[0] != "codex_exec" {
		t.Fatalf("expected codex-driven master tools, got %+v", got)
	}
	if !strings.Contains(draft.MasterAgent.SystemPrompt, "codex_exec") || !strings.Contains(strings.ToLower(draft.MasterAgent.SystemPrompt), "assistant") {
		t.Fatalf("expected codex assistant master prompt, got %q", draft.MasterAgent.SystemPrompt)
	}
	if !strings.Contains(draft.MasterAgent.SystemPrompt, "SKILL.md") {
		t.Fatalf("expected travel master prompt to mention private skill, got %q", draft.MasterAgent.SystemPrompt)
	}
	if got := draft.MasterAgent.Skills; len(got) != 1 || got[0] != "memory" {
		t.Fatalf("expected memory skill only, got %+v", got)
	}
	if len(draft.WorkerMatrix) != 0 {
		t.Fatalf("expected single-agent travel draft, got %+v", draft.WorkerMatrix)
	}
	if len(draft.Product.WorkerAgents) != 0 {
		t.Fatalf("expected single-agent travel product, got %+v", draft.Product.WorkerAgents)
	}
	if len(draft.Product.ArtifactTypes) == 0 || len(draft.Product.KnowledgeSources) == 0 {
		t.Fatalf("expected artifact types and knowledge sources, got %+v", draft.Product)
	}
}

func TestDraftServiceGenerateMatrixDraftKeepsExistingWorkersWithoutAddingNewOnes(t *testing.T) {
	products := NewService()
	if _, err := products.CreateProduct(productdomain.Product{
		Name:          "Research Hub",
		Slug:          "research-hub",
		Summary:       "Research workflows.",
		Status:        productdomain.StatusDraft,
		Visibility:    productdomain.VisibilityPrivate,
		MasterAgentID: "research-hub-master",
		WorkerAgents: []productdomain.WorkerAgent{
			{AgentID: "research-hub-research", Role: "research", Responsibility: "Research domain context", Enabled: true},
		},
	}); err != nil {
		t.Fatalf("seed product failed: %v", err)
	}
	drafts := NewDraftService(products)

	draft, err := drafts.GenerateMatrixDraft("research-hub", productdomain.ProductDraftGenerateInput{
		Name:             "Research Hub Expansion",
		Goal:             "Add delivery and planning specialists.",
		CoreCapabilities: []string{"planning", "delivery"},
	})
	if err != nil {
		t.Fatalf("generate matrix draft failed: %v", err)
	}
	if draft.Mode != productdomain.GenerationModeExpand {
		t.Fatalf("expected expand mode, got %s", draft.Mode)
	}
	if len(draft.WorkerMatrix) != 0 {
		t.Fatalf("expected no generated worker drafts, got %+v", draft.WorkerMatrix)
	}
	if len(draft.Product.WorkerAgents) != 1 {
		t.Fatalf("expected existing worker agents to remain unchanged, got %+v", draft.Product.WorkerAgents)
	}
	if got := draft.MasterAgent.Tools; len(got) != 4 || got[0] != "codex_exec" {
		t.Fatalf("expected single-agent master tools, got %+v", got)
	}
}

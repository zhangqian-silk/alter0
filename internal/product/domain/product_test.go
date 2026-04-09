package domain

import (
	"testing"
	"time"
)

func TestProductNormalizedAppliesStableDefaultsAndOrdering(t *testing.T) {
	product := Product{
		ID:            " Travel.Planner ",
		Name:          "  Travel Planner  ",
		Slug:          " Travel ",
		Summary:       "  City guide  ",
		Status:        " ACTIVE ",
		Visibility:    " PUBLIC ",
		OwnerType:     " BUILTIN ",
		MasterAgentID: " Travel-Master ",
		EntryRoute:    "  /products/travel  ",
		Tags:          []string{" Guide ", "guide", "", "Trip"},
		ArtifactTypes: []string{" html ", "HTML", "json"},
		KnowledgeSources: []string{
			" Skill ",
			"skill",
			"Memory",
		},
		WorkerAgents: []WorkerAgent{
			{AgentID: " zeta ", Role: " maps ", Responsibility: " routes ", Capabilities: []string{" map ", "MAP"}, Enabled: true},
			{AgentID: ""},
			{AgentID: "alpha", Role: " food "},
			{AgentID: "ZETA", Role: "duplicate is ignored"},
		},
	}

	normalized := product.Normalized()

	if normalized.ID != "travel.planner" {
		t.Fatalf("ID = %q, want travel.planner", normalized.ID)
	}
	if normalized.Name != "Travel Planner" {
		t.Fatalf("Name = %q, want Travel Planner", normalized.Name)
	}
	if normalized.Slug != "travel" {
		t.Fatalf("Slug = %q, want travel", normalized.Slug)
	}
	if normalized.Status != StatusActive {
		t.Fatalf("Status = %q, want %q", normalized.Status, StatusActive)
	}
	if normalized.Visibility != VisibilityPublic {
		t.Fatalf("Visibility = %q, want %q", normalized.Visibility, VisibilityPublic)
	}
	if normalized.OwnerType != OwnerTypeBuiltin {
		t.Fatalf("OwnerType = %q, want %q", normalized.OwnerType, OwnerTypeBuiltin)
	}
	if normalized.MasterAgentID != "travel-master" {
		t.Fatalf("MasterAgentID = %q, want travel-master", normalized.MasterAgentID)
	}
	if normalized.Version != DefaultVersion {
		t.Fatalf("Version = %q, want %q", normalized.Version, DefaultVersion)
	}
	assertStringSlice(t, normalized.Tags, []string{"Guide", "Trip"})
	assertStringSlice(t, normalized.ArtifactTypes, []string{"html", "json"})
	assertStringSlice(t, normalized.KnowledgeSources, []string{"Memory", "Skill"})
	if len(normalized.WorkerAgents) != 2 {
		t.Fatalf("WorkerAgents length = %d, want 2: %+v", len(normalized.WorkerAgents), normalized.WorkerAgents)
	}
	if normalized.WorkerAgents[0].AgentID != "alpha" || normalized.WorkerAgents[1].AgentID != "zeta" {
		t.Fatalf("WorkerAgents order = %+v, want alpha then zeta", normalized.WorkerAgents)
	}
	assertStringSlice(t, normalized.WorkerAgents[1].Capabilities, []string{"map"})
}

func TestProductValidateCoversIdentityAndActiveMasterAgent(t *testing.T) {
	valid := Product{
		ID:            "travel",
		Name:          "Travel",
		Status:        StatusActive,
		Visibility:    VisibilityPublic,
		OwnerType:     OwnerTypeBuiltin,
		MasterAgentID: "travel-master",
		Version:       DefaultVersion,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	cases := []struct {
		name    string
		mutate  func(Product) Product
		wantErr string
	}{
		{
			name:    "blank id",
			mutate:  func(p Product) Product { p.ID = " "; return p },
			wantErr: "product id is required",
		},
		{
			name:    "invalid id",
			mutate:  func(p Product) Product { p.ID = "-bad"; return p },
			wantErr: "product id must match",
		},
		{
			name:    "blank name",
			mutate:  func(p Product) Product { p.Name = " "; return p },
			wantErr: "product name is required",
		},
		{
			name:    "invalid slug",
			mutate:  func(p Product) Product { p.Slug = " bad slug "; return p },
			wantErr: "product slug must match",
		},
		{
			name:    "unsupported status",
			mutate:  func(p Product) Product { p.Status = "paused"; return p },
			wantErr: "product status must be one of",
		},
		{
			name:    "unsupported visibility",
			mutate:  func(p Product) Product { p.Visibility = "team"; return p },
			wantErr: "product visibility must be one of",
		},
		{
			name:    "unsupported owner",
			mutate:  func(p Product) Product { p.OwnerType = "external"; return p },
			wantErr: "product owner_type must be one of",
		},
		{
			name:    "invalid version",
			mutate:  func(p Product) Product { p.Version = "1.0.0"; return p },
			wantErr: "must match vMAJOR.MINOR.PATCH",
		},
		{
			name:    "active without master agent",
			mutate:  func(p Product) Product { p.MasterAgentID = " "; return p },
			wantErr: "product master_agent_id is required when status is active",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.mutate(valid).Validate()
			if err == nil {
				t.Fatalf("Validate() error = nil, want containing %q", tc.wantErr)
			}
			assertErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestProductDraftGenerateInputNormalizeAndValidate(t *testing.T) {
	input := ProductDraftGenerateInput{
		Name:                    " Product Studio ",
		Goal:                    " Build products ",
		TargetUsers:             []string{" PM ", "pm", "R&D"},
		CoreCapabilities:        []string{" draft ", "publish"},
		Constraints:             []string{" TDD ", "tdd"},
		ExpectedArtifacts:       []string{" HTML ", ""},
		IntegrationRequirements: []string{" Agent ", "agent"},
		Mode:                    " EXPAND ",
	}

	normalized := input.Normalized()

	if normalized.Name != "Product Studio" || normalized.Goal != "Build products" {
		t.Fatalf("Normalized identity = (%q, %q), want trimmed values", normalized.Name, normalized.Goal)
	}
	if normalized.Mode != GenerationModeExpand {
		t.Fatalf("Mode = %q, want %q", normalized.Mode, GenerationModeExpand)
	}
	assertStringSlice(t, normalized.TargetUsers, []string{"PM", "R&D"})
	assertStringSlice(t, normalized.Constraints, []string{"TDD"})
	assertStringSlice(t, normalized.ExpectedArtifacts, []string{"HTML"})
	assertStringSlice(t, normalized.IntegrationRequirements, []string{"Agent"})

	defaulted := ProductDraftGenerateInput{Name: "A", Goal: "B"}.Normalized()
	if defaulted.Mode != GenerationModeBootstrap {
		t.Fatalf("default Mode = %q, want %q", defaulted.Mode, GenerationModeBootstrap)
	}

	for _, tc := range []ProductDraftGenerateInput{
		{Goal: "goal"},
		{Name: "name"},
		{Name: "name", Goal: "goal", Mode: "unknown"},
	} {
		if err := tc.Validate(); err == nil {
			t.Fatalf("Validate(%+v) error = nil, want error", tc)
		}
	}
}

func TestProductDraftNormalizeAndValidateWorkerMatrix(t *testing.T) {
	now := time.Date(2026, 4, 8, 8, 0, 0, 0, time.UTC)
	draft := ProductDraft{
		DraftID:     " Draft-01 ",
		Product:     Product{ID: " Travel ", Name: "Travel"},
		GeneratedBy: " Product-Builder ",
		GeneratedAt: now,
		UpdatedAt:   now,
		Goal:        " Guide ",
		TargetUsers: []string{" traveler ", "Traveler"},
		MasterAgent: ProductAgentDraft{AgentID: " Travel-Master ", Name: " Travel Master ", Tools: []string{" codex ", "codex"}},
		WorkerMatrix: []ProductWorkerDraft{
			{AgentID: "food", Name: "Food", Role: "Dining", Priority: 20},
			{AgentID: " map ", Name: " Map ", Role: "Transit", Priority: -1, MaxIterations: -2},
			{AgentID: "food", Name: "Duplicate", Role: "Dining", Priority: 1},
			{AgentID: ""},
		},
		PublishedProductID: " Travel ",
	}

	normalized := draft.Normalized()

	if normalized.DraftID != "draft-01" {
		t.Fatalf("DraftID = %q, want draft-01", normalized.DraftID)
	}
	if normalized.Mode != GenerationModeBootstrap {
		t.Fatalf("Mode = %q, want %q", normalized.Mode, GenerationModeBootstrap)
	}
	if normalized.ReviewStatus != ReviewStatusDraft {
		t.Fatalf("ReviewStatus = %q, want %q", normalized.ReviewStatus, ReviewStatusDraft)
	}
	if normalized.MasterAgent.AgentID != "travel-master" || normalized.MasterAgent.MaxIterations != 6 {
		t.Fatalf("MasterAgent = %+v, want normalized id and default max iterations", normalized.MasterAgent)
	}
	assertStringSlice(t, normalized.MasterAgent.Tools, []string{"codex"})
	if len(normalized.WorkerMatrix) != 2 {
		t.Fatalf("WorkerMatrix length = %d, want 2: %+v", len(normalized.WorkerMatrix), normalized.WorkerMatrix)
	}
	if normalized.WorkerMatrix[0].AgentID != "map" || normalized.WorkerMatrix[0].Priority != 0 || normalized.WorkerMatrix[0].MaxIterations != 4 {
		t.Fatalf("WorkerMatrix[0] = %+v, want map with default priority and max iterations", normalized.WorkerMatrix[0])
	}
	if normalized.WorkerMatrix[1].AgentID != "food" {
		t.Fatalf("WorkerMatrix[1] = %+v, want food", normalized.WorkerMatrix[1])
	}
	if normalized.PublishedProductID != "travel" {
		t.Fatalf("PublishedProductID = %q, want travel", normalized.PublishedProductID)
	}
	if err := normalized.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestProductDraftValidateRejectsInvalidDraftAndAgentFields(t *testing.T) {
	valid := ProductDraft{
		DraftID:      "draft-01",
		Product:      Product{ID: "travel", Name: "Travel"},
		Mode:         GenerationModeBootstrap,
		ReviewStatus: ReviewStatusDraft,
		MasterAgent:  ProductAgentDraft{AgentID: "travel-master", Name: "Travel Master"},
	}

	cases := []struct {
		name    string
		mutate  func(ProductDraft) ProductDraft
		wantErr string
	}{
		{
			name:    "blank draft id",
			mutate:  func(d ProductDraft) ProductDraft { d.DraftID = " "; return d },
			wantErr: "draft_id is required",
		},
		{
			name:    "unsupported mode",
			mutate:  func(d ProductDraft) ProductDraft { d.Mode = "unknown"; return d },
			wantErr: "draft mode must be one of",
		},
		{
			name:    "unsupported review status",
			mutate:  func(d ProductDraft) ProductDraft { d.ReviewStatus = "approved"; return d },
			wantErr: "review_status must be one of",
		},
		{
			name:    "blank master agent",
			mutate:  func(d ProductDraft) ProductDraft { d.MasterAgent.AgentID = " "; return d },
			wantErr: "draft agent_id is required",
		},
		{
			name: "blank worker role",
			mutate: func(d ProductDraft) ProductDraft {
				d.WorkerMatrix = []ProductWorkerDraft{{AgentID: "w1", Name: "Worker"}}
				return d
			},
			wantErr: "draft worker role is required",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.mutate(valid).Validate()
			if err == nil {
				t.Fatalf("Validate() error = nil, want containing %q", tc.wantErr)
			}
			assertErrorContains(t, err, tc.wantErr)
		})
	}
}

func assertStringSlice(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("slice length = %d, want %d: got=%v want=%v", len(got), len(want), got, want)
	}
	for idx := range want {
		if got[idx] != want[idx] {
			t.Fatalf("slice[%d] = %q, want %q: got=%v want=%v", idx, got[idx], want[idx], got, want)
		}
	}
}

func assertErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want containing %q", want)
	}
	if !contains(err.Error(), want) {
		t.Fatalf("error = %q, want containing %q", err.Error(), want)
	}
}

func contains(value string, want string) bool {
	for idx := 0; idx+len(want) <= len(value); idx++ {
		if value[idx:idx+len(want)] == want {
			return true
		}
	}
	return false
}

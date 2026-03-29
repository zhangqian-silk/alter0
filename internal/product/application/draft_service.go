package application

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	productdomain "alter0/internal/product/domain"
)

type DraftStore interface {
	Load(ctx context.Context) ([]productdomain.ProductDraft, error)
	Save(ctx context.Context, drafts []productdomain.ProductDraft) error
}

type productCatalog interface {
	ListProducts() []productdomain.Product
	ResolveProduct(id string) (productdomain.Product, bool)
}

type DraftService struct {
	mu       sync.RWMutex
	store    DraftStore
	products productCatalog
	drafts   map[string]productdomain.ProductDraft
	now      func() time.Time
}

func NewDraftService(products productCatalog) *DraftService {
	return newDraftService(nil, products)
}

func NewDraftServiceWithStore(ctx context.Context, store DraftStore, products productCatalog) (*DraftService, error) {
	service := newDraftService(store, products)
	if store == nil {
		return service, nil
	}
	items, err := store.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("load product drafts: %w", err)
	}
	for _, item := range items {
		normalized := item.Normalized()
		if normalized.DraftID == "" {
			continue
		}
		if err := normalized.Validate(); err != nil {
			return nil, fmt.Errorf("invalid product draft in store: %w", err)
		}
		service.drafts[normalized.DraftID] = cloneDraft(normalized)
	}
	return service, nil
}

func newDraftService(store DraftStore, products productCatalog) *DraftService {
	return &DraftService{
		store:    store,
		products: products,
		drafts:   map[string]productdomain.ProductDraft{},
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *DraftService) ListDrafts() []productdomain.ProductDraft {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]productdomain.ProductDraft, 0, len(s.drafts))
	for _, item := range s.drafts {
		items = append(items, cloneDraft(item))
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].DraftID < items[j].DraftID
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	return items
}

func (s *DraftService) GetDraft(id string) (productdomain.ProductDraft, bool) {
	key := normalizeID(id)
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.drafts[key]
	if !ok {
		return productdomain.ProductDraft{}, false
	}
	return cloneDraft(item), true
}

func (s *DraftService) GenerateDraft(input productdomain.ProductDraftGenerateInput) (productdomain.ProductDraft, error) {
	normalized := input.Normalized()
	if err := normalized.Validate(); err != nil {
		return productdomain.ProductDraft{}, err
	}
	if normalized.Mode != productdomain.GenerationModeBootstrap {
		return productdomain.ProductDraft{}, errors.New("generate draft only supports bootstrap mode")
	}
	now := s.now().UTC()
	draft := s.buildBootstrapDraft(normalized, now)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.drafts[draft.DraftID] = cloneDraft(draft)
	if err := s.storeLocked(); err != nil {
		delete(s.drafts, draft.DraftID)
		return productdomain.ProductDraft{}, err
	}
	return cloneDraft(draft), nil
}

func (s *DraftService) GenerateMatrixDraft(productID string, input productdomain.ProductDraftGenerateInput) (productdomain.ProductDraft, error) {
	normalized := input.Normalized()
	if err := normalized.Validate(); err != nil {
		return productdomain.ProductDraft{}, err
	}
	baseProduct, ok := s.products.ResolveProduct(productID)
	if !ok {
		return productdomain.ProductDraft{}, errors.New("product not found")
	}
	now := s.now().UTC()
	draft := s.buildExpandDraft(baseProduct, normalized, now)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.drafts[draft.DraftID] = cloneDraft(draft)
	if err := s.storeLocked(); err != nil {
		delete(s.drafts, draft.DraftID)
		return productdomain.ProductDraft{}, err
	}
	return cloneDraft(draft), nil
}

func (s *DraftService) SaveDraft(id string, draft productdomain.ProductDraft) (productdomain.ProductDraft, error) {
	key := normalizeID(id)
	if key == "" {
		return productdomain.ProductDraft{}, errors.New("draft_id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	previous, ok := s.drafts[key]
	if !ok {
		return productdomain.ProductDraft{}, errors.New("product draft not found")
	}
	draft.DraftID = key
	draft.GeneratedAt = previous.GeneratedAt
	draft.GeneratedBy = previous.GeneratedBy
	draft.UpdatedAt = s.now().UTC()
	normalized := draft.Normalized()
	if normalized.ReviewStatus == "" {
		normalized.ReviewStatus = previous.ReviewStatus
	}
	if err := normalized.Validate(); err != nil {
		return productdomain.ProductDraft{}, err
	}
	s.drafts[key] = cloneDraft(normalized)
	if err := s.storeLocked(); err != nil {
		s.drafts[key] = previous
		return productdomain.ProductDraft{}, err
	}
	return cloneDraft(normalized), nil
}

func (s *DraftService) MarkPublished(id string, productID string) (productdomain.ProductDraft, error) {
	key := normalizeID(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.drafts[key]
	if !ok {
		return productdomain.ProductDraft{}, errors.New("product draft not found")
	}
	previous := item
	item.ReviewStatus = productdomain.ReviewStatusPublished
	item.PublishedProductID = normalizeID(productID)
	item.UpdatedAt = s.now().UTC()
	normalized := item.Normalized()
	s.drafts[key] = cloneDraft(normalized)
	if err := s.storeLocked(); err != nil {
		s.drafts[key] = previous
		return productdomain.ProductDraft{}, err
	}
	return cloneDraft(normalized), nil
}

func (s *DraftService) buildBootstrapDraft(input productdomain.ProductDraftGenerateInput, now time.Time) productdomain.ProductDraft {
	productID := s.nextProductID(productIDBase(input.Name))
	domain := inferProductDomain(input)
	workers := buildWorkerDrafts(productID, domain, input)
	productWorkers := buildPublishedWorkers(workers)
	artifacts := resolveArtifactTypes(input, domain)
	knowledgeSources := resolveKnowledgeSources(input, domain)
	product := productdomain.Product{
		ID:               productID,
		Name:             input.Name,
		Slug:             productID,
		Summary:          buildProductSummary(input, domain),
		Status:           productdomain.StatusDraft,
		Visibility:       productdomain.VisibilityPrivate,
		OwnerType:        productdomain.OwnerTypeManaged,
		MasterAgentID:    productID + "-master",
		EntryRoute:       "products",
		Tags:             resolveProductTags(input, domain),
		Version:          productdomain.DefaultVersion,
		WorkerAgents:     productWorkers,
		ArtifactTypes:    artifacts,
		KnowledgeSources: knowledgeSources,
	}.Normalized()
	return productdomain.ProductDraft{
		DraftID:                 nextDraftID(productID),
		Product:                 product,
		Mode:                    productdomain.GenerationModeBootstrap,
		ReviewStatus:            productdomain.ReviewStatusDraft,
		GeneratedBy:             "product-builder",
		GeneratedAt:             now,
		UpdatedAt:               now,
		Goal:                    input.Goal,
		TargetUsers:             input.TargetUsers,
		CoreCapabilities:        input.CoreCapabilities,
		Constraints:             input.Constraints,
		ExpectedArtifacts:       input.ExpectedArtifacts,
		IntegrationRequirements: input.IntegrationRequirements,
		ConflictSuggestions:     s.buildConflictSuggestions(input.Name, productID),
		MasterAgent:             buildMasterDraft(productID, domain, input, workers),
		WorkerMatrix:            workers,
	}.Normalized()
}

func (s *DraftService) buildExpandDraft(base productdomain.Product, input productdomain.ProductDraftGenerateInput, now time.Time) productdomain.ProductDraft {
	domain := inferProductDomain(input)
	workers := buildWorkerDrafts(base.ID, domain, input)
	mergedWorkers := mergePublishedWorkers(base.WorkerAgents, buildPublishedWorkers(workers))
	artifacts := mergeStringLists(base.ArtifactTypes, resolveArtifactTypes(input, domain))
	knowledgeSources := mergeStringLists(base.KnowledgeSources, resolveKnowledgeSources(input, domain))
	tags := mergeStringLists(base.Tags, resolveProductTags(input, domain))
	product := base
	product.Summary = strings.TrimSpace(strings.Join([]string{
		strings.TrimSpace(base.Summary),
		buildExpansionSummary(input),
	}, " "))
	product.WorkerAgents = mergedWorkers
	product.ArtifactTypes = artifacts
	product.KnowledgeSources = knowledgeSources
	product.Tags = tags
	product.Status = productdomain.StatusDraft
	return productdomain.ProductDraft{
		DraftID:                 nextDraftID(base.ID + "-expand"),
		Product:                 product.Normalized(),
		Mode:                    productdomain.GenerationModeExpand,
		ReviewStatus:            productdomain.ReviewStatusDraft,
		GeneratedBy:             "product-builder",
		GeneratedAt:             now,
		UpdatedAt:               now,
		Goal:                    input.Goal,
		TargetUsers:             input.TargetUsers,
		CoreCapabilities:        input.CoreCapabilities,
		Constraints:             input.Constraints,
		ExpectedArtifacts:       input.ExpectedArtifacts,
		IntegrationRequirements: input.IntegrationRequirements,
		ConflictSuggestions:     nil,
		MasterAgent:             buildMasterDraft(base.ID, domain, input, workers),
		WorkerMatrix:            workers,
	}.Normalized()
}

func (s *DraftService) nextProductID(base string) string {
	if base == "" {
		base = "product"
	}
	candidate := base
	index := 2
	for {
		if s.products == nil {
			return candidate
		}
		if _, exists := s.products.ResolveProduct(candidate); !exists {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", base, index)
		index++
	}
}

func (s *DraftService) buildConflictSuggestions(name string, productID string) []string {
	if s.products == nil {
		return nil
	}
	normalizedName := strings.ToLower(strings.TrimSpace(name))
	items := make([]string, 0, 4)
	for _, item := range s.products.ListProducts() {
		score := 0
		itemName := strings.ToLower(strings.TrimSpace(item.Name))
		if itemName != "" && (strings.Contains(itemName, normalizedName) || strings.Contains(normalizedName, itemName)) {
			score++
		}
		if strings.Contains(productID, item.ID) || strings.Contains(item.ID, productID) {
			score++
		}
		if score == 0 {
			continue
		}
		items = append(items, fmt.Sprintf("Existing product %s (%s) may overlap; consider reuse or merge before publish.", item.Name, item.ID))
	}
	return normalizeDraftStringList(items)
}

func (s *DraftService) storeLocked() error {
	if s == nil || s.store == nil {
		return nil
	}
	items := make([]productdomain.ProductDraft, 0, len(s.drafts))
	for _, item := range s.drafts {
		items = append(items, cloneDraft(item))
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].DraftID < items[j].DraftID
		}
		return items[i].UpdatedAt.Before(items[j].UpdatedAt)
	})
	return s.store.Save(context.Background(), items)
}

func cloneDraft(draft productdomain.ProductDraft) productdomain.ProductDraft {
	cloned := draft.Normalized()
	cloned.Product = cloneProduct(draft.Product)
	cloned.TargetUsers = cloneStringList(draft.TargetUsers)
	cloned.CoreCapabilities = cloneStringList(draft.CoreCapabilities)
	cloned.Constraints = cloneStringList(draft.Constraints)
	cloned.ExpectedArtifacts = cloneStringList(draft.ExpectedArtifacts)
	cloned.IntegrationRequirements = cloneStringList(draft.IntegrationRequirements)
	cloned.ConflictSuggestions = cloneStringList(draft.ConflictSuggestions)
	cloned.MasterAgent = draft.MasterAgent.Normalized()
	if len(draft.WorkerMatrix) > 0 {
		cloned.WorkerMatrix = append([]productdomain.ProductWorkerDraft(nil), draft.WorkerMatrix...)
	}
	return cloned
}

func inferProductDomain(input productdomain.ProductDraftGenerateInput) string {
	parts := []string{input.Name, input.Goal}
	parts = append(parts, input.CoreCapabilities...)
	parts = append(parts, input.ExpectedArtifacts...)
	parts = append(parts, input.IntegrationRequirements...)
	text := strings.ToLower(strings.Join(parts, " "))
	switch {
	case containsAny(text, "travel", "trip", "itinerary", "guide", "tour", "旅游", "旅行", "攻略", "地铁", "景点"):
		return "travel"
	default:
		return "generic"
	}
}

func buildMasterDraft(productID string, domain string, input productdomain.ProductDraftGenerateInput, workers []productdomain.ProductWorkerDraft) productdomain.ProductAgentDraft {
	targets := make([]string, 0, len(workers))
	for _, worker := range workers {
		targets = append(targets, worker.AgentID)
	}
	systemPrompt := "Act as the master agent for the " + productID + " product. Understand the user's goal, decompose work into the defined worker matrix, keep outputs coherent, and return both user-facing results and structured artifacts."
	if domain == "travel" {
		systemPrompt = "Act as the master agent for the travel product. Understand city-trip requirements, preserve structured travel fields for revision, orchestrate city guide, route, metro, food, and map subtasks, and return one coherent guide."
	}
	return productdomain.ProductAgentDraft{
		AgentID:                productID + "-master",
		Name:                   input.Name + " Master Agent",
		Description:            "Master agent draft generated for product orchestration.",
		SystemPrompt:           systemPrompt,
		MaxIterations:          8,
		Tools:                  []string{"delegate_agent", "complete"},
		Skills:                 []string{"memory"},
		MemoryFiles:            []string{"user_md", "soul_md", "agents_md", "memory_long_term", "memory_daily_today"},
		Capabilities:           []string{domain, "product-master"},
		AllowedDelegateTargets: targets,
		Enabled:                true,
		Delegatable:            true,
	}.Normalized()
}

func buildWorkerDrafts(productID string, domain string, input productdomain.ProductDraftGenerateInput) []productdomain.ProductWorkerDraft {
	if domain == "travel" {
		return []productdomain.ProductWorkerDraft{
			newWorkerDraft(productID+"-city-guide", "City Guide", "city-guide", "Aggregate scenic spots, neighborhoods, and city overview.", "User trip brief and city constraints.", "POI set and city overview notes.", []string{"travel", "city-guide"}, []string{"complete"}, 10),
			newWorkerDraft(productID+"-route-planner", "Route Planner", "route-planner", "Produce daily route order and pacing.", "POIs and user rhythm constraints.", "Day-by-day routes with stop order.", []string{"travel", "route-planning"}, []string{"complete"}, 20),
			newWorkerDraft(productID+"-metro-guide", "Metro Guide", "metro-guide", "Recommend metro and transit segments between stops.", "Daily routes and transport constraints.", "Metro lines and transfer guidance.", []string{"travel", "metro"}, []string{"complete"}, 30),
			newWorkerDraft(productID+"-food-recommender", "Food Recommender", "food-recommender", "Recommend meals by area and time slot.", "Route plan and food preferences.", "Food stops and dining windows.", []string{"travel", "food"}, []string{"complete"}, 40),
			newWorkerDraft(productID+"-map-annotator", "Map Annotator", "map-annotator", "Prepare map layers, points, and route lines.", "POIs, routes, and transit outputs.", "Map layer descriptors and highlight references.", []string{"travel", "map"}, []string{"complete"}, 50),
		}
	}

	capabilities := input.CoreCapabilities
	if len(capabilities) == 0 {
		capabilities = []string{"research", "planning", "delivery"}
	}
	items := make([]productdomain.ProductWorkerDraft, 0, len(capabilities))
	for index, capability := range capabilities {
		roleID := productIDBase(capability)
		if roleID == "" {
			roleID = fmt.Sprintf("worker-%d", index+1)
		}
		name := displayNameFromID(roleID)
		items = append(items, newWorkerDraft(
			productID+"-"+roleID,
			name,
			roleID,
			"Own the "+capability+" slice for the product matrix.",
			"Master-agent task packet and product constraints.",
			"Structured output for "+capability+" with explicit boundaries.",
			[]string{roleID},
			[]string{"complete"},
			(index+1)*10,
		))
	}
	return items
}

func newWorkerDraft(agentID, name, role, responsibility, inputContract, outputContract string, capabilities, tools []string, priority int) productdomain.ProductWorkerDraft {
	return productdomain.ProductWorkerDraft{
		AgentID:                agentID,
		Name:                   name,
		Role:                   role,
		Responsibility:         responsibility,
		Description:            responsibility,
		SystemPrompt:           "Focus on the " + role + " responsibility within the current product boundary. Return concrete, structured output that the master agent can merge.",
		InputContract:          inputContract,
		OutputContract:         outputContract,
		AllowedTools:           tools,
		AllowedDelegateTargets: nil,
		Dependencies:           nil,
		Skills:                 []string{"memory"},
		MemoryFiles:            []string{"user_md", "soul_md", "agents_md", "memory_long_term", "memory_daily_today"},
		Capabilities:           capabilities,
		Priority:               priority,
		MaxIterations:          4,
		Enabled:                true,
	}.Normalized()
}

func buildPublishedWorkers(workers []productdomain.ProductWorkerDraft) []productdomain.WorkerAgent {
	items := make([]productdomain.WorkerAgent, 0, len(workers))
	for _, worker := range workers {
		items = append(items, productdomain.WorkerAgent{
			AgentID:        worker.AgentID,
			Role:           worker.Role,
			Responsibility: worker.Responsibility,
			Capabilities:   cloneStringList(worker.Capabilities),
			Enabled:        worker.Enabled,
		})
	}
	return items
}

func mergePublishedWorkers(left []productdomain.WorkerAgent, right []productdomain.WorkerAgent) []productdomain.WorkerAgent {
	items := make([]productdomain.WorkerAgent, 0, len(left)+len(right))
	seen := map[string]struct{}{}
	for _, item := range append(append([]productdomain.WorkerAgent(nil), left...), right...) {
		normalized := item
		normalized.AgentID = normalizeID(normalized.AgentID)
		if normalized.AgentID == "" {
			continue
		}
		if _, ok := seen[normalized.AgentID]; ok {
			continue
		}
		seen[normalized.AgentID] = struct{}{}
		items = append(items, normalized)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].AgentID < items[j].AgentID })
	return items
}

func resolveArtifactTypes(input productdomain.ProductDraftGenerateInput, domain string) []string {
	if len(input.ExpectedArtifacts) > 0 {
		return input.ExpectedArtifacts
	}
	if domain == "travel" {
		return []string{"city_guide", "itinerary", "map_layers"}
	}
	return []string{"brief", "plan", "delivery"}
}

func resolveKnowledgeSources(input productdomain.ProductDraftGenerateInput, domain string) []string {
	items := cloneStringList(input.IntegrationRequirements)
	if domain == "travel" {
		items = mergeStringLists(items, []string{"city_profile", "poi_catalog", "metro_network", "food_catalog"})
	}
	if len(items) == 0 {
		items = []string{"reference_docs", "domain_playbook"}
	}
	return items
}

func resolveProductTags(input productdomain.ProductDraftGenerateInput, domain string) []string {
	items := mergeStringLists(input.CoreCapabilities, input.ExpectedArtifacts)
	items = mergeStringLists(items, []string{domain})
	return items
}

func buildProductSummary(input productdomain.ProductDraftGenerateInput, domain string) string {
	goal := strings.TrimSpace(input.Goal)
	if goal == "" {
		goal = "Execution-ready product generated from the product matrix workflow."
	}
	if domain == "travel" {
		return "Travel-focused product for city guides, route planning, metro guidance, food recommendations, and map-oriented delivery. Goal: " + goal
	}
	return goal
}

func buildExpansionSummary(input productdomain.ProductDraftGenerateInput) string {
	if len(input.CoreCapabilities) == 0 {
		return ""
	}
	return "Expanded capabilities: " + strings.Join(input.CoreCapabilities, ", ") + "."
}

func nextDraftID(base string) string {
	id := productIDBase(base)
	if id == "" {
		id = "product-draft"
	}
	return id + "-draft"
}

func mergeStringLists(left []string, right []string) []string {
	return normalizeDraftStringList(append(append([]string(nil), left...), right...))
}

func normalizeDraftStringList(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
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

func containsAny(content string, terms ...string) bool {
	for _, term := range terms {
		if strings.Contains(content, strings.ToLower(strings.TrimSpace(term))) {
			return true
		}
	}
	return false
}

func displayNameFromID(id string) string {
	parts := strings.Split(strings.TrimSpace(id), "-")
	for index, part := range parts {
		if part == "" {
			continue
		}
		parts[index] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

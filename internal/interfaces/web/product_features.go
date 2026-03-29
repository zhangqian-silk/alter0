package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	agentapp "alter0/internal/agent/application"
	controldomain "alter0/internal/control/domain"
	execdomain "alter0/internal/execution/domain"
	productdomain "alter0/internal/product/domain"
	shareddomain "alter0/internal/shared/domain"
)

type productRoutingDecision struct {
	Matches         []productdomain.Product
	Selected        *productdomain.Product
	SelectionReason string
	ExecutionMode   string
	RouteToMaster   bool
}

func (s *Server) productDraftGenerateHandler(w http.ResponseWriter, r *http.Request) {
	if s.productDrafts == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "product draft service unavailable"})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	defer r.Body.Close()
	var req productDraftGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	draft, err := s.productDrafts.GenerateDraft(buildProductDraftGenerateInput(req))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, draft)
}

func (s *Server) productDraftListHandler(w http.ResponseWriter, r *http.Request) {
	if s.productDrafts == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "product draft service unavailable"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": s.productDrafts.ListDrafts()})
}

func (s *Server) productDraftItemHandler(w http.ResponseWriter, r *http.Request) {
	if s.productDrafts == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "product draft service unavailable"})
		return
	}
	parts, ok := productDraftResourceParts(r.URL.Path)
	if !ok || len(parts) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product draft path"})
		return
	}
	draftID := parts[0]
	if len(parts) == 2 {
		if parts[1] != "publish" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product draft action"})
			return
		}
		s.productDraftPublishHandler(w, r, draftID)
		return
	}
	switch r.Method {
	case http.MethodGet:
		item, found := s.productDrafts.GetDraft(draftID)
		if !found {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "product draft not found"})
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodPut:
		defer r.Body.Close()
		var draft productdomain.ProductDraft
		if err := json.NewDecoder(r.Body).Decode(&draft); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		saved, err := s.productDrafts.SaveDraft(draftID, draft)
		if err != nil {
			status := http.StatusBadRequest
			if strings.Contains(strings.ToLower(err.Error()), "not found") {
				status = http.StatusNotFound
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, saved)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) productMatrixGenerateHandler(w http.ResponseWriter, r *http.Request, productID string) {
	if s.productDrafts == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "product draft service unavailable"})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	defer r.Body.Close()
	var req productDraftGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	input := buildProductDraftGenerateInput(req)
	input.Mode = productdomain.GenerationModeExpand
	draft, err := s.productDrafts.GenerateMatrixDraft(productID, input)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, draft)
}

func (s *Server) productDraftPublishHandler(w http.ResponseWriter, r *http.Request, draftID string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if s.control == nil || s.products == nil || s.agents == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "publish dependencies unavailable"})
		return
	}
	draft, found := s.productDrafts.GetDraft(draftID)
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "product draft not found"})
		return
	}
	productID := strings.TrimSpace(draft.Product.ID)
	if productID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "draft product id is required"})
		return
	}
	if s.products.IsBuiltinID(productID) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "builtin products cannot be published from drafts"})
		return
	}
	agents, err := s.publishDraftAgents(draft)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	product, err := s.products.SaveProduct(productID, draft.Product)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	updatedDraft, err := s.productDrafts.MarkPublished(draftID, product.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"draft":   updatedDraft,
		"product": product,
		"agents":  agents,
	})
}

func (s *Server) publishDraftAgents(draft productdomain.ProductDraft) ([]controldomain.Agent, error) {
	items := make([]controldomain.Agent, 0, len(draft.WorkerMatrix)+1)
	master, err := s.saveDraftAgent(buildControlAgentFromMasterDraft(draft.MasterAgent))
	if err != nil {
		return nil, err
	}
	items = append(items, master)
	for _, worker := range draft.WorkerMatrix {
		saved, saveErr := s.saveDraftAgent(buildControlAgentFromWorkerDraft(worker))
		if saveErr != nil {
			return nil, saveErr
		}
		items = append(items, saved)
	}
	return items, nil
}

func (s *Server) saveDraftAgent(agent controldomain.Agent) (controldomain.Agent, error) {
	if s.agents.IsBuiltinID(agent.ID) {
		return controldomain.Agent{}, fmt.Errorf("builtin agent %s cannot be overwritten", agent.ID)
	}
	return s.control.SaveAgent(agent.ID, agent)
}

func buildControlAgentFromMasterDraft(draft productdomain.ProductAgentDraft) controldomain.Agent {
	return controldomain.Agent{
		ID:            strings.TrimSpace(draft.AgentID),
		Name:          strings.TrimSpace(draft.Name),
		Type:          controldomain.CapabilityTypeAgent,
		Enabled:       draft.Enabled,
		Scope:         controldomain.CapabilityScopeGlobal,
		SystemPrompt:  strings.TrimSpace(draft.SystemPrompt),
		MaxIterations: draft.MaxIterations,
		Tools:         append([]string(nil), draft.Tools...),
		Skills:        append([]string(nil), draft.Skills...),
		MCPs:          append([]string(nil), draft.MCPs...),
		MemoryFiles:   append([]string(nil), draft.MemoryFiles...),
		Source:        controldomain.AgentSourceManaged,
		Kind:          controldomain.AgentKindSpecialist,
		Description:   strings.TrimSpace(draft.Description),
		Delegatable:   draft.Delegatable,
		Capabilities:  append([]string(nil), draft.Capabilities...),
	}
}

func buildControlAgentFromWorkerDraft(draft productdomain.ProductWorkerDraft) controldomain.Agent {
	return controldomain.Agent{
		ID:            strings.TrimSpace(draft.AgentID),
		Name:          strings.TrimSpace(draft.Name),
		Type:          controldomain.CapabilityTypeAgent,
		Enabled:       draft.Enabled,
		Scope:         controldomain.CapabilityScopeGlobal,
		SystemPrompt:  strings.TrimSpace(draft.SystemPrompt),
		MaxIterations: draft.MaxIterations,
		Tools:         append([]string(nil), draft.AllowedTools...),
		Skills:        append([]string(nil), draft.Skills...),
		MCPs:          append([]string(nil), draft.MCPs...),
		MemoryFiles:   append([]string(nil), draft.MemoryFiles...),
		Source:        controldomain.AgentSourceManaged,
		Kind:          controldomain.AgentKindSpecialist,
		Description:   strings.TrimSpace(draft.Description),
		Delegatable:   false,
		Capabilities:  append([]string(nil), draft.Capabilities...),
	}
}

func buildProductDraftGenerateInput(req productDraftGenerateRequest) productdomain.ProductDraftGenerateInput {
	return productdomain.ProductDraftGenerateInput{
		Name:                    strings.TrimSpace(req.Name),
		Goal:                    strings.TrimSpace(req.Goal),
		TargetUsers:             req.TargetUsers,
		CoreCapabilities:        req.CoreCapabilities,
		Constraints:             req.Constraints,
		ExpectedArtifacts:       req.ExpectedArtifacts,
		IntegrationRequirements: req.IntegrationRequirements,
		Mode:                    productdomain.GenerationMode(strings.ToLower(strings.TrimSpace(req.Mode))),
	}
}

func productDraftResourceParts(path string) ([]string, bool) {
	const prefix = "/api/control/products/drafts/"
	if !strings.HasPrefix(path, prefix) {
		return nil, false
	}
	trimmed := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	if trimmed == "" {
		return nil, false
	}
	parts := strings.Split(trimmed, "/")
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			return nil, false
		}
		cleaned = append(cleaned, item)
	}
	return cleaned, true
}

func productControlResourceParts(path string) ([]string, bool) {
	const prefix = "/api/control/products/"
	if !strings.HasPrefix(path, prefix) {
		return nil, false
	}
	trimmed := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	if trimmed == "" {
		return nil, false
	}
	parts := strings.Split(trimmed, "/")
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			return nil, false
		}
		cleaned = append(cleaned, item)
	}
	return cleaned, true
}

func (s *Server) applyMainAgentProductRouting(msg shareddomain.UnifiedMessage, requestedAgentID string) (shareddomain.UnifiedMessage, error) {
	if s == nil || s.products == nil || s.agents == nil {
		return msg, nil
	}
	if !strings.EqualFold(strings.TrimSpace(requestedAgentID), "main") {
		return msg, nil
	}
	decision := s.discoverProductRouting(msg.Content)
	if len(decision.Matches) == 0 {
		return msg, nil
	}
	msg.Metadata = cloneStringMap(msg.Metadata)
	attachProductRoutingMetadata(msg.Metadata, decision)
	attachProductDiscoveryContext(msg.Metadata, decision)
	if decision.Selected != nil {
		rawContext, err := json.Marshal(buildProductExecutionContext(*decision.Selected))
		if err != nil {
			return shareddomain.UnifiedMessage{}, err
		}
		msg.Metadata[execdomain.ProductContextMetadataKey] = string(rawContext)
	}
	if !decision.RouteToMaster || decision.Selected == nil {
		return msg, nil
	}
	masterID := strings.TrimSpace(decision.Selected.MasterAgentID)
	if masterID == "" {
		return msg, nil
	}
	agent, ok := s.agents.ResolveAgent(masterID)
	if !ok || !agent.Enabled {
		return msg, nil
	}
	msg.Metadata = agentapp.ApplyProfileMetadata(msg.Metadata, agent)
	msg.Metadata[execdomain.AgentDelegatedByMetadataKey] = "main"
	attachProductRoutingMetadata(msg.Metadata, decision)
	return msg, nil
}

func (s *Server) discoverProductRouting(content string) productRoutingDecision {
	items := s.products.ListPublicProducts()
	if len(items) == 0 {
		return productRoutingDecision{}
	}
	normalized := strings.ToLower(strings.TrimSpace(content))
	if normalized == "" {
		return productRoutingDecision{}
	}
	type scoredProduct struct {
		item   productdomain.Product
		score  int
		reason string
	}
	scored := make([]scoredProduct, 0, len(items))
	for _, item := range items {
		score, reason := scoreProductMatch(item, normalized)
		if score <= 0 {
			continue
		}
		scored = append(scored, scoredProduct{item: item, score: score, reason: reason})
	}
	if len(scored) == 0 {
		return productRoutingDecision{}
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].item.ID < scored[j].item.ID
		}
		return scored[i].score > scored[j].score
	})
	matches := make([]productdomain.Product, 0, len(scored))
	for _, item := range scored {
		matches = append(matches, item.item)
	}
	selected := scored[0].item
	decision := productRoutingDecision{
		Matches:         matches,
		Selected:        &selected,
		SelectionReason: scored[0].reason,
		ExecutionMode:   "catalog-read",
		RouteToMaster:   false,
	}
	if isProductExecutionIntent(normalized) {
		decision.ExecutionMode = "product-master"
		decision.RouteToMaster = true
		if decision.SelectionReason == "" {
			decision.SelectionReason = "matched product intent and switched to the product master agent"
		}
		return decision
	}
	if decision.SelectionReason == "" {
		decision.SelectionReason = "matched product capability keywords for catalog read"
	}
	return decision
}

func scoreProductMatch(item productdomain.Product, content string) (int, string) {
	score := 0
	reasons := []string{}
	if containsProductToken(content, item.ID) {
		score += 8
		reasons = append(reasons, "matched product id")
	}
	if containsProductToken(content, item.Name) {
		score += 6
		reasons = append(reasons, "matched product name")
	}
	if containsProductToken(content, item.Slug) {
		score += 5
		reasons = append(reasons, "matched product slug")
	}
	for _, tag := range item.Tags {
		if containsProductToken(content, tag) {
			score += 3
			reasons = append(reasons, "matched product tag "+tag)
		}
	}
	for _, worker := range item.WorkerAgents {
		if containsProductToken(content, worker.Role) {
			score += 2
			reasons = append(reasons, "matched worker role "+worker.Role)
		}
		if containsProductToken(content, worker.Responsibility) {
			score += 1
			reasons = append(reasons, "matched worker responsibility")
		}
	}
	if strings.EqualFold(item.ID, productdomain.TravelProductID) {
		travelTerms := []string{"travel", "trip", "itinerary", "guide", "metro", "food", "tour", "journey", "旅游", "旅行", "攻略", "行程", "地铁", "景点", "美食", "路线", "城市"}
		for _, term := range travelTerms {
			if strings.Contains(content, strings.ToLower(term)) {
				score += 2
				reasons = append(reasons, "matched travel domain keyword")
				break
			}
		}
	}
	if score == 0 {
		return 0, ""
	}
	return score, strings.Join(uniqueStrings(reasons), ", ")
}

func isProductExecutionIntent(content string) bool {
	terms := []string{"生成", "规划", "安排", "做一份", "制定", "修改", "调整", "revise", "update", "create", "plan", "build", "recommend", "route"}
	for _, term := range terms {
		if strings.Contains(content, strings.ToLower(term)) {
			return true
		}
	}
	return false
}

func attachProductDiscoveryContext(metadata map[string]string, decision productRoutingDecision) {
	if len(metadata) == 0 || len(decision.Matches) == 0 {
		return
	}
	matched := make([]execdomain.ProductContext, 0, len(decision.Matches))
	for _, item := range decision.Matches {
		matched = append(matched, buildProductExecutionContext(item))
	}
	payload := execdomain.ProductDiscoveryContext{
		Protocol:        execdomain.ProductDiscoveryProtocolVersion,
		MatchedProducts: matched,
		SelectionReason: decision.SelectionReason,
		ExecutionMode:   decision.ExecutionMode,
	}
	if decision.Selected != nil {
		payload.SelectedProduct = decision.Selected.ID
	}
	if raw, err := json.Marshal(payload); err == nil {
		metadata[execdomain.ProductDiscoveryMetadataKey] = string(raw)
	}
}

func attachProductRoutingMetadata(metadata map[string]string, decision productRoutingDecision) {
	if len(metadata) == 0 || len(decision.Matches) == 0 {
		return
	}
	ids := make([]string, 0, len(decision.Matches))
	for _, item := range decision.Matches {
		ids = append(ids, item.ID)
	}
	metadata[execdomain.ProductMatchedIDsMetadataKey] = strings.Join(ids, ",")
	metadata[execdomain.ProductSelectionReasonMetadataKey] = strings.TrimSpace(decision.SelectionReason)
	metadata[execdomain.ProductExecutionModeMetadataKey] = strings.TrimSpace(decision.ExecutionMode)
	if decision.Selected != nil {
		metadata[execdomain.ProductSelectedIDMetadataKey] = decision.Selected.ID
		metadata[execdomain.ProductMasterAgentMetadataKey] = strings.TrimSpace(decision.Selected.MasterAgentID)
	}
}

func attachProductRouteResultMetadata(result shareddomain.OrchestrationResult, metadata map[string]string) shareddomain.OrchestrationResult {
	if len(metadata) == 0 {
		return result
	}
	result.Metadata = cloneStringMap(result.Metadata)
	for _, key := range []string{
		execdomain.ProductMatchedIDsMetadataKey,
		execdomain.ProductSelectedIDMetadataKey,
		execdomain.ProductSelectionReasonMetadataKey,
		execdomain.ProductMasterAgentMetadataKey,
		execdomain.ProductExecutionModeMetadataKey,
	} {
		if value := strings.TrimSpace(metadata[key]); value != "" {
			result.Metadata[key] = value
		}
	}
	return result
}

func containsProductToken(content string, value string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return false
	}
	return strings.Contains(content, trimmed)
}

func uniqueStrings(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func ensureDraftPublishable(draft productdomain.ProductDraft) error {
	if strings.TrimSpace(draft.Product.ID) == "" {
		return errors.New("draft product id is required")
	}
	if strings.TrimSpace(draft.MasterAgent.AgentID) == "" {
		return errors.New("draft master agent is required")
	}
	return nil
}

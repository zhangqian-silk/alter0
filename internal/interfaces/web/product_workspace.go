package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	agentapp "alter0/internal/agent/application"
	controldomain "alter0/internal/control/domain"
	execdomain "alter0/internal/execution/domain"
	productdomain "alter0/internal/product/domain"
)

const travelWorkspaceReplyOnlyAction = "reply_only"

type productWorkspaceResponse struct {
	Product       productdomain.Product          `json:"product"`
	MasterAgent   *productWorkspaceMasterAgent   `json:"master_agent,omitempty"`
	SpaceType     string                         `json:"space_type,omitempty"`
	SpaceLabel    string                         `json:"space_label,omitempty"`
	WorkspaceHint string                         `json:"workspace_hint,omitempty"`
	Spaces        []productWorkspaceSpaceSummary `json:"spaces,omitempty"`
}

type productWorkspaceMasterAgent struct {
	AgentID      string   `json:"agent_id"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	Tools        []string `json:"tools,omitempty"`
	Skills       []string `json:"skills,omitempty"`
	MCPs         []string `json:"mcps,omitempty"`
	MemoryFiles  []string `json:"memory_files,omitempty"`
}

type productWorkspaceSpaceSummary struct {
	SpaceID   string    `json:"space_id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug,omitempty"`
	Summary   string    `json:"summary,omitempty"`
	Type      string    `json:"type,omitempty"`
	Status    string    `json:"status,omitempty"`
	Revision  int       `json:"revision,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
}

type productWorkspaceSpaceDetail struct {
	Space productWorkspaceSpaceSummary `json:"space"`
	Guide productdomain.TravelGuide    `json:"guide"`
}

type productWorkspaceChatRequest struct {
	SessionID     string            `json:"session_id"`
	UserID        string            `json:"user_id,omitempty"`
	ChannelID     string            `json:"channel_id,omitempty"`
	CorrelationID string            `json:"correlation_id,omitempty"`
	SpaceID       string            `json:"space_id,omitempty"`
	Content       string            `json:"content"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type productWorkspaceChatResponse struct {
	Reply   string                     `json:"reply"`
	Action  string                     `json:"action,omitempty"`
	SpaceID string                     `json:"space_id,omitempty"`
	Guide   *productdomain.TravelGuide `json:"guide,omitempty"`
}

type travelWorkspaceAgentEnvelope struct {
	Action         string                               `json:"action"`
	TargetCity     string                               `json:"target_city,omitempty"`
	AssistantReply string                               `json:"assistant_reply,omitempty"`
	CreateInput    productdomain.TravelGuideCreateInput `json:"create_input"`
	ReviseInput    productdomain.TravelGuideReviseInput `json:"revise_input"`
}

func (s *Server) productWorkspaceSummaryHandler(w http.ResponseWriter, r *http.Request, productID string) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	product, statusCode, err := s.resolvePublicProduct(productID)
	if err != nil {
		writeJSON(w, statusCode, map[string]string{"error": err.Error()})
		return
	}
	payload, err := s.buildProductWorkspaceResponse(product)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) productWorkspaceSpaceItemHandler(w http.ResponseWriter, r *http.Request, productID string, spaceID string) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	product, statusCode, err := s.resolvePublicProduct(productID)
	if err != nil {
		writeJSON(w, statusCode, map[string]string{"error": err.Error()})
		return
	}
	if !strings.EqualFold(product.ID, productdomain.TravelProductID) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "product space not found"})
		return
	}
	if s.travelGuides == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "travel guide service unavailable"})
		return
	}
	guide, found := s.travelGuides.GetGuide(spaceID)
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "product space not found"})
		return
	}
	writeJSON(w, http.StatusOK, productWorkspaceSpaceDetail{
		Space: buildTravelWorkspaceSpaceSummary(guide),
		Guide: guide,
	})
}

func (s *Server) productWorkspaceChatHandler(w http.ResponseWriter, r *http.Request, productID string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	product, statusCode, err := s.resolvePublicProduct(productID)
	if err != nil {
		writeJSON(w, statusCode, map[string]string{"error": err.Error()})
		return
	}
	if !strings.EqualFold(product.ID, productdomain.TravelProductID) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "workspace chat sync is only supported for travel"})
		return
	}
	if s.orchestrator == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "orchestrator unavailable"})
		return
	}
	if s.travelGuides == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "travel guide service unavailable"})
		return
	}
	if s.agents == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "agent catalog unavailable"})
		return
	}

	defer r.Body.Close()
	var req productWorkspaceChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "content is required"})
		return
	}

	var selectedGuide productdomain.TravelGuide
	if spaceID := strings.TrimSpace(req.SpaceID); spaceID != "" {
		item, found := s.travelGuides.GetGuide(spaceID)
		if !found {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "product space not found"})
			return
		}
		selectedGuide = item
	}

	envelope, err := s.executeTravelWorkspaceOperator(r, req, product, selectedGuide)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	action := normalizeTravelWorkspaceAction(envelope.Action)
	reply := strings.TrimSpace(envelope.AssistantReply)
	switch action {
	case "create":
		createInput := envelope.CreateInput
		if strings.TrimSpace(createInput.City) == "" {
			createInput.City = strings.TrimSpace(envelope.TargetCity)
		}
		createInput = createInput.Normalized()
		if strings.TrimSpace(createInput.City) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "travel workspace chat must provide a city for create"})
			return
		}
		if existing, found := s.findTravelGuideByCity(createInput.City); found {
			revised, reviseErr := s.travelGuides.ReviseGuide(existing.ID, buildReviseInputFromCreateInput(createInput))
			if reviseErr != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": reviseErr.Error()})
				return
			}
			reply = chooseWorkspaceReply(reply, "已同步更新 "+revised.City+" 页面。")
			writeJSON(w, http.StatusOK, productWorkspaceChatResponse{
				Reply:   reply,
				Action:  "revise",
				SpaceID: revised.ID,
				Guide:   &revised,
			})
			return
		}
		created, createErr := s.travelGuides.CreateGuide(createInput)
		if createErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": createErr.Error()})
			return
		}
		reply = chooseWorkspaceReply(reply, "已创建 "+created.City+" 页面。")
		writeJSON(w, http.StatusOK, productWorkspaceChatResponse{
			Reply:   reply,
			Action:  "create",
			SpaceID: created.ID,
			Guide:   &created,
		})
		return
	case "revise":
		targetGuide, findErr := s.resolveTravelGuideForRevision(selectedGuide, envelope.TargetCity)
		if findErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": findErr.Error()})
			return
		}
		revised, reviseErr := s.travelGuides.ReviseGuide(targetGuide.ID, envelope.ReviseInput)
		if reviseErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": reviseErr.Error()})
			return
		}
		reply = chooseWorkspaceReply(reply, "已更新 "+revised.City+" 页面。")
		writeJSON(w, http.StatusOK, productWorkspaceChatResponse{
			Reply:   reply,
			Action:  "revise",
			SpaceID: revised.ID,
			Guide:   &revised,
		})
		return
	default:
		reply = chooseWorkspaceReply(reply, "请补充城市、天数或要调整的页面信息。")
		response := productWorkspaceChatResponse{
			Reply:  reply,
			Action: travelWorkspaceReplyOnlyAction,
		}
		if strings.TrimSpace(selectedGuide.ID) != "" {
			guide := selectedGuide
			response.SpaceID = guide.ID
			response.Guide = &guide
		}
		writeJSON(w, http.StatusOK, response)
	}
}

func (s *Server) buildProductWorkspaceResponse(product productdomain.Product) (productWorkspaceResponse, error) {
	payload := productWorkspaceResponse{
		Product: product,
	}
	if s.agents != nil {
		if agent, ok := s.agents.ResolveAgent(product.MasterAgentID); ok {
			summary := buildProductWorkspaceMasterAgent(agent)
			payload.MasterAgent = &summary
		}
	}
	if strings.EqualFold(product.ID, productdomain.TravelProductID) && s.travelGuides != nil {
		payload.SpaceType = "travel-guide"
		payload.SpaceLabel = "City Pages"
		payload.WorkspaceHint = "与 travel 主 Agent 对话后，可直接创建或更新具体城市页面。"
		guides := s.travelGuides.ListGuides()
		payload.Spaces = make([]productWorkspaceSpaceSummary, 0, len(guides))
		for _, guide := range guides {
			payload.Spaces = append(payload.Spaces, buildTravelWorkspaceSpaceSummary(guide))
		}
	}
	return payload, nil
}

func buildProductWorkspaceMasterAgent(agent controldomain.Agent) productWorkspaceMasterAgent {
	return productWorkspaceMasterAgent{
		AgentID:      strings.TrimSpace(agent.ID),
		Name:         strings.TrimSpace(agent.Name),
		Description:  strings.TrimSpace(agent.Description),
		Capabilities: append([]string(nil), agent.Capabilities...),
		Tools:        append([]string(nil), agent.Tools...),
		Skills:       append([]string(nil), agent.Skills...),
		MCPs:         append([]string(nil), agent.MCPs...),
		MemoryFiles:  append([]string(nil), agent.MemoryFiles...),
	}
}

func buildTravelWorkspaceSpaceSummary(guide productdomain.TravelGuide) productWorkspaceSpaceSummary {
	tags := []string{
		strings.TrimSpace(guide.City),
		fmt.Sprintf("%d days", guide.Days),
		"revision " + fmt.Sprintf("%d", guide.Revision),
	}
	if style := strings.TrimSpace(guide.TravelStyle); style != "" {
		tags = append(tags, style)
	}
	return productWorkspaceSpaceSummary{
		SpaceID:   strings.TrimSpace(guide.ID),
		Title:     strings.TrimSpace(guide.City),
		Slug:      strings.TrimSpace(guide.ID),
		Summary:   summaryText(guide.Content, 140),
		Type:      "travel-guide",
		Status:    "active",
		Revision:  guide.Revision,
		UpdatedAt: guide.UpdatedAt,
		Tags:      tags,
	}
}

func (s *Server) resolvePublicProduct(id string) (productdomain.Product, int, error) {
	if s.products == nil {
		return productdomain.Product{}, http.StatusServiceUnavailable, errors.New("product service unavailable")
	}
	product, ok := s.products.ResolveProduct(id)
	if !ok || product.Status != productdomain.StatusActive || product.Visibility != productdomain.VisibilityPublic {
		return productdomain.Product{}, http.StatusNotFound, errors.New("product not found")
	}
	return product, http.StatusOK, nil
}

func (s *Server) executeTravelWorkspaceOperator(
	r *http.Request,
	req productWorkspaceChatRequest,
	product productdomain.Product,
	selectedGuide productdomain.TravelGuide,
) (travelWorkspaceAgentEnvelope, error) {
	agent, ok := s.agents.ResolveAgent(product.MasterAgentID)
	if !ok {
		return travelWorkspaceAgentEnvelope{}, errors.New("product master agent not found")
	}
	if !agent.Enabled {
		return travelWorkspaceAgentEnvelope{}, errors.New("product master agent is disabled")
	}
	msg, statusCode, err := s.prepareMessageFromRequest(messageRequest{
		SessionID:     req.SessionID,
		UserID:        req.UserID,
		ChannelID:     req.ChannelID,
		CorrelationID: req.CorrelationID,
		Content:       req.Content,
		Metadata:      cloneStringMap(req.Metadata),
	})
	if err != nil {
		return travelWorkspaceAgentEnvelope{}, fmt.Errorf("prepare workspace message (%d): %w", statusCode, err)
	}
	msg.Metadata = agentapp.ApplyProfileMetadata(msg.Metadata, agent)
	msg.Metadata[execdomain.AgentSystemPromptMetadataKey] = buildTravelWorkspaceOperatorPrompt(product, selectedGuide, s.travelGuides.ListGuides())
	msg.Metadata[execdomain.AgentToolsMetadataKey] = `["complete"]`
	rawProductContext, err := json.Marshal(buildProductExecutionContext(product))
	if err != nil {
		return travelWorkspaceAgentEnvelope{}, fmt.Errorf("encode product context: %w", err)
	}
	msg.Metadata[execdomain.ProductContextMetadataKey] = string(rawProductContext)
	result, err := s.orchestrator.Handle(r.Context(), msg)
	if err != nil {
		return travelWorkspaceAgentEnvelope{}, err
	}
	return parseTravelWorkspaceAgentEnvelope(result.Output)
}

func buildTravelWorkspaceOperatorPrompt(
	product productdomain.Product,
	selectedGuide productdomain.TravelGuide,
	guides []productdomain.TravelGuide,
) string {
	var builder strings.Builder
	builder.WriteString("You are the workspace operator for the travel product master agent.\n")
	builder.WriteString("Convert the user's latest message into a structured page action for the travel workspace.\n")
	builder.WriteString("Return raw JSON only. Do not wrap the JSON in markdown.\n")
	builder.WriteString("Schema:\n")
	builder.WriteString("{\n")
	builder.WriteString(`  "action": "create" | "revise" | "reply_only",` + "\n")
	builder.WriteString(`  "target_city": "string",` + "\n")
	builder.WriteString(`  "assistant_reply": "string",` + "\n")
	builder.WriteString(`  "create_input": {"city":"string","days":3,"travel_style":"string","budget":"string","companions":[],"must_visit":[],"avoid":[],"additional_requirements":[]},` + "\n")
	builder.WriteString(`  "revise_input": {"days":3,"travel_style":"string","budget":"string","companions":[],"must_visit":[],"avoid":[],"additional_requirements":[],"keep_conditions":[],"replace_conditions":[]}` + "\n")
	builder.WriteString("}\n")
	builder.WriteString("Rules:\n")
	builder.WriteString("- Use action=create when the user is creating a brand-new city page.\n")
	builder.WriteString("- Use action=revise when the user is updating an existing city page.\n")
	builder.WriteString("- Use action=reply_only when the user has not provided enough information to create or revise a page.\n")
	builder.WriteString("- If a current page is selected, revise that page unless the user clearly switches to another city.\n")
	builder.WriteString("- Keep assistant_reply concise and product-facing.\n")
	builder.WriteString("- Prefer days=3 when the user asks for a new page without a duration.\n")
	builder.WriteString("- Never invent invalid JSON.\n")
	builder.WriteString("\nProduct:\n")
	builder.WriteString("- product_id: ")
	builder.WriteString(strings.TrimSpace(product.ID))
	builder.WriteString("\n- name: ")
	builder.WriteString(strings.TrimSpace(product.Name))
	if len(guides) > 0 {
		builder.WriteString("\nExisting city pages:\n")
		for _, guide := range guides {
			builder.WriteString("- ")
			builder.WriteString(strings.TrimSpace(guide.City))
			builder.WriteString(" | id=")
			builder.WriteString(strings.TrimSpace(guide.ID))
			builder.WriteString(" | days=")
			builder.WriteString(fmt.Sprintf("%d", guide.Days))
			builder.WriteString(" | revision=")
			builder.WriteString(fmt.Sprintf("%d", guide.Revision))
			builder.WriteString("\n")
		}
	}
	if strings.TrimSpace(selectedGuide.ID) != "" {
		builder.WriteString("\nCurrent selected page JSON:\n")
		if raw, err := json.Marshal(selectedGuide); err == nil {
			builder.Write(raw)
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

func parseTravelWorkspaceAgentEnvelope(raw string) (travelWorkspaceAgentEnvelope, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return travelWorkspaceAgentEnvelope{}, errors.New("workspace agent returned empty output")
	}
	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(trimmed, "```")
	trimmed = strings.TrimSpace(trimmed)
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start >= 0 && end >= start {
		trimmed = trimmed[start : end+1]
	}
	envelope := travelWorkspaceAgentEnvelope{}
	if err := json.Unmarshal([]byte(trimmed), &envelope); err != nil {
		return travelWorkspaceAgentEnvelope{}, fmt.Errorf("invalid workspace agent json: %w", err)
	}
	return envelope, nil
}

func normalizeTravelWorkspaceAction(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "create":
		return "create"
	case "revise":
		return "revise"
	default:
		return travelWorkspaceReplyOnlyAction
	}
}

func buildReviseInputFromCreateInput(input productdomain.TravelGuideCreateInput) productdomain.TravelGuideReviseInput {
	revised := productdomain.TravelGuideReviseInput{
		TravelStyle:            strings.TrimSpace(input.TravelStyle),
		Budget:                 strings.TrimSpace(input.Budget),
		Companions:             append([]string(nil), input.Companions...),
		MustVisit:              append([]string(nil), input.MustVisit...),
		Avoid:                  append([]string(nil), input.Avoid...),
		AdditionalRequirements: append([]string(nil), input.AdditionalRequirements...),
	}
	if input.Days > 0 {
		days := input.Days
		revised.Days = &days
	}
	return revised
}

func (s *Server) findTravelGuideByCity(city string) (productdomain.TravelGuide, bool) {
	target := strings.TrimSpace(city)
	if target == "" || s.travelGuides == nil {
		return productdomain.TravelGuide{}, false
	}
	for _, guide := range s.travelGuides.ListGuides() {
		if strings.EqualFold(strings.TrimSpace(guide.City), target) {
			return guide, true
		}
	}
	return productdomain.TravelGuide{}, false
}

func (s *Server) resolveTravelGuideForRevision(selectedGuide productdomain.TravelGuide, targetCity string) (productdomain.TravelGuide, error) {
	if strings.TrimSpace(selectedGuide.ID) != "" {
		if city := strings.TrimSpace(targetCity); city == "" || strings.EqualFold(strings.TrimSpace(selectedGuide.City), city) {
			return selectedGuide, nil
		}
	}
	if guide, found := s.findTravelGuideByCity(targetCity); found {
		return guide, nil
	}
	if strings.TrimSpace(selectedGuide.ID) != "" {
		return selectedGuide, nil
	}
	return productdomain.TravelGuide{}, errors.New("choose a city page first or mention an existing city to revise")
}

func chooseWorkspaceReply(preferred string, fallback string) string {
	if strings.TrimSpace(preferred) != "" {
		return strings.TrimSpace(preferred)
	}
	return strings.TrimSpace(fallback)
}

package web

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	agentapp "alter0/internal/agent/application"
	controlapp "alter0/internal/control/application"
	controldomain "alter0/internal/control/domain"
	execdomain "alter0/internal/execution/domain"
	productapp "alter0/internal/product/application"
	productdomain "alter0/internal/product/domain"
	shareddomain "alter0/internal/shared/domain"
	"alter0/internal/shared/infrastructure/observability"
)

type stubWebOrchestrator struct {
	result      shareddomain.OrchestrationResult
	err         error
	lastMessage shareddomain.UnifiedMessage
}

func (s *stubWebOrchestrator) Handle(_ context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error) {
	s.lastMessage = msg
	return s.result, s.err
}

type sequenceIDGenerator struct {
	ids  []string
	next int
}

func (g *sequenceIDGenerator) NewID() string {
	if g.next >= len(g.ids) {
		id := "generated-" + strconv.Itoa(g.next)
		g.next++
		return id
	}
	id := g.ids[g.next]
	g.next++
	return id
}

func newMessageTestServer(orchestrator Orchestrator) *Server {
	return &Server{
		orchestrator: orchestrator,
		telemetry:    observability.NewTelemetry(),
		idGenerator: &sequenceIDGenerator{
			ids: []string{"session-generated", "message-generated", "trace-generated"},
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func TestMessageHandlerJSONFallbackPath(t *testing.T) {
	orchestrator := &stubWebOrchestrator{
		result: shareddomain.OrchestrationResult{
			MessageID: "message-generated",
			SessionID: "session-fixed",
			Route:     shareddomain.RouteCommand,
			Output:    "fallback-ok",
		},
	}
	server := newMessageTestServer(orchestrator)

	req := httptest.NewRequest(http.MethodPost, "/api/messages", strings.NewReader(`{"session_id":"session-fixed","content":"hello"}`))
	rec := httptest.NewRecorder()
	server.messageHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var body messageResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Result.Output != "fallback-ok" {
		t.Fatalf("expected output fallback-ok, got %q", body.Result.Output)
	}
	if body.Result.Route != shareddomain.RouteCommand {
		t.Fatalf("expected route command, got %q", body.Result.Route)
	}
	if orchestrator.lastMessage.SessionID != "session-fixed" {
		t.Fatalf("expected session-fixed, got %q", orchestrator.lastMessage.SessionID)
	}
	if orchestrator.lastMessage.ChannelID != "web-default" {
		t.Fatalf("expected default channel, got %q", orchestrator.lastMessage.ChannelID)
	}
}

func TestMessageStreamHandlerEmitsStartDeltaDone(t *testing.T) {
	output := strings.Repeat("stream-", 10)
	orchestrator := &stubWebOrchestrator{
		result: shareddomain.OrchestrationResult{
			MessageID: "message-generated",
			SessionID: "session-fixed",
			Route:     shareddomain.RouteNL,
			Output:    output,
		},
	}
	server := newMessageTestServer(orchestrator)

	req := httptest.NewRequest(http.MethodPost, "/api/messages/stream", strings.NewReader(`{"session_id":"session-fixed","content":"hello"}`))
	rec := httptest.NewRecorder()
	server.messageStreamHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("expected text/event-stream, got %q", contentType)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "event: start\n") {
		t.Fatalf("expected start event, got body: %s", body)
	}
	if strings.Count(body, "event: delta\n") < 2 {
		t.Fatalf("expected multiple delta events, got body: %s", body)
	}
	if !strings.Contains(body, "event: done\n") {
		t.Fatalf("expected done event, got body: %s", body)
	}
	if !strings.Contains(body, `"output":"`+output+`"`) {
		t.Fatalf("expected full output in done payload, got body: %s", body)
	}
}

func TestMessageStreamHandlerEmitsError(t *testing.T) {
	orchestrator := &stubWebOrchestrator{
		result: shareddomain.OrchestrationResult{
			MessageID: "message-generated",
			SessionID: "session-fixed",
			Route:     shareddomain.RouteNL,
			ErrorCode: "nl_execution_failed",
		},
		err: errors.New("processor failed"),
	}
	server := newMessageTestServer(orchestrator)

	req := httptest.NewRequest(http.MethodPost, "/api/messages/stream", strings.NewReader(`{"session_id":"session-fixed","content":"hello"}`))
	rec := httptest.NewRecorder()
	server.messageStreamHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "event: start\n") {
		t.Fatalf("expected start event before failure, got body: %s", body)
	}
	if !strings.Contains(body, "event: error\n") {
		t.Fatalf("expected error event, got body: %s", body)
	}
	if strings.Contains(body, "event: done\n") {
		t.Fatalf("did not expect done event on error, got body: %s", body)
	}
	if !strings.Contains(body, `"error":"processor failed"`) {
		t.Fatalf("expected error message in payload, got body: %s", body)
	}
}

func TestAgentMessageHandlerInjectsAgentProfileMetadata(t *testing.T) {
	orchestrator := &stubWebOrchestrator{
		result: shareddomain.OrchestrationResult{
			MessageID: "message-generated",
			SessionID: "session-fixed",
			Route:     shareddomain.RouteNL,
			Output:    "agent-ok",
		},
	}
	control := controlapp.NewService()
	if err := control.UpsertChannel(controldomain.Channel{
		ID:      "web-default",
		Type:    shareddomain.ChannelTypeWeb,
		Enabled: true,
	}); err != nil {
		t.Fatalf("upsert channel failed: %v", err)
	}
	if err := control.UpsertAgent(controldomain.Agent{
		ID:            "researcher",
		Name:          "Researcher",
		Type:          controldomain.CapabilityTypeAgent,
		Enabled:       true,
		Scope:         controldomain.CapabilityScopeGlobal,
		Version:       "v1.0.0",
		ProviderID:    "openai",
		Model:         "gpt-4o",
		SystemPrompt:  "Execute first.",
		MaxIterations: 7,
		Tools:         []string{"codex_exec"},
		Skills:        []string{"summary"},
		MCPs:          []string{"github"},
		MemoryFiles:   []string{"user_md", "soul_md"},
	}); err != nil {
		t.Fatalf("upsert agent failed: %v", err)
	}
	server := newMessageTestServer(orchestrator)
	server.control = control
	server.agents = agentapp.NewCatalog(control)

	req := httptest.NewRequest(http.MethodPost, "/api/agent/messages", strings.NewReader(`{"agent_id":"researcher","session_id":"session-fixed","content":"完成仓库整理"}`))
	rec := httptest.NewRecorder()
	server.agentMessageHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if orchestrator.lastMessage.Metadata[execdomain.ExecutionEngineMetadataKey] != execdomain.ExecutionEngineAgent {
		t.Fatalf("expected agent execution engine, got %+v", orchestrator.lastMessage.Metadata)
	}
	if orchestrator.lastMessage.Metadata[execdomain.AgentIDMetadataKey] != "researcher" {
		t.Fatalf("expected agent id researcher, got %+v", orchestrator.lastMessage.Metadata)
	}
	if orchestrator.lastMessage.Metadata[execdomain.AgentSystemPromptMetadataKey] != "Execute first." {
		t.Fatalf("expected system prompt injected, got %+v", orchestrator.lastMessage.Metadata)
	}
	if !strings.Contains(orchestrator.lastMessage.Metadata["alter0.skills.include"], "summary") {
		t.Fatalf("expected skill selection injected, got %+v", orchestrator.lastMessage.Metadata)
	}
	if !strings.Contains(orchestrator.lastMessage.Metadata["alter0.mcp.request.enable"], "github") {
		t.Fatalf("expected mcp selection injected, got %+v", orchestrator.lastMessage.Metadata)
	}
	if !strings.Contains(orchestrator.lastMessage.Metadata["alter0.memory.include"], "user_md") {
		t.Fatalf("expected memory file selection injected, got %+v", orchestrator.lastMessage.Metadata)
	}
}

func TestAgentMessageHandlerKeepsExplicitRuntimeSelections(t *testing.T) {
	orchestrator := &stubWebOrchestrator{
		result: shareddomain.OrchestrationResult{
			MessageID: "message-generated",
			SessionID: "session-fixed",
			Route:     shareddomain.RouteNL,
			Output:    "agent-ok",
		},
	}
	control := controlapp.NewService()
	if err := control.UpsertChannel(controldomain.Channel{
		ID:      "web-default",
		Type:    shareddomain.ChannelTypeWeb,
		Enabled: true,
	}); err != nil {
		t.Fatalf("upsert channel failed: %v", err)
	}
	if err := control.UpsertAgent(controldomain.Agent{
		ID:         "researcher",
		Name:       "Researcher",
		Type:       controldomain.CapabilityTypeAgent,
		Enabled:    true,
		Scope:      controldomain.CapabilityScopeGlobal,
		Version:    "v1.0.0",
		ProviderID: "openai",
		Model:      "gpt-4o",
		Tools:      []string{"codex_exec"},
		Skills:     []string{"summary"},
		MCPs:       []string{"github"},
	}); err != nil {
		t.Fatalf("upsert agent failed: %v", err)
	}
	server := newMessageTestServer(orchestrator)
	server.control = control
	server.agents = agentapp.NewCatalog(control)

	req := httptest.NewRequest(http.MethodPost, "/api/agent/messages", strings.NewReader(`{
		"agent_id":"researcher",
		"session_id":"session-fixed",
		"content":"完成仓库整理",
		"metadata":{
			"alter0.agent.tools":"[]",
			"alter0.skills.include":"[\"planner\"]",
			"alter0.mcp.request.enable":"[]"
		}
	}`))
	rec := httptest.NewRecorder()
	server.agentMessageHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if got := orchestrator.lastMessage.Metadata[execdomain.AgentToolsMetadataKey]; got != "[]" {
		t.Fatalf("expected explicit tool selection preserved, got %q", got)
	}
	if got := orchestrator.lastMessage.Metadata["alter0.skills.include"]; got != "[\"planner\"]" {
		t.Fatalf("expected explicit skill selection preserved, got %q", got)
	}
	if got := orchestrator.lastMessage.Metadata["alter0.mcp.request.enable"]; got != "[]" {
		t.Fatalf("expected explicit mcp selection preserved, got %q", got)
	}
}

func TestProductMessageHandlerRoutesToProductMasterAgent(t *testing.T) {
	orchestrator := &stubWebOrchestrator{
		result: shareddomain.OrchestrationResult{
			MessageID: "message-generated",
			SessionID: "session-fixed",
			Route:     shareddomain.RouteNL,
			Output:    "travel-guide-ok",
		},
	}
	control := controlapp.NewService()
	if err := control.UpsertChannel(controldomain.Channel{
		ID:      "web-default",
		Type:    shareddomain.ChannelTypeWeb,
		Enabled: true,
	}); err != nil {
		t.Fatalf("upsert channel failed: %v", err)
	}
	server := newMessageTestServer(orchestrator)
	server.control = control
	server.agents = agentapp.NewCatalog(control)
	server.products = productapp.NewService()

	req := httptest.NewRequest(http.MethodPost, "/api/products/travel/messages", strings.NewReader(`{"session_id":"session-fixed","content":"生成一个上海三日游攻略"}`))
	rec := httptest.NewRecorder()
	server.publicProductItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if orchestrator.lastMessage.Metadata[execdomain.ExecutionEngineMetadataKey] != execdomain.ExecutionEngineAgent {
		t.Fatalf("expected agent execution engine, got %+v", orchestrator.lastMessage.Metadata)
	}
	if orchestrator.lastMessage.Metadata[execdomain.AgentIDMetadataKey] != "travel-master" {
		t.Fatalf("expected master agent travel-master, got %+v", orchestrator.lastMessage.Metadata)
	}
	rawProductContext := orchestrator.lastMessage.Metadata[execdomain.ProductContextMetadataKey]
	if strings.TrimSpace(rawProductContext) == "" {
		t.Fatalf("expected product context metadata, got %+v", orchestrator.lastMessage.Metadata)
	}
	if !strings.Contains(rawProductContext, `"product_id":"travel"`) {
		t.Fatalf("expected travel product context, got %q", rawProductContext)
	}
	if !strings.Contains(rawProductContext, `"master_agent_id":"travel-master"`) {
		t.Fatalf("expected travel master agent in product context, got %q", rawProductContext)
	}
}

func TestProductMessageHandlerRejectsNonPublicProductExecution(t *testing.T) {
	orchestrator := &stubWebOrchestrator{
		result: shareddomain.OrchestrationResult{
			MessageID: "message-generated",
			SessionID: "session-fixed",
			Route:     shareddomain.RouteNL,
			Output:    "hidden-product",
		},
	}
	control := controlapp.NewService()
	if err := control.UpsertChannel(controldomain.Channel{
		ID:      "web-default",
		Type:    shareddomain.ChannelTypeWeb,
		Enabled: true,
	}); err != nil {
		t.Fatalf("upsert channel failed: %v", err)
	}
	products := productapp.NewService()
	if _, err := products.CreateProduct(productdomain.Product{
		Name:          "Internal Planner",
		Slug:          "internal-planner",
		Summary:       "Private product should not expose public execution endpoints.",
		Status:        productdomain.StatusActive,
		Visibility:    productdomain.VisibilityPrivate,
		MasterAgentID: "travel-master",
	}); err != nil {
		t.Fatalf("create product failed: %v", err)
	}
	server := newMessageTestServer(orchestrator)
	server.control = control
	server.agents = agentapp.NewCatalog(control)
	server.products = products

	req := httptest.NewRequest(http.MethodPost, "/api/products/internal-planner/messages", strings.NewReader(`{"session_id":"session-fixed","content":"执行内部产品任务"}`))
	rec := httptest.NewRecorder()
	server.publicProductItemHandler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, rec.Code, rec.Body.String())
	}
}

func TestTravelGuideEndpointsLifecycle(t *testing.T) {
	server := newMessageTestServer(&stubWebOrchestrator{})
	server.products = productapp.NewService()
	server.travelGuides = productapp.NewTravelGuideService()

	createReq := httptest.NewRequest(http.MethodPost, "/api/products/travel/guides", strings.NewReader(`{
		"city":"Shanghai",
		"days":3,
		"travel_style":"metro-first",
		"budget":"mid-range",
		"must_visit":["The Bund","Yu Garden"],
		"additional_requirements":["more local food"]
	}`))
	createRec := httptest.NewRecorder()
	server.publicProductItemHandler(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d: %s", createRec.Code, createRec.Body.String())
	}
	var created productdomain.TravelGuide
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatalf("decode created guide failed: %v", err)
	}
	if created.ID == "" || created.City != "Shanghai" {
		t.Fatalf("unexpected created guide: %+v", created)
	}
	if len(created.MapLayers) == 0 {
		t.Fatalf("expected map layers in guide: %+v", created)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/products/travel/guides/"+created.ID, nil)
	getRec := httptest.NewRecorder()
	server.publicProductItemHandler(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected get 200, got %d: %s", getRec.Code, getRec.Body.String())
	}

	reviseReq := httptest.NewRequest(http.MethodPost, "/api/products/travel/guides/"+created.ID+"/revise", strings.NewReader(`{
		"days":4,
		"keep_conditions":["keep The Bund"],
		"replace_conditions":["less museum time"],
		"additional_requirements":["slow mornings"]
	}`))
	reviseRec := httptest.NewRecorder()
	server.publicProductItemHandler(reviseRec, reviseReq)
	if reviseRec.Code != http.StatusOK {
		t.Fatalf("expected revise 200, got %d: %s", reviseRec.Code, reviseRec.Body.String())
	}
	var revised productdomain.TravelGuide
	if err := json.NewDecoder(reviseRec.Body).Decode(&revised); err != nil {
		t.Fatalf("decode revised guide failed: %v", err)
	}
	if revised.Revision != 2 || revised.Days != 4 {
		t.Fatalf("unexpected revised guide: %+v", revised)
	}
	if len(revised.KeepConditions) == 0 || len(revised.ReplaceConditions) == 0 {
		t.Fatalf("expected revision conditions to persist: %+v", revised)
	}
}

func TestProductWorkspaceSummaryExposesTravelSpaces(t *testing.T) {
	server := newMessageTestServer(&stubWebOrchestrator{})
	server.products = productapp.NewService()
	server.agents = agentapp.NewCatalog(controlapp.NewService())
	server.travelGuides = productapp.NewTravelGuideService()
	if _, err := server.travelGuides.CreateGuide(productdomain.TravelGuideCreateInput{
		City:      "武汉",
		Days:      3,
		MustVisit: []string{"黄鹤楼"},
	}); err != nil {
		t.Fatalf("seed guide failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/products/travel/workspace", nil)
	rec := httptest.NewRecorder()
	server.publicProductItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected workspace 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload productWorkspaceResponse
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode workspace failed: %v", err)
	}
	if payload.Product.ID != "travel" {
		t.Fatalf("expected travel product, got %+v", payload)
	}
	if payload.MasterAgent == nil || payload.MasterAgent.AgentID != "travel-master" {
		t.Fatalf("expected travel master agent, got %+v", payload.MasterAgent)
	}
	if len(payload.Spaces) != 1 || payload.Spaces[0].Title != "武汉" {
		t.Fatalf("expected wuhan space summary, got %+v", payload.Spaces)
	}
}

func TestProductWorkspaceSpaceDetailReturnsGuide(t *testing.T) {
	server := newMessageTestServer(&stubWebOrchestrator{})
	server.products = productapp.NewService()
	server.travelGuides = productapp.NewTravelGuideService()
	guide, err := server.travelGuides.CreateGuide(productdomain.TravelGuideCreateInput{
		City:      "北京",
		Days:      4,
		MustVisit: []string{"故宫"},
	})
	if err != nil {
		t.Fatalf("seed guide failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/products/travel/workspace/spaces/"+guide.ID, nil)
	rec := httptest.NewRecorder()
	server.publicProductItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected workspace detail 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload productWorkspaceSpaceDetail
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode workspace detail failed: %v", err)
	}
	if payload.Space.SpaceID != guide.ID || payload.Guide.City != "北京" {
		t.Fatalf("unexpected workspace detail: %+v", payload)
	}
}

func TestTravelWorkspaceChatCreatesGuideFromMasterAgentReply(t *testing.T) {
	orchestrator := &stubWebOrchestrator{
		result: shareddomain.OrchestrationResult{
			MessageID: "message-generated",
			SessionID: "session-fixed",
			Route:     shareddomain.RouteNL,
			Output:    `{"action":"create","target_city":"武汉","assistant_reply":"已为武汉创建城市页。","create_input":{"city":"武汉","days":3,"travel_style":"citywalk","budget":"mid-range","must_visit":["黄鹤楼"],"additional_requirements":["地铁优先"]}}`,
		},
	}
	server := newMessageTestServer(orchestrator)
	server.products = productapp.NewService()
	server.travelGuides = productapp.NewTravelGuideService()
	server.agents = agentapp.NewCatalog(controlapp.NewService())

	req := httptest.NewRequest(http.MethodPost, "/api/products/travel/workspace/chat", strings.NewReader(`{"content":"给武汉创建一个地铁优先的三日游页面"}`))
	rec := httptest.NewRecorder()
	server.publicProductItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected workspace chat 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload productWorkspaceChatResponse
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode workspace chat failed: %v", err)
	}
	if payload.Action != "create" || payload.Guide == nil || payload.Guide.City != "武汉" {
		t.Fatalf("unexpected workspace chat payload: %+v", payload)
	}
	if got := orchestrator.lastMessage.Metadata[execdomain.AgentIDMetadataKey]; got != "travel-master" {
		t.Fatalf("expected travel-master metadata, got %+v", orchestrator.lastMessage.Metadata)
	}
	if got := orchestrator.lastMessage.Metadata[execdomain.AgentToolsMetadataKey]; got != `["complete"]` {
		t.Fatalf("expected complete-only tools, got %+v", orchestrator.lastMessage.Metadata)
	}
}

func TestTravelWorkspaceChatRevisesSelectedGuide(t *testing.T) {
	orchestrator := &stubWebOrchestrator{
		result: shareddomain.OrchestrationResult{
			MessageID: "message-generated",
			SessionID: "session-fixed",
			Route:     shareddomain.RouteNL,
			Output:    `{"action":"revise","assistant_reply":"已更新成都页面。","revise_input":{"days":4,"additional_requirements":["更多美食","慢节奏早晨"],"keep_conditions":["保留锦里"]}}`,
		},
	}
	server := newMessageTestServer(orchestrator)
	server.products = productapp.NewService()
	server.travelGuides = productapp.NewTravelGuideService()
	server.agents = agentapp.NewCatalog(controlapp.NewService())
	guide, err := server.travelGuides.CreateGuide(productdomain.TravelGuideCreateInput{
		City:      "成都",
		Days:      2,
		MustVisit: []string{"锦里"},
	})
	if err != nil {
		t.Fatalf("seed guide failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/products/travel/workspace/chat", strings.NewReader(`{"space_id":"`+guide.ID+`","content":"把成都页面调整成四天，并保留锦里，增加更多美食"}`))
	rec := httptest.NewRecorder()
	server.publicProductItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected revise 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload productWorkspaceChatResponse
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode revise response failed: %v", err)
	}
	if payload.Action != "revise" || payload.Guide == nil {
		t.Fatalf("unexpected revise payload: %+v", payload)
	}
	if payload.Guide.Revision != 2 || payload.Guide.Days != 4 {
		t.Fatalf("expected revised guide, got %+v", payload.Guide)
	}
}

func TestMainAgentMessageRoutesMatchedProductTask(t *testing.T) {
	orchestrator := &stubWebOrchestrator{
		result: shareddomain.OrchestrationResult{
			MessageID: "message-generated",
			SessionID: "session-fixed",
			Route:     shareddomain.RouteNL,
			Output:    "travel-guide-ok",
		},
	}
	control := controlapp.NewService()
	if err := control.UpsertChannel(controldomain.Channel{
		ID:      "web-default",
		Type:    shareddomain.ChannelTypeWeb,
		Enabled: true,
	}); err != nil {
		t.Fatalf("upsert channel failed: %v", err)
	}
	server := newMessageTestServer(orchestrator)
	server.control = control
	server.agents = agentapp.NewCatalog(control)
	server.products = productapp.NewService()

	req := httptest.NewRequest(http.MethodPost, "/api/agent/messages", strings.NewReader(`{"agent_id":"main","session_id":"session-fixed","content":"生成一份上海三日游攻略，重点地铁和美食"}`))
	rec := httptest.NewRecorder()
	server.agentMessageHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if got := orchestrator.lastMessage.Metadata[execdomain.AgentIDMetadataKey]; got != "travel-master" {
		t.Fatalf("expected travel-master routing, got %+v", orchestrator.lastMessage.Metadata)
	}
	if got := orchestrator.lastMessage.Metadata[execdomain.AgentDelegatedByMetadataKey]; got != "main" {
		t.Fatalf("expected delegated_by main, got %+v", orchestrator.lastMessage.Metadata)
	}
	if got := orchestrator.lastMessage.Metadata[execdomain.ProductSelectedIDMetadataKey]; got != "travel" {
		t.Fatalf("expected selected product travel, got %+v", orchestrator.lastMessage.Metadata)
	}
	if got := orchestrator.lastMessage.Metadata[execdomain.ProductExecutionModeMetadataKey]; got != "product-master" {
		t.Fatalf("expected product execution mode product-master, got %+v", orchestrator.lastMessage.Metadata)
	}
}

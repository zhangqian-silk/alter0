package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	agentapp "alter0/internal/agent/application"
	controlapp "alter0/internal/control/application"
	controldomain "alter0/internal/control/domain"
	productapp "alter0/internal/product/application"
	productdomain "alter0/internal/product/domain"
)

type stubRuntimeRestarter struct {
	accepted bool
	err      error
	called   int
	options  RuntimeRestartOptions
}

type stubRuntimeInfoProvider struct {
	info RuntimeInfo
}

func (s *stubRuntimeRestarter) RequestRestart(options RuntimeRestartOptions) (bool, error) {
	s.called++
	s.options = options
	return s.accepted, s.err
}

func (s *stubRuntimeInfoProvider) GetRuntimeInfo() RuntimeInfo {
	if s == nil {
		return RuntimeInfo{}
	}
	return s.info
}

func TestSkillEndpointUsesUnifiedCapabilityFields(t *testing.T) {
	server := &Server{
		control: controlapp.NewService(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	putReq := httptest.NewRequest(http.MethodPut, "/api/control/skills/summary", strings.NewReader(`{"name":"Summary","scope":"global","version":"v1.2.3","enabled":true,"metadata":{"owner":"platform"}}`))
	putRec := httptest.NewRecorder()
	server.skillItemHandler(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("expected put 200, got %d: %s", putRec.Code, putRec.Body.String())
	}

	var capability controldomain.Capability
	if err := json.NewDecoder(putRec.Body).Decode(&capability); err != nil {
		t.Fatalf("decode put response failed: %v", err)
	}
	if capability.Type != controldomain.CapabilityTypeSkill {
		t.Fatalf("expected type skill, got %s", capability.Type)
	}
	if capability.Scope != controldomain.CapabilityScopeGlobal {
		t.Fatalf("expected scope global, got %s", capability.Scope)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/control/skills", nil)
	listRec := httptest.NewRecorder()
	server.skillListHandler(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d", listRec.Code)
	}
	var listResp struct {
		Items []controldomain.Capability `json:"items"`
	}
	if err := json.NewDecoder(listRec.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list response failed: %v", err)
	}
	if len(listResp.Items) != 1 || listResp.Items[0].Type != controldomain.CapabilityTypeSkill {
		t.Fatalf("unexpected list items: %+v", listResp.Items)
	}
}

func TestMCPLifecycleAndAudit(t *testing.T) {
	server := &Server{
		control: controlapp.NewService(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	putReq := httptest.NewRequest(http.MethodPut, "/api/control/mcps/github", strings.NewReader(`{"name":"GitHub MCP","scope":"session","version":"v2.0.0","enabled":true}`))
	putRec := httptest.NewRecorder()
	server.mcpItemHandler(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("expected put 200, got %d: %s", putRec.Code, putRec.Body.String())
	}

	disableReq := httptest.NewRequest(http.MethodPost, "/api/control/mcps/github", strings.NewReader(`{"action":"disable"}`))
	disableRec := httptest.NewRecorder()
	server.mcpItemHandler(disableRec, disableReq)
	if disableRec.Code != http.StatusOK {
		t.Fatalf("expected disable 200, got %d: %s", disableRec.Code, disableRec.Body.String())
	}

	var disabled controldomain.Capability
	if err := json.NewDecoder(disableRec.Body).Decode(&disabled); err != nil {
		t.Fatalf("decode disable response failed: %v", err)
	}
	if disabled.Enabled {
		t.Fatalf("expected capability disabled")
	}

	auditReq := httptest.NewRequest(http.MethodGet, "/api/control/capabilities/audit?type=mcp", nil)
	auditRec := httptest.NewRecorder()
	server.capabilityAuditListHandler(auditRec, auditReq)
	if auditRec.Code != http.StatusOK {
		t.Fatalf("expected audit 200, got %d", auditRec.Code)
	}
	var auditResp struct {
		Items []controldomain.CapabilityAudit `json:"items"`
	}
	if err := json.NewDecoder(auditRec.Body).Decode(&auditResp); err != nil {
		t.Fatalf("decode audit response failed: %v", err)
	}
	if len(auditResp.Items) < 2 {
		t.Fatalf("expected at least 2 audit entries, got %d", len(auditResp.Items))
	}
	if auditResp.Items[0].Action != controldomain.CapabilityLifecycleUpdate {
		t.Fatalf("expected first audit update, got %s", auditResp.Items[0].Action)
	}
	if auditResp.Items[1].Action != controldomain.CapabilityLifecycleDisable {
		t.Fatalf("expected second audit disable, got %s", auditResp.Items[1].Action)
	}
}

func TestCapabilityUnifiedAPI(t *testing.T) {
	server := &Server{
		control: controlapp.NewService(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	putReq := httptest.NewRequest(http.MethodPut, "/api/control/capabilities/skill/default-nl", strings.NewReader(`{"name":"Default NL","version":"v1.0.0","scope":"global"}`))
	putRec := httptest.NewRecorder()
	server.capabilityItemHandler(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("expected capability put 200, got %d: %s", putRec.Code, putRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/control/capabilities?type=skill", nil)
	listRec := httptest.NewRecorder()
	server.capabilityListHandler(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected capability list 200, got %d", listRec.Code)
	}
	var listResp struct {
		Items []controldomain.Capability `json:"items"`
	}
	if err := json.NewDecoder(listRec.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode capability list failed: %v", err)
	}
	if len(listResp.Items) != 1 || listResp.Items[0].ID != "default-nl" {
		t.Fatalf("unexpected capability list: %+v", listResp.Items)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/control/capabilities/skill/default-nl", nil)
	deleteRec := httptest.NewRecorder()
	server.capabilityItemHandler(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("expected capability delete 200, got %d", deleteRec.Code)
	}
}

func TestAgentEndpointCreatesManagedIdentityAndVersion(t *testing.T) {
	control := controlapp.NewService()
	server := &Server{
		control: control,
		agents:  agentapp.NewCatalog(control),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	createReq := httptest.NewRequest(http.MethodPost, "/api/control/agents", strings.NewReader(`{
		"name":"Researcher",
		"enabled":true,
		"system_prompt":"Execute first, summarize later.",
		"max_iterations":8,
		"tools":["codex_exec"],
		"skills":["summary"],
		"mcps":["github"]
	}`))
	createRec := httptest.NewRecorder()
	server.agentListHandler(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d: %s", createRec.Code, createRec.Body.String())
	}

	var agent controldomain.Agent
	if err := json.NewDecoder(createRec.Body).Decode(&agent); err != nil {
		t.Fatalf("decode create response failed: %v", err)
	}
	if agent.Type != controldomain.CapabilityTypeAgent {
		t.Fatalf("expected type agent, got %s", agent.Type)
	}
	if agent.ID != "researcher" {
		t.Fatalf("expected generated id researcher, got %s", agent.ID)
	}
	if agent.Version != "v1.0.0" {
		t.Fatalf("expected managed version v1.0.0, got %s", agent.Version)
	}
	if agent.MaxIterations != 8 {
		t.Fatalf("expected max_iterations 8, got %d", agent.MaxIterations)
	}
	if len(agent.Tools) != 1 || agent.Tools[0] != "codex_exec" {
		t.Fatalf("unexpected tools: %+v", agent.Tools)
	}

	updateReq := httptest.NewRequest(http.MethodPut, "/api/control/agents/researcher", strings.NewReader(`{
		"name":"Researcher",
		"enabled":true,
		"system_prompt":"Execute directly and close the loop.",
		"max_iterations":10,
		"tools":["codex_exec"],
		"skills":["summary"],
		"mcps":["github"]
	}`))
	updateRec := httptest.NewRecorder()
	server.agentItemHandler(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected update 200, got %d: %s", updateRec.Code, updateRec.Body.String())
	}
	if err := json.NewDecoder(updateRec.Body).Decode(&agent); err != nil {
		t.Fatalf("decode update response failed: %v", err)
	}
	if agent.Version != "v1.0.1" {
		t.Fatalf("expected managed version bump v1.0.1, got %s", agent.Version)
	}
	if agent.ProviderID != "" || agent.Model != "" {
		t.Fatalf("expected provider/model to be empty, got %+v", agent)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/control/agents", nil)
	listRec := httptest.NewRecorder()
	server.agentListHandler(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d", listRec.Code)
	}
	var listResp struct {
		Items []controldomain.Agent `json:"items"`
	}
	if err := json.NewDecoder(listRec.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list response failed: %v", err)
	}
	if len(listResp.Items) != 1 || listResp.Items[0].ID != "researcher" {
		t.Fatalf("unexpected list items: %+v", listResp.Items)
	}
}

func TestRuntimeAgentCatalogListsBuiltinEntrypoints(t *testing.T) {
	control := controlapp.NewService()
	server := &Server{
		control: control,
		agents:  agentapp.NewCatalog(control),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	rec := httptest.NewRecorder()
	server.runtimeAgentListHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var response struct {
		Items []controldomain.Agent `json:"items"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode runtime agent list failed: %v", err)
	}
	if len(response.Items) < 3 {
		t.Fatalf("expected builtin agent entries, got %+v", response.Items)
	}
	if response.Items[0].ID != "main" || response.Items[0].Source != controldomain.AgentSourceBuiltin {
		t.Fatalf("expected main builtin first, got %+v", response.Items[0])
	}
}

func TestManagedAgentCreateSkipsBuiltinReservedID(t *testing.T) {
	control := controlapp.NewService()
	server := &Server{
		control: control,
		agents:  agentapp.NewCatalog(control),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	createReq := httptest.NewRequest(http.MethodPost, "/api/control/agents", strings.NewReader(`{
		"name":"Coding",
		"enabled":true
	}`))
	createRec := httptest.NewRecorder()
	server.agentListHandler(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d: %s", createRec.Code, createRec.Body.String())
	}
	var agent controldomain.Agent
	if err := json.NewDecoder(createRec.Body).Decode(&agent); err != nil {
		t.Fatalf("decode create response failed: %v", err)
	}
	if agent.ID != "coding-2" {
		t.Fatalf("expected managed agent id coding-2 to avoid builtin collision, got %s", agent.ID)
	}
}

func TestBuiltinAgentCannotBeOverwrittenFromControlEndpoint(t *testing.T) {
	control := controlapp.NewService()
	server := &Server{
		control: control,
		agents:  agentapp.NewCatalog(control),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodPut, "/api/control/agents/main", strings.NewReader(`{"name":"Main Override","enabled":true}`))
	rec := httptest.NewRecorder()
	server.agentItemHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected builtin overwrite rejection 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProductEndpointsExposeBuiltinAndManagedLifecycle(t *testing.T) {
	server := &Server{
		products: productapp.NewService(),
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/control/products", nil)
	listRec := httptest.NewRecorder()
	server.productListHandler(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected product list 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	var initialList struct {
		Items []productdomain.Product `json:"items"`
	}
	if err := json.NewDecoder(listRec.Body).Decode(&initialList); err != nil {
		t.Fatalf("decode initial product list failed: %v", err)
	}
	if len(initialList.Items) == 0 || initialList.Items[0].ID != "travel" {
		t.Fatalf("expected builtin travel product, got %+v", initialList.Items)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/api/control/products", strings.NewReader(`{
		"name":"City Guides",
		"slug":"city-guides",
		"summary":"Managed product for curated city plans.",
		"status":"active",
		"visibility":"public",
		"master_agent_id":"city-guides-master",
		"entry_route":"products",
		"tags":["travel","guide"],
		"artifact_types":["city_guide","map_layers"],
		"knowledge_sources":["poi_catalog","metro_network"],
		"worker_agents":[
			{"agent_id":"city-guides-planner","role":"planner","responsibility":"Build day plans","enabled":true}
		]
	}`))
	createRec := httptest.NewRecorder()
	server.productListHandler(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d: %s", createRec.Code, createRec.Body.String())
	}
	var created productdomain.Product
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatalf("decode product create response failed: %v", err)
	}
	if created.ID != "city-guides" {
		t.Fatalf("expected generated product id city-guides, got %s", created.ID)
	}
	if created.OwnerType != productdomain.OwnerTypeManaged {
		t.Fatalf("expected managed owner type, got %s", created.OwnerType)
	}
	if created.Version != productdomain.DefaultVersion {
		t.Fatalf("expected default version %s, got %s", productdomain.DefaultVersion, created.Version)
	}

	updateReq := httptest.NewRequest(http.MethodPut, "/api/control/products/city-guides", strings.NewReader(`{
		"name":"City Guides",
		"slug":"city-guides",
		"summary":"Updated managed product summary.",
		"status":"active",
		"visibility":"public",
		"master_agent_id":"city-guides-master",
		"entry_route":"products",
		"worker_agents":[
			{"agent_id":"city-guides-planner","role":"planner","responsibility":"Refine city routes","enabled":true},
			{"agent_id":"city-guides-food","role":"food","responsibility":"Recommend local food","enabled":true}
		]
	}`))
	updateRec := httptest.NewRecorder()
	server.productItemHandler(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected update 200, got %d: %s", updateRec.Code, updateRec.Body.String())
	}
	var updated productdomain.Product
	if err := json.NewDecoder(updateRec.Body).Decode(&updated); err != nil {
		t.Fatalf("decode product update response failed: %v", err)
	}
	if updated.Version != "v1.0.1" {
		t.Fatalf("expected version bump v1.0.1, got %s", updated.Version)
	}
	if len(updated.WorkerAgents) != 2 {
		t.Fatalf("expected 2 worker agents after update, got %+v", updated.WorkerAgents)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/control/products/city-guides", nil)
	deleteRec := httptest.NewRecorder()
	server.productItemHandler(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("expected delete 200, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
}

func TestPublicProductEndpointsOnlyExposeActivePublicProducts(t *testing.T) {
	products := productapp.NewService()
	if _, err := products.CreateProduct(productdomain.Product{
		Name:          "Internal Planner",
		Slug:          "internal-planner",
		Summary:       "Private draft product should stay hidden from public APIs.",
		Status:        productdomain.StatusDraft,
		Visibility:    productdomain.VisibilityPrivate,
		MasterAgentID: "internal-planner-master",
	}); err != nil {
		t.Fatalf("seed managed product failed: %v", err)
	}

	server := &Server{
		products: products,
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/products", nil)
	listRec := httptest.NewRecorder()
	server.publicProductListHandler(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected public product list 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	var listResp struct {
		Items []productdomain.Product `json:"items"`
	}
	if err := json.NewDecoder(listRec.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode public product list failed: %v", err)
	}
	if len(listResp.Items) != 1 || listResp.Items[0].ID != "travel" {
		t.Fatalf("expected only builtin travel product in public list, got %+v", listResp.Items)
	}

	itemReq := httptest.NewRequest(http.MethodGet, "/api/products/travel", nil)
	itemRec := httptest.NewRecorder()
	server.publicProductItemHandler(itemRec, itemReq)
	if itemRec.Code != http.StatusOK {
		t.Fatalf("expected public product item 200, got %d: %s", itemRec.Code, itemRec.Body.String())
	}

	hiddenReq := httptest.NewRequest(http.MethodGet, "/api/products/internal-planner", nil)
	hiddenRec := httptest.NewRecorder()
	server.publicProductItemHandler(hiddenRec, hiddenReq)
	if hiddenRec.Code != http.StatusNotFound {
		t.Fatalf("expected hidden product 404, got %d: %s", hiddenRec.Code, hiddenRec.Body.String())
	}
}

func TestEnvironmentConfigEndpoints(t *testing.T) {
	control := controlapp.NewService()
	control.SetEnvironmentRuntime(map[string]string{
		"worker_pool_size": "4",
		"queue_timeout":    "5s",
	})
	server := &Server{
		control: control,
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/control/environments", nil)
	listRec := httptest.NewRecorder()
	server.environmentConfigHandler(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d", listRec.Code)
	}
	var listResp struct {
		Items []controldomain.EnvironmentConfigItem `json:"items"`
	}
	if err := json.NewDecoder(listRec.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list response failed: %v", err)
	}
	if len(listResp.Items) == 0 {
		t.Fatalf("expected non-empty environments")
	}

	putReq := httptest.NewRequest(http.MethodPut, "/api/control/environments", strings.NewReader(`{"operator":"tester","values":{"worker_pool_size":8,"queue_timeout":"8s"}}`))
	putRec := httptest.NewRecorder()
	server.environmentConfigHandler(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("expected put 200, got %d: %s", putRec.Code, putRec.Body.String())
	}
	var putResp controldomain.EnvironmentUpdateResult
	if err := json.NewDecoder(putRec.Body).Decode(&putResp); err != nil {
		t.Fatalf("decode put response failed: %v", err)
	}
	if !putResp.NeedsRestart {
		t.Fatalf("expected restart required")
	}
	if len(putResp.Changed) != 2 {
		t.Fatalf("expected 2 changed items, got %d", len(putResp.Changed))
	}

	auditReq := httptest.NewRequest(http.MethodGet, "/api/control/environments/audits", nil)
	auditRec := httptest.NewRecorder()
	server.environmentAuditListHandler(auditRec, auditReq)
	if auditRec.Code != http.StatusOK {
		t.Fatalf("expected audit 200, got %d", auditRec.Code)
	}
	var auditResp struct {
		Items []controldomain.EnvironmentAudit `json:"items"`
	}
	if err := json.NewDecoder(auditRec.Body).Decode(&auditResp); err != nil {
		t.Fatalf("decode audit response failed: %v", err)
	}
	if len(auditResp.Items) != 1 {
		t.Fatalf("expected 1 audit, got %d", len(auditResp.Items))
	}
	if auditResp.Items[0].Operator != "tester" {
		t.Fatalf("expected operator tester, got %s", auditResp.Items[0].Operator)
	}
}

func TestRuntimeRestartEndpointAcceptsRequest(t *testing.T) {
	restarter := &stubRuntimeRestarter{accepted: true}
	server := &Server{
		runtime: restarter,
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/control/runtime/restart", nil)
	rec := httptest.NewRecorder()
	server.runtimeRestartHandler(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected accepted status, got %d: %s", rec.Code, rec.Body.String())
	}
	if restarter.options.SyncRemoteMaster {
		t.Fatalf("expected sync_remote_master false by default")
	}
}

func TestRuntimeRestartEndpointAcceptsSyncRemoteMasterOption(t *testing.T) {
	restarter := &stubRuntimeRestarter{accepted: true}
	server := &Server{
		runtime: restarter,
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/control/runtime/restart", strings.NewReader(`{"sync_remote_master":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.runtimeRestartHandler(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected accepted status, got %d: %s", rec.Code, rec.Body.String())
	}
	if !restarter.options.SyncRemoteMaster {
		t.Fatalf("expected sync_remote_master forwarded to runtime restarter")
	}
}

func TestRuntimeInfoEndpointReturnsStartedAtAndCommitHash(t *testing.T) {
	startedAt := time.Date(2026, time.March, 27, 14, 30, 0, 0, time.UTC)
	server := &Server{
		runtimeInfo: &stubRuntimeInfoProvider{
			info: RuntimeInfo{
				StartedAt:  startedAt,
				CommitHash: "0123456789abcdef0123456789abcdef01234567",
			},
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/control/runtime", nil)
	rec := httptest.NewRecorder()
	server.runtimeInfoHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected ok status, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload RuntimeInfo
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode runtime info failed: %v", err)
	}
	if !payload.StartedAt.Equal(startedAt) {
		t.Fatalf("expected started_at %s, got %s", startedAt, payload.StartedAt)
	}
	if payload.CommitHash != "0123456789abcdef0123456789abcdef01234567" {
		t.Fatalf("unexpected commit hash %q", payload.CommitHash)
	}
}

func TestRuntimeRestartEndpointRejectsInvalidJSON(t *testing.T) {
	server := &Server{
		runtime: &stubRuntimeRestarter{accepted: true},
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/control/runtime/restart", strings.NewReader(`{"sync_remote_master":`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.runtimeRestartHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request status, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRuntimeRestartEndpointRejectsConcurrentRestart(t *testing.T) {
	server := &Server{
		runtime: &stubRuntimeRestarter{accepted: false},
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/control/runtime/restart", nil)
	rec := httptest.NewRecorder()
	server.runtimeRestartHandler(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected conflict status, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProductDraftEndpointsGenerateAndPublish(t *testing.T) {
	control := controlapp.NewService()
	products := productapp.NewService()
	server := &Server{
		control:       control,
		agents:        agentapp.NewCatalog(control),
		products:      products,
		productDrafts: productapp.NewDraftService(products),
		logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	generateReq := httptest.NewRequest(http.MethodPost, "/api/control/products/generate", strings.NewReader(`{
		"name":"Travel Premium",
		"goal":"生成城市旅游攻略并输出路线、地铁和地图图层。",
		"core_capabilities":["city guide","itinerary","metro","food","map"],
		"expected_artifacts":["city_guide","itinerary","map_layers"]
	}`))
	generateRec := httptest.NewRecorder()
	server.productDraftGenerateHandler(generateRec, generateReq)
	if generateRec.Code != http.StatusCreated {
		t.Fatalf("expected generate 201, got %d: %s", generateRec.Code, generateRec.Body.String())
	}
	var draft productdomain.ProductDraft
	if err := json.NewDecoder(generateRec.Body).Decode(&draft); err != nil {
		t.Fatalf("decode draft failed: %v", err)
	}
	if draft.DraftID == "" || draft.Product.ID != "travel-premium" {
		t.Fatalf("unexpected draft: %+v", draft)
	}

	publishReq := httptest.NewRequest(http.MethodPost, "/api/control/products/drafts/"+draft.DraftID+"/publish", nil)
	publishRec := httptest.NewRecorder()
	server.productDraftItemHandler(publishRec, publishReq)
	if publishRec.Code != http.StatusOK {
		t.Fatalf("expected publish 200, got %d: %s", publishRec.Code, publishRec.Body.String())
	}
	if _, ok := products.ResolveProduct("travel-premium"); !ok {
		t.Fatalf("expected published product to be saved")
	}
	if agent, ok := control.ResolveAgent("travel-premium-master"); !ok || !agent.Enabled {
		t.Fatalf("expected published master agent, got %+v, %v", agent, ok)
	}
	if agent, ok := control.ResolveAgent("travel-premium-city-guide"); !ok || !agent.Enabled {
		t.Fatalf("expected published worker agent, got %+v, %v", agent, ok)
	}
	storedDraft, ok := server.productDrafts.GetDraft(draft.DraftID)
	if !ok || storedDraft.ReviewStatus != productdomain.ReviewStatusPublished {
		t.Fatalf("expected draft to be marked published, got %+v", storedDraft)
	}
}

func TestProductItemHandlerRoutesDraftCollection(t *testing.T) {
	products := productapp.NewService()
	server := &Server{
		products:      products,
		productDrafts: productapp.NewDraftService(products),
		logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	_, err := server.productDrafts.GenerateDraft(productdomain.ProductDraftGenerateInput{
		Name:             "Travel Premium",
		Goal:             "生成城市旅游攻略并输出地图图层。",
		CoreCapabilities: []string{"city guide", "map"},
	})
	if err != nil {
		t.Fatalf("generate draft failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/control/products/drafts", nil)
	rec := httptest.NewRecorder()
	server.productItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	var payload struct {
		Items []productdomain.ProductDraft `json:"items"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode drafts failed: %v", err)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected one draft, got %+v", payload.Items)
	}
}

func TestProductItemHandlerRoutesMatrixGenerate(t *testing.T) {
	products := productapp.NewService()
	server := &Server{
		products:      products,
		productDrafts: productapp.NewDraftService(products),
		logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/control/products/travel/matrix/generate", strings.NewReader(`{
		"name":"Travel Dining",
		"goal":"补充城市美食和夜间路线能力。",
		"core_capabilities":["food","night route"]
	}`))
	rec := httptest.NewRecorder()
	server.productItemHandler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}
	var draft productdomain.ProductDraft
	if err := json.NewDecoder(rec.Body).Decode(&draft); err != nil {
		t.Fatalf("decode draft failed: %v", err)
	}
	if draft.Mode != productdomain.GenerationModeExpand {
		t.Fatalf("expected expand draft, got %+v", draft)
	}
	if draft.Product.ID != "travel" {
		t.Fatalf("expected draft for travel product, got %+v", draft.Product)
	}
}

func TestNewServerAssignsProductDraftService(t *testing.T) {
	products := productapp.NewService()
	drafts := productapp.NewDraftService(products)

	server := NewServer(
		"127.0.0.1:0",
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		drafts,
		nil,
		AgentMemoryOptions{},
		WebSecurityOptions{},
		nil,
		nil,
		products,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	if server.productDrafts == nil {
		t.Fatalf("expected product draft service to be assigned")
	}
}

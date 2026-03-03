package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	controlapp "alter0/internal/control/application"
	controldomain "alter0/internal/control/domain"
)

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

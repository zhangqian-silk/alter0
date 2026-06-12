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

	controlapp "alter0/internal/control/application"
	controldomain "alter0/internal/control/domain"
	schedulerapp "alter0/internal/scheduler/application"
	schedulerdomain "alter0/internal/scheduler/domain"
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

	putReq := httptest.NewRequest(http.MethodPut, "/api/control/capabilities/skill/sample-skill", strings.NewReader(`{"name":"Sample Skill","version":"v1.0.0","scope":"global"}`))
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
	if len(listResp.Items) != 1 || listResp.Items[0].ID != "sample-skill" {
		t.Fatalf("unexpected capability list: %+v", listResp.Items)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/control/capabilities/skill/sample-skill", nil)
	deleteRec := httptest.NewRecorder()
	server.capabilityItemHandler(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("expected capability delete 200, got %d", deleteRec.Code)
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

func TestCronJobEndpointProtectsBuiltinJobsAndAllowsDisable(t *testing.T) {
	scheduler := schedulerapp.NewManager(nil, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := scheduler.RegisterBuiltinJobs([]schedulerdomain.Job{
		{
			ID:             "system-memory-maintenance",
			Name:           "Memory Maintenance",
			Enabled:        true,
			Timezone:       "Asia/Shanghai",
			ScheduleMode:   schedulerdomain.ScheduleModeDaily,
			CronExpression: "10 5 * * *",
			TaskConfig: schedulerdomain.TaskConfig{
				Input: "Run system memory maintenance.",
			},
		},
	}); err != nil {
		t.Fatalf("register builtin job failed: %v", err)
	}
	server := &Server{
		scheduler: scheduler,
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/control/cron/jobs", nil)
	listRec := httptest.NewRecorder()
	server.cronJobListHandler(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	var listResp struct {
		Items []cronJobResponse `json:"items"`
	}
	if err := json.NewDecoder(listRec.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list response failed: %v", err)
	}
	if len(listResp.Items) != 1 || !listResp.Items[0].Builtin {
		t.Fatalf("expected builtin cron job in list, got %+v", listResp.Items)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/control/cron/jobs/system-memory-maintenance", nil)
	deleteRec := httptest.NewRecorder()
	server.cronJobItemHandler(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusConflict {
		t.Fatalf("expected builtin delete 409, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}

	disableReq := httptest.NewRequest(http.MethodPut, "/api/control/cron/jobs/system-memory-maintenance", strings.NewReader(`{"enabled":false}`))
	disableRec := httptest.NewRecorder()
	server.cronJobItemHandler(disableRec, disableReq)
	if disableRec.Code != http.StatusOK {
		t.Fatalf("expected builtin disable 200, got %d: %s", disableRec.Code, disableRec.Body.String())
	}
	var disabled cronJobResponse
	if err := json.NewDecoder(disableRec.Body).Decode(&disabled); err != nil {
		t.Fatalf("decode disable response failed: %v", err)
	}
	if !disabled.Builtin || disabled.Enabled {
		t.Fatalf("expected disabled builtin response, got %+v", disabled)
	}
}

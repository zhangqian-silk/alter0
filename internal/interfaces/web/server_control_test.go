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
	status   RuntimeRestartStatus
}

type stubRuntimeInfoProvider struct {
	info RuntimeInfo
}

func (s *stubRuntimeRestarter) RequestRestart(options RuntimeRestartOptions) (bool, error) {
	s.called++
	s.options = options
	return s.accepted, s.err
}

func (s *stubRuntimeRestarter) GetRestartStatus() RuntimeRestartStatus {
	if s == nil || s.status.Status == "" {
		return RuntimeRestartStatus{Status: "idle"}
	}
	return s.status
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

	req := httptest.NewRequest(http.MethodPost, "/api/control/runtime/restart", strings.NewReader(`{"sync_remote_master":true,"confirm_discard_tracked_changes":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.runtimeRestartHandler(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected accepted status, got %d: %s", rec.Code, rec.Body.String())
	}
	if !restarter.options.SyncRemoteMaster {
		t.Fatalf("expected sync_remote_master forwarded to runtime restarter")
	}
	if !restarter.options.ConfirmDiscardTrackedChanges {
		t.Fatalf("expected confirm_discard_tracked_changes forwarded to runtime restarter")
	}
}

func TestRuntimeRestartEndpointReturnsDiscardConfirmationCode(t *testing.T) {
	restarter := &stubRuntimeRestarter{
		err: NewRuntimeRestartError(
			RuntimeRestartDiscardConfirmationRequired,
			"sync remote master requires discard confirmation because tracked working tree changes exist: M README.md",
		),
	}
	server := &Server{
		runtime: restarter,
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/control/runtime/restart", strings.NewReader(`{"sync_remote_master":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.runtimeRestartHandler(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected conflict status, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Code  string `json:"code"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode restart error failed: %v", err)
	}
	if payload.Code != RuntimeRestartDiscardConfirmationRequired {
		t.Fatalf("expected discard confirmation code, got %q", payload.Code)
	}
	if !strings.Contains(payload.Error, "tracked working tree changes") {
		t.Fatalf("expected detailed tracked changes error, got %q", payload.Error)
	}
}

func TestRuntimeRestartEndpointReturnsLastStatus(t *testing.T) {
	updatedAt := time.Date(2026, time.June, 23, 5, 20, 0, 0, time.UTC)
	restarter := &stubRuntimeRestarter{
		status: RuntimeRestartStatus{
			Status:                       "failed",
			Error:                        "candidate runtime exited before ready: flag provided but not defined",
			SyncRemoteMaster:             true,
			ConfirmDiscardTrackedChanges: true,
			UpdatedAt:                    updatedAt,
		},
	}
	server := &Server{
		runtime: restarter,
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/control/runtime/restart", nil)
	rec := httptest.NewRecorder()
	server.runtimeRestartHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected ok status, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload RuntimeRestartStatus
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode restart status failed: %v", err)
	}
	if payload.Status != "failed" {
		t.Fatalf("expected failed status, got %q", payload.Status)
	}
	if !payload.SyncRemoteMaster || !payload.ConfirmDiscardTrackedChanges {
		t.Fatalf("expected restart options in status: %+v", payload)
	}
	if !strings.Contains(payload.Error, "flag provided") {
		t.Fatalf("unexpected restart error %q", payload.Error)
	}
	if !payload.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("unexpected updated_at %s", payload.UpdatedAt)
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

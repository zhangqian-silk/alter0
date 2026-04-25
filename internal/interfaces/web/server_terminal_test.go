package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	controlapp "alter0/internal/control/application"
	controldomain "alter0/internal/control/domain"
	terminalapp "alter0/internal/terminal/application"
	terminaldomain "alter0/internal/terminal/domain"
)

type stubWebTerminalService struct {
	createReq   terminalapp.CreateRequest
	createResp  terminaldomain.Session
	createErr   error
	recoverReq  terminalapp.RecoverRequest
	recoverResp terminaldomain.Session
	recoverErr  error
	listResp    []terminaldomain.Session
	getResp     terminaldomain.Session
	getOK       bool
	inputResp   terminaldomain.Session
	inputErr    error
	deleteResp  terminaldomain.Session
	deleteErr   error
	turnsResp   []terminalapp.TurnSummary
	turnsErr    error
	stepResp    terminalapp.StepDetail
	stepErr     error
	entryPage   terminalapp.EntryPage
	entryErr    error
	lastOwnerID string
	lastID      string
	lastInput   string
	inputReq    terminalapp.InputRequest
}

func (s *stubWebTerminalService) Create(req terminalapp.CreateRequest) (terminaldomain.Session, error) {
	s.createReq = req
	return s.createResp, s.createErr
}

func (s *stubWebTerminalService) Recover(req terminalapp.RecoverRequest) (terminaldomain.Session, error) {
	s.recoverReq = req
	return s.recoverResp, s.recoverErr
}

func (s *stubWebTerminalService) List(ownerID string) []terminaldomain.Session {
	s.lastOwnerID = ownerID
	return append([]terminaldomain.Session{}, s.listResp...)
}

func (s *stubWebTerminalService) Get(ownerID string, sessionID string) (terminaldomain.Session, bool) {
	s.lastOwnerID = ownerID
	s.lastID = sessionID
	return s.getResp, s.getOK
}

func (s *stubWebTerminalService) ListTurns(ownerID string, sessionID string) ([]terminalapp.TurnSummary, error) {
	s.lastOwnerID = ownerID
	s.lastID = sessionID
	return append([]terminalapp.TurnSummary{}, s.turnsResp...), s.turnsErr
}

func (s *stubWebTerminalService) GetStepDetail(ownerID string, sessionID string, turnID string, stepID string) (terminalapp.StepDetail, error) {
	s.lastOwnerID = ownerID
	s.lastID = sessionID + ":" + turnID + ":" + stepID
	return s.stepResp, s.stepErr
}

func (s *stubWebTerminalService) ListEntries(ownerID string, sessionID string, _ int, _ int) (terminalapp.EntryPage, error) {
	s.lastOwnerID = ownerID
	s.lastID = sessionID
	return s.entryPage, s.entryErr
}

func (s *stubWebTerminalService) Input(ownerID string, sessionID string, input string) (terminaldomain.Session, error) {
	s.lastOwnerID = ownerID
	s.lastID = sessionID
	s.lastInput = input
	return s.inputResp, s.inputErr
}

func (s *stubWebTerminalService) InputWithAttachments(req terminalapp.InputRequest) (terminaldomain.Session, error) {
	s.lastOwnerID = req.OwnerID
	s.lastID = req.SessionID
	s.lastInput = req.Input
	s.inputReq = req
	return s.inputResp, s.inputErr
}
func (s *stubWebTerminalService) Delete(ownerID string, sessionID string) (terminaldomain.Session, error) {
	s.lastOwnerID = ownerID
	s.lastID = sessionID
	return s.deleteResp, s.deleteErr
}

func TestTerminalSessionCollectionHandlerCreatesSession(t *testing.T) {
	service := &stubWebTerminalService{
		createResp: terminaldomain.Session{
			ID:           "terminal-1",
			OwnerID:      sharedTerminalClientID,
			Title:        "terminal-1",
			Status:       terminaldomain.SessionStatusReady,
			CreatedAt:    time.Now().UTC(),
			LastOutputAt: time.Now().UTC(),
			UpdatedAt:    time.Now().UTC(),
		},
	}
	server := &Server{terminals: service}

	req := httptest.NewRequest(http.MethodPost, "/api/terminal/sessions", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()

	server.terminalSessionCollectionHandler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}
	if service.createReq.OwnerID != sharedTerminalClientID {
		t.Fatalf("expected shared owner, got %q", service.createReq.OwnerID)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	session, ok := payload["session"].(map[string]any)
	if !ok {
		t.Fatalf("expected session payload, got %v", payload)
	}
	if session["id"] != "terminal-1" {
		t.Fatalf("expected terminal id terminal-1, got %v", session["id"])
	}
	if _, ok := session["last_output_at"].(string); !ok {
		t.Fatalf("expected last_output_at in session payload, got %v", session["last_output_at"])
	}
}

func TestTerminalSessionItemHandlerWritesInput(t *testing.T) {
	service := &stubWebTerminalService{
		inputResp: terminaldomain.Session{
			ID:        "terminal-2",
			OwnerID:   sharedTerminalClientID,
			Title:     "terminal-2",
			Status:    terminaldomain.SessionStatusBusy,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
	}
	server := &Server{terminals: service}

	req := httptest.NewRequest(http.MethodPost, "/api/terminal/sessions/terminal-2/input", bytes.NewBufferString(`{"input":"pwd"}`))
	rec := httptest.NewRecorder()

	server.terminalSessionItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if service.lastOwnerID != sharedTerminalClientID {
		t.Fatalf("expected shared owner, got %q", service.lastOwnerID)
	}
	if service.lastID != "terminal-2" {
		t.Fatalf("expected session terminal-2, got %q", service.lastID)
	}
	if service.lastInput != "pwd" {
		t.Fatalf("expected input pwd, got %q", service.lastInput)
	}
}

func TestTerminalSessionItemHandlerWritesImageAttachments(t *testing.T) {
	service := &stubWebTerminalService{
		inputResp: terminaldomain.Session{
			ID:        "terminal-2",
			OwnerID:   sharedTerminalClientID,
			Title:     "terminal-2",
			Status:    terminaldomain.SessionStatusBusy,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
	}
	server := &Server{terminals: service}

	req := httptest.NewRequest(http.MethodPost, "/api/terminal/sessions/terminal-2/input", bytes.NewBufferString(`{"attachments":[{"name":"diagram.png","content_type":"image/png","data_url":"data:image/png;base64,ZmFrZQ=="}]}`))
	rec := httptest.NewRecorder()

	server.terminalSessionItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if service.inputReq.Input != "Attached image." {
		t.Fatalf("expected default image input, got %q", service.inputReq.Input)
	}
	if len(service.inputReq.Attachments) != 1 {
		t.Fatalf("expected image attachments, got %+v", service.inputReq.Attachments)
	}
	if service.inputReq.Attachments[0].DataURL != "data:image/png;base64,ZmFrZQ==" {
		t.Fatalf("expected attachment data url, got %+v", service.inputReq.Attachments[0])
	}
}

func TestTerminalSessionItemHandlerPassesSelectedSkills(t *testing.T) {
	service := &stubWebTerminalService{
		inputResp: terminaldomain.Session{
			ID:        "terminal-2",
			OwnerID:   sharedTerminalClientID,
			Title:     "terminal-2",
			Status:    terminaldomain.SessionStatusBusy,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
	}
	control := controlapp.NewService()
	if err := control.UpsertCapability(controldomain.Capability{
		ID:      "summary",
		Name:    "Summary",
		Type:    controldomain.CapabilityTypeSkill,
		Enabled: true,
		Scope:   controldomain.CapabilityScopeGlobal,
		Version: controldomain.DefaultCapabilityVersion,
		Metadata: map[string]string{
			"skill.description": "Summarize terminal work.",
			"skill.guide":       "Use concise structured summaries.",
			"skill.file_path":   ".alter0/skills/summary/SKILL.md",
		},
	}); err != nil {
		t.Fatalf("upsert skill failed: %v", err)
	}
	if err := control.UpsertCapability(controldomain.Capability{
		ID:      "private",
		Name:    "Private",
		Type:    controldomain.CapabilityTypeSkill,
		Enabled: true,
		Scope:   controldomain.CapabilityScopeGlobal,
		Version: controldomain.DefaultCapabilityVersion,
		Metadata: map[string]string{
			"alter0.skill.visibility": "agent-private",
		},
	}); err != nil {
		t.Fatalf("upsert private skill failed: %v", err)
	}
	server := &Server{terminals: service, control: control}

	req := httptest.NewRequest(http.MethodPost, "/api/terminal/sessions/terminal-2/input", bytes.NewBufferString(`{"input":"summarize","skill_ids":["summary","private","missing"]}`))
	rec := httptest.NewRecorder()

	server.terminalSessionItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if service.inputReq.SkillContext == nil {
		t.Fatalf("expected skill context")
	}
	if len(service.inputReq.SkillContext.Skills) != 1 {
		t.Fatalf("expected only public selected skill, got %+v", service.inputReq.SkillContext.Skills)
	}
	if service.inputReq.SkillContext.Skills[0].ID != "summary" {
		t.Fatalf("expected summary skill, got %+v", service.inputReq.SkillContext.Skills[0])
	}
	if service.inputReq.SkillContext.Skills[0].Guide != "Use concise structured summaries." {
		t.Fatalf("expected skill guide, got %+v", service.inputReq.SkillContext.Skills[0])
	}
}

func TestTerminalSessionRecoverHandlerRestoresStoredSession(t *testing.T) {
	service := &stubWebTerminalService{
		recoverResp: terminaldomain.Session{
			ID:                "terminal-recover",
			OwnerID:           sharedTerminalClientID,
			Title:             "Recovered",
			TerminalSessionID: "thread-recover",
			Status:            terminaldomain.SessionStatusReady,
			CreatedAt:         time.Date(2026, 3, 19, 10, 0, 0, 0, time.UTC),
			UpdatedAt:         time.Date(2026, 3, 19, 10, 5, 0, 0, time.UTC),
		},
	}
	server := &Server{terminals: service}

	req := httptest.NewRequest(http.MethodPost, "/api/terminal/sessions/recover", bytes.NewBufferString(`{"id":"terminal-recover","terminal_session_id":"thread-recover","title":"Recovered","created_at":"2026-03-19T10:00:00Z","updated_at":"2026-03-19T10:05:00Z"}`))
	rec := httptest.NewRecorder()

	server.terminalSessionRecoverHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if service.recoverReq.OwnerID != sharedTerminalClientID {
		t.Fatalf("expected shared owner, got %q", service.recoverReq.OwnerID)
	}
	if service.recoverReq.SessionID != "terminal-recover" {
		t.Fatalf("expected recover session id, got %q", service.recoverReq.SessionID)
	}
	if service.recoverReq.TerminalSessionID != "thread-recover" {
		t.Fatalf("expected recover thread id, got %q", service.recoverReq.TerminalSessionID)
	}
}

func TestTerminalSessionItemHandlerRejectsRemovedCloseRoute(t *testing.T) {
	service := &stubWebTerminalService{}
	server := &Server{terminals: service}

	req := httptest.NewRequest(http.MethodPost, "/api/terminal/sessions/terminal-3/close", nil)
	rec := httptest.NewRecorder()

	server.terminalSessionItemHandler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
	if service.lastID != "" {
		t.Fatalf("expected close route to bypass terminal service, got %q", service.lastID)
	}
}

func TestTerminalSessionItemHandlerDeletesSession(t *testing.T) {
	service := &stubWebTerminalService{
		deleteResp: terminaldomain.Session{
			ID:         "terminal-4",
			OwnerID:    sharedTerminalClientID,
			Title:      "terminal-4",
			Status:     terminaldomain.SessionStatusExited,
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
			FinishedAt: time.Now().UTC(),
		},
	}
	server := &Server{terminals: service}

	req := httptest.NewRequest(http.MethodDelete, "/api/terminal/sessions/terminal-4", nil)
	rec := httptest.NewRecorder()

	server.terminalSessionItemHandler(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rec.Code)
	}
	if service.lastOwnerID != sharedTerminalClientID {
		t.Fatalf("expected shared owner, got %q", service.lastOwnerID)
	}
	if service.lastID != "terminal-4" {
		t.Fatalf("expected session terminal-4, got %q", service.lastID)
	}
	if body := strings.TrimSpace(rec.Body.String()); body != "" {
		t.Fatalf("expected empty response body, got %q", body)
	}
}

func TestTerminalSessionItemHandlerReturnsTurnsInSessionDetail(t *testing.T) {
	service := &stubWebTerminalService{
		getResp: terminaldomain.Session{
			ID:        "terminal-4",
			OwnerID:   sharedTerminalClientID,
			Title:     "terminal-4",
			Status:    terminaldomain.SessionStatusReady,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
		getOK: true,
		turnsResp: []terminalapp.TurnSummary{{
			ID:          "turn-1",
			Prompt:      "pwd",
			Status:      "completed",
			FinalOutput: "/workspace/alter0",
		}},
	}
	server := &Server{terminals: service}

	req := httptest.NewRequest(http.MethodGet, "/api/terminal/sessions/terminal-4", nil)
	rec := httptest.NewRecorder()

	server.terminalSessionItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	session, ok := payload["session"].(map[string]any)
	if !ok {
		t.Fatalf("expected session payload, got %v", payload)
	}
	turns, ok := session["turns"].([]any)
	if !ok || len(turns) != 1 {
		t.Fatalf("expected turns payload, got %v", session["turns"])
	}
}

func TestTerminalSessionItemHandlerReturnsStepDetail(t *testing.T) {
	service := &stubWebTerminalService{
		stepResp: terminalapp.StepDetail{
			TurnID: "turn-1",
			Step: terminalapp.StepSummary{
				ID:     "step-1",
				Type:   "command",
				Title:  "pwd",
				Status: "completed",
			},
			Blocks: []terminalapp.StepDetailBlock{{
				Type:    "terminal",
				Title:   "Shell",
				Content: "pwd\n\n/workspace/alter0",
				Status:  "completed",
			}},
			Searchable: true,
		},
	}
	server := &Server{terminals: service}

	req := httptest.NewRequest(http.MethodGet, "/api/terminal/sessions/terminal-4/turns/turn-1/steps/step-1", nil)
	rec := httptest.NewRecorder()

	server.terminalSessionItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if service.lastID != "terminal-4:turn-1:step-1" {
		t.Fatalf("expected step lookup path, got %q", service.lastID)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	step, ok := payload["step"].(map[string]any)
	if !ok {
		t.Fatalf("expected step payload, got %v", payload)
	}
	if step["turn_id"] != "turn-1" {
		t.Fatalf("expected turn_id turn-1, got %v", step["turn_id"])
	}
}

func TestTerminalViewPreservesInputDraftAcrossPaint(t *testing.T) {
	script := readWorkspaceFile(t, "frontend/src/features/shell/components/ReactManagedTerminalRouteBody.tsx") +
		readWorkspaceFile(t, "frontend/src/features/shell/components/RuntimeComposer.tsx")
	markers := []string{
		"window.localStorage.getItem(`terminal:${activeSessionID}`) || \"\"",
		"window.localStorage.setItem(`terminal:${activeSessionID}`, inputValue);",
		"window.localStorage.removeItem(`terminal:${sessionID}`);",
		"window.localStorage.removeItem(`terminal:${session.id}`);",
		"const timer = window.setTimeout(() => {",
		`runtimeKind: "terminal"`,
		`data-runtime-composer-kind={runtimeKind}`,
		`data-runtime-composer-input={runtimeKind}`,
		`data-runtime-composer-submit={runtimeKind}`,
		`data-composer-form={composerAlias}`,
		`data-composer-input={composerAlias}`,
		`data-composer-submit={composerAlias}`,
	}
	for _, marker := range markers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected terminal focus marker %q", marker)
		}
	}
}

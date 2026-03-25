package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	closeResp   terminaldomain.Session
	closeErr    error
	turnsResp   []terminalapp.TurnSummary
	turnsErr    error
	stepResp    terminalapp.StepDetail
	stepErr     error
	entryPage   terminalapp.EntryPage
	entryErr    error
	lastOwnerID string
	lastID      string
	lastInput   string
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

func (s *stubWebTerminalService) Close(ownerID string, sessionID string) (terminaldomain.Session, error) {
	s.lastOwnerID = ownerID
	s.lastID = sessionID
	return s.closeResp, s.closeErr
}

func TestTerminalSessionCollectionHandlerCreatesSession(t *testing.T) {
	service := &stubWebTerminalService{
		createResp: terminaldomain.Session{
			ID:           "terminal-1",
			OwnerID:      "client-a",
			Title:        "terminal-1",
			Status:       terminaldomain.SessionStatusRunning,
			CreatedAt:    time.Now().UTC(),
			LastOutputAt: time.Now().UTC(),
			UpdatedAt:    time.Now().UTC(),
		},
	}
	server := &Server{terminals: service}

	req := httptest.NewRequest(http.MethodPost, "/api/terminal/sessions", bytes.NewBufferString(`{}`))
	req.Header.Set(terminalClientIDHeader, "client-a")
	rec := httptest.NewRecorder()

	server.terminalSessionCollectionHandler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}
	if service.createReq.OwnerID != "client-a" {
		t.Fatalf("expected owner client-a, got %q", service.createReq.OwnerID)
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
			OwnerID:   "client-b",
			Title:     "terminal-2",
			Status:    terminaldomain.SessionStatusRunning,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
	}
	server := &Server{terminals: service}

	req := httptest.NewRequest(http.MethodPost, "/api/terminal/sessions/terminal-2/input", bytes.NewBufferString(`{"input":"pwd"}`))
	req.Header.Set(terminalClientIDHeader, "client-b")
	rec := httptest.NewRecorder()

	server.terminalSessionItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if service.lastOwnerID != "client-b" {
		t.Fatalf("expected owner client-b, got %q", service.lastOwnerID)
	}
	if service.lastID != "terminal-2" {
		t.Fatalf("expected session terminal-2, got %q", service.lastID)
	}
	if service.lastInput != "pwd" {
		t.Fatalf("expected input pwd, got %q", service.lastInput)
	}
}

func TestTerminalSessionRecoverHandlerRestoresStoredSession(t *testing.T) {
	service := &stubWebTerminalService{
		recoverResp: terminaldomain.Session{
			ID:                "terminal-recover",
			OwnerID:           "client-recover",
			Title:             "Recovered",
			TerminalSessionID: "thread-recover",
			Status:            terminaldomain.SessionStatusRunning,
			CreatedAt:         time.Date(2026, 3, 19, 10, 0, 0, 0, time.UTC),
			UpdatedAt:         time.Date(2026, 3, 19, 10, 5, 0, 0, time.UTC),
		},
	}
	server := &Server{terminals: service}

	req := httptest.NewRequest(http.MethodPost, "/api/terminal/sessions/recover", bytes.NewBufferString(`{"id":"terminal-recover","terminal_session_id":"thread-recover","title":"Recovered","created_at":"2026-03-19T10:00:00Z","updated_at":"2026-03-19T10:05:00Z"}`))
	req.Header.Set(terminalClientIDHeader, "client-recover")
	rec := httptest.NewRecorder()

	server.terminalSessionRecoverHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if service.recoverReq.OwnerID != "client-recover" {
		t.Fatalf("expected owner client-recover, got %q", service.recoverReq.OwnerID)
	}
	if service.recoverReq.SessionID != "terminal-recover" {
		t.Fatalf("expected recover session id, got %q", service.recoverReq.SessionID)
	}
	if service.recoverReq.TerminalSessionID != "thread-recover" {
		t.Fatalf("expected recover thread id, got %q", service.recoverReq.TerminalSessionID)
	}
}

func TestTerminalSessionItemHandlerClosesSession(t *testing.T) {
	service := &stubWebTerminalService{
		closeResp: terminaldomain.Session{
			ID:         "terminal-3",
			OwnerID:    "client-c",
			Title:      "terminal-3",
			Status:     terminaldomain.SessionStatusExited,
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
			FinishedAt: time.Now().UTC(),
		},
	}
	server := &Server{terminals: service}

	req := httptest.NewRequest(http.MethodDelete, "/api/terminal/sessions/terminal-3", nil)
	req.Header.Set(terminalClientIDHeader, "client-c")
	rec := httptest.NewRecorder()

	server.terminalSessionItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if service.lastOwnerID != "client-c" {
		t.Fatalf("expected owner client-c, got %q", service.lastOwnerID)
	}
	if service.lastID != "terminal-3" {
		t.Fatalf("expected session terminal-3, got %q", service.lastID)
	}
}

func TestTerminalSessionItemHandlerReturnsTurnsInSessionDetail(t *testing.T) {
	service := &stubWebTerminalService{
		getResp: terminaldomain.Session{
			ID:        "terminal-4",
			OwnerID:   "client-d",
			Title:     "terminal-4",
			Status:    terminaldomain.SessionStatusRunning,
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
	req.Header.Set(terminalClientIDHeader, "client-d")
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
	req.Header.Set(terminalClientIDHeader, "client-d")
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
	script := readEmbeddedAsset(t, "static/assets/chat.js")
	markers := []string{
		"drafts: {}",
		"focusedInputSessionID: \"\"",
		"composingInputSessionID: \"\"",
		"const terminalComposer = createReusableComposer();",
		"const bindTerminalComposer = (session) => {",
		"terminalComposer.bind(inputNode, formNode, {",
		"draftStorage: \"local\",",
		"draftKey: () => `terminal:${normalizeText(session?.id || \"default\")}`,",
		"const isTerminalInputComposing = (sessionID) => {",
		"const requestTerminalPaint = (options = {}) => {",
		"const rememberTerminalInputFocus = (sessionID, inputNode = null) => {",
		"const previousWorkspace = container.querySelector(\"[data-terminal-workspace]\");",
		"writeTerminalDraft(previousSessionID, previousInput.value);",
		"const previousInput = container.querySelector(\"[data-terminal-input]\");",
		"bindTerminalComposer(active);",
		"rememberTerminalInputComposition(session.id);",
		"flushDeferredTerminalPaint();",
		"nextInput.focus({ preventScroll: true });",
	}
	for _, marker := range markers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected terminal focus marker %q", marker)
		}
	}
}

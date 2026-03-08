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
	listResp    []terminaldomain.Session
	getResp     terminaldomain.Session
	getOK       bool
	inputResp   terminaldomain.Session
	inputErr    error
	closeResp   terminaldomain.Session
	closeErr    error
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

func (s *stubWebTerminalService) List(ownerID string) []terminaldomain.Session {
	s.lastOwnerID = ownerID
	return append([]terminaldomain.Session{}, s.listResp...)
}

func (s *stubWebTerminalService) Get(ownerID string, sessionID string) (terminaldomain.Session, bool) {
	s.lastOwnerID = ownerID
	s.lastID = sessionID
	return s.getResp, s.getOK
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

func (s *stubWebTerminalService) MaxSessions() int {
	return 5
}

func TestTerminalSessionCollectionHandlerCreatesSession(t *testing.T) {
	service := &stubWebTerminalService{
		createResp: terminaldomain.Session{
			ID:        "terminal-1",
			OwnerID:   "client-a",
			Title:     "terminal-1",
			Status:    terminaldomain.SessionStatusRunning,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
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

package web

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	createReq      terminalapp.CreateRequest
	createResp     terminaldomain.Session
	createErr      error
	recoverReq     terminalapp.RecoverRequest
	recoverResp    terminaldomain.Session
	recoverErr     error
	listResp       []terminaldomain.Session
	listByOwner    map[string][]terminaldomain.Session
	getResp        terminaldomain.Session
	getOK          bool
	inputResp      terminaldomain.Session
	inputErr       error
	pinResp        terminaldomain.Session
	pinErr         error
	deleteResp     terminaldomain.Session
	deleteErr      error
	deleteIDs      []string
	deleteOwnerIDs []string
	turnsResp      []terminalapp.TurnSummary
	turnsErr       error
	eventResp      terminalapp.RuntimeTraceEventDetail
	stepErr        error
	entryPage      terminalapp.EntryPage
	entryErr       error
	lastOwnerID    string
	lastID         string
	lastInput      string
	lastPinned     bool
	inputReq       terminalapp.InputRequest
	updateHook     terminalapp.SessionUpdateHook
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
	if s.listByOwner != nil {
		return append([]terminaldomain.Session{}, s.listByOwner[ownerID]...)
	}
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

func (s *stubWebTerminalService) GetRuntimeTraceEventDetail(ownerID string, sessionID string, turnID string, eventID string) (terminalapp.RuntimeTraceEventDetail, error) {
	s.lastOwnerID = ownerID
	s.lastID = sessionID + ":" + turnID + ":" + eventID
	return s.eventResp, s.stepErr
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

func (s *stubWebTerminalService) SetPinned(ownerID string, sessionID string, pinned bool) (terminaldomain.Session, error) {
	s.lastOwnerID = ownerID
	s.lastID = sessionID
	s.lastPinned = pinned
	return s.pinResp, s.pinErr
}

func (s *stubWebTerminalService) Delete(ownerID string, sessionID string) (terminaldomain.Session, error) {
	s.lastOwnerID = ownerID
	s.lastID = sessionID
	s.deleteIDs = append(s.deleteIDs, sessionID)
	s.deleteOwnerIDs = append(s.deleteOwnerIDs, ownerID+":"+sessionID)
	return s.deleteResp, s.deleteErr
}

func (s *stubWebTerminalService) SetSessionUpdateHook(hook terminalapp.SessionUpdateHook) {
	s.updateHook = hook
}

func TestTerminalSessionCollectionHandlerCreatesSession(t *testing.T) {
	service := &stubWebTerminalService{
		createResp: terminaldomain.Session{
			ID:           "terminal-1",
			OwnerID:      terminalSessionOwnerID,
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
	if service.createReq.OwnerID != terminalSessionOwnerID {
		t.Fatalf("expected terminal owner, got %q", service.createReq.OwnerID)
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
	if _, ok := session["activity_at"].(string); !ok {
		t.Fatalf("expected activity_at in session payload, got %v", session["activity_at"])
	}
	if revision, ok := session["revision"].(float64); !ok || revision <= 0 {
		t.Fatalf("expected positive revision in session payload, got %v", session["revision"])
	}
}

func TestTerminalSessionCollectionHandlerReturnsComparableSessionSummaries(t *testing.T) {
	service := &stubWebTerminalService{
		listResp: []terminaldomain.Session{
			{
				ID:           "terminal-older-active",
				OwnerID:      terminalSessionOwnerID,
				Title:        "Older active",
				Status:       terminaldomain.SessionStatusReady,
				CreatedAt:    time.Date(2026, 4, 21, 3, 30, 0, 0, time.UTC),
				LastOutputAt: time.Date(2026, 4, 23, 4, 30, 0, 0, time.UTC),
				UpdatedAt:    time.Date(2026, 4, 23, 4, 31, 0, 0, time.UTC),
			},
			{
				ID:        "terminal-new-idle",
				OwnerID:   terminalSessionOwnerID,
				Title:     "New idle",
				Status:    terminaldomain.SessionStatusReady,
				CreatedAt: time.Date(2026, 4, 23, 3, 30, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, 4, 23, 3, 30, 0, 0, time.UTC),
			},
		},
	}
	server := &Server{terminals: service}

	req := httptest.NewRequest(http.MethodGet, "/api/terminal/sessions", nil)
	rec := httptest.NewRecorder()

	server.terminalSessionCollectionHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var payload map[string][]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	items := payload["items"]
	if len(items) != 2 {
		t.Fatalf("expected two session summaries, got %v", payload)
	}
	for _, session := range items {
		if _, ok := session["activity_at"].(string); !ok {
			t.Fatalf("expected activity_at in session summary, got %v", session)
		}
		if revision, ok := session["revision"].(float64); !ok || revision <= 0 {
			t.Fatalf("expected positive revision in session summary, got %v", session)
		}
		if _, hasTurns := session["turns"]; hasTurns {
			t.Fatalf("expected collection summary without turns, got %v", session)
		}
	}
}

func TestTerminalSessionSummaryRevisionDistinguishesSubMillisecondUpdates(t *testing.T) {
	first := buildTerminalSessionSummary(terminaldomain.Session{
		ID:        "terminal-fast-1",
		OwnerID:   terminalSessionOwnerID,
		Title:     "Fast 1",
		Status:    terminaldomain.SessionStatusReady,
		CreatedAt: time.Date(2026, 4, 23, 4, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 23, 4, 30, 1, 123400, time.UTC),
	})
	second := buildTerminalSessionSummary(terminaldomain.Session{
		ID:        "terminal-fast-2",
		OwnerID:   terminalSessionOwnerID,
		Title:     "Fast 2",
		Status:    terminaldomain.SessionStatusReady,
		CreatedAt: time.Date(2026, 4, 23, 4, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 23, 4, 30, 1, 124400, time.UTC),
	})

	firstRevision := first.(map[string]any)["revision"]
	secondRevision := second.(map[string]any)["revision"]
	if firstRevision == secondRevision {
		t.Fatalf("expected sub-millisecond revisions to differ, got %v", firstRevision)
	}
}

func TestChatSessionUpdatesHandlerReturnsIncrementalOwnerEvents(t *testing.T) {
	server := &Server{sessionEvents: newSessionUpdateBroker(8)}
	server.publishTerminalSessionEvent(chatSessionOwnerID, "chat-1", "session.updated", terminaldomain.Session{
		ID:      "chat-1",
		OwnerID: chatSessionOwnerID,
		Title:   "Running chat",
		Status:  terminaldomain.SessionStatusBusy,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/chat/sessions/updates?since_event_id=0", nil)
	rec := httptest.NewRecorder()

	server.chatSessionUpdatesHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var payload struct {
		Cursor         int64                `json:"cursor"`
		ResyncRequired bool                 `json:"resync_required"`
		Events         []sessionUpdateEvent `json:"events"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.ResyncRequired {
		t.Fatalf("did not expect resync_required")
	}
	if payload.Cursor == 0 || len(payload.Events) != 1 {
		t.Fatalf("expected one incremental event and non-zero cursor, got %+v", payload)
	}
	event := payload.Events[0]
	if event.OwnerID != chatSessionOwnerID || event.SessionID != "chat-1" || event.EventType != "session.updated" {
		t.Fatalf("expected chat session.updated event, got %+v", event)
	}
	session, ok := event.Payload["session"].(map[string]any)
	if !ok || session["status"] != "busy" {
		t.Fatalf("expected busy session payload, got %+v", event.Payload)
	}
}

func TestChatSessionUpdatesHandlerPrunesKnownRuntimeTraceEvents(t *testing.T) {
	server := &Server{sessionEvents: newSessionUpdateBroker(8)}
	server.sessionUpdateBroker().publish(chatSessionOwnerID, "chat-1", "session.updated", map[string]any{
		"session": map[string]any{
			"id":     "chat-1",
			"status": "busy",
			"turns": []any{
				map[string]any{
					"id":     "turn-1",
					"prompt": "hello",
					"status": "running",
					"runtime_trace_events": []any{
						map[string]any{"id": "step-1", "seq": 1, "kind": "reasoning", "title": "known 1"},
						map[string]any{"id": "step-2", "seq": 2, "kind": "reasoning", "title": "known 2"},
						map[string]any{"id": "step-3", "seq": 3, "kind": "reasoning", "title": "missing 3"},
					},
				},
			},
		},
	})
	body := strings.NewReader(`{
		"since_event_id": 0,
		"limit": 50,
		"byte_limit": 65536,
		"sessions": [{
			"id": "chat-1",
			"turns": [{
				"id": "turn-1",
				"event_seq_ranges": [[1, 2]]
			}]
		}]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/chat/sessions/updates", body)
	rec := httptest.NewRecorder()

	server.chatSessionUpdatesHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var payload struct {
		Events []sessionUpdateEvent `json:"events"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(payload.Events) != 1 {
		t.Fatalf("expected one event, got %+v", payload.Events)
	}
	session, ok := payload.Events[0].Payload["session"].(map[string]any)
	if !ok {
		t.Fatalf("expected session payload, got %+v", payload.Events[0].Payload)
	}
	turns, ok := session["turns"].([]any)
	if !ok || len(turns) != 1 {
		t.Fatalf("expected one turn payload, got %+v", session["turns"])
	}
	turn, ok := turns[0].(map[string]any)
	if !ok {
		t.Fatalf("expected turn payload, got %+v", turns[0])
	}
	events, ok := turn["runtime_trace_events"].([]any)
	if !ok || len(events) != 1 {
		t.Fatalf("expected one missing runtime trace event, got %+v", turn["runtime_trace_events"])
	}
	event, ok := events[0].(map[string]any)
	if !ok || event["id"] != "step-3" {
		t.Fatalf("expected missing runtime trace event step-3, got %+v", events[0])
	}
	if turn["runtime_trace_events_partial"] != true {
		t.Fatalf("expected partial runtime trace event marker, got %+v", turn["runtime_trace_events_partial"])
	}
}

func TestTerminalSessionUpdateHookReturnsBoundedSessionDetailTurns(t *testing.T) {
	service := &stubWebTerminalService{
		turnsResp: []terminalapp.TurnSummary{{
			ID:          "turn-old",
			Prompt:      "old",
			Status:      "completed",
			FinalOutput: "old done",
		}, {
			ID:          "turn-2",
			Prompt:      "hello",
			Status:      "completed",
			FinalOutput: "done",
		}},
	}
	server := &Server{terminals: service, sessionEvents: newSessionUpdateBroker(8)}
	server.registerTerminalSessionUpdateHook()
	if service.updateHook == nil {
		t.Fatalf("expected session update hook to be registered")
	}
	service.updateHook(chatSessionOwnerID, "chat-1", terminaldomain.Session{
		ID:      "chat-1",
		OwnerID: chatSessionOwnerID,
		Title:   "Running chat",
		Status:  terminaldomain.SessionStatusBusy,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/chat/sessions/updates?since_event_id=0", nil)
	rec := httptest.NewRecorder()

	server.chatSessionUpdatesHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var payload struct {
		Events []sessionUpdateEvent `json:"events"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(payload.Events) != 1 {
		t.Fatalf("expected one event, got %+v", payload.Events)
	}
	raw, _ := json.Marshal(payload.Events[0].Payload)
	body := string(raw)
	if !strings.Contains(body, `"turns"`) || !strings.Contains(body, `"final_output":"done"`) {
		t.Fatalf("expected session detail turns in update payload, got %q", body)
	}
	if strings.Contains(body, `"final_output":"old done"`) {
		t.Fatalf("expected update session detail to include only the latest turn page, got %q", body)
	}
}

func TestChatSessionUpdatesHandlerRequestsResyncWhenCursorCannotResume(t *testing.T) {
	server := &Server{sessionEvents: newSessionUpdateBroker(8)}

	req := httptest.NewRequest(http.MethodGet, "/api/chat/sessions/updates?since_event_id=42", nil)
	rec := httptest.NewRecorder()

	server.chatSessionUpdatesHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var payload struct {
		OwnerID        string               `json:"owner_id"`
		ResyncRequired bool                 `json:"resync_required"`
		Events         []sessionUpdateEvent `json:"events"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.OwnerID != chatSessionOwnerID || !payload.ResyncRequired {
		t.Fatalf("expected chat owner resync payload, got %+v", payload)
	}
	if len(payload.Events) != 0 {
		t.Fatalf("expected no incremental events when resync is required, got %+v", payload.Events)
	}
}

func TestSessionUpdatePollRequestsResyncWhenCursorFallsBehindGlobalWindow(t *testing.T) {
	broker := newSessionUpdateBroker(2)
	broker.publish(chatSessionOwnerID, "chat-1", "session.updated", nil)
	broker.publish(terminalSessionOwnerID, "terminal-1", "session.updated", nil)
	broker.publish(terminalSessionOwnerID, "terminal-2", "session.updated", nil)

	events, cursor, resyncRequired, hasMore := broker.poll(chatSessionOwnerID, 1, 50, 64*1024)

	if len(events) != 0 || !resyncRequired || hasMore {
		t.Fatalf("expected resync without events when cursor falls behind global window, got events=%+v cursor=%d resync=%v has_more=%v", events, cursor, resyncRequired, hasMore)
	}
	if cursor != 3 {
		t.Fatalf("expected cursor to advance to latest event id, got %d", cursor)
	}
}

func TestChatSessionNamedRouteHandlersUseChatOwner(t *testing.T) {
	service := &stubWebTerminalService{
		createResp: terminaldomain.Session{
			ID:        "chat-1",
			OwnerID:   chatSessionOwnerID,
			Title:     "chat-1",
			Status:    terminaldomain.SessionStatusReady,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
		inputResp: terminaldomain.Session{
			ID:        "chat-1",
			OwnerID:   chatSessionOwnerID,
			Title:     "chat-1",
			Status:    terminaldomain.SessionStatusBusy,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
		eventResp: terminalapp.RuntimeTraceEventDetail{
			TurnID: "turn-1",
			Blocks: []terminalapp.RuntimeBlock{{
				Type: "markdown",
				Text: "detail",
			}},
		},
	}
	server := &Server{terminals: service}

	createReq := httptest.NewRequest(http.MethodPost, "/api/chat/sessions", bytes.NewBufferString(`{}`))
	createRec := httptest.NewRecorder()
	server.chatSessionCollectionHandler(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d", createRec.Code)
	}
	if service.createReq.OwnerID != chatSessionOwnerID {
		t.Fatalf("expected chat owner for named chat create route, got %q", service.createReq.OwnerID)
	}

	inputReq := httptest.NewRequest(http.MethodPost, "/api/chat/sessions/chat-1/input", bytes.NewBufferString(`{"input":"hello"}`))
	inputRec := httptest.NewRecorder()
	server.chatSessionItemHandler(inputRec, inputReq)

	if inputRec.Code != http.StatusOK {
		t.Fatalf("expected input status 200, got %d", inputRec.Code)
	}
	if service.lastOwnerID != chatSessionOwnerID {
		t.Fatalf("expected chat owner for named chat input route, got %q", service.lastOwnerID)
	}

	detailReq := httptest.NewRequest(http.MethodGet, "/api/chat/sessions/chat-1/turns/turn-1/events/event-1", nil)
	detailRec := httptest.NewRecorder()
	server.chatSessionItemHandler(detailRec, detailReq)

	if detailRec.Code != http.StatusOK {
		t.Fatalf("expected event detail status 200, got %d", detailRec.Code)
	}
	if service.lastOwnerID != chatSessionOwnerID {
		t.Fatalf("expected chat owner for named chat event detail route, got %q", service.lastOwnerID)
	}
	if service.lastID != "chat-1:turn-1:event-1" {
		t.Fatalf("expected event detail lookup, got %q", service.lastID)
	}
}

func TestTerminalSessionItemHandlerWritesInput(t *testing.T) {
	service := &stubWebTerminalService{
		inputResp: terminaldomain.Session{
			ID:        "terminal-2",
			OwnerID:   terminalSessionOwnerID,
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
	if service.lastOwnerID != terminalSessionOwnerID {
		t.Fatalf("expected terminal owner, got %q", service.lastOwnerID)
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
			OwnerID:   terminalSessionOwnerID,
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
			OwnerID:   terminalSessionOwnerID,
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
			"alter0.skill.visibility": "private",
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

func TestTerminalSessionItemHandlerDefaultsMissingSkillIDsToAllPublicSkills(t *testing.T) {
	service := &stubWebTerminalService{
		inputResp: terminaldomain.Session{
			ID:        "terminal-2",
			OwnerID:   terminalSessionOwnerID,
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
		},
	}); err != nil {
		t.Fatalf("upsert summary skill failed: %v", err)
	}
	if err := control.UpsertCapability(controldomain.Capability{
		ID:      "memory",
		Name:    "Memory",
		Type:    controldomain.CapabilityTypeSkill,
		Enabled: true,
		Scope:   controldomain.CapabilityScopeGlobal,
		Version: controldomain.DefaultCapabilityVersion,
	}); err != nil {
		t.Fatalf("upsert memory skill failed: %v", err)
	}
	if err := control.UpsertCapability(controldomain.Capability{
		ID:      "disabled",
		Name:    "Disabled",
		Type:    controldomain.CapabilityTypeSkill,
		Enabled: false,
		Scope:   controldomain.CapabilityScopeGlobal,
		Version: controldomain.DefaultCapabilityVersion,
	}); err != nil {
		t.Fatalf("upsert disabled skill failed: %v", err)
	}
	if err := control.UpsertCapability(controldomain.Capability{
		ID:      "private",
		Name:    "Private",
		Type:    controldomain.CapabilityTypeSkill,
		Enabled: true,
		Scope:   controldomain.CapabilityScopeGlobal,
		Version: controldomain.DefaultCapabilityVersion,
		Metadata: map[string]string{
			"skill.visibility": "private",
		},
	}); err != nil {
		t.Fatalf("upsert private skill failed: %v", err)
	}
	server := &Server{terminals: service, control: control}

	req := httptest.NewRequest(http.MethodPost, "/api/terminal/sessions/terminal-2/input", bytes.NewBufferString(`{"input":"summarize"}`))
	rec := httptest.NewRecorder()

	server.terminalSessionItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if service.inputReq.SkillContext == nil {
		t.Fatalf("expected default skill context")
	}
	got := make(map[string]bool)
	for _, skill := range service.inputReq.SkillContext.Skills {
		got[skill.ID] = true
	}
	for _, id := range []string{"summary", "memory"} {
		if !got[id] {
			t.Fatalf("expected default skill %q in %+v", id, service.inputReq.SkillContext.Skills)
		}
	}
	for _, id := range []string{"disabled", "private"} {
		if got[id] {
			t.Fatalf("did not expect default skill %q in %+v", id, service.inputReq.SkillContext.Skills)
		}
	}
}

func TestTerminalSessionItemHandlerTreatsExplicitEmptySkillIDsAsNoSkills(t *testing.T) {
	service := &stubWebTerminalService{
		inputResp: terminaldomain.Session{
			ID:        "terminal-2",
			OwnerID:   terminalSessionOwnerID,
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
	}); err != nil {
		t.Fatalf("upsert summary skill failed: %v", err)
	}
	server := &Server{terminals: service, control: control}

	req := httptest.NewRequest(http.MethodPost, "/api/terminal/sessions/terminal-2/input", bytes.NewBufferString(`{"input":"summarize","skill_ids":[]}`))
	rec := httptest.NewRecorder()

	server.terminalSessionItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if service.inputReq.SkillContext != nil {
		t.Fatalf("expected explicit empty skill_ids to disable skills, got %+v", service.inputReq.SkillContext)
	}
}

func TestTerminalSessionRecoverHandlerRestoresStoredSession(t *testing.T) {
	service := &stubWebTerminalService{
		recoverResp: terminaldomain.Session{
			ID:                "terminal-recover",
			OwnerID:           terminalSessionOwnerID,
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
	if service.recoverReq.OwnerID != terminalSessionOwnerID {
		t.Fatalf("expected terminal owner, got %q", service.recoverReq.OwnerID)
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

func TestTerminalSessionItemHandlerPinsSession(t *testing.T) {
	service := &stubWebTerminalService{
		pinResp: terminaldomain.Session{
			ID:        "terminal-4",
			OwnerID:   terminalSessionOwnerID,
			Title:     "terminal-4",
			Status:    terminaldomain.SessionStatusReady,
			Pinned:    true,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
	}
	server := &Server{terminals: service}

	req := httptest.NewRequest(http.MethodPost, "/api/terminal/sessions/terminal-4/pin", bytes.NewBufferString(`{"pinned":true}`))
	rec := httptest.NewRecorder()

	server.terminalSessionItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if service.lastOwnerID != terminalSessionOwnerID {
		t.Fatalf("expected terminal owner, got %q", service.lastOwnerID)
	}
	if service.lastID != "terminal-4" {
		t.Fatalf("expected session terminal-4, got %q", service.lastID)
	}
	if !service.lastPinned {
		t.Fatalf("expected pinned true")
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	session, ok := payload["session"].(map[string]any)
	if !ok {
		t.Fatalf("expected session payload, got %v", payload)
	}
	if session["pinned"] != true {
		t.Fatalf("expected pinned session payload, got %v", session["pinned"])
	}
}

func TestTerminalSessionItemHandlerUnpinsSessionWithExplicitFalse(t *testing.T) {
	service := &stubWebTerminalService{
		pinResp: terminaldomain.Session{
			ID:        "terminal-4",
			OwnerID:   terminalSessionOwnerID,
			Title:     "terminal-4",
			Status:    terminaldomain.SessionStatusReady,
			Pinned:    false,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
	}
	server := &Server{terminals: service}

	req := httptest.NewRequest(http.MethodPost, "/api/terminal/sessions/terminal-4/pin", bytes.NewBufferString(`{"pinned":false}`))
	rec := httptest.NewRecorder()

	server.terminalSessionItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if service.lastPinned {
		t.Fatalf("expected pinned false")
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	session, ok := payload["session"].(map[string]any)
	if !ok {
		t.Fatalf("expected session payload, got %v", payload)
	}
	pinned, ok := session["pinned"].(bool)
	if !ok || pinned {
		t.Fatalf("expected explicit pinned false payload, got value=%v present=%v", session["pinned"], ok)
	}
}

func TestTerminalSessionItemHandlerDeletesSession(t *testing.T) {
	service := &stubWebTerminalService{
		deleteResp: terminaldomain.Session{
			ID:         "terminal-4",
			OwnerID:    terminalSessionOwnerID,
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
	if service.lastOwnerID != terminalSessionOwnerID {
		t.Fatalf("expected terminal owner, got %q", service.lastOwnerID)
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
			OwnerID:   terminalSessionOwnerID,
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

func TestTerminalSessionItemHandlerPagesTurnsInSessionDetail(t *testing.T) {
	service := &stubWebTerminalService{
		getResp: terminaldomain.Session{
			ID:      "terminal-4",
			OwnerID: terminalSessionOwnerID,
			Status:  terminaldomain.SessionStatusReady,
		},
		getOK: true,
		turnsResp: []terminalapp.TurnSummary{
			{ID: "turn-1", Prompt: "one", Status: "completed", FinalOutput: "1"},
			{ID: "turn-2", Prompt: "two", Status: "completed", FinalOutput: "2"},
			{ID: "turn-3", Prompt: "three", Status: "completed", FinalOutput: "3"},
			{ID: "turn-4", Prompt: "four", Status: "completed", FinalOutput: "4"},
		},
	}
	server := &Server{terminals: service}

	req := httptest.NewRequest(http.MethodGet, "/api/terminal/sessions/terminal-4?turn_limit=2", nil)
	rec := httptest.NewRecorder()

	server.terminalSessionItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	session := payload["session"].(map[string]any)
	turns := session["turns"].([]any)
	if len(turns) != 2 {
		t.Fatalf("expected two turns, got %d", len(turns))
	}
	if turns[0].(map[string]any)["id"] != "turn-3" || turns[1].(map[string]any)["id"] != "turn-4" {
		t.Fatalf("expected latest turn page, got %v", turns)
	}
	paging := session["turns_paging"].(map[string]any)
	if paging["has_more_before"] != true {
		t.Fatalf("expected earlier turns to be available, got %v", paging)
	}
	if paging["oldest_turn_id"] != "turn-3" || paging["newest_turn_id"] != "turn-4" {
		t.Fatalf("expected page boundary ids, got %v", paging)
	}
}

func TestTerminalSessionItemHandlerUsesCompactDefaultTurnPage(t *testing.T) {
	turns := make([]terminalapp.TurnSummary, 45)
	for index := range turns {
		turnNumber := index + 1
		turns[index] = terminalapp.TurnSummary{
			ID:          fmt.Sprintf("turn-%02d", turnNumber),
			Prompt:      fmt.Sprintf("prompt-%02d", turnNumber),
			Status:      "completed",
			FinalOutput: fmt.Sprintf("output-%02d", turnNumber),
		}
	}
	service := &stubWebTerminalService{
		getResp: terminaldomain.Session{
			ID:      "terminal-4",
			OwnerID: terminalSessionOwnerID,
			Status:  terminaldomain.SessionStatusReady,
		},
		getOK:     true,
		turnsResp: turns,
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
	session := payload["session"].(map[string]any)
	page := session["turns"].([]any)
	if len(page) != 20 {
		t.Fatalf("expected compact default page of 20 turns, got %d", len(page))
	}
	if page[0].(map[string]any)["id"] != "turn-26" || page[len(page)-1].(map[string]any)["id"] != "turn-45" {
		t.Fatalf("expected latest compact turn page, got first=%v last=%v", page[0], page[len(page)-1])
	}
	paging := session["turns_paging"].(map[string]any)
	if paging["limit"] != float64(20) || paging["has_more_before"] != true {
		t.Fatalf("expected default paging metadata for compact page, got %v", paging)
	}
}

func TestTerminalSessionItemHandlerCapsTurnPageByApproximatePayloadSize(t *testing.T) {
	largeOutput := strings.Repeat("large terminal output\n", 5000)
	turns := make([]terminalapp.TurnSummary, 5)
	for index := range turns {
		turnNumber := index + 1
		turns[index] = terminalapp.TurnSummary{
			ID:          fmt.Sprintf("turn-%02d", turnNumber),
			Prompt:      fmt.Sprintf("prompt-%02d", turnNumber),
			Status:      "completed",
			FinalOutput: largeOutput,
		}
	}
	service := &stubWebTerminalService{
		getResp: terminaldomain.Session{
			ID:      "terminal-4",
			OwnerID: terminalSessionOwnerID,
			Status:  terminaldomain.SessionStatusReady,
		},
		getOK:     true,
		turnsResp: turns,
	}
	server := &Server{terminals: service}

	req := httptest.NewRequest(http.MethodGet, "/api/terminal/sessions/terminal-4?turn_limit=5", nil)
	rec := httptest.NewRecorder()

	server.terminalSessionItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	session := payload["session"].(map[string]any)
	page := session["turns"].([]any)
	if len(page) >= len(turns) {
		t.Fatalf("expected payload budget to return fewer turns than requested, got %d", len(page))
	}
	if page[len(page)-1].(map[string]any)["id"] != "turn-05" {
		t.Fatalf("expected capped page to keep the latest turn, got %v", page)
	}
	paging := session["turns_paging"].(map[string]any)
	if paging["has_more_before"] != true {
		t.Fatalf("expected payload cap to keep earlier turns available, got %v", paging)
	}
}

func TestTerminalSessionItemHandlerReturnsRuntimeTraceEventDetail(t *testing.T) {
	service := &stubWebTerminalService{
		eventResp: terminalapp.RuntimeTraceEventDetail{
			TurnID: "turn-1",
			Event: terminalapp.RuntimeTraceEvent{
				ID:         "event-1",
				TurnID:     "turn-1",
				Seq:        1,
				Source:     "adapter",
				Provider:   terminalapp.RuntimeProviderRef{Engine: "codex", Adapter: "codex_cli_json"},
				Role:       "assistant",
				Kind:       "shell_command",
				Lifecycle:  "completed",
				Status:     "completed",
				Title:      "pwd",
				Blocks:     []terminalapp.RuntimeBlock{},
				Visibility: "collapsed",
			},
			Blocks: []terminalapp.RuntimeBlock{{
				Type:    "terminal",
				Title:   "Shell",
				Command: "pwd",
				Output:  "/workspace/alter0",
			}},
			Searchable: true,
		},
	}
	server := &Server{terminals: service}

	req := httptest.NewRequest(http.MethodGet, "/api/terminal/sessions/terminal-4/turns/turn-1/events/event-1", nil)
	rec := httptest.NewRecorder()

	server.terminalSessionItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if service.lastID != "terminal-4:turn-1:event-1" {
		t.Fatalf("expected event lookup path, got %q", service.lastID)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	event, ok := payload["event"].(map[string]any)
	if !ok {
		t.Fatalf("expected event payload, got %v", payload)
	}
	if event["turn_id"] != "turn-1" {
		t.Fatalf("expected turn_id turn-1, got %v", event["turn_id"])
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

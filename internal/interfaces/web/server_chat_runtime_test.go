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

	chatruntimeapp "alter0/internal/chatruntime/application"
	chatruntimedomain "alter0/internal/chatruntime/domain"
	controlapp "alter0/internal/control/application"
	controldomain "alter0/internal/control/domain"
)

type stubWebChatRuntimeService struct {
	createReq      chatruntimeapp.CreateRequest
	createResp     chatruntimedomain.Session
	createErr      error
	recoverReq     chatruntimeapp.RecoverRequest
	recoverResp    chatruntimedomain.Session
	recoverErr     error
	listResp       []chatruntimedomain.Session
	listByOwner    map[string][]chatruntimedomain.Session
	getResp        chatruntimedomain.Session
	getOK          bool
	inputResp      chatruntimedomain.Session
	inputErr       error
	pinResp        chatruntimedomain.Session
	pinErr         error
	deleteResp     chatruntimedomain.Session
	deleteErr      error
	deleteIDs      []string
	deleteOwnerIDs []string
	turnsResp      []chatruntimeapp.TurnSummary
	turnsErr       error
	eventResp      chatruntimeapp.RuntimeTraceEventDetail
	stepErr        error
	entryPage      chatruntimeapp.EntryPage
	entryErr       error
	lastOwnerID    string
	lastID         string
	lastInput      string
	lastPinned     bool
	inputReq       chatruntimeapp.InputRequest
	eventHook      chatruntimeapp.SessionEventHook
}

func (s *stubWebChatRuntimeService) Create(req chatruntimeapp.CreateRequest) (chatruntimedomain.Session, error) {
	s.createReq = req
	return s.createResp, s.createErr
}

func (s *stubWebChatRuntimeService) Recover(req chatruntimeapp.RecoverRequest) (chatruntimedomain.Session, error) {
	s.recoverReq = req
	return s.recoverResp, s.recoverErr
}

func (s *stubWebChatRuntimeService) List(ownerID string) []chatruntimedomain.Session {
	s.lastOwnerID = ownerID
	if s.listByOwner != nil {
		return append([]chatruntimedomain.Session{}, s.listByOwner[ownerID]...)
	}
	return append([]chatruntimedomain.Session{}, s.listResp...)
}

func (s *stubWebChatRuntimeService) Get(ownerID string, sessionID string) (chatruntimedomain.Session, bool) {
	s.lastOwnerID = ownerID
	s.lastID = sessionID
	return s.getResp, s.getOK
}

func (s *stubWebChatRuntimeService) ListTurns(ownerID string, sessionID string) ([]chatruntimeapp.TurnSummary, error) {
	s.lastOwnerID = ownerID
	s.lastID = sessionID
	return append([]chatruntimeapp.TurnSummary{}, s.turnsResp...), s.turnsErr
}

func (s *stubWebChatRuntimeService) GetRuntimeTraceEventDetail(ownerID string, sessionID string, turnID string, eventID string) (chatruntimeapp.RuntimeTraceEventDetail, error) {
	s.lastOwnerID = ownerID
	s.lastID = sessionID + ":" + turnID + ":" + eventID
	return s.eventResp, s.stepErr
}

func (s *stubWebChatRuntimeService) ListEntries(ownerID string, sessionID string, _ int, _ int) (chatruntimeapp.EntryPage, error) {
	s.lastOwnerID = ownerID
	s.lastID = sessionID
	return s.entryPage, s.entryErr
}

func (s *stubWebChatRuntimeService) Input(ownerID string, sessionID string, input string) (chatruntimedomain.Session, error) {
	s.lastOwnerID = ownerID
	s.lastID = sessionID
	s.lastInput = input
	return s.inputResp, s.inputErr
}

func (s *stubWebChatRuntimeService) InputWithAttachments(req chatruntimeapp.InputRequest) (chatruntimedomain.Session, error) {
	s.lastOwnerID = req.OwnerID
	s.lastID = req.SessionID
	s.lastInput = req.Input
	s.inputReq = req
	return s.inputResp, s.inputErr
}

func (s *stubWebChatRuntimeService) SetPinned(ownerID string, sessionID string, pinned bool) (chatruntimedomain.Session, error) {
	s.lastOwnerID = ownerID
	s.lastID = sessionID
	s.lastPinned = pinned
	return s.pinResp, s.pinErr
}

func (s *stubWebChatRuntimeService) Delete(ownerID string, sessionID string) (chatruntimedomain.Session, error) {
	s.lastOwnerID = ownerID
	s.lastID = sessionID
	s.deleteIDs = append(s.deleteIDs, sessionID)
	s.deleteOwnerIDs = append(s.deleteOwnerIDs, ownerID+":"+sessionID)
	return s.deleteResp, s.deleteErr
}

func (s *stubWebChatRuntimeService) SetSessionEventHook(hook chatruntimeapp.SessionEventHook) {
	s.eventHook = hook
}

func assertNoLegacyChatRuntimeAPIFields(t *testing.T, body string) {
	t.Helper()
	for _, field := range []string{
		`"runtime_session_id"`,
		`"owner_id"`,
		`"shell"`,
		`"working_dir"`,
		`"revision"`,
		`"version"`,
		`"event_id"`,
		`"cursor"`,
		`"since_event_id"`,
		`"seq"`,
		`"provider"`,
		`"source"`,
		`"role"`,
		`"lifecycle"`,
		`"summary"`,
		`"visibility"`,
		`"raw"`,
		`"action"`,
	} {
		if strings.Contains(body, field) {
			t.Fatalf("expected chat runtime API payload to omit legacy field %s, got %s", field, body)
		}
	}
}

func TestChatRuntimeSessionCollectionHandlerCreatesSession(t *testing.T) {
	service := &stubWebChatRuntimeService{
		createResp: chatruntimedomain.Session{
			ID:           "chatRuntime-1",
			OwnerID:      chatSessionOwnerID,
			Title:        "chatRuntime-1",
			Status:       chatruntimedomain.SessionStatusReady,
			CreatedAt:    time.Now().UTC(),
			LastOutputAt: time.Now().UTC(),
			UpdatedAt:    time.Now().UTC(),
		},
	}
	server := &Server{chatRuntimes: service}

	req := httptest.NewRequest(http.MethodPost, "/api/chat/sessions", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()

	server.chatRuntimeSessionCollectionHandler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}
	if service.createReq.OwnerID != chatSessionOwnerID {
		t.Fatalf("expected chatRuntime owner, got %q", service.createReq.OwnerID)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	session, ok := payload["session"].(map[string]any)
	if !ok {
		t.Fatalf("expected session payload, got %v", payload)
	}
	if session["id"] != "chatRuntime-1" {
		t.Fatalf("expected chatRuntime id chatRuntime-1, got %v", session["id"])
	}
	if value, ok := session["last_output_at"].(float64); !ok || value <= 0 {
		t.Fatalf("expected last_output_at in session payload, got %v", session["last_output_at"])
	}
	if value, ok := session["activity_at"].(float64); !ok || value <= 0 {
		t.Fatalf("expected activity_at in session payload, got %v", session["activity_at"])
	}
	assertNoLegacyChatRuntimeAPIFields(t, rec.Body.String())
}

func TestChatRuntimeSessionCollectionHandlerReturnsComparableSessionSummaries(t *testing.T) {
	service := &stubWebChatRuntimeService{
		listResp: []chatruntimedomain.Session{
			{
				ID:           "chatRuntime-older-active",
				OwnerID:      chatSessionOwnerID,
				Title:        "Older active",
				Status:       chatruntimedomain.SessionStatusReady,
				CreatedAt:    time.Date(2026, 4, 21, 3, 30, 0, 0, time.UTC),
				LastOutputAt: time.Date(2026, 4, 23, 4, 30, 0, 0, time.UTC),
				UpdatedAt:    time.Date(2026, 4, 23, 4, 31, 0, 0, time.UTC),
			},
			{
				ID:        "chatRuntime-new-idle",
				OwnerID:   chatSessionOwnerID,
				Title:     "New idle",
				Status:    chatruntimedomain.SessionStatusReady,
				CreatedAt: time.Date(2026, 4, 23, 3, 30, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, 4, 23, 3, 30, 0, 0, time.UTC),
			},
		},
	}
	server := &Server{chatRuntimes: service}

	req := httptest.NewRequest(http.MethodGet, "/api/chat/sessions", nil)
	rec := httptest.NewRecorder()

	server.chatRuntimeSessionCollectionHandler(rec, req)

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
		if value, ok := session["activity_at"].(float64); !ok || value <= 0 {
			t.Fatalf("expected activity_at in session summary, got %v", session)
		}
		if _, hasTurns := session["turns"]; hasTurns {
			t.Fatalf("expected collection summary without turns, got %v", session)
		}
	}
	assertNoLegacyChatRuntimeAPIFields(t, rec.Body.String())
}

func TestChatSessionItemHandlerRejectsRemovedRuntimePathFallback(t *testing.T) {
	service := &stubWebChatRuntimeService{
		getResp: chatruntimedomain.Session{
			ID:        "chat-1",
			OwnerID:   chatSessionOwnerID,
			Title:     "Chat 1",
			Status:    chatruntimedomain.SessionStatusReady,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
		getOK: true,
	}
	server := &Server{chatRuntimes: service}

	req := httptest.NewRequest(http.MethodGet, "/removed-runtime/sessions/chat-1", nil)
	rec := httptest.NewRecorder()

	server.chatSessionItemHandler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected removed runtime session path to be rejected, got %d", rec.Code)
	}
	if service.lastID != "" {
		t.Fatalf("expected no runtime session lookup for removed runtime path, got %q", service.lastID)
	}
}

func TestChatRuntimeSessionSummaryUsesMillisecondUpdatedAtWithoutRevision(t *testing.T) {
	first := buildChatRuntimeSessionSummary(chatruntimedomain.Session{
		ID:        "chatRuntime-fast-1",
		OwnerID:   chatSessionOwnerID,
		Title:     "Fast 1",
		Status:    chatruntimedomain.SessionStatusReady,
		CreatedAt: time.Date(2026, 4, 23, 4, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 23, 4, 30, 1, 123400000, time.UTC),
	})
	second := buildChatRuntimeSessionSummary(chatruntimedomain.Session{
		ID:        "chatRuntime-fast-2",
		OwnerID:   chatSessionOwnerID,
		Title:     "Fast 2",
		Status:    chatruntimedomain.SessionStatusReady,
		CreatedAt: time.Date(2026, 4, 23, 4, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 23, 4, 30, 1, 124400000, time.UTC),
	})

	firstMap := first.(map[string]any)
	secondMap := second.(map[string]any)
	if _, ok := firstMap["revision"]; ok {
		t.Fatalf("expected revision to be removed from session summary, got %v", firstMap)
	}
	if firstMap["updated_at"] == secondMap["updated_at"] {
		t.Fatalf("expected millisecond updated_at values to differ, got %v", firstMap["updated_at"])
	}
}

func TestChatSessionUpdatesHandlerReturnsIncrementalOwnerEvents(t *testing.T) {
	server := &Server{sessionEvents: newSessionUpdateBroker(8)}
	server.publishChatRuntimeSessionSummaryEvent(chatSessionOwnerID, "chat-1", "session.updated", chatruntimedomain.Session{
		ID:      "chat-1",
		OwnerID: chatSessionOwnerID,
		Title:   "Running chat",
		Status:  chatruntimedomain.SessionStatusBusy,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/chat/sessions/updates", strings.NewReader(`{"after_update_id":0}`))
	rec := httptest.NewRecorder()

	server.chatSessionUpdatesHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var payload struct {
		LatestUpdateID int64                `json:"latest_update_id"`
		ResyncRequired bool                 `json:"resync_required"`
		Updates        []sessionUpdateEvent `json:"updates"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.ResyncRequired {
		t.Fatalf("did not expect resync_required")
	}
	if payload.LatestUpdateID == 0 || len(payload.Updates) != 1 {
		t.Fatalf("expected one incremental update and non-zero latest_update_id, got %+v", payload)
	}
	event := payload.Updates[0]
	if event.SessionID != "chat-1" || event.EventType != "session.updated" || event.EventID == 0 {
		t.Fatalf("expected chat session.updated update, got %+v", event)
	}
	session, ok := event.Payload["session"].(map[string]any)
	if !ok || session["status"] != "busy" {
		t.Fatalf("expected busy session payload, got %+v", event.Payload)
	}
	assertNoLegacyChatRuntimeAPIFields(t, rec.Body.String())
}

func TestChatSessionUpdatesHandlerRequiresPost(t *testing.T) {
	server := &Server{sessionEvents: newSessionUpdateBroker(8)}
	req := httptest.NewRequest(http.MethodGet, "/api/chat/sessions/updates?since_event_id=0", nil)
	rec := httptest.NewRecorder()

	server.chatSessionUpdatesHandler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405 for unsupported updates method, got %d", rec.Code)
	}
}

func TestChatSessionUpdatesHandlerPrunesKnownRuntimeTraceEvents(t *testing.T) {
	server := &Server{sessionEvents: newSessionUpdateBroker(8)}
	server.sessionUpdateBroker().publishWithTurnID(chatSessionOwnerID, "chat-1", "turn-1", "turn.event.appended", map[string]any{
		"session":       map[string]any{"id": "chat-1", "status": "busy"},
		"turn":          map[string]any{"id": 1, "prompt": "hello", "status": "running"},
		"runtime_event": map[string]any{"id": 3, "kind": "reasoning", "status": "running", "text": "missing 3"},
	})
	body := strings.NewReader(`{
		"after_update_id": 0,
		"limit": 50,
		"byte_limit": 1048576,
		"sessions": [{
			"id": "chat-1",
			"turns": [{
				"id": 1,
				"event_ids": [1, 2]
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
		Updates []sessionUpdateAPIUpdate `json:"updates"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(payload.Updates) != 1 {
		t.Fatalf("expected one update, got %+v", payload.Updates)
	}
	event, ok := payload.Updates[0].Payload["runtime_event"].(map[string]any)
	if !ok || event["id"] != float64(3) {
		t.Fatalf("expected missing runtime_event 3, got %+v", payload.Updates[0].Payload)
	}
	session, _ := payload.Updates[0].Payload["session"].(map[string]any)
	if _, hasTurns := session["turns"]; hasTurns {
		t.Fatalf("typed update session payload must not carry turns, got %+v", session)
	}
	assertNoLegacyChatRuntimeAPIFields(t, rec.Body.String())
}

func TestSessionUpdatePollAllowsOneMiBByteBudget(t *testing.T) {
	broker := newSessionUpdateBroker(8)
	largeVisiblePayload := strings.Repeat("x", 160*1024)
	broker.publish(chatSessionOwnerID, "chat-1", "session.updated", map[string]any{
		"session": map[string]any{
			"id":      "chat-1",
			"status":  "busy",
			"summary": largeVisiblePayload,
		},
	})
	broker.publish(chatSessionOwnerID, "chat-1", "session.updated", map[string]any{
		"session": map[string]any{
			"id":      "chat-1",
			"status":  "busy",
			"summary": largeVisiblePayload,
		},
	})

	events, cursor, resyncRequired, hasMore := broker.poll(chatSessionOwnerID, 0, 50, 1024*1024)

	if resyncRequired {
		t.Fatalf("did not expect resync for fresh cursor")
	}
	if hasMore {
		t.Fatalf("did not expect 1MiB poll budget to truncate two visible updates")
	}
	if cursor != 2 || len(events) != 2 {
		t.Fatalf("expected both updates under 1MiB budget, cursor=%d events=%d", cursor, len(events))
	}
}

func TestChatSessionUpdatesHandlerSkipsInvisibleRuntimeTraceEventUpdates(t *testing.T) {
	server := &Server{sessionEvents: newSessionUpdateBroker(8)}
	server.sessionUpdateBroker().publishWithTurnID(chatSessionOwnerID, "chat-1", "turn-1", "turn.event.appended", map[string]any{
		"session":       map[string]any{"id": "chat-1", "status": "busy"},
		"turn":          map[string]any{"id": 1, "prompt": "hello", "status": "running"},
		"runtime_event": map[string]any{"id": 1, "kind": "commands", "status": "running", "text": "npm test"},
	})
	server.sessionUpdateBroker().publishWithTurnID(chatSessionOwnerID, "chat-1", "turn-1", "turn.event.appended", map[string]any{
		"session":       map[string]any{"id": "chat-1", "status": "busy"},
		"turn":          map[string]any{"id": 1, "prompt": "hello", "status": "running"},
		"runtime_event": map[string]any{"id": 2, "kind": "reasoning", "status": "running", "text": "Thinking"},
	})
	body := strings.NewReader(`{
		"after_update_id": 0,
		"limit": 50,
		"byte_limit": 1048576,
		"visible_event_kinds": ["reasoning"]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/chat/sessions/updates", body)
	rec := httptest.NewRecorder()

	server.chatSessionUpdatesHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var payload struct {
		LatestUpdateID int64                    `json:"latest_update_id"`
		ResyncRequired bool                     `json:"resync_required"`
		Updates        []sessionUpdateAPIUpdate `json:"updates"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.ResyncRequired {
		t.Fatalf("did not expect resync when invisible events are skipped")
	}
	if payload.LatestUpdateID != 2 {
		t.Fatalf("expected latest_update_id to advance over skipped invisible update, got %d", payload.LatestUpdateID)
	}
	if len(payload.Updates) != 1 || payload.Updates[0].UpdateID != 2 {
		t.Fatalf("expected only visible reasoning update, got %+v", payload.Updates)
	}
	raw, _ := json.Marshal(payload.Updates[0].Payload)
	if strings.Contains(string(raw), "npm test") || !strings.Contains(string(raw), "Thinking") {
		t.Fatalf("expected payload to include reasoning and omit command, got %s", raw)
	}
	assertNoLegacyChatRuntimeAPIFields(t, rec.Body.String())
}

func TestChatSessionUpdatesHandlerPrunesInvisibleRuntimeTraceEventsWithoutAckManifest(t *testing.T) {
	server := &Server{sessionEvents: newSessionUpdateBroker(8)}
	server.sessionUpdateBroker().publishWithTurnID(chatSessionOwnerID, "chat-1", "turn-1", "turn.event.appended", map[string]any{
		"session":       map[string]any{"id": "chat-1", "status": "busy"},
		"turn":          map[string]any{"id": 1, "prompt": "hello", "status": "running"},
		"runtime_event": map[string]any{"id": 1, "kind": "commands", "status": "running", "text": "npm test"},
	})
	server.sessionUpdateBroker().publishWithTurnID(chatSessionOwnerID, "chat-1", "turn-1", "turn.event.appended", map[string]any{
		"session":       map[string]any{"id": "chat-1", "status": "busy"},
		"turn":          map[string]any{"id": 1, "prompt": "hello", "status": "running"},
		"runtime_event": map[string]any{"id": 2, "kind": "reasoning", "status": "running", "text": "Thinking"},
	})
	body := strings.NewReader(`{
		"after_update_id": 0,
		"limit": 50,
		"byte_limit": 1048576,
		"visible_event_kinds": ["reasoning"]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/chat/sessions/updates", body)
	rec := httptest.NewRecorder()

	server.chatSessionUpdatesHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var payload struct {
		Updates []sessionUpdateAPIUpdate `json:"updates"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(payload.Updates) != 1 || payload.Updates[0].UpdateID != 2 {
		t.Fatalf("expected only visible reasoning update, got %+v", payload.Updates)
	}
	raw, _ := json.Marshal(payload.Updates[0].Payload)
	if strings.Contains(string(raw), "npm test") || !strings.Contains(string(raw), "Thinking") {
		t.Fatalf("expected payload to include reasoning and omit command without ack manifest, got %s", raw)
	}
	assertNoLegacyChatRuntimeAPIFields(t, rec.Body.String())
}

func TestChatSessionUpdatesHandlerUsesFrontendRuntimeEventCategories(t *testing.T) {
	server := &Server{sessionEvents: newSessionUpdateBroker(8)}
	server.sessionUpdateBroker().publishWithTurnID(chatSessionOwnerID, "chat-1", "turn-1", "turn.event.appended", map[string]any{
		"session":       map[string]any{"id": "chat-1", "status": "busy"},
		"turn":          map[string]any{"id": 1, "prompt": "hello", "status": "running"},
		"runtime_event": map[string]any{"id": 1, "kind": "important_text", "status": "running", "text": "Visible commentary"},
	})
	body := strings.NewReader(`{
		"after_update_id": 0,
		"limit": 50,
		"byte_limit": 1048576,
		"visible_event_kinds": ["reasoning"]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/chat/sessions/updates", body)
	rec := httptest.NewRecorder()

	server.chatSessionUpdatesHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var payload struct {
		LatestUpdateID int64                    `json:"latest_update_id"`
		Updates        []sessionUpdateAPIUpdate `json:"updates"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.LatestUpdateID != 1 {
		t.Fatalf("expected latest_update_id to advance over hidden important text, got %d", payload.LatestUpdateID)
	}
	if len(payload.Updates) != 0 {
		t.Fatalf("expected important_text update to be hidden by reasoning-only filter, got %+v", payload.Updates)
	}
	assertNoLegacyChatRuntimeAPIFields(t, rec.Body.String())
}

func TestChatRuntimeSessionEventHookPublishesMinimalTurnPatches(t *testing.T) {
	now := time.Now().UTC()
	service := &stubWebChatRuntimeService{}
	server := &Server{chatRuntimes: service, sessionEvents: newSessionUpdateBroker(8)}
	server.registerChatRuntimeSessionEventHook()
	if service.eventHook == nil {
		t.Fatalf("expected typed session event hook to be registered")
	}

	service.eventHook(chatruntimeapp.SessionEvent{
		OwnerID:   chatSessionOwnerID,
		SessionID: "chat-1",
		EventType: chatruntimeapp.SessionEventTurnEventAppended,
		Session: chatruntimedomain.Session{
			ID:         "chat-1",
			OwnerID:    chatSessionOwnerID,
			Title:      "Running chat",
			Status:     chatruntimedomain.SessionStatusBusy,
			Shell:      "codex exec",
			WorkingDir: "/workspace/chat-1",
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		Turn: &chatruntimeapp.TurnSummary{
			ID:     "turn-1",
			Prompt: "hello",
			Status: "running",
			RuntimeTraceEvents: []chatruntimeapp.RuntimeTraceEvent{{
				ID:     "old-event",
				TurnID: "turn-1",
				Seq:    1,
				Kind:   "reasoning",
				Status: "completed",
			}},
		},
		RuntimeEvent: &chatruntimeapp.RuntimeTraceEvent{
			ID:         "event-1",
			SessionID:  "chat-1",
			TurnID:     "turn-1",
			Seq:        2,
			Source:     "adapter",
			Provider:   chatruntimeapp.RuntimeProviderRef{Engine: "codex", Adapter: "codex_cli_json", EventType: "command"},
			Role:       "assistant",
			Kind:       "shell_command",
			Lifecycle:  "started",
			Status:     "running",
			Title:      "echo hi",
			Blocks:     []chatruntimeapp.RuntimeBlock{},
			Visibility: "collapsed",
		},
	})
	service.eventHook(chatruntimeapp.SessionEvent{
		OwnerID:   chatSessionOwnerID,
		SessionID: "chat-1",
		EventType: chatruntimeapp.SessionEventTurnCompleted,
		Session: chatruntimedomain.Session{
			ID:         "chat-1",
			OwnerID:    chatSessionOwnerID,
			Title:      "Running chat",
			Status:     chatruntimedomain.SessionStatusReady,
			Shell:      "codex exec",
			WorkingDir: "/workspace/chat-1",
			CreatedAt:  now,
			UpdatedAt:  now.Add(time.Second),
		},
		Turn: &chatruntimeapp.TurnSummary{
			ID:          "turn-1",
			Prompt:      "hello",
			Status:      "completed",
			FinalOutput: "done",
			RuntimeTraceEvents: []chatruntimeapp.RuntimeTraceEvent{{
				ID:     "event-1",
				TurnID: "turn-1",
				Seq:    2,
				Kind:   "shell_command",
				Status: "completed",
			}},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/chat/sessions/updates", strings.NewReader(`{"after_update_id":0}`))
	rec := httptest.NewRecorder()

	server.chatSessionUpdatesHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var payload struct {
		Updates []sessionUpdateAPIUpdate `json:"updates"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(payload.Updates) != 2 {
		t.Fatalf("expected two typed turn updates, got %+v", payload.Updates)
	}
	if payload.Updates[0].Type != chatruntimeapp.SessionEventTurnEventAppended || payload.Updates[0].TurnID != 1 {
		t.Fatalf("expected first update to be turn event patch, got %+v", payload.Updates[0])
	}
	if payload.Updates[1].Type != chatruntimeapp.SessionEventTurnCompleted || payload.Updates[1].TurnID != 1 {
		t.Fatalf("expected second update to be turn completion, got %+v", payload.Updates[1])
	}
	firstRaw, _ := json.Marshal(payload.Updates[0].Payload)
	firstBody := string(firstRaw)
	if !strings.Contains(firstBody, `"runtime_event":`) || !strings.Contains(firstBody, `"id":2`) || !strings.Contains(firstBody, `"kind":"commands"`) {
		t.Fatalf("expected one runtime event patch in update payload, got %q", firstBody)
	}
	if strings.Contains(firstBody, `"old-event"`) {
		t.Fatalf("expected runtime event patch to omit pre-existing turn event list, got %q", firstBody)
	}
	if strings.Contains(firstBody, `"runtime_trace_events"`) || strings.Contains(firstBody, `"runtime_trace_events_partial"`) {
		t.Fatalf("typed update payload must not wrap runtime_event in turn runtime_trace_events, got %q", firstBody)
	}
	if strings.Contains(firstBody, `"turns"`) {
		t.Fatalf("typed update session payload must not carry turns, got %q", firstBody)
	}
	secondRaw, _ := json.Marshal(payload.Updates[1].Payload)
	secondBody := string(secondRaw)
	if !strings.Contains(secondBody, `"final_output":"done"`) {
		t.Fatalf("expected turn completion payload to include final output, got %q", secondBody)
	}
	if strings.Contains(secondBody, `"runtime_trace_events"`) || strings.Contains(secondBody, `"turns"`) {
		t.Fatalf("expected turn completion payload to omit full runtime event history, got %q", secondBody)
	}
	for _, body := range []string{firstBody, secondBody} {
		if strings.Contains(body, `"owner_id"`) || strings.Contains(body, `"shell"`) || strings.Contains(body, `"working_dir"`) || strings.Contains(body, `"runtime_session_id"`) {
			t.Fatalf("expected typed update payload to omit non-consumed runtime metadata, got %q", body)
		}
	}
	assertNoLegacyChatRuntimeAPIFields(t, rec.Body.String())
}

func TestChatSessionUpdatesHandlerRequestsResyncWhenCursorCannotResume(t *testing.T) {
	server := &Server{sessionEvents: newSessionUpdateBroker(8)}

	req := httptest.NewRequest(http.MethodPost, "/api/chat/sessions/updates", strings.NewReader(`{"after_update_id":42}`))
	rec := httptest.NewRecorder()

	server.chatSessionUpdatesHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var payload struct {
		LatestUpdateID int64                    `json:"latest_update_id"`
		ResyncRequired bool                     `json:"resync_required"`
		Updates        []sessionUpdateAPIUpdate `json:"updates"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !payload.ResyncRequired {
		t.Fatalf("expected chat resync payload, got %+v", payload)
	}
	if len(payload.Updates) != 0 {
		t.Fatalf("expected no incremental updates when resync is required, got %+v", payload.Updates)
	}
	assertNoLegacyChatRuntimeAPIFields(t, rec.Body.String())
}

func TestSessionUpdatePollRequestsResyncWhenCursorFallsBehindGlobalWindow(t *testing.T) {
	broker := newSessionUpdateBroker(2)
	broker.publish(chatSessionOwnerID, "chat-1", "session.updated", nil)
	broker.publish(chatSessionOwnerID, "chatRuntime-1", "session.updated", nil)
	broker.publish(chatSessionOwnerID, "chatRuntime-2", "session.updated", nil)

	events, cursor, resyncRequired, hasMore := broker.poll(chatSessionOwnerID, 1, 50, 64*1024)

	if len(events) != 0 || !resyncRequired || hasMore {
		t.Fatalf("expected resync without events when cursor falls behind global window, got events=%+v cursor=%d resync=%v has_more=%v", events, cursor, resyncRequired, hasMore)
	}
	if cursor != 3 {
		t.Fatalf("expected cursor to advance to latest event id, got %d", cursor)
	}
}

func TestChatSessionNamedRouteHandlersUseChatOwner(t *testing.T) {
	service := &stubWebChatRuntimeService{
		createResp: chatruntimedomain.Session{
			ID:        "chat-1",
			OwnerID:   chatSessionOwnerID,
			Title:     "chat-1",
			Status:    chatruntimedomain.SessionStatusReady,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
		inputResp: chatruntimedomain.Session{
			ID:        "chat-1",
			OwnerID:   chatSessionOwnerID,
			Title:     "chat-1",
			Status:    chatruntimedomain.SessionStatusBusy,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
		eventResp: chatruntimeapp.RuntimeTraceEventDetail{
			TurnID: "turn-1",
			Blocks: []chatruntimeapp.RuntimeBlock{{
				Type: "markdown",
				Text: "detail",
			}},
		},
	}
	server := &Server{chatRuntimes: service}

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

func TestChatRuntimeSessionItemHandlerWritesInput(t *testing.T) {
	service := &stubWebChatRuntimeService{
		inputResp: chatruntimedomain.Session{
			ID:        "chatRuntime-2",
			OwnerID:   chatSessionOwnerID,
			Title:     "chatRuntime-2",
			Status:    chatruntimedomain.SessionStatusBusy,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
	}
	server := &Server{chatRuntimes: service}

	req := httptest.NewRequest(http.MethodPost, "/api/chat/sessions/chatRuntime-2/input", bytes.NewBufferString(`{"input":"pwd"}`))
	rec := httptest.NewRecorder()

	server.chatSessionItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if service.lastOwnerID != chatSessionOwnerID {
		t.Fatalf("expected chatRuntime owner, got %q", service.lastOwnerID)
	}
	if service.lastID != "chatRuntime-2" {
		t.Fatalf("expected session chatRuntime-2, got %q", service.lastID)
	}
	if service.lastInput != "pwd" {
		t.Fatalf("expected input pwd, got %q", service.lastInput)
	}
	events, _, _, _ := server.sessionUpdateBroker().poll(chatSessionOwnerID, 0, 10, 1024*1024)
	if len(events) != 0 {
		t.Fatalf("input handler must not publish a duplicate session.updated event; application turn.started owns runtime updates, got %+v", events)
	}
}

func TestChatRuntimeSessionItemHandlerWritesImageAttachments(t *testing.T) {
	service := &stubWebChatRuntimeService{
		inputResp: chatruntimedomain.Session{
			ID:        "chatRuntime-2",
			OwnerID:   chatSessionOwnerID,
			Title:     "chatRuntime-2",
			Status:    chatruntimedomain.SessionStatusBusy,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
	}
	server := &Server{chatRuntimes: service}

	req := httptest.NewRequest(http.MethodPost, "/api/chat/sessions/chatRuntime-2/input", bytes.NewBufferString(`{"attachments":[{"name":"diagram.png","content_type":"image/png","data_url":"data:image/png;base64,ZmFrZQ=="}]}`))
	rec := httptest.NewRecorder()

	server.chatSessionItemHandler(rec, req)

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

func TestChatRuntimeSessionItemHandlerPassesSelectedSkills(t *testing.T) {
	service := &stubWebChatRuntimeService{
		inputResp: chatruntimedomain.Session{
			ID:        "chatRuntime-2",
			OwnerID:   chatSessionOwnerID,
			Title:     "chatRuntime-2",
			Status:    chatruntimedomain.SessionStatusBusy,
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
			"skill.description": "Summarize chatRuntime work.",
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
	server := &Server{chatRuntimes: service, control: control}

	req := httptest.NewRequest(http.MethodPost, "/api/chat/sessions/chatRuntime-2/input", bytes.NewBufferString(`{"input":"summarize","skill_ids":["summary","private","missing"]}`))
	rec := httptest.NewRecorder()

	server.chatSessionItemHandler(rec, req)

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

func TestChatRuntimeSessionItemHandlerDefaultsMissingSkillIDsToAllPublicSkills(t *testing.T) {
	service := &stubWebChatRuntimeService{
		inputResp: chatruntimedomain.Session{
			ID:        "chatRuntime-2",
			OwnerID:   chatSessionOwnerID,
			Title:     "chatRuntime-2",
			Status:    chatruntimedomain.SessionStatusBusy,
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
			"skill.description": "Summarize chatRuntime work.",
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
	server := &Server{chatRuntimes: service, control: control}

	req := httptest.NewRequest(http.MethodPost, "/api/chat/sessions/chatRuntime-2/input", bytes.NewBufferString(`{"input":"summarize"}`))
	rec := httptest.NewRecorder()

	server.chatSessionItemHandler(rec, req)

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

func TestChatRuntimeSessionItemHandlerTreatsExplicitEmptySkillIDsAsNoSkills(t *testing.T) {
	service := &stubWebChatRuntimeService{
		inputResp: chatruntimedomain.Session{
			ID:        "chatRuntime-2",
			OwnerID:   chatSessionOwnerID,
			Title:     "chatRuntime-2",
			Status:    chatruntimedomain.SessionStatusBusy,
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
	server := &Server{chatRuntimes: service, control: control}

	req := httptest.NewRequest(http.MethodPost, "/api/chat/sessions/chatRuntime-2/input", bytes.NewBufferString(`{"input":"summarize","skill_ids":[]}`))
	rec := httptest.NewRecorder()

	server.chatSessionItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if service.inputReq.SkillContext != nil {
		t.Fatalf("expected explicit empty skill_ids to disable skills, got %+v", service.inputReq.SkillContext)
	}
}

func TestChatRuntimeSessionRecoverHandlerRestoresStoredSession(t *testing.T) {
	service := &stubWebChatRuntimeService{
		recoverResp: chatruntimedomain.Session{
			ID:               "chatRuntime-recover",
			OwnerID:          chatSessionOwnerID,
			Title:            "Recovered",
			RuntimeSessionID: "thread-recover",
			Status:           chatruntimedomain.SessionStatusReady,
			CreatedAt:        time.Date(2026, 3, 19, 10, 0, 0, 0, time.UTC),
			UpdatedAt:        time.Date(2026, 3, 19, 10, 5, 0, 0, time.UTC),
		},
	}
	server := &Server{chatRuntimes: service}

	req := httptest.NewRequest(http.MethodPost, "/api/chat/sessions/recover", bytes.NewBufferString(`{"id":"chatRuntime-recover","runtime_session_id":"thread-recover","title":"Recovered","created_at":"2026-03-19T10:00:00Z","updated_at":"2026-03-19T10:05:00Z"}`))
	rec := httptest.NewRecorder()

	server.chatRuntimeSessionRecoverHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if service.recoverReq.OwnerID != chatSessionOwnerID {
		t.Fatalf("expected chatRuntime owner, got %q", service.recoverReq.OwnerID)
	}
	if service.recoverReq.SessionID != "chatRuntime-recover" {
		t.Fatalf("expected recover session id, got %q", service.recoverReq.SessionID)
	}
	if service.recoverReq.RuntimeSessionID != "thread-recover" {
		t.Fatalf("expected recover thread id, got %q", service.recoverReq.RuntimeSessionID)
	}
}

func TestChatRuntimeSessionItemHandlerRejectsRemovedCloseRoute(t *testing.T) {
	service := &stubWebChatRuntimeService{}
	server := &Server{chatRuntimes: service}

	req := httptest.NewRequest(http.MethodPost, "/api/chat/sessions/chatRuntime-3/close", nil)
	rec := httptest.NewRecorder()

	server.chatSessionItemHandler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
	if service.lastID != "" {
		t.Fatalf("expected close route to bypass chatRuntime service, got %q", service.lastID)
	}
}

func TestChatRuntimeSessionItemHandlerPinsSession(t *testing.T) {
	service := &stubWebChatRuntimeService{
		pinResp: chatruntimedomain.Session{
			ID:        "chatRuntime-4",
			OwnerID:   chatSessionOwnerID,
			Title:     "chatRuntime-4",
			Status:    chatruntimedomain.SessionStatusReady,
			Pinned:    true,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
	}
	server := &Server{chatRuntimes: service}

	req := httptest.NewRequest(http.MethodPost, "/api/chat/sessions/chatRuntime-4/pin", bytes.NewBufferString(`{"pinned":true}`))
	rec := httptest.NewRecorder()

	server.chatSessionItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if service.lastOwnerID != chatSessionOwnerID {
		t.Fatalf("expected chatRuntime owner, got %q", service.lastOwnerID)
	}
	if service.lastID != "chatRuntime-4" {
		t.Fatalf("expected session chatRuntime-4, got %q", service.lastID)
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

func TestChatRuntimeSessionItemHandlerUnpinsSessionWithExplicitFalse(t *testing.T) {
	service := &stubWebChatRuntimeService{
		pinResp: chatruntimedomain.Session{
			ID:        "chatRuntime-4",
			OwnerID:   chatSessionOwnerID,
			Title:     "chatRuntime-4",
			Status:    chatruntimedomain.SessionStatusReady,
			Pinned:    false,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
	}
	server := &Server{chatRuntimes: service}

	req := httptest.NewRequest(http.MethodPost, "/api/chat/sessions/chatRuntime-4/pin", bytes.NewBufferString(`{"pinned":false}`))
	rec := httptest.NewRecorder()

	server.chatSessionItemHandler(rec, req)

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

func TestChatRuntimeSessionItemHandlerDeletesSession(t *testing.T) {
	service := &stubWebChatRuntimeService{
		deleteResp: chatruntimedomain.Session{
			ID:         "chatRuntime-4",
			OwnerID:    chatSessionOwnerID,
			Title:      "chatRuntime-4",
			Status:     chatruntimedomain.SessionStatusExited,
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
			FinishedAt: time.Now().UTC(),
		},
	}
	server := &Server{chatRuntimes: service}

	req := httptest.NewRequest(http.MethodDelete, "/api/chat/sessions/chatRuntime-4", nil)
	rec := httptest.NewRecorder()

	server.chatSessionItemHandler(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rec.Code)
	}
	if service.lastOwnerID != chatSessionOwnerID {
		t.Fatalf("expected chatRuntime owner, got %q", service.lastOwnerID)
	}
	if service.lastID != "chatRuntime-4" {
		t.Fatalf("expected session chatRuntime-4, got %q", service.lastID)
	}
	if body := strings.TrimSpace(rec.Body.String()); body != "" {
		t.Fatalf("expected empty response body, got %q", body)
	}
}

func TestChatRuntimeSessionItemHandlerReturnsTurnsInSessionDetail(t *testing.T) {
	service := &stubWebChatRuntimeService{
		getResp: chatruntimedomain.Session{
			ID:        "chatRuntime-4",
			OwnerID:   chatSessionOwnerID,
			Title:     "chatRuntime-4",
			Status:    chatruntimedomain.SessionStatusReady,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
		getOK: true,
		turnsResp: []chatruntimeapp.TurnSummary{{
			ID:          "turn-1",
			Prompt:      "pwd",
			Status:      "completed",
			FinalOutput: "/workspace/alter0",
		}},
	}
	server := &Server{chatRuntimes: service}

	req := httptest.NewRequest(http.MethodGet, "/api/chat/sessions/chatRuntime-4", nil)
	rec := httptest.NewRecorder()

	server.chatSessionItemHandler(rec, req)

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

func TestChatRuntimeSessionItemHandlerPagesTurnsInSessionDetail(t *testing.T) {
	service := &stubWebChatRuntimeService{
		getResp: chatruntimedomain.Session{
			ID:      "chatRuntime-4",
			OwnerID: chatSessionOwnerID,
			Status:  chatruntimedomain.SessionStatusReady,
		},
		getOK: true,
		turnsResp: []chatruntimeapp.TurnSummary{
			{ID: "turn-1", Prompt: "one", Status: "completed", FinalOutput: "1"},
			{ID: "turn-2", Prompt: "two", Status: "completed", FinalOutput: "2"},
			{ID: "turn-3", Prompt: "three", Status: "completed", FinalOutput: "3"},
			{ID: "turn-4", Prompt: "four", Status: "completed", FinalOutput: "4"},
		},
	}
	server := &Server{chatRuntimes: service}

	req := httptest.NewRequest(http.MethodGet, "/api/chat/sessions/chatRuntime-4?turn_limit=2", nil)
	rec := httptest.NewRecorder()

	server.chatSessionItemHandler(rec, req)

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
	if turns[0].(map[string]any)["id"] != float64(3) || turns[1].(map[string]any)["id"] != float64(4) {
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

func TestChatRuntimeSessionItemHandlerUsesCompactDefaultTurnPage(t *testing.T) {
	turns := make([]chatruntimeapp.TurnSummary, 45)
	for index := range turns {
		turnNumber := index + 1
		turns[index] = chatruntimeapp.TurnSummary{
			ID:          fmt.Sprintf("turn-%02d", turnNumber),
			Prompt:      fmt.Sprintf("prompt-%02d", turnNumber),
			Status:      "completed",
			FinalOutput: fmt.Sprintf("output-%02d", turnNumber),
		}
	}
	service := &stubWebChatRuntimeService{
		getResp: chatruntimedomain.Session{
			ID:      "chatRuntime-4",
			OwnerID: chatSessionOwnerID,
			Status:  chatruntimedomain.SessionStatusReady,
		},
		getOK:     true,
		turnsResp: turns,
	}
	server := &Server{chatRuntimes: service}

	req := httptest.NewRequest(http.MethodGet, "/api/chat/sessions/chatRuntime-4", nil)
	rec := httptest.NewRecorder()

	server.chatSessionItemHandler(rec, req)

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
	if page[0].(map[string]any)["id"] != float64(26) || page[len(page)-1].(map[string]any)["id"] != float64(45) {
		t.Fatalf("expected latest compact turn page, got first=%v last=%v", page[0], page[len(page)-1])
	}
	paging := session["turns_paging"].(map[string]any)
	if paging["limit"] != float64(20) || paging["has_more_before"] != true {
		t.Fatalf("expected default paging metadata for compact page, got %v", paging)
	}
}

func TestChatRuntimeSessionItemHandlerCapsTurnPageByApproximatePayloadSize(t *testing.T) {
	largeOutput := strings.Repeat("large chatRuntime output\n", 25000)
	turns := make([]chatruntimeapp.TurnSummary, 5)
	for index := range turns {
		turnNumber := index + 1
		turns[index] = chatruntimeapp.TurnSummary{
			ID:          fmt.Sprintf("turn-%02d", turnNumber),
			Prompt:      fmt.Sprintf("prompt-%02d", turnNumber),
			Status:      "completed",
			FinalOutput: largeOutput,
		}
	}
	service := &stubWebChatRuntimeService{
		getResp: chatruntimedomain.Session{
			ID:      "chatRuntime-4",
			OwnerID: chatSessionOwnerID,
			Status:  chatruntimedomain.SessionStatusReady,
		},
		getOK:     true,
		turnsResp: turns,
	}
	server := &Server{chatRuntimes: service}

	req := httptest.NewRequest(http.MethodGet, "/api/chat/sessions/chatRuntime-4?turn_limit=5", nil)
	rec := httptest.NewRecorder()

	server.chatSessionItemHandler(rec, req)

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
	if page[len(page)-1].(map[string]any)["id"] != float64(5) {
		t.Fatalf("expected capped page to keep the latest turn, got %v", page)
	}
	paging := session["turns_paging"].(map[string]any)
	if paging["has_more_before"] != true {
		t.Fatalf("expected payload cap to keep earlier turns available, got %v", paging)
	}
}

func TestChatRuntimeSessionItemHandlerDefaultTurnPageKeepsRecentLargeContext(t *testing.T) {
	largeOutput := strings.Repeat("large chatRuntime output\n", 16000)
	turns := []chatruntimeapp.TurnSummary{
		{ID: "turn-1", Prompt: "one", Status: "completed", FinalOutput: largeOutput},
		{ID: "turn-2", Prompt: "two", Status: "completed", FinalOutput: largeOutput},
		{ID: "turn-3", Prompt: "three", Status: "completed", FinalOutput: largeOutput},
	}
	service := &stubWebChatRuntimeService{
		getResp: chatruntimedomain.Session{
			ID:      "chatRuntime-4",
			OwnerID: chatSessionOwnerID,
			Status:  chatruntimedomain.SessionStatusReady,
		},
		getOK:     true,
		turnsResp: turns,
	}
	server := &Server{chatRuntimes: service}

	req := httptest.NewRequest(http.MethodGet, "/api/chat/sessions/chatRuntime-4", nil)
	rec := httptest.NewRecorder()

	server.chatSessionItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	session := payload["session"].(map[string]any)
	page := session["turns"].([]any)
	if len(page) != 2 {
		t.Fatalf("expected default detail page to keep two recent large turns, got %d", len(page))
	}
	if page[0].(map[string]any)["id"] != float64(2) || page[1].(map[string]any)["id"] != float64(3) {
		t.Fatalf("expected latest two large turns, got %v", page)
	}
	paging := session["turns_paging"].(map[string]any)
	if paging["byte_limit"] != float64(1048576) || paging["has_more_before"] != true {
		t.Fatalf("expected one MiB paging budget with earlier turns available, got %v", paging)
	}
}

func TestChatRuntimeSessionItemHandlerBudgetsFinalTurnDTOInsteadOfHiddenEventBlocks(t *testing.T) {
	hiddenBlock := strings.Repeat("hidden runtime detail block\n", 40000)
	turns := make([]chatruntimeapp.TurnSummary, 3)
	for index := range turns {
		turnNumber := index + 1
		turnID := fmt.Sprintf("turn-%d", turnNumber)
		turns[index] = chatruntimeapp.TurnSummary{
			ID:          turnID,
			Prompt:      fmt.Sprintf("prompt-%d", turnNumber),
			Status:      "completed",
			FinalOutput: fmt.Sprintf("visible output %d", turnNumber),
			RuntimeTraceEvents: []chatruntimeapp.RuntimeTraceEvent{{
				ID:     "event-1",
				TurnID: turnID,
				Seq:    1,
				Kind:   "commands",
				Status: "completed",
				Summary: "visible process summary",
				Blocks: []chatruntimeapp.RuntimeBlock{{
					Type: "text",
					Text: hiddenBlock,
				}},
				Raw: chatruntimeapp.RuntimeTraceEventRaw{HasDetail: true},
			}},
		}
	}
	service := &stubWebChatRuntimeService{
		getResp: chatruntimedomain.Session{
			ID:      "chatRuntime-4",
			OwnerID: chatSessionOwnerID,
			Status:  chatruntimedomain.SessionStatusReady,
		},
		getOK:     true,
		turnsResp: turns,
	}
	server := &Server{chatRuntimes: service}

	req := httptest.NewRequest(http.MethodGet, "/api/chat/sessions/chatRuntime-4", nil)
	rec := httptest.NewRecorder()

	server.chatSessionItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	session := payload["session"].(map[string]any)
	page := session["turns"].([]any)
	if len(page) != 3 {
		t.Fatalf("expected hidden event detail blocks not to cap visible turn page, got %d", len(page))
	}
	if rec.Body.Len() > 20000 {
		t.Fatalf("expected response DTO to omit hidden blocks, got %d bytes", rec.Body.Len())
	}
	paging := session["turns_paging"].(map[string]any)
	if paging["has_more_before"] != false {
		t.Fatalf("expected all visible turns in page, got %v", paging)
	}
}

func TestChatRuntimeSessionItemHandlerReturnsRuntimeTraceEventDetail(t *testing.T) {
	service := &stubWebChatRuntimeService{
		eventResp: chatruntimeapp.RuntimeTraceEventDetail{
			TurnID: "turn-1",
			Event: chatruntimeapp.RuntimeTraceEvent{
				ID:         "event-1",
				TurnID:     "turn-1",
				Seq:        1,
				Source:     "adapter",
				Provider:   chatruntimeapp.RuntimeProviderRef{Engine: "codex", Adapter: "codex_cli_json"},
				Role:       "assistant",
				Kind:       "shell_command",
				Lifecycle:  "completed",
				Status:     "completed",
				Title:      "pwd",
				Blocks:     []chatruntimeapp.RuntimeBlock{},
				Visibility: "collapsed",
			},
			Blocks: []chatruntimeapp.RuntimeBlock{{
				Type:    "chatRuntime",
				Title:   "Shell",
				Command: "pwd",
				Output:  "/workspace/alter0",
			}},
			Searchable: true,
		},
	}
	server := &Server{chatRuntimes: service}

	req := httptest.NewRequest(http.MethodGet, "/api/chat/sessions/chatRuntime-4/turns/turn-1/events/event-1", nil)
	rec := httptest.NewRecorder()

	server.chatSessionItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if service.lastID != "chatRuntime-4:turn-1:event-1" {
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
	if event["id"] != float64(1) || event["kind"] != "commands" || event["status"] != "completed" || event["text"] != "pwd" {
		t.Fatalf("expected lightweight runtime event detail, got %v", event)
	}
	if _, hasTurnID := event["turn_id"]; hasTurnID {
		t.Fatalf("event detail must omit nested turn_id, got %v", event)
	}
	assertNoLegacyChatRuntimeAPIFields(t, rec.Body.String())
}

package web

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	execdomain "alter0/internal/execution/domain"
	sessionapp "alter0/internal/session/application"
	sessiondomain "alter0/internal/session/domain"
	shareddomain "alter0/internal/shared/domain"
	taskapp "alter0/internal/task/application"
	taskdomain "alter0/internal/task/domain"
	terminaldomain "alter0/internal/terminal/domain"
)

type stubSessionHistory struct {
	sessionPage       sessionapp.SessionPage
	messagePage       sessionapp.MessagePage
	lastSessionQuery  sessionapp.SessionQuery
	lastMessageQuery  sessionapp.MessageQuery
	deleteErr         error
	lastDeletedID     string
	lastPinnedID      string
	lastPinnedValue   bool
	pinErr            error
	lastTouchedID     string
	cleanupResult     sessionapp.CleanupInactiveSessionsResult
	lastCleanupOption sessionapp.CleanupInactiveSessionsOptions
}

func (s *stubSessionHistory) ListSessions(query sessionapp.SessionQuery) sessionapp.SessionPage {
	s.lastSessionQuery = query
	return s.sessionPage
}

func (s *stubSessionHistory) ListMessages(query sessionapp.MessageQuery) sessionapp.MessagePage {
	s.lastMessageQuery = query
	return s.messagePage
}

func (s *stubSessionHistory) DeleteSession(sessionID string) error {
	s.lastDeletedID = sessionID
	return s.deleteErr
}

func (s *stubSessionHistory) SetSessionPinned(sessionID string, pinned bool) error {
	s.lastPinnedID = sessionID
	s.lastPinnedValue = pinned
	return s.pinErr
}

func (s *stubSessionHistory) TouchSession(sessionID string, at time.Time) error {
	s.lastTouchedID = sessionID
	return nil
}

func (s *stubSessionHistory) CleanupInactiveSessions(options sessionapp.CleanupInactiveSessionsOptions) (sessionapp.CleanupInactiveSessionsResult, error) {
	s.lastCleanupOption = options
	return s.cleanupResult, nil
}

type stubSessionTaskService struct {
	lastDeletedSessionID string
	deleteErr            error
	items                []taskdomain.Task
}

func (s *stubSessionTaskService) List(query taskapp.ListQuery) taskapp.TaskPage {
	items := make([]taskdomain.Task, 0, len(s.items))
	for _, item := range s.items {
		if strings.TrimSpace(query.SessionID) != "" && item.SessionID != query.SessionID {
			continue
		}
		if strings.TrimSpace(string(query.Status)) != "" && item.Status != query.Status {
			continue
		}
		items = append(items, item)
	}
	return taskapp.TaskPage{
		Items: items,
		Pagination: taskapp.Pagination{
			Page:     1,
			PageSize: len(items),
			Total:    len(items),
		},
	}
}

func (s *stubSessionTaskService) Get(string) (taskdomain.Task, bool) {
	return taskdomain.Task{}, false
}

func (s *stubSessionTaskService) ListBySession(string) []taskdomain.Task {
	return append([]taskdomain.Task(nil), s.items...)
}

func (s *stubSessionTaskService) ListLogs(string, int, int) (taskapp.TaskLogPage, error) {
	return taskapp.TaskLogPage{}, nil
}

func (s *stubSessionTaskService) ListArtifacts(string) ([]taskdomain.TaskArtifact, error) {
	return []taskdomain.TaskArtifact{}, nil
}

func (s *stubSessionTaskService) DeleteBySession(sessionID string) error {
	s.lastDeletedSessionID = sessionID
	return s.deleteErr
}

func TestSessionListHandlerReturnsPagedData(t *testing.T) {
	history := &stubSessionHistory{
		sessionPage: sessionapp.SessionPage{
			Items: []sessiondomain.SessionSummary{
				{
					SessionID:    "s-1",
					MessageCount: 2,
				},
			},
			Pagination: sessionapp.Pagination{
				Page:     2,
				PageSize: 10,
				Total:    21,
				HasNext:  true,
			},
		},
	}
	server := &Server{
		sessions: history,
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/sessions?page=2&page_size=10&trigger_type=cron&channel_type=scheduler&channel_id=scheduler-default&message_id=msg-1&job_id=job-daily",
		nil,
	)
	rec := httptest.NewRecorder()
	server.sessionListHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if history.lastSessionQuery.Page != 2 || history.lastSessionQuery.PageSize != 10 {
		t.Fatalf("unexpected query %+v", history.lastSessionQuery)
	}
	if history.lastSessionQuery.TriggerType != shareddomain.TriggerTypeCron {
		t.Fatalf("expected trigger_type cron, got %s", history.lastSessionQuery.TriggerType)
	}
	if history.lastSessionQuery.ChannelType != shareddomain.ChannelTypeScheduler {
		t.Fatalf("expected channel_type scheduler, got %s", history.lastSessionQuery.ChannelType)
	}
	if history.lastSessionQuery.ChannelID != "scheduler-default" {
		t.Fatalf("expected channel_id scheduler-default, got %s", history.lastSessionQuery.ChannelID)
	}
	if history.lastSessionQuery.MessageID != "msg-1" {
		t.Fatalf("expected message_id msg-1, got %s", history.lastSessionQuery.MessageID)
	}
	if history.lastSessionQuery.JobID != "job-daily" {
		t.Fatalf("expected job_id job-daily, got %s", history.lastSessionQuery.JobID)
	}

	var body sessionapp.SessionPage
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body failed: %v", err)
	}
	if len(body.Items) != 1 || body.Items[0].SessionID != "s-1" {
		t.Fatalf("unexpected body %+v", body)
	}
}

func TestSessionMessageListHandlerSupportsTimeRange(t *testing.T) {
	history := &stubSessionHistory{
		messagePage: sessionapp.MessagePage{
			Items: []sessiondomain.MessageRecord{
				{
					MessageID: "m-1",
					SessionID: "s-1",
					Role:      sessiondomain.MessageRoleAssistant,
					Content:   "answer",
					Timestamp: time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC),
					RouteResult: sessiondomain.RouteResult{
						Route: shareddomain.RouteAgent,
					},
				},
			},
			Pagination: sessionapp.Pagination{
				Page:     1,
				PageSize: 20,
				Total:    1,
				HasNext:  false,
			},
		},
	}
	server := &Server{
		sessions: history,
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/sessions/s-1/messages?page=1&page_size=20&start_at=2026-03-03T00:00:00Z&end_at=2026-03-03T23:59:59Z",
		nil,
	)
	rec := httptest.NewRecorder()
	server.sessionMessageListHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if history.lastMessageQuery.SessionID != "s-1" {
		t.Fatalf("expected session id s-1, got %q", history.lastMessageQuery.SessionID)
	}
	if history.lastMessageQuery.StartAt.IsZero() || history.lastMessageQuery.EndAt.IsZero() {
		t.Fatalf("expected non-zero time range, got %+v", history.lastMessageQuery)
	}
}

func TestSessionHandlersValidateInputs(t *testing.T) {
	history := &stubSessionHistory{}
	server := &Server{
		sessions: history,
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	invalidPageReq := httptest.NewRequest(http.MethodGet, "/api/sessions?page=0", nil)
	invalidPageRec := httptest.NewRecorder()
	server.sessionListHandler(invalidPageRec, invalidPageReq)
	if invalidPageRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, invalidPageRec.Code)
	}

	invalidPathReq := httptest.NewRequest(http.MethodGet, "/api/sessions/s-1/records", nil)
	invalidPathRec := httptest.NewRecorder()
	server.sessionMessageListHandler(invalidPathRec, invalidPathReq)
	if invalidPathRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, invalidPathRec.Code)
	}

	invalidTriggerReq := httptest.NewRequest(http.MethodGet, "/api/sessions?trigger_type=timer", nil)
	invalidTriggerRec := httptest.NewRecorder()
	server.sessionListHandler(invalidTriggerRec, invalidTriggerReq)
	if invalidTriggerRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, invalidTriggerRec.Code)
	}

	invalidChannelReq := httptest.NewRequest(http.MethodGet, "/api/sessions?channel_type=mobile", nil)
	invalidChannelRec := httptest.NewRecorder()
	server.sessionListHandler(invalidChannelRec, invalidChannelReq)
	if invalidChannelRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, invalidChannelRec.Code)
	}
}

func TestSessionDeleteHandlerRemovesHistoryTasksAndWorkspace(t *testing.T) {
	baseDir := t.TempDir()
	workspaceDir := filepath.Join(baseDir, ".alter0", "workspaces", "sessions", "session-delete")
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		t.Fatalf("prepare workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspaceDir, "artifact.txt"), []byte("payload"), 0o644); err != nil {
		t.Fatalf("write workspace file: %v", err)
	}

	history := &stubSessionHistory{}
	tasks := &stubSessionTaskService{}
	server := &Server{
		sessions:      history,
		tasks:         tasks,
		logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		workspaceRoot: baseDir,
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/sessions/session-delete", nil)
	rec := httptest.NewRecorder()
	server.sessionMessageListHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if history.lastDeletedID != "session-delete" {
		t.Fatalf("expected history delete for session-delete, got %q", history.lastDeletedID)
	}
	if tasks.lastDeletedSessionID != "session-delete" {
		t.Fatalf("expected task delete for session-delete, got %q", tasks.lastDeletedSessionID)
	}
	if _, err := os.Stat(workspaceDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected workspace removed, got %v", err)
	}
}

func TestSessionPinHandlerUpdatesPinnedState(t *testing.T) {
	history := &stubSessionHistory{}
	server := &Server{
		sessions: history,
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/session-pin/pin", strings.NewReader(`{"pinned":true}`))
	rec := httptest.NewRecorder()
	server.sessionMessageListHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if history.lastPinnedID != "session-pin" || !history.lastPinnedValue {
		t.Fatalf("expected pinned session-pin=true, got id=%q value=%v", history.lastPinnedID, history.lastPinnedValue)
	}
}

func TestSessionPinHandlerUpdatesConversationRuntimeRegistryWhenHistoryIsMissing(t *testing.T) {
	registry, err := newFileConversationRuntimeSessionRegistry(filepath.Join(t.TempDir(), "conversation-runtime.json"))
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}
	if _, err := registry.Upsert(conversationRuntimeSessionRegistryEntry{
		SessionID:  "blank-chat",
		Route:      conversationRuntimeRouteChat,
		Status:     conversationRuntimeSessionStatusReady,
		Title:      "New",
		TitleAuto:  true,
		CreatedAt:  time.Date(2026, 4, 23, 9, 0, 0, 0, time.UTC),
		TargetType: "model",
		TargetID:   "raw-model",
		TargetName: "Raw Model",
	}); err != nil {
		t.Fatalf("upsert registry entry: %v", err)
	}

	history := &stubSessionHistory{pinErr: sessionapp.ErrSessionNotFound}
	server := &Server{
		sessions:                    history,
		conversationRuntimeSessions: registry,
		logger:                      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/blank-chat/pin", strings.NewReader(`{"pinned":true}`))
	rec := httptest.NewRecorder()
	server.sessionMessageListHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if history.lastPinnedID != "blank-chat" || !history.lastPinnedValue {
		t.Fatalf("expected history pin attempt blank-chat=true, got id=%q value=%v", history.lastPinnedID, history.lastPinnedValue)
	}
	entry, ok := registry.Resolve(conversationRuntimeRouteChat, "blank-chat")
	if !ok {
		t.Fatalf("expected registry entry")
	}
	if !entry.Pinned {
		t.Fatalf("expected registry entry to be pinned")
	}
}

func TestSessionCleanupHandlerDeletesInactiveSessionsAndWorkspaces(t *testing.T) {
	baseDir := t.TempDir()
	for _, sessionID := range []string{"old-a", "old-b"} {
		workspaceDir := filepath.Join(baseDir, ".alter0", "workspaces", "sessions", sessionID)
		if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
			t.Fatalf("prepare workspace: %v", err)
		}
	}
	history := &stubSessionHistory{
		cleanupResult: sessionapp.CleanupInactiveSessionsResult{
			DeletedSessionIDs:  []string{"old-a", "old-b"},
			DeletedCount:       2,
			SkippedPinnedCount: 1,
			ScannedCount:       4,
		},
	}
	tasks := &stubSessionTaskService{}
	server := &Server{
		sessions:      history,
		tasks:         tasks,
		logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		workspaceRoot: baseDir,
	}
	server.ensureMaintenanceService()

	body := server.maintenance.RunSessionCleanup(time.Now().UTC())
	if history.lastCleanupOption.InactiveDuration != 7*24*time.Hour {
		t.Fatalf("expected fixed 7 day inactive cleanup, got %+v", history.lastCleanupOption)
	}
	for _, sessionID := range []string{"old-a", "old-b"} {
		if _, err := os.Stat(filepath.Join(baseDir, ".alter0", "workspaces", "sessions", sessionID)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected workspace %s removed, got %v", sessionID, err)
		}
	}
	if body.DeletedCount != 2 || body.SkippedPinnedCount != 1 {
		t.Fatalf("unexpected cleanup body %+v", body)
	}
}

func TestSessionCleanupHandlerDeletesInactiveTerminalSessions(t *testing.T) {
	now := time.Date(2026, 4, 20, 9, 0, 0, 0, time.UTC)
	old := now.Add(-8 * 24 * time.Hour)
	recent := now.Add(-2 * time.Hour)
	history := &stubSessionHistory{}
	terminals := &stubWebTerminalService{
		listResp: []terminaldomain.Session{
			{
				ID:           "terminal-old",
				OwnerID:      sharedTerminalClientID,
				Status:       terminaldomain.SessionStatusReady,
				CreatedAt:    old,
				LastOutputAt: old,
				UpdatedAt:    old,
			},
			{
				ID:           "terminal-pinned",
				OwnerID:      sharedTerminalClientID,
				Status:       terminaldomain.SessionStatusReady,
				Pinned:       true,
				CreatedAt:    old,
				LastOutputAt: old,
				UpdatedAt:    old,
			},
			{
				ID:           "terminal-busy",
				OwnerID:      sharedTerminalClientID,
				Status:       terminaldomain.SessionStatusBusy,
				CreatedAt:    old,
				LastOutputAt: old,
				UpdatedAt:    old,
			},
			{
				ID:           "terminal-recent",
				OwnerID:      sharedTerminalClientID,
				Status:       terminaldomain.SessionStatusReady,
				CreatedAt:    old,
				LastOutputAt: recent,
				UpdatedAt:    recent,
			},
		},
	}
	server := &Server{
		sessions:  history,
		terminals: terminals,
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	server.ensureMaintenanceService()

	body := server.maintenance.RunSessionCleanup(now)

	if terminals.lastOwnerID != sharedTerminalClientID {
		t.Fatalf("expected shared terminal owner, got %q", terminals.lastOwnerID)
	}
	if len(terminals.deleteIDs) != 1 || terminals.deleteIDs[0] != "terminal-old" {
		t.Fatalf("expected only old terminal deleted, got %+v", terminals.deleteIDs)
	}
	if body.DeletedCount != 1 || body.SkippedPinnedCount != 1 || body.SkippedProtectedCount != 1 || body.ScannedCount != 4 {
		t.Fatalf("expected combined terminal cleanup counts, got %+v", body)
	}
	if body.TerminalDeletedCount != 1 || body.TerminalSkippedPinnedCount != 1 || body.TerminalSkippedProtectedCount != 1 || body.TerminalScannedCount != 4 {
		t.Fatalf("expected terminal cleanup counts, got %+v", body)
	}
	if history.lastCleanupOption.InactiveDuration != 7*24*time.Hour {
		t.Fatalf("expected session cleanup still invoked with fixed threshold, got %+v", history.lastCleanupOption)
	}
}

func TestMaintenanceMemoryRunReportsUnavailableOrchestrator(t *testing.T) {
	server := &Server{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	server.ensureMaintenanceService()

	body := server.maintenance.RunMemoryMaintenance(context.Background(), time.Now().UTC())
	if body.Status != "failed" || !strings.Contains(body.Error, "memory maintenance unavailable") {
		t.Fatalf("expected unavailable memory maintenance failure, got %+v", body)
	}
}

func TestMaintenanceMemoryRunUsesStructuredSummaryPrompt(t *testing.T) {
	orchestrator := &stubOrchestrator{}
	server := &Server{
		orchestrator: orchestrator,
		logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	server.ensureMaintenanceService()

	body := server.maintenance.RunMemoryMaintenance(context.Background(), time.Now().UTC())
	if body.Status != "success" {
		t.Fatalf("expected successful memory maintenance, got %+v", body)
	}
	content := orchestrator.last.Content
	for _, required := range []string{
		"Summarize daily memory into durable memory candidates",
		"Do not copy raw transcript text",
		"Return changed files and skipped candidates",
	} {
		if !strings.Contains(content, required) {
			t.Fatalf("expected maintenance prompt to contain %q, got %q", required, content)
		}
	}
	if orchestrator.last.Metadata["alter0.skills.include"] != `["memory-maintenance"]` {
		t.Fatalf("expected memory-maintenance skill, got %+v", orchestrator.last.Metadata)
	}
	if !strings.Contains(orchestrator.last.Metadata["alter0.memory.include"], "memory_daily_today") {
		t.Fatalf("expected daily memory context, got %+v", orchestrator.last.Metadata)
	}
}

func TestSessionCleanupHandlerReturnsTaskDeleteFailure(t *testing.T) {
	history := &stubSessionHistory{
		cleanupResult: sessionapp.CleanupInactiveSessionsResult{
			DeletedSessionIDs: []string{"old-a"},
			DeletedCount:      1,
			ScannedCount:      1,
		},
	}
	tasks := &stubSessionTaskService{deleteErr: errors.New("task delete failed")}
	server := &Server{
		sessions:      history,
		tasks:         tasks,
		logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		workspaceRoot: t.TempDir(),
	}
	server.ensureMaintenanceService()

	body := server.maintenance.RunSessionCleanup(time.Now().UTC())
	if body.Status != "failed" || !strings.Contains(body.Error, "task delete failed") {
		t.Fatalf("expected task delete failure, got %+v", body)
	}
}

func TestSessionCleanupHandlerProtectsSessionsWithActiveTasks(t *testing.T) {
	history := &stubSessionHistory{}
	tasks := &stubSessionTaskService{
		items: []taskdomain.Task{
			{ID: "task-queued", SessionID: "old-queued", Status: taskdomain.TaskStatusQueued},
			{ID: "task-running", SessionID: "old-running", Status: taskdomain.TaskStatusRunning},
			{ID: "task-success", SessionID: "old-success", Status: taskdomain.TaskStatusSuccess},
		},
	}
	server := &Server{
		sessions: history,
		tasks:    tasks,
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	server.ensureMaintenanceService()

	body := server.maintenance.RunSessionCleanup(time.Now().UTC())
	if body.Status != "success" {
		t.Fatalf("expected successful cleanup, got %+v", body)
	}
	protected := map[string]bool{}
	for _, sessionID := range history.lastCleanupOption.ProtectedSessionIDs {
		protected[sessionID] = true
	}
	if !protected["old-queued"] || !protected["old-running"] {
		t.Fatalf("expected queued and running task sessions protected, got %+v", history.lastCleanupOption.ProtectedSessionIDs)
	}
	if protected["old-success"] {
		t.Fatalf("expected terminal task session unprotected, got %+v", history.lastCleanupOption.ProtectedSessionIDs)
	}
}

func TestConversationRuntimeSessionCollectionHandlerFiltersByRoute(t *testing.T) {
	service := sessionapp.NewService()
	base := time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC)
	if err := service.Append(
		sessiondomain.MessageRecord{
			MessageID: "chat-user",
			SessionID: "chat-session",
			Role:      sessiondomain.MessageRoleUser,
			Content:   "Inspect this repository",
			Timestamp: base,
			Source: sessiondomain.MessageSource{
				TriggerType: shareddomain.TriggerTypeUser,
				ChannelType: shareddomain.ChannelTypeWeb,
				ChannelID:   "web-default",
			},
			Metadata: map[string]string{
				"alter0.llm.provider_id":    "openai",
				"alter0.llm.model":          "gpt-5.4",
				"alter0.skills.include":     `["frontend-design"]`,
				"alter0.mcp.request.enable": `["filesystem"]`,
			},
		},
		sessiondomain.MessageRecord{
			MessageID: "chat-assistant",
			SessionID: "chat-session",
			Role:      sessiondomain.MessageRoleAssistant,
			Content:   "Repository loaded.",
			Timestamp: base.Add(time.Minute),
			Source: sessiondomain.MessageSource{
				TriggerType: shareddomain.TriggerTypeUser,
				ChannelType: shareddomain.ChannelTypeWeb,
				ChannelID:   "web-default",
			},
			RouteResult: sessiondomain.RouteResult{
				Route: shareddomain.RouteAgent,
			},
		},
		sessiondomain.MessageRecord{
			MessageID: "runtime-user",
			SessionID: "runtime-session",
			Role:      sessiondomain.MessageRoleUser,
			Content:   "Ship the bug fix",
			Timestamp: base.Add(2 * time.Minute),
			Source: sessiondomain.MessageSource{
				TriggerType: shareddomain.TriggerTypeUser,
				ChannelType: shareddomain.ChannelTypeWeb,
				ChannelID:   "web-default",
			},
			Metadata: map[string]string{
				"alter0.execution.engine": "codex",
				"alter0.skills.include":   `["preview-publish"]`,
			},
		},
		sessiondomain.MessageRecord{
			MessageID: "runtime-assistant",
			SessionID: "runtime-session",
			Role:      sessiondomain.MessageRoleAssistant,
			Content:   "Patch applied.",
			Timestamp: base.Add(3 * time.Minute),
			Source: sessiondomain.MessageSource{
				TriggerType: shareddomain.TriggerTypeUser,
				ChannelType: shareddomain.ChannelTypeWeb,
				ChannelID:   "web-default",
			},
			RouteResult: sessiondomain.RouteResult{
				Route: shareddomain.RouteCommand,
			},
		},
	); err != nil {
		t.Fatalf("append records: %v", err)
	}

	server := &Server{
		sessions: service,
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/conversation-runtime/sessions?route=chat", nil)
	server.conversationRuntimeSessionCollectionHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var payload struct {
		Items []struct {
			ID              string   `json:"id"`
			Title           string   `json:"title"`
			TargetType      string   `json:"target_type"`
			TargetID        string   `json:"target_id"`
			ModelProviderID string   `json:"model_provider_id"`
			ModelID         string   `json:"model_id"`
			ToolIDs         []string `json:"tool_ids"`
			SkillIDs        []string `json:"skill_ids"`
			MCPIDs          []string `json:"mcp_ids"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if len(payload.Items) != 2 {
		t.Fatalf("expected chat runtime sessions to include migrated legacy runtime sessions, got %d", len(payload.Items))
	}
	byID := map[string]struct {
		ID              string   `json:"id"`
		Title           string   `json:"title"`
		TargetType      string   `json:"target_type"`
		TargetID        string   `json:"target_id"`
		ModelProviderID string   `json:"model_provider_id"`
		ModelID         string   `json:"model_id"`
		ToolIDs         []string `json:"tool_ids"`
		SkillIDs        []string `json:"skill_ids"`
		MCPIDs          []string `json:"mcp_ids"`
	}{}
	for _, item := range payload.Items {
		byID[item.ID] = item
	}
	chatItem := byID["chat-session"]
	if chatItem.ID != "chat-session" || chatItem.TargetType != "model" {
		t.Fatalf("unexpected chat item %+v", chatItem)
	}
	if chatItem.Title != "Inspect this repository" {
		t.Fatalf("expected title from first user message, got %+v", chatItem)
	}
	if chatItem.ModelProviderID != "openai" || chatItem.ModelID != "gpt-5.4" {
		t.Fatalf("expected provider/model from metadata, got %+v", chatItem)
	}
	if len(chatItem.ToolIDs) != 0 {
		t.Fatalf("expected no tool ids, got %+v", chatItem.ToolIDs)
	}
	if len(chatItem.SkillIDs) != 1 || chatItem.SkillIDs[0] != "frontend-design" {
		t.Fatalf("expected skill ids from metadata, got %+v", chatItem.SkillIDs)
	}
	if len(chatItem.MCPIDs) != 1 || chatItem.MCPIDs[0] != "filesystem" {
		t.Fatalf("expected mcp ids from metadata, got %+v", chatItem.MCPIDs)
	}
	runtimeItem := byID["runtime-session"]
	if runtimeItem.ID != "runtime-session" || runtimeItem.TargetType != "model" {
		t.Fatalf("unexpected chat item %+v", runtimeItem)
	}
	if runtimeItem.ModelProviderID != "alter0-codex" || runtimeItem.ModelID != "codex" {
		t.Fatalf("expected codex runtime model selection, got %+v", runtimeItem)
	}
}

func TestConversationRuntimeSessionItemHandlerReturnsMessagesAndAttachments(t *testing.T) {
	service := sessionapp.NewService()
	base := time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC)
	rawAttachments, err := execdomain.EncodeUserAttachments([]execdomain.UserAttachment{
		{
			ID:          "asset-1",
			Kind:        execdomain.UserAttachmentKindImage,
			Name:        "diagram.png",
			ContentType: "image/png",
			AssetURL:    "/api/sessions/chat-session/attachments/asset-1/original",
			PreviewURL:  "/api/sessions/chat-session/attachments/asset-1/preview",
		},
	})
	if err != nil {
		t.Fatalf("encode attachments: %v", err)
	}
	if err := service.Append(
		sessiondomain.MessageRecord{
			MessageID: "chat-user",
			SessionID: "chat-session",
			Role:      sessiondomain.MessageRoleUser,
			Content:   "Inspect this repository",
			Timestamp: base,
			Source: sessiondomain.MessageSource{
				TriggerType: shareddomain.TriggerTypeUser,
				ChannelType: shareddomain.ChannelTypeWeb,
				ChannelID:   "web-default",
			},
			Metadata: map[string]string{
				execdomain.UserAttachmentsMetadataKey: rawAttachments,
			},
		},
		sessiondomain.MessageRecord{
			MessageID: "chat-assistant",
			SessionID: "chat-session",
			Role:      sessiondomain.MessageRoleAssistant,
			Content:   "Repository loaded.",
			Timestamp: base.Add(time.Minute),
			Source: sessiondomain.MessageSource{
				TriggerType: shareddomain.TriggerTypeUser,
				ChannelType: shareddomain.ChannelTypeWeb,
				ChannelID:   "web-default",
			},
			RouteResult: sessiondomain.RouteResult{
				Route:     shareddomain.RouteAgent,
				ErrorCode: "",
				ProcessSteps: []shareddomain.ProcessStep{
					{
						ID:     "step-1",
						Kind:   "action",
						Title:  "codex_exec",
						Detail: "Checked git status",
						Status: "completed",
					},
				},
			},
		},
	); err != nil {
		t.Fatalf("append records: %v", err)
	}

	server := &Server{
		sessions: service,
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/conversation-runtime/sessions/chat-session?route=chat", nil)
	server.conversationRuntimeSessionItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var payload struct {
		Session struct {
			ID       string `json:"id"`
			Messages []struct {
				ID           string `json:"id"`
				Role         string `json:"role"`
				Text         string `json:"text"`
				Status       string `json:"status"`
				Error        bool   `json:"error"`
				ProcessSteps []struct {
					Title string `json:"title"`
				} `json:"process_steps"`
				Attachments []struct {
					ID         string `json:"id"`
					AssetURL   string `json:"asset_url"`
					PreviewURL string `json:"preview_url"`
				} `json:"attachments"`
			} `json:"messages"`
		} `json:"session"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Session.ID != "chat-session" {
		t.Fatalf("unexpected session id %q", payload.Session.ID)
	}
	if len(payload.Session.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(payload.Session.Messages))
	}
	if payload.Session.Messages[0].Attachments[0].PreviewURL != "/api/sessions/chat-session/attachments/asset-1/preview" {
		t.Fatalf("expected user attachment preview url, got %+v", payload.Session.Messages[0].Attachments)
	}
	if payload.Session.Messages[1].Status != "done" || payload.Session.Messages[1].Error {
		t.Fatalf("expected assistant message to restore as done, got %+v", payload.Session.Messages[1])
	}
	if len(payload.Session.Messages[1].ProcessSteps) != 1 || payload.Session.Messages[1].ProcessSteps[0].Title != "codex_exec" {
		t.Fatalf("expected process steps, got %+v", payload.Session.Messages[1].ProcessSteps)
	}
}

func TestConversationRuntimeSessionItemHandlerFallsBackToRegistryWhileHistoryIsPending(t *testing.T) {
	registry, err := newFileConversationRuntimeSessionRegistry(filepath.Join(t.TempDir(), "conversation-runtime.json"))
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	server := &Server{
		sessions:                    sessionapp.NewService(),
		conversationRuntimeSessions: registry,
		logger:                      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	msg := shareddomain.UnifiedMessage{
		SessionID:  "chat-pending-registry",
		Content:    "Inspect this codebase",
		ReceivedAt: time.Date(2026, 5, 6, 8, 5, 0, 0, time.UTC),
		Metadata: map[string]string{
			"alter0.llm.provider_id": "openai",
			"alter0.llm.model":       "gpt-5.4",
		},
	}
	server.markConversationRuntimeSessionStarted(conversationRuntimeRouteChat, msg)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/conversation-runtime/sessions/chat-pending-registry?route=chat", nil)
	server.conversationRuntimeSessionItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var payload struct {
		Session struct {
			ID              string `json:"id"`
			Status          string `json:"status"`
			Title           string `json:"title"`
			TargetType      string `json:"target_type"`
			ModelProviderID string `json:"model_provider_id"`
			ModelID         string `json:"model_id"`
			Messages        []any  `json:"messages"`
		} `json:"session"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Session.ID != "chat-pending-registry" || payload.Session.Status != conversationRuntimeSessionStatusBusy {
		t.Fatalf("unexpected session payload %+v", payload.Session)
	}
	if payload.Session.TargetType != "model" || payload.Session.ModelProviderID != "openai" || payload.Session.ModelID != "gpt-5.4" {
		t.Fatalf("expected model metadata from registry, got %+v", payload.Session)
	}
	if len(payload.Session.Messages) != 0 {
		t.Fatalf("expected no messages while history is pending, got %+v", payload.Session.Messages)
	}
}

func TestConversationRuntimeSessionItemHandlerPatchesSessionConfiguration(t *testing.T) {
	registry, err := newFileConversationRuntimeSessionRegistry(filepath.Join(t.TempDir(), "conversation-runtime.json"))
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	server := &Server{
		sessions:                    sessionapp.NewService(),
		conversationRuntimeSessions: registry,
		logger:                      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	patchReq := httptest.NewRequest(http.MethodPatch, "/api/conversation-runtime/sessions/chat-config?route=chat", strings.NewReader(`{
		"title":"Configurable chat",
		"model_provider_id":"openai",
		"model_id":"gpt-5.4",
		"tool_ids":["terminal"],
		"skill_ids":["memory","frontend-design"],
		"mcp_ids":["github"]
	}`))
	patchRec := httptest.NewRecorder()
	server.conversationRuntimeSessionItemHandler(patchRec, patchReq)
	if patchRec.Code != http.StatusOK {
		t.Fatalf("expected patch status %d, got %d: %s", http.StatusOK, patchRec.Code, patchRec.Body.String())
	}

	var patched struct {
		Session struct {
			ID              string   `json:"id"`
			Title           string   `json:"title"`
			ModelProviderID string   `json:"model_provider_id"`
			ModelID         string   `json:"model_id"`
			ToolIDs         []string `json:"tool_ids"`
			SkillIDs        []string `json:"skill_ids"`
			MCPIDs          []string `json:"mcp_ids"`
		} `json:"session"`
	}
	if err := json.Unmarshal(patchRec.Body.Bytes(), &patched); err != nil {
		t.Fatalf("decode patched payload: %v", err)
	}
	if patched.Session.ID != "chat-config" || patched.Session.Title != "Configurable chat" {
		t.Fatalf("unexpected patched session %+v", patched.Session)
	}
	if patched.Session.ModelProviderID != "openai" || patched.Session.ModelID != "gpt-5.4" {
		t.Fatalf("expected patched model selection, got %+v", patched.Session)
	}
	if strings.Join(patched.Session.SkillIDs, ",") != "memory,frontend-design" {
		t.Fatalf("expected patched skill ids, got %+v", patched.Session.SkillIDs)
	}

	clearReq := httptest.NewRequest(http.MethodPatch, "/api/conversation-runtime/sessions/chat-config?route=chat", strings.NewReader(`{
		"skill_ids":[],
		"mcp_ids":[]
	}`))
	clearRec := httptest.NewRecorder()
	server.conversationRuntimeSessionItemHandler(clearRec, clearReq)
	if clearRec.Code != http.StatusOK {
		t.Fatalf("expected clear status %d, got %d: %s", http.StatusOK, clearRec.Code, clearRec.Body.String())
	}

	getRec := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/api/conversation-runtime/sessions/chat-config?route=chat", nil)
	server.conversationRuntimeSessionItemHandler(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected get status %d, got %d: %s", http.StatusOK, getRec.Code, getRec.Body.String())
	}

	var loaded struct {
		Session struct {
			ID       string   `json:"id"`
			ToolIDs  []string `json:"tool_ids"`
			SkillIDs []string `json:"skill_ids"`
			MCPIDs   []string `json:"mcp_ids"`
		} `json:"session"`
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &loaded); err != nil {
		t.Fatalf("decode loaded payload: %v", err)
	}
	if len(loaded.Session.ToolIDs) != 1 || loaded.Session.ToolIDs[0] != "terminal" {
		t.Fatalf("expected untouched tool ids to remain, got %+v", loaded.Session.ToolIDs)
	}
	if len(loaded.Session.SkillIDs) != 0 || len(loaded.Session.MCPIDs) != 0 {
		t.Fatalf("expected skill and mcp ids to be cleared, got skills=%+v mcps=%+v", loaded.Session.SkillIDs, loaded.Session.MCPIDs)
	}
}

func TestSessionDeleteHandlerAllowsMissingHistory(t *testing.T) {
	history := &stubSessionHistory{deleteErr: sessionapp.ErrSessionNotFound}
	tasks := &stubSessionTaskService{}
	server := &Server{
		sessions:      history,
		tasks:         tasks,
		logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		workspaceRoot: t.TempDir(),
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/sessions/session-missing", nil)
	rec := httptest.NewRecorder()
	server.sessionMessageListHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestSessionDeleteHandlerReturnsTaskDeleteFailure(t *testing.T) {
	history := &stubSessionHistory{}
	tasks := &stubSessionTaskService{deleteErr: errors.New("task delete failed")}
	server := &Server{
		sessions:      history,
		tasks:         tasks,
		logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		workspaceRoot: t.TempDir(),
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/sessions/session-delete", nil)
	rec := httptest.NewRecorder()
	server.sessionMessageListHandler(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

package web

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	chatruntimedomain "alter0/internal/chatruntime/domain"
	sessionapp "alter0/internal/session/application"
	sessiondomain "alter0/internal/session/domain"
	shareddomain "alter0/internal/shared/domain"
	taskapp "alter0/internal/task/application"
	taskdomain "alter0/internal/task/domain"
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

func (s *stubSessionTaskService) ListBySession(string) []taskdomain.Task {
	return append([]taskdomain.Task(nil), s.items...)
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
	workspaceDir := filepath.Join(baseDir, "workspaces", "sessions", "session-delete")
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

func TestSessionCleanupHandlerDeletesInactiveSessionsAndWorkspaces(t *testing.T) {
	baseDir := t.TempDir()
	for _, sessionID := range []string{"old-a", "old-b"} {
		workspaceDir := filepath.Join(baseDir, "workspaces", "sessions", sessionID)
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
		if _, err := os.Stat(filepath.Join(baseDir, "workspaces", "sessions", sessionID)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected workspace %s removed, got %v", sessionID, err)
		}
	}
	if body.DeletedCount != 2 || body.SkippedPinnedCount != 1 {
		t.Fatalf("unexpected cleanup body %+v", body)
	}
}

func TestSessionCleanupHandlerSkipsPinnedBusyAndRecentChatRuntimeSessions(t *testing.T) {
	now := time.Date(2026, 4, 20, 9, 0, 0, 0, time.UTC)
	old := now.Add(-8 * 24 * time.Hour)
	recent := now.Add(-2 * time.Hour)
	history := &stubSessionHistory{}
	chatRuntimes := &stubWebChatRuntimeService{
		listByOwner: map[string][]chatruntimedomain.Session{
			chatSessionOwnerID: {
				{
					ID:           "chatRuntime-old",
					OwnerID:      chatSessionOwnerID,
					Status:       chatruntimedomain.SessionStatusReady,
					CreatedAt:    old,
					LastOutputAt: old,
					UpdatedAt:    old,
				},
				{
					ID:           "chatRuntime-pinned",
					OwnerID:      chatSessionOwnerID,
					Status:       chatruntimedomain.SessionStatusReady,
					Pinned:       true,
					CreatedAt:    old,
					LastOutputAt: old,
					UpdatedAt:    old,
				},
				{
					ID:           "chatRuntime-busy",
					OwnerID:      chatSessionOwnerID,
					Status:       chatruntimedomain.SessionStatusBusy,
					CreatedAt:    old,
					LastOutputAt: old,
					UpdatedAt:    old,
				},
				{
					ID:           "chatRuntime-recent",
					OwnerID:      chatSessionOwnerID,
					Status:       chatruntimedomain.SessionStatusReady,
					CreatedAt:    old,
					LastOutputAt: recent,
					UpdatedAt:    recent,
				},
			},
		},
	}
	server := &Server{
		sessions:     history,
		chatRuntimes: chatRuntimes,
		logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	server.ensureMaintenanceService()

	body := server.maintenance.RunSessionCleanup(now)

	if len(chatRuntimes.deleteIDs) != 1 || chatRuntimes.deleteIDs[0] != "chatRuntime-old" {
		t.Fatalf("expected only old chatRuntime deleted, got %+v", chatRuntimes.deleteIDs)
	}
	if len(chatRuntimes.deleteOwnerIDs) != 1 || chatRuntimes.deleteOwnerIDs[0] != chatSessionOwnerID+":chatRuntime-old" {
		t.Fatalf("expected only old chatRuntime session deleted, got %+v", chatRuntimes.deleteOwnerIDs)
	}
	if body.DeletedCount != 1 || body.SkippedPinnedCount != 1 || body.SkippedProtectedCount != 1 || body.ScannedCount != 4 {
		t.Fatalf("expected combined chatRuntime cleanup counts, got %+v", body)
	}
	if body.ChatRuntimeDeletedCount != 1 || body.ChatRuntimeSkippedPinnedCount != 1 || body.ChatRuntimeSkippedProtectedCount != 1 || body.ChatRuntimeScannedCount != 4 {
		t.Fatalf("expected chatRuntime cleanup counts, got %+v", body)
	}
	if history.lastCleanupOption.InactiveDuration != 7*24*time.Hour {
		t.Fatalf("expected session cleanup still invoked with fixed threshold, got %+v", history.lastCleanupOption)
	}
}

func TestSessionCleanupHandlerDeletesInactiveChatRuntimeSessions(t *testing.T) {
	now := time.Date(2026, 4, 20, 9, 0, 0, 0, time.UTC)
	old := now.Add(-8 * 24 * time.Hour)
	history := &stubSessionHistory{}
	chatRuntimes := &stubWebChatRuntimeService{
		listByOwner: map[string][]chatruntimedomain.Session{
			chatSessionOwnerID: {
				{
					ID:           "chat-old",
					OwnerID:      chatSessionOwnerID,
					Status:       chatruntimedomain.SessionStatusReady,
					CreatedAt:    old,
					LastOutputAt: old,
					UpdatedAt:    old,
				},
				{
					ID:           "chat-older",
					OwnerID:      chatSessionOwnerID,
					Status:       chatruntimedomain.SessionStatusReady,
					CreatedAt:    old,
					LastOutputAt: old,
					UpdatedAt:    old,
				},
			},
		},
	}
	server := &Server{
		sessions:     history,
		chatRuntimes: chatRuntimes,
		logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	server.ensureMaintenanceService()

	body := server.maintenance.RunSessionCleanup(now)

	expectedDeletes := []string{
		chatSessionOwnerID + ":chat-old",
		chatSessionOwnerID + ":chat-older",
	}
	if !reflect.DeepEqual(chatRuntimes.deleteOwnerIDs, expectedDeletes) {
		t.Fatalf("expected chat runtime deletes, got %+v", chatRuntimes.deleteOwnerIDs)
	}
	if body.ChatRuntimeDeletedCount != 2 || body.ChatRuntimeScannedCount != 2 {
		t.Fatalf("expected combined chatRuntime cleanup counts, got %+v", body)
	}
}

func TestMaintenanceStatusOnlyIncludesSessionCleanup(t *testing.T) {
	server := &Server{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	server.ensureMaintenanceService()

	status := server.maintenance.Status(time.Now().UTC())
	if len(status.Items) != 1 || status.Items[0].JobID != defaultSessionCleanupJobID {
		t.Fatalf("expected only session cleanup maintenance, got %+v", status.Items)
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
		t.Fatalf("expected chatRuntime task session unprotected, got %+v", history.lastCleanupOption.ProtectedSessionIDs)
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

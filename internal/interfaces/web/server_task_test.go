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

	sessionapp "alter0/internal/session/application"
	sessiondomain "alter0/internal/session/domain"
	shareddomain "alter0/internal/shared/domain"
	"alter0/internal/shared/infrastructure/observability"
	taskapp "alter0/internal/task/application"
	taskdomain "alter0/internal/task/domain"
)

type stubWebTaskService struct {
	shouldAsync bool
	submitTask  taskdomain.Task
	submitErr   error
	items       map[string]taskdomain.Task
	bySession   map[string][]taskdomain.Task
	logPages    map[string]taskapp.TaskLogPage
	artifacts   map[string][]taskdomain.TaskArtifact
	listPage    taskapp.TaskPage
	lastList    taskapp.ListQuery
	cancelErr   error
	retryErr    error
	getFn       func(taskID string) (taskdomain.Task, bool)
	listLogsFn  func(taskID string, cursor int, limit int) (taskapp.TaskLogPage, error)
}

func (s *stubWebTaskService) ShouldRunAsync(_ shareddomain.UnifiedMessage) bool {
	return s.shouldAsync
}

func (s *stubWebTaskService) Submit(_ shareddomain.UnifiedMessage) (taskdomain.Task, error) {
	if s.submitErr != nil {
		return taskdomain.Task{}, s.submitErr
	}
	return s.submitTask, nil
}

func (s *stubWebTaskService) List(query taskapp.ListQuery) taskapp.TaskPage {
	s.lastList = query
	return s.listPage
}

func (s *stubWebTaskService) Get(taskID string) (taskdomain.Task, bool) {
	if s.getFn != nil {
		return s.getFn(taskID)
	}
	if s.items == nil {
		return taskdomain.Task{}, false
	}
	item, ok := s.items[taskID]
	return item, ok
}

func (s *stubWebTaskService) ListBySession(sessionID string) []taskdomain.Task {
	if s.bySession == nil {
		return []taskdomain.Task{}
	}
	items := s.bySession[sessionID]
	out := make([]taskdomain.Task, 0, len(items))
	out = append(out, items...)
	return out
}

func (s *stubWebTaskService) Cancel(taskID string) (taskdomain.Task, error) {
	if s.cancelErr != nil {
		return taskdomain.Task{}, s.cancelErr
	}
	item := s.items[taskID]
	item.Status = taskdomain.TaskStatusCanceled
	item.ErrorCode = "task_canceled"
	s.items[taskID] = item
	return item, nil
}

func (s *stubWebTaskService) ListLogs(taskID string, cursor int, limit int) (taskapp.TaskLogPage, error) {
	if s.listLogsFn != nil {
		return s.listLogsFn(taskID, cursor, limit)
	}
	if s.logPages == nil {
		return taskapp.TaskLogPage{}, taskapp.ErrTaskNotFound
	}
	page, ok := s.logPages[taskID]
	if !ok {
		return taskapp.TaskLogPage{}, taskapp.ErrTaskNotFound
	}
	return page, nil
}

func (s *stubWebTaskService) ListArtifacts(taskID string) ([]taskdomain.TaskArtifact, error) {
	if s.artifacts == nil {
		return nil, taskapp.ErrTaskNotFound
	}
	items, ok := s.artifacts[taskID]
	if !ok {
		return nil, taskapp.ErrTaskNotFound
	}
	out := make([]taskdomain.TaskArtifact, 0, len(items))
	out = append(out, items...)
	return out, nil
}

func (s *stubWebTaskService) Retry(taskID string) (taskdomain.Task, error) {
	if s.retryErr != nil {
		return taskdomain.Task{}, s.retryErr
	}
	item := s.items[taskID]
	item.Status = taskdomain.TaskStatusQueued
	item.Progress = 0
	s.items[taskID] = item
	return item, nil
}

func TestMessageHandlerReturnsAcceptedForAsyncTask(t *testing.T) {
	orchestrator := &stubWebOrchestrator{}
	taskSvc := &stubWebTaskService{
		shouldAsync: true,
		submitTask: taskdomain.Task{
			ID:        "task-accepted",
			SessionID: "session-fixed",
			Status:    taskdomain.TaskStatusQueued,
		},
	}
	server := &Server{
		orchestrator: orchestrator,
		tasks:        taskSvc,
		telemetry:    observability.NewTelemetry(),
		idGenerator: &sequenceIDGenerator{
			ids: []string{"session-generated", "message-generated", "trace-generated"},
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/messages", strings.NewReader(`{"session_id":"session-fixed","content":"generate report artifact"}`))
	rec := httptest.NewRecorder()
	server.messageHandler(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}
	var body messageResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.TaskID != "task-accepted" {
		t.Fatalf("expected task-accepted, got %q", body.TaskID)
	}
	if body.TaskStatus != string(taskdomain.TaskStatusQueued) {
		t.Fatalf("expected queued status, got %q", body.TaskStatus)
	}
	if body.Result.ErrorCode != "task_accepted" {
		t.Fatalf("expected task_accepted, got %q", body.Result.ErrorCode)
	}
	if orchestrator.lastMessage.MessageID != "" {
		t.Fatalf("expected orchestrator not called")
	}
}

func TestMessageStreamHandlerReturnsDoneForAsyncTask(t *testing.T) {
	server := &Server{
		orchestrator: &stubWebOrchestrator{},
		tasks: &stubWebTaskService{
			shouldAsync: true,
			submitTask: taskdomain.Task{
				ID:        "task-stream",
				SessionID: "session-fixed",
				Status:    taskdomain.TaskStatusRunning,
			},
		},
		telemetry: observability.NewTelemetry(),
		idGenerator: &sequenceIDGenerator{
			ids: []string{"session-generated", "message-generated", "trace-generated"},
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/messages/stream", strings.NewReader(`{"session_id":"session-fixed","content":"generate artifact report"}`))
	rec := httptest.NewRecorder()
	server.messageStreamHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "event: start\n") {
		t.Fatalf("expected start event, got body %q", body)
	}
	if strings.Contains(body, "event: delta\n") {
		t.Fatalf("did not expect delta events for async accept path, got body %q", body)
	}
	if !strings.Contains(body, "event: done\n") {
		t.Fatalf("expected done event, got body %q", body)
	}
	if !strings.Contains(body, `"task_id":"task-stream"`) {
		t.Fatalf("expected task id in done payload, got body %q", body)
	}
}

func TestSessionTaskListAndTaskItemEndpoints(t *testing.T) {
	taskSvc := &stubWebTaskService{
		items: map[string]taskdomain.Task{
			"task-1": {
				ID:        "task-1",
				SessionID: "session-a",
				Status:    taskdomain.TaskStatusRunning,
			},
		},
		bySession: map[string][]taskdomain.Task{
			"session-a": {
				{
					ID:        "task-1",
					SessionID: "session-a",
					Status:    taskdomain.TaskStatusRunning,
				},
			},
		},
	}
	server := &Server{
		tasks:  taskSvc,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/sessions/session-a/tasks", nil)
	listRec := httptest.NewRecorder()
	server.sessionMessageListHandler(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, listRec.Code)
	}
	if !strings.Contains(listRec.Body.String(), `"task-1"`) {
		t.Fatalf("expected task in session list payload, got %s", listRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/tasks/task-1", nil)
	getRec := httptest.NewRecorder()
	server.taskItemHandler(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, getRec.Code)
	}

	cancelReq := httptest.NewRequest(http.MethodPost, "/api/tasks/task-1/cancel", nil)
	cancelRec := httptest.NewRecorder()
	server.taskItemHandler(cancelRec, cancelReq)
	if cancelRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, cancelRec.Code)
	}
	if !strings.Contains(cancelRec.Body.String(), `"status":"canceled"`) {
		t.Fatalf("expected canceled task response, got %s", cancelRec.Body.String())
	}
}

func TestTaskItemHandlerMethodNotAllowed(t *testing.T) {
	server := &Server{
		tasks: &stubWebTaskService{
			items: map[string]taskdomain.Task{
				"task-1": {ID: "task-1", SessionID: "s-1", Status: taskdomain.TaskStatusQueued},
			},
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	req := httptest.NewRequest(http.MethodDelete, "/api/tasks/task-1", nil)
	rec := httptest.NewRecorder()
	server.taskItemHandler(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestTaskItemHandlerNotFound(t *testing.T) {
	server := &Server{
		tasks: &stubWebTaskService{
			items: map[string]taskdomain.Task{},
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/tasks/not-found", nil)
	rec := httptest.NewRecorder()
	server.taskItemHandler(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestTaskServiceUnavailable(t *testing.T) {
	server := &Server{
		orchestrator: &stubWebOrchestrator{
			result: shareddomain.OrchestrationResult{
				Route:  shareddomain.RouteNL,
				Output: "ok",
			},
		},
		telemetry: observability.NewTelemetry(),
		idGenerator: &sequenceIDGenerator{
			ids: []string{"session-generated", "message-generated", "trace-generated"},
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/task-1", nil)
	rec := httptest.NewRecorder()
	server.taskItemHandler(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestTaskCollectionEndpoints(t *testing.T) {
	now := time.Date(2026, 3, 4, 2, 0, 0, 0, time.UTC)
	taskSvc := &stubWebTaskService{
		submitTask: taskdomain.Task{
			ID:        "task-created",
			SessionID: "session-a",
			Status:    taskdomain.TaskStatusQueued,
			CreatedAt: now,
			TimeoutMS: 8000,
		},
		listPage: taskapp.TaskPage{
			Items: []taskdomain.Task{
				{ID: "task-created", SessionID: "session-a", Status: taskdomain.TaskStatusQueued},
			},
			Pagination: taskapp.Pagination{Page: 1, PageSize: 10, Total: 1, HasNext: false},
		},
	}
	server := &Server{
		tasks: taskSvc,
		idGenerator: &sequenceIDGenerator{
			ids: []string{"id-1", "id-2", "id-3"},
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	createReq := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{"session_id":"session-a","source_message_id":"msg-1","task_type":"artifact","input":"generate report","idempotency_key":"idem-1"}`))
	createRec := httptest.NewRecorder()
	server.taskCollectionHandler(createRec, createReq)
	if createRec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, createRec.Code)
	}
	if !strings.Contains(createRec.Body.String(), `"task_id":"task-created"`) {
		t.Fatalf("expected created task response, got %s", createRec.Body.String())
	}
	if !strings.Contains(createRec.Body.String(), `"accepted_at":"`) {
		t.Fatalf("expected accepted_at in response, got %s", createRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/tasks?session_id=session-a&status=queued&page=1&page_size=10", nil)
	listRec := httptest.NewRecorder()
	server.taskCollectionHandler(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, listRec.Code)
	}
	if taskSvc.lastList.SessionID != "session-a" || taskSvc.lastList.Status != taskdomain.TaskStatusQueued {
		t.Fatalf("unexpected list query %+v", taskSvc.lastList)
	}
	if !strings.Contains(listRec.Body.String(), `"task-created"`) {
		t.Fatalf("expected list payload with task-created, got %s", listRec.Body.String())
	}
}

func TestTaskItemLogsArtifactsAndRetryEndpoints(t *testing.T) {
	now := time.Date(2026, 3, 4, 3, 0, 0, 0, time.UTC)
	taskSvc := &stubWebTaskService{
		items: map[string]taskdomain.Task{
			"task-1": {ID: "task-1", SessionID: "session-a", Status: taskdomain.TaskStatusFailed},
		},
		logPages: map[string]taskapp.TaskLogPage{
			"task-1": {
				Items: []taskdomain.TaskLog{
					{Seq: 1, Stage: "accept", CreatedAt: now, Level: taskdomain.TaskLogLevelInfo, Message: "accepted"},
				},
				Cursor:     0,
				NextCursor: 1,
				HasMore:    false,
			},
		},
		artifacts: map[string][]taskdomain.TaskArtifact{
			"task-1": {
				{ArtifactID: "a-1", ArtifactType: "report", Name: "report.md", ContentType: "text/markdown", URI: "file:///tmp/report.md", CreatedAt: now},
			},
		},
	}
	server := &Server{
		tasks:  taskSvc,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	logReq := httptest.NewRequest(http.MethodGet, "/api/tasks/task-1/logs?cursor=0&limit=10", nil)
	logRec := httptest.NewRecorder()
	server.taskItemHandler(logRec, logReq)
	if logRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, logRec.Code)
	}
	if !strings.Contains(logRec.Body.String(), `"stage":"accept"`) {
		t.Fatalf("expected log stage in payload, got %s", logRec.Body.String())
	}

	artifactReq := httptest.NewRequest(http.MethodGet, "/api/tasks/task-1/artifacts", nil)
	artifactRec := httptest.NewRecorder()
	server.taskItemHandler(artifactRec, artifactReq)
	if artifactRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, artifactRec.Code)
	}
	if !strings.Contains(artifactRec.Body.String(), `"artifact_id":"a-1"`) {
		t.Fatalf("expected artifact payload, got %s", artifactRec.Body.String())
	}

	retryReq := httptest.NewRequest(http.MethodPost, "/api/tasks/task-1/retry", nil)
	retryRec := httptest.NewRecorder()
	server.taskItemHandler(retryRec, retryReq)
	if retryRec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, retryRec.Code)
	}
	if !strings.Contains(retryRec.Body.String(), `"status":"queued"`) {
		t.Fatalf("expected queued status after retry, got %s", retryRec.Body.String())
	}
}

func TestControlTaskLogsBackfillEndpoint(t *testing.T) {
	now := time.Date(2026, 3, 4, 6, 0, 0, 0, time.UTC)
	taskSvc := &stubWebTaskService{
		logPages: map[string]taskapp.TaskLogPage{
			"task-control-1": {
				Items: []taskdomain.TaskLog{
					{Seq: 3, Stage: "running", CreatedAt: now, Level: taskdomain.TaskLogLevelInfo, Message: "step-3"},
				},
				Cursor:     2,
				NextCursor: 3,
				HasMore:    false,
			},
		},
	}
	server := &Server{
		tasks:  taskSvc,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/control/tasks/task-control-1/logs?cursor=2&limit=10", nil)
	rec := httptest.NewRecorder()
	server.controlTaskItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"seq":3`) {
		t.Fatalf("expected seq=3 in payload, got %s", rec.Body.String())
	}
}

func TestControlTaskLogStreamSupportsCursorReplay(t *testing.T) {
	now := time.Date(2026, 3, 4, 7, 0, 0, 0, time.UTC)
	logs := []taskdomain.TaskLog{
		{Seq: 1, Stage: "accept", CreatedAt: now, Level: taskdomain.TaskLogLevelInfo, Message: "accepted"},
		{Seq: 2, Stage: "running", CreatedAt: now.Add(time.Second), Level: taskdomain.TaskLogLevelInfo, Message: "running"},
		{Seq: 3, Stage: "success", CreatedAt: now.Add(2 * time.Second), Level: taskdomain.TaskLogLevelInfo, Message: "done"},
	}
	taskSvc := &stubWebTaskService{
		getFn: func(taskID string) (taskdomain.Task, bool) {
			if taskID != "task-control-stream" {
				return taskdomain.Task{}, false
			}
			return taskdomain.Task{ID: taskID, Status: taskdomain.TaskStatusSuccess}, true
		},
		listLogsFn: func(taskID string, cursor int, limit int) (taskapp.TaskLogPage, error) {
			if taskID != "task-control-stream" {
				return taskapp.TaskLogPage{}, taskapp.ErrTaskNotFound
			}
			if cursor < 0 {
				cursor = 0
			}
			if cursor > len(logs) {
				cursor = len(logs)
			}
			if limit <= 0 {
				limit = 50
			}
			end := cursor + limit
			if end > len(logs) {
				end = len(logs)
			}
			items := make([]taskdomain.TaskLog, 0, end-cursor)
			items = append(items, logs[cursor:end]...)
			return taskapp.TaskLogPage{
				Items:      items,
				Cursor:     cursor,
				NextCursor: end,
				HasMore:    end < len(logs),
			}, nil
		},
	}
	server := &Server{
		tasks:  taskSvc,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/control/tasks/task-control-stream/logs/stream?cursor=1", nil)
	rec := httptest.NewRecorder()
	server.controlTaskItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("expected text/event-stream content type, got %q", contentType)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "event: start\n") {
		t.Fatalf("expected start event, got %q", body)
	}
	if !strings.Contains(body, "event: log\n") {
		t.Fatalf("expected log event, got %q", body)
	}
	if strings.Contains(body, `"seq":1`) {
		t.Fatalf("did not expect seq=1 after cursor replay, got %q", body)
	}
	if !strings.Contains(body, `"seq":2`) || !strings.Contains(body, `"seq":3`) {
		t.Fatalf("expected seq=2 and seq=3 in stream payload, got %q", body)
	}
	if !strings.Contains(body, "event: done\n") {
		t.Fatalf("expected done event, got %q", body)
	}
	if !strings.Contains(body, `"next_cursor":3`) {
		t.Fatalf("expected next_cursor=3 in done payload, got %q", body)
	}
}

func TestSessionTaskLatestQuery(t *testing.T) {
	taskSvc := &stubWebTaskService{
		bySession: map[string][]taskdomain.Task{
			"session-a": {
				{ID: "task-2", SessionID: "session-a", Status: taskdomain.TaskStatusSuccess},
				{ID: "task-1", SessionID: "session-a", Status: taskdomain.TaskStatusRunning},
			},
		},
	}
	server := &Server{
		tasks:  taskSvc,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/session-a/tasks?latest=true", nil)
	rec := httptest.NewRecorder()
	server.sessionMessageListHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"task-2"`) {
		t.Fatalf("expected latest task in body, got %s", body)
	}
	if strings.Contains(body, `"task-1"`) {
		t.Fatalf("expected only latest task, got %s", body)
	}
}

func TestSessionMessageEndpointStillWorks(t *testing.T) {
	server := &Server{
		sessions: &stubSessionHistoryForTaskTests{
			messages: map[string][]sessiondomain.MessageRecord{
				"session-a": {
					{MessageID: "m1", SessionID: "session-a", Role: sessiondomain.MessageRoleUser, Content: "hello", Timestamp: time.Now().UTC()},
				},
			},
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/session-a/messages", nil)
	rec := httptest.NewRecorder()
	server.sessionMessageListHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

type stubSessionHistoryForTaskTests struct {
	messages map[string][]sessiondomain.MessageRecord
}

func (s *stubSessionHistoryForTaskTests) ListSessions(sessionapp.SessionQuery) sessionapp.SessionPage {
	return sessionapp.SessionPage{}
}

func (s *stubSessionHistoryForTaskTests) ListMessages(query sessionapp.MessageQuery) sessionapp.MessagePage {
	items := s.messages[query.SessionID]
	return sessionapp.MessagePage{
		Items: items,
		Pagination: sessionapp.Pagination{
			Page:     1,
			PageSize: len(items),
			Total:    len(items),
			HasNext:  false,
		},
	}
}

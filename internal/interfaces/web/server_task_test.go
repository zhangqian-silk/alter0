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
	taskdomain "alter0/internal/task/domain"
)

type stubWebTaskService struct {
	shouldAsync bool
	submitTask  taskdomain.Task
	submitErr   error
	items       map[string]taskdomain.Task
	bySession   map[string][]taskdomain.Task
	cancelErr   error
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

func (s *stubWebTaskService) Get(taskID string) (taskdomain.Task, bool) {
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

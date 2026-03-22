package web

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
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
	shouldAsync     bool
	assessment      taskapp.ComplexityAssessment
	submitTask      taskdomain.Task
	submitErr       error
	lastSubmitMsg   shareddomain.UnifiedMessage
	submitCallCount int
	items           map[string]taskdomain.Task
	bySession       map[string][]taskdomain.Task
	logPages        map[string]taskapp.TaskLogPage
	artifacts       map[string][]taskdomain.TaskArtifact
	artifactRaw     map[string][]byte
	listPage        taskapp.TaskPage
	lastList        taskapp.ListQuery
	listFn          func(query taskapp.ListQuery) taskapp.TaskPage
	cancelErr       error
	retryErr        error
	getFn           func(taskID string) (taskdomain.Task, bool)
	listLogsFn      func(taskID string, cursor int, limit int) (taskapp.TaskLogPage, error)
}

func (s *stubWebTaskService) AssessComplexity(_ shareddomain.UnifiedMessage) taskapp.ComplexityAssessment {
	if s.assessment.ExecutionMode != "" {
		return s.assessment
	}
	if s.shouldAsync {
		return taskapp.ComplexityAssessment{
			TaskSummary:              "生成异步任务卡片",
			TaskApproach:             "先评估请求，再转入后台任务执行。",
			EstimatedDurationSeconds: 420,
			ComplexityLevel:          taskapp.ComplexityLevelHigh,
			ExecutionMode:            taskapp.ExecutionModeAsync,
		}
	}
	return taskapp.ComplexityAssessment{
		EstimatedDurationSeconds: 12,
		ComplexityLevel:          taskapp.ComplexityLevelLow,
		ExecutionMode:            taskapp.ExecutionModeStreaming,
	}
}

func (s *stubWebTaskService) ShouldRunAsync(_ shareddomain.UnifiedMessage) bool {
	assessment := s.AssessComplexity(shareddomain.UnifiedMessage{})
	return assessment.ExecutionMode == taskapp.ExecutionModeAsync
}

func (s *stubWebTaskService) Submit(msg shareddomain.UnifiedMessage) (taskdomain.Task, error) {
	s.lastSubmitMsg = msg
	s.submitCallCount++
	if s.submitErr != nil {
		return taskdomain.Task{}, s.submitErr
	}
	return s.submitTask, nil
}

func (s *stubWebTaskService) List(query taskapp.ListQuery) taskapp.TaskPage {
	s.lastList = query
	if s.listFn != nil {
		return s.listFn(query)
	}
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

func (s *stubWebTaskService) ReadArtifact(_ context.Context, taskID string, artifactID string) (taskdomain.TaskArtifact, []byte, error) {
	items, err := s.ListArtifacts(taskID)
	if err != nil {
		return taskdomain.TaskArtifact{}, nil, err
	}
	for _, item := range items {
		if strings.TrimSpace(item.ArtifactID) != strings.TrimSpace(artifactID) {
			continue
		}
		key := taskID + "/" + artifactID
		if s.artifactRaw != nil {
			if raw, ok := s.artifactRaw[key]; ok {
				return item, append([]byte{}, raw...), nil
			}
		}
		if strings.TrimSpace(item.Content) != "" {
			return item, []byte(item.Content), nil
		}
		return taskdomain.TaskArtifact{}, nil, taskapp.ErrArtifactNotFound
	}
	return taskdomain.TaskArtifact{}, nil, taskapp.ErrArtifactNotFound
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
	if body.ExecutionMode != taskapp.ExecutionModeAsync {
		t.Fatalf("expected execution_mode async, got %q", body.ExecutionMode)
	}
	if body.EstimatedDurationSeconds <= 300 {
		t.Fatalf("expected estimated_duration_seconds > 300, got %d", body.EstimatedDurationSeconds)
	}
	if body.ComplexityLevel != taskapp.ComplexityLevelHigh {
		t.Fatalf("expected complexity_level high, got %q", body.ComplexityLevel)
	}
	if body.TaskCard == nil {
		t.Fatalf("expected task card in async response")
	}
	if body.TaskCard.TaskSummary != "生成异步任务卡片" {
		t.Fatalf("expected task summary from complexity assessment, got %q", body.TaskCard.TaskSummary)
	}
	if body.TaskCard.TaskID != "task-accepted" {
		t.Fatalf("expected task card task id task-accepted, got %q", body.TaskCard.TaskID)
	}
	if !strings.Contains(body.TaskCard.TaskDetailURL, "/api/control/tasks/task-accepted") {
		t.Fatalf("expected task detail url in task card, got %q", body.TaskCard.TaskDetailURL)
	}
	if taskSvc.lastSubmitMsg.Metadata[taskapp.MetadataTaskSummary] != "生成异步任务卡片" {
		t.Fatalf("expected submit metadata to include task summary, got %+v", taskSvc.lastSubmitMsg.Metadata)
	}
	if taskSvc.lastSubmitMsg.Metadata[taskapp.MetadataTaskApproach] == "" {
		t.Fatalf("expected submit metadata to include task approach, got %+v", taskSvc.lastSubmitMsg.Metadata)
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
	if !strings.Contains(body, `"execution_mode":"async"`) {
		t.Fatalf("expected execution_mode in done payload, got body %q", body)
	}
	if !strings.Contains(body, `"task_card":{"notice":"`) {
		t.Fatalf("expected task_card in done payload, got body %q", body)
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
			ID:            "task-created",
			SessionID:     "session-a",
			Status:        taskdomain.TaskStatusQueued,
			Phase:         string(taskdomain.TaskStatusQueued),
			QueuePosition: 2,
			CreatedAt:     now,
			AcceptedAt:    now,
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
	if !strings.Contains(createRec.Body.String(), `"queue_position":2`) {
		t.Fatalf("expected queue_position in response, got %s", createRec.Body.String())
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

func TestTaskCollectionAppliesTerminalExecutionDefaults(t *testing.T) {
	taskSvc := &stubWebTaskService{
		submitTask: taskdomain.Task{
			ID:        "task-terminal-created",
			SessionID: "terminal-session-a",
			Status:    taskdomain.TaskStatusQueued,
		},
	}
	server := &Server{
		tasks:       taskSvc,
		idGenerator: &sequenceIDGenerator{ids: []string{"trace-1"}},
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	createReq := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{"session_id":"terminal-session-a","task_type":"terminal","input":"ls -la","metadata":{"alter0.task.terminal_session_id":"terminal-a","alter0.task.terminal_interactive":"true"}}`))
	createRec := httptest.NewRecorder()
	server.taskCollectionHandler(createRec, createReq)

	if createRec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, createRec.Code)
	}
	if got := taskSvc.lastSubmitMsg.Metadata[codexSandboxMetadataKey]; got != codexSandboxDangerFullAccess {
		t.Fatalf("expected codex sandbox %q, got %q", codexSandboxDangerFullAccess, got)
	}
	if got := strings.TrimSpace(taskSvc.lastSubmitMsg.Metadata["codex_workspace_mode"]); got != "" {
		t.Fatalf("expected codex workspace mode unset, got %q", got)
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
				{
					ArtifactID:   "a-1",
					ArtifactType: "report",
					Name:         "report.md",
					ContentType:  "text/markdown",
					Summary:      "weekly report",
					Size:         42,
					DownloadURL:  "/api/tasks/task-1/artifacts/a-1/download",
					PreviewURL:   "/api/tasks/task-1/artifacts/a-1/preview",
					URI:          "file:///tmp/report.md",
					CreatedAt:    now,
				},
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
	if strings.Contains(artifactRec.Body.String(), "file:///tmp/report.md") {
		t.Fatalf("expected local path hidden in artifact payload, got %s", artifactRec.Body.String())
	}
	if !strings.Contains(artifactRec.Body.String(), `"download_url":"/api/tasks/task-1/artifacts/a-1/download"`) {
		t.Fatalf("expected artifact download url in payload, got %s", artifactRec.Body.String())
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

func TestTaskItemEndpointIncludesAsyncExecutionFields(t *testing.T) {
	now := time.Date(2026, 3, 4, 5, 0, 0, 0, time.UTC)
	taskSvc := &stubWebTaskService{
		items: map[string]taskdomain.Task{
			"task-async-1": {
				ID:             "task-async-1",
				SessionID:      "session-a",
				Status:         taskdomain.TaskStatusRunning,
				Phase:          "running",
				QueueWaitMS:    320,
				QueuePosition:  0,
				AcceptedAt:     now.Add(-time.Second),
				StartedAt:      now,
				CreatedAt:      now.Add(-2 * time.Second),
				UpdatedAt:      now,
				RequestContent: "run deploy",
			},
		},
	}
	server := &Server{
		tasks:  taskSvc,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/task-async-1", nil)
	rec := httptest.NewRecorder()
	server.taskItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"phase":"running"`) {
		t.Fatalf("expected phase field in payload, got %s", body)
	}
	if !strings.Contains(body, `"queue_wait_ms":320`) {
		t.Fatalf("expected queue_wait_ms field in payload, got %s", body)
	}
	if !strings.Contains(body, `"accepted_at":"`) || !strings.Contains(body, `"started_at":"`) {
		t.Fatalf("expected accepted_at/started_at fields in payload, got %s", body)
	}
}

func TestTaskArtifactDownloadAndPreviewEndpoints(t *testing.T) {
	now := time.Date(2026, 3, 4, 4, 0, 0, 0, time.UTC)
	taskSvc := &stubWebTaskService{
		artifacts: map[string][]taskdomain.TaskArtifact{
			"task-1": {
				{
					ArtifactID:  "a-1",
					Name:        "report.txt",
					ContentType: "text/plain",
					DownloadURL: "/api/tasks/task-1/artifacts/a-1/download",
					PreviewURL:  "/api/tasks/task-1/artifacts/a-1/preview",
					CreatedAt:   now,
				},
				{
					ArtifactID:  "a-2",
					Name:        "bundle.zip",
					ContentType: "application/zip",
					DownloadURL: "/api/tasks/task-1/artifacts/a-2/download",
					CreatedAt:   now,
				},
			},
		},
		artifactRaw: map[string][]byte{
			"task-1/a-1": []byte("hello artifact"),
			"task-1/a-2": []byte("PK"),
		},
	}
	server := &Server{
		tasks:  taskSvc,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	downloadReq := httptest.NewRequest(http.MethodGet, "/api/tasks/task-1/artifacts/a-1/download", nil)
	downloadRec := httptest.NewRecorder()
	server.taskItemHandler(downloadRec, downloadReq)
	if downloadRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, downloadRec.Code)
	}
	if body := downloadRec.Body.String(); body != "hello artifact" {
		t.Fatalf("expected downloaded artifact body, got %q", body)
	}
	if contentDisposition := downloadRec.Header().Get("Content-Disposition"); !strings.Contains(contentDisposition, "report.txt") {
		t.Fatalf("expected attachment filename report.txt, got %q", contentDisposition)
	}

	previewReq := httptest.NewRequest(http.MethodGet, "/api/tasks/task-1/artifacts/a-1/preview", nil)
	previewRec := httptest.NewRecorder()
	server.taskItemHandler(previewRec, previewReq)
	if previewRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, previewRec.Code)
	}
	if body := previewRec.Body.String(); body != "hello artifact" {
		t.Fatalf("expected preview artifact body, got %q", body)
	}

	unsupportedPreviewReq := httptest.NewRequest(http.MethodGet, "/api/tasks/task-1/artifacts/a-2/preview", nil)
	unsupportedPreviewRec := httptest.NewRecorder()
	server.taskItemHandler(unsupportedPreviewRec, unsupportedPreviewReq)
	if unsupportedPreviewRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, unsupportedPreviewRec.Code)
	}
	if !strings.Contains(unsupportedPreviewRec.Body.String(), `"error_code":"artifact_preview_not_supported"`) {
		t.Fatalf("expected preview unsupported error code, got %s", unsupportedPreviewRec.Body.String())
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

func TestControlTaskViewAndActionConstraints(t *testing.T) {
	now := time.Date(2026, 3, 4, 4, 0, 0, 0, time.UTC)
	taskSvc := &stubWebTaskService{
		items: map[string]taskdomain.Task{
			"task-1": {
				ID:              "task-1",
				SessionID:       "session-a",
				SourceMessageID: "msg-user-1",
				Status:          taskdomain.TaskStatusFailed,
				CreatedAt:       now,
				UpdatedAt:       now,
				RequestContent:  "generate report",
				RequestMetadata: map[string]string{
					taskapp.MetadataTaskTriggerTypeKey: "cron",
					taskapp.MetadataTaskChannelTypeKey: "scheduler",
					taskapp.MetadataTaskChannelIDKey:   "scheduler-default",
					taskapp.MetadataTaskCorrelationKey: "job-nightly",
					"job_id":                           "job-nightly",
					"job_name":                         "Nightly Sync",
					"fired_at":                         "2026-03-04T03:45:00Z",
				},
				MessageLink: taskdomain.TaskMessageLink{
					TaskID:           "task-1",
					SessionID:        "session-a",
					RequestMessageID: "msg-user-1",
					ResultMessageID:  "msg-assistant-1",
				},
			},
		},
	}
	server := &Server{
		tasks:  taskSvc,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/control/tasks/task-1", nil)
	getRec := httptest.NewRecorder()
	server.controlTaskItemHandler(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, getRec.Code)
	}
	body := getRec.Body.String()
	if !strings.Contains(body, `"allowed_statuses":["failed","canceled"]`) {
		t.Fatalf("expected retry constraint payload, got %s", body)
	}
	if !strings.Contains(body, `"task_detail_path":"/api/control/tasks/task-1"`) {
		t.Fatalf("expected task detail link, got %s", body)
	}
	if !strings.Contains(body, `"session_messages_path":"/api/sessions/session-a/messages"`) {
		t.Fatalf("expected session message link, got %s", body)
	}
	if !strings.Contains(body, `"terminal_session_id":"session-a"`) {
		t.Fatalf("expected terminal session id in detail view, got %s", body)
	}
	if !strings.Contains(body, `"terminal_max_sessions":5`) {
		t.Fatalf("expected terminal max sessions in detail view, got %s", body)
	}
	if !strings.Contains(body, `"trigger_type":"cron"`) || !strings.Contains(body, `"channel_type":"scheduler"`) {
		t.Fatalf("expected source payload in detail view, got %s", body)
	}
	if !strings.Contains(body, `"job_id":"job-nightly"`) || !strings.Contains(body, `"fired_at":"2026-03-04T03:45:00Z"`) {
		t.Fatalf("expected cron source fields in detail view, got %s", body)
	}

	taskSvc.retryErr = taskapp.ErrTaskConflict
	retryReq := httptest.NewRequest(http.MethodPost, "/api/control/tasks/task-1/retry", nil)
	retryRec := httptest.NewRecorder()
	server.controlTaskItemHandler(retryRec, retryReq)
	if retryRec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, retryRec.Code)
	}
	if !strings.Contains(retryRec.Body.String(), `"view"`) {
		t.Fatalf("expected conflict view payload, got %s", retryRec.Body.String())
	}
}

func TestControlTaskTerminalInputCreatesFollowUpTask(t *testing.T) {
	now := time.Date(2026, 3, 5, 11, 0, 0, 0, time.UTC)
	taskSvc := &stubWebTaskService{
		items: map[string]taskdomain.Task{
			"task-root": {
				ID:        "task-root",
				SessionID: "session-a",
				Status:    taskdomain.TaskStatusSuccess,
				CreatedAt: now.Add(-2 * time.Minute),
				UpdatedAt: now.Add(-2 * time.Minute),
			},
			"task-1": {
				ID:        "task-1",
				SessionID: "session-a",
				Status:    taskdomain.TaskStatusRunning,
				CreatedAt: now,
				UpdatedAt: now,
				RequestMetadata: map[string]string{
					taskapp.MetadataTaskUserIDKey:      "user-a",
					taskapp.MetadataTaskCorrelationKey: "corr-a",
					taskapp.MetadataTaskChannelTypeKey: "web",
					taskapp.MetadataTaskChannelIDKey:   "web-default",
					controlTaskTerminalSessionIDKey:    "terminal-a",
					controlTaskTerminalInteractiveKey:  "true",
					controlTaskTerminalParentIDKey:     "task-root",
				},
			},
		},
		submitTask: taskdomain.Task{
			ID:        "task-2",
			SessionID: "session-a",
			Status:    taskdomain.TaskStatusQueued,
		},
	}
	server := &Server{
		tasks:       taskSvc,
		idGenerator: &sequenceIDGenerator{ids: []string{"msg-followup", "trace-followup"}},
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/control/tasks/task-1/terminal/input", strings.NewReader(`{"input":"continue with next step","reuse_task":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.controlTaskItemHandler(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}
	if taskSvc.submitCallCount != 1 {
		t.Fatalf("expected submit called once, got %d", taskSvc.submitCallCount)
	}
	if taskSvc.lastSubmitMsg.Content != "continue with next step" {
		t.Fatalf("expected forwarded input content, got %q", taskSvc.lastSubmitMsg.Content)
	}
	if got := taskSvc.lastSubmitMsg.Metadata[controlTaskTerminalSessionIDKey]; got != "terminal-a" {
		t.Fatalf("expected terminal session metadata terminal-a, got %q", got)
	}
	if got := taskSvc.lastSubmitMsg.Metadata[controlTaskTerminalParentIDKey]; got != "task-1" {
		t.Fatalf("expected terminal parent metadata task-1, got %q", got)
	}
	if got := taskSvc.lastSubmitMsg.Metadata[controlTaskTerminalInteractiveKey]; got != "true" {
		t.Fatalf("expected interactive metadata true, got %q", got)
	}
	if !strings.Contains(rec.Body.String(), `"task_id":"task-2"`) {
		t.Fatalf("expected follow-up task id in response, got %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"anchor_task_id":"task-root"`) {
		t.Fatalf("expected terminal anchor task id in response, got %s", rec.Body.String())
	}
}

func TestControlTaskTerminalInputRejectsWhenLimitReached(t *testing.T) {
	taskSvc := &stubWebTaskService{
		items: map[string]taskdomain.Task{
			"task-target": {
				ID:        "task-target",
				SessionID: "session-target",
				Status:    taskdomain.TaskStatusRunning,
				RequestMetadata: map[string]string{
					controlTaskTerminalSessionIDKey: "session-target",
				},
			},
		},
		listFn: func(query taskapp.ListQuery) taskapp.TaskPage {
			if query.Status != taskdomain.TaskStatusQueued {
				return taskapp.TaskPage{Items: []taskdomain.Task{}, Pagination: taskapp.Pagination{Page: query.Page, PageSize: query.PageSize, Total: 0, HasNext: false}}
			}
			items := []taskdomain.Task{}
			for i := 1; i <= 5; i++ {
				suffix := strconv.Itoa(i)
				items = append(items, taskdomain.Task{
					ID:        "task-active-" + suffix,
					SessionID: "session-active-" + suffix,
					Status:    taskdomain.TaskStatusQueued,
					RequestMetadata: map[string]string{
						controlTaskTerminalSessionIDKey: "session-active-" + suffix,
					},
				})
			}
			return taskapp.TaskPage{Items: items, Pagination: taskapp.Pagination{Page: query.Page, PageSize: query.PageSize, Total: len(items), HasNext: false}}
		},
	}
	server := &Server{
		tasks:       taskSvc,
		idGenerator: &sequenceIDGenerator{ids: []string{"msg-followup", "trace-followup"}},
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/control/tasks/task-target/terminal/input", strings.NewReader(`{"input":"run"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.controlTaskItemHandler(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, rec.Code)
	}
	if taskSvc.submitCallCount != 0 {
		t.Fatalf("expected submit not called when limit reached, got %d", taskSvc.submitCallCount)
	}
	if !strings.Contains(rec.Body.String(), `"error_code":"terminal_session_limit_reached"`) {
		t.Fatalf("expected terminal limit error code, got %s", rec.Body.String())
	}
}

func TestControlTaskCollectionEndpointFiltersAndPagination(t *testing.T) {
	now := time.Date(2026, 3, 4, 9, 0, 0, 0, time.UTC)
	source := []taskdomain.Task{
		{
			ID:              "task-a",
			SessionID:       "session-a",
			SourceMessageID: "msg-user-a",
			Status:          taskdomain.TaskStatusFailed,
			Progress:        60,
			RetryCount:      1,
			CreatedAt:       now.Add(-4 * time.Hour),
			UpdatedAt:       now.Add(-3 * time.Hour),
			FinishedAt:      now.Add(-3 * time.Hour),
			RequestContent:  "build report",
			ErrorCode:       "task_failed",
			RequestMetadata: map[string]string{
				taskapp.MetadataTaskTriggerTypeKey: "user",
				taskapp.MetadataTaskChannelTypeKey: "web",
				taskapp.MetadataTaskChannelIDKey:   "web-primary",
				taskapp.MetadataTaskCorrelationKey: "corr-user-a",
			},
		},
		{
			ID:              "task-b",
			SessionID:       "session-a",
			SourceMessageID: "msg-user-b",
			Status:          taskdomain.TaskStatusFailed,
			Progress:        80,
			RetryCount:      0,
			CreatedAt:       now.Add(-2 * time.Hour),
			UpdatedAt:       now.Add(-90 * time.Minute),
			FinishedAt:      now.Add(-90 * time.Minute),
			RequestContent:  "build report",
			ErrorMessage:    "network timeout",
			RequestMetadata: map[string]string{
				taskapp.MetadataTaskTriggerTypeKey: "cron",
				taskapp.MetadataTaskChannelTypeKey: "scheduler",
				taskapp.MetadataTaskChannelIDKey:   "scheduler-default",
				taskapp.MetadataTaskCorrelationKey: "job-nightly",
				"job_id":                           "job-nightly",
				"job_name":                         "Nightly Build",
				"fired_at":                         "2026-03-04T07:20:00Z",
			},
			MessageLink: taskdomain.TaskMessageLink{
				TaskID:           "task-b",
				SessionID:        "session-a",
				RequestMessageID: "msg-user-b",
				ResultMessageID:  "msg-assistant-b",
			},
		},
		{
			ID:              "task-c",
			SessionID:       "session-b",
			SourceMessageID: "msg-user-c",
			Status:          taskdomain.TaskStatusRunning,
			Progress:        30,
			RetryCount:      0,
			CreatedAt:       now.Add(-2 * time.Hour),
			UpdatedAt:       now.Add(-30 * time.Minute),
			RequestContent:  "other session",
			RequestMetadata: map[string]string{
				taskapp.MetadataTaskTriggerTypeKey: "system",
				taskapp.MetadataTaskChannelTypeKey: "cli",
				taskapp.MetadataTaskChannelIDKey:   "cli-default",
			},
		},
	}
	taskSvc := &stubWebTaskService{
		listFn: func(query taskapp.ListQuery) taskapp.TaskPage {
			items := make([]taskdomain.Task, 0, len(source))
			for _, item := range source {
				if query.SessionID != "" && item.SessionID != query.SessionID {
					continue
				}
				if query.Status.IsValid() && item.Status != query.Status {
					continue
				}
				items = append(items, item)
			}
			return taskapp.TaskPage{
				Items: items,
				Pagination: taskapp.Pagination{
					Page:     query.Page,
					PageSize: query.PageSize,
					Total:    len(items),
					HasNext:  false,
				},
			}
		},
	}
	server := &Server{
		tasks:  taskSvc,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/control/tasks?session_id=session-a&status=failed&trigger_type=cron&channel_type=scheduler&channel_id=scheduler-default&source_message_id=msg-user-b&message_id=msg-assistant-b&time_range=2026-03-04T05:00:00Z,2026-03-04T08:00:00Z&page=1&page_size=10",
		nil,
	)
	rec := httptest.NewRecorder()
	server.controlTaskCollectionHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var payload struct {
		Items      []controlTaskListItem `json:"items"`
		Pagination taskapp.Pagination    `json:"pagination"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode control task page failed: %v", err)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 task in filtered page, got %d", len(payload.Items))
	}
	if payload.Items[0].TaskID != "task-b" {
		t.Fatalf("expected filtered task-b, got %+v", payload.Items)
	}
	if payload.Items[0].SourceMessageID != "msg-user-b" {
		t.Fatalf("expected source_message_id msg-user-b, got %+v", payload.Items[0])
	}
	if payload.Items[0].Error != "network timeout" {
		t.Fatalf("expected error_message in list item, got %+v", payload.Items[0])
	}
	if payload.Items[0].TriggerType != shareddomain.TriggerTypeCron || payload.Items[0].ChannelType != shareddomain.ChannelTypeScheduler {
		t.Fatalf("expected trigger/channel fields in list item, got %+v", payload.Items[0])
	}
	if payload.Items[0].JobID != "job-nightly" || payload.Items[0].JobName != "Nightly Build" {
		t.Fatalf("expected cron fields in list item, got %+v", payload.Items[0])
	}
	if payload.Items[0].FiredAt.Format(time.RFC3339) != "2026-03-04T07:20:00Z" {
		t.Fatalf("expected fired_at in list item, got %+v", payload.Items[0])
	}
	if payload.Pagination.Total != 1 || payload.Pagination.HasNext {
		t.Fatalf("unexpected pagination %+v", payload.Pagination)
	}
}

func TestControlTaskCollectionRejectsInvalidTimeRange(t *testing.T) {
	server := &Server{
		tasks:  &stubWebTaskService{},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/control/tasks?time_range=invalid", nil)
	rec := httptest.NewRecorder()
	server.controlTaskCollectionHandler(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestControlTaskCollectionRejectsInvalidSourceFilters(t *testing.T) {
	server := &Server{
		tasks:  &stubWebTaskService{},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/control/tasks?trigger_type=manual", nil)
	rec := httptest.NewRecorder()
	server.controlTaskCollectionHandler(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/control/tasks?channel_type=mobile", nil)
	rec = httptest.NewRecorder()
	server.controlTaskCollectionHandler(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestSessionTaskMessageIDBacklinkQuery(t *testing.T) {
	taskSvc := &stubWebTaskService{
		bySession: map[string][]taskdomain.Task{
			"session-a": {
				{
					ID:              "task-1",
					SessionID:       "session-a",
					SourceMessageID: "msg-user-1",
					Status:          taskdomain.TaskStatusSuccess,
					MessageLink: taskdomain.TaskMessageLink{
						TaskID:           "task-1",
						SessionID:        "session-a",
						RequestMessageID: "msg-user-1",
						ResultMessageID:  "msg-assistant-1",
					},
				},
				{
					ID:              "task-2",
					SessionID:       "session-a",
					SourceMessageID: "msg-user-2",
					Status:          taskdomain.TaskStatusQueued,
					MessageLink: taskdomain.TaskMessageLink{
						TaskID:           "task-2",
						SessionID:        "session-a",
						RequestMessageID: "msg-user-2",
						ResultMessageID:  "",
					},
				},
			},
		},
	}
	server := &Server{
		tasks:  taskSvc,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/session-a/tasks?message_id=msg-user-2", nil)
	rec := httptest.NewRecorder()
	server.sessionMessageListHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"task-2"`) || strings.Contains(body, `"task-1"`) {
		t.Fatalf("expected filtered task payload, got %s", body)
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

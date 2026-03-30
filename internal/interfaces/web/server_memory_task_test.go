package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	taskapp "alter0/internal/task/application"
	taskdomain "alter0/internal/task/domain"
	tasksummaryapp "alter0/internal/tasksummary/application"
)

func TestMemoryTaskCollectionHandlerFiltersByStatusTypeAndTime(t *testing.T) {
	now := time.Date(2026, 3, 4, 9, 0, 0, 0, time.UTC)
	taskSvc := &stubWebTaskService{
		listPage: taskapp.TaskPage{
			Items: []taskdomain.Task{
				newMemoryTask("task-1", "release", taskdomain.TaskStatusSuccess, now.Add(-time.Hour)),
				newMemoryTask("task-2", "ops", taskdomain.TaskStatusFailed, now.Add(-time.Hour)),
				newMemoryTask("task-3", "release", taskdomain.TaskStatusSuccess, now.Add(-10*24*time.Hour)),
			},
			Pagination: taskapp.Pagination{Page: 1, PageSize: 200, Total: 3, HasNext: false},
		},
	}
	server := &Server{
		tasks:  taskSvc,
		memory: newAgentMemoryService(AgentMemoryOptions{}),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	startAt := now.Add(-48 * time.Hour).Format(time.RFC3339)
	endAt := now.Add(48 * time.Hour).Format(time.RFC3339)
	url := "/api/memory/tasks?status=success&task_type=release&start_at=" + startAt + "&end_at=" + endAt + "&page=1&page_size=10"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()

	server.memoryTaskCollectionHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	var body struct {
		Items      []memoryTaskSummaryItem `json:"items"`
		Pagination taskapp.Pagination      `json:"pagination"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(body.Items) != 1 {
		t.Fatalf("expected one filtered item, got %+v", body.Items)
	}
	if body.Items[0].TaskID != "task-1" {
		t.Fatalf("expected task-1, got %+v", body.Items[0])
	}
	if body.Items[0].LastHeartbeatAt.IsZero() || body.Items[0].TimeoutAt.IsZero() {
		t.Fatalf("expected heartbeat fields in summary item, got %+v", body.Items[0])
	}
	if body.Pagination.Total != 1 || body.Pagination.HasNext {
		t.Fatalf("unexpected pagination: %+v", body.Pagination)
	}
}

func TestMemoryTaskDetailAndRebuildSummaryEndpoints(t *testing.T) {
	now := time.Date(2026, 3, 4, 10, 0, 0, 0, time.UTC)
	task := newMemoryTask("task-detail", "release", taskdomain.TaskStatusSuccess, now)
	taskSvc := &stubWebTaskService{
		items: map[string]taskdomain.Task{
			"task-detail": task,
		},
		listPage: taskapp.TaskPage{
			Items:      []taskdomain.Task{task},
			Pagination: taskapp.Pagination{Page: 1, PageSize: 200, Total: 1, HasNext: false},
		},
	}

	root := t.TempDir()
	runtimeStore := tasksummaryapp.NewRuntimeMarkdownStore(tasksummaryapp.RuntimeMarkdownOptions{
		DailyDir:    filepath.Join(root, "memory"),
		LongTermDir: filepath.Join(root, "memory", "long-term"),
	})
	runtimeStore.Record(task)

	server := &Server{
		tasks: taskSvc,
		memory: newAgentMemoryService(AgentMemoryOptions{
			DailyDir:           filepath.Join(root, "memory"),
			TaskSummaryRuntime: runtimeStore,
		}),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	detailReq := httptest.NewRequest(http.MethodGet, "/api/memory/tasks/task-detail", nil)
	detailRec := httptest.NewRecorder()
	server.memoryTaskItemHandler(detailRec, detailReq)
	if detailRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, detailRec.Code)
	}

	var detailBody struct {
		Meta        memoryTaskMeta                    `json:"meta"`
		SummaryRefs []tasksummaryapp.SummaryReference `json:"summary_refs"`
	}
	if err := json.NewDecoder(detailRec.Body).Decode(&detailBody); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if detailBody.Meta.TaskID != "task-detail" {
		t.Fatalf("unexpected meta payload: %+v", detailBody.Meta)
	}
	if detailBody.Meta.LastHeartbeatAt.IsZero() {
		t.Fatalf("expected heartbeat timestamp in meta payload: %+v", detailBody.Meta)
	}
	if detailBody.Meta.TimeoutAt.IsZero() {
		t.Fatalf("expected timeout window in meta payload: %+v", detailBody.Meta)
	}
	if len(detailBody.SummaryRefs) == 0 {
		t.Fatalf("expected summary refs in detail response")
	}

	rebuildReq := httptest.NewRequest(http.MethodPost, "/api/memory/tasks/task-detail/rebuild-summary", nil)
	rebuildRec := httptest.NewRecorder()
	server.memoryTaskItemHandler(rebuildRec, rebuildReq)
	if rebuildRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rebuildRec.Code)
	}
	var rebuildBody struct {
		TaskID      string                            `json:"task_id"`
		Status      string                            `json:"status"`
		SummaryRefs []tasksummaryapp.SummaryReference `json:"summary_refs"`
	}
	if err := json.NewDecoder(rebuildRec.Body).Decode(&rebuildBody); err != nil {
		t.Fatalf("decode rebuild: %v", err)
	}
	if rebuildBody.TaskID != "task-detail" || rebuildBody.Status != "rebuilt" {
		t.Fatalf("unexpected rebuild payload: %+v", rebuildBody)
	}
	if len(rebuildBody.SummaryRefs) == 0 {
		t.Fatalf("expected rebuilt summary refs")
	}
}

func TestMemoryTaskLogsEndpointReturnsRebuildHintWhenUnavailable(t *testing.T) {
	server := &Server{
		tasks: &stubWebTaskService{
			items: map[string]taskdomain.Task{
				"task-missing-logs": newMemoryTask("task-missing-logs", "task", taskdomain.TaskStatusSuccess, time.Now().UTC()),
			},
		},
		memory: newAgentMemoryService(AgentMemoryOptions{}),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/memory/tasks/task-missing-logs/logs?cursor=0&limit=20", nil)
	rec := httptest.NewRecorder()
	server.memoryTaskItemHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error_code"] != "task_logs_unavailable" {
		t.Fatalf("expected error_code task_logs_unavailable, got %+v", body)
	}
	if body["rebuild_hint"] == "" {
		t.Fatalf("expected rebuild hint in response, got %+v", body)
	}
}

func newMemoryTask(taskID string, taskType string, status taskdomain.TaskStatus, finishedAt time.Time) taskdomain.Task {
	if finishedAt.IsZero() {
		finishedAt = time.Now().UTC()
	}
	return taskdomain.Task{
		ID:              taskID,
		SessionID:       "session-" + taskID,
		SourceMessageID: "message-" + taskID,
		MessageID:       "message-" + taskID,
		TaskType:        taskType,
		Status:          status,
		Progress:        100,
		MaxRetries:      1,
		TimeoutMS:       90000,
		CreatedAt:       finishedAt.Add(-time.Minute),
		UpdatedAt:       finishedAt,
		LastHeartbeatAt: finishedAt.Add(-10 * time.Second),
		TimeoutAt:       finishedAt.Add(2 * time.Minute),
		FinishedAt:      finishedAt,
		RequestContent:  "handle " + taskID,
		Summary:         "summary " + taskID,
		TaskSummary: taskdomain.TaskSummary{
			TaskID:     taskID,
			TaskType:   taskType,
			Goal:       "handle " + taskID,
			Result:     "result " + taskID,
			Status:     status,
			FinishedAt: finishedAt,
			Tags:       []string{"task", taskType},
		},
	}
}

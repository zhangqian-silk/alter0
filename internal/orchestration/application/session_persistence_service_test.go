package application

import (
	"context"
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

	sessiondomain "alter0/internal/session/domain"
	shareddomain "alter0/internal/shared/domain"
)

type stubPersistenceDownstream struct {
	result shareddomain.OrchestrationResult
	err    error
}

func (s *stubPersistenceDownstream) Handle(_ context.Context, _ shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error) {
	return s.result, s.err
}

type spySessionRecorder struct {
	records []sessiondomain.MessageRecord
	err     error
}

func (s *spySessionRecorder) Append(records ...sessiondomain.MessageRecord) error {
	s.records = append(s.records, records...)
	return s.err
}

type fixedIDGenerator struct {
	nextID string
}

func (g *fixedIDGenerator) NewID() string {
	return g.nextID
}

func TestSessionPersistenceServiceRecordsUserAndAssistantMessages(t *testing.T) {
	downstream := &stubPersistenceDownstream{
		result: shareddomain.OrchestrationResult{
			MessageID: "msg-1",
			SessionID: "s-1",
			Route:     shareddomain.RouteNL,
			Output:    "answer",
		},
	}
	recorder := &spySessionRecorder{}
	service := &SessionPersistenceService{
		downstream:  downstream,
		recorder:    recorder,
		idGenerator: &fixedIDGenerator{nextID: "assistant-1"},
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	msg := shareddomain.UnifiedMessage{
		MessageID:   "msg-1",
		SessionID:   "s-1",
		Content:     "question",
		ReceivedAt:  time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC),
		TriggerType: shareddomain.TriggerTypeUser,
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TraceID:     "trace-1",
	}

	if _, err := service.Handle(context.Background(), msg); err != nil {
		t.Fatalf("handle failed: %v", err)
	}
	if len(recorder.records) != 2 {
		t.Fatalf("expected 2 persisted records, got %d", len(recorder.records))
	}
	if recorder.records[0].Role != sessiondomain.MessageRoleUser || recorder.records[1].Role != sessiondomain.MessageRoleAssistant {
		t.Fatalf("unexpected roles: %+v", recorder.records)
	}
	if recorder.records[1].MessageID != "assistant-1" {
		t.Fatalf("expected assistant message id assistant-1, got %q", recorder.records[1].MessageID)
	}
	if recorder.records[1].RouteResult.Route != shareddomain.RouteNL {
		t.Fatalf("expected route nl, got %q", recorder.records[1].RouteResult.Route)
	}
	if recorder.records[0].Source.TriggerType != shareddomain.TriggerTypeUser {
		t.Fatalf("expected trigger_type user, got %s", recorder.records[0].Source.TriggerType)
	}
	if recorder.records[0].Source.ChannelType != shareddomain.ChannelTypeWeb {
		t.Fatalf("expected channel_type web, got %s", recorder.records[0].Source.ChannelType)
	}
}

func TestSessionPersistenceServiceKeepsResultWhenStoreFails(t *testing.T) {
	expectedErr := errors.New("nl failed")
	downstream := &stubPersistenceDownstream{
		result: shareddomain.OrchestrationResult{
			MessageID: "msg-1",
			SessionID: "s-1",
			Route:     shareddomain.RouteNL,
			ErrorCode: "nl_execution_failed",
		},
		err: expectedErr,
	}
	recorder := &spySessionRecorder{err: errors.New("disk unavailable")}
	service := &SessionPersistenceService{
		downstream:  downstream,
		recorder:    recorder,
		idGenerator: &fixedIDGenerator{nextID: "assistant-1"},
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	msg := shareddomain.UnifiedMessage{
		MessageID:   "msg-1",
		SessionID:   "s-1",
		Content:     "question",
		ReceivedAt:  time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC),
		TriggerType: shareddomain.TriggerTypeUser,
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TraceID:     "trace-1",
	}

	result, err := service.Handle(context.Background(), msg)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected downstream error %v, got %v", expectedErr, err)
	}
	if result.ErrorCode != "nl_execution_failed" {
		t.Fatalf("expected error_code nl_execution_failed, got %q", result.ErrorCode)
	}
	if len(recorder.records) != 2 {
		t.Fatalf("expected 2 persisted records, got %d", len(recorder.records))
	}
	if recorder.records[1].Content != expectedErr.Error() {
		t.Fatalf("expected assistant content %q, got %q", expectedErr.Error(), recorder.records[1].Content)
	}
}

func TestSessionPersistenceServicePersistsTaskIDFromMetadata(t *testing.T) {
	downstream := &stubPersistenceDownstream{
		result: shareddomain.OrchestrationResult{
			MessageID: "msg-task",
			SessionID: "s-task",
			Route:     shareddomain.RouteNL,
			Output:    "done",
		},
	}
	recorder := &spySessionRecorder{}
	service := &SessionPersistenceService{
		downstream:  downstream,
		recorder:    recorder,
		idGenerator: &fixedIDGenerator{nextID: "assistant-task"},
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	msg := shareddomain.UnifiedMessage{
		MessageID:   "msg-task",
		SessionID:   "s-task",
		Content:     "question",
		Metadata:    map[string]string{"task_id": "task-123"},
		ReceivedAt:  time.Date(2026, 3, 3, 13, 0, 0, 0, time.UTC),
		TriggerType: shareddomain.TriggerTypeUser,
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TraceID:     "trace-task",
	}

	if _, err := service.Handle(context.Background(), msg); err != nil {
		t.Fatalf("handle failed: %v", err)
	}
	if len(recorder.records) != 2 {
		t.Fatalf("expected 2 persisted records, got %d", len(recorder.records))
	}
	if recorder.records[0].RouteResult.TaskID != "task-123" {
		t.Fatalf("expected user record task id task-123, got %q", recorder.records[0].RouteResult.TaskID)
	}
	if recorder.records[1].RouteResult.TaskID != "task-123" {
		t.Fatalf("expected assistant record task id task-123, got %q", recorder.records[1].RouteResult.TaskID)
	}
}

func TestSessionPersistenceServicePersistsStructuredProcessSteps(t *testing.T) {
	downstream := &stubPersistenceDownstream{
		result: shareddomain.OrchestrationResult{
			MessageID: "msg-steps",
			SessionID: "s-steps",
			Route:     shareddomain.RouteNL,
			Output:    "任务已完成",
			ProcessSteps: []shareddomain.ProcessStep{
				{
					Kind:   "action",
					Title:  "codex_exec",
					Detail: "检查仓库状态",
					Status: "completed",
				},
			},
		},
	}
	recorder := &spySessionRecorder{}
	service := &SessionPersistenceService{
		downstream:  downstream,
		recorder:    recorder,
		idGenerator: &fixedIDGenerator{nextID: "assistant-steps"},
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	msg := shareddomain.UnifiedMessage{
		MessageID:   "msg-steps",
		SessionID:   "s-steps",
		Content:     "执行一次结构化任务",
		ReceivedAt:  time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC),
		TriggerType: shareddomain.TriggerTypeUser,
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TraceID:     "trace-steps",
	}

	if _, err := service.Handle(context.Background(), msg); err != nil {
		t.Fatalf("handle failed: %v", err)
	}
	if len(recorder.records) != 2 {
		t.Fatalf("expected 2 persisted records, got %d", len(recorder.records))
	}
	if len(recorder.records[1].RouteResult.ProcessSteps) != 1 {
		t.Fatalf("expected assistant process steps persisted, got %+v", recorder.records[1].RouteResult.ProcessSteps)
	}
	if recorder.records[1].RouteResult.ProcessSteps[0].Title != "codex_exec" {
		t.Fatalf("expected persisted step title codex_exec, got %+v", recorder.records[1].RouteResult.ProcessSteps[0])
	}
}

func TestSessionPersistenceServiceLocalizesAssistantMarkdownImagesIntoSessionWorkspace(t *testing.T) {
	t.Parallel()

	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("fake-image"))
	}))
	defer imageServer.Close()

	workspaceRoot := t.TempDir()
	downstream := &stubPersistenceDownstream{
		result: shareddomain.OrchestrationResult{
			MessageID: "msg-image",
			SessionID: "s-image",
			Route:     shareddomain.RouteNL,
			Output:    "Preview:\n\n![Generated image](" + imageServer.URL + "/generated.png)",
			ProcessSteps: []shareddomain.ProcessStep{
				{
					Kind:   "observation",
					Title:  "rendered",
					Detail: "Step image ![Trace](" + imageServer.URL + "/trace.png)",
					Status: "completed",
				},
			},
		},
	}
	recorder := &spySessionRecorder{}
	service := &SessionPersistenceService{
		downstream:    downstream,
		recorder:      recorder,
		idGenerator:   &fixedIDGenerator{nextID: "assistant-image"},
		logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		workspaceRoot: workspaceRoot,
		httpClient:    imageServer.Client(),
	}

	msg := shareddomain.UnifiedMessage{
		MessageID:   "msg-image",
		SessionID:   "s-image",
		Content:     "show the generated image",
		ReceivedAt:  time.Date(2026, 4, 23, 6, 0, 0, 0, time.UTC),
		TriggerType: shareddomain.TriggerTypeUser,
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TraceID:     "trace-image",
	}

	result, err := service.Handle(context.Background(), msg)
	if err != nil {
		t.Fatalf("handle failed: %v", err)
	}

	expectedPrefix := "/api/sessions/s-image/attachments/assistant-"
	if !strings.Contains(result.Output, expectedPrefix) {
		t.Fatalf("expected localized assistant image URL in result output, got %q", result.Output)
	}
	if len(result.ProcessSteps) != 1 || !strings.Contains(result.ProcessSteps[0].Detail, expectedPrefix) {
		t.Fatalf("expected localized process step image URL, got %+v", result.ProcessSteps)
	}
	if len(recorder.records) != 2 || !strings.Contains(recorder.records[1].Content, expectedPrefix) {
		t.Fatalf("expected persisted assistant record to contain localized image URL, got %+v", recorder.records)
	}

	attachmentRoot := filepath.Join(workspaceRoot, ".alter0", "workspaces", "sessions", "s-image", "attachments")
	entries, readErr := os.ReadDir(attachmentRoot)
	if readErr != nil {
		t.Fatalf("read attachment workspace: %v", readErr)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 localized assistant image assets, got %d", len(entries))
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			t.Fatalf("expected attachment asset dir, got %s", entry.Name())
		}
		matches, globErr := filepath.Glob(filepath.Join(attachmentRoot, entry.Name(), "original.*"))
		if globErr != nil || len(matches) != 1 {
			t.Fatalf("expected original image file for %s, got %v %v", entry.Name(), matches, globErr)
		}
	}
}

func TestSessionPersistenceServicePersistsCronSourceMetadata(t *testing.T) {
	downstream := &stubPersistenceDownstream{
		result: shareddomain.OrchestrationResult{
			MessageID: "msg-cron",
			SessionID: "s-cron",
			Route:     shareddomain.RouteCommand,
			Output:    "done",
		},
	}
	recorder := &spySessionRecorder{}
	service := &SessionPersistenceService{
		downstream:  downstream,
		recorder:    recorder,
		idGenerator: &fixedIDGenerator{nextID: "assistant-cron"},
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	msg := shareddomain.UnifiedMessage{
		MessageID:     "msg-cron",
		SessionID:     "s-cron",
		Content:       "/time",
		ReceivedAt:    time.Date(2026, 3, 5, 10, 0, 0, 0, time.UTC),
		TriggerType:   shareddomain.TriggerTypeCron,
		ChannelID:     "scheduler-default",
		ChannelType:   shareddomain.ChannelTypeScheduler,
		CorrelationID: "job-daily",
		TraceID:       "trace-cron",
		Metadata: map[string]string{
			"job_id":   "job-daily",
			"job_name": "Daily Report",
			"fired_at": "2026-03-05T10:00:00Z",
		},
	}

	if _, err := service.Handle(context.Background(), msg); err != nil {
		t.Fatalf("handle failed: %v", err)
	}
	if len(recorder.records) != 2 {
		t.Fatalf("expected 2 persisted records, got %d", len(recorder.records))
	}
	source := recorder.records[0].Source
	if source.TriggerType != shareddomain.TriggerTypeCron {
		t.Fatalf("expected trigger_type cron, got %s", source.TriggerType)
	}
	if source.JobID != "job-daily" {
		t.Fatalf("expected job id job-daily, got %s", source.JobID)
	}
	if source.JobName != "Daily Report" {
		t.Fatalf("expected job name Daily Report, got %s", source.JobName)
	}
	if source.FiredAt.IsZero() {
		t.Fatalf("expected fired_at to be set")
	}
}

func TestSessionPersistenceServicePersistsAgentSourceMetadata(t *testing.T) {
	downstream := &stubPersistenceDownstream{
		result: shareddomain.OrchestrationResult{
			MessageID: "msg-agent",
			SessionID: "s-agent",
			Route:     shareddomain.RouteNL,
			Output:    "done",
		},
	}
	recorder := &spySessionRecorder{}
	service := &SessionPersistenceService{
		downstream:  downstream,
		recorder:    recorder,
		idGenerator: &fixedIDGenerator{nextID: "assistant-agent"},
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	msg := shareddomain.UnifiedMessage{
		MessageID:   "msg-agent",
		SessionID:   "s-agent",
		Content:     "整理仓库",
		ReceivedAt:  time.Date(2026, 3, 5, 11, 0, 0, 0, time.UTC),
		TriggerType: shareddomain.TriggerTypeUser,
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TraceID:     "trace-agent",
		Metadata: map[string]string{
			"alter0.agent.id":   "researcher",
			"alter0.agent.name": "Research Agent",
		},
	}

	if _, err := service.Handle(context.Background(), msg); err != nil {
		t.Fatalf("handle failed: %v", err)
	}
	if len(recorder.records) != 2 {
		t.Fatalf("expected 2 persisted records, got %d", len(recorder.records))
	}
	source := recorder.records[0].Source
	if source.AgentID != "researcher" {
		t.Fatalf("expected agent_id researcher, got %s", source.AgentID)
	}
	if source.AgentName != "Research Agent" {
		t.Fatalf("expected agent_name Research Agent, got %s", source.AgentName)
	}
}

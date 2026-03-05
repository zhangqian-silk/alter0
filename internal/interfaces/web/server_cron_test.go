package web

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	schedulerapp "alter0/internal/scheduler/application"
	sessionapp "alter0/internal/session/application"
	sessiondomain "alter0/internal/session/domain"
	sharedapp "alter0/internal/shared/application"
	shareddomain "alter0/internal/shared/domain"
)

type stubCronOrchestrator struct{}

func (s *stubCronOrchestrator) Handle(_ context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error) {
	return shareddomain.OrchestrationResult{
		MessageID: msg.MessageID,
		SessionID: msg.SessionID,
		Route:     shareddomain.RouteCommand,
		Output:    "ok",
	}, nil
}

type stubCronTelemetry struct{}

func (s *stubCronTelemetry) CountGateway(string)                   {}
func (s *stubCronTelemetry) CountRoute(string)                     {}
func (s *stubCronTelemetry) CountCommand(string)                   {}
func (s *stubCronTelemetry) CountError(string)                     {}
func (s *stubCronTelemetry) CountMemoryEvent(string)               {}
func (s *stubCronTelemetry) ObserveDuration(string, time.Duration) {}

type stubCronIDGenerator struct{}

func (s *stubCronIDGenerator) NewID() string {
	return "cron-id"
}

func buildCronManagerForTest(t *testing.T) *schedulerapp.Manager {
	t.Helper()
	var telemetry sharedapp.Telemetry = &stubCronTelemetry{}
	return schedulerapp.NewManager(
		&stubCronOrchestrator{},
		telemetry,
		&stubCronIDGenerator{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
}

func TestCronJobHandlerSupportsVisualConfigPayload(t *testing.T) {
	server := &Server{
		scheduler: buildCronManagerForTest(t),
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/control/cron/jobs/job-daily",
		strings.NewReader(`{"name":"Daily Summary","enabled":true,"timezone":"Asia/Shanghai","schedule_mode":"daily","cron_expression":"30 9 * * *","task_config":{"input":"summarize latest tasks","retry_limit":2}}`),
	)
	rec := httptest.NewRecorder()
	server.cronJobItemHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload cronJobResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if payload.ScheduleMode != "daily" {
		t.Fatalf("expected schedule_mode daily, got %s", payload.ScheduleMode)
	}
	if payload.CronExpression != "30 9 * * *" {
		t.Fatalf("expected cron expression 30 9 * * *, got %s", payload.CronExpression)
	}
	if payload.TaskConfig.Input != "summarize latest tasks" {
		t.Fatalf("expected task input to be set, got %s", payload.TaskConfig.Input)
	}
	if payload.TaskConfig.RetryLimit != 2 {
		t.Fatalf("expected retry limit 2, got %d", payload.TaskConfig.RetryLimit)
	}
}

func TestCronJobRunsHandlerReturnsSessionLinks(t *testing.T) {
	history := &stubSessionHistory{
		sessionPage: sessionapp.SessionPage{
			Items: []sessiondomain.SessionSummary{
				{
					SessionID:     "cron-session-1",
					StartedAt:     time.Date(2026, 3, 5, 1, 30, 0, 0, time.UTC),
					LastMessageAt: time.Date(2026, 3, 5, 1, 31, 0, 0, time.UTC),
					TriggerType:   shareddomain.TriggerTypeCron,
					JobID:         "job-daily",
					FiredAt:       time.Date(2026, 3, 5, 1, 30, 0, 0, time.UTC),
				},
			},
			Pagination: sessionapp.Pagination{
				Page:     1,
				PageSize: 10,
				Total:    1,
			},
		},
	}

	server := &Server{
		scheduler: buildCronManagerForTest(t),
		sessions:  history,
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/control/cron/jobs/job-daily/runs?page=1&page_size=10", nil)
	rec := httptest.NewRecorder()
	server.cronJobItemHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if history.lastSessionQuery.TriggerType != shareddomain.TriggerTypeCron {
		t.Fatalf("expected trigger_type cron, got %s", history.lastSessionQuery.TriggerType)
	}
	if history.lastSessionQuery.JobID != "job-daily" {
		t.Fatalf("expected job_id job-daily, got %s", history.lastSessionQuery.JobID)
	}

	var body struct {
		Items []cronJobRunResponse `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body failed: %v", err)
	}
	if len(body.Items) != 1 {
		t.Fatalf("expected 1 run item, got %d", len(body.Items))
	}
	if body.Items[0].SessionID != "cron-session-1" {
		t.Fatalf("expected session_id cron-session-1, got %s", body.Items[0].SessionID)
	}
}

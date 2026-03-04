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
	schedulerdomain "alter0/internal/scheduler/domain"
	shareddomain "alter0/internal/shared/domain"
	"alter0/internal/shared/infrastructure/observability"
)

func TestCronJobItemHandlerSupportsVisualConfigFields(t *testing.T) {
	orchestrator := &stubOrchestrator{
		result: shareddomain.OrchestrationResult{
			Route:  shareddomain.RouteNL,
			Output: "ok",
		},
	}
	manager := schedulerapp.NewManager(
		orchestrator,
		observability.NewTelemetry(),
		&incrementalIDGenerator{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager.Start(ctx)

	server := &Server{
		scheduler: manager,
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/control/cron/jobs/job-daily",
		strings.NewReader(`{"name":"daily-report","schedule_mode":"daily","cron_expression":"30 9 * * *","timezone":"Asia/Shanghai","content":"/time","enabled":true,"task_config":{"retry_strategy":"once"}}`),
	)
	rec := httptest.NewRecorder()
	server.cronJobItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var payload cronJobResponse
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if payload.ID != "job-daily" {
		t.Fatalf("expected id job-daily, got %s", payload.ID)
	}
	if payload.ScheduleMode != schedulerdomain.ScheduleModeDaily {
		t.Fatalf("expected schedule_mode daily, got %s", payload.ScheduleMode)
	}
	if payload.CronExpression != "30 9 * * *" {
		t.Fatalf("expected cron_expression 30 9 * * *, got %s", payload.CronExpression)
	}
	if payload.Timezone != "Asia/Shanghai" {
		t.Fatalf("expected timezone Asia/Shanghai, got %s", payload.Timezone)
	}
	if payload.TaskConfig.RetryStrategy != "once" {
		t.Fatalf("expected retry strategy once, got %s", payload.TaskConfig.RetryStrategy)
	}
}

func TestCronJobRunsEndpointReturnsSessionLinks(t *testing.T) {
	orchestrator := &stubOrchestrator{
		result: shareddomain.OrchestrationResult{
			Route:  shareddomain.RouteNL,
			Output: "ok",
		},
	}
	manager := schedulerapp.NewManager(
		orchestrator,
		observability.NewTelemetry(),
		&incrementalIDGenerator{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager.Start(ctx)

	if err := manager.Upsert(schedulerdomain.Job{
		ID:       "job-run",
		Name:     "job-run",
		Interval: 20 * time.Millisecond,
		Enabled:  true,
		Content:  "/time",
	}); err != nil {
		t.Fatalf("upsert job failed: %v", err)
	}

	deadline := time.Now().Add(400 * time.Millisecond)
	for {
		if len(manager.ListRuns("job-run")) > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected at least one cron run in 400ms")
		}
		time.Sleep(10 * time.Millisecond)
	}

	server := &Server{
		scheduler: manager,
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/control/cron/jobs/job-run/runs", nil)
	rec := httptest.NewRecorder()
	server.cronJobItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var payload struct {
		Items []cronRunResponse `json:"items"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if len(payload.Items) == 0 {
		t.Fatalf("expected non-empty run list")
	}
	first := payload.Items[0]
	if first.JobID != "job-run" {
		t.Fatalf("expected job_id job-run, got %s", first.JobID)
	}
	if first.Status != schedulerdomain.RunStatusSuccess {
		t.Fatalf("expected status success, got %s", first.Status)
	}
	if !strings.HasPrefix(first.SessionID, "cron-job-run-") {
		t.Fatalf("expected dedicated session id, got %s", first.SessionID)
	}
}

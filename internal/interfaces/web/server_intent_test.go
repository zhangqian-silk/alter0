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

	orchdomain "alter0/internal/orchestration/domain"
	shareddomain "alter0/internal/shared/domain"
	"alter0/internal/shared/infrastructure/observability"
)

type stubIntentWebOrchestrator struct {
	stubWebOrchestrator
	intent orchdomain.Intent
}

func (s *stubIntentWebOrchestrator) Classify(_ string) orchdomain.Intent {
	return s.intent
}

func TestMessageHandlerRoutesCommandBeforeComplexityAssessment(t *testing.T) {
	orchestrator := &stubIntentWebOrchestrator{
		stubWebOrchestrator: stubWebOrchestrator{
			result: shareddomain.OrchestrationResult{
				MessageID: "message-generated",
				SessionID: "session-fixed",
				Route:     shareddomain.RouteCommand,
				Output:    "command-ok",
			},
		},
		intent: orchdomain.Intent{Type: orchdomain.IntentTypeCommand, CommandName: "help"},
	}
	tasks := &stubWebTaskService{shouldAsync: true}
	server := &Server{
		orchestrator: orchestrator,
		tasks:        tasks,
		telemetry:    observability.NewTelemetry(),
		idGenerator: &sequenceIDGenerator{
			ids: []string{"session-generated", "message-generated", "trace-generated"},
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/messages", strings.NewReader(`{"session_id":"session-fixed","content":"/help"}`))
	rec := httptest.NewRecorder()
	server.messageHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if tasks.submitCallCount != 0 {
		t.Fatalf("expected command route skip async submit, got %d submit calls", tasks.submitCallCount)
	}

	var body messageResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Result.Route != shareddomain.RouteCommand {
		t.Fatalf("expected command route, got %q", body.Result.Route)
	}
}

var _ interface {
	Handle(ctx context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error)
	Classify(content string) orchdomain.Intent
} = (*stubIntentWebOrchestrator)(nil)

package application

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	orchdomain "alter0/internal/orchestration/domain"
	shareddomain "alter0/internal/shared/domain"
)

type stubClassifier struct {
	intent orchdomain.Intent
}

func (s *stubClassifier) Classify(_ string) orchdomain.Intent {
	return s.intent
}

type stubRegistry struct {
	handlers map[string]orchdomain.CommandHandler
}

func (r *stubRegistry) Register(handler orchdomain.CommandHandler) error {
	if r.handlers == nil {
		r.handlers = map[string]orchdomain.CommandHandler{}
	}
	r.handlers[handler.Name()] = handler
	return nil
}

func (r *stubRegistry) Resolve(name string) (orchdomain.CommandHandler, bool) {
	if r.handlers == nil {
		return nil, false
	}
	handler, ok := r.handlers[name]
	return handler, ok
}

func (r *stubRegistry) Exists(name string) bool {
	_, ok := r.Resolve(name)
	return ok
}

func (r *stubRegistry) List() []string {
	return nil
}

type stubHandler struct {
	name   string
	output string
}

func (h *stubHandler) Name() string {
	return h.name
}

func (h *stubHandler) Aliases() []string {
	return nil
}

func (h *stubHandler) Execute(_ context.Context, _ orchdomain.CommandRequest) (orchdomain.CommandResult, error) {
	return orchdomain.CommandResult{Output: h.output}, nil
}

type stubExecutor struct {
	output string
	called int
}

func (e *stubExecutor) ExecuteNaturalLanguage(_ context.Context, _ shareddomain.UnifiedMessage) (string, error) {
	e.called++
	return e.output, nil
}

type spyTelemetry struct {
	routeCount map[string]int
	commandCnt map[string]int
	errorCount map[string]int
}

func newSpyTelemetry() *spyTelemetry {
	return &spyTelemetry{
		routeCount: map[string]int{},
		commandCnt: map[string]int{},
		errorCount: map[string]int{},
	}
}

func (t *spyTelemetry) CountGateway(_ string) {}

func (t *spyTelemetry) CountRoute(route string) {
	t.routeCount[route]++
}

func (t *spyTelemetry) CountCommand(command string) {
	t.commandCnt[command]++
}

func (t *spyTelemetry) CountError(route string) {
	t.errorCount[route]++
}

func (t *spyTelemetry) ObserveDuration(_ string, _ time.Duration) {}

func TestHandleCommand(t *testing.T) {
	registry := &stubRegistry{
		handlers: map[string]orchdomain.CommandHandler{
			"echo": &stubHandler{name: "echo", output: "ok"},
		},
	}
	telemetry := newSpyTelemetry()
	executor := &stubExecutor{output: "nl"}
	service := NewService(
		&stubClassifier{
			intent: orchdomain.Intent{
				Type:        orchdomain.IntentTypeCommand,
				CommandName: "echo",
			},
		},
		registry,
		executor,
		telemetry,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	result, err := service.Handle(context.Background(), validMessage("/echo hi"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Route != shareddomain.RouteCommand {
		t.Fatalf("expected command route, got %s", result.Route)
	}
	if result.Output != "ok" {
		t.Fatalf("unexpected output: %q", result.Output)
	}
	if executor.called != 0 {
		t.Fatalf("executor should not be called for command route")
	}
}

func TestHandleNL(t *testing.T) {
	registry := &stubRegistry{}
	telemetry := newSpyTelemetry()
	executor := &stubExecutor{output: "nl response"}
	service := NewService(
		&stubClassifier{
			intent: orchdomain.Intent{Type: orchdomain.IntentTypeNL},
		},
		registry,
		executor,
		telemetry,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	result, err := service.Handle(context.Background(), validMessage("hello"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Route != shareddomain.RouteNL {
		t.Fatalf("expected nl route, got %s", result.Route)
	}
	if result.Output != "nl response" {
		t.Fatalf("unexpected output: %q", result.Output)
	}
	if executor.called != 1 {
		t.Fatalf("executor should be called exactly once, got %d", executor.called)
	}
}

func TestHandleUnknownCommand(t *testing.T) {
	registry := &stubRegistry{}
	telemetry := newSpyTelemetry()
	service := NewService(
		&stubClassifier{
			intent: orchdomain.Intent{
				Type:        orchdomain.IntentTypeCommand,
				CommandName: "missing",
			},
		},
		registry,
		&stubExecutor{output: "nl"},
		telemetry,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	result, err := service.Handle(context.Background(), validMessage("/missing"))
	if err == nil {
		t.Fatalf("expected error for unknown command")
	}
	if result.ErrorCode != "command_not_found" {
		t.Fatalf("expected command_not_found, got %q", result.ErrorCode)
	}
}

func validMessage(content string) shareddomain.UnifiedMessage {
	return shareddomain.UnifiedMessage{
		MessageID:  "m1",
		SessionID:  "s1",
		Source:     shareddomain.SourceCLI,
		Content:    content,
		TraceID:    "t1",
		ReceivedAt: time.Now().UTC(),
	}
}

package application

import (
	"context"
	"io"
	"log/slog"
	"strings"
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
	output      string
	outputs     []string
	called      int
	lastMessage shareddomain.UnifiedMessage
}

func (e *stubExecutor) ExecuteNaturalLanguage(_ context.Context, msg shareddomain.UnifiedMessage) (string, error) {
	e.called++
	e.lastMessage = msg
	if len(e.outputs) > 0 {
		output := e.outputs[0]
		e.outputs = e.outputs[1:]
		return output, nil
	}
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

func TestHandleNLInjectsSessionMemory(t *testing.T) {
	registry := &stubRegistry{}
	telemetry := newSpyTelemetry()
	executor := &stubExecutor{
		outputs: []string{
			"plan alpha: blue green rollout",
			"refined execution plan",
		},
	}
	service := NewServiceWithOptions(
		&stubClassifier{
			intent: orchdomain.Intent{Type: orchdomain.IntentTypeNL},
		},
		registry,
		executor,
		telemetry,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		WithSessionMemoryOptions(SessionMemoryOptions{
			MaxTurns:    4,
			TTL:         30 * time.Minute,
			MaxSnippets: 180,
		}),
	)

	firstMessage := validMessage("Give me a release plan for this sprint")
	firstMessage.SessionID = "session-memory"
	if _, err := service.Handle(context.Background(), firstMessage); err != nil {
		t.Fatalf("first message failed: %v", err)
	}

	secondMessage := validMessage("Please expand that plan with rollback details")
	secondMessage.SessionID = "session-memory"
	result, err := service.Handle(context.Background(), secondMessage)
	if err != nil {
		t.Fatalf("second message failed: %v", err)
	}

	prompt := executor.lastMessage.Content
	if !strings.Contains(prompt, "Recent turns:") {
		t.Fatalf("expected recent turns in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "release plan for this sprint") {
		t.Fatalf("expected previous user message in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "plan alpha: blue green rollout") {
		t.Fatalf("expected previous assistant output in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "Reference resolution:") {
		t.Fatalf("expected reference resolution section, got %q", prompt)
	}
	if result.Metadata["memory_reference_resolved"] != "true" {
		t.Fatalf("expected memory_reference_resolved=true, got %q", result.Metadata["memory_reference_resolved"])
	}
}

func TestHandleNLMemoryKeepsStableReferenceAcrossFollowUps(t *testing.T) {
	registry := &stubRegistry{}
	telemetry := newSpyTelemetry()
	executor := &stubExecutor{
		outputs: []string{
			"plan alpha: baseline rollout",
			"plan alpha: add canary validation",
			"plan alpha: include release calendar",
		},
	}
	service := NewServiceWithOptions(
		&stubClassifier{
			intent: orchdomain.Intent{Type: orchdomain.IntentTypeNL},
		},
		registry,
		executor,
		telemetry,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		WithSessionMemoryOptions(SessionMemoryOptions{
			MaxTurns:    3,
			TTL:         time.Hour,
			MaxSnippets: 180,
		}),
	)

	messages := []string{
		"Draft a release plan",
		"Refine that plan with verification gates",
		"For that plan, add owner and timeline",
	}
	for _, content := range messages {
		msg := validMessage(content)
		msg.SessionID = "session-stable"
		if _, err := service.Handle(context.Background(), msg); err != nil {
			t.Fatalf("handle message %q failed: %v", content, err)
		}
	}

	prompt := executor.lastMessage.Content
	if !strings.Contains(prompt, "plan alpha: add canary validation") {
		t.Fatalf("expected latest plan context in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "Reference resolution:") {
		t.Fatalf("expected reference resolution in prompt, got %q", prompt)
	}
}

func validMessage(content string) shareddomain.UnifiedMessage {
	return shareddomain.UnifiedMessage{
		MessageID:   "m1",
		SessionID:   "s1",
		ChannelID:   "cli-default",
		ChannelType: shareddomain.ChannelTypeCLI,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     content,
		TraceID:     "t1",
		ReceivedAt:  time.Now().UTC(),
	}
}

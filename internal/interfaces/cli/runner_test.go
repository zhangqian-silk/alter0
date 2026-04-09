package cli

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	shareddomain "alter0/internal/shared/domain"
	"alter0/internal/shared/infrastructure/observability"
)

type fakeOrchestrator struct {
	result shareddomain.OrchestrationResult
	err    error
	got    []shareddomain.UnifiedMessage
}

func (f *fakeOrchestrator) Handle(_ context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error) {
	f.got = append(f.got, msg)
	return f.result, f.err
}

type sequenceIDGenerator struct {
	ids  []string
	next int
}

func (g *sequenceIDGenerator) NewID() string {
	if g.next >= len(g.ids) {
		return "extra-id"
	}
	id := g.ids[g.next]
	g.next++
	return id
}

func TestRunnerRunRoutesNonEmptyInputAsCLIUnifiedMessage(t *testing.T) {
	orchestrator := &fakeOrchestrator{
		result: shareddomain.OrchestrationResult{Route: shareddomain.RouteNL, Output: "handled"},
	}
	telemetry := observability.NewTelemetry()
	runner := NewRunner(
		orchestrator,
		telemetry,
		&sequenceIDGenerator{ids: []string{"session-1", "message-1", "trace-1"}},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	output := runRunnerWithInput(t, runner, "\n  hello cli  \n/quit\n")

	if len(orchestrator.got) != 1 {
		t.Fatalf("orchestrator calls = %d, want 1", len(orchestrator.got))
	}
	msg := orchestrator.got[0]
	if msg.MessageID != "message-1" || msg.SessionID != "session-1" || msg.TraceID != "trace-1" {
		t.Fatalf("message ids = (%q, %q, %q), want generated ids", msg.MessageID, msg.SessionID, msg.TraceID)
	}
	if msg.ChannelID != "cli-default" || msg.ChannelType != shareddomain.ChannelTypeCLI {
		t.Fatalf("channel = (%q, %q), want cli-default/cli", msg.ChannelID, msg.ChannelType)
	}
	if msg.TriggerType != shareddomain.TriggerTypeUser {
		t.Fatalf("TriggerType = %q, want user", msg.TriggerType)
	}
	if msg.Content != "hello cli" {
		t.Fatalf("Content = %q, want trimmed input", msg.Content)
	}
	if msg.ReceivedAt.IsZero() {
		t.Fatalf("ReceivedAt is zero, want timestamp")
	}
	if !strings.Contains(output, "alter0 cli session: session-1") {
		t.Fatalf("stdout missing session banner:\n%s", output)
	}
	if !strings.Contains(output, "handled") {
		t.Fatalf("stdout missing orchestrator output:\n%s", output)
	}

	metrics := renderMetrics(t, telemetry)
	if !strings.Contains(metrics, `alter0_gateway_messages_total{channel_type="cli"} 1`) {
		t.Fatalf("metrics missing cli gateway count:\n%s", metrics)
	}
}

func TestRunnerRunPrintsErrorAndContinuesUntilExit(t *testing.T) {
	orchestrator := &fakeOrchestrator{
		result: shareddomain.OrchestrationResult{ErrorCode: "execution_failed"},
		err:    errors.New("boom"),
	}
	runner := NewRunner(
		orchestrator,
		observability.NewTelemetry(),
		&sequenceIDGenerator{ids: []string{"session-1", "message-1", "trace-1"}},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	output := runRunnerWithInput(t, runner, "run failing task\n/exit\n")

	if len(orchestrator.got) != 1 {
		t.Fatalf("orchestrator calls = %d, want 1", len(orchestrator.got))
	}
	if !strings.Contains(output, "error: boom (execution_failed)") {
		t.Fatalf("stdout missing error output:\n%s", output)
	}
}

func runRunnerWithInput(t *testing.T, runner *Runner, input string) string {
	t.Helper()

	originalStdin := os.Stdin
	originalStdout := os.Stdout
	inputRead, inputWrite, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdin pipe: %v", err)
	}
	outputRead, outputWrite, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	defer func() {
		os.Stdin = originalStdin
		os.Stdout = originalStdout
		_ = inputRead.Close()
		_ = inputWrite.Close()
		_ = outputRead.Close()
		_ = outputWrite.Close()
	}()

	if _, err := inputWrite.WriteString(input); err != nil {
		t.Fatalf("write stdin: %v", err)
	}
	if err := inputWrite.Close(); err != nil {
		t.Fatalf("close stdin writer: %v", err)
	}

	os.Stdin = inputRead
	os.Stdout = outputWrite
	runErr := runner.Run(context.Background())
	if closeErr := outputWrite.Close(); closeErr != nil {
		t.Fatalf("close stdout writer: %v", closeErr)
	}
	if runErr != nil {
		t.Fatalf("Run() error = %v", runErr)
	}

	raw, err := io.ReadAll(outputRead)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return string(raw)
}

func renderMetrics(t *testing.T, telemetry *observability.Telemetry) string {
	t.Helper()
	response := httptest.NewRecorder()
	telemetry.MetricsHandler().ServeHTTP(response, httptest.NewRequest("GET", "/metrics", nil))
	return response.Body.String()
}

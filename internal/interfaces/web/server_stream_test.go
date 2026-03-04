package web

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	sharedapp "alter0/internal/shared/application"
	shareddomain "alter0/internal/shared/domain"
	"alter0/internal/shared/infrastructure/observability"
)

type stubOrchestrator struct {
	result shareddomain.OrchestrationResult
	err    error
	last   shareddomain.UnifiedMessage
}

func (s *stubOrchestrator) Handle(_ context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error) {
	s.last = msg
	if s.result.MessageID == "" {
		s.result.MessageID = msg.MessageID
	}
	if s.result.SessionID == "" {
		s.result.SessionID = msg.SessionID
	}
	return s.result, s.err
}

type stubStreamOrchestrator struct {
	stubOrchestrator
	events []string
}

func (s *stubStreamOrchestrator) HandleStream(
	_ context.Context,
	msg shareddomain.UnifiedMessage,
	onDelta func(string) error,
) (shareddomain.OrchestrationResult, error) {
	s.last = msg
	for _, item := range s.events {
		if err := onDelta(item); err != nil {
			return shareddomain.OrchestrationResult{}, err
		}
	}
	if s.result.MessageID == "" {
		s.result.MessageID = msg.MessageID
	}
	if s.result.SessionID == "" {
		s.result.SessionID = msg.SessionID
	}
	return s.result, s.err
}

type incrementalIDGenerator struct {
	next int
}

func (g *incrementalIDGenerator) NewID() string {
	g.next++
	return "id-" + strconv.Itoa(g.next)
}

func testWebServer(orch Orchestrator) *Server {
	var idGen sharedapp.IDGenerator = &incrementalIDGenerator{}
	return &Server{
		orchestrator: orch,
		telemetry:    observability.NewTelemetry(),
		idGenerator:  idGen,
		logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func TestMessageStreamHandlerSuccess(t *testing.T) {
	orch := &stubOrchestrator{
		result: shareddomain.OrchestrationResult{
			Route:  shareddomain.RouteNL,
			Output: "这是一次流式响应输出",
		},
	}
	server := testWebServer(orch)

	req := httptest.NewRequest(http.MethodPost, "/api/messages/stream", strings.NewReader(`{"session_id":"session-1","content":"你好"}`))
	rec := httptest.NewRecorder()

	server.messageStreamHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("expected text/event-stream, got %q", contentType)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "event: start") {
		t.Fatalf("expected start event, got body %q", body)
	}
	if !strings.Contains(body, "event: delta") {
		t.Fatalf("expected delta event, got body %q", body)
	}
	if !strings.Contains(body, "event: done") {
		t.Fatalf("expected done event, got body %q", body)
	}
	if !strings.Contains(body, `"output":"这是一次流式响应输出"`) {
		t.Fatalf("expected output in done payload, got body %q", body)
	}
}

func TestMessageStreamHandlerErrorEvent(t *testing.T) {
	orch := &stubOrchestrator{
		result: shareddomain.OrchestrationResult{
			Route:     shareddomain.RouteNL,
			ErrorCode: "nl_execution_failed",
		},
		err: errors.New("processor timeout"),
	}
	server := testWebServer(orch)

	req := httptest.NewRequest(http.MethodPost, "/api/messages/stream", strings.NewReader(`{"session_id":"session-2","content":"hello"}`))
	rec := httptest.NewRecorder()

	server.messageStreamHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "event: error") {
		t.Fatalf("expected error event, got body %q", body)
	}
	if strings.Contains(body, "event: done") {
		t.Fatalf("expected no done event on error, got body %q", body)
	}
	if !strings.Contains(body, `"error":"processor timeout"`) {
		t.Fatalf("expected error payload, got body %q", body)
	}
}

func TestMessageHandlerKeepsJSONCompatibility(t *testing.T) {
	orch := &stubOrchestrator{
		result: shareddomain.OrchestrationResult{
			Route:  shareddomain.RouteNL,
			Output: "fallback-response",
		},
	}
	server := testWebServer(orch)

	req := httptest.NewRequest(http.MethodPost, "/api/messages", strings.NewReader(`{"session_id":"session-3","content":"hi"}`))
	rec := httptest.NewRecorder()

	server.messageHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp messageResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Result.Output != "fallback-response" {
		t.Fatalf("expected fallback response output, got %q", resp.Result.Output)
	}
	if resp.Error != "" {
		t.Fatalf("expected empty error, got %q", resp.Error)
	}
}

func TestMessageStreamHandlerMethodNotAllowed(t *testing.T) {
	server := testWebServer(&stubOrchestrator{})

	req := httptest.NewRequest(http.MethodGet, "/api/messages/stream", nil)
	rec := httptest.NewRecorder()

	server.messageStreamHandler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestMessageStreamHandlerUsesStreamingOrchestrator(t *testing.T) {
	orch := &stubStreamOrchestrator{
		stubOrchestrator: stubOrchestrator{
			result: shareddomain.OrchestrationResult{
				Route:  shareddomain.RouteNL,
				Output: "stream-finished",
			},
		},
		events: []string{"part-1", "part-2"},
	}
	server := testWebServer(orch)

	req := httptest.NewRequest(http.MethodPost, "/api/messages/stream", strings.NewReader(`{"session_id":"session-stream","content":"hello"}`))
	rec := httptest.NewRecorder()

	server.messageStreamHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	if strings.Count(body, "event: delta") != 2 {
		t.Fatalf("expected two streamed delta events, got body %q", body)
	}
	if !strings.Contains(body, `"delta":"part-1"`) || !strings.Contains(body, `"delta":"part-2"`) {
		t.Fatalf("expected streamed delta payloads, got body %q", body)
	}
	if !strings.Contains(body, "event: done") {
		t.Fatalf("expected done event, got body %q", body)
	}
}

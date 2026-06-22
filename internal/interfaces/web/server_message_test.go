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

	shareddomain "alter0/internal/shared/domain"
	"alter0/internal/shared/infrastructure/observability"
)

type stubWebOrchestrator struct {
	result      shareddomain.OrchestrationResult
	err         error
	lastMessage shareddomain.UnifiedMessage
	lastCtxErr  error
	handleCount int
}

func (s *stubWebOrchestrator) Handle(ctx context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error) {
	s.lastMessage = msg
	s.lastCtxErr = ctx.Err()
	s.handleCount++
	return s.result, s.err
}

type sequenceIDGenerator struct {
	ids  []string
	next int
}

func (g *sequenceIDGenerator) NewID() string {
	if g.next >= len(g.ids) {
		id := "generated-" + strconv.Itoa(g.next)
		g.next++
		return id
	}
	id := g.ids[g.next]
	g.next++
	return id
}

func newMessageTestServer(orchestrator Orchestrator) *Server {
	return &Server{
		orchestrator: orchestrator,
		telemetry:    observability.NewTelemetry(),
		idGenerator: &sequenceIDGenerator{
			ids: []string{"session-generated", "message-generated", "trace-generated"},
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func TestMessageHandlerJSONFallbackPath(t *testing.T) {
	orchestrator := &stubWebOrchestrator{
		result: shareddomain.OrchestrationResult{
			MessageID: "message-generated",
			SessionID: "session-fixed",
			Route:     shareddomain.RouteCommand,
			Output:    "fallback-ok",
		},
	}
	server := newMessageTestServer(orchestrator)

	req := httptest.NewRequest(http.MethodPost, "/api/messages", strings.NewReader(`{"session_id":"session-fixed","content":"hello"}`))
	rec := httptest.NewRecorder()
	server.messageHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var body messageResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Result.Output != "fallback-ok" {
		t.Fatalf("expected output fallback-ok, got %q", body.Result.Output)
	}
	if body.Result.Route != shareddomain.RouteCommand {
		t.Fatalf("expected route command, got %q", body.Result.Route)
	}
	if orchestrator.lastMessage.SessionID != "session-fixed" {
		t.Fatalf("expected session-fixed, got %q", orchestrator.lastMessage.SessionID)
	}
	if orchestrator.lastMessage.ChannelID != "web-default" {
		t.Fatalf("expected default channel, got %q", orchestrator.lastMessage.ChannelID)
	}
}

func TestMessageHandlerUsesCanonicalChatSessionWhenRequestOmitsSessionID(t *testing.T) {
	orchestrator := &stubWebOrchestrator{
		result: shareddomain.OrchestrationResult{
			MessageID: "message-generated",
			Route:     shareddomain.RouteAgent,
			Output:    "ok",
		},
	}
	server := newMessageTestServer(orchestrator)

	req := httptest.NewRequest(http.MethodPost, "/api/messages", strings.NewReader(`{"content":"hello"}`))
	rec := httptest.NewRecorder()
	server.messageHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if orchestrator.lastMessage.SessionID != canonicalChatSessionID {
		t.Fatalf("expected canonical chat session %q, got %q", canonicalChatSessionID, orchestrator.lastMessage.SessionID)
	}
}

func TestMessageHandlerCarriesImageAttachmentsInMetadata(t *testing.T) {
	t.Parallel()

	orchestrator := &stubWebOrchestrator{
		result: shareddomain.OrchestrationResult{
			MessageID: "message-generated",
			SessionID: "session-fixed",
			Route:     shareddomain.RouteAgent,
			Output:    "ok",
		},
	}
	server := newMessageTestServer(orchestrator)

	req := httptest.NewRequest(http.MethodPost, "/api/messages", strings.NewReader(`{
		"session_id":"session-fixed",
		"content":"请分析这张图",
		"attachments":[
			{
				"name":"diagram.png",
				"content_type":"image/png",
				"data_url":"data:image/png;base64,ZmFrZQ=="
			}
		]
	}`))
	rec := httptest.NewRecorder()
	server.messageHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if orchestrator.lastMessage.Metadata == nil {
		t.Fatalf("expected metadata to be populated")
	}
	raw := orchestrator.lastMessage.Metadata["alter0.user_input.image_attachments"]
	if !strings.Contains(raw, `"name":"diagram.png"`) {
		t.Fatalf("expected attachment metadata, got %q", raw)
	}
}

func TestMessageHandlerUsesRequestContextForWebConversation(t *testing.T) {
	orchestrator := &stubWebOrchestrator{
		result: shareddomain.OrchestrationResult{
			MessageID: "message-generated",
			SessionID: "session-fixed",
			Route:     shareddomain.RouteAgent,
			Output:    "chat-ok",
		},
	}
	server := newMessageTestServer(orchestrator)

	req := httptest.NewRequest(http.MethodPost, "/api/messages", strings.NewReader(`{"session_id":"session-fixed","content":"hello"}`))
	canceledCtx, cancel := context.WithCancel(req.Context())
	cancel()
	req = req.WithContext(canceledCtx)
	rec := httptest.NewRecorder()

	server.messageHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if orchestrator.handleCount != 1 {
		t.Fatalf("expected orchestrator handle count 1, got %d", orchestrator.handleCount)
	}
	if !errors.Is(orchestrator.lastCtxErr, context.Canceled) {
		t.Fatalf("expected request context cancellation to reach orchestrator, got %v", orchestrator.lastCtxErr)
	}
}

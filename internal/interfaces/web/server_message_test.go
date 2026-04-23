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
	"time"

	agentapp "alter0/internal/agent/application"
	controlapp "alter0/internal/control/application"
	controldomain "alter0/internal/control/domain"
	execdomain "alter0/internal/execution/domain"
	shareddomain "alter0/internal/shared/domain"
	"alter0/internal/shared/infrastructure/observability"
)

type stubWebOrchestrator struct {
	result      shareddomain.OrchestrationResult
	err         error
	lastMessage shareddomain.UnifiedMessage
	delay       time.Duration
	lastCtxErr  error
	handleCount int
}

func (s *stubWebOrchestrator) Handle(ctx context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error) {
	s.lastMessage = msg
	s.lastCtxErr = ctx.Err()
	s.handleCount++
	if s.delay > 0 {
		time.Sleep(s.delay)
	}
	return s.result, s.err
}

type stubWebStreamOrchestrator struct {
	stubWebOrchestrator
	events       []string
	streamEvents []shareddomain.StreamEvent
	deltaCalls   int
	callbackErr  error
}

func (s *stubWebStreamOrchestrator) HandleStream(
	ctx context.Context,
	msg shareddomain.UnifiedMessage,
	onEvent func(shareddomain.StreamEvent) error,
) (shareddomain.OrchestrationResult, error) {
	s.lastMessage = msg
	s.lastCtxErr = ctx.Err()
	s.handleCount++
	if len(s.streamEvents) > 0 {
		for _, item := range s.streamEvents {
			if item.Type == shareddomain.StreamEventTypeOutput {
				s.deltaCalls++
			}
			if err := onEvent(item); err != nil {
				s.callbackErr = err
				return shareddomain.OrchestrationResult{}, err
			}
		}
		return s.result, s.err
	}
	for _, item := range s.events {
		s.deltaCalls++
		if err := onEvent(shareddomain.StreamEvent{
			Type: shareddomain.StreamEventTypeOutput,
			Text: item,
		}); err != nil {
			s.callbackErr = err
			return shareddomain.OrchestrationResult{}, err
		}
	}
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

func newAgentControlForTests(t *testing.T, agent controldomain.Agent) *controlapp.Service {
	t.Helper()

	control := controlapp.NewService()
	if err := control.UpsertChannel(controldomain.Channel{
		ID:      "web-default",
		Type:    shareddomain.ChannelTypeWeb,
		Enabled: true,
	}); err != nil {
		t.Fatalf("upsert channel failed: %v", err)
	}
	if err := control.UpsertAgent(agent); err != nil {
		t.Fatalf("upsert agent failed: %v", err)
	}
	return control
}

type failingSSEWriter struct {
	header     http.Header
	writeCount int
	failAfter  int
}

func (w *failingSSEWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *failingSSEWriter) Write(p []byte) (int, error) {
	w.writeCount++
	if w.failAfter > 0 && w.writeCount > w.failAfter {
		return 0, errors.New("client disconnected")
	}
	return len(p), nil
}

func (w *failingSSEWriter) WriteHeader(_ int) {}

func (w *failingSSEWriter) Flush() {}

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

func TestMessageHandlerCarriesImageAttachmentsInMetadata(t *testing.T) {
	t.Parallel()

	orchestrator := &stubWebOrchestrator{
		result: shareddomain.OrchestrationResult{
			MessageID: "message-generated",
			SessionID: "session-fixed",
			Route:     shareddomain.RouteNL,
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

func TestMessageStreamHandlerEmitsStartDeltaDone(t *testing.T) {
	output := strings.Repeat("stream-", 10)
	orchestrator := &stubWebOrchestrator{
		result: shareddomain.OrchestrationResult{
			MessageID: "message-generated",
			SessionID: "session-fixed",
			Route:     shareddomain.RouteNL,
			Output:    output,
		},
	}
	server := newMessageTestServer(orchestrator)

	req := httptest.NewRequest(http.MethodPost, "/api/messages/stream", strings.NewReader(`{"session_id":"session-fixed","content":"hello"}`))
	rec := httptest.NewRecorder()
	server.messageStreamHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("expected text/event-stream, got %q", contentType)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "event: start\n") {
		t.Fatalf("expected start event, got body: %s", body)
	}
	if strings.Count(body, "event: delta\n") < 2 {
		t.Fatalf("expected multiple delta events, got body: %s", body)
	}
	if !strings.Contains(body, "event: done\n") {
		t.Fatalf("expected done event, got body: %s", body)
	}
	if !strings.Contains(body, `"output":"`+output+`"`) {
		t.Fatalf("expected full output in done payload, got body: %s", body)
	}
}

func TestMessageStreamHandlerEmitsError(t *testing.T) {
	orchestrator := &stubWebOrchestrator{
		result: shareddomain.OrchestrationResult{
			MessageID: "message-generated",
			SessionID: "session-fixed",
			Route:     shareddomain.RouteNL,
			ErrorCode: "nl_execution_failed",
		},
		err: errors.New("processor failed"),
	}
	server := newMessageTestServer(orchestrator)

	req := httptest.NewRequest(http.MethodPost, "/api/messages/stream", strings.NewReader(`{"session_id":"session-fixed","content":"hello"}`))
	rec := httptest.NewRecorder()
	server.messageStreamHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "event: start\n") {
		t.Fatalf("expected start event before failure, got body: %s", body)
	}
	if !strings.Contains(body, "event: error\n") {
		t.Fatalf("expected error event, got body: %s", body)
	}
	if strings.Contains(body, "event: done\n") {
		t.Fatalf("did not expect done event on error, got body: %s", body)
	}
	if !strings.Contains(body, `"error":"processor failed"`) {
		t.Fatalf("expected error message in payload, got body: %s", body)
	}
}

func TestMessageStreamHandlerEmitsKeepAliveWhileWaiting(t *testing.T) {
	previousInterval := sseHeartbeatInterval
	sseHeartbeatInterval = 10 * time.Millisecond
	defer func() {
		sseHeartbeatInterval = previousInterval
	}()

	orchestrator := &stubWebOrchestrator{
		result: shareddomain.OrchestrationResult{
			MessageID: "message-generated",
			SessionID: "session-fixed",
			Route:     shareddomain.RouteNL,
			Output:    "slow-response",
		},
		delay: 35 * time.Millisecond,
	}
	server := newMessageTestServer(orchestrator)

	req := httptest.NewRequest(http.MethodPost, "/api/messages/stream", strings.NewReader(`{"session_id":"session-fixed","content":"hello"}`))
	rec := httptest.NewRecorder()
	server.messageStreamHandler(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, ": keep-alive\n\n") {
		t.Fatalf("expected keep-alive comment in stream body, got: %s", body)
	}
	if !strings.Contains(body, "event: done\n") {
		t.Fatalf("expected done event after keep-alive, got body: %s", body)
	}
}

func TestAgentMessageHandlerInjectsAgentProfileMetadata(t *testing.T) {
	orchestrator := &stubWebOrchestrator{
		result: shareddomain.OrchestrationResult{
			MessageID: "message-generated",
			SessionID: "session-fixed",
			Route:     shareddomain.RouteNL,
			Output:    "agent-ok",
		},
	}
	control := controlapp.NewService()
	if err := control.UpsertChannel(controldomain.Channel{
		ID:      "web-default",
		Type:    shareddomain.ChannelTypeWeb,
		Enabled: true,
	}); err != nil {
		t.Fatalf("upsert channel failed: %v", err)
	}
	if err := control.UpsertAgent(controldomain.Agent{
		ID:            "researcher",
		Name:          "Researcher",
		Type:          controldomain.CapabilityTypeAgent,
		Enabled:       true,
		Scope:         controldomain.CapabilityScopeGlobal,
		Version:       "v1.0.0",
		ProviderID:    "openai",
		Model:         "gpt-4o",
		SystemPrompt:  "Execute first.",
		MaxIterations: 7,
		Tools:         []string{"codex_exec"},
		Skills:        []string{"summary"},
		MCPs:          []string{"github"},
		MemoryFiles:   []string{"user_md", "soul_md"},
	}); err != nil {
		t.Fatalf("upsert agent failed: %v", err)
	}
	server := newMessageTestServer(orchestrator)
	server.control = control
	server.agents = agentapp.NewCatalog(control)

	req := httptest.NewRequest(http.MethodPost, "/api/agent/messages", strings.NewReader(`{"agent_id":"researcher","session_id":"session-fixed","content":"完成仓库整理"}`))
	rec := httptest.NewRecorder()
	server.agentMessageHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if orchestrator.lastMessage.Metadata[execdomain.ExecutionEngineMetadataKey] != execdomain.ExecutionEngineAgent {
		t.Fatalf("expected agent execution engine, got %+v", orchestrator.lastMessage.Metadata)
	}
	if orchestrator.lastMessage.Metadata[execdomain.AgentIDMetadataKey] != "researcher" {
		t.Fatalf("expected agent id researcher, got %+v", orchestrator.lastMessage.Metadata)
	}
	if orchestrator.lastMessage.Metadata[execdomain.AgentSystemPromptMetadataKey] != "Execute first." {
		t.Fatalf("expected system prompt injected, got %+v", orchestrator.lastMessage.Metadata)
	}
	if !strings.Contains(orchestrator.lastMessage.Metadata["alter0.skills.include"], "summary") {
		t.Fatalf("expected skill selection injected, got %+v", orchestrator.lastMessage.Metadata)
	}
	if !strings.Contains(orchestrator.lastMessage.Metadata["alter0.mcp.request.enable"], "github") {
		t.Fatalf("expected mcp selection injected, got %+v", orchestrator.lastMessage.Metadata)
	}
	if !strings.Contains(orchestrator.lastMessage.Metadata["alter0.memory.include"], "user_md") {
		t.Fatalf("expected memory file selection injected, got %+v", orchestrator.lastMessage.Metadata)
	}
}

func TestAgentMessageHandlerIgnoresCanceledRequestContext(t *testing.T) {
	orchestrator := &stubWebOrchestrator{
		result: shareddomain.OrchestrationResult{
			MessageID: "message-generated",
			SessionID: "session-fixed",
			Route:     shareddomain.RouteNL,
			Output:    "agent-ok",
		},
	}
	control := newAgentControlForTests(t, controldomain.Agent{
		ID:         "researcher",
		Name:       "Researcher",
		Type:       controldomain.CapabilityTypeAgent,
		Enabled:    true,
		Scope:      controldomain.CapabilityScopeGlobal,
		Version:    "v1.0.0",
		ProviderID: "openai",
		Model:      "gpt-4o",
		Tools:      []string{"codex_exec"},
	})
	server := newMessageTestServer(orchestrator)
	server.control = control
	server.agents = agentapp.NewCatalog(control)

	req := httptest.NewRequest(http.MethodPost, "/api/agent/messages", strings.NewReader(`{"agent_id":"researcher","session_id":"session-fixed","content":"完成仓库整理"}`))
	canceledCtx, cancel := context.WithCancel(req.Context())
	cancel()
	req = req.WithContext(canceledCtx)
	rec := httptest.NewRecorder()

	server.agentMessageHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if orchestrator.handleCount != 1 {
		t.Fatalf("expected orchestrator handle count 1, got %d", orchestrator.handleCount)
	}
	if orchestrator.lastCtxErr != nil {
		t.Fatalf("expected detached execution context, got %v", orchestrator.lastCtxErr)
	}
}

func TestAgentMessageStreamHandlerContinuesWhenStreamWriteFails(t *testing.T) {
	orchestrator := &stubWebStreamOrchestrator{
		stubWebOrchestrator: stubWebOrchestrator{
			result: shareddomain.OrchestrationResult{
				MessageID: "message-generated",
				SessionID: "session-fixed",
				Route:     shareddomain.RouteNL,
				Output:    "agent-ok",
			},
		},
		events: []string{"part-1", "part-2"},
	}
	control := newAgentControlForTests(t, controldomain.Agent{
		ID:         "researcher",
		Name:       "Researcher",
		Type:       controldomain.CapabilityTypeAgent,
		Enabled:    true,
		Scope:      controldomain.CapabilityScopeGlobal,
		Version:    "v1.0.0",
		ProviderID: "openai",
		Model:      "gpt-4o",
		Tools:      []string{"codex_exec"},
	})
	server := newMessageTestServer(orchestrator)
	server.control = control
	server.agents = agentapp.NewCatalog(control)

	req := httptest.NewRequest(http.MethodPost, "/api/agent/messages/stream", strings.NewReader(`{"agent_id":"researcher","session_id":"session-fixed","content":"完成仓库整理"}`))
	rec := &failingSSEWriter{failAfter: 2}

	server.agentMessageStreamHandler(rec, req)

	if orchestrator.handleCount != 1 {
		t.Fatalf("expected orchestrator handle count 1, got %d", orchestrator.handleCount)
	}
	if orchestrator.deltaCalls != 2 {
		t.Fatalf("expected all deltas to be processed, got %d", orchestrator.deltaCalls)
	}
	if orchestrator.callbackErr != nil {
		t.Fatalf("expected stream write errors to be swallowed, got %v", orchestrator.callbackErr)
	}
	if orchestrator.lastCtxErr != nil {
		t.Fatalf("expected detached execution context, got %v", orchestrator.lastCtxErr)
	}
}

func TestAgentMessageStreamHandlerEmitsStructuredProcessEvents(t *testing.T) {
	orchestrator := &stubWebStreamOrchestrator{
		stubWebOrchestrator: stubWebOrchestrator{
			result: shareddomain.OrchestrationResult{
				MessageID: "message-generated",
				SessionID: "session-fixed",
				Route:     shareddomain.RouteNL,
				Output:    "agent-ok",
			},
		},
		streamEvents: []shareddomain.StreamEvent{
			{
				Type: shareddomain.StreamEventTypeProcess,
				ProcessStep: &shareddomain.ProcessStep{
					ID:     "step-1",
					Kind:   "action",
					Title:  "codex_exec",
					Status: "running",
				},
			},
			{
				Type: shareddomain.StreamEventTypeOutput,
				Text: "agent-ok",
			},
		},
	}
	control := newAgentControlForTests(t, controldomain.Agent{
		ID:         "researcher",
		Name:       "Researcher",
		Type:       controldomain.CapabilityTypeAgent,
		Enabled:    true,
		Scope:      controldomain.CapabilityScopeGlobal,
		Version:    "v1.0.0",
		ProviderID: "openai",
		Model:      "gpt-4o",
		Tools:      []string{"codex_exec"},
	})
	server := newMessageTestServer(orchestrator)
	server.control = control
	server.agents = agentapp.NewCatalog(control)

	req := httptest.NewRequest(http.MethodPost, "/api/agent/messages/stream", strings.NewReader(`{"agent_id":"researcher","session_id":"session-fixed","content":"完成仓库整理"}`))
	rec := httptest.NewRecorder()

	server.agentMessageStreamHandler(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "event: process\n") {
		t.Fatalf("expected process event, got body %q", body)
	}
	if !strings.Contains(body, `"title":"codex_exec"`) {
		t.Fatalf("expected process step title in body, got %q", body)
	}
}

func TestAgentMessageHandlerKeepsExplicitRuntimeSelections(t *testing.T) {
	orchestrator := &stubWebOrchestrator{
		result: shareddomain.OrchestrationResult{
			MessageID: "message-generated",
			SessionID: "session-fixed",
			Route:     shareddomain.RouteNL,
			Output:    "agent-ok",
		},
	}
	control := controlapp.NewService()
	if err := control.UpsertChannel(controldomain.Channel{
		ID:      "web-default",
		Type:    shareddomain.ChannelTypeWeb,
		Enabled: true,
	}); err != nil {
		t.Fatalf("upsert channel failed: %v", err)
	}
	if err := control.UpsertAgent(controldomain.Agent{
		ID:         "researcher",
		Name:       "Researcher",
		Type:       controldomain.CapabilityTypeAgent,
		Enabled:    true,
		Scope:      controldomain.CapabilityScopeGlobal,
		Version:    "v1.0.0",
		ProviderID: "openai",
		Model:      "gpt-4o",
		Tools:      []string{"codex_exec"},
		Skills:     []string{"summary"},
		MCPs:       []string{"github"},
	}); err != nil {
		t.Fatalf("upsert agent failed: %v", err)
	}
	server := newMessageTestServer(orchestrator)
	server.control = control
	server.agents = agentapp.NewCatalog(control)

	req := httptest.NewRequest(http.MethodPost, "/api/agent/messages", strings.NewReader(`{
		"agent_id":"researcher",
		"session_id":"session-fixed",
		"content":"完成仓库整理",
		"metadata":{
			"alter0.agent.tools":"[]",
			"alter0.skills.include":"[\"planner\"]",
			"alter0.mcp.request.enable":"[]"
		}
	}`))
	rec := httptest.NewRecorder()
	server.agentMessageHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if got := orchestrator.lastMessage.Metadata[execdomain.AgentToolsMetadataKey]; got != "[]" {
		t.Fatalf("expected explicit tool selection preserved, got %q", got)
	}
	if got := orchestrator.lastMessage.Metadata["alter0.skills.include"]; got != "[\"planner\"]" {
		t.Fatalf("expected explicit skill selection preserved, got %q", got)
	}
	if got := orchestrator.lastMessage.Metadata["alter0.mcp.request.enable"]; got != "[]" {
		t.Fatalf("expected explicit mcp selection preserved, got %q", got)
	}
}

package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	shareddomain "alter0/internal/shared/domain"
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

func TestMessageStreamEndpointIsGone(t *testing.T) {
	server := newMessageTestServer(&stubWebOrchestrator{})

	req := httptest.NewRequest(http.MethodPost, "/api/messages/stream", nil)
	rec := httptest.NewRecorder()

	server.messageStreamHandler(rec, req)

	if rec.Code != http.StatusGone {
		t.Fatalf("expected status %d, got %d", http.StatusGone, rec.Code)
	}
	if contentType := rec.Header().Get("Content-Type"); contentType == "text/event-stream" {
		t.Fatalf("did not expect Chat stream content type")
	}
}

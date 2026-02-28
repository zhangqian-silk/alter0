package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"alter0/app/pkg/types"
)

func TestHandleStatusReturnsJSONSnapshot(t *testing.T) {
	ch := NewHTTPChannel(8080)
	ch.startedUnix.Store(time.Now().Add(-5 * time.Second).Unix())

	ch.pendingMu.Lock()
	ch.pending["req-1"] = make(chan types.Message)
	ch.pending["req-2"] = make(chan types.Message)
	ch.pendingMu.Unlock()

	ch.SetStatusProvider(func(ctx context.Context) map[string]interface{} {
		return map[string]interface{}{"ok": true}
	})

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rr := httptest.NewRecorder()
	ch.handleStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", rr.Code)
	}
	if !strings.Contains(rr.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("unexpected content type: %s", rr.Header().Get("Content-Type"))
	}

	var payload statusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if payload.ChannelID != "http" {
		t.Fatalf("unexpected channel id: %s", payload.ChannelID)
	}
	if payload.PendingRequests != 2 {
		t.Fatalf("unexpected pending requests: %d", payload.PendingRequests)
	}
	if payload.StartedAt == "" {
		t.Fatal("expected started_at to be set")
	}
	if payload.UptimeSec <= 0 {
		t.Fatalf("expected positive uptime, got %d", payload.UptimeSec)
	}
	if payload.Runtime == nil {
		t.Fatal("expected runtime payload")
	}
	ok, found := payload.Runtime["ok"].(bool)
	if !found || !ok {
		t.Fatalf("unexpected runtime payload: %+v", payload.Runtime)
	}
}

func TestHandleStatusRejectsNonGet(t *testing.T) {
	ch := NewHTTPChannel(8080)
	req := httptest.NewRequest(http.MethodPost, "/api/status", nil)
	rr := httptest.NewRecorder()
	ch.handleStatus(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for non-GET, got %d", rr.Code)
	}
}

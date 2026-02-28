package http

import (
	"bytes"
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

func TestHandleMessageReturnsJSONResponse(t *testing.T) {
	ch := NewHTTPChannel(8080)
	ch.handler = func(msg types.Message) {
		err := ch.Send(context.Background(), types.Message{
			RequestID: msg.RequestID,
			TaskID:    "task-1",
			Content:   "hello",
			Meta: map[string]interface{}{
				"closed":   false,
				"decision": "existing",
			},
		})
		if err != nil {
			t.Fatalf("send response failed: %v", err)
		}
	}

	body := []byte(`{"content":"hi","user_id":"u-1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/message", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	ch.handleMessage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("unexpected content type: %s", rr.Header().Get("Content-Type"))
	}

	var payload outgoingResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if payload.TaskID != "task-1" || payload.Response != "hello" || payload.Closed || payload.Decision != "existing" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestHandleMessageStreamModeReturnsChunks(t *testing.T) {
	ch := NewHTTPChannel(8080)
	ch.handler = func(msg types.Message) {
		err := ch.Send(context.Background(), types.Message{
			RequestID: msg.RequestID,
			TaskID:    "task-9",
			Content:   "abcdefghij",
			Meta: map[string]interface{}{
				"closed":   true,
				"decision": "new",
			},
		})
		if err != nil {
			t.Fatalf("send response failed: %v", err)
		}
	}

	body := []byte(`{"content":"hi","user_id":"u-1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/message?stream=1&chunk_size=4", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	ch.handleMessage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Header().Get("Content-Type"), "application/x-ndjson") {
		t.Fatalf("unexpected content type: %s", rr.Header().Get("Content-Type"))
	}

	decoder := json.NewDecoder(strings.NewReader(rr.Body.String()))
	events := make([]streamResponseEvent, 0, 4)
	for decoder.More() {
		var ev streamResponseEvent
		if err := decoder.Decode(&ev); err != nil {
			t.Fatalf("decode stream event failed: %v", err)
		}
		events = append(events, ev)
	}

	if len(events) != 4 {
		t.Fatalf("expected 4 events, got %d: %+v", len(events), events)
	}
	if events[0].Type != "chunk" || events[0].Chunk != "abcd" {
		t.Fatalf("unexpected first event: %+v", events[0])
	}
	if events[1].Chunk != "efgh" || events[2].Chunk != "ij" {
		t.Fatalf("unexpected chunk sequence: %+v", events)
	}
	done := events[3]
	if done.Type != "done" || done.TaskID != "task-9" || done.Decision != "new" || !done.Closed || done.Total != 3 {
		t.Fatalf("unexpected done event: %+v", done)
	}
}

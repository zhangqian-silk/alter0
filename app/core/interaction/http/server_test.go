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

func TestSetShutdownTimeout(t *testing.T) {
	ch := NewHTTPChannel(8080)
	if ch.shutdownTimeout != 5*time.Second {
		t.Fatalf("unexpected default shutdown timeout: %s", ch.shutdownTimeout)
	}

	ch.SetShutdownTimeout(12 * time.Second)
	if ch.shutdownTimeout != 12*time.Second {
		t.Fatalf("unexpected shutdown timeout after set: %s", ch.shutdownTimeout)
	}

	ch.SetShutdownTimeout(0)
	if ch.shutdownTimeout != 12*time.Second {
		t.Fatalf("zero timeout should be ignored, got: %s", ch.shutdownTimeout)
	}
}

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

func TestAsyncTaskSubmitAndStatus(t *testing.T) {
	ch := NewHTTPChannel(8080)
	ch.handler = func(msg types.Message) {
		err := ch.Send(context.Background(), types.Message{
			RequestID: msg.RequestID,
			TaskID:    "task-async",
			Content:   "done",
			Meta: map[string]interface{}{
				"closed":   false,
				"decision": "existing",
			},
		})
		if err != nil {
			t.Fatalf("send response failed: %v", err)
		}
	}

	body := []byte(`{"content":"async","user_id":"u-2"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	ch.handleAsyncTasks(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("unexpected status code: %d body=%s", rr.Code, rr.Body.String())
	}

	var submit asyncSubmitResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &submit); err != nil {
		t.Fatalf("decode submit response failed: %v", err)
	}
	if submit.RequestID == "" || submit.Status != "pending" {
		t.Fatalf("unexpected submit payload: %+v", submit)
	}

	time.Sleep(30 * time.Millisecond)

	statusReq := httptest.NewRequest(http.MethodGet, submit.StatusURL, nil)
	statusRR := httptest.NewRecorder()
	ch.handleAsyncTasks(statusRR, statusReq)
	if statusRR.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d body=%s", statusRR.Code, statusRR.Body.String())
	}

	var status asyncTaskResponse
	if err := json.Unmarshal(statusRR.Body.Bytes(), &status); err != nil {
		t.Fatalf("decode status response failed: %v", err)
	}
	if status.Status != "completed" {
		t.Fatalf("expected completed status, got %+v", status)
	}
	if status.Result == nil || status.Result.TaskID != "task-async" || status.Result.Response != "done" {
		t.Fatalf("unexpected async result: %+v", status.Result)
	}
	if status.Content != "async" || status.UserID != "u-2" {
		t.Fatalf("unexpected async metadata: %+v", status)
	}
}

func TestAsyncTaskCancel(t *testing.T) {
	ch := NewHTTPChannel(8080)
	ch.handler = func(msg types.Message) {}

	body := []byte(`{"content":"async","user_id":"u-2"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	ch.handleAsyncTasks(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("unexpected status code: %d body=%s", rr.Code, rr.Body.String())
	}

	var submit asyncSubmitResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &submit); err != nil {
		t.Fatalf("decode submit response failed: %v", err)
	}

	cancelReq := httptest.NewRequest(http.MethodPost, submit.CancelURL, nil)
	cancelRR := httptest.NewRecorder()
	ch.handleAsyncTasks(cancelRR, cancelReq)
	if cancelRR.Code != http.StatusOK {
		t.Fatalf("unexpected cancel status: %d body=%s", cancelRR.Code, cancelRR.Body.String())
	}

	var canceled asyncTaskResponse
	if err := json.Unmarshal(cancelRR.Body.Bytes(), &canceled); err != nil {
		t.Fatalf("decode cancel response failed: %v", err)
	}
	if canceled.Status != "canceled" {
		t.Fatalf("expected canceled status, got %+v", canceled)
	}

	statusReq := httptest.NewRequest(http.MethodGet, submit.StatusURL, nil)
	statusRR := httptest.NewRecorder()
	ch.handleAsyncTasks(statusRR, statusReq)
	if statusRR.Code != http.StatusOK {
		t.Fatalf("unexpected status status: %d body=%s", statusRR.Code, statusRR.Body.String())
	}
	var status asyncTaskResponse
	if err := json.Unmarshal(statusRR.Body.Bytes(), &status); err != nil {
		t.Fatalf("decode status response failed: %v", err)
	}
	if status.Status != "canceled" {
		t.Fatalf("expected canceled status, got %+v", status)
	}
}

func TestAsyncTaskListIncludesLatestFirst(t *testing.T) {
	ch := NewHTTPChannel(8080)
	ch.handler = func(msg types.Message) {}

	firstReq := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewReader([]byte(`{"content":"first","user_id":"u-1"}`)))
	firstRR := httptest.NewRecorder()
	ch.handleAsyncTasks(firstRR, firstReq)
	if firstRR.Code != http.StatusAccepted {
		t.Fatalf("unexpected first submit status: %d", firstRR.Code)
	}

	time.Sleep(2 * time.Millisecond)

	secondReq := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewReader([]byte(`{"content":"second","user_id":"u-1"}`)))
	secondRR := httptest.NewRecorder()
	ch.handleAsyncTasks(secondRR, secondReq)
	if secondRR.Code != http.StatusAccepted {
		t.Fatalf("unexpected second submit status: %d", secondRR.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/tasks?limit=1", nil)
	listRR := httptest.NewRecorder()
	ch.handleAsyncTasks(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("unexpected list status: %d body=%s", listRR.Code, listRR.Body.String())
	}

	var listed asyncTaskListResponse
	if err := json.Unmarshal(listRR.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode list response failed: %v", err)
	}
	if len(listed.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(listed.Tasks))
	}
	if listed.Tasks[0].Content != "second" {
		t.Fatalf("expected latest task first, got %+v", listed.Tasks[0])
	}

	statusFilterReq := httptest.NewRequest(http.MethodGet, "/api/tasks?status=completed", nil)
	statusFilterRR := httptest.NewRecorder()
	ch.handleAsyncTasks(statusFilterRR, statusFilterReq)
	if statusFilterRR.Code != http.StatusOK {
		t.Fatalf("unexpected filtered list status: %d", statusFilterRR.Code)
	}
	var filtered asyncTaskListResponse
	if err := json.Unmarshal(statusFilterRR.Body.Bytes(), &filtered); err != nil {
		t.Fatalf("decode filtered list failed: %v", err)
	}
	if len(filtered.Tasks) != 0 {
		t.Fatalf("expected no completed tasks, got %d", len(filtered.Tasks))
	}
}

package slack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"alter0/app/pkg/types"
)

func TestHandleEventBuildsInboundMessage(t *testing.T) {
	ch := NewChannel(Config{AppID: "app-1"})

	var got types.Message
	ch.handler = func(msg types.Message) {
		got = msg
	}

	body := `{"type":"event_callback","event":{"type":"message","user":"u1","text":"hello","channel":"C123","ts":"1.2"}}`
	req := httptest.NewRequest(http.MethodPost, "/events/slack", strings.NewReader(body))
	w := httptest.NewRecorder()

	ch.handleEvent(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	if got.ChannelID != "slack" {
		t.Fatalf("unexpected channel id: %s", got.ChannelID)
	}
	if got.Envelope == nil || got.Envelope.AccountID != "app-1" {
		t.Fatalf("unexpected account id: %+v", got.Envelope)
	}
	if got.Envelope.PeerID != "C123" {
		t.Fatalf("unexpected peer id: %s", got.Envelope.PeerID)
	}
}

func TestSendPostsToSlackAPI(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.URL.Path != "/chat.postMessage" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer xoxb-token" {
			t.Fatalf("unexpected auth header: %s", got)
		}
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload["channel"] != "C123" {
			t.Fatalf("unexpected channel: %v", payload["channel"])
		}
		if payload["text"] != "pong" {
			t.Fatalf("unexpected text: %v", payload["text"])
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	}))
	defer server.Close()

	ch := NewChannel(Config{BotToken: "xoxb-token", APIRoot: server.URL})

	err := ch.Send(context.Background(), types.Message{
		Content: "pong",
		Envelope: &types.MessageEnvelope{
			PeerID: "C123",
		},
	})
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}
	if !called {
		t.Fatal("expected API to be called")
	}
}

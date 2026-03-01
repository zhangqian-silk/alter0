package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"alter0/app/pkg/types"
)

func TestPollOnceDispatchesMessage(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/getUpdates") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"result": []map[string]interface{}{
				{
					"update_id": 101,
					"message": map[string]interface{}{
						"message_id": 77,
						"text":       "hello",
						"from":       map[string]interface{}{"id": 11},
						"chat":       map[string]interface{}{"id": 22},
					},
				},
			},
		})
	}))
	defer server.Close()

	ch := NewChannel(Config{BotToken: "token", APIRoot: server.URL})
	ch.handler = func(msg types.Message) {
		called = true
		if msg.ChannelID != "telegram" {
			t.Fatalf("unexpected channel: %s", msg.ChannelID)
		}
		if msg.Envelope == nil || msg.Envelope.PeerID != "22" {
			t.Fatalf("unexpected envelope: %+v", msg.Envelope)
		}
	}

	if err := ch.pollOnce(context.Background()); err != nil {
		t.Fatalf("poll failed: %v", err)
	}
	if !called {
		t.Fatal("expected handler call")
	}
}

func TestSendMessage(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if !strings.HasSuffix(r.URL.Path, "/sendMessage") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload["chat_id"] != "22" {
			t.Fatalf("unexpected chat id: %v", payload["chat_id"])
		}
		if payload["text"] != "pong" {
			t.Fatalf("unexpected text: %v", payload["text"])
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "result": map[string]interface{}{}})
	}))
	defer server.Close()

	ch := NewChannel(Config{BotToken: "token", APIRoot: server.URL})
	err := ch.Send(context.Background(), types.Message{
		Content: "pong",
		Envelope: &types.MessageEnvelope{
			PeerID: "22",
		},
	})
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}
	if !called {
		t.Fatal("expected API call")
	}
}

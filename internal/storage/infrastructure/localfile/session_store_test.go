package localfile

import (
	"context"
	"testing"
	"time"

	sessiondomain "alter0/internal/session/domain"
	shareddomain "alter0/internal/shared/domain"
)

func TestSessionStoreJSONRoundTrip(t *testing.T) {
	store := NewSessionStore(t.TempDir(), FormatJSON)
	ts := time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC)
	records := []sessiondomain.MessageRecord{
		{
			MessageID: "m-1",
			SessionID: "s-1",
			Role:      sessiondomain.MessageRoleUser,
			Content:   "hello",
			Timestamp: ts,
			RouteResult: sessiondomain.RouteResult{
				Route: shareddomain.RouteNL,
			},
		},
	}
	if err := store.Save(context.Background(), records); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 record, got %d", len(loaded))
	}
	if loaded[0].MessageID != "m-1" || loaded[0].SessionID != "s-1" {
		t.Fatalf("unexpected record: %+v", loaded[0])
	}
	if loaded[0].RouteResult.Route != shareddomain.RouteNL {
		t.Fatalf("expected route nl, got %q", loaded[0].RouteResult.Route)
	}
}

func TestSessionStoreMarkdownRoundTrip(t *testing.T) {
	store := NewSessionStore(t.TempDir(), FormatMarkdown)
	ts := time.Date(2026, 3, 3, 11, 0, 0, 0, time.UTC)
	records := []sessiondomain.MessageRecord{
		{
			MessageID: "m-2",
			SessionID: "s-2",
			Role:      sessiondomain.MessageRoleAssistant,
			Content:   "done",
			Timestamp: ts,
			RouteResult: sessiondomain.RouteResult{
				Route:     shareddomain.RouteCommand,
				ErrorCode: "command_failed",
			},
		},
	}
	if err := store.Save(context.Background(), records); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 record, got %d", len(loaded))
	}
	if loaded[0].RouteResult.ErrorCode != "command_failed" {
		t.Fatalf("expected command_failed, got %q", loaded[0].RouteResult.ErrorCode)
	}
}

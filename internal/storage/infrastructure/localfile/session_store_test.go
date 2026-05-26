package localfile

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
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

func TestSessionStoreSavesAgentSessionsIntoSeparateFiles(t *testing.T) {
	baseDir := t.TempDir()
	store := NewSessionStore(baseDir, FormatJSON)
	ts := time.Date(2026, 5, 24, 9, 0, 0, 0, time.UTC)
	records := []sessiondomain.MessageRecord{
		{
			MessageID: "coding-user-1",
			SessionID: "session-coding",
			Role:      sessiondomain.MessageRoleUser,
			Content:   "implement feature",
			Timestamp: ts,
			Source: sessiondomain.MessageSource{
				AgentID:   "coding",
				AgentName: "Coding Agent",
			},
		},
		{
			MessageID: "coding-assistant-1",
			SessionID: "session-coding",
			Role:      sessiondomain.MessageRoleAssistant,
			Content:   "done",
			Timestamp: ts.Add(time.Minute),
			Source: sessiondomain.MessageSource{
				AgentID:   "coding",
				AgentName: "Coding Agent",
			},
		},
		{
			MessageID: "travel-user-1",
			SessionID: "session-travel",
			Role:      sessiondomain.MessageRoleUser,
			Content:   "plan trip",
			Timestamp: ts.Add(2 * time.Minute),
			Source: sessiondomain.MessageSource{
				AgentID:   "travel",
				AgentName: "Travel Agent",
			},
		},
	}

	if err := store.Save(context.Background(), records); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	assertSessionStoreFileMessages(t, filepath.Join(baseDir, "sessions", "coding", "session-coding.json"), []string{"coding-user-1", "coding-assistant-1"})
	assertSessionStoreFileMessages(t, filepath.Join(baseDir, "sessions", "travel", "session-travel.json"), []string{"travel-user-1"})
	if _, err := os.Stat(filepath.Join(baseDir, "sessions.json")); !os.IsNotExist(err) {
		t.Fatalf("expected aggregate sessions.json not to be written, stat error: %v", err)
	}

	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(loaded) != len(records) {
		t.Fatalf("expected %d loaded records, got %d", len(records), len(loaded))
	}
}

func TestSessionStoreSplitsCanonicalChatSessionByArchiveDay(t *testing.T) {
	baseDir := t.TempDir()
	store := NewSessionStore(baseDir, FormatJSON)
	records := []sessiondomain.MessageRecord{
		{
			MessageID: "chat-before-cutoff",
			SessionID: "alter0-chat",
			Role:      sessiondomain.MessageRoleUser,
			Content:   "late night",
			Timestamp: time.Date(2026, 5, 23, 20, 30, 0, 0, time.UTC), // 2026-05-24 04:30 Asia/Shanghai
		},
		{
			MessageID: "chat-after-cutoff",
			SessionID: "alter0-chat",
			Role:      sessiondomain.MessageRoleAssistant,
			Content:   "new day",
			Timestamp: time.Date(2026, 5, 23, 21, 30, 0, 0, time.UTC), // 2026-05-24 05:30 Asia/Shanghai
		},
	}

	if err := store.Save(context.Background(), records); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	assertSessionStoreFileMessages(t, filepath.Join(baseDir, "sessions", "_default", "alter0-chat", "2026-05-23.json"), []string{"chat-before-cutoff"})
	assertSessionStoreFileMessages(t, filepath.Join(baseDir, "sessions", "_default", "alter0-chat", "2026-05-24.json"), []string{"chat-after-cutoff"})

	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(loaded) != len(records) {
		t.Fatalf("expected %d loaded records, got %d", len(records), len(loaded))
	}
}

func TestSessionStoreMigratesPreviousCanonicalChatSessionFileToArchiveDayDirectory(t *testing.T) {
	baseDir := t.TempDir()
	store := NewSessionStore(baseDir, FormatJSON)
	previousPath := filepath.Join(baseDir, "sessions", "_default", "alter0-chat.json")
	previous := sessionState{
		Messages: []sessiondomain.MessageRecord{
			{
				MessageID: "chat-legacy-layout",
				SessionID: sessiondomain.CanonicalChatSessionID,
				Role:      sessiondomain.MessageRoleUser,
				Content:   "legacy layout",
				Timestamp: time.Date(2026, 5, 24, 2, 0, 0, 0, time.UTC),
			},
		},
	}
	raw, err := json.MarshalIndent(previous, "", "  ")
	if err != nil {
		t.Fatalf("marshal previous layout failed: %v", err)
	}
	if err := writeFile(previousPath, append(raw, '\n')); err != nil {
		t.Fatalf("write previous layout failed: %v", err)
	}

	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(loaded) != 1 || loaded[0].MessageID != "chat-legacy-layout" {
		t.Fatalf("unexpected loaded messages: %+v", loaded)
	}
	if _, err := os.Stat(previousPath); !os.IsNotExist(err) {
		t.Fatalf("expected previous chat file removed, stat error: %v", err)
	}
	assertSessionStoreFileMessages(t, filepath.Join(baseDir, "sessions", "_default", "alter0-chat", "2026-05-24.json"), []string{"chat-legacy-layout"})
}

func TestSessionStoreRemovesStaleSessionFilesOnSave(t *testing.T) {
	baseDir := t.TempDir()
	store := NewSessionStore(baseDir, FormatJSON)
	ts := time.Date(2026, 5, 24, 10, 0, 0, 0, time.UTC)

	if err := store.Save(context.Background(), []sessiondomain.MessageRecord{
		{
			MessageID: "m-1",
			SessionID: "session-keep",
			Role:      sessiondomain.MessageRoleUser,
			Content:   "keep",
			Timestamp: ts,
			Source:    sessiondomain.MessageSource{AgentID: "coding"},
		},
		{
			MessageID: "m-2",
			SessionID: "session-delete",
			Role:      sessiondomain.MessageRoleUser,
			Content:   "delete",
			Timestamp: ts,
			Source:    sessiondomain.MessageSource{AgentID: "travel"},
		},
	}); err != nil {
		t.Fatalf("initial save failed: %v", err)
	}

	if err := store.Save(context.Background(), []sessiondomain.MessageRecord{
		{
			MessageID: "m-1",
			SessionID: "session-keep",
			Role:      sessiondomain.MessageRoleUser,
			Content:   "keep",
			Timestamp: ts,
			Source:    sessiondomain.MessageSource{AgentID: "coding"},
		},
	}); err != nil {
		t.Fatalf("second save failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(baseDir, "sessions", "travel", "session-delete.json")); !os.IsNotExist(err) {
		t.Fatalf("expected stale session file removed, stat error: %v", err)
	}
	assertSessionStoreFileMessages(t, filepath.Join(baseDir, "sessions", "coding", "session-keep.json"), []string{"m-1"})
}

func TestSessionStoreMigratesLegacyAggregateFileOnLoad(t *testing.T) {
	baseDir := t.TempDir()
	store := NewSessionStore(baseDir, FormatJSON)
	ts := time.Date(2026, 5, 24, 11, 0, 0, 0, time.UTC)
	legacy := sessionState{
		Messages: []sessiondomain.MessageRecord{
			{
				MessageID: "legacy-m-1",
				SessionID: "legacy-session",
				Role:      sessiondomain.MessageRoleUser,
				Content:   "legacy",
				Timestamp: ts,
				Source:    sessiondomain.MessageSource{AgentID: "coding"},
			},
		},
	}
	raw, err := json.MarshalIndent(legacy, "", "  ")
	if err != nil {
		t.Fatalf("marshal legacy failed: %v", err)
	}
	if err := writeFile(filepath.Join(baseDir, "sessions.json"), append(raw, '\n')); err != nil {
		t.Fatalf("write legacy failed: %v", err)
	}

	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(loaded) != 1 || loaded[0].MessageID != "legacy-m-1" {
		t.Fatalf("unexpected legacy records: %+v", loaded)
	}
	assertSessionStoreFileMessages(t, filepath.Join(baseDir, "sessions", "coding", "legacy-session.json"), []string{"legacy-m-1"})
	if _, err := os.Stat(filepath.Join(baseDir, "sessions.json")); !os.IsNotExist(err) {
		t.Fatalf("expected legacy aggregate removed after migration, stat error: %v", err)
	}
}

func TestSessionStoreMergesAndMigratesLayoutAndLegacyAggregateOnLoad(t *testing.T) {
	baseDir := t.TempDir()
	store := NewSessionStore(baseDir, FormatJSON)
	ts := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	layoutRecord := sessiondomain.MessageRecord{
		MessageID: "layout-m-1",
		SessionID: "layout-session",
		Role:      sessiondomain.MessageRoleUser,
		Content:   "layout",
		Timestamp: ts,
		Source:    sessiondomain.MessageSource{AgentID: "coding"},
	}
	legacyRecord := sessiondomain.MessageRecord{
		MessageID: "legacy-m-1",
		SessionID: "legacy-session",
		Role:      sessiondomain.MessageRoleUser,
		Content:   "legacy",
		Timestamp: ts.Add(time.Minute),
		Source:    sessiondomain.MessageSource{AgentID: "travel"},
	}
	if err := store.Save(context.Background(), []sessiondomain.MessageRecord{layoutRecord}); err != nil {
		t.Fatalf("save layout failed: %v", err)
	}
	legacy := sessionState{Messages: []sessiondomain.MessageRecord{layoutRecord, legacyRecord}}
	raw, err := json.MarshalIndent(legacy, "", "  ")
	if err != nil {
		t.Fatalf("marshal legacy failed: %v", err)
	}
	if err := writeFile(filepath.Join(baseDir, "sessions.json"), append(raw, '\n')); err != nil {
		t.Fatalf("write legacy failed: %v", err)
	}

	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected merged layout and legacy records without duplicates, got %+v", loaded)
	}
	ids := map[string]bool{}
	for _, item := range loaded {
		ids[item.MessageID] = true
	}
	if !ids["layout-m-1"] || !ids["legacy-m-1"] {
		t.Fatalf("expected both layout and legacy messages, got %+v", loaded)
	}
	assertSessionStoreFileMessages(t, filepath.Join(baseDir, "sessions", "coding", "layout-session.json"), []string{"layout-m-1"})
	assertSessionStoreFileMessages(t, filepath.Join(baseDir, "sessions", "travel", "legacy-session.json"), []string{"legacy-m-1"})
	if _, err := os.Stat(filepath.Join(baseDir, "sessions.json")); !os.IsNotExist(err) {
		t.Fatalf("expected legacy aggregate removed after merged migration, stat error: %v", err)
	}
}

func assertSessionStoreFileMessages(t *testing.T, path string, expected []string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read session file %s: %v", path, err)
	}
	state := sessionState{}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("decode session file %s: %v", path, err)
	}
	if len(state.Messages) != len(expected) {
		t.Fatalf("expected %d messages in %s, got %d", len(expected), path, len(state.Messages))
	}
	for idx, id := range expected {
		if state.Messages[idx].MessageID != id {
			t.Fatalf("expected message %d in %s to be %s, got %s", idx, path, id, state.Messages[idx].MessageID)
		}
	}
}

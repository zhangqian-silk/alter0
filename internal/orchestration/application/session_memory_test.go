package application

import (
	"strings"
	"testing"
	"time"

	shareddomain "alter0/internal/shared/domain"
)

func TestSessionMemoryWindowAndKeyState(t *testing.T) {
	store := newSessionMemoryStore(SessionMemoryOptions{
		MaxTurns:    2,
		TTL:         time.Hour,
		MaxSnippets: 160,
	})

	now := time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC)
	store.Record(memoryMessage("s1", "first input", now.Add(-4*time.Minute)), shareddomain.RouteNL, "plan p0")
	store.Record(memoryMessage("s1", "second input", now.Add(-3*time.Minute)), shareddomain.RouteNL, "plan p1")
	store.Record(memoryMessage("s1", "third input", now.Add(-2*time.Minute)), shareddomain.RouteNL, "plan p2")

	snapshot := store.Snapshot("s1", "continue", now)
	if len(snapshot.RecentTurns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(snapshot.RecentTurns))
	}
	if snapshot.RecentTurns[0].UserInput != "second input" {
		t.Fatalf("unexpected first turn after window prune: %q", snapshot.RecentTurns[0].UserInput)
	}
	if snapshot.RecentTurns[1].UserInput != "third input" {
		t.Fatalf("unexpected second turn after window prune: %q", snapshot.RecentTurns[1].UserInput)
	}
	if snapshot.KeyState["last_user_message"] != "third input" {
		t.Fatalf("unexpected key state last_user_message: %q", snapshot.KeyState["last_user_message"])
	}
	if snapshot.KeyState["latest_plan"] != "plan p2" {
		t.Fatalf("unexpected latest_plan: %q", snapshot.KeyState["latest_plan"])
	}
}

func TestSessionMemoryTTLPrunesExpiredTurns(t *testing.T) {
	store := newSessionMemoryStore(SessionMemoryOptions{
		MaxTurns:    4,
		TTL:         2 * time.Minute,
		MaxSnippets: 160,
	})

	now := time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC)
	store.Record(memoryMessage("s1", "expired turn", now.Add(-5*time.Minute)), shareddomain.RouteNL, "plan old")
	store.Record(memoryMessage("s1", "fresh turn", now.Add(-30*time.Second)), shareddomain.RouteNL, "plan new")

	snapshot := store.Snapshot("s1", "follow", now)
	if len(snapshot.RecentTurns) != 1 {
		t.Fatalf("expected 1 turn after ttl prune, got %d", len(snapshot.RecentTurns))
	}
	if snapshot.RecentTurns[0].UserInput != "fresh turn" {
		t.Fatalf("unexpected turn after ttl prune: %q", snapshot.RecentTurns[0].UserInput)
	}
}

func TestSessionMemoryResolvesInSessionReference(t *testing.T) {
	store := newSessionMemoryStore(SessionMemoryOptions{
		MaxTurns:    4,
		TTL:         time.Hour,
		MaxSnippets: 160,
	})

	now := time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC)
	store.Record(memoryMessage("s1", "给我一个方案", now.Add(-2*time.Minute)), shareddomain.RouteNL, "方案A：蓝绿发布")
	store.Record(memoryMessage("s2", "other session", now.Add(-2*time.Minute)), shareddomain.RouteNL, "plan from session2")

	snapshot := store.Snapshot("s1", "这个方案的风险是什么", now)
	if snapshot.Reference == "" {
		t.Fatalf("expected reference resolution for same session")
	}
	if !strings.Contains(snapshot.Reference, "方案A") {
		t.Fatalf("unexpected reference target: %q", snapshot.Reference)
	}

	otherSnapshot := store.Snapshot("s2", "这个方案还能优化吗", now)
	if !strings.Contains(otherSnapshot.Reference, "session2") {
		t.Fatalf("expected isolated session reference, got %q", otherSnapshot.Reference)
	}
}

func memoryMessage(sessionID, content string, at time.Time) shareddomain.UnifiedMessage {
	return shareddomain.UnifiedMessage{
		MessageID:   "m-" + sessionID + "-" + strings.ReplaceAll(content, " ", "-"),
		SessionID:   sessionID,
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     content,
		TraceID:     "t-" + sessionID,
		ReceivedAt:  at,
	}
}

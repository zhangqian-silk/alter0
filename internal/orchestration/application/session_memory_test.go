package application

import (
	"os"
	"path/filepath"
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

func TestSessionMemoryCompressionBuildsStructuredFragments(t *testing.T) {
	store := newSessionMemoryStore(SessionMemoryOptions{
		MaxTurns:                 8,
		TTL:                      time.Hour,
		MaxSnippets:              220,
		CompressionTriggerTokens: 80,
		CompressionSummaryTokens: 60,
		CompressionRetainTurns:   2,
		CompressionMaxFacts:      6,
	})

	now := time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC)
	longTail := strings.Repeat(" rollout verification checklist", 10)

	store.Record(memoryMessageWithID("s1", "m1", "release_window: friday 22:00 slo_target: 99.95%"+longTail, now.Add(-4*time.Minute)), shareddomain.RouteNL, "plan alpha: phase 1")
	store.Record(memoryMessageWithID("s1", "m2", "rollback_owner: sre-oncall region: us-east-1"+longTail, now.Add(-3*time.Minute)), shareddomain.RouteNL, "plan alpha: phase 2")
	store.Record(memoryMessageWithID("s1", "m3", "canary_ratio: 10% approval_owner: platform-lead"+longTail, now.Add(-2*time.Minute)), shareddomain.RouteNL, "plan alpha: phase 3")
	store.Record(memoryMessageWithID("s1", "m4", "incident_channel: #release-war-room check_window: 30m"+longTail, now.Add(-1*time.Minute)), shareddomain.RouteNL, "plan alpha: phase 4")

	snapshot := store.Snapshot("s1", "continue release plan", now)
	if len(snapshot.Fragments) == 0 {
		t.Fatalf("expected compressed fragment, got none")
	}
	if len(snapshot.RecentTurns) != 2 {
		t.Fatalf("expected retain 2 recent turns after compression, got %d", len(snapshot.RecentTurns))
	}

	fragment := snapshot.Fragments[0]
	if fragment.Summary == "" {
		t.Fatalf("expected fragment summary")
	}
	if len(fragment.KeyFacts) == 0 {
		t.Fatalf("expected key facts extracted from compressed turns")
	}
	sourceRefCount := 0
	for _, item := range snapshot.Fragments {
		sourceRefCount += len(item.SourceTurns)
	}
	if sourceRefCount < 2 {
		t.Fatalf("expected source turn refs keep compressed history, got %d", sourceRefCount)
	}
	if fragment.SourceTurns[0].UserMessageID != "m1" {
		t.Fatalf("expected first source turn to keep original message id, got %q", fragment.SourceTurns[0].UserMessageID)
	}
	if len(fragment.SourceTurns) == 0 || fragment.SourceTurns[0].AssistantReplyRef == "" {
		t.Fatalf("expected assistant reply reference kept in compressed fragment")
	}
}

func TestSessionMemoryCompressionPreservesReferenceResolutionFromFragments(t *testing.T) {
	store := newSessionMemoryStore(SessionMemoryOptions{
		MaxTurns:                 6,
		TTL:                      time.Hour,
		MaxSnippets:              200,
		CompressionTriggerTokens: 70,
		CompressionSummaryTokens: 40,
		CompressionRetainTurns:   1,
		CompressionMaxFacts:      4,
	})

	now := time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC)
	longTail := strings.Repeat(" release guardrail", 12)

	store.Record(memoryMessageWithID("s1", "m1", "给我一个发布方案"+longTail, now.Add(-3*time.Minute)), shareddomain.RouteNL, "方案A：蓝绿发布")
	store.Record(memoryMessageWithID("s1", "m2", "owner: sre-team rollback_window: 30m"+longTail, now.Add(-2*time.Minute)), shareddomain.RouteNL, "方案A：补充灰度计划")
	store.Record(memoryMessageWithID("s1", "m3", "请继续补充验证步骤"+longTail, now.Add(-1*time.Minute)), shareddomain.RouteNL, "方案A：增加验证清单")

	snapshot := store.Snapshot("s1", "这个方案还有什么风险", now)
	if len(snapshot.Fragments) == 0 {
		t.Fatalf("expected compressed fragments for long session")
	}
	if snapshot.Reference == "" {
		t.Fatalf("expected reference resolved from compressed memory")
	}
	if !strings.Contains(snapshot.Reference, "方案A") {
		t.Fatalf("expected reference keep historical plan hint, got %q", snapshot.Reference)
	}
	if snapshot.Metadata()["memory_context_compressed"] != "true" {
		t.Fatalf("expected compressed metadata=true, got %+v", snapshot.Metadata())
	}

	prompt := buildSessionMemoryPrompt("这个方案还有什么风险", snapshot)
	if !strings.Contains(prompt, "Compressed memory fragments:") {
		t.Fatalf("expected prompt include compressed fragment section, got %q", prompt)
	}
	if !strings.Contains(prompt, "source_turn_refs:") {
		t.Fatalf("expected prompt include source turn references, got %q", prompt)
	}
}

func TestSessionMemoryDailyMarkdownPersistenceAndLongTermCandidates(t *testing.T) {
	dir := t.TempDir()
	store := newSessionMemoryStore(SessionMemoryOptions{
		MaxTurns:                 8,
		TTL:                      time.Hour,
		MaxSnippets:              220,
		CompressionTriggerTokens: 70,
		CompressionSummaryTokens: 60,
		CompressionRetainTurns:   2,
		CompressionMaxFacts:      6,
		DailyMemoryDir:           dir,
	})

	now := time.Date(2026, 3, 3, 11, 0, 0, 0, time.UTC)
	longTail := strings.Repeat(" release guardrail", 10)
	store.Record(memoryMessageWithID("s-day", "m1", "release_window: friday 22:00"+longTail, now.Add(-3*time.Minute)), shareddomain.RouteNL, "plan alpha: phase 1")
	store.Record(memoryMessageWithID("s-day", "m2", "rollback_owner: sre-oncall"+longTail, now.Add(-2*time.Minute)), shareddomain.RouteNL, "plan alpha: phase 2")
	store.Record(memoryMessageWithID("s-day", "m3", "slo_target: 99.95%"+longTail, now.Add(-time.Minute)), shareddomain.RouteNL, "plan alpha: phase 3")

	dayFile := filepath.Join(dir, "2026-03-03.md")
	raw, err := os.ReadFile(dayFile)
	if err != nil {
		t.Fatalf("expected day-level markdown memory file: %v", err)
	}
	content := string(raw)
	if !strings.Contains(content, "# Daily Memory 2026-03-03") {
		t.Fatalf("expected day-level heading, got %q", content)
	}
	if !strings.Contains(content, "## L2") {
		t.Fatalf("expected compressed tier section in day-level file, got %q", content)
	}
	if !strings.Contains(content, "key_facts:") {
		t.Fatalf("expected key facts persisted in markdown, got %q", content)
	}

	candidateFile := filepath.Join(dir, "long-term", "2026-03-03.md")
	candidateRaw, err := os.ReadFile(candidateFile)
	if err != nil {
		t.Fatalf("expected long-term candidate markdown file: %v", err)
	}
	if !strings.Contains(string(candidateRaw), "# Long-Term Memory Candidates 2026-03-03") {
		t.Fatalf("expected candidate markdown heading, got %q", string(candidateRaw))
	}
}

func TestSessionMemoryDailyTierConstraintsApplyLengthCapacityAndTTL(t *testing.T) {
	dir := t.TempDir()
	store := newSessionMemoryStore(SessionMemoryOptions{
		MaxTurns:                 8,
		TTL:                      24 * time.Hour,
		MaxSnippets:              320,
		CompressionTriggerTokens: 20_000,
		CompressionSummaryTokens: 120,
		CompressionRetainTurns:   6,
		CompressionMaxFacts:      4,
		DailyMemoryDir:           dir,
		L1: SessionMemoryTierOptions{
			MaxEntryLength: 18,
			MaxLayerTokens: 12,
			TTL:            time.Hour,
		},
		L2: SessionMemoryTierOptions{
			MaxEntryLength: 60,
			MaxLayerTokens: 200,
			TTL:            48 * time.Hour,
		},
		L3: SessionMemoryTierOptions{
			MaxEntryLength: 80,
			MaxLayerTokens: 200,
			TTL:            72 * time.Hour,
		},
	})

	now := time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC)
	store.Record(memoryMessageWithID("s-cap", "old", "very old entry should expire", now.Add(-3*time.Hour)), shareddomain.RouteNL, "assistant old summary should be dropped")
	store.Record(memoryMessageWithID("s-cap", "new", "latest release checklist entry", now), shareddomain.RouteNL, "assistant latest response")

	dayFile := filepath.Join(dir, "2026-03-03.md")
	state := sessionMemoryDailyState{}
	raw, err := os.ReadFile(dayFile)
	if err != nil {
		t.Fatalf("read day markdown file: %v", err)
	}
	if err := parseSessionMemoryDailyState(raw, &state); err != nil {
		t.Fatalf("parse day markdown structured state: %v", err)
	}
	l1 := state.TierValues[sessionMemoryDailyTierL1]
	if len(l1) != 1 {
		t.Fatalf("expected expired l1 entry pruned and capacity enforced, got %d entries", len(l1))
	}
	if strings.Contains(l1[0].Summary, "very old") {
		t.Fatalf("expected old entry removed by l1 ttl, got %q", l1[0].Summary)
	}
	if len([]rune(l1[0].Summary)) > 21 {
		t.Fatalf("expected l1 summary truncated by max entry length, got %q", l1[0].Summary)
	}
}

func memoryMessage(sessionID, content string, at time.Time) shareddomain.UnifiedMessage {
	messageID := "m-" + sessionID + "-" + strings.ReplaceAll(content, " ", "-")
	return memoryMessageWithID(sessionID, messageID, content, at)
}

func memoryMessageWithID(sessionID, messageID, content string, at time.Time) shareddomain.UnifiedMessage {
	return shareddomain.UnifiedMessage{
		MessageID:   messageID,
		SessionID:   sessionID,
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     content,
		TraceID:     "t-" + sessionID,
		ReceivedAt:  at,
	}
}

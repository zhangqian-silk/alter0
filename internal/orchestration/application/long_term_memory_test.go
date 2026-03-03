package application

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	shareddomain "alter0/internal/shared/domain"
)

func TestLongTermMemoryRecordAndScopedRetrieval(t *testing.T) {
	store := newLongTermMemoryStore(LongTermMemoryOptions{
		MaxEntriesPerScope: 16,
		MaxHits:            3,
		MaxSnippet:         160,
	})
	now := time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC)

	store.Record(longTermMessage("s-a", "u-1", "tenant-a", "remember my style", now, map[string]string{
		longTermMemoryStrategyMetadataKey: "add",
		longTermMemoryKindMetadataKey:     "preference",
		longTermMemoryKeyMetadataKey:      "response style",
		longTermMemoryValueMetadataKey:    "concise bullet answers",
		longTermMemoryTagsMetadataKey:     "style, concise",
	}), shareddomain.RouteNL, "")
	store.Record(longTermMessage("s-b", "u-2", "tenant-a", "remember my style", now.Add(30*time.Second), map[string]string{
		longTermMemoryStrategyMetadataKey: "add",
		longTermMemoryKindMetadataKey:     "preference",
		longTermMemoryKeyMetadataKey:      "response style",
		longTermMemoryValueMetadataKey:    "detailed narrative",
		longTermMemoryTagsMetadataKey:     "style, detailed",
	}), shareddomain.RouteNL, "")
	store.Record(longTermMessage("s-c", "u-1", "tenant-b", "remember my style", now.Add(time.Minute), map[string]string{
		longTermMemoryStrategyMetadataKey: "add",
		longTermMemoryKindMetadataKey:     "preference",
		longTermMemoryKeyMetadataKey:      "response style",
		longTermMemoryValueMetadataKey:    "formal tone",
		longTermMemoryTagsMetadataKey:     "style, formal",
	}), shareddomain.RouteNL, "")

	query := longTermMessage("s-new", "u-1", "tenant-a", "Can you keep concise style?", now.Add(2*time.Minute), nil)
	snapshot := store.Snapshot(query, query.Content, query.ReceivedAt)
	if len(snapshot.Hits) != 1 {
		t.Fatalf("expected 1 long-term memory hit, got %d", len(snapshot.Hits))
	}
	hit := snapshot.Hits[0]
	if !strings.Contains(hit.Entry.Value, "concise") {
		t.Fatalf("expected concise preference hit, got %q", hit.Entry.Value)
	}
	if hit.Entry.Scope.UserID != "u-1" || hit.Entry.Scope.TenantID != "tenant-a" {
		t.Fatalf("unexpected hit scope: tenant=%q user=%q", hit.Entry.Scope.TenantID, hit.Entry.Scope.UserID)
	}
	if snapshot.Metadata()["memory_long_term_hit_count"] != "1" {
		t.Fatalf("expected hit count metadata 1, got %q", snapshot.Metadata()["memory_long_term_hit_count"])
	}
}

func TestLongTermMemoryUpdateStrategiesAndAuditFields(t *testing.T) {
	store := newLongTermMemoryStore(LongTermMemoryOptions{
		MaxEntriesPerScope: 16,
		MaxHits:            3,
		MaxSnippet:         160,
	})
	now := time.Date(2026, 3, 3, 11, 0, 0, 0, time.UTC)

	addMessage := longTermMessage("session-add", "u-1", "tenant-a", "store timezone", now, map[string]string{
		longTermMemoryStrategyMetadataKey: "add",
		longTermMemoryKindMetadataKey:     "fact",
		longTermMemoryKeyMetadataKey:      "timezone",
		longTermMemoryValueMetadataKey:    "UTC+8",
		longTermMemoryTagsMetadataKey:     "profile, timezone",
	})
	store.Record(addMessage, shareddomain.RouteNL, "")

	overwriteAt := now.Add(time.Minute)
	overwriteMessage := longTermMessage("session-overwrite", "u-1", "tenant-a", "update timezone", overwriteAt, map[string]string{
		longTermMemoryStrategyMetadataKey: "overwrite",
		longTermMemoryKindMetadataKey:     "fact",
		longTermMemoryKeyMetadataKey:      "timezone",
		longTermMemoryValueMetadataKey:    "UTC+1",
		longTermMemoryTagsMetadataKey:     "profile, timezone, latest",
	})
	store.Record(overwriteMessage, shareddomain.RouteNL, "")

	scope := resolveLongTermMemoryScope(overwriteMessage, store.options)
	entries := getScopeEntries(store, scope)
	if len(entries) != 2 {
		t.Fatalf("expected two entries after overwrite, got %d", len(entries))
	}

	activeCount := 0
	invalidatedCount := 0
	for _, entry := range entries {
		switch entry.Status {
		case longTermMemoryStatusActive:
			activeCount++
			if entry.Value != "UTC+1" {
				t.Fatalf("expected active entry value UTC+1, got %q", entry.Value)
			}
			if entry.SourceSessionID != "session-overwrite" {
				t.Fatalf("expected active entry source session session-overwrite, got %q", entry.SourceSessionID)
			}
			if !entry.UpdatedAt.Equal(overwriteAt) {
				t.Fatalf("expected active entry updated at overwrite time, got %s", entry.UpdatedAt)
			}
		case longTermMemoryStatusInvalidated:
			invalidatedCount++
			if entry.UpdatedBy != longTermMemoryStrategyOverwrite {
				t.Fatalf("expected invalidated entry updated by overwrite, got %q", entry.UpdatedBy)
			}
			if !entry.UpdatedAt.Equal(overwriteAt) {
				t.Fatalf("expected invalidated entry updated at overwrite time, got %s", entry.UpdatedAt)
			}
			if entry.SourceSessionID != "session-overwrite" {
				t.Fatalf("expected invalidated entry source session session-overwrite, got %q", entry.SourceSessionID)
			}
		default:
			t.Fatalf("unexpected entry status: %q", entry.Status)
		}
	}
	if activeCount != 1 || invalidatedCount != 1 {
		t.Fatalf("expected one active and one invalidated entry, got active=%d invalidated=%d", activeCount, invalidatedCount)
	}

	invalidateAt := overwriteAt.Add(time.Minute)
	invalidateMessage := longTermMessage("session-invalidate", "u-1", "tenant-a", "timezone invalid", invalidateAt, map[string]string{
		longTermMemoryStrategyMetadataKey: "invalidate",
		longTermMemoryKindMetadataKey:     "fact",
		longTermMemoryKeyMetadataKey:      "timezone",
	})
	store.Record(invalidateMessage, shareddomain.RouteNL, "")

	snapshot := store.Snapshot(invalidateMessage, "timezone", invalidateAt.Add(time.Second))
	if len(snapshot.Hits) != 0 {
		t.Fatalf("expected no active hits after invalidate, got %d", len(snapshot.Hits))
	}

	entries = getScopeEntries(store, scope)
	for _, entry := range entries {
		if entry.Status != longTermMemoryStatusInvalidated {
			t.Fatalf("expected all entries invalidated after invalidate strategy, got %q", entry.Status)
		}
	}
}

func TestBuildLongTermMemoryPromptInjectsSection(t *testing.T) {
	snapshot := longTermMemorySnapshot{
		Scope: longTermMemoryScope{
			TenantID: "tenant-a",
			UserID:   "u-1",
		},
		Hits: []longTermMemoryHit{
			{
				Entry: longTermMemoryEntry{
					Kind:            longTermMemoryKindPreference,
					Key:             "response_style",
					Value:           "concise bullet answers",
					Tags:            []string{"concise", "style"},
					SourceSessionID: "s-1",
					UpdatedAt:       time.Date(2026, 3, 3, 9, 0, 0, 0, time.UTC),
				},
				Score: 3.4,
			},
		},
	}
	basePrompt := "[SESSION SHORT TERM MEMORY]\nScope: current session only.\nRecent turns:\n1) user: hello\n   assistant: hi\nCurrent user input:\nPlease answer with rollout details"

	prompt := buildLongTermMemoryPrompt(basePrompt, snapshot)
	if !strings.Contains(prompt, "[LONG TERM MEMORY]") {
		t.Fatalf("expected long-term memory section in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "response_style: concise bullet answers") {
		t.Fatalf("expected memory entry in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "Current user input:\nPlease answer with rollout details") {
		t.Fatalf("expected current user input preserved, got %q", prompt)
	}
}

func TestLongTermMemoryImplicitPreferencePersistsAcrossSessions(t *testing.T) {
	store := newLongTermMemoryStore(LongTermMemoryOptions{
		MaxEntriesPerScope: 16,
		MaxHits:            3,
		MaxSnippet:         160,
	})
	now := time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC)

	writeMessage := longTermMessage("session-preference", "u-7", "tenant-x", "I prefer concise responses", now, nil)
	store.Record(writeMessage, shareddomain.RouteNL, "")

	query := longTermMessage("session-new", "u-7", "tenant-x", "Can you keep concise format?", now.Add(time.Minute), nil)
	snapshot := store.Snapshot(query, query.Content, query.ReceivedAt)
	if len(snapshot.Hits) == 0 {
		t.Fatalf("expected implicit preference memory to be retrieved")
	}
	if snapshot.Hits[0].Entry.Key != "default_preference" {
		t.Fatalf("expected default_preference key, got %q", snapshot.Hits[0].Entry.Key)
	}
}

func TestLongTermMemoryTierConstraintsWithTTLAndLRUEviction(t *testing.T) {
	store := newLongTermMemoryStore(LongTermMemoryOptions{
		MaxEntriesPerScope: 16,
		MaxHits:            6,
		MaxSnippet:         200,
		L1: LongTermMemoryTierOptions{
			MaxEntryLength: 12,
			MaxLayerTokens: 16,
			TTL:            time.Minute,
			EvictionPolicy: longTermMemoryEvictionPolicyLRU,
		},
		L2: LongTermMemoryTierOptions{
			MaxEntryLength: 160,
			MaxLayerTokens: 200,
			TTL:            time.Hour,
			EvictionPolicy: longTermMemoryEvictionPolicyLRU,
		},
		L3: LongTermMemoryTierOptions{
			MaxEntryLength: 160,
			MaxLayerTokens: 200,
			TTL:            time.Hour,
			EvictionPolicy: longTermMemoryEvictionPolicyLRU,
		},
	})
	now := time.Date(2026, 3, 3, 14, 0, 0, 0, time.UTC)

	store.Record(longTermMessage("l1-a", "u-1", "tenant-a", "save high priority", now, map[string]string{
		longTermMemoryStrategyMetadataKey: "add",
		longTermMemoryKindMetadataKey:     "fact",
		longTermMemoryTierMetadataKey:     "L1",
		longTermMemoryKeyMetadataKey:      "critical-note",
		longTermMemoryValueMetadataKey:    "abcdefghijklmno",
	}), shareddomain.RouteNL, "")

	scope := resolveLongTermMemoryScope(longTermMessage("scope", "u-1", "tenant-a", "", now, nil), store.options)
	entries := getScopeEntries(store, scope)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Tier != longTermMemoryTierL1 {
		t.Fatalf("expected l1 tier entry, got %q", entries[0].Tier)
	}
	if entries[0].Value != "abcdefghijkl..." {
		t.Fatalf("expected value truncated by tier max entry length, got %q", entries[0].Value)
	}

	store.Record(longTermMessage("l1-b", "u-1", "tenant-a", "save second", now.Add(10*time.Second), map[string]string{
		longTermMemoryStrategyMetadataKey: "add",
		longTermMemoryKindMetadataKey:     "fact",
		longTermMemoryTierMetadataKey:     "L1",
		longTermMemoryKeyMetadataKey:      "second-note",
		longTermMemoryValueMetadataKey:    "second",
	}), shareddomain.RouteNL, "")

	entries = getScopeEntries(store, scope)
	if len(entries) != 1 {
		t.Fatalf("expected l1 token capacity to evict oldest entry, got %d entries", len(entries))
	}
	if entries[0].Key != "second-note" {
		t.Fatalf("expected oldest l1 entry evicted by LRU, got key %q", entries[0].Key)
	}

	query := longTermMessage("l1-query", "u-1", "tenant-a", "second-note", now.Add(2*time.Minute), nil)
	snapshot := store.Snapshot(query, query.Content, query.ReceivedAt)
	if len(snapshot.Hits) != 0 {
		t.Fatalf("expected l1 entry expired by ttl after 2m, got %d hits", len(snapshot.Hits))
	}
}

func TestLongTermMemoryHitChainAndBudgetTruncation(t *testing.T) {
	store := newLongTermMemoryStore(LongTermMemoryOptions{
		MaxEntriesPerScope:   24,
		MaxHits:              6,
		MaxSnippet:           200,
		InjectionTokenBudget: 20,
		DefaultTenantID:      "tenant-a",
		DefaultUserID:        "u-1",
	})
	now := time.Date(2026, 3, 3, 15, 0, 0, 0, time.UTC)

	store.Record(longTermMessage("hit-l1", "u-1", "tenant-a", "remember l1", now, map[string]string{
		longTermMemoryStrategyMetadataKey: "add",
		longTermMemoryKindMetadataKey:     "fact",
		longTermMemoryTierMetadataKey:     "L1",
		longTermMemoryKeyMetadataKey:      "release-plan-hot",
		longTermMemoryValueMetadataKey:    "priority rollout owner",
	}), shareddomain.RouteNL, "")
	store.Record(longTermMessage("hit-l2", "u-1", "tenant-a", "remember l2", now.Add(10*time.Second), map[string]string{
		longTermMemoryStrategyMetadataKey: "add",
		longTermMemoryKindMetadataKey:     "fact",
		longTermMemoryTierMetadataKey:     "L2",
		longTermMemoryKeyMetadataKey:      "release-plan-warm",
		longTermMemoryValueMetadataKey:    "weekly status context",
	}), shareddomain.RouteNL, "")
	store.Record(longTermMessage("hit-l3", "u-1", "tenant-a", "remember l3", now.Add(20*time.Second), map[string]string{
		longTermMemoryStrategyMetadataKey: "add",
		longTermMemoryKindMetadataKey:     "fact",
		longTermMemoryTierMetadataKey:     "L3",
		longTermMemoryKeyMetadataKey:      "release-plan-archive",
		longTermMemoryValueMetadataKey:    "legacy migration runbook",
	}), shareddomain.RouteNL, "")

	query := longTermMessage("query", "u-1", "tenant-a", "Need release plan context", now.Add(time.Minute), nil)
	snapshot := store.Snapshot(query, query.Content, query.ReceivedAt)
	if len(snapshot.CandidateTierHits[longTermMemoryTierL1]) != 1 ||
		len(snapshot.CandidateTierHits[longTermMemoryTierL2]) != 1 ||
		len(snapshot.CandidateTierHits[longTermMemoryTierL3]) != 1 {
		t.Fatalf("expected hit chain candidates in all tiers, got l1=%d l2=%d l3=%d",
			len(snapshot.CandidateTierHits[longTermMemoryTierL1]),
			len(snapshot.CandidateTierHits[longTermMemoryTierL2]),
			len(snapshot.CandidateTierHits[longTermMemoryTierL3]))
	}
	if len(snapshot.Hits) == 0 {
		t.Fatalf("expected selected hits under budget")
	}
	if snapshot.Hits[0].Entry.Tier != longTermMemoryTierL1 {
		t.Fatalf("expected L1 prioritized first hit, got %q", snapshot.Hits[0].Entry.Tier)
	}
	if len(snapshot.TierHits[longTermMemoryTierL3]) > 0 {
		t.Fatalf("expected low-priority L3 truncated under budget, got %d hits", len(snapshot.TierHits[longTermMemoryTierL3]))
	}
	if snapshot.Metadata()["memory_long_term_truncated"] != "true" {
		t.Fatalf("expected truncation metadata true, got %q", snapshot.Metadata()["memory_long_term_truncated"])
	}
}

func TestLongTermMemoryPromotionAndDemotionObservable(t *testing.T) {
	store := newLongTermMemoryStore(LongTermMemoryOptions{
		MaxEntriesPerScope:   24,
		MaxHits:              6,
		MaxSnippet:           200,
		InjectionTokenBudget: 160,
		L1: LongTermMemoryTierOptions{
			MaxEntryLength: 200,
			MaxLayerTokens: 200,
			TTL:            time.Hour,
			DemoteIdleTTL:  5 * time.Minute,
			EvictionPolicy: longTermMemoryEvictionPolicyLRU,
		},
		L2: LongTermMemoryTierOptions{
			MaxEntryLength: 200,
			MaxLayerTokens: 400,
			TTL:            time.Hour,
			DemoteIdleTTL:  10 * time.Minute,
			PromoteHits:    1,
			EvictionPolicy: longTermMemoryEvictionPolicyLRU,
		},
		L3: LongTermMemoryTierOptions{
			MaxEntryLength: 200,
			MaxLayerTokens: 600,
			TTL:            time.Hour,
			PromoteHits:    2,
			EvictionPolicy: longTermMemoryEvictionPolicyLRU,
		},
	})
	now := time.Date(2026, 3, 3, 16, 0, 0, 0, time.UTC)

	store.Record(longTermMessage("promote-seed", "u-1", "tenant-a", "seed", now, map[string]string{
		longTermMemoryStrategyMetadataKey: "add",
		longTermMemoryKindMetadataKey:     "fact",
		longTermMemoryKeyMetadataKey:      "release-checklist",
		longTermMemoryValueMetadataKey:    "canary and rollback",
	}), shareddomain.RouteNL, "")
	store.Record(longTermMessage("important-seed", "u-1", "tenant-a", "important seed", now.Add(time.Second), map[string]string{
		longTermMemoryStrategyMetadataKey:  "add",
		longTermMemoryKindMetadataKey:      "fact",
		longTermMemoryImportantMetadataKey: "true",
		longTermMemoryKeyMetadataKey:       "critical-policy",
		longTermMemoryValueMetadataKey:     "always verify before rollout",
	}), shareddomain.RouteNL, "")

	scope := resolveLongTermMemoryScope(longTermMessage("scope-promote", "u-1", "tenant-a", "", now, nil), store.options)
	entries := getScopeEntries(store, scope)
	hasImportantL1 := false
	for _, entry := range entries {
		if entry.Key == "critical-policy" && entry.Tier == longTermMemoryTierL1 {
			hasImportantL1 = true
		}
	}
	if !hasImportantL1 {
		t.Fatalf("expected important memory promoted directly to L1")
	}

	first := longTermMessage("query-1", "u-1", "tenant-a", "release checklist", now.Add(2*time.Minute), nil)
	firstSnapshot := store.Snapshot(first, first.Content, first.ReceivedAt)
	if len(firstSnapshot.Promotions) != 0 {
		t.Fatalf("expected no promotion on first hit, got %d", len(firstSnapshot.Promotions))
	}

	second := longTermMessage("query-2", "u-1", "tenant-a", "release checklist", now.Add(3*time.Minute), nil)
	secondSnapshot := store.Snapshot(second, second.Content, second.ReceivedAt)
	if len(secondSnapshot.Promotions) == 0 {
		t.Fatalf("expected L3 to L2 promotion on repeated hits")
	}
	if secondSnapshot.Metadata()["memory_long_term_promotion_count"] == "0" {
		t.Fatalf("expected promotion count metadata")
	}

	third := longTermMessage("query-3", "u-1", "tenant-a", "release checklist", now.Add(4*time.Minute), nil)
	thirdSnapshot := store.Snapshot(third, third.Content, third.ReceivedAt)
	if len(thirdSnapshot.Promotions) == 0 {
		t.Fatalf("expected L2 to L1 promotion")
	}

	idleQuery := longTermMessage("query-idle", "u-1", "tenant-a", "release checklist", now.Add(20*time.Minute), nil)
	idleSnapshot := store.Snapshot(idleQuery, idleQuery.Content, idleQuery.ReceivedAt)
	if len(idleSnapshot.Demotions) == 0 {
		t.Fatalf("expected idle demotion to be observable")
	}
	if idleSnapshot.Metadata()["memory_long_term_demotion_count"] == "0" {
		t.Fatalf("expected demotion count metadata")
	}
}

func TestLongTermMemoryPersistenceAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "MEMORY.md")
	options := LongTermMemoryOptions{
		MaxEntriesPerScope:   24,
		MaxHits:              6,
		MaxSnippet:           200,
		InjectionTokenBudget: 120,
		PersistencePath:      path,
		WritePolicy:          longTermMemoryWritePolicyWriteThrough,
	}
	now := time.Date(2026, 3, 3, 17, 0, 0, 0, time.UTC)

	store := newLongTermMemoryStore(options)
	store.Record(longTermMessage("persist-seed", "u-1", "tenant-a", "remember setting", now, map[string]string{
		longTermMemoryStrategyMetadataKey: "add",
		longTermMemoryKindMetadataKey:     "preference",
		longTermMemoryKeyMetadataKey:      "response-style",
		longTermMemoryValueMetadataKey:    "concise bullets",
	}), shareddomain.RouteNL, "")

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected persistence file after write-through record: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read markdown persistence file: %v", err)
	}
	if !strings.Contains(string(content), "# Long-Term Memory") {
		t.Fatalf("expected markdown persistence heading, got %q", string(content))
	}

	reloaded := newLongTermMemoryStore(options)
	query := longTermMessage("persist-query", "u-1", "tenant-a", "keep concise", now.Add(time.Minute), nil)
	snapshot := reloaded.Snapshot(query, query.Content, query.ReceivedAt)
	if len(snapshot.Hits) == 0 {
		t.Fatalf("expected memory recovered after restart")
	}
	if !strings.Contains(snapshot.Hits[0].Entry.Value, "concise") {
		t.Fatalf("expected recovered value to contain concise, got %q", snapshot.Hits[0].Entry.Value)
	}
}

func TestLongTermMemoryWriteBackRequiresFlush(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "memory_writeback.md")
	options := LongTermMemoryOptions{
		MaxEntriesPerScope:   24,
		MaxHits:              6,
		MaxSnippet:           200,
		InjectionTokenBudget: 120,
		PersistencePath:      path,
		WritePolicy:          longTermMemoryWritePolicyWriteBack,
		WriteBackFlush:       time.Hour,
	}
	now := time.Date(2026, 3, 3, 17, 30, 0, 0, time.UTC)

	store := newLongTermMemoryStore(options)
	store.Record(longTermMessage("wb-seed", "u-1", "tenant-a", "remember setting", now, map[string]string{
		longTermMemoryStrategyMetadataKey: "add",
		longTermMemoryKindMetadataKey:     "fact",
		longTermMemoryKeyMetadataKey:      "timezone",
		longTermMemoryValueMetadataKey:    "UTC+8",
	}), shareddomain.RouteNL, "")

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected no persisted file before write-back flush, got err=%v", err)
	}
	if err := store.Flush(); err != nil {
		t.Fatalf("flush write-back memory: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected persisted file after flush: %v", err)
	}

	reloaded := newLongTermMemoryStore(options)
	query := longTermMessage("wb-query", "u-1", "tenant-a", "timezone", now.Add(time.Minute), nil)
	snapshot := reloaded.Snapshot(query, query.Content, query.ReceivedAt)
	if len(snapshot.Hits) == 0 {
		t.Fatalf("expected write-back data recovered after flush and restart")
	}
}

func longTermMessage(
	sessionID string,
	userID string,
	tenantID string,
	content string,
	receivedAt time.Time,
	metadata map[string]string,
) shareddomain.UnifiedMessage {
	meta := map[string]string{}
	for key, value := range metadata {
		meta[key] = value
	}
	if strings.TrimSpace(tenantID) != "" {
		meta[longTermMemoryTenantMetadataKey] = tenantID
	}
	if len(meta) == 0 {
		meta = nil
	}

	return shareddomain.UnifiedMessage{
		MessageID:   fmt.Sprintf("m-%s", sessionID),
		SessionID:   sessionID,
		UserID:      userID,
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     content,
		Metadata:    meta,
		TraceID:     fmt.Sprintf("trace-%s", sessionID),
		ReceivedAt:  receivedAt,
	}
}

func getScopeEntries(store *longTermMemoryStore, scope longTermMemoryScope) []longTermMemoryEntry {
	store.mu.Lock()
	defer store.mu.Unlock()
	return copyLongTermMemoryEntries(store.scopes[scope.Key()])
}

package application

import (
	"path/filepath"
	"testing"
	"time"

	shareddomain "alter0/internal/shared/domain"
)

func TestTieredLongTermMemoryAcceptancePersistenceAndLifecycleObservability(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tiered_memory.json")
	options := LongTermMemoryOptions{
		MaxEntriesPerScope:   32,
		MaxHits:              6,
		MaxSnippet:           220,
		InjectionTokenBudget: 160,
		PersistencePath:      path,
		WritePolicy:          longTermMemoryWritePolicyWriteThrough,
		L1: LongTermMemoryTierOptions{
			MaxEntryLength: 220,
			MaxLayerTokens: 220,
			TTL:            24 * time.Hour,
			DemoteIdleTTL:  5 * time.Minute,
			EvictionPolicy: longTermMemoryEvictionPolicyLRU,
		},
		L2: LongTermMemoryTierOptions{
			MaxEntryLength: 220,
			MaxLayerTokens: 440,
			TTL:            24 * time.Hour,
			DemoteIdleTTL:  8 * time.Minute,
			PromoteHits:    1,
			EvictionPolicy: longTermMemoryEvictionPolicyLRU,
		},
		L3: LongTermMemoryTierOptions{
			MaxEntryLength: 220,
			MaxLayerTokens: 880,
			TTL:            24 * time.Hour,
			PromoteHits:    1,
			EvictionPolicy: longTermMemoryEvictionPolicyLRU,
		},
	}
	start := time.Date(2026, 3, 3, 18, 0, 0, 0, time.UTC)

	store := newLongTermMemoryStore(options)
	store.Record(longTermMessage("accept-seed", "u-1", "tenant-a", "remember this", start, map[string]string{
		longTermMemoryStrategyMetadataKey: "add",
		longTermMemoryKindMetadataKey:     "fact",
		longTermMemoryTierMetadataKey:     "L3",
		longTermMemoryKeyMetadataKey:      "release_guardrail",
		longTermMemoryValueMetadataKey:    "slo_target 99.95%",
	}), shareddomain.RouteNL, "")

	firstQuery := longTermMessage("accept-query-1", "u-1", "tenant-a", "Need release guardrail", start.Add(time.Minute), nil)
	firstSnapshot := store.Snapshot(firstQuery, firstQuery.Content, firstQuery.ReceivedAt)
	if len(firstSnapshot.Promotions) == 0 {
		t.Fatalf("expected promotion observable after hit")
	}
	if firstSnapshot.Metadata()["memory_long_term_hit_chain"] != "L1>L2>L3" {
		t.Fatalf("expected hit chain metadata")
	}

	idleQuery := longTermMessage("accept-query-2", "u-1", "tenant-a", "Need release guardrail", start.Add(20*time.Minute), nil)
	idleSnapshot := store.Snapshot(idleQuery, idleQuery.Content, idleQuery.ReceivedAt)
	if len(idleSnapshot.Demotions) == 0 {
		t.Fatalf("expected idle demotion observable")
	}

	reloaded := newLongTermMemoryStore(options)
	restoreQuery := longTermMessage("accept-query-3", "u-1", "tenant-a", "Need release guardrail", start.Add(21*time.Minute), nil)
	restoredSnapshot := reloaded.Snapshot(restoreQuery, restoreQuery.Content, restoreQuery.ReceivedAt)
	if len(restoredSnapshot.Hits) == 0 {
		t.Fatalf("expected persisted memory recovered after restart")
	}
}

func TestTieredLongTermMemoryAcceptanceHighValueHitRateImprovesWithSameBudget(t *testing.T) {
	baselineRate := runHighValueHitRateScenario(t, false)
	optimizedRate := runHighValueHitRateScenario(t, true)
	if optimizedRate <= baselineRate {
		t.Fatalf("expected optimized tiered memory hit rate improve, baseline=%.2f optimized=%.2f", baselineRate, optimizedRate)
	}
}

func runHighValueHitRateScenario(t *testing.T, important bool) float64 {
	t.Helper()

	store := newLongTermMemoryStore(LongTermMemoryOptions{
		MaxEntriesPerScope:   32,
		MaxHits:              3,
		MaxSnippet:           180,
		InjectionTokenBudget: 18,
		L1: LongTermMemoryTierOptions{
			MaxEntryLength: 180,
			MaxLayerTokens: 180,
			TTL:            time.Hour,
			EvictionPolicy: longTermMemoryEvictionPolicyLRU,
		},
		L2: LongTermMemoryTierOptions{
			MaxEntryLength: 180,
			MaxLayerTokens: 180,
			TTL:            time.Hour,
			EvictionPolicy: longTermMemoryEvictionPolicyLRU,
		},
		L3: LongTermMemoryTierOptions{
			MaxEntryLength: 180,
			MaxLayerTokens: 180,
			TTL:            time.Hour,
			EvictionPolicy: longTermMemoryEvictionPolicyLRU,
		},
	})
	start := time.Date(2026, 3, 3, 19, 0, 0, 0, time.UTC)

	store.Record(longTermMessage("scenario-l1", "u-2", "tenant-a", "seed", start, map[string]string{
		longTermMemoryStrategyMetadataKey: "add",
		longTermMemoryKindMetadataKey:     "fact",
		longTermMemoryTierMetadataKey:     "L1",
		longTermMemoryKeyMetadataKey:      "release_context",
		longTermMemoryValueMetadataKey:    "release operations baseline",
	}), shareddomain.RouteNL, "")
	store.Record(longTermMessage("scenario-l2", "u-2", "tenant-a", "seed", start.Add(time.Second), map[string]string{
		longTermMemoryStrategyMetadataKey: "add",
		longTermMemoryKindMetadataKey:     "fact",
		longTermMemoryTierMetadataKey:     "L2",
		longTermMemoryKeyMetadataKey:      "operations_context",
		longTermMemoryValueMetadataKey:    "release ops notes",
	}), shareddomain.RouteNL, "")

	highValueMetadata := map[string]string{
		longTermMemoryStrategyMetadataKey: "add",
		longTermMemoryKindMetadataKey:     "fact",
		longTermMemoryTierMetadataKey:     "L3",
		longTermMemoryKeyMetadataKey:      "slo_target",
		longTermMemoryValueMetadataKey:    "99.95 release objective",
	}
	if important {
		highValueMetadata[longTermMemoryImportantMetadataKey] = "true"
	}
	store.Record(longTermMessage("scenario-high", "u-2", "tenant-a", "seed", start.Add(2*time.Second), highValueMetadata), shareddomain.RouteNL, "")

	total := 8
	hits := 0
	for idx := 0; idx < total; idx++ {
		queryTime := start.Add(time.Duration(idx+1) * time.Minute)
		query := longTermMessage("scenario-query", "u-2", "tenant-a", "release operations slo target", queryTime, nil)
		snapshot := store.Snapshot(query, query.Content, query.ReceivedAt)
		for _, hit := range snapshot.Hits {
			if hit.Entry.Key == "slo_target" {
				hits++
				break
			}
		}
	}

	return float64(hits) / float64(total)
}

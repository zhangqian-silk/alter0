package application

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	shareddomain "alter0/internal/shared/domain"
)

const (
	defaultLongTermMemoryMaxEntriesPerScope = 128
	defaultLongTermMemoryMaxHits            = 6
	defaultLongTermMemoryMaxSnippet         = 240
	defaultLongTermMemoryInjectionBudget    = 220
	defaultLongTermMemoryL1MaxLayerTokens   = 240
	defaultLongTermMemoryL2MaxLayerTokens   = 640
	defaultLongTermMemoryL3MaxLayerTokens   = 1400
	defaultLongTermMemoryL1TTL              = 12 * time.Hour
	defaultLongTermMemoryL2TTL              = 7 * 24 * time.Hour
	defaultLongTermMemoryL3TTL              = 45 * 24 * time.Hour
	defaultLongTermMemoryL1DemoteIdle       = 6 * time.Hour
	defaultLongTermMemoryL2DemoteIdle       = 72 * time.Hour
	defaultLongTermMemoryL2PromoteHits      = 3
	defaultLongTermMemoryL3PromoteHits      = 2
	defaultLongTermMemoryTenantID           = "default"
	defaultLongTermMemoryUserID             = "anonymous"
	longTermMemoryTenantMetadataKey         = "tenant_id"
	longTermMemoryKindMetadataKey           = "memory_long_term_kind"
	longTermMemoryKeyMetadataKey            = "memory_long_term_key"
	longTermMemoryValueMetadataKey          = "memory_long_term_value"
	longTermMemoryTagsMetadataKey           = "memory_long_term_tags"
	longTermMemoryStrategyMetadataKey       = "memory_long_term_strategy"
	longTermMemoryTierMetadataKey           = "memory_long_term_tier"
	longTermMemoryTokenBudgetMetadataKey    = "memory_long_term_token_budget"
	longTermMemoryImportantMetadataKey      = "memory_long_term_important"
	longTermMemoryCurrentInputMarker        = "\nCurrent user input:\n"
)

var (
	longTermMemoryTokenPattern      = regexp.MustCompile(`[\p{Han}]+|[a-z0-9]+`)
	implicitPreferencePatternEN     = regexp.MustCompile(`(?i)^\s*(?:i\s+prefer|my\s+preference\s+is)\s+(.+)$`)
	implicitPreferencePatternZH     = regexp.MustCompile(`^\s*(?:我偏好|我的偏好是|我喜欢)\s*(.+)$`)
	implicitConstraintPatternEN     = regexp.MustCompile(`(?i)^\s*(?:always|please\s+always)\s+(.+)$`)
	implicitConstraintPatternZH     = regexp.MustCompile(`^\s*(?:请始终|必须)\s*(.+)$`)
	longTermMemorySeparatorPatterns = regexp.MustCompile(`[,\n;|]+`)
)

type LongTermMemoryOptions struct {
	MaxEntriesPerScope   int
	MaxHits              int
	MaxSnippet           int
	InjectionTokenBudget int
	DefaultTenantID      string
	DefaultUserID        string
	L1                   LongTermMemoryTierOptions
	L2                   LongTermMemoryTierOptions
	L3                   LongTermMemoryTierOptions
}

type LongTermMemoryTierOptions struct {
	MaxEntryLength int
	MaxLayerTokens int
	TTL            time.Duration
	DemoteIdleTTL  time.Duration
	PromoteHits    int
	EvictionPolicy LongTermMemoryEvictionPolicy
}

type LongTermMemoryTier string

const (
	longTermMemoryTierL1 LongTermMemoryTier = "L1"
	longTermMemoryTierL2 LongTermMemoryTier = "L2"
	longTermMemoryTierL3 LongTermMemoryTier = "L3"
)

var longTermMemoryTierOrder = []LongTermMemoryTier{
	longTermMemoryTierL1,
	longTermMemoryTierL2,
	longTermMemoryTierL3,
}

type LongTermMemoryEvictionPolicy string

const (
	longTermMemoryEvictionPolicyLRU   LongTermMemoryEvictionPolicy = "lru"
	longTermMemoryEvictionPolicyScore LongTermMemoryEvictionPolicy = "score"
)

type longTermMemoryStrategy string

const (
	longTermMemoryStrategyAdd        longTermMemoryStrategy = "add"
	longTermMemoryStrategyOverwrite  longTermMemoryStrategy = "overwrite"
	longTermMemoryStrategyInvalidate longTermMemoryStrategy = "invalidate"
)

type longTermMemoryKind string

const (
	longTermMemoryKindPreference longTermMemoryKind = "preference"
	longTermMemoryKindFact       longTermMemoryKind = "fact"
	longTermMemoryKindConstraint longTermMemoryKind = "constraint"
)

type longTermMemoryStatus string

const (
	longTermMemoryStatusActive      longTermMemoryStatus = "active"
	longTermMemoryStatusInvalidated longTermMemoryStatus = "invalidated"
)

type longTermMemoryScope struct {
	TenantID string
	UserID   string
}

func (s longTermMemoryScope) Key() string {
	return s.TenantID + "\x1f" + s.UserID
}

type longTermMemoryEntry struct {
	ID              string
	Scope           longTermMemoryScope
	Tier            LongTermMemoryTier
	Important       bool
	Kind            longTermMemoryKind
	Key             string
	Value           string
	Tags            []string
	Status          longTermMemoryStatus
	SourceSessionID string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	LastAccessAt    time.Time
	HitCount        int
	UpdatedBy       longTermMemoryStrategy
	Route           shareddomain.Route
}

type longTermMemoryHit struct {
	Entry longTermMemoryEntry
	Score float64
}

type longTermMemorySnapshot struct {
	Scope             longTermMemoryScope
	Hits              []longTermMemoryHit
	TierHits          map[LongTermMemoryTier][]longTermMemoryHit
	CandidateTierHits map[LongTermMemoryTier][]longTermMemoryHit
	Promotions        []longTermMemoryMigration
	Demotions         []longTermMemoryMigration
	TokenBudget       int
	TokenUsed         int
	Truncated         bool
}

type longTermMemoryMigration struct {
	EntryID string
	From    LongTermMemoryTier
	To      LongTermMemoryTier
	Reason  string
}

func (s longTermMemorySnapshot) Metadata() map[string]string {
	metadata := map[string]string{
		"memory_long_term_hit_count":              strconv.Itoa(len(s.Hits)),
		"memory_long_term_hit_count_l1":           strconv.Itoa(len(s.TierHits[longTermMemoryTierL1])),
		"memory_long_term_hit_count_l2":           strconv.Itoa(len(s.TierHits[longTermMemoryTierL2])),
		"memory_long_term_hit_count_l3":           strconv.Itoa(len(s.TierHits[longTermMemoryTierL3])),
		"memory_long_term_candidate_hit_count_l1": strconv.Itoa(len(s.CandidateTierHits[longTermMemoryTierL1])),
		"memory_long_term_candidate_hit_count_l2": strconv.Itoa(len(s.CandidateTierHits[longTermMemoryTierL2])),
		"memory_long_term_candidate_hit_count_l3": strconv.Itoa(len(s.CandidateTierHits[longTermMemoryTierL3])),
		"memory_long_term_hit_chain":              "L1>L2>L3",
		"memory_long_term_token_budget":           strconv.Itoa(s.TokenBudget),
		"memory_long_term_token_used":             strconv.Itoa(s.TokenUsed),
		"memory_long_term_truncated":              strconv.FormatBool(s.Truncated),
		"memory_long_term_promotion_count":        strconv.Itoa(len(s.Promotions)),
		"memory_long_term_demotion_count":         strconv.Itoa(len(s.Demotions)),
	}
	if len(s.Hits) > 0 {
		metadata["memory_long_term_scope_resolved"] = "true"
	}
	return metadata
}

func (s longTermMemorySnapshot) ResultMetadata() map[string]string {
	if len(s.Hits) == 0 {
		return nil
	}
	return map[string]string{
		"memory_long_term_injected":        "true",
		"memory_long_term_hit_count":       strconv.Itoa(len(s.Hits)),
		"memory_long_term_token_used":      strconv.Itoa(s.TokenUsed),
		"memory_long_term_truncated":       strconv.FormatBool(s.Truncated),
		"memory_long_term_promotion_count": strconv.Itoa(len(s.Promotions)),
		"memory_long_term_demotion_count":  strconv.Itoa(len(s.Demotions)),
	}
}

type longTermMemoryUpdate struct {
	Strategy        longTermMemoryStrategy
	Tier            LongTermMemoryTier
	Important       bool
	Kind            longTermMemoryKind
	Key             string
	Value           string
	Tags            []string
	SourceSessionID string
	UpdatedAt       time.Time
	Route           shareddomain.Route
}

type longTermMemoryStore struct {
	mu       sync.Mutex
	options  LongTermMemoryOptions
	sequence int64
	scopes   map[string][]longTermMemoryEntry
}

func newLongTermMemoryStore(options LongTermMemoryOptions) *longTermMemoryStore {
	normalized := normalizeLongTermMemoryOptions(options)
	return &longTermMemoryStore{
		options: normalized,
		scopes:  map[string][]longTermMemoryEntry{},
	}
}

func normalizeLongTermMemoryOptions(options LongTermMemoryOptions) LongTermMemoryOptions {
	if options.MaxEntriesPerScope <= 0 {
		options.MaxEntriesPerScope = defaultLongTermMemoryMaxEntriesPerScope
	}
	if options.MaxHits <= 0 {
		options.MaxHits = defaultLongTermMemoryMaxHits
	}
	if options.MaxSnippet <= 0 {
		options.MaxSnippet = defaultLongTermMemoryMaxSnippet
	}
	if options.InjectionTokenBudget <= 0 {
		options.InjectionTokenBudget = defaultLongTermMemoryInjectionBudget
	}
	if strings.TrimSpace(options.DefaultTenantID) == "" {
		options.DefaultTenantID = defaultLongTermMemoryTenantID
	}
	if strings.TrimSpace(options.DefaultUserID) == "" {
		options.DefaultUserID = defaultLongTermMemoryUserID
	}
	options.L1 = normalizeLongTermMemoryTierOptions(options.L1, LongTermMemoryTierOptions{
		MaxEntryLength: options.MaxSnippet,
		MaxLayerTokens: defaultLongTermMemoryL1MaxLayerTokens,
		TTL:            defaultLongTermMemoryL1TTL,
		DemoteIdleTTL:  defaultLongTermMemoryL1DemoteIdle,
		EvictionPolicy: longTermMemoryEvictionPolicyLRU,
	})
	options.L2 = normalizeLongTermMemoryTierOptions(options.L2, LongTermMemoryTierOptions{
		MaxEntryLength: options.MaxSnippet,
		MaxLayerTokens: defaultLongTermMemoryL2MaxLayerTokens,
		TTL:            defaultLongTermMemoryL2TTL,
		DemoteIdleTTL:  defaultLongTermMemoryL2DemoteIdle,
		PromoteHits:    defaultLongTermMemoryL2PromoteHits,
		EvictionPolicy: longTermMemoryEvictionPolicyLRU,
	})
	options.L3 = normalizeLongTermMemoryTierOptions(options.L3, LongTermMemoryTierOptions{
		MaxEntryLength: options.MaxSnippet,
		MaxLayerTokens: defaultLongTermMemoryL3MaxLayerTokens,
		TTL:            defaultLongTermMemoryL3TTL,
		PromoteHits:    defaultLongTermMemoryL3PromoteHits,
		EvictionPolicy: longTermMemoryEvictionPolicyLRU,
	})
	return options
}

func normalizeLongTermMemoryTierOptions(options LongTermMemoryTierOptions, defaults LongTermMemoryTierOptions) LongTermMemoryTierOptions {
	if options.MaxEntryLength <= 0 {
		options.MaxEntryLength = defaults.MaxEntryLength
	}
	if options.MaxLayerTokens <= 0 {
		options.MaxLayerTokens = defaults.MaxLayerTokens
	}
	if options.TTL <= 0 {
		options.TTL = defaults.TTL
	}
	if options.DemoteIdleTTL <= 0 {
		options.DemoteIdleTTL = defaults.DemoteIdleTTL
	}
	if options.PromoteHits <= 0 {
		options.PromoteHits = defaults.PromoteHits
	}
	switch options.EvictionPolicy {
	case longTermMemoryEvictionPolicyScore:
	default:
		options.EvictionPolicy = defaults.EvictionPolicy
	}
	return options
}

func (o LongTermMemoryOptions) TierOptions(tier LongTermMemoryTier) LongTermMemoryTierOptions {
	switch tier {
	case longTermMemoryTierL1:
		return o.L1
	case longTermMemoryTierL2:
		return o.L2
	default:
		return o.L3
	}
}

func (s *longTermMemoryStore) Snapshot(msg shareddomain.UnifiedMessage, query string, now time.Time) longTermMemorySnapshot {
	scope := resolveLongTermMemoryScope(msg, s.options)
	if now.IsZero() {
		now = time.Now().UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	scopeKey := scope.Key()
	entries := copyLongTermMemoryEntries(s.scopes[scopeKey])
	entries = applyLongTermMemoryTTL(entries, s.options, now)
	entries, demotions := applyLongTermMemoryDemotion(entries, s.options, now)

	candidateTierHits := buildLongTermMemoryTierHits(entries, query, s.options.MaxHits, now)
	tokenBudget := resolveLongTermMemoryTokenBudget(msg.Metadata, s.options.InjectionTokenBudget)
	hits, tierHits, tokenUsed, truncated := trimLongTermMemoryTierHitsByBudget(candidateTierHits, tokenBudget, s.options.MaxHits)
	entries, promotions := applyLongTermMemoryHitFeedback(entries, hits, s.options, now)
	entries = enforceLongTermMemoryTierCapacity(entries, s.options, now)
	if len(entries) == 0 {
		delete(s.scopes, scopeKey)
	} else {
		s.scopes[scopeKey] = copyLongTermMemoryEntries(entries)
	}
	return longTermMemorySnapshot{
		Scope:             scope,
		Hits:              hits,
		TierHits:          tierHits,
		CandidateTierHits: candidateTierHits,
		Promotions:        promotions,
		Demotions:         demotions,
		TokenBudget:       tokenBudget,
		TokenUsed:         tokenUsed,
		Truncated:         truncated,
	}
}

func (s *longTermMemoryStore) Record(msg shareddomain.UnifiedMessage, route shareddomain.Route, output string) {
	now := msg.ReceivedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}

	updates := resolveLongTermMemoryUpdates(msg, route, output, now, s.options.MaxSnippet)
	if len(updates) == 0 {
		return
	}

	scope := resolveLongTermMemoryScope(msg, s.options)
	scopeKey := scope.Key()

	s.mu.Lock()
	defer s.mu.Unlock()

	entries := s.scopes[scopeKey]
	entries = applyLongTermMemoryTTL(entries, s.options, now)
	entries, _ = applyLongTermMemoryDemotion(entries, s.options, now)
	for _, update := range updates {
		entries = applyLongTermMemoryUpdate(entries, scope, update, &s.sequence, s.options)
	}

	entries = enforceLongTermMemoryTierCapacity(entries, s.options, now)
	if len(entries) == 0 {
		delete(s.scopes, scopeKey)
		return
	}

	s.scopes[scopeKey] = copyLongTermMemoryEntries(entries)
}

func resolveLongTermMemoryScope(msg shareddomain.UnifiedMessage, options LongTermMemoryOptions) longTermMemoryScope {
	tenantID := strings.TrimSpace(options.DefaultTenantID)
	if metadataTenant := strings.TrimSpace(msg.Metadata[longTermMemoryTenantMetadataKey]); metadataTenant != "" {
		tenantID = metadataTenant
	}
	userID := strings.TrimSpace(options.DefaultUserID)
	if messageUser := strings.TrimSpace(msg.UserID); messageUser != "" {
		userID = messageUser
	}
	return longTermMemoryScope{
		TenantID: tenantID,
		UserID:   userID,
	}
}

func resolveLongTermMemoryUpdates(
	msg shareddomain.UnifiedMessage,
	route shareddomain.Route,
	output string,
	now time.Time,
	maxSnippet int,
) []longTermMemoryUpdate {
	if explicit, ok := parseExplicitLongTermMemoryUpdate(msg, route, output, now, maxSnippet); ok {
		return []longTermMemoryUpdate{explicit}
	}
	if inferred, ok := inferImplicitLongTermMemoryUpdate(msg, route, now, maxSnippet); ok {
		return []longTermMemoryUpdate{inferred}
	}
	return nil
}

func parseExplicitLongTermMemoryUpdate(
	msg shareddomain.UnifiedMessage,
	route shareddomain.Route,
	output string,
	now time.Time,
	maxSnippet int,
) (longTermMemoryUpdate, bool) {
	if len(msg.Metadata) == 0 {
		return longTermMemoryUpdate{}, false
	}

	key := normalizeLongTermMemoryKey(msg.Metadata[longTermMemoryKeyMetadataKey])
	if key == "" {
		return longTermMemoryUpdate{}, false
	}

	strategy := normalizeLongTermMemoryStrategy(msg.Metadata[longTermMemoryStrategyMetadataKey])
	kind := normalizeLongTermMemoryKind(msg.Metadata[longTermMemoryKindMetadataKey])
	value := strings.TrimSpace(msg.Metadata[longTermMemoryValueMetadataKey])
	if value == "" && strategy != longTermMemoryStrategyInvalidate {
		value = strings.TrimSpace(msg.Content)
	}
	if value == "" && strategy != longTermMemoryStrategyInvalidate {
		value = strings.TrimSpace(output)
	}
	if strategy != longTermMemoryStrategyInvalidate {
		value = normalizeSnippet(value, maxSnippet)
		if value == "" {
			return longTermMemoryUpdate{}, false
		}
	}

	return longTermMemoryUpdate{
		Strategy:        strategy,
		Tier:            normalizeLongTermMemoryTier(msg.Metadata[longTermMemoryTierMetadataKey]),
		Important:       parseLongTermMemoryImportant(msg.Metadata[longTermMemoryImportantMetadataKey]),
		Kind:            kind,
		Key:             key,
		Value:           value,
		Tags:            parseLongTermMemoryTags(msg.Metadata[longTermMemoryTagsMetadataKey]),
		SourceSessionID: msg.SessionID,
		UpdatedAt:       now,
		Route:           route,
	}, true
}

func inferImplicitLongTermMemoryUpdate(
	msg shareddomain.UnifiedMessage,
	route shareddomain.Route,
	now time.Time,
	maxSnippet int,
) (longTermMemoryUpdate, bool) {
	trimmed := strings.TrimSpace(msg.Content)
	if trimmed == "" {
		return longTermMemoryUpdate{}, false
	}

	if value, ok := captureImplicitMemoryValue(trimmed, implicitPreferencePatternEN, implicitPreferencePatternZH); ok {
		return longTermMemoryUpdate{
			Strategy:        longTermMemoryStrategyOverwrite,
			Tier:            longTermMemoryTierL2,
			Kind:            longTermMemoryKindPreference,
			Key:             "default_preference",
			Value:           normalizeSnippet(value, maxSnippet),
			Tags:            []string{"implicit", "preference"},
			SourceSessionID: msg.SessionID,
			UpdatedAt:       now,
			Route:           route,
		}, true
	}

	if value, ok := captureImplicitMemoryValue(trimmed, implicitConstraintPatternEN, implicitConstraintPatternZH); ok {
		return longTermMemoryUpdate{
			Strategy:        longTermMemoryStrategyOverwrite,
			Tier:            longTermMemoryTierL2,
			Kind:            longTermMemoryKindConstraint,
			Key:             "default_constraint",
			Value:           normalizeSnippet(value, maxSnippet),
			Tags:            []string{"constraint", "implicit"},
			SourceSessionID: msg.SessionID,
			UpdatedAt:       now,
			Route:           route,
		}, true
	}

	return longTermMemoryUpdate{}, false
}

func captureImplicitMemoryValue(content string, patterns ...*regexp.Regexp) (string, bool) {
	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(content)
		if len(matches) < 2 {
			continue
		}
		value := strings.TrimSpace(matches[1])
		if value == "" {
			continue
		}
		return value, true
	}
	return "", false
}

func applyLongTermMemoryUpdate(
	entries []longTermMemoryEntry,
	scope longTermMemoryScope,
	update longTermMemoryUpdate,
	sequence *int64,
	options LongTermMemoryOptions,
) []longTermMemoryEntry {
	update.Strategy = normalizeLongTermMemoryStrategy(string(update.Strategy))
	update.Tier = resolveLongTermMemoryUpdateTier(update)
	update.Kind = normalizeLongTermMemoryKind(string(update.Kind))
	update.Key = normalizeLongTermMemoryKey(update.Key)
	maxEntryLength := options.TierOptions(update.Tier).MaxEntryLength
	if maxEntryLength <= 0 {
		maxEntryLength = options.MaxSnippet
	}
	update.Value = normalizeSnippet(update.Value, maxEntryLength)
	update.Tags = normalizeLongTermMemoryTags(update.Tags)
	if update.Key == "" {
		return entries
	}
	if update.UpdatedAt.IsZero() {
		update.UpdatedAt = time.Now().UTC()
	}

	switch update.Strategy {
	case longTermMemoryStrategyInvalidate:
		for idx := range entries {
			if entries[idx].Status != longTermMemoryStatusActive {
				continue
			}
			if !isSameLongTermMemoryKey(entries[idx], update.Kind, update.Key) {
				continue
			}
			entries[idx].Status = longTermMemoryStatusInvalidated
			entries[idx].UpdatedAt = update.UpdatedAt
			entries[idx].UpdatedBy = update.Strategy
			entries[idx].SourceSessionID = update.SourceSessionID
			entries[idx].Route = update.Route
		}
		return entries
	case longTermMemoryStrategyOverwrite:
		for idx := range entries {
			if entries[idx].Status != longTermMemoryStatusActive {
				continue
			}
			if !isSameLongTermMemoryKey(entries[idx], update.Kind, update.Key) {
				continue
			}
			entries[idx].Status = longTermMemoryStatusInvalidated
			entries[idx].UpdatedAt = update.UpdatedAt
			entries[idx].UpdatedBy = update.Strategy
			entries[idx].SourceSessionID = update.SourceSessionID
			entries[idx].Route = update.Route
		}
	case longTermMemoryStrategyAdd:
	default:
		update.Strategy = longTermMemoryStrategyAdd
	}

	if update.Value == "" {
		return entries
	}

	*sequence++
	entry := longTermMemoryEntry{
		ID:              fmt.Sprintf("ltm-%d", *sequence),
		Scope:           scope,
		Tier:            update.Tier,
		Important:       update.Important,
		Kind:            update.Kind,
		Key:             update.Key,
		Value:           update.Value,
		Tags:            update.Tags,
		Status:          longTermMemoryStatusActive,
		SourceSessionID: update.SourceSessionID,
		CreatedAt:       update.UpdatedAt,
		UpdatedAt:       update.UpdatedAt,
		LastAccessAt:    update.UpdatedAt,
		UpdatedBy:       update.Strategy,
		Route:           update.Route,
	}
	return append(entries, entry)
}

func applyLongTermMemoryTTL(
	entries []longTermMemoryEntry,
	options LongTermMemoryOptions,
	now time.Time,
) []longTermMemoryEntry {
	if len(entries) == 0 {
		return nil
	}
	filtered := make([]longTermMemoryEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Status != longTermMemoryStatusActive {
			filtered = append(filtered, entry)
			continue
		}
		tierOptions := options.TierOptions(entry.Tier)
		if tierOptions.TTL <= 0 {
			filtered = append(filtered, entry)
			continue
		}
		marker := entry.UpdatedAt
		if !entry.LastAccessAt.IsZero() && entry.LastAccessAt.After(marker) {
			marker = entry.LastAccessAt
		}
		if marker.IsZero() {
			marker = now
		}
		if marker.Add(tierOptions.TTL).Before(now) {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

func enforceLongTermMemoryTierCapacity(
	entries []longTermMemoryEntry,
	options LongTermMemoryOptions,
	now time.Time,
) []longTermMemoryEntry {
	trimmed := copyLongTermMemoryEntries(entries)
	for _, tier := range longTermMemoryTierOrder {
		tierOptions := options.TierOptions(tier)
		if tierOptions.MaxLayerTokens <= 0 {
			continue
		}
		for longTermMemoryLayerTokenUsage(trimmed, tier) > tierOptions.MaxLayerTokens {
			evictIdx := selectLongTermMemoryEvictIndex(trimmed, tier, tierOptions.EvictionPolicy, now)
			if evictIdx < 0 {
				break
			}
			trimmed = append(trimmed[:evictIdx], trimmed[evictIdx+1:]...)
		}
	}
	if options.MaxEntriesPerScope > 0 {
		for activeLongTermMemoryEntryCount(trimmed) > options.MaxEntriesPerScope {
			evictIdx := selectLongTermMemoryGlobalEvictIndex(trimmed, now)
			if evictIdx < 0 {
				break
			}
			trimmed = append(trimmed[:evictIdx], trimmed[evictIdx+1:]...)
		}
	}
	return trimmed
}

func longTermMemoryLayerTokenUsage(entries []longTermMemoryEntry, tier LongTermMemoryTier) int {
	total := 0
	for _, entry := range entries {
		if entry.Status != longTermMemoryStatusActive || entry.Tier != tier {
			continue
		}
		total += estimateLongTermMemoryEntryTokens(entry)
	}
	return total
}

func estimateLongTermMemoryEntryTokens(entry longTermMemoryEntry) int {
	text := strings.Join([]string{
		string(entry.Kind),
		entry.Key,
		entry.Value,
		strings.Join(entry.Tags, " "),
	}, " ")
	return estimateTokenCount(text) + 4
}

func selectLongTermMemoryEvictIndex(
	entries []longTermMemoryEntry,
	tier LongTermMemoryTier,
	policy LongTermMemoryEvictionPolicy,
	now time.Time,
) int {
	candidate := -1
	for idx := range entries {
		if entries[idx].Status != longTermMemoryStatusActive || entries[idx].Tier != tier {
			continue
		}
		if candidate < 0 {
			candidate = idx
			continue
		}
		switch policy {
		case longTermMemoryEvictionPolicyScore:
			leftScore := longTermMemoryRetentionScore(entries[idx], now)
			rightScore := longTermMemoryRetentionScore(entries[candidate], now)
			if leftScore < rightScore {
				candidate = idx
			}
		default:
			if longTermMemoryLRUAt(entries[idx]).Before(longTermMemoryLRUAt(entries[candidate])) {
				candidate = idx
			}
		}
	}
	return candidate
}

func selectLongTermMemoryGlobalEvictIndex(entries []longTermMemoryEntry, now time.Time) int {
	candidate := -1
	for idx := range entries {
		if entries[idx].Status != longTermMemoryStatusActive {
			continue
		}
		if candidate < 0 {
			candidate = idx
			continue
		}
		if entries[idx].Tier != entries[candidate].Tier {
			if longTermMemoryTierRank(entries[idx].Tier) > longTermMemoryTierRank(entries[candidate].Tier) {
				candidate = idx
			}
			continue
		}
		if longTermMemoryRetentionScore(entries[idx], now) < longTermMemoryRetentionScore(entries[candidate], now) {
			candidate = idx
		}
	}
	return candidate
}

func longTermMemoryRetentionScore(entry longTermMemoryEntry, now time.Time) float64 {
	base := float64(entry.HitCount)
	if base < 0 {
		base = 0
	}
	age := now.Sub(longTermMemoryLRUAt(entry)).Hours()
	if age < 0 {
		age = 0
	}
	return base + (1.0 / (1.0 + age/24.0))
}

func longTermMemoryLRUAt(entry longTermMemoryEntry) time.Time {
	if !entry.LastAccessAt.IsZero() {
		return entry.LastAccessAt
	}
	if !entry.UpdatedAt.IsZero() {
		return entry.UpdatedAt
	}
	return entry.CreatedAt
}

func activeLongTermMemoryEntryCount(entries []longTermMemoryEntry) int {
	count := 0
	for _, entry := range entries {
		if entry.Status == longTermMemoryStatusActive {
			count++
		}
	}
	return count
}

func applyLongTermMemoryDemotion(
	entries []longTermMemoryEntry,
	options LongTermMemoryOptions,
	now time.Time,
) ([]longTermMemoryEntry, []longTermMemoryMigration) {
	if len(entries) == 0 {
		return nil, nil
	}
	updated := copyLongTermMemoryEntries(entries)
	migrations := make([]longTermMemoryMigration, 0)
	for idx := range updated {
		entry := updated[idx]
		if entry.Status != longTermMemoryStatusActive {
			continue
		}
		tierOptions := options.TierOptions(entry.Tier)
		if tierOptions.DemoteIdleTTL <= 0 {
			continue
		}
		idleMarker := longTermMemoryLRUAt(entry)
		if idleMarker.IsZero() || now.Sub(idleMarker) < tierOptions.DemoteIdleTTL {
			continue
		}
		nextTier := demoteLongTermMemoryTier(entry.Tier)
		if nextTier == entry.Tier {
			continue
		}
		updated[idx].Tier = nextTier
		updated[idx].HitCount = 0
		updated[idx].UpdatedAt = now
		migrations = append(migrations, longTermMemoryMigration{
			EntryID: entry.ID,
			From:    entry.Tier,
			To:      nextTier,
			Reason:  "idle_demotion",
		})
	}
	return updated, migrations
}

func applyLongTermMemoryHitFeedback(
	entries []longTermMemoryEntry,
	hits []longTermMemoryHit,
	options LongTermMemoryOptions,
	now time.Time,
) ([]longTermMemoryEntry, []longTermMemoryMigration) {
	if len(entries) == 0 || len(hits) == 0 {
		return entries, nil
	}
	updated := copyLongTermMemoryEntries(entries)
	indexByID := make(map[string]int, len(updated))
	for idx, entry := range updated {
		indexByID[entry.ID] = idx
	}

	migrations := make([]longTermMemoryMigration, 0)
	for _, hit := range hits {
		idx, ok := indexByID[hit.Entry.ID]
		if !ok {
			continue
		}
		entry := updated[idx]
		if entry.Status != longTermMemoryStatusActive {
			continue
		}
		entry.LastAccessAt = now
		entry.HitCount++

		shouldPromote := entry.Important && entry.Tier != longTermMemoryTierL1
		if !shouldPromote {
			promoteHits := options.TierOptions(entry.Tier).PromoteHits
			if promoteHits > 0 && entry.HitCount >= promoteHits {
				shouldPromote = true
			}
		}
		if shouldPromote {
			nextTier := promoteLongTermMemoryTier(entry.Tier)
			if nextTier != entry.Tier {
				entry.HitCount = 0
				entry.Tier = nextTier
				maxEntryLength := options.TierOptions(nextTier).MaxEntryLength
				if maxEntryLength <= 0 {
					maxEntryLength = options.MaxSnippet
				}
				entry.Value = normalizeSnippet(entry.Value, maxEntryLength)
				migrations = append(migrations, longTermMemoryMigration{
					EntryID: entry.ID,
					From:    updated[idx].Tier,
					To:      nextTier,
					Reason:  "hit_promotion",
				})
			}
		}
		updated[idx] = entry
	}
	return updated, migrations
}

func promoteLongTermMemoryTier(tier LongTermMemoryTier) LongTermMemoryTier {
	switch tier {
	case longTermMemoryTierL3:
		return longTermMemoryTierL2
	case longTermMemoryTierL2:
		return longTermMemoryTierL1
	default:
		return longTermMemoryTierL1
	}
}

func demoteLongTermMemoryTier(tier LongTermMemoryTier) LongTermMemoryTier {
	switch tier {
	case longTermMemoryTierL1:
		return longTermMemoryTierL2
	case longTermMemoryTierL2:
		return longTermMemoryTierL3
	default:
		return longTermMemoryTierL3
	}
}

func buildLongTermMemoryTierHits(
	entries []longTermMemoryEntry,
	query string,
	limit int,
	now time.Time,
) map[LongTermMemoryTier][]longTermMemoryHit {
	grouped := map[LongTermMemoryTier][]longTermMemoryHit{
		longTermMemoryTierL1: {},
		longTermMemoryTierL2: {},
		longTermMemoryTierL3: {},
	}
	if limit <= 0 || len(entries) == 0 {
		return grouped
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return grouped
	}

	queryLower := strings.ToLower(query)
	queryTokens := tokenizeLongTermMemory(queryLower)
	if len(queryTokens) == 0 {
		queryTokens = []string{queryLower}
	}

	hits := make([]longTermMemoryHit, 0, len(entries))
	for _, entry := range entries {
		if entry.Status != longTermMemoryStatusActive {
			continue
		}
		score := scoreLongTermMemoryHit(queryLower, queryTokens, entry, now)
		if score <= 0 {
			continue
		}
		hits = append(hits, longTermMemoryHit{
			Entry: entry,
			Score: score,
		})
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			if hits[i].Entry.UpdatedAt.Equal(hits[j].Entry.UpdatedAt) {
				return hits[i].Entry.ID > hits[j].Entry.ID
			}
			return hits[i].Entry.UpdatedAt.After(hits[j].Entry.UpdatedAt)
		}
		return hits[i].Score > hits[j].Score
	})
	for _, hit := range hits {
		tier := normalizeLongTermMemoryTier(string(hit.Entry.Tier))
		grouped[tier] = append(grouped[tier], hit)
	}
	for _, tier := range longTermMemoryTierOrder {
		if len(grouped[tier]) > limit {
			grouped[tier] = grouped[tier][:limit]
		}
	}
	return grouped
}

func trimLongTermMemoryTierHitsByBudget(
	candidateTierHits map[LongTermMemoryTier][]longTermMemoryHit,
	tokenBudget int,
	limit int,
) ([]longTermMemoryHit, map[LongTermMemoryTier][]longTermMemoryHit, int, bool) {
	selectedTierHits := map[LongTermMemoryTier][]longTermMemoryHit{
		longTermMemoryTierL1: {},
		longTermMemoryTierL2: {},
		longTermMemoryTierL3: {},
	}
	if limit <= 0 {
		return nil, selectedTierHits, 0, false
	}

	selected := make([]longTermMemoryHit, 0, limit)
	tokenUsed := 0
	truncated := false

	for _, tier := range longTermMemoryTierOrder {
		candidates := candidateTierHits[tier]
		for _, hit := range candidates {
			if len(selected) >= limit {
				truncated = true
				break
			}
			cost := estimateLongTermMemoryEntryTokens(hit.Entry)
			if tokenBudget > 0 && tokenUsed+cost > tokenBudget {
				truncated = true
				continue
			}
			selected = append(selected, hit)
			selectedTierHits[tier] = append(selectedTierHits[tier], hit)
			tokenUsed += cost
		}
	}

	if len(selected) == 0 {
		for _, tier := range longTermMemoryTierOrder {
			candidates := candidateTierHits[tier]
			if len(candidates) == 0 {
				continue
			}
			selected = append(selected, candidates[0])
			selectedTierHits[tier] = append(selectedTierHits[tier], candidates[0])
			tokenUsed = estimateLongTermMemoryEntryTokens(candidates[0].Entry)
			if tokenBudget > 0 && tokenUsed > tokenBudget {
				truncated = true
			}
			break
		}
	}
	return selected, selectedTierHits, tokenUsed, truncated
}

func groupSelectedLongTermMemoryHitsByTier(hits []longTermMemoryHit) map[LongTermMemoryTier][]longTermMemoryHit {
	grouped := map[LongTermMemoryTier][]longTermMemoryHit{
		longTermMemoryTierL1: {},
		longTermMemoryTierL2: {},
		longTermMemoryTierL3: {},
	}
	for _, hit := range hits {
		tier := normalizeLongTermMemoryTier(string(hit.Entry.Tier))
		grouped[tier] = append(grouped[tier], hit)
	}
	return grouped
}

func scoreLongTermMemoryHit(
	queryLower string,
	queryTokens []string,
	entry longTermMemoryEntry,
	now time.Time,
) float64 {
	text := strings.ToLower(strings.Join([]string{
		string(entry.Kind),
		entry.Key,
		entry.Value,
		strings.Join(entry.Tags, " "),
	}, " "))

	score := 0.0
	if strings.Contains(text, queryLower) {
		score += 3
	}
	for _, token := range queryTokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		if strings.Contains(text, token) {
			score += 1
		}
	}
	if score == 0 {
		return 0
	}

	switch entry.Kind {
	case longTermMemoryKindConstraint:
		score += 0.4
	case longTermMemoryKindPreference:
		score += 0.2
	}
	if entry.Important {
		score += 0.8
	}
	score += float64(entry.HitCount) * 0.1

	ageHours := now.Sub(longTermMemoryLRUAt(entry)).Hours()
	if ageHours < 0 {
		ageHours = 0
	}
	recency := 0.5 / (1 + ageHours/24.0)
	return score + recency
}

func buildLongTermMemoryPrompt(prompt string, snapshot longTermMemorySnapshot) string {
	trimmedPrompt := strings.TrimSpace(prompt)
	if len(snapshot.Hits) == 0 {
		return trimmedPrompt
	}

	section := renderLongTermMemorySection(snapshot)
	if trimmedPrompt == "" {
		return section
	}

	if markerIndex := strings.LastIndex(trimmedPrompt, longTermMemoryCurrentInputMarker); markerIndex >= 0 {
		var builder strings.Builder
		builder.WriteString(trimmedPrompt[:markerIndex])
		builder.WriteString("\n\n")
		builder.WriteString(section)
		builder.WriteString(trimmedPrompt[markerIndex:])
		return builder.String()
	}

	var builder strings.Builder
	builder.WriteString(section)
	builder.WriteString("\nCurrent user input:\n")
	builder.WriteString(trimmedPrompt)
	return builder.String()
}

func renderLongTermMemorySection(snapshot longTermMemorySnapshot) string {
	var builder strings.Builder
	builder.WriteString("[LONG TERM MEMORY]\n")
	builder.WriteString("Scope: tenant=")
	builder.WriteString(snapshot.Scope.TenantID)
	builder.WriteString(", user=")
	builder.WriteString(snapshot.Scope.UserID)
	builder.WriteByte('\n')
	builder.WriteString("Hit chain: L1 -> L2 -> L3\n")
	builder.WriteString("Token budget: ")
	builder.WriteString(strconv.Itoa(snapshot.TokenUsed))
	builder.WriteString("/")
	builder.WriteString(strconv.Itoa(snapshot.TokenBudget))
	builder.WriteString("\n")
	builder.WriteString("Migrations: promotions=")
	builder.WriteString(strconv.Itoa(len(snapshot.Promotions)))
	builder.WriteString(", demotions=")
	builder.WriteString(strconv.Itoa(len(snapshot.Demotions)))
	builder.WriteString("\n")
	builder.WriteString("Relevant entries:\n")
	for idx, hit := range snapshot.Hits {
		builder.WriteString(fmt.Sprintf("%d) [%s/%s] %s: %s\n", idx+1, hit.Entry.Tier, hit.Entry.Kind, hit.Entry.Key, hit.Entry.Value))
		if len(hit.Entry.Tags) > 0 {
			builder.WriteString("   tags: ")
			builder.WriteString(strings.Join(hit.Entry.Tags, ", "))
			builder.WriteByte('\n')
		}
		builder.WriteString("   source_session: ")
		builder.WriteString(hit.Entry.SourceSessionID)
		builder.WriteByte('\n')
		builder.WriteString("   updated_at: ")
		builder.WriteString(hit.Entry.UpdatedAt.UTC().Format(time.RFC3339))
		builder.WriteByte('\n')
		builder.WriteString("   relevance: ")
		builder.WriteString(fmt.Sprintf("%.2f", hit.Score))
		builder.WriteByte('\n')
	}
	return builder.String()
}

func copyLongTermMemoryEntries(entries []longTermMemoryEntry) []longTermMemoryEntry {
	if len(entries) == 0 {
		return nil
	}
	copied := make([]longTermMemoryEntry, len(entries))
	copy(copied, entries)
	for idx := range copied {
		copied[idx].Tags = append([]string(nil), entries[idx].Tags...)
	}
	return copied
}

func isSameLongTermMemoryKey(entry longTermMemoryEntry, kind longTermMemoryKind, key string) bool {
	return entry.Kind == kind && entry.Key == key
}

func tokenizeLongTermMemory(content string) []string {
	if strings.TrimSpace(content) == "" {
		return nil
	}
	matches := longTermMemoryTokenPattern.FindAllString(strings.ToLower(content), -1)
	if len(matches) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	tokens := make([]string, 0, len(matches))
	for _, token := range matches {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		tokens = append(tokens, token)
	}
	return tokens
}

func parseLongTermMemoryTags(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := longTermMemorySeparatorPatterns.Split(raw, -1)
	return normalizeLongTermMemoryTags(parts)
}

func resolveLongTermMemoryTokenBudget(metadata map[string]string, fallback int) int {
	if fallback <= 0 {
		fallback = defaultLongTermMemoryInjectionBudget
	}
	if len(metadata) == 0 {
		return fallback
	}
	raw := strings.TrimSpace(metadata[longTermMemoryTokenBudgetMetadataKey])
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func parseLongTermMemoryImportant(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "important", "high":
		return true
	default:
		return false
	}
}

func normalizeLongTermMemoryStrategy(raw string) longTermMemoryStrategy {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(longTermMemoryStrategyOverwrite):
		return longTermMemoryStrategyOverwrite
	case string(longTermMemoryStrategyInvalidate):
		return longTermMemoryStrategyInvalidate
	default:
		return longTermMemoryStrategyAdd
	}
}

func normalizeLongTermMemoryKind(raw string) longTermMemoryKind {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(longTermMemoryKindFact):
		return longTermMemoryKindFact
	case string(longTermMemoryKindConstraint):
		return longTermMemoryKindConstraint
	default:
		return longTermMemoryKindPreference
	}
}

func normalizeLongTermMemoryTier(raw string) LongTermMemoryTier {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case string(longTermMemoryTierL1):
		return longTermMemoryTierL1
	case string(longTermMemoryTierL2):
		return longTermMemoryTierL2
	default:
		return longTermMemoryTierL3
	}
}

func resolveLongTermMemoryUpdateTier(update longTermMemoryUpdate) LongTermMemoryTier {
	if update.Important {
		return longTermMemoryTierL1
	}
	if tier := normalizeLongTermMemoryTier(string(update.Tier)); tier != longTermMemoryTierL3 || strings.TrimSpace(string(update.Tier)) != "" {
		return tier
	}
	switch update.Kind {
	case longTermMemoryKindConstraint:
		return longTermMemoryTierL2
	default:
		return longTermMemoryTierL3
	}
}

func longTermMemoryTierRank(tier LongTermMemoryTier) int {
	switch tier {
	case longTermMemoryTierL1:
		return 0
	case longTermMemoryTierL2:
		return 1
	default:
		return 2
	}
}

func normalizeLongTermMemoryTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = normalizeLongTermMemoryKey(tag)
		if tag == "" {
			continue
		}
		if _, exists := seen[tag]; exists {
			continue
		}
		seen[tag] = struct{}{}
		normalized = append(normalized, tag)
	}
	if len(normalized) == 0 {
		return nil
	}
	sort.Strings(normalized)
	return normalized
}

func normalizeLongTermMemoryKey(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return ""
	}

	var builder strings.Builder
	wasSeparator := false
	for _, r := range raw {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			builder.WriteRune(r)
			wasSeparator = false
			continue
		}
		if !wasSeparator {
			builder.WriteByte('_')
			wasSeparator = true
		}
	}

	return strings.Trim(builder.String(), "_")
}

package application

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	sessionMemoryDailyTierL1 = "L1"
	sessionMemoryDailyTierL2 = "L2"
	sessionMemoryDailyTierL3 = "L3"
)

var (
	sessionMemoryDailyFilePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\.md$`)
	sessionMemoryDailyTierOrder   = []string{
		sessionMemoryDailyTierL1,
		sessionMemoryDailyTierL2,
		sessionMemoryDailyTierL3,
	}
)

type sessionMemoryDailyStore struct {
	mu      sync.Mutex
	options SessionMemoryOptions
}

type sessionMemoryDailyState struct {
	Date       string                              `json:"date"`
	Generated  time.Time                           `json:"generated_at"`
	TierValues map[string][]sessionMemoryDailyItem `json:"tiers"`
}

type sessionMemoryDailyItem struct {
	EntryID     string                       `json:"entry_id"`
	Tier        string                       `json:"tier"`
	Source      string                       `json:"source"`
	SessionID   string                       `json:"session_id"`
	Route       string                       `json:"route,omitempty"`
	Summary     string                       `json:"summary"`
	KeyFacts    []sessionMemoryFact          `json:"key_facts,omitempty"`
	SourceTurns []sessionMemoryTurnReference `json:"source_turn_refs,omitempty"`
	OccurredAt  time.Time                    `json:"occurred_at"`
}

type sessionMemoryLongTermCandidateState struct {
	Date       string                   `json:"date"`
	Generated  time.Time                `json:"generated_at"`
	Candidates []sessionMemoryDailyItem `json:"candidates"`
}

func newSessionMemoryDailyStore(options SessionMemoryOptions) *sessionMemoryDailyStore {
	if strings.TrimSpace(options.DailyMemoryDir) == "" {
		return nil
	}
	return &sessionMemoryDailyStore{
		options: options,
	}
}

func (s *sessionMemoryDailyStore) Record(
	sessionID string,
	turn sessionMemoryTurn,
	fragments []sessionMemoryFragment,
) {
	if s == nil || strings.TrimSpace(s.options.DailyMemoryDir) == "" {
		return
	}
	occurredAt := turn.RecordedAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	occurredAt = occurredAt.UTC()
	day := occurredAt.Format("2006-01-02")

	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.loadStateByDayLocked(day)
	if err != nil {
		state = newSessionMemoryDailyState(day)
	}

	turnSummary := fmt.Sprintf("user: %s | assistant: %s", turn.UserInput, turn.AssistantOutput)
	turnItem := sessionMemoryDailyItem{
		EntryID:    fmt.Sprintf("turn-%s-%d", sanitizeSessionMemoryDailyID(sessionID), turn.Sequence),
		Tier:       sessionMemoryDailyTierL1,
		Source:     "turn",
		SessionID:  sessionID,
		Route:      string(turn.Route),
		Summary:    normalizeSnippet(turnSummary, s.options.L1.MaxEntryLength),
		OccurredAt: occurredAt,
	}
	state.TierValues[sessionMemoryDailyTierL1] = upsertSessionMemoryDailyItem(state.TierValues[sessionMemoryDailyTierL1], turnItem)

	for _, fragment := range fragments {
		fragmentAt := fragment.LastSource
		if fragmentAt.IsZero() {
			fragmentAt = fragment.CreatedAt
		}
		if fragmentAt.IsZero() {
			fragmentAt = occurredAt
		}
		fragmentItem := sessionMemoryDailyItem{
			EntryID:     fmt.Sprintf("fragment-%s-%s", sanitizeSessionMemoryDailyID(sessionID), strings.TrimSpace(fragment.FragmentID)),
			Tier:        sessionMemoryDailyTierL2,
			Source:      "compressed_fragment",
			SessionID:   sessionID,
			Summary:     normalizeSnippet(fragment.Summary, s.options.L2.MaxEntryLength),
			KeyFacts:    limitSessionMemoryFacts(fragment.KeyFacts, s.options.CompressionMaxFacts, s.options.L2.MaxEntryLength),
			SourceTurns: copyTurnReferences(fragment.SourceTurns),
			OccurredAt:  fragmentAt.UTC(),
		}
		state.TierValues[sessionMemoryDailyTierL2] = upsertSessionMemoryDailyItem(state.TierValues[sessionMemoryDailyTierL2], fragmentItem)
	}

	digest := buildSessionMemoryDailyDigest(day, state.TierValues[sessionMemoryDailyTierL2], occurredAt, s.options)
	if digest.EntryID != "" {
		state.TierValues[sessionMemoryDailyTierL3] = upsertSessionMemoryDailyItem(state.TierValues[sessionMemoryDailyTierL3], digest)
	}

	state = applySessionMemoryDailyTierConstraints(state, occurredAt, s.options)
	state.Generated = occurredAt
	if err := s.persistStateByDayLocked(day, state); err != nil {
		return
	}
	_ = s.persistLongTermCandidateByDayLocked(day, state, occurredAt)
	s.pruneExpiredDailyFilesLocked(occurredAt)
}

func newSessionMemoryDailyState(day string) sessionMemoryDailyState {
	return sessionMemoryDailyState{
		Date: day,
		TierValues: map[string][]sessionMemoryDailyItem{
			sessionMemoryDailyTierL1: {},
			sessionMemoryDailyTierL2: {},
			sessionMemoryDailyTierL3: {},
		},
	}
}

func (s *sessionMemoryDailyStore) loadStateByDayLocked(day string) (sessionMemoryDailyState, error) {
	path := filepath.Join(s.options.DailyMemoryDir, day+".md")
	raw, err := os.ReadFile(path)
	if err != nil {
		return sessionMemoryDailyState{}, err
	}
	state := newSessionMemoryDailyState(day)
	if err := parseSessionMemoryDailyState(raw, &state); err != nil {
		return sessionMemoryDailyState{}, err
	}
	if state.Date == "" {
		state.Date = day
	}
	for _, tier := range sessionMemoryDailyTierOrder {
		if state.TierValues[tier] == nil {
			state.TierValues[tier] = []sessionMemoryDailyItem{}
		}
	}
	return state, nil
}

func (s *sessionMemoryDailyStore) persistStateByDayLocked(day string, state sessionMemoryDailyState) error {
	if strings.TrimSpace(s.options.DailyMemoryDir) == "" {
		return nil
	}
	path := filepath.Join(s.options.DailyMemoryDir, day+".md")
	payload, err := renderSessionMemoryDailyStateMarkdown(state)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *sessionMemoryDailyStore) persistLongTermCandidateByDayLocked(
	day string,
	state sessionMemoryDailyState,
	now time.Time,
) error {
	candidateDir := strings.TrimSpace(s.options.DailyLongTermDir)
	if candidateDir == "" {
		return nil
	}
	candidates := append([]sessionMemoryDailyItem{}, state.TierValues[sessionMemoryDailyTierL2]...)
	candidates = append(candidates, state.TierValues[sessionMemoryDailyTierL3]...)
	if len(candidates) == 0 {
		return nil
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].OccurredAt.Equal(candidates[j].OccurredAt) {
			return candidates[i].EntryID < candidates[j].EntryID
		}
		return candidates[i].OccurredAt.Before(candidates[j].OccurredAt)
	})

	candidateState := sessionMemoryLongTermCandidateState{
		Date:       day,
		Generated:  now.UTC(),
		Candidates: candidates,
	}
	payload, err := renderSessionMemoryLongTermCandidateMarkdown(candidateState)
	if err != nil {
		return err
	}
	path := filepath.Join(candidateDir, day+".md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *sessionMemoryDailyStore) pruneExpiredDailyFilesLocked(now time.Time) {
	if strings.TrimSpace(s.options.DailyMemoryDir) == "" {
		return
	}
	entries, err := os.ReadDir(s.options.DailyMemoryDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !sessionMemoryDailyFilePattern.MatchString(entry.Name()) {
			continue
		}
		day := strings.TrimSuffix(entry.Name(), ".md")
		state, err := s.loadStateByDayLocked(day)
		if err != nil {
			continue
		}
		normalized := applySessionMemoryDailyTierConstraints(state, now.UTC(), s.options)
		if totalSessionMemoryDailyItemCount(normalized) == 0 {
			_ = os.Remove(filepath.Join(s.options.DailyMemoryDir, entry.Name()))
			continue
		}
		_ = s.persistStateByDayLocked(day, normalized)
	}
}

func applySessionMemoryDailyTierConstraints(
	state sessionMemoryDailyState,
	now time.Time,
	options SessionMemoryOptions,
) sessionMemoryDailyState {
	for _, tier := range sessionMemoryDailyTierOrder {
		tierOptions := options.dailyTierOptions(tier)
		items := append([]sessionMemoryDailyItem{}, state.TierValues[tier]...)
		filtered := make([]sessionMemoryDailyItem, 0, len(items))
		for _, item := range items {
			item.Tier = tier
			item.Summary = normalizeSnippet(item.Summary, tierOptions.MaxEntryLength)
			item.KeyFacts = limitSessionMemoryFacts(item.KeyFacts, options.CompressionMaxFacts, tierOptions.MaxEntryLength)
			if item.OccurredAt.IsZero() {
				item.OccurredAt = now
			}
			if tierOptions.TTL > 0 && item.OccurredAt.Add(tierOptions.TTL).Before(now) {
				continue
			}
			filtered = append(filtered, item)
		}
		sort.Slice(filtered, func(i, j int) bool {
			if filtered[i].OccurredAt.Equal(filtered[j].OccurredAt) {
				return filtered[i].EntryID < filtered[j].EntryID
			}
			return filtered[i].OccurredAt.Before(filtered[j].OccurredAt)
		})
		if tierOptions.MaxLayerTokens > 0 {
			for sessionMemoryDailyTierTokens(filtered) > tierOptions.MaxLayerTokens && len(filtered) > 0 {
				filtered = filtered[1:]
			}
		}
		state.TierValues[tier] = filtered
	}
	return state
}

func upsertSessionMemoryDailyItem(items []sessionMemoryDailyItem, item sessionMemoryDailyItem) []sessionMemoryDailyItem {
	if strings.TrimSpace(item.EntryID) == "" {
		return items
	}
	for idx := range items {
		if items[idx].EntryID == item.EntryID {
			items[idx] = item
			return items
		}
	}
	return append(items, item)
}

func buildSessionMemoryDailyDigest(
	day string,
	l2 []sessionMemoryDailyItem,
	now time.Time,
	options SessionMemoryOptions,
) sessionMemoryDailyItem {
	if len(l2) == 0 {
		return sessionMemoryDailyItem{}
	}
	maxEntryLength := options.L3.MaxEntryLength
	if maxEntryLength <= 0 {
		maxEntryLength = defaultSessionMemoryDailyL3MaxEntryLength
	}
	summaries := make([]string, 0, len(l2))
	facts := make([]sessionMemoryFact, 0, options.CompressionMaxFacts)
	seenFacts := map[string]struct{}{}
	for _, item := range l2 {
		if item.Summary != "" {
			summaries = append(summaries, item.Summary)
		}
		for _, fact := range item.KeyFacts {
			if len(facts) >= options.CompressionMaxFacts {
				break
			}
			identity := fact.Key + "=" + fact.Value
			if _, exists := seenFacts[identity]; exists {
				continue
			}
			seenFacts[identity] = struct{}{}
			facts = append(facts, fact)
		}
	}
	return sessionMemoryDailyItem{
		EntryID:    "digest-" + day,
		Tier:       sessionMemoryDailyTierL3,
		Source:     "daily_digest",
		SessionID:  "global",
		Summary:    normalizeSnippet(strings.Join(summaries, " | "), maxEntryLength),
		KeyFacts:   limitSessionMemoryFacts(facts, options.CompressionMaxFacts, maxEntryLength),
		OccurredAt: now.UTC(),
	}
}

func limitSessionMemoryFacts(facts []sessionMemoryFact, maxFacts int, maxEntryLength int) []sessionMemoryFact {
	if len(facts) == 0 {
		return nil
	}
	if maxFacts <= 0 {
		maxFacts = len(facts)
	}
	limited := make([]sessionMemoryFact, 0, minInt(maxFacts, len(facts)))
	for _, fact := range facts {
		if len(limited) >= maxFacts {
			break
		}
		key := normalizeFactKey(fact.Key)
		value := normalizeSnippet(fact.Value, maxEntryLength)
		if key == "" || value == "" {
			continue
		}
		limited = append(limited, sessionMemoryFact{
			Key:   key,
			Value: value,
		})
	}
	if len(limited) == 0 {
		return nil
	}
	return limited
}

func sessionMemoryDailyTierTokens(items []sessionMemoryDailyItem) int {
	total := 0
	for _, item := range items {
		total += estimateTokenCount(item.Summary)
		for _, fact := range item.KeyFacts {
			total += estimateTokenCount(fact.Key)
			total += estimateTokenCount(fact.Value)
			total += 2
		}
		total += 4
	}
	return total
}

func totalSessionMemoryDailyItemCount(state sessionMemoryDailyState) int {
	total := 0
	for _, tier := range sessionMemoryDailyTierOrder {
		total += len(state.TierValues[tier])
	}
	return total
}

func renderSessionMemoryDailyStateMarkdown(state sessionMemoryDailyState) ([]byte, error) {
	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return nil, err
	}
	var builder strings.Builder
	builder.WriteString("# Daily Memory ")
	builder.WriteString(state.Date)
	builder.WriteString("\n\n")
	builder.WriteString("GeneratedAt: ")
	builder.WriteString(state.Generated.UTC().Format(time.RFC3339))
	builder.WriteString("\n\n")

	for _, tier := range sessionMemoryDailyTierOrder {
		builder.WriteString("## ")
		builder.WriteString(tier)
		builder.WriteByte('\n')
		items := state.TierValues[tier]
		if len(items) == 0 {
			builder.WriteString("- none\n\n")
			continue
		}
		for _, item := range items {
			builder.WriteString("- [")
			builder.WriteString(item.OccurredAt.UTC().Format(time.RFC3339))
			builder.WriteString("] ")
			builder.WriteString(item.Summary)
			builder.WriteByte('\n')
			if len(item.KeyFacts) > 0 {
				builder.WriteString("  - key_facts: ")
				builder.WriteString(compactSessionMemoryFacts(item.KeyFacts))
				builder.WriteByte('\n')
			}
		}
		builder.WriteByte('\n')
	}

	builder.WriteString("## Structured State\n\n```json\n")
	builder.Write(payload)
	builder.WriteString("\n```\n")
	return []byte(builder.String()), nil
}

func renderSessionMemoryLongTermCandidateMarkdown(state sessionMemoryLongTermCandidateState) ([]byte, error) {
	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return nil, err
	}
	var builder strings.Builder
	builder.WriteString("# Long-Term Memory Candidates ")
	builder.WriteString(state.Date)
	builder.WriteString("\n\n")
	builder.WriteString("GeneratedAt: ")
	builder.WriteString(state.Generated.UTC().Format(time.RFC3339))
	builder.WriteString("\n\n")
	builder.WriteString("## Candidates\n")
	if len(state.Candidates) == 0 {
		builder.WriteString("- none\n")
	} else {
		for _, item := range state.Candidates {
			builder.WriteString("- [")
			builder.WriteString(item.Tier)
			builder.WriteString("] ")
			builder.WriteString(item.Summary)
			builder.WriteByte('\n')
			if len(item.KeyFacts) > 0 {
				builder.WriteString("  - key_facts: ")
				builder.WriteString(compactSessionMemoryFacts(item.KeyFacts))
				builder.WriteByte('\n')
			}
		}
	}
	builder.WriteString("\n## Structured State\n\n```json\n")
	builder.Write(payload)
	builder.WriteString("\n```\n")
	return []byte(builder.String()), nil
}

func parseSessionMemoryDailyState(raw []byte, out *sessionMemoryDailyState) error {
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err == nil {
		return nil
	}
	payload, err := extractMarkdownJSONPayload(string(raw))
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(payload), out)
}

func compactSessionMemoryFacts(facts []sessionMemoryFact) string {
	if len(facts) == 0 {
		return ""
	}
	pairs := make([]string, 0, len(facts))
	for _, fact := range facts {
		if fact.Key == "" || fact.Value == "" {
			continue
		}
		pairs = append(pairs, fact.Key+"="+fact.Value)
	}
	return strings.Join(pairs, "; ")
}

func sanitizeSessionMemoryDailyID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "anonymous"
	}
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, "\\", "-")
	value = strings.ReplaceAll(value, " ", "-")
	return value
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

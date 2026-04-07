package application

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	sessionapp "alter0/internal/session/application"
	sessiondomain "alter0/internal/session/domain"
	shareddomain "alter0/internal/shared/domain"
)

const (
	defaultSessionMemoryMaxTurns                  = 6
	defaultSessionMemoryTTL                       = 20 * time.Minute
	defaultSessionMemorySnippetSize               = 240
	defaultSessionMemoryCompressionTriggerTokens  = 1200
	defaultSessionMemoryCompressionSummaryTokens  = 220
	defaultSessionMemoryCompressionRetainTurns    = 4
	defaultSessionMemoryCompressionMaxFacts       = 8
	defaultSessionMemoryCompressionReferenceLimit = 160
	defaultSessionMemoryDailyL1MaxEntryLength     = 180
	defaultSessionMemoryDailyL2MaxEntryLength     = 220
	defaultSessionMemoryDailyL3MaxEntryLength     = 280
	defaultSessionMemoryDailyL1MaxLayerTokens     = 900
	defaultSessionMemoryDailyL2MaxLayerTokens     = 2200
	defaultSessionMemoryDailyL3MaxLayerTokens     = 4600
	defaultSessionMemoryDailyL1TTL                = 36 * time.Hour
	defaultSessionMemoryDailyL2TTL                = 12 * 24 * time.Hour
	defaultSessionMemoryDailyL3TTL                = 45 * 24 * time.Hour
)

var (
	sessionMemoryFactPattern  = regexp.MustCompile(`([\\p{Han}A-Za-z0-9_./-]{2,48})\\s*[:：=]\\s*([^\\n,;，；]{2,180})`)
	sessionMemoryTokenPattern = regexp.MustCompile(`[\\p{Han}]|[A-Za-z0-9_]+`)
)

type SessionMemoryOptions struct {
	MaxTurns                 int
	TTL                      time.Duration
	MaxSnippets              int
	CompressionTriggerTokens int
	CompressionSummaryTokens int
	CompressionRetainTurns   int
	CompressionMaxFacts      int
	DailyMemoryDir           string
	DailyLongTermDir         string
	L1                       SessionMemoryTierOptions
	L2                       SessionMemoryTierOptions
	L3                       SessionMemoryTierOptions
}

type SessionMemoryTierOptions struct {
	MaxEntryLength int
	MaxLayerTokens int
	TTL            time.Duration
}

type sessionMemoryTurn struct {
	Sequence          int64
	UserMessageID     string
	AssistantReplyRef string
	UserInput         string
	AssistantOutput   string
	Route             shareddomain.Route
	RecordedAt        time.Time
	PlanHint          string
}

type sessionMemoryFact struct {
	Key   string
	Value string
}

type sessionMemoryTurnReference struct {
	TurnSequence      int64
	UserMessageID     string
	AssistantReplyRef string
}

type sessionMemoryFragment struct {
	FragmentID  string
	Summary     string
	KeyFacts    []sessionMemoryFact
	SourceTurns []sessionMemoryTurnReference
	CreatedAt   time.Time
	LastSource  time.Time
}

type sessionMemorySession struct {
	Turns                []sessionMemoryTurn
	Fragments            []sessionMemoryFragment
	NextTurnSequence     int64
	NextFragmentSequence int64
}

type sessionMemorySnapshot struct {
	Fragments         []sessionMemoryFragment
	RecentTurns       []sessionMemoryTurn
	KeyState          map[string]string
	Reference         string
	HistoryRehydrated bool
}

func (s sessionMemorySnapshot) Metadata() map[string]string {
	referenceCount := 0
	for _, fragment := range s.Fragments {
		referenceCount += len(fragment.SourceTurns)
	}

	metadata := map[string]string{
		"memory_turn_count":         strconv.Itoa(len(s.RecentTurns)),
		"memory_fragment_count":     strconv.Itoa(len(s.Fragments)),
		"memory_fragment_ref_count": strconv.Itoa(referenceCount),
		"memory_context_compressed": strconv.FormatBool(len(s.Fragments) > 0),
	}
	if len(s.Reference) > 0 {
		metadata["memory_reference_resolved"] = "true"
	}
	if s.HistoryRehydrated {
		metadata["memory_history_rehydrated"] = "true"
	}
	return metadata
}

func (s sessionMemorySnapshot) ResultMetadata() map[string]string {
	metadata := map[string]string{}
	if len(s.Reference) > 0 {
		metadata["memory_reference_resolved"] = "true"
	}
	if plan, ok := s.KeyState["latest_plan"]; ok && plan != "" {
		metadata["memory_latest_plan"] = plan
	}
	if len(s.Fragments) > 0 {
		metadata["memory_context_compressed"] = "true"
		metadata["memory_fragment_count"] = strconv.Itoa(len(s.Fragments))
	}
	if s.HistoryRehydrated {
		metadata["memory_history_rehydrated"] = "true"
	}
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

type sessionMemoryStore struct {
	mu       sync.Mutex
	options  SessionMemoryOptions
	sessions map[string]*sessionMemorySession
	daily    *sessionMemoryDailyStore
}

func newSessionMemoryStore(options SessionMemoryOptions) *sessionMemoryStore {
	normalized := normalizeSessionMemoryOptions(options)
	store := &sessionMemoryStore{
		options:  normalized,
		sessions: map[string]*sessionMemorySession{},
	}
	store.daily = newSessionMemoryDailyStore(normalized)
	return store
}

func normalizeSessionMemoryOptions(options SessionMemoryOptions) SessionMemoryOptions {
	if options.MaxTurns <= 0 {
		options.MaxTurns = defaultSessionMemoryMaxTurns
	}
	if options.TTL <= 0 {
		options.TTL = defaultSessionMemoryTTL
	}
	if options.MaxSnippets <= 0 {
		options.MaxSnippets = defaultSessionMemorySnippetSize
	}
	if options.CompressionTriggerTokens <= 0 {
		options.CompressionTriggerTokens = defaultSessionMemoryCompressionTriggerTokens
	}
	if options.CompressionSummaryTokens <= 0 {
		options.CompressionSummaryTokens = defaultSessionMemoryCompressionSummaryTokens
	}
	if options.CompressionRetainTurns <= 0 {
		options.CompressionRetainTurns = defaultSessionMemoryCompressionRetainTurns
	}
	if options.MaxTurns > 1 && options.CompressionRetainTurns >= options.MaxTurns {
		options.CompressionRetainTurns = options.MaxTurns - 1
	}
	if options.CompressionRetainTurns <= 0 {
		options.CompressionRetainTurns = 1
	}
	if options.CompressionMaxFacts <= 0 {
		options.CompressionMaxFacts = defaultSessionMemoryCompressionMaxFacts
	}
	options.DailyMemoryDir = strings.TrimSpace(options.DailyMemoryDir)
	options.DailyLongTermDir = strings.TrimSpace(options.DailyLongTermDir)
	if options.DailyMemoryDir != "" {
		options.DailyMemoryDir = filepath.Clean(options.DailyMemoryDir)
		options.L1 = normalizeSessionMemoryTierOptions(options.L1, SessionMemoryTierOptions{
			MaxEntryLength: defaultSessionMemoryDailyL1MaxEntryLength,
			MaxLayerTokens: defaultSessionMemoryDailyL1MaxLayerTokens,
			TTL:            defaultSessionMemoryDailyL1TTL,
		})
		options.L2 = normalizeSessionMemoryTierOptions(options.L2, SessionMemoryTierOptions{
			MaxEntryLength: defaultSessionMemoryDailyL2MaxEntryLength,
			MaxLayerTokens: defaultSessionMemoryDailyL2MaxLayerTokens,
			TTL:            defaultSessionMemoryDailyL2TTL,
		})
		options.L3 = normalizeSessionMemoryTierOptions(options.L3, SessionMemoryTierOptions{
			MaxEntryLength: defaultSessionMemoryDailyL3MaxEntryLength,
			MaxLayerTokens: defaultSessionMemoryDailyL3MaxLayerTokens,
			TTL:            defaultSessionMemoryDailyL3TTL,
		})
		if options.DailyLongTermDir == "" {
			options.DailyLongTermDir = filepath.Join(options.DailyMemoryDir, "long-term")
		}
		options.DailyLongTermDir = filepath.Clean(options.DailyLongTermDir)
	}
	return options
}

func normalizeSessionMemoryTierOptions(options SessionMemoryTierOptions, defaults SessionMemoryTierOptions) SessionMemoryTierOptions {
	if options.MaxEntryLength <= 0 {
		options.MaxEntryLength = defaults.MaxEntryLength
	}
	if options.MaxLayerTokens <= 0 {
		options.MaxLayerTokens = defaults.MaxLayerTokens
	}
	if options.TTL <= 0 {
		options.TTL = defaults.TTL
	}
	return options
}

func (o SessionMemoryOptions) dailyTierOptions(tier string) SessionMemoryTierOptions {
	switch tier {
	case sessionMemoryDailyTierL1:
		return o.L1
	case sessionMemoryDailyTierL2:
		return o.L2
	default:
		return o.L3
	}
}

func (s *sessionMemoryStore) Snapshot(sessionID string, input string, now time.Time) sessionMemorySnapshot {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return sessionMemorySnapshot{
			KeyState: map[string]string{},
		}
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	session := s.pruneAndGetSessionLocked(sessionID, now)
	if session == nil {
		return sessionMemorySnapshot{
			KeyState: map[string]string{},
		}
	}

	snapshot := sessionMemorySnapshot{
		Fragments:   copyFragments(session.Fragments),
		RecentTurns: copyTurns(session.Turns),
		KeyState:    buildKeyState(session.Turns, session.Fragments),
	}
	snapshot.Reference = resolveReferenceHint(strings.TrimSpace(input), snapshot)
	return snapshot
}

func resolvedSessionMemoryMaxTurns(memory sessionMemory) int {
	if store, ok := memory.(*sessionMemoryStore); ok && store != nil && store.options.MaxTurns > 0 {
		return store.options.MaxTurns
	}
	return defaultSessionMemoryMaxTurns
}

func hydrateSessionMemoryFromHistory(
	snapshot sessionMemorySnapshot,
	history sessionHistoryMemory,
	msg shareddomain.UnifiedMessage,
	maxTurns int,
) sessionMemorySnapshot {
	if history == nil || strings.TrimSpace(msg.SessionID) == "" {
		return snapshot
	}
	if maxTurns <= 0 {
		maxTurns = defaultSessionMemoryMaxTurns
	}
	if len(snapshot.RecentTurns) >= maxTurns {
		return snapshot
	}

	records := latestSessionHistoryMessages(history, strings.TrimSpace(msg.SessionID), maxTurns)
	if len(records) == 0 {
		return snapshot
	}
	historyTurns := sessionHistoryRecordsToTurns(records, strings.TrimSpace(msg.MessageID), maxTurns)
	if len(historyTurns) == 0 {
		return snapshot
	}

	existing := map[string]struct{}{}
	for _, turn := range snapshot.RecentTurns {
		if id := strings.TrimSpace(turn.UserMessageID); id != "" {
			existing[id] = struct{}{}
		}
	}
	missing := make([]sessionMemoryTurn, 0, len(historyTurns))
	for _, turn := range historyTurns {
		if id := strings.TrimSpace(turn.UserMessageID); id != "" {
			if _, ok := existing[id]; ok {
				continue
			}
		}
		missing = append(missing, turn)
	}
	if len(missing) == 0 {
		return snapshot
	}

	needed := maxTurns - len(snapshot.RecentTurns)
	if len(missing) > needed {
		missing = missing[len(missing)-needed:]
	}
	merged := make([]sessionMemoryTurn, 0, len(missing)+len(snapshot.RecentTurns))
	merged = append(merged, missing...)
	merged = append(merged, snapshot.RecentTurns...)
	if len(merged) > maxTurns {
		merged = merged[len(merged)-maxTurns:]
	}
	snapshot.RecentTurns = merged
	snapshot.KeyState = buildKeyState(snapshot.RecentTurns, snapshot.Fragments)
	snapshot.Reference = resolveReferenceHint(strings.TrimSpace(msg.Content), snapshot)
	snapshot.HistoryRehydrated = true
	return snapshot
}

func latestSessionHistoryMessages(history sessionHistoryMemory, sessionID string, maxTurns int) []sessiondomain.MessageRecord {
	if history == nil || strings.TrimSpace(sessionID) == "" {
		return nil
	}
	pageSize := maxTurns * 6
	if pageSize < 24 {
		pageSize = 24
	}
	if pageSize > 200 {
		pageSize = 200
	}
	page := history.ListMessages(sessionapp.MessageQuery{
		SessionID: sessionID,
		Page:      1,
		PageSize:  pageSize,
	})
	if page.Pagination.Total > pageSize && page.Pagination.PageSize > 0 {
		lastPage := (page.Pagination.Total + page.Pagination.PageSize - 1) / page.Pagination.PageSize
		if lastPage > 1 {
			page = history.ListMessages(sessionapp.MessageQuery{
				SessionID: sessionID,
				Page:      lastPage,
				PageSize:  pageSize,
			})
		}
	}
	return page.Items
}

func sessionHistoryRecordsToTurns(records []sessiondomain.MessageRecord, currentMessageID string, maxTurns int) []sessionMemoryTurn {
	if len(records) == 0 {
		return nil
	}
	if maxTurns <= 0 {
		maxTurns = defaultSessionMemoryMaxTurns
	}
	currentMessageID = strings.TrimSpace(currentMessageID)
	turns := make([]sessionMemoryTurn, 0, maxTurns)
	var pending *sessiondomain.MessageRecord
	sequence := int64(0)
	for _, record := range records {
		switch record.Role {
		case sessiondomain.MessageRoleUser:
			if currentMessageID != "" && strings.TrimSpace(record.MessageID) == currentMessageID {
				pending = nil
				continue
			}
			item := record
			pending = &item
		case sessiondomain.MessageRoleAssistant:
			if pending == nil {
				continue
			}
			if strings.TrimSpace(record.Content) == "" {
				pending = nil
				continue
			}
			sequence++
			route := record.RouteResult.Route
			if route == "" {
				route = pending.RouteResult.Route
			}
			turn := sessionMemoryTurn{
				Sequence:          sequence,
				UserMessageID:     strings.TrimSpace(pending.MessageID),
				AssistantReplyRef: strings.TrimSpace(record.MessageID),
				UserInput:         normalizeSnippet(pending.Content, defaultSessionMemorySnippetSize),
				AssistantOutput:   normalizeSnippet(record.Content, defaultSessionMemorySnippetSize),
				Route:             route,
				RecordedAt:        pending.Timestamp,
			}
			if turn.AssistantReplyRef == "" {
				turn.AssistantReplyRef = buildAssistantReplyReference(turn.UserMessageID, sequence)
			}
			turn.PlanHint = extractPlanHint(turn.AssistantOutput, turn.UserInput, defaultSessionMemorySnippetSize)
			turns = append(turns, turn)
			pending = nil
		default:
			continue
		}
	}
	if len(turns) > maxTurns {
		turns = turns[len(turns)-maxTurns:]
	}
	return turns
}

func (s *sessionMemoryStore) Record(msg shareddomain.UnifiedMessage, route shareddomain.Route, output string) {
	sessionID := strings.TrimSpace(msg.SessionID)
	if sessionID == "" {
		return
	}
	now := msg.ReceivedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}

	turn := sessionMemoryTurn{
		UserMessageID:   strings.TrimSpace(msg.MessageID),
		UserInput:       normalizeSnippet(msg.Content, s.options.MaxSnippets),
		AssistantOutput: normalizeSnippet(output, s.options.MaxSnippets),
		Route:           route,
		RecordedAt:      now,
	}
	turn.PlanHint = extractPlanHint(turn.AssistantOutput, turn.UserInput, s.options.MaxSnippets)

	s.mu.Lock()

	session := s.pruneAndGetSessionLocked(sessionID, now)
	if session == nil {
		session = &sessionMemorySession{}
		s.sessions[sessionID] = session
	}

	session.NextTurnSequence++
	turn.Sequence = session.NextTurnSequence
	if turn.UserMessageID == "" {
		turn.UserMessageID = fmt.Sprintf("%s-user-%d", sessionID, turn.Sequence)
	}
	turn.AssistantReplyRef = buildAssistantReplyReference(turn.UserMessageID, turn.Sequence)

	session.Turns = append(session.Turns, turn)
	s.compressSessionLocked(session, now)
	if len(session.Turns) > s.options.MaxTurns {
		session.Turns = copyTurns(session.Turns[len(session.Turns)-s.options.MaxTurns:])
	}
	if len(session.Turns) == 0 && len(session.Fragments) == 0 {
		delete(s.sessions, sessionID)
	}
	fragments := copyFragments(session.Fragments)
	s.mu.Unlock()

	if s.daily != nil {
		s.daily.Record(sessionID, turn, fragments)
	}
}

func (s *sessionMemoryStore) pruneAndGetSessionLocked(sessionID string, now time.Time) *sessionMemorySession {
	session, ok := s.sessions[sessionID]
	if !ok || session == nil {
		delete(s.sessions, sessionID)
		return nil
	}

	cutoff := time.Time{}
	if s.options.TTL > 0 {
		cutoff = now.Add(-s.options.TTL)
	}

	if len(session.Turns) > 0 {
		filteredTurns := session.Turns[:0]
		for _, turn := range session.Turns {
			if cutoff.IsZero() || !turn.RecordedAt.Before(cutoff) {
				filteredTurns = append(filteredTurns, turn)
			}
		}
		session.Turns = copyTurns(filteredTurns)
	}

	if len(session.Fragments) > 0 {
		filteredFragments := session.Fragments[:0]
		for _, fragment := range session.Fragments {
			marker := fragment.LastSource
			if marker.IsZero() {
				marker = fragment.CreatedAt
			}
			if cutoff.IsZero() || !marker.Before(cutoff) {
				filteredFragments = append(filteredFragments, fragment)
			}
		}
		session.Fragments = copyFragments(filteredFragments)
	}

	if len(session.Turns) > s.options.MaxTurns {
		session.Turns = copyTurns(session.Turns[len(session.Turns)-s.options.MaxTurns:])
	}
	if len(session.Turns) == 0 && len(session.Fragments) == 0 {
		delete(s.sessions, sessionID)
		return nil
	}
	return session
}

func (s *sessionMemoryStore) compressSessionLocked(session *sessionMemorySession, now time.Time) {
	if session == nil || len(session.Turns) == 0 {
		return
	}
	if s.options.CompressionTriggerTokens <= 0 {
		return
	}

	for {
		totalTokens := estimateSessionContextTokens(session.Turns, session.Fragments)
		if totalTokens <= s.options.CompressionTriggerTokens {
			return
		}

		compressible := len(session.Turns) - s.options.CompressionRetainTurns
		if compressible <= 0 {
			if len(session.Fragments) > 1 {
				s.mergeFragmentsLocked(session, now)
				continue
			}
			return
		}

		turnsToCompress := copyTurns(session.Turns[:compressible])
		fragment := s.buildFragmentLocked(session, turnsToCompress, now)
		session.Fragments = append(session.Fragments, fragment)
		if len(session.Fragments) > 1 {
			s.mergeFragmentsLocked(session, now)
		}
		session.Turns = copyTurns(session.Turns[compressible:])
	}
}

func (s *sessionMemoryStore) buildFragmentLocked(
	session *sessionMemorySession,
	turns []sessionMemoryTurn,
	now time.Time,
) sessionMemoryFragment {
	session.NextFragmentSequence++
	fragmentID := fmt.Sprintf("fragment-%d", session.NextFragmentSequence)
	if now.IsZero() {
		now = time.Now().UTC()
	}

	summary := buildCompressionSummary(turns, s.options.CompressionSummaryTokens, s.options.MaxSnippets)
	facts := extractKeyFactsFromTurns(turns, s.options.CompressionMaxFacts, s.options.MaxSnippets)
	sourceRefs := buildTurnReferences(turns)
	lastSource := now
	if len(turns) > 0 {
		lastSource = turns[len(turns)-1].RecordedAt
	}

	return sessionMemoryFragment{
		FragmentID:  fragmentID,
		Summary:     summary,
		KeyFacts:    facts,
		SourceTurns: sourceRefs,
		CreatedAt:   now,
		LastSource:  lastSource,
	}
}

func (s *sessionMemoryStore) mergeFragmentsLocked(session *sessionMemorySession, now time.Time) {
	if session == nil || len(session.Fragments) <= 1 {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	summaries := make([]string, 0, len(session.Fragments))
	facts := make([]sessionMemoryFact, 0, s.options.CompressionMaxFacts)
	factSeen := map[string]struct{}{}
	references := make([]sessionMemoryTurnReference, 0)
	lastSource := time.Time{}

	appendFact := func(fact sessionMemoryFact) {
		if len(facts) >= s.options.CompressionMaxFacts {
			return
		}
		if fact.Key == "" || fact.Value == "" {
			return
		}
		identity := fact.Key + "=" + fact.Value
		if _, exists := factSeen[identity]; exists {
			return
		}
		factSeen[identity] = struct{}{}
		facts = append(facts, fact)
	}

	for _, fragment := range session.Fragments {
		if fragment.Summary != "" {
			summaries = append(summaries, fragment.Summary)
		}
		for _, fact := range fragment.KeyFacts {
			appendFact(fact)
		}
		references = append(references, fragment.SourceTurns...)
		if fragment.LastSource.After(lastSource) {
			lastSource = fragment.LastSource
		}
	}

	session.NextFragmentSequence++
	merged := sessionMemoryFragment{
		FragmentID:  fmt.Sprintf("fragment-%d", session.NextFragmentSequence),
		Summary:     truncateToTokenCount(strings.Join(summaries, " | "), s.options.CompressionSummaryTokens),
		KeyFacts:    facts,
		SourceTurns: references,
		CreatedAt:   now,
		LastSource:  lastSource,
	}
	session.Fragments = []sessionMemoryFragment{merged}
}

func buildKeyState(turns []sessionMemoryTurn, fragments []sessionMemoryFragment) map[string]string {
	state := map[string]string{}
	if len(turns) > 0 {
		last := turns[len(turns)-1]
		state["last_user_message"] = last.UserInput
		state["last_assistant_reply"] = last.AssistantOutput
		state["last_route"] = string(last.Route)

		for i := len(turns) - 1; i >= 0; i-- {
			if turns[i].PlanHint != "" {
				state["latest_plan"] = turns[i].PlanHint
				break
			}
		}
	}

	if _, ok := state["latest_plan"]; ok {
		return state
	}

	for i := len(fragments) - 1; i >= 0; i-- {
		plan := latestPlanFromFragment(fragments[i])
		if plan == "" {
			continue
		}
		state["latest_plan"] = plan
		break
	}
	return state
}

func latestPlanFromFragment(fragment sessionMemoryFragment) string {
	for i := len(fragment.KeyFacts) - 1; i >= 0; i-- {
		key := normalizeMandatoryContextKey(fragment.KeyFacts[i].Key)
		if strings.Contains(key, "plan") || strings.Contains(fragment.KeyFacts[i].Key, "方案") {
			return fragment.KeyFacts[i].Value
		}
	}
	return strings.TrimSpace(fragment.Summary)
}

func resolveReferenceHint(input string, snapshot sessionMemorySnapshot) string {
	if input == "" || !containsReferenceToken(input) {
		return ""
	}
	if plan := strings.TrimSpace(snapshot.KeyState["latest_plan"]); plan != "" {
		return plan
	}
	if len(snapshot.RecentTurns) > 0 {
		last := snapshot.RecentTurns[len(snapshot.RecentTurns)-1]
		if last.AssistantOutput != "" {
			return last.AssistantOutput
		}
	}
	if len(snapshot.Fragments) > 0 {
		return strings.TrimSpace(snapshot.Fragments[len(snapshot.Fragments)-1].Summary)
	}
	return ""
}

func containsReferenceToken(content string) bool {
	chineseTokens := []string{
		"这个",
		"那个",
		"刚才",
		"上一个",
		"上一条",
		"前面",
		"上述",
		"这套",
		"那套",
	}
	for _, token := range chineseTokens {
		if strings.Contains(content, token) {
			return true
		}
	}

	lower := strings.ToLower(content)
	englishTokens := []string{
		"this plan",
		"that plan",
		"the previous plan",
		"the earlier plan",
		"this one",
		"that one",
		"the one above",
	}
	for _, token := range englishTokens {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func buildSessionMemoryPrompt(input string, snapshot sessionMemorySnapshot) string {
	trimmed := strings.TrimSpace(input)
	if len(snapshot.RecentTurns) == 0 && len(snapshot.Fragments) == 0 {
		return trimmed
	}

	var builder strings.Builder
	builder.WriteString("[SESSION SHORT TERM MEMORY]\n")
	builder.WriteString("Scope: current session only.\n")

	if len(snapshot.KeyState) > 0 {
		builder.WriteString("Key state:\n")
		keys := make([]string, 0, len(snapshot.KeyState))
		for key := range snapshot.KeyState {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			value := strings.TrimSpace(snapshot.KeyState[key])
			if value == "" {
				continue
			}
			builder.WriteString("- ")
			builder.WriteString(key)
			builder.WriteString(": ")
			builder.WriteString(value)
			builder.WriteByte('\n')
		}
	}

	if len(snapshot.Fragments) > 0 {
		builder.WriteString("Compressed memory fragments:\n")
		for _, fragment := range snapshot.Fragments {
			builder.WriteString("- fragment_id: ")
			builder.WriteString(fragment.FragmentID)
			builder.WriteByte('\n')
			if fragment.Summary != "" {
				builder.WriteString("  summary: ")
				builder.WriteString(fragment.Summary)
				builder.WriteByte('\n')
			}
			if len(fragment.KeyFacts) > 0 {
				builder.WriteString("  key_facts:\n")
				for _, fact := range fragment.KeyFacts {
					if fact.Key == "" || fact.Value == "" {
						continue
					}
					builder.WriteString("    - ")
					builder.WriteString(fact.Key)
					builder.WriteString(": ")
					builder.WriteString(fact.Value)
					builder.WriteByte('\n')
				}
			}
			if len(fragment.SourceTurns) > 0 {
				builder.WriteString("  source_turn_refs: ")
				builder.WriteString(compactSourceTurnReferences(fragment.SourceTurns))
				builder.WriteByte('\n')
			}
		}
	}

	builder.WriteString("Recent turns:\n")
	for idx, turn := range snapshot.RecentTurns {
		builder.WriteString(fmt.Sprintf("%d) user: %s\n", idx+1, turn.UserInput))
		builder.WriteString(fmt.Sprintf("   assistant: %s\n", turn.AssistantOutput))
	}

	if snapshot.Reference != "" {
		builder.WriteString("Reference resolution:\n")
		builder.WriteString("- Current utterance likely refers to: ")
		builder.WriteString(snapshot.Reference)
		builder.WriteByte('\n')
	}

	builder.WriteString("Current user input:\n")
	builder.WriteString(trimmed)
	return builder.String()
}

func compactSourceTurnReferences(references []sessionMemoryTurnReference) string {
	if len(references) == 0 {
		return ""
	}
	first := references[0]
	last := references[len(references)-1]
	if len(references) == 1 {
		return fmt.Sprintf("turn#%d user=%s assistant=%s", first.TurnSequence, first.UserMessageID, first.AssistantReplyRef)
	}
	summary := fmt.Sprintf(
		"count=%d turn#%d..turn#%d first_user=%s last_user=%s",
		len(references),
		first.TurnSequence,
		last.TurnSequence,
		first.UserMessageID,
		last.UserMessageID,
	)
	return normalizeSnippet(summary, defaultSessionMemoryCompressionReferenceLimit)
}

func copyTurns(turns []sessionMemoryTurn) []sessionMemoryTurn {
	if len(turns) == 0 {
		return nil
	}
	copied := make([]sessionMemoryTurn, len(turns))
	copy(copied, turns)
	return copied
}

func copyFragments(fragments []sessionMemoryFragment) []sessionMemoryFragment {
	if len(fragments) == 0 {
		return nil
	}
	copied := make([]sessionMemoryFragment, 0, len(fragments))
	for _, fragment := range fragments {
		copied = append(copied, sessionMemoryFragment{
			FragmentID:  fragment.FragmentID,
			Summary:     fragment.Summary,
			KeyFacts:    copyFacts(fragment.KeyFacts),
			SourceTurns: copyTurnReferences(fragment.SourceTurns),
			CreatedAt:   fragment.CreatedAt,
			LastSource:  fragment.LastSource,
		})
	}
	return copied
}

func copyFacts(facts []sessionMemoryFact) []sessionMemoryFact {
	if len(facts) == 0 {
		return nil
	}
	copied := make([]sessionMemoryFact, len(facts))
	copy(copied, facts)
	return copied
}

func copyTurnReferences(references []sessionMemoryTurnReference) []sessionMemoryTurnReference {
	if len(references) == 0 {
		return nil
	}
	copied := make([]sessionMemoryTurnReference, len(references))
	copy(copied, references)
	return copied
}

func mergeStringMap(base map[string]string, overlay map[string]string) map[string]string {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}
	merged := map[string]string{}
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range overlay {
		merged[key] = value
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}

func extractPlanHint(output string, fallback string, maxSnippets int) string {
	keywords := []string{
		"plan",
		"approach",
		"proposal",
		"option",
		"steps",
		"timeline",
		"方案",
		"计划",
		"步骤",
		"建议",
	}
	candidate := firstLineWithKeyword(output, keywords)
	if candidate != "" {
		return normalizeSnippet(candidate, maxSnippets)
	}
	candidate = firstLineWithKeyword(fallback, keywords)
	if candidate != "" {
		return normalizeSnippet(candidate, maxSnippets)
	}
	return ""
}

func firstLineWithKeyword(content string, keywords []string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	lines := strings.Split(content, "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		lineLower := strings.ToLower(line)
		for _, keyword := range keywords {
			if strings.Contains(lineLower, keyword) || strings.Contains(line, keyword) {
				return line
			}
		}
	}
	return ""
}

func normalizeSnippet(content string, maxSnippets int) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ""
	}
	normalized := strings.Join(strings.Fields(trimmed), " ")
	if maxSnippets <= 0 {
		return normalized
	}
	runes := []rune(normalized)
	if len(runes) <= maxSnippets {
		return normalized
	}
	return string(runes[:maxSnippets]) + "..."
}

func buildAssistantReplyReference(userMessageID string, sequence int64) string {
	base := strings.TrimSpace(userMessageID)
	if base == "" {
		base = fmt.Sprintf("turn-%d", sequence)
	}
	return base + "#assistant"
}

func buildTurnReferences(turns []sessionMemoryTurn) []sessionMemoryTurnReference {
	if len(turns) == 0 {
		return nil
	}
	references := make([]sessionMemoryTurnReference, 0, len(turns))
	for _, turn := range turns {
		references = append(references, sessionMemoryTurnReference{
			TurnSequence:      turn.Sequence,
			UserMessageID:     turn.UserMessageID,
			AssistantReplyRef: turn.AssistantReplyRef,
		})
	}
	return references
}

func buildCompressionSummary(turns []sessionMemoryTurn, summaryTokenLimit int, maxSnippets int) string {
	if len(turns) == 0 {
		return ""
	}
	segments := make([]string, 0, len(turns))
	for _, turn := range turns {
		assistant := turn.PlanHint
		if assistant == "" {
			assistant = turn.AssistantOutput
		}
		segment := fmt.Sprintf(
			"turn#%d user=%s assistant=%s",
			turn.Sequence,
			normalizeSnippet(turn.UserInput, maxSnippets/2),
			normalizeSnippet(assistant, maxSnippets/2),
		)
		candidate := strings.Join(append(segments, segment), " | ")
		if summaryTokenLimit > 0 && estimateTokenCount(candidate) > summaryTokenLimit {
			if len(segments) == 0 {
				segments = append(segments, truncateToTokenCount(segment, summaryTokenLimit))
			}
			break
		}
		segments = append(segments, segment)
	}
	if len(segments) == 0 {
		segments = append(segments, normalizeSnippet(turns[len(turns)-1].AssistantOutput, maxSnippets))
	}
	summary := strings.Join(segments, " | ")
	return truncateToTokenCount(summary, summaryTokenLimit)
}

func extractKeyFactsFromTurns(turns []sessionMemoryTurn, maxFacts int, maxSnippet int) []sessionMemoryFact {
	if maxFacts <= 0 || len(turns) == 0 {
		return nil
	}
	factSnippetLimit := maxSnippet / 3
	if factSnippetLimit < 48 {
		factSnippetLimit = 48
	}
	facts := make([]sessionMemoryFact, 0, maxFacts)
	seen := map[string]struct{}{}
	appendFact := func(fact sessionMemoryFact) {
		if fact.Key == "" || fact.Value == "" {
			return
		}
		identity := fact.Key + "=" + fact.Value
		if _, ok := seen[identity]; ok {
			return
		}
		seen[identity] = struct{}{}
		facts = append(facts, fact)
	}

	for _, turn := range turns {
		for _, fact := range extractFactsFromText(turn.UserInput, factSnippetLimit) {
			appendFact(fact)
			if len(facts) >= maxFacts {
				return facts
			}
		}
		for _, fact := range extractFactsFromText(turn.AssistantOutput, factSnippetLimit) {
			appendFact(fact)
			if len(facts) >= maxFacts {
				return facts
			}
		}
		if turn.PlanHint != "" {
			appendFact(sessionMemoryFact{
				Key:   "plan_hint",
				Value: normalizeSnippet(turn.PlanHint, factSnippetLimit),
			})
			if len(facts) >= maxFacts {
				return facts
			}
		}
	}

	if len(facts) == 0 {
		last := turns[len(turns)-1]
		facts = append(facts, sessionMemoryFact{
			Key:   "conversation_focus",
			Value: normalizeSnippet(last.UserInput, factSnippetLimit),
		})
	}
	return facts
}

func extractFactsFromText(content string, maxSnippet int) []sessionMemoryFact {
	matches := sessionMemoryFactPattern.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}
	facts := make([]sessionMemoryFact, 0, len(matches))
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		key := normalizeFactKey(match[1])
		value := compactFactValue(match[2], maxSnippet)
		if key == "" || value == "" {
			continue
		}
		facts = append(facts, sessionMemoryFact{Key: key, Value: value})
	}
	return facts
}

func compactFactValue(value string, maxSnippet int) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	separators := []string{
		" and ",
		" with ",
		" before ",
		" after ",
		" please ",
	}
	for _, separator := range separators {
		idx := strings.Index(lower, separator)
		if idx <= 0 {
			continue
		}
		trimmed = strings.TrimSpace(trimmed[:idx])
		break
	}
	return normalizeSnippet(trimmed, maxSnippet)
}

func normalizeFactKey(key string) string {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return ""
	}
	trimmed = strings.Trim(trimmed, "-_:;,.，；：")
	trimmed = strings.ReplaceAll(trimmed, " ", "_")
	trimmed = strings.ToLower(trimmed)
	if len([]rune(trimmed)) > 48 {
		trimmed = string([]rune(trimmed)[:48])
	}
	return trimmed
}

func estimateSessionContextTokens(turns []sessionMemoryTurn, fragments []sessionMemoryFragment) int {
	total := 0
	for _, turn := range turns {
		total += estimateTokenCount(turn.UserInput)
		total += estimateTokenCount(turn.AssistantOutput)
		total += 8
	}
	for _, fragment := range fragments {
		total += estimateTokenCount(fragment.Summary)
		for _, fact := range fragment.KeyFacts {
			total += estimateTokenCount(fact.Key)
			total += estimateTokenCount(fact.Value)
			total += 2
		}
		total += 6
	}
	return total
}

func estimateTokenCount(content string) int {
	if strings.TrimSpace(content) == "" {
		return 0
	}
	tokens := sessionMemoryTokenPattern.FindAllString(content, -1)
	return len(tokens)
}

func truncateToTokenCount(content string, maxTokens int) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" || maxTokens <= 0 {
		return trimmed
	}
	indexes := sessionMemoryTokenPattern.FindAllStringIndex(trimmed, -1)
	if len(indexes) <= maxTokens {
		return trimmed
	}
	cutoff := indexes[maxTokens-1][1]
	if cutoff <= 0 || cutoff > len(trimmed) {
		return trimmed
	}
	truncated := strings.TrimSpace(trimmed[:cutoff])
	if truncated == "" {
		return ""
	}
	return truncated + "..."
}

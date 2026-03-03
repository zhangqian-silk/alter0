package application

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	shareddomain "alter0/internal/shared/domain"
)

const (
	defaultSessionMemoryMaxTurns    = 6
	defaultSessionMemoryTTL         = 20 * time.Minute
	defaultSessionMemorySnippetSize = 240
)

type SessionMemoryOptions struct {
	MaxTurns    int
	TTL         time.Duration
	MaxSnippets int
}

type sessionMemoryTurn struct {
	UserInput       string
	AssistantOutput string
	Route           shareddomain.Route
	RecordedAt      time.Time
	PlanHint        string
}

type sessionMemorySnapshot struct {
	RecentTurns []sessionMemoryTurn
	KeyState    map[string]string
	Reference   string
}

func (s sessionMemorySnapshot) Metadata() map[string]string {
	metadata := map[string]string{
		"memory_turn_count": strconv.Itoa(len(s.RecentTurns)),
	}
	if len(s.Reference) > 0 {
		metadata["memory_reference_resolved"] = "true"
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
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

type sessionMemoryStore struct {
	mu       sync.Mutex
	options  SessionMemoryOptions
	sessions map[string][]sessionMemoryTurn
}

func newSessionMemoryStore(options SessionMemoryOptions) *sessionMemoryStore {
	normalized := normalizeSessionMemoryOptions(options)
	return &sessionMemoryStore{
		options:  normalized,
		sessions: map[string][]sessionMemoryTurn{},
	}
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
	return options
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

	turns := s.pruneAndGetTurnsLocked(sessionID, now)
	snapshot := sessionMemorySnapshot{
		RecentTurns: copyTurns(turns),
		KeyState:    buildKeyState(turns),
	}
	snapshot.Reference = resolveReferenceHint(strings.TrimSpace(input), snapshot)
	return snapshot
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
		UserInput:       normalizeSnippet(msg.Content, s.options.MaxSnippets),
		AssistantOutput: normalizeSnippet(output, s.options.MaxSnippets),
		Route:           route,
		RecordedAt:      now,
	}
	turn.PlanHint = extractPlanHint(turn.AssistantOutput, turn.UserInput, s.options.MaxSnippets)

	s.mu.Lock()
	defer s.mu.Unlock()

	turns := s.pruneAndGetTurnsLocked(sessionID, now)
	turns = append(turns, turn)
	if len(turns) > s.options.MaxTurns {
		turns = turns[len(turns)-s.options.MaxTurns:]
	}
	s.sessions[sessionID] = turns
}

func (s *sessionMemoryStore) pruneAndGetTurnsLocked(sessionID string, now time.Time) []sessionMemoryTurn {
	turns, ok := s.sessions[sessionID]
	if !ok || len(turns) == 0 {
		delete(s.sessions, sessionID)
		return nil
	}

	cutoff := time.Time{}
	if s.options.TTL > 0 {
		cutoff = now.Add(-s.options.TTL)
	}
	filtered := turns[:0]
	for _, turn := range turns {
		if cutoff.IsZero() || !turn.RecordedAt.Before(cutoff) {
			filtered = append(filtered, turn)
		}
	}
	if len(filtered) > s.options.MaxTurns {
		filtered = filtered[len(filtered)-s.options.MaxTurns:]
	}
	if len(filtered) == 0 {
		delete(s.sessions, sessionID)
		return nil
	}
	persisted := make([]sessionMemoryTurn, len(filtered))
	copy(persisted, filtered)
	s.sessions[sessionID] = persisted
	return persisted
}

func buildKeyState(turns []sessionMemoryTurn) map[string]string {
	state := map[string]string{}
	if len(turns) == 0 {
		return state
	}

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
	return state
}

func resolveReferenceHint(input string, snapshot sessionMemorySnapshot) string {
	if input == "" || len(snapshot.RecentTurns) == 0 || !containsReferenceToken(input) {
		return ""
	}
	if plan := strings.TrimSpace(snapshot.KeyState["latest_plan"]); plan != "" {
		return plan
	}
	last := snapshot.RecentTurns[len(snapshot.RecentTurns)-1]
	return last.AssistantOutput
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
	if len(snapshot.RecentTurns) == 0 {
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

func copyTurns(turns []sessionMemoryTurn) []sessionMemoryTurn {
	if len(turns) == 0 {
		return nil
	}
	copied := make([]sessionMemoryTurn, len(turns))
	copy(copied, turns)
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

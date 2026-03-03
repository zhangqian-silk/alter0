package application

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	defaultMandatoryContextFilePath = "SOUL.md"
	mandatoryContextVersionSize     = 12

	mandatoryContextInjectedMetadataKey       = "mandatory_context.injected"
	mandatoryContextFileMetadataKey           = "mandatory_context.file"
	mandatoryContextVersionMetadataKey        = "mandatory_context.version"
	mandatoryContextUpdatedAtMetadataKey      = "mandatory_context.updated_at"
	mandatoryContextLoadedAtMetadataKey       = "mandatory_context.loaded_at"
	mandatoryContextRuleCountMetadataKey      = "mandatory_context.rule_count"
	mandatoryContextIsolatedMetadataKey       = "mandatory_context.isolated"
	mandatoryContextConflictCountMetadataKey  = "mandatory_context.conflict_count"
	mandatoryContextConflictKeysMetadataKey   = "mandatory_context.conflict_keys"
	mandatoryContextConflictDetailMetadataKey = "mandatory_context.conflicts"
)

type MandatoryContextOptions struct {
	FilePath string
}

type mandatoryContext interface {
	Snapshot(now time.Time) mandatoryContextSnapshot
}

type mandatoryContextSnapshot struct {
	FilePath  string
	Content   string
	Version   string
	UpdatedAt time.Time
	LoadedAt  time.Time
	Rules     map[string]string
}

func (s mandatoryContextSnapshot) Active() bool {
	return strings.TrimSpace(s.Content) != ""
}

func (s mandatoryContextSnapshot) Metadata() map[string]string {
	if !s.Active() {
		return nil
	}

	metadata := map[string]string{
		mandatoryContextInjectedMetadataKey:  "true",
		mandatoryContextFileMetadataKey:      s.FilePath,
		mandatoryContextVersionMetadataKey:   s.Version,
		mandatoryContextRuleCountMetadataKey: fmt.Sprintf("%d", len(s.Rules)),
		mandatoryContextIsolatedMetadataKey:  "true",
	}
	if !s.UpdatedAt.IsZero() {
		metadata[mandatoryContextUpdatedAtMetadataKey] = s.UpdatedAt.UTC().Format(time.RFC3339)
	}
	if !s.LoadedAt.IsZero() {
		metadata[mandatoryContextLoadedAtMetadataKey] = s.LoadedAt.UTC().Format(time.RFC3339)
	}
	return metadata
}

func (s mandatoryContextSnapshot) ResultMetadata() map[string]string {
	return s.Metadata()
}

type mandatoryContextConflict struct {
	Source         string
	Key            string
	MemoryValue    string
	MandatoryValue string
}

type mandatoryContextStore struct {
	mu      sync.RWMutex
	options MandatoryContextOptions
	state   mandatoryContextState
}

type mandatoryContextState struct {
	Content    string
	Version    string
	UpdatedAt  time.Time
	LoadedAt   time.Time
	FileDigest string
	FileModAt  time.Time
	FileSize   int64
	Rules      map[string]string
}

func newMandatoryContextStore(options MandatoryContextOptions) *mandatoryContextStore {
	store := &mandatoryContextStore{
		options: normalizeMandatoryContextOptions(options),
	}
	store.refresh(time.Now().UTC())
	return store
}

func normalizeMandatoryContextOptions(options MandatoryContextOptions) MandatoryContextOptions {
	path := strings.TrimSpace(options.FilePath)
	if path == "" {
		path = defaultMandatoryContextFilePath
	}
	options.FilePath = filepath.Clean(path)
	return options
}

func (s *mandatoryContextStore) Snapshot(now time.Time) mandatoryContextSnapshot {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	s.refresh(now)

	s.mu.RLock()
	defer s.mu.RUnlock()

	return mandatoryContextSnapshot{
		FilePath:  s.options.FilePath,
		Content:   s.state.Content,
		Version:   s.state.Version,
		UpdatedAt: s.state.UpdatedAt,
		LoadedAt:  s.state.LoadedAt,
		Rules:     cloneStringMap(s.state.Rules),
	}
}

func (s *mandatoryContextStore) refresh(now time.Time) {
	path := strings.TrimSpace(s.options.FilePath)
	if path == "" {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.mu.Lock()
			s.state = mandatoryContextState{}
			s.mu.Unlock()
		}
		return
	}
	if info.IsDir() {
		return
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	content := normalizeMandatoryContextContent(raw)
	digest := mandatoryContextDigest(content)
	modAt := info.ModTime().UTC()
	fileSize := info.Size()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state.FileDigest == digest && s.state.FileModAt.Equal(modAt) && s.state.FileSize == fileSize {
		return
	}

	s.state = mandatoryContextState{
		Content:    content,
		Version:    mandatoryContextVersion(digest),
		UpdatedAt:  modAt,
		LoadedAt:   now.UTC(),
		FileDigest: digest,
		FileModAt:  modAt,
		FileSize:   fileSize,
		Rules:      parseMandatoryContextRules(content),
	}
}

func normalizeMandatoryContextContent(raw []byte) string {
	content := strings.TrimSpace(string(raw))
	if content == "" {
		return ""
	}
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	return content
}

func mandatoryContextDigest(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func mandatoryContextVersion(digest string) string {
	digest = strings.TrimSpace(digest)
	if digest == "" {
		return ""
	}
	if len(digest) <= mandatoryContextVersionSize {
		return digest
	}
	return digest[:mandatoryContextVersionSize]
}

func parseMandatoryContextRules(content string) map[string]string {
	if strings.TrimSpace(content) == "" {
		return nil
	}

	rules := map[string]string{}
	lines := strings.Split(content, "\n")
	for _, rawLine := range lines {
		line := normalizeMandatoryRuleLine(rawLine)
		if line == "" {
			continue
		}
		separator := strings.IndexAny(line, ":：=")
		if separator <= 0 {
			continue
		}
		key := normalizeMandatoryContextKey(line[:separator])
		value := normalizeMandatoryComparisonValue(line[separator+1:])
		if key == "" || value == "" {
			continue
		}
		rules[key] = value
	}
	if len(rules) == 0 {
		return nil
	}
	return rules
}

func normalizeMandatoryRuleLine(rawLine string) string {
	line := strings.TrimSpace(rawLine)
	if line == "" {
		return ""
	}
	if strings.HasPrefix(line, "#") {
		return ""
	}

	line = strings.TrimLeft(line, "-*+ \t")
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}

	if idx := strings.Index(line, ". "); idx > 0 {
		prefix := strings.TrimSpace(line[:idx])
		if isDigitsOnly(prefix) {
			line = strings.TrimSpace(line[idx+2:])
		}
	}

	line = strings.TrimSpace(strings.TrimPrefix(line, "[ ]"))
	line = strings.TrimSpace(strings.TrimPrefix(line, "[x]"))
	line = strings.TrimSpace(strings.TrimPrefix(line, "[X]"))
	return line
}

func isDigitsOnly(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func normalizeMandatoryComparisonValue(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	return strings.Join(strings.Fields(value), " ")
}

func buildMandatoryContextPrompt(prompt string, snapshot mandatoryContextSnapshot) string {
	base := strings.TrimSpace(prompt)
	if !snapshot.Active() {
		return base
	}

	section := renderMandatoryContextSection(snapshot)
	if base == "" {
		return section
	}
	return section + "\n\n" + base
}

func renderMandatoryContextSection(snapshot mandatoryContextSnapshot) string {
	var builder strings.Builder
	builder.WriteString("[MANDATORY CONTEXT]\n")
	builder.WriteString("Priority: highest. Must override conflicting session/long-term memory.\n")
	builder.WriteString("Source file: ")
	builder.WriteString(snapshot.FilePath)
	builder.WriteByte('\n')
	if snapshot.Version != "" {
		builder.WriteString("Version: ")
		builder.WriteString(snapshot.Version)
		builder.WriteByte('\n')
	}
	if !snapshot.UpdatedAt.IsZero() {
		builder.WriteString("Updated at: ")
		builder.WriteString(snapshot.UpdatedAt.UTC().Format(time.RFC3339))
		builder.WriteByte('\n')
	}
	builder.WriteString("Requirements:\n")
	builder.WriteString(snapshot.Content)
	return strings.TrimSpace(builder.String())
}

func applyMandatoryContextToSessionMemory(
	snapshot sessionMemorySnapshot,
	mandatorySnapshot mandatoryContextSnapshot,
) (sessionMemorySnapshot, []mandatoryContextConflict) {
	if len(snapshot.KeyState) == 0 || len(mandatorySnapshot.Rules) == 0 {
		return snapshot, nil
	}

	keyState := cloneStringMap(snapshot.KeyState)
	conflicts := make([]mandatoryContextConflict, 0)
	for key, value := range snapshot.KeyState {
		required, ok := mandatorySnapshot.Rules[normalizeMandatoryContextKey(key)]
		if !ok || !mandatoryContextValuesConflict(required, value) {
			continue
		}

		conflicts = append(conflicts, mandatoryContextConflict{
			Source:         "session_memory",
			Key:            key,
			MemoryValue:    value,
			MandatoryValue: required,
		})
		keyState[key] = required
	}
	if len(conflicts) == 0 {
		return snapshot, nil
	}
	snapshot.KeyState = keyState
	return snapshot, conflicts
}

func applyMandatoryContextToLongTermMemory(
	snapshot longTermMemorySnapshot,
	mandatorySnapshot mandatoryContextSnapshot,
) (longTermMemorySnapshot, []mandatoryContextConflict) {
	if len(snapshot.Hits) == 0 || len(mandatorySnapshot.Rules) == 0 {
		return snapshot, nil
	}

	filteredHits := make([]longTermMemoryHit, 0, len(snapshot.Hits))
	conflicts := make([]mandatoryContextConflict, 0)
	for _, hit := range snapshot.Hits {
		required, ok := mandatorySnapshot.Rules[normalizeMandatoryContextKey(hit.Entry.Key)]
		if !ok || !mandatoryContextValuesConflict(required, hit.Entry.Value) {
			filteredHits = append(filteredHits, hit)
			continue
		}

		conflicts = append(conflicts, mandatoryContextConflict{
			Source:         "long_term_memory",
			Key:            hit.Entry.Key,
			MemoryValue:    hit.Entry.Value,
			MandatoryValue: required,
		})
	}
	if len(conflicts) == 0 {
		return snapshot, nil
	}
	snapshot.Hits = filteredHits
	snapshot.TierHits = groupSelectedLongTermMemoryHitsByTier(filteredHits)
	snapshot.TokenUsed = 0
	for _, hit := range filteredHits {
		snapshot.TokenUsed += estimateLongTermMemoryEntryTokens(hit.Entry)
	}
	return snapshot, conflicts
}

func mandatoryContextValuesConflict(required string, memoryValue string) bool {
	required = normalizeMandatoryComparisonValue(required)
	memoryValue = normalizeMandatoryComparisonValue(memoryValue)
	if required == "" || memoryValue == "" {
		return false
	}
	if required == memoryValue {
		return false
	}
	if strings.Contains(required, memoryValue) || strings.Contains(memoryValue, required) {
		return false
	}
	return true
}

func mandatoryContextConflictMetadata(conflicts []mandatoryContextConflict) map[string]string {
	if len(conflicts) == 0 {
		return nil
	}

	keys := make([]string, 0, len(conflicts))
	details := make([]string, 0, len(conflicts))
	seenKeys := map[string]struct{}{}
	for _, conflict := range conflicts {
		normalizedKey := normalizeMandatoryContextKey(conflict.Key)
		if normalizedKey != "" {
			if _, exists := seenKeys[normalizedKey]; !exists {
				seenKeys[normalizedKey] = struct{}{}
				keys = append(keys, normalizedKey)
			}
		}
		details = append(details, fmt.Sprintf("%s:%s", conflict.Source, conflict.Key))
	}
	sort.Strings(keys)
	sort.Strings(details)
	metadata := map[string]string{
		mandatoryContextConflictCountMetadataKey:  fmt.Sprintf("%d", len(conflicts)),
		mandatoryContextConflictDetailMetadataKey: strings.Join(details, ";"),
	}
	if len(keys) > 0 {
		metadata[mandatoryContextConflictKeysMetadataKey] = strings.Join(keys, ",")
	}
	return metadata
}

func cloneStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func normalizeMandatoryContextKey(raw string) string {
	normalized := normalizeLongTermMemoryKey(raw)
	if normalized == "" {
		return ""
	}
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.Trim(normalized, "_")
	normalized = strings.Join(strings.FieldsFunc(normalized, func(r rune) bool {
		return r == '_'
	}), "_")
	return normalized
}

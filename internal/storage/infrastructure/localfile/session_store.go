package localfile

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	sessionapp "alter0/internal/session/application"
	sessiondomain "alter0/internal/session/domain"
)

const defaultSessionAgentDir = "_default"

var chatSessionArchiveLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

type sessionState struct {
	Messages []sessiondomain.MessageRecord `json:"messages"`
}

type SessionStore struct {
	legacyPath  string
	sessionsDir string
	format      Format
	mu          sync.Mutex
}

func NewSessionStore(baseDir string, format Format) *SessionStore {
	return &SessionStore{
		legacyPath:  filepath.Join(baseDir, "sessions."+extension(format)),
		sessionsDir: filepath.Join(baseDir, "sessions"),
		format:      format,
	}
}

var _ sessionapp.Store = (*SessionStore)(nil)

func (s *SessionStore) Load(_ context.Context) ([]sessiondomain.MessageRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	layoutItems, layoutFound, layoutNeedsRewrite, err := s.loadSessionLayout()
	if err != nil {
		return nil, err
	}
	if layoutFound {
		legacyItems, legacyFound, err := s.loadLegacyAggregate()
		if err != nil {
			return nil, err
		}
		if legacyFound {
			merged := mergeSessionRecords(layoutItems, legacyItems)
			if err := s.saveSessionLayoutLocked(merged); err != nil {
				return nil, err
			}
			return merged, nil
		}
		if layoutNeedsRewrite {
			if err := s.saveSessionLayoutLocked(layoutItems); err != nil {
				return nil, err
			}
		}
		return layoutItems, nil
	}

	legacyItems, legacyFound, err := s.loadLegacyAggregate()
	if err != nil {
		return nil, err
	}
	if legacyFound {
		if err := s.saveSessionLayoutLocked(legacyItems); err != nil {
			return nil, err
		}
	}
	return legacyItems, nil
}

func (s *SessionStore) Save(_ context.Context, records []sessiondomain.MessageRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.saveSessionLayoutLocked(records)
}

func (s *SessionStore) saveSessionLayoutLocked(records []sessiondomain.MessageRecord) error {
	groups := groupSessionRecords(records)
	kept := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		path := s.sessionFilePath(group.agentID, group.sessionID, group.archiveDay)
		raw, err := marshalPayload(s.format, "alter0 session history", sessionState{Messages: group.records})
		if err != nil {
			return err
		}
		if err := writeFile(path, raw); err != nil {
			return err
		}
		kept[filepath.Clean(path)] = struct{}{}
	}
	if err := s.cleanupRemovedSessionFiles(kept); err != nil {
		return err
	}
	_ = os.Remove(s.legacyPath)
	return nil
}

type groupedSessionRecords struct {
	agentID    string
	sessionID  string
	archiveDay string
	records    []sessiondomain.MessageRecord
}

func (s *SessionStore) loadLegacyAggregate() ([]sessiondomain.MessageRecord, bool, error) {
	raw, ok, err := readIfExists(s.legacyPath)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return []sessiondomain.MessageRecord{}, false, nil
	}

	state := sessionState{}
	if err := unmarshalPayload(s.format, raw, &state); err != nil {
		return nil, true, err
	}
	if len(state.Messages) == 0 {
		return []sessiondomain.MessageRecord{}, true, nil
	}

	items := make([]sessiondomain.MessageRecord, 0, len(state.Messages))
	for _, item := range state.Messages {
		items = append(items, item)
	}
	return items, true, nil
}

func (s *SessionStore) loadSessionLayout() ([]sessiondomain.MessageRecord, bool, bool, error) {
	info, err := os.Stat(s.sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []sessiondomain.MessageRecord{}, false, false, nil
		}
		return nil, false, false, err
	}
	if !info.IsDir() {
		return []sessiondomain.MessageRecord{}, false, false, nil
	}

	paths := []string{}
	if err := filepath.WalkDir(s.sessionsDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), "."+extension(s.format)) {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		return nil, false, false, err
	}
	if len(paths) == 0 {
		return []sessiondomain.MessageRecord{}, false, false, nil
	}
	sort.Strings(paths)

	items := []sessiondomain.MessageRecord{}
	needsRewrite := false
	for _, path := range paths {
		raw, ok, err := readIfExists(path)
		if err != nil {
			return nil, true, needsRewrite, err
		}
		if !ok {
			continue
		}
		state := sessionState{}
		if err := unmarshalPayload(s.format, raw, &state); err != nil {
			return nil, true, needsRewrite, err
		}
		for _, message := range state.Messages {
			if strings.TrimSpace(message.SessionID) == "" {
				continue
			}
			if filepath.Clean(path) != filepath.Clean(s.sessionFilePath("", message.SessionID, "")) {
				needsRewrite = true
				break
			}
		}
		items = append(items, state.Messages...)
	}
	return items, true, needsRewrite, nil
}

func (s *SessionStore) cleanupRemovedSessionFiles(kept map[string]struct{}) error {
	info, err := os.Stat(s.sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}

	dirs := []string{}
	if err := filepath.WalkDir(s.sessionsDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != s.sessionsDir {
				dirs = append(dirs, path)
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".json" && ext != ".md" {
			return nil
		}
		if _, ok := kept[filepath.Clean(path)]; ok {
			return nil
		}
		return os.Remove(path)
	}); err != nil {
		return err
	}

	sort.Slice(dirs, func(i, j int) bool {
		return len(dirs[i]) > len(dirs[j])
	})
	for _, dir := range dirs {
		_ = os.Remove(dir)
	}
	return nil
}

func (s *SessionStore) sessionFilePath(agentID string, sessionID string, archiveDay string) string {
	_, _ = agentID, archiveDay
	return filepath.Join(
		s.sessionsDir,
		defaultSessionAgentDir,
		sanitizeSessionStoreSegment(sessionID)+"."+extension(s.format),
	)
}

func groupSessionRecords(records []sessiondomain.MessageRecord) []groupedSessionRecords {
	bySession := map[string][]sessiondomain.MessageRecord{}
	order := []string{}
	for _, record := range records {
		sessionID := strings.TrimSpace(record.SessionID)
		if sessionID == "" {
			continue
		}
		key := strings.ToLower(sessionID)
		if _, ok := bySession[key]; !ok {
			order = append(order, key)
		}
		bySession[key] = append(bySession[key], record)
	}

	groups := make([]groupedSessionRecords, 0, len(order))
	for _, key := range order {
		items := append([]sessiondomain.MessageRecord(nil), bySession[key]...)
		sessionID := strings.TrimSpace(items[0].SessionID)
		groups = append(groups, groupedSessionRecords{
			agentID:   defaultSessionAgentDir,
			sessionID: sessionID,
			records:   items,
		})
	}
	return groups
}

func isCanonicalChatSessionGroup(agentID string, sessionID string) bool {
	return resolveSessionAgentID(agentID) == defaultSessionAgentDir &&
		strings.EqualFold(strings.TrimSpace(sessionID), sessiondomain.CanonicalChatSessionID)
}

func groupCanonicalChatSessionRecords(agentID string, sessionID string, records []sessiondomain.MessageRecord) []groupedSessionRecords {
	byDay := map[string][]sessiondomain.MessageRecord{}
	order := []string{}
	for _, record := range records {
		day := chatSessionArchiveDay(record.Timestamp)
		if _, ok := byDay[day]; !ok {
			order = append(order, day)
		}
		byDay[day] = append(byDay[day], record)
	}
	groups := make([]groupedSessionRecords, 0, len(order))
	for _, day := range order {
		groups = append(groups, groupedSessionRecords{
			agentID:    agentID,
			sessionID:  sessionID,
			archiveDay: day,
			records:    append([]sessiondomain.MessageRecord(nil), byDay[day]...),
		})
	}
	return groups
}

func chatSessionArchiveDay(ts time.Time) string {
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	local := ts.In(chatSessionArchiveLocation)
	if local.Hour() < 5 {
		local = local.AddDate(0, 0, -1)
	}
	return local.Format("2006-01-02")
}

func isPreviousCanonicalChatSessionFile(path string, records []sessiondomain.MessageRecord, format Format) bool {
	if !strings.EqualFold(filepath.Base(path), sessiondomain.CanonicalChatSessionID+"."+extension(format)) {
		return false
	}
	for _, record := range records {
		if strings.EqualFold(strings.TrimSpace(record.SessionID), sessiondomain.CanonicalChatSessionID) {
			return true
		}
	}
	return false
}

func mergeSessionRecords(primary []sessiondomain.MessageRecord, fallback []sessiondomain.MessageRecord) []sessiondomain.MessageRecord {
	if len(primary) == 0 {
		return append([]sessiondomain.MessageRecord(nil), fallback...)
	}
	if len(fallback) == 0 {
		return append([]sessiondomain.MessageRecord(nil), primary...)
	}
	merged := make([]sessiondomain.MessageRecord, 0, len(primary)+len(fallback))
	seen := map[string]struct{}{}
	appendUnique := func(records []sessiondomain.MessageRecord) {
		for _, record := range records {
			key := sessionRecordStorageKey(record)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			merged = append(merged, record)
		}
	}
	appendUnique(primary)
	appendUnique(fallback)
	return merged
}

func sessionRecordStorageKey(record sessiondomain.MessageRecord) string {
	parts := []string{
		strings.ToLower(strings.TrimSpace(record.SessionID)),
		strings.TrimSpace(record.MessageID),
		string(record.Role),
		record.Timestamp.UTC().Format("2006-01-02T15:04:05.000000000Z07:00"),
	}
	return strings.Join(parts, "\x00")
}

func resolveSessionAgentID(agentID string) string {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return defaultSessionAgentDir
	}
	return agentID
}

func sanitizeSessionStoreSegment(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return defaultSessionAgentDir
	}
	if trimmed == defaultSessionAgentDir {
		return defaultSessionAgentDir
	}
	var builder strings.Builder
	lastUnderscore := false
	for _, r := range trimmed {
		allowed := unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.'
		if allowed {
			builder.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}
	result := strings.Trim(builder.String(), "._-")
	if result == "" {
		return defaultSessionAgentDir
	}
	return result
}

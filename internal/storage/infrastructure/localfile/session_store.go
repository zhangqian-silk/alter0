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

const defaultSessionBucketDir = "_default"

var chatSessionArchiveLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

type sessionState struct {
	Messages []sessiondomain.MessageRecord `json:"messages"`
}

type SessionStore struct {
	sessionsDir string
	format      Format
	mu          sync.Mutex
}

func NewSessionStore(baseDir string, format Format) *SessionStore {
	return &SessionStore{
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
		if layoutNeedsRewrite {
			if err := s.saveSessionLayoutLocked(layoutItems); err != nil {
				return nil, err
			}
		}
		return layoutItems, nil
	}

	return []sessiondomain.MessageRecord{}, nil
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
		path := s.sessionFilePath(group.bucketID, group.sessionID, group.archiveDay)
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
	return nil
}

type groupedSessionRecords struct {
	bucketID   string
	sessionID  string
	archiveDay string
	records    []sessiondomain.MessageRecord
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

func (s *SessionStore) sessionFilePath(bucketID string, sessionID string, archiveDay string) string {
	_, _ = bucketID, archiveDay
	return filepath.Join(
		s.sessionsDir,
		defaultSessionBucketDir,
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
			bucketID:  defaultSessionBucketDir,
			sessionID: sessionID,
			records:   items,
		})
	}
	return groups
}

func isCanonicalChatSessionGroup(bucketID string, sessionID string) bool {
	return strings.TrimSpace(bucketID) == defaultSessionBucketDir &&
		strings.EqualFold(strings.TrimSpace(sessionID), sessiondomain.CanonicalChatSessionID)
}

func groupCanonicalChatSessionRecords(bucketID string, sessionID string, records []sessiondomain.MessageRecord) []groupedSessionRecords {
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
			bucketID:   bucketID,
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

func sanitizeSessionStoreSegment(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return defaultSessionBucketDir
	}
	if trimmed == defaultSessionBucketDir {
		return defaultSessionBucketDir
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
		return defaultSessionBucketDir
	}
	return result
}

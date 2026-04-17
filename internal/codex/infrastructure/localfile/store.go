package localfile

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	codexapp "alter0/internal/codex/application"
)

type Record = codexapp.Record

type Store struct {
	root string
	mu   sync.Mutex
}

func NewStore(root string) (*Store, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "" {
		return nil, fmt.Errorf("store root is required")
	}
	store := &Store{root: root}
	for _, dir := range []string{store.accountsDir(), store.backupsDir(), store.loginSessionsDir()} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create store dir %s: %w", dir, err)
		}
	}
	return store, nil
}

func (s *Store) Save(record Record, overwrite bool) error {
	name, err := NormalizeName(record.Name)
	if err != nil {
		return err
	}
	record.Name = name

	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.recordPath(record.Name)
	if !overwrite {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("account %q already exists", record.Name)
		}
	}
	raw, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal account record: %w", err)
	}
	return writeFileWithBackup(path, raw)
}

func (s *Store) Load(name string) (*Record, error) {
	name, err := NormalizeName(name)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record, err := s.loadRecordLocked(name)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (s *Store) List() ([]Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.accountsDir())
	if err != nil {
		return nil, fmt.Errorf("read account dir: %w", err)
	}
	items := make([]Record, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(s.accountsDir(), entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read account file %s: %w", entry.Name(), err)
		}
		var record Record
		if err := json.Unmarshal(raw, &record); err != nil {
			return nil, fmt.Errorf("parse account file %s: %w", entry.Name(), err)
		}
		items = append(items, record)
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	return items, nil
}

func (s *Store) CreateBackup(raw []byte) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.backupsDir(), "auth-"+time.Now().UTC().Format("20060102-150405.000")+".json")
	if err := writeFileWithBackup(path, raw); err != nil {
		return "", err
	}
	return path, nil
}

func (s *Store) PrepareLoginHome(sessionID string) (string, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", fmt.Errorf("login session id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	home := filepath.Join(s.loginSessionsDir(), sessionID, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		return "", fmt.Errorf("create login home: %w", err)
	}
	return home, nil
}

func (s *Store) accountsDir() string {
	return filepath.Join(s.root, "accounts")
}

func (s *Store) backupsDir() string {
	return filepath.Join(s.root, "backups")
}

func (s *Store) loginSessionsDir() string {
	return filepath.Join(s.root, "login-sessions")
}

func (s *Store) recordPath(name string) string {
	return filepath.Join(s.accountsDir(), slugForName(name)+".json")
}

func (s *Store) loadRecordLocked(name string) (Record, error) {
	records, err := s.listRecordsLocked()
	if err != nil {
		return Record{}, err
	}
	name = strings.TrimSpace(name)
	nameFolded := strings.ToLower(name)
	nameShell := ShellName(name)
	for _, record := range records {
		if strings.EqualFold(strings.TrimSpace(record.Name), name) {
			return record, nil
		}
		if ShellName(record.Name) == nameShell {
			return record, nil
		}
		if strings.ToLower(strings.TrimSpace(record.Name)) == nameFolded {
			return record, nil
		}
	}
	return Record{}, fmt.Errorf("account %q not found", name)
}

func (s *Store) listRecordsLocked() ([]Record, error) {
	entries, err := os.ReadDir(s.accountsDir())
	if err != nil {
		return nil, fmt.Errorf("read account dir: %w", err)
	}
	items := make([]Record, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(s.accountsDir(), entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read account file %s: %w", entry.Name(), err)
		}
		var record Record
		if err := json.Unmarshal(raw, &record); err != nil {
			return nil, fmt.Errorf("parse account file %s: %w", entry.Name(), err)
		}
		items = append(items, record)
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	return items, nil
}

func NormalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("account name is required")
	}
	return name, nil
}

func ShellName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	var builder strings.Builder
	lastDash := false
	for _, r := range name {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(unicode.ToLower(r))
			lastDash = false
		case builder.Len() > 0 && !lastDash:
			builder.WriteByte('-')
			lastDash = true
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "codex-account"
	}
	return result
}

func slugForName(name string) string {
	sum := sha1.Sum([]byte(strings.ToLower(strings.TrimSpace(name))))
	return hex.EncodeToString(sum[:])
}

func writeFileWithBackup(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create dir %s: %w", filepath.Dir(path), err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, content, 0o600); err != nil {
		return fmt.Errorf("write temp file %s: %w", tmp, err)
	}
	backup := path + ".bak"
	_ = os.Remove(backup)
	if _, err := os.Stat(path); err == nil {
		if err := os.Rename(path, backup); err != nil {
			_ = os.Remove(tmp)
			return fmt.Errorf("create backup %s: %w", backup, err)
		}
	}
	if err := os.Rename(tmp, path); err != nil {
		if _, restoreErr := os.Stat(backup); restoreErr == nil {
			_ = os.Rename(backup, path)
		}
		_ = os.Remove(tmp)
		return fmt.Errorf("replace file %s: %w", path, err)
	}
	_ = os.Remove(backup)
	return nil
}

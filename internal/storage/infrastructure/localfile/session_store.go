package localfile

import (
	"context"
	"path/filepath"
	"sync"

	sessionapp "alter0/internal/session/application"
	sessiondomain "alter0/internal/session/domain"
)

type sessionState struct {
	Messages []sessiondomain.MessageRecord `json:"messages"`
}

type SessionStore struct {
	path   string
	format Format
	mu     sync.Mutex
}

func NewSessionStore(baseDir string, format Format) *SessionStore {
	return &SessionStore{
		path:   filepath.Join(baseDir, "sessions."+extension(format)),
		format: format,
	}
}

var _ sessionapp.Store = (*SessionStore)(nil)

func (s *SessionStore) Load(_ context.Context) ([]sessiondomain.MessageRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	raw, ok, err := readIfExists(s.path)
	if err != nil {
		return nil, err
	}
	if !ok {
		return []sessiondomain.MessageRecord{}, nil
	}

	state := sessionState{}
	if err := unmarshalPayload(s.format, raw, &state); err != nil {
		return nil, err
	}
	if len(state.Messages) == 0 {
		return []sessiondomain.MessageRecord{}, nil
	}

	items := make([]sessiondomain.MessageRecord, 0, len(state.Messages))
	for _, item := range state.Messages {
		items = append(items, item)
	}
	return items, nil
}

func (s *SessionStore) Save(_ context.Context, records []sessiondomain.MessageRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]sessiondomain.MessageRecord, 0, len(records))
	for _, item := range records {
		items = append(items, item)
	}

	raw, err := marshalPayload(s.format, "alter0 session history", sessionState{Messages: items})
	if err != nil {
		return err
	}
	return writeFile(s.path, raw)
}

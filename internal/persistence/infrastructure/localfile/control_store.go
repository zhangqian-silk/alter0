package localfile

import (
	"context"
	"path/filepath"
	"sync"

	controlapp "alter0/internal/control/application"
	controldomain "alter0/internal/control/domain"
)

type controlState struct {
	Channels []controldomain.Channel `json:"channels"`
	Skills   []controldomain.Skill   `json:"skills"`
}

type ControlStore struct {
	path   string
	format Format
	mu     sync.Mutex
}

func NewControlStore(baseDir string, format Format) *ControlStore {
	return &ControlStore{
		path:   filepath.Join(baseDir, "control."+extension(format)),
		format: format,
	}
}

var _ controlapp.Persistence = (*ControlStore)(nil)

func (s *ControlStore) Load(_ context.Context) ([]controldomain.Channel, []controldomain.Skill, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	raw, ok, err := readIfExists(s.path)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		return []controldomain.Channel{}, []controldomain.Skill{}, nil
	}

	state := controlState{}
	if err := unmarshalPayload(s.format, raw, &state); err != nil {
		return nil, nil, err
	}
	return state.Channels, state.Skills, nil
}

func (s *ControlStore) Save(_ context.Context, channels []controldomain.Channel, skills []controldomain.Skill) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	raw, err := marshalPayload(s.format, "alter0 control state", controlState{
		Channels: channels,
		Skills:   skills,
	})
	if err != nil {
		return err
	}
	return writeFile(s.path, raw)
}

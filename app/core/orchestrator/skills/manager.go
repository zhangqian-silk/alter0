package skills

import (
	"alter0/app/pkg/types"
	"context"
	"errors"
	"sync"
)

type Manager struct {
	skills map[string]types.Skill
	mu     sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		skills: make(map[string]types.Skill),
	}
}

func (m *Manager) Register(s types.Skill) {
	m.mu.Lock()
	defer m.mu.Unlock()
	manifest := s.Manifest()
	m.skills[manifest.Name] = s
}

func (m *Manager) GetSkill(name string) (types.Skill, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.skills[name]
	return s, ok
}

func (m *Manager) ListSkills() []types.SkillManifest {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]types.SkillManifest, 0, len(m.skills))
	for _, s := range m.skills {
		list = append(list, s.Manifest())
	}
	return list
}

func (m *Manager) Execute(ctx context.Context, skillName string, args map[string]interface{}) (interface{}, error) {
	s, ok := m.GetSkill(skillName)
	if !ok {
		return nil, errors.New("skill not found: " + skillName)
	}
	return s.Execute(ctx, args)
}

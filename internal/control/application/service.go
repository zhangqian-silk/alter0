package application

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	controldomain "alter0/internal/control/domain"
)

type Store interface {
	Load(ctx context.Context) (channels []controldomain.Channel, skills []controldomain.Skill, err error)
	Save(ctx context.Context, channels []controldomain.Channel, skills []controldomain.Skill) error
}

type Service struct {
	mu       sync.RWMutex
	channels map[string]controldomain.Channel
	skills   map[string]controldomain.Skill
	store    Store
}

func NewService() *Service {
	return newService(nil)
}

func NewServiceWithStore(ctx context.Context, store Store) (*Service, error) {
	service := newService(store)
	if store == nil {
		return service, nil
	}

	channels, skills, err := store.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("load control state: %w", err)
	}

	for _, channel := range channels {
		if err := channel.Validate(); err != nil {
			return nil, fmt.Errorf("invalid channel in store: %w", err)
		}
		service.channels[normalize(channel.ID)] = channel
	}
	for _, skill := range skills {
		if err := skill.Validate(); err != nil {
			return nil, fmt.Errorf("invalid skill in store: %w", err)
		}
		service.skills[normalize(skill.ID)] = skill
	}
	return service, nil
}

func newService(store Store) *Service {
	return &Service{
		channels: map[string]controldomain.Channel{},
		skills:   map[string]controldomain.Skill{},
		store:    store,
	}
}

func (s *Service) UpsertChannel(channel controldomain.Channel) error {
	if err := channel.Validate(); err != nil {
		return err
	}

	key := normalize(channel.ID)
	s.mu.Lock()
	defer s.mu.Unlock()
	previous, existed := s.channels[key]
	s.channels[key] = channel
	if err := s.storeLocked(); err != nil {
		if existed {
			s.channels[key] = previous
		} else {
			delete(s.channels, key)
		}
		return err
	}
	return nil
}

func (s *Service) ResolveChannel(id string) (controldomain.Channel, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	channel, ok := s.channels[normalize(id)]
	return channel, ok
}

func (s *Service) DeleteChannel(id string) bool {
	key := normalize(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	previous, ok := s.channels[key]
	if !ok {
		return false
	}
	delete(s.channels, key)
	if err := s.storeLocked(); err != nil {
		s.channels[key] = previous
		return false
	}
	return true
}

func (s *Service) ListChannels() []controldomain.Channel {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]string, 0, len(s.channels))
	for _, channel := range s.channels {
		ids = append(ids, channel.ID)
	}
	sort.Strings(ids)

	items := make([]controldomain.Channel, 0, len(ids))
	for _, id := range ids {
		items = append(items, s.channels[normalize(id)])
	}
	return items
}

func (s *Service) UpsertSkill(skill controldomain.Skill) error {
	if err := skill.Validate(); err != nil {
		return err
	}

	key := normalize(skill.ID)
	s.mu.Lock()
	defer s.mu.Unlock()
	previous, existed := s.skills[key]
	s.skills[key] = skill
	if err := s.storeLocked(); err != nil {
		if existed {
			s.skills[key] = previous
		} else {
			delete(s.skills, key)
		}
		return err
	}
	return nil
}

func (s *Service) ResolveSkill(id string) (controldomain.Skill, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	skill, ok := s.skills[normalize(id)]
	return skill, ok
}

func (s *Service) DeleteSkill(id string) bool {
	key := normalize(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	previous, ok := s.skills[key]
	if !ok {
		return false
	}
	delete(s.skills, key)
	if err := s.storeLocked(); err != nil {
		s.skills[key] = previous
		return false
	}
	return true
}

func (s *Service) ListSkills() []controldomain.Skill {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]string, 0, len(s.skills))
	for _, skill := range s.skills {
		ids = append(ids, skill.ID)
	}
	sort.Strings(ids)

	items := make([]controldomain.Skill, 0, len(ids))
	for _, id := range ids {
		items = append(items, s.skills[normalize(id)])
	}
	return items
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func (s *Service) storeLocked() error {
	if s.store == nil {
		return nil
	}
	if err := s.store.Save(context.Background(), snapshotChannels(s.channels), snapshotSkills(s.skills)); err != nil {
		return fmt.Errorf("store control state: %w", err)
	}
	return nil
}

func snapshotChannels(items map[string]controldomain.Channel) []controldomain.Channel {
	out := make([]controldomain.Channel, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		return normalize(out[i].ID) < normalize(out[j].ID)
	})
	return out
}

func snapshotSkills(items map[string]controldomain.Skill) []controldomain.Skill {
	out := make([]controldomain.Skill, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		return normalize(out[i].ID) < normalize(out[j].ID)
	})
	return out
}

package application

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	controldomain "alter0/internal/control/domain"
)

const capabilityAuditLimit = 1000

type Store interface {
	Load(ctx context.Context) (channels []controldomain.Channel, capabilities []controldomain.Capability, audits []controldomain.CapabilityAudit, err error)
	Save(ctx context.Context, channels []controldomain.Channel, capabilities []controldomain.Capability, audits []controldomain.CapabilityAudit) error
}

type Service struct {
	mu           sync.RWMutex
	channels     map[string]controldomain.Channel
	capabilities map[string]controldomain.Capability
	audits       []controldomain.CapabilityAudit
	store        Store
}

func NewService() *Service {
	return newService(nil)
}

func NewServiceWithStore(ctx context.Context, store Store) (*Service, error) {
	service := newService(store)
	if store == nil {
		return service, nil
	}

	channels, capabilities, audits, err := store.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("load control state: %w", err)
	}

	for _, channel := range channels {
		if err := channel.Validate(); err != nil {
			return nil, fmt.Errorf("invalid channel in store: %w", err)
		}
		service.channels[normalize(channel.ID)] = cloneChannel(channel)
	}
	for _, capability := range capabilities {
		normalized := capability.Normalized()
		if err := normalized.Validate(); err != nil {
			return nil, fmt.Errorf("invalid capability in store: %w", err)
		}
		service.capabilities[capabilityKey(normalized.Type, normalized.ID)] = cloneCapability(normalized)
	}
	service.audits = cloneAudits(audits)
	return service, nil
}

func newService(store Store) *Service {
	return &Service{
		channels:     map[string]controldomain.Channel{},
		capabilities: map[string]controldomain.Capability{},
		audits:       []controldomain.CapabilityAudit{},
		store:        store,
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
	s.channels[key] = cloneChannel(channel)
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
	if !ok {
		return controldomain.Channel{}, false
	}
	return cloneChannel(channel), true
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
		items = append(items, cloneChannel(s.channels[normalize(id)]))
	}
	return items
}

func (s *Service) UpsertCapability(capability controldomain.Capability) error {
	normalized := capability.Normalized()
	if err := normalized.Validate(); err != nil {
		return err
	}

	key := capabilityKey(normalized.Type, normalized.ID)
	s.mu.Lock()
	defer s.mu.Unlock()
	previous, existed := s.capabilities[key]
	previousAuditLen := len(s.audits)

	s.capabilities[key] = cloneCapability(normalized)
	action := resolveLifecycleAction(existed, previous, normalized)
	s.appendAuditLocked(newCapabilityAudit(normalized, action))
	if err := s.storeLocked(); err != nil {
		if existed {
			s.capabilities[key] = previous
		} else {
			delete(s.capabilities, key)
		}
		s.audits = s.audits[:previousAuditLen]
		return err
	}
	return nil
}

func (s *Service) ResolveCapability(capabilityType controldomain.CapabilityType, id string) (controldomain.Capability, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	capability, ok := s.capabilities[capabilityKey(capabilityType, id)]
	if !ok {
		return controldomain.Capability{}, false
	}
	return cloneCapability(capability), true
}

func (s *Service) DeleteCapability(capabilityType controldomain.CapabilityType, id string) bool {
	key := capabilityKey(capabilityType, id)
	s.mu.Lock()
	defer s.mu.Unlock()
	previous, ok := s.capabilities[key]
	if !ok {
		return false
	}
	previousAuditLen := len(s.audits)
	delete(s.capabilities, key)
	s.appendAuditLocked(newCapabilityAudit(previous, controldomain.CapabilityLifecycleDelete))
	if err := s.storeLocked(); err != nil {
		s.capabilities[key] = previous
		s.audits = s.audits[:previousAuditLen]
		return false
	}
	return true
}

func (s *Service) SetCapabilityEnabled(capabilityType controldomain.CapabilityType, id string, enabled bool) (controldomain.Capability, error) {
	key := capabilityKey(capabilityType, id)
	s.mu.Lock()
	defer s.mu.Unlock()

	capability, ok := s.capabilities[key]
	if !ok {
		return controldomain.Capability{}, errors.New("capability not found")
	}
	if capability.Enabled == enabled {
		return cloneCapability(capability), nil
	}

	previousAuditLen := len(s.audits)
	previous := cloneCapability(capability)
	capability.Enabled = enabled
	action := controldomain.CapabilityLifecycleDisable
	if enabled {
		action = controldomain.CapabilityLifecycleEnable
	}
	s.capabilities[key] = cloneCapability(capability)
	s.appendAuditLocked(newCapabilityAudit(capability, action))
	if err := s.storeLocked(); err != nil {
		s.capabilities[key] = previous
		s.audits = s.audits[:previousAuditLen]
		return controldomain.Capability{}, err
	}
	return cloneCapability(capability), nil
}

func (s *Service) ListCapabilities() []controldomain.Capability {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.capabilities))
	for key := range s.capabilities {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	items := make([]controldomain.Capability, 0, len(keys))
	for _, key := range keys {
		items = append(items, cloneCapability(s.capabilities[key]))
	}
	return items
}

func (s *Service) ListCapabilitiesByType(capabilityType controldomain.CapabilityType) []controldomain.Capability {
	if !capabilityType.IsSupported() {
		return []controldomain.Capability{}
	}
	items := s.ListCapabilities()
	filtered := make([]controldomain.Capability, 0, len(items))
	for _, item := range items {
		if item.Type == capabilityType {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func (s *Service) ListCapabilityAudits() []controldomain.CapabilityAudit {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneAudits(s.audits)
}

func (s *Service) UpsertSkill(skill controldomain.Skill) error {
	capability := skill.AsCapability()
	capability.Type = controldomain.CapabilityTypeSkill
	return s.UpsertCapability(capability)
}

func (s *Service) ResolveSkill(id string) (controldomain.Skill, bool) {
	capability, ok := s.ResolveCapability(controldomain.CapabilityTypeSkill, id)
	if !ok {
		return controldomain.Skill{}, false
	}
	return controldomain.SkillFromCapability(capability), true
}

func (s *Service) DeleteSkill(id string) bool {
	return s.DeleteCapability(controldomain.CapabilityTypeSkill, id)
}

func (s *Service) ListSkills() []controldomain.Skill {
	capabilities := s.ListCapabilitiesByType(controldomain.CapabilityTypeSkill)
	items := make([]controldomain.Skill, 0, len(capabilities))
	for _, capability := range capabilities {
		items = append(items, controldomain.SkillFromCapability(capability))
	}
	return items
}

func (s *Service) UpsertMCP(capability controldomain.Capability) error {
	capability.Type = controldomain.CapabilityTypeMCP
	return s.UpsertCapability(capability)
}

func (s *Service) ResolveMCP(id string) (controldomain.Capability, bool) {
	return s.ResolveCapability(controldomain.CapabilityTypeMCP, id)
}

func (s *Service) DeleteMCP(id string) bool {
	return s.DeleteCapability(controldomain.CapabilityTypeMCP, id)
}

func (s *Service) ListMCPs() []controldomain.Capability {
	return s.ListCapabilitiesByType(controldomain.CapabilityTypeMCP)
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func capabilityKey(capabilityType controldomain.CapabilityType, id string) string {
	return normalize(string(capabilityType)) + ":" + normalize(id)
}

func (s *Service) appendAuditLocked(entry controldomain.CapabilityAudit) {
	s.audits = append(s.audits, entry)
	if len(s.audits) > capabilityAuditLimit {
		s.audits = cloneAudits(s.audits[len(s.audits)-capabilityAuditLimit:])
	}
}

func newCapabilityAudit(capability controldomain.Capability, action controldomain.CapabilityLifecycleAction) controldomain.CapabilityAudit {
	normalized := capability.Normalized()
	return controldomain.CapabilityAudit{
		CapabilityID:   normalized.ID,
		CapabilityType: normalized.Type,
		Action:         action,
		Version:        normalized.Version,
		Enabled:        normalized.Enabled,
		Scope:          normalized.Scope,
		Metadata:       cloneMetadata(normalized.Metadata),
		OccurredAt:     time.Now().UTC(),
	}
}

func resolveLifecycleAction(existed bool, previous, current controldomain.Capability) controldomain.CapabilityLifecycleAction {
	if existed {
		if !previous.Enabled && current.Enabled {
			return controldomain.CapabilityLifecycleEnable
		}
		if previous.Enabled && !current.Enabled {
			return controldomain.CapabilityLifecycleDisable
		}
	}
	return controldomain.CapabilityLifecycleUpdate
}

func (s *Service) storeLocked() error {
	if s.store == nil {
		return nil
	}
	if err := s.store.Save(context.Background(), snapshotChannels(s.channels), snapshotCapabilities(s.capabilities), cloneAudits(s.audits)); err != nil {
		return fmt.Errorf("store control state: %w", err)
	}
	return nil
}

func snapshotChannels(items map[string]controldomain.Channel) []controldomain.Channel {
	out := make([]controldomain.Channel, 0, len(items))
	for _, item := range items {
		out = append(out, cloneChannel(item))
	}
	sort.Slice(out, func(i, j int) bool {
		return normalize(out[i].ID) < normalize(out[j].ID)
	})
	return out
}

func snapshotCapabilities(items map[string]controldomain.Capability) []controldomain.Capability {
	out := make([]controldomain.Capability, 0, len(items))
	for _, item := range items {
		out = append(out, cloneCapability(item))
	}
	sort.Slice(out, func(i, j int) bool {
		if normalize(string(out[i].Type)) == normalize(string(out[j].Type)) {
			return normalize(out[i].ID) < normalize(out[j].ID)
		}
		return normalize(string(out[i].Type)) < normalize(string(out[j].Type))
	})
	return out
}

func cloneChannel(channel controldomain.Channel) controldomain.Channel {
	out := channel
	out.Metadata = cloneMetadata(channel.Metadata)
	return out
}

func cloneCapability(capability controldomain.Capability) controldomain.Capability {
	out := capability
	out.Metadata = cloneMetadata(capability.Metadata)
	return out
}

func cloneMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}
	copied := make(map[string]string, len(metadata))
	for key, value := range metadata {
		copied[key] = value
	}
	return copied
}

func cloneAudits(items []controldomain.CapabilityAudit) []controldomain.CapabilityAudit {
	if len(items) == 0 {
		return []controldomain.CapabilityAudit{}
	}
	out := make([]controldomain.CapabilityAudit, 0, len(items))
	for _, item := range items {
		audit := item
		audit.Metadata = cloneMetadata(item.Metadata)
		out = append(out, audit)
	}
	return out
}

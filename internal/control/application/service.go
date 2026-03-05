package application

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sort"
	"strings"
	"sync"
	"time"

	controldomain "alter0/internal/control/domain"
)

const capabilityAuditLimit = 1000
const environmentAuditLimit = 1000

type Store interface {
	Load(ctx context.Context) (
		channels []controldomain.Channel,
		capabilities []controldomain.Capability,
		audits []controldomain.CapabilityAudit,
		environments map[string]string,
		environmentAudits []controldomain.EnvironmentAudit,
		err error,
	)
	Save(
		ctx context.Context,
		channels []controldomain.Channel,
		capabilities []controldomain.Capability,
		audits []controldomain.CapabilityAudit,
		environments map[string]string,
		environmentAudits []controldomain.EnvironmentAudit,
	) error
}

type Service struct {
	mu                sync.RWMutex
	channels          map[string]controldomain.Channel
	capabilities      map[string]controldomain.Capability
	audits            []controldomain.CapabilityAudit
	environments      map[string]string
	environmentRuntime map[string]string
	environmentAudits []controldomain.EnvironmentAudit
	environmentDefs   map[string]controldomain.EnvironmentDefinition
	environmentOrder  []string
	store             Store
}

func NewService() *Service {
	return newService(nil)
}

func NewServiceWithStore(ctx context.Context, store Store) (*Service, error) {
	service := newService(store)
	if store == nil {
		return service, nil
	}

	channels, capabilities, audits, environments, environmentAudits, err := store.Load(ctx)
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

	for key, value := range environments {
		normalizedValue, ok := service.normalizeEnvironmentValue(key, value)
		if !ok {
			continue
		}
		service.environments[normalize(key)] = normalizedValue
	}
	service.audits = cloneAudits(audits)
	service.environmentAudits = cloneEnvironmentAudits(environmentAudits)
	return service, nil
}

func newService(store Store) *Service {
	definitions := defaultEnvironmentDefinitions()
	defMap := make(map[string]controldomain.EnvironmentDefinition, len(definitions))
	order := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		key := normalize(definition.Key)
		definition.Key = key
		defaultValue, err := definition.NormalizeValue(definition.DefaultValue)
		if err != nil {
			panic(fmt.Sprintf("invalid environment default for %s: %v", definition.Key, err))
		}
		definition.DefaultValue = defaultValue
		if definition.ApplyMode == "" {
			definition.ApplyMode = controldomain.EnvironmentApplyModeRestart
		}
		defMap[key] = definition
		order = append(order, key)
	}

	return &Service{
		channels:           map[string]controldomain.Channel{},
		capabilities:       map[string]controldomain.Capability{},
		audits:             []controldomain.CapabilityAudit{},
		environments:       map[string]string{},
		environmentRuntime: map[string]string{},
		environmentAudits:  []controldomain.EnvironmentAudit{},
		environmentDefs:    defMap,
		environmentOrder:   order,
		store:              store,
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

func (s *Service) ResolveEnvironmentString(key string, fallback string) string {
	normalizedKey := normalize(key)
	s.mu.RLock()
	definition, ok := s.environmentDefs[normalizedKey]
	value, hasValue := s.environments[normalizedKey]
	s.mu.RUnlock()
	if !ok {
		return strings.TrimSpace(fallback)
	}
	if hasValue {
		return value
	}
	if normalizedFallback, err := definition.NormalizeValue(fallback); err == nil {
		return normalizedFallback
	}
	return definition.DefaultValue
}

func (s *Service) ResolveEnvironmentInt(key string, fallback int) int {
	resolved := s.ResolveEnvironmentString(key, strconv.Itoa(fallback))
	value, err := strconv.Atoi(resolved)
	if err != nil {
		return fallback
	}
	return value
}

func (s *Service) ResolveEnvironmentDuration(key string, fallback time.Duration) time.Duration {
	resolved := s.ResolveEnvironmentString(key, fallback.String())
	value, err := time.ParseDuration(resolved)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func (s *Service) SetEnvironmentRuntime(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	runtimeValues := make(map[string]string, len(s.environmentOrder))
	for _, key := range s.environmentOrder {
		definition := s.environmentDefs[key]
		value := definition.DefaultValue
		if raw, ok := values[key]; ok {
			if normalizedValue, err := definition.NormalizeValue(raw); err == nil {
				value = normalizedValue
			}
		}
		runtimeValues[key] = value
	}
	s.environmentRuntime = runtimeValues
}

func (s *Service) ListEnvironmentConfigs(revealSensitive bool) []controldomain.EnvironmentConfigItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]controldomain.EnvironmentConfigItem, 0, len(s.environmentOrder))
	for _, key := range s.environmentOrder {
		definition := s.environmentDefs[key]
		desiredValue, source := s.desiredEnvironmentValueLocked(key)
		effectiveValue, hasEffective := s.environmentRuntime[key]
		if !hasEffective {
			effectiveValue = desiredValue
		}
		pendingRestart := definition.ApplyMode == controldomain.EnvironmentApplyModeRestart && hasEffective && effectiveValue != desiredValue
		masked := false
		renderedValue := desiredValue
		renderedEffective := effectiveValue
		if definition.Sensitive && !revealSensitive {
			renderedValue = controldomain.MaskEnvironmentValue(desiredValue)
			renderedEffective = controldomain.MaskEnvironmentValue(effectiveValue)
			masked = true
		}
		items = append(items, controldomain.EnvironmentConfigItem{
			Definition:     cloneEnvironmentDefinition(definition),
			Value:          renderedValue,
			EffectiveValue: renderedEffective,
			ValueSource:    source,
			PendingRestart: pendingRestart,
			Masked:         masked,
		})
	}
	return items
}

func (s *Service) UpdateEnvironmentConfigs(values map[string]string, operator string) (controldomain.EnvironmentUpdateResult, error) {
	if len(values) == 0 {
		return controldomain.EnvironmentUpdateResult{}, errors.New("environment values are required")
	}
	normalizedOperator := strings.TrimSpace(operator)
	if normalizedOperator == "" {
		normalizedOperator = "unknown"
	}

	normalizedValues := make(map[string]string, len(values))
	for key, value := range values {
		normalizedValues[normalize(key)] = value
	}
	keys := make([]string, 0, len(normalizedValues))
	for key := range normalizedValues {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	s.mu.Lock()
	defer s.mu.Unlock()

	previousValues := cloneEnvironmentValues(s.environments)
	previousAuditLen := len(s.environmentAudits)
	changes := make([]controldomain.EnvironmentAuditChange, 0, len(keys))

	for _, key := range keys {
		definition, ok := s.environmentDefs[key]
		if !ok {
			return controldomain.EnvironmentUpdateResult{}, fmt.Errorf("unsupported environment key %s", key)
		}

		normalizedValue, err := definition.NormalizeValue(normalizedValues[key])
		if err != nil {
			return controldomain.EnvironmentUpdateResult{}, err
		}
		currentValue, _ := s.desiredEnvironmentValueLocked(key)
		if currentValue == normalizedValue {
			continue
		}

		if normalizedValue == definition.DefaultValue {
			delete(s.environments, key)
		} else {
			s.environments[key] = normalizedValue
		}
		changes = append(changes, controldomain.EnvironmentAuditChange{
			Key:       key,
			OldValue:  currentValue,
			NewValue:  normalizedValue,
			ApplyMode: definition.ApplyMode,
		})
	}

	if len(changes) == 0 {
		return controldomain.EnvironmentUpdateResult{Changed: []controldomain.EnvironmentAuditChange{}}, nil
	}

	requiresRestart := false
	restartKeys := make([]string, 0, len(changes))
	for _, change := range changes {
		if change.ApplyMode != controldomain.EnvironmentApplyModeRestart {
			continue
		}
		requiresRestart = true
		restartKeys = append(restartKeys, change.Key)
	}

	s.appendEnvironmentAuditLocked(controldomain.EnvironmentAudit{
		Operator:        normalizedOperator,
		OccurredAt:      time.Now().UTC(),
		Changes:         cloneEnvironmentAuditChanges(changes),
		RequiresRestart: requiresRestart,
	})
	if err := s.storeLocked(); err != nil {
		s.environments = previousValues
		s.environmentAudits = s.environmentAudits[:previousAuditLen]
		return controldomain.EnvironmentUpdateResult{}, err
	}

	return controldomain.EnvironmentUpdateResult{
		Changed:      cloneEnvironmentAuditChanges(changes),
		NeedsRestart: requiresRestart,
		RestartKeys:  restartKeys,
	}, nil
}

func (s *Service) ListEnvironmentAudits(revealSensitive bool) []controldomain.EnvironmentAudit {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if revealSensitive {
		return cloneEnvironmentAudits(s.environmentAudits)
	}

	items := make([]controldomain.EnvironmentAudit, 0, len(s.environmentAudits))
	for _, audit := range s.environmentAudits {
		cloned := audit
		cloned.Changes = cloneEnvironmentAuditChanges(audit.Changes)
		for idx := range cloned.Changes {
			definition, ok := s.environmentDefs[normalize(cloned.Changes[idx].Key)]
			if !ok || !definition.Sensitive {
				continue
			}
			cloned.Changes[idx].OldValue = controldomain.MaskEnvironmentValue(cloned.Changes[idx].OldValue)
			cloned.Changes[idx].NewValue = controldomain.MaskEnvironmentValue(cloned.Changes[idx].NewValue)
		}
		items = append(items, cloned)
	}
	return items
}

func (s *Service) desiredEnvironmentValueLocked(key string) (string, string) {
	if value, ok := s.environments[key]; ok {
		return value, "persisted"
	}
	if value, ok := s.environmentRuntime[key]; ok {
		return value, "runtime"
	}
	definition := s.environmentDefs[key]
	return definition.DefaultValue, "default"
}

func (s *Service) normalizeEnvironmentValue(key string, raw string) (string, bool) {
	normalizedKey := normalize(key)
	definition, ok := s.environmentDefs[normalizedKey]
	if !ok {
		return "", false
	}
	normalizedValue, err := definition.NormalizeValue(raw)
	if err != nil {
		return "", false
	}
	if normalizedValue == definition.DefaultValue {
		return "", false
	}
	return normalizedValue, true
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

func (s *Service) appendEnvironmentAuditLocked(entry controldomain.EnvironmentAudit) {
	s.environmentAudits = append(s.environmentAudits, entry)
	if len(s.environmentAudits) > environmentAuditLimit {
		s.environmentAudits = cloneEnvironmentAudits(s.environmentAudits[len(s.environmentAudits)-environmentAuditLimit:])
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
	if err := s.store.Save(
		context.Background(),
		snapshotChannels(s.channels),
		snapshotCapabilities(s.capabilities),
		cloneAudits(s.audits),
		cloneEnvironmentValues(s.environments),
		cloneEnvironmentAudits(s.environmentAudits),
	); err != nil {
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

func cloneEnvironmentValues(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	copied := make(map[string]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}

func cloneEnvironmentAuditChanges(changes []controldomain.EnvironmentAuditChange) []controldomain.EnvironmentAuditChange {
	if len(changes) == 0 {
		return []controldomain.EnvironmentAuditChange{}
	}
	copied := make([]controldomain.EnvironmentAuditChange, 0, len(changes))
	for _, change := range changes {
		copied = append(copied, change)
	}
	return copied
}

func cloneEnvironmentAudits(items []controldomain.EnvironmentAudit) []controldomain.EnvironmentAudit {
	if len(items) == 0 {
		return []controldomain.EnvironmentAudit{}
	}
	copied := make([]controldomain.EnvironmentAudit, 0, len(items))
	for _, item := range items {
		cloned := item
		cloned.Changes = cloneEnvironmentAuditChanges(item.Changes)
		copied = append(copied, cloned)
	}
	return copied
}

func cloneEnvironmentDefinition(definition controldomain.EnvironmentDefinition) controldomain.EnvironmentDefinition {
	cloned := definition
	cloned.Validation.Allowed = append([]string(nil), definition.Validation.Allowed...)
	return cloned
}

package web

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	shareddomain "alter0/internal/shared/domain"
)

const (
	conversationRuntimeSessionRegistryFilename = "conversation-runtime-sessions.json"
	conversationRuntimeSessionStatusReady      = "ready"
	conversationRuntimeSessionStatusBusy       = "busy"
	conversationRuntimeSessionStatusFailed     = "failed"
)

type conversationRuntimeSessionRegistryEntry struct {
	SessionID       string                   `json:"session_id"`
	Route           conversationRuntimeRoute `json:"route"`
	Status          string                   `json:"status"`
	Title           string                   `json:"title"`
	TitleAuto       bool                     `json:"title_auto"`
	TitleScore      int                      `json:"title_score"`
	CreatedAt       time.Time                `json:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at"`
	TargetType      string                   `json:"target_type"`
	TargetID        string                   `json:"target_id,omitempty"`
	TargetName      string                   `json:"target_name,omitempty"`
	ModelProviderID string                   `json:"model_provider_id,omitempty"`
	ModelID         string                   `json:"model_id,omitempty"`
	ToolIDs         []string                 `json:"tool_ids,omitempty"`
	SkillIDs        []string                 `json:"skill_ids,omitempty"`
	MCPIDs          []string                 `json:"mcp_ids,omitempty"`
}

type conversationRuntimeSessionRegistry struct {
	path string

	mu      sync.RWMutex
	entries map[string]conversationRuntimeSessionRegistryEntry
}

func newFileConversationRuntimeSessionRegistry(path string) (*conversationRuntimeSessionRegistry, error) {
	registry := &conversationRuntimeSessionRegistry{
		path:    strings.TrimSpace(path),
		entries: map[string]conversationRuntimeSessionRegistryEntry{},
	}
	if registry.path == "" {
		return registry, nil
	}
	if err := registry.load(); err != nil {
		return nil, err
	}
	return registry, nil
}

func (r *conversationRuntimeSessionRegistry) load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	raw, err := os.ReadFile(r.path)
	if errors.Is(err, os.ErrNotExist) {
		r.entries = map[string]conversationRuntimeSessionRegistryEntry{}
		return nil
	}
	if err != nil {
		return err
	}

	var payload struct {
		Items []conversationRuntimeSessionRegistryEntry `json:"items"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}

	entries := make(map[string]conversationRuntimeSessionRegistryEntry, len(payload.Items))
	needsRewrite := false
	for _, item := range payload.Items {
		sessionID := strings.TrimSpace(item.SessionID)
		if sessionID == "" {
			continue
		}
		route, migrated, ok := parseStoredConversationRuntimeRoute(string(item.Route))
		if !ok {
			continue
		}
		if migrated {
			needsRewrite = true
		}
		item.SessionID = sessionID
		item.Route = route
		item.Status = normalizeConversationRuntimeSessionStatus(item.Status)
		item.TargetType, item.TargetID, item.TargetName = normalizeConversationRuntimeRegistryTarget(route, item.TargetType, item.TargetID, item.TargetName)
		item.ToolIDs = normalizeConversationRuntimeRegistryList(item.ToolIDs)
		item.SkillIDs = normalizeConversationRuntimeRegistryList(item.SkillIDs)
		item.MCPIDs = normalizeConversationRuntimeRegistryList(item.MCPIDs)
		entries[conversationRuntimeSessionRegistryKey(route, sessionID)] = item
	}
	r.entries = entries
	if needsRewrite {
		return r.persistLocked()
	}
	return nil
}

func (r *conversationRuntimeSessionRegistry) List(route conversationRuntimeRoute) []conversationRuntimeSessionRegistryEntry {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]conversationRuntimeSessionRegistryEntry, 0, len(r.entries))
	for _, item := range r.entries {
		if item.Route != route {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		left := items[i].UpdatedAt
		right := items[j].UpdatedAt
		if left.Equal(right) {
			return items[i].SessionID < items[j].SessionID
		}
		return left.After(right)
	})
	return items
}

func (r *conversationRuntimeSessionRegistry) Resolve(route conversationRuntimeRoute, sessionID string) (conversationRuntimeSessionRegistryEntry, bool) {
	if r == nil {
		return conversationRuntimeSessionRegistryEntry{}, false
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return conversationRuntimeSessionRegistryEntry{}, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.entries[conversationRuntimeSessionRegistryKey(route, sessionID)]
	return entry, ok
}

func (r *conversationRuntimeSessionRegistry) Upsert(entry conversationRuntimeSessionRegistryEntry) (conversationRuntimeSessionRegistryEntry, error) {
	if r == nil {
		return conversationRuntimeSessionRegistryEntry{}, errors.New("conversation runtime session registry unavailable")
	}
	sessionID := strings.TrimSpace(entry.SessionID)
	if sessionID == "" {
		return conversationRuntimeSessionRegistryEntry{}, errors.New("session_id is required")
	}
	route, ok := parseConversationRuntimeRoute(string(entry.Route))
	if !ok {
		return conversationRuntimeSessionRegistryEntry{}, errors.New("route is required")
	}
	now := time.Now().UTC()

	r.mu.Lock()
	defer r.mu.Unlock()

	key := conversationRuntimeSessionRegistryKey(route, sessionID)
	current, hadCurrent := r.entries[key]

	normalized := entry
	normalized.SessionID = sessionID
	normalized.Route = route
	normalized.Status = normalizeConversationRuntimeSessionStatus(entry.Status)
	normalized.TargetType, normalized.TargetID, normalized.TargetName = normalizeConversationRuntimeRegistryTarget(
		route,
		entry.TargetType,
		entry.TargetID,
		entry.TargetName,
	)
	normalized.ToolIDs = normalizeConversationRuntimeRegistryList(entry.ToolIDs)
	normalized.SkillIDs = normalizeConversationRuntimeRegistryList(entry.SkillIDs)
	normalized.MCPIDs = normalizeConversationRuntimeRegistryList(entry.MCPIDs)
	if normalized.CreatedAt.IsZero() {
		if hadCurrent && !current.CreatedAt.IsZero() {
			normalized.CreatedAt = current.CreatedAt
		} else {
			normalized.CreatedAt = now
		}
	}
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = now
	}
	if normalized.Title == "" {
		if hadCurrent && current.Title != "" {
			normalized.Title = current.Title
			normalized.TitleAuto = current.TitleAuto
			normalized.TitleScore = current.TitleScore
		} else {
			normalized.Title = "New"
		}
	}
	if normalized.ModelProviderID == "" && hadCurrent {
		normalized.ModelProviderID = current.ModelProviderID
	}
	if normalized.ModelID == "" && hadCurrent {
		normalized.ModelID = current.ModelID
	}
	if len(normalized.ToolIDs) == 0 && hadCurrent {
		normalized.ToolIDs = append([]string(nil), current.ToolIDs...)
	}
	if len(normalized.SkillIDs) == 0 && hadCurrent {
		normalized.SkillIDs = append([]string(nil), current.SkillIDs...)
	}
	if len(normalized.MCPIDs) == 0 && hadCurrent {
		normalized.MCPIDs = append([]string(nil), current.MCPIDs...)
	}
	if normalized.TargetID == "" && hadCurrent {
		normalized.TargetType = current.TargetType
		normalized.TargetID = current.TargetID
		normalized.TargetName = current.TargetName
	}

	r.entries[key] = normalized
	if err := r.persistLocked(); err != nil {
		if hadCurrent {
			r.entries[key] = current
		} else {
			delete(r.entries, key)
		}
		return conversationRuntimeSessionRegistryEntry{}, err
	}
	return normalized, nil
}

func (r *conversationRuntimeSessionRegistry) Delete(sessionID string) error {
	if r == nil {
		return nil
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return errors.New("session_id is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	key := conversationRuntimeSessionRegistryKey(conversationRuntimeRouteChat, sessionID)
	removed := false
	if _, ok := r.entries[key]; ok {
		delete(r.entries, key)
		removed = true
	}
	if !removed {
		return nil
	}
	return r.persistLocked()
}

func (r *conversationRuntimeSessionRegistry) persistLocked() error {
	if strings.TrimSpace(r.path) == "" {
		return nil
	}
	items := make([]conversationRuntimeSessionRegistryEntry, 0, len(r.entries))
	for _, item := range r.entries {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Route == items[j].Route {
			return items[i].SessionID < items[j].SessionID
		}
		return items[i].Route < items[j].Route
	})
	payload, err := json.MarshalIndent(struct {
		Items []conversationRuntimeSessionRegistryEntry `json:"items"`
	}{Items: items}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(r.path, payload, 0o644)
}

func conversationRuntimeSessionRegistryKey(route conversationRuntimeRoute, sessionID string) string {
	return string(route) + "\x00" + strings.TrimSpace(sessionID)
}

func normalizeConversationRuntimeSessionStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case conversationRuntimeSessionStatusBusy:
		return conversationRuntimeSessionStatusBusy
	case conversationRuntimeSessionStatusFailed:
		return conversationRuntimeSessionStatusFailed
	default:
		return conversationRuntimeSessionStatusReady
	}
}

func normalizeConversationRuntimeRegistryTarget(route conversationRuntimeRoute, targetType string, targetID string, targetName string) (string, string, string) {
	normalizedType := strings.TrimSpace(targetType)
	normalizedID := strings.TrimSpace(targetID)
	normalizedName := strings.TrimSpace(targetName)
	if normalizedType == "agent" {
		if normalizedID == "" {
			normalizedID = "unknown"
		}
		if normalizedName == "" {
			normalizedName = normalizedID
		}
		return "agent", normalizedID, normalizedName
	}
	if normalizedType != "model" {
		normalizedType = "model"
	}
	if normalizedID == "" {
		normalizedID = "raw-model"
	}
	if normalizedName == "" {
		normalizedName = "Raw Model"
	}
	return normalizedType, normalizedID, normalizedName
}

func normalizeConversationRuntimeRegistryList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, item := range values {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func buildConversationRuntimeRegistryEntryFromMessage(route conversationRuntimeRoute, msg shareddomain.UnifiedMessage, status string) conversationRuntimeSessionRegistryEntry {
	title := deriveConversationRuntimeTitleFromText(msg.Content)
	modelProviderID, modelID := resolveConversationRuntimeModelFromMetadata(msg.Metadata)
	targetType, targetID, targetName := resolveConversationRuntimeRegistryTargetFromMessage(route, msg.Metadata)
	toolIDs, skillIDs, mcpIDs := resolveConversationRuntimeCapabilitiesFromMetadata(msg.Metadata)
	createdAt := msg.ReceivedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	return conversationRuntimeSessionRegistryEntry{
		SessionID:       strings.TrimSpace(msg.SessionID),
		Route:           route,
		Status:          normalizeConversationRuntimeSessionStatus(status),
		Title:           title,
		TitleAuto:       false,
		TitleScore:      1,
		CreatedAt:       createdAt,
		UpdatedAt:       time.Now().UTC(),
		TargetType:      targetType,
		TargetID:        targetID,
		TargetName:      targetName,
		ModelProviderID: modelProviderID,
		ModelID:         modelID,
		ToolIDs:         toolIDs,
		SkillIDs:        skillIDs,
		MCPIDs:          mcpIDs,
	}
}

func resolveConversationRuntimeRegistryTargetFromMessage(route conversationRuntimeRoute, metadata map[string]string) (string, string, string) {
	return normalizeConversationRuntimeRegistryTarget(route, "model", "raw-model", "Raw Model")
}

func resolveConversationRuntimeModelFromMetadata(metadata map[string]string) (string, string) {
	if strings.TrimSpace(metadata[conversationRuntimeExecutionEngineMetadataKey]) == conversationRuntimeExecutionEngineCodex {
		return conversationRuntimeCodexProviderID, conversationRuntimeCodexModelID
	}
	return strings.TrimSpace(metadata[conversationRuntimeLLMProviderMetadataKey]), strings.TrimSpace(metadata[conversationRuntimeLLMModelMetadataKey])
}

func resolveConversationRuntimeCapabilitiesFromMetadata(metadata map[string]string) ([]string, []string, []string) {
	if len(metadata) == 0 {
		return nil, nil, nil
	}
	return normalizeConversationRuntimeIDs(metadata["alter0.agent.tools"]),
		normalizeConversationRuntimeIDs(metadata["alter0.skills.include"]),
		normalizeConversationRuntimeIDs(metadata["alter0.mcp.request.enable"])
}

func deriveConversationRuntimeTitleFromText(value string) string {
	title := strings.TrimSpace(value)
	if title == "" {
		return "New"
	}
	runes := []rune(title)
	if len(runes) > conversationRuntimeTitleMaxRunes {
		runes = runes[:conversationRuntimeTitleMaxRunes]
	}
	return string(runes)
}

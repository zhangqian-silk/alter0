package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	chatruntimeapp "alter0/internal/chatruntime/application"
)

const defaultSessionUpdatePollLimit = 50
const maxSessionUpdatePollLimit = 200
const defaultSessionUpdatePollBytes = 1024 * 1024
const maxSessionUpdatePollBytes = 1024 * 1024

type sessionUpdateEvent struct {
	EventID   int64          `json:"update_id"`
	OwnerID   string         `json:"-"`
	SessionID string         `json:"session_id,omitempty"`
	TurnID    string         `json:"turn_id,omitempty"`
	EventType string         `json:"type"`
	CreatedAt time.Time      `json:"-"`
	Payload   map[string]any `json:"payload,omitempty"`
}

type sessionUpdatePollEnvelope struct {
	LatestUpdateID int64                    `json:"latest_update_id"`
	ResyncRequired bool                     `json:"resync_required"`
	HasMore        bool                     `json:"has_more,omitempty"`
	Updates        []sessionUpdateAPIUpdate `json:"updates"`
}

type sessionUpdateAPIUpdate struct {
	UpdateID  int64          `json:"update_id"`
	Type      string         `json:"type"`
	SessionID string         `json:"session_id,omitempty"`
	TurnID    string         `json:"turn_id,omitempty"`
	CreatedAt int64          `json:"created_at"`
	Payload   map[string]any `json:"payload,omitempty"`
}

type sessionUpdatePollBody struct {
	AfterUpdateID     any                         `json:"after_update_id,omitempty"`
	Limit             int                         `json:"limit,omitempty"`
	ByteLimit         int                         `json:"byte_limit,omitempty"`
	Sessions          []sessionUpdateKnownSession `json:"sessions,omitempty"`
	VisibleEventKinds []string                    `json:"visible_event_kinds,omitempty"`
}

type sessionUpdateKnownSession struct {
	ID    string                   `json:"id"`
	Turns []sessionUpdateKnownTurn `json:"turns,omitempty"`
}

type sessionUpdateKnownTurn struct {
	ID       any   `json:"id"`
	EventIDs []any `json:"event_ids,omitempty"`
}

type sessionUpdateAckManifest struct {
	sessions map[string]map[string]sessionUpdateKnownTurn
}

type sessionUpdateBroker struct {
	mu          sync.Mutex
	nextEventID int64
	windowSize  int
	recent      []sessionUpdateEvent
}

func newSessionUpdateBroker(windowSize int) *sessionUpdateBroker {
	if windowSize <= 0 {
		windowSize = 128
	}
	return &sessionUpdateBroker{
		windowSize: windowSize,
	}
}

func (b *sessionUpdateBroker) publish(ownerID string, sessionID string, eventType string, payload map[string]any) sessionUpdateEvent {
	return b.publishWithTurnID(ownerID, sessionID, "", eventType, payload)
}

func (b *sessionUpdateBroker) publishWithTurnID(ownerID string, sessionID string, turnID string, eventType string, payload map[string]any) sessionUpdateEvent {
	if b == nil {
		return sessionUpdateEvent{}
	}
	ownerID = strings.TrimSpace(ownerID)
	eventType = strings.TrimSpace(eventType)
	if ownerID == "" || eventType == "" {
		return sessionUpdateEvent{}
	}
	b.mu.Lock()
	b.nextEventID++
	event := sessionUpdateEvent{
		EventID:   b.nextEventID,
		OwnerID:   ownerID,
		SessionID: strings.TrimSpace(sessionID),
		TurnID:    strings.TrimSpace(turnID),
		EventType: eventType,
		CreatedAt: time.Now().UTC(),
		Payload:   payload,
	}
	b.recent = append(b.recent, event)
	if len(b.recent) > b.windowSize {
		b.recent = append([]sessionUpdateEvent{}, b.recent[len(b.recent)-b.windowSize:]...)
	}
	b.mu.Unlock()
	return event
}

func (b *sessionUpdateBroker) poll(ownerID string, sinceEventID int64, limit int, byteLimit int) ([]sessionUpdateEvent, int64, bool, bool) {
	if b == nil {
		return nil, 0, false, false
	}
	ownerID = strings.TrimSpace(ownerID)
	if limit <= 0 {
		limit = defaultSessionUpdatePollLimit
	}
	if limit > maxSessionUpdatePollLimit {
		limit = maxSessionUpdatePollLimit
	}
	if byteLimit <= 0 {
		byteLimit = defaultSessionUpdatePollBytes
	}
	if byteLimit > maxSessionUpdatePollBytes {
		byteLimit = maxSessionUpdatePollBytes
	}
	b.mu.Lock()
	recent := make([]sessionUpdateEvent, 0)
	latestEventID := b.nextEventID
	oldestEventID := int64(0)
	oldestOwnerEventID := int64(0)
	for _, event := range b.recent {
		if oldestEventID == 0 || event.EventID < oldestEventID {
			oldestEventID = event.EventID
		}
		if event.OwnerID != ownerID {
			continue
		}
		if oldestOwnerEventID == 0 || event.EventID < oldestOwnerEventID {
			oldestOwnerEventID = event.EventID
		}
		if event.EventID > sinceEventID {
			recent = append(recent, event)
		}
	}
	resyncRequired := sinceEventID > 0 && (sinceEventID > latestEventID ||
		(oldestEventID > 0 && sinceEventID < oldestEventID) ||
		(oldestOwnerEventID > 0 && sinceEventID < oldestOwnerEventID))
	b.mu.Unlock()
	if resyncRequired {
		return nil, latestEventID, true, false
	}
	out := make([]sessionUpdateEvent, 0, len(recent))
	approxBytes := 0
	hasMore := false
	for _, event := range recent {
		if len(out) >= limit {
			hasMore = true
			break
		}
		eventBytes := approximateSessionUpdateEventBytes(event)
		if len(out) > 0 && approxBytes+eventBytes > byteLimit {
			hasMore = true
			break
		}
		out = append(out, event)
		approxBytes += eventBytes
	}
	cursor := latestEventID
	if hasMore && len(out) > 0 {
		cursor = out[len(out)-1].EventID
	}
	return out, cursor, false, hasMore
}

func approximateSessionUpdateEventBytes(event sessionUpdateEvent) int {
	raw, err := json.Marshal(event)
	if err != nil {
		return len(event.OwnerID) + len(event.SessionID) + len(event.EventType) + 64
	}
	return len(raw)
}

func (s *Server) sessionUpdateBroker() *sessionUpdateBroker {
	if s.sessionEvents != nil {
		return s.sessionEvents
	}
	s.sessionEvents = newSessionUpdateBroker(256)
	return s.sessionEvents
}

func (s *Server) registerChatRuntimeSessionEventHook() {
	if s == nil {
		return
	}
	if setter, ok := s.chatRuntimes.(chatRuntimeSessionEventHookSetter); ok {
		setter.SetSessionEventHook(func(event chatruntimeapp.SessionEvent) {
			s.publishChatRuntimeSessionTypedEvent(event)
		})
	}
}

func (s *Server) publishChatRuntimeSessionTypedEvent(event chatruntimeapp.SessionEvent) {
	ownerID := strings.TrimSpace(event.OwnerID)
	sessionID := strings.TrimSpace(event.SessionID)
	eventType := strings.TrimSpace(event.EventType)
	if ownerID == "" || sessionID == "" || eventType == "" {
		return
	}
	turnID := ""
	if event.Turn != nil {
		turnID = strings.TrimSpace(event.Turn.ID)
	}
	if turnID == "" && event.RuntimeEvent != nil {
		turnID = strings.TrimSpace(event.RuntimeEvent.TurnID)
	}
	s.sessionUpdateBroker().publishWithTurnID(ownerID, sessionID, turnID, eventType, buildChatRuntimeSessionTypedEventPayload(event))
}

func buildChatRuntimeSessionTypedEventPayload(event chatruntimeapp.SessionEvent) map[string]any {
	session := buildChatRuntimeSessionUpdateSummary(event.Session)
	if session == nil {
		return nil
	}
	payload := map[string]any{"session": session}
	if event.Turn != nil {
		turn := buildChatRuntimeTurnUpdatePatch(*event.Turn)
		payload["turn"] = turn
	}
	if event.RuntimeEvent != nil {
		payload["runtime_event"] = buildChatRuntimeEventDTO(*event.RuntimeEvent, 1)
	}
	return payload
}

func buildChatRuntimeSessionUpdateSummary(session any) map[string]any {
	sessionMap, ok := chatRuntimeSessionMap(session)
	if !ok {
		return nil
	}
	applyChatRuntimeSessionAPIFields(sessionMap)
	trimChatRuntimeSessionUpdateMetadata(sessionMap)
	delete(sessionMap, "turns")
	delete(sessionMap, "turns_paging")
	return sessionMap
}

func buildChatRuntimeTurnUpdatePatch(turn chatruntimeapp.TurnSummary) map[string]any {
	patch := buildChatRuntimeTurnDTO(turn, numericChatRuntimeID(turn.ID, "turn", 1))
	delete(patch, "runtime_trace_events")
	return patch
}

func trimChatRuntimeSessionUpdateMetadata(sessionMap map[string]any) {
	delete(sessionMap, "owner_id")
	delete(sessionMap, "shell")
	delete(sessionMap, "working_dir")
	delete(sessionMap, "runtime_session_id")
	delete(sessionMap, "finished_at")
}

func (s *Server) publishChatRuntimeSessionSummaryEvent(ownerID string, sessionID string, eventType string, session any) {
	payload := map[string]any{}
	if session != nil {
		if summary := buildChatRuntimeSessionUpdateSummary(session); summary != nil {
			payload["session"] = summary
		}
	}
	s.sessionUpdateBroker().publish(ownerID, sessionID, eventType, payload)
}

func (s *Server) chatSessionUpdatesHandler(w http.ResponseWriter, r *http.Request) {
	s.chatRuntimeSessionUpdatesHandler(w, withChatRuntimeClientID(r, chatSessionOwnerID))
}

func (s *Server) chatRuntimeSessionUpdatesHandler(w http.ResponseWriter, r *http.Request) {
	disableConversationHTTPResponseCaching(w)
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	ownerID := resolveChatRuntimeClientID(r)
	if ownerID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":      "chatRuntime client id is required",
			"error_code": "chatRuntime_client_required",
		})
		return
	}
	defer r.Body.Close()
	var body sessionUpdatePollBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	var since int64
	if parsedSince, ok := flexibleInt64(body.AfterUpdateID); ok {
		since = parsedSince
	}
	limit := body.Limit
	byteLimit := body.ByteLimit
	manifest := newSessionUpdateAckManifest(body.Sessions)
	events, cursor, resyncRequired, hasMore := s.sessionUpdateBroker().poll(ownerID, since, limit, byteLimit)
	events = pruneSessionUpdateEvents(events, manifest, visibleRuntimeEventCategories(body.VisibleEventKinds))
	writeJSON(w, http.StatusOK, sessionUpdatePollEnvelope{
		LatestUpdateID: cursor,
		ResyncRequired: resyncRequired,
		HasMore:        hasMore,
		Updates:        buildSessionUpdateAPIUpdates(events),
	})
}

func buildSessionUpdateAPIUpdates(events []sessionUpdateEvent) []sessionUpdateAPIUpdate {
	if len(events) == 0 {
		return []sessionUpdateAPIUpdate{}
	}
	updates := make([]sessionUpdateAPIUpdate, 0, len(events))
	for _, event := range events {
		update := sessionUpdateAPIUpdate{
			UpdateID:  event.EventID,
			Type:      event.EventType,
			SessionID: event.SessionID,
			TurnID:    canonicalChatRuntimeID(event.TurnID, "turn", 0),
			CreatedAt: unixMillis(event.CreatedAt),
			Payload:   event.Payload,
		}
		updates = append(updates, update)
	}
	return updates
}

func flexibleInt64(value any) (int64, bool) {
	switch typed := value.(type) {
	case nil:
		return 0, false
	case int:
		return int64(typed), true
	case int64:
		return typed, true
	case float64:
		return int64(typed), true
	case json.Number:
		parsed, err := typed.Int64()
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func newSessionUpdateAckManifest(items []sessionUpdateKnownSession) sessionUpdateAckManifest {
	manifest := sessionUpdateAckManifest{sessions: map[string]map[string]sessionUpdateKnownTurn{}}
	for _, session := range items {
		sessionID := strings.TrimSpace(session.ID)
		if sessionID == "" {
			continue
		}
		turns := manifest.sessions[sessionID]
		if turns == nil {
			turns = map[string]sessionUpdateKnownTurn{}
			manifest.sessions[sessionID] = turns
		}
		for _, turn := range session.Turns {
			turnID := strings.TrimSpace(fmtAny(turn.ID))
			if turnID == "" {
				continue
			}
			turn.ID = turnID
			turn.EventIDs = compactCanonicalRuntimeIDSet(turn.EventIDs, "event")
			turns[turnID] = turn
		}
	}
	return manifest
}

func compactCanonicalRuntimeIDSet(values []any, prefix string) []any {
	seen := map[string]struct{}{}
	out := make([]any, 0, len(values))
	for _, value := range values {
		normalized := canonicalChatRuntimeID(fmtAny(value), prefix, 0)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func pruneSessionUpdateEvents(
	events []sessionUpdateEvent,
	manifest sessionUpdateAckManifest,
	visibleCategories map[string]struct{},
) []sessionUpdateEvent {
	if len(events) == 0 {
		return events
	}
	out := make([]sessionUpdateEvent, 0, len(events))
	for _, event := range events {
		payload := cloneSessionUpdatePayload(event.Payload)
		if len(payload) > 0 {
			if runtimeEvent, ok := mapFromAny(payload["runtime_event"]); ok {
				if len(visibleCategories) > 0 && !runtimeTraceEventVisible(runtimeEvent, visibleCategories) {
					continue
				}
				if typedRuntimeTraceEventKnown(event, payload, runtimeEvent, manifest) {
					continue
				}
				payload["runtime_event"] = runtimeEvent
			}
			event.Payload = payload
		}
		out = append(out, event)
	}
	return out
}

func cloneSessionUpdatePayload(payload map[string]any) map[string]any {
	if len(payload) == 0 {
		return payload
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return payload
	}
	var cloned map[string]any
	if err := json.Unmarshal(raw, &cloned); err != nil {
		return payload
	}
	return cloned
}

func mapFromAny(value any) (map[string]any, bool) {
	if value == nil {
		return nil, false
	}
	if typed, ok := value.(map[string]any); ok {
		return typed, true
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, false
	}
	var mapped map[string]any
	if err := json.Unmarshal(raw, &mapped); err != nil {
		return nil, false
	}
	return mapped, true
}

func typedRuntimeTraceEventKnown(
	update sessionUpdateEvent,
	payload map[string]any,
	runtimeEvent map[string]any,
	manifest sessionUpdateAckManifest,
) bool {
	sessionID := strings.TrimSpace(update.SessionID)
	if sessionID == "" {
		if session, ok := mapFromAny(payload["session"]); ok {
			sessionID = strings.TrimSpace(fmtAny(session["id"]))
		}
	}
	turnID := strings.TrimSpace(update.TurnID)
	if turnID == "" {
		if turn, ok := mapFromAny(payload["turn"]); ok {
			turnID = strings.TrimSpace(fmtAny(turn["id"]))
		}
	}
	if turnID == "" {
		turnID = strings.TrimSpace(fmtAny(runtimeEvent["turn_id"]))
	}
	known := knownManifestTurn(manifest, sessionID, turnID)
	idSet := knownRuntimeTraceEventIDSet(known.EventIDs)
	return runtimeTraceEventKnown(runtimeEvent, idSet)
}

func knownManifestTurn(manifest sessionUpdateAckManifest, sessionID string, turnID string) sessionUpdateKnownTurn {
	turns := manifest.sessions[strings.TrimSpace(sessionID)]
	if len(turns) == 0 {
		return sessionUpdateKnownTurn{}
	}
	normalized := strings.TrimSpace(turnID)
	if known, ok := turns[normalized]; ok {
		return known
	}
	if numeric := numericChatRuntimeID(normalized, "turn", 0); numeric > 0 {
		if known, ok := turns[strconv.Itoa(numeric)]; ok {
			return known
		}
	}
	return sessionUpdateKnownTurn{}
}

func visibleRuntimeEventCategories(values []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, value := range values {
		category := strings.TrimSpace(strings.ToLower(value))
		if category == "" {
			continue
		}
		out[category] = struct{}{}
	}
	return out
}

func runtimeTraceEventVisible(event map[string]any, visibleCategories map[string]struct{}) bool {
	category := runtimeTraceEventCategory(event)
	if category == "" {
		return true
	}
	_, ok := visibleCategories[category]
	return ok
}

func runtimeTraceEventCategory(event map[string]any) string {
	kind := strings.TrimSpace(strings.ToLower(fmtAny(event["kind"])))
	switch kind {
	case "assistant_commentary", "analysis", "commentary", "important_text", "message", "text":
		return "important_text"
	case "plan":
		return "plan"
	case "reasoning", "thinking":
		return "reasoning"
	case "shell_command", "command", "command_execution":
		return "commands"
	case "tool_call", "tool_result", "file_read", "file_write", "file_edit", "web_search", "web_fetch", "mcp_call", "skill_context", "skill_use", "hook_event", "approval_request", "subagent_start", "subagent_progress", "subagent_result":
		return "tools"
	case "system_event", "log", "error", "rate_limit", "unknown_provider_event":
		return "system"
	default:
		return "system"
	}
}

func knownRuntimeTraceEventIDSet(ids []any) map[string]struct{} {
	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		normalized := canonicalChatRuntimeID(fmtAny(id), "event", 0)
		if normalized == "" {
			continue
		}
		idSet[normalized] = struct{}{}
	}
	return idSet
}

func runtimeTraceEventKnown(event map[string]any, idSet map[string]struct{}) bool {
	eventID := canonicalChatRuntimeID(fmtAny(event["id"]), "event", 0)
	if eventID == "" {
		return false
	}
	_, known := idSet[eventID]
	return known
}

func flexibleInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case json.Number:
		parsed, err := typed.Int64()
		return int(parsed), err == nil
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		return parsed, err == nil
	default:
		return 0, false
	}
}

func fmtAny(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(strings.Trim(fmt.Sprint(value), "\""))
}

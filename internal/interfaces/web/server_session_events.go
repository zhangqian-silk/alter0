package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	chatruntimedomain "alter0/internal/chatruntime/domain"
)

const defaultSessionUpdatePollLimit = 50
const maxSessionUpdatePollLimit = 200
const defaultSessionUpdatePollBytes = 64 * 1024
const maxSessionUpdatePollBytes = 256 * 1024

type sessionUpdateEvent struct {
	EventID   int64          `json:"event_id"`
	OwnerID   string         `json:"owner_id"`
	SessionID string         `json:"session_id,omitempty"`
	EventType string         `json:"event_type"`
	Revision  int64          `json:"revision,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	Payload   map[string]any `json:"payload,omitempty"`
}

type sessionUpdatePollEnvelope struct {
	OwnerID        string               `json:"owner_id"`
	Cursor         int64                `json:"cursor"`
	ResyncRequired bool                 `json:"resync_required"`
	HasMore        bool                 `json:"has_more,omitempty"`
	Events         []sessionUpdateEvent `json:"events"`
}

type sessionUpdatePollBody struct {
	SinceEventID any                         `json:"since_event_id,omitempty"`
	Limit        int                         `json:"limit,omitempty"`
	ByteLimit    int                         `json:"byte_limit,omitempty"`
	Sessions     []sessionUpdateKnownSession `json:"sessions,omitempty"`
}

type sessionUpdateKnownSession struct {
	ID    string                   `json:"id"`
	Turns []sessionUpdateKnownTurn `json:"turns,omitempty"`
}

type sessionUpdateKnownTurn struct {
	ID             string   `json:"id"`
	EventIDs       []string `json:"event_ids,omitempty"`
	EventSeqRanges [][]int  `json:"event_seq_ranges,omitempty"`
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
		EventType: eventType,
		Revision:  b.nextEventID,
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

func (s *Server) registerChatRuntimeSessionUpdateHook() {
	if s == nil {
		return
	}
	setter, ok := s.chatRuntimes.(chatRuntimeSessionUpdateHookSetter)
	if !ok {
		return
	}
	setter.SetSessionUpdateHook(func(ownerID string, sessionID string, session chatruntimedomain.Session) {
		s.sessionUpdateBroker().publish(ownerID, sessionID, "session.updated", map[string]any{
			"session": s.buildChatRuntimeSessionEventDetail(ownerID, session),
		})
	})
}

func (s *Server) buildChatRuntimeSessionEventDetail(ownerID string, session chatruntimedomain.Session) any {
	return s.buildChatRuntimeSessionDetail(ownerID, session, &http.Request{
		URL: &url.URL{RawQuery: "turn_limit=1"},
	})
}

func (s *Server) chatRuntimeServicePublishesSessionEvents() bool {
	_, ok := s.chatRuntimes.(chatRuntimeSessionUpdateHookSetter)
	return ok
}

func (s *Server) publishChatRuntimeSessionEvent(ownerID string, sessionID string, eventType string, session any) {
	if eventType == "session.updated" && s.chatRuntimeServicePublishesSessionEvents() {
		return
	}
	payload := map[string]any{}
	if session != nil {
		payload["session"] = session
	}
	s.sessionUpdateBroker().publish(ownerID, sessionID, eventType, payload)
}

func (s *Server) chatSessionUpdatesHandler(w http.ResponseWriter, r *http.Request) {
	s.chatRuntimeSessionUpdatesHandler(w, withChatRuntimeClientID(r, chatSessionOwnerID))
}

func (s *Server) chatRuntimeSessionUpdatesHandler(w http.ResponseWriter, r *http.Request) {
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
	if parsedSince, ok := flexibleInt64(body.SinceEventID); ok {
		since = parsedSince
	}
	limit := body.Limit
	byteLimit := body.ByteLimit
	manifest := newSessionUpdateAckManifest(body.Sessions)
	events, cursor, resyncRequired, hasMore := s.sessionUpdateBroker().poll(ownerID, since, limit, byteLimit)
	events = pruneSessionUpdateEvents(events, manifest)
	writeJSON(w, http.StatusOK, sessionUpdatePollEnvelope{
		OwnerID:        ownerID,
		Cursor:         cursor,
		ResyncRequired: resyncRequired,
		HasMore:        hasMore,
		Events:         events,
	})
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
			turnID := strings.TrimSpace(turn.ID)
			if turnID == "" {
				continue
			}
			turn.ID = turnID
			turn.EventIDs = compactStringSet(turn.EventIDs)
			turn.EventSeqRanges = normalizeSeqRanges(turn.EventSeqRanges)
			turns[turnID] = turn
		}
	}
	return manifest
}

func compactStringSet(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
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

func normalizeSeqRanges(ranges [][]int) [][]int {
	out := make([][]int, 0, len(ranges))
	for _, item := range ranges {
		if len(item) < 2 {
			continue
		}
		start := item[0]
		end := item[1]
		if start <= 0 || end <= 0 {
			continue
		}
		if end < start {
			start, end = end, start
		}
		out = append(out, []int{start, end})
	}
	return out
}

func pruneSessionUpdateEvents(events []sessionUpdateEvent, manifest sessionUpdateAckManifest) []sessionUpdateEvent {
	if len(events) == 0 || len(manifest.sessions) == 0 {
		return events
	}
	out := make([]sessionUpdateEvent, 0, len(events))
	for _, event := range events {
		payload := cloneSessionUpdatePayload(event.Payload)
		if len(payload) > 0 {
			if session, ok := mapFromAny(payload["session"]); ok {
				pruneSessionPayloadRuntimeTraceEvents(session, manifest)
				payload["session"] = session
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

func pruneSessionPayloadRuntimeTraceEvents(session map[string]any, manifest sessionUpdateAckManifest) {
	sessionID := strings.TrimSpace(fmtAny(session["id"]))
	turnsByID := manifest.sessions[sessionID]
	if len(turnsByID) == 0 {
		return
	}
	turns, ok := session["turns"].([]any)
	if !ok {
		return
	}
	for _, item := range turns {
		turn, ok := mapFromAny(item)
		if !ok {
			continue
		}
		turnID := strings.TrimSpace(fmtAny(turn["id"]))
		known, ok := turnsByID[turnID]
		if !ok {
			continue
		}
		pruneTurnRuntimeTraceEvents(turn, known)
	}
}

func pruneTurnRuntimeTraceEvents(turn map[string]any, known sessionUpdateKnownTurn) {
	events, ok := turn["runtime_trace_events"].([]any)
	if !ok || len(events) == 0 {
		return
	}
	idSet := knownRuntimeTraceEventIDSet(known.EventIDs)
	filtered := make([]any, 0, len(events))
	for _, item := range events {
		event, ok := mapFromAny(item)
		if !ok {
			filtered = append(filtered, item)
			continue
		}
		if runtimeTraceEventKnown(event, idSet, known.EventSeqRanges) {
			continue
		}
		filtered = append(filtered, event)
	}
	turn["runtime_trace_events"] = filtered
	if len(filtered) != len(events) {
		turn["runtime_trace_events_partial"] = true
	}
}

func knownRuntimeTraceEventIDSet(ids []string) map[string]struct{} {
	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		normalized := strings.TrimSpace(id)
		if normalized == "" {
			continue
		}
		idSet[normalized] = struct{}{}
	}
	return idSet
}

func runtimeTraceEventKnown(event map[string]any, idSet map[string]struct{}, ranges [][]int) bool {
	eventID := strings.TrimSpace(fmtAny(event["id"]))
	if eventID != "" {
		if _, ok := idSet[eventID]; ok {
			return true
		}
	}
	eventSeq, hasSeq := flexibleInt(event["seq"])
	return hasSeq && seqInRanges(eventSeq, ranges)
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

func seqInRanges(seq int, ranges [][]int) bool {
	for _, item := range ranges {
		if len(item) >= 2 && seq >= item[0] && seq <= item[1] {
			return true
		}
	}
	return false
}

func fmtAny(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(strings.Trim(fmt.Sprint(value), "\""))
}

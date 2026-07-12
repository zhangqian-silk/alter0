package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	chatruntimeapp "alter0/internal/chatruntime/application"
	chatruntimedomain "alter0/internal/chatruntime/domain"
)

const chatSessionOwnerID = "chat"
const defaultChatRuntimeTurnDetailLimit = 20
const maxChatRuntimeTurnDetailLimit = 160
const maxChatRuntimeTurnDetailBytes = 1024 * 1024

type chatRuntimeClientIDContextKey struct{}

type chatRuntimeSessionCreateRequest struct {
	Title string `json:"title,omitempty"`
}

type chatRuntimeSessionInputRequest struct {
	Input           string                           `json:"input"`
	ClientRequestID string                           `json:"client_request_id,omitempty"`
	Attachments     []messageAttachmentRequest       `json:"attachments,omitempty"`
	Repository      *chatruntimedomain.RepositoryRef `json:"repository,omitempty"`
}

type chatRuntimeSessionPinRequest struct {
	Pinned *bool `json:"pinned"`
}

type chatRuntimeSessionRecoverRequest struct {
	ID               string    `json:"id"`
	RuntimeSessionID string    `json:"runtime_session_id,omitempty"`
	Title            string    `json:"title,omitempty"`
	CreatedAt        time.Time `json:"created_at,omitempty"`
	LastOutputAt     time.Time `json:"last_output_at,omitempty"`
	UpdatedAt        time.Time `json:"updated_at,omitempty"`
}

type chatRuntimeSessionEnvelope struct {
	Session any `json:"session"`
}

type chatRuntimeSessionListEnvelope struct {
	Items []any `json:"items"`
}

type chatRuntimeTurnPagingEnvelope struct {
	Limit            int    `json:"limit"`
	Total            int    `json:"total"`
	ByteLimit        int    `json:"byte_limit"`
	ApproxBytes      int    `json:"approx_bytes"`
	HasMoreBefore    bool   `json:"has_more_before"`
	HasMoreAfter     bool   `json:"has_more_after,omitempty"`
	OldestTurnID     string `json:"oldest_turn_id,omitempty"`
	NewestTurnID     string `json:"newest_turn_id,omitempty"`
	NextBeforeTurnID string `json:"next_before_turn_id,omitempty"`
	BeforeTurnFound  bool   `json:"before_turn_found"`
}

type chatRuntimeEventDetailEnvelope struct {
	Event  map[string]any                `json:"event"`
	Blocks []chatruntimeapp.RuntimeBlock `json:"blocks,omitempty"`
}

func disableConversationHTTPResponseCaching(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
}

func (s *Server) chatRepositoryCollectionHandler(w http.ResponseWriter, r *http.Request) {
	disableConversationHTTPResponseCaching(w)
	if s.chatRuntimes == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error":      "repository service is unavailable",
			"error_code": "repository_unavailable",
		})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	page, err := s.chatRuntimes.ListRepositories(r.Context(), r.URL.Query().Get("query"), r.URL.Query().Get("cursor"))
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error":      "GitHub repositories are unavailable. Check the server GitHub login and retry.",
			"error_code": "repository_unavailable",
		})
		return
	}
	items := make([]map[string]any, 0, len(page.Items))
	for _, item := range page.Items {
		repository := map[string]any{
			"id":             strings.TrimSpace(item.ID),
			"full_name":      strings.TrimSpace(item.FullName),
			"private":        item.Private,
			"default_branch": strings.TrimSpace(item.DefaultBranch),
		}
		if !item.UpdatedAt.IsZero() {
			repository["updated_at"] = item.UpdatedAt.UTC()
		}
		items = append(items, repository)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"repositories": items,
		"next_cursor":  strings.TrimSpace(page.NextCursor),
	})
}

func (s *Server) chatRuntimeSessionCollectionHandler(w http.ResponseWriter, r *http.Request) {
	disableConversationHTTPResponseCaching(w)
	if s.chatRuntimes == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "chatRuntime service unavailable"})
		return
	}

	ownerID := resolveChatRuntimeClientID(r)
	switch r.Method {
	case http.MethodGet:
		items := s.chatRuntimes.List(ownerID)
		summaries := make([]any, 0, len(items))
		for _, item := range items {
			summaries = append(summaries, buildChatRuntimeSessionSummary(item))
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": summaries})
	case http.MethodPost:
		defer r.Body.Close()
		if ownerID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":      "chatRuntime client id is required",
				"error_code": "chatRuntime_client_required",
			})
			return
		}
		var req chatRuntimeSessionCreateRequest
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&req)
		}
		session, err := s.chatRuntimes.Create(chatruntimeapp.CreateRequest{
			OwnerID: ownerID,
			Title:   strings.TrimSpace(req.Title),
		})
		if err != nil {
			s.writeChatRuntimeError(w, err)
			return
		}
		s.publishChatRuntimeSessionSummaryEvent(ownerID, session.ID, "session.created", session)
		writeJSON(w, http.StatusCreated, map[string]any{"session": s.buildChatRuntimeSessionDetail(ownerID, session, r)})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) chatRuntimeSessionRecoverHandler(w http.ResponseWriter, r *http.Request) {
	disableConversationHTTPResponseCaching(w)
	if s.chatRuntimes == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "chatRuntime service unavailable"})
		return
	}
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
	var req chatRuntimeSessionRecoverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}

	session, err := s.chatRuntimes.Recover(chatruntimeapp.RecoverRequest{
		OwnerID:          ownerID,
		SessionID:        strings.TrimSpace(req.ID),
		RuntimeSessionID: strings.TrimSpace(req.RuntimeSessionID),
		Title:            strings.TrimSpace(req.Title),
		CreatedAt:        req.CreatedAt,
		LastOutputAt:     req.LastOutputAt,
		UpdatedAt:        req.UpdatedAt,
	})
	if err != nil {
		s.writeChatRuntimeError(w, err)
		return
	}
	s.publishChatRuntimeSessionSummaryEvent(ownerID, session.ID, "session.updated", session)
	writeJSON(w, http.StatusOK, map[string]any{"session": s.buildChatRuntimeSessionDetail(ownerID, session, r)})
}

func (s *Server) chatSessionCollectionHandler(w http.ResponseWriter, r *http.Request) {
	s.chatRuntimeSessionCollectionHandler(w, withChatRuntimeClientID(r, chatSessionOwnerID))
}

func (s *Server) chatSessionRecoverHandler(w http.ResponseWriter, r *http.Request) {
	s.chatRuntimeSessionRecoverHandler(w, withChatRuntimeClientID(r, chatSessionOwnerID))
}

func (s *Server) chatSessionItemHandler(w http.ResponseWriter, r *http.Request) {
	next := withAttachmentRoutePrefix(withChatRuntimeClientID(r, chatSessionOwnerID), "/api/chat/sessions")
	s.chatRuntimeSessionItemHandler(w, next)
}

func withChatRuntimeClientID(r *http.Request, clientID string) *http.Request {
	if r == nil {
		return r
	}
	ctx := context.WithValue(r.Context(), chatRuntimeClientIDContextKey{}, strings.TrimSpace(clientID))
	return r.WithContext(ctx)
}

func (s *Server) chatRuntimeSessionItemHandler(w http.ResponseWriter, r *http.Request) {
	disableConversationHTTPResponseCaching(w)
	if s.chatRuntimes == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "chatRuntime service unavailable"})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/chat/sessions/")
	if path == r.URL.Path {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
		return
	}
	path = strings.Trim(path, "/")
	if path == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
		return
	}

	parts := strings.Split(path, "/")
	sessionID := strings.TrimSpace(parts[0])
	ownerID := resolveChatRuntimeClientID(r)
	if len(parts) == 1 {
		if r.Method == http.MethodDelete {
			session, err := s.chatRuntimes.Delete(ownerID, sessionID)
			if err != nil {
				s.writeChatRuntimeError(w, err)
				return
			}
			s.publishChatRuntimeSessionSummaryEvent(ownerID, sessionID, "session.deleted", session)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		session, ok := s.chatRuntimes.Get(ownerID, sessionID)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error":      "chatRuntime session not found",
				"error_code": "chatRuntime_session_not_found",
			})
			return
		}
		s.touchSessionActivity(sessionID)
		writeJSON(w, http.StatusOK, map[string]any{"session": s.buildChatRuntimeSessionDetail(ownerID, session, r)})
		return
	}

	switch parts[1] {
	case "pin":
		if len(parts) != 2 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "session action not found"})
			return
		}
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		defer r.Body.Close()
		var req chatRuntimeSessionPinRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		if req.Pinned == nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "pinned is required"})
			return
		}
		session, err := s.chatRuntimes.SetPinned(ownerID, sessionID, *req.Pinned)
		if err != nil {
			s.writeChatRuntimeError(w, err)
			return
		}
		s.publishChatRuntimeSessionSummaryEvent(ownerID, session.ID, "session.updated", session)
		writeJSON(w, http.StatusOK, map[string]any{"session": s.buildChatRuntimeSessionDetail(ownerID, session, r)})
	case "turns":
		if len(parts) == 2 {
			if r.Method != http.MethodGet {
				writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
				return
			}
			items, err := s.chatRuntimes.ListTurns(ownerID, sessionID)
			if err != nil {
				s.writeChatRuntimeError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"items": buildChatRuntimeTurnDTOs(items)})
			return
		}
		if len(parts) == 5 && parts[3] == "events" {
			if r.Method != http.MethodGet {
				writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
				return
			}
			detail, err := s.chatRuntimes.GetRuntimeTraceEventDetail(ownerID, sessionID, strings.TrimSpace(parts[2]), strings.TrimSpace(parts[4]))
			if err != nil {
				s.writeChatRuntimeError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, buildChatRuntimeEventDetailDTO(detail))
			return
		}
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session action not found"})
	case "entries":
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		cursor, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("cursor")))
		limit, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("limit")))
		page, err := s.chatRuntimes.ListEntries(ownerID, sessionID, cursor, limit)
		if err != nil {
			s.writeChatRuntimeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, page)
	case "attachments":
		switch {
		case len(parts) == 2:
			s.handleSessionAttachmentUpload(w, r, sessionID)
		case len(parts) == 4 && (parts[3] == "original" || parts[3] == "preview"):
			s.handleSessionAttachmentRead(w, r, sessionID, strings.TrimSpace(parts[2]), strings.TrimSpace(parts[3]))
		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid session path"})
		}
	case "repository":
		if len(parts) != 3 || parts[2] != "retry" {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "session action not found"})
			return
		}
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		session, err := s.chatRuntimes.RetryRepository(ownerID, sessionID)
		if err != nil {
			s.writeChatRuntimeError(w, err)
			return
		}
		s.touchSessionActivity(sessionID)
		writeJSON(w, http.StatusOK, map[string]any{"session": s.buildChatRuntimeSessionDetail(ownerID, session, r)})
	case "input":
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		defer r.Body.Close()
		var req chatRuntimeSessionInputRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		attachments, err := s.normalizeConversationMessageAttachments(sessionID, req.Attachments, attachmentRoutePrefixFromRequest(r))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		input := strings.TrimSpace(req.Input)
		if input == "" && len(attachments) > 0 {
			input = defaultAttachmentContent(attachments)
		}
		session, err := s.chatRuntimes.InputWithAttachments(chatruntimeapp.InputRequest{
			OwnerID:         ownerID,
			SessionID:       sessionID,
			Input:           input,
			ClientRequestID: strings.TrimSpace(req.ClientRequestID),
			Attachments:     attachments,
			Repository:      req.Repository,
		})
		if err != nil {
			s.writeChatRuntimeError(w, err)
			return
		}
		s.touchSessionActivity(sessionID)
		writeJSON(w, http.StatusOK, map[string]any{"session": s.buildChatRuntimeSessionDetail(ownerID, session, r)})
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session action not found"})
	}
}

func (s *Server) buildChatRuntimeSessionDetail(ownerID string, session any, r *http.Request) any {
	if s.chatRuntimes == nil {
		return session
	}
	sessionMap, ok := chatRuntimeSessionMap(session)
	if !ok {
		return session
	}
	applyChatRuntimeSessionAPIFields(sessionMap)
	sessionID := strings.TrimSpace(fmt.Sprintf("%v", sessionMap["id"]))
	if sessionID == "" {
		return session
	}
	turns := []chatruntimeapp.TurnSummary(nil)
	if snapshotService, ok := s.chatRuntimes.(chatRuntimeDetailSnapshotService); ok {
		if snapshot, found := snapshotService.GetDetail(ownerID, sessionID); found {
			if snapshotMap, mapped := chatRuntimeSessionMap(snapshot.Session); mapped {
				sessionMap = snapshotMap
				applyChatRuntimeSessionAPIFields(sessionMap)
			}
			turns = snapshot.Turns
		}
	}
	if turns == nil {
		items, err := s.chatRuntimes.ListTurns(ownerID, sessionID)
		if err == nil {
			turns = items
		}
	}
	if turns != nil {
		items, paging := pageChatRuntimeTurns(turns, r)
		sessionMap["turns"] = buildChatRuntimeTurnDTOs(items)
		sessionMap["turns_paging"] = paging
	}
	return sessionMap
}

func buildChatRuntimeSessionSummary(session any) any {
	sessionMap, ok := chatRuntimeSessionMap(session)
	if !ok {
		return session
	}
	applyChatRuntimeSessionAPIFields(sessionMap)
	allowed := map[string]struct{}{
		"id": {}, "title": {}, "status": {}, "pinned": {},
		"created_at": {}, "updated_at": {},
	}
	for key := range sessionMap {
		if _, ok := allowed[key]; !ok {
			delete(sessionMap, key)
		}
	}
	return sessionMap
}

func chatRuntimeSessionMap(session any) (map[string]any, bool) {
	sessionMap := map[string]any{}
	encoded, err := json.Marshal(session)
	if err != nil {
		return nil, false
	}
	if err := json.Unmarshal(encoded, &sessionMap); err != nil {
		return nil, false
	}
	return sessionMap, true
}

func applyChatRuntimeSessionAPIFields(sessionMap map[string]any) {
	if sessionMap == nil {
		return
	}
	for _, key := range []string{"created_at", "updated_at"} {
		if parsed := parseChatRuntimeSessionPayloadTime(sessionMap[key]); !parsed.IsZero() {
			sessionMap[key] = unixMillis(parsed)
		} else {
			delete(sessionMap, key)
		}
	}
	delete(sessionMap, "activity_at")
	delete(sessionMap, "last_output_at")
	if parsed := parseChatRuntimeSessionPayloadTime(sessionMap["finished_at"]); !parsed.IsZero() {
		sessionMap["finished_at"] = unixMillis(parsed)
	} else {
		sessionMap["finished_at"] = nil
	}
	for _, key := range []string{"owner_id", "shell", "working_dir", "runtime_session_id", "revision", "version"} {
		delete(sessionMap, key)
	}
}

func parseChatRuntimeSessionPayloadTime(value any) time.Time {
	switch typed := value.(type) {
	case time.Time:
		if typed.IsZero() || typed.Year() <= 1 {
			return time.Time{}
		}
		return typed.UTC()
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return time.Time{}
		}
		parsed, err := time.Parse(time.RFC3339Nano, trimmed)
		if err != nil || parsed.IsZero() || parsed.Year() <= 1 {
			return time.Time{}
		}
		return parsed.UTC()
	default:
		return time.Time{}
	}
}

func unixMillis(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return value.UTC().UnixNano() / int64(time.Millisecond)
}

func buildChatRuntimeTurnDTOs(turns []chatruntimeapp.TurnSummary) []map[string]any {
	items := make([]map[string]any, 0, len(turns))
	for index, turn := range turns {
		items = append(items, buildChatRuntimeTurnDTO(turn, index+1))
	}
	return items
}

func buildChatRuntimeTurnDTO(turn chatruntimeapp.TurnSummary, fallbackID int) map[string]any {
	item := map[string]any{
		"id":          canonicalChatRuntimeID(turn.ID, "turn", fallbackID),
		"prompt":      turn.Prompt,
		"attachments": turn.Attachments,
		"status":      turn.Status,
	}
	if clientRequestID := strings.TrimSpace(turn.ClientRequestID); clientRequestID != "" {
		item["client_request_id"] = clientRequestID
	}
	if !turn.StartedAt.IsZero() {
		item["started_at"] = unixMillis(turn.StartedAt)
	}
	if !turn.FinishedAt.IsZero() {
		item["finished_at"] = unixMillis(turn.FinishedAt)
	} else {
		item["finished_at"] = nil
	}
	if turn.DurationMS > 0 {
		item["duration_ms"] = turn.DurationMS
	}
	if strings.TrimSpace(turn.FinalOutput) != "" {
		item["final_output"] = turn.FinalOutput
	}
	if len(turn.RuntimeTraceEvents) > 0 {
		item["runtime_trace_events"] = buildChatRuntimeEventDTOs(turn.RuntimeTraceEvents)
	}
	return item
}

func buildChatRuntimeEventDTOs(events []chatruntimeapp.RuntimeTraceEvent) []map[string]any {
	items := make([]map[string]any, 0, len(events))
	for index, event := range events {
		items = append(items, buildChatRuntimeEventDTO(event, index+1))
	}
	return items
}

func buildChatRuntimeEventDTO(event chatruntimeapp.RuntimeTraceEvent, fallbackID int) map[string]any {
	id := canonicalChatRuntimeID(event.ID, "event", event.Seq)
	if id == "" {
		id = canonicalChatRuntimeID("", "event", fallbackID)
	}
	text := strings.TrimSpace(event.Summary)
	if text == "" {
		text = strings.TrimSpace(event.Title)
	}
	item := map[string]any{
		"id":     id,
		"kind":   chatRuntimeAPIEventKind(event.Kind),
		"status": strings.TrimSpace(event.Status),
	}
	if text != "" {
		item["text"] = text
	}
	if event.Raw.HasDetail || len(event.Blocks) > 0 {
		item["detail_available"] = true
	}
	if !event.StartedAt.IsZero() {
		item["created_at"] = unixMillis(event.StartedAt)
	}
	if !event.CompletedAt.IsZero() {
		item["completed_at"] = unixMillis(event.CompletedAt)
	}
	if event.DurationMS > 0 {
		item["duration_ms"] = event.DurationMS
	}
	return item
}

func buildChatRuntimeEventDetailDTO(detail chatruntimeapp.RuntimeTraceEventDetail) chatRuntimeEventDetailEnvelope {
	return chatRuntimeEventDetailEnvelope{
		Event:  buildChatRuntimeEventDTO(detail.Event, numericChatRuntimeID(detail.Event.ID, "event", 1)),
		Blocks: append([]chatruntimeapp.RuntimeBlock{}, detail.Blocks...),
	}
}

func numericChatRuntimeID(value string, prefix string, fallback int) int {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return fallback
	}
	if parsed, err := strconv.Atoi(normalized); err == nil && parsed > 0 {
		return parsed
	}
	for _, marker := range []string{prefix + "-", prefix + "_", prefix + ":"} {
		if strings.HasPrefix(normalized, marker) {
			if parsed, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(normalized, marker))); err == nil && parsed > 0 {
				return parsed
			}
		}
	}
	lastDigits := ""
	for index := len(normalized) - 1; index >= 0; index-- {
		if normalized[index] < '0' || normalized[index] > '9' {
			break
		}
		lastDigits = string(normalized[index]) + lastDigits
	}
	if lastDigits != "" {
		if parsed, err := strconv.Atoi(lastDigits); err == nil && parsed > 0 {
			return parsed
		}
	}
	return fallback
}

func canonicalChatRuntimeID(value string, prefix string, fallback int) string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		if fallback <= 0 {
			return ""
		}
		return fmt.Sprintf("%s-%d", prefix, fallback)
	}
	if numeric, err := strconv.Atoi(normalized); err == nil && numeric > 0 {
		return fmt.Sprintf("%s-%d", prefix, numeric)
	}
	for _, marker := range []string{prefix + "-", prefix + "_", prefix + ":"} {
		if strings.HasPrefix(normalized, marker) {
			suffix := strings.TrimSpace(strings.TrimPrefix(normalized, marker))
			if numeric, err := strconv.Atoi(suffix); err == nil && numeric > 0 {
				return fmt.Sprintf("%s-%d", prefix, numeric)
			}
		}
	}
	return normalized
}

func chatRuntimeAPIEventKind(kind string) string {
	switch strings.TrimSpace(strings.ToLower(kind)) {
	case "assistant_commentary", "important_text", "message":
		return "important_text"
	case "reasoning":
		return "reasoning"
	case "plan":
		return "plan"
	case "shell_command", "command", "command_execution", "commands":
		return "commands"
	case "tool", "tools", "tool_call", "tool_result", "file_edit":
		return "tools"
	case "system", "system_event", "unknown_provider_event", "":
		return "system"
	default:
		return "system"
	}
}

func pageChatRuntimeTurns(turns []chatruntimeapp.TurnSummary, r *http.Request) ([]chatruntimeapp.TurnSummary, chatRuntimeTurnPagingEnvelope) {
	limit := defaultChatRuntimeTurnDetailLimit
	beforeTurnID := ""
	if r != nil {
		query := r.URL.Query()
		beforeTurnID = strings.TrimSpace(query.Get("turn_before"))
		if rawLimit := strings.TrimSpace(query.Get("turn_limit")); rawLimit != "" {
			if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 {
				limit = parsed
			}
		}
	}
	if limit > maxChatRuntimeTurnDetailLimit {
		limit = maxChatRuntimeTurnDetailLimit
	}
	total := len(turns)
	end := total
	beforeTurnFound := beforeTurnID == ""
	if beforeTurnID != "" {
		for index, turn := range turns {
			if strings.TrimSpace(turn.ID) == beforeTurnID {
				end = index
				beforeTurnFound = true
				break
			}
		}
	}
	if !beforeTurnFound {
		return []chatruntimeapp.TurnSummary{}, chatRuntimeTurnPagingEnvelope{
			Limit:           limit,
			Total:           total,
			ByteLimit:       maxChatRuntimeTurnDetailBytes,
			BeforeTurnFound: false,
		}
	}
	if end < 0 {
		end = 0
	}
	if end > total {
		end = total
	}
	candidateStart := end - limit
	if candidateStart < 0 {
		candidateStart = 0
	}
	start := end
	approxBytes := 0
	for index := end - 1; index >= candidateStart; index-- {
		turnBytes := approximateChatRuntimeTurnBytes(turns[index])
		if start < end && approxBytes+turnBytes > maxChatRuntimeTurnDetailBytes {
			break
		}
		approxBytes += turnBytes
		start = index
	}
	items := append([]chatruntimeapp.TurnSummary{}, turns[start:end]...)
	paging := chatRuntimeTurnPagingEnvelope{
		Limit:           limit,
		Total:           total,
		ByteLimit:       maxChatRuntimeTurnDetailBytes,
		ApproxBytes:     approxBytes,
		HasMoreBefore:   start > 0,
		HasMoreAfter:    end < total,
		BeforeTurnFound: true,
	}
	if len(items) > 0 {
		paging.OldestTurnID = items[0].ID
		paging.NewestTurnID = items[len(items)-1].ID
		paging.NextBeforeTurnID = items[0].ID
	}
	return items, paging
}

func approximateChatRuntimeTurnBytes(turn chatruntimeapp.TurnSummary) int {
	raw, err := json.Marshal(buildChatRuntimeTurnDTO(turn, 1))
	if err != nil {
		return len(turn.ID) + len(turn.Prompt) + len(turn.FinalOutput)
	}
	return len(raw)
}

func (s *Server) writeChatRuntimeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, chatruntimeapp.ErrSessionOwnerRequired):
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":      err.Error(),
			"error_code": "chatRuntime_client_required",
		})
	case errors.Is(err, chatruntimeapp.ErrSessionInputRequired):
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":      err.Error(),
			"error_code": "chatRuntime_input_required",
		})
	case errors.Is(err, chatruntimeapp.ErrSessionRecoverIDRequired):
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":      err.Error(),
			"error_code": "chatRuntime_recover_session_required",
		})
	case errors.Is(err, chatruntimeapp.ErrSessionNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error":      err.Error(),
			"error_code": "chatRuntime_session_not_found",
		})
	case errors.Is(err, chatruntimeapp.ErrTurnNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error":      err.Error(),
			"error_code": "chatRuntime_turn_not_found",
		})
	case errors.Is(err, chatruntimeapp.ErrRuntimeEventNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error":      err.Error(),
			"error_code": "chatRuntime_step_not_found",
		})
	case errors.Is(err, chatruntimeapp.ErrSessionBusy):
		writeJSON(w, http.StatusConflict, map[string]string{
			"error":      err.Error(),
			"error_code": "chatRuntime_session_busy",
		})
	case errors.Is(err, chatruntimeapp.ErrSessionNotRunning):
		writeJSON(w, http.StatusConflict, map[string]string{
			"error":      err.Error(),
			"error_code": "chatRuntime_session_not_running",
		})
	case errors.Is(err, chatruntimeapp.ErrRepositoryBindingConflict):
		writeJSON(w, http.StatusConflict, map[string]string{
			"error":      err.Error(),
			"error_code": "repository_binding_conflict",
		})
	case errors.Is(err, chatruntimeapp.ErrRepositoryUnavailable):
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error":      "GitHub repository is unavailable. Check the server GitHub login and retry.",
			"error_code": "repository_unavailable",
		})
	case errors.Is(err, chatruntimeapp.ErrRepositoryInvalid):
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":      err.Error(),
			"error_code": "repository_invalid",
		})
	case errors.Is(err, chatruntimeapp.ErrRepositoryRetryUnavailable):
		writeJSON(w, http.StatusConflict, map[string]string{
			"error":      err.Error(),
			"error_code": "repository_retry_unavailable",
		})
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":      err.Error(),
			"error_code": "chatRuntime_request_invalid",
		})
	}
}

func resolveChatRuntimeClientID(r *http.Request) string {
	if r == nil {
		return chatSessionOwnerID
	}
	if value, ok := r.Context().Value(chatRuntimeClientIDContextKey{}).(string); ok {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case chatSessionOwnerID:
			return chatSessionOwnerID
		}
	}
	return chatSessionOwnerID
}

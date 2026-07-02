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

	controldomain "alter0/internal/control/domain"
	execdomain "alter0/internal/execution/domain"
	terminalapp "alter0/internal/terminal/application"
)

const terminalSessionOwnerID = "terminal"
const chatSessionOwnerID = "chat"
const defaultTerminalTurnDetailLimit = 20
const maxTerminalTurnDetailLimit = 160
const maxTerminalTurnDetailBytes = 256 * 1024

type terminalClientIDContextKey struct{}

type terminalSessionCreateRequest struct {
	Title string `json:"title,omitempty"`
}

type terminalSessionInputRequest struct {
	Input       string                     `json:"input"`
	Attachments []messageAttachmentRequest `json:"attachments,omitempty"`
	SkillIDs    *[]string                  `json:"skill_ids,omitempty"`
}

type terminalSessionPinRequest struct {
	Pinned *bool `json:"pinned"`
}

type terminalSessionRecoverRequest struct {
	ID                string    `json:"id"`
	TerminalSessionID string    `json:"terminal_session_id,omitempty"`
	Title             string    `json:"title,omitempty"`
	CreatedAt         time.Time `json:"created_at,omitempty"`
	LastOutputAt      time.Time `json:"last_output_at,omitempty"`
	UpdatedAt         time.Time `json:"updated_at,omitempty"`
}

type terminalSessionEnvelope struct {
	Session any `json:"session"`
}

type terminalSessionListEnvelope struct {
	Items []any `json:"items"`
}

type terminalTurnPagingEnvelope struct {
	Limit            int    `json:"limit"`
	Total            int    `json:"total"`
	ByteLimit        int    `json:"byte_limit"`
	ApproxBytes      int    `json:"approx_bytes"`
	HasMoreBefore    bool   `json:"has_more_before"`
	HasMoreAfter     bool   `json:"has_more_after,omitempty"`
	OldestTurnID     string `json:"oldest_turn_id,omitempty"`
	NewestTurnID     string `json:"newest_turn_id,omitempty"`
	NextBeforeTurnID string `json:"next_before_turn_id,omitempty"`
}

func (s *Server) terminalSessionCollectionHandler(w http.ResponseWriter, r *http.Request) {
	if s.terminals == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "terminal service unavailable"})
		return
	}

	ownerID := resolveTerminalClientID(r)
	switch r.Method {
	case http.MethodGet:
		items := s.terminals.List(ownerID)
		summaries := make([]any, 0, len(items))
		for _, item := range items {
			summaries = append(summaries, buildTerminalSessionSummary(item))
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": summaries})
	case http.MethodPost:
		defer r.Body.Close()
		if ownerID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":      "terminal client id is required",
				"error_code": "terminal_client_required",
			})
			return
		}
		var req terminalSessionCreateRequest
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&req)
		}
		session, err := s.terminals.Create(terminalapp.CreateRequest{
			OwnerID: ownerID,
			Title:   strings.TrimSpace(req.Title),
		})
		if err != nil {
			s.writeTerminalError(w, err)
			return
		}
		s.publishTerminalSessionEvent(ownerID, session.ID, "session.created", session)
		writeJSON(w, http.StatusCreated, map[string]any{"session": s.buildTerminalSessionDetail(ownerID, session, r)})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) terminalSessionRecoverHandler(w http.ResponseWriter, r *http.Request) {
	if s.terminals == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "terminal service unavailable"})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	ownerID := resolveTerminalClientID(r)
	if ownerID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":      "terminal client id is required",
			"error_code": "terminal_client_required",
		})
		return
	}

	defer r.Body.Close()
	var req terminalSessionRecoverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}

	session, err := s.terminals.Recover(terminalapp.RecoverRequest{
		OwnerID:           ownerID,
		SessionID:         strings.TrimSpace(req.ID),
		TerminalSessionID: strings.TrimSpace(req.TerminalSessionID),
		Title:             strings.TrimSpace(req.Title),
		CreatedAt:         req.CreatedAt,
		LastOutputAt:      req.LastOutputAt,
		UpdatedAt:         req.UpdatedAt,
	})
	if err != nil {
		s.writeTerminalError(w, err)
		return
	}
	s.publishTerminalSessionEvent(ownerID, session.ID, "session.updated", session)
	writeJSON(w, http.StatusOK, map[string]any{"session": s.buildTerminalSessionDetail(ownerID, session, r)})
}

func (s *Server) chatSessionCollectionHandler(w http.ResponseWriter, r *http.Request) {
	s.terminalSessionCollectionHandler(w, withTerminalClientID(r, chatSessionOwnerID))
}

func (s *Server) chatSessionRecoverHandler(w http.ResponseWriter, r *http.Request) {
	s.terminalSessionRecoverHandler(w, withTerminalClientID(r, chatSessionOwnerID))
}

func (s *Server) chatSessionItemHandler(w http.ResponseWriter, r *http.Request) {
	next := withAttachmentRoutePrefix(withTerminalClientID(r, chatSessionOwnerID), "/api/chat/sessions")
	if next != nil && next.URL != nil {
		urlCopy := *next.URL
		urlCopy.Path = strings.TrimPrefix(urlCopy.Path, "/api/chat/sessions/")
		urlCopy.Path = "/api/terminal/sessions/" + strings.Trim(urlCopy.Path, "/")
		next = next.Clone(next.Context())
		next.URL = &urlCopy
	}
	s.terminalSessionItemHandler(w, next)
}

func withTerminalClientID(r *http.Request, clientID string) *http.Request {
	if r == nil {
		return r
	}
	ctx := context.WithValue(r.Context(), terminalClientIDContextKey{}, strings.TrimSpace(clientID))
	return r.WithContext(ctx)
}

func (s *Server) terminalSessionItemHandler(w http.ResponseWriter, r *http.Request) {
	if s.terminals == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "terminal service unavailable"})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/terminal/sessions/")
	path = strings.Trim(path, "/")
	if path == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
		return
	}

	parts := strings.Split(path, "/")
	sessionID := strings.TrimSpace(parts[0])
	ownerID := resolveTerminalClientID(r)
	if len(parts) == 1 {
		if r.Method == http.MethodDelete {
			session, err := s.terminals.Delete(ownerID, sessionID)
			if err != nil {
				s.writeTerminalError(w, err)
				return
			}
			s.publishTerminalSessionEvent(ownerID, sessionID, "session.deleted", session)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		session, ok := s.terminals.Get(ownerID, sessionID)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error":      "terminal session not found",
				"error_code": "terminal_session_not_found",
			})
			return
		}
		s.touchSessionActivity(sessionID)
		writeJSON(w, http.StatusOK, map[string]any{"session": s.buildTerminalSessionDetail(ownerID, session, r)})
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
		var req terminalSessionPinRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		if req.Pinned == nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "pinned is required"})
			return
		}
		session, err := s.terminals.SetPinned(ownerID, sessionID, *req.Pinned)
		if err != nil {
			s.writeTerminalError(w, err)
			return
		}
		s.publishTerminalSessionEvent(ownerID, session.ID, "session.updated", session)
		writeJSON(w, http.StatusOK, map[string]any{"session": s.buildTerminalSessionDetail(ownerID, session, r)})
	case "turns":
		if len(parts) == 2 {
			if r.Method != http.MethodGet {
				writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
				return
			}
			items, err := s.terminals.ListTurns(ownerID, sessionID)
			if err != nil {
				s.writeTerminalError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"items": items})
			return
		}
		if len(parts) == 5 && parts[3] == "events" {
			if r.Method != http.MethodGet {
				writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
				return
			}
			detail, err := s.terminals.GetRuntimeTraceEventDetail(ownerID, sessionID, strings.TrimSpace(parts[2]), strings.TrimSpace(parts[4]))
			if err != nil {
				s.writeTerminalError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"event": detail})
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
		page, err := s.terminals.ListEntries(ownerID, sessionID, cursor, limit)
		if err != nil {
			s.writeTerminalError(w, err)
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
	case "input":
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		defer r.Body.Close()
		var req terminalSessionInputRequest
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
		session, err := s.terminals.InputWithAttachments(terminalapp.InputRequest{
			OwnerID:      ownerID,
			SessionID:    sessionID,
			Input:        input,
			Attachments:  attachments,
			SkillContext: s.resolveTerminalSkillContext(req.SkillIDs),
		})
		if err != nil {
			s.writeTerminalError(w, err)
			return
		}
		s.touchSessionActivity(sessionID)
		s.publishTerminalSessionEvent(ownerID, session.ID, "session.updated", session)
		writeJSON(w, http.StatusOK, map[string]any{"session": s.buildTerminalSessionDetail(ownerID, session, r)})
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session action not found"})
	}
}

func (s *Server) buildTerminalSessionDetail(ownerID string, session any, r *http.Request) any {
	if s.terminals == nil {
		return session
	}
	sessionMap, ok := terminalSessionMap(session)
	if !ok {
		return session
	}
	applyTerminalSessionComparableFields(sessionMap)
	sessionID := strings.TrimSpace(fmt.Sprintf("%v", sessionMap["id"]))
	if sessionID == "" {
		return session
	}
	turns, err := s.terminals.ListTurns(ownerID, sessionID)
	if err == nil {
		items, paging := pageTerminalTurns(turns, r)
		sessionMap["turns"] = items
		sessionMap["turns_paging"] = paging
	}
	return sessionMap
}

func buildTerminalSessionSummary(session any) any {
	sessionMap, ok := terminalSessionMap(session)
	if !ok {
		return session
	}
	applyTerminalSessionComparableFields(sessionMap)
	delete(sessionMap, "turns")
	delete(sessionMap, "turns_paging")
	return sessionMap
}

func terminalSessionMap(session any) (map[string]any, bool) {
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

func applyTerminalSessionComparableFields(sessionMap map[string]any) {
	if sessionMap == nil {
		return
	}
	activityAt := latestNonZeroTime(
		parseTerminalSessionPayloadTime(sessionMap["last_output_at"]),
		parseTerminalSessionPayloadTime(sessionMap["updated_at"]),
		parseTerminalSessionPayloadTime(sessionMap["created_at"]),
	)
	revisionAt := latestNonZeroTime(
		activityAt,
		parseTerminalSessionPayloadTime(sessionMap["finished_at"]),
	)
	if !activityAt.IsZero() {
		sessionMap["activity_at"] = activityAt.UTC().Format(time.RFC3339Nano)
	}
	if !revisionAt.IsZero() {
		sessionMap["revision"] = revisionAt.UnixMicro()
	}
}

func parseTerminalSessionPayloadTime(value any) time.Time {
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

func latestNonZeroTime(values ...time.Time) time.Time {
	var latest time.Time
	for _, value := range values {
		if value.IsZero() {
			continue
		}
		if latest.IsZero() || value.After(latest) {
			latest = value
		}
	}
	return latest
}

func pageTerminalTurns(turns []terminalapp.TurnSummary, r *http.Request) ([]terminalapp.TurnSummary, terminalTurnPagingEnvelope) {
	limit := defaultTerminalTurnDetailLimit
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
	if limit > maxTerminalTurnDetailLimit {
		limit = maxTerminalTurnDetailLimit
	}
	total := len(turns)
	end := total
	if beforeTurnID != "" {
		for index, turn := range turns {
			if strings.TrimSpace(turn.ID) == beforeTurnID {
				end = index
				break
			}
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
		turnBytes := approximateTerminalTurnBytes(turns[index])
		if start < end && approxBytes+turnBytes > maxTerminalTurnDetailBytes {
			break
		}
		approxBytes += turnBytes
		start = index
	}
	items := append([]terminalapp.TurnSummary{}, turns[start:end]...)
	paging := terminalTurnPagingEnvelope{
		Limit:         limit,
		Total:         total,
		ByteLimit:     maxTerminalTurnDetailBytes,
		ApproxBytes:   approxBytes,
		HasMoreBefore: start > 0,
		HasMoreAfter:  end < total,
	}
	if len(items) > 0 {
		paging.OldestTurnID = items[0].ID
		paging.NewestTurnID = items[len(items)-1].ID
		paging.NextBeforeTurnID = items[0].ID
	}
	return items, paging
}

func approximateTerminalTurnBytes(turn terminalapp.TurnSummary) int {
	raw, err := json.Marshal(turn)
	if err != nil {
		return len(turn.ID) + len(turn.Prompt) + len(turn.FinalOutput)
	}
	return len(raw)
}

func (s *Server) resolveTerminalSkillContext(skillIDs *[]string) *execdomain.SkillContext {
	if s.control == nil {
		return nil
	}
	selectedOnly := skillIDs != nil
	include := map[string]struct{}{}
	if selectedOnly {
		include = normalizeTerminalSkillIDSet(*skillIDs)
		if len(include) == 0 {
			return nil
		}
	}
	skills := make([]execdomain.SkillSpec, 0)
	for _, capability := range s.control.ListCapabilitiesByType(controldomain.CapabilityTypeSkill) {
		if !capability.Enabled || !isPublicTerminalSkillCapability(capability) {
			continue
		}
		id := strings.TrimSpace(capability.ID)
		if id == "" {
			continue
		}
		if selectedOnly {
			if _, ok := include[id]; !ok {
				continue
			}
		}
		skills = append(skills, terminalSkillSpecFromCapability(capability))
	}
	if len(skills) == 0 {
		return nil
	}
	return &execdomain.SkillContext{
		Protocol: execdomain.SkillContextProtocolVersion,
		Skills:   skills,
	}
}

func normalizeTerminalSkillIDSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			out[trimmed] = struct{}{}
		}
	}
	return out
}

func isPublicTerminalSkillCapability(capability controldomain.Capability) bool {
	metadata := capability.Metadata
	visibility := strings.ToLower(strings.TrimSpace(metadata["alter0.skill.visibility"]))
	if visibility == "" {
		visibility = strings.ToLower(strings.TrimSpace(metadata["skill.visibility"]))
	}
	return visibility != "private"
}

func terminalSkillSpecFromCapability(capability controldomain.Capability) execdomain.SkillSpec {
	metadata := capability.Metadata
	description := strings.TrimSpace(metadata["skill.description"])
	if description == "" {
		description = strings.TrimSpace(capability.Name)
	}
	return execdomain.SkillSpec{
		ID:          strings.TrimSpace(capability.ID),
		Name:        strings.TrimSpace(capability.Name),
		Description: description,
		Guide:       strings.TrimSpace(metadata["skill.guide"]),
		Priority:    parseTerminalSkillPriority(metadata["skill.priority"]),
		Constraints: parseTerminalSkillList(metadata["skill.constraints"]),
		Abilities:   parseTerminalSkillList(metadata["skill.abilities"]),
		FilePath:    strings.TrimSpace(metadata["skill.file_path"]),
		Writable:    parseTerminalSkillWritable(metadata["skill.writable"]),
	}
}

func parseTerminalSkillPriority(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 100
	}
	return value
}

func parseTerminalSkillWritable(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func parseTerminalSkillList(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	var decoded []string
	if err := json.Unmarshal([]byte(trimmed), &decoded); err == nil {
		return normalizeTerminalSkillStringList(decoded)
	}
	return normalizeTerminalSkillStringList(strings.Split(trimmed, ","))
}

func normalizeTerminalSkillStringList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (s *Server) writeTerminalError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, terminalapp.ErrSessionOwnerRequired):
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":      err.Error(),
			"error_code": "terminal_client_required",
		})
	case errors.Is(err, terminalapp.ErrSessionInputRequired):
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":      err.Error(),
			"error_code": "terminal_input_required",
		})
	case errors.Is(err, terminalapp.ErrSessionRecoverIDRequired):
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":      err.Error(),
			"error_code": "terminal_recover_session_required",
		})
	case errors.Is(err, terminalapp.ErrSessionNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error":      err.Error(),
			"error_code": "terminal_session_not_found",
		})
	case errors.Is(err, terminalapp.ErrTurnNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error":      err.Error(),
			"error_code": "terminal_turn_not_found",
		})
	case errors.Is(err, terminalapp.ErrRuntimeEventNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error":      err.Error(),
			"error_code": "terminal_step_not_found",
		})
	case errors.Is(err, terminalapp.ErrSessionBusy):
		writeJSON(w, http.StatusConflict, map[string]string{
			"error":      err.Error(),
			"error_code": "terminal_session_busy",
		})
	case errors.Is(err, terminalapp.ErrSessionNotRunning):
		writeJSON(w, http.StatusConflict, map[string]string{
			"error":      err.Error(),
			"error_code": "terminal_session_not_running",
		})
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":      err.Error(),
			"error_code": "terminal_request_invalid",
		})
	}
}

func resolveTerminalClientID(r *http.Request) string {
	if r == nil {
		return terminalSessionOwnerID
	}
	if value, ok := r.Context().Value(terminalClientIDContextKey{}).(string); ok {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case chatSessionOwnerID:
			return chatSessionOwnerID
		case terminalSessionOwnerID:
			return terminalSessionOwnerID
		}
	}
	return terminalSessionOwnerID
}

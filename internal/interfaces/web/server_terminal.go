package web

import (
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

const terminalClientIDHeader = "X-Alter0-Terminal-Client"
const sharedTerminalClientID = "shared"

type terminalSessionCreateRequest struct {
	Title string `json:"title,omitempty"`
}

type terminalSessionInputRequest struct {
	Input       string                     `json:"input"`
	Attachments []messageAttachmentRequest `json:"attachments,omitempty"`
	SkillIDs    []string                   `json:"skill_ids,omitempty"`
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

type terminalStepEnvelope struct {
	Step any `json:"step"`
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
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
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
		writeJSON(w, http.StatusCreated, map[string]any{"session": s.buildTerminalSessionDetail(ownerID, session)})
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
	writeJSON(w, http.StatusOK, map[string]any{"session": s.buildTerminalSessionDetail(ownerID, session)})
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
			_, err := s.terminals.Delete(ownerID, sessionID)
			if err != nil {
				s.writeTerminalError(w, err)
				return
			}
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
		writeJSON(w, http.StatusOK, map[string]any{"session": s.buildTerminalSessionDetail(ownerID, session)})
		return
	}

	switch parts[1] {
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
		if len(parts) == 5 && parts[3] == "steps" {
			if r.Method != http.MethodGet {
				writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
				return
			}
			detail, err := s.terminals.GetStepDetail(ownerID, sessionID, strings.TrimSpace(parts[2]), strings.TrimSpace(parts[4]))
			if err != nil {
				s.writeTerminalError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"step": detail})
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
		attachments, err := s.normalizeConversationMessageAttachments(sessionID, req.Attachments)
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
		writeJSON(w, http.StatusOK, map[string]any{"session": s.buildTerminalSessionDetail(ownerID, session)})
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session action not found"})
	}
}

func (s *Server) buildTerminalSessionDetail(ownerID string, session any) any {
	if s.terminals == nil {
		return session
	}
	sessionMap := map[string]any{}
	encoded, err := json.Marshal(session)
	if err != nil {
		return session
	}
	if err := json.Unmarshal(encoded, &sessionMap); err != nil {
		return session
	}
	sessionID := strings.TrimSpace(fmt.Sprintf("%v", sessionMap["id"]))
	if sessionID == "" {
		return session
	}
	turns, err := s.terminals.ListTurns(ownerID, sessionID)
	if err == nil {
		sessionMap["turns"] = turns
	}
	return sessionMap
}

func (s *Server) resolveTerminalSkillContext(skillIDs []string) *execdomain.SkillContext {
	include := normalizeTerminalSkillIDSet(skillIDs)
	if len(include) == 0 || s.control == nil {
		return nil
	}
	skills := make([]execdomain.SkillSpec, 0, len(include))
	for _, capability := range s.control.ListCapabilitiesByType(controldomain.CapabilityTypeSkill) {
		if !capability.Enabled || !isPublicTerminalSkillCapability(capability) {
			continue
		}
		id := strings.TrimSpace(capability.ID)
		if _, ok := include[id]; !ok {
			continue
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
	return visibility != "agent-private" && visibility != "private"
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
	case errors.Is(err, terminalapp.ErrStepNotFound):
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
	return sharedTerminalClientID
}

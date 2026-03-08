package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	terminalapp "alter0/internal/terminal/application"
)

const terminalClientIDHeader = "X-Alter0-Terminal-Client"

type terminalSessionCreateRequest struct {
	Title string `json:"title,omitempty"`
}

type terminalSessionInputRequest struct {
	Input string `json:"input"`
}

type terminalSessionEnvelope struct {
	Session any `json:"session"`
}

type terminalSessionListEnvelope struct {
	Items []any `json:"items"`
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
		writeJSON(w, http.StatusCreated, map[string]any{"session": session})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
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
			session, err := s.terminals.Close(ownerID, sessionID)
			if err != nil {
				s.writeTerminalError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"session": session})
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
		writeJSON(w, http.StatusOK, map[string]any{"session": session})
		return
	}

	switch parts[1] {
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
		session, err := s.terminals.Input(ownerID, sessionID, req.Input)
		if err != nil {
			s.writeTerminalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"session": session})
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session action not found"})
	}
}

func (s *Server) writeTerminalError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, terminalapp.ErrSessionOwnerRequired):
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":      err.Error(),
			"error_code": "terminal_client_required",
		})
	case errors.Is(err, terminalapp.ErrSessionLimitReached):
		maxSessions := 5
		if s.terminals != nil {
			maxSessions = s.terminals.MaxSessions()
		}
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":        fmt.Sprintf("terminal session limit reached (%d)", maxSessions),
			"error_code":   "terminal_session_limit_reached",
			"max_sessions": maxSessions,
		})
	case errors.Is(err, terminalapp.ErrSessionNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error":      err.Error(),
			"error_code": "terminal_session_not_found",
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
		return ""
	}
	return strings.TrimSpace(r.Header.Get(terminalClientIDHeader))
}

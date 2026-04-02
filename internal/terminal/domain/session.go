package domain

import (
	"strings"
	"time"
)

type SessionStatus string

const (
	SessionStatusReady       SessionStatus = "ready"
	SessionStatusBusy        SessionStatus = "busy"
	SessionStatusExited      SessionStatus = "exited"
	SessionStatusFailed      SessionStatus = "failed"
	SessionStatusInterrupted SessionStatus = "interrupted"

	// Legacy values kept for persisted-state compatibility.
	SessionStatusStarting SessionStatus = "starting"
	SessionStatusRunning  SessionStatus = "running"
)

func NormalizeSessionStatus(status SessionStatus) SessionStatus {
	switch strings.ToLower(strings.TrimSpace(string(status))) {
	case "", string(SessionStatusReady), string(SessionStatusRunning):
		return SessionStatusReady
	case string(SessionStatusBusy), string(SessionStatusStarting):
		return SessionStatusBusy
	case string(SessionStatusExited):
		return SessionStatusExited
	case string(SessionStatusInterrupted):
		return SessionStatusInterrupted
	case string(SessionStatusFailed):
		return SessionStatusFailed
	default:
		return SessionStatusReady
	}
}

func IsSessionOpenStatus(status SessionStatus) bool {
	switch NormalizeSessionStatus(status) {
	case SessionStatusReady, SessionStatusBusy:
		return true
	default:
		return false
	}
}

func CanSessionAcceptInput(status SessionStatus) bool {
	return NormalizeSessionStatus(status) != SessionStatusBusy
}

type Entry struct {
	Cursor    int       `json:"cursor"`
	Stream    string    `json:"stream,omitempty"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

type Session struct {
	ID                string        `json:"id"`
	TerminalSessionID string        `json:"terminal_session_id,omitempty"`
	OwnerID           string        `json:"owner_id,omitempty"`
	Title             string        `json:"title,omitempty"`
	Shell             string        `json:"shell,omitempty"`
	WorkingDir        string        `json:"working_dir,omitempty"`
	Status            SessionStatus `json:"status"`
	CreatedAt         time.Time     `json:"created_at,omitempty"`
	LastOutputAt      time.Time     `json:"last_output_at,omitempty"`
	UpdatedAt         time.Time     `json:"updated_at,omitempty"`
	FinishedAt        time.Time     `json:"finished_at,omitempty"`
	ExitCode          *int          `json:"exit_code,omitempty"`
	ErrorMessage      string        `json:"error_message,omitempty"`
}

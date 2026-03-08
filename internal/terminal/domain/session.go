package domain

import "time"

type SessionStatus string

const (
	SessionStatusStarting    SessionStatus = "starting"
	SessionStatusRunning     SessionStatus = "running"
	SessionStatusExited      SessionStatus = "exited"
	SessionStatusFailed      SessionStatus = "failed"
	SessionStatusInterrupted SessionStatus = "interrupted"
)

type Entry struct {
	Cursor    int       `json:"cursor"`
	Stream    string    `json:"stream,omitempty"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

type Session struct {
	ID           string        `json:"id"`
	OwnerID      string        `json:"owner_id,omitempty"`
	Title        string        `json:"title,omitempty"`
	Shell        string        `json:"shell,omitempty"`
	WorkingDir   string        `json:"working_dir,omitempty"`
	Status       SessionStatus `json:"status"`
	CreatedAt    time.Time     `json:"created_at,omitempty"`
	LastOutputAt time.Time     `json:"last_output_at,omitempty"`
	UpdatedAt    time.Time     `json:"updated_at,omitempty"`
	FinishedAt   time.Time     `json:"finished_at,omitempty"`
	ExitCode     *int          `json:"exit_code,omitempty"`
	ErrorMessage string        `json:"error_message,omitempty"`
}

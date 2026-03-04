package domain

import "time"

type RunStatus string

const (
	RunStatusSuccess RunStatus = "success"
	RunStatusFailed  RunStatus = "failed"
)

type Run struct {
	ID        string    `json:"run_id"`
	JobID     string    `json:"job_id"`
	FiredAt   time.Time `json:"fired_at"`
	SessionID string    `json:"session_id"`
	Status    RunStatus `json:"status"`
}

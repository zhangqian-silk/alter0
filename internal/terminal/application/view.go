package application

import "time"

type TurnSummary struct {
	ID          string           `json:"id"`
	Prompt      string           `json:"prompt"`
	Attachments []TurnAttachment `json:"attachments,omitempty"`
	Status      string           `json:"status"`
	StartedAt   time.Time        `json:"started_at,omitempty"`
	FinishedAt  time.Time        `json:"finished_at,omitempty"`
	DurationMS  int64            `json:"duration_ms,omitempty"`
	FinalOutput string           `json:"final_output,omitempty"`
	Steps       []StepSummary    `json:"steps,omitempty"`
}

type TurnAttachment struct {
	Name          string `json:"name"`
	ContentType   string `json:"content_type"`
	DataURL       string `json:"data_url,omitempty"`
	AssetURL      string `json:"asset_url,omitempty"`
	PreviewURL    string `json:"preview_url,omitempty"`
	WorkspacePath string `json:"-"`
}

type StepSummary struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Title      string    `json:"title"`
	Status     string    `json:"status"`
	StartedAt  time.Time `json:"started_at,omitempty"`
	FinishedAt time.Time `json:"finished_at,omitempty"`
	DurationMS int64     `json:"duration_ms,omitempty"`
	Preview    string    `json:"preview,omitempty"`
	HasDetail  bool      `json:"has_detail"`
}

type StepDetail struct {
	TurnID     string            `json:"turn_id"`
	Step       StepSummary       `json:"step"`
	Blocks     []StepDetailBlock `json:"blocks,omitempty"`
	Searchable bool              `json:"searchable,omitempty"`
}

type StepDetailBlock struct {
	Type      string `json:"type"`
	Title     string `json:"title,omitempty"`
	Content   string `json:"content,omitempty"`
	Language  string `json:"language,omitempty"`
	File      string `json:"file,omitempty"`
	StartLine int    `json:"start_line,omitempty"`
	Status    string `json:"status,omitempty"`
	ExitCode  *int   `json:"exit_code,omitempty"`
}

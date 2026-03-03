package domain

type Route string

const (
	RouteCommand Route = "command"
	RouteNL      Route = "nl"
)

type OrchestrationResult struct {
	MessageID string            `json:"message_id"`
	SessionID string            `json:"session_id"`
	Route     Route             `json:"route"`
	Output    string            `json:"output"`
	ErrorCode string            `json:"error_code,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type ExecutionResult struct {
	Output   string            `json:"output"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

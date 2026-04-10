package domain

type Route string

const (
	RouteCommand Route = "command"
	RouteNL      Route = "nl"
)

type OrchestrationResult struct {
	MessageID    string            `json:"message_id"`
	SessionID    string            `json:"session_id"`
	Route        Route             `json:"route"`
	Output       string            `json:"output"`
	ErrorCode    string            `json:"error_code,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	ProcessSteps []ProcessStep     `json:"process_steps,omitempty"`
}

type ExecutionResult struct {
	Output       string            `json:"output"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	ProcessSteps []ProcessStep     `json:"process_steps,omitempty"`
}

type ProcessStep struct {
	Kind   string `json:"kind,omitempty"`
	ID     string `json:"id,omitempty"`
	Title  string `json:"title,omitempty"`
	Detail string `json:"detail,omitempty"`
	Status string `json:"status,omitempty"`
}

type StreamEventType string

const (
	StreamEventTypeOutput  StreamEventType = "output"
	StreamEventTypeProcess StreamEventType = "process"
)

type StreamEvent struct {
	Type        StreamEventType `json:"type"`
	Text        string          `json:"text,omitempty"`
	ProcessStep *ProcessStep    `json:"process_step,omitempty"`
}

package domain

const (
	RuntimeSessionIDMetadataKey   = "alter0.runtime.session_id"
	RuntimeMessageIDMetadataKey   = "alter0.runtime.message_id"
	RuntimeTraceIDMetadataKey     = "alter0.runtime.trace_id"
	RuntimeContextProtocolVersion = "alter0.runtime-context/v1"
)

type RuntimeContext struct {
	Protocol   string              `json:"protocol"`
	SessionID  string              `json:"session_id,omitempty"`
	MessageID  string              `json:"message_id,omitempty"`
	TraceID    string              `json:"trace_id,omitempty"`
	Workspace  *RuntimeWorkspace   `json:"workspace,omitempty"`
	Repository *RuntimeRepository  `json:"repository,omitempty"`
	Preview    *RuntimePreviewRule `json:"preview,omitempty"`
}

type RuntimeWorkspace struct {
	Mode           string `json:"mode,omitempty"`
	SessionPath    string `json:"session_path,omitempty"`
	TaskPath       string `json:"task_path,omitempty"`
	RepositoryPath string `json:"repository_path,omitempty"`
}

type RuntimeRepository struct {
	SourcePath    string `json:"source_path,omitempty"`
	RemoteURL     string `json:"remote_url,omitempty"`
	ActiveBranch  string `json:"active_branch,omitempty"`
	DefaultBranch string `json:"default_branch,omitempty"`
}

type RuntimePreviewRule struct {
	URL                  string `json:"url,omitempty"`
	RequiredOnCompletion bool   `json:"required_on_completion,omitempty"`
}

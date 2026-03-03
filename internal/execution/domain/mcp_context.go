package domain

import "time"

const (
	MCPContextProtocolVersion = "alter0.mcp-context/v1"
	MCPContextMetadataKey     = "alter0.mcp_context"
)

type MCPContext struct {
	Protocol string         `json:"protocol"`
	Servers  []MCPServer    `json:"servers"`
	Audit    []MCPAuditItem `json:"audit,omitempty"`
}

type MCPServer struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Scope            string            `json:"scope"`
	Transport        string            `json:"transport"`
	Command          string            `json:"command,omitempty"`
	Args             []string          `json:"args,omitempty"`
	Env              map[string]string `json:"env,omitempty"`
	URL              string            `json:"url,omitempty"`
	Headers          map[string]string `json:"headers,omitempty"`
	ToolWhitelist    []string          `json:"tool_whitelist,omitempty"`
	TimeoutMS        int               `json:"timeout_ms"`
	FailureIsolation bool              `json:"failure_isolation"`
}

type MCPAuditItem struct {
	ServerID   string    `json:"server_id"`
	ServerName string    `json:"server_name"`
	Scope      string    `json:"scope"`
	Decision   string    `json:"decision"`
	Reason     string    `json:"reason,omitempty"`
	TraceID    string    `json:"trace_id"`
	SessionID  string    `json:"session_id"`
	MessageID  string    `json:"message_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

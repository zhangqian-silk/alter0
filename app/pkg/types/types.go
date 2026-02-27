package types

import "context"

// Message represents a user input or system event
type Message struct {
	ID        string
	Content   string
	Role      string // "user", "assistant", "system"
	ChannelID string // Source channel identifier (e.g., "lark", "cli")
	UserID    string
	TaskID    string
	RequestID string
	Meta      map[string]interface{}
}

// Agent represents the core reasoning entity
type Agent interface {
	Process(ctx context.Context, msg Message) (Message, error)
	Name() string
}

// Channel represents an input/output interface (Lark, CLI, Web)
type Channel interface {
	Start(ctx context.Context, handler func(Message)) error
	Send(ctx context.Context, msg Message) error
	ID() string
}

// Skill represents a capability or tool
type Skill interface {
	Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
	Manifest() SkillManifest
}

type SkillManifest struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
}

// Gateway orchestrates channels and the agent
type Gateway interface {
	RegisterChannel(c Channel)
	Start(ctx context.Context) error
}

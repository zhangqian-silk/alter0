package types

import "context"

const (
	MessageRoleUser      = "user"
	MessageRoleAssistant = "assistant"
	MessageRoleSystem    = "system"
)

const (
	EnvelopeDirectionInbound  = "inbound"
	EnvelopeDirectionOutbound = "outbound"
)

const (
	EnvelopePartText      = "text"
	EnvelopePartImage     = "image"
	EnvelopePartAudio     = "audio"
	EnvelopePartFile      = "file"
	EnvelopePartReference = "reference"
)

// EnvelopePart describes a typed message unit for channel adapters.
type EnvelopePart struct {
	Type     string                 `json:"type"`
	Text     string                 `json:"text,omitempty"`
	URL      string                 `json:"url,omitempty"`
	MimeType string                 `json:"mime_type,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Size     int64                  `json:"size,omitempty"`
	Meta     map[string]interface{} `json:"meta,omitempty"`
}

// MessageEnvelope keeps channel-level context in a transport-neutral structure.
type MessageEnvelope struct {
	Direction string                 `json:"direction,omitempty"`
	Channel   string                 `json:"channel,omitempty"`
	AccountID string                 `json:"account_id,omitempty"`
	PeerID    string                 `json:"peer_id,omitempty"`
	PeerType  string                 `json:"peer_type,omitempty"`
	MessageID string                 `json:"message_id,omitempty"`
	ReplyToID string                 `json:"reply_to_id,omitempty"`
	Parts     []EnvelopePart         `json:"parts,omitempty"`
	Meta      map[string]interface{} `json:"meta,omitempty"`
}

func (e *MessageEnvelope) clone() *MessageEnvelope {
	if e == nil {
		return nil
	}
	clone := &MessageEnvelope{
		Direction: e.Direction,
		Channel:   e.Channel,
		AccountID: e.AccountID,
		PeerID:    e.PeerID,
		PeerType:  e.PeerType,
		MessageID: e.MessageID,
		ReplyToID: e.ReplyToID,
	}
	if len(e.Parts) > 0 {
		clone.Parts = make([]EnvelopePart, 0, len(e.Parts))
		for _, part := range e.Parts {
			copied := part
			if len(part.Meta) > 0 {
				copied.Meta = map[string]interface{}{}
				for k, v := range part.Meta {
					copied.Meta[k] = v
				}
			}
			clone.Parts = append(clone.Parts, copied)
		}
	}
	if len(e.Meta) > 0 {
		clone.Meta = map[string]interface{}{}
		for k, v := range e.Meta {
			clone.Meta[k] = v
		}
	}
	return clone
}

// Message represents a user input or system event.
type Message struct {
	ID        string
	Content   string
	Role      string // "user", "assistant", "system"
	ChannelID string // Source channel identifier (e.g., "lark", "cli")
	UserID    string
	TaskID    string
	RequestID string
	Envelope  *MessageEnvelope
	Meta      map[string]interface{}
}

// EnsureEnvelope returns a mutable envelope and applies sensible defaults.
func (m *Message) EnsureEnvelope() *MessageEnvelope {
	if m.Envelope == nil {
		m.Envelope = &MessageEnvelope{}
	}
	if m.Envelope.Channel == "" {
		m.Envelope.Channel = m.ChannelID
	}
	if m.Envelope.PeerID == "" {
		m.Envelope.PeerID = m.UserID
	}
	if m.Envelope.MessageID == "" {
		m.Envelope.MessageID = m.ID
	}
	if m.Envelope.Direction == "" {
		m.Envelope.Direction = EnvelopeDirectionInbound
	}
	if len(m.Envelope.Parts) == 0 && m.Content != "" {
		m.Envelope.Parts = []EnvelopePart{{Type: EnvelopePartText, Text: m.Content}}
	}
	return m.Envelope
}

// CloneEnvelopeFrom copies envelope context from source if target has none.
func (m *Message) CloneEnvelopeFrom(source Message) {
	if m.Envelope != nil || source.Envelope == nil {
		return
	}
	m.Envelope = source.Envelope.clone()
}

// Agent represents the core reasoning entity.
type Agent interface {
	Process(ctx context.Context, msg Message) (Message, error)
	Name() string
}

// Channel represents an input/output interface (Lark, CLI, Web).
type Channel interface {
	Start(ctx context.Context, handler func(Message)) error
	Send(ctx context.Context, msg Message) error
	ID() string
}

// Skill represents a capability or tool.
type Skill interface {
	Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
	Manifest() SkillManifest
}

type SkillManifest struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
}

// Gateway orchestrates channels and the agent.
type Gateway interface {
	RegisterChannel(c Channel)
	Start(ctx context.Context) error
}

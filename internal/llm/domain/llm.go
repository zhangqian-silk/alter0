package domain

import (
	"context"
)

// Message represents a single message in the conversation.
type Message struct {
	Role    string // "system", "user", "assistant", "tool"
	Content string
	// For tool calls
	ToolCalls  []ToolCall
	ToolCallID string // For tool response messages
}

// ToolCall represents a tool call from the model.
type ToolCall struct {
	ID       string
	Name     string
	Arguments string // JSON string
}

// Tool represents a tool that can be called by the model.
type Tool struct {
	Name        string
	Description string
	Parameters  map[string]interface{} // JSON Schema
}

// ToolResult represents the result of a tool execution.
type ToolResult struct {
	ToolCallID string
	Name       string
	Result     string
	IsError    bool
}

// ChatRequest represents a chat completion request.
type ChatRequest struct {
	Model       string
	Messages    []Message
	Tools       []Tool
	Temperature *float64
	MaxTokens   *int
	// For multi-turn with previous response ID (OpenAI Responses API)
	PreviousResponseID string
}

// ChatResponse represents a chat completion response.
type ChatResponse struct {
	ID      string
	Message Message
	// Usage statistics
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	// For streaming
	Delta string
}

// StreamEvent represents a streaming event.
type StreamEvent struct {
	Type    string // "delta", "done", "error"
	Delta   string
	Message *Message
	Error   error
}

// LLMClient defines the interface for LLM clients.
type LLMClient interface {
	// Chat performs a chat completion.
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	// ChatStream performs a streaming chat completion.
	ChatStream(ctx context.Context, req ChatRequest, onEvent func(StreamEvent) error) (*ChatResponse, error)
	// Close closes the client.
	Close() error
}

// LLMConfig represents the configuration for an LLM client.
type LLMConfig struct {
	APIKey      string
	BaseURL     string // For OpenAI-compatible APIs
	Model       string
	Temperature *float64
	MaxTokens   *int
}
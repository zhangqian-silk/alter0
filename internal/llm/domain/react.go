package domain

import (
	"context"
	"strings"
)

// ReActAgentConfig represents the configuration for a ReAct agent.
type ReActAgentConfig struct {
	// Client is the LLM client to use.
	Client LLMClient
	// Model is the model to use.
	Model string
	// SystemPrompt is the system prompt for the agent.
	SystemPrompt string
	// Tools are the tools available to the agent.
	Tools []Tool
	// ToolExecutor executes tool calls.
	ToolExecutor ToolExecutor
	// MaxIterations is the maximum number of ReAct iterations.
	MaxIterations int
	// Temperature is the temperature for the LLM.
	Temperature *float64
}

// ToolExecutor executes tool calls.
type ToolExecutor interface {
	// Execute executes a tool call and returns the result.
	Execute(ctx context.Context, toolCall ToolCall) (*ToolResult, error)
}

// ToolExecutorFunc is a function that implements ToolExecutor.
type ToolExecutorFunc func(ctx context.Context, toolCall ToolCall) (*ToolResult, error)

// Execute implements ToolExecutor.
func (f ToolExecutorFunc) Execute(ctx context.Context, toolCall ToolCall) (*ToolResult, error) {
	return f(ctx, toolCall)
}

// ReActState represents the current state of a ReAct loop.
type ReActState struct {
	// Iteration is the current iteration number.
	Iteration int
	// Thought is the model's thought.
	Thought string
	// Action is the action the model decided to take.
	Action *ToolCall
	// Observation is the result of the action.
	Observation string
	// Answer is the final answer (if the loop is complete).
	Answer string
	// IsComplete indicates whether the ReAct loop is complete.
	IsComplete bool
	// Messages is the conversation history.
	Messages []Message
	// Error is the error that occurred (if any).
	Error error
}

// ReActEvent represents an event in the ReAct loop.
type ReActEvent struct {
	Type string // "thought", "action", "observation", "answer", "error"
	// State is the current state.
	State *ReActState
	// Delta is the streaming delta (for streaming responses).
	Delta string
}

// ReActAgent implements the ReAct pattern.
type ReActAgent struct {
	config ReActAgentConfig
}

// NewReActAgent creates a new ReAct agent.
func NewReActAgent(config ReActAgentConfig) *ReActAgent {
	if config.MaxIterations <= 0 {
		config.MaxIterations = 10
	}
	return &ReActAgent{config: config}
}

// Run runs the ReAct loop and returns the final answer.
func (a *ReActAgent) Run(ctx context.Context, userMessage string) (string, error) {
	state, err := a.RunWithState(ctx, userMessage, nil)
	if err != nil {
		return "", err
	}
	return state.Answer, nil
}

// RunWithState runs the ReAct loop and returns the final state.
func (a *ReActAgent) RunWithState(ctx context.Context, userMessage string, onEvent func(ReActEvent) error) (*ReActState, error) {
	state := &ReActState{
		Messages: []Message{},
	}

	// Build system prompt with ReAct format
	systemPrompt := a.buildSystemPrompt()
	state.Messages = append(state.Messages, Message{
		Role:    "system",
		Content: systemPrompt,
	})

	// Add user message
	state.Messages = append(state.Messages, Message{
		Role:    "user",
		Content: userMessage,
	})

	for state.Iteration < a.config.MaxIterations {
		state.Iteration++

		// Call LLM
		resp, err := a.config.Client.Chat(ctx, ChatRequest{
			Model:       a.config.Model,
			Messages:    state.Messages,
			Tools:       a.config.Tools,
			Temperature: a.config.Temperature,
		})
		if err != nil {
			state.Error = err
			if onEvent != nil {
				_ = onEvent(ReActEvent{Type: "error", State: state})
			}
			return state, err
		}

		state.Thought = resp.Message.Content
		state.Action = nil
		state.Observation = ""

		// Check if the model wants to use a tool
		if len(resp.Message.ToolCalls) > 0 {
			state.Action = &resp.Message.ToolCalls[0]

			if onEvent != nil {
				_ = onEvent(ReActEvent{Type: "thought", State: state})
				_ = onEvent(ReActEvent{Type: "action", State: state})
			}

			// Add assistant message with tool call
			state.Messages = append(state.Messages, resp.Message)

			// Execute tool
			result, err := a.config.ToolExecutor.Execute(ctx, *state.Action)
			if err != nil {
				state.Observation = "Error: " + err.Error()
			} else {
				state.Observation = result.Result
				if result.IsFinal {
					state.Answer = strings.TrimSpace(result.FinalAnswer)
					if state.Answer == "" {
						state.Answer = strings.TrimSpace(result.Result)
					}
					state.IsComplete = true
					if onEvent != nil {
						_ = onEvent(ReActEvent{Type: "answer", State: state})
					}
					return state, nil
				}
			}

			if onEvent != nil {
				_ = onEvent(ReActEvent{Type: "observation", State: state})
			}

			// Add tool result message
			state.Messages = append(state.Messages, Message{
				Role:       "tool",
				Content:    state.Observation,
				ToolCallID: state.Action.ID,
			})
		} else {
			// No tool call, the model has provided an answer
			state.Answer = resp.Message.Content
			state.IsComplete = true

			if onEvent != nil {
				_ = onEvent(ReActEvent{Type: "answer", State: state})
			}

			return state, nil
		}
	}

	// Max iterations reached
	state.Answer = state.Thought
	state.IsComplete = true
	return state, nil
}

// RunStream runs the ReAct loop with streaming.
func (a *ReActAgent) RunStream(ctx context.Context, userMessage string, onEvent func(ReActEvent) error) (string, error) {
	if a != nil && a.config.Client != nil && len(a.config.Tools) == 0 {
		messages := []Message{
			{
				Role:    "system",
				Content: a.buildSystemPrompt(),
			},
			{
				Role:    "user",
				Content: userMessage,
			},
		}
		state := &ReActState{
			Messages: messages,
		}
		resp, err := a.config.Client.ChatStream(ctx, ChatRequest{
			Model:       a.config.Model,
			Messages:    messages,
			Temperature: a.config.Temperature,
		}, func(event StreamEvent) error {
			if onEvent == nil {
				return nil
			}
			if event.Type != "delta" || event.Delta == "" {
				return nil
			}
			state.Answer += event.Delta
			state.IsComplete = false
			return onEvent(ReActEvent{
				Type:  "answer",
				State: state,
				Delta: event.Delta,
			})
		})
		if err != nil {
			state.Error = err
			if onEvent != nil {
				_ = onEvent(ReActEvent{Type: "error", State: state})
			}
			return "", err
		}
		state.Answer = resp.Message.Content
		state.IsComplete = true
		if onEvent != nil {
			_ = onEvent(ReActEvent{Type: "answer", State: state})
		}
		return state.Answer, nil
	}
	state, err := a.RunWithState(ctx, userMessage, onEvent)
	if err != nil {
		return "", err
	}
	return state.Answer, nil
}

func (a *ReActAgent) buildSystemPrompt() string {
	base := a.config.SystemPrompt
	if base == "" {
		base = "You are a helpful AI assistant."
	}

	toolDescriptions := ""
	if len(a.config.Tools) > 0 {
		toolDescriptions = "\n\nYou have access to the following tools:\n"
		for _, tool := range a.config.Tools {
			toolDescriptions += "- " + tool.Name + ": " + tool.Description + "\n"
		}
	}

	return base + toolDescriptions + `

Use the ReAct (Reasoning + Acting) pattern:

1. **Thought**: Think about what you need to do.
2. **Action**: If you need to use a tool, call it. Otherwise, provide your final answer.
3. **Observation**: Observe the result of your action.
4. Repeat until you can provide a final answer.

When you have a final answer, respond directly without using any tools.`
}

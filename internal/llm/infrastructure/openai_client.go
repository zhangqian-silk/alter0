package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	"alter0/internal/llm/domain"
)

// OpenAIClient implements domain.LLMClient using the OpenAI Go SDK.
type OpenAIClient struct {
	client      openai.Client
	model       string
	temperature *float64
	maxTokens   *int
}

// OpenAIClientConfig is the configuration for OpenAIClient.
type OpenAIClientConfig struct {
	APIKey      string
	BaseURL     string // For OpenAI-compatible APIs
	Model       string
	Temperature *float64
	MaxTokens   *int
}

// NewOpenAIClient creates a new OpenAI client.
func NewOpenAIClient(config OpenAIClientConfig) *OpenAIClient {
	opts := []option.RequestOption{
		option.WithAPIKey(config.APIKey),
	}
	if config.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(config.BaseURL))
	}

	return &OpenAIClient{
		client:      openai.NewClient(opts...),
		model:       config.Model,
		temperature: config.Temperature,
		maxTokens:   config.MaxTokens,
	}
}

// Chat performs a chat completion.
func (c *OpenAIClient) Chat(ctx context.Context, req domain.ChatRequest) (*domain.ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = c.model
	}
	if model == "" {
		return nil, errors.New("model is required")
	}

	// Convert messages
	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(req.Messages))
	for _, msg := range req.Messages {
		switch msg.Role {
		case "system":
			messages = append(messages, openai.SystemMessage(msg.Content))
		case "user":
			messages = append(messages, openai.UserMessage(msg.Content))
		case "assistant":
			messages = append(messages, openai.AssistantMessage(msg.Content))
		case "tool":
			messages = append(messages, openai.ToolMessage(msg.Content, msg.ToolCallID))
		}
	}

	// Build request params
	params := openai.ChatCompletionNewParams{
		Model:    model,
		Messages: messages,
	}

	if len(req.Tools) > 0 {
		params.Tools = c.convertTools(req.Tools)
	}

	if req.Temperature != nil {
		params.Temperature = openai.Float(*req.Temperature)
	} else if c.temperature != nil {
		params.Temperature = openai.Float(*c.temperature)
	}

	if req.MaxTokens != nil {
		params.MaxTokens = openai.Int(int64(*req.MaxTokens))
	} else if c.maxTokens != nil {
		params.MaxTokens = openai.Int(int64(*c.maxTokens))
	}

	// Make the request
	completion, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("openai chat completion: %w", err)
	}

	if len(completion.Choices) == 0 {
		return nil, errors.New("no choices in response")
	}

	choice := completion.Choices[0]
	resp := &domain.ChatResponse{
		ID:               completion.ID,
		PromptTokens:     int(completion.Usage.PromptTokens),
		CompletionTokens: int(completion.Usage.CompletionTokens),
		TotalTokens:      int(completion.Usage.TotalTokens),
		Message: domain.Message{
			Role:    "assistant",
			Content: choice.Message.Content,
		},
	}

	// Convert tool calls
	if len(choice.Message.ToolCalls) > 0 {
		resp.Message.ToolCalls = make([]domain.ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			// Extract function tool call
			if tc.Function.Name != "" {
				resp.Message.ToolCalls[i] = domain.ToolCall{
					ID:        tc.ID,
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				}
			}
		}
	}

	return resp, nil
}

// ChatStream performs a streaming chat completion.
func (c *OpenAIClient) ChatStream(ctx context.Context, req domain.ChatRequest, onEvent func(domain.StreamEvent) error) (*domain.ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = c.model
	}
	if model == "" {
		return nil, errors.New("model is required")
	}

	// Convert messages
	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(req.Messages))
	for _, msg := range req.Messages {
		switch msg.Role {
		case "system":
			messages = append(messages, openai.SystemMessage(msg.Content))
		case "user":
			messages = append(messages, openai.UserMessage(msg.Content))
		case "assistant":
			messages = append(messages, openai.AssistantMessage(msg.Content))
		case "tool":
			messages = append(messages, openai.ToolMessage(msg.Content, msg.ToolCallID))
		}
	}

	// Build request params
	params := openai.ChatCompletionNewParams{
		Model:    model,
		Messages: messages,
	}

	if len(req.Tools) > 0 {
		params.Tools = c.convertTools(req.Tools)
	}

	if req.Temperature != nil {
		params.Temperature = openai.Float(*req.Temperature)
	} else if c.temperature != nil {
		params.Temperature = openai.Float(*c.temperature)
	}

	if req.MaxTokens != nil {
		params.MaxTokens = openai.Int(int64(*req.MaxTokens))
	} else if c.maxTokens != nil {
		params.MaxTokens = openai.Int(int64(*c.maxTokens))
	}

	// Make the streaming request
	stream := c.client.Chat.Completions.NewStreaming(ctx, params)

	var fullContent string
	var toolCalls []domain.ToolCall
	var completionID string
	var promptTokens, completionTokens, totalTokens int

	for stream.Next() {
		chunk := stream.Current()
		completionID = chunk.ID

		if len(chunk.Choices) == 0 {
			continue
		}

		choice := chunk.Choices[0]
		if choice.Delta.Content != "" {
			fullContent += choice.Delta.Content
			if onEvent != nil {
				if err := onEvent(domain.StreamEvent{
					Type:  "delta",
					Delta: choice.Delta.Content,
				}); err != nil {
					return nil, err
				}
			}
		}

		// Handle tool calls in streaming
		for _, tc := range choice.Delta.ToolCalls {
			// Find or create tool call
			found := false
			for i := range toolCalls {
				if toolCalls[i].ID == tc.ID {
					toolCalls[i].Name = tc.Function.Name
					toolCalls[i].Arguments += tc.Function.Arguments
					found = true
					break
				}
			}
			if !found && tc.ID != "" {
				toolCalls = append(toolCalls, domain.ToolCall{
					ID:        tc.ID,
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				})
			}
		}

		// Update usage if available
		if chunk.Usage.PromptTokens > 0 {
			promptTokens = int(chunk.Usage.PromptTokens)
			completionTokens = int(chunk.Usage.CompletionTokens)
			totalTokens = int(chunk.Usage.TotalTokens)
		}
	}

	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("openai streaming: %w", err)
	}

	if onEvent != nil {
		_ = onEvent(domain.StreamEvent{Type: "done"})
	}

	return &domain.ChatResponse{
		ID:               completionID,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		Message: domain.Message{
			Role:      "assistant",
			Content:   fullContent,
			ToolCalls: toolCalls,
		},
	}, nil
}

func (c *OpenAIClient) convertTools(tools []domain.Tool) []openai.ChatCompletionToolUnionParam {
	result := make([]openai.ChatCompletionToolUnionParam, len(tools))
	for i, tool := range tools {
		result[i] = openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        tool.Name,
			Description: openai.String(tool.Description),
			Parameters:  openai.FunctionParameters(tool.Parameters),
		})
	}
	return result
}

// Close closes the client (no-op for this implementation).
func (c *OpenAIClient) Close() error {
	return nil
}

// Ensure OpenAIClient implements domain.LLMClient
var _ domain.LLMClient = (*OpenAIClient)(nil)
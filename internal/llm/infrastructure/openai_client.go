package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"

	"alter0/internal/llm/domain"
)

// OpenAIClient implements domain.LLMClient using the OpenAI Go SDK.
type OpenAIClient struct {
	client      openai.Client
	apiType     string
	model       string
	temperature *float64
	maxTokens   *int
}

// OpenAIClientConfig is the configuration for OpenAIClient.
type OpenAIClientConfig struct {
	APIKey      string
	APIType     string
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
	apiType := domain.DefaultProviderAPIType
	if config.APIType != "" {
		apiType = config.APIType
	}

	return &OpenAIClient{
		client:      openai.NewClient(opts...),
		apiType:     apiType,
		model:       config.Model,
		temperature: config.Temperature,
		maxTokens:   config.MaxTokens,
	}
}

// Chat performs a response request.
func (c *OpenAIClient) Chat(ctx context.Context, req domain.ChatRequest) (*domain.ChatResponse, error) {
	apiType := domain.DefaultProviderAPIType
	if c != nil && c.apiType != "" {
		apiType = c.apiType
	}
	switch apiType {
	case domain.ProviderAPITypeOpenAICompletions:
		return c.chatCompletions(ctx, req)
	case domain.ProviderAPITypeOpenAIResponses:
		return c.responsesChat(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported api_type: %s", apiType)
	}
}

// ChatStream performs a streaming response request.
func (c *OpenAIClient) ChatStream(ctx context.Context, req domain.ChatRequest, onEvent func(domain.StreamEvent) error) (*domain.ChatResponse, error) {
	apiType := domain.DefaultProviderAPIType
	if c != nil && c.apiType != "" {
		apiType = c.apiType
	}
	switch apiType {
	case domain.ProviderAPITypeOpenAICompletions:
		return c.chatCompletionsStream(ctx, req, onEvent)
	case domain.ProviderAPITypeOpenAIResponses:
		return c.responsesChatStream(ctx, req, onEvent)
	default:
		return nil, fmt.Errorf("unsupported api_type: %s", apiType)
	}
}

func (c *OpenAIClient) responsesChat(ctx context.Context, req domain.ChatRequest) (*domain.ChatResponse, error) {
	params, err := c.buildResponseParams(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Responses.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("openai responses: %w", err)
	}

	return convertResponse(resp), nil
}

func (c *OpenAIClient) responsesChatStream(ctx context.Context, req domain.ChatRequest, onEvent func(domain.StreamEvent) error) (*domain.ChatResponse, error) {
	params, err := c.buildResponseParams(req)
	if err != nil {
		return nil, err
	}

	stream := c.client.Responses.NewStreaming(ctx, params)

	var finalResponse *responses.Response
	for stream.Next() {
		event := stream.Current()
		switch item := event.AsAny().(type) {
		case responses.ResponseTextDeltaEvent:
			if item.Delta == "" || onEvent == nil {
				continue
			}
			if err := onEvent(domain.StreamEvent{
				Type:  "delta",
				Delta: item.Delta,
			}); err != nil {
				return nil, err
			}
		case responses.ResponseCompletedEvent:
			finalResponse = &item.Response
		case responses.ResponseFailedEvent:
			finalResponse = &item.Response
		case responses.ResponseErrorEvent:
			return nil, fmt.Errorf("openai streaming: %s", item.Message)
		}
	}

	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("openai streaming: %w", err)
	}

	if onEvent != nil {
		_ = onEvent(domain.StreamEvent{Type: "done"})
	}

	if finalResponse != nil {
		return convertResponse(finalResponse), nil
	}

	return &domain.ChatResponse{
		Message: domain.Message{
			Role: "assistant",
		},
	}, nil
}

func (c *OpenAIClient) chatCompletions(ctx context.Context, req domain.ChatRequest) (*domain.ChatResponse, error) {
	params, err := c.buildChatCompletionParams(req)
	if err != nil {
		return nil, err
	}

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

	if len(choice.Message.ToolCalls) > 0 {
		resp.Message.ToolCalls = make([]domain.ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
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

func (c *OpenAIClient) chatCompletionsStream(ctx context.Context, req domain.ChatRequest, onEvent func(domain.StreamEvent) error) (*domain.ChatResponse, error) {
	params, err := c.buildChatCompletionParams(req)
	if err != nil {
		return nil, err
	}

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

		for _, tc := range choice.Delta.ToolCalls {
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

func (c *OpenAIClient) buildResponseParams(req domain.ChatRequest) (responses.ResponseNewParams, error) {
	model := req.Model
	if model == "" {
		model = c.model
	}
	if model == "" {
		return responses.ResponseNewParams{}, errors.New("model is required")
	}

	input := make(responses.ResponseInputParam, 0, len(req.Messages))
	for _, msg := range req.Messages {
		items := convertMessageToResponseInput(msg)
		input = append(input, items...)
	}

	params := responses.ResponseNewParams{
		Model: model,
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: input,
		},
	}

	if req.Temperature != nil {
		params.Temperature = openai.Float(*req.Temperature)
	} else if c.temperature != nil {
		params.Temperature = openai.Float(*c.temperature)
	}

	if req.MaxTokens != nil {
		params.MaxOutputTokens = openai.Int(int64(*req.MaxTokens))
	} else if c.maxTokens != nil {
		params.MaxOutputTokens = openai.Int(int64(*c.maxTokens))
	}

	if req.PreviousResponseID != "" {
		params.PreviousResponseID = openai.String(req.PreviousResponseID)
	}

	if len(req.Tools) > 0 {
		params.Tools = c.convertTools(req.Tools)
		params.ParallelToolCalls = openai.Bool(false)
	}

	return params, nil
}

func (c *OpenAIClient) buildChatCompletionParams(req domain.ChatRequest) (openai.ChatCompletionNewParams, error) {
	model := req.Model
	if model == "" {
		model = c.model
	}
	if model == "" {
		return openai.ChatCompletionNewParams{}, errors.New("model is required")
	}

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

	params := openai.ChatCompletionNewParams{
		Model:    model,
		Messages: messages,
	}

	if len(req.Tools) > 0 {
		params.Tools = c.convertCompletionTools(req.Tools)
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

	return params, nil
}

func convertMessageToResponseInput(msg domain.Message) []responses.ResponseInputItemUnionParam {
	items := make([]responses.ResponseInputItemUnionParam, 0, 2)

	switch msg.Role {
	case "system":
		if msg.Content != "" {
			items = append(items, responses.ResponseInputItemParamOfMessage(msg.Content, responses.EasyInputMessageRoleSystem))
		}
	case "user":
		if msg.Content != "" {
			items = append(items, responses.ResponseInputItemParamOfMessage(msg.Content, responses.EasyInputMessageRoleUser))
		}
	case "assistant":
		if msg.Content != "" {
			items = append(items, responses.ResponseInputItemParamOfMessage(msg.Content, responses.EasyInputMessageRoleAssistant))
		}
		for _, toolCall := range msg.ToolCalls {
			items = append(items, responses.ResponseInputItemParamOfFunctionCall(toolCall.Arguments, toolCall.ID, toolCall.Name))
		}
	case "tool":
		if msg.ToolCallID != "" {
			items = append(items, responses.ResponseInputItemParamOfFunctionCallOutput(msg.ToolCallID, msg.Content))
		}
	}

	return items
}

func convertResponse(resp *responses.Response) *domain.ChatResponse {
	if resp == nil {
		return &domain.ChatResponse{
			Message: domain.Message{Role: "assistant"},
		}
	}

	toolCalls := make([]domain.ToolCall, 0)
	for _, item := range resp.Output {
		if item.Type != "function_call" {
			continue
		}
		call := item.AsFunctionCall()
		toolCalls = append(toolCalls, domain.ToolCall{
			ID:        call.CallID,
			Name:      call.Name,
			Arguments: call.Arguments,
		})
	}

	return &domain.ChatResponse{
		ID:               resp.ID,
		PromptTokens:     int(resp.Usage.InputTokens),
		CompletionTokens: int(resp.Usage.OutputTokens),
		TotalTokens:      int(resp.Usage.TotalTokens),
		Message: domain.Message{
			Role:      "assistant",
			Content:   resp.OutputText(),
			ToolCalls: toolCalls,
		},
	}
}

func (c *OpenAIClient) convertTools(tools []domain.Tool) []responses.ToolUnionParam {
	result := make([]responses.ToolUnionParam, len(tools))
	for i, tool := range tools {
		result[i] = responses.ToolUnionParam{
			OfFunction: &responses.FunctionToolParam{
				Name:        tool.Name,
				Description: openai.String(tool.Description),
				Parameters:  tool.Parameters,
				Strict:      openai.Bool(false),
			},
		}
	}
	return result
}

func (c *OpenAIClient) convertCompletionTools(tools []domain.Tool) []openai.ChatCompletionToolUnionParam {
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

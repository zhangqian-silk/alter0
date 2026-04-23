package infrastructure

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"alter0/internal/llm/domain"
)

func TestOpenAIClientChatUsesResponsesAPI(t *testing.T) {
	t.Parallel()

	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("expected /v1/responses path, got %s", r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if err := json.Unmarshal(body, &requestBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"resp_123",
			"object":"response",
			"model":"gpt-4o",
			"output":[
				{
					"id":"msg_1",
					"type":"message",
					"role":"assistant",
					"status":"completed",
					"content":[{"type":"output_text","text":"done","annotations":[]}]
				},
				{
					"id":"fc_1",
					"type":"function_call",
					"call_id":"call_2",
					"name":"lookup",
					"arguments":"{\"q\":\"weather\"}",
					"status":"completed"
				}
			],
			"usage":{
				"input_tokens":11,
				"input_tokens_details":{"cached_tokens":0},
				"output_tokens":7,
				"output_tokens_details":{"reasoning_tokens":0},
				"total_tokens":18
			}
		}`))
	}))
	defer server.Close()

	client := NewOpenAIClient(OpenAIClientConfig{
		APIKey:  "sk-test",
		APIType: domain.ProviderAPITypeOpenAIResponses,
		BaseURL: server.URL + "/v1",
		Model:   "gpt-4o",
	})

	resp, err := client.Chat(context.Background(), domain.ChatRequest{
		Messages: []domain.Message{
			{Role: "system", Content: "system prompt"},
			{Role: "user", Content: "hello"},
			{
				Role:    "assistant",
				Content: "calling tool",
				ToolCalls: []domain.ToolCall{
					{ID: "call_1", Name: "lookup", Arguments: `{"q":"hello"}`},
				},
			},
			{Role: "tool", ToolCallID: "call_1", Content: `{"result":"ok"}`},
		},
		Tools: []domain.Tool{
			{
				Name:        "lookup",
				Description: "Lookup data",
				Parameters: map[string]interface{}{
					"type": "object",
				},
			},
		},
		PreviousResponseID: "resp_prev",
	})
	if err != nil {
		t.Fatalf("chat request failed: %v", err)
	}

	if requestBody["previous_response_id"] != "resp_prev" {
		t.Fatalf("expected previous_response_id to be forwarded, got %#v", requestBody["previous_response_id"])
	}

	inputItems, ok := requestBody["input"].([]any)
	if !ok {
		t.Fatalf("expected input array, got %#v", requestBody["input"])
	}
	if len(inputItems) != 5 {
		t.Fatalf("expected 5 input items, got %d", len(inputItems))
	}

	assertInputRole(t, inputItems[0], "system")
	assertInputRole(t, inputItems[1], "user")
	assertInputRole(t, inputItems[2], "assistant")
	assertInputType(t, inputItems[3], "function_call")
	assertInputType(t, inputItems[4], "function_call_output")

	tools, ok := requestBody["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("expected one tool definition, got %#v", requestBody["tools"])
	}
	tool := tools[0].(map[string]any)
	if tool["type"] != "function" {
		t.Fatalf("expected function tool, got %#v", tool["type"])
	}

	if resp.ID != "resp_123" {
		t.Fatalf("expected response id resp_123, got %q", resp.ID)
	}
	if resp.Message.Content != "done" {
		t.Fatalf("expected response text 'done', got %q", resp.Message.Content)
	}
	if len(resp.Message.ToolCalls) != 1 {
		t.Fatalf("expected one tool call, got %d", len(resp.Message.ToolCalls))
	}
	if resp.Message.ToolCalls[0].ID != "call_2" {
		t.Fatalf("expected tool call id call_2, got %q", resp.Message.ToolCalls[0].ID)
	}
	if resp.PromptTokens != 11 || resp.CompletionTokens != 7 || resp.TotalTokens != 18 {
		t.Fatalf("unexpected usage: %+v", resp)
	}
}

func TestOpenAIClientResponsesIncludesUserImageParts(t *testing.T) {
	t.Parallel()

	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if err := json.Unmarshal(body, &requestBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"resp_img",
			"object":"response",
			"model":"gpt-4o",
			"output":[{"id":"msg_1","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"done","annotations":[]}]}]
		}`))
	}))
	defer server.Close()

	client := NewOpenAIClient(OpenAIClientConfig{
		APIKey:  "sk-test",
		APIType: domain.ProviderAPITypeOpenAIResponses,
		BaseURL: server.URL + "/v1",
		Model:   "gpt-4o",
	})

	_, err := client.Chat(context.Background(), domain.ChatRequest{
		Messages: []domain.Message{{
			Role:    "user",
			Content: "describe this image",
			Parts: []domain.MessagePart{
				{Type: domain.MessagePartTypeText, Text: "describe this image"},
				{Type: domain.MessagePartTypeImage, ImageURL: "data:image/png;base64,ZmFrZQ=="},
			},
		}},
	})
	if err != nil {
		t.Fatalf("chat request failed: %v", err)
	}

	inputItems, ok := requestBody["input"].([]any)
	if !ok || len(inputItems) != 1 {
		t.Fatalf("expected one input item, got %#v", requestBody["input"])
	}
	item, ok := inputItems[0].(map[string]any)
	if !ok {
		t.Fatalf("expected message item map, got %#v", inputItems[0])
	}
	content, ok := item["content"].([]any)
	if !ok || len(content) != 2 {
		t.Fatalf("expected content list with text and image, got %#v", item["content"])
	}
	imagePart, ok := content[1].(map[string]any)
	if !ok {
		t.Fatalf("expected image content map, got %#v", content[1])
	}
	if imagePart["type"] != "input_image" {
		t.Fatalf("expected input_image part, got %#v", imagePart["type"])
	}
	if imagePart["image_url"] != "data:image/png;base64,ZmFrZQ==" {
		t.Fatalf("expected data URL image, got %#v", imagePart["image_url"])
	}
}

func TestOpenAIClientChatCompletionsIncludesUserImageParts(t *testing.T) {
	t.Parallel()

	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if err := json.Unmarshal(body, &requestBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl_img",
			"choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"ok"}}],
			"usage":{"prompt_tokens":4,"completion_tokens":2,"total_tokens":6}
		}`))
	}))
	defer server.Close()

	client := NewOpenAIClient(OpenAIClientConfig{
		APIKey:  "sk-test",
		APIType: domain.ProviderAPITypeOpenAICompletions,
		BaseURL: server.URL + "/v1",
		Model:   "gpt-4o",
	})

	_, err := client.Chat(context.Background(), domain.ChatRequest{
		Messages: []domain.Message{{
			Role: "user",
			Parts: []domain.MessagePart{
				{Type: domain.MessagePartTypeText, Text: "what is in this image"},
				{Type: domain.MessagePartTypeImage, ImageURL: "data:image/png;base64,ZmFrZQ=="},
			},
		}},
	})
	if err != nil {
		t.Fatalf("chat request failed: %v", err)
	}

	messages, ok := requestBody["messages"].([]any)
	if !ok || len(messages) != 1 {
		t.Fatalf("expected one chat message, got %#v", requestBody["messages"])
	}
	userMessage, ok := messages[0].(map[string]any)
	if !ok {
		t.Fatalf("expected user message map, got %#v", messages[0])
	}
	content, ok := userMessage["content"].([]any)
	if !ok || len(content) != 2 {
		t.Fatalf("expected multipart user content, got %#v", userMessage["content"])
	}
	imagePart, ok := content[1].(map[string]any)
	if !ok {
		t.Fatalf("expected image content map, got %#v", content[1])
	}
	if imagePart["type"] != "image_url" {
		t.Fatalf("expected image_url part, got %#v", imagePart["type"])
	}
}

func TestOpenAIClientChatStreamUsesResponsesAPI(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("expected /v1/responses path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		stream := strings.Join([]string{
			`data: {"type":"response.output_text.delta","delta":"Hel","content_index":0,"item_id":"msg_1","output_index":0,"sequence_number":1}`,
			"",
			`data: {"type":"response.output_text.delta","delta":"lo","content_index":0,"item_id":"msg_1","output_index":0,"sequence_number":2}`,
			"",
			`data: {"type":"response.completed","sequence_number":3,"response":{"id":"resp_stream","object":"response","model":"gpt-4o","output":[{"id":"msg_1","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"Hello","annotations":[]}]}],"usage":{"input_tokens":3,"input_tokens_details":{"cached_tokens":0},"output_tokens":2,"output_tokens_details":{"reasoning_tokens":0},"total_tokens":5}}}`,
			"",
			"",
		}, "\n")
		_, _ = w.Write([]byte(stream))
	}))
	defer server.Close()

	client := NewOpenAIClient(OpenAIClientConfig{
		APIKey:  "sk-test",
		APIType: domain.ProviderAPITypeOpenAIResponses,
		BaseURL: server.URL + "/v1",
		Model:   "gpt-4o",
	})

	var deltas []string
	resp, err := client.ChatStream(context.Background(), domain.ChatRequest{
		Messages: []domain.Message{
			{Role: "user", Content: "hello"},
		},
	}, func(event domain.StreamEvent) error {
		if event.Type == "delta" {
			deltas = append(deltas, event.Delta)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("stream request failed: %v", err)
	}

	if strings.Join(deltas, "") != "Hello" {
		t.Fatalf("expected streamed deltas to form Hello, got %q", strings.Join(deltas, ""))
	}
	if resp.ID != "resp_stream" {
		t.Fatalf("expected response id resp_stream, got %q", resp.ID)
	}
	if resp.Message.Content != "Hello" {
		t.Fatalf("expected final content Hello, got %q", resp.Message.Content)
	}
}

func TestOpenAIClientChatUsesChatCompletionsAPI(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("expected /v1/chat/completions path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl_123",
			"choices":[
				{
					"index":0,
					"finish_reason":"tool_calls",
					"message":{
						"role":"assistant",
						"content":"completion text",
						"tool_calls":[
							{
								"id":"call_completion",
								"type":"function",
								"function":{"name":"lookup","arguments":"{\"q\":\"hello\"}"}
							}
						]
					}
				}
			],
			"usage":{"prompt_tokens":5,"completion_tokens":6,"total_tokens":11}
		}`))
	}))
	defer server.Close()

	client := NewOpenAIClient(OpenAIClientConfig{
		APIKey:  "sk-test",
		APIType: domain.ProviderAPITypeOpenAICompletions,
		BaseURL: server.URL + "/v1",
		Model:   "gpt-4o",
	})

	resp, err := client.Chat(context.Background(), domain.ChatRequest{
		Messages: []domain.Message{{Role: "user", Content: "hello"}},
		Tools: []domain.Tool{{
			Name:        "lookup",
			Description: "Lookup data",
			Parameters: map[string]interface{}{
				"type": "object",
			},
		}},
	})
	if err != nil {
		t.Fatalf("chat completions request failed: %v", err)
	}

	if resp.ID != "chatcmpl_123" {
		t.Fatalf("expected response id chatcmpl_123, got %q", resp.ID)
	}
	if resp.Message.Content != "completion text" {
		t.Fatalf("expected completion text, got %q", resp.Message.Content)
	}
	if len(resp.Message.ToolCalls) != 1 || resp.Message.ToolCalls[0].ID != "call_completion" {
		t.Fatalf("expected tool call from chat completions, got %+v", resp.Message.ToolCalls)
	}
}

func TestOpenAIClientChatCompletionsPreservesAssistantToolCallsInRequest(t *testing.T) {
	t.Parallel()

	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("expected /v1/chat/completions path, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if err := json.Unmarshal(body, &requestBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl_followup",
			"choices":[
				{
					"index":0,
					"finish_reason":"stop",
					"message":{"role":"assistant","content":"done"}
				}
			],
			"usage":{"prompt_tokens":8,"completion_tokens":3,"total_tokens":11}
		}`))
	}))
	defer server.Close()

	client := NewOpenAIClient(OpenAIClientConfig{
		APIKey:  "sk-test",
		APIType: domain.ProviderAPITypeOpenAICompletions,
		BaseURL: server.URL + "/v1",
		Model:   "gpt-4o",
	})

	_, err := client.Chat(context.Background(), domain.ChatRequest{
		Messages: []domain.Message{
			{Role: "system", Content: "system prompt"},
			{Role: "user", Content: "hello"},
			{
				Role:    "assistant",
				Content: "calling tool",
				ToolCalls: []domain.ToolCall{
					{ID: "call_1", Name: "lookup", Arguments: `{"q":"hello"}`},
				},
			},
			{Role: "tool", ToolCallID: "call_1", Content: `{"result":"ok"}`},
		},
		Tools: []domain.Tool{{
			Name:        "lookup",
			Description: "Lookup data",
			Parameters: map[string]interface{}{
				"type": "object",
			},
		}},
	})
	if err != nil {
		t.Fatalf("chat completions request failed: %v", err)
	}

	messages, ok := requestBody["messages"].([]any)
	if !ok {
		t.Fatalf("expected messages array, got %#v", requestBody["messages"])
	}
	if len(messages) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(messages))
	}

	assistant, ok := messages[2].(map[string]any)
	if !ok {
		t.Fatalf("expected assistant message map, got %#v", messages[2])
	}
	toolCalls, ok := assistant["tool_calls"].([]any)
	if !ok || len(toolCalls) != 1 {
		t.Fatalf("expected one assistant tool call, got %#v", assistant["tool_calls"])
	}
	call, ok := toolCalls[0].(map[string]any)
	if !ok {
		t.Fatalf("expected tool call map, got %#v", toolCalls[0])
	}
	if call["id"] != "call_1" {
		t.Fatalf("expected tool call id call_1, got %#v", call["id"])
	}
	if call["type"] != "function" {
		t.Fatalf("expected tool call type function, got %#v", call["type"])
	}
	function, ok := call["function"].(map[string]any)
	if !ok {
		t.Fatalf("expected function payload, got %#v", call["function"])
	}
	if function["name"] != "lookup" {
		t.Fatalf("expected function name lookup, got %#v", function["name"])
	}
	if function["arguments"] != `{"q":"hello"}` {
		t.Fatalf("expected function arguments to be preserved, got %#v", function["arguments"])
	}

	toolMessage, ok := messages[3].(map[string]any)
	if !ok {
		t.Fatalf("expected tool message map, got %#v", messages[3])
	}
	if toolMessage["tool_call_id"] != "call_1" {
		t.Fatalf("expected tool_call_id call_1, got %#v", toolMessage["tool_call_id"])
	}
}

func TestOpenAIClientChatUsesOpenRouterHeadersAndRoutingOptions(t *testing.T) {
	t.Parallel()

	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/chat/completions" {
			t.Fatalf("expected /api/v1/chat/completions path, got %s", r.URL.Path)
		}
		if got := r.Header.Get("HTTP-Referer"); got != "https://alter0.example" {
			t.Fatalf("expected HTTP-Referer header, got %q", got)
		}
		if got := r.Header.Get("X-OpenRouter-Title"); got != "Alter0" {
			t.Fatalf("expected X-OpenRouter-Title header, got %q", got)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if err := json.Unmarshal(body, &requestBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl_router",
			"choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"ok"}}],
			"usage":{"prompt_tokens":4,"completion_tokens":2,"total_tokens":6}
		}`))
	}))
	defer server.Close()

	allowFallbacks := true
	requireParameters := true
	client := NewOpenAIClient(OpenAIClientConfig{
		APIKey:       "sk-or-test",
		ProviderType: domain.ProviderTypeOpenRouter,
		APIType:      domain.ProviderAPITypeOpenAICompletions,
		BaseURL:      server.URL + "/api/v1",
		Model:        "openai/gpt-5.4",
		OpenRouter: &domain.OpenRouterConfig{
			SiteURL:           "https://alter0.example",
			AppName:           "Alter0",
			FallbackModels:    []string{"anthropic/claude-3.7-sonnet", "google/gemini-2.5-pro"},
			ProviderOrder:     []string{"openai", "anthropic"},
			AllowFallbacks:    &allowFallbacks,
			RequireParameters: &requireParameters,
		},
	})

	resp, err := client.Chat(context.Background(), domain.ChatRequest{
		Messages: []domain.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("chat request failed: %v", err)
	}

	models, ok := requestBody["models"].([]any)
	if !ok || len(models) != 2 {
		t.Fatalf("expected fallback models in request body, got %#v", requestBody["models"])
	}
	provider, ok := requestBody["provider"].(map[string]any)
	if !ok {
		t.Fatalf("expected provider routing object, got %#v", requestBody["provider"])
	}
	order, ok := provider["order"].([]any)
	if !ok || len(order) != 2 {
		t.Fatalf("expected provider order, got %#v", provider["order"])
	}
	if provider["allow_fallbacks"] != true {
		t.Fatalf("expected allow_fallbacks true, got %#v", provider["allow_fallbacks"])
	}
	if provider["require_parameters"] != true {
		t.Fatalf("expected require_parameters true, got %#v", provider["require_parameters"])
	}
	if resp.Message.Content != "ok" {
		t.Fatalf("expected response content ok, got %q", resp.Message.Content)
	}
}

func assertInputType(t *testing.T, item any, expected string) {
	t.Helper()

	entry, ok := item.(map[string]any)
	if !ok {
		t.Fatalf("expected map input item, got %#v", item)
	}
	if entry["type"] != expected {
		t.Fatalf("expected input item type %q, got %#v", expected, entry["type"])
	}
}

func assertInputRole(t *testing.T, item any, expected string) {
	t.Helper()

	entry, ok := item.(map[string]any)
	if !ok {
		t.Fatalf("expected map input item, got %#v", item)
	}
	if entry["role"] != expected {
		t.Fatalf("expected input item role %q, got %#v", expected, entry["role"])
	}
}

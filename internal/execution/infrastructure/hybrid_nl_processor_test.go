package infrastructure

import (
	"context"
	"strings"
	"testing"

	agentapp "alter0/internal/agent/application"
	controlapp "alter0/internal/control/application"
	controldomain "alter0/internal/control/domain"
	execdomain "alter0/internal/execution/domain"
	llmdomain "alter0/internal/llm/domain"
)

type stubReactFactory struct {
	client         llmdomain.LLMClient
	lastConfig     llmdomain.ReActAgentConfig
	lastProviderID string
}

func (s *stubReactFactory) GetReActAgent(_ context.Context, providerID string, config llmdomain.ReActAgentConfig) (*llmdomain.ReActAgent, error) {
	s.lastConfig = config
	s.lastProviderID = providerID
	s.lastConfig.Client = nil
	config.Client = s.client
	if config.Model == "" {
		config.Model = "test-model"
	}
	return llmdomain.NewReActAgent(config), nil
}

type scriptedLLMClient struct {
	call int
}

func (c *scriptedLLMClient) Chat(_ context.Context, req llmdomain.ChatRequest) (*llmdomain.ChatResponse, error) {
	c.call++
	switch c.call {
	case 1:
		return &llmdomain.ChatResponse{
			Message: llmdomain.Message{
				Role: "assistant",
				ToolCalls: []llmdomain.ToolCall{
					{ID: "call-1", Name: "codex_exec", Arguments: `{"instruction":"整理仓库"}`},
				},
			},
		}, nil
	case 2:
		last := req.Messages[len(req.Messages)-1]
		if last.Role != "tool" || !strings.Contains(last.Content, "mock response") {
			return nil, errUnexpectedToolObservation
		}
		return &llmdomain.ChatResponse{
			Message: llmdomain.Message{
				Role: "assistant",
				ToolCalls: []llmdomain.ToolCall{
					{ID: "call-2", Name: "complete", Arguments: `{"result":"任务已完成"}`},
				},
			},
		}, nil
	default:
		return &llmdomain.ChatResponse{
			Message: llmdomain.Message{
				Role:    "assistant",
				Content: "unexpected",
			},
		}, nil
	}
}

func (c *scriptedLLMClient) ChatStream(_ context.Context, _ llmdomain.ChatRequest, _ func(llmdomain.StreamEvent) error) (*llmdomain.ChatResponse, error) {
	return nil, nil
}

func (c *scriptedLLMClient) Close() error {
	return nil
}

type answerOnlyLLMClient struct{}

func (c *answerOnlyLLMClient) Chat(_ context.Context, _ llmdomain.ChatRequest) (*llmdomain.ChatResponse, error) {
	return &llmdomain.ChatResponse{
		Message: llmdomain.Message{
			Role:    "assistant",
			Content: "已按指定模型执行",
		},
	}, nil
}

func (c *answerOnlyLLMClient) ChatStream(_ context.Context, _ llmdomain.ChatRequest, _ func(llmdomain.StreamEvent) error) (*llmdomain.ChatResponse, error) {
	return nil, nil
}

func (c *answerOnlyLLMClient) Close() error {
	return nil
}

type delegatingLLMClient struct {
	call int
}

func (c *delegatingLLMClient) Chat(_ context.Context, req llmdomain.ChatRequest) (*llmdomain.ChatResponse, error) {
	if len(req.Messages) > 0 && strings.Contains(req.Messages[0].Content, "engineering delivery") {
		return &llmdomain.ChatResponse{
			Message: llmdomain.Message{
				Role:    "assistant",
				Content: "coding agent completed",
			},
		}, nil
	}
	c.call++
	if c.call == 1 {
		return &llmdomain.ChatResponse{
			Message: llmdomain.Message{
				Role: "assistant",
				ToolCalls: []llmdomain.ToolCall{
					{ID: "delegate-1", Name: "delegate_agent", Arguments: `{"agent_id":"coding","task":"Implement the requested code change"}`},
				},
			},
		}, nil
	}
	last := req.Messages[len(req.Messages)-1]
	if last.Role != "tool" || !strings.Contains(last.Content, "coding agent completed") {
		return nil, errUnexpectedToolObservation
	}
	return &llmdomain.ChatResponse{
		Message: llmdomain.Message{
			Role: "assistant",
			ToolCalls: []llmdomain.ToolCall{
				{ID: "complete-1", Name: "complete", Arguments: `{"result":"Delegation wrapped successfully"}`},
			},
		},
	}, nil
}

func (c *delegatingLLMClient) ChatStream(_ context.Context, _ llmdomain.ChatRequest, _ func(llmdomain.StreamEvent) error) (*llmdomain.ChatResponse, error) {
	return nil, nil
}

func (c *delegatingLLMClient) Close() error {
	return nil
}

var errUnexpectedToolObservation = &testError{text: "unexpected tool observation"}

type testError struct {
	text string
}

func (e *testError) Error() string {
	return e.text
}

func TestHybridNLProcessorAgentModeExecutesCodexToolLoop(t *testing.T) {
	reactFactory := &stubReactFactory{client: &scriptedLLMClient{}}
	processor := NewHybridNLProcessor(newTestProcessor("success", "整理仓库"), reactFactory, nil)

	metadata := testRuntimeMetadata()
	metadata[execdomain.AgentIDMetadataKey] = "researcher"
	metadata[execdomain.ExecutionEngineMetadataKey] = execdomain.ExecutionEngineAgent
	metadata[execdomain.AgentSystemPromptMetadataKey] = "先执行，再汇报。"
	metadata[execdomain.AgentToolsMetadataKey] = `["codex_exec"]`

	output, err := processor.Process(context.Background(), "完成仓库整理", metadata)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if output != "任务已完成" {
		t.Fatalf("Process() output = %q, want %q", output, "任务已完成")
	}
	if !strings.Contains(reactFactory.lastConfig.SystemPrompt, "先执行，再汇报。") {
		t.Fatalf("expected custom system prompt in config, got %q", reactFactory.lastConfig.SystemPrompt)
	}
	if len(reactFactory.lastConfig.Tools) != 2 {
		t.Fatalf("expected explicit codex_exec selection plus complete, got %+v", reactFactory.lastConfig.Tools)
	}
	if reactFactory.lastConfig.Tools[0].Name != "codex_exec" || reactFactory.lastConfig.Tools[1].Name != "complete" {
		t.Fatalf("unexpected tools: %+v", reactFactory.lastConfig.Tools)
	}
}

func TestHybridNLProcessorUsesChatLevelModelOverride(t *testing.T) {
	reactFactory := &stubReactFactory{client: &answerOnlyLLMClient{}}
	processor := NewHybridNLProcessor(newTestProcessor("success", "整理仓库"), reactFactory, nil)

	metadata := testRuntimeMetadata()
	metadata[execdomain.LLMProviderIDMetadataKey] = "openai"
	metadata[execdomain.LLMModelMetadataKey] = "gpt-5.4"

	_, err := processor.Process(context.Background(), "总结当前改动", metadata)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if reactFactory.lastProviderID != "openai" {
		t.Fatalf("expected provider override openai, got %s", reactFactory.lastProviderID)
	}
	if reactFactory.lastConfig.Model != "gpt-5.4" {
		t.Fatalf("expected model override gpt-5.4, got %s", reactFactory.lastConfig.Model)
	}
}

func TestHybridNLProcessorReactModeInjectsSelectedNativeTools(t *testing.T) {
	reactFactory := &stubReactFactory{client: &answerOnlyLLMClient{}}
	processor := NewHybridNLProcessor(newTestProcessor("success", "整理仓库"), reactFactory, nil)

	metadata := testRuntimeMetadata()
	metadata[execdomain.AgentToolsMetadataKey] = `["read","bash"]`

	_, err := processor.Process(context.Background(), "读取文件并总结", metadata)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if len(reactFactory.lastConfig.Tools) != 2 {
		t.Fatalf("expected selected native tools only, got %+v", reactFactory.lastConfig.Tools)
	}
	if reactFactory.lastConfig.Tools[0].Name != "read" || reactFactory.lastConfig.Tools[1].Name != "bash" {
		t.Fatalf("unexpected tool order: %+v", reactFactory.lastConfig.Tools)
	}
	if reactFactory.lastConfig.ToolExecutor == nil {
		t.Fatalf("expected tool executor to be configured")
	}
	if reactFactory.lastConfig.MaxIterations != 6 {
		t.Fatalf("expected max iterations 6 for tool-enabled chat, got %d", reactFactory.lastConfig.MaxIterations)
	}
}

func TestHybridNLProcessorAgentModeDefaultsToCoreTools(t *testing.T) {
	reactFactory := &stubReactFactory{client: &answerOnlyLLMClient{}}
	processor := NewHybridNLProcessor(newTestProcessor("success", "整理仓库"), reactFactory, nil)

	metadata := testRuntimeMetadata()
	metadata[execdomain.AgentIDMetadataKey] = "researcher"
	metadata[execdomain.ExecutionEngineMetadataKey] = execdomain.ExecutionEngineAgent

	_, err := processor.Process(context.Background(), "完成仓库整理", metadata)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	toolNames := []string{}
	for _, item := range reactFactory.lastConfig.Tools {
		toolNames = append(toolNames, item.Name)
	}
	for _, expected := range []string{"list_dir", "read", "write", "edit", "bash", "codex_exec", "complete"} {
		if !strings.Contains(strings.Join(toolNames, ","), expected) {
			t.Fatalf("expected tool %s in %+v", expected, toolNames)
		}
	}
}

func TestHybridNLProcessorMarksModelExecutionSource(t *testing.T) {
	reactFactory := &stubReactFactory{client: &answerOnlyLLMClient{}}
	processor := NewHybridNLProcessor(newTestProcessor("success", "整理仓库"), reactFactory, nil)

	metadata := testRuntimeMetadata()
	output, err := processor.Process(context.Background(), "总结当前改动", metadata)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if output == "" {
		t.Fatal("expected non-empty output")
	}
	if got := metadata[execdomain.ExecutionSourceMetadataKey]; got != execdomain.ExecutionSourceModel {
		t.Fatalf("expected execution source %q, got %q", execdomain.ExecutionSourceModel, got)
	}
	if reactFactory.lastConfig.UserMessagePuller == nil {
		t.Fatalf("expected react user message puller to be configured")
	}
}

func TestHybridNLProcessorMarksCodexExecutionSource(t *testing.T) {
	processor := NewHybridNLProcessor(newTestProcessor("success", "整理仓库"), nil, nil)

	metadata := testRuntimeMetadata()
	output, err := processor.Process(context.Background(), "整理仓库", metadata)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if output == "" {
		t.Fatal("expected non-empty output")
	}
	if got := metadata[execdomain.ExecutionSourceMetadataKey]; got != execdomain.ExecutionSourceCodexCLI {
		t.Fatalf("expected execution source %q, got %q", execdomain.ExecutionSourceCodexCLI, got)
	}
}

func TestHybridNLProcessorAgentModeSupportsDelegation(t *testing.T) {
	control := controlapp.NewService()
	if err := control.UpsertAgent(controldomain.Agent{
		ID:          "managed-reviewer",
		Name:        "Managed Reviewer",
		Enabled:     true,
		Type:        controldomain.CapabilityTypeAgent,
		Scope:       controldomain.CapabilityScopeGlobal,
		Version:     controldomain.DefaultCapabilityVersion,
		Delegatable: true,
	}); err != nil {
		t.Fatalf("upsert managed agent failed: %v", err)
	}
	catalog := agentapp.NewCatalog(control)
	reactFactory := &stubReactFactory{client: &delegatingLLMClient{}}
	processor := NewHybridNLProcessorWithCatalog(newTestProcessor("success", "整理仓库"), reactFactory, catalog, nil)

	metadata := testRuntimeMetadata()
	metadata[execdomain.AgentIDMetadataKey] = "main"
	metadata[execdomain.ExecutionEngineMetadataKey] = execdomain.ExecutionEngineAgent
	metadata[execdomain.AgentToolsMetadataKey] = `["delegate_agent"]`

	output, err := processor.Process(context.Background(), "完成一个需要编码的任务", metadata)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if output != "Delegation wrapped successfully" {
		t.Fatalf("Process() output = %q", output)
	}
	toolNames := []string{}
	for _, item := range reactFactory.lastConfig.Tools {
		toolNames = append(toolNames, item.Name)
	}
	if !strings.Contains(strings.Join(toolNames, ","), "delegate_agent") {
		t.Fatalf("expected delegate_agent in %+v", toolNames)
	}
	if !strings.Contains(reactFactory.lastConfig.SystemPrompt, "coding") {
		t.Fatalf("expected delegation targets in prompt, got %q", reactFactory.lastConfig.SystemPrompt)
	}
}

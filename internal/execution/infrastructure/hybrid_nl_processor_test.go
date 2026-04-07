package infrastructure

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
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
	if len(req.Messages) > 0 && (strings.Contains(req.Messages[0].Content, "engineering delivery") || strings.Contains(req.Messages[0].Content, "dedicated coding assistant")) {
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
	processor := NewHybridNLProcessor(newTestProcessor(
		"success",
		mustBuildTestPrompt(t, "整理仓库", map[string]string{
			execdomain.RuntimeSessionIDMetadataKey: "session-default",
			execdomain.AgentIDMetadataKey:          "researcher",
		}),
	), reactFactory, nil)

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

func TestHybridNLProcessorCodingAgentPromptEmphasizesCodexLoop(t *testing.T) {
	reactFactory := &stubReactFactory{client: &answerOnlyLLMClient{}}
	processor := NewHybridNLProcessor(newTestProcessor("success", mustBuildTestPrompt(t, "整理仓库", testRuntimeMetadata())), reactFactory, nil)

	metadata := testRuntimeMetadata()
	metadata[execdomain.AgentIDMetadataKey] = "coding"
	metadata[execdomain.ExecutionEngineMetadataKey] = execdomain.ExecutionEngineAgent
	metadata[execdomain.AgentSystemPromptMetadataKey] = "Own coding delivery."

	if _, err := processor.Process(context.Background(), "完成仓库改造", metadata); err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	prompt := reactFactory.lastConfig.SystemPrompt
	for _, expected := range []string{
		"You are alter0's session-aware execution assistant.",
		"Act like the user's ongoing assistant for this session",
		"codex_exec already carries structured contexts for stable execution facts",
		"agent session profile",
		"You are the dedicated coding assistant.",
		"Do not implement or verify changes yourself.",
		"After every codex_exec result, decide whether the requirement is satisfied.",
		"Do not restate the full repository, workspace, and preview rulebook",
		"full clone with its own .git metadata",
		"Do not stop after the first Codex attempt",
		"Keep the coding session profile current in your reasoning",
		"Current coding workspace context:",
		"Dedicated repository workspace path:",
		"When repository code needs to be pulled, inspected, edited, built, or tested",
		"Preview URL:",
		"PR or merge handoff details",
		"Own coding delivery.",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected coding prompt to contain %q, got %q", expected, prompt)
		}
	}
	if !strings.Contains(prompt, ".alter0.cn") {
		t.Fatalf("expected coding prompt to contain preview domain rule, got %q", prompt)
	}
}

func TestHybridNLProcessorCodingAgentCodexExecUsesEffectivePrompt(t *testing.T) {
	reactFactory := &stubReactFactory{client: &scriptedLLMClient{}}
	processor := NewHybridNLProcessor(nil, reactFactory, nil)
	sourceRepoRoot := t.TempDir()
	initGitRepoWithCommit(t, sourceRepoRoot)

	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(sourceRepoRoot); err != nil {
		t.Fatalf("chdir source repo root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	metadata := testRuntimeMetadata()
	metadata[execdomain.AgentIDMetadataKey] = "coding"
	metadata[execdomain.AgentNameMetadataKey] = "Coding Agent"
	metadata[execdomain.ExecutionEngineMetadataKey] = execdomain.ExecutionEngineAgent
	metadata[execdomain.AgentSystemPromptMetadataKey] = "Own coding delivery."

	expectedPrompt, err := buildCodexPrompt("整理仓库", buildCodexExecMetadata(metadata))
	if err != nil {
		t.Fatalf("buildCodexPrompt() error = %v", err)
	}
	processor.codex = newTestProcessor("success", expectedPrompt, filepath.Join(".alter0", "workspaces", "sessions", "session-default", "repo"))

	output, err := processor.Process(context.Background(), "完成仓库整理", metadata)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if output != "任务已完成" {
		t.Fatalf("Process() output = %q, want %q", output, "任务已完成")
	}
}

func TestHybridNLProcessorUsesChatLevelModelOverride(t *testing.T) {
	reactFactory := &stubReactFactory{client: &answerOnlyLLMClient{}}
	processor := NewHybridNLProcessor(newTestProcessor("success", mustBuildTestPrompt(t, "整理仓库", testRuntimeMetadata())), reactFactory, nil)

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

func TestHybridNLProcessorReactModeIgnoresLegacyNativeTools(t *testing.T) {
	reactFactory := &stubReactFactory{client: &answerOnlyLLMClient{}}
	processor := NewHybridNLProcessor(newTestProcessor("success", mustBuildTestPrompt(t, "整理仓库", testRuntimeMetadata())), reactFactory, nil)

	metadata := testRuntimeMetadata()
	metadata[execdomain.AgentToolsMetadataKey] = `["read","bash"]`

	_, err := processor.Process(context.Background(), "读取文件并总结", metadata)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if len(reactFactory.lastConfig.Tools) != 0 {
		t.Fatalf("expected no legacy native tools in react mode, got %+v", reactFactory.lastConfig.Tools)
	}
	if reactFactory.lastConfig.ToolExecutor != nil {
		t.Fatalf("expected no tool executor for legacy native tools")
	}
	if reactFactory.lastConfig.MaxIterations != 1 {
		t.Fatalf("expected max iterations 1 without tool-enabled chat, got %d", reactFactory.lastConfig.MaxIterations)
	}
}

func TestHybridNLProcessorAgentModeDefaultsToCoreTools(t *testing.T) {
	reactFactory := &stubReactFactory{client: &answerOnlyLLMClient{}}
	processor := NewHybridNLProcessor(newTestProcessor("success", mustBuildTestPrompt(t, "整理仓库", testRuntimeMetadata())), reactFactory, nil)

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
	for _, expected := range []string{"codex_exec", "search_memory", "read_memory", "write_memory", "complete"} {
		if !strings.Contains(strings.Join(toolNames, ","), expected) {
			t.Fatalf("expected tool %s in %+v", expected, toolNames)
		}
	}
}

func TestHybridNLProcessorAgentModeAllowsHigherIterationLimit(t *testing.T) {
	reactFactory := &stubReactFactory{client: &answerOnlyLLMClient{}}
	processor := NewHybridNLProcessor(newTestProcessor("success", mustBuildTestPrompt(t, "整理仓库", testRuntimeMetadata())), reactFactory, nil)

	metadata := testRuntimeMetadata()
	metadata[execdomain.AgentIDMetadataKey] = "coding"
	metadata[execdomain.ExecutionEngineMetadataKey] = execdomain.ExecutionEngineAgent
	metadata[execdomain.AgentMaxIterationsMetadataKey] = "48"

	if _, err := processor.Process(context.Background(), "持续推进仓库改造", metadata); err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if reactFactory.lastConfig.MaxIterations != 48 {
		t.Fatalf("expected max iterations 48, got %d", reactFactory.lastConfig.MaxIterations)
	}
}

func TestHybridNLProcessorMarksModelExecutionSource(t *testing.T) {
	reactFactory := &stubReactFactory{client: &answerOnlyLLMClient{}}
	processor := NewHybridNLProcessor(newTestProcessor("success", mustBuildTestPrompt(t, "整理仓库", testRuntimeMetadata())), reactFactory, nil)

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
	processor := NewHybridNLProcessor(newTestProcessor("success", mustBuildTestPrompt(t, "整理仓库", testRuntimeMetadata())), nil, nil)

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
	processor := NewHybridNLProcessorWithCatalog(newTestProcessor("success", mustBuildTestPrompt(t, "整理仓库", testRuntimeMetadata())), reactFactory, catalog, nil)

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

func TestHybridNLProcessorAgentModeIncludesProductContext(t *testing.T) {
	reactFactory := &stubReactFactory{client: &answerOnlyLLMClient{}}
	processor := NewHybridNLProcessor(newTestProcessor("success", mustBuildTestPrompt(t, "整理仓库", testRuntimeMetadata())), reactFactory, nil)

	productContext := execdomain.ProductContext{
		Protocol:         execdomain.ProductContextProtocolVersion,
		ProductID:        "travel",
		Name:             "Travel",
		MasterAgentID:    "travel-master",
		ArtifactTypes:    []string{"city_guide", "itinerary", "map_layers"},
		KnowledgeSources: []string{"city_profile", "poi_catalog", "metro_network", "food_catalog"},
	}
	rawProductContext, err := json.Marshal(productContext)
	if err != nil {
		t.Fatalf("marshal product context failed: %v", err)
	}

	metadata := testRuntimeMetadata()
	metadata[execdomain.AgentIDMetadataKey] = "travel-master"
	metadata[execdomain.ExecutionEngineMetadataKey] = execdomain.ExecutionEngineAgent
	metadata[execdomain.ProductContextMetadataKey] = string(rawProductContext)

	if _, err := processor.Process(context.Background(), "生成一个三日游攻略", metadata); err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if !strings.Contains(reactFactory.lastConfig.SystemPrompt, "Resolved product context") {
		t.Fatalf("expected product context section in prompt, got %q", reactFactory.lastConfig.SystemPrompt)
	}
	if !strings.Contains(reactFactory.lastConfig.SystemPrompt, "city_profile") {
		t.Fatalf("expected knowledge source in prompt, got %q", reactFactory.lastConfig.SystemPrompt)
	}
}

func TestHybridNLProcessorAgentModeSupportsMemoryTools(t *testing.T) {
	reactFactory := &stubReactFactory{client: &answerOnlyLLMClient{}}
	processor := NewHybridNLProcessor(newTestProcessor("success", mustBuildTestPrompt(t, "整理仓库", testRuntimeMetadata())), reactFactory, nil)

	rawMemoryContext, err := json.Marshal(execdomain.MemoryContext{
		Protocol: execdomain.MemoryContextProtocolVersion,
		Files: []execdomain.MemoryFileSpec{
			{ID: "user_md", Selection: "user_md", Title: "USER.md", Path: "/repo/USER.md", Exists: true, Writable: true},
		},
		Recall: []execdomain.MemoryRecallHit{
			{MemoryID: "user_md", Title: "USER.md", Path: "/repo/USER.md", Line: 2, Snippet: "L2: response_style: concise"},
		},
	})
	if err != nil {
		t.Fatalf("marshal memory context failed: %v", err)
	}

	metadata := testRuntimeMetadata()
	metadata[execdomain.AgentIDMetadataKey] = "researcher"
	metadata[execdomain.ExecutionEngineMetadataKey] = execdomain.ExecutionEngineAgent
	metadata[execdomain.AgentToolsMetadataKey] = `["search_memory","read_memory","write_memory"]`
	metadata[execdomain.MemoryContextMetadataKey] = string(rawMemoryContext)

	if _, err := processor.Process(context.Background(), "更新用户缩写和偏好", metadata); err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	toolNames := []string{}
	for _, item := range reactFactory.lastConfig.Tools {
		toolNames = append(toolNames, item.Name)
	}
	if !strings.Contains(strings.Join(toolNames, ","), "search_memory") || !strings.Contains(strings.Join(toolNames, ","), "read_memory") || !strings.Contains(strings.Join(toolNames, ","), "write_memory") {
		t.Fatalf("expected skill tools in %+v", toolNames)
	}
	if !strings.Contains(reactFactory.lastConfig.SystemPrompt, "Use search_memory") || !strings.Contains(reactFactory.lastConfig.SystemPrompt, "Use read_memory") || !strings.Contains(reactFactory.lastConfig.SystemPrompt, "Use write_memory") {
		t.Fatalf("expected memory tool instructions in prompt, got %q", reactFactory.lastConfig.SystemPrompt)
	}
	if !strings.Contains(reactFactory.lastConfig.SystemPrompt, "Auto-recalled memory snippets") || !strings.Contains(reactFactory.lastConfig.SystemPrompt, "response_style: concise") {
		t.Fatalf("expected auto-recalled memory snippets in prompt, got %q", reactFactory.lastConfig.SystemPrompt)
	}
}

func TestHybridNLProcessorExecutesMemorySearchReadWriteTools(t *testing.T) {
	processor := NewHybridNLProcessor(newTestProcessor("success", mustBuildTestPrompt(t, "整理仓库", testRuntimeMetadata())), nil, nil)
	memoryPath := filepath.Join(t.TempDir(), "USER.md")
	rawMemoryContext, err := json.Marshal(execdomain.MemoryContext{
		Protocol: execdomain.MemoryContextProtocolVersion,
		Files: []execdomain.MemoryFileSpec{
			{ID: "user_md", Selection: "user_md", Title: "USER.md", Path: memoryPath, Exists: false, Writable: true},
		},
	})
	if err != nil {
		t.Fatalf("marshal memory context failed: %v", err)
	}

	metadata := testRuntimeMetadata()
	metadata[execdomain.MemoryContextMetadataKey] = string(rawMemoryContext)

	writeResult, err := processor.executeModelTool(context.Background(), metadata, llmdomain.ToolCall{
		ID:        "write-memory-1",
		Name:      "write_memory",
		Arguments: `{"path":"` + filepath.ToSlash(memoryPath) + `","content":"name: alter0\nalias: a0\n","mode":"overwrite"}`,
	})
	if err != nil {
		t.Fatalf("write_memory error = %v", err)
	}
	if writeResult.IsError {
		t.Fatalf("expected write_memory success, got %+v", writeResult)
	}

	searchResult, err := processor.executeModelTool(context.Background(), metadata, llmdomain.ToolCall{
		ID:        "search-memory-1",
		Name:      "search_memory",
		Arguments: `{"query":"alias"}`,
	})
	if err != nil {
		t.Fatalf("search_memory error = %v", err)
	}
	if searchResult.IsError || !strings.Contains(searchResult.Result, "alias") {
		t.Fatalf("expected search_memory content, got %+v", searchResult)
	}

	readResult, err := processor.executeModelTool(context.Background(), metadata, llmdomain.ToolCall{
		ID:        "read-memory-1",
		Name:      "read_memory",
		Arguments: `{"path":"` + filepath.ToSlash(memoryPath) + `"}`,
	})
	if err != nil {
		t.Fatalf("read_memory error = %v", err)
	}
	if readResult.IsError || !strings.Contains(readResult.Result, "alias") {
		t.Fatalf("expected read_memory content, got %+v", readResult)
	}
}

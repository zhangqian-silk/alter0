package infrastructure

import (
	execdomain "alter0/internal/execution/domain"
	llmdomain "alter0/internal/llm/domain"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type stubReactFactory struct {
	client         llmdomain.LLMClient
	lastConfig     llmdomain.ReActRunnerConfig
	lastProviderID string
}

func (s *stubReactFactory) GetReActRunner(_ context.Context, providerID string, config llmdomain.ReActRunnerConfig) (*llmdomain.ReActRunner, error) {
	s.lastConfig = config
	s.lastProviderID = providerID
	s.lastConfig.Client = nil
	config.Client = s.client
	if config.Model == "" {
		config.Model = "test-model"
	}
	return llmdomain.NewReActRunner(config), nil
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

type captureMessagePartsLLMClient struct {
	lastRequest llmdomain.ChatRequest
}

func (c *captureMessagePartsLLMClient) Chat(_ context.Context, req llmdomain.ChatRequest) (*llmdomain.ChatResponse, error) {
	c.lastRequest = req
	return &llmdomain.ChatResponse{
		Message: llmdomain.Message{
			Role:    "assistant",
			Content: "已读取图片",
		},
	}, nil
}

func (c *captureMessagePartsLLMClient) ChatStream(_ context.Context, _ llmdomain.ChatRequest, _ func(llmdomain.StreamEvent) error) (*llmdomain.ChatResponse, error) {
	return nil, nil
}

func (c *captureMessagePartsLLMClient) Close() error {
	return nil
}

type failingLLMClient struct {
	err error
}

func (c *failingLLMClient) Chat(_ context.Context, _ llmdomain.ChatRequest) (*llmdomain.ChatResponse, error) {
	return nil, c.err
}

func (c *failingLLMClient) ChatStream(_ context.Context, _ llmdomain.ChatRequest, _ func(llmdomain.StreamEvent) error) (*llmdomain.ChatResponse, error) {
	return nil, c.err
}

func (c *failingLLMClient) Close() error {
	return nil
}

var errUnexpectedToolObservation = &testError{text: "unexpected tool observation"}

type testError struct {
	text string
}

func (e *testError) Error() string {
	return e.text
}

func markTravelSkill(t *testing.T, metadata map[string]string) {
	t.Helper()
	raw, err := json.Marshal(execdomain.SkillContext{
		Protocol: execdomain.SkillContextProtocolVersion,
		Skills: []execdomain.SkillSpec{{
			ID:   "travel",
			Name: "Travel",
		}},
	})
	if err != nil {
		t.Fatalf("marshal skill context: %v", err)
	}
	metadata[execdomain.SkillContextMetadataKey] = string(raw)
}

func travelCompletionChecksJSON(t *testing.T) string {
	t.Helper()
	raw, err := json.Marshal([]execdomain.CompletionCheck{
		{
			ID:                "travel-index-html",
			Label:             "Travel guide HTML",
			Type:              execdomain.CompletionCheckTypeSessionFileExists,
			Required:          true,
			SessionPath:       "index.html",
			FailureMessage:    "missing travel guide HTML: {{session_file}}",
			RepairInstruction: "Create or update {{session_workspace}}/index.html for the current travel request.",
		},
		{
			ID:                    "travel-service",
			Label:                 "Published travel guide",
			Type:                  execdomain.CompletionCheckTypeWorkspaceServicePublished,
			Required:              true,
			ServiceID:             "travel",
			RequirePublicReadOnly: true,
			RequireServiceURL:     true,
			FailureMessage:        "missing published travel guide service",
			RepairInstruction:     "Publish the session workspace root as service travel.",
		},
	})
	if err != nil {
		t.Fatalf("marshal completion checks: %v", err)
	}
	return string(raw)
}

func genericFileCompletionChecksJSON(t *testing.T, sessionPath string) string {
	t.Helper()
	raw, err := json.Marshal([]execdomain.CompletionCheck{{
		ID:                "final-file",
		Label:             "Final artifact",
		Type:              execdomain.CompletionCheckTypeSessionFileExists,
		Required:          true,
		SessionPath:       sessionPath,
		FailureMessage:    "missing final artifact: {{session_file}}",
		RepairInstruction: "Create {{session_workspace}}/" + strings.Trim(strings.TrimSpace(sessionPath), "/") + ".",
	}})
	if err != nil {
		t.Fatalf("marshal completion checks: %v", err)
	}
	return string(raw)
}

func TestHybridNLProcessorValidatesTravelCompletionAfterCodexFallback(t *testing.T) {
	rootDir := t.TempDir()
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("chdir root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	processor := NewHybridNLProcessor(newTestProcessor("success", ""), nil, nil)
	processor.serviceDeployer = nil

	metadata := testRuntimeMetadata()
	markTravelSkill(t, metadata)
	metadata[execdomain.CompletionChecksMetadataKey] = travelCompletionChecksJSON(t)

	_, err = processor.Process(context.Background(), "整理武汉攻略", metadata)
	if err == nil {
		t.Fatal("Process() error = nil, want travel completion validation failure")
	}
	if !strings.Contains(err.Error(), "missing") || !strings.Contains(err.Error(), "index.html") {
		t.Fatalf("expected missing index.html error, got %q", err.Error())
	}
}

func TestHybridNLProcessorAllowsTravelCompletionAfterCodexFallbackWhenDeliverablesExist(t *testing.T) {
	rootDir := t.TempDir()
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("chdir root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	sessionWorkspace := filepath.Join(rootDir, ".alter0", "workspaces", "sessions", "session-default")
	if err := os.MkdirAll(sessionWorkspace, 0o755); err != nil {
		t.Fatalf("mkdir session workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionWorkspace, "index.html"), []byte("<!doctype html><title>travel</title>"), 0o644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}
	registryPayload := `{"items":[{"session_id":"session-default","service_id":"travel","service_type":"frontend_dist","url":"https://travel-4e8f5f54.alter0.cn","public_read_only":true}]}`
	if err := os.WriteFile(filepath.Join(rootDir, ".alter0", "workspace-services.json"), []byte(registryPayload), 0o644); err != nil {
		t.Fatalf("write workspace service registry: %v", err)
	}

	metadata := testRuntimeMetadata()
	markTravelSkill(t, metadata)
	metadata[execdomain.CompletionChecksMetadataKey] = travelCompletionChecksJSON(t)
	processor := NewHybridNLProcessor(newTestProcessor("success", "", filepath.Join(".alter0", "workspaces", "sessions", "session-default")), nil, nil)

	output, err := processor.Process(context.Background(), "整理武汉攻略", metadata)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if !strings.Contains(output, "mock response") || !strings.Contains(output, "https://travel-4e8f5f54.alter0.cn") {
		t.Fatalf("Process() output = %q, want guide content plus published url", output)
	}
}

func TestHybridNLProcessorRepairsTravelCompletionAfterCodexFallback(t *testing.T) {
	rootDir := t.TempDir()
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("chdir root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	metadata := testRuntimeMetadata()
	markTravelSkill(t, metadata)
	metadata[execdomain.CompletionChecksMetadataKey] = travelCompletionChecksJSON(t)

	sessionWorkspaceSuffix := filepath.Join(".alter0", "workspaces", "sessions", "session-default")
	processor := NewHybridNLProcessor(newSequencedTestProcessor(
		codexTestInvocation{
			mode:              "success",
			expectedPrompt:    "整理武汉攻略",
			expectedWorkspace: sessionWorkspaceSuffix,
		},
		codexTestInvocation{
			mode:                   "travel-repair-success",
			expectedPromptContains: "Repair the missing required deliverables for the current skill run now.",
			expectedWorkspace:      sessionWorkspaceSuffix,
		},
	), nil, nil)

	output, err := processor.Process(context.Background(), "整理武汉攻略", metadata)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if !strings.Contains(output, "mock response") || !strings.Contains(output, "https://travel-4e8f5f54.alter0.cn") {
		t.Fatalf("Process() output = %q, want repaired guide content plus published url", output)
	}
	if _, statErr := os.Stat(filepath.Join(rootDir, ".alter0", "workspaces", "sessions", "session-default", "index.html")); statErr != nil {
		t.Fatalf("expected repair flow to create index.html, stat err = %v", statErr)
	}
}

func TestHybridNLProcessorValidatesTravelCompletionForDirectCodexEngine(t *testing.T) {
	rootDir := t.TempDir()
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("chdir root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	processor := NewHybridNLProcessor(newTestProcessor("success", ""), nil, nil)
	processor.serviceDeployer = nil

	metadata := testRuntimeMetadata()
	markTravelSkill(t, metadata)
	metadata[execdomain.ExecutionEngineMetadataKey] = execdomain.ExecutionEngineCodex
	metadata[execdomain.CompletionChecksMetadataKey] = travelCompletionChecksJSON(t)

	_, err = processor.Process(context.Background(), "整理武汉攻略", metadata)
	if err == nil {
		t.Fatal("Process() error = nil, want travel completion validation failure on direct codex path")
	}
	if !strings.Contains(err.Error(), "missing") || !strings.Contains(err.Error(), "index.html") {
		t.Fatalf("expected missing index.html error, got %q", err.Error())
	}
}

func TestHybridNLProcessorRepairsTravelCompletionForDirectCodexEngine(t *testing.T) {
	rootDir := t.TempDir()
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("chdir root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	metadata := testRuntimeMetadata()
	markTravelSkill(t, metadata)
	metadata[execdomain.ExecutionEngineMetadataKey] = execdomain.ExecutionEngineCodex
	metadata[execdomain.CompletionChecksMetadataKey] = travelCompletionChecksJSON(t)

	sessionWorkspaceSuffix := filepath.Join(".alter0", "workspaces", "sessions", "session-default")
	processor := NewHybridNLProcessor(newSequencedTestProcessor(
		codexTestInvocation{
			mode:              "success",
			expectedPrompt:    "整理武汉攻略",
			expectedWorkspace: sessionWorkspaceSuffix,
		},
		codexTestInvocation{
			mode:                   "travel-repair-success",
			expectedPromptContains: "Repair the missing required deliverables for the current skill run now.",
			expectedWorkspace:      sessionWorkspaceSuffix,
		},
	), nil, nil)

	output, err := processor.Process(context.Background(), "整理武汉攻略", metadata)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if !strings.Contains(output, "mock response") || !strings.Contains(output, "https://travel-4e8f5f54.alter0.cn") {
		t.Fatalf("Process() output = %q, want repaired guide content plus published url", output)
	}
	if _, statErr := os.Stat(filepath.Join(rootDir, ".alter0", "workspaces", "sessions", "session-default", "index.html")); statErr != nil {
		t.Fatalf("expected repair flow to create index.html, stat err = %v", statErr)
	}
}

func TestHybridNLProcessorKeepsTravelValidationBlockedWhenRepairDoesNotProduceArtifactsForDirectCodexEngine(t *testing.T) {
	rootDir := t.TempDir()
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("chdir root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	deployer := &stubWorkspaceServiceDeployer{
		result: WorkspaceServiceDeployResult{
			SessionID:   "session-default",
			ServiceID:   "travel",
			ServiceType: workspaceServiceTypeFrontendDist,
			Host:        "travel-4e8f5f54.alter0.cn",
			URL:         "https://travel-4e8f5f54.alter0.cn",
			Status:      "deployed",
		},
	}
	processor := NewHybridNLProcessor(newTestProcessor("success", ""), nil, nil)
	processor.serviceDeployer = deployer

	metadata := testRuntimeMetadata()
	markTravelSkill(t, metadata)
	metadata[execdomain.ExecutionEngineMetadataKey] = execdomain.ExecutionEngineCodex
	metadata[execdomain.CompletionChecksMetadataKey] = travelCompletionChecksJSON(t)

	_, err = processor.Process(context.Background(), "整理武汉攻略", metadata)
	if err == nil {
		t.Fatal("Process() error = nil, want travel validation failure after ineffective repair")
	}

	indexPath := filepath.Join(rootDir, ".alter0", "workspaces", "sessions", "session-default", "index.html")
	if _, statErr := os.Stat(indexPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected repair to stay blocked without a real generated index.html, stat err = %v", statErr)
	}
	if deployer.lastRequest.ServiceID != "" || deployer.lastRequest.SessionID != "" {
		t.Fatalf("expected no model-tool publish attempt outside codex repair flow, got %+v", deployer.lastRequest)
	}
}

func TestHybridNLProcessorAllowsTravelCompletionForDirectCodexEngineWhenDeliverablesExist(t *testing.T) {
	rootDir := t.TempDir()
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("chdir root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	sessionWorkspace := filepath.Join(rootDir, ".alter0", "workspaces", "sessions", "session-default")
	if err := os.MkdirAll(sessionWorkspace, 0o755); err != nil {
		t.Fatalf("mkdir session workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionWorkspace, "index.html"), []byte("<!doctype html><title>travel</title>"), 0o644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}
	registryPayload := `{"items":[{"session_id":"session-default","service_id":"travel","service_type":"frontend_dist","url":"https://travel-4e8f5f54.alter0.cn","public_read_only":true}]}`
	if err := os.WriteFile(filepath.Join(rootDir, ".alter0", "workspace-services.json"), []byte(registryPayload), 0o644); err != nil {
		t.Fatalf("write workspace service registry: %v", err)
	}

	metadata := testRuntimeMetadata()
	markTravelSkill(t, metadata)
	metadata[execdomain.ExecutionEngineMetadataKey] = execdomain.ExecutionEngineCodex
	metadata[execdomain.CompletionChecksMetadataKey] = travelCompletionChecksJSON(t)
	processor := NewHybridNLProcessor(newTestProcessor("success", "", filepath.Join(".alter0", "workspaces", "sessions", "session-default")), nil, nil)

	output, err := processor.Process(context.Background(), "整理武汉攻略", metadata)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if !strings.Contains(output, "mock response") || !strings.Contains(output, "https://travel-4e8f5f54.alter0.cn") {
		t.Fatalf("Process() output = %q, want guide content plus published url", output)
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

func TestHybridNLProcessorRepairsGenericDeliveryCompletionChecks(t *testing.T) {
	rootDir := t.TempDir()
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("chdir root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	metadata := testRuntimeMetadata()
	metadata[execdomain.ExecutionEngineMetadataKey] = execdomain.ExecutionEngineCodex
	metadata[execdomain.CompletionChecksMetadataKey] = genericFileCompletionChecksJSON(t, "artifacts/final.md")

	sessionWorkspaceSuffix := filepath.Join(".alter0", "workspaces", "sessions", "session-default")
	processor := NewHybridNLProcessor(newSequencedTestProcessor(
		codexTestInvocation{
			mode:              "success",
			expectedPrompt:    "整理交付物",
			expectedWorkspace: sessionWorkspaceSuffix,
		},
		codexTestInvocation{
			mode:                   "write-session-file-success",
			expectedPromptContains: "Repair the missing required deliverables for the current skill run now.",
			expectedWorkspace:      sessionWorkspaceSuffix,
			writeSessionFilePath:   "artifacts/final.md",
			writeSessionFileBody:   "# done\n",
		},
	), nil, nil)

	output, err := processor.Process(context.Background(), "整理交付物", metadata)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if !strings.Contains(output, "mock response") {
		t.Fatalf("Process() output = %q, want original codex result retained", output)
	}
	if _, statErr := os.Stat(filepath.Join(rootDir, ".alter0", "workspaces", "sessions", "session-default", "artifacts", "final.md")); statErr != nil {
		t.Fatalf("expected generic repair flow to create required artifact, stat err = %v", statErr)
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

func TestHybridNLProcessorIncludesImageAttachmentsInReactUserMessage(t *testing.T) {
	captureClient := &captureMessagePartsLLMClient{}
	reactFactory := &stubReactFactory{client: captureClient}
	processor := NewHybridNLProcessor(newTestProcessor("success", mustBuildTestPrompt(t, "describe image", testRuntimeMetadata())), reactFactory, nil)

	metadata := testRuntimeMetadata()
	rawAttachments, err := execdomain.EncodeUserImageAttachments([]execdomain.UserImageAttachment{{
		Name:        "diagram.png",
		ContentType: "image/png",
		DataURL:     "data:image/png;base64,ZmFrZQ==",
	}})
	if err != nil {
		t.Fatalf("EncodeUserImageAttachments() error = %v", err)
	}
	metadata[execdomain.UserImageAttachmentsMetadataKey] = rawAttachments

	output, err := processor.Process(context.Background(), "describe image", metadata)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if output != "已读取图片" {
		t.Fatalf("Process() output = %q", output)
	}
	if len(captureClient.lastRequest.Messages) < 2 {
		t.Fatalf("expected system and user messages, got %+v", captureClient.lastRequest.Messages)
	}
	userMessage := captureClient.lastRequest.Messages[len(captureClient.lastRequest.Messages)-1]
	if len(userMessage.Parts) != 2 {
		t.Fatalf("expected text and image parts, got %+v", userMessage.Parts)
	}
	if userMessage.Parts[0].Type != llmdomain.MessagePartTypeText || userMessage.Parts[1].Type != llmdomain.MessagePartTypeImage {
		t.Fatalf("unexpected message parts %+v", userMessage.Parts)
	}
	if userMessage.Parts[1].ImageURL != "data:image/png;base64,ZmFrZQ==" {
		t.Fatalf("expected image data URL, got %+v", userMessage.Parts[1])
	}
}

func TestHybridNLProcessorLoadsWorkspaceAttachmentFilesIntoReactUserMessage(t *testing.T) {
	t.Parallel()

	captureClient := &captureMessagePartsLLMClient{}
	reactFactory := &stubReactFactory{client: captureClient}
	processor := NewHybridNLProcessor(newTestProcessor("success", mustBuildTestPrompt(t, "describe image", testRuntimeMetadata())), reactFactory, nil)

	file := filepath.Join(t.TempDir(), "diagram.png")
	if err := os.WriteFile(file, []byte("fake"), 0o644); err != nil {
		t.Fatalf("write workspace attachment: %v", err)
	}

	metadata := testRuntimeMetadata()
	rawAttachments, err := execdomain.EncodeUserImageAttachments([]execdomain.UserImageAttachment{{
		Name:          "diagram.png",
		ContentType:   "image/png",
		WorkspacePath: file,
	}})
	if err != nil {
		t.Fatalf("EncodeUserImageAttachments() error = %v", err)
	}
	metadata[execdomain.UserImageAttachmentsMetadataKey] = rawAttachments

	output, err := processor.Process(context.Background(), "describe image", metadata)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if output != "已读取图片" {
		t.Fatalf("Process() output = %q", output)
	}
	userMessage := captureClient.lastRequest.Messages[len(captureClient.lastRequest.Messages)-1]
	if got := userMessage.Parts[1].ImageURL; got != "data:image/png;base64,ZmFrZQ==" {
		t.Fatalf("expected workspace file converted to data URL, got %q", got)
	}
}

func TestHybridNLProcessorDoesNotFallbackToCodexWhenImagesNeedVision(t *testing.T) {
	reactErr := &testError{text: "vision model unavailable"}
	reactFactory := &stubReactFactory{client: &failingLLMClient{err: reactErr}}
	processor := NewHybridNLProcessor(newTestProcessor("success", mustBuildTestPrompt(t, "describe image", testRuntimeMetadata())), reactFactory, nil)

	metadata := testRuntimeMetadata()
	rawAttachments, err := execdomain.EncodeUserImageAttachments([]execdomain.UserImageAttachment{{
		Name:        "diagram.png",
		ContentType: "image/png",
		DataURL:     "data:image/png;base64,ZmFrZQ==",
	}})
	if err != nil {
		t.Fatalf("EncodeUserImageAttachments() error = %v", err)
	}
	metadata[execdomain.UserImageAttachmentsMetadataKey] = rawAttachments

	output, err := processor.Process(context.Background(), "describe image", metadata)
	if err != reactErr {
		t.Fatalf("expected react error %v, got %v", reactErr, err)
	}
	if output != "" {
		t.Fatalf("expected empty output when react fails with images, got %q", output)
	}
	if got := metadata[execdomain.ExecutionSourceMetadataKey]; got != "" {
		t.Fatalf("expected no codex fallback execution source, got %q", got)
	}
}

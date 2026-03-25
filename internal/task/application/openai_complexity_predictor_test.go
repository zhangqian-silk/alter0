package application

import (
	"context"
	"errors"
	"testing"

	llmdomain "alter0/internal/llm/domain"
	shareddomain "alter0/internal/shared/domain"
)

type stubComplexityLLMClientSource struct {
	client         llmdomain.LLMClient
	err            error
	lastProviderID string
	defaultCalls   int
}

func (s *stubComplexityLLMClientSource) GetClient(_ context.Context, providerID string) (llmdomain.LLMClient, error) {
	s.lastProviderID = providerID
	if s.err != nil {
		return nil, s.err
	}
	return s.client, nil
}

func (s *stubComplexityLLMClientSource) GetDefaultClient(_ context.Context) (llmdomain.LLMClient, error) {
	s.defaultCalls++
	if s.err != nil {
		return nil, s.err
	}
	return s.client, nil
}

type stubComplexityLLMClient struct {
	response *llmdomain.ChatResponse
	err      error
	lastReq  llmdomain.ChatRequest
}

func (s *stubComplexityLLMClient) Chat(_ context.Context, req llmdomain.ChatRequest) (*llmdomain.ChatResponse, error) {
	s.lastReq = req
	if s.err != nil {
		return nil, s.err
	}
	return s.response, nil
}

func (s *stubComplexityLLMClient) ChatStream(_ context.Context, _ llmdomain.ChatRequest, _ func(llmdomain.StreamEvent) error) (*llmdomain.ChatResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *stubComplexityLLMClient) Close() error {
	return nil
}

type stubFallbackComplexityPredictor struct {
	assessment ComplexityAssessment
	err        error
	callCount  int
}

func (s *stubFallbackComplexityPredictor) Predict(_ context.Context, _ shareddomain.UnifiedMessage) (ComplexityAssessment, error) {
	s.callCount++
	if s.err != nil {
		return ComplexityAssessment{}, s.err
	}
	return s.assessment, nil
}

func TestOpenAIComplexityPredictorUsesModelResponse(t *testing.T) {
	client := &stubComplexityLLMClient{
		response: &llmdomain.ChatResponse{
			Message: llmdomain.Message{
				Role:    "assistant",
				Content: `{"task_summary":"处理仓库修改","task_approach":"先分析再执行","estimated_duration_seconds":420,"complexity_level":"high","execution_mode":"async"}`,
			},
		},
	}
	source := &stubComplexityLLMClientSource{client: client}
	predictor := NewOpenAIComplexityPredictor(source, nil, nil)

	assessment, err := predictor.Predict(context.Background(), shareddomain.UnifiedMessage{
		MessageID:   "msg-1",
		SessionID:   "session-1",
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     "请分析仓库并完成一组跨文件修改",
		Metadata: map[string]string{
			complexityProviderIDMetadataKey: "provider-selected",
			complexityModelMetadataKey:      "gpt-5.4",
		},
	})
	if err != nil {
		t.Fatalf("predict: %v", err)
	}
	if assessment.ExecutionMode != ExecutionModeAsync {
		t.Fatalf("expected async mode, got %q", assessment.ExecutionMode)
	}
	if assessment.ComplexityLevel != ComplexityLevelHigh {
		t.Fatalf("expected high complexity, got %q", assessment.ComplexityLevel)
	}
	if assessment.EstimatedDurationSeconds != 420 {
		t.Fatalf("expected 420 seconds, got %d", assessment.EstimatedDurationSeconds)
	}
	if source.lastProviderID != "provider-selected" {
		t.Fatalf("expected selected provider, got %q", source.lastProviderID)
	}
	if source.defaultCalls != 0 {
		t.Fatalf("expected no default client lookup, got %d", source.defaultCalls)
	}
	if client.lastReq.Model != "gpt-5.4" {
		t.Fatalf("expected selected model gpt-5.4, got %q", client.lastReq.Model)
	}
}

func TestOpenAIComplexityPredictorFallsBackToDefaultClientWhenProviderNotSpecified(t *testing.T) {
	client := &stubComplexityLLMClient{
		response: &llmdomain.ChatResponse{
			Message: llmdomain.Message{
				Role:    "assistant",
				Content: `{"task_summary":"短任务","task_approach":"直接处理","estimated_duration_seconds":60,"complexity_level":"low","execution_mode":"streaming"}`,
			},
		},
	}
	source := &stubComplexityLLMClientSource{client: client}
	predictor := NewOpenAIComplexityPredictor(source, nil, nil)

	_, err := predictor.Predict(context.Background(), shareddomain.UnifiedMessage{
		MessageID:   "msg-default",
		SessionID:   "session-default",
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     "简短请求",
	})
	if err != nil {
		t.Fatalf("predict: %v", err)
	}
	if source.lastProviderID != "" {
		t.Fatalf("expected no explicit provider lookup, got %q", source.lastProviderID)
	}
	if source.defaultCalls != 1 {
		t.Fatalf("expected one default client lookup, got %d", source.defaultCalls)
	}
}

func TestOpenAIComplexityPredictorFallsBackWhenClientUnavailable(t *testing.T) {
	fallback := &stubFallbackComplexityPredictor{
		assessment: ComplexityAssessment{
			TaskSummary:              "fallback",
			TaskApproach:             "fallback approach",
			EstimatedDurationSeconds: 180,
			ComplexityLevel:          ComplexityLevelMedium,
			ExecutionMode:            ExecutionModeStreaming,
		},
	}
	predictor := NewOpenAIComplexityPredictor(&stubComplexityLLMClientSource{
		err: errors.New("provider disabled"),
	}, fallback, nil)

	assessment, err := predictor.Predict(context.Background(), shareddomain.UnifiedMessage{
		MessageID:   "msg-2",
		SessionID:   "session-2",
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     "短任务",
	})
	if err != nil {
		t.Fatalf("predict: %v", err)
	}
	if fallback.callCount != 1 {
		t.Fatalf("expected fallback invoked once, got %d", fallback.callCount)
	}
	if assessment.TaskSummary != "fallback" {
		t.Fatalf("expected fallback assessment, got %+v", assessment)
	}
}

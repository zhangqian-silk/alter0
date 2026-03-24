package application

import (
	"context"
	"errors"
	"testing"
	"time"

	llmdomain "alter0/internal/llm/domain"
	shareddomain "alter0/internal/shared/domain"
)

type stubComplexityLLMClientSource struct {
	client llmdomain.LLMClient
	err    error
}

func (s *stubComplexityLLMClientSource) GetDefaultClient(_ context.Context) (llmdomain.LLMClient, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.client, nil
}

type stubComplexityLLMClient struct {
	response *llmdomain.ChatResponse
	err      error
}

func (s *stubComplexityLLMClient) Chat(_ context.Context, _ llmdomain.ChatRequest) (*llmdomain.ChatResponse, error) {
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

type blockingFallbackComplexityPredictor struct {
	started chan struct{}
	release chan struct{}
}

func (s *blockingFallbackComplexityPredictor) Predict(_ context.Context, _ shareddomain.UnifiedMessage) (ComplexityAssessment, error) {
	close(s.started)
	<-s.release
	return ComplexityAssessment{}, nil
}

func TestOpenAIComplexityPredictorUsesModelResponse(t *testing.T) {
	predictor := NewOpenAIComplexityPredictor(&stubComplexityLLMClientSource{
		client: &stubComplexityLLMClient{
			response: &llmdomain.ChatResponse{
				Message: llmdomain.Message{
					Role:    "assistant",
					Content: `{"task_summary":"处理仓库修改","task_approach":"先分析再执行","estimated_duration_seconds":420,"complexity_level":"high","execution_mode":"async"}`,
				},
			},
		},
	}, nil, nil)

	assessment, err := predictor.Predict(context.Background(), shareddomain.UnifiedMessage{
		MessageID:   "msg-1",
		SessionID:   "session-1",
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     "请分析仓库并完成一组跨文件修改",
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

func TestOpenAIComplexityPredictorFallbackTimesOut(t *testing.T) {
	fallback := &blockingFallbackComplexityPredictor{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	predictor := NewOpenAIComplexityPredictor(&stubComplexityLLMClientSource{
		err: errors.New("provider disabled"),
	}, fallback, nil)
	predictor.fallbackWait = 20 * time.Millisecond

	startedAt := time.Now()
	_, err := predictor.Predict(context.Background(), shareddomain.UnifiedMessage{
		MessageID:   "msg-3",
		SessionID:   "session-3",
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     "阻塞 fallback",
	})
	if err == nil {
		close(fallback.release)
		t.Fatal("expected fallback timeout error")
	}
	<-fallback.started
	close(fallback.release)

	if elapsed := time.Since(startedAt); elapsed > 250*time.Millisecond {
		t.Fatalf("expected fallback timeout quickly, got %s", elapsed)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
}

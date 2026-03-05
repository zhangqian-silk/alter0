package application

import (
	"strings"
	"testing"
	"time"

	shareddomain "alter0/internal/shared/domain"
)

func TestParseCodexComplexityOutputParsesJSONFence(t *testing.T) {
	raw := "```json\n{\"estimated_duration_seconds\":42,\"complexity_level\":\"high\",\"execution_mode\":\"async\"}\n```"
	assessment, err := parseCodexComplexityOutput(raw)
	if err != nil {
		t.Fatalf("parse output: %v", err)
	}
	if assessment.EstimatedDurationSeconds != 42 {
		t.Fatalf("expected estimate 42, got %d", assessment.EstimatedDurationSeconds)
	}
	if assessment.ComplexityLevel != "high" {
		t.Fatalf("expected complexity high, got %q", assessment.ComplexityLevel)
	}
	if assessment.ExecutionMode != "async" {
		t.Fatalf("expected execution_mode async, got %q", assessment.ExecutionMode)
	}
}

func TestBuildCodexComplexityPromptRequiresNonThinkingMode(t *testing.T) {
	prompt := buildCodexComplexityPrompt(shareddomain.UnifiedMessage{
		Content: "生成一篇介绍 ngram 的文档",
		Metadata: map[string]string{
			MetadataTaskTypeKey:   "doc",
			MetadataTaskAsyncMode: "",
		},
	})
	if !strings.Contains(prompt, "非思考模式") {
		t.Fatalf("expected prompt to require non-thinking mode, got %q", prompt)
	}
	if !strings.Contains(prompt, "仅输出一行 JSON") {
		t.Fatalf("expected strict json output instruction, got %q", prompt)
	}
}

func TestCodexComplexityPredictorInitialTimeoutUsesBase(t *testing.T) {
	predictor := NewCodexQuickComplexityPredictor()
	predictor.timeout = 2 * time.Second

	timeout := predictor.nextTimeout()
	if timeout != 2*time.Second {
		t.Fatalf("expected initial timeout 2s, got %s", timeout)
	}
}

func TestCodexComplexityPredictorAdaptiveTimeoutUsesEWMA(t *testing.T) {
	predictor := NewCodexQuickComplexityPredictor()
	predictor.timeout = 2 * time.Second

	predictor.observeExecution(3*time.Second, false)
	timeout1 := predictor.nextTimeout()
	if timeout1 <= 3*time.Second {
		t.Fatalf("expected timeout above 3s after first slow sample, got %s", timeout1)
	}

	predictor.observeExecution(1*time.Second, false)
	timeout2 := predictor.nextTimeout()
	if timeout2 >= timeout1 {
		t.Fatalf("expected smoothed timeout to decrease after faster sample, timeout1=%s timeout2=%s", timeout1, timeout2)
	}
	if timeout2 <= 2*time.Second {
		t.Fatalf("expected timeout to remain above base due to smoothing, got %s", timeout2)
	}
}

func TestCodexComplexityPredictorTimeoutBackoffAndRecovery(t *testing.T) {
	predictor := NewCodexQuickComplexityPredictor()
	predictor.timeout = 2 * time.Second

	predictor.observeExecution(2*time.Second, true)
	timeout1 := predictor.nextTimeout()
	predictor.observeExecution(2*time.Second, true)
	timeout2 := predictor.nextTimeout()
	if timeout2 <= timeout1 {
		t.Fatalf("expected timeout backoff to increase timeout, timeout1=%s timeout2=%s", timeout1, timeout2)
	}

	predictor.observeExecution(2*time.Second, false)
	timeout3 := predictor.nextTimeout()
	if timeout3 >= timeout2 {
		t.Fatalf("expected timeout to recover after successful run, timeout2=%s timeout3=%s", timeout2, timeout3)
	}
}

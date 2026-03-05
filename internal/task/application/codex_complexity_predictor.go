package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	shareddomain "alter0/internal/shared/domain"
)

const (
	defaultCodexComplexityPredictorCommand = "codex"
	defaultCodexComplexityPredictorTimeout = 2 * time.Second
	defaultCodexComplexityTimeoutMin       = 1200 * time.Millisecond
	defaultCodexComplexityTimeoutMax       = 8 * time.Second
	defaultCodexComplexityEWMAAlpha        = 0.25
	defaultCodexComplexitySafetyFactor     = 1.7
	defaultCodexComplexityTimeoutBackoff   = 0.2
	maxCodexComplexityTimeoutBackoffSteps  = 5
	maxCodexComplexityPromptRunes          = 1200
)

type codexComplexityCommandRunner func(ctx context.Context, name string, args ...string) *exec.Cmd

type CodexQuickComplexityPredictor struct {
	command string
	timeout time.Duration
	runner  codexComplexityCommandRunner

	mu              sync.Mutex
	latencyEWMA     time.Duration
	latencySamples  int
	timeoutBackoffN int
	ewmaAlpha       float64
	safetyFactor    float64
	minTimeout      time.Duration
	maxTimeout      time.Duration
	timeoutBackoffK float64
}

func NewCodexQuickComplexityPredictor() *CodexQuickComplexityPredictor {
	return &CodexQuickComplexityPredictor{
		command:         defaultCodexComplexityPredictorCommand,
		timeout:         defaultCodexComplexityPredictorTimeout,
		runner:          exec.CommandContext,
		ewmaAlpha:       defaultCodexComplexityEWMAAlpha,
		safetyFactor:    defaultCodexComplexitySafetyFactor,
		minTimeout:      defaultCodexComplexityTimeoutMin,
		maxTimeout:      defaultCodexComplexityTimeoutMax,
		timeoutBackoffK: defaultCodexComplexityTimeoutBackoff,
	}
}

func (p *CodexQuickComplexityPredictor) Predict(ctx context.Context, msg shareddomain.UnifiedMessage) (ComplexityAssessment, error) {
	if p == nil {
		return ComplexityAssessment{}, errors.New("complexity predictor is nil")
	}
	prompt := buildCodexComplexityPrompt(msg)
	if strings.TrimSpace(prompt) == "" {
		return ComplexityAssessment{}, errors.New("complexity prompt is empty")
	}

	command := strings.TrimSpace(p.command)
	if command == "" {
		command = defaultCodexComplexityPredictorCommand
	}
	runner := p.runner
	if runner == nil {
		runner = exec.CommandContext
	}

	if ctx == nil {
		ctx = context.Background()
	}
	timeout := p.nextTimeout()
	predictCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	outputFile, err := os.CreateTemp("", "alter0-complexity-predictor-*.json")
	if err != nil {
		return ComplexityAssessment{}, fmt.Errorf("create complexity predictor output file: %w", err)
	}
	outputPath := outputFile.Name()
	if closeErr := outputFile.Close(); closeErr != nil {
		_ = os.Remove(outputPath)
		return ComplexityAssessment{}, fmt.Errorf("prepare complexity predictor output file: %w", closeErr)
	}
	defer func() {
		_ = os.Remove(outputPath)
	}()

	args := []string{
		"exec",
		"--color", "never",
		"--skip-git-repo-check",
		"-o", outputPath,
		prompt,
	}
	cmd := runner(predictCtx, command, args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	startedAt := time.Now()
	runErr := cmd.Run()
	elapsed := time.Since(startedAt)
	p.observeExecution(elapsed, errors.Is(predictCtx.Err(), context.DeadlineExceeded))
	if runErr != nil {
		details := strings.TrimSpace(stderr.String())
		if details == "" {
			details = strings.TrimSpace(stdout.String())
		}
		if details == "" {
			return ComplexityAssessment{}, fmt.Errorf("complexity predictor command failed: %w", runErr)
		}
		return ComplexityAssessment{}, fmt.Errorf("complexity predictor command failed: %w: %s", runErr, details)
	}

	raw, err := os.ReadFile(outputPath)
	if err != nil {
		return ComplexityAssessment{}, fmt.Errorf("read complexity predictor output: %w", err)
	}
	assessment, err := parseCodexComplexityOutput(string(raw))
	if err != nil {
		return ComplexityAssessment{}, err
	}
	return normalizeComplexityAssessment(assessment), nil
}

func (p *CodexQuickComplexityPredictor) nextTimeout() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.currentTimeoutLocked()
}

func (p *CodexQuickComplexityPredictor) currentTimeoutLocked() time.Duration {
	baseTimeout := p.timeout
	if baseTimeout <= 0 {
		baseTimeout = defaultCodexComplexityPredictorTimeout
	}
	alpha := p.ewmaAlpha
	if alpha <= 0 || alpha >= 1 {
		alpha = defaultCodexComplexityEWMAAlpha
	}
	_ = alpha // alpha is validated here for observeExecution; keep both paths consistent.
	factor := p.safetyFactor
	if factor < 1 {
		factor = defaultCodexComplexitySafetyFactor
	}
	minTimeout := p.minTimeout
	if minTimeout <= 0 {
		minTimeout = defaultCodexComplexityTimeoutMin
	}
	maxTimeout := p.maxTimeout
	if maxTimeout <= 0 {
		maxTimeout = defaultCodexComplexityTimeoutMax
	}
	if maxTimeout < minTimeout {
		maxTimeout = minTimeout
	}

	timeout := baseTimeout
	if p.latencySamples > 0 && p.latencyEWMA > 0 {
		timeout = time.Duration(float64(p.latencyEWMA) * factor)
	}

	backoffK := p.timeoutBackoffK
	if backoffK <= 0 {
		backoffK = defaultCodexComplexityTimeoutBackoff
	}
	backoffN := p.timeoutBackoffN
	if backoffN < 0 {
		backoffN = 0
	}
	if backoffN > maxCodexComplexityTimeoutBackoffSteps {
		backoffN = maxCodexComplexityTimeoutBackoffSteps
	}
	if backoffN > 0 {
		timeout = time.Duration(float64(timeout) * (1 + backoffK*float64(backoffN)))
	}

	return clampDuration(timeout, minTimeout, maxTimeout)
}

func (p *CodexQuickComplexityPredictor) observeExecution(elapsed time.Duration, timedOut bool) {
	if elapsed <= 0 {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	alpha := p.ewmaAlpha
	if alpha <= 0 || alpha >= 1 {
		alpha = defaultCodexComplexityEWMAAlpha
	}
	if p.latencySamples == 0 || p.latencyEWMA <= 0 {
		p.latencyEWMA = elapsed
		p.latencySamples = 1
	} else {
		blended := (1-alpha)*float64(p.latencyEWMA) + alpha*float64(elapsed)
		p.latencyEWMA = time.Duration(math.Round(blended))
		p.latencySamples++
	}

	if timedOut {
		if p.timeoutBackoffN < maxCodexComplexityTimeoutBackoffSteps {
			p.timeoutBackoffN++
		}
		return
	}
	p.timeoutBackoffN = 0
}

func clampDuration(value time.Duration, minValue time.Duration, maxValue time.Duration) time.Duration {
	if minValue > 0 && value < minValue {
		return minValue
	}
	if maxValue > 0 && value > maxValue {
		return maxValue
	}
	return value
}

func buildCodexComplexityPrompt(msg shareddomain.UnifiedMessage) string {
	content := truncateRunes(strings.TrimSpace(msg.Content), maxCodexComplexityPromptRunes)
	taskType := strings.TrimSpace(metadataValue(msg.Metadata, MetadataTaskTypeKey))
	asyncHint := strings.TrimSpace(metadataValue(msg.Metadata, MetadataTaskAsyncMode))

	return strings.TrimSpace(
		`你是 alter0 的复杂度判定器。
要求：
1) 使用非思考模式，不输出推理过程。
2) 仅输出一行 JSON，不要 markdown，不要额外文本。
3) JSON 字段固定为：
   - estimated_duration_seconds: int（6-180）
   - complexity_level: "low"|"medium"|"high"
   - execution_mode: "streaming"|"async"
4) 当预计耗时 > 30 秒时 execution_mode 必须为 "async"，否则为 "streaming"。

task_type: ` + taskType + `
async_hint: ` + asyncHint + `
user_input:
` + content,
	)
}

func parseCodexComplexityOutput(raw string) (ComplexityAssessment, error) {
	normalized := strings.TrimSpace(raw)
	if normalized == "" {
		return ComplexityAssessment{}, errors.New("complexity predictor returned empty output")
	}
	normalized = stripCodeFence(normalized)
	normalized = extractJSONObject(normalized)
	if strings.TrimSpace(normalized) == "" {
		return ComplexityAssessment{}, errors.New("complexity predictor returned invalid json")
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(normalized), &payload); err != nil {
		return ComplexityAssessment{}, fmt.Errorf("parse complexity predictor json: %w", err)
	}

	estimated, err := parseEstimatedDuration(payload["estimated_duration_seconds"])
	if err != nil {
		return ComplexityAssessment{}, err
	}
	return ComplexityAssessment{
		EstimatedDurationSeconds: estimated,
		ComplexityLevel:          strings.ToLower(strings.TrimSpace(stringValue(payload["complexity_level"]))),
		ExecutionMode:            strings.ToLower(strings.TrimSpace(stringValue(payload["execution_mode"]))),
	}, nil
}

func parseEstimatedDuration(value any) (int, error) {
	switch v := value.(type) {
	case float64:
		return int(v), nil
	case float32:
		return int(v), nil
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case json.Number:
		parsed, err := v.Int64()
		if err != nil {
			return 0, errors.New("estimated_duration_seconds is invalid")
		}
		return int(parsed), nil
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, errors.New("estimated_duration_seconds is invalid")
		}
		return parsed, nil
	default:
		return 0, errors.New("estimated_duration_seconds is required")
	}
}

func stringValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	default:
		return ""
	}
}

func stripCodeFence(value string) string {
	trimmed := strings.TrimSpace(value)
	if !strings.HasPrefix(trimmed, "```") {
		return trimmed
	}
	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```JSON")
	trimmed = strings.TrimPrefix(trimmed, "```")
	if idx := strings.LastIndex(trimmed, "```"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	return strings.TrimSpace(trimmed)
}

func extractJSONObject(value string) string {
	start := strings.Index(value, "{")
	end := strings.LastIndex(value, "}")
	if start < 0 || end <= start {
		return strings.TrimSpace(value)
	}
	return strings.TrimSpace(value[start : end+1])
}

func truncateRunes(content string, maxRunes int) string {
	if maxRunes <= 0 {
		return strings.TrimSpace(content)
	}
	trimmed := strings.TrimSpace(content)
	if utf8.RuneCountInString(trimmed) <= maxRunes {
		return trimmed
	}
	runes := []rune(trimmed)
	return strings.TrimSpace(string(runes[:maxRunes]))
}

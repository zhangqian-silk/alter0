package application

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	shareddomain "alter0/internal/shared/domain"
)

const (
	MetadataEstimatedDurationSeconds = "estimated_duration_seconds"
	MetadataComplexityLevel          = "complexity_level"
	MetadataExecutionMode            = "execution_mode"
	MetadataComplexityFallback       = "complexity_fallback"
	MetadataComplexityPredictorMode  = "alter0.complexity.predictor_mode"

	ExecutionModeStreaming = "streaming"
	ExecutionModeAsync     = "async"

	ComplexityLevelLow    = "low"
	ComplexityLevelMedium = "medium"
	ComplexityLevelHigh   = "high"

	defaultAsyncRouteThresholdSeconds = 30
	defaultFallbackEstimateSeconds    = 45
)

var errComplexityPredictorUnavailable = errors.New("complexity predictor unavailable")

type ComplexityAssessment struct {
	EstimatedDurationSeconds int    `json:"estimated_duration_seconds"`
	ComplexityLevel          string `json:"complexity_level"`
	ExecutionMode            string `json:"execution_mode"`
	Fallback                 bool   `json:"fallback,omitempty"`
}

type ComplexityPredictor interface {
	Predict(ctx context.Context, msg shareddomain.UnifiedMessage) (ComplexityAssessment, error)
}

func (s *Service) AssessComplexity(msg shareddomain.UnifiedMessage) ComplexityAssessment {
	assessment, err := s.predictComplexity(msg)
	if err != nil {
		return ComplexityAssessment{
			EstimatedDurationSeconds: defaultFallbackEstimateSeconds,
			ComplexityLevel:          ComplexityLevelHigh,
			ExecutionMode:            ExecutionModeAsync,
			Fallback:                 true,
		}
	}
	return normalizeComplexityAssessment(assessment)
}

func (s *Service) predictComplexity(msg shareddomain.UnifiedMessage) (ComplexityAssessment, error) {
	if isComplexityPredictorUnavailable(msg) {
		return ComplexityAssessment{}, errComplexityPredictorUnavailable
	}

	trimmed := strings.TrimSpace(msg.Content)
	lowerContent := strings.ToLower(trimmed)
	taskType := strings.ToLower(strings.TrimSpace(metadataValue(msg.Metadata, MetadataTaskTypeKey)))
	mode := strings.ToLower(strings.TrimSpace(metadataValue(msg.Metadata, MetadataTaskAsyncMode)))
	hasArtifactHint := hasArtifactSignal(msg.Metadata, taskType, lowerContent, s.options.ArtifactKeywords)

	assessment := ComplexityAssessment{}
	if s.options.ComplexityPredictor != nil {
		predicted, err := s.options.ComplexityPredictor.Predict(context.Background(), msg)
		if err != nil {
			return ComplexityAssessment{}, errComplexityPredictorUnavailable
		}
		assessment = normalizeComplexityAssessment(predicted)
	} else {
		assessment = predictComplexityByRules(trimmed, hasArtifactHint, s.options.LongContentThreshold)
	}

	estimated := assessment.EstimatedDurationSeconds
	if estimated <= 0 {
		estimated = estimateDurationSeconds(trimmed, s.options.LongContentThreshold)
	}
	executionMode := strings.ToLower(strings.TrimSpace(assessment.ExecutionMode))
	if hasArtifactHint && estimated <= defaultAsyncRouteThresholdSeconds {
		estimated = defaultAsyncRouteThresholdSeconds + 12
	}

	switch mode {
	case "sync", "inline", "off", "disable":
		executionMode = ExecutionModeStreaming
		if estimated > defaultAsyncRouteThresholdSeconds {
			estimated = defaultAsyncRouteThresholdSeconds - 2
		}
	case "force", "async", "background":
		executionMode = ExecutionModeAsync
		if estimated <= defaultAsyncRouteThresholdSeconds {
			estimated = defaultAsyncRouteThresholdSeconds + 15
		}
	default:
		if estimated > defaultAsyncRouteThresholdSeconds || hasArtifactHint || executionMode == ExecutionModeAsync {
			executionMode = ExecutionModeAsync
			if estimated <= defaultAsyncRouteThresholdSeconds {
				estimated = defaultAsyncRouteThresholdSeconds + 1
			}
		} else {
			executionMode = ExecutionModeStreaming
		}
	}

	return ComplexityAssessment{
		EstimatedDurationSeconds: estimated,
		ComplexityLevel:          resolveComplexityLevel(estimated),
		ExecutionMode:            executionMode,
	}, nil
}

func predictComplexityByRules(content string, hasArtifactHint bool, threshold int) ComplexityAssessment {
	estimated := estimateDurationSeconds(content, threshold)
	executionMode := ExecutionModeStreaming
	if hasArtifactHint && estimated <= defaultAsyncRouteThresholdSeconds {
		estimated = defaultAsyncRouteThresholdSeconds + 12
	}
	if estimated > defaultAsyncRouteThresholdSeconds || hasArtifactHint {
		executionMode = ExecutionModeAsync
	}
	return ComplexityAssessment{
		EstimatedDurationSeconds: estimated,
		ComplexityLevel:          resolveComplexityLevel(estimated),
		ExecutionMode:            executionMode,
	}
}

func isComplexityPredictorUnavailable(msg shareddomain.UnifiedMessage) bool {
	mode := strings.ToLower(strings.TrimSpace(metadataValue(msg.Metadata, MetadataComplexityPredictorMode)))
	switch mode {
	case "unavailable", "timeout", "error":
		return true
	default:
		return false
	}
}

func hasArtifactSignal(metadata map[string]string, taskType string, lowerContent string, artifactKeywords []string) bool {
	if isTruthy(metadataValue(metadata, MetadataTaskArtifact)) {
		return true
	}
	if strings.Contains(taskType, "artifact") || strings.Contains(taskType, "export") {
		return true
	}
	if hasFileGenerationIntent(lowerContent) {
		return true
	}
	for _, keyword := range artifactKeywords {
		normalized := strings.ToLower(strings.TrimSpace(keyword))
		if normalized == "" {
			continue
		}
		if strings.Contains(lowerContent, normalized) {
			return true
		}
	}
	return false
}

func hasFileGenerationIntent(lowerContent string) bool {
	content := strings.TrimSpace(lowerContent)
	if content == "" {
		return false
	}

	verbs := []string{
		"generate",
		"create",
		"write",
		"save",
		"export",
		"produce",
		"draft",
		"生成",
		"创建",
		"写入",
		"输出",
		"导出",
		"产出",
		"整理",
	}
	targets := []string{
		"file",
		"document",
		"doc",
		"markdown",
		"readme",
		"report",
		"artifact",
		"zip",
		".md",
		"文档",
		"文件",
		"报告",
		"产物",
		"压缩包",
	}

	hasVerb := false
	for _, verb := range verbs {
		if strings.Contains(content, verb) {
			hasVerb = true
			break
		}
	}
	if !hasVerb {
		return false
	}

	for _, target := range targets {
		if strings.Contains(content, target) {
			return true
		}
	}
	return false
}

func estimateDurationSeconds(content string, threshold int) int {
	if threshold <= 0 {
		threshold = defaultLongContentThreshold
	}
	trimmed := strings.TrimSpace(content)
	runeCount := utf8.RuneCountInString(trimmed)
	if runeCount <= 0 {
		return 8
	}

	estimate := 8 + runeCount/12
	if runeCount >= threshold && estimate <= defaultAsyncRouteThresholdSeconds {
		estimate = defaultAsyncRouteThresholdSeconds + 1 + (runeCount-threshold)/18
	}
	if strings.Contains(trimmed, "\n") {
		estimate += 3
	}
	if strings.Contains(trimmed, "```") {
		estimate += 6
	}
	if estimate > 180 {
		estimate = 180
	}
	if estimate < 6 {
		estimate = 6
	}
	return estimate
}

func resolveComplexityLevel(estimatedSeconds int) string {
	switch {
	case estimatedSeconds <= 15:
		return ComplexityLevelLow
	case estimatedSeconds <= defaultAsyncRouteThresholdSeconds:
		return ComplexityLevelMedium
	default:
		return ComplexityLevelHigh
	}
}

func normalizeComplexityAssessment(assessment ComplexityAssessment) ComplexityAssessment {
	if assessment.EstimatedDurationSeconds <= 0 {
		assessment.EstimatedDurationSeconds = 10
	}
	switch strings.ToLower(strings.TrimSpace(assessment.ComplexityLevel)) {
	case ComplexityLevelLow, ComplexityLevelMedium, ComplexityLevelHigh:
	default:
		assessment.ComplexityLevel = resolveComplexityLevel(assessment.EstimatedDurationSeconds)
	}
	switch strings.ToLower(strings.TrimSpace(assessment.ExecutionMode)) {
	case ExecutionModeAsync:
		assessment.ExecutionMode = ExecutionModeAsync
	default:
		assessment.ExecutionMode = ExecutionModeStreaming
	}
	return assessment
}

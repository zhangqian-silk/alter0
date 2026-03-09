package application

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	shareddomain "alter0/internal/shared/domain"
)

const (
	MetadataEstimatedDurationSeconds = "estimated_duration_seconds"
	MetadataComplexityLevel          = "complexity_level"
	MetadataExecutionMode            = "execution_mode"
	MetadataComplexityFallback       = "complexity_fallback"
	MetadataComplexityPredictorMode  = "alter0.complexity.predictor_mode"
	MetadataTaskSummary              = "task_summary"
	MetadataTaskApproach             = "task_approach"

	ExecutionModeStreaming = "streaming"
	ExecutionModeAsync     = "async"

	ComplexityLevelLow    = "low"
	ComplexityLevelMedium = "medium"
	ComplexityLevelHigh   = "high"

	defaultAsyncRouteThreshold        = 5 * time.Minute
	defaultAsyncRouteThresholdSeconds = int(defaultAsyncRouteThreshold / time.Second)
	defaultMinimumEstimateSeconds     = 10
	defaultMaximumEstimateSeconds     = 60 * 60
	defaultArtifactDurationBoost      = 90
)

var errComplexityPredictorUnavailable = errors.New("complexity predictor unavailable")

type ComplexityAssessment struct {
	TaskSummary              string `json:"task_summary,omitempty"`
	TaskApproach             string `json:"task_approach,omitempty"`
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
		assessment = fallbackComplexityAssessment(
			msg,
			s.options.LongContentThreshold,
			s.asyncTriggerThresholdSeconds(),
			s.options.ArtifactKeywords,
		)
		assessment.Fallback = true
	}
	return s.finalizeComplexityAssessment(msg, assessment)
}

func (s *Service) predictComplexity(msg shareddomain.UnifiedMessage) (ComplexityAssessment, error) {
	if isComplexityPredictorUnavailable(msg) {
		return ComplexityAssessment{}, errComplexityPredictorUnavailable
	}

	trimmed := strings.TrimSpace(msg.Content)
	lowerContent := strings.ToLower(trimmed)
	taskType := strings.ToLower(strings.TrimSpace(metadataValue(msg.Metadata, MetadataTaskTypeKey)))
	hasArtifactHint := hasArtifactSignal(msg.Metadata, taskType, lowerContent, s.options.ArtifactKeywords)
	triggerThresholdSeconds := s.asyncTriggerThresholdSeconds()

	if s.options.ComplexityPredictor != nil {
		predicted, err := s.options.ComplexityPredictor.Predict(context.Background(), msg)
		if err != nil {
			return ComplexityAssessment{}, errComplexityPredictorUnavailable
		}
		return normalizeComplexityAssessment(
			predicted,
			trimmed,
			hasArtifactHint,
			s.options.LongContentThreshold,
			triggerThresholdSeconds,
		), nil
	}

	return predictComplexityByRules(
		trimmed,
		hasArtifactHint,
		s.options.LongContentThreshold,
		triggerThresholdSeconds,
	), nil
}

func (s *Service) finalizeComplexityAssessment(
	msg shareddomain.UnifiedMessage,
	assessment ComplexityAssessment,
) ComplexityAssessment {
	trimmed := strings.TrimSpace(msg.Content)
	lowerContent := strings.ToLower(trimmed)
	taskType := strings.ToLower(strings.TrimSpace(metadataValue(msg.Metadata, MetadataTaskTypeKey)))
	mode := strings.ToLower(strings.TrimSpace(metadataValue(msg.Metadata, MetadataTaskAsyncMode)))
	hasArtifactHint := hasArtifactSignal(msg.Metadata, taskType, lowerContent, s.options.ArtifactKeywords)
	triggerThresholdSeconds := s.asyncTriggerThresholdSeconds()

	assessment = normalizeComplexityAssessment(
		assessment,
		trimmed,
		hasArtifactHint,
		s.options.LongContentThreshold,
		triggerThresholdSeconds,
	)

	switch mode {
	case "sync", "inline", "off", "disable":
		assessment.ExecutionMode = ExecutionModeStreaming
		if assessment.EstimatedDurationSeconds >= triggerThresholdSeconds {
			assessment.EstimatedDurationSeconds = maxInt(defaultMinimumEstimateSeconds, triggerThresholdSeconds-1)
		}
	case "force", "async", "background":
		assessment.ExecutionMode = ExecutionModeAsync
		if assessment.EstimatedDurationSeconds <= triggerThresholdSeconds {
			assessment.EstimatedDurationSeconds = triggerThresholdSeconds + 60
		}
	default:
		if assessment.EstimatedDurationSeconds > triggerThresholdSeconds {
			assessment.ExecutionMode = ExecutionModeAsync
		} else {
			assessment.ExecutionMode = ExecutionModeStreaming
		}
	}

	assessment.ComplexityLevel = resolveComplexityLevel(
		assessment.EstimatedDurationSeconds,
		triggerThresholdSeconds,
	)
	return assessment
}

func fallbackComplexityAssessment(
	msg shareddomain.UnifiedMessage,
	threshold int,
	asyncThresholdSeconds int,
	artifactKeywords []string,
) ComplexityAssessment {
	trimmed := strings.TrimSpace(msg.Content)
	lowerContent := strings.ToLower(trimmed)
	taskType := strings.ToLower(strings.TrimSpace(metadataValue(msg.Metadata, MetadataTaskTypeKey)))
	hasArtifactHint := hasArtifactSignal(msg.Metadata, taskType, lowerContent, artifactKeywords)
	return predictComplexityByRules(trimmed, hasArtifactHint, threshold, asyncThresholdSeconds)
}

func predictComplexityByRules(
	content string,
	hasArtifactHint bool,
	threshold int,
	asyncThresholdSeconds int,
) ComplexityAssessment {
	estimated := estimateDurationSeconds(content, threshold, hasArtifactHint, asyncThresholdSeconds)
	executionMode := ExecutionModeStreaming
	if estimated > asyncThresholdSeconds {
		executionMode = ExecutionModeAsync
	}
	return ComplexityAssessment{
		TaskSummary:              buildTaskSummaryName(content),
		TaskApproach:             buildTaskApproach(content, hasArtifactHint),
		EstimatedDurationSeconds: estimated,
		ComplexityLevel:          resolveComplexityLevel(estimated, asyncThresholdSeconds),
		ExecutionMode:            executionMode,
	}
}

func (s *Service) asyncTriggerThresholdSeconds() int {
	threshold := int(s.options.AsyncTriggerThreshold / time.Second)
	if threshold <= 0 {
		return defaultAsyncRouteThresholdSeconds
	}
	return threshold
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

func estimateDurationSeconds(
	content string,
	threshold int,
	hasArtifactHint bool,
	asyncThresholdSeconds int,
) int {
	if threshold <= 0 {
		threshold = defaultLongContentThreshold
	}
	if asyncThresholdSeconds <= 0 {
		asyncThresholdSeconds = defaultAsyncRouteThresholdSeconds
	}

	trimmed := strings.TrimSpace(content)
	runeCount := utf8.RuneCountInString(trimmed)
	if runeCount <= 0 {
		return defaultMinimumEstimateSeconds
	}

	estimate := defaultMinimumEstimateSeconds + runeCount/6
	if runeCount >= threshold {
		estimate += 30 + (runeCount-threshold)/4
	}

	lineBreaks := strings.Count(trimmed, "\n")
	if lineBreaks > 0 {
		estimate += 12 + minInt(lineBreaks, 12)*4
	}
	if strings.Contains(trimmed, "```") {
		estimate += 45
	}
	if strings.Contains(trimmed, "http://") || strings.Contains(trimmed, "https://") {
		estimate += 25
	}
	if hasArtifactHint {
		estimate += defaultArtifactDurationBoost
	}
	if runeCount >= threshold*4 {
		estimate += asyncThresholdSeconds / 3
	}

	if estimate > defaultMaximumEstimateSeconds {
		estimate = defaultMaximumEstimateSeconds
	}
	if estimate < defaultMinimumEstimateSeconds {
		estimate = defaultMinimumEstimateSeconds
	}
	return estimate
}

func resolveComplexityLevel(estimatedSeconds int, asyncThresholdSeconds int) string {
	if asyncThresholdSeconds <= 0 {
		asyncThresholdSeconds = defaultAsyncRouteThresholdSeconds
	}

	lowThreshold := asyncThresholdSeconds / 3
	if lowThreshold < 30 {
		lowThreshold = 30
	}
	if lowThreshold > 90 {
		lowThreshold = 90
	}

	switch {
	case estimatedSeconds <= lowThreshold:
		return ComplexityLevelLow
	case estimatedSeconds <= asyncThresholdSeconds:
		return ComplexityLevelMedium
	default:
		return ComplexityLevelHigh
	}
}

func normalizeComplexityAssessment(
	assessment ComplexityAssessment,
	content string,
	hasArtifactHint bool,
	threshold int,
	asyncThresholdSeconds int,
) ComplexityAssessment {
	if assessment.EstimatedDurationSeconds <= 0 {
		assessment.EstimatedDurationSeconds = estimateDurationSeconds(
			content,
			threshold,
			hasArtifactHint,
			asyncThresholdSeconds,
		)
	}
	if assessment.EstimatedDurationSeconds > defaultMaximumEstimateSeconds {
		assessment.EstimatedDurationSeconds = defaultMaximumEstimateSeconds
	}
	if assessment.EstimatedDurationSeconds < defaultMinimumEstimateSeconds {
		assessment.EstimatedDurationSeconds = defaultMinimumEstimateSeconds
	}

	assessment.TaskSummary = normalizeAssessmentSummary(assessment.TaskSummary, content)
	assessment.TaskApproach = normalizeAssessmentApproach(assessment.TaskApproach, content, hasArtifactHint)
	assessment.ComplexityLevel = resolveComplexityLevel(assessment.EstimatedDurationSeconds, asyncThresholdSeconds)

	switch strings.ToLower(strings.TrimSpace(assessment.ExecutionMode)) {
	case ExecutionModeAsync:
		assessment.ExecutionMode = ExecutionModeAsync
	default:
		assessment.ExecutionMode = ExecutionModeStreaming
	}
	return assessment
}

func normalizeAssessmentSummary(summary string, content string) string {
	normalized := strings.TrimSpace(summary)
	if normalized != "" {
		return normalized
	}
	return buildTaskSummaryName(content)
}

func normalizeAssessmentApproach(approach string, content string, hasArtifactHint bool) string {
	normalized := strings.TrimSpace(approach)
	if normalized != "" {
		return normalized
	}
	return buildTaskApproach(content, hasArtifactHint)
}

func buildTaskSummaryName(content string) string {
	snippet := strings.TrimSpace(summarySnippet(content, 24))
	snippet = strings.Trim(snippet, " \t\r\n,.;:!?，。；：！？'\"`()[]{}<>")
	if snippet == "" {
		return "处理当前请求"
	}
	return "处理「" + snippet + "」"
}

func buildTaskApproach(content string, hasArtifactHint bool) string {
	trimmed := strings.TrimSpace(content)
	switch {
	case hasArtifactHint:
		return "先梳理需求与交付边界，再分步执行生成或修改，最后校验结果并整理可交付物。"
	case strings.Contains(trimmed, "```") || strings.Contains(trimmed, "\n"):
		return "先分析上下文与约束，再拆分主要步骤执行，最后汇总结果与后续动作。"
	case utf8.RuneCountInString(trimmed) >= 240:
		return "先提炼目标和关键限制，再按优先级逐步处理，最后输出精简结论。"
	default:
		return "直接理解目标并完成必要操作，随后返回结果。"
	}
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

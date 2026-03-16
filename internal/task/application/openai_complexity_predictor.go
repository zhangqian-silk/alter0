package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	llmdomain "alter0/internal/llm/domain"
	shareddomain "alter0/internal/shared/domain"
)

type defaultLLMClientSource interface {
	GetDefaultClient(ctx context.Context) (llmdomain.LLMClient, error)
}

type OpenAIComplexityPredictor struct {
	clientSource defaultLLMClientSource
	fallback     ComplexityPredictor
	logger       *slog.Logger
}

type openAIComplexityPayload struct {
	TaskSummary              string `json:"task_summary"`
	TaskApproach             string `json:"task_approach"`
	EstimatedDurationSeconds int    `json:"estimated_duration_seconds"`
	ComplexityLevel          string `json:"complexity_level"`
	ExecutionMode            string `json:"execution_mode"`
}

func NewOpenAIComplexityPredictor(
	clientSource defaultLLMClientSource,
	fallback ComplexityPredictor,
	logger *slog.Logger,
) *OpenAIComplexityPredictor {
	if logger == nil {
		logger = slog.Default()
	}
	return &OpenAIComplexityPredictor{
		clientSource: clientSource,
		fallback:     fallback,
		logger:       logger,
	}
}

func (p *OpenAIComplexityPredictor) Predict(ctx context.Context, msg shareddomain.UnifiedMessage) (ComplexityAssessment, error) {
	if p == nil {
		return ComplexityAssessment{}, errors.New("complexity predictor is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := p.resolveClient(ctx)
	if err != nil {
		return p.fallbackPredict(ctx, msg, err)
	}
	request := llmdomain.ChatRequest{
		Messages: []llmdomain.Message{
			{
				Role: "system",
				Content: strings.TrimSpace(`You classify request complexity for alter0.
Return strict JSON only.

Required fields:
- task_summary: concise Chinese task summary
- task_approach: concise Chinese execution approach
- estimated_duration_seconds: integer
- complexity_level: one of low, medium, high
- execution_mode: one of streaming, async

Rules:
- If estimated_duration_seconds is 300 or more, execution_mode must be async.
- If estimated_duration_seconds is below 300, execution_mode must be streaming.
- Use medium or high when the work is likely to take 300 seconds or more.
- Prefer conservative estimates for requests involving multi-step analysis, code changes, repository work, or file generation.`),
			},
			{
				Role:    "user",
				Content: buildOpenAIComplexityPrompt(msg),
			},
		},
	}
	response, err := client.Chat(ctx, request)
	if err != nil {
		return p.fallbackPredict(ctx, msg, err)
	}
	payload, err := parseOpenAIComplexityPayload(response.Message.Content)
	if err != nil {
		return p.fallbackPredict(ctx, msg, err)
	}
	return ComplexityAssessment{
		TaskSummary:              strings.TrimSpace(payload.TaskSummary),
		TaskApproach:             strings.TrimSpace(payload.TaskApproach),
		EstimatedDurationSeconds: payload.EstimatedDurationSeconds,
		ComplexityLevel:          strings.ToLower(strings.TrimSpace(payload.ComplexityLevel)),
		ExecutionMode:            strings.ToLower(strings.TrimSpace(payload.ExecutionMode)),
	}, nil
}

func (p *OpenAIComplexityPredictor) resolveClient(ctx context.Context) (llmdomain.LLMClient, error) {
	if p.clientSource == nil {
		return nil, errors.New("default llm client source is unavailable")
	}
	client, err := p.clientSource.GetDefaultClient(ctx)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, errors.New("default llm client is unavailable")
	}
	return client, nil
}

func (p *OpenAIComplexityPredictor) fallbackPredict(
	ctx context.Context,
	msg shareddomain.UnifiedMessage,
	cause error,
) (ComplexityAssessment, error) {
	if p.logger != nil && cause != nil {
		p.logger.Warn("openai complexity predictor unavailable, falling back",
			slog.String("session_id", strings.TrimSpace(msg.SessionID)),
			slog.String("message_id", strings.TrimSpace(msg.MessageID)),
			slog.String("error", strings.TrimSpace(cause.Error())),
		)
	}
	if p.fallback == nil {
		if cause == nil {
			cause = errComplexityPredictorUnavailable
		}
		return ComplexityAssessment{}, cause
	}
	return p.fallback.Predict(ctx, msg)
}

func buildOpenAIComplexityPrompt(msg shareddomain.UnifiedMessage) string {
	lines := []string{
		"session_id: " + strings.TrimSpace(msg.SessionID),
		"message_id: " + strings.TrimSpace(msg.MessageID),
		"channel_id: " + strings.TrimSpace(msg.ChannelID),
		"channel_type: " + strings.TrimSpace(string(msg.ChannelType)),
		"trigger_type: " + strings.TrimSpace(string(msg.TriggerType)),
		"user_prompt:",
		strings.TrimSpace(msg.Content),
	}
	if len(msg.Metadata) > 0 {
		rawMetadata, err := json.Marshal(msg.Metadata)
		if err == nil {
			lines = append(lines, "metadata: "+string(rawMetadata))
		}
	}
	return strings.Join(lines, "\n")
}

func parseOpenAIComplexityPayload(raw string) (openAIComplexityPayload, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return openAIComplexityPayload{}, errors.New("empty complexity response")
	}
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start >= 0 && end >= start {
		trimmed = trimmed[start : end+1]
	}
	payload := openAIComplexityPayload{}
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return openAIComplexityPayload{}, fmt.Errorf("parse complexity response: %w", err)
	}
	return payload, nil
}

package application

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"testing"
	"time"

	orchdomain "alter0/internal/orchestration/domain"
	shareddomain "alter0/internal/shared/domain"
)

type qualityProbeExecutor struct {
	requiredFacts []string
	prompts       []string
}

func (e *qualityProbeExecutor) ExecuteNaturalLanguage(_ context.Context, msg shareddomain.UnifiedMessage) (shareddomain.ExecutionResult, error) {
	e.prompts = append(e.prompts, msg.Content)
	content := strings.ToLower(msg.Content)
	score := 0
	for _, fact := range e.requiredFacts {
		if strings.Contains(content, strings.ToLower(fact)) {
			score++
		}
	}
	return shareddomain.ExecutionResult{Output: fmt.Sprintf("quality_score:%d", score)}, nil
}

func (e *qualityProbeExecutor) LastPrompt() string {
	if len(e.prompts) == 0 {
		return ""
	}
	return e.prompts[len(e.prompts)-1]
}

func TestContextCompressionAcceptanceTokenReductionAndQuality(t *testing.T) {
	requiredFacts := []string{
		"release_window: friday 22:00",
		"slo_target: 99.95%",
		"rollback_owner: sre-oncall",
		"incident_channel: #release-war-room",
	}

	baselineTokens, baselineScore := runCompressionScenario(t, SessionMemoryOptions{
		MaxTurns:                 24,
		TTL:                      time.Hour,
		MaxSnippets:              220,
		CompressionTriggerTokens: 20_000,
		CompressionSummaryTokens: 160,
		CompressionRetainTurns:   4,
		CompressionMaxFacts:      8,
	}, requiredFacts)

	compressedTokens, compressedScore := runCompressionScenario(t, SessionMemoryOptions{
		MaxTurns:                 24,
		TTL:                      time.Hour,
		MaxSnippets:              220,
		CompressionTriggerTokens: 140,
		CompressionSummaryTokens: 40,
		CompressionRetainTurns:   1,
		CompressionMaxFacts:      6,
	}, requiredFacts)

	if baselineScore < len(requiredFacts)-1 {
		t.Fatalf("expected baseline score close to full facts, got %d", baselineScore)
	}
	if compressedScore < baselineScore-1 {
		t.Fatalf("expected compressed quality not degrade significantly, baseline=%d compressed=%d", baselineScore, compressedScore)
	}
	if compressedTokens >= baselineTokens {
		t.Fatalf("expected compressed prompt tokens lower than baseline, got baseline=%d compressed=%d", baselineTokens, compressedTokens)
	}

	reduction := float64(baselineTokens-compressedTokens) / float64(baselineTokens)
	t.Logf("context compression token reduction: baseline=%d compressed=%d reduction=%.2f%%", baselineTokens, compressedTokens, reduction*100)
	if reduction < 0.30 {
		t.Fatalf("expected token reduction >= 30%%, got %.2f%%", reduction*100)
	}
}

func runCompressionScenario(t *testing.T, options SessionMemoryOptions, requiredFacts []string) (int, int) {
	t.Helper()

	executor := &qualityProbeExecutor{requiredFacts: requiredFacts}
	service := NewServiceWithOptions(
		&stubClassifier{intent: orchdomain.Intent{Type: orchdomain.IntentTypeNL}},
		&stubRegistry{},
		executor,
		newSpyTelemetry(),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		WithSessionMemoryOptions(options),
	)

	filler := strings.Repeat(" regression-checklist canary-verification release-audit", 10)
	messages := []string{
		"Please remember release_window: friday 22:00 and slo_target: 99.95% for go-live" + filler,
		"Also remember rollback_owner: sre-oncall and region: us-east-1 before release" + filler,
		"Track incident_channel: #release-war-room and canary_ratio: 10%" + filler,
		"Keep verification_gate: smoke+synthetic and rollback_window: 30m" + filler,
		"Add business_guardrail: payment-latency<200ms and db_error_budget: 0.1%" + filler,
		"Store customer_notice_window: 48h and freeze_window: friday 18:00" + filler,
		"Capture release_owner: platform-ops and traffic_shift_step: 25%" + filler,
		"Track dependency_lock: enabled and deploy_freeze_exception: false" + filler,
		"Remember database_guard: read-replica-check and queue_backlog_limit: 5k" + filler,
		"Keep oncall_handoff: required and rollback_drill: weekly" + filler,
	}

	start := time.Date(2026, 3, 3, 8, 0, 0, 0, time.UTC)
	for idx, content := range messages {
		msg := scenarioMessage("session-compression", fmt.Sprintf("m-%02d", idx+1), content, start.Add(time.Duration(idx)*time.Minute))
		if _, err := service.Handle(context.Background(), msg); err != nil {
			t.Fatalf("handle message %d failed: %v", idx+1, err)
		}
	}

	finalMessage := scenarioMessage(
		"session-compression",
		"m-final",
		"Based on previous facts, draft the final release checklist and owners.",
		start.Add(time.Duration(len(messages))*time.Minute),
	)
	result, err := service.Handle(context.Background(), finalMessage)
	if err != nil {
		t.Fatalf("handle final message failed: %v", err)
	}

	finalPrompt := executor.LastPrompt()
	if finalPrompt == "" {
		t.Fatalf("expected captured final prompt")
	}
	score, err := parseQualityScore(result.Output)
	if err != nil {
		t.Fatalf("parse quality score: %v", err)
	}
	return estimateTokenCount(finalPrompt), score
}

func scenarioMessage(sessionID, messageID, content string, at time.Time) shareddomain.UnifiedMessage {
	return shareddomain.UnifiedMessage{
		MessageID:   messageID,
		SessionID:   sessionID,
		ChannelID:   "cli-default",
		ChannelType: shareddomain.ChannelTypeCLI,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     content,
		TraceID:     "trace-" + messageID,
		ReceivedAt:  at,
	}
}

func parseQualityScore(output string) (int, error) {
	const prefix = "quality_score:"
	if !strings.HasPrefix(output, prefix) {
		return 0, fmt.Errorf("unexpected output %q", output)
	}
	return strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(output, prefix)))
}

package application

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orchdomain "alter0/internal/orchestration/domain"
	shareddomain "alter0/internal/shared/domain"
)

func TestDailyAndLongTermMemoryMarkdownAcceptance(t *testing.T) {
	rootDir := t.TempDir()
	dailyDir := filepath.Join(rootDir, "memory")
	longTermPath := filepath.Join(dailyDir, "long-term", "MEMORY.md")

	executor := &stubExecutor{
		outputs: []string{
			"Preference saved.",
			"plan alpha: add verification and rollback.",
			"plan alpha: confirm canary and rollback windows.",
			"Using concise bullet answers with rollout details.",
		},
	}
	service := NewServiceWithOptions(
		&stubClassifier{intent: orchdomain.Intent{Type: orchdomain.IntentTypeNL}},
		&stubRegistry{},
		executor,
		newSpyTelemetry(),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		WithSessionMemoryOptions(SessionMemoryOptions{
			MaxTurns:                 8,
			TTL:                      24 * time.Hour,
			MaxSnippets:              220,
			CompressionTriggerTokens: 70,
			CompressionSummaryTokens: 60,
			CompressionRetainTurns:   2,
			CompressionMaxFacts:      6,
			DailyMemoryDir:           dailyDir,
		}),
		WithLongTermMemoryOptions(LongTermMemoryOptions{
			MaxEntriesPerScope:   32,
			MaxHits:              4,
			MaxSnippet:           200,
			InjectionTokenBudget: 160,
			PersistencePath:      longTermPath,
			WritePolicy:          longTermMemoryWritePolicyWriteThrough,
		}),
	)

	day1 := time.Date(2026, 3, 3, 9, 0, 0, 0, time.UTC)
	day2 := day1.Add(24 * time.Hour)

	msg1 := acceptanceMessage("r025-day1-1", "session-day1", "user-r025", "remember my style", day1, map[string]string{
		longTermMemoryTenantMetadataKey:   "tenant-r025",
		longTermMemoryStrategyMetadataKey: "add",
		longTermMemoryKindMetadataKey:     "preference",
		longTermMemoryKeyMetadataKey:      "response-style",
		longTermMemoryValueMetadataKey:    "concise bullet answers",
	})
	if _, err := service.Handle(context.Background(), msg1); err != nil {
		t.Fatalf("handle day1 explicit long-term memory write: %v", err)
	}

	msg2 := acceptanceMessage("r025-day1-2", "session-day1", "user-r025", "release_window: friday 22:00 rollback_owner: sre-oncall"+strings.Repeat(" release guardrail", 8), day1.Add(time.Minute), map[string]string{
		longTermMemoryTenantMetadataKey: "tenant-r025",
	})
	if _, err := service.Handle(context.Background(), msg2); err != nil {
		t.Fatalf("handle day1 compression writeback: %v", err)
	}

	msg2b := acceptanceMessage("r025-day1-3", "session-day1", "user-r025", "slo_target: 99.95% incident_channel: #release-war-room"+strings.Repeat(" release guardrail", 8), day1.Add(2*time.Minute), map[string]string{
		longTermMemoryTenantMetadataKey: "tenant-r025",
	})
	if _, err := service.Handle(context.Background(), msg2b); err != nil {
		t.Fatalf("handle day1 compression accumulation: %v", err)
	}

	msg3 := acceptanceMessage("r025-day2-1", "session-day2", "user-r025", "Please keep concise bullet style for this session", day2, map[string]string{
		longTermMemoryTenantMetadataKey: "tenant-r025",
	})
	result, err := service.Handle(context.Background(), msg3)
	if err != nil {
		t.Fatalf("handle day2 long-term retrieval: %v", err)
	}
	if result.Metadata["memory_long_term_injected"] != "true" {
		t.Fatalf("expected cross-session long-term memory injected, got %q", result.Metadata["memory_long_term_injected"])
	}

	day1File := filepath.Join(dailyDir, "2026-03-03.md")
	day1Raw, err := os.ReadFile(day1File)
	if err != nil {
		t.Fatalf("read day1 markdown memory file: %v", err)
	}
	day1Content := string(day1Raw)
	if !strings.Contains(day1Content, "# Daily Memory 2026-03-03") {
		t.Fatalf("expected day1 markdown heading, got %q", day1Content)
	}
	if !strings.Contains(day1Content, "## L2") {
		t.Fatalf("expected day1 compressed tier persisted, got %q", day1Content)
	}

	day2File := filepath.Join(dailyDir, "2026-03-04.md")
	if _, err := os.Stat(day2File); err != nil {
		t.Fatalf("expected day2 markdown memory file: %v", err)
	}

	candidateFile := filepath.Join(dailyDir, "long-term", "2026-03-03.md")
	if _, err := os.Stat(candidateFile); err != nil {
		t.Fatalf("expected day1 long-term candidate markdown file: %v", err)
	}

	longTermRaw, err := os.ReadFile(longTermPath)
	if err != nil {
		t.Fatalf("read long-term markdown persistence file: %v", err)
	}
	if !strings.Contains(string(longTermRaw), "# Long-Term Memory") {
		t.Fatalf("expected long-term markdown heading, got %q", string(longTermRaw))
	}
}

func acceptanceMessage(
	messageID string,
	sessionID string,
	userID string,
	content string,
	at time.Time,
	metadata map[string]string,
) shareddomain.UnifiedMessage {
	meta := map[string]string{}
	for key, value := range metadata {
		meta[key] = value
	}
	if len(meta) == 0 {
		meta = nil
	}
	return shareddomain.UnifiedMessage{
		MessageID:   messageID,
		SessionID:   sessionID,
		UserID:      userID,
		ChannelID:   "cli-default",
		ChannelType: shareddomain.ChannelTypeCLI,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     content,
		Metadata:    meta,
		TraceID:     "trace-" + messageID,
		ReceivedAt:  at,
	}
}

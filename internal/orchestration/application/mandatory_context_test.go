package application

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMandatoryContextStoreLoadsAndHotReloads(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "SOUL.md")

	firstContent := "# Soul\nresponse_style: concise bullet answers\n"
	if err := os.WriteFile(filePath, []byte(firstContent), 0o644); err != nil {
		t.Fatalf("write first content: %v", err)
	}

	store := newMandatoryContextStore(MandatoryContextOptions{FilePath: filePath})
	firstSnapshot := store.Snapshot(time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC))
	if !firstSnapshot.Active() {
		t.Fatalf("expected first snapshot active")
	}
	if firstSnapshot.Version == "" {
		t.Fatalf("expected first snapshot version")
	}
	if got := firstSnapshot.Rules["response_style"]; got != "concise bullet answers" {
		t.Fatalf("unexpected parsed rule value: %q", got)
	}

	secondContent := "# Soul\nresponse_style: detailed narrative\n"
	if err := os.WriteFile(filePath, []byte(secondContent), 0o644); err != nil {
		t.Fatalf("write second content: %v", err)
	}

	secondSnapshot := store.Snapshot(time.Date(2026, 3, 3, 10, 1, 0, 0, time.UTC))
	if !secondSnapshot.Active() {
		t.Fatalf("expected second snapshot active")
	}
	if secondSnapshot.Version == firstSnapshot.Version {
		t.Fatalf("expected version change after hot reload")
	}
	if !strings.Contains(secondSnapshot.Content, "detailed narrative") {
		t.Fatalf("expected latest content after hot reload, got %q", secondSnapshot.Content)
	}
	if secondSnapshot.LoadedAt.IsZero() {
		t.Fatalf("expected loaded_at audit field")
	}
	if secondSnapshot.UpdatedAt.IsZero() {
		t.Fatalf("expected updated_at audit field")
	}
}

func TestBuildMandatoryContextPromptPlacesSectionAtTop(t *testing.T) {
	snapshot := mandatoryContextSnapshot{
		FilePath:  "SOUL.md",
		Content:   "response_style: concise",
		Version:   "abc123",
		UpdatedAt: time.Date(2026, 3, 3, 9, 0, 0, 0, time.UTC),
	}
	basePrompt := "[SESSION SHORT TERM MEMORY]\nCurrent user input:\nhello"

	prompt := buildMandatoryContextPrompt(basePrompt, snapshot)
	if !strings.HasPrefix(prompt, "[MANDATORY CONTEXT]") {
		t.Fatalf("expected prompt prefixed by mandatory section, got %q", prompt)
	}
	if !strings.Contains(prompt, "response_style: concise") {
		t.Fatalf("expected mandatory content in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, basePrompt) {
		t.Fatalf("expected original prompt preserved, got %q", prompt)
	}
}

func TestApplyMandatoryContextToLongTermMemoryFiltersConflict(t *testing.T) {
	mandatorySnapshot := mandatoryContextSnapshot{
		FilePath: "SOUL.md",
		Content:  "response_style: concise",
		Rules: map[string]string{
			"response_style": "concise",
		},
	}
	longTermSnapshot := longTermMemorySnapshot{
		Hits: []longTermMemoryHit{
			{
				Entry: longTermMemoryEntry{
					Key:   "response_style",
					Value: "detailed narrative",
				},
				Score: 2.0,
			},
		},
	}

	filtered, conflicts := applyMandatoryContextToLongTermMemory(longTermSnapshot, mandatorySnapshot)
	if len(filtered.Hits) != 0 {
		t.Fatalf("expected conflicting long-term memory filtered out, got %d", len(filtered.Hits))
	}
	if len(conflicts) != 1 {
		t.Fatalf("expected one conflict, got %d", len(conflicts))
	}

	metadata := mandatoryContextConflictMetadata(conflicts)
	if metadata[mandatoryContextConflictCountMetadataKey] != "1" {
		t.Fatalf("expected conflict metadata count=1, got %q", metadata[mandatoryContextConflictCountMetadataKey])
	}
	if metadata[mandatoryContextConflictKeysMetadataKey] != "response_style" {
		t.Fatalf("expected conflict key response_style, got %q", metadata[mandatoryContextConflictKeysMetadataKey])
	}
}

package infrastructure

import (
	"context"
	"strings"
	"testing"

	controldomain "alter0/internal/control/domain"
	execapp "alter0/internal/execution/application"
)

type stubSessionProfileProcessor struct {
	output       string
	lastContent  string
	lastMetadata map[string]string
}

func (s *stubSessionProfileProcessor) Process(_ context.Context, content string, metadata map[string]string) (string, error) {
	s.lastContent = content
	s.lastMetadata = map[string]string{}
	for key, value := range metadata {
		s.lastMetadata[key] = value
	}
	return s.output, nil
}

func TestSessionProfileCodexExtractorBuildsNarrowPromptAndParsesJSON(t *testing.T) {
	processor := &stubSessionProfileProcessor{
		output: "```json\n{\"city\":\"东京\",\"days\":4}\n```",
	}
	extractor := NewSessionProfileCodexExtractor(processor)

	patch, err := extractor.ExtractPatch(context.Background(), execapp.SessionProfileExtractionRequest{
		Agent: controldomain.Agent{
			ID: "travel",
			SessionProfileFields: []controldomain.AgentSessionProfileField{
				{Key: "city", Label: "City"},
				{Key: "days", Label: "Days"},
				{Key: "preview_subdomain", Label: "Preview Subdomain", ReadOnly: true},
			},
		},
		ExistingAttributes: map[string]string{"city": "大阪"},
	})
	if err != nil {
		t.Fatalf("ExtractPatch() error = %v", err)
	}
	if patch["city"] != "东京" || patch["days"] != "4" {
		t.Fatalf("unexpected patch: %+v", patch)
	}
	if processor.lastMetadata["alter0.codex.runtime_strategy"] != "plain" {
		t.Fatalf("expected plain codex runtime strategy, got %+v", processor.lastMetadata)
	}
	if processor.lastContent == "" {
		t.Fatal("expected prompt content")
	}
	if contains := "preview_subdomain"; processor.lastContent != "" && strings.Contains(processor.lastContent, contains) {
		t.Fatalf("expected readonly field %q excluded from prompt, got %q", contains, processor.lastContent)
	}
}

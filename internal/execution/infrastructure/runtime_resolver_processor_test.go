package infrastructure

import (
	"context"
	"errors"
	"testing"

	execdomain "alter0/internal/execution/domain"
	llmdomain "alter0/internal/llm/domain"
)

type stubProviderSource struct {
	defaultProvider *llmdomain.ModelProvider
	providers       map[string]*llmdomain.ModelProvider
}

func (s stubProviderSource) GetDefaultProvider(_ context.Context) (*llmdomain.ModelProvider, error) {
	return s.defaultProvider, nil
}

func (s stubProviderSource) GetProvider(_ context.Context, providerID string) (*llmdomain.ModelProvider, error) {
	if s.providers == nil {
		return nil, nil
	}
	return s.providers[providerID], nil
}

type recordingRuntimeProcessor struct {
	output      string
	err         error
	called      int
	lastContent string
	lastMeta    map[string]string
}

func (p *recordingRuntimeProcessor) Process(_ context.Context, content string, metadata map[string]string) (string, error) {
	p.called++
	p.lastContent = content
	p.lastMeta = copyTestMetadata(metadata)
	if p.err != nil {
		return "", p.err
	}
	return p.output, nil
}

func copyTestMetadata(metadata map[string]string) map[string]string {
	out := make(map[string]string, len(metadata))
	for key, value := range metadata {
		out[key] = value
	}
	return out
}

func TestRuntimeResolverProcessorUsesClaudeWhenProviderIsEnabled(t *testing.T) {
	claude := &recordingRuntimeProcessor{output: "claude result"}
	codex := &recordingRuntimeProcessor{output: "codex result"}
	processor := NewRuntimeResolverProcessor(RuntimeResolverOptions{
		ProviderSource: stubProviderSource{
			defaultProvider: &llmdomain.ModelProvider{
				ID:           "openrouter",
				APIKey:       "token",
				DefaultModel: "anthropic/claude-3.7-sonnet",
				IsEnabled:    true,
			},
		},
		Claude: claude,
		Codex:  codex,
	})

	output, err := processor.Process(context.Background(), "整理方案", map[string]string{})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if output != "claude result" {
		t.Fatalf("Process() output = %q, want claude result", output)
	}
	if claude.called != 1 {
		t.Fatalf("claude called %d times, want 1", claude.called)
	}
	if codex.called != 0 {
		t.Fatalf("codex called %d times, want 0", codex.called)
	}
	if claude.lastMeta[execdomain.ExecutionSourceMetadataKey] != execdomain.ExecutionSourceClaudeCode {
		t.Fatalf("execution source = %q, want %q", claude.lastMeta[execdomain.ExecutionSourceMetadataKey], execdomain.ExecutionSourceClaudeCode)
	}
	if claude.lastMeta[execdomain.LLMProviderIDMetadataKey] != "openrouter" {
		t.Fatalf("provider id = %q, want openrouter", claude.lastMeta[execdomain.LLMProviderIDMetadataKey])
	}
}

func TestRuntimeResolverProcessorFallsBackToCodexWhenClaudeFails(t *testing.T) {
	claude := &recordingRuntimeProcessor{err: errors.New("claude auth failed")}
	codex := &recordingRuntimeProcessor{output: "codex result"}
	processor := NewRuntimeResolverProcessor(RuntimeResolverOptions{
		ProviderSource: stubProviderSource{
			defaultProvider: &llmdomain.ModelProvider{
				ID:           "openrouter",
				APIKey:       "token",
				DefaultModel: "anthropic/claude-3.7-sonnet",
				IsEnabled:    true,
			},
		},
		Claude: claude,
		Codex:  codex,
	})

	output, err := processor.Process(context.Background(), "整理方案", map[string]string{})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if output != "codex result" {
		t.Fatalf("Process() output = %q, want codex result", output)
	}
	if claude.called != 1 || codex.called != 1 {
		t.Fatalf("calls claude=%d codex=%d, want 1/1", claude.called, codex.called)
	}
	if codex.lastMeta[execdomain.ExecutionSourceMetadataKey] != execdomain.ExecutionSourceCodexCLI {
		t.Fatalf("execution source = %q, want %q", codex.lastMeta[execdomain.ExecutionSourceMetadataKey], execdomain.ExecutionSourceCodexCLI)
	}
}

func TestRuntimeResolverProcessorHonorsExplicitCodexEngine(t *testing.T) {
	claude := &recordingRuntimeProcessor{output: "claude result"}
	codex := &recordingRuntimeProcessor{output: "codex result"}
	processor := NewRuntimeResolverProcessor(RuntimeResolverOptions{
		ProviderSource: stubProviderSource{
			defaultProvider: &llmdomain.ModelProvider{
				ID:           "openrouter",
				APIKey:       "token",
				DefaultModel: "anthropic/claude-3.7-sonnet",
				IsEnabled:    true,
			},
		},
		Claude: claude,
		Codex:  codex,
	})

	output, err := processor.Process(context.Background(), "整理方案", map[string]string{
		execdomain.ExecutionEngineMetadataKey: execdomain.ExecutionEngineCodex,
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if output != "codex result" {
		t.Fatalf("Process() output = %q, want codex result", output)
	}
	if claude.called != 0 || codex.called != 1 {
		t.Fatalf("calls claude=%d codex=%d, want 0/1", claude.called, codex.called)
	}
}

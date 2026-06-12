package infrastructure

import (
	"context"
	"log/slog"
	"strings"

	execdomain "alter0/internal/execution/domain"
	llmdomain "alter0/internal/llm/domain"
)

type runtimeModelProviderSource interface {
	GetDefaultProvider(ctx context.Context) (*llmdomain.ModelProvider, error)
	GetProvider(ctx context.Context, providerID string) (*llmdomain.ModelProvider, error)
}

type RuntimeResolverOptions struct {
	ProviderSource runtimeModelProviderSource
	Claude         execdomain.AgentProcessor
	Codex          execdomain.AgentProcessor
	Logger         *slog.Logger
}

type RuntimeResolverProcessor struct {
	providers runtimeModelProviderSource
	claude    execdomain.AgentProcessor
	codex     execdomain.AgentProcessor
	logger    *slog.Logger
}

func NewRuntimeResolverProcessor(options RuntimeResolverOptions) *RuntimeResolverProcessor {
	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}
	codex := options.Codex
	if codex == nil {
		codex = NewCodexCLIProcessor()
	}
	return &RuntimeResolverProcessor{
		providers: options.ProviderSource,
		claude:    options.Claude,
		codex:     codex,
		logger:    logger,
	}
}

func (p *RuntimeResolverProcessor) Process(ctx context.Context, content string, metadata map[string]string) (string, error) {
	if p.shouldForceCodex(metadata) {
		return p.processCodex(ctx, content, metadata)
	}
	if provider := p.resolveClaudeProvider(ctx, metadata); provider != nil && p.claude != nil {
		p.applyProviderMetadata(metadata, *provider)
		setExecutionSource(metadata, execdomain.ExecutionSourceClaudeCode)
		return p.claude.Process(ctx, content, metadata)
	}
	return p.processCodex(ctx, content, metadata)
}

func (p *RuntimeResolverProcessor) ProcessStream(
	ctx context.Context,
	content string,
	metadata map[string]string,
	emit func(event execdomain.StreamEvent) error,
) (string, error) {
	if p.shouldForceCodex(metadata) {
		return p.processCodexStream(ctx, content, metadata, emit)
	}
	if provider := p.resolveClaudeProvider(ctx, metadata); provider != nil && p.claude != nil {
		p.applyProviderMetadata(metadata, *provider)
		setExecutionSource(metadata, execdomain.ExecutionSourceClaudeCode)
		return processRuntimeStream(ctx, p.claude, content, metadata, emit)
	}
	return p.processCodexStream(ctx, content, metadata, emit)
}

func (p *RuntimeResolverProcessor) shouldForceCodex(metadata map[string]string) bool {
	engine := strings.ToLower(strings.TrimSpace(metadataValue(metadata, execdomain.ExecutionEngineMetadataKey)))
	return engine == execdomain.ExecutionEngineCodex
}

func (p *RuntimeResolverProcessor) resolveClaudeProvider(ctx context.Context, metadata map[string]string) *llmdomain.ModelProvider {
	if p == nil || p.providers == nil {
		return nil
	}
	providerID := strings.TrimSpace(metadataValue(metadata, execdomain.LLMProviderIDMetadataKey))
	var (
		provider *llmdomain.ModelProvider
		err      error
	)
	if providerID != "" {
		provider, err = p.providers.GetProvider(ctx, providerID)
	} else {
		provider, err = p.providers.GetDefaultProvider(ctx)
	}
	if err != nil {
		if p.logger != nil {
			p.logger.Warn("failed to resolve model provider for runtime", slog.String("error", err.Error()))
		}
		return nil
	}
	if !isUsableClaudeProvider(provider) {
		return nil
	}
	return provider
}

func isUsableClaudeProvider(provider *llmdomain.ModelProvider) bool {
	if provider == nil {
		return false
	}
	return provider.IsEnabled &&
		strings.TrimSpace(provider.APIKey) != "" &&
		strings.TrimSpace(provider.DefaultModel) != ""
}

func (p *RuntimeResolverProcessor) applyProviderMetadata(metadata map[string]string, provider llmdomain.ModelProvider) {
	if metadata == nil {
		return
	}
	metadata[execdomain.LLMProviderIDMetadataKey] = strings.TrimSpace(provider.ID)
	metadata[execdomain.ClaudeAPIKeyMetadataKey] = strings.TrimSpace(provider.APIKey)
	metadata[execdomain.ClaudeBaseURLMetadataKey] = strings.TrimSpace(provider.BaseURL)
	if strings.TrimSpace(metadata[execdomain.LLMModelMetadataKey]) == "" {
		metadata[execdomain.LLMModelMetadataKey] = strings.TrimSpace(provider.DefaultModel)
	}
}

func (p *RuntimeResolverProcessor) processCodex(ctx context.Context, content string, metadata map[string]string) (string, error) {
	setExecutionSource(metadata, execdomain.ExecutionSourceCodexCLI)
	return p.codex.Process(ctx, content, metadata)
}

func (p *RuntimeResolverProcessor) processCodexStream(
	ctx context.Context,
	content string,
	metadata map[string]string,
	emit func(event execdomain.StreamEvent) error,
) (string, error) {
	setExecutionSource(metadata, execdomain.ExecutionSourceCodexCLI)
	return processRuntimeStream(ctx, p.codex, content, metadata, emit)
}

func processRuntimeStream(
	ctx context.Context,
	processor execdomain.AgentProcessor,
	content string,
	metadata map[string]string,
	emit func(event execdomain.StreamEvent) error,
) (string, error) {
	if stream, ok := processor.(interface {
		ProcessStream(context.Context, string, map[string]string, func(execdomain.StreamEvent) error) (string, error)
	}); ok {
		return stream.ProcessStream(ctx, content, metadata, emit)
	}
	output, err := processor.Process(ctx, content, metadata)
	if err != nil {
		return "", err
	}
	if emit != nil && strings.TrimSpace(output) != "" {
		if emitErr := emit(execdomain.StreamEvent{Type: execdomain.StreamEventTypeOutput, Text: output}); emitErr != nil {
			return "", emitErr
		}
	}
	return output, nil
}

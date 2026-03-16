package infrastructure

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	execdomain "alter0/internal/execution/domain"
	llmdomain "alter0/internal/llm/domain"
	taskapp "alter0/internal/task/application"
)

type reactAgentFactory interface {
	GetReActAgent(ctx context.Context, providerID string, config llmdomain.ReActAgentConfig) (*llmdomain.ReActAgent, error)
}

type HybridNLProcessor struct {
	codex  *CodexCLIProcessor
	react  reactAgentFactory
	logger *slog.Logger
}

func NewHybridNLProcessor(
	codex *CodexCLIProcessor,
	react reactAgentFactory,
	logger *slog.Logger,
) *HybridNLProcessor {
	if codex == nil {
		codex = NewCodexCLIProcessor()
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &HybridNLProcessor{
		codex:  codex,
		react:  react,
		logger: logger,
	}
}

func (p *HybridNLProcessor) Process(ctx context.Context, content string, metadata map[string]string) (string, error) {
	engine := p.resolveEngine(metadata)
	if engine == execdomain.ExecutionEngineReact {
		output, err := p.processWithReact(ctx, content, metadata, nil)
		if err == nil {
			return output, nil
		}
		p.logReactFallback(metadata, err)
	}
	return p.codex.Process(ctx, content, metadata)
}

func (p *HybridNLProcessor) ProcessStream(
	ctx context.Context,
	content string,
	metadata map[string]string,
	emit func(event execdomain.StreamEvent) error,
) (string, error) {
	engine := p.resolveEngine(metadata)
	if engine == execdomain.ExecutionEngineReact {
		output, err := p.processWithReact(ctx, content, metadata, emit)
		if err == nil {
			return output, nil
		}
		p.logReactFallback(metadata, err)
	}
	return p.codex.ProcessStream(ctx, content, metadata, emit)
}

func (p *HybridNLProcessor) resolveEngine(metadata map[string]string) string {
	override := strings.ToLower(strings.TrimSpace(metadataValue(metadata, execdomain.ExecutionEngineMetadataKey)))
	switch override {
	case execdomain.ExecutionEngineCodex:
		return execdomain.ExecutionEngineCodex
	case execdomain.ExecutionEngineReact:
		return execdomain.ExecutionEngineReact
	}
	if strings.EqualFold(strings.TrimSpace(metadataValue(metadata, taskapp.MetadataExecutionMode)), taskapp.ExecutionModeAsync) {
		return execdomain.ExecutionEngineCodex
	}
	return execdomain.ExecutionEngineReact
}

func (p *HybridNLProcessor) processWithReact(
	ctx context.Context,
	content string,
	metadata map[string]string,
	emit func(event execdomain.StreamEvent) error,
) (string, error) {
	if p.react == nil {
		return "", errors.New("react agent factory unavailable")
	}
	agent, err := p.react.GetReActAgent(ctx, "", llmdomain.ReActAgentConfig{
		SystemPrompt:  buildHybridReActSystemPrompt(metadata),
		MaxIterations: 1,
	})
	if err != nil {
		return "", err
	}
	if emit == nil {
		return agent.Run(ctx, content)
	}
	return agent.RunStream(ctx, content, func(event llmdomain.ReActEvent) error {
		if event.Type != "answer" || strings.TrimSpace(event.Delta) == "" {
			return nil
		}
		return emit(execdomain.StreamEvent{
			Type: execdomain.StreamEventTypeOutput,
			Text: event.Delta,
		})
	})
}

func (p *HybridNLProcessor) logReactFallback(metadata map[string]string, err error) {
	if p.logger == nil || err == nil {
		return
	}
	p.logger.Warn("react processor unavailable, falling back to codex",
		slog.String("session_id", strings.TrimSpace(metadataValue(metadata, execdomain.RuntimeSessionIDMetadataKey))),
		slog.String("message_id", strings.TrimSpace(metadataValue(metadata, execdomain.RuntimeMessageIDMetadataKey))),
		slog.String("error", strings.TrimSpace(err.Error())),
	)
}

func buildHybridReActSystemPrompt(metadata map[string]string) string {
	parts := []string{
		"You are alter0's execution assistant.",
		"Follow the ReAct pattern internally, but do not expose Thought, Action, or Observation headings.",
		"Return only the final user-facing answer.",
	}
	if rawSkillContext := strings.TrimSpace(metadataValue(metadata, execdomain.SkillContextMetadataKey)); rawSkillContext != "" {
		parts = append(parts, "Skill context (JSON): "+rawSkillContext)
	}
	if rawMCPContext := strings.TrimSpace(metadataValue(metadata, execdomain.MCPContextMetadataKey)); rawMCPContext != "" {
		parts = append(parts, "MCP context (JSON): "+rawMCPContext)
	}
	return strings.Join(parts, "\n\n")
}

var _ interface {
	Process(ctx context.Context, content string, metadata map[string]string) (string, error)
	ProcessStream(ctx context.Context, content string, metadata map[string]string, emit func(event execdomain.StreamEvent) error) (string, error)
} = (*HybridNLProcessor)(nil)

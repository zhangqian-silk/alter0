package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
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
	if engine == execdomain.ExecutionEngineAgent {
		output, err := p.processWithAgent(ctx, content, metadata, nil)
		if err == nil {
			return output, nil
		}
		p.logReactFallback(metadata, err)
		return p.codex.Process(ctx, content, metadata)
	}
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
	if engine == execdomain.ExecutionEngineAgent {
		output, err := p.processWithAgent(ctx, content, metadata, emit)
		if err == nil {
			return output, nil
		}
		p.logReactFallback(metadata, err)
		return p.codex.ProcessStream(ctx, content, metadata, emit)
	}
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
	case execdomain.ExecutionEngineAgent:
		return execdomain.ExecutionEngineAgent
	}
	if strings.TrimSpace(metadataValue(metadata, execdomain.AgentIDMetadataKey)) != "" {
		return execdomain.ExecutionEngineAgent
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
	providerID := strings.TrimSpace(metadataValue(metadata, execdomain.LLMProviderIDMetadataKey))
	modelID := strings.TrimSpace(metadataValue(metadata, execdomain.LLMModelMetadataKey))
	agent, err := p.react.GetReActAgent(ctx, providerID, llmdomain.ReActAgentConfig{
		Model:         modelID,
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

func (p *HybridNLProcessor) processWithAgent(
	ctx context.Context,
	content string,
	metadata map[string]string,
	emit func(event execdomain.StreamEvent) error,
) (string, error) {
	if p.react == nil {
		return "", errors.New("react agent factory unavailable")
	}
	tools := buildAgentTools(metadata)
	providerID := strings.TrimSpace(metadataValue(metadata, execdomain.LLMProviderIDMetadataKey))
	if providerID == "" {
		providerID = strings.TrimSpace(metadataValue(metadata, execdomain.AgentProviderIDMetadataKey))
	}
	modelID := strings.TrimSpace(metadataValue(metadata, execdomain.LLMModelMetadataKey))
	if modelID == "" {
		modelID = strings.TrimSpace(metadataValue(metadata, execdomain.AgentModelMetadataKey))
	}
	agent, err := p.react.GetReActAgent(ctx, providerID, llmdomain.ReActAgentConfig{
		Model:        modelID,
		SystemPrompt: buildAgentSystemPrompt(metadata),
		Tools:        tools,
		ToolExecutor: llmdomain.ToolExecutorFunc(func(runCtx context.Context, toolCall llmdomain.ToolCall) (*llmdomain.ToolResult, error) {
			return p.executeAgentTool(runCtx, metadata, toolCall)
		}),
		MaxIterations: parseAgentMaxIterations(metadata),
	})
	if err != nil {
		return "", err
	}
	state, err := agent.RunWithState(ctx, content, func(event llmdomain.ReActEvent) error {
		if emit == nil {
			return nil
		}
		text := agentEventText(event)
		if strings.TrimSpace(text) == "" {
			return nil
		}
		return emit(execdomain.StreamEvent{
			Type: execdomain.StreamEventTypeOutput,
			Text: text,
		})
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(state.Answer), nil
}

func (p *HybridNLProcessor) executeAgentTool(
	ctx context.Context,
	metadata map[string]string,
	toolCall llmdomain.ToolCall,
) (*llmdomain.ToolResult, error) {
	name := strings.ToLower(strings.TrimSpace(toolCall.Name))
	switch name {
	case "codex_exec":
		payload := struct {
			Instruction string `json:"instruction"`
		}{}
		if err := json.Unmarshal([]byte(strings.TrimSpace(toolCall.Arguments)), &payload); err != nil {
			return nil, fmt.Errorf("parse codex_exec arguments: %w", err)
		}
		instruction := strings.TrimSpace(payload.Instruction)
		if instruction == "" {
			return nil, errors.New("codex_exec instruction is required")
		}
		output, err := p.codex.Process(ctx, instruction, metadata)
		if err != nil {
			return &llmdomain.ToolResult{
				ToolCallID: toolCall.ID,
				Name:       toolCall.Name,
				Result:     "codex_exec failed: " + err.Error(),
				IsError:    true,
			}, nil
		}
		return &llmdomain.ToolResult{
			ToolCallID: toolCall.ID,
			Name:       toolCall.Name,
			Result:     output,
		}, nil
	case "complete":
		payload := struct {
			Result string `json:"result"`
		}{}
		if err := json.Unmarshal([]byte(strings.TrimSpace(toolCall.Arguments)), &payload); err != nil {
			return nil, fmt.Errorf("parse complete arguments: %w", err)
		}
		result := strings.TrimSpace(payload.Result)
		if result == "" {
			return nil, errors.New("complete result is required")
		}
		return &llmdomain.ToolResult{
			ToolCallID:  toolCall.ID,
			Name:        toolCall.Name,
			Result:      result,
			IsFinal:     true,
			FinalAnswer: result,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported agent tool: %s", toolCall.Name)
	}
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

func buildAgentSystemPrompt(metadata map[string]string) string {
	parts := []string{
		"You are alter0's agent execution mode.",
		"Your job is to complete the user's goal by actually executing work, not by only offering suggestions.",
		"Use codex_exec whenever concrete action is needed.",
		"Only call complete after the goal has been completed or when execution is truly blocked and you can clearly explain why.",
		"Do not expose hidden chain-of-thought. Keep outputs concise and execution-oriented.",
	}
	if custom := strings.TrimSpace(metadataValue(metadata, execdomain.AgentSystemPromptMetadataKey)); custom != "" {
		parts = append(parts, "Agent profile system prompt:\n"+custom)
	}
	if agentName := strings.TrimSpace(metadataValue(metadata, execdomain.AgentNameMetadataKey)); agentName != "" {
		parts = append(parts, "Current agent profile: "+agentName)
	}
	if rawSkillContext := strings.TrimSpace(metadataValue(metadata, execdomain.SkillContextMetadataKey)); rawSkillContext != "" {
		parts = append(parts, "Resolved skill context (JSON): "+rawSkillContext)
	}
	if rawMCPContext := strings.TrimSpace(metadataValue(metadata, execdomain.MCPContextMetadataKey)); rawMCPContext != "" {
		parts = append(parts, "Resolved MCP context (JSON): "+rawMCPContext)
	}
	return strings.Join(parts, "\n\n")
}

func buildAgentTools(metadata map[string]string) []llmdomain.Tool {
	allowed := parseAgentToolIDs(metadata)
	items := make([]llmdomain.Tool, 0, len(allowed)+1)
	if _, ok := allowed["codex_exec"]; ok {
		items = append(items, llmdomain.Tool{
			Name:        "codex_exec",
			Description: "Execute a concrete implementation step with Codex CLI and return the real execution result.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"instruction": map[string]interface{}{
						"type":        "string",
						"description": "A precise execution instruction for Codex CLI. Ask it to perform the next concrete step, not to discuss options.",
					},
				},
				"required": []string{"instruction"},
			},
		})
	}
	items = append(items, llmdomain.Tool{
		Name:        "complete",
		Description: "Finish the task and return the final user-facing result once the goal is actually complete or clearly blocked.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"result": map[string]interface{}{
					"type":        "string",
					"description": "The final concise user-facing result or the blocking reason.",
				},
			},
			"required": []string{"result"},
		},
	})
	return items
}

func parseAgentToolIDs(metadata map[string]string) map[string]struct{} {
	raw := strings.TrimSpace(metadataValue(metadata, execdomain.AgentToolsMetadataKey))
	items := []string{"codex_exec"}
	if raw != "" {
		if strings.HasPrefix(raw, "[") {
			var parsed []string
			if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
				items = parsed
			}
		} else {
			items = strings.Split(raw, ",")
		}
	}
	allowed := map[string]struct{}{}
	for _, item := range items {
		normalized := strings.ToLower(strings.TrimSpace(item))
		if normalized == "" {
			continue
		}
		allowed[normalized] = struct{}{}
	}
	if len(allowed) == 0 {
		allowed["codex_exec"] = struct{}{}
	}
	return allowed
}

func parseAgentMaxIterations(metadata map[string]string) int {
	raw := strings.TrimSpace(metadataValue(metadata, execdomain.AgentMaxIterationsMetadataKey))
	if raw == "" {
		return 6
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 6
	}
	if value > 20 {
		return 20
	}
	return value
}

func agentEventText(event llmdomain.ReActEvent) string {
	if event.State == nil {
		return ""
	}
	switch event.Type {
	case "action":
		if event.State.Action == nil {
			return ""
		}
		return "[agent] action: " + strings.TrimSpace(event.State.Action.Name) + "\n"
	case "observation":
		observation := strings.TrimSpace(event.State.Observation)
		if observation == "" {
			return ""
		}
		return "[agent] observation:\n" + observation + "\n"
	case "answer":
		answer := strings.TrimSpace(event.State.Answer)
		if answer == "" {
			return ""
		}
		return answer
	default:
		return ""
	}
}

var _ interface {
	Process(ctx context.Context, content string, metadata map[string]string) (string, error)
	ProcessStream(ctx context.Context, content string, metadata map[string]string, emit func(event execdomain.StreamEvent) error) (string, error)
} = (*HybridNLProcessor)(nil)

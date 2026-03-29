package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	agentapp "alter0/internal/agent/application"
	controldomain "alter0/internal/control/domain"
	execdomain "alter0/internal/execution/domain"
	llmdomain "alter0/internal/llm/domain"
	shareddomain "alter0/internal/shared/domain"
	taskapp "alter0/internal/task/application"
)

type reactAgentFactory interface {
	GetReActAgent(ctx context.Context, providerID string, config llmdomain.ReActAgentConfig) (*llmdomain.ReActAgent, error)
}

type runtimeAgentCatalog interface {
	ResolveAgent(id string) (controldomain.Agent, bool)
	ListDelegatableAgents(excludeID string) []controldomain.Agent
}

type HybridNLProcessor struct {
	codex  *CodexCLIProcessor
	react  reactAgentFactory
	agents runtimeAgentCatalog
	logger *slog.Logger
}

func NewHybridNLProcessor(
	codex *CodexCLIProcessor,
	react reactAgentFactory,
	logger *slog.Logger,
) *HybridNLProcessor {
	return NewHybridNLProcessorWithCatalog(codex, react, nil, logger)
}

func NewHybridNLProcessorWithCatalog(
	codex *CodexCLIProcessor,
	react reactAgentFactory,
	agents runtimeAgentCatalog,
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
		agents: agents,
		logger: logger,
	}
}

func (p *HybridNLProcessor) Process(ctx context.Context, content string, metadata map[string]string) (string, error) {
	engine := p.resolveEngine(metadata)
	if engine == execdomain.ExecutionEngineAgent {
		output, err := p.processWithAgent(ctx, content, metadata, nil)
		if err == nil {
			setExecutionSource(metadata, execdomain.ExecutionSourceModel)
			return output, nil
		}
		p.logReactFallback(metadata, err)
		output, codexErr := p.codex.Process(ctx, content, metadata)
		if codexErr == nil {
			setExecutionSource(metadata, execdomain.ExecutionSourceCodexCLI)
		}
		return output, codexErr
	}
	if engine == execdomain.ExecutionEngineReact {
		output, err := p.processWithReact(ctx, content, metadata, nil)
		if err == nil {
			setExecutionSource(metadata, execdomain.ExecutionSourceModel)
			return output, nil
		}
		p.logReactFallback(metadata, err)
	}
	output, err := p.codex.Process(ctx, content, metadata)
	if err == nil {
		setExecutionSource(metadata, execdomain.ExecutionSourceCodexCLI)
	}
	return output, err
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
			setExecutionSource(metadata, execdomain.ExecutionSourceModel)
			return output, nil
		}
		p.logReactFallback(metadata, err)
		output, codexErr := p.codex.ProcessStream(ctx, content, metadata, emit)
		if codexErr == nil {
			setExecutionSource(metadata, execdomain.ExecutionSourceCodexCLI)
		}
		return output, codexErr
	}
	if engine == execdomain.ExecutionEngineReact {
		output, err := p.processWithReact(ctx, content, metadata, emit)
		if err == nil {
			setExecutionSource(metadata, execdomain.ExecutionSourceModel)
			return output, nil
		}
		p.logReactFallback(metadata, err)
	}
	output, err := p.codex.ProcessStream(ctx, content, metadata, emit)
	if err == nil {
		setExecutionSource(metadata, execdomain.ExecutionSourceCodexCLI)
	}
	return output, err
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
	tools := buildNativeTools(metadata, nil)
	var toolExecutor llmdomain.ToolExecutor
	maxIterations := 1
	if len(tools) > 0 {
		toolExecutor = llmdomain.ToolExecutorFunc(func(runCtx context.Context, toolCall llmdomain.ToolCall) (*llmdomain.ToolResult, error) {
			return p.executeModelTool(runCtx, metadata, toolCall)
		})
		maxIterations = 6
	}
	providerID := strings.TrimSpace(metadataValue(metadata, execdomain.LLMProviderIDMetadataKey))
	modelID := strings.TrimSpace(metadataValue(metadata, execdomain.LLMModelMetadataKey))
	agent, err := p.react.GetReActAgent(ctx, providerID, llmdomain.ReActAgentConfig{
		Model:             modelID,
		SystemPrompt:      buildHybridReActSystemPrompt(metadata),
		Tools:             tools,
		ToolExecutor:      toolExecutor,
		MaxIterations:     maxIterations,
		UserMessagePuller: shareddomain.ConsumeLiveUserMessage,
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
	tools := p.buildAgentTools(metadata)
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
		SystemPrompt: p.buildAgentSystemPrompt(metadata),
		Tools:        tools,
		ToolExecutor: llmdomain.ToolExecutorFunc(func(runCtx context.Context, toolCall llmdomain.ToolCall) (*llmdomain.ToolResult, error) {
			return p.executeModelTool(runCtx, metadata, toolCall)
		}),
		MaxIterations:     parseAgentMaxIterations(metadata),
		UserMessagePuller: shareddomain.ConsumeLiveUserMessage,
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

func (p *HybridNLProcessor) executeModelTool(
	ctx context.Context,
	metadata map[string]string,
	toolCall llmdomain.ToolCall,
) (*llmdomain.ToolResult, error) {
	name := strings.ToLower(strings.TrimSpace(toolCall.Name))
	switch name {
	case toolListDir, toolRead, toolWrite, toolEdit, toolBash:
		executor, err := newNativeToolExecutor(metadata)
		if err != nil {
			return nil, err
		}
		return executor.Execute(ctx, toolCall)
	case toolReadSkill:
		return p.executeReadSkillTool(metadata, toolCall)
	case toolWriteSkill:
		return p.executeWriteSkillTool(metadata, toolCall)
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
	case toolDelegateAgent:
		return p.executeDelegatedAgent(ctx, metadata, toolCall)
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
	if len(buildNativeTools(metadata, nil)) > 0 {
		parts = append(parts, "Native tools are enabled. Relative tool paths use the repo root by default; set base=workspace to operate on the session workspace.")
	}
	if rawSkillContext := strings.TrimSpace(metadataValue(metadata, execdomain.SkillContextMetadataKey)); rawSkillContext != "" {
		parts = append(parts, "Skill context (JSON): "+rawSkillContext)
	}
	if rawMCPContext := strings.TrimSpace(metadataValue(metadata, execdomain.MCPContextMetadataKey)); rawMCPContext != "" {
		parts = append(parts, "MCP context (JSON): "+rawMCPContext)
	}
	if rawMemoryContext := strings.TrimSpace(metadataValue(metadata, execdomain.MemoryContextMetadataKey)); rawMemoryContext != "" {
		parts = append(parts, renderMemoryContextInstruction(rawMemoryContext))
	}
	if rawProductContext := strings.TrimSpace(metadataValue(metadata, execdomain.ProductContextMetadataKey)); rawProductContext != "" {
		parts = append(parts, renderProductContextInstruction(rawProductContext))
	}
	if rawProductDiscovery := strings.TrimSpace(metadataValue(metadata, execdomain.ProductDiscoveryMetadataKey)); rawProductDiscovery != "" {
		parts = append(parts, renderProductDiscoveryInstruction(rawProductDiscovery))
	}
	return strings.Join(parts, "\n\n")
}

func (p *HybridNLProcessor) buildAgentSystemPrompt(metadata map[string]string) string {
	allowedTools := parseSelectedToolIDs(metadata, defaultAgentToolIDs)
	parts := []string{
		"You are alter0's agent execution mode.",
		"Your job is to complete the user's goal by actually executing work, not by only offering suggestions.",
		"Prefer native tools for listing directories, reading files, writing files, editing files, and running shell commands.",
		"Use codex_exec when the task needs a deeper autonomous implementation run than a single native tool step.",
		"Only call complete after the goal has been completed or when execution is truly blocked and you can clearly explain why.",
		"Do not expose hidden chain-of-thought. Keep outputs concise and execution-oriented.",
		"Relative native-tool paths use the repo root by default; set base=workspace to operate on the session workspace.",
	}
	if _, ok := allowedTools[toolDelegateAgent]; ok {
		parts = append(parts, "Use delegate_agent when a specialist agent is better suited for a coding or writing subtask and you need that agent to return a concrete result.")
	}
	if _, ok := allowedTools[toolReadSkill]; ok {
		parts = append(parts, "Use read_skill to inspect file-backed reusable skills before applying page, rule, or preference changes.")
	}
	if _, ok := allowedTools[toolWriteSkill]; ok {
		parts = append(parts, "Use write_skill only when the user provides a durable, reusable preference that should update a file-backed skill instead of a one-off task artifact.")
	}
	if custom := strings.TrimSpace(metadataValue(metadata, execdomain.AgentSystemPromptMetadataKey)); custom != "" {
		parts = append(parts, "Agent profile system prompt:\n"+custom)
	}
	if agentName := strings.TrimSpace(metadataValue(metadata, execdomain.AgentNameMetadataKey)); agentName != "" {
		parts = append(parts, "Current agent profile: "+agentName)
	}
	if _, ok := allowedTools[toolDelegateAgent]; ok {
		if targets := p.renderDelegationTargets(metadata); targets != "" {
			parts = append(parts, targets)
		}
	}
	if rawSkillContext := strings.TrimSpace(metadataValue(metadata, execdomain.SkillContextMetadataKey)); rawSkillContext != "" {
		parts = append(parts, "Resolved skill context (JSON): "+rawSkillContext)
	}
	if rawMCPContext := strings.TrimSpace(metadataValue(metadata, execdomain.MCPContextMetadataKey)); rawMCPContext != "" {
		parts = append(parts, "Resolved MCP context (JSON): "+rawMCPContext)
	}
	if rawMemoryContext := strings.TrimSpace(metadataValue(metadata, execdomain.MemoryContextMetadataKey)); rawMemoryContext != "" {
		parts = append(parts, renderMemoryContextInstruction(rawMemoryContext))
		parts = append(parts, "When the user asks to remember, update, or persist durable guidance, use read/write/edit tools to modify the appropriate memory files directly.")
	}
	if rawProductContext := strings.TrimSpace(metadataValue(metadata, execdomain.ProductContextMetadataKey)); rawProductContext != "" {
		parts = append(parts, renderProductContextInstruction(rawProductContext))
		parts = append(parts, "When operating as a product master or worker agent, stay within the current product boundary unless the user explicitly asks for a cross-product comparison or coordination.")
	}
	if rawProductDiscovery := strings.TrimSpace(metadataValue(metadata, execdomain.ProductDiscoveryMetadataKey)); rawProductDiscovery != "" {
		parts = append(parts, renderProductDiscoveryInstruction(rawProductDiscovery))
	}
	return strings.Join(parts, "\n\n")
}

func renderMemoryContextInstruction(raw string) string {
	context := execdomain.MemoryContext{}
	if err := json.Unmarshal([]byte(raw), &context); err != nil || len(context.Files) == 0 {
		return "Resolved memory context (JSON): " + raw
	}

	var builder strings.Builder
	builder.WriteString("Resolved memory files:\n")
	for _, file := range context.Files {
		builder.WriteString("- ")
		builder.WriteString(strings.TrimSpace(file.Title))
		builder.WriteString("\n  path: ")
		builder.WriteString(strings.TrimSpace(file.Path))
		builder.WriteString("\n  exists: ")
		if file.Exists {
			builder.WriteString("true")
		} else {
			builder.WriteString("false")
		}
		if updatedAt := strings.TrimSpace(file.UpdatedAt); updatedAt != "" {
			builder.WriteString("\n  updated_at: ")
			builder.WriteString(updatedAt)
		}
		if content := strings.TrimSpace(file.Content); content != "" {
			builder.WriteString("\n  content:\n")
			builder.WriteString(content)
		}
		builder.WriteString("\n")
	}
	return strings.TrimSpace(builder.String())
}

func renderProductContextInstruction(raw string) string {
	context := execdomain.ProductContext{}
	if err := json.Unmarshal([]byte(raw), &context); err != nil || strings.TrimSpace(context.ProductID) == "" {
		return "Resolved product context (JSON): " + raw
	}

	var builder strings.Builder
	builder.WriteString("Resolved product context:\n")
	builder.WriteString("- product_id: ")
	builder.WriteString(strings.TrimSpace(context.ProductID))
	if name := strings.TrimSpace(context.Name); name != "" {
		builder.WriteString("\n- name: ")
		builder.WriteString(name)
	}
	if summary := strings.TrimSpace(context.Summary); summary != "" {
		builder.WriteString("\n- summary: ")
		builder.WriteString(summary)
	}
	if master := strings.TrimSpace(context.MasterAgentID); master != "" {
		builder.WriteString("\n- master_agent_id: ")
		builder.WriteString(master)
	}
	if entryRoute := strings.TrimSpace(context.EntryRoute); entryRoute != "" {
		builder.WriteString("\n- entry_route: ")
		builder.WriteString(entryRoute)
	}
	if len(context.ArtifactTypes) > 0 {
		builder.WriteString("\n- artifact_types: ")
		builder.WriteString(strings.Join(context.ArtifactTypes, ", "))
	}
	if len(context.KnowledgeSources) > 0 {
		builder.WriteString("\n- knowledge_sources: ")
		builder.WriteString(strings.Join(context.KnowledgeSources, ", "))
	}
	if len(context.WorkerAgents) > 0 {
		builder.WriteString("\n- worker_agents:")
		for _, worker := range context.WorkerAgents {
			builder.WriteString("\n  - ")
			builder.WriteString(strings.TrimSpace(worker.AgentID))
			if role := strings.TrimSpace(worker.Role); role != "" {
				builder.WriteString(" (")
				builder.WriteString(role)
				builder.WriteString(")")
			}
			if responsibility := strings.TrimSpace(worker.Responsibility); responsibility != "" {
				builder.WriteString(": ")
				builder.WriteString(responsibility)
			}
		}
	}
	return strings.TrimSpace(builder.String())
}

func renderProductDiscoveryInstruction(raw string) string {
	context := execdomain.ProductDiscoveryContext{}
	if err := json.Unmarshal([]byte(raw), &context); err != nil || len(context.MatchedProducts) == 0 {
		return "Resolved product discovery (JSON): " + raw
	}

	var builder strings.Builder
	builder.WriteString("Resolved product discovery:\n")
	if selected := strings.TrimSpace(context.SelectedProduct); selected != "" {
		builder.WriteString("- selected_product_id: ")
		builder.WriteString(selected)
		builder.WriteString("\n")
	}
	if reason := strings.TrimSpace(context.SelectionReason); reason != "" {
		builder.WriteString("- selection_reason: ")
		builder.WriteString(reason)
		builder.WriteString("\n")
	}
	if mode := strings.TrimSpace(context.ExecutionMode); mode != "" {
		builder.WriteString("- product_execution_mode: ")
		builder.WriteString(mode)
		builder.WriteString("\n")
	}
	builder.WriteString("- matched_products:")
	for _, item := range context.MatchedProducts {
		builder.WriteString("\n  - ")
		builder.WriteString(strings.TrimSpace(item.ProductID))
		if name := strings.TrimSpace(item.Name); name != "" {
			builder.WriteString(" (")
			builder.WriteString(name)
			builder.WriteString(")")
		}
		if summary := strings.TrimSpace(item.Summary); summary != "" {
			builder.WriteString(": ")
			builder.WriteString(summary)
		}
	}
	return strings.TrimSpace(builder.String())
}

func (p *HybridNLProcessor) buildAgentTools(metadata map[string]string) []llmdomain.Tool {
	allowed := parseSelectedToolIDs(metadata, defaultAgentToolIDs)
	items := make([]llmdomain.Tool, 0, len(allowed)+2)
	items = append(items, buildNativeTools(metadata, defaultAgentToolIDs)...)
	if _, ok := allowed[toolReadSkill]; ok {
		items = append(items, llmdomain.Tool{
			Name:        toolReadSkill,
			Description: "Read the file content behind a resolved file-backed skill.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"skill_id": map[string]interface{}{
						"type":        "string",
						"description": "Resolved skill ID to read, for example `travel-page`.",
					},
					"offset": map[string]interface{}{
						"type":        "integer",
						"description": "Byte offset to start reading from. Defaults to 0.",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum bytes to return. Defaults to 32768.",
					},
				},
				"required": []string{"skill_id"},
			},
		})
	}
	if _, ok := allowed[toolWriteSkill]; ok {
		items = append(items, llmdomain.Tool{
			Name:        toolWriteSkill,
			Description: "Write or append content to the file behind a resolved writable skill.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"skill_id": map[string]interface{}{
						"type":        "string",
						"description": "Resolved writable skill ID to update, for example `travel-page`.",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Content to write into the skill file.",
					},
					"mode": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"overwrite", "append"},
						"description": "Write mode. Defaults to overwrite.",
					},
				},
				"required": []string{"skill_id", "content"},
			},
		})
	}
	if _, ok := allowed[toolCodexExec]; ok {
		items = append(items, llmdomain.Tool{
			Name:        toolCodexExec,
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
	if _, ok := allowed[toolDelegateAgent]; ok {
		items = append(items, llmdomain.Tool{
			Name:        toolDelegateAgent,
			Description: p.delegateToolDescription(metadata),
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"agent_id": map[string]interface{}{
						"type":        "string",
						"description": "The specialist agent ID to delegate to.",
					},
					"task": map[string]interface{}{
						"type":        "string",
						"description": "The delegated subtask the specialist agent should complete.",
					},
					"context": map[string]interface{}{
						"type":        "string",
						"description": "Optional extra context or constraints for the delegated subtask.",
					},
				},
				"required": []string{"agent_id", "task"},
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

func (p *HybridNLProcessor) executeReadSkillTool(
	metadata map[string]string,
	toolCall llmdomain.ToolCall,
) (*llmdomain.ToolResult, error) {
	payload := struct {
		SkillID string `json:"skill_id"`
		Offset  int    `json:"offset"`
		Limit   int    `json:"limit"`
	}{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(toolCall.Arguments)), &payload); err != nil {
		return nil, fmt.Errorf("parse %s arguments: %w", toolReadSkill, err)
	}
	skill, executor, resolvedPath, err := resolveSkillToolTarget(metadata, payload.SkillID)
	if err != nil {
		return toolErrorResult(toolCall, err), nil
	}
	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return toolErrorResult(toolCall, err), nil
	}
	if payload.Offset < 0 {
		payload.Offset = 0
	}
	if payload.Offset > len(data) {
		payload.Offset = len(data)
	}
	limit := payload.Limit
	if limit <= 0 {
		limit = defaultReadLimitBytes
	}
	if limit > maxReadLimitBytes {
		limit = maxReadLimitBytes
	}
	end := payload.Offset + limit
	if end > len(data) {
		end = len(data)
	}
	content := string(data[payload.Offset:end])
	return toolSuccessResult(toolCall, map[string]any{
		"skill_id":        skill.ID,
		"skill_name":      skill.Name,
		"path":            executor.runtime.displayPath(resolvedPath),
		"size_bytes":      len(data),
		"offset":          payload.Offset,
		"returned_bytes":  len(content),
		"truncated":       end < len(data),
		"content":         content,
		"writable":        skill.Writable,
		"file_backed":     true,
		"resolved_skill":  true,
		"resolved_prompt": "skill file content",
	}), nil
}

func (p *HybridNLProcessor) executeWriteSkillTool(
	metadata map[string]string,
	toolCall llmdomain.ToolCall,
) (*llmdomain.ToolResult, error) {
	payload := struct {
		SkillID string `json:"skill_id"`
		Content string `json:"content"`
		Mode    string `json:"mode"`
	}{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(toolCall.Arguments)), &payload); err != nil {
		return nil, fmt.Errorf("parse %s arguments: %w", toolWriteSkill, err)
	}
	skill, executor, resolvedPath, err := resolveSkillToolTarget(metadata, payload.SkillID)
	if err != nil {
		return toolErrorResult(toolCall, err), nil
	}
	if !skill.Writable {
		return toolErrorResult(toolCall, errors.New("skill is read-only")), nil
	}
	if err := os.MkdirAll(filepath.Dir(resolvedPath), 0o755); err != nil {
		return toolErrorResult(toolCall, err), nil
	}
	mode := strings.ToLower(strings.TrimSpace(payload.Mode))
	if mode == "" {
		mode = "overwrite"
	}
	switch mode {
	case "overwrite":
		if err := os.WriteFile(resolvedPath, []byte(payload.Content), 0o644); err != nil {
			return toolErrorResult(toolCall, err), nil
		}
	case "append":
		file, err := os.OpenFile(resolvedPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return toolErrorResult(toolCall, err), nil
		}
		defer file.Close()
		if _, err := file.WriteString(payload.Content); err != nil {
			return toolErrorResult(toolCall, err), nil
		}
	default:
		return toolErrorResult(toolCall, fmt.Errorf("unsupported write mode: %s", payload.Mode)), nil
	}
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return toolErrorResult(toolCall, err), nil
	}
	return toolSuccessResult(toolCall, map[string]any{
		"skill_id":    skill.ID,
		"skill_name":  skill.Name,
		"path":        executor.runtime.displayPath(resolvedPath),
		"size_bytes":  info.Size(),
		"mode":        mode,
		"writable":    skill.Writable,
		"file_backed": true,
	}), nil
}

func resolveSkillToolTarget(
	metadata map[string]string,
	skillID string,
) (execdomain.SkillSpec, *nativeToolExecutor, string, error) {
	skill, err := findResolvedSkill(metadata, skillID)
	if err != nil {
		return execdomain.SkillSpec{}, nil, "", err
	}
	if strings.TrimSpace(skill.FilePath) == "" {
		return execdomain.SkillSpec{}, nil, "", errors.New("skill does not expose a file path")
	}
	executor, err := newNativeToolExecutor(metadata)
	if err != nil {
		return execdomain.SkillSpec{}, nil, "", err
	}
	resolvedPath, err := executor.runtime.resolvePath(skill.FilePath, toolBaseRepo)
	if err != nil {
		return execdomain.SkillSpec{}, nil, "", err
	}
	return skill, executor, resolvedPath, nil
}

func findResolvedSkill(metadata map[string]string, skillID string) (execdomain.SkillSpec, error) {
	targetID := strings.ToLower(strings.TrimSpace(skillID))
	if targetID == "" {
		return execdomain.SkillSpec{}, errors.New("skill_id is required")
	}
	rawSkillContext := strings.TrimSpace(metadataValue(metadata, execdomain.SkillContextMetadataKey))
	if rawSkillContext == "" {
		return execdomain.SkillSpec{}, errors.New("skill context is unavailable")
	}
	var skillContext execdomain.SkillContext
	if err := json.Unmarshal([]byte(rawSkillContext), &skillContext); err != nil {
		return execdomain.SkillSpec{}, fmt.Errorf("invalid skill context metadata: %w", err)
	}
	for _, skill := range skillContext.Skills {
		if strings.EqualFold(strings.TrimSpace(skill.ID), targetID) {
			return skill, nil
		}
	}
	return execdomain.SkillSpec{}, fmt.Errorf("skill %q is not available in the resolved skill context", skillID)
}

func (p *HybridNLProcessor) delegateToolDescription(metadata map[string]string) string {
	agents := p.delegationTargets(metadata)
	if len(agents) == 0 {
		return "Delegate a specialist subtask to another agent and receive its concrete result."
	}
	parts := make([]string, 0, len(agents))
	for _, item := range agents {
		parts = append(parts, item.ID)
	}
	return "Delegate a specialist subtask to another agent and receive its concrete result. Available agents: " + strings.Join(parts, ", ")
}

func (p *HybridNLProcessor) renderDelegationTargets(metadata map[string]string) string {
	agents := p.delegationTargets(metadata)
	if len(agents) == 0 {
		return ""
	}
	parts := make([]string, 0, len(agents))
	for _, item := range agents {
		label := item.ID
		if strings.TrimSpace(item.Description) != "" {
			label += ": " + strings.TrimSpace(item.Description)
		}
		parts = append(parts, "- "+label)
	}
	return "Available specialist agents:\n" + strings.Join(parts, "\n")
}

func (p *HybridNLProcessor) delegationTargets(metadata map[string]string) []controldomain.Agent {
	if p == nil || p.agents == nil {
		return nil
	}
	currentAgentID := strings.TrimSpace(metadataValue(metadata, execdomain.AgentIDMetadataKey))
	return p.agents.ListDelegatableAgents(currentAgentID)
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

func parseDelegationDepth(metadata map[string]string) int {
	raw := strings.TrimSpace(metadataValue(metadata, execdomain.AgentDelegationDepthMetadataKey))
	if raw == "" {
		return 0
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0
	}
	return value
}

func buildDelegatedTaskPrompt(task string, extraContext string) string {
	parts := []string{strings.TrimSpace(task)}
	if extra := strings.TrimSpace(extraContext); extra != "" {
		parts = append(parts, "Delegation context:\n"+extra)
	}
	return strings.Join(parts, "\n\n")
}

func (p *HybridNLProcessor) executeDelegatedAgent(
	ctx context.Context,
	metadata map[string]string,
	toolCall llmdomain.ToolCall,
) (*llmdomain.ToolResult, error) {
	if p == nil || p.agents == nil {
		return nil, errors.New("delegate_agent unavailable")
	}
	payload := struct {
		AgentID string `json:"agent_id"`
		Task    string `json:"task"`
		Context string `json:"context,omitempty"`
	}{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(toolCall.Arguments)), &payload); err != nil {
		return nil, fmt.Errorf("parse delegate_agent arguments: %w", err)
	}
	agentID := strings.TrimSpace(payload.AgentID)
	if agentID == "" {
		return nil, errors.New("delegate_agent agent_id is required")
	}
	task := strings.TrimSpace(payload.Task)
	if task == "" {
		return nil, errors.New("delegate_agent task is required")
	}
	depth := parseDelegationDepth(metadata)
	if depth >= 2 {
		return &llmdomain.ToolResult{
			ToolCallID: toolCall.ID,
			Name:       toolCall.Name,
			Result:     "delegate_agent blocked: maximum delegation depth reached",
			IsError:    true,
		}, nil
	}
	currentAgentID := strings.TrimSpace(metadataValue(metadata, execdomain.AgentIDMetadataKey))
	if strings.EqualFold(currentAgentID, agentID) {
		return &llmdomain.ToolResult{
			ToolCallID: toolCall.ID,
			Name:       toolCall.Name,
			Result:     "delegate_agent blocked: cannot delegate to the current agent",
			IsError:    true,
		}, nil
	}
	target, ok := p.agents.ResolveAgent(agentID)
	if !ok || !target.Enabled {
		return &llmdomain.ToolResult{
			ToolCallID: toolCall.ID,
			Name:       toolCall.Name,
			Result:     "delegate_agent failed: target agent not found or disabled",
			IsError:    true,
		}, nil
	}
	if !target.Delegatable {
		return &llmdomain.ToolResult{
			ToolCallID: toolCall.ID,
			Name:       toolCall.Name,
			Result:     "delegate_agent failed: target agent is not delegatable",
			IsError:    true,
		}, nil
	}
	childMetadata := agentapp.ApplyProfileMetadata(metadata, target)
	childMetadata[execdomain.AgentDelegatedByMetadataKey] = currentAgentID
	childMetadata[execdomain.AgentDelegationDepthMetadataKey] = strconv.Itoa(depth + 1)
	output, err := p.processWithAgent(ctx, buildDelegatedTaskPrompt(task, payload.Context), childMetadata, nil)
	if err != nil {
		return &llmdomain.ToolResult{
			ToolCallID: toolCall.ID,
			Name:       toolCall.Name,
			Result:     "delegate_agent failed: " + err.Error(),
			IsError:    true,
		}, nil
	}
	result := strings.TrimSpace(output)
	if result == "" {
		result = "delegated agent completed without a textual result"
	}
	return &llmdomain.ToolResult{
		ToolCallID: toolCall.ID,
		Name:       toolCall.Name,
		Result:     result,
	}, nil
}

func setExecutionSource(metadata map[string]string, source string) {
	if len(metadata) == 0 {
		return
	}
	metadata[execdomain.ExecutionSourceMetadataKey] = strings.TrimSpace(source)
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

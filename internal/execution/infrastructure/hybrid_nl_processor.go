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
	codex           *CodexCLIProcessor
	react           reactAgentFactory
	agents          runtimeAgentCatalog
	serviceDeployer workspaceServiceDeployer
	logger          *slog.Logger
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
		codex:           codex,
		react:           react,
		agents:          agents,
		serviceDeployer: newWorkspaceServiceDeployer(),
		logger:          logger,
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
	providerID := strings.TrimSpace(metadataValue(metadata, execdomain.LLMProviderIDMetadataKey))
	modelID := strings.TrimSpace(metadataValue(metadata, execdomain.LLMModelMetadataKey))
	agent, err := p.react.GetReActAgent(ctx, providerID, llmdomain.ReActAgentConfig{
		Model:             modelID,
		SystemPrompt:      buildHybridReActSystemPrompt(metadata),
		MaxIterations:     1,
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
	collector := newAgentProcessCollector()
	state, err := agent.RunWithState(ctx, content, func(event llmdomain.ReActEvent) error {
		processStep := collector.Consume(event)
		if emit != nil && processStep != nil {
			if err := emit(execdomain.StreamEvent{
				Type:        execdomain.StreamEventTypeProcess,
				ProcessStep: processStep,
			}); err != nil {
				return err
			}
		}
		if emit == nil {
			return nil
		}
		if event.Type != "answer" {
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
	collector.Store(metadata)
	return strings.TrimSpace(state.Answer), nil
}

func (p *HybridNLProcessor) executeModelTool(
	ctx context.Context,
	metadata map[string]string,
	toolCall llmdomain.ToolCall,
) (*llmdomain.ToolResult, error) {
	name := strings.ToLower(strings.TrimSpace(toolCall.Name))
	switch name {
	case toolSearchMemory:
		return p.executeSearchMemoryTool(metadata, toolCall)
	case toolReadMemory:
		return p.executeReadMemoryTool(metadata, toolCall)
	case toolWriteMemory:
		return p.executeWriteMemoryTool(metadata, toolCall)
	case toolDeployTestService:
		return p.executeDeployTestServiceTool(ctx, metadata, toolCall)
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
		codexMetadata := buildCodexExecMetadata(metadata)
		codexMetadata[execdomain.CodexRuntimeStrategyMetadataKey] = execdomain.CodexRuntimeStrategyPlain
		output, err := p.codex.Process(ctx, instruction, codexMetadata)
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
	allowedTools[toolCodexExec] = struct{}{}
	parts := []string{
		"You are alter0's session-aware execution assistant.",
		"Act like the user's ongoing assistant for this session: carry forward stable context, resolve shorthand from memory, and convert the user's intent into the next precise Codex CLI instruction.",
		"Codex CLI is the concrete executor for repository, file, shell, and product work. Use codex_exec for every concrete action.",
		"codex_exec already carries structured contexts for stable execution facts, including runtime metadata and any resolved product, skill, MCP, or memory payloads. Use the instruction body for the task-specific action instead of re-sending the full agent prompt.",
		"Use the resolved memory files, especially the agent session profile, to preserve session-level facts instead of repeatedly asking for context that is already known.",
		"Use any writable file-backed skill owned by the current agent as the durable rulebook for reusable working patterns, and update it when the user changes stable agent-specific preferences. Keep one-off task details out of that skill.",
		"Use search_memory, read_memory, and write_memory only for the resolved memory files when you need to inspect or persist durable user guidance.",
		"Only call complete after the goal has been completed or when execution is truly blocked and you can clearly explain why.",
		"Do not expose hidden chain-of-thought. Keep outputs concise and execution-oriented.",
	}
	if isCodingAgent(metadata) {
		parts = append(parts,
			"You are the dedicated coding assistant. You are responsible for understanding the user's engineering requirement, keeping the conversation coherent, and only ending after the requested development work is complete or a concrete blocker remains.",
			"Do not implement or verify changes yourself. Translate the user's requirement, repository conventions, and memory into precise codex_exec instructions so Codex performs the concrete development steps.",
			"After every codex_exec result, decide whether the requirement is satisfied. If it is not, issue the next concrete codex_exec instruction based on the observed result and continue the loop until the change, validation, required documentation updates, preview deployment success when needed, and any requested Git delivery steps are finished.",
			"Do not restate the full repository, workspace, and preview rulebook in every codex_exec call. Rely on the injected runtime_context for stable delivery facts and use the instruction text for the incremental action of this turn.",
			"Treat the session repository workspace as a full clone with its own .git metadata. Do not rely on git worktree semantics for coding delivery.",
			"Do not stop after the first Codex attempt when the task is still fixable. Refine the next instruction from the latest Codex output and keep driving execution.",
			"Keep the coding session profile current in your reasoning: source repository identity, dedicated repository workspace path, active branch, preview URL, and other stable delivery facts should be reused as session context rather than rediscovered from scratch.",
			"Treat repository context, branch readiness, dedicated workspace hygiene, test-page verification, preview deployment success, and PR handoff quality as part of the coding task. Do not treat them as optional follow-up details.",
			"Keep the test repository configuration aligned with the production service for model provider, Codex executable path, agent path, and other execution-critical settings. Only session cache or other session-scoped runtime data may differ.",
			"After preview deployment succeeds, ask the user whether they want to perform manual testing before continuing with GitHub delivery or final completion.",
			"When the task includes GitHub delivery, move directly from successful preview deployment to signed commit, push, gh PR creation, and gh merge without unnecessary pauses.",
			"Before completing, make sure the final result is explicit about code changes, validation status, documentation updates when user-visible behavior changed, preview deployment when needed, and PR or merge handoff details.",
		)
		if codingContext := renderCodingAgentExecutionContext(metadata); codingContext != "" {
			parts = append(parts, codingContext)
		}
	}
	if _, ok := allowedTools[toolDelegateAgent]; ok {
		parts = append(parts, "Use delegate_agent when a specialist agent is better suited for a coding or writing subtask and you need that agent to return a concrete result.")
	}
	if _, ok := allowedTools[toolSearchMemory]; ok {
		parts = append(parts, "Use search_memory first when you need to locate a past preference, shorthand, or historical note across the resolved memory files before opening a specific file.")
	}
	if _, ok := allowedTools[toolReadMemory]; ok {
		parts = append(parts, "Use read_memory to inspect the latest on-disk version of a resolved memory file when the injected snapshot is insufficient or may have changed.")
	}
	if _, ok := allowedTools[toolWriteMemory]; ok {
		parts = append(parts, "Use write_memory only for durable user guidance, preference updates, naming conventions, or shorthand mappings that belong in the resolved memory files.")
	}
	if _, ok := allowedTools[toolDeployTestService]; ok {
		parts = append(parts, "Use deploy_test_service when the user needs a session-scoped preview host or a separately routed test service on the shared gateway. Use web for the main short-hash host and a short service label such as api or docs for additional subdomains.")
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
		parts = append(parts, "When the user asks to remember, update, or persist durable guidance, use search_memory to locate the right file when needed, then use read_memory and write_memory on the appropriate resolved memory files.")
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

func isCodingAgent(metadata map[string]string) bool {
	return strings.EqualFold(strings.TrimSpace(metadataValue(metadata, execdomain.AgentIDMetadataKey)), "coding")
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
	if len(context.Recall) > 0 {
		builder.WriteString("\nAuto-recalled memory snippets:\n")
		for _, hit := range context.Recall {
			builder.WriteString("- ")
			if title := strings.TrimSpace(hit.Title); title != "" {
				builder.WriteString(title)
			} else {
				builder.WriteString(strings.TrimSpace(hit.MemoryID))
			}
			if hit.Line > 0 {
				builder.WriteString(fmt.Sprintf(":%d", hit.Line))
			}
			if path := strings.TrimSpace(hit.Path); path != "" {
				builder.WriteString("\n  path: ")
				builder.WriteString(path)
			}
			if snippet := strings.TrimSpace(hit.Snippet); snippet != "" {
				builder.WriteString("\n  snippet:\n")
				builder.WriteString(snippet)
			}
			builder.WriteString("\n")
		}
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
	allowed[toolCodexExec] = struct{}{}
	items := make([]llmdomain.Tool, 0, len(allowed)+1)
	if _, ok := allowed[toolSearchMemory]; ok {
		items = append(items, llmdomain.Tool{
			Name:        toolSearchMemory,
			Description: "Search the resolved memory files for a keyword or phrase and return matching files plus nearby context snippets.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Keyword or phrase to search for in the resolved memory files.",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of matches to return. Defaults to 20.",
					},
				},
				"required": []string{"query"},
			},
		})
	}
	if _, ok := allowed[toolReadMemory]; ok {
		items = append(items, llmdomain.Tool{
			Name:        toolReadMemory,
			Description: "Read one of the resolved memory files that was injected into the current agent context.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path of a resolved memory file from the current memory context.",
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
				"required": []string{"path"},
			},
		})
	}
	if _, ok := allowed[toolWriteMemory]; ok {
		items = append(items, llmdomain.Tool{
			Name:        toolWriteMemory,
			Description: "Write or append content to one of the resolved writable memory files from the current memory context.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path of a resolved writable memory file from the current memory context.",
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
				"required": []string{"path", "content"},
			},
		})
	}
	if _, ok := allowed[toolDeployTestService]; ok {
		items = append(items, llmdomain.Tool{
			Name:        toolDeployTestService,
			Description: "Deploy or update a workspace test service on the shared alter0 gateway and return the preview host.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"service_name": map[string]interface{}{
						"type":        "string",
						"description": "Service subdomain label. Use web for the main short-hash host or a short label such as api or docs for additional subdomains.",
					},
					"service_type": map[string]interface{}{
						"type":        "string",
						"enum":        []string{workspaceServiceTypeFrontendDist, workspaceServiceTypeHTTP},
						"description": "Deployment type. frontend_dist serves a built web dist, http proxies a local or remote upstream.",
					},
					"repository_path": map[string]interface{}{
						"type":        "string",
						"description": "Git workspace path that contains internal/interfaces/web/static/dist for frontend_dist deployments.",
					},
					"upstream_url": map[string]interface{}{
						"type":        "string",
						"description": "Existing http(s) upstream to register for http deployments.",
					},
					"start_command": map[string]interface{}{
						"type":        "string",
						"description": "Optional command to start an HTTP service before registration. The deployer injects PORT.",
					},
					"workdir": map[string]interface{}{
						"type":        "string",
						"description": "Working directory for start_command. Defaults to the session repository workspace.",
					},
					"port": map[string]interface{}{
						"type":        "integer",
						"description": "Optional fixed local port for start_command.",
					},
					"health_path": map[string]interface{}{
						"type":        "string",
						"description": "Optional health probe path such as /healthz.",
					},
					"skip_build": map[string]interface{}{
						"type":        "boolean",
						"description": "Skip npm run build for frontend_dist when the dist is already current.",
					},
				},
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

func (p *HybridNLProcessor) executeDeployTestServiceTool(
	ctx context.Context,
	metadata map[string]string,
	toolCall llmdomain.ToolCall,
) (*llmdomain.ToolResult, error) {
	if p == nil || p.serviceDeployer == nil {
		return toolErrorResult(toolCall, errors.New("workspace service deployer unavailable")), nil
	}

	payload := struct {
		ServiceName    string `json:"service_name"`
		ServiceType    string `json:"service_type"`
		RepositoryPath string `json:"repository_path"`
		UpstreamURL    string `json:"upstream_url"`
		StartCommand   string `json:"start_command"`
		Workdir        string `json:"workdir"`
		Port           int    `json:"port"`
		HealthPath     string `json:"health_path"`
		SkipBuild      bool   `json:"skip_build"`
	}{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(toolCall.Arguments)), &payload); err != nil {
		return nil, fmt.Errorf("parse %s arguments: %w", toolDeployTestService, err)
	}

	sessionID := strings.TrimSpace(metadataValue(metadata, execdomain.RuntimeSessionIDMetadataKey))
	if sessionID == "" {
		return toolErrorResult(toolCall, errors.New("runtime session id is required for deploy_test_service")), nil
	}

	result, err := p.serviceDeployer.Deploy(ctx, WorkspaceServiceDeployRequest{
		SessionID:      sessionID,
		ServiceID:      payload.ServiceName,
		ServiceType:    payload.ServiceType,
		RepositoryPath: payload.RepositoryPath,
		UpstreamURL:    payload.UpstreamURL,
		StartCommand:   payload.StartCommand,
		Workdir:        payload.Workdir,
		Port:           payload.Port,
		HealthPath:     payload.HealthPath,
		SkipBuild:      payload.SkipBuild,
	})
	if err != nil {
		return toolErrorResult(toolCall, err), nil
	}

	return toolSuccessResult(toolCall, map[string]any{
		"success":         true,
		"session_id":      result.SessionID,
		"service_id":      result.ServiceID,
		"service_type":    result.ServiceType,
		"host":            result.Host,
		"url":             result.URL,
		"short_hash":      result.ShortHash,
		"repository_path": result.RepositoryPath,
		"dist_path":       result.DistPath,
		"upstream_url":    result.UpstreamURL,
		"runtime_dir":     result.RuntimeDir,
		"log_path":        result.LogPath,
		"pid":             result.PID,
		"status":          result.Status,
	}), nil
}

func (p *HybridNLProcessor) executeReadMemoryTool(
	metadata map[string]string,
	toolCall llmdomain.ToolCall,
) (*llmdomain.ToolResult, error) {
	payload := struct {
		Path   string `json:"path"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
	}{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(toolCall.Arguments)), &payload); err != nil {
		return nil, fmt.Errorf("parse %s arguments: %w", toolReadMemory, err)
	}
	file, err := findResolvedMemoryFile(metadata, payload.Path)
	if err != nil {
		return toolErrorResult(toolCall, err), nil
	}
	resolvedPath := strings.TrimSpace(file.Path)
	info, statErr := os.Stat(resolvedPath)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return toolSuccessResult(toolCall, map[string]any{
				"path":      strings.TrimSpace(file.Path),
				"memory_id": strings.TrimSpace(file.ID),
				"title":     strings.TrimSpace(file.Title),
				"exists":    false,
				"content":   "",
			}), nil
		}
		return toolErrorResult(toolCall, statErr), nil
	}
	if info.IsDir() {
		return toolErrorResult(toolCall, errors.New("memory path is a directory")), nil
	}
	if !file.Exists {
		file.Exists = true
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
		"path":           strings.TrimSpace(file.Path),
		"memory_id":      strings.TrimSpace(file.ID),
		"title":          strings.TrimSpace(file.Title),
		"exists":         true,
		"size_bytes":     len(data),
		"offset":         payload.Offset,
		"returned_bytes": len(content),
		"truncated":      end < len(data),
		"content":        content,
	}), nil
}

func (p *HybridNLProcessor) executeSearchMemoryTool(
	metadata map[string]string,
	toolCall llmdomain.ToolCall,
) (*llmdomain.ToolResult, error) {
	payload := struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(toolCall.Arguments)), &payload); err != nil {
		return nil, fmt.Errorf("parse %s arguments: %w", toolSearchMemory, err)
	}
	query := strings.TrimSpace(payload.Query)
	if query == "" {
		return toolErrorResult(toolCall, errors.New("query is required")), nil
	}
	rawMemoryContext := strings.TrimSpace(metadataValue(metadata, execdomain.MemoryContextMetadataKey))
	if rawMemoryContext == "" {
		return toolErrorResult(toolCall, errors.New("memory context is unavailable")), nil
	}
	var memoryContext execdomain.MemoryContext
	if err := json.Unmarshal([]byte(rawMemoryContext), &memoryContext); err != nil {
		return toolErrorResult(toolCall, fmt.Errorf("invalid memory context metadata: %w", err)), nil
	}

	limit := payload.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	queryLower := strings.ToLower(query)
	type memoryMatch struct {
		Path     string `json:"path"`
		MemoryID string `json:"memory_id"`
		Title    string `json:"title"`
		Line     int    `json:"line"`
		Snippet  string `json:"snippet"`
	}
	matches := make([]memoryMatch, 0, limit)
	filesScanned := 0
	truncated := false
	for _, file := range memoryContext.Files {
		content, exists, err := loadResolvedMemoryContent(file)
		if err != nil || !exists || strings.TrimSpace(content) == "" {
			continue
		}
		filesScanned++
		lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
		for idx, line := range lines {
			if !strings.Contains(strings.ToLower(line), queryLower) {
				continue
			}
			if len(matches) >= limit {
				truncated = true
				break
			}
			matches = append(matches, memoryMatch{
				Path:     strings.TrimSpace(file.Path),
				MemoryID: strings.TrimSpace(file.ID),
				Title:    strings.TrimSpace(file.Title),
				Line:     idx + 1,
				Snippet:  buildMemorySearchSnippet(lines, idx),
			})
		}
		if truncated {
			break
		}
	}
	return toolSuccessResult(toolCall, map[string]any{
		"query":         query,
		"files_scanned": filesScanned,
		"match_count":   len(matches),
		"truncated":     truncated,
		"matches":       matches,
	}), nil
}

func (p *HybridNLProcessor) executeWriteMemoryTool(
	metadata map[string]string,
	toolCall llmdomain.ToolCall,
) (*llmdomain.ToolResult, error) {
	payload := struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		Mode    string `json:"mode"`
	}{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(toolCall.Arguments)), &payload); err != nil {
		return nil, fmt.Errorf("parse %s arguments: %w", toolWriteMemory, err)
	}
	file, err := findResolvedMemoryFile(metadata, payload.Path)
	if err != nil {
		return toolErrorResult(toolCall, err), nil
	}
	if !file.Writable {
		return toolErrorResult(toolCall, errors.New("memory file is read-only")), nil
	}
	resolvedPath := strings.TrimSpace(file.Path)
	if resolvedPath == "" {
		return toolErrorResult(toolCall, errors.New("memory file path is required")), nil
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
		handle, err := os.OpenFile(resolvedPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return toolErrorResult(toolCall, err), nil
		}
		defer handle.Close()
		if _, err := handle.WriteString(payload.Content); err != nil {
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
		"path":       resolvedPath,
		"memory_id":  strings.TrimSpace(file.ID),
		"title":      strings.TrimSpace(file.Title),
		"mode":       mode,
		"exists":     true,
		"size_bytes": info.Size(),
	}), nil
}

func findResolvedMemoryFile(metadata map[string]string, path string) (execdomain.MemoryFileSpec, error) {
	targetPath := normalizeResolvedFilePath(path)
	if targetPath == "" {
		return execdomain.MemoryFileSpec{}, errors.New("path is required")
	}
	rawMemoryContext := strings.TrimSpace(metadataValue(metadata, execdomain.MemoryContextMetadataKey))
	if rawMemoryContext == "" {
		return execdomain.MemoryFileSpec{}, errors.New("memory context is unavailable")
	}
	var memoryContext execdomain.MemoryContext
	if err := json.Unmarshal([]byte(rawMemoryContext), &memoryContext); err != nil {
		return execdomain.MemoryFileSpec{}, fmt.Errorf("invalid memory context metadata: %w", err)
	}
	for _, file := range memoryContext.Files {
		if normalizeResolvedFilePath(file.Path) == targetPath {
			return file, nil
		}
	}
	return execdomain.MemoryFileSpec{}, fmt.Errorf("memory file %q is not available in the resolved memory context", path)
}

func loadResolvedMemoryContent(file execdomain.MemoryFileSpec) (string, bool, error) {
	resolvedPath := strings.TrimSpace(file.Path)
	if resolvedPath == "" {
		return "", false, errors.New("memory file path is required")
	}
	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	return string(data), true, nil
}

func buildMemorySearchSnippet(lines []string, index int) string {
	if index < 0 || index >= len(lines) {
		return ""
	}
	start := index - 1
	if start < 0 {
		start = 0
	}
	end := index + 2
	if end > len(lines) {
		end = len(lines)
	}
	return strings.TrimSpace(strings.Join(lines[start:end], "\n"))
}

func normalizeResolvedFilePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	return filepath.Clean(trimmed)
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
		return 8
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 8
	}
	if value > 64 {
		return 64
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

type agentProcessCollector struct {
	pending *shareddomain.ProcessStep
	steps   []shareddomain.ProcessStep
	nextID  int
}

func newAgentProcessCollector() *agentProcessCollector {
	return &agentProcessCollector{
		steps: []shareddomain.ProcessStep{},
	}
}

func (c *agentProcessCollector) Consume(event llmdomain.ReActEvent) *shareddomain.ProcessStep {
	if c == nil || event.State == nil {
		return nil
	}
	switch event.Type {
	case "action":
		c.flush()
		if event.State.Action == nil {
			return nil
		}
		title := strings.TrimSpace(event.State.Action.Name)
		if strings.EqualFold(title, "complete") {
			return nil
		}
		c.pending = &shareddomain.ProcessStep{
			ID:     c.newID(),
			Kind:   "action",
			Title:  title,
			Status: "running",
		}
		step := *c.pending
		return &step
	case "observation":
		observation := strings.TrimSpace(event.State.Observation)
		if observation == "" {
			return nil
		}
		if c.pending == nil {
			c.pending = &shareddomain.ProcessStep{
				ID:     c.newID(),
				Kind:   "observation",
				Title:  "Observation",
				Status: "running",
			}
		}
		c.pending.Detail = observation
		if strings.HasPrefix(strings.ToLower(observation), "error:") {
			c.pending.Status = "failed"
		} else {
			c.pending.Status = "completed"
		}
		step := *c.pending
		c.flush()
		return &step
	case "answer":
		c.flush()
		return nil
	case "error":
		if c.pending != nil {
			c.pending.Status = "failed"
			step := *c.pending
			c.flush()
			return &step
		}
		return nil
	}
	return nil
}

func (c *agentProcessCollector) Store(metadata map[string]string) {
	if c == nil || metadata == nil {
		return
	}
	c.flush()
	if len(c.steps) == 0 {
		return
	}
	raw, err := json.Marshal(c.steps)
	if err != nil {
		return
	}
	metadata[execdomain.AgentProcessStepsMetadataKey] = string(raw)
}

func (c *agentProcessCollector) flush() {
	if c == nil || c.pending == nil {
		return
	}
	step := shareddomain.ProcessStep{
		ID:     strings.TrimSpace(c.pending.ID),
		Kind:   strings.TrimSpace(c.pending.Kind),
		Title:  strings.TrimSpace(c.pending.Title),
		Detail: strings.TrimSpace(c.pending.Detail),
		Status: strings.TrimSpace(c.pending.Status),
	}
	if step.Title != "" || step.Detail != "" {
		if step.Status == "" || step.Status == "running" {
			step.Status = "completed"
		}
		c.steps = append(c.steps, step)
	}
	c.pending = nil
}

func (c *agentProcessCollector) newID() string {
	if c == nil {
		return ""
	}
	c.nextID++
	return fmt.Sprintf("agent-step-%d", c.nextID)
}

var _ interface {
	Process(ctx context.Context, content string, metadata map[string]string) (string, error)
	ProcessStream(ctx context.Context, content string, metadata map[string]string, emit func(event execdomain.StreamEvent) error) (string, error)
} = (*HybridNLProcessor)(nil)

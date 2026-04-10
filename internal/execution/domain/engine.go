package domain

const (
	ExecutionEngineMetadataKey      = "alter0.execution.engine"
	ExecutionSourceMetadataKey      = "alter0.execution.source"
	LLMProviderIDMetadataKey        = "alter0.llm.provider_id"
	LLMModelMetadataKey             = "alter0.llm.model"
	AgentIDMetadataKey              = "alter0.agent.id"
	AgentNameMetadataKey            = "alter0.agent.name"
	AgentProviderIDMetadataKey      = "alter0.agent.provider_id"
	AgentModelMetadataKey           = "alter0.agent.model"
	AgentSystemPromptMetadataKey    = "alter0.agent.system_prompt"
	AgentMaxIterationsMetadataKey   = "alter0.agent.max_iterations"
	AgentToolsMetadataKey           = "alter0.agent.tools"
	AgentProcessStepsMetadataKey    = "alter0.agent.process_steps"
	AgentCapabilitiesMetadataKey    = "alter0.agent.capabilities"
	AgentDelegatedByMetadataKey     = "alter0.agent.delegated_by"
	AgentDelegationDepthMetadataKey = "alter0.agent.delegation_depth"

	ExecutionEngineAuto  = "auto"
	ExecutionEngineCodex = "codex"
	ExecutionEngineReact = "react"
	ExecutionEngineAgent = "agent"

	ExecutionSourceModel    = "model"
	ExecutionSourceCodexCLI = "codex_cli"
)

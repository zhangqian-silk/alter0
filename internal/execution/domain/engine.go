package domain

const (
	ExecutionEngineMetadataKey    = "alter0.execution.engine"
	LLMProviderIDMetadataKey      = "alter0.llm.provider_id"
	LLMModelMetadataKey           = "alter0.llm.model"
	AgentIDMetadataKey            = "alter0.agent.id"
	AgentNameMetadataKey          = "alter0.agent.name"
	AgentProviderIDMetadataKey    = "alter0.agent.provider_id"
	AgentModelMetadataKey         = "alter0.agent.model"
	AgentSystemPromptMetadataKey  = "alter0.agent.system_prompt"
	AgentMaxIterationsMetadataKey = "alter0.agent.max_iterations"
	AgentToolsMetadataKey         = "alter0.agent.tools"

	ExecutionEngineAuto  = "auto"
	ExecutionEngineCodex = "codex"
	ExecutionEngineReact = "react"
	ExecutionEngineAgent = "agent"
)

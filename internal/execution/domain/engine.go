package domain

const (
	ExecutionEngineMetadataKey      = "alter0.execution.engine"
	ExecutionSourceMetadataKey      = "alter0.execution.source"
	ProcessStepsMetadataKey         = "alter0.process_steps"
	LLMProviderIDMetadataKey        = "alter0.llm.provider_id"
	LLMModelMetadataKey             = "alter0.llm.model"
	DeliveryAttributesMetadataKey   = "alter0.delivery.attributes"
	DeliveryAttributeMetadataPrefix = "alter0.delivery.attr."
	DeliverablesMetadataKey         = "alter0.delivery.deliverables"
	CompletionChecksMetadataKey     = "alter0.delivery.completion_checks"
	CodexRuntimeStrategyMetadataKey = "alter0.codex.runtime_strategy"
	ClaudeAPIKeyMetadataKey         = "alter0.claude.api_key"
	ClaudeBaseURLMetadataKey        = "alter0.claude.base_url"
	ClaudeConfigDirMetadataKey      = "alter0.claude.config_dir"

	ExecutionEngineAuto   = "auto"
	ExecutionEngineCodex  = "codex"
	ExecutionEngineClaude = "claude"
	ExecutionEngineReact  = "react"

	ExecutionSourceModel      = "model"
	ExecutionSourceClaudeCode = "claude_code"
	ExecutionSourceCodexCLI   = "codex_cli"

	CodexRuntimeStrategyNative = "native"
	CodexRuntimeStrategyPlain  = "plain"
)

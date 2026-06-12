package application

import "testing"

func TestDefaultEnvironmentDefinitionsIncludeTerminalConfig(t *testing.T) {
	definitions := defaultEnvironmentDefinitions()
	index := map[string]bool{}
	for _, item := range definitions {
		index[item.Key] = true
	}

	for _, key := range []string{"task_terminal_shell", "async_task_trigger_threshold"} {
		if !index[key] {
			t.Fatalf("expected environment definition %q", key)
		}
	}

	for _, removedKey := range []string{
		"llm_default_provider",
		"llm_default_model",
		"session_memory_turns",
		"session_memory_ttl",
		"context_compression_threshold",
		"context_compression_summary_tokens",
		"context_compression_retain_turns",
	} {
		if index[removedKey] {
			t.Fatalf("did not expect environment definition %q", removedKey)
		}
	}
}

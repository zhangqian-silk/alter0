package application

import "testing"

func TestDefaultEnvironmentDefinitionsIncludeTerminalConfig(t *testing.T) {
	definitions := defaultEnvironmentDefinitions()
	index := map[string]bool{}
	for _, item := range definitions {
		index[item.Key] = true
	}

	for _, key := range []string{"task_terminal_max_sessions", "task_terminal_shell"} {
		if !index[key] {
			t.Fatalf("expected environment definition %q", key)
		}
	}
}

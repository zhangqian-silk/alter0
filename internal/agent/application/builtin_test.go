package application

import (
	"strings"
	"testing"
)

func TestBuiltinTravelAgentsUseAssistantCodexModel(t *testing.T) {
	agents := builtinAgents()
	index := map[string]string{}
	tools := map[string][]string{}
	for _, agent := range agents {
		index[agent.ID] = agent.SystemPrompt
		tools[agent.ID] = agent.Tools
	}

	if prompt := index["travel-master"]; !strings.Contains(prompt, "codex_exec") || !strings.Contains(strings.ToLower(prompt), "assistant") {
		t.Fatalf("expected travel-master prompt to describe assistant/codex execution, got %q", prompt)
	}
	if got := tools["travel-master"]; len(got) != 4 || got[0] != "codex_exec" {
		t.Fatalf("expected travel-master tools to start with codex_exec, got %+v", got)
	}
	for _, agentID := range []string{
		"travel-city-guide",
		"travel-route-planner",
		"travel-metro-guide",
		"travel-food-recommender",
		"travel-map-annotator",
	} {
		if _, ok := index[agentID]; ok {
			t.Fatalf("expected builtin travel worker agent %s to be removed", agentID)
		}
	}
}

package application

import (
	"strings"
	"testing"
)

func TestBuiltinTravelAgentsUseAssistantCodexModel(t *testing.T) {
	agents := builtinAgents()
	index := map[string]string{}
	names := map[string]string{}
	tools := map[string][]string{}
	skills := map[string][]string{}
	entrypoints := map[string]bool{}
	for _, agent := range agents {
		names[agent.ID] = agent.Name
		index[agent.ID] = agent.SystemPrompt
		tools[agent.ID] = agent.Tools
		skills[agent.ID] = agent.Skills
		entrypoints[agent.ID] = agent.EntryPoint
	}

	if prompt := index["travel"]; !strings.Contains(prompt, "codex_exec") || !strings.Contains(strings.ToLower(prompt), "assistant") {
		t.Fatalf("expected travel prompt to describe assistant/codex execution, got %q", prompt)
	}
	if got := names["travel"]; strings.Contains(strings.ToLower(got), "master") {
		t.Fatalf("expected travel display name without master, got %q", got)
	}
	if _, ok := index["product-builder"]; ok {
		t.Fatalf("expected product-builder builtin agent to be removed")
	}
	if got := tools["travel"]; len(got) != 5 || got[0] != "codex_exec" {
		t.Fatalf("expected travel tools to start with codex_exec, got %+v", got)
	}
	if got := strings.Join(tools["travel"], ","); !strings.Contains(got, "deploy_test_service") {
		t.Fatalf("expected travel tools to include deploy_test_service, got %+v", tools["travel"])
	}
	if got := strings.Join(skills["travel"], ","); !strings.Contains(got, "deploy-test-service") {
		t.Fatalf("expected travel skills to include deploy-test-service, got %+v", skills["travel"])
	}
	if prompt := index["travel"]; !strings.Contains(strings.ToLower(prompt), "html") || !strings.Contains(prompt, "alter0.cn") {
		t.Fatalf("expected travel prompt to require html guide and session subdomain deployment, got %q", prompt)
	}
	if !entrypoints["travel"] {
		t.Fatalf("expected travel to be available as a runtime entrypoint")
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
	if got := strings.Join(tools["coding"], ","); !strings.Contains(got, "deploy_test_service") {
		t.Fatalf("expected coding agent tools to include deploy_test_service, got %+v", tools["coding"])
	}
	if got := strings.Join(skills["coding"], ","); !strings.Contains(got, "deploy-test-service") {
		t.Fatalf("expected coding agent skills to include deploy-test-service, got %+v", skills["coding"])
	}
	if got := strings.Join(skills["coding"], ","); !strings.Contains(got, "frontend-design") {
		t.Fatalf("expected coding agent skills to include frontend-design, got %+v", skills["coding"])
	}
}

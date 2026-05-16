package application

import (
	"slices"
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
	deliverables := map[string]int{}
	completionChecks := map[string]int{}
	for _, agent := range agents {
		names[agent.ID] = agent.Name
		index[agent.ID] = agent.SystemPrompt
		tools[agent.ID] = agent.Tools
		skills[agent.ID] = agent.Skills
		entrypoints[agent.ID] = agent.EntryPoint
		deliverables[agent.ID] = len(agent.Deliverables)
		completionChecks[agent.ID] = len(agent.CompletionChecks)
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
	if got := strings.Join(skills["travel"], ","); !strings.Contains(got, "frontend-design") {
		t.Fatalf("expected travel skills to include frontend-design, got %+v", skills["travel"])
	}
	if got := strings.Join(skills["travel"], ","); !strings.Contains(got, "artifact-preview") {
		t.Fatalf("expected travel skills to include artifact-preview, got %+v", skills["travel"])
	}
	for _, skillID := range []string{
		"memory",
		"deploy-test-service",
		"frontend-design",
		"artifact-preview",
		"doc-coauthoring",
		"find-skills",
		"ui-ux-pro-max",
		"brainstorming",
	} {
		if !slices.Contains(skills["travel"], skillID) {
			t.Fatalf("expected travel skills to include %s, got %+v", skillID, skills["travel"])
		}
	}
	if prompt := index["travel"]; !strings.Contains(strings.ToLower(prompt), "html") || !strings.Contains(prompt, "alter0.cn") {
		t.Fatalf("expected travel prompt to require html guide and session subdomain deployment, got %q", prompt)
	}
	if prompt := strings.ToLower(index["travel"]); !strings.Contains(prompt, "desktop") || !strings.Contains(prompt, "mobile") {
		t.Fatalf("expected travel prompt to require desktop and mobile compatibility, got %q", index["travel"])
	}
	if prompt := strings.ToLower(index["travel"]); !strings.Contains(prompt, "before drafting the itinerary") || !strings.Contains(prompt, "data source") || !strings.Contains(prompt, "city-specific") {
		t.Fatalf("expected travel prompt to require recommendation listing before itinerary planning with flexible city-specific categories and data sources, got %q", index["travel"])
	}
	if !entrypoints["travel"] {
		t.Fatalf("expected travel to be available as a runtime entrypoint")
	}
	if deliverables["main"] != 0 {
		t.Fatalf("expected main agent to remain contract-light, got %d deliverables", deliverables["main"])
	}
	if deliverables["coding"] < 2 {
		t.Fatalf("expected coding agent to declare delivery contract, got %d deliverables", deliverables["coding"])
	}
	if deliverables["writing"] < 1 {
		t.Fatalf("expected writing agent to declare delivery contract, got %d deliverables", deliverables["writing"])
	}
	if deliverables["travel"] < 2 {
		t.Fatalf("expected travel agent to declare delivery contract, got %d deliverables", deliverables["travel"])
	}
	if completionChecks["travel"] < 2 {
		t.Fatalf("expected travel agent to declare artifact completion checks, got %d checks", completionChecks["travel"])
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
	for _, skillID := range []string{
		"memory",
		"deploy-test-service",
		"frontend-design",
		"artifact-preview",
		"doc-coauthoring",
		"fullstack-developer",
		"code-reviewer",
		"webapp-testing",
		"find-skills",
		"test-driven-development",
		"ui-ux-pro-max",
		"code-simplifier",
		"code-review",
		"brainstorming",
	} {
		if !slices.Contains(skills["coding"], skillID) {
			t.Fatalf("expected coding skills to include %s, got %+v", skillID, skills["coding"])
		}
	}
	for _, agentID := range []string{"main", "coding", "writing", "travel"} {
		prompt := index[agentID]
		if !strings.Contains(prompt, "current session workspace") {
			t.Fatalf("expected %s prompt to anchor work in the current session workspace, got %q", agentID, prompt)
		}
		if !strings.Contains(prompt, "Do not modify other sessions") {
			t.Fatalf("expected %s prompt to forbid cross-workspace/service edits, got %q", agentID, prompt)
		}
	}
}

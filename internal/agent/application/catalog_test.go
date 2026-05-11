package application

import (
	"slices"
	"testing"

	controldomain "alter0/internal/control/domain"
)

type stubManagedAgentSource struct {
	agents []controldomain.Agent
}

func (s stubManagedAgentSource) ResolveAgent(id string) (controldomain.Agent, bool) {
	for _, agent := range s.agents {
		if agent.ID == id {
			return agent, true
		}
	}
	return controldomain.Agent{}, false
}

func (s stubManagedAgentSource) ListAgents() []controldomain.Agent {
	return append([]controldomain.Agent(nil), s.agents...)
}

func TestCatalogNormalizesManagedAgentToolsWithDefaultSearchMemory(t *testing.T) {
	catalog := NewCatalog(stubManagedAgentSource{
		agents: []controldomain.Agent{
			{
				ID:      "research",
				Name:    "Research Agent",
				Enabled: true,
				Tools:   []string{"codex_exec"},
			},
		},
	})

	agent, ok := catalog.ResolveAgent("research")
	if !ok {
		t.Fatalf("expected managed agent to resolve")
	}
	if !slices.Contains(agent.Tools, "search_memory") {
		t.Fatalf("expected normalized tools to include search_memory, got %+v", agent.Tools)
	}
	if agent.Tools[0] != "codex_exec" {
		t.Fatalf("expected existing tool order to stay stable, got %+v", agent.Tools)
	}
}

func TestCatalogDoesNotDuplicateDefaultSearchMemoryTool(t *testing.T) {
	agent := normalizeRuntimeAgent(controldomain.Agent{
		ID:    "writer",
		Name:  "Writer",
		Tools: []string{"codex_exec", "search_memory", "read_memory"},
	})

	count := 0
	for _, tool := range agent.Tools {
		if tool == "search_memory" {
			count += 1
		}
	}
	if count != 1 {
		t.Fatalf("expected search_memory only once, got %+v", agent.Tools)
	}
}

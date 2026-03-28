package domain

import "testing"

func TestAgentCapabilityRoundTripPreservesMemoryFiles(t *testing.T) {
	agent := Agent{
		ID:          "researcher",
		Name:        "Researcher",
		Type:        CapabilityTypeAgent,
		Enabled:     true,
		Scope:       CapabilityScopeGlobal,
		Version:     "v1",
		Tools:       []string{"read", "write"},
		Skills:      []string{"summary"},
		MCPs:        []string{"github"},
		MemoryFiles: []string{"user_md", "soul_md", "memory_long_term"},
		Metadata: map[string]string{
			"custom.key": "custom-value",
		},
	}

	capability := agent.AsCapability()
	decoded := AgentFromCapability(capability)

	if len(decoded.MemoryFiles) != 3 {
		t.Fatalf("expected 3 memory files, got %+v", decoded.MemoryFiles)
	}
	if decoded.MemoryFiles[0] != "user_md" || decoded.MemoryFiles[1] != "soul_md" || decoded.MemoryFiles[2] != "memory_long_term" {
		t.Fatalf("unexpected memory files: %+v", decoded.MemoryFiles)
	}
	if decoded.Metadata["custom.key"] != "custom-value" {
		t.Fatalf("expected custom metadata preserved, got %+v", decoded.Metadata)
	}
}

func TestAgentCapabilityRoundTripPreservesRuntimeCatalogFields(t *testing.T) {
	agent := Agent{
		ID:           "main",
		Name:         "Alter0",
		Type:         CapabilityTypeAgent,
		Enabled:      true,
		Scope:        CapabilityScopeGlobal,
		Version:      DefaultCapabilityVersion,
		Source:       AgentSourceBuiltin,
		Kind:         AgentKindMain,
		Description:  "Primary orchestrator",
		EntryPoint:   true,
		Delegatable:  true,
		UIRoute:      "chat",
		Capabilities: []string{"general", "orchestration"},
	}

	decoded := AgentFromCapability(agent.AsCapability())

	if decoded.Source != AgentSourceBuiltin || decoded.Kind != AgentKindMain {
		t.Fatalf("expected builtin/main runtime fields, got %+v", decoded)
	}
	if !decoded.EntryPoint || !decoded.Delegatable {
		t.Fatalf("expected entrypoint and delegatable preserved, got %+v", decoded)
	}
	if decoded.UIRoute != "chat" {
		t.Fatalf("expected ui route chat, got %q", decoded.UIRoute)
	}
	if len(decoded.Capabilities) != 2 || decoded.Capabilities[0] != "general" {
		t.Fatalf("unexpected capabilities: %+v", decoded.Capabilities)
	}
}

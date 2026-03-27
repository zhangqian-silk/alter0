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

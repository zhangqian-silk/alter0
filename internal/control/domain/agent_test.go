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

func TestAgentCapabilityRoundTripPreservesSessionProfileFields(t *testing.T) {
	agent := Agent{
		ID:      "coding",
		Name:    "Coding Agent",
		Type:    CapabilityTypeAgent,
		Enabled: true,
		Scope:   CapabilityScopeGlobal,
		SessionProfileFields: []AgentSessionProfileField{
			{
				Key:         "repository_path",
				Label:       "Repository",
				Description: "Dedicated repository workspace path",
				ReadOnly:    true,
			},
			{
				Key:      "branch",
				Label:    "Branch",
				ReadOnly: true,
			},
		},
	}

	decoded := AgentFromCapability(agent.AsCapability())

	if len(decoded.SessionProfileFields) != 2 {
		t.Fatalf("expected 2 session profile fields, got %+v", decoded.SessionProfileFields)
	}
	if decoded.SessionProfileFields[0].Key != "repository_path" || decoded.SessionProfileFields[0].Label != "Repository" {
		t.Fatalf("unexpected first field: %+v", decoded.SessionProfileFields[0])
	}
	if decoded.SessionProfileFields[0].Description != "Dedicated repository workspace path" || !decoded.SessionProfileFields[0].ReadOnly {
		t.Fatalf("unexpected first field metadata: %+v", decoded.SessionProfileFields[0])
	}
	if decoded.SessionProfileFields[1].Key != "branch" || decoded.SessionProfileFields[1].Label != "Branch" || !decoded.SessionProfileFields[1].ReadOnly {
		t.Fatalf("unexpected second field: %+v", decoded.SessionProfileFields[1])
	}
}

func TestAgentCapabilityRoundTripPreservesDeliverables(t *testing.T) {
	agent := Agent{
		ID:      "travel",
		Name:    "Travel Agent",
		Type:    CapabilityTypeAgent,
		Enabled: true,
		Scope:   CapabilityScopeGlobal,
		Deliverables: []AgentDeliverable{
			{
				ID:                  "guide-markdown",
				Label:               "Travel Guide",
				Description:         "Structured city guide aligned with the current trip request.",
				Format:              "markdown",
				Required:            true,
				SessionAttributeKey: "",
			},
			{
				ID:                  "guide-html",
				Label:               "HTML Guide",
				Description:         "Published HTML travel guide for the current session.",
				Format:              "html",
				Required:            true,
				SessionAttributeKey: "guide_html_url",
			},
		},
	}

	decoded := AgentFromCapability(agent.AsCapability())

	if len(decoded.Deliverables) != 2 {
		t.Fatalf("expected 2 deliverables, got %+v", decoded.Deliverables)
	}
	if decoded.Deliverables[0].ID != "guide-markdown" || decoded.Deliverables[0].Label != "Travel Guide" {
		t.Fatalf("unexpected first deliverable: %+v", decoded.Deliverables[0])
	}
	if !decoded.Deliverables[0].Required || decoded.Deliverables[0].Format != "markdown" {
		t.Fatalf("unexpected first deliverable metadata: %+v", decoded.Deliverables[0])
	}
	if decoded.Deliverables[1].SessionAttributeKey != "guide_html_url" {
		t.Fatalf("unexpected second deliverable attribute binding: %+v", decoded.Deliverables[1])
	}
}

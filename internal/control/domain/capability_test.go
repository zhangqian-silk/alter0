package domain

import "testing"

func TestCapabilityValidateAndDefaults(t *testing.T) {
	capability := Capability{
		ID:      "summary",
		Name:    "Summary",
		Type:    CapabilityTypeSkill,
		Enabled: true,
	}

	if err := capability.Validate(); err != nil {
		t.Fatalf("expected capability valid, got %v", err)
	}

	normalized := capability.Normalized()
	if normalized.Scope != CapabilityScopeGlobal {
		t.Fatalf("expected default scope global, got %s", normalized.Scope)
	}
	if normalized.Version != DefaultCapabilityVersion {
		t.Fatalf("expected default version %s, got %s", DefaultCapabilityVersion, normalized.Version)
	}
}

func TestCapabilityValidateVersionRule(t *testing.T) {
	capability := Capability{
		ID:      "summary",
		Name:    "Summary",
		Type:    CapabilityTypeSkill,
		Enabled: true,
		Scope:   CapabilityScopeGlobal,
		Version: "1.0.0",
	}

	if err := capability.Validate(); err == nil {
		t.Fatal("expected invalid version error")
	}
}

func TestSkillValidateUsesUnifiedModel(t *testing.T) {
	skill := Skill{
		ID:      "default-nl",
		Name:    "Default NL",
		Enabled: true,
	}
	if err := skill.Validate(); err != nil {
		t.Fatalf("expected skill valid under unified model, got %v", err)
	}

	capability := skill.AsCapability().Normalized()
	if capability.Type != CapabilityTypeSkill {
		t.Fatalf("expected skill type, got %s", capability.Type)
	}
	if capability.Scope != CapabilityScopeGlobal {
		t.Fatalf("expected default scope global, got %s", capability.Scope)
	}
}

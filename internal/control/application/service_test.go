package application

import (
	"testing"

	controldomain "alter0/internal/control/domain"
	shareddomain "alter0/internal/shared/domain"
)

func TestChannelCRUD(t *testing.T) {
	service := NewService()

	err := service.UpsertChannel(controldomain.Channel{
		ID:      "web-default",
		Type:    shareddomain.ChannelTypeWeb,
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	channel, ok := service.ResolveChannel("web-default")
	if !ok {
		t.Fatalf("expected channel exists")
	}
	if channel.Type != shareddomain.ChannelTypeWeb {
		t.Fatalf("expected web type, got %s", channel.Type)
	}

	if !service.DeleteChannel("web-default") {
		t.Fatalf("expected delete success")
	}
}

func TestCapabilityLifecycleAndAudit(t *testing.T) {
	service := NewService()

	err := service.UpsertSkill(controldomain.Skill{
		ID:      "summary",
		Name:    "Summary",
		Enabled: true,
		Scope:   controldomain.CapabilityScopeGlobal,
		Version: "v1.1.0",
		Metadata: map[string]string{
			"owner": "runtime",
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	skill, ok := service.ResolveSkill("summary")
	if !ok {
		t.Fatalf("expected skill exists")
	}
	if skill.Type != controldomain.CapabilityTypeSkill {
		t.Fatalf("expected skill type, got %s", skill.Type)
	}
	if skill.Scope != controldomain.CapabilityScopeGlobal {
		t.Fatalf("expected global scope, got %s", skill.Scope)
	}

	if _, err := service.SetCapabilityEnabled(controldomain.CapabilityTypeSkill, "summary", false); err != nil {
		t.Fatalf("expected disable success, got %v", err)
	}
	if _, err := service.SetCapabilityEnabled(controldomain.CapabilityTypeSkill, "summary", true); err != nil {
		t.Fatalf("expected enable success, got %v", err)
	}
	if !service.DeleteSkill("summary") {
		t.Fatalf("expected delete success")
	}

	audits := service.ListCapabilityAudits()
	if len(audits) != 4 {
		t.Fatalf("expected 4 audit entries, got %d", len(audits))
	}
	if audits[0].Action != controldomain.CapabilityLifecycleUpdate {
		t.Fatalf("expected first action update, got %s", audits[0].Action)
	}
	if audits[1].Action != controldomain.CapabilityLifecycleDisable {
		t.Fatalf("expected second action disable, got %s", audits[1].Action)
	}
	if audits[2].Action != controldomain.CapabilityLifecycleEnable {
		t.Fatalf("expected third action enable, got %s", audits[2].Action)
	}
	if audits[3].Action != controldomain.CapabilityLifecycleDelete {
		t.Fatalf("expected fourth action delete, got %s", audits[3].Action)
	}
}

func TestUnifiedSkillAndMCPStorage(t *testing.T) {
	service := NewService()

	if err := service.UpsertCapability(controldomain.Capability{
		ID:      "summary",
		Name:    "Summary",
		Type:    controldomain.CapabilityTypeSkill,
		Enabled: true,
		Scope:   controldomain.CapabilityScopeGlobal,
		Version: "v1.0.0",
	}); err != nil {
		t.Fatalf("unexpected skill upsert error: %v", err)
	}
	if err := service.UpsertCapability(controldomain.Capability{
		ID:      "github-mcp",
		Name:    "GitHub MCP",
		Type:    controldomain.CapabilityTypeMCP,
		Enabled: true,
		Scope:   controldomain.CapabilityScopeSession,
		Version: "v2.0.0",
	}); err != nil {
		t.Fatalf("unexpected mcp upsert error: %v", err)
	}

	capabilities := service.ListCapabilities()
	if len(capabilities) != 2 {
		t.Fatalf("expected 2 capabilities, got %d", len(capabilities))
	}

	skills := service.ListSkills()
	if len(skills) != 1 || skills[0].Type != controldomain.CapabilityTypeSkill {
		t.Fatalf("unexpected skills list: %+v", skills)
	}

	mcps := service.ListMCPs()
	if len(mcps) != 1 || mcps[0].Type != controldomain.CapabilityTypeMCP {
		t.Fatalf("unexpected mcp list: %+v", mcps)
	}
}

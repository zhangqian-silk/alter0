package application

import (
	"context"
	"encoding/json"
	"testing"

	controldomain "alter0/internal/control/domain"
	execdomain "alter0/internal/execution/domain"
	shareddomain "alter0/internal/shared/domain"
)

type stubProcessor struct {
	output       string
	lastContent  string
	lastMetadata map[string]string
}

func (s *stubProcessor) Process(_ context.Context, content string, metadata map[string]string) (string, error) {
	s.lastContent = content
	s.lastMetadata = map[string]string{}
	for key, value := range metadata {
		s.lastMetadata[key] = value
	}
	return s.output, nil
}

type stubSkillSource struct {
	items []controldomain.Capability
}

func (s *stubSkillSource) ListCapabilitiesByType(capabilityType controldomain.CapabilityType) []controldomain.Capability {
	if capabilityType != controldomain.CapabilityTypeSkill {
		return nil
	}
	out := make([]controldomain.Capability, 0, len(s.items))
	for _, item := range s.items {
		out = append(out, item)
	}
	return out
}

func TestExecuteNaturalLanguageInjectsEnabledSkillsByPriority(t *testing.T) {
	processor := &stubProcessor{output: "ok"}
	source := &stubSkillSource{items: []controldomain.Capability{
		{
			ID:      "summary",
			Name:    "Summary",
			Type:    controldomain.CapabilityTypeSkill,
			Enabled: true,
			Metadata: map[string]string{
				skillPriorityKey:          "200",
				skillDescriptionKey:       "summary documents",
				skillParameterTemplateKey: `{"lang":"zh-CN"}`,
				skillConstraintsKey:       "max:300, keep-tone",
			},
		},
		{
			ID:      "rewrite",
			Name:    "Rewrite",
			Type:    controldomain.CapabilityTypeSkill,
			Enabled: true,
			Metadata: map[string]string{
				skillPriorityKey: "100",
			},
		},
		{
			ID:      "disabled",
			Name:    "Disabled",
			Type:    controldomain.CapabilityTypeSkill,
			Enabled: false,
			Metadata: map[string]string{
				skillPriorityKey: "999",
			},
		},
	}}
	service := NewServiceWithSkills(processor, source, nil)

	result, err := service.ExecuteNaturalLanguage(context.Background(), shareddomain.UnifiedMessage{
		MessageID:   "m1",
		SessionID:   "s1",
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     "write release note",
		TraceID:     "t1",
	})
	if err != nil {
		t.Fatalf("ExecuteNaturalLanguage() error = %v", err)
	}
	if result.Output != "ok" {
		t.Fatalf("ExecuteNaturalLanguage() output = %q, want %q", result.Output, "ok")
	}
	if got := result.Metadata[resultSkillInjectedKey]; got != "2" {
		t.Fatalf("skills injected count = %q, want 2", got)
	}
	if got := result.Metadata[resultSkillInjectedIDsKey]; got != "summary,rewrite" {
		t.Fatalf("skills injected ids = %q, want summary,rewrite", got)
	}
	if got := result.Metadata[resultSkillProtocolKey]; got != execdomain.SkillContextProtocolVersion {
		t.Fatalf("skills protocol = %q, want %q", got, execdomain.SkillContextProtocolVersion)
	}

	rawSkillContext := processor.lastMetadata[execdomain.SkillContextMetadataKey]
	if rawSkillContext == "" {
		t.Fatalf("missing %s metadata", execdomain.SkillContextMetadataKey)
	}
	var skillContext execdomain.SkillContext
	if err := json.Unmarshal([]byte(rawSkillContext), &skillContext); err != nil {
		t.Fatalf("unmarshal skill context: %v", err)
	}
	if len(skillContext.Skills) != 2 {
		t.Fatalf("skill context size = %d, want 2", len(skillContext.Skills))
	}
	if skillContext.Skills[0].ID != "summary" || skillContext.Skills[1].ID != "rewrite" {
		t.Fatalf("unexpected skill order: %+v", skillContext.Skills)
	}
	if skillContext.Skills[0].Description != "summary documents" {
		t.Fatalf("skill description = %q, want summary documents", skillContext.Skills[0].Description)
	}
	if got := skillContext.Skills[0].ParameterTemplate["lang"]; got != "zh-CN" {
		t.Fatalf("unexpected parameter template lang = %q", got)
	}
	if len(skillContext.Skills[0].Constraints) != 2 {
		t.Fatalf("constraints size = %d, want 2", len(skillContext.Skills[0].Constraints))
	}
}

func TestExecuteNaturalLanguageRespectsIncludeExcludeSelection(t *testing.T) {
	processor := &stubProcessor{output: "ok"}
	source := &stubSkillSource{items: []controldomain.Capability{
		{ID: "summary", Name: "Summary", Type: controldomain.CapabilityTypeSkill, Enabled: true},
		{ID: "rewrite", Name: "Rewrite", Type: controldomain.CapabilityTypeSkill, Enabled: true},
		{ID: "proofread", Name: "Proofread", Type: controldomain.CapabilityTypeSkill, Enabled: true},
	}}
	service := NewServiceWithSkills(processor, source, nil)

	result, err := service.ExecuteNaturalLanguage(context.Background(), shareddomain.UnifiedMessage{
		MessageID:   "m1",
		SessionID:   "s1",
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     "refine output",
		TraceID:     "t1",
		Metadata: map[string]string{
			skillIncludeFilterKey: "summary,rewrite",
			skillExcludeFilterKey: "rewrite",
		},
	})
	if err != nil {
		t.Fatalf("ExecuteNaturalLanguage() error = %v", err)
	}
	if got := result.Metadata[resultSkillInjectedIDsKey]; got != "summary" {
		t.Fatalf("skills injected ids = %q, want summary", got)
	}
	if got := result.Metadata[resultSkillInjectedKey]; got != "1" {
		t.Fatalf("skills injected count = %q, want 1", got)
	}
}

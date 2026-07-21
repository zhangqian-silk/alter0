package main

import (
	"strings"
	"testing"

	codexapp "alter0/internal/codex/application"
	controlapp "alter0/internal/control/application"
	controldomain "alter0/internal/control/domain"
)

func TestRegisterBuiltinSkillsSeedsOnlyNativeFileBackedSkills(t *testing.T) {
	service := controlapp.NewService()
	for _, legacyID := range append([]string{"memory", "memory-maintenance"}, retiredBuiltinSkillIDs()...) {
		if err := service.UpsertSkill(controldomain.Skill{ID: legacyID, Name: legacyID, Enabled: true, Scope: controldomain.CapabilityScopeGlobal}); err != nil {
			t.Fatalf("seed legacy skill %s: %v", legacyID, err)
		}
	}
	if err := service.UpsertSkill(controldomain.Skill{ID: "travel", Name: "Travel", Enabled: false, Scope: controldomain.CapabilityScopeGlobal}); err != nil {
		t.Fatalf("seed disabled travel skill: %v", err)
	}

	registerBuiltinSkills(service)

	for _, retiredSkillID := range append([]string{"memory", "memory-maintenance"}, retiredBuiltinSkillIDs()...) {
		if _, ok := service.ResolveSkill(retiredSkillID); ok {
			t.Fatalf("did not expect retired skill %q to remain registered", retiredSkillID)
		}
	}

	if _, ok := service.ResolveSkill("travel-page"); ok {
		t.Fatalf("did not expect legacy travel-page skill to remain registered")
	}
	expectedFileBackedSkills := map[string]string{
		"preview-publish": "docs/skills/preview-publish/SKILL.md",
		"travel":          "docs/skills/travel/SKILL.md",
	}
	for skillID, expectedPath := range expectedFileBackedSkills {
		skill, ok := service.ResolveSkill(skillID)
		if !ok {
			t.Fatalf("expected %s skill exists", skillID)
		}
		if got := skill.Metadata[builtinSkillFilePathKey]; got != expectedPath {
			t.Fatalf("%s skill file path = %q, want %s", skillID, got, expectedPath)
		}
	}

	previewSkill, ok := service.ResolveSkill("preview-publish")
	if !ok {
		t.Fatalf("expected preview-publish skill exists")
	}
	if got := previewSkill.Metadata[builtinSkillFilePathKey]; got != "docs/skills/preview-publish/SKILL.md" {
		t.Fatalf("preview-publish skill file path = %q, want docs/skills/preview-publish/SKILL.md", got)
	}
	previewGuide := previewSkill.Metadata[builtinSkillGuideKey]
	if !strings.Contains(previewGuide, "deploy_test_service") || !strings.Contains(previewGuide, "workspace-services") {
		t.Fatalf("expected preview-publish guide covers tool and gateway route, got %q", previewGuide)
	}
	if !strings.Contains(previewGuide, "service_type=http") || !strings.Contains(previewGuide, "frontend_dist") {
		t.Fatalf("expected preview-publish guide covers full-stack web preview and static fallback, got %q", previewGuide)
	}
	if !strings.Contains(previewGuide, "single-label") || !strings.Contains(previewGuide, "*.alter0.cn") {
		t.Fatalf("expected preview-publish guide to describe certificate-safe single-label subdomains, got %q", previewGuide)
	}
	if !strings.Contains(previewGuide, "text") || !strings.Contains(previewGuide, "image") || !strings.Contains(previewGuide, "code") {
		t.Fatalf("expected preview-publish guide covers static artifact previews, got %q", previewGuide)
	}
	if !strings.Contains(previewGuide, "local HTML/file path") || !strings.Contains(previewGuide, "https://*.alter0.cn") {
		t.Fatalf("expected preview-publish guide to reject local artifact links, got %q", previewGuide)
	}

	travel, ok := service.ResolveSkill("travel")
	if !ok {
		t.Fatalf("expected travel skill exists")
	}
	if travel.Enabled {
		t.Fatalf("expected an explicitly disabled builtin skill to remain disabled after restart")
	}
	travelGuide := travel.Metadata[builtinSkillGuideKey]
	if !strings.Contains(travelGuide, "city guide") || !strings.Contains(travelGuide, "docs/skills/travel/SKILL.md") {
		t.Fatalf("expected travel guide covers city guide workflow and canonical skill file, got %q", travelGuide)
	}

	if _, ok := service.ResolveSkill("coding"); ok {
		t.Fatalf("did not expect coding skill to be registered")
	}
}

func TestEnsureBuiltinSkillFilesSkipsWhenNoBuiltinFileBackedSkillExists(t *testing.T) {
	if err := ensureBuiltinSkillFiles(); err != nil {
		t.Fatalf("ensureBuiltinSkillFiles() error = %v", err)
	}
}

func TestBuiltinSkillsReconcileIntoNativeCodexCatalog(t *testing.T) {
	service := controlapp.NewService()
	registerBuiltinSkills(service)
	if err := service.UpsertSkill(controldomain.Skill{
		ID: "custom", Name: "Custom", Enabled: true, Scope: controldomain.CapabilityScopeGlobal,
		Metadata: map[string]string{builtinSkillFilePathKey: "docs/skills/travel/SKILL.md"},
	}); err != nil {
		t.Fatalf("seed custom skill: %v", err)
	}
	result := codexapp.NewNativeSkillReconciler(t.TempDir()).Reconcile(nativeSkillSources(service.ListCapabilities()))
	if len(result.Errors) != 0 {
		t.Fatalf("expected every builtin skill to be a valid native Codex skill, got %+v", result.Errors)
	}
	if len(result.Installed) != len(builtinSkills()) {
		t.Fatalf("installed %d native skills, want %d", len(result.Installed), len(builtinSkills()))
	}
}

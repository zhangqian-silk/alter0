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
	for _, legacyID := range []string{"memory", "memory-maintenance"} {
		if err := service.UpsertSkill(controldomain.Skill{ID: legacyID, Name: legacyID, Enabled: true, Scope: controldomain.CapabilityScopeGlobal}); err != nil {
			t.Fatalf("seed legacy skill %s: %v", legacyID, err)
		}
	}

	registerBuiltinSkills(service)

	for _, retiredSkillID := range []string{"memory", "memory-maintenance"} {
		if _, ok := service.ResolveSkill(retiredSkillID); ok {
			t.Fatalf("did not expect retired skill %q to remain registered", retiredSkillID)
		}
	}

	if _, ok := service.ResolveSkill("travel-page"); ok {
		t.Fatalf("did not expect legacy travel-page skill to remain registered")
	}
	expectedFileBackedSkills := map[string]string{
		"preview-publish":         "docs/skills/preview-publish/SKILL.md",
		"frontend-design":         "docs/skills/frontend-design/SKILL.md",
		"doc-coauthoring":         "docs/skills/doc-coauthoring/SKILL.md",
		"fullstack-developer":     "docs/skills/fullstack-developer/SKILL.md",
		"code-reviewer":           "docs/skills/code-reviewer/SKILL.md",
		"webapp-testing":          "docs/skills/webapp-testing/SKILL.md",
		"find-skills":             "docs/skills/find-skills/SKILL.md",
		"test-driven-development": "docs/skills/test-driven-development/SKILL.md",
		"ui-ux-pro-max":           "docs/skills/ui-ux-pro-max/SKILL.md",
		"code-simplifier":         "docs/skills/code-simplifier/SKILL.md",
		"code-review":             "docs/skills/code-review/SKILL.md",
		"brainstorming":           "docs/skills/brainstorming/SKILL.md",
		"travel":                  "docs/skills/travel/SKILL.md",
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

	frontendDesign, ok := service.ResolveSkill("frontend-design")
	if !ok {
		t.Fatalf("expected frontend-design skill exists")
	}
	if got := frontendDesign.Metadata[builtinSkillFilePathKey]; got != "docs/skills/frontend-design/SKILL.md" {
		t.Fatalf("frontend-design skill file path = %q, want docs/skills/frontend-design/SKILL.md", got)
	}
	frontendGuide := frontendDesign.Metadata[builtinSkillGuideKey]
	if !strings.Contains(frontendGuide, "BOLD aesthetic direction") || !strings.Contains(frontendGuide, "Avoid generic fonts like Arial and Inter") {
		t.Fatalf("expected frontend-design guide covers imported frontend direction, got %q", frontendGuide)
	}

	codeSimplifier, ok := service.ResolveSkill("code-simplifier")
	if !ok {
		t.Fatalf("expected code-simplifier skill exists")
	}
	codeSimplifierGuide := codeSimplifier.Metadata[builtinSkillGuideKey]
	if !strings.Contains(codeSimplifierGuide, "preserving all functionality") || !strings.Contains(codeSimplifierGuide, "docs/skills/code-simplifier/SKILL.md") {
		t.Fatalf("expected code-simplifier guide covers simplification contract and canonical file, got %q", codeSimplifierGuide)
	}

	codeReview, ok := service.ResolveSkill("code-review")
	if !ok {
		t.Fatalf("expected code-review skill exists")
	}
	codeReviewGuide := codeReview.Metadata[builtinSkillGuideKey]
	if !strings.Contains(codeReviewGuide, "pull request") || !strings.Contains(codeReviewGuide, "docs/skills/code-review/SKILL.md") {
		t.Fatalf("expected code-review guide covers PR review workflow and canonical file, got %q", codeReviewGuide)
	}

	travel, ok := service.ResolveSkill("travel")
	if !ok {
		t.Fatalf("expected travel skill exists")
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
	result := codexapp.NewNativeSkillReconciler(t.TempDir()).Reconcile(nativeSkillSources(service.ListCapabilities()))
	if len(result.Errors) != 0 {
		t.Fatalf("expected every builtin skill to be a valid native Codex skill, got %+v", result.Errors)
	}
	if len(result.Installed) != len(builtinSkills()) {
		t.Fatalf("installed %d native skills, want %d", len(result.Installed), len(builtinSkills()))
	}
}

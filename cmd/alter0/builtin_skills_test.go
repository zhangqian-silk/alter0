package main

import (
	"strings"
	"testing"

	controlapp "alter0/internal/control/application"
)

func TestRegisterBuiltinSkillsSeedsMemorySkill(t *testing.T) {
	service := controlapp.NewService()

	registerBuiltinSkills(service)

	memory, ok := service.ResolveSkill("memory")
	if !ok {
		t.Fatalf("expected memory skill exists")
	}
	if !memory.Enabled {
		t.Fatalf("expected memory skill enabled")
	}
	if got := memory.Metadata[builtinSkillDescriptionKey]; got == "" {
		t.Fatalf("expected memory skill description")
	}
	guide := memory.Metadata[builtinSkillGuideKey]
	if !strings.Contains(guide, "USER.md") || !strings.Contains(guide, "SOUL.md") || !strings.Contains(guide, "AGENTS.md") {
		t.Fatalf("expected memory skill guide covers system files, got %q", guide)
	}
	if !strings.Contains(guide, "Write routing") || !strings.Contains(guide, "Read logic") {
		t.Fatalf("expected memory skill guide covers explicit read/write routing, got %q", guide)
	}
	if !strings.Contains(guide, "Conflict rules") || !strings.Contains(guide, "Write constraints") {
		t.Fatalf("expected memory skill guide covers conflict and write constraints, got %q", guide)
	}

	if _, ok := service.ResolveSkill("travel-page"); ok {
		t.Fatalf("did not expect legacy travel-page skill to remain registered")
	}

	expectedFileBackedSkills := map[string]string{
		"memory-maintenance":      "docs/skills/memory-maintenance/SKILL.md",
		"deploy-test-service":     "docs/skills/deploy-test-service/SKILL.md",
		"frontend-design":         "docs/skills/frontend-design/SKILL.md",
		"artifact-preview":        "docs/skills/artifact-preview/SKILL.md",
		"doc-coauthoring":         "docs/skills/doc-coauthoring/SKILL.md",
		"fullstack-developer":     "docs/skills/fullstack-developer/SKILL.md",
		"code-reviewer":           "docs/skills/code-reviewer/SKILL.md",
		"webapp-testing":          "docs/skills/webapp-testing/SKILL.md",
		"find-skills":             "docs/skills/find-skills/SKILL.md",
		"test-driven-development": "docs/skills/test-driven-development/SKILL.md",
		"ui-ux-pro-max":           "docs/skills/ui-ux-pro-max/SKILL.md",
		"code-simplifier":         "docs/skills/code-simplifier/agents/code-simplifier.md",
		"code-review":             "docs/skills/code-review/commands/code-review.md",
		"brainstorming":           "docs/skills/brainstorming/SKILL.md",
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

	memoryMaintenance, ok := service.ResolveSkill("memory-maintenance")
	if !ok {
		t.Fatalf("expected memory-maintenance skill exists")
	}
	memoryMaintenanceGuide := memoryMaintenance.Metadata[builtinSkillGuideKey]
	if !strings.Contains(memoryMaintenanceGuide, "daily memory") || !strings.Contains(memoryMaintenanceGuide, "long-term memory") {
		t.Fatalf("expected memory-maintenance guide covers memory consolidation, got %q", memoryMaintenanceGuide)
	}

	deploySkill, ok := service.ResolveSkill("deploy-test-service")
	if !ok {
		t.Fatalf("expected deploy-test-service skill exists")
	}
	if got := deploySkill.Metadata[builtinSkillFilePathKey]; got != "docs/skills/deploy-test-service/SKILL.md" {
		t.Fatalf("deploy-test-service skill file path = %q, want docs/skills/deploy-test-service/SKILL.md", got)
	}
	deployGuide := deploySkill.Metadata[builtinSkillGuideKey]
	if !strings.Contains(deployGuide, "deploy_test_service") || !strings.Contains(deployGuide, "workspace-services") {
		t.Fatalf("expected deploy-test-service guide covers tool and gateway route, got %q", deployGuide)
	}
	if !strings.Contains(deployGuide, "service_type=http") || !strings.Contains(deployGuide, "frontend_dist") {
		t.Fatalf("expected deploy-test-service guide covers full-stack web preview and static fallback, got %q", deployGuide)
	}
	if !strings.Contains(deployGuide, "single-label") || !strings.Contains(deployGuide, "*.alter0.cn") {
		t.Fatalf("expected deploy-test-service guide to describe certificate-safe single-label subdomains, got %q", deployGuide)
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

	artifactPreview, ok := service.ResolveSkill("artifact-preview")
	if !ok {
		t.Fatalf("expected artifact-preview skill exists")
	}
	if got := artifactPreview.Metadata[builtinSkillFilePathKey]; got != "docs/skills/artifact-preview/SKILL.md" {
		t.Fatalf("artifact-preview skill file path = %q, want docs/skills/artifact-preview/SKILL.md", got)
	}
	artifactGuide := artifactPreview.Metadata[builtinSkillGuideKey]
	if !strings.Contains(artifactGuide, "docs/skills/artifact-preview/scripts/publish_preview_artifact.sh") || !strings.Contains(artifactGuide, "<service>-<short_hash>.alter0.cn") {
		t.Fatalf("expected artifact-preview guide covers helper script and session subdomain host, got %q", artifactGuide)
	}
	if !strings.Contains(artifactGuide, "single-label") || !strings.Contains(artifactGuide, "*.alter0.cn") {
		t.Fatalf("expected artifact-preview guide to describe certificate-safe single-label subdomains, got %q", artifactGuide)
	}
	if !strings.Contains(artifactGuide, "text") || !strings.Contains(artifactGuide, "image") || !strings.Contains(artifactGuide, "code") {
		t.Fatalf("expected artifact-preview guide covers text, image, and code previews, got %q", artifactGuide)
	}
	if !strings.Contains(artifactGuide, "local HTML/file path") || !strings.Contains(artifactGuide, "https://*.alter0.cn") {
		t.Fatalf("expected artifact-preview guide to reject local artifact links, got %q", artifactGuide)
	}

	codeSimplifier, ok := service.ResolveSkill("code-simplifier")
	if !ok {
		t.Fatalf("expected code-simplifier skill exists")
	}
	codeSimplifierGuide := codeSimplifier.Metadata[builtinSkillGuideKey]
	if !strings.Contains(codeSimplifierGuide, "preserving all functionality") || !strings.Contains(codeSimplifierGuide, "docs/skills/code-simplifier/agents/code-simplifier.md") {
		t.Fatalf("expected code-simplifier guide covers simplification contract and canonical file, got %q", codeSimplifierGuide)
	}

	codeReview, ok := service.ResolveSkill("code-review")
	if !ok {
		t.Fatalf("expected code-review skill exists")
	}
	codeReviewGuide := codeReview.Metadata[builtinSkillGuideKey]
	if !strings.Contains(codeReviewGuide, "pull request") || !strings.Contains(codeReviewGuide, "docs/skills/code-review/commands/code-review.md") {
		t.Fatalf("expected code-review guide covers PR review workflow and canonical file, got %q", codeReviewGuide)
	}
}

func TestEnsureBuiltinSkillFilesSkipsWhenNoBuiltinFileBackedSkillExists(t *testing.T) {
	if err := ensureBuiltinSkillFiles(); err != nil {
		t.Fatalf("ensureBuiltinSkillFiles() error = %v", err)
	}
}

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

	deploySkill, ok := service.ResolveSkill("deploy-test-service")
	if !ok {
		t.Fatalf("expected deploy-test-service skill exists")
	}
	deployGuide := deploySkill.Metadata[builtinSkillGuideKey]
	if !strings.Contains(deployGuide, "deploy_test_service") || !strings.Contains(deployGuide, "workspace-services") {
		t.Fatalf("expected deploy-test-service guide covers tool and gateway route, got %q", deployGuide)
	}
	if !strings.Contains(deployGuide, "service_type=http") || !strings.Contains(deployGuide, "frontend_dist") {
		t.Fatalf("expected deploy-test-service guide covers full-stack web preview and static fallback, got %q", deployGuide)
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
	if got := artifactPreview.Metadata[builtinSkillFilePathKey]; got != ".alter0/skills/artifact-preview/SKILL.md" {
		t.Fatalf("artifact-preview skill file path = %q, want .alter0/skills/artifact-preview/SKILL.md", got)
	}
	artifactGuide := artifactPreview.Metadata[builtinSkillGuideKey]
	if !strings.Contains(artifactGuide, "scripts/publish_preview_artifact.sh") || !strings.Contains(artifactGuide, "<service>.<short_hash>.alter0.cn") {
		t.Fatalf("expected artifact-preview guide covers helper script and session subdomain host, got %q", artifactGuide)
	}
	if !strings.Contains(artifactGuide, "text") || !strings.Contains(artifactGuide, "image") || !strings.Contains(artifactGuide, "code") {
		t.Fatalf("expected artifact-preview guide covers text, image, and code previews, got %q", artifactGuide)
	}
}

func TestEnsureBuiltinSkillFilesSkipsWhenNoBuiltinFileBackedSkillExists(t *testing.T) {
	if err := ensureBuiltinSkillFiles(); err != nil {
		t.Fatalf("ensureBuiltinSkillFiles() error = %v", err)
	}
}

package main

import (
	"os"
	"path/filepath"
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

	travelPage, ok := service.ResolveSkill("travel-page")
	if !ok {
		t.Fatalf("expected travel-page skill exists")
	}
	if travelPage.Metadata[builtinSkillFilePathKey] != ".alter0/skills/travel-page.md" {
		t.Fatalf("unexpected travel-page skill file path: %+v", travelPage.Metadata)
	}
	if travelPage.Metadata[builtinSkillWritableKey] != "true" {
		t.Fatalf("expected travel-page skill writable metadata, got %+v", travelPage.Metadata)
	}
	if guide := travelPage.Metadata[builtinSkillGuideKey]; !strings.Contains(guide, "read_skill") || !strings.Contains(guide, "durable") {
		t.Fatalf("expected travel-page skill guide covers read/write policy, got %q", guide)
	}
}

func TestEnsureBuiltinSkillFilesCreatesTravelPageRulebook(t *testing.T) {
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	root := t.TempDir()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir temp root failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	if err := ensureBuiltinSkillFiles(); err != nil {
		t.Fatalf("ensureBuiltinSkillFiles() error = %v", err)
	}

	path := filepath.Join(root, ".alter0", "skills", "travel-page.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read travel-page.md failed: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "# Travel Page Rulebook") || !strings.Contains(content, "durable, reusable preferences") {
		t.Fatalf("unexpected travel-page skill file content: %q", content)
	}
}

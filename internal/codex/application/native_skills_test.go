package application

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNativeSkillReconcilerInstallsEnabledPublicSkillDirectory(t *testing.T) {
	sourceRoot := t.TempDir()
	skillDir := filepath.Join(sourceRoot, "frontend-design")
	if err := os.MkdirAll(filepath.Join(skillDir, "scripts"), 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: frontend-design\ndescription: Build interfaces.\n---\n"), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "scripts", "helper.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write helper: %v", err)
	}
	if err := os.Chmod(filepath.Join(skillDir, "scripts", "helper.sh"), 0o755); err != nil {
		t.Fatalf("chmod helper: %v", err)
	}

	destinationRoot := t.TempDir()
	reconciler := NewNativeSkillReconciler(destinationRoot)
	result := reconciler.Reconcile([]NativeSkillSource{{
		ID:       "frontend-design",
		Enabled:  true,
		Public:   true,
		FilePath: filepath.Join(skillDir, "SKILL.md"),
	}})
	if len(result.Errors) != 0 {
		t.Fatalf("unexpected reconcile errors: %+v", result.Errors)
	}
	if len(result.Installed) != 1 || result.Installed[0] != "frontend-design" {
		t.Fatalf("unexpected installed skills: %+v", result.Installed)
	}
	installedDir := filepath.Join(destinationRoot, "alter0-frontend-design")
	if _, err := os.Stat(filepath.Join(installedDir, "SKILL.md")); err != nil {
		t.Fatalf("expected native skill: %v", err)
	}
	info, err := os.Stat(filepath.Join(installedDir, "scripts", "helper.sh"))
	if err != nil {
		t.Fatalf("expected helper: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("expected executable mode, got %o", info.Mode().Perm())
	}
}

func TestNativeSkillReconcilerRemovesOnlyManagedDisabledSkills(t *testing.T) {
	sourceDir := filepath.Join(t.TempDir(), "summary")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("---\nname: summary\ndescription: Summarize.\n---\n"), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	destinationRoot := t.TempDir()
	reconciler := NewNativeSkillReconciler(destinationRoot)
	first := reconciler.Reconcile([]NativeSkillSource{{ID: "summary", Enabled: true, Public: true, FilePath: filepath.Join(sourceDir, "SKILL.md")}})
	if len(first.Errors) != 0 {
		t.Fatalf("install errors: %+v", first.Errors)
	}
	unmanaged := filepath.Join(destinationRoot, "custom-user-skill")
	if err := os.MkdirAll(unmanaged, 0o755); err != nil {
		t.Fatalf("mkdir unmanaged: %v", err)
	}
	if err := os.WriteFile(filepath.Join(unmanaged, "SKILL.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write unmanaged: %v", err)
	}

	second := reconciler.Reconcile([]NativeSkillSource{{ID: "summary", Enabled: false, Public: true, FilePath: filepath.Join(sourceDir, "SKILL.md")}})
	if len(second.Errors) != 0 {
		t.Fatalf("disable errors: %+v", second.Errors)
	}
	if _, err := os.Stat(filepath.Join(destinationRoot, "alter0-summary")); !os.IsNotExist(err) {
		t.Fatalf("expected managed disabled skill removed, got err=%v", err)
	}
	if _, err := os.Stat(unmanaged); err != nil {
		t.Fatalf("expected unmanaged skill preserved: %v", err)
	}
}

func TestNativeSkillReconcilerPreservesUnmanagedDestinationCollision(t *testing.T) {
	sourceDir := filepath.Join(t.TempDir(), "summary")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("---\nname: summary\ndescription: Summarize.\n---\n"), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	destinationRoot := t.TempDir()
	collision := filepath.Join(destinationRoot, "alter0-summary")
	if err := os.MkdirAll(collision, 0o755); err != nil {
		t.Fatalf("mkdir collision: %v", err)
	}
	if err := os.WriteFile(filepath.Join(collision, "SKILL.md"), []byte("unmanaged"), 0o644); err != nil {
		t.Fatalf("write collision: %v", err)
	}

	result := NewNativeSkillReconciler(destinationRoot).Reconcile([]NativeSkillSource{{ID: "summary", Enabled: true, Public: true, FilePath: filepath.Join(sourceDir, "SKILL.md")}})
	if len(result.Errors) != 1 || result.Errors[0].Code != "destination_collision" {
		t.Fatalf("expected collision error, got %+v", result.Errors)
	}
	content, err := os.ReadFile(filepath.Join(collision, "SKILL.md"))
	if err != nil || string(content) != "unmanaged" {
		t.Fatalf("expected unmanaged collision preserved, content=%q err=%v", string(content), err)
	}
}

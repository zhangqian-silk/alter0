package application_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	codexapp "alter0/internal/codex/application"
)

func TestServiceListSkillsUsesCodexCatalogAndSanitizesPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	activeHome := filepath.Join(home, ".codex")
	cwd := filepath.Join(home, "workspace", "repo")
	userSkillPath := filepath.Join(home, ".agents", "skills", "frontend-design", "SKILL.md")
	codexSkillPath := filepath.Join(activeHome, "skills", "frontend-design", "SKILL.md")
	managedSkillDir := filepath.Join(home, ".agents", "skills", "alter0-preview-publish")
	managedSkillPath := filepath.Join(managedSkillDir, "SKILL.md")
	repoSkillPath := filepath.Join(cwd, ".agents", "skills", "project-workflow", "SKILL.md")
	brokenSkillPath := filepath.Join(cwd, ".agents", "skills", "broken", "SKILL.md")
	if err := os.MkdirAll(managedSkillDir, 0o755); err != nil {
		t.Fatalf("mkdir managed skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(managedSkillDir, ".alter0-managed.json"), []byte(`{"managed_by":"alter0","skill_id":"preview-publish"}`), 0o644); err != nil {
		t.Fatalf("write managed marker: %v", err)
	}

	service := codexapp.NewService(codexapp.ServiceOptions{
		Command:           "/usr/local/bin/codex",
		ResolveActiveHome: func() (string, error) { return activeHome, nil },
		RunCommand: func(_ context.Context, _ string, args []string, options codexapp.CommandOptions) error {
			if len(args) == 0 || args[0] != "app-server" {
				t.Fatalf("command args = %v, want app-server", args)
			}
			if got := envValue(options.Env, "CODEX_HOME"); got != activeHome {
				t.Fatalf("CODEX_HOME = %q, want %q", got, activeHome)
			}
			requests := decodeInteractiveAppServerRequests(t, options.Stdin, options.Stdout)
			if method := appServerMethod(requests); method != "skills/list" {
				t.Fatalf("method = %q, want skills/list", method)
			}
			params, _ := requests[len(requests)-1]["params"].(map[string]any)
			if forceReload, _ := params["forceReload"].(bool); !forceReload {
				t.Fatalf("forceReload = %v, want true", params["forceReload"])
			}
			cwds, _ := params["cwds"].([]any)
			if len(cwds) != 1 || cwds[0] != cwd {
				t.Fatalf("cwds = %#v, want [%q]", cwds, cwd)
			}
			writeAppServerResponse(t, options.Stdout, 2, map[string]any{
				"data": []map[string]any{{
					"cwd": cwd,
					"skills": []map[string]any{
						{
							"name": "preview-publish", "description": "Publish previews.", "enabled": true,
							"path": managedSkillPath, "scope": "user",
						},
						{
							"name": "frontend-design", "description": "User copy.", "enabled": true,
							"path": userSkillPath, "scope": "user",
							"interface":    map[string]any{"displayName": "Frontend Design", "shortDescription": "Design production interfaces."},
							"dependencies": map[string]any{"tools": []map[string]any{{"type": "command", "value": "node", "command": "node"}}},
						},
						{
							"name": "frontend-design", "description": "Codex home copy.", "enabled": false,
							"path": codexSkillPath, "scope": "user",
						},
						{
							"name": "project-workflow", "description": "Repository workflow.", "enabled": true,
							"path": repoSkillPath, "scope": "repo",
						},
					},
					"errors": []map[string]any{{"path": brokenSkillPath, "message": "invalid frontmatter at " + brokenSkillPath}},
				}},
			})
			return nil
		},
	})

	catalog, err := service.ListSkills(context.Background(), cwd)
	if err != nil {
		t.Fatalf("ListSkills returned error: %v", err)
	}
	if len(catalog.Items) != 4 {
		t.Fatalf("items = %d, want 4", len(catalog.Items))
	}
	byLocation := map[codexapp.NativeSkillLocation]codexapp.NativeSkillCatalogItem{}
	for _, item := range catalog.Items {
		byLocation[item.Location] = item
	}
	if item := byLocation[codexapp.NativeSkillLocationAlter0]; item.ManagedSkillID != "preview-publish" {
		t.Fatalf("managed skill = %+v, want preview-publish", item)
	}
	if item := byLocation[codexapp.NativeSkillLocationUserAgents]; item.Name != "frontend-design" || !item.Duplicate {
		t.Fatalf("user skill = %+v, want duplicate frontend-design", item)
	}
	if item := byLocation[codexapp.NativeSkillLocationCodexHome]; item.Name != "frontend-design" || !item.Duplicate || item.Enabled {
		t.Fatalf("codex home skill = %+v, want disabled duplicate frontend-design", item)
	}
	if item := byLocation[codexapp.NativeSkillLocationRepo]; item.Name != "project-workflow" || item.Duplicate {
		t.Fatalf("repo skill = %+v, want unique project-workflow", item)
	}
	if len(catalog.Errors) != 1 || catalog.Errors[0].Location != codexapp.NativeSkillLocationRepo {
		t.Fatalf("errors = %+v, want one repo parse error", catalog.Errors)
	}

	raw, err := json.Marshal(catalog)
	if err != nil {
		t.Fatalf("marshal catalog: %v", err)
	}
	if strings.Contains(string(raw), home) || strings.Contains(string(raw), "SKILL.md") {
		t.Fatalf("catalog leaked an absolute path: %s", raw)
	}
}

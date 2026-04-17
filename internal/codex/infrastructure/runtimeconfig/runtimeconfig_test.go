package runtimeconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareCopiesBaselineAuthAndCompilesManagedConfig(t *testing.T) {
	activeHome := t.TempDir()
	if err := os.WriteFile(filepath.Join(activeHome, authFileName), []byte(`{"auth_mode":"apikey","OPENAI_API_KEY":"sk-test"}`), 0o600); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	baseConfig := strings.Join([]string{
		`model = "gpt-5.4"`,
		`[mcp_servers.legacy]`,
		`command = "legacy"`,
		"",
		`[features]`,
		`rmcp_client = true`,
	}, "\n")
	if err := os.WriteFile(filepath.Join(activeHome, configFileName), []byte(baseConfig), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	workspaceDir := filepath.Join(t.TempDir(), "workspace")
	runtimeHome := filepath.Join(t.TempDir(), "codex-home")
	prepared, err := Prepare(Spec{
		ActiveHome:   activeHome,
		RuntimeHome:  runtimeHome,
		WorkspaceDir: workspaceDir,
		MCPServers: []MCPServer{
			{
				Name:              "filesystem",
				Transport:         "stdio",
				Command:           "npx",
				Args:              []string{"-y", "@modelcontextprotocol/server-filesystem"},
				Env:               map[string]string{"NODE_ENV": "production"},
				EnabledTools:      []string{"read_file", "list_dir"},
				StartupTimeoutSec: 12,
				ToolTimeoutSec:    34,
				Required:          true,
			},
		},
		ManagedFiles: []ManagedFile{
			{RelativePath: ".alter0/codex-runtime/skills.md", Content: "# Skills\n"},
		},
		RootInstructions: "- Read `.alter0/codex-runtime/skills.md` before acting.",
	})
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if len(prepared.Env) != 1 || prepared.Env[0] != "CODEX_HOME="+runtimeHome {
		t.Fatalf("unexpected runtime env: %+v", prepared.Env)
	}

	authCopy, err := os.ReadFile(filepath.Join(runtimeHome, authFileName))
	if err != nil {
		t.Fatalf("read copied auth: %v", err)
	}
	if !strings.Contains(string(authCopy), "sk-test") {
		t.Fatalf("unexpected copied auth: %s", string(authCopy))
	}

	configCopy, err := os.ReadFile(filepath.Join(runtimeHome, configFileName))
	if err != nil {
		t.Fatalf("read compiled config: %v", err)
	}
	configText := string(configCopy)
	for _, expected := range []string{
		`model = "gpt-5.4"`,
		`[features]`,
		`[mcp_servers.filesystem]`,
		`command = "npx"`,
		`enabled_tools = ["read_file", "list_dir"]`,
		`startup_timeout_sec = 12`,
		`tool_timeout_sec = 34`,
		`required = true`,
	} {
		if !strings.Contains(configText, expected) {
			t.Fatalf("expected compiled config to contain %q, got:\n%s", expected, configText)
		}
	}
	if strings.Contains(configText, "[mcp_servers.legacy]") {
		t.Fatalf("expected legacy MCP sections stripped, got:\n%s", configText)
	}

	skillFile, err := os.ReadFile(filepath.Join(workspaceDir, ".alter0", "codex-runtime", "skills.md"))
	if err != nil {
		t.Fatalf("read managed skill file: %v", err)
	}
	if string(skillFile) != "# Skills\n" {
		t.Fatalf("unexpected skill file: %q", string(skillFile))
	}

	agentsFile, err := os.ReadFile(filepath.Join(workspaceDir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if !strings.Contains(string(agentsFile), "Alter0 Codex Runtime") || !strings.Contains(string(agentsFile), ".alter0/codex-runtime/skills.md") {
		t.Fatalf("unexpected AGENTS.md:\n%s", string(agentsFile))
	}
}

func TestPreparePreservesExistingAgentsFileOutsideManagedBlock(t *testing.T) {
	workspaceDir := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	existing := "# Repo Policy\n\n- Keep tests updated.\n"
	if err := os.WriteFile(filepath.Join(workspaceDir, "AGENTS.md"), []byte(existing), 0o644); err != nil {
		t.Fatalf("write existing AGENTS: %v", err)
	}

	_, err := Prepare(Spec{
		RuntimeHome:      filepath.Join(t.TempDir(), "codex-home"),
		WorkspaceDir:     workspaceDir,
		RootInstructions: "- Read `.alter0/codex-runtime/runtime.md`.",
	})
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	agentsFile, err := os.ReadFile(filepath.Join(workspaceDir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	agentsText := string(agentsFile)
	if !strings.Contains(agentsText, "Repo Policy") {
		t.Fatalf("expected original AGENTS content preserved, got:\n%s", agentsText)
	}
	if !strings.Contains(agentsText, managedAgentsStartMarker) {
		t.Fatalf("expected managed block in AGENTS.md, got:\n%s", agentsText)
	}
}

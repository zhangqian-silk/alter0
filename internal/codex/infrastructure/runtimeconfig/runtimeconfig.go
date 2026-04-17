package runtimeconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	codexapp "alter0/internal/codex/application"
)

const (
	configFileName           = "config.toml"
	authFileName             = "auth.json"
	managedConfigStartMarker = "# alter0:codex-runtime:start"
	managedConfigEndMarker   = "# alter0:codex-runtime:end"
	managedAgentsStartMarker = "<!-- alter0:codex-runtime:start -->"
	managedAgentsEndMarker   = "<!-- alter0:codex-runtime:end -->"
)

type Spec struct {
	ActiveHome       string
	RuntimeHome      string
	WorkspaceDir     string
	MCPServers       []MCPServer
	ManagedFiles     []ManagedFile
	RootInstructions string
}

type MCPServer struct {
	Name              string
	Transport         string
	Command           string
	Args              []string
	Env               map[string]string
	URL               string
	Headers           map[string]string
	EnvHeaders        map[string]string
	BearerTokenEnvVar string
	EnabledTools      []string
	DisabledTools     []string
	StartupTimeoutSec int
	ToolTimeoutSec    int
	Required          bool
}

type ManagedFile struct {
	RelativePath string
	Content      string
	Mode         os.FileMode
}

type Prepared struct {
	RuntimeHome string
	Env         []string
}

func Prepare(spec Spec) (Prepared, error) {
	runtimeHome := strings.TrimSpace(spec.RuntimeHome)
	workspaceDir := strings.TrimSpace(spec.WorkspaceDir)
	if runtimeHome == "" {
		return Prepared{}, fmt.Errorf("runtime codex home is required")
	}
	if workspaceDir == "" {
		return Prepared{}, fmt.Errorf("workspace dir is required")
	}
	if err := os.MkdirAll(runtimeHome, 0o755); err != nil {
		return Prepared{}, fmt.Errorf("prepare runtime codex home: %w", err)
	}
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		return Prepared{}, fmt.Errorf("prepare runtime workspace: %w", err)
	}

	activeHome := strings.TrimSpace(spec.ActiveHome)
	if activeHome == "" {
		resolvedHome, err := codexapp.ResolveActiveHome()
		if err != nil {
			return Prepared{}, fmt.Errorf("resolve active codex home: %w", err)
		}
		activeHome = resolvedHome
	}
	if err := copyRuntimeFile(filepath.Join(activeHome, authFileName), filepath.Join(runtimeHome, authFileName)); err != nil {
		return Prepared{}, err
	}

	baseConfig, err := readOptionalFile(filepath.Join(activeHome, configFileName))
	if err != nil {
		return Prepared{}, err
	}
	mergedConfig := mergeConfig(baseConfig, renderManagedMCPConfig(spec.MCPServers))
	if strings.TrimSpace(mergedConfig) != "" {
		if err := os.WriteFile(filepath.Join(runtimeHome, configFileName), []byte(mergedConfig), 0o600); err != nil {
			return Prepared{}, fmt.Errorf("write runtime codex config: %w", err)
		}
	}

	for _, file := range spec.ManagedFiles {
		if err := writeManagedFile(workspaceDir, file); err != nil {
			return Prepared{}, err
		}
	}
	if strings.TrimSpace(spec.RootInstructions) != "" {
		if err := writeManagedAgentsFile(workspaceDir, spec.RootInstructions); err != nil {
			return Prepared{}, err
		}
	}

	return Prepared{
		RuntimeHome: runtimeHome,
		Env:         []string{"CODEX_HOME=" + runtimeHome},
	}, nil
}

func copyRuntimeFile(source string, destination string) error {
	content, err := os.ReadFile(source)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read codex runtime file %s: %w", source, err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fmt.Errorf("prepare runtime file dir %s: %w", filepath.Dir(destination), err)
	}
	if err := os.WriteFile(destination, content, 0o600); err != nil {
		return fmt.Errorf("write runtime file %s: %w", destination, err)
	}
	return nil
}

func readOptionalFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read file %s: %w", path, err)
	}
	return string(content), nil
}

func mergeConfig(baseConfig string, managedConfig string) string {
	baseConfig = stripManagedBlock(baseConfig, managedConfigStartMarker, managedConfigEndMarker)
	baseConfig = stripMCPServerSections(baseConfig)
	baseConfig = strings.TrimRight(baseConfig, "\n")
	managedConfig = strings.TrimSpace(managedConfig)
	switch {
	case baseConfig == "" && managedConfig == "":
		return ""
	case baseConfig == "":
		return managedConfig + "\n"
	case managedConfig == "":
		return baseConfig + "\n"
	default:
		return baseConfig + "\n\n" + managedConfig + "\n"
	}
}

func stripManagedBlock(content string, startMarker string, endMarker string) string {
	start := strings.Index(content, startMarker)
	end := strings.Index(content, endMarker)
	if start < 0 || end < 0 || end < start {
		return content
	}
	end += len(endMarker)
	trimmed := content[:start] + content[end:]
	return strings.TrimSpace(trimmed)
}

func stripMCPServerSections(content string) string {
	lines := strings.Split(content, "\n")
	filtered := make([]string, 0, len(lines))
	skipping := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[mcp_servers.") && strings.HasSuffix(trimmed, "]") {
			skipping = true
			continue
		}
		if skipping && strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			skipping = false
		}
		if skipping {
			continue
		}
		filtered = append(filtered, line)
	}
	return strings.TrimSpace(strings.Join(filtered, "\n"))
}

func renderManagedMCPConfig(servers []MCPServer) string {
	if len(servers) == 0 {
		return ""
	}
	items := append([]MCPServer(nil), servers...)
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(strings.TrimSpace(items[i].Name)) < strings.ToLower(strings.TrimSpace(items[j].Name))
	})
	lines := []string{managedConfigStartMarker}
	for _, server := range items {
		name := strings.TrimSpace(server.Name)
		if name == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("[mcp_servers.%s]", name))
		switch strings.ToLower(strings.TrimSpace(server.Transport)) {
		case "stdio":
			lines = append(lines, fmt.Sprintf("command = %q", strings.TrimSpace(server.Command)))
			if len(server.Args) > 0 {
				lines = append(lines, fmt.Sprintf("args = %s", renderStringArray(server.Args)))
			}
			if len(server.Env) > 0 {
				lines = append(lines, fmt.Sprintf("env = %s", renderStringMap(server.Env)))
			}
		case "http", "streamable_http":
			lines = append(lines, fmt.Sprintf("url = %q", strings.TrimSpace(server.URL)))
			if strings.TrimSpace(server.BearerTokenEnvVar) != "" {
				lines = append(lines, fmt.Sprintf("bearer_token_env_var = %q", strings.TrimSpace(server.BearerTokenEnvVar)))
			}
			if len(server.Headers) > 0 {
				lines = append(lines, fmt.Sprintf("http_headers = %s", renderStringMap(server.Headers)))
			}
			if len(server.EnvHeaders) > 0 {
				lines = append(lines, fmt.Sprintf("env_http_headers = %s", renderStringMap(server.EnvHeaders)))
			}
		default:
			continue
		}
		if len(server.EnabledTools) > 0 {
			lines = append(lines, fmt.Sprintf("enabled_tools = %s", renderStringArray(server.EnabledTools)))
		}
		if len(server.DisabledTools) > 0 {
			lines = append(lines, fmt.Sprintf("disabled_tools = %s", renderStringArray(server.DisabledTools)))
		}
		if server.StartupTimeoutSec > 0 {
			lines = append(lines, fmt.Sprintf("startup_timeout_sec = %d", server.StartupTimeoutSec))
		}
		if server.ToolTimeoutSec > 0 {
			lines = append(lines, fmt.Sprintf("tool_timeout_sec = %d", server.ToolTimeoutSec))
		}
		if server.Required {
			lines = append(lines, "required = true")
		}
		lines = append(lines, "")
	}
	lines = append(lines, managedConfigEndMarker)
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func renderStringArray(values []string) string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		items = append(items, fmt.Sprintf("%q", trimmed))
	}
	return "[" + strings.Join(items, ", ") + "]"
}

func renderStringMap(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	items := make([]string, 0, len(keys))
	for _, key := range keys {
		items = append(items, fmt.Sprintf("%q = %q", key, values[key]))
	}
	return "{ " + strings.Join(items, ", ") + " }"
}

func writeManagedFile(workspaceDir string, file ManagedFile) error {
	relativePath := strings.TrimSpace(file.RelativePath)
	if relativePath == "" {
		return nil
	}
	mode := file.Mode
	if mode == 0 {
		mode = 0o644
	}
	path := filepath.Join(workspaceDir, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("prepare managed runtime file dir %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(file.Content), mode); err != nil {
		return fmt.Errorf("write managed runtime file %s: %w", path, err)
	}
	return nil
}

func writeManagedAgentsFile(workspaceDir string, instructions string) error {
	path := filepath.Join(workspaceDir, "AGENTS.md")
	existing, err := readOptionalFile(path)
	if err != nil {
		return err
	}
	block := strings.Join([]string{
		managedAgentsStartMarker,
		"## Alter0 Codex Runtime",
		"",
		strings.TrimSpace(instructions),
		managedAgentsEndMarker,
	}, "\n")
	content := upsertManagedBlock(existing, block, managedAgentsStartMarker, managedAgentsEndMarker)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write runtime AGENTS.md: %w", err)
	}
	return nil
}

func upsertManagedBlock(existing string, block string, startMarker string, endMarker string) string {
	trimmedExisting := strings.TrimSpace(existing)
	trimmedBlock := strings.TrimSpace(block)
	if trimmedExisting == "" {
		return trimmedBlock + "\n"
	}
	start := strings.Index(trimmedExisting, startMarker)
	end := strings.Index(trimmedExisting, endMarker)
	if start >= 0 && end >= start {
		end += len(endMarker)
		updated := strings.TrimSpace(trimmedExisting[:start]) + "\n\n" + trimmedBlock + "\n\n" + strings.TrimSpace(trimmedExisting[end:])
		return strings.TrimSpace(updated) + "\n"
	}
	return trimmedBlock + "\n\n" + trimmedExisting + "\n"
}

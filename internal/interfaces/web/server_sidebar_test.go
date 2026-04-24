package web

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

func readEmbeddedAssetRaw(t *testing.T, path string) string {
	t.Helper()

	resolvedPath := resolveEmbeddedAssetPath(path)
	content, err := webStaticFS.ReadFile(resolvedPath)
	if err != nil {
		t.Fatalf("read asset %s: %v", resolvedPath, err)
	}
	return string(content)
}

func resolveEmbeddedAssetPath(assetPath string) string {
	if strings.HasPrefix(assetPath, "static/assets/") {
		relativePath := strings.TrimPrefix(assetPath, "static/assets/")
		return path.Join("static", "dist", "legacy", relativePath)
	}
	return assetPath
}

func readEmbeddedAsset(t *testing.T, assetPath string) string {
	t.Helper()

	content := readEmbeddedAssetRaw(t, assetPath)
	if !strings.HasSuffix(assetPath, ".css") {
		return content
	}
	return expandEmbeddedCSSImports(t, assetPath, content, map[string]bool{assetPath: true})
}

func readWorkspaceFile(t *testing.T, relativePath string) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Clean(relativePath))
	if err != nil {
		t.Fatalf("read workspace file %s: %v", relativePath, err)
	}
	return string(content)
}

func expandEmbeddedCSSImports(t *testing.T, assetPath, content string, seen map[string]bool) string {
	t.Helper()

	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	var builder strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		importPath, ok := parseEmbeddedCSSImport(trimmed)
		if !ok {
			builder.WriteString(line)
			builder.WriteByte('\n')
			continue
		}
		resolved := path.Clean(path.Join(path.Dir(assetPath), importPath))
		if seen[resolved] {
			continue
		}
		seen[resolved] = true
		builder.WriteString(expandEmbeddedCSSImports(t, resolved, readEmbeddedAssetRaw(t, resolved), seen))
	}
	return builder.String()
}

func parseEmbeddedCSSImport(line string) (string, bool) {
	const prefix = "@import url(\""
	const suffix = "\");"
	if !strings.HasPrefix(line, prefix) || !strings.HasSuffix(line, suffix) {
		return "", false
	}
	pathValue := strings.TrimSuffix(strings.TrimPrefix(line, prefix), suffix)
	pathValue = strings.TrimSpace(pathValue)
	if pathValue == "" {
		return "", false
	}
	return pathValue, true
}

func normalizeEmbeddedAsset(content string) string {
	return strings.ReplaceAll(content, "\r\n", "\n")
}

func TestWorkbenchAppOwnsRouteShellState(t *testing.T) {
	source := readWorkspaceFile(t, "frontend/src/app/WorkbenchApp.tsx")
	markers := []string{
		"const [navCollapsed, setNavCollapsed] = useState(false);",
		`const [mobilePanel, setMobilePanel] = useState<"nav" | "sessions" | null>(null);`,
		`const navOpen = mobilePanel === "nav";`,
		`classNames.push("nav-open", "overlay-open")`,
		`<div className={shellClassName} data-workbench-route={route}>`,
		`isConversationRoute(route) || route === "terminal" ? (`,
		`<RuntimeRouteHost route={route} language={language} />`,
	}
	for _, marker := range markers {
		if !strings.Contains(source, marker) {
			t.Fatalf("expected source marker %q", marker)
		}
	}
}

func TestWorkbenchRouteStateHashNavigationPresent(t *testing.T) {
	source := readWorkspaceFile(t, "frontend/src/app/routeState.ts")
	markers := []string{
		`export const DEFAULT_WORKBENCH_ROUTE = "chat";`,
		"const KNOWN_ROUTES = new Set(",
		"window.addEventListener(\"hashchange\", syncRoute);",
		"window.dispatchEvent(new HashChangeEvent(\"hashchange\"));",
		"return route === \"chat\" || route === \"agent-runtime\";",
	}
	for _, marker := range markers {
		if !strings.Contains(source, marker) {
			t.Fatalf("expected route source marker %q", marker)
		}
	}
}

func TestSidebarPageRouteRuntimeDependenciesPresent(t *testing.T) {
	script := readWorkspaceFile(t, "frontend/src/features/shell/components/ReactManagedRouteBody.tsx")
	markers := []string{
		"const REACT_MANAGED_ROUTE_BODY_RENDERERS",
		`terminal: () => <ReactManagedTerminalRouteBody />`,
		`tasks: ({ language }) => <ReactManagedTasksRouteBody language={language} />`,
		`channels: ({ language }) => <ReactManagedControlRouteBody route="channels" language={language} />`,
		"if (!isReactManagedRouteBody(route)) {",
	}
	for _, marker := range markers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected route source marker %q", marker)
		}
	}
}

func TestLegacyRuntimeDropsCronVisualHelpers(t *testing.T) {
	script := readWorkspaceFile(t, "frontend/src/features/shell/components/ReactManagedControlRouteBody.tsx")
	forbiddenMarkers := []string{
		"function parseCronExpressionVisual(expression)",
		"function buildCronExpressionVisual(options = {})",
		"expressionInput.value = buildCronExpressionVisual({",
		"const parsed = parseCronExpressionVisual(expressionInput.value);",
	}
	for _, marker := range forbiddenMarkers {
		if strings.Contains(script, marker) {
			t.Fatalf("unexpected managed legacy marker %q", marker)
		}
	}
}

func TestPageSharedHelpersPresent(t *testing.T) {
	clientSource := readWorkspaceFile(t, "frontend/src/shared/api/client.ts")
	tasksSource := readWorkspaceFile(t, "frontend/src/features/shell/components/ReactManagedTasksRouteBody.tsx")
	markers := []string{
		"export class APIClientError extends Error {",
		"function readResponsePayload(response: Response): Promise<unknown>",
		"createAPIClient(options: APIClientOptions = {})",
		"const TASK_ROUTE_FILTERS_STORAGE_KEY = \"alter0.web.tasks.route-filters.v1\";",
		"handleTerminalSubmit(terminalInput);",
		"control-task-log-stream",
	}
	for _, marker := range markers {
		if !strings.Contains(clientSource+tasksSource, marker) {
			t.Fatalf("expected shared helper marker %q", marker)
		}
	}
}

func TestSidebarInfoModeStylesPresent(t *testing.T) {
	styles := readWorkspaceFile(t, "frontend/src/styles/shell.css")
	markers := []string{
		".app-shell.info-mode {",
		".app-shell.nav-collapsed.info-mode {",
		".workbench-main {",
	}
	for _, marker := range markers {
		if !strings.Contains(styles, marker) {
			t.Fatalf("expected style marker %q", marker)
		}
	}
}

func TestSidebarScrollIsolationStylesPresent(t *testing.T) {
	styles := normalizeEmbeddedAsset(readEmbeddedAsset(t, "static/assets/chat.css"))
	markers := []string{
		"@media (min-width: 1101px) {",
		"html,\n  body {\n    overflow: hidden;",
		".app-shell {\n    min-height: 0;\n    height: 100vh;",
		".menu {\n    overflow-y: auto;",
		".session-list,\n  .message-area,\n  .route-view {\n    overscroll-behavior: contain;",
		".route-view {\n    flex: 1;\n    min-height: 0;\n    overflow-y: auto;",
	}
	for _, marker := range markers {
		if !strings.Contains(styles, marker) {
			t.Fatalf("expected style marker %q", marker)
		}
	}
}

func TestSidebarGroupTitlesHaveDedicatedI18NKeys(t *testing.T) {
	script := readWorkspaceFile(t, "frontend/src/features/shell/legacyShellCopy.ts")
	scriptMarkers := []string{
		`Workspace: "Workspace"`,
		`Control: "Control"`,
		`agent: "Profiles"`,
		`"agent-runtime": "Agent"`,
		`Settings: "Settings"`,
		`Workspace: "工作区"`,
		`Control: "控制台"`,
		`agent: "配置"`,
		`Settings: "设置"`,
	}
	for _, marker := range scriptMarkers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}
}

func TestPrimaryNavCollapseStateHooksPresent(t *testing.T) {
	source := readWorkspaceFile(t, "frontend/src/app/WorkbenchApp.tsx")
	markers := []string{
		"const [navCollapsed, setNavCollapsed] = useState(false);",
		"setNavCollapsed((current) => !current);",
		`setMobilePanel((current) => current === "nav" ? null : "nav");`,
	}
	for _, marker := range markers {
		if !strings.Contains(source, marker) {
			t.Fatalf("expected source marker %q", marker)
		}
	}

	styles := readWorkspaceFile(t, "frontend/src/styles/shell.css")
	styleMarkers := []string{
		".app-shell.nav-collapsed {",
		".app-shell.nav-collapsed .menu-label {",
		".menu-icon {",
	}
	for _, marker := range styleMarkers {
		if !strings.Contains(styles, marker) {
			t.Fatalf("expected style marker %q", marker)
		}
	}
}

func TestConversationWorkspaceSessionSurfacePresent(t *testing.T) {
	source := readWorkspaceFile(t, "frontend/src/features/conversation-runtime/ConversationWorkspace.tsx") +
		readWorkspaceFile(t, "frontend/src/features/shell/components/RuntimeWorkspacePage.tsx")
	markers := []string{
		`"data-runtime-view": "conversation"`,
		`"data-runtime-route": runtime.route`,
		`data-runtime-session-pane`,
		`data-runtime-session-list`,
		`data-runtime-workspace`,
		`data-runtime-screen`,
		`deleteLabel: deleteSessionLabel,`,
		`"runtime-session-delete"`,
		`{item.shortHash}`,
	}
	for _, marker := range markers {
		if !strings.Contains(source, marker) {
			t.Fatalf("expected source marker %q", marker)
		}
	}

	styles := readWorkspaceFile(t, "frontend/src/styles/shell.css")
	styleMarkers := []string{
		`[data-runtime-view="conversation"] {`,
		".runtime-workspace-session-pane {",
		".runtime-workspace {",
	}
	for _, marker := range styleMarkers {
		if !strings.Contains(styles, marker) {
			t.Fatalf("expected style marker %q", marker)
		}
	}
}

func TestWorkbenchConversationAndManagedRoutesPresent(t *testing.T) {
	source := readWorkspaceFile(t, "frontend/src/app/WorkbenchApp.tsx") +
		readWorkspaceFile(t, "frontend/src/features/shell/components/RuntimeRouteHost.tsx") +
		readWorkspaceFile(t, "frontend/src/features/shell/components/ReactManagedRouteBody.tsx")
	markers := []string{
		"<ConversationRuntimeProvider route={route} language={language}>",
		"<ConversationWorkspace language={language} />",
		"<RuntimeRouteHost route={route} language={language} />",
		"<ReactManagedRouteBody route={route} language={language} />",
		`isConversationRoute(route) || route === "terminal"`,
		`agent: ({ language }) => <ReactManagedAgentRouteBody language={language} />`,
		`memory: ({ language }) => <ReactManagedMemoryRouteBody language={language} />`,
	}
	for _, marker := range markers {
		if !strings.Contains(source, marker) {
			t.Fatalf("expected source marker %q", marker)
		}
	}
}

func TestWorkbenchRouteHeadingCopyRemainsCentralized(t *testing.T) {
	source := readWorkspaceFile(t, "frontend/src/features/shell/legacyShellCopy.ts")
	requiredMarkers := []string{
		`tasks: "Tasks"`,
		`tasks: "Observe runtime tasks with source, status, and timeline filters"`,
		`tasks: "Tasks"`,
		`Observe runtime tasks with source, status, and timeline filters`,
	}
	for _, marker := range requiredMarkers {
		if !strings.Contains(source, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}
}

func TestWorkbenchRetiresManagedWorkflowLoaders(t *testing.T) {
	script := readWorkspaceFile(t, "frontend/src/features/shell/components/ReactManagedRouteBody.tsx")
	requiredMarkers := []string{
		`sessions: ({ language }) => <ReactManagedSessionsRouteBody language={language} />`,
		`tasks: ({ language }) => <ReactManagedTasksRouteBody language={language} />`,
	}
	for _, marker := range requiredMarkers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}
}

func TestWorkbenchRetiresManagedControlLoaders(t *testing.T) {
	script := readWorkspaceFile(t, "frontend/src/features/shell/components/ReactManagedRouteBody.tsx")
	requiredMarkers := []string{
		`channels: ({ language }) => <ReactManagedControlRouteBody route="channels" language={language} />`,
		`skills: ({ language }) => <ReactManagedControlRouteBody route="skills" language={language} />`,
		`mcp: ({ language }) => <ReactManagedControlRouteBody route="mcp" language={language} />`,
		`models: ({ language }) => <ReactManagedControlRouteBody route="models" language={language} />`,
		`environments: ({ language }) => <ReactManagedControlRouteBody route="environments" language={language} />`,
		`"cron-jobs": ({ language }) => <ReactManagedControlRouteBody route="cron-jobs" language={language} />`,
	}
	for _, marker := range requiredMarkers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}
}

func TestSidebarAgentMemoryTabStylesPresent(t *testing.T) {
	styles := readEmbeddedAsset(t, "static/assets/chat.css")
	markers := []string{
		".agent-studio-view {",
		".agent-route-card {",
		".agent-builder-form {",
		".agent-builder-option {",
		".memory-view {",
		".memory-tabs {",
		".memory-tab.active {",
		".memory-content {",
		".memory-spec-sections {",
	}
	for _, marker := range markers {
		if !strings.Contains(styles, marker) {
			t.Fatalf("expected style marker %q", marker)
		}
	}
}

func TestSidebarTerminalModulePresent(t *testing.T) {
	script := readWorkspaceFile(t, "frontend/src/features/shell/components/ReactManagedTerminalRouteBody.tsx")
	scriptMarkers := []string{
		"data-runtime-view",
		"data-runtime-session-pane",
		"data-runtime-create-session",
		"data-runtime-delete-session",
		"data-terminal-step-toggle",
	}
	for _, marker := range scriptMarkers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}

	styles := readWorkspaceFile(t, "frontend/src/styles/shell.css")
	styleMarkers := []string{
		`[data-runtime-view="terminal"] .runtime-session-card {`,
		".runtime-session-card {",
		".runtime-composer-form {",
	}
	for _, marker := range styleMarkers {
		if !strings.Contains(styles, marker) {
			t.Fatalf("expected style marker %q", marker)
		}
	}
}

func TestRuntimeRestartNoticeBridgeRemainsInShell(t *testing.T) {
	script := readWorkspaceFile(t, "frontend/src/features/shell/components/ReactManagedControlRouteBody.tsx")
	markers := []string{
		`statusPendingRestart: "Pending restart"`,
		`statusPendingRestart: "待重启"`,
		`path: "/api/control/environments"`,
		"pending_restart?: boolean;",
		"enabled: (item) => !Boolean((item as EnvironmentRouteRecord).pending_restart),",
	}
	for _, marker := range markers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected environment restart marker %q", marker)
		}
	}
}

func TestControlTaskLogStreamMobileStickMarkersPresent(t *testing.T) {
	script := readWorkspaceFile(t, "frontend/src/features/shell/components/ReactManagedTasksRouteBody.tsx")
	scriptMarkers := []string{
		"className=\"control-task-log-stream\"",
		"className=\"control-task-terminal-screen\"",
		"className=\"control-task-terminal-input\"",
		"void reconnectLogs()",
		"event.preventDefault();",
	}
	for _, marker := range scriptMarkers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}

	styles := readEmbeddedAsset(t, "static/assets/chat.css")
	styleMarkers := []string{
		".control-task-log-stream {",
		"overscroll-behavior-y: none;",
		"overflow-anchor: none;",
	}
	for _, marker := range styleMarkers {
		if !strings.Contains(styles, marker) {
			t.Fatalf("expected style marker %q", marker)
		}
	}
}

func TestMobilePollingPerformanceGuardsPresent(t *testing.T) {
	script := readWorkspaceFile(t, "frontend/src/features/conversation-runtime/ConversationRuntimeProvider.tsx") +
		readWorkspaceFile(t, "frontend/src/shared/viewport/mobileViewport.ts") +
		readWorkspaceFile(t, "frontend/src/features/shell/components/ReactManagedTerminalRouteBody.tsx")
	markers := []string{
		"const CHAT_TASK_POLL_INTERVAL_MS = 3000;",
		"message.taskID && message.taskPending && !message.taskResultDelivered",
		"export const MOBILE_KEYBOARD_MIN_OFFSET_PX = 120;",
		"const keyboardOffset = rawKeyboardOffset >= MOBILE_KEYBOARD_MIN_OFFSET_PX",
		"const timer = window.setTimeout(() => {",
	}
	for _, marker := range markers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected mobile polling marker %q", marker)
		}
	}
}

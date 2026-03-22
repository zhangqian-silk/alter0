package web

import (
	"path"
	"strings"
	"testing"
)

func readEmbeddedAssetRaw(t *testing.T, path string) string {
	t.Helper()

	content, err := webStaticFS.ReadFile(path)
	if err != nil {
		t.Fatalf("read asset %s: %v", path, err)
	}
	return string(content)
}

func readEmbeddedAsset(t *testing.T, assetPath string) string {
	t.Helper()

	content := readEmbeddedAssetRaw(t, assetPath)
	if !strings.HasSuffix(assetPath, ".css") {
		return content
	}
	return expandEmbeddedCSSImports(t, assetPath, content, map[string]bool{assetPath: true})
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

func TestSidebarRoutesSwitchToInfoMode(t *testing.T) {
	script := readEmbeddedAsset(t, "static/assets/chat.js")
	markers := []string{
		"function setMainContentMode(mode)",
		`appShell.classList.toggle("info-mode", infoMode)`,
		"chatView.hidden = infoMode;",
		"routeView.hidden = !infoMode;",
	}
	for _, marker := range markers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}
}

func TestSidebarPageRouteRuntimeDependenciesPresent(t *testing.T) {
	script := readEmbeddedAsset(t, "static/assets/chat.js")
	markers := []string{
		"function syncRouteAction(route)",
		"async function fetchJSON(path)",
		"async function renderRoute(route)",
		"await config.loader(routeBody);",
		"routeBody.innerHTML = `<p class=\"route-loading\">${t(\"loading\")}</p>`;",
	}
	for _, marker := range markers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected route runtime marker %q", marker)
		}
	}
}

func TestCronVisualHelpersPresent(t *testing.T) {
	script := readEmbeddedAsset(t, "static/assets/chat.js")
	markers := []string{
		"function parseCronExpressionVisual(expression)",
		"function buildCronExpressionVisual(options = {})",
		"expressionInput.value = buildCronExpressionVisual({",
		"const parsed = parseCronExpressionVisual(expressionInput.value);",
	}
	for _, marker := range markers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected cron helper marker %q", marker)
		}
	}
}

func TestPageSharedHelpersPresent(t *testing.T) {
	script := readEmbeddedAsset(t, "static/assets/chat.js")
	markers := []string{
		"function routeFieldRow(labelKey, value, options = {})",
		"async function copyTextValue(value)",
		"function routeCardTemplate(title, type, fields = [], enabled = false, body = \"\")",
		"function escapeQueryValue(value)",
		"async function safeReadJSON(response)",
		"function renderAdvancedToggleLabel(expanded)",
		"async function downloadTaskArtifact(downloadURL, fallbackName)",
		"routeBody.addEventListener(\"click\", async (event) => {",
		"const target = event.target.closest(\"[data-copy-value]\");",
	}
	for _, marker := range markers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected shared helper marker %q", marker)
		}
	}
}

func TestSidebarInfoModeStylesPresent(t *testing.T) {
	styles := readEmbeddedAsset(t, "static/assets/chat.css")
	markers := []string{
		".app-shell.info-mode {",
		".app-shell.info-mode .session-pane {",
		".app-shell.panel-open:not(.info-mode) .session-pane {",
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

func TestSidebarCollapseEntryPresent(t *testing.T) {
	html := readEmbeddedAsset(t, "static/chat.html")
	markers := []string{
		`id="navCollapseButton"`,
		`class="menu-icon"`,
		`class="menu-label"`,
	}
	for _, marker := range markers {
		if !strings.Contains(html, marker) {
			t.Fatalf("expected html marker %q", marker)
		}
	}
}

func TestSidebarChannelsRouteMovesToSettingsGroup(t *testing.T) {
	html := readEmbeddedAsset(t, "static/chat.html")
	controlMarker := `data-i18n="nav.control"`
	settingsMarker := `data-i18n="nav.settings"`
	channelsMarker := `data-route="channels"`
	modelsMarker := `data-route="models"`

	controlIndex := strings.Index(html, controlMarker)
	settingsIndex := strings.Index(html, settingsMarker)
	channelsIndex := strings.Index(html, channelsMarker)
	modelsIndex := strings.Index(html, modelsMarker)
	if controlIndex == -1 || settingsIndex == -1 || channelsIndex == -1 || modelsIndex == -1 {
		t.Fatalf("expected menu markers for control/settings/channels/models")
	}
	if controlIndex >= settingsIndex {
		t.Fatalf("expected control group before settings")
	}
	if channelsIndex < settingsIndex {
		t.Fatalf("expected channels route under settings group")
	}
	if channelsIndex > modelsIndex {
		t.Fatalf("expected channels route before models in settings group")
	}

	controlSection := html[controlIndex:settingsIndex]
	if strings.Contains(controlSection, channelsMarker) {
		t.Fatalf("unexpected channels route in control group")
	}
	settingsSection := html[settingsIndex:]
	if !strings.Contains(settingsSection, channelsMarker) {
		t.Fatalf("expected channels route in settings group")
	}
}

func TestSidebarTerminalRouteMovesToChatGroup(t *testing.T) {
	html := readEmbeddedAsset(t, "static/chat.html")
	workspaceMarker := `data-i18n="nav.workspace"`
	agentStudioMarker := `data-i18n="nav.agent_studio"`
	controlMarker := `data-i18n="nav.control"`
	terminalMarker := `data-route="terminal"`
	chatRouteMarker := `data-route="chat"`

	workspaceIndex := strings.Index(html, workspaceMarker)
	agentStudioIndex := strings.Index(html, agentStudioMarker)
	controlIndex := strings.Index(html, controlMarker)
	terminalIndex := strings.Index(html, terminalMarker)
	chatRouteIndex := strings.Index(html, chatRouteMarker)
	if workspaceIndex == -1 || agentStudioIndex == -1 || controlIndex == -1 || terminalIndex == -1 || chatRouteIndex == -1 {
		t.Fatalf("expected menu markers for workspace/agent-studio/control/terminal")
	}
	if !(workspaceIndex < agentStudioIndex && agentStudioIndex < controlIndex) {
		t.Fatalf("expected menu groups order workspace -> agent-studio -> control")
	}
	if terminalIndex < workspaceIndex || terminalIndex > agentStudioIndex {
		t.Fatalf("expected terminal route under workspace group")
	}
	if terminalIndex < chatRouteIndex {
		t.Fatalf("expected terminal route after chat route in workspace group")
	}

	workspaceSection := html[workspaceIndex:agentStudioIndex]
	if !strings.Contains(workspaceSection, terminalMarker) {
		t.Fatalf("expected terminal route in workspace group")
	}
	controlSection := html[controlIndex:]
	if strings.Contains(controlSection, terminalMarker) {
		t.Fatalf("unexpected terminal route in control group")
	}
}

func TestSidebarGroupTitlesHaveDedicatedI18NKeys(t *testing.T) {
	html := readEmbeddedAsset(t, "static/chat.html")
	htmlMarkers := []string{
		`data-i18n="nav.workspace"`,
		`data-i18n="nav.agent_studio"`,
		`data-i18n="nav.control"`,
		`data-i18n="nav.settings"`,
	}
	for _, marker := range htmlMarkers {
		if !strings.Contains(html, marker) {
			t.Fatalf("expected html marker %q", marker)
		}
	}

	script := readEmbeddedAsset(t, "static/assets/chat.js")
	scriptMarkers := []string{
		`"nav.workspace": "Workspace"`,
		`"nav.agent_studio": "Agent Studio"`,
		`"nav.control": "Control"`,
		`"nav.agent": "Agents"`,
		`"nav.settings": "Settings"`,
		`"nav.workspace": "工作区"`,
		`"nav.agent_studio": "Agent Studio"`,
		`"nav.control": "控制台"`,
		`"nav.agent": "Agents"`,
		`"nav.settings": "设置"`,
	}
	for _, marker := range scriptMarkers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}
}

func TestSidebarCollapseStateHooksPresent(t *testing.T) {
	script := readEmbeddedAsset(t, "static/assets/chat.js")
	markers := []string{
		"function setSidebarCollapsed(collapsed)",
		"function collapseMobileSidebar()",
		`appShell.classList.toggle("nav-collapsed", collapsed)`,
		`navCollapseButton.addEventListener("click", () => {`,
		"collapseMobileSidebar();",
	}
	for _, marker := range markers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}

	styles := readEmbeddedAsset(t, "static/assets/chat.css")
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

func TestSessionHistoryCollapseControlsPresent(t *testing.T) {
	html := readEmbeddedAsset(t, "static/chat.html")
	htmlMarkers := []string{
		`id="sessionHistoryToggle"`,
		`id="sessionHistoryPanel"`,
		`class="session-history-toggle"`,
		`class="session-history-panel"`,
	}
	for _, marker := range htmlMarkers {
		if !strings.Contains(html, marker) {
			t.Fatalf("expected html marker %q", marker)
		}
	}

	script := readEmbeddedAsset(t, "static/assets/chat.js")
	scriptMarkers := []string{
		"function setSessionHistoryCollapsed(collapsed)",
		"function syncSessionHistoryPanel()",
		`sessionHistoryPanel.hidden = collapsed;`,
		`sessionHistoryToggle.dataset.collapsedState = collapsed ? "collapsed" : "expanded";`,
		`sessionHistoryToggle.addEventListener("click", () => {`,
	}
	for _, marker := range scriptMarkers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}

	styles := readEmbeddedAsset(t, "static/assets/chat.css")
	styleMarkers := []string{
		".session-history-head {",
		".session-history-toggle {",
		".session-history-panel {",
	}
	for _, marker := range styleMarkers {
		if !strings.Contains(styles, marker) {
			t.Fatalf("expected style marker %q", marker)
		}
	}
}

func TestSessionHistoryCollapseStatePersistsInBrowserSession(t *testing.T) {
	script := readEmbeddedAsset(t, "static/assets/chat.js")
	markers := []string{
		`const SESSION_HISTORY_PANEL_STORAGE_KEY = "alter0.web.session-history-panel.v1";`,
		"function getBrowserSessionStorage()",
		"window.sessionStorage",
		"function loadSessionHistoryCollapsedState()",
		"function persistSessionHistoryCollapsedState()",
		`collapsed_state: state.sessionHistoryCollapsed`,
		"setSessionHistoryCollapsed(loadSessionHistoryCollapsedState());",
		"persistSessionHistoryCollapsedState();",
	}
	for _, marker := range markers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}
}

func TestSidebarAgentMemoryConvergesRoutes(t *testing.T) {
	html := readEmbeddedAsset(t, "static/chat.html")
	expected := []string{
		`data-route="agent"`,
		`data-route="memory"`,
		`data-i18n="nav.memory"`,
	}
	for _, marker := range expected {
		if !strings.Contains(html, marker) {
			t.Fatalf("expected html marker %q", marker)
		}
	}
	forbidden := []string{
		`data-route="configuration"`,
		`data-i18n="nav.configuration"`,
	}
	for _, marker := range forbidden {
		if strings.Contains(html, marker) {
			t.Fatalf("unexpected html marker %q", marker)
		}
	}

	script := readEmbeddedAsset(t, "static/assets/chat.js")
	scriptMarkers := []string{
		`agent: {`,
		"loader: loadAgentView",
		`"route.agent.title"`,
		`"/api/control/agents"`,
		`data-agent-chat-now`,
		`openChatWithTarget({`,
		`memory: {`,
		"loader: loadMemoryView",
		`"/api/agent/memory"`,
		`data-memory-tab`,
		`route.memory.tab.specification`,
		`data-memory-panel="specification"`,
	}
	for _, marker := range scriptMarkers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}
	scriptForbidden := []string{
		"workspace: {",
		"configuration: {",
	}
	for _, marker := range scriptForbidden {
		if strings.Contains(script, marker) {
			t.Fatalf("unexpected script marker %q", marker)
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
	html := readEmbeddedAsset(t, "static/chat.html")
	htmlMarkers := []string{
		`data-route="terminal"`,
		`data-i18n="nav.terminal"`,
	}
	for _, marker := range htmlMarkers {
		if !strings.Contains(html, marker) {
			t.Fatalf("expected html marker %q", marker)
		}
	}

	script := readEmbeddedAsset(t, "static/assets/chat.js")
	scriptMarkers := []string{
		`const TERMINAL_STORAGE_KEY = "alter0.web.terminal.sessions.v2";`,
		`terminal: {`,
		`loader: loadTerminalView`,
		`"route.terminal.title"`,
		`"route.terminal.send"`,
		`"route.terminal.close"`,
	}
	for _, marker := range scriptMarkers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}

	styles := readEmbeddedAsset(t, "static/assets/chat.css")
	styleMarkers := []string{
		".terminal-view {",
		".terminal-session-card {",
		".terminal-chat-form {",
		".terminal-session-close {",
	}
	for _, marker := range styleMarkers {
		if !strings.Contains(styles, marker) {
			t.Fatalf("expected style marker %q", marker)
		}
	}
}

func TestEnvironmentRestartControlsPresent(t *testing.T) {
	script := readEmbeddedAsset(t, "static/assets/chat.js")
	markers := []string{
		`"route.envs.restart_service": "Restart Service"`,
		`"route.envs.restart_service": "重启服务"`,
		`"route.envs.restart_sync_master": "Sync remote master before restart?"`,
		`"route.envs.restart_sync_master": "重启前先同步远端 master 分支？"`,
		`data-environment-restart`,
		"const requestRuntimeRestart = async () => {",
		`fetch("/api/control/runtime/restart", {`,
		`"sync_remote_master": shouldSyncRemoteMaster`,
		"const waitForRuntimeReady = async () => {",
	}
	for _, marker := range markers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected environment restart marker %q", marker)
		}
	}
}

func TestControlTaskLogStreamMobileStickMarkersPresent(t *testing.T) {
	script := readEmbeddedAsset(t, "static/assets/chat.js")
	scriptMarkers := []string{
		"logStickToBottom: true",
		"const isNearLogBottom = (node, threshold = 24) => {",
		"const scrollLogToBottom = (node) => {",
		"const bindLogStreamNode = (streamNode) => {",
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

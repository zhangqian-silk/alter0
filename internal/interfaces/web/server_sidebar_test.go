package web

import (
	"path"
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

func TestLegacyRuntimeDoesNotToggleReactOwnedRouteShellState(t *testing.T) {
	script := readEmbeddedAsset(t, "static/assets/chat.js")
	requiredMarkers := []string{
		"function navigateToRoute(route, options = {})",
		"setMainContentMode(\"chat\")",
		"setMainContentMode(\"page\")",
		`window.location.hash = targetHash;`,
	}
	for _, marker := range requiredMarkers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}

	forbiddenMarkers := []string{
		`appShell.classList.toggle("info-mode", infoMode)`,
		`chatPane.classList.toggle("page-mode", infoMode)`,
		"chatView.hidden = infoMode;",
		"routeView.hidden = !infoMode;",
		"chatPane.dataset.route = safe;",
		"routeView.dataset.route = safe;",
		"routeBody.dataset.route = safe;",
		`routeView.classList.toggle("terminal-route", safe === "terminal");`,
		`routeBody.classList.toggle("terminal-route-body", safe === "terminal");`,
	}
	for _, marker := range forbiddenMarkers {
		if strings.Contains(script, marker) {
			t.Fatalf("unexpected script marker %q", marker)
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
	script := readEmbeddedAsset(t, "static/assets/chat.js")
	markers := []string{
		`const navCollapseButton = document.getElementById("navCollapseButton");`,
		`appShell.classList.toggle("nav-collapsed", collapsed)`,
		`navCollapseButton.addEventListener("click", () => {`,
	}
	for _, marker := range markers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}
}

func TestSidebarGroupTitlesHaveDedicatedI18NKeys(t *testing.T) {
	script := readEmbeddedAsset(t, "static/assets/chat.js")
	scriptMarkers := []string{
		`"nav.workspace": "Workspace"`,
		`"nav.agent_studio": "Agent Studio"`,
		`"nav.control": "Control"`,
		`"nav.agent": "Profiles"`,
		`"nav.agent_runtime": "Agent"`,
		`"nav.settings": "Settings"`,
		`"nav.workspace": "工作区"`,
		`"nav.agent_studio": "Agent Studio"`,
		`"nav.control": "控制台"`,
		`"nav.agent": "配置"`,
		`"nav.agent_runtime": "Agent"`,
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

func TestLegacyRuntimeDoesNotRewriteReactOwnedShellActionLabels(t *testing.T) {
	script := readEmbeddedAsset(t, "static/assets/chat.js")
	requiredMarkers := []string{
		"newChatButton.addEventListener(\"click\", () => {",
		"mobileNewChatButton.addEventListener(\"click\", () => {",
		"sessionToggle.addEventListener(\"click\", (event) => {",
	}
	for _, marker := range requiredMarkers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}

	forbiddenMarkers := []string{
		"newChatButton.textContent = newSessionLabel;",
		"mobileNewChatButton.textContent = t(\"route.terminal.new_short\");",
		"mobileNewChatButton.setAttribute(\"aria-label\", t(\"route.terminal.new_short\"));",
		"mobileNewChatButton.textContent = newSessionLabel;",
		"mobileNewChatButton.setAttribute(\"aria-label\", newSessionLabel);",
		"sessionToggle.textContent = t(\"route.terminal.sessions\");",
		"sessionToggle.setAttribute(\"aria-label\", t(\"route.terminal.sessions\"));",
		"sessionToggle.textContent = t(\"chat.sessions\");",
		"sessionToggle.setAttribute(\"aria-label\", t(\"chat.sessions\"));",
	}
	for _, marker := range forbiddenMarkers {
		if strings.Contains(script, marker) {
			t.Fatalf("unexpected script marker %q", marker)
		}
	}
}

func TestLegacyRuntimeDoesNotRewriteReactOwnedRouteHeadingCopy(t *testing.T) {
	script := readEmbeddedAsset(t, "static/assets/chat.js")
	requiredMarkers := []string{
		"const titleKey = `route.${routeKey}.title`;",
		"const subtitleKey = `route.${routeKey}.subtitle`;",
		"syncRouteAction(safe);",
	}
	for _, marker := range requiredMarkers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}

	forbiddenMarkers := []string{
		"routeTitle.textContent = t(titleKey);",
		"routeSubtitle.textContent = t(subtitleKey);",
	}
	for _, marker := range forbiddenMarkers {
		if strings.Contains(script, marker) {
			t.Fatalf("unexpected script marker %q", marker)
		}
	}
}

func TestSidebarAgentMemoryConvergesRoutes(t *testing.T) {
	script := readEmbeddedAsset(t, "static/assets/chat.js")
	scriptMarkers := []string{
		`agent: {`,
		"loader: loadAgentView",
		`"route.agent.title"`,
		`"/api/control/agents"`,
		`data-agent-chat-now`,
		`openAgentRuntimeWithTarget({`,
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
		`"route.envs.restart_sync_master": "Sync remote master changes before restart"`,
		`"route.envs.restart_sync_master": "重启前同步远端 master 最新改动"`,
		`"route.envs.restart_confirm_desc": "The page will reload automatically after the new runtime passes health checks."`,
		`"route.envs.restart_confirm_desc": "新实例探活通过后，当前页面会自动刷新并重新连接。"`,
		`const RUNTIME_RESTART_NOTICE_STORAGE_KEY = "alter0.web.runtime.restart-notice.v1";`,
		`data-environment-restart`,
		"const requestRuntimeRestart = async () => {",
		"showGlobalConfirmModal({",
		`data-global-modal-checkbox`,
		`fetch("/api/control/runtime/restart", {`,
		`"sync_remote_master": shouldSyncRemoteMaster`,
		"const waitForRuntimeReady = async () => {",
		"persistRuntimeRestartNotice();",
		"showPendingRuntimeRestartNotice();",
		`data-global-modal-root`,
	}
	for _, marker := range markers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected environment restart marker %q", marker)
		}
	}
	if strings.Contains(script, `window.confirm(t("route.envs.restart_confirm"))`) {
		t.Fatalf("expected restart confirmation to avoid window.confirm")
	}
	if strings.Contains(script, `window.confirm(t("route.envs.restart_sync_master"))`) {
		t.Fatalf("expected restart sync option to avoid window.confirm")
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

func TestMobilePollingPerformanceGuardsPresent(t *testing.T) {
	script := readEmbeddedAsset(t, "static/assets/chat.js")
	markers := []string{
		"const CHAT_TASK_POLL_HIDDEN_INTERVAL_MS = 15000;",
		"function scheduleChatTaskPolling(options = {}) {",
		"document.addEventListener(\"visibilitychange\", () => {",
		"const TERMINAL_POLL_INTERVAL_HIDDEN_MS = 12000;",
		"const computeTerminalPollDelay = (session) => {",
		"localState.nextListSyncAt = Date.now() + computeTerminalListPollInterval();",
		"window.visualViewport.addEventListener(\"scroll\", () => {",
		"if (!isMobileViewport() || !activeViewportInput()) {",
	}
	for _, marker := range markers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected mobile polling marker %q", marker)
		}
	}
}

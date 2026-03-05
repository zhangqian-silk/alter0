package web

import (
	"strings"
	"testing"
)

func readEmbeddedAsset(t *testing.T, path string) string {
	t.Helper()

	content, err := webStaticFS.ReadFile(path)
	if err != nil {
		t.Fatalf("read asset %s: %v", path, err)
	}
	return string(content)
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
	styles := readEmbeddedAsset(t, "static/assets/chat.css")
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
	agentMarker := `data-i18n="nav.agent"`
	settingsMarker := `data-i18n="nav.settings"`
	channelsMarker := `data-route="channels"`
	modelsMarker := `data-route="models"`

	controlIndex := strings.Index(html, controlMarker)
	agentIndex := strings.Index(html, agentMarker)
	settingsIndex := strings.Index(html, settingsMarker)
	channelsIndex := strings.Index(html, channelsMarker)
	modelsIndex := strings.Index(html, modelsMarker)
	if controlIndex == -1 || agentIndex == -1 || settingsIndex == -1 || channelsIndex == -1 || modelsIndex == -1 {
		t.Fatalf("expected menu markers for control/agent/settings/channels/models")
	}
	if !(controlIndex < agentIndex && agentIndex < settingsIndex) {
		t.Fatalf("expected menu groups order control -> agent -> settings")
	}
	if channelsIndex < settingsIndex {
		t.Fatalf("expected channels route under settings group")
	}
	if channelsIndex > modelsIndex {
		t.Fatalf("expected channels route before models in settings group")
	}

	controlSection := html[controlIndex:agentIndex]
	if strings.Contains(controlSection, channelsMarker) {
		t.Fatalf("unexpected channels route in control group")
	}
	settingsSection := html[settingsIndex:]
	if !strings.Contains(settingsSection, channelsMarker) {
		t.Fatalf("expected channels route in settings group")
	}
}

func TestSidebarGroupTitlesHaveDedicatedI18NKeys(t *testing.T) {
	html := readEmbeddedAsset(t, "static/chat.html")
	htmlMarkers := []string{
		`data-i18n="nav.control"`,
		`data-i18n="nav.agent"`,
		`data-i18n="nav.settings"`,
	}
	for _, marker := range htmlMarkers {
		if !strings.Contains(html, marker) {
			t.Fatalf("expected html marker %q", marker)
		}
	}

	script := readEmbeddedAsset(t, "static/assets/chat.js")
	scriptMarkers := []string{
		`"nav.control": "Control"`,
		`"nav.agent": "Agent"`,
		`"nav.settings": "Settings"`,
		`"nav.control": "控制台"`,
		`"nav.agent": "智能体"`,
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
		`data-route="memory"`,
		`data-i18n="nav.memory"`,
	}
	for _, marker := range expected {
		if !strings.Contains(html, marker) {
			t.Fatalf("expected html marker %q", marker)
		}
	}
	forbidden := []string{
		`data-route="workspace"`,
		`data-route="configuration"`,
		`data-i18n="nav.workspace"`,
		`data-i18n="nav.configuration"`,
	}
	for _, marker := range forbidden {
		if strings.Contains(html, marker) {
			t.Fatalf("unexpected html marker %q", marker)
		}
	}

	script := readEmbeddedAsset(t, "static/assets/chat.js")
	scriptMarkers := []string{
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
		`const TERMINAL_STORAGE_KEY = "alter0.web.terminal.sessions.v1";`,
		`terminal: {`,
		`loader: loadTerminalView`,
		`"route.terminal.title"`,
		`"route.terminal.send"`,
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
	}
	for _, marker := range styleMarkers {
		if !strings.Contains(styles, marker) {
			t.Fatalf("expected style marker %q", marker)
		}
	}
}

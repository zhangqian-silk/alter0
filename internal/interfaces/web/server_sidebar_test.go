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

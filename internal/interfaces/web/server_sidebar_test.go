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

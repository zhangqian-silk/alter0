package web

import (
	"strings"
	"testing"
)

func TestMobileNewChatEntryReachable(t *testing.T) {
	html := readEmbeddedAsset(t, "static/chat.html")
	htmlMarkers := []string{
		`id="mobileNewChatButton"`,
		`id="newChatButton"`,
	}
	for _, marker := range htmlMarkers {
		if !strings.Contains(html, marker) {
			t.Fatalf("expected html marker %q", marker)
		}
	}

	styles := readEmbeddedAsset(t, "static/assets/chat.css")
	styleMarkers := []string{
		".mobile-new-chat {",
		".mobile-new-chat,",
	}
	for _, marker := range styleMarkers {
		if !strings.Contains(styles, marker) {
			t.Fatalf("expected style marker %q", marker)
		}
	}
}

func TestNewChatReusesLatestBlankSessionAndSwitchesContext(t *testing.T) {
	script := readEmbeddedAsset(t, "static/assets/chat.js")
	markers := []string{
		"function startNewChatSession()",
		"function getLatestBlankSession(mode = routeConversationMode())",
		"function enforceSingleBlankSession(mode = routeConversationMode())",
		"const existingBlank = getLatestBlankSession(\"chat\");",
		"setActiveConversationSessionID(existingBlank.id, \"chat\");",
		"createSession(defaultChatTarget(), \"chat\");",
		`navigateToRoute("chat", { skipConfirm: true });`,
		"newChatButton.addEventListener(\"click\", () => {",
		"mobileNewChatButton.addEventListener(\"click\", () => {",
	}
	for _, marker := range markers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}
}

func TestSessionListShowsEmptyAndLoadFailureFeedback(t *testing.T) {
	html := readEmbeddedAsset(t, "static/chat.html")
	if !strings.Contains(html, `id="sessionLoadError"`) {
		t.Fatalf("expected html marker %q", `id="sessionLoadError"`)
	}

	script := readEmbeddedAsset(t, "static/assets/chat.js")
	scriptMarkers := []string{
		"function loadSessionsFromStorage(mode = routeConversationMode())",
		`setConversationSessionLoadError("session_save_failed", mode);`,
		"setConversationSessionLoadError(message, mode);",
		`sessionEmpty.textContent = isAgentConversationRoute() ? t("session.empty_agent") : t("session.empty");`,
	}
	for _, marker := range scriptMarkers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}

	styles := readEmbeddedAsset(t, "static/assets/chat.css")
	if !strings.Contains(styles, ".session-error {") {
		t.Fatalf("expected style marker %q", ".session-error {")
	}
}

func TestSessionDeleteHooksAndStylesPresent(t *testing.T) {
	script := readEmbeddedAsset(t, "static/assets/chat.js")
	scriptMarkers := []string{
		"function removeSession(sessionID)",
		`row.className = "session-card-row"`,
		`deleteButton.className = "session-card-delete"`,
		"removeSession(item.id);",
	}
	for _, marker := range scriptMarkers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}

	styles := readEmbeddedAsset(t, "static/assets/chat.css")
	styleMarkers := []string{
		".session-card-row {",
		".session-card-delete {",
	}
	for _, marker := range styleMarkers {
		if !strings.Contains(styles, marker) {
			t.Fatalf("expected style marker %q", marker)
		}
	}
}

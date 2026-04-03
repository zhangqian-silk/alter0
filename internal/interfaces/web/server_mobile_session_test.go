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
		"function getLatestBlankSession(route = state.currentRoute)",
		"function enforceSingleBlankSession(route = state.currentRoute)",
		"const existingBlank = getLatestBlankSession(\"chat\");",
		"setActiveConversationSessionID(existingBlank.id, \"chat\");",
		"createSession(routeDefaultTarget(\"chat\"), routeConversationMode(\"chat\"), \"chat\");",
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
		`sessionEmpty.textContent = routeAllowsTargetPicker() ? t("session.empty_agent") : t("session.empty");`,
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

func TestMobileRuntimeSheetUsesDedicatedStackingLayer(t *testing.T) {
	html := readEmbeddedAsset(t, "static/chat.html")
	if !strings.Contains(html, `id="chatRuntimeSheetHost"`) {
		t.Fatalf("expected html marker %q", `id="chatRuntimeSheetHost"`)
	}

	script := readEmbeddedAsset(t, "static/assets/chat.js")
	scriptMarkers := []string{
		`class="composer-runtime-sheet-backdrop"`,
		`class="composer-runtime-popover-mobile-close"`,
		`data-runtime-close`,
		`data-runtime-scroll-container="mobile"`,
		`role="dialog" aria-modal="true"`,
		`const chatRuntimeSheetHost = document.getElementById("chatRuntimeSheetHost");`,
		`function chatAgentOptionSummary(agent = {}) {`,
		`function captureChatRuntimeScrollState(popover = state.chatRuntime.openPopover) {`,
		`function restoreChatRuntimeScrollState(snapshot) {`,
		`restoreChatRuntimeScrollState(scrollSnapshot);`,
		`subtitle: chatAgentOptionSummary(agent)`,
		`chatRuntimeSheetHost.innerHTML = openPopover === "mobile" ? renderChatRuntimeCompactPopover({`,
		`appShell.classList.toggle("runtime-sheet-open", compactRuntime && openPopover === "mobile");`,
		`if (state.currentRoute === "chat" || state.currentRoute === "agent-runtime") {`,
	}
	for _, marker := range scriptMarkers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}

	styles := readEmbeddedAsset(t, "static/assets/chat.css")
	styleMarkers := []string{
		".app-shell.runtime-sheet-open {",
		".composer-runtime-sheet-backdrop {",
		".composer-runtime-popover-mobile-topbar {",
		".composer-runtime-popover-mobile-close {",
		".composer-runtime-popover-mobile-body {",
		".composer-runtime-option-summary {",
	}
	for _, marker := range styleMarkers {
		if !strings.Contains(styles, marker) {
			t.Fatalf("expected style marker %q", marker)
		}
	}
}

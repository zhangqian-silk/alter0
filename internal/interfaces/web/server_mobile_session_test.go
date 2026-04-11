package web

import (
	"strings"
	"testing"
)

func TestMobileNewChatEntryReachable(t *testing.T) {
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
		`sheetContentRoot.innerHTML = openPopover === "mobile" ? renderChatRuntimeCompactPopover({`,
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

func TestMobileViewportKeyboardOffsetOnlyAppliesForFocusedInput(t *testing.T) {
	script := readEmbeddedAsset(t, "static/assets/chat.js")
	markers := []string{
		"const MOBILE_KEYBOARD_MIN_OFFSET_PX = 120;",
		"const rawKeyboardOffset = activeInput",
		"? Math.max(0, state.mobileViewport.baselineHeight - effectiveHeight)",
		"const keyboardOffset = rawKeyboardOffset >= MOBILE_KEYBOARD_MIN_OFFSET_PX",
	}
	for _, marker := range markers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}
}

func TestMobileRuntimeSheetClosesWhenComposerTakesFocus(t *testing.T) {
	script := readEmbeddedAsset(t, "static/assets/chat.js")
	markers := []string{
		"onFocus: () => {",
		"if (isMobileViewport() && state.chatRuntime.openPopover) {",
		"closeChatRuntimePopover();",
	}
	for _, marker := range markers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}
}

func TestMobileRuntimeSheetBlursFocusedComposerBeforeOpen(t *testing.T) {
	script := readEmbeddedAsset(t, "static/assets/chat.js")
	markers := []string{
		"if (isTerminalSessionSheetViewport() && nextPopover === \"mobile\" && state.chatRuntime.openPopover !== nextPopover) {",
		"const activeInput = activeViewportInput();",
		"activeInput.blur();",
		"scheduleViewportInsetSync();",
	}
	for _, marker := range markers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}
}

func TestMobileTerminalComposerConsumesViewportInsetVariables(t *testing.T) {
	coreStyles := readEmbeddedAsset(t, "static/assets/chat-core.css")
	coreMarkers := []string{
		"--mobile-viewport-height: 100dvh;",
		"--keyboard-offset: 0px;",
	}
	for _, marker := range coreMarkers {
		if !strings.Contains(coreStyles, marker) {
			t.Fatalf("expected core style marker %q", marker)
		}
	}

	terminalStyles := readEmbeddedAsset(t, "static/assets/chat-terminal.css")
	terminalMarkers := []string{
		"height: min(100%, var(--mobile-viewport-height, 100dvh));",
		"padding: 0 10px calc(10px + env(safe-area-inset-bottom) + var(--keyboard-offset));",
	}
	for _, marker := range terminalMarkers {
		if !strings.Contains(terminalStyles, marker) {
			t.Fatalf("expected terminal style marker %q", marker)
		}
	}
}

func TestMobileRoutePagesConsumeViewportMetrics(t *testing.T) {
	styles := readEmbeddedAsset(t, "static/assets/chat-terminal.css")
	markers := []string{
		".chat-pane.page-mode {",
		"height: min(100%, calc(var(--mobile-viewport-height, 100dvh) + var(--keyboard-offset)));",
	}
	for _, marker := range markers {
		if !strings.Contains(styles, marker) {
			t.Fatalf("expected style marker %q", marker)
		}
	}
}

func TestNarrowPhoneTerminalWorkspaceAllowsActionWrap(t *testing.T) {
	styles := readEmbeddedAsset(t, "static/assets/chat-terminal.css")
	markers := []string{
		"@media (max-width: 420px) {",
		".terminal-workspace-row {",
		"flex-wrap: wrap;",
		".terminal-workspace-actions {",
		"justify-content: flex-start;",
		"-webkit-line-clamp: 2;",
	}
	for _, marker := range markers {
		if !strings.Contains(styles, marker) {
			t.Fatalf("expected style marker %q", marker)
		}
	}
}

func TestDesktopChatColumnExpandsOnWideViewports(t *testing.T) {
	styles := readEmbeddedAsset(t, "static/assets/chat-core.css")
	markers := []string{
		"--content-width: clamp(",
		"calc(100vw - var(--nav-width) - var(--session-width) - 160px)",
		"--user-message-max-width: min(80%, 880px);",
		"max-width: min(100%, var(--content-width));",
		"width: min(var(--content-width), calc(100% - 28px));",
		"width: min(var(--content-width), 100%);",
	}
	for _, marker := range markers {
		if !strings.Contains(styles, marker) {
			t.Fatalf("expected style marker %q", marker)
		}
	}
}

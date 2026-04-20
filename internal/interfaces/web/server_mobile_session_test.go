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
	script := readWorkspaceFile(t, "frontend/src/bootstrap/ReactRuntimeFacade.tsx")
	markers := []string{
		"const created: ChatSession = {",
		"const nextSessionsByRoute: SessionsState = {",
		"const nextActiveState = { ...activeSessionByRoute, [route]: created.id };",
		"window.location.hash = `#${route}`;",
		`createSession: () => {`,
		`window.__alter0LegacyRuntime?.createSession?.();`,
		`document.addEventListener(LEGACY_SHELL_CREATE_SESSION_EVENT, handleCreateSession);`,
	}
	for _, marker := range markers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}
}

func TestSessionListShowsEmptyAndLoadFailureFeedback(t *testing.T) {
	source := readWorkspaceFile(t, "frontend/src/features/shell/components/SessionPane.tsx")
	scriptMarkers := []string{
		"loadError: string;",
		`id="sessionLoadError"`,
		`role="status"`,
		`aria-live="polite"`,
		"loadError={sessionPaneSnapshot.loadError}",
	}
	for _, marker := range scriptMarkers {
		if !strings.Contains(source, marker) {
			t.Fatalf("expected source marker %q", marker)
		}
	}

	styles := readEmbeddedAsset(t, "static/assets/chat.css")
	if !strings.Contains(styles, ".session-error {") {
		t.Fatalf("expected style marker %q", ".session-error {")
	}
}

func TestSessionDeleteHooksAndStylesPresent(t *testing.T) {
	script := readWorkspaceFile(t, "frontend/src/bootstrap/ReactRuntimeFacade.tsx")
	scriptMarkers := []string{
		"const removeSession = async (sessionID: string) => {",
		"window.__alter0LegacyRuntime?.removeSession?.(sessionID);",
		"void removeSession(sessionID);",
		"document.addEventListener(LEGACY_SHELL_REMOVE_SESSION_EVENT, handleRemoveSession);",
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
	hostSource := readWorkspaceFile(t, "frontend/src/features/shell/components/ChatRuntimeHost.tsx")
	hostMarkers := []string{
		`className="composer-runtime-sheet-backdrop"`,
		`className="composer-runtime-popover-mobile-close"`,
		`data-runtime-close`,
		`data-runtime-scroll-container="mobile"`,
		`role="dialog"`,
		`if (!snapshot.compact || snapshot.openPopover !== "mobile") {`,
		`requestLegacyChatRuntimePopover("mobile")`,
	}
	for _, marker := range hostMarkers {
		if !strings.Contains(hostSource, marker) {
			t.Fatalf("expected runtime host marker %q", marker)
		}
	}

	facadeSource := readWorkspaceFile(t, "frontend/src/bootstrap/ReactRuntimeFacade.tsx")
	facadeMarkers := []string{
		`compact: compactRuntime,`,
		`appShell.classList.toggle(`,
		`"runtime-sheet-open"`,
	}
	for _, marker := range facadeMarkers {
		if !strings.Contains(facadeSource, marker) {
			t.Fatalf("expected runtime facade marker %q", marker)
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
	script := readWorkspaceFile(t, "frontend/src/shared/viewport/mobileViewport.ts")
	markers := []string{
		"export const MOBILE_KEYBOARD_MIN_OFFSET_PX = 120;",
		"const rawKeyboardOffset = input.hasActiveInput",
		"? Math.max(0, baselineHeight - effectiveHeight)",
		"const keyboardOffset = rawKeyboardOffset >= MOBILE_KEYBOARD_MIN_OFFSET_PX",
	}
	for _, marker := range markers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}
}

func TestMobileRuntimeSheetClosesWhenComposerTakesFocus(t *testing.T) {
	script := readWorkspaceFile(t, "frontend/src/bootstrap/ReactRuntimeFacade.tsx")
	markers := []string{
		"const handleFocus = () => {",
		"if (compactRuntime && runtimeOpenPopover) {",
		`setRuntimeOpenPopover("");`,
	}
	for _, marker := range markers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}
}

func TestMobileRuntimeSheetBlursFocusedComposerBeforeOpen(t *testing.T) {
	script := readWorkspaceFile(t, "frontend/src/bootstrap/ReactRuntimeFacade.tsx")
	markers := []string{
		"const activeInput = activeViewportInput();",
		"activeInput?.blur();",
		`nextPopover === "mobile"`,
		"if (compactRuntime && runtimeOpenPopover) {",
		`inputNode.addEventListener("focus", handleFocus);`,
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

func TestNarrowTerminalWorkspaceHidesDuplicateSessionToggle(t *testing.T) {
	styles := readEmbeddedAsset(t, "static/assets/chat-terminal.css")
	markers := []string{
		"@media (max-width: 1100px) {",
		".terminal-workspace-actions [data-terminal-session-pane-toggle] {",
		"display: none;",
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

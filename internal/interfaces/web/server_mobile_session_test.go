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

func TestConversationRuntimeCreatesAndDeletesTerminalBackedSessions(t *testing.T) {
	source := readWorkspaceFile(t, "frontend/src/features/conversation-runtime/ConversationRuntimeProvider.tsx") +
		readWorkspaceFile(t, "frontend/src/features/shell/components/runtimeSessionApi.ts") +
		readWorkspaceFile(t, "frontend/src/features/conversation-runtime/ConversationWorkspace.tsx")
	markers := []string{
		`export function runtimeSessionEndpoint(`,
		`runtimeSessionEndpoint("chat", path, query)`,
		"const createTerminalRuntimeSession = useCallback(async (routeKey: ConversationRoute, title: string = \"\"): Promise<ChatSession | null> => {",
		"chatTerminalSessionEndpoint(),",
		"upsertRuntimeSession(routeKey, nextSession);",
		"const nextSessionsByRoute: SessionsState = {",
		"setActiveSessionByRoute((current) => {",
		"const nextActiveState = { ...current, [routeKey]: nextSession.id };",
		"createSession: () => {",
		"void createTerminalRuntimeSession(route);",
		"const removeSession = useCallback(async (sessionID: string) => {",
		"await apiClient.delete(chatTerminalSessionEndpoint(encodeURIComponent(sessionID)));",
		"const handleCreateSession = useCallback(() => {",
		"runtime.createSession();",
		"const handleRemoveSession = useCallback((sessionID: string) => {",
		"return runtime.removeSession(sessionID);",
		"onSessionPanePrimaryAction: handleCreateSession,",
		"onMobilePrimary: handleCreateSession,",
		"const isDraft = Boolean(item.draft);",
		"onDelete: isDraft ? undefined : () => void handleRemoveSession(item.id),",
	}
	for _, marker := range markers {
		if !strings.Contains(source, marker) {
			t.Fatalf("expected source marker %q", marker)
		}
	}
}

func TestConversationSessionListShowsTitleOnlyRowsWithBusyLoading(t *testing.T) {
	source := readWorkspaceFile(t, "frontend/src/features/conversation-runtime/ConversationWorkspace.tsx") +
		readWorkspaceFile(t, "frontend/src/features/shell/components/RuntimeWorkspacePage.tsx") +
		readWorkspaceFile(t, "frontend/src/features/shell/components/RuntimeSessionList.tsx")
	markers := []string{
		`data-runtime-session-pane`,
		`"data-testid": "conversation-session-pane"`,
		`"runtime-session-list"`,
		`"runtime-session-title"`,
		`"runtime-session-loading"`,
		`data-runtime-session-loading`,
		`item.statusTone === "busy"`,
		"runtime.sessionItems.length",
	}
	for _, marker := range markers {
		if !strings.Contains(source, marker) {
			t.Fatalf("expected source marker %q", marker)
		}
	}
	forbiddenMarkers := []string{
		`"runtime-session-meta"`,
		`"runtime-session-hash"`,
		`"runtime-session-summary-row"`,
		`{item.shortHash}`,
	}
	for _, marker := range forbiddenMarkers {
		if strings.Contains(source, marker) {
			t.Fatalf("unexpected source marker %q", marker)
		}
	}

	styles := readWorkspaceFile(t, "frontend/src/styles/shell.css")
	styleMarkers := []string{
		".runtime-session-card {",
		".runtime-session-card.is-active {",
		".runtime-session-delete {",
		".runtime-session-loading {",
		"@keyframes runtime-session-loading-spin {",
	}
	for _, marker := range styleMarkers {
		if !strings.Contains(styles, marker) {
			t.Fatalf("expected style marker %q", marker)
		}
	}
}

func TestConversationDesktopSessionPaneConstrainsHeightForScroll(t *testing.T) {
	styles := readWorkspaceFile(t, "frontend/src/styles/shell.css")
	markers := []string{
		"[data-runtime-view=\"conversation\"],\n[data-runtime-view=\"terminal\"] {\n  min-height: 100%;\n  height: 100%;",
		".runtime-workspace-session-pane {\n  background: transparent;\n  min-height: 0;\n  height: 100%;",
		".runtime-workspace-session-pane-shell {\n  min-height: 0;\n  height: 100%;",
		".runtime-workspace {\n  padding: 8px 0 18px;\n  min-width: 0;\n  max-width: 100%;",
		".runtime-workspace-body {\n  --runtime-composer-inset: 0px;\n  min-width: 0;\n  max-width: 100%;",
	}
	for _, marker := range markers {
		if !strings.Contains(styles, marker) {
			t.Fatalf("expected style marker %q", marker)
		}
	}
}

func TestConversationDetailsUseSharedWorkspaceHeader(t *testing.T) {
	source := readWorkspaceFile(t, "frontend/src/features/conversation-runtime/ConversationRuntimeProvider.tsx") +
		readWorkspaceFile(t, "frontend/src/features/conversation-runtime/ConversationWorkspace.tsx") +
		readWorkspaceFile(t, "frontend/src/features/shell/components/RuntimeWorkspaceHeader.tsx")
	markers := []string{
		"toggleInspector: (tab) => {",
		`data-runtime-workspace-header="true"`,
		`headerProps: { "data-runtime-header-kind": "conversation" },`,
		`data-runtime-details-panel`,
		`detailsSummary: conversationDetailsSummary,`,
		`detailsClassName: "conversation-inspector conversation-session-details workspace-details-content"`,
	}
	for _, marker := range markers {
		if !strings.Contains(source, marker) {
			t.Fatalf("expected source marker %q", marker)
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

func TestWorkbenchMobileNavOverlayStylesPresent(t *testing.T) {
	source := readWorkspaceFile(t, "frontend/src/app/WorkbenchApp.tsx")
	sourceMarkers := []string{
		"const [isMobileViewport, setIsMobileViewport] = useState(() => isLegacyShellMobileViewport());",
		`const [mobilePanel, setMobilePanel] = useState<"nav" | "sessions" | null>(null);`,
		"const runtimeSessionsUseNav = Boolean(visibleSessionRail);",
		`const navOpen = mobilePanel === "nav";`,
		`const sessionPaneOpen = !runtimeSessionsUseNav && mobilePanel === "sessions";`,
		`classNames.push("nav-open", "overlay-open")`,
		"if (!mobile) {",
		"setMobilePanel(null);",
		"if (isMobileViewport) {",
	}
	for _, marker := range sourceMarkers {
		if !strings.Contains(source, marker) {
			t.Fatalf("expected source marker %q", marker)
		}
	}

	styles := readWorkspaceFile(t, "frontend/src/styles/shell.css")
	styleMarkers := []string{
		".app-shell.nav-open .primary-nav {",
		".app-shell.overlay-open .mobile-backdrop {",
		".mobile-backdrop {",
	}
	for _, marker := range styleMarkers {
		if !strings.Contains(styles, marker) {
			t.Fatalf("expected style marker %q", marker)
		}
	}
}

func TestWorkbenchMobileLayoutUsesConversationDrawer(t *testing.T) {
	styles := readWorkspaceFile(t, "frontend/src/styles/shell.css")
	markers := []string{
		"@media (max-width: 1100px) {",
		`[data-runtime-view="conversation"],`,
		`[data-runtime-view="terminal"] {`,
		"grid-template-columns: 1fr;",
		".runtime-workspace-session-pane {",
		"position: fixed;",
		"width: min(88vw, 340px);",
		".runtime-workspace-session-pane.is-open {",
		"transform: translateX(0);",
	}
	for _, marker := range markers {
		if !strings.Contains(styles, marker) {
			t.Fatalf("expected style marker %q", marker)
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
		"position: fixed;",
		"bottom: var(--keyboard-offset);",
		"padding: 0 10px calc(10px + env(safe-area-inset-bottom));",
		"gap: 6px;",
		"padding: var(--terminal-chat-screen-padding-top) var(--terminal-chat-screen-padding-x) 20px;",
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
		".app-shell.info-mode {",
		"height: calc(var(--mobile-viewport-height, 100dvh) + var(--keyboard-offset));",
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
		".terminal-mobile-header {",
		".terminal-mobile-header-actions {",
		".terminal-workspace-actions [data-terminal-session-pane-toggle] {",
		"display: none;",
	}
	for _, marker := range markers {
		if !strings.Contains(styles, marker) {
			t.Fatalf("expected style marker %q", marker)
		}
	}
}

func TestTerminalMobileActionsLinkWorkbenchNavAndSessionDrawer(t *testing.T) {
	source := readWorkspaceFile(t, "frontend/src/features/shell/components/ReactManagedTerminalRouteBody.tsx")
	markers := []string{
		`const workbench = useWorkbenchContext();`,
		`const shellCopy = getLegacyShellCopy(workbench.language);`,
		`mobileHeaderProps: { "data-runtime-mobile-variant": "terminal" },`,
		`mobileNavButtonClassName: "is-quiet conversation-mobile-nav-toggle",`,
		`mobileNavButtonProps: { "aria-expanded": workbench.mobileNavOpen },`,
		`onMobileNav: workbench.toggleMobileNav,`,
		`mobilePrimaryButtonClassName: "is-primary conversation-mobile-new-session",`,
		`mobilePrimaryButtonProps: {`,
		`"data-runtime-create-session": "terminal",`,
		`"data-runtime-mobile-primary": "terminal",`,
		`mobileNavButtonLabel: shellCopy.chatMenu,`,
		`mobilePrimaryButtonLabel: copy.newShort,`,
	}
	for _, marker := range markers {
		if !strings.Contains(source, marker) {
			t.Fatalf("expected source marker %q", marker)
		}
	}
	forbiddenMarkers := []string{
		`mobileSessionButtonClassName: "is-quiet conversation-mobile-session-toggle",`,
		`mobileSessionButtonProps: { "aria-expanded": workbench.mobileSessionPaneOpen },`,
		`const toggleMobileSessionPane = () => {`,
		`onMobileSession: toggleMobileSessionPane,`,
		`mobileSessionButtonLabel: copy.sessions,`,
	}
	for _, marker := range forbiddenMarkers {
		if strings.Contains(source, marker) {
			t.Fatalf("unexpected source marker %q", marker)
		}
	}
}

func TestTerminalRouteKeepsDedicatedScrollShell(t *testing.T) {
	styles := readWorkspaceFile(t, "frontend/src/styles/shell.css") +
		readWorkspaceFile(t, "frontend/public/legacy/chat-terminal.css")
	markers := []string{
		".route-view.terminal-route {",
		"flex-direction: column;",
		".route-body.terminal-route-body {",
		"display: flex;",
		"overflow: hidden;",
		".terminal-chat-screen {",
		"overflow-y: auto;",
		"-webkit-overflow-scrolling: touch;",
		"touch-action: pan-y;",
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

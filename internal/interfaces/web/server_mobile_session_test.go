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

func TestConversationRuntimeCreatesAndDeletesChatRuntimeBackedSessions(t *testing.T) {
	source := readWorkspaceFile(t, "frontend/src/features/conversation-runtime/ConversationRuntimeProvider.tsx") +
		readWorkspaceFile(t, "frontend/src/features/shell/components/runtimeSessionApi.ts") +
		readWorkspaceFile(t, "frontend/src/features/conversation-runtime/ConversationWorkspace.tsx")
	markers := []string{
		`export function runtimeSessionEndpoint(`,
		`useRuntimeSessionController`,
		`createRuntimeSession(`,
		`refreshRuntimeSession(sessionID`,
		`sendRuntimeInput(session.id`,
		"const createRuntimeBackedSession = useCallback(async (routeKey: ConversationRoute, title: string = \"\"): Promise<ChatSession | null> => {",
		"upsertRuntimeSession(routeKey, nextSession);",
		"const nextSessionsByRoute: SessionsState = {",
		"setActiveSessionByRoute((current) => {",
		"const nextActiveState = { ...current, [routeKey]: nextSession.id };",
		"createSession: () => {",
		"void createRuntimeBackedSession(route);",
		"const removeSession = useCallback(async (sessionID: string) => {",
		`await deleteRuntimeSession(sessionID);`,
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
		"[data-runtime-view=\"conversation\"],\n[data-runtime-view=\"chatRuntime\"] {\n  min-height: 100%;\n  height: 100%;",
		".runtime-workspace-session-pane {\n  background: transparent;\n  min-height: 0;\n  height: 100%;",
		".runtime-workspace-session-pane-shell {\n  min-height: 0;\n  height: 100%;",
		".runtime-workspace {\n  padding: 8px 0 18px;\n  min-width: 0;\n  max-width: 100%;",
		".runtime-workspace-body {\n  min-width: 0;\n  max-width: 100%;",
	}
	for _, marker := range markers {
		if !strings.Contains(styles, marker) {
			t.Fatalf("expected style marker %q", marker)
		}
	}
	if strings.Contains(styles, "--runtime-composer-inset") {
		t.Fatal("runtime workspace should not rely on the legacy composer inset variable")
	}
}

func TestConversationDetailsUseSharedWorkspaceHeader(t *testing.T) {
	source := readWorkspaceFile(t, "frontend/src/features/conversation-runtime/ConversationRuntimeProvider.tsx") +
		readWorkspaceFile(t, "frontend/src/features/conversation-runtime/ConversationWorkspace.tsx") +
		readWorkspaceFile(t, "frontend/src/features/shell/components/RuntimeWorkspaceHeader.tsx")
	markers := []string{
		"const [sessionDetailsOpen, setSessionDetailsOpen] = useState(false);",
		"onToggleDetails: activeSessionIsDraft ? () => undefined : () => setSessionDetailsOpen((current) => !current),",
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
		".app-shell.nav-open .primary-nav,",
		".workbench-mobile-overlay-portal.nav-open .primary-nav {",
		".app-shell.overlay-open .mobile-backdrop,",
		".workbench-mobile-overlay-portal.overlay-open .mobile-backdrop {",
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
		`[data-runtime-view="chatRuntime"] {`,
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

func TestMobileRoutePagesConsumeViewportMetrics(t *testing.T) {
	styles := readEmbeddedAsset(t, "static/assets/chat-runtime.css")
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

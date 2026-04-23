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

func TestConversationRuntimeCreatesAndDeletesSessionsInReactState(t *testing.T) {
	source := readWorkspaceFile(t, "frontend/src/features/conversation-runtime/ConversationRuntimeProvider.tsx") +
		readWorkspaceFile(t, "frontend/src/features/conversation-runtime/ConversationWorkspace.tsx")
	markers := []string{
		"const created: ChatSession = {",
		"const nextSessionsByRoute: SessionsState = {",
		"const nextActiveState = { ...preferredActiveState, [route]: created.id };",
		"createSession: () => {",
		"ensureSession(null, { ...activeSessionByRoute, [route]: \"\" });",
		"const removeSession = async (sessionID: string) => {",
		"const handleCreateSession = () => {",
		"runtime.createSession();",
		"const handleRemoveSession = (sessionID: string) => {",
		"return runtime.removeSession(sessionID);",
		"onClick={handleCreateSession}",
		"onClick={() => void handleRemoveSession(item.id)}",
	}
	for _, marker := range markers {
		if !strings.Contains(source, marker) {
			t.Fatalf("expected source marker %q", marker)
		}
	}
}

func TestConversationSessionListShowsCompactMetadata(t *testing.T) {
	source := readWorkspaceFile(t, "frontend/src/features/conversation-runtime/ConversationWorkspace.tsx")
	markers := []string{
		`data-conversation-session-pane`,
		`"data-testid": "conversation-session-pane"`,
		`className="conversation-session-list menu"`,
		`className="conversation-session-title"`,
		`className="conversation-session-meta"`,
		`className="conversation-session-hash"`,
		`className="conversation-session-bottomline"`,
		`{item.shortHash}`,
		"runtime.sessionItems.length",
	}
	for _, marker := range markers {
		if !strings.Contains(source, marker) {
			t.Fatalf("expected source marker %q", marker)
		}
	}

	styles := readWorkspaceFile(t, "frontend/src/styles/shell.css")
	styleMarkers := []string{
		".conversation-session-card {",
		".conversation-session-card.is-active {",
		".conversation-session-delete {",
	}
	for _, marker := range styleMarkers {
		if !strings.Contains(styles, marker) {
			t.Fatalf("expected style marker %q", marker)
		}
	}
}

func TestConversationInspectorUsesInlinePanel(t *testing.T) {
	source := readWorkspaceFile(t, "frontend/src/features/conversation-runtime/ConversationRuntimeProvider.tsx") +
		readWorkspaceFile(t, "frontend/src/features/conversation-runtime/ConversationWorkspace.tsx")
	markers := []string{
		"const [inspectorOpen, setInspectorOpen] = useState(false);",
		"toggleInspector: (tab) => {",
		"setInspectorOpen((current) => (tab ? true : !current));",
		`className="conversation-inspector"`,
		`data-conversation-inspector`,
		"onClick={() => runtime.closeInspector()}",
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
		`const navOpen = mobilePanel === "nav";`,
		`const sessionPaneOpen = mobilePanel === "sessions";`,
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
		".conversation-runtime-view {",
		"grid-template-columns: 1fr;",
		".conversation-session-pane {",
		"position: fixed;",
		"width: min(88vw, 340px);",
		".conversation-session-pane.is-open {",
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
		`className="terminal-mobile-header" data-terminal-mobile-header`,
		`className="terminal-mobile-header-actions"`,
		`aria-expanded={workbench.mobileNavOpen}`,
		`onClick={workbench.toggleMobileNav}`,
		`aria-expanded={workbench.mobileSessionPaneOpen}`,
		`onClick={workbench.toggleMobileSessionPane}`,
		`{shellCopy.chatMenu}`,
		`{copy.sessions}`,
		`{copy.newShort}`,
	}
	for _, marker := range markers {
		if !strings.Contains(source, marker) {
			t.Fatalf("expected source marker %q", marker)
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

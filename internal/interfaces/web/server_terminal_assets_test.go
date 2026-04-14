package web

import (
	"strings"
	"testing"
)

func TestTerminalHelperModulesExposeExpectedMarkers(t *testing.T) {
	files := map[string][]string{
		"static/assets/chat-terminal-session.js": {
			"function renderTerminalStatus(status) {",
			"function renderTerminalSessionCards(sessions, activeSessionID, options = {}) {",
			"function renderTerminalWorkspaceMetaPanel(session) {",
		},
		"static/assets/chat-terminal-process.js": {
			"function renderTerminalRichBlock(stepID, block, searchQuery = \"\") {",
			"function renderTerminalTurns(session) {",
			"function renderTerminalFinalOutput(session, turn) {",
		},
		"static/assets/chat-terminal-patch.js": {
			"function syncRenderedBlock(parent, selector, html, anchorNode = null, position = \"beforeend\") {",
			"function patchTerminalSessionPane(container, sessions, activeSessionID, mobileSessionListOpen, options = {}) {",
			"function patchTerminalWorkspaceNode(container, session, sending, closing = false, deleting = false, options = {}) {",
		},
		"static/assets/chat-terminal-shell.js": {
			"function renderTerminalViewShell(sessions, activeSessionID, session, sending, closing = false, deleting = false, options = {}) {",
		},
	}

	for path, markers := range files {
		content := readEmbeddedAsset(t, path)
		for _, marker := range markers {
			if !strings.Contains(content, marker) {
				t.Fatalf("expected %s to contain marker %q", path, marker)
			}
		}
	}
}

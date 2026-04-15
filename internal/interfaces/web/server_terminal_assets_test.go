package web

import (
	"strings"
	"testing"
)

func TestTerminalHelperModulesExposeExpectedMarkers(t *testing.T) {
	files := map[string][]string{
		"static/assets/chat.js": {
			"patchTerminalSessionPane(container, localState.sessions, localState.activeSessionID, sessionSheetOpen, {",
			"patchTerminalWorkspaceNode(container, active, localState.sending, localState.closing, localState.deleting, {",
			"renderTerminalSessionCards(localState.sessions, localState.activeSessionID, {",
			"renderTerminalWorkspace(active, localState.sending, localState.closing, localState.deleting, {",
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

package web

import (
	"strings"
	"testing"
)

func TestTerminalHelperModulesExposeExpectedMarkers(t *testing.T) {
	files := map[string][]string{
		"frontend/src/features/shell/components/ReactManagedTerminalRouteBody.tsx": {
			"data-runtime-session-pane",
			"data-runtime-session-list",
			"data-runtime-workspace",
			"data-runtime-screen",
			"data-terminal-step-toggle",
			"data-terminal-final-output",
		},
	}

	for path, markers := range files {
		content := readWorkspaceFile(t, path)
		for _, marker := range markers {
			if !strings.Contains(content, marker) {
				t.Fatalf("expected %s to contain marker %q", path, marker)
			}
		}
	}
}

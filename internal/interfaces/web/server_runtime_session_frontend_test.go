package web

import (
	"strings"
	"testing"
)

func TestChatAndTerminalShareRuntimeSessionController(t *testing.T) {
	controller := readWorkspaceFile(t, "frontend/src/features/shell/components/runtimeSessionController.ts")
	catalogs := readWorkspaceFile(t, "frontend/src/features/shell/components/runtimeSessionCatalogs.ts")
	viewModel := readWorkspaceFile(t, "frontend/src/features/shell/components/runtimeSessionViewModel.ts")
	chat := readWorkspaceFile(t, "frontend/src/features/conversation-runtime/ConversationRuntimeProvider.tsx")
	terminal := readWorkspaceFile(t, "frontend/src/features/shell/components/ReactManagedTerminalRouteBody.tsx")

	controllerMarkers := []string{
		"export function useRuntimeSessionController",
		"route: RuntimeSessionRoute",
		"runtimeSessionDetailEndpoint(options.route",
		"runtimeSessionInputEndpoint(options.route",
		"RUNTIME_SESSION_HISTORY_PAGE_TURN_LIMIT",
	}
	for _, marker := range controllerMarkers {
		if !strings.Contains(controller, marker) {
			t.Fatalf("expected shared runtime session controller marker %q", marker)
		}
	}
	for _, marker := range []string{
		"export function useRuntimeSessionCatalogs(",
		`"/api/control/llm/providers"`,
		`"/api/control/skills"`,
		`"/api/control/mcps"`,
	} {
		if !strings.Contains(catalogs, marker) {
			t.Fatalf("expected shared runtime catalog marker %q", marker)
		}
	}
	for _, marker := range []string{
		"export type RuntimeSessionViewSession",
		"export function mergeRuntimeSessionViewSession",
		"export function runtimeSessionTurnsToTimelineMessages",
		"export function runtimeTimelineMessageTurnID",
	} {
		if !strings.Contains(viewModel, marker) {
			t.Fatalf("expected shared runtime session view model marker %q", marker)
		}
	}

	importMarkers := []string{"useRuntimeSessionController", "useRuntimeSessionCatalogs", "runtimeSessionTurnsToTimelineMessages"}
	for _, source := range []struct {
		name string
		body string
	}{
		{name: "chat", body: chat},
		{name: "terminal", body: terminal},
	} {
		for _, importMarker := range importMarkers {
			if !strings.Contains(source.body, importMarker) {
				t.Fatalf("expected %s runtime page to consume shared %s", source.name, importMarker)
			}
		}
	}
}

func TestConversationRuntimeMigratesLegacySnapshotsToRuntimeCache(t *testing.T) {
	chat := readWorkspaceFile(t, "frontend/src/features/conversation-runtime/ConversationRuntimeProvider.tsx")

	for _, marker := range []string{
		"loadLegacySessionSnapshots",
		"clearLegacySessionSnapshots",
		"writeLongTermConversationRuntimeCache(activeSessionByRoute, snapshotLoad.sessionsByRoute)",
		"writeSessionInfoConversationRuntimeCache(activeSessionByRoute, snapshotLoad.sessionsByRoute)",
	} {
		if !strings.Contains(chat, marker) {
			t.Fatalf("expected legacy snapshot migration marker %q", marker)
		}
	}
	for _, forbidden := range []string{
		"persistActiveSessionSnapshots",
		"writeJSONStorage(ACTIVE_SESSION_SNAPSHOT_STORAGE_KEY",
		"writeJSONStorage(RECENT_SESSION_SNAPSHOT_STORAGE_KEY",
	} {
		if strings.Contains(chat, forbidden) {
			t.Fatalf("did not expect legacy snapshot write marker %q", forbidden)
		}
	}
}

func TestRuntimePagesDoNotForkSessionStateMachines(t *testing.T) {
	controller := readWorkspaceFile(t, "frontend/src/features/shell/components/runtimeSessionController.ts")
	timeline := readWorkspaceFile(t, "frontend/src/features/shell/components/ChatMessageRegion.tsx")
	viewModel := readWorkspaceFile(t, "frontend/src/features/shell/components/runtimeSessionViewModel.ts")
	chat := readWorkspaceFile(t, "frontend/src/features/conversation-runtime/ConversationRuntimeProvider.tsx")
	chatWorkspace := readWorkspaceFile(t, "frontend/src/features/conversation-runtime/ConversationWorkspace.tsx")
	terminal := readWorkspaceFile(t, "frontend/src/features/shell/components/ReactManagedTerminalRouteBody.tsx")

	for _, marker := range []string{
		"resolveRuntimeSessionPollPlan",
		"progressiveHistoryLoadsRef",
		"refreshActiveSession",
	} {
		if !strings.Contains(controller, marker) {
			t.Fatalf("expected shared controller to own session state marker %q", marker)
		}
	}
	if !strings.Contains(timeline, "buildRuntimeSessionTimelineItems") {
		t.Fatalf("expected shared runtime timeline builder")
	}
	for _, marker := range []string{
		"mergeRuntimeSessionTurns",
		"mergeRuntimeSessionTurnPaging",
		"runtimeSessionTurnsToTimelineMessages",
	} {
		if !strings.Contains(viewModel, marker) {
			t.Fatalf("expected shared runtime view model to own %q", marker)
		}
	}
	if !strings.Contains(chatWorkspace, "buildRuntimeSessionTimelineItems") {
		t.Fatalf("expected Chat workspace to consume shared runtime timeline builder")
	}
	if !strings.Contains(terminal, "buildRuntimeSessionTimelineItems") {
		t.Fatalf("expected Terminal workspace to consume shared runtime timeline builder")
	}
	if strings.Contains(terminal, "buildTerminalTimelineItems") {
		t.Fatalf("did not expect Terminal to keep a private timeline item builder")
	}

	for _, source := range []struct {
		name string
		body string
	}{
		{name: "chat", body: chat},
		{name: "terminal", body: terminal},
	} {
		for _, forbidden := range []string{
			"manageState: false",
			"enableProgressiveHistory: false",
			"progressiveHistoryLoadsRef",
			"progressiveHistoryLoadedRef",
			"buildTerminalTimelineItems",
			"terminalTurnsToRuntimeTimelineMessages",
			"mergeTerminalTurns",
			"mergeTerminalTurnPaging",
			"terminalTurnRuntimeEvents",
			"compareTerminalTurns",
		} {
			if strings.Contains(source.body, forbidden) {
				t.Fatalf("did not expect %s page to fork runtime session state with %q", source.name, forbidden)
			}
		}
	}
}

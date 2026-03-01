package executor

import "testing"

func TestListExecutorCapabilitiesIncludesKnownExecutors(t *testing.T) {
	caps := listExecutorCapabilities(func(name string) bool {
		return name == "codex"
	})

	if len(caps) != 2 {
		t.Fatalf("expected 2 capabilities, got %d", len(caps))
	}

	if caps[0].Name != "claude_code" {
		t.Fatalf("unexpected first capability name: %s", caps[0].Name)
	}
	if caps[0].Command != "claude" {
		t.Fatalf("unexpected first capability command: %s", caps[0].Command)
	}
	if caps[0].Installed {
		t.Fatalf("expected claude_code installed=false")
	}
	if len(caps[0].Aliases) != 1 || caps[0].Aliases[0] != "claude" {
		t.Fatalf("unexpected aliases for claude_code: %#v", caps[0].Aliases)
	}

	if caps[1].Name != "codex" {
		t.Fatalf("unexpected second capability name: %s", caps[1].Name)
	}
	if caps[1].Command != "codex" {
		t.Fatalf("unexpected second capability command: %s", caps[1].Command)
	}
	if !caps[1].Installed {
		t.Fatalf("expected codex installed=true")
	}
}

func TestListExecutorCapabilitiesNilLookupDefaultsToFalse(t *testing.T) {
	caps := listExecutorCapabilities(nil)
	for _, cap := range caps {
		if cap.Installed {
			t.Fatalf("expected installed=false for %s", cap.Name)
		}
	}
}

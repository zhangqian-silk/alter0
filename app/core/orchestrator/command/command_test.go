package command

import "testing"

func TestAuthorizeCommand_AdminOnlyCommands(t *testing.T) {
	exec := NewExecutor(nil, nil, []string{"admin_user"})

	if err := exec.authorizeCommand("guest", []string{"executor"}); err == nil {
		t.Fatalf("expected /executor to require admin")
	}
	if err := exec.authorizeCommand("guest", []string{"config", "set", "executor.name", "codex"}); err == nil {
		t.Fatalf("expected /config set to require admin")
	}
	if err := exec.authorizeCommand("guest", []string{"config", "get"}); err != nil {
		t.Fatalf("expected /config get to be allowed: %v", err)
	}
	if err := exec.authorizeCommand("admin_user", []string{"executor"}); err != nil {
		t.Fatalf("expected admin to pass /executor check: %v", err)
	}
}

func TestSetAdminUsers_TrimsAndDedupes(t *testing.T) {
	exec := NewExecutor(nil, nil, nil)
	exec.SetAdminUsers([]string{"  alice ", "alice", "", "bob"})

	if !exec.isAdminUser("alice") {
		t.Fatalf("alice should be admin")
	}
	if !exec.isAdminUser("bob") {
		t.Fatalf("bob should be admin")
	}
	if exec.isAdminUser("carol") {
		t.Fatalf("carol should not be admin")
	}
}

func TestFormatAuditCommandLine_DefaultsAndReason(t *testing.T) {
	line := formatAuditCommandLine("", "", "", "config set executor.name codex", "deny", "permission denied")
	expected := "[AUDIT] user=anonymous channel=unknown request=n/a decision=deny command=\"config set executor.name codex\" reason=\"permission denied\""
	if line != expected {
		t.Fatalf("unexpected audit line:\n got: %s\nwant: %s", line, expected)
	}
}

func TestFormatAuditCommandLine_WithoutReason(t *testing.T) {
	line := formatAuditCommandLine("u1", "cli", "req-1", "help", "allow", "")
	expected := "[AUDIT] user=u1 channel=cli request=req-1 decision=allow command=\"help\""
	if line != expected {
		t.Fatalf("unexpected audit line:\n got: %s\nwant: %s", line, expected)
	}
}

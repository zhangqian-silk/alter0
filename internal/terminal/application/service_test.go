package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	terminaldomain "alter0/internal/terminal/domain"
)

func TestResolveCodexCommandUsesDefaultCommand(t *testing.T) {
	command := resolveCodexCommand(Options{})

	if command.path != defaultCodexCommand {
		t.Fatalf("expected default codex command, got %q", command.path)
	}
	if command.label != "codex exec" {
		t.Fatalf("expected codex exec label, got %q", command.label)
	}
}

func TestBuildCodexTurnArgsUsesResumeWhenThreadExists(t *testing.T) {
	command := resolveCodexCommand(Options{
		Shell:     "codex.exe",
		ShellArgs: []string{"--profile", "test"},
	})

	args := buildCodexTurnArgs(command, "thread-123", "reply exactly")

	got := strings.Join(args, " ")
	for _, part := range []string{"--profile", "test", "exec", "resume", "--json", "--skip-git-repo-check", "thread-123", "reply exactly"} {
		if !strings.Contains(got, part) {
			t.Fatalf("expected args to contain %q, got %v", part, args)
		}
	}
}

func TestNormalizeOptionsParsesShellArgsLine(t *testing.T) {
	options := normalizeOptions(Options{
		Shell:         "bash",
		ShellArgsLine: `"./fixtures/codex mock.sh" --profile test`,
	})

	expected := []string{"./fixtures/codex mock.sh", "--profile", "test"}
	if strings.Join(options.ShellArgs, "|") != strings.Join(expected, "|") {
		t.Fatalf("expected parsed shell args %v, got %v", expected, options.ShellArgs)
	}
}

func TestCreateAssignsSessionWorkspaceDir(t *testing.T) {
	baseDir := t.TempDir()
	service := NewService(context.Background(), nil, nil, Options{WorkingDir: baseDir})

	session, err := service.Create(CreateRequest{OwnerID: "owner-workspace"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	expected := filepath.Join(baseDir, ".alter0", "workspaces", "terminal", "sessions", session.ID)
	if filepath.Clean(session.WorkingDir) != filepath.Clean(expected) {
		t.Fatalf("expected workspace %q, got %q", expected, session.WorkingDir)
	}
	info, statErr := os.Stat(session.WorkingDir)
	if statErr != nil {
		t.Fatalf("stat workspace dir: %v", statErr)
	}
	if !info.IsDir() {
		t.Fatalf("expected workspace directory, got file")
	}
}

func TestRecoverAssignsDeterministicWorkspaceDir(t *testing.T) {
	baseDir := t.TempDir()
	service := NewService(context.Background(), nil, nil, Options{WorkingDir: baseDir})

	session, err := service.Recover(RecoverRequest{
		OwnerID:   "owner-recover",
		SessionID: "terminal-recover",
	})
	if err != nil {
		t.Fatalf("recover session: %v", err)
	}

	expected := filepath.Join(baseDir, ".alter0", "workspaces", "terminal", "sessions", "terminal-recover")
	if filepath.Clean(session.WorkingDir) != filepath.Clean(expected) {
		t.Fatalf("expected recovered workspace %q, got %q", expected, session.WorkingDir)
	}
}

func TestServiceInputStartsAndResumesCodexSession(t *testing.T) {
	service := newTestService("success")

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-a",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if _, err := service.Input("owner-a", session.ID, "first prompt"); err != nil {
		t.Fatalf("first input: %v", err)
	}

	firstSnapshot, firstEntries := waitForSessionEntries(t, service, "owner-a", session.ID, 2)
	if firstSnapshot.TerminalSessionID != "thread-first-prompt" {
		t.Fatalf("expected runtime thread id, got %q", firstSnapshot.TerminalSessionID)
	}
	if firstSnapshot.Status != terminaldomain.SessionStatusRunning {
		t.Fatalf("expected running after first turn, got %q", firstSnapshot.Status)
	}
	if got := firstEntries[1].Text; got != "mock:first prompt" {
		t.Fatalf("expected first reply, got %q", got)
	}

	if _, err := service.Input("owner-a", session.ID, "second prompt"); err != nil {
		t.Fatalf("second input: %v", err)
	}

	secondSnapshot, secondEntries := waitForSessionEntries(t, service, "owner-a", session.ID, 4)
	if secondSnapshot.TerminalSessionID != "thread-first-prompt" {
		t.Fatalf("expected resumed thread id, got %q", secondSnapshot.TerminalSessionID)
	}
	if got := secondEntries[3].Text; got != "mock:second prompt" {
		t.Fatalf("expected second reply, got %q", got)
	}
}

func TestServiceRecoverRestoresCodexThreadForFollowUpInput(t *testing.T) {
	service := newTestService("success")

	session, err := service.Recover(RecoverRequest{
		OwnerID:           "owner-recover",
		SessionID:         "terminal-recover",
		TerminalSessionID: "thread-recovered",
		Title:             "terminal-recover",
		CreatedAt:         time.Date(2026, 3, 19, 10, 0, 0, 0, time.UTC),
		UpdatedAt:         time.Date(2026, 3, 19, 10, 5, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("recover session: %v", err)
	}
	if session.TerminalSessionID != "thread-recovered" {
		t.Fatalf("expected recovered thread id, got %q", session.TerminalSessionID)
	}

	if _, err := service.Input("owner-recover", session.ID, "follow-up prompt"); err != nil {
		t.Fatalf("recovered input: %v", err)
	}

	snapshot, entries := waitForSessionEntries(t, service, "owner-recover", session.ID, 2)
	if snapshot.TerminalSessionID != "thread-recovered" {
		t.Fatalf("expected recovered runtime thread id, got %q", snapshot.TerminalSessionID)
	}
	if got := entries[1].Text; got != "mock:follow-up prompt" {
		t.Fatalf("expected resumed reply, got %q", got)
	}
}

func TestServiceInputResumesExitedSession(t *testing.T) {
	service := newTestService("success")

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-exit",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if _, err := service.Input("owner-exit", session.ID, "first prompt"); err != nil {
		t.Fatalf("first input: %v", err)
	}
	firstSnapshot, _ := waitForSessionEntries(t, service, "owner-exit", session.ID, 2)
	if firstSnapshot.TerminalSessionID != "thread-first-prompt" {
		t.Fatalf("expected first thread id, got %q", firstSnapshot.TerminalSessionID)
	}

	closed, err := service.Close("owner-exit", session.ID)
	if err != nil {
		t.Fatalf("close session: %v", err)
	}
	if closed.Status != terminaldomain.SessionStatusExited {
		t.Fatalf("expected exited status after close, got %q", closed.Status)
	}

	if _, err := service.Input("owner-exit", session.ID, "resume prompt"); err != nil {
		t.Fatalf("resume input: %v", err)
	}

	resumedSnapshot, resumedEntries := waitForSessionEntries(t, service, "owner-exit", session.ID, 5)
	if resumedSnapshot.TerminalSessionID != "thread-first-prompt" {
		t.Fatalf("expected resumed thread id, got %q", resumedSnapshot.TerminalSessionID)
	}
	if got := resumedEntries[4].Text; got != "mock:resume prompt" {
		t.Fatalf("expected resumed reply, got %q", got)
	}
}

func TestServiceLoadsPersistedSessionsAfterRestart(t *testing.T) {
	baseDir := t.TempDir()
	service := newTestServiceWithBaseDir("success", baseDir)

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-restart",
		Title:   "persisted-session",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := service.Input("owner-restart", session.ID, "first prompt"); err != nil {
		t.Fatalf("first input: %v", err)
	}
	firstSnapshot, _ := waitForSessionEntries(t, service, "owner-restart", session.ID, 2)
	if firstSnapshot.TerminalSessionID != "thread-first-prompt" {
		t.Fatalf("expected persisted thread id, got %q", firstSnapshot.TerminalSessionID)
	}

	restarted := newTestServiceWithBaseDir("success", baseDir)
	restored, ok := restarted.Get("owner-restart", session.ID)
	if !ok {
		t.Fatalf("expected restored session after restart")
	}
	if restored.Title != "persisted-session" {
		t.Fatalf("expected restored title, got %q", restored.Title)
	}
	if restored.TerminalSessionID != "thread-first-prompt" {
		t.Fatalf("expected restored thread id, got %q", restored.TerminalSessionID)
	}

	if _, err := restarted.Input("owner-restart", session.ID, "after restart"); err != nil {
		t.Fatalf("restart input: %v", err)
	}
	snapshot, entries := waitForSessionEntries(t, restarted, "owner-restart", session.ID, 4)
	if snapshot.TerminalSessionID != "thread-first-prompt" {
		t.Fatalf("expected resumed thread id after restart, got %q", snapshot.TerminalSessionID)
	}
	if got := entries[3].Text; got != "mock:after restart" {
		t.Fatalf("expected resumed reply after restart, got %q", got)
	}
}

func TestServiceInputRejectsConcurrentTurns(t *testing.T) {
	service := newTestService("sleep")

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-b",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if _, err := service.Input("owner-b", session.ID, "long prompt"); err != nil {
		t.Fatalf("first input: %v", err)
	}
	if _, err := service.Input("owner-b", session.ID, "second prompt"); !errors.Is(err, ErrSessionBusy) {
		t.Fatalf("expected busy error, got %v", err)
	}
}

func TestServiceInputFailsFastOnCodexAuthError(t *testing.T) {
	service := newTestService("auth-error")

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-auth",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	startedAt := time.Now()
	if _, err := service.Input("owner-auth", session.ID, "hello"); err != nil {
		t.Fatalf("input: %v", err)
	}

	snapshot, entries := waitForSessionError(t, service, "owner-auth", session.ID)
	if !strings.Contains(snapshot.ErrorMessage, "codex authentication failed") {
		t.Fatalf("expected auth failure in session error, got %q", snapshot.ErrorMessage)
	}
	if elapsed := time.Since(startedAt); elapsed > 2*time.Second {
		t.Fatalf("expected fast auth failure, got %s", elapsed)
	}
	found := false
	for _, entry := range entries {
		if strings.Contains(entry.Text, "codex request failed: codex authentication failed") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected auth failure entry, got %+v", entries)
	}
}

func TestServiceListPrefersLastOutputAtOverUpdatedAt(t *testing.T) {
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	service := &Service{
		sessions: map[string]*runtimeSession{
			"terminal-output-newer": {
				summary: terminaldomain.Session{
					ID:           "terminal-output-newer",
					OwnerID:      "owner-a",
					CreatedAt:    now.Add(-10 * time.Minute),
					LastOutputAt: now.Add(-2 * time.Minute),
					UpdatedAt:    now.Add(-4 * time.Minute),
				},
			},
			"terminal-updated-newer": {
				summary: terminaldomain.Session{
					ID:           "terminal-updated-newer",
					OwnerID:      "owner-a",
					CreatedAt:    now.Add(-9 * time.Minute),
					LastOutputAt: now.Add(-3 * time.Minute),
					UpdatedAt:    now.Add(-1 * time.Minute),
				},
			},
		},
	}

	items := service.List("owner-a")
	if len(items) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(items))
	}
	if items[0].ID != "terminal-output-newer" {
		t.Fatalf("expected last output ordering, got first session %q", items[0].ID)
	}
}

func TestRuntimeSessionAppendEntryLockedUpdatesLastOutputAtOnlyForRealOutput(t *testing.T) {
	session := &runtimeSession{
		summary: terminaldomain.Session{
			ID:        "terminal-output-flags",
			OwnerID:   "owner-a",
			CreatedAt: time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC),
		},
	}

	session.appendEntryLocked("system", "session ready")
	if !session.summary.LastOutputAt.IsZero() {
		t.Fatalf("expected system entry to keep last_output_at empty, got %s", session.summary.LastOutputAt)
	}

	session.appendEntryLocked("input", "prompt")
	if !session.summary.LastOutputAt.IsZero() {
		t.Fatalf("expected input entry to keep last_output_at empty, got %s", session.summary.LastOutputAt)
	}

	session.appendEntryLocked("stdout", "assistant output")
	if session.summary.LastOutputAt.IsZero() {
		t.Fatalf("expected stdout entry to update last_output_at")
	}

	lastOutputAt := session.summary.LastOutputAt
	session.appendEntryLocked("stderr", "warning")
	if session.summary.LastOutputAt.IsZero() || session.summary.LastOutputAt.Before(lastOutputAt) {
		t.Fatalf("expected stderr entry to preserve or advance last_output_at")
	}
}

func newTestService(mode string) *Service {
	baseDir, err := os.MkdirTemp("", "alter0-terminal-service-test-*")
	if err != nil {
		panic(err)
	}
	return newTestServiceWithBaseDir(mode, baseDir)
}

func newTestServiceWithBaseDir(mode string, baseDir string) *Service {
	service := NewService(context.Background(), nil, nil, Options{WorkingDir: baseDir})
	service.runner = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		cmdArgs := append([]string{"-test.run=TestTerminalServiceHelperProcess", "--", name}, args...)
		cmd := exec.CommandContext(ctx, os.Args[0], cmdArgs...)
		cmd.Env = append(
			os.Environ(),
			"GO_WANT_TERMINAL_HELPER_PROCESS=1",
			"TERMINAL_HELPER_MODE="+mode,
		)
		return cmd
	}
	return service
}

func waitForSessionEntries(t *testing.T, service *Service, ownerID string, sessionID string, want int) (terminaldomain.Session, []terminaldomain.Entry) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		snapshot, ok := service.Get(ownerID, sessionID)
		if !ok {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		page, err := service.ListEntries(ownerID, sessionID, 0, 32)
		if err != nil {
			t.Fatalf("list entries: %v", err)
		}
		if len(page.Items) >= want && snapshot.Status == terminaldomain.SessionStatusRunning {
			return snapshot, page.Items
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d terminal entries", want)
	return terminaldomain.Session{}, nil
}

func waitForSessionError(t *testing.T, service *Service, ownerID string, sessionID string) (terminaldomain.Session, []terminaldomain.Entry) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		snapshot, ok := service.Get(ownerID, sessionID)
		if !ok {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		page, err := service.ListEntries(ownerID, sessionID, 0, 32)
		if err != nil {
			t.Fatalf("list entries: %v", err)
		}
		if strings.TrimSpace(snapshot.ErrorMessage) != "" && len(page.Items) >= 2 {
			return snapshot, page.Items
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("timed out waiting for terminal auth failure")
	return terminaldomain.Session{}, nil
}

func TestTerminalServiceHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_TERMINAL_HELPER_PROCESS") != "1" {
		return
	}

	separatorIndex := -1
	for i, arg := range os.Args {
		if arg == "--" {
			separatorIndex = i
			break
		}
	}
	if separatorIndex < 0 || separatorIndex+1 >= len(os.Args) {
		os.Exit(2)
	}

	forwarded := os.Args[separatorIndex+1:]
	if len(forwarded) < 2 || forwarded[0] != defaultCodexCommand {
		os.Exit(2)
	}
	execIndex := -1
	for index, arg := range forwarded {
		if arg == "exec" {
			execIndex = index
			break
		}
	}
	if execIndex < 1 {
		os.Exit(2)
	}
	terminalArgs := forwarded[execIndex+1:]

	mode := os.Getenv("TERMINAL_HELPER_MODE")
	if mode == "sleep" {
		time.Sleep(300 * time.Millisecond)
	}

	resumeIndex := -1
	for index, arg := range terminalArgs {
		if arg == "resume" {
			resumeIndex = index
			break
		}
	}
	if resumeIndex >= 0 {
		if len(terminalArgs) < resumeIndex+4 {
			os.Exit(2)
		}
		threadID := terminalArgs[len(terminalArgs)-2]
		prompt := terminalArgs[len(terminalArgs)-1]
		fmt.Fprintf(os.Stdout, "{\"type\":\"thread.started\",\"thread_id\":%q}\n", threadID)
		fmt.Fprintln(os.Stdout, `{"type":"turn.started"}`)
		fmt.Fprintf(os.Stdout, "{\"type\":\"item.completed\",\"item\":{\"id\":\"item_0\",\"type\":\"agent_message\",\"text\":%q}}\n", "mock:"+prompt)
		fmt.Fprintln(os.Stdout, `{"type":"turn.completed"}`)
		os.Exit(0)
	}

	prompt := terminalArgs[len(terminalArgs)-1]
	threadID := "thread-" + strings.ReplaceAll(prompt, " ", "-")
	fmt.Fprintf(os.Stdout, "{\"type\":\"thread.started\",\"thread_id\":%q}\n", threadID)
	fmt.Fprintln(os.Stdout, `{"type":"turn.started"}`)
	if mode == "auth-error" {
		fmt.Fprintln(os.Stdout, `{"type":"error","message":"Reconnecting... 1/5 (unexpected status 401 Unauthorized: Missing bearer or basic authentication in header)"}`)
		time.Sleep(5 * time.Second)
		os.Exit(19)
	}
	fmt.Fprintf(os.Stdout, "{\"type\":\"item.completed\",\"item\":{\"id\":\"item_0\",\"type\":\"agent_message\",\"text\":%q}}\n", "mock:"+prompt)
	fmt.Fprintln(os.Stdout, `{"type":"turn.completed"}`)
	os.Exit(0)
}

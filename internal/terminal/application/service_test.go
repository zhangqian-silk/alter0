package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
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

	expected := []string{"--profile", "test", "exec", "resume", "--json", "--skip-git-repo-check", "thread-123", "reply exactly"}
	if strings.Join(args, " ") != strings.Join(expected, " ") {
		t.Fatalf("unexpected args: %v", args)
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
	service := NewService(context.Background(), nil, nil, Options{
		WorkingDir: "D:/GitHubRepositories/alter0",
	})
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
	if len(forwarded) < 2 || forwarded[0] != defaultCodexCommand || forwarded[1] != "exec" {
		os.Exit(2)
	}

	mode := os.Getenv("TERMINAL_HELPER_MODE")
	if mode == "sleep" {
		time.Sleep(300 * time.Millisecond)
	}

	if len(forwarded) >= 3 && forwarded[2] == "resume" {
		if len(forwarded) < 7 {
			os.Exit(2)
		}
		threadID := forwarded[len(forwarded)-2]
		prompt := forwarded[len(forwarded)-1]
		fmt.Fprintf(os.Stdout, "{\"type\":\"thread.started\",\"thread_id\":%q}\n", threadID)
		fmt.Fprintln(os.Stdout, `{"type":"turn.started"}`)
		fmt.Fprintf(os.Stdout, "{\"type\":\"item.completed\",\"item\":{\"id\":\"item_0\",\"type\":\"agent_message\",\"text\":%q}}\n", "mock:"+prompt)
		fmt.Fprintln(os.Stdout, `{"type":"turn.completed"}`)
		os.Exit(0)
	}

	prompt := forwarded[len(forwarded)-1]
	threadID := "thread-" + strings.ReplaceAll(prompt, " ", "-")
	fmt.Fprintf(os.Stdout, "{\"type\":\"thread.started\",\"thread_id\":%q}\n", threadID)
	fmt.Fprintln(os.Stdout, `{"type":"turn.started"}`)
	fmt.Fprintf(os.Stdout, "{\"type\":\"item.completed\",\"item\":{\"id\":\"item_0\",\"type\":\"agent_message\",\"text\":%q}}\n", "mock:"+prompt)
	fmt.Fprintln(os.Stdout, `{"type":"turn.completed"}`)
	os.Exit(0)
}

package runtime

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"alter0/app/core/agent"
	"alter0/app/core/queue"
)

func TestSnapshotIncludesExecutorCapabilities(t *testing.T) {
	collector := &StatusCollector{RepoPath: "."}

	snap := collector.Snapshot(context.Background())
	raw, ok := snap["executors"]
	if !ok {
		t.Fatalf("expected executors in snapshot")
	}

	executors, ok := raw.([]agent.ExecutorCapability)
	if !ok {
		t.Fatalf("expected executors to be []agent.ExecutorCapability, got %T", raw)
	}
	if len(executors) != 2 {
		t.Fatalf("expected 2 executor capabilities, got %d", len(executors))
	}

	for _, item := range executors {
		if item.Name == "" {
			t.Fatalf("executor entry missing name: %#v", item)
		}
		if item.Command == "" {
			t.Fatalf("executor entry missing command: %#v", item)
		}
	}
}

func TestSnapshotIncludesCommandAuditTail(t *testing.T) {
	baseDir := t.TempDir()
	dayDir := filepath.Join(baseDir, "2026-02-28")
	if err := os.MkdirAll(dayDir, 0755); err != nil {
		t.Fatalf("failed to create audit dir: %v", err)
	}

	lines := "" +
		"{\"timestamp\":\"2026-02-28T18:00:00Z\",\"user_id\":\"u1\",\"channel_id\":\"cli\",\"request_id\":\"r1\",\"command\":\"help\",\"decision\":\"allow\"}\n" +
		"{\"timestamp\":\"2026-02-28T18:01:00Z\",\"user_id\":\"u1\",\"channel_id\":\"cli\",\"request_id\":\"r2\",\"command\":\"status\",\"decision\":\"allow\"}\n" +
		"{\"timestamp\":\"2026-02-28T18:02:00Z\",\"user_id\":\"u2\",\"channel_id\":\"http\",\"request_id\":\"r3\",\"command\":\"executor\",\"decision\":\"deny\",\"reason\":\"permission denied\"}\n"
	if err := os.WriteFile(filepath.Join(dayDir, "command_permission.jsonl"), []byte(lines), 0644); err != nil {
		t.Fatalf("failed to seed audit log: %v", err)
	}

	collector := &StatusCollector{
		RepoPath:             ".",
		CommandAuditBasePath: baseDir,
		CommandAuditTailSize: 2,
	}

	snap := collector.Snapshot(context.Background())
	raw, ok := snap["command_audit_tail"]
	if !ok {
		t.Fatalf("expected command_audit_tail in snapshot")
	}

	tail, ok := raw.([]commandAuditTailEntry)
	if !ok {
		t.Fatalf("expected command_audit_tail to be []commandAuditTailEntry, got %T", raw)
	}
	if len(tail) != 2 {
		t.Fatalf("expected 2 audit tail entries, got %d", len(tail))
	}
	if tail[0].RequestID != "r2" || tail[1].RequestID != "r3" {
		t.Fatalf("expected newest 2 entries in chronological order, got %#v", tail)
	}
	if tail[1].Reason != "permission denied" {
		t.Fatalf("expected reason in latest entry, got %#v", tail[1])
	}
}

func TestSnapshotIncludesQueueInFlight(t *testing.T) {
	q := queue.New(2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := q.Start(ctx, 1); err != nil {
		t.Fatalf("start queue failed: %v", err)
	}
	defer q.Stop(200 * time.Millisecond)

	started := make(chan struct{}, 1)
	release := make(chan struct{})
	if _, err := q.Enqueue(queue.Job{Run: func(context.Context) error {
		started <- struct{}{}
		<-release
		return nil
	}}); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	select {
	case <-started:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected queue job to start")
	}

	collector := &StatusCollector{Queue: q}
	snap := collector.Snapshot(context.Background())
	raw, ok := snap["queue"]
	if !ok {
		t.Fatal("expected queue stats in snapshot")
	}

	stats, ok := raw.(queue.Stats)
	if !ok {
		t.Fatalf("expected queue stats type, got %T", raw)
	}
	if stats.Workers != 1 {
		t.Fatalf("expected workers=1, got %+v", stats)
	}
	if stats.InFlight != 1 {
		t.Fatalf("expected in_flight=1, got %+v", stats)
	}

	close(release)
}

func TestSnapshotIncludesQueueLastShutdownReport(t *testing.T) {
	q := queue.New(2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := q.Start(ctx, 1); err != nil {
		t.Fatalf("start queue failed: %v", err)
	}
	if _, err := q.Enqueue(queue.Job{Run: func(context.Context) error { return nil }}); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	if _, err := q.StopWithReport(200 * time.Millisecond); err != nil {
		t.Fatalf("stop queue failed: %v", err)
	}

	collector := &StatusCollector{Queue: q}
	snap := collector.Snapshot(context.Background())
	raw, ok := snap["queue"]
	if !ok {
		t.Fatal("expected queue stats in snapshot")
	}

	stats, ok := raw.(queue.Stats)
	if !ok {
		t.Fatalf("expected queue stats type, got %T", raw)
	}
	if stats.LastShutdown == nil {
		t.Fatalf("expected last_shutdown report in queue stats, got %+v", stats)
	}
	if stats.LastShutdown.TimedOut {
		t.Fatalf("expected graceful shutdown report, got %+v", stats.LastShutdown)
	}
}

func TestSnapshotIncludesAgentRegistry(t *testing.T) {
	collector := &StatusCollector{
		AgentEntries: []AgentEntry{
			{AgentID: "beta", Workspace: "/tmp/w2", AgentDir: "/tmp/a2", Executor: "codex"},
			{AgentID: "alpha", Workspace: "/tmp/w1", AgentDir: "/tmp/a1", Executor: "claude_code"},
		},
	}

	snap := collector.Snapshot(context.Background())
	raw, ok := snap["agents"]
	if !ok {
		t.Fatal("expected agents in snapshot")
	}
	items, ok := raw.([]AgentEntry)
	if !ok {
		t.Fatalf("expected []AgentEntry, got %T", raw)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(items))
	}
	if items[0].AgentID != "alpha" || items[1].AgentID != "beta" {
		t.Fatalf("expected sorted agents [alpha beta], got %+v", items)
	}
}

func TestSnapshotIncludesGitDivergenceAgainstUpstream(t *testing.T) {
	base := t.TempDir()
	remote := filepath.Join(base, "remote.git")
	runGitCmd(t, base, "git", "init", "--bare", remote)

	seed := filepath.Join(base, "seed")
	runGitCmd(t, base, "git", "clone", remote, seed)
	writeRepoFile(t, seed, "README.md", "seed\n")
	runGitCmd(t, seed, "git", "add", "README.md")
	runGitCmd(t, seed, "git", "commit", "-m", "seed")
	runGitCmd(t, seed, "git", "push", "origin", "master")

	local := filepath.Join(base, "local")
	runGitCmd(t, base, "git", "clone", remote, local)
	writeRepoFile(t, local, "local.txt", "local\n")
	runGitCmd(t, local, "git", "add", "local.txt")
	runGitCmd(t, local, "git", "commit", "-m", "local commit")

	other := filepath.Join(base, "other")
	runGitCmd(t, base, "git", "clone", remote, other)
	writeRepoFile(t, other, "remote.txt", "remote\n")
	runGitCmd(t, other, "git", "add", "remote.txt")
	runGitCmd(t, other, "git", "commit", "-m", "remote commit")
	runGitCmd(t, other, "git", "push", "origin", "master")

	runGitCmd(t, local, "git", "fetch", "origin")

	collector := &StatusCollector{RepoPath: local}
	snap := collector.Snapshot(context.Background())
	raw, ok := snap["git"]
	if !ok {
		t.Fatal("expected git status in snapshot")
	}

	git, ok := raw.(map[string]interface{})
	if !ok {
		t.Fatalf("expected git status map, got %T", raw)
	}
	if git["upstream"] != "origin/master" {
		t.Fatalf("expected upstream origin/master, got %#v", git["upstream"])
	}
	if git["ahead"] != 1 {
		t.Fatalf("expected ahead=1, got %#v", git["ahead"])
	}
	if git["behind"] != 1 {
		t.Fatalf("expected behind=1, got %#v", git["behind"])
	}
}

func runGitCmd(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=alter0-test",
		"GIT_AUTHOR_EMAIL=alter0-test@example.com",
		"GIT_COMMITTER_NAME=alter0-test",
		"GIT_COMMITTER_EMAIL=alter0-test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed (%s %s): %v\n%s", name, strings.Join(args, " "), err, string(out))
	}
}

func writeRepoFile(t *testing.T, repoPath, relPath, content string) {
	t.Helper()
	absPath := filepath.Join(repoPath, relPath)
	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
}

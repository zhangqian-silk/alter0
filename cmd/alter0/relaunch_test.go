package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncRemoteMasterBranchRejectsTrackedChangesAndPreservesFiles(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git not available: %v", err)
	}

	baseDir := t.TempDir()
	remoteDir := filepath.Join(baseDir, "remote.git")
	seedDir := filepath.Join(baseDir, "seed")
	localDir := filepath.Join(baseDir, "local")

	runGitCommand(t, baseDir, "init", "--bare", "--initial-branch=master", remoteDir)
	runGitCommand(t, baseDir, "clone", remoteDir, seedDir)
	configureTestGitIdentity(t, seedDir)

	trackedPath := filepath.Join(seedDir, "tracked.txt")
	writeTestFile(t, trackedPath, "initial\n")
	runGitCommand(t, seedDir, "add", "tracked.txt")
	runGitCommand(t, seedDir, "commit", "-m", "initial")
	runGitCommand(t, seedDir, "push", "origin", "master")

	runGitCommand(t, baseDir, "clone", remoteDir, localDir)
	configureTestGitIdentity(t, localDir)

	writeTestFile(t, trackedPath, "remote-update\n")
	runGitCommand(t, seedDir, "commit", "-am", "remote update")
	runGitCommand(t, seedDir, "push", "origin", "master")

	localTrackedPath := filepath.Join(localDir, "tracked.txt")
	writeTestFile(t, localTrackedPath, "local-staged\n")
	runGitCommand(t, localDir, "add", "tracked.txt")
	writeTestFile(t, localTrackedPath, "local-unstaged\n")

	untrackedPath := filepath.Join(localDir, "scratch", "notes.txt")
	writeTestFile(t, untrackedPath, "keep me\n")

	err := syncRemoteMasterBranch(localDir, false, "")
	if err == nil {
		t.Fatal("expected sync remote master to reject tracked working tree changes")
	}
	if !strings.Contains(err.Error(), "requires discard confirmation") {
		t.Fatalf("expected tracked changes error, got %v", err)
	}

	if got := readTestFile(t, localTrackedPath); got != "local-unstaged\n" {
		t.Fatalf("expected tracked file preserved, got %q", got)
	}
	if got := readTestFile(t, untrackedPath); got != "keep me\n" {
		t.Fatalf("expected untracked file preserved, got %q", got)
	}
}

func TestSyncRemoteMasterBranchDiscardsTrackedChangesAfterConfirmation(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git not available: %v", err)
	}

	baseDir := t.TempDir()
	remoteDir := filepath.Join(baseDir, "remote.git")
	seedDir := filepath.Join(baseDir, "seed")
	localDir := filepath.Join(baseDir, "local")

	runGitCommand(t, baseDir, "init", "--bare", "--initial-branch=master", remoteDir)
	runGitCommand(t, baseDir, "clone", remoteDir, seedDir)
	configureTestGitIdentity(t, seedDir)

	trackedPath := filepath.Join(seedDir, "tracked.txt")
	writeTestFile(t, trackedPath, "initial\n")
	runGitCommand(t, seedDir, "add", "tracked.txt")
	runGitCommand(t, seedDir, "commit", "-m", "initial")
	runGitCommand(t, seedDir, "push", "origin", "master")

	runGitCommand(t, baseDir, "clone", remoteDir, localDir)
	configureTestGitIdentity(t, localDir)

	writeTestFile(t, trackedPath, "remote-update\n")
	runGitCommand(t, seedDir, "commit", "-am", "remote update")
	runGitCommand(t, seedDir, "push", "origin", "master")

	localTrackedPath := filepath.Join(localDir, "tracked.txt")
	writeTestFile(t, localTrackedPath, "local-unstaged\n")

	untrackedPath := filepath.Join(localDir, "scratch", "notes.txt")
	writeTestFile(t, untrackedPath, "keep me\n")

	if err := syncRemoteMasterBranch(localDir, true, ""); err != nil {
		t.Fatalf("sync remote master branch failed after confirmation: %v", err)
	}

	if got := readTestFile(t, localTrackedPath); got != "remote-update\n" {
		t.Fatalf("expected tracked file reset to remote content, got %q", got)
	}
	if got := readTestFile(t, untrackedPath); got != "keep me\n" {
		t.Fatalf("expected untracked file preserved, got %q", got)
	}
}

func TestListRuntimeRestartCandidatesFetchesAllNewerCommitsAndCurrentHistory(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git not available: %v", err)
	}

	baseDir := t.TempDir()
	remoteDir := filepath.Join(baseDir, "remote.git")
	seedDir := filepath.Join(baseDir, "seed")
	localDir := filepath.Join(baseDir, "local")

	runGitCommand(t, baseDir, "init", "--bare", "--initial-branch=master", remoteDir)
	runGitCommand(t, baseDir, "clone", remoteDir, seedDir)
	configureTestGitIdentity(t, seedDir)

	trackedPath := filepath.Join(seedDir, "tracked.txt")
	writeTestFile(t, trackedPath, "commit-00\n")
	runGitCommand(t, seedDir, "add", "tracked.txt")
	runGitCommand(t, seedDir, "commit", "-m", "commit 00")

	for i := 1; i <= 12; i++ {
		writeTestFile(t, trackedPath, fmt.Sprintf("commit-%02d\n", i))
		runGitCommand(t, seedDir, "commit", "-am", fmt.Sprintf("commit %02d", i))
		if i == 10 {
			runGitCommand(t, seedDir, "push", "origin", "master")
			runGitCommand(t, baseDir, "clone", remoteDir, localDir)
			configureTestGitIdentity(t, localDir)
		}
	}
	currentCommit := runGitCommand(t, localDir, "rev-parse", "HEAD")
	runGitCommand(t, seedDir, "push", "origin", "master")

	candidates, err := listRuntimeRestartCandidates(localDir, currentCommit)
	if err != nil {
		t.Fatalf("list restart candidates failed: %v", err)
	}
	if candidates.CurrentCommit != currentCommit {
		t.Fatalf("expected current commit %q, got %q", currentCommit, candidates.CurrentCommit)
	}
	if len(candidates.Items) != 13 {
		t.Fatalf("expected all newer commits plus current commit and 10 historical candidates, got %d", len(candidates.Items))
	}
	if candidates.Items[0].Message != "commit 11" || candidates.Items[1].Message != "commit 12" {
		t.Fatalf("expected newer commits to be listed before current history, got first items %+v and %+v", candidates.Items[0], candidates.Items[1])
	}
	if !candidates.Items[2].Current || candidates.Items[2].Hash != currentCommit {
		t.Fatalf("expected current history window to start with current commit, got %+v", candidates.Items[2])
	}
	if candidates.Items[2].ShortHash != currentCommit[:7] {
		t.Fatalf("expected short current hash %q, got %q", currentCommit[:7], candidates.Items[2].ShortHash)
	}
	if candidates.Items[12].Message != "commit 00" {
		t.Fatalf("expected history window to include 10 commits before current, got %+v", candidates.Items[12])
	}
}

func TestSyncRemoteMasterBranchCanResetToSelectedMasterCommit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git not available: %v", err)
	}

	baseDir := t.TempDir()
	remoteDir := filepath.Join(baseDir, "remote.git")
	seedDir := filepath.Join(baseDir, "seed")
	localDir := filepath.Join(baseDir, "local")

	runGitCommand(t, baseDir, "init", "--bare", "--initial-branch=master", remoteDir)
	runGitCommand(t, baseDir, "clone", remoteDir, seedDir)
	configureTestGitIdentity(t, seedDir)

	trackedPath := filepath.Join(seedDir, "tracked.txt")
	writeTestFile(t, trackedPath, "initial\n")
	runGitCommand(t, seedDir, "add", "tracked.txt")
	runGitCommand(t, seedDir, "commit", "-m", "initial")
	runGitCommand(t, seedDir, "push", "origin", "master")

	runGitCommand(t, baseDir, "clone", remoteDir, localDir)
	configureTestGitIdentity(t, localDir)

	writeTestFile(t, trackedPath, "selected\n")
	runGitCommand(t, seedDir, "commit", "-am", "selected update")
	selectedCommit := runGitCommand(t, seedDir, "rev-parse", "HEAD")
	writeTestFile(t, trackedPath, "latest\n")
	runGitCommand(t, seedDir, "commit", "-am", "latest update")
	runGitCommand(t, seedDir, "push", "origin", "master")

	if err := syncRemoteMasterBranch(localDir, true, selectedCommit[:8]); err != nil {
		t.Fatalf("sync selected master commit failed: %v", err)
	}

	if got := runGitCommand(t, localDir, "rev-parse", "HEAD"); got != selectedCommit {
		t.Fatalf("expected HEAD reset to selected commit %q, got %q", selectedCommit, got)
	}
	if branch := runGitCommand(t, localDir, "branch", "--show-current"); branch != "master" {
		t.Fatalf("expected to remain on master branch, got %q", branch)
	}
	if got := readTestFile(t, filepath.Join(localDir, "tracked.txt")); got != "selected\n" {
		t.Fatalf("expected selected commit content, got %q", got)
	}
}

func configureTestGitIdentity(t *testing.T, repoDir string) {
	t.Helper()
	runGitCommand(t, repoDir, "config", "user.name", "Alter0 Tests")
	runGitCommand(t, repoDir, "config", "user.email", "alter0-tests@example.com")
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create parent dir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}
	return string(content)
}

func runGitCommand(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output))
}

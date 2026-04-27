package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncRemoteMasterBranchDiscardsTrackedChangesButPreservesUntrackedFiles(t *testing.T) {
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

	if err := syncRemoteMasterBranch(localDir); err != nil {
		t.Fatalf("sync remote master branch failed: %v", err)
	}

	if got := readTestFile(t, localTrackedPath); got != "remote-update\n" {
		t.Fatalf("expected tracked file reset to remote content, got %q", got)
	}
	if got := readTestFile(t, untrackedPath); got != "keep me\n" {
		t.Fatalf("expected untracked file preserved, got %q", got)
	}

	status := runGitCommand(t, localDir, "status", "--porcelain")
	lines := strings.Fields(status)
	if len(lines) != 2 || lines[0] != "??" || lines[1] != "scratch/" {
		t.Fatalf("expected only untracked scratch directory to remain, got %q", status)
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

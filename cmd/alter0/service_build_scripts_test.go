package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServiceBuildScriptsUseUnifiedFrontendAwareBuild(t *testing.T) {
	repoRoot := findRepositoryRoot(t)
	buildScriptPath := filepath.Join(repoRoot, "scripts/build_alter0_service.sh")
	buildScript := readRepositoryFile(t, repoRoot, "scripts/build_alter0_service.sh")
	startScript := readRepositoryFile(t, repoRoot, "scripts/start_alter0_service.sh")
	relaunchScript := readRepositoryFile(t, repoRoot, "scripts/relaunch_service.sh")
	authSetupScript := readRepositoryFile(t, repoRoot, "scripts/setup_alter0_runtime_auth.sh")
	nodeSetupScript := readRepositoryFile(t, repoRoot, "scripts/setup_alter0_runtime_node.sh")
	makefile := readRepositoryFile(t, repoRoot, "Makefile")
	buildScriptInfo, err := os.Stat(buildScriptPath)
	if err != nil {
		t.Fatalf("stat %s: %v", buildScriptPath, err)
	}

	assertContains(t, buildScript, "npm run build")
	assertContains(t, buildScript, "go build -o")
	assertContains(t, buildScript, "internal/interfaces/web/static/dist/index.html")
	if buildScriptInfo.Mode()&0o111 == 0 {
		t.Fatalf("%s must be executable", buildScriptPath)
	}

	assertContains(t, startScript, "scripts/build_alter0_service.sh")
	assertContains(t, relaunchScript, "scripts/build_alter0_service.sh")
	assertContains(t, relaunchScript, "git merge --ff-only")
	assertNotContains(t, startScript, "go build -o")
	assertNotContains(t, relaunchScript, "go build -o")
	assertNotContains(t, relaunchScript, "git reset --hard")
	assertNotContains(t, startScript, "-daily-memory-dir")
	assertNotContains(t, startScript, "-long-term-memory-path")
	assertNotContains(t, startScript, "-mandatory-context-file")
	assertNotContains(t, startScript, "ALTER0_STORAGE_DIR")
	for _, script := range map[string]string{
		"start_alter0_service.sh":      startScript,
		"relaunch_service.sh":          relaunchScript,
		"setup_alter0_runtime_auth.sh": authSetupScript,
		"setup_alter0_runtime_node.sh": nodeSetupScript,
	} {
		assertNotContains(t, script, "/srv/alter0")
		assertNotContains(t, script, "/var/lib/alter0")
		assertNotContains(t, script, "should run as")
		assertNotContains(t, script, "runuser -u")
	}
	assertContains(t, makefile, "build:")
	assertContains(t, makefile, "scripts/build_alter0_service.sh")
}

func findRepositoryRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above working directory")
		}
		dir = parent
	}
}

func readRepositoryFile(t *testing.T, repoRoot string, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(repoRoot, name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(content)
}

func assertContains(t *testing.T, content string, want string) {
	t.Helper()
	if !strings.Contains(content, want) {
		t.Fatalf("expected content to contain %q", want)
	}
}

func assertNotContains(t *testing.T, content string, want string) {
	t.Helper()
	if strings.Contains(content, want) {
		t.Fatalf("expected content not to contain %q", want)
	}
}

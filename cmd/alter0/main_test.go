package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildDefaultRuntimePathPrependsRuntimeDirs(t *testing.T) {
	home := t.TempDir()
	localBin := filepath.Join(home, ".local", "bin")
	pnpmBin := filepath.Join(home, ".local", "share", "pnpm")
	if err := os.MkdirAll(localBin, 0o755); err != nil {
		t.Fatalf("mkdir local bin: %v", err)
	}
	if err := os.MkdirAll(pnpmBin, 0o755); err != nil {
		t.Fatalf("mkdir pnpm bin: %v", err)
	}

	got := buildDefaultRuntimePath(home, "/usr/bin:/bin")
	parts := strings.Split(got, string(filepath.ListSeparator))
	if len(parts) < 4 {
		t.Fatalf("unexpected path parts: %v", parts)
	}
	if parts[0] != localBin {
		t.Fatalf("first path = %q, want %q", parts[0], localBin)
	}
	if parts[1] != pnpmBin {
		t.Fatalf("second path = %q, want %q", parts[1], pnpmBin)
	}
	if parts[2] != "/usr/local/bin" {
		t.Fatalf("third path = %q, want /usr/local/bin", parts[2])
	}
}

func TestBuildDefaultRuntimePathDeduplicatesEntries(t *testing.T) {
	got := buildDefaultRuntimePath("", "/usr/bin:/bin:/usr/bin")
	if strings.Count(got, "/usr/bin") != 1 {
		t.Fatalf("path should only contain /usr/bin once: %q", got)
	}
}

func TestMergeNoProxyEntriesAppendsLocalhostWithoutDuplicates(t *testing.T) {
	got := mergeNoProxyEntries("example.com, localhost", "127.0.0.1", "localhost")
	if got != "example.com,localhost,127.0.0.1" {
		t.Fatalf("merged no_proxy = %q", got)
	}
}

func TestEnsureChildProcessWebLoginPasswordSetsAndClearsEnv(t *testing.T) {
	t.Setenv("ALTER0_WEB_LOGIN_PASSWORD", "")

	ensureChildProcessWebLoginPassword(" secret ")
	if got := os.Getenv("ALTER0_WEB_LOGIN_PASSWORD"); got != "secret" {
		t.Fatalf("ALTER0_WEB_LOGIN_PASSWORD = %q, want secret", got)
	}

	ensureChildProcessWebLoginPassword("")
	if got := os.Getenv("ALTER0_WEB_LOGIN_PASSWORD"); got != "" {
		t.Fatalf("ALTER0_WEB_LOGIN_PASSWORD should be cleared, got %q", got)
	}
}

package main

import (
	"os"
	"path/filepath"
	"runtime"
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
	if len(parts) < 2 {
		t.Fatalf("unexpected path parts: %v", parts)
	}
	if parts[0] != localBin {
		t.Fatalf("first path = %q, want %q", parts[0], localBin)
	}
	if parts[1] != pnpmBin {
		t.Fatalf("second path = %q, want %q", parts[1], pnpmBin)
	}
}

func TestBuildDefaultRuntimePathDeduplicatesEntries(t *testing.T) {
	dir := t.TempDir()
	got := buildDefaultRuntimePath("", strings.Join([]string{dir, dir}, string(filepath.ListSeparator)))
	if countPathEntry(got, dir) != 1 {
		t.Fatalf("path should only contain temp dir once: %q", got)
	}
}

func TestRuntimePathSeenKeyFollowsPlatformCaseRules(t *testing.T) {
	left := runtimePathSeenKey(filepath.Join("A", "Bin"))
	right := runtimePathSeenKey(filepath.Join("a", "bin"))
	if runtime.GOOS == "windows" {
		if left != right {
			t.Fatalf("windows path key should be case-insensitive, got %q and %q", left, right)
		}
		return
	}
	if left == right {
		t.Fatalf("non-windows path key should preserve case sensitivity, got %q", left)
	}
}

func countPathEntry(raw string, entry string) int {
	count := 0
	expected := filepath.Clean(entry)
	for _, part := range filepath.SplitList(raw) {
		if filepath.Clean(part) == expected {
			count++
		}
	}
	return count
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

func TestResolveRuntimeChildWebLoginPasswordKeepsGatewayPasswordForChild(t *testing.T) {
	if got := resolveRuntimeChildWebLoginPassword(false, " secret "); got != "secret" {
		t.Fatalf("non-child password = %q, want secret", got)
	}
	if got := resolveRuntimeChildWebLoginPassword(true, " secret "); got != "secret" {
		t.Fatalf("runtime child password = %q, want secret", got)
	}
	t.Setenv("ALTER0_WEB_REUSE_GATEWAY_AUTH", "1")
	if got := resolveRuntimeChildWebLoginPassword(true, " secret "); got != "" {
		t.Fatalf("workspace service child password = %q, want empty", got)
	}
}

func TestValidateRequiredWebLoginPasswordRequiresPasswordForGateway(t *testing.T) {
	if err := validateRequiredWebLoginPassword(false, " secret "); err != nil {
		t.Fatalf("expected non-child password to pass validation, got %v", err)
	}
	if err := validateRequiredWebLoginPassword(true, " secret "); err != nil {
		t.Fatalf("expected runtime child password to pass validation, got %v", err)
	}
	if err := validateRequiredWebLoginPassword(true, ""); err == nil {
		t.Fatal("expected runtime child to reject empty web login password")
	}
	if err := validateRequiredWebLoginPassword(false, ""); err == nil {
		t.Fatal("expected non-child runtime to reject empty web login password")
	}
	t.Setenv("ALTER0_WEB_REUSE_GATEWAY_AUTH", "1")
	if err := validateRequiredWebLoginPassword(true, ""); err != nil {
		t.Fatalf("expected workspace service child to allow empty web login password, got %v", err)
	}
}

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

func TestResolveConfiguredCodexCommandSelectsNewestInstalledVersion(t *testing.T) {
	home := t.TempDir()
	oldCommand := writeFakeCodexCommand(t, filepath.Join(home, ".nvm", "versions", "node", "v22.22.0", "bin", "codex"), "0.141.0")
	currentCommand := writeFakeCodexCommand(t, filepath.Join(home, ".nvm", "current", "bin", "codex"), "99.144.1")

	t.Setenv("HOME", home)
	t.Setenv(codexCommandModeEnvKey, codexCommandModeAuto)

	if got := resolveConfiguredCodexCommand(oldCommand); got != currentCommand {
		t.Fatalf("resolved codex command = %q, want newest installed %q", got, currentCommand)
	}
}

func TestResolveConfiguredCodexCommandPrefersStableManagedPathForEqualVersion(t *testing.T) {
	home := t.TempDir()
	managedCommand := writeFakeCodexCommand(t, filepath.Join(home, ".local", "bin", "codex"), "99.144.1")
	oldCommand := writeFakeCodexCommand(t, filepath.Join(home, ".nvm", "versions", "node", "v22.22.0", "bin", "codex"), "99.144.1")
	writeFakeCodexCommand(t, filepath.Join(home, ".nvm", "current", "bin", "codex"), "99.144.1")

	t.Setenv("HOME", home)
	t.Setenv(codexCommandModeEnvKey, codexCommandModeAuto)

	if got := resolveConfiguredCodexCommand(oldCommand); got != managedCommand {
		t.Fatalf("resolved codex command = %q, want managed stable path %q", got, managedCommand)
	}
}

func TestResolveConfiguredCodexCommandHonorsPinnedMode(t *testing.T) {
	home := t.TempDir()
	oldCommand := writeFakeCodexCommand(t, filepath.Join(home, ".nvm", "versions", "node", "v22.22.0", "bin", "codex"), "0.141.0")
	writeFakeCodexCommand(t, filepath.Join(home, ".nvm", "current", "bin", "codex"), "99.144.1")

	t.Setenv("HOME", home)
	t.Setenv(codexCommandModeEnvKey, codexCommandModePinned)

	if got := resolveConfiguredCodexCommand(oldCommand); got != oldCommand {
		t.Fatalf("resolved codex command = %q, want pinned %q", got, oldCommand)
	}
}

func TestCompareCodexVersionsUsesSemanticVersionOrder(t *testing.T) {
	tests := []struct {
		name  string
		left  string
		right string
		want  int
	}{
		{name: "minor", left: "0.144.1", right: "0.141.0", want: 1},
		{name: "numeric segment", left: "0.10.0", right: "0.9.9", want: 1},
		{name: "release over prerelease", left: "1.0.0", right: "1.0.0-beta.1", want: 1},
		{name: "equal", left: "0.144.1", right: "0.144.1", want: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			left, ok := parseCodexVersion(test.left)
			if !ok {
				t.Fatalf("parse left version %q", test.left)
			}
			right, ok := parseCodexVersion(test.right)
			if !ok {
				t.Fatalf("parse right version %q", test.right)
			}
			if got := compareCodexVersions(left, right); got != test.want {
				t.Fatalf("compareCodexVersions(%q, %q) = %d, want %d", test.left, test.right, got, test.want)
			}
		})
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

func TestResolveRuntimePathsDerivesStorageFromRuntimeRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "runtime")

	paths, err := resolveRuntimePaths(root, "")
	if err != nil {
		t.Fatalf("resolve runtime paths: %v", err)
	}

	if paths.Root != filepath.Clean(root) {
		t.Fatalf("runtime root = %q, want %q", paths.Root, filepath.Clean(root))
	}
	expectedStorage := filepath.Join(filepath.Clean(root), "storage")
	if paths.StorageDir != expectedStorage {
		t.Fatalf("storage dir = %q, want %q", paths.StorageDir, expectedStorage)
	}
}

func TestResolveRuntimePathsRejectsSplitRuntimeAndStorage(t *testing.T) {
	root := filepath.Join(t.TempDir(), "runtime")
	otherStorage := filepath.Join(t.TempDir(), "storage")

	_, err := resolveRuntimePaths(root, otherStorage)
	if err == nil {
		t.Fatal("expected split runtime/storage paths to be rejected")
	}
	if !strings.Contains(err.Error(), runtimeRootEnvKey) || !strings.Contains(err.Error(), storageDirEnvKey) {
		t.Fatalf("error should name conflicting env keys, got %v", err)
	}
}

func TestResolveRuntimePathsAllowsLegacyStorageDirOnly(t *testing.T) {
	legacyStorage := filepath.Join(t.TempDir(), "legacy-storage")

	paths, err := resolveRuntimePaths("", legacyStorage)
	if err != nil {
		t.Fatalf("resolve runtime paths: %v", err)
	}

	if paths.Root != filepath.Clean(legacyStorage) {
		t.Fatalf("legacy runtime root = %q, want %q", paths.Root, filepath.Clean(legacyStorage))
	}
	if paths.StorageDir != filepath.Clean(legacyStorage) {
		t.Fatalf("legacy storage dir = %q, want %q", paths.StorageDir, filepath.Clean(legacyStorage))
	}
}

func TestShouldExportRuntimeRootEnvForAlignedStorage(t *testing.T) {
	root := filepath.Join(t.TempDir(), "runtime")
	paths := runtimePaths{
		Root:       root,
		StorageDir: filepath.Join(root, "storage"),
	}

	if !shouldExportRuntimeRootEnv(paths) {
		t.Fatal("expected aligned runtime storage layout to export runtime root")
	}
}

func TestShouldExportRuntimeRootEnvSkipsUnalignedLegacyStorage(t *testing.T) {
	legacyStorage := filepath.Join(t.TempDir(), "legacy-storage")
	paths := runtimePaths{
		Root:       legacyStorage,
		StorageDir: legacyStorage,
	}

	if shouldExportRuntimeRootEnv(paths) {
		t.Fatal("expected legacy storage-only layout to avoid exporting conflicting runtime root")
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

func writeFakeCodexCommand(t *testing.T, path string, version string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fake codex dir: %v", err)
	}
	content := "#!/bin/sh\nprintf 'codex-cli %s\\n' " + version + "\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write fake codex command: %v", err)
	}
	return path
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

func TestResolveRuntimeChildWebLoginPasswordKeepsGatewayPasswordForSupervisorChild(t *testing.T) {
	t.Setenv(supervisorAddrEnv, "http://127.0.0.1:19090")
	t.Setenv(supervisorTokenEnv, "token-1")

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

func TestResolveRuntimeChildWebLoginPasswordClearsPasswordForWorkspaceServiceChild(t *testing.T) {
	t.Setenv(supervisorAddrEnv, "")
	t.Setenv(supervisorTokenEnv, "")

	if got := resolveRuntimeChildWebLoginPassword(true, " secret "); got != "" {
		t.Fatalf("workspace service child password = %q, want empty", got)
	}
}

func TestValidateRequiredWebLoginPasswordRequiresPasswordForGateway(t *testing.T) {
	t.Setenv(supervisorAddrEnv, "http://127.0.0.1:19090")
	t.Setenv(supervisorTokenEnv, "token-1")

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

func TestValidateRequiredWebLoginPasswordAllowsWorkspaceServiceChildWithoutPassword(t *testing.T) {
	t.Setenv(supervisorAddrEnv, "")
	t.Setenv(supervisorTokenEnv, "")

	if err := validateRequiredWebLoginPassword(true, ""); err != nil {
		t.Fatalf("expected workspace service child to allow empty web login password, got %v", err)
	}
}

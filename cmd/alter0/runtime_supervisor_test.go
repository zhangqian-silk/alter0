package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"alter0/internal/interfaces/web"
)

func TestFilterInternalRuntimeArgsKeepsOnlySupportedPublicFlags(t *testing.T) {
	args := []string{
		"-" + runtimeChildFlag,
		"-" + relaunchHelperFlag,
		"-" + relaunchParentPIDFlag,
		"42",
		"-" + relaunchExecPathFlag + "=runtime.exe",
		"-" + relaunchArgsFlag + "=encoded",
		"-" + relaunchWorkingDirFlag,
		"/tmp/repo",
		"-daily-memory-dir",
		"/var/lib/alter0/storage/memory",
		"-long-term-memory-path=/var/lib/alter0/storage/memory/long-term/MEMORY.md",
		"-web-addr",
		"127.0.0.1:18088",
		"-web-bind-localhost-only=false",
		"--codex-command=/usr/local/bin/codex",
		"-unknown-public-flag",
		"value",
	}

	filtered := filterInternalRuntimeArgs(args)
	expected := []string{
		"-web-addr",
		"127.0.0.1:18088",
		"-web-bind-localhost-only=false",
		"--codex-command=/usr/local/bin/codex",
	}
	if strings.Join(filtered, "\n") != strings.Join(expected, "\n") {
		t.Fatalf("unexpected filtered args:\n got: %v\nwant: %v", filtered, expected)
	}
}

func TestSupervisorClientRestarterReturnsDetailedError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(supervisorTokenHeader) != "secret" {
			t.Fatalf("expected supervisor token header")
		}
		if r.Method != http.MethodPost || r.URL.Path != "/restart" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"sync origin/master: auth failed"}`))
	}))
	defer server.Close()

	restarter := &supervisorClientRestarter{
		addr:   server.URL,
		token:  "secret",
		client: server.Client(),
	}
	accepted, err := restarter.RequestRestart(web.RuntimeRestartOptions{SyncRemoteMaster: true})
	if accepted {
		t.Fatalf("expected restart request rejected")
	}
	if err == nil || !strings.Contains(err.Error(), "auth failed") {
		t.Fatalf("expected detailed error, got %v", err)
	}
}

func TestSupervisorClientRestarterPreservesRestartErrorCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"code":"runtime_restart_discard_confirmation_required","error":"tracked changes exist"}`))
	}))
	defer server.Close()

	restarter := &supervisorClientRestarter{
		addr:   server.URL,
		token:  "secret",
		client: server.Client(),
	}
	accepted, err := restarter.RequestRestart(web.RuntimeRestartOptions{SyncRemoteMaster: true})
	if accepted {
		t.Fatalf("expected restart request rejected")
	}
	var restartErr *web.RuntimeRestartError
	if !errors.As(err, &restartErr) {
		t.Fatalf("expected runtime restart error, got %T %v", err, err)
	}
	if restartErr.Code != web.RuntimeRestartDiscardConfirmationRequired {
		t.Fatalf("expected discard confirmation code, got %q", restartErr.Code)
	}
}

func TestBuildRuntimeProbeAddrNormalizesWildcardHost(t *testing.T) {
	probeAddr, err := buildRuntimeProbeAddr("0.0.0.0:18088")
	if err != nil {
		t.Fatalf("build probe addr failed: %v", err)
	}
	if probeAddr != "http://127.0.0.1:18088" {
		t.Fatalf("unexpected probe addr %q", probeAddr)
	}
}

func TestBuildRelaunchBinaryUsesFrontendAwareBuildScript(t *testing.T) {
	repoDir := t.TempDir()
	scriptDir := filepath.Join(repoDir, "scripts")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatalf("create script dir: %v", err)
	}
	scriptPath := filepath.Join(scriptDir, "build_alter0_service.sh")
	script := `#!/bin/sh
set -eu
: "${ALTER0_BUILD_OUTPUT:?}"
printf '%s\n' "$ALTER0_BUILD_OUTPUT" > "$PWD/build-output.txt"
mkdir -p "$(dirname "$ALTER0_BUILD_OUTPUT")"
printf '#!/bin/sh\n' > "$ALTER0_BUILD_OUTPUT"
chmod +x "$ALTER0_BUILD_OUTPUT"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake build script: %v", err)
	}

	candidate, err := buildRelaunchBinary(repoDir)
	if err != nil {
		t.Fatalf("build relaunch binary: %v", err)
	}
	if _, err := os.Stat(candidate); err != nil {
		t.Fatalf("expected candidate binary to exist: %v", err)
	}
	expectedDir := filepath.Join(repoDir, "output", "runtime")
	if filepath.Dir(candidate) != expectedDir {
		t.Fatalf("candidate written to %q, want %q", filepath.Dir(candidate), expectedDir)
	}
	if runtime.GOOS == "windows" && filepath.Ext(candidate) != ".exe" {
		t.Fatalf("windows candidate must use .exe extension, got %q", candidate)
	}

	recorded, err := os.ReadFile(filepath.Join(repoDir, "build-output.txt"))
	if err != nil {
		t.Fatalf("read recorded build output: %v", err)
	}
	if strings.TrimSpace(string(recorded)) != candidate {
		t.Fatalf("build script received output %q, want %q", strings.TrimSpace(string(recorded)), candidate)
	}
}

func TestSupervisorClientRestarterRejectsEmptyBodyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "", http.StatusBadGateway)
	}))
	defer server.Close()

	restarter := &supervisorClientRestarter{
		addr:   server.URL,
		token:  "secret",
		client: server.Client(),
	}
	_, err := restarter.RequestRestart(web.RuntimeRestartOptions{})
	if err == nil || !strings.Contains(err.Error(), "HTTP 502") {
		t.Fatalf("expected restart error")
	}
}

func TestSupervisorClientRestarterReturnsRestartStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/restart" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get(supervisorTokenHeader) != "secret" {
			t.Fatalf("expected supervisor token header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"failed","error":"candidate runtime exited before ready","sync_remote_master":true}`))
	}))
	defer server.Close()

	restarter := &supervisorClientRestarter{
		addr:   server.URL,
		token:  "secret",
		client: server.Client(),
	}
	status := restarter.GetRestartStatus()
	if status.Status != "failed" {
		t.Fatalf("expected failed status, got %q", status.Status)
	}
	if !status.SyncRemoteMaster {
		t.Fatalf("expected sync flag from status")
	}
	if !strings.Contains(status.Error, "candidate runtime exited") {
		t.Fatalf("unexpected status error %q", status.Error)
	}
}

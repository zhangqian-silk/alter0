package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"alter0/internal/interfaces/web"
)

func TestFilterInternalRuntimeArgsRemovesInternalFlags(t *testing.T) {
	args := []string{
		"-" + runtimeChildFlag,
		"-" + relaunchHelperFlag,
		"-" + relaunchParentPIDFlag,
		"42",
		"-" + relaunchExecPathFlag + "=runtime.exe",
		"-" + relaunchArgsFlag + "=encoded",
		"-" + relaunchWorkingDirFlag,
		"/tmp/repo",
		"-web-addr",
		"127.0.0.1:18088",
	}

	filtered := filterInternalRuntimeArgs(args)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 args after filtering, got %d: %v", len(filtered), filtered)
	}
	if filtered[0] != "-web-addr" || filtered[1] != "127.0.0.1:18088" {
		t.Fatalf("unexpected filtered args: %v", filtered)
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

func TestBuildRuntimeProbeAddrNormalizesWildcardHost(t *testing.T) {
	probeAddr, err := buildRuntimeProbeAddr("0.0.0.0:18088")
	if err != nil {
		t.Fatalf("build probe addr failed: %v", err)
	}
	if probeAddr != "http://127.0.0.1:18088" {
		t.Fatalf("unexpected probe addr %q", probeAddr)
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

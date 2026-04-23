package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceServiceGatewayProxiesRegisteredHTTPService(t *testing.T) {
	upstreamCalled := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalled = true
		if r.URL.Path != "/v1/status" {
			t.Fatalf("expected upstream path /v1/status, got %s", r.URL.Path)
		}
		if r.URL.RawQuery != "ok=1" {
			t.Fatalf("expected upstream query ok=1, got %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":"ok"}`)
	}))
	defer upstream.Close()

	registry, err := newFileWorkspaceServiceRegistry(filepath.Join(t.TempDir(), workspaceServiceRegistryFilename), "alter0.cn")
	if err != nil {
		t.Fatalf("new workspace service registry: %v", err)
	}
	entry, err := registry.Upsert(workspaceServiceRegistrationInput{
		SessionID:   "session-http-service",
		ServiceID:   "api",
		ServiceType: workspaceServiceTypeHTTP,
		UpstreamURL: upstream.URL,
	})
	if err != nil {
		t.Fatalf("register workspace service: %v", err)
	}

	server := &Server{
		logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
		workspaceService: registry,
	}

	calledNext := false
	handler := server.withWorkspaceServiceGateway(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledNext = true
		http.NotFound(w, r)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/status?ok=1", nil)
	req.Host = entry.Host
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if calledNext {
		t.Fatalf("expected workspace service gateway to handle registered host")
	}
	if !upstreamCalled {
		t.Fatalf("expected upstream service to be called")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if got := rec.Header().Get("X-Alter0-Workspace-Service"); got != "api" {
		t.Fatalf("expected workspace service header api, got %q", got)
	}
	if strings.TrimSpace(rec.Body.String()) != `{"status":"ok"}` {
		t.Fatalf("unexpected proxy body %q", rec.Body.String())
	}
}

func TestWorkspaceServiceGatewayRewritesProxyHostToUpstream(t *testing.T) {
	var gotHost string
	var gotForwardedHost string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHost = r.Host
		gotForwardedHost = r.Header.Get("X-Forwarded-Host")
		_, _ = io.WriteString(w, `{"status":"ok"}`)
	}))
	defer upstream.Close()

	registry, err := newFileWorkspaceServiceRegistry(filepath.Join(t.TempDir(), workspaceServiceRegistryFilename), "alter0.cn")
	if err != nil {
		t.Fatalf("new workspace service registry: %v", err)
	}
	entry, err := registry.Upsert(workspaceServiceRegistrationInput{
		SessionID:   "session-http-host-rewrite",
		ServiceID:   defaultWorkspaceServiceID,
		ServiceType: workspaceServiceTypeHTTP,
		UpstreamURL: upstream.URL,
	})
	if err != nil {
		t.Fatalf("register workspace service: %v", err)
	}

	server := &Server{
		logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
		workspaceService: registry,
	}
	handler := server.withWorkspaceServiceGateway(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	req.Host = entry.Host
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	expectedUpstreamHost := strings.TrimPrefix(upstream.URL, "http://")
	if gotHost != expectedUpstreamHost {
		t.Fatalf("expected upstream host %q, got %q", expectedUpstreamHost, gotHost)
	}
	if gotForwardedHost != entry.Host {
		t.Fatalf("expected x-forwarded-host %q, got %q", entry.Host, gotForwardedHost)
	}
}

func TestWorkspaceServiceGatewayProxiesRegisteredHTTPWebService(t *testing.T) {
	upstreamCalled := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalled = true
		if r.URL.Path != "/api/terminal/sessions" {
			t.Fatalf("expected upstream path /api/terminal/sessions, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"items":[]}`)
	}))
	defer upstream.Close()

	registry, err := newFileWorkspaceServiceRegistry(filepath.Join(t.TempDir(), workspaceServiceRegistryFilename), "alter0.cn")
	if err != nil {
		t.Fatalf("new workspace service registry: %v", err)
	}
	entry, err := registry.Upsert(workspaceServiceRegistrationInput{
		SessionID:   "session-http-web",
		ServiceID:   defaultWorkspaceServiceID,
		ServiceType: workspaceServiceTypeHTTP,
		UpstreamURL: upstream.URL,
	})
	if err != nil {
		t.Fatalf("register workspace service: %v", err)
	}

	server := &Server{
		logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
		workspaceService: registry,
	}

	handler := server.withWorkspaceServiceGateway(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/terminal/sessions", nil)
	req.Host = entry.Host
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !upstreamCalled {
		t.Fatalf("expected root web host to proxy into upstream backend")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if got := rec.Header().Get("X-Alter0-Workspace-Service"); got != defaultWorkspaceServiceID {
		t.Fatalf("expected workspace service header %q, got %q", defaultWorkspaceServiceID, got)
	}
	if strings.TrimSpace(rec.Body.String()) != `{"items":[]}` {
		t.Fatalf("unexpected proxy body %q", rec.Body.String())
	}
}

func TestWorkspaceServiceGatewayStartsManagedHTTPServiceBeforeProxy(t *testing.T) {
	upstreamCalled := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalled = true
		_, _ = io.WriteString(w, `{"status":"ok"}`)
	}))
	defer upstream.Close()

	registry, err := newFileWorkspaceServiceRegistry(filepath.Join(t.TempDir(), workspaceServiceRegistryFilename), "alter0.cn")
	if err != nil {
		t.Fatalf("new workspace service registry: %v", err)
	}
	entry, err := registry.Upsert(workspaceServiceRegistrationInput{
		SessionID:    "session-managed-http",
		ServiceID:    defaultWorkspaceServiceID,
		ServiceType:  workspaceServiceTypeHTTP,
		StartCommand: "go run ./cmd/alter0",
		Workdir:      t.TempDir(),
		Port:         19191,
		HealthPath:   "/readyz",
	})
	if err != nil {
		t.Fatalf("register managed workspace service: %v", err)
	}

	runtime := &stubWorkspaceServiceRuntime{
		ensureStarted: func(entry workspaceServiceRegistration) (workspaceServiceRegistration, workspaceServiceRuntimeStatus, error) {
			entry.UpstreamURL = upstream.URL
			return entry, workspaceServiceRuntimeStatus{Status: "running"}, nil
		},
	}
	server := &Server{
		logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
		workspaceService:  registry,
		workspaceRuntime:  runtime,
	}

	handler := server.withWorkspaceServiceGateway(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/runtime", nil)
	req.Host = entry.Host
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if runtime.ensureCalls != 1 {
		t.Fatalf("expected runtime ensureStarted to be called once, got %d", runtime.ensureCalls)
	}
	if !upstreamCalled {
		t.Fatalf("expected managed upstream to be called")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestWorkspaceServiceGatewayServesRegisteredFrontendDist(t *testing.T) {
	repoPath := preparePreviewRepo(t, "workspace frontend")
	registry, err := newFileWorkspaceServiceRegistry(filepath.Join(t.TempDir(), workspaceServiceRegistryFilename), "alter0.cn")
	if err != nil {
		t.Fatalf("new workspace service registry: %v", err)
	}
	entry, err := registry.Upsert(workspaceServiceRegistrationInput{
		SessionID:      "session-frontend-service",
		ServiceID:      defaultWorkspaceServiceID,
		ServiceType:    workspaceServiceTypeFrontendDist,
		RepositoryPath: repoPath,
	})
	if err != nil {
		t.Fatalf("register workspace service: %v", err)
	}

	server := &Server{
		logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
		workspaceService: registry,
	}

	handler := server.withWorkspaceServiceGateway(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))

	chatReq := httptest.NewRequest(http.MethodGet, "/chat", nil)
	chatReq.Host = entry.Host
	chatRec := httptest.NewRecorder()
	handler.ServeHTTP(chatRec, chatReq)

	if chatRec.Code != http.StatusOK {
		t.Fatalf("expected chat status %d, got %d", http.StatusOK, chatRec.Code)
	}
	if got := chatRec.Header().Get("X-Alter0-Workspace-Service"); got != defaultWorkspaceServiceID {
		t.Fatalf("expected workspace service header %q, got %q", defaultWorkspaceServiceID, got)
	}
	if !strings.Contains(chatRec.Body.String(), "workspace frontend") {
		t.Fatalf("expected frontend html body, got %q", chatRec.Body.String())
	}

	assetReq := httptest.NewRequest(http.MethodGet, "/assets/index-preview.js", nil)
	assetReq.Host = entry.Host
	assetRec := httptest.NewRecorder()
	handler.ServeHTTP(assetRec, assetReq)

	if assetRec.Code != http.StatusOK {
		t.Fatalf("expected asset status %d, got %d", http.StatusOK, assetRec.Code)
	}
	if got := strings.TrimSpace(assetRec.Body.String()); got != "console.log('workspace frontend');" {
		t.Fatalf("unexpected asset body %q", got)
	}
}

func TestWorkspaceServiceRegistrationCRUD(t *testing.T) {
	repoPath := preparePreviewRepo(t, "workspace service")
	registry, err := newFileWorkspaceServiceRegistry(filepath.Join(t.TempDir(), workspaceServiceRegistryFilename), "alter0.cn")
	if err != nil {
		t.Fatalf("new workspace service registry: %v", err)
	}
	server := &Server{
		logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
		workspaceService: registry,
	}

	putWebReq := httptest.NewRequest(
		http.MethodPut,
		"/api/control/workspace-services/session-workspace-service",
		strings.NewReader(`{"service_type":"frontend_dist","repository_path":"`+repoPath+`"}`),
	)
	putWebRec := httptest.NewRecorder()
	server.workspaceServiceItemHandler(putWebRec, putWebReq)
	if putWebRec.Code != http.StatusOK {
		t.Fatalf("expected web put 200, got %d: %s", putWebRec.Code, putWebRec.Body.String())
	}

	var webEntry workspaceServiceRegistration
	if err := json.NewDecoder(putWebRec.Body).Decode(&webEntry); err != nil {
		t.Fatalf("decode web entry: %v", err)
	}
	if webEntry.ServiceID != defaultWorkspaceServiceID {
		t.Fatalf("expected default service id %q, got %+v", defaultWorkspaceServiceID, webEntry)
	}
	if webEntry.Host == "" || webEntry.DistPath == "" || webEntry.URL == "" {
		t.Fatalf("expected frontend registration fields, got %+v", webEntry)
	}

	putAPIReq := httptest.NewRequest(
		http.MethodPut,
		"/api/control/workspace-services/session-workspace-service/api",
		strings.NewReader(`{"service_type":"http","upstream_url":"http://127.0.0.1:19191"}`),
	)
	putAPIRec := httptest.NewRecorder()
	server.workspaceServiceItemHandler(putAPIRec, putAPIReq)
	if putAPIRec.Code != http.StatusOK {
		t.Fatalf("expected api put 200, got %d: %s", putAPIRec.Code, putAPIRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/control/workspace-services", nil)
	listRec := httptest.NewRecorder()
	server.workspaceServiceCollectionHandler(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", listRec.Code, listRec.Body.String())
	}

	var listResp struct {
		Items []workspaceServiceRegistration `json:"items"`
	}
	if err := json.NewDecoder(listRec.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listResp.Items) != 2 {
		t.Fatalf("expected two services, got %+v", listResp.Items)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/control/workspace-services/session-workspace-service/api", nil)
	deleteRec := httptest.NewRecorder()
	server.workspaceServiceItemHandler(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("expected delete 200, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
}

func TestWorkspaceServiceRegistrationStoresManagedHTTPCommandConfig(t *testing.T) {
	workdir := t.TempDir()
	registry, err := newFileWorkspaceServiceRegistry(filepath.Join(t.TempDir(), workspaceServiceRegistryFilename), "alter0.cn")
	if err != nil {
		t.Fatalf("new workspace service registry: %v", err)
	}

	entry, err := registry.Upsert(workspaceServiceRegistrationInput{
		SessionID:    "session-command-http",
		ServiceID:    "web",
		ServiceType:  workspaceServiceTypeHTTP,
		StartCommand: "go run ./cmd/alter0",
		Workdir:      workdir,
		Port:         18091,
		HealthPath:   "/readyz",
	})
	if err != nil {
		t.Fatalf("register managed workspace service: %v", err)
	}

	if entry.UpstreamURL != "http://127.0.0.1:18091" {
		t.Fatalf("expected managed upstream url, got %+v", entry)
	}
	if entry.StartCommand != "go run ./cmd/alter0" {
		t.Fatalf("expected start command to persist, got %+v", entry)
	}
	if entry.Workdir != filepath.ToSlash(workdir) {
		t.Fatalf("expected normalized workdir, got %+v", entry)
	}
	if entry.Port != 18091 || entry.HealthPath != "/readyz" {
		t.Fatalf("expected managed runtime metadata, got %+v", entry)
	}
}

func preparePreviewRepo(t *testing.T, marker string) string {
	t.Helper()

	repoPath := filepath.Join(t.TempDir(), "repo")
	distPath := filepath.Join(repoPath, "internal", "interfaces", "web", "static", "dist")
	assetsPath := filepath.Join(distPath, "assets")
	legacyPath := filepath.Join(distPath, "legacy")
	if err := os.MkdirAll(assetsPath, 0o755); err != nil {
		t.Fatalf("mkdir assets path: %v", err)
	}
	if err := os.MkdirAll(legacyPath, 0o755); err != nil {
		t.Fatalf("mkdir legacy path: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, ".git"), []byte("gitdir: /tmp/mock\n"), 0o644); err != nil {
		t.Fatalf("write git marker: %v", err)
	}
	if err := os.WriteFile(filepath.Join(distPath, "index.html"), []byte("<!doctype html><title>"+marker+"</title>"), 0o644); err != nil {
		t.Fatalf("write preview html: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetsPath, "index-preview.js"), []byte("console.log('"+marker+"');"), 0o644); err != nil {
		t.Fatalf("write preview asset: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyPath, "chat.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatalf("write preview legacy asset: %v", err)
	}
	return repoPath
}

type stubWorkspaceServiceRuntime struct {
	ensureCalls   int
	stopped       []string
	ensureStarted func(entry workspaceServiceRegistration) (workspaceServiceRegistration, workspaceServiceRuntimeStatus, error)
	stop          func(entry workspaceServiceRegistration) error
}

func (s *stubWorkspaceServiceRuntime) EnsureStarted(entry workspaceServiceRegistration) (workspaceServiceRegistration, workspaceServiceRuntimeStatus, error) {
	s.ensureCalls++
	if s.ensureStarted != nil {
		return s.ensureStarted(entry)
	}
	return entry, workspaceServiceRuntimeStatus{}, nil
}

func (s *stubWorkspaceServiceRuntime) Stop(entry workspaceServiceRegistration) error {
	s.stopped = append(s.stopped, entry.SessionID+":"+entry.ServiceID)
	if s.stop != nil {
		return s.stop(entry)
	}
	return nil
}

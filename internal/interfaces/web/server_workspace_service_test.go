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

package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSessionPreviewHandlerServesRegisteredChatPage(t *testing.T) {
	repoPath := preparePreviewRepo(t, "preview chat")
	registry, err := newFileSessionPreviewRegistry(filepath.Join(t.TempDir(), "session-previews.json"), "alter0.cn")
	if err != nil {
		t.Fatalf("new preview registry: %v", err)
	}
	entry, err := registry.Upsert(sessionPreviewRegistrationInput{
		SessionID:      "session-preview-chat",
		RepositoryPath: repoPath,
	})
	if err != nil {
		t.Fatalf("register preview: %v", err)
	}

	server := &Server{
		logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		previewRegistry: registry,
	}

	calledNext := false
	handler := server.withSessionPreview(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledNext = true
		http.NotFound(w, r)
	}))

	req := httptest.NewRequest(http.MethodGet, "/chat", nil)
	req.Host = entry.Host
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if calledNext {
		t.Fatalf("expected preview handler to serve chat page before shared handler")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if cache := rec.Header().Get("Cache-Control"); cache != webPageCacheControl {
		t.Fatalf("expected cache-control %q, got %q", webPageCacheControl, cache)
	}
	if got := rec.Header().Get("X-Alter0-Preview-Session"); got != "session-preview-chat" {
		t.Fatalf("expected preview session header, got %q", got)
	}
	if !strings.Contains(rec.Body.String(), "preview chat") {
		t.Fatalf("expected preview html body, got %q", rec.Body.String())
	}
}

func TestSessionPreviewHandlerServesRegisteredAssets(t *testing.T) {
	repoPath := preparePreviewRepo(t, "preview assets")
	registry, err := newFileSessionPreviewRegistry(filepath.Join(t.TempDir(), "session-previews.json"), "alter0.cn")
	if err != nil {
		t.Fatalf("new preview registry: %v", err)
	}
	entry, err := registry.Upsert(sessionPreviewRegistrationInput{
		SessionID:      "session-preview-assets",
		RepositoryPath: repoPath,
	})
	if err != nil {
		t.Fatalf("register preview: %v", err)
	}

	server := &Server{
		logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		previewRegistry: registry,
	}

	handler := server.withSessionPreview(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))

	req := httptest.NewRequest(http.MethodGet, "/assets/index-preview.js", nil)
	req.Host = entry.Host
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if cache := rec.Header().Get("Cache-Control"); cache != immutableStaticAssetCacheControl {
		t.Fatalf("expected immutable cache-control, got %q", cache)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "console.log('preview assets');" {
		t.Fatalf("unexpected asset body %q", got)
	}
}

func TestSessionPreviewHandlerFallsBackForSharedHost(t *testing.T) {
	server := &Server{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	calledNext := false
	handler := server.withSessionPreview(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledNext = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/chat", nil)
	req.Host = "alter0.cn"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !calledNext {
		t.Fatalf("expected shared handler for primary host")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected fallback response, got %d", rec.Code)
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

package web

import (
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"
)

func TestRootHandlerRedirectsToChat(t *testing.T) {
	server := &Server{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	server.rootHandler(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected status %d, got %d", http.StatusTemporaryRedirect, rec.Code)
	}
	location := rec.Header().Get("Location")
	if location != "/chat" {
		t.Fatalf("expected redirect location /chat, got %q", location)
	}
}

func TestChatPageHandlerServesEmbeddedFrontendDist(t *testing.T) {
	server := &Server{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	req := httptest.NewRequest(http.MethodGet, "/chat", nil)
	rec := httptest.NewRecorder()

	server.chatPageHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Fatalf("expected text/html response, got %q", contentType)
	}
	body := rec.Body.String()
	markers := []string{
		"<title>alter0 Chat</title>",
		`id="frontend-root"`,
		`<script type="module" crossorigin src="/assets/`,
		`data-frontend-shell="legacy-bridge"`,
	}
	for _, marker := range markers {
		if !strings.Contains(body, marker) {
			t.Fatalf("expected frontend dist marker %q", marker)
		}
	}
}

func TestChatPageHandlerMethodNotAllowed(t *testing.T) {
	server := &Server{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	req := httptest.NewRequest(http.MethodPost, "/chat", nil)
	rec := httptest.NewRecorder()

	server.chatPageHandler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestEmbeddedFrontendDistAssetsAvailable(t *testing.T) {
	content, err := webStaticFS.ReadFile("static/dist/index.html")
	if err != nil {
		t.Fatalf("expected embedded frontend dist index, got error: %v", err)
	}
	if len(content) == 0 {
		t.Fatal("expected embedded frontend dist index content")
	}

	legacyBridge, err := webStaticFS.ReadFile("static/dist/legacy/chat.js")
	if err != nil {
		t.Fatalf("expected embedded legacy bridge runtime, got error: %v", err)
	}
	if len(legacyBridge) == 0 {
		t.Fatal("expected embedded legacy bridge runtime content")
	}
}

func TestChatPageHandlerDisablesCaching(t *testing.T) {
	server := &Server{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	req := httptest.NewRequest(http.MethodGet, "/chat", nil)
	rec := httptest.NewRecorder()

	server.chatPageHandler(rec, req)

	if cacheControl := rec.Header().Get("Cache-Control"); cacheControl != "no-cache" {
		t.Fatalf("expected chat page Cache-Control no-cache, got %q", cacheControl)
	}
}

func TestChatPageHandlerProxiesToFrontendDevServerWhenConfigured(t *testing.T) {
	devServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat" {
			t.Fatalf("expected proxied path /chat, got %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><body>frontend-dev-server</body></html>"))
	}))
	defer devServer.Close()

	server := &Server{
		logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
		frontendDevOrigin: devServer.URL,
		frontendDevProxy:  newFrontendDevProxy(devServer.URL, slog.New(slog.NewTextHandler(io.Discard, nil))),
	}
	req := httptest.NewRequest(http.MethodGet, "/chat", nil)
	rec := httptest.NewRecorder()

	server.chatPageHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "frontend-dev-server") {
		t.Fatalf("expected proxied dev server body, got %q", body)
	}
}

func TestRootHandlerProxiesViteRuntimeAssetRequestsWhenConfigured(t *testing.T) {
	devServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/src/main.tsx" {
			t.Fatalf("expected proxied path /src/main.tsx, got %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/javascript")
		_, _ = w.Write([]byte("console.log('vite-runtime');"))
	}))
	defer devServer.Close()

	server := &Server{
		logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
		frontendDevOrigin: devServer.URL,
		frontendDevProxy:  newFrontendDevProxy(devServer.URL, slog.New(slog.NewTextHandler(io.Discard, nil))),
	}
	req := httptest.NewRequest(http.MethodGet, "/src/main.tsx", nil)
	rec := httptest.NewRecorder()

	server.rootHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "vite-runtime") {
		t.Fatalf("expected proxied vite asset body, got %q", body)
	}
}

func TestResolveFrontendDevOriginHonorsEnvironment(t *testing.T) {
	t.Setenv(frontendDevOriginEnvKey, " http://127.0.0.1:15173/ ")

	if got := resolveFrontendDevOrigin(); got != "http://127.0.0.1:15173" {
		t.Fatalf("expected normalized frontend dev origin, got %q", got)
	}
}

func TestEmbeddedLegacySourceAssetsRemoved(t *testing.T) {
	legacyPaths := []string{
		"static/chat.html",
		"static/assets/chat.js",
		"static/assets/chat.css",
		"static/assets/chat-core.css",
		"static/assets/chat-routes.css",
		"static/assets/chat-terminal.css",
	}

	for _, assetPath := range legacyPaths {
		if _, err := webStaticFS.ReadFile(assetPath); err == nil {
			t.Fatalf("expected legacy source asset %q to be removed from embedded web assets", assetPath)
		}
	}
}

func TestFrontendHashedAssetsUseImmutableCaching(t *testing.T) {
	assetPath := findEmbeddedAsset(t, "static/dist/assets/*.js")
	assetsFS, err := webAssetFS("assets")
	if err != nil {
		t.Fatalf("load dist assets fs: %v", err)
	}

	handler := cacheControlledFileServer("/assets/", assetsFS, immutableStaticAssetCacheControl)
	req := httptest.NewRequest(http.MethodGet, "/assets/"+path.Base(assetPath), nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if cacheControl := rec.Header().Get("Cache-Control"); cacheControl != immutableStaticAssetCacheControl {
		t.Fatalf("expected immutable asset Cache-Control %q, got %q", immutableStaticAssetCacheControl, cacheControl)
	}
}

func TestLegacyBridgeAssetsDisableCaching(t *testing.T) {
	legacyFS, err := webAssetFS("legacy")
	if err != nil {
		t.Fatalf("load legacy assets fs: %v", err)
	}

	handler := cacheControlledFileServer("/legacy/", legacyFS, bridgeStaticAssetCacheControl)
	req := httptest.NewRequest(http.MethodGet, "/legacy/chat.js", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if cacheControl := rec.Header().Get("Cache-Control"); cacheControl != bridgeStaticAssetCacheControl {
		t.Fatalf("expected legacy asset Cache-Control %q, got %q", bridgeStaticAssetCacheControl, cacheControl)
	}
}

func TestChatComposerUsesReusableComponent(t *testing.T) {
	script := readEmbeddedAsset(t, "static/assets/chat.js")
	markers := []string{
		"function createReusableComposer() {",
		"const COMPOSER_DRAFT_STORAGE_KEY = \"alter0.web.composer.drafts.v1\";",
		"const composerNavigationRegistry = new Map();",
		"function confirmComposerNavigation() {",
		"suppressHashRouteConfirm: \"\",",
		"function readComposerDraftValue(storage, key) {",
		"function writeComposerDraftValue(storage, key, value) {",
		"function getMainChatDraftKey(sessionID = activeConversationSessionID(), mode = routeConversationMode()) {",
		"function clearMainChatDraft(sessionID) {",
		"const mainChatComposer = createReusableComposer();",
		"function syncMainChatComposerDraft(sessionID = activeConversationSessionID(), options = {}) {",
		"mainChatComposer.bind(input, chatForm, {",
		"draftStorage: \"session\",",
		"draftKey: () => getMainChatDraftKey(),",
		"switchDraftKey(nextKey, options = {}) {",
		"counterNode: charCount,",
		"submitNode: sendButton,",
		`document.body.setAttribute("data-composer-unsaved-state", hasDraft ? "dirty" : "clean");`,
		`document.body.setAttribute("data-composer-unsaved-confirm", String(state || "idle"));`,
		`inputNode.setAttribute("data-composer-ready", "true");`,
		`stableName: "chat-main",`,
		`stableName: "cron-prompt",`,
		`stableName: "terminal-runtime",`,
		`data-terminal-workspace-status="${escapeHTML(normalizeTerminalSessionStatus(session.status) || "unknown")}"`,
		"document.body.setAttribute(\"data-app-ready\", \"true\");",
		"document.addEventListener(LEGACY_SHELL_CREATE_SESSION_EVENT, handleLegacyShellSessionCreation);",
		"const cronComposer = createReusableComposer();",
		"cronComposer.bind(promptInput, form, {",
		"const controlTaskTerminalComposer = createReusableComposer();",
		"controlTaskTerminalComposer.bind(inputNode, formNode, {",
		"window.addEventListener(\"beforeunload\", (event) => {",
	}
	for _, marker := range markers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected reusable composer marker %q", marker)
		}
	}
}

func TestChatScriptRecoversInterruptedStreams(t *testing.T) {
	script := readEmbeddedAsset(t, "static/assets/chat.js")
	markers := []string{
		`"msg.stream_interrupted": "stream interrupted"`,
		`function recoverInterruptedStreamingMessage(message) {`,
		`function finalizeInterruptedStreamMessage(message, errorText) {`,
		`return recoverInterruptedStreamingMessage({`,
		`finalizeInterruptedStreamMessage(assistantMessage, streamResult.error || "unknown");`,
		`const taskID = String(payload?.task_id || "").trim();`,
	}
	for _, marker := range markers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected interrupted stream recovery marker %q", marker)
		}
	}
}

func findEmbeddedAsset(t *testing.T, pattern string) string {
	t.Helper()

	matches, err := fs.Glob(webStaticFS, pattern)
	if err != nil {
		t.Fatalf("glob embedded assets %q: %v", pattern, err)
	}
	if len(matches) == 0 {
		t.Fatalf("expected embedded asset matching %q", pattern)
	}
	return matches[0]
}

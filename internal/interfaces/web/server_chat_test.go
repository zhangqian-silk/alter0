package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
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

func TestChatPageHandlerServesEmbeddedHTML(t *testing.T) {
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
	if !strings.Contains(rec.Body.String(), "alter0 Chat") {
		t.Fatalf("expected chat page content")
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

func TestEmbeddedAssetsAvailable(t *testing.T) {
	paths := []string{
		"static/dist/index.html",
		"static/dist/legacy/chat.js",
		"static/dist/legacy/chat.css",
	}
	for _, path := range paths {
		content, err := webStaticFS.ReadFile(path)
		if err != nil {
			t.Fatalf("expected embedded %s, got error: %v", path, err)
		}
		if len(content) == 0 {
			t.Fatalf("expected embedded %s content", path)
		}
	}
}

func TestChatPageLoadsBridgeBundleAfterLegacyStyles(t *testing.T) {
	html := readEmbeddedAssetRaw(t, "static/dist/index.html")
	styleMarker := `/legacy/chat.css`
	scriptMarker := `/assets/index-`

	styleIndex := strings.Index(html, styleMarker)
	scriptIndex := strings.Index(html, scriptMarker)
	if styleIndex == -1 || scriptIndex == -1 {
		t.Fatalf("expected chat page to load legacy styles and frontend bundle")
	}
	if styleIndex >= scriptIndex {
		t.Fatalf("expected legacy styles before frontend bundle")
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
		`stableName: "terminal-runtime",`,
		"document.body.setAttribute(\"data-app-ready\", \"true\");",
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

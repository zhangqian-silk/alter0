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
		"card.dataset.sessionId = item.id;",
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

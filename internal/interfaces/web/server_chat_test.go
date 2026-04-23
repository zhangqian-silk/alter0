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
	if !strings.Contains(rec.Body.String(), "Alter0 Chat") {
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
	if strings.Contains(html, `/legacy/chat.js`) {
		t.Fatalf("expected chat page to stop loading legacy runtime script")
	}
}

func TestChatPagesDefaultToEnglishDocumentLanguage(t *testing.T) {
	sourceHTML := readWorkspaceFile(t, "frontend/index.html")
	if !strings.Contains(sourceHTML, `<html lang="en">`) {
		t.Fatalf("expected frontend source entry to default document language to English")
	}

	embeddedHTML := readEmbeddedAssetRaw(t, "static/dist/index.html")
	if !strings.Contains(embeddedHTML, `<html lang="en">`) {
		t.Fatalf("expected embedded chat page to default document language to English")
	}

	legacyHTML := readWorkspaceFile(t, "static/chat.html")
	if !strings.Contains(legacyHTML, `<html lang="en">`) {
		t.Fatalf("expected legacy chat page to default document language to English")
	}
}

func TestLoginPageDefaultsToEnglishDocumentLanguage(t *testing.T) {
	server := &Server{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	rec := httptest.NewRecorder()

	server.renderLoginPage(rec, "", "/chat")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `<html lang="en">`) {
		t.Fatalf("expected login page to default document language to English")
	}
	if !strings.Contains(rec.Body.String(), "Alter0 Login") {
		t.Fatalf("expected login page title to expose Alter0 branding")
	}
	if !strings.Contains(rec.Body.String(), "Alter0 Console Login") {
		t.Fatalf("expected login page heading to expose Alter0 branding")
	}
	if !strings.Contains(rec.Body.String(), "IBM Plex Sans") {
		t.Fatalf("expected login page to use the shared workbench typography baseline")
	}
	if !strings.Contains(rec.Body.String(), "Start in a secure Alter0 workspace.") {
		t.Fatalf("expected login page to describe the workbench entry point")
	}
}

func TestChatComposerUsesReusableComponent(t *testing.T) {
	runtimeSource := readWorkspaceFile(t, "frontend/src/features/conversation-runtime/ConversationRuntimeProvider.tsx")
	runtimeMarkers := []string{
		`const COMPOSER_DRAFT_STORAGE_KEY = "alter0.web.composer.drafts.v1";`,
		`const MAX_COMPOSER_CHARS = 10000;`,
		"function loadComposerDrafts()",
		"function persistComposerDrafts(drafts: ComposerDraftMap)",
		"const [composerDrafts, setComposerDrafts] = useState<ComposerDraftMap>(() => loadComposerDrafts());",
		"const nextDrafts = { ...composerDrafts, [session.id]: value.slice(0, MAX_COMPOSER_CHARS) };",
		"persistComposerDrafts(nextDrafts);",
	}
	for _, marker := range runtimeMarkers {
		if !strings.Contains(runtimeSource, marker) {
			t.Fatalf("expected runtime composer marker %q", marker)
		}
	}

	workspaceSource := readWorkspaceFile(t, "frontend/src/features/conversation-runtime/ConversationWorkspace.tsx")
	workspaceMarkers := []string{
		`className="terminal-chat-form conversation-chat-form"`,
		`className="terminal-composer-input conversation-composer-input"`,
		`className="terminal-chat-submit conversation-chat-submit"`,
		`maxLength={10000}`,
	}
	for _, marker := range workspaceMarkers {
		if !strings.Contains(workspaceSource, marker) {
			t.Fatalf("expected chat workspace composer marker %q", marker)
		}
	}

	terminalSource := readWorkspaceFile(t, "frontend/src/features/shell/components/ReactManagedTerminalRouteBody.tsx")
	terminalMarkers := []string{
		`data-composer-form="terminal-runtime"`,
		`data-composer-input="terminal-runtime"`,
		`data-composer-submit="terminal-runtime"`,
	}
	for _, marker := range terminalMarkers {
		if !strings.Contains(terminalSource, marker) {
			t.Fatalf("expected terminal composer marker %q", marker)
		}
	}
}

func TestChatScriptRecoversInterruptedStreams(t *testing.T) {
	script := readWorkspaceFile(t, "frontend/src/features/conversation-runtime/ConversationRuntimeProvider.tsx")
	markers := []string{
		`status: "streaming",`,
		`error: sawDone ? "" : "stream interrupted",`,
		`taskPending: Boolean(body?.task_id && !isTerminalTaskStatus(normalizeText(body?.task_status))),`,
		`taskResultDelivered: Boolean(record.task_result_delivered),`,
		`message.taskID && message.taskPending && !message.taskResultDelivered`,
		`taskPending: !isTerminalTaskStatus(status),`,
		`taskResultDelivered: isTerminalTaskStatus(status),`,
	}
	for _, marker := range markers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected interrupted stream recovery marker %q", marker)
		}
	}
}

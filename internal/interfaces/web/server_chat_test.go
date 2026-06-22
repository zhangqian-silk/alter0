package web

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
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

func TestRootHandlerRedirectsToChatWithoutQueryInference(t *testing.T) {
	server := &Server{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	req := httptest.NewRequest(http.MethodGet, "/?session_id=terminal-2&foo=bar", nil)
	rec := httptest.NewRecorder()

	server.rootHandler(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected status %d, got %d", http.StatusTemporaryRedirect, rec.Code)
	}
	if location := rec.Header().Get("Location"); location != "/chat" {
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

func TestChatPageHandlerVersionsImmutableAssetsFromContent(t *testing.T) {
	server := &Server{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	req := httptest.NewRequest(http.MethodGet, "/chat", nil)
	rec := httptest.NewRecorder()

	server.chatPageHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	scriptAsset, scriptVersion := extractVersionedAssetReferenceForTest(t, body, "js")
	styleAsset, styleVersion := extractVersionedAssetReferenceForTest(t, body, "css")
	if scriptVersion != embeddedAssetVersionForTest(t, scriptAsset) {
		t.Fatalf("script asset %s version = %q", scriptAsset, scriptVersion)
	}
	if styleVersion != embeddedAssetVersionForTest(t, styleAsset) {
		t.Fatalf("stylesheet asset %s version = %q", styleAsset, styleVersion)
	}
	assertNotContains(t, body, `?v=20260604-md-table`)
}

func TestTerminalPageHandlerServesEmbeddedHTML(t *testing.T) {
	server := &Server{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	req := httptest.NewRequest(http.MethodGet, "/terminal", nil)
	rec := httptest.NewRecorder()

	server.chatPageHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("expected text/html response, got %q", rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(rec.Body.String(), "Alter0 Chat") {
		t.Fatalf("expected workbench page content")
	}
}

func embeddedAssetVersionForTest(t *testing.T, assetPath string) string {
	t.Helper()
	content, err := webStaticFS.ReadFile("static/dist/" + assetPath)
	if err != nil {
		t.Fatalf("read embedded asset %s: %v", assetPath, err)
	}
	return shortContentHashForTest(content)
}

func extractVersionedAssetReferenceForTest(t *testing.T, content string, extension string) (string, string) {
	t.Helper()
	pattern := regexp.MustCompile(`/((?:assets/index-[^"?]+\.)` + regexp.QuoteMeta(extension) + `)\?v=([a-f0-9]{12})`)
	match := pattern.FindStringSubmatch(content)
	if len(match) != 3 {
		t.Fatalf("expected versioned %s asset reference in %q", extension, content)
	}
	return match[1], match[2]
}

func shortContentHashForTest(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])[:12]
}

func assertContains(t *testing.T, content string, want string) {
	t.Helper()
	if !strings.Contains(content, want) {
		t.Fatalf("expected content to contain %q", want)
	}
}

func assertNotContains(t *testing.T, content string, want string) {
	t.Helper()
	if strings.Contains(content, want) {
		t.Fatalf("expected content not to contain %q", want)
	}
}

func TestWorkbenchPageHandlerServesAllCanonicalPagePaths(t *testing.T) {
	server := &Server{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	for _, path := range []string{
		"/chat",
		"/terminal",
		"/settings",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()

		server.chatPageHandler(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected %s to serve workbench HTML, got %d", path, rec.Code)
		}
		if !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
			t.Fatalf("expected %s to return text/html, got %q", path, rec.Header().Get("Content-Type"))
		}
	}
}

func TestWorkbenchInteractivePagePathsOnlyIncludeTopLevelRoutes(t *testing.T) {
	for _, path := range []string{"/", "/chat", "/terminal", "/settings"} {
		if !isInteractivePagePath(path) {
			t.Fatalf("expected %s to be an interactive workbench path", path)
		}
	}
	for _, path := range []string{"/legacy-profile", "/memory", "/tasks", "/models", "/codex-accounts"} {
		if isInteractivePagePath(path) {
			t.Fatalf("expected retired settings path %s to stop being an interactive workbench path", path)
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

func TestLoginPageUsesMobileViewportSafeLayoutContract(t *testing.T) {
	server := &Server{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	rec := httptest.NewRecorder()

	server.renderLoginPage(rec, "", "/chat")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	markers := []string{
		`width=device-width,initial-scale=1,viewport-fit=cover`,
		`min-height:100dvh`,
		`padding:max(18px,env(safe-area-inset-top))`,
		`padding-bottom:max(18px,env(safe-area-inset-bottom))`,
		`@media (max-width: 640px)`,
	}
	for _, marker := range markers {
		if !strings.Contains(body, marker) {
			t.Fatalf("expected login page to include mobile layout marker %q", marker)
		}
	}
}

func TestChatComposerUsesReusableComponent(t *testing.T) {
	runtimeSource := readWorkspaceFile(t, "frontend/src/features/conversation-runtime/ConversationRuntimeProvider.tsx")
	runtimeMarkers := []string{
		`const COMPOSER_DRAFT_STORAGE_KEY = "alter0.web.composer.drafts.v1";`,
		`const MAX_COMPOSER_CHARS = 10000;`,
		`const NEW_CHAT_DRAFT_KEY = "__chat_new__";`,
		"function loadComposerDrafts()",
		"function persistComposerDrafts(drafts: ComposerDraftMap)",
		"const [composerDrafts, setComposerDrafts] = useState<ComposerDraftMap>(() => loadComposerDrafts());",
		"const activeDraftKey = activeSessionID || NEW_CHAT_DRAFT_KEY;",
		"const nextDrafts = { ...composerDrafts, [activeDraftKey]: value.slice(0, MAX_COMPOSER_CHARS) };",
		"const nextDrafts = { ...composerDrafts, [session.id]: \"\", [NEW_CHAT_DRAFT_KEY]: \"\" };",
	}
	for _, marker := range runtimeMarkers {
		if !strings.Contains(runtimeSource, marker) {
			t.Fatalf("expected runtime composer marker %q", marker)
		}
	}

	workspaceSource := readWorkspaceFile(t, "frontend/src/features/conversation-runtime/ConversationWorkspace.tsx") +
		readWorkspaceFile(t, "frontend/src/features/shell/components/RuntimeWorkspacePage.tsx") +
		readWorkspaceFile(t, "frontend/src/features/shell/components/RuntimeComposer.tsx")
	workspaceMarkers := []string{
		`<RuntimeWorkspacePage controller={controller} />`,
		`rootClassName: "runtime-workspace-view"`,
		`const composerAlias = runtimeKind === "terminal" ? "terminal" : "conversation";`,
		`data-composer-form={composerAlias}`,
		`data-composer-input={composerAlias}`,
		`data-composer-submit={composerAlias}`,
		`data-runtime-composer="true"`,
		`"runtime-composer-input"`,
		`"runtime-composer-submit"`,
	}
	for _, marker := range workspaceMarkers {
		if !strings.Contains(workspaceSource, marker) {
			t.Fatalf("expected chat workspace composer marker %q", marker)
		}
	}

	terminalSource := readWorkspaceFile(t, "frontend/src/features/shell/components/ReactManagedTerminalRouteBody.tsx") +
		readWorkspaceFile(t, "frontend/src/features/shell/components/RuntimeComposer.tsx")
	terminalMarkers := []string{
		`runtimeKind: "terminal"`,
		`data-composer-form={composerAlias}`,
		`data-composer-input={composerAlias}`,
		`data-composer-submit={composerAlias}`,
	}
	for _, marker := range terminalMarkers {
		if !strings.Contains(terminalSource, marker) {
			t.Fatalf("expected terminal composer marker %q", marker)
		}
	}
}

func TestChatScriptUsesTerminalSessionInput(t *testing.T) {
	script := readWorkspaceFile(t, "frontend/src/features/conversation-runtime/ConversationRuntimeProvider.tsx")
	markers := []string{
		`const TERMINAL_SESSION_COLLECTION_ENDPOINT = "/api/terminal/sessions";`,
		"chatTerminalSessionEndpoint(`${encodeURIComponent(session.id)}/input`)",
		`skill_ids: activeSkillIDs,`,
		`await apiClient.post<`,
	}
	for _, marker := range markers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected Terminal input marker %q", marker)
		}
	}
	removedMarkers := []string{
		`sendMessageJSON`,
		`MESSAGE_ENDPOINT`,
		`/api/messages`,
		`conversation-runtime/sessions`,
		`sendMessageStream`,
		`parseSSEBlock`,
		`/api/messages/stream`,
		`stream interrupted`,
	}
	for _, marker := range removedMarkers {
		if strings.Contains(script, marker) {
			t.Fatalf("did not expect removed stream marker %q", marker)
		}
	}
}

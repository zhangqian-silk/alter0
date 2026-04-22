package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAuthMiddlewareRedirectsHTMLToLogin(t *testing.T) {
	server := &Server{
		logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
		webLoginEnabled:  true,
		webLoginPassword: "secret",
		webSessionToken:  "token-1",
	}
	handler := server.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/chat", nil)
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected %d, got %d", http.StatusTemporaryRedirect, rec.Code)
	}
	location := rec.Header().Get("Location")
	if !strings.HasPrefix(location, "/login?next=") {
		t.Fatalf("expected redirect to login, got %q", location)
	}
}

func TestAuthMiddlewareRejectsAPIWithoutSession(t *testing.T) {
	server := &Server{
		logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
		webLoginEnabled:  true,
		webLoginPassword: "secret",
		webSessionToken:  "token-1",
	}
	handler := server.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/messages", strings.NewReader(`{"content":"hello"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "authentication required") {
		t.Fatalf("expected auth error response, got %s", rec.Body.String())
	}
}

func TestAuthMiddlewareRedirectsChatWithoutHTMLAccept(t *testing.T) {
	server := &Server{
		logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
		webLoginEnabled:  true,
		webLoginPassword: "secret",
		webSessionToken:  "token-1",
	}
	handler := server.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/chat", nil)
	req.Header.Set("Accept", "*/*")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected %d, got %d", http.StatusTemporaryRedirect, rec.Code)
	}
	location := rec.Header().Get("Location")
	if !strings.HasPrefix(location, "/login?next=") {
		t.Fatalf("expected login redirect, got %q", location)
	}
}

func TestAuthMiddlewareAllowsFaviconWithoutSession(t *testing.T) {
	server := &Server{
		logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
		webLoginEnabled:  true,
		webLoginPassword: "secret",
		webSessionToken:  "token-1",
	}
	handler := server.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, rec.Code)
	}
}

func TestLoginFlowSetsCookieAndAllowsAccess(t *testing.T) {
	server := &Server{
		logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
		webLoginEnabled:  true,
		webLoginPassword: "secret",
		webSessionToken:  "token-1",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/login", server.loginHandler)
	protected := server.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	mux.Handle("/chat", protected)

	form := url.Values{}
	form.Set("password", "secret")
	form.Set("next", "/chat")
	loginReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginRec := httptest.NewRecorder()
	mux.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, loginRec.Code)
	}
	if loginRec.Header().Get("Location") != "/chat" {
		t.Fatalf("expected redirect to /chat, got %q", loginRec.Header().Get("Location"))
	}
	cookies := loginRec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected session cookie after login")
	}

	accessReq := httptest.NewRequest(http.MethodGet, "/chat", nil)
	accessReq.Header.Set("Accept", "text/html")
	accessReq.AddCookie(cookies[0])
	accessRec := httptest.NewRecorder()
	mux.ServeHTTP(accessRec, accessReq)

	if accessRec.Code != http.StatusNoContent {
		t.Fatalf("expected protected resource success, got %d", accessRec.Code)
	}
}

func TestLoginFlowSharesCookieAcrossPreviewSubdomains(t *testing.T) {
	registry, err := newFileWorkspaceServiceRegistry("", "alter0.cn")
	if err != nil {
		t.Fatalf("new workspace service registry: %v", err)
	}
	server := &Server{
		logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
		webLoginEnabled:  true,
		webLoginPassword: "secret",
		webSessionToken:  "token-1",
		workspaceService: registry,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/login", server.loginHandler)
	protected := server.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	mux.Handle("/chat", protected)

	form := url.Values{}
	form.Set("password", "secret")
	form.Set("next", "/chat")
	loginReq := httptest.NewRequest(http.MethodPost, "https://alter0.cn/login", strings.NewReader(form.Encode()))
	loginReq.Host = "alter0.cn"
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginRec := httptest.NewRecorder()
	mux.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, loginRec.Code)
	}
	cookies := loginRec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected session cookie after login")
	}
	if cookies[0].Domain != "alter0.cn" {
		t.Fatalf("expected shared cookie domain alter0.cn, got %q", cookies[0].Domain)
	}

	accessReq := httptest.NewRequest(http.MethodGet, "https://63717262.alter0.cn/chat", nil)
	accessReq.Host = "63717262.alter0.cn"
	accessReq.Header.Set("Accept", "text/html")
	accessReq.AddCookie(cookies[0])
	accessRec := httptest.NewRecorder()
	mux.ServeHTTP(accessRec, accessReq)

	if accessRec.Code != http.StatusNoContent {
		t.Fatalf("expected preview host access success, got %d", accessRec.Code)
	}
}

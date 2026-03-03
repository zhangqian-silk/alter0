package web

import (
	"strings"
	"testing"
)

func TestMobileNewChatEntryReachable(t *testing.T) {
	html := readEmbeddedAsset(t, "static/chat.html")
	htmlMarkers := []string{
		`id="mobileNewChatButton"`,
		`id="newChatButton"`,
	}
	for _, marker := range htmlMarkers {
		if !strings.Contains(html, marker) {
			t.Fatalf("expected html marker %q", marker)
		}
	}

	styles := readEmbeddedAsset(t, "static/assets/chat.css")
	styleMarkers := []string{
		".mobile-new-chat {",
		".mobile-new-chat,",
	}
	for _, marker := range styleMarkers {
		if !strings.Contains(styles, marker) {
			t.Fatalf("expected style marker %q", marker)
		}
	}
}

func TestNewChatCreatesSessionAndSwitchesContext(t *testing.T) {
	script := readEmbeddedAsset(t, "static/assets/chat.js")
	markers := []string{
		"function startNewChatSession()",
		"createSession();",
		`navigateToRoute("chat");`,
		"newChatButton.addEventListener(\"click\", startNewChatSession);",
		"mobileNewChatButton.addEventListener(\"click\", startNewChatSession);",
	}
	for _, marker := range markers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}
}

func TestSessionListShowsEmptyAndLoadFailureFeedback(t *testing.T) {
	html := readEmbeddedAsset(t, "static/chat.html")
	if !strings.Contains(html, `id="sessionLoadError"`) {
		t.Fatalf("expected html marker %q", `id="sessionLoadError"`)
	}

	script := readEmbeddedAsset(t, "static/assets/chat.js")
	scriptMarkers := []string{
		"function loadSessionsFromStorage()",
		`state.sessionLoadError = "会话列表保存失败，请检查浏览器存储权限。";`,
		"state.sessionLoadError = `会话列表加载失败：${message}`;",
		`sessionEmpty.textContent = "会话列表为空，点击 New Chat 创建会话。";`,
	}
	for _, marker := range scriptMarkers {
		if !strings.Contains(script, marker) {
			t.Fatalf("expected script marker %q", marker)
		}
	}

	styles := readEmbeddedAsset(t, "static/assets/chat.css")
	if !strings.Contains(styles, ".session-error {") {
		t.Fatalf("expected style marker %q", ".session-error {")
	}
}

package web

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readEmbeddedAssetRaw(t *testing.T, assetPath string) string {
	t.Helper()
	if assetPath == "static/assets/chat.css" {
		parts := []string{
			"static/dist/legacy/chat.css",
			"static/dist/legacy/chat-core.css",
			"static/dist/legacy/chat-terminal.css",
			"static/dist/legacy/chat-routes.css",
		}
		var combined strings.Builder
		for _, part := range parts {
			content, err := webStaticFS.ReadFile(part)
			if err != nil {
				t.Fatalf("read embedded asset %s: %v", part, err)
			}
			combined.Write(content)
			combined.WriteByte('\n')
		}
		return combined.String()
	}
	if strings.HasPrefix(assetPath, "static/assets/") && strings.HasSuffix(assetPath, ".css") {
		assetPath = "static/dist/legacy/" + strings.TrimPrefix(assetPath, "static/assets/")
	}
	content, err := webStaticFS.ReadFile(assetPath)
	if err != nil {
		t.Fatalf("read embedded asset %s: %v", assetPath, err)
	}
	return string(content)
}

func readEmbeddedAsset(t *testing.T, assetPath string) string {
	t.Helper()
	return readEmbeddedAssetRaw(t, assetPath)
}

func readWorkspaceFile(t *testing.T, relativePath string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Clean(relativePath))
	if err != nil {
		t.Fatalf("read workspace file %s: %v", relativePath, err)
	}
	return string(content)
}

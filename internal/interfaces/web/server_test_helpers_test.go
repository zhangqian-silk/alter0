package web

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	shareddomain "alter0/internal/shared/domain"
)

func readEmbeddedAssetRaw(t *testing.T, assetPath string) string {
	t.Helper()
	if assetPath == "static/assets/chat.css" {
		parts := []string{
			"static/dist/legacy/chat.css",
			"static/dist/legacy/chat-core.css",
			"static/dist/legacy/chat-runtime.css",
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
	return strings.ReplaceAll(string(content), "\r\n", "\n")
}

type sequenceIDGenerator struct {
	ids  []string
	next int
}

func (g *sequenceIDGenerator) NewID() string {
	if g.next >= len(g.ids) {
		id := "generated-" + strconv.Itoa(g.next)
		g.next++
		return id
	}
	id := g.ids[g.next]
	g.next++
	return id
}

type stubOrchestrator struct {
	result shareddomain.OrchestrationResult
	err    error
	last   shareddomain.UnifiedMessage
}

func (s *stubOrchestrator) Handle(_ context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error) {
	s.last = msg
	if s.result.MessageID == "" {
		s.result.MessageID = msg.MessageID
	}
	if s.result.SessionID == "" {
		s.result.SessionID = msg.SessionID
	}
	return s.result, s.err
}

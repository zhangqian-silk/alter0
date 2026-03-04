package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestAgentMemoryHandlerReturnsUnifiedPayload(t *testing.T) {
	root := t.TempDir()
	dailyDir := filepath.Join(root, "memory")
	if err := os.MkdirAll(dailyDir, 0o755); err != nil {
		t.Fatalf("create daily dir: %v", err)
	}
	longTermPath := filepath.Join(dailyDir, "long-term", "MEMORY.md")
	if err := os.MkdirAll(filepath.Dir(longTermPath), 0o755); err != nil {
		t.Fatalf("create long-term dir: %v", err)
	}
	mandatoryPath := filepath.Join(root, "SOUL.md")

	if err := os.WriteFile(longTermPath, []byte("# Long-Term Memory\n- key: value"), 0o644); err != nil {
		t.Fatalf("write long-term memory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dailyDir, "2026-03-03.md"), []byte("# Daily Memory"), 0o644); err != nil {
		t.Fatalf("write daily memory: %v", err)
	}
	if err := os.WriteFile(mandatoryPath, []byte("# SOUL\n- tone: concise"), 0o644); err != nil {
		t.Fatalf("write mandatory memory: %v", err)
	}

	server := &Server{
		memory: newAgentMemoryService(AgentMemoryOptions{
			LongTermPath:         longTermPath,
			DailyDir:             dailyDir,
			MandatoryContextPath: mandatoryPath,
		}),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agent/memory", nil)
	rec := httptest.NewRecorder()
	server.agentMemoryHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	var body agentMemoryResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if !body.LongTerm.Exists {
		t.Fatal("expected long-term memory exists")
	}
	if body.LongTerm.Content == "" {
		t.Fatal("expected long-term memory content")
	}
	if len(body.Daily.Items) != 1 || body.Daily.Items[0].Date != "2026-03-03" {
		t.Fatalf("unexpected daily memory payload: %+v", body.Daily.Items)
	}
	if !body.Mandatory.Exists {
		t.Fatal("expected mandatory memory exists")
	}
}

func TestAgentMemoryHandlerMethodNotAllowed(t *testing.T) {
	server := &Server{
		memory: newAgentMemoryService(AgentMemoryOptions{}),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	req := httptest.NewRequest(http.MethodPost, "/api/agent/memory", nil)
	rec := httptest.NewRecorder()

	server.agentMemoryHandler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestAgentMemoryHandlerMissingFileReturnsEmptyState(t *testing.T) {
	root := t.TempDir()
	server := &Server{
		memory: newAgentMemoryService(AgentMemoryOptions{
			LongTermPath:         filepath.Join(root, "missing", "MEMORY.md"),
			DailyDir:             filepath.Join(root, "missing-memory"),
			MandatoryContextPath: filepath.Join(root, "missing", "SOUL.md"),
		}),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agent/memory", nil)
	rec := httptest.NewRecorder()
	server.agentMemoryHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var body agentMemoryResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.LongTerm.Exists || body.Mandatory.Exists {
		t.Fatal("expected missing files to produce empty state")
	}
	if len(body.Daily.Items) != 0 {
		t.Fatalf("expected empty daily memory list, got %d items", len(body.Daily.Items))
	}
}

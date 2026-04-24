package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	execdomain "alter0/internal/execution/domain"
	"alter0/internal/shared/infrastructure/observability"
)

type sessionAttachmentUploadResponse struct {
	Items []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		ContentType string `json:"content_type"`
		AssetURL    string `json:"asset_url"`
		PreviewURL  string `json:"preview_url"`
	} `json:"items"`
}

func TestSessionAttachmentHandlerStoresImagesInWorkspaceAndServesThem(t *testing.T) {
	t.Parallel()

	workspaceRoot := t.TempDir()
	server := &Server{
		idGenerator: &sequenceIDGenerator{ids: []string{"asset-1"}},
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		telemetry:   observability.NewTelemetry(),
		workspaceRoot: workspaceRoot,
	}

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/session-images/attachments", strings.NewReader(`{
		"attachments":[
			{
				"name":"diagram.png",
				"content_type":"image/png",
				"data_url":"data:image/png;base64,ZmFrZQ==",
				"preview_data_url":"data:image/webp;base64,QUJD"
			}
		]
	}`))
	rec := httptest.NewRecorder()
	server.sessionMessageListHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var payload sessionAttachmentUploadResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 uploaded asset, got %+v", payload.Items)
	}
	if payload.Items[0].ID != "asset-1" {
		t.Fatalf("expected asset id asset-1, got %+v", payload.Items[0])
	}

	originalPath := filepath.Join(workspaceRoot, ".alter0", "workspaces", "sessions", "session-images", "attachments", "asset-1", "original.png")
	previewPath := filepath.Join(workspaceRoot, ".alter0", "workspaces", "sessions", "session-images", "attachments", "asset-1", "preview.webp")
	if _, err := os.Stat(originalPath); err != nil {
		t.Fatalf("expected original attachment file, got %v", err)
	}
	if _, err := os.Stat(previewPath); err != nil {
		t.Fatalf("expected preview attachment file, got %v", err)
	}

	originalReq := httptest.NewRequest(http.MethodGet, payload.Items[0].AssetURL, nil)
	originalRec := httptest.NewRecorder()
	server.sessionMessageListHandler(originalRec, originalReq)
	if originalRec.Code != http.StatusOK {
		t.Fatalf("expected original asset 200, got %d", originalRec.Code)
	}
	if body := strings.TrimSpace(originalRec.Body.String()); body != "fake" {
		t.Fatalf("expected original asset body fake, got %q", body)
	}

	previewReq := httptest.NewRequest(http.MethodGet, payload.Items[0].PreviewURL, nil)
	previewRec := httptest.NewRecorder()
	server.sessionMessageListHandler(previewRec, previewReq)
	if previewRec.Code != http.StatusOK {
		t.Fatalf("expected preview asset 200, got %d", previewRec.Code)
	}
	if body := strings.TrimSpace(previewRec.Body.String()); body != "ABC" {
		t.Fatalf("expected preview asset body ABC, got %q", body)
	}
}

func TestSessionAttachmentHandlerStoresFilesWithoutPreviewVariant(t *testing.T) {
	t.Parallel()

	workspaceRoot := t.TempDir()
	server := &Server{
		idGenerator:   &sequenceIDGenerator{ids: []string{"asset-file-1"}},
		logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		telemetry:     observability.NewTelemetry(),
		workspaceRoot: workspaceRoot,
	}

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/session-files/attachments", strings.NewReader(`{
		"attachments":[
			{
				"name":"requirements.md",
				"content_type":"text/markdown",
				"data_url":"data:text/markdown;base64,IyBSZXF1aXJlbWVudHMK"
			}
		]
	}`))
	rec := httptest.NewRecorder()
	server.sessionMessageListHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var payload sessionAttachmentUploadResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 uploaded asset, got %+v", payload.Items)
	}
	if payload.Items[0].PreviewURL != payload.Items[0].AssetURL {
		t.Fatalf("expected file preview url to fall back to original asset url, got %+v", payload.Items[0])
	}

	originalPath := filepath.Join(workspaceRoot, ".alter0", "workspaces", "sessions", "session-files", "attachments", "asset-file-1", "original.md")
	if _, err := os.Stat(originalPath); err != nil {
		t.Fatalf("expected original file attachment, got %v", err)
	}

	previewReq := httptest.NewRequest(http.MethodGet, payload.Items[0].PreviewURL, nil)
	previewRec := httptest.NewRecorder()
	server.sessionMessageListHandler(previewRec, previewReq)
	if previewRec.Code != http.StatusOK {
		t.Fatalf("expected preview asset 200, got %d", previewRec.Code)
	}
	if body := strings.TrimSpace(previewRec.Body.String()); body != "# Requirements" {
		t.Fatalf("expected file preview body '# Requirements', got %q", body)
	}
}

func TestMessageHandlerEncodesWorkspaceAttachmentReferences(t *testing.T) {
	t.Parallel()

	workspaceRoot := t.TempDir()
	orchestrator := &stubWebOrchestrator{}
	server := &Server{
		orchestrator:  orchestrator,
		telemetry:     observability.NewTelemetry(),
		idGenerator:   &sequenceIDGenerator{ids: []string{"asset-1", "message-1", "trace-1"}},
		logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		workspaceRoot: workspaceRoot,
	}

	uploadReq := httptest.NewRequest(http.MethodPost, "/api/sessions/session-images/attachments", strings.NewReader(`{
		"attachments":[
			{
				"name":"diagram.png",
				"content_type":"image/png",
				"data_url":"data:image/png;base64,ZmFrZQ==",
				"preview_data_url":"data:image/webp;base64,QUJD"
			}
		]
	}`))
	uploadRec := httptest.NewRecorder()
	server.sessionMessageListHandler(uploadRec, uploadReq)
	if uploadRec.Code != http.StatusOK {
		t.Fatalf("expected upload 200, got %d: %s", uploadRec.Code, uploadRec.Body.String())
	}

	var uploadPayload sessionAttachmentUploadResponse
	if err := json.Unmarshal(uploadRec.Body.Bytes(), &uploadPayload); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/messages", strings.NewReader(`{
		"session_id":"session-images",
		"content":"请分析这张图",
		"attachments":[
			{
				"id":"`+uploadPayload.Items[0].ID+`",
				"name":"diagram.png",
				"content_type":"image/png",
				"asset_url":"`+uploadPayload.Items[0].AssetURL+`",
				"preview_url":"`+uploadPayload.Items[0].PreviewURL+`"
			}
		]
	}`))
	rec := httptest.NewRecorder()
	server.messageHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	attachments := execdomain.DecodeUserImageAttachments(orchestrator.lastMessage.Metadata)
	if len(attachments) != 1 {
		t.Fatalf("expected 1 attachment in metadata, got %+v", attachments)
	}
	if attachments[0].WorkspacePath == "" {
		t.Fatalf("expected workspace path in metadata, got %+v", attachments[0])
	}
	if attachments[0].DataURL != "" {
		t.Fatalf("expected message metadata to avoid inline data URLs, got %+v", attachments[0])
	}
	if attachments[0].AssetURL != uploadPayload.Items[0].AssetURL {
		t.Fatalf("expected asset URL preserved, got %+v", attachments[0])
	}
	if attachments[0].PreviewURL != uploadPayload.Items[0].PreviewURL {
		t.Fatalf("expected preview URL preserved, got %+v", attachments[0])
	}
}

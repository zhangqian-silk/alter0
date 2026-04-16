package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreviewRegistrationCRUD(t *testing.T) {
	repoPath := preparePreviewRepo(t, "preview control")
	registry, err := newFileSessionPreviewRegistry(filepath.Join(t.TempDir(), "session-previews.json"), "alter0.cn")
	if err != nil {
		t.Fatalf("new preview registry: %v", err)
	}
	server := &Server{
		logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		previewRegistry: registry,
	}

	putReq := httptest.NewRequest(http.MethodPut, "/api/control/previews/session-preview-control", strings.NewReader(`{"repository_path":"`+repoPath+`"}`))
	putRec := httptest.NewRecorder()
	server.previewItemHandler(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("expected put 200, got %d: %s", putRec.Code, putRec.Body.String())
	}

	var created sessionPreviewRegistration
	if err := json.NewDecoder(putRec.Body).Decode(&created); err != nil {
		t.Fatalf("decode put response: %v", err)
	}
	if created.SessionID != "session-preview-control" {
		t.Fatalf("expected session id, got %+v", created)
	}
	if created.ShortHash == "" || created.URL == "" || created.DistPath == "" {
		t.Fatalf("expected preview routing fields, got %+v", created)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/control/previews", nil)
	listRec := httptest.NewRecorder()
	server.previewCollectionHandler(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	var listResp struct {
		Items []sessionPreviewRegistration `json:"items"`
	}
	if err := json.NewDecoder(listRec.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listResp.Items) != 1 || listResp.Items[0].SessionID != "session-preview-control" {
		t.Fatalf("unexpected list response %+v", listResp.Items)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/control/previews/session-preview-control", nil)
	deleteRec := httptest.NewRecorder()
	server.previewItemHandler(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("expected delete 200, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}

	listAfterReq := httptest.NewRequest(http.MethodGet, "/api/control/previews", nil)
	listAfterRec := httptest.NewRecorder()
	server.previewCollectionHandler(listAfterRec, listAfterReq)
	if listAfterRec.Code != http.StatusOK {
		t.Fatalf("expected list after delete 200, got %d", listAfterRec.Code)
	}
	var listAfterResp struct {
		Items []sessionPreviewRegistration `json:"items"`
	}
	if err := json.NewDecoder(listAfterRec.Body).Decode(&listAfterResp); err != nil {
		t.Fatalf("decode list after delete: %v", err)
	}
	if len(listAfterResp.Items) != 0 {
		t.Fatalf("expected empty preview registry, got %+v", listAfterResp.Items)
	}
}

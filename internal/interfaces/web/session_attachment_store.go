package web

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	execdomain "alter0/internal/execution/domain"
)

const conversationSessionAttachmentDirName = "attachments"

type conversationAttachmentManifest struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	ContentType        string `json:"content_type"`
	PreviewContentType string `json:"preview_content_type"`
	OriginalFileName   string `json:"original_file_name,omitempty"`
	PreviewFileName    string `json:"preview_file_name,omitempty"`
	Size               int64  `json:"size"`
}

type conversationAttachmentAsset struct {
	ID            string
	Name          string
	ContentType   string
	Size          int64
	WorkspacePath string
	PreviewPath   string
	AssetURL      string
	PreviewURL    string
}

type conversationAttachmentUploadRequest struct {
	Attachments []messageAttachmentRequest `json:"attachments,omitempty"`
}

type conversationAttachmentUploadResponse struct {
	Items []conversationAttachmentUploadItem `json:"items"`
}

type conversationAttachmentUploadItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	AssetURL    string `json:"asset_url"`
	PreviewURL  string `json:"preview_url"`
}

func (s *Server) handleSessionAttachmentUpload(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	defer r.Body.Close()

	var req conversationAttachmentUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if len(req.Attachments) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "attachments are required"})
		return
	}
	items, err := s.storeConversationAttachmentBatch(sessionID, req.Attachments)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	response := conversationAttachmentUploadResponse{
		Items: make([]conversationAttachmentUploadItem, 0, len(items)),
	}
	for _, item := range items {
		response.Items = append(response.Items, conversationAttachmentUploadItem{
			ID:          item.ID,
			Name:        item.Name,
			ContentType: item.ContentType,
			Size:        item.Size,
			AssetURL:    item.AssetURL,
			PreviewURL:  item.PreviewURL,
		})
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleSessionAttachmentRead(w http.ResponseWriter, r *http.Request, sessionID string, attachmentID string, variant string) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	asset, err := s.resolveConversationAttachment(sessionID, attachmentID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "attachment not found"})
		return
	}
	path := asset.WorkspacePath
	contentType := asset.ContentType
	if variant == "preview" {
		path = asset.PreviewPath
		contentType = readConversationPreviewContentType(sessionID, attachmentID, asset)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "attachment not found"})
		return
	}
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func readConversationPreviewContentType(sessionID string, attachmentID string, asset conversationAttachmentAsset) string {
	manifest, err := readConversationAttachmentManifestFromAsset(asset)
	if err != nil {
		return asset.ContentType
	}
	if strings.TrimSpace(manifest.PreviewContentType) != "" {
		return strings.TrimSpace(manifest.PreviewContentType)
	}
	return asset.ContentType
}

func (s *Server) normalizeConversationMessageAttachments(sessionID string, values []messageAttachmentRequest) ([]execdomain.UserImageAttachment, error) {
	if len(values) == 0 {
		return nil, nil
	}
	items := make([]execdomain.UserImageAttachment, 0, len(values))
	for _, value := range values {
		item, err := s.materializeConversationAttachment(sessionID, value)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return execdomain.NormalizeUserImageAttachments(items), nil
}

func (s *Server) materializeConversationAttachment(sessionID string, value messageAttachmentRequest) (execdomain.UserImageAttachment, error) {
	if strings.TrimSpace(value.ID) != "" {
		asset, err := s.resolveConversationAttachment(sessionID, value.ID)
		if err != nil {
			return execdomain.UserImageAttachment{}, err
		}
		return execdomain.UserImageAttachment{
			ID:            asset.ID,
			Name:          asset.Name,
			ContentType:   asset.ContentType,
			WorkspacePath: asset.WorkspacePath,
			AssetURL:      asset.AssetURL,
			PreviewURL:    asset.PreviewURL,
		}, nil
	}
	if strings.TrimSpace(value.DataURL) == "" {
		return execdomain.UserImageAttachment{}, errors.New("attachment payload is required")
	}
	if s == nil || strings.TrimSpace(s.workspaceRoot) == "" {
		return execdomain.UserImageAttachment{
			Name:        strings.TrimSpace(value.Name),
			ContentType: strings.TrimSpace(value.ContentType),
			DataURL:     strings.TrimSpace(value.DataURL),
		}, nil
	}
	asset, err := s.storeConversationAttachment(sessionID, value)
	if err != nil {
		return execdomain.UserImageAttachment{}, err
	}
	return execdomain.UserImageAttachment{
		ID:            asset.ID,
		Name:          asset.Name,
		ContentType:   asset.ContentType,
		WorkspacePath: asset.WorkspacePath,
		AssetURL:      asset.AssetURL,
		PreviewURL:    asset.PreviewURL,
	}, nil
}

func (s *Server) storeConversationAttachmentBatch(sessionID string, values []messageAttachmentRequest) ([]conversationAttachmentAsset, error) {
	items := make([]conversationAttachmentAsset, 0, len(values))
	for _, value := range values {
		item, err := s.storeConversationAttachment(sessionID, value)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Server) storeConversationAttachment(sessionID string, value messageAttachmentRequest) (conversationAttachmentAsset, error) {
	if s == nil || strings.TrimSpace(s.workspaceRoot) == "" {
		return conversationAttachmentAsset{}, errors.New("workspace root unavailable")
	}
	contentType := strings.TrimSpace(value.ContentType)
	data, err := decodeConversationAttachmentDataURL(value.DataURL)
	if err != nil {
		return conversationAttachmentAsset{}, err
	}
	previewDataURL := strings.TrimSpace(value.PreviewDataURL)
	if previewDataURL == "" {
		previewDataURL = strings.TrimSpace(value.DataURL)
	}
	previewData, err := decodeConversationAttachmentDataURL(previewDataURL)
	if err != nil {
		return conversationAttachmentAsset{}, err
	}
	previewContentType := conversationAttachmentContentType(previewDataURL, contentType)
	assetID := sanitizeWorkspaceSegment(s.idGenerator.NewID())
	if assetID == "" {
		return conversationAttachmentAsset{}, errors.New("failed to allocate attachment id")
	}
	dir := conversationAttachmentDir(s.workspaceRoot, sessionID, assetID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return conversationAttachmentAsset{}, fmt.Errorf("prepare attachment dir: %w", err)
	}
	originalFileName := conversationAttachmentFileName("original", contentType)
	previewFileName := conversationAttachmentFileName("preview", previewContentType)
	originalPath := filepath.Join(dir, originalFileName)
	previewPath := filepath.Join(dir, previewFileName)
	if err := os.WriteFile(originalPath, data, 0o644); err != nil {
		return conversationAttachmentAsset{}, fmt.Errorf("write attachment file: %w", err)
	}
	if err := os.WriteFile(previewPath, previewData, 0o644); err != nil {
		return conversationAttachmentAsset{}, fmt.Errorf("write attachment preview: %w", err)
	}
	manifest := conversationAttachmentManifest{
		ID:                 assetID,
		Name:               strings.TrimSpace(value.Name),
		ContentType:        contentType,
		PreviewContentType: previewContentType,
		OriginalFileName:   originalFileName,
		PreviewFileName:    previewFileName,
		Size:               int64(len(data)),
	}
	if err := writeConversationAttachmentManifest(dir, manifest); err != nil {
		return conversationAttachmentAsset{}, err
	}
	return conversationAttachmentAsset{
		ID:            assetID,
		Name:          manifest.Name,
		ContentType:   manifest.ContentType,
		Size:          manifest.Size,
		WorkspacePath: originalPath,
		PreviewPath:   previewPath,
		AssetURL:      conversationAttachmentURL(sessionID, assetID, "original"),
		PreviewURL:    conversationAttachmentURL(sessionID, assetID, "preview"),
	}, nil
}

func (s *Server) resolveConversationAttachment(sessionID string, attachmentID string) (conversationAttachmentAsset, error) {
	dir := conversationAttachmentDir(s.workspaceRoot, sessionID, attachmentID)
	manifest, err := readConversationAttachmentManifest(dir)
	if err != nil {
		return conversationAttachmentAsset{}, err
	}
	return conversationAttachmentAsset{
		ID:            manifest.ID,
		Name:          manifest.Name,
		ContentType:   manifest.ContentType,
		Size:          manifest.Size,
		WorkspacePath: filepath.Join(dir, conversationAttachmentStoredFileName(manifest.OriginalFileName, "original", manifest.ContentType)),
		PreviewPath:   filepath.Join(dir, conversationAttachmentStoredFileName(manifest.PreviewFileName, "preview", manifest.PreviewContentType)),
		AssetURL:      conversationAttachmentURL(sessionID, manifest.ID, "original"),
		PreviewURL:    conversationAttachmentURL(sessionID, manifest.ID, "preview"),
	}, nil
}

func readConversationAttachmentManifest(dir string) (conversationAttachmentManifest, error) {
	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return conversationAttachmentManifest{}, err
	}
	var manifest conversationAttachmentManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return conversationAttachmentManifest{}, err
	}
	return manifest, nil
}

func readConversationAttachmentManifestFromAsset(asset conversationAttachmentAsset) (conversationAttachmentManifest, error) {
	return readConversationAttachmentManifest(filepath.Dir(asset.WorkspacePath))
}

func writeConversationAttachmentManifest(dir string, manifest conversationAttachmentManifest) error {
	data, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("encode attachment manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0o644); err != nil {
		return fmt.Errorf("write attachment manifest: %w", err)
	}
	return nil
}

func conversationAttachmentDir(baseDir string, sessionID string, attachmentID string) string {
	return filepath.Join(baseDir, ".alter0", "workspaces", "sessions", sanitizeWorkspaceSegment(sessionID), conversationSessionAttachmentDirName, sanitizeWorkspaceSegment(attachmentID))
}

func conversationAttachmentURL(sessionID string, attachmentID string, variant string) string {
	return "/api/sessions/" + sanitizeWorkspaceSegment(sessionID) + "/attachments/" + sanitizeWorkspaceSegment(attachmentID) + "/" + variant
}

func decodeConversationAttachmentDataURL(raw string) ([]byte, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, errors.New("attachment data url is empty")
	}
	if !strings.HasPrefix(value, "data:") {
		return nil, errors.New("attachment data url is invalid")
	}
	parts := strings.SplitN(value, ",", 2)
	if len(parts) != 2 {
		return nil, errors.New("attachment data url is invalid")
	}
	if !strings.HasSuffix(strings.ToLower(parts[0]), ";base64") {
		return nil, errors.New("attachment data url is invalid")
	}
	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode attachment data url: %w", err)
	}
	return decoded, nil
}

func conversationAttachmentContentType(raw string, fallback string) string {
	value := strings.TrimSpace(raw)
	if strings.HasPrefix(strings.ToLower(value), "data:") {
		parts := strings.SplitN(value, ",", 2)
		if len(parts) == 2 {
			meta := strings.TrimPrefix(parts[0], "data:")
			meta = strings.TrimSuffix(meta, ";base64")
			if strings.TrimSpace(meta) != "" {
				return strings.TrimSpace(meta)
			}
		}
	}
	return strings.TrimSpace(fallback)
}

func conversationAttachmentFileName(base string, contentType string) string {
	return base + conversationAttachmentFileExtension(contentType)
}

func conversationAttachmentStoredFileName(stored string, base string, contentType string) string {
	if strings.TrimSpace(stored) != "" {
		return strings.TrimSpace(stored)
	}
	return conversationAttachmentFileName(base, contentType)
}

func conversationAttachmentFileExtension(contentType string) string {
	normalized := strings.TrimSpace(contentType)
	if normalized == "" {
		return ""
	}
	extensions, err := mime.ExtensionsByType(normalized)
	if err != nil || len(extensions) == 0 {
		return ""
	}
	return extensions[0]
}

package application

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	shareddomain "alter0/internal/shared/domain"
)

const (
	sessionAttachmentDirName          = "attachments"
	maxAssistantImageDownloadBytes    = 12 * 1024 * 1024
	assistantImageLocalizationTimeout = 12 * time.Second
)

var markdownImagePattern = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)

type assistantImageFetcher interface {
	Do(req *http.Request) (*http.Response, error)
}

type assistantAttachmentManifest struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	ContentType        string `json:"content_type"`
	PreviewContentType string `json:"preview_content_type"`
	OriginalFileName   string `json:"original_file_name,omitempty"`
	PreviewFileName    string `json:"preview_file_name,omitempty"`
	RemoteURL          string `json:"remote_url,omitempty"`
	Size               int64  `json:"size"`
}

func (s *SessionPersistenceService) materializeAssistantImages(
	sessionID string,
	result shareddomain.OrchestrationResult,
) shareddomain.OrchestrationResult {
	if s == nil || strings.TrimSpace(s.workspaceRoot) == "" || strings.TrimSpace(sessionID) == "" {
		return result
	}
	result.Output = s.localizeMarkdownImageContent(sessionID, result.Output)
	if len(result.ProcessSteps) == 0 {
		return result
	}
	nextSteps := make([]shareddomain.ProcessStep, 0, len(result.ProcessSteps))
	for _, step := range result.ProcessSteps {
		step.Detail = s.localizeMarkdownImageContent(sessionID, step.Detail)
		nextSteps = append(nextSteps, step)
	}
	result.ProcessSteps = nextSteps
	return result
}

func (s *SessionPersistenceService) localizeMarkdownImageContent(sessionID string, content string) string {
	if strings.TrimSpace(content) == "" {
		return content
	}
	return markdownImagePattern.ReplaceAllStringFunc(content, func(match string) string {
		submatches := markdownImagePattern.FindStringSubmatch(match)
		if len(submatches) != 3 {
			return match
		}
		altText := submatches[1]
		rawTarget := strings.TrimSpace(submatches[2])
		rawURL, suffix := splitMarkdownImageTarget(rawTarget)
		localURL, ok := s.localizeAssistantImageURL(sessionID, rawURL)
		if !ok {
			return match
		}
		if suffix != "" {
			return "![" + altText + "](" + localURL + " " + suffix + ")"
		}
		return "![" + altText + "](" + localURL + ")"
	})
}

func splitMarkdownImageTarget(value string) (string, string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", ""
	}
	if strings.HasPrefix(trimmed, "<") {
		if end := strings.Index(trimmed, ">"); end > 0 {
			return trimmed[1:end], strings.TrimSpace(trimmed[end+1:])
		}
	}
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return "", ""
	}
	rawURL := fields[0]
	suffix := strings.TrimSpace(strings.TrimPrefix(trimmed, rawURL))
	return rawURL, suffix
}

func (s *SessionPersistenceService) localizeAssistantImageURL(sessionID string, rawURL string) (string, bool) {
	normalized := strings.TrimSpace(strings.Trim(rawURL, "<>"))
	if normalized == "" || strings.HasPrefix(normalized, "/api/sessions/") || strings.HasPrefix(normalized, "data:image/") {
		return normalized, normalized != ""
	}
	parsed, err := url.Parse(normalized)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || strings.TrimSpace(parsed.Host) == "" {
		return "", false
	}
	assetID := buildAssistantAttachmentID(normalized)
	dir := sessionAttachmentDir(s.workspaceRoot, sessionID, assetID)
	if manifest, readErr := readAssistantAttachmentManifest(dir); readErr == nil {
		if strings.TrimSpace(manifest.ID) != "" {
			return sessionAttachmentURL(sessionID, manifest.ID, "original"), true
		}
	}
	client := s.httpClient
	if client == nil {
		client = &http.Client{Timeout: assistantImageLocalizationTimeout}
	}
	req, reqErr := http.NewRequest(http.MethodGet, normalized, nil)
	if reqErr != nil {
		s.logAssistantImageLocalizationFailure(sessionID, normalized, reqErr)
		return "", false
	}
	resp, doErr := client.Do(req)
	if doErr != nil {
		s.logAssistantImageLocalizationFailure(sessionID, normalized, doErr)
		return "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		s.logAssistantImageLocalizationFailure(sessionID, normalized, fmt.Errorf("unexpected status %d", resp.StatusCode))
		return "", false
	}
	data, readErr := io.ReadAll(io.LimitReader(resp.Body, maxAssistantImageDownloadBytes+1))
	if readErr != nil {
		s.logAssistantImageLocalizationFailure(sessionID, normalized, readErr)
		return "", false
	}
	if len(data) == 0 || len(data) > maxAssistantImageDownloadBytes {
		s.logAssistantImageLocalizationFailure(sessionID, normalized, fmt.Errorf("image download exceeds limit"))
		return "", false
	}
	contentType := normalizeAssistantImageContentType(resp.Header.Get("Content-Type"), data)
	if !strings.HasPrefix(strings.ToLower(contentType), "image/") {
		s.logAssistantImageLocalizationFailure(sessionID, normalized, fmt.Errorf("unsupported content type %q", contentType))
		return "", false
	}
	if writeErr := writeAssistantAttachmentAsset(dir, assistantAttachmentManifest{
		ID:                 assetID,
		Name:               assistantAttachmentName(parsed),
		ContentType:        contentType,
		PreviewContentType: contentType,
		OriginalFileName:   sessionAttachmentFileName("original", contentType),
		PreviewFileName:    sessionAttachmentFileName("preview", contentType),
		RemoteURL:          normalized,
		Size:               int64(len(data)),
	}, data); writeErr != nil {
		s.logAssistantImageLocalizationFailure(sessionID, normalized, writeErr)
		return "", false
	}
	return sessionAttachmentURL(sessionID, assetID, "original"), true
}

func (s *SessionPersistenceService) logAssistantImageLocalizationFailure(sessionID string, rawURL string, err error) {
	if s == nil || s.logger == nil || err == nil {
		return
	}
	s.logger.Warn(
		"failed to localize assistant image",
		"session_id", strings.TrimSpace(sessionID),
		"url", strings.TrimSpace(rawURL),
		"error", strings.TrimSpace(err.Error()),
	)
}

func buildAssistantAttachmentID(rawURL string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(rawURL)))
	return "assistant-" + hex.EncodeToString(sum[:6])
}

func sessionAttachmentDir(baseDir string, sessionID string, attachmentID string) string {
	return filepath.Join(baseDir, ".alter0", "workspaces", "sessions", sanitizeAttachmentSegment(sessionID), sessionAttachmentDirName, sanitizeAttachmentSegment(attachmentID))
}

func sessionAttachmentURL(sessionID string, attachmentID string, variant string) string {
	return "/api/sessions/" + sanitizeAttachmentSegment(sessionID) + "/attachments/" + sanitizeAttachmentSegment(attachmentID) + "/" + variant
}

func sanitizeAttachmentSegment(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	var builder strings.Builder
	builder.Grow(len(trimmed))
	for _, char := range trimmed {
		switch {
		case char >= 'a' && char <= 'z':
			builder.WriteRune(char)
		case char >= 'A' && char <= 'Z':
			builder.WriteRune(char + 32)
		case char >= '0' && char <= '9':
			builder.WriteRune(char)
		case char == '-' || char == '_' || char == '.':
			builder.WriteRune(char)
		default:
			builder.WriteByte('-')
		}
	}
	return strings.Trim(builder.String(), "-")
}

func readAssistantAttachmentManifest(dir string) (assistantAttachmentManifest, error) {
	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return assistantAttachmentManifest{}, err
	}
	var manifest assistantAttachmentManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return assistantAttachmentManifest{}, err
	}
	return manifest, nil
}

func writeAssistantAttachmentAsset(dir string, manifest assistantAttachmentManifest, data []byte) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("prepare assistant attachment dir: %w", err)
	}
	originalPath := filepath.Join(dir, manifest.OriginalFileName)
	previewPath := filepath.Join(dir, manifest.PreviewFileName)
	if err := os.WriteFile(originalPath, data, 0o644); err != nil {
		return fmt.Errorf("write assistant attachment original: %w", err)
	}
	if err := os.WriteFile(previewPath, data, 0o644); err != nil {
		return fmt.Errorf("write assistant attachment preview: %w", err)
	}
	payload, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("encode assistant attachment manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), payload, 0o644); err != nil {
		return fmt.Errorf("write assistant attachment manifest: %w", err)
	}
	return nil
}

func assistantAttachmentName(parsed *url.URL) string {
	if parsed == nil {
		return "assistant-image"
	}
	name := filepath.Base(strings.TrimSpace(parsed.Path))
	if name == "" || name == "." || name == "/" {
		return "assistant-image"
	}
	return name
}

func normalizeAssistantImageContentType(headerValue string, data []byte) string {
	contentType := strings.ToLower(strings.TrimSpace(strings.Split(strings.TrimSpace(headerValue), ";")[0]))
	if strings.HasPrefix(contentType, "image/") {
		return contentType
	}
	return strings.ToLower(strings.TrimSpace(http.DetectContentType(data)))
}

func sessionAttachmentFileName(base string, contentType string) string {
	return base + sessionAttachmentFileExtension(contentType)
}

func sessionAttachmentFileExtension(contentType string) string {
	extensions, err := mime.ExtensionsByType(strings.TrimSpace(contentType))
	if err != nil || len(extensions) == 0 {
		return ""
	}
	return extensions[0]
}

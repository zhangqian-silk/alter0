package domain

import (
	"encoding/json"
	"strings"
)

const UserImageAttachmentsMetadataKey = "alter0.user_input.image_attachments"

type UserImageAttachment struct {
	ID            string `json:"id,omitempty"`
	Name          string `json:"name"`
	ContentType   string `json:"content_type"`
	DataURL       string `json:"data_url,omitempty"`
	WorkspacePath string `json:"workspace_path,omitempty"`
	AssetURL      string `json:"asset_url,omitempty"`
	PreviewURL    string `json:"preview_url,omitempty"`
}

func EncodeUserImageAttachments(items []UserImageAttachment) (string, error) {
	normalized := NormalizeUserImageAttachments(items)
	if len(normalized) == 0 {
		return "", nil
	}
	raw, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func DecodeUserImageAttachments(metadata map[string]string) []UserImageAttachment {
	if len(metadata) == 0 {
		return nil
	}
	raw := strings.TrimSpace(metadata[UserImageAttachmentsMetadataKey])
	if raw == "" {
		return nil
	}
	var items []UserImageAttachment
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	return NormalizeUserImageAttachments(items)
}

func NormalizeUserImageAttachments(items []UserImageAttachment) []UserImageAttachment {
	if len(items) == 0 {
		return nil
	}
	out := make([]UserImageAttachment, 0, len(items))
	for _, item := range items {
		id := strings.TrimSpace(item.ID)
		name := strings.TrimSpace(item.Name)
		contentType := strings.TrimSpace(item.ContentType)
		dataURL := strings.TrimSpace(item.DataURL)
		workspacePath := strings.TrimSpace(item.WorkspacePath)
		assetURL := strings.TrimSpace(item.AssetURL)
		previewURL := strings.TrimSpace(item.PreviewURL)
		if !strings.HasPrefix(strings.ToLower(contentType), "image/") {
			continue
		}
		if dataURL == "" && workspacePath == "" && assetURL == "" {
			continue
		}
		out = append(out, UserImageAttachment{
			ID:            id,
			Name:          name,
			ContentType:   contentType,
			DataURL:       dataURL,
			WorkspacePath: workspacePath,
			AssetURL:      assetURL,
			PreviewURL:    previewURL,
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

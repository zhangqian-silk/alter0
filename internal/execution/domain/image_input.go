package domain

import (
	"encoding/json"
	"strings"
)

const (
	UserAttachmentsMetadataKey      = "alter0.user_input.attachments"
	UserImageAttachmentsMetadataKey = "alter0.user_input.image_attachments"
	UserAttachmentKindImage         = "image"
	UserAttachmentKindFile          = "file"
)

type UserAttachment struct {
	ID            string `json:"id,omitempty"`
	Kind          string `json:"kind,omitempty"`
	Name          string `json:"name"`
	ContentType   string `json:"content_type"`
	DataURL       string `json:"data_url,omitempty"`
	WorkspacePath string `json:"workspace_path,omitempty"`
	AssetURL      string `json:"asset_url,omitempty"`
	PreviewURL    string `json:"preview_url,omitempty"`
}

type UserImageAttachment struct {
	ID            string `json:"id,omitempty"`
	Name          string `json:"name"`
	ContentType   string `json:"content_type"`
	DataURL       string `json:"data_url,omitempty"`
	WorkspacePath string `json:"workspace_path,omitempty"`
	AssetURL      string `json:"asset_url,omitempty"`
	PreviewURL    string `json:"preview_url,omitempty"`
}

func EncodeUserAttachments(items []UserAttachment) (string, error) {
	normalized := NormalizeUserAttachments(items)
	if len(normalized) == 0 {
		return "", nil
	}
	raw, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func DecodeUserAttachments(metadata map[string]string) []UserAttachment {
	if len(metadata) == 0 {
		return nil
	}
	if raw := strings.TrimSpace(metadata[UserAttachmentsMetadataKey]); raw != "" {
		var items []UserAttachment
		if err := json.Unmarshal([]byte(raw), &items); err == nil {
			return NormalizeUserAttachments(items)
		}
	}
	raw := strings.TrimSpace(metadata[UserImageAttachmentsMetadataKey])
	if raw == "" {
		return nil
	}
	var items []UserImageAttachment
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	return NormalizeUserAttachments(userImageAttachmentsToGeneric(items))
}

func NormalizeUserAttachments(items []UserAttachment) []UserAttachment {
	if len(items) == 0 {
		return nil
	}
	out := make([]UserAttachment, 0, len(items))
	for _, item := range items {
		id := strings.TrimSpace(item.ID)
		kind := normalizeUserAttachmentKind(item.Kind, item.ContentType)
		name := strings.TrimSpace(item.Name)
		contentType := strings.TrimSpace(item.ContentType)
		dataURL := strings.TrimSpace(item.DataURL)
		workspacePath := strings.TrimSpace(item.WorkspacePath)
		assetURL := strings.TrimSpace(item.AssetURL)
		previewURL := strings.TrimSpace(item.PreviewURL)
		if contentType == "" {
			continue
		}
		if dataURL == "" && workspacePath == "" && assetURL == "" {
			continue
		}
		out = append(out, UserAttachment{
			ID:            id,
			Kind:          kind,
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
	return NormalizeUserImageAttachments(FilterUserImageAttachments(DecodeUserAttachments(metadata)))
}

func NormalizeUserImageAttachments(items []UserImageAttachment) []UserImageAttachment {
	if len(items) == 0 {
		return nil
	}
	return FilterUserImageAttachments(userImageAttachmentsToGeneric(items))
}

func FilterUserImageAttachments(items []UserAttachment) []UserImageAttachment {
	if len(items) == 0 {
		return nil
	}
	out := make([]UserImageAttachment, 0, len(items))
	for _, item := range NormalizeUserAttachments(items) {
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(item.ContentType)), "image/") {
			continue
		}
		out = append(out, UserImageAttachment{
			ID:            item.ID,
			Name:          item.Name,
			ContentType:   item.ContentType,
			DataURL:       item.DataURL,
			WorkspacePath: item.WorkspacePath,
			AssetURL:      item.AssetURL,
			PreviewURL:    item.PreviewURL,
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeUserAttachmentKind(kind string, contentType string) string {
	normalizedKind := strings.ToLower(strings.TrimSpace(kind))
	if normalizedKind == UserAttachmentKindImage || normalizedKind == UserAttachmentKindFile {
		return normalizedKind
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(contentType)), "image/") {
		return UserAttachmentKindImage
	}
	return UserAttachmentKindFile
}

func userImageAttachmentsToGeneric(items []UserImageAttachment) []UserAttachment {
	if len(items) == 0 {
		return nil
	}
	out := make([]UserAttachment, 0, len(items))
	for _, item := range items {
		out = append(out, UserAttachment{
			ID:            item.ID,
			Kind:          UserAttachmentKindImage,
			Name:          item.Name,
			ContentType:   item.ContentType,
			DataURL:       item.DataURL,
			WorkspacePath: item.WorkspacePath,
			AssetURL:      item.AssetURL,
			PreviewURL:    item.PreviewURL,
		})
	}
	return out
}

package domain

import (
	"encoding/json"
	"strings"
)

const UserImageAttachmentsMetadataKey = "alter0.user_input.image_attachments"

type UserImageAttachment struct {
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	DataURL     string `json:"data_url"`
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
		name := strings.TrimSpace(item.Name)
		contentType := strings.TrimSpace(item.ContentType)
		dataURL := strings.TrimSpace(item.DataURL)
		if !strings.HasPrefix(strings.ToLower(contentType), "image/") || dataURL == "" {
			continue
		}
		out = append(out, UserImageAttachment{
			Name:        name,
			ContentType: contentType,
			DataURL:     dataURL,
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

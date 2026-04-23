package domain

import "testing"

func TestEncodeDecodeUserImageAttachmentsRoundTrip(t *testing.T) {
	t.Parallel()

	raw, err := EncodeUserImageAttachments([]UserImageAttachment{
		{
			Name:        "diagram.png",
			ContentType: "image/png",
			WorkspacePath: "/tmp/session-images/diagram.png",
			AssetURL:      "/api/sessions/session-1/attachments/asset-1/original",
			PreviewURL:    "/api/sessions/session-1/attachments/asset-1/preview",
		},
	})
	if err != nil {
		t.Fatalf("EncodeUserImageAttachments() error = %v", err)
	}

	items := DecodeUserImageAttachments(map[string]string{
		UserImageAttachmentsMetadataKey: raw,
	})
	if len(items) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(items))
	}
	if items[0].Name != "diagram.png" ||
		items[0].ContentType != "image/png" ||
		items[0].WorkspacePath != "/tmp/session-images/diagram.png" ||
		items[0].AssetURL != "/api/sessions/session-1/attachments/asset-1/original" ||
		items[0].PreviewURL != "/api/sessions/session-1/attachments/asset-1/preview" {
		t.Fatalf("unexpected attachment %#v", items[0])
	}
}

func TestNormalizeUserImageAttachmentsDropsInvalidEntries(t *testing.T) {
	t.Parallel()

	items := NormalizeUserImageAttachments([]UserImageAttachment{
		{
			Name:        "diagram.png",
			ContentType: "image/png",
			DataURL:     "data:image/png;base64,ZmFrZQ==",
		},
		{
			Name:        "notes.txt",
			ContentType: "text/plain",
			DataURL:     "data:text/plain;base64,ZmFrZQ==",
		},
		{
			Name:        "broken.png",
			ContentType: "image/png",
			DataURL:     "",
		},
		{
			Name:          "workspace.png",
			ContentType:   "image/png",
			WorkspacePath: "/tmp/session-images/workspace.png",
		},
	})
	if len(items) != 2 {
		t.Fatalf("expected only valid image attachment, got %#v", items)
	}
}

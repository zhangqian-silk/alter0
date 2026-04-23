package domain

import "testing"

func TestEncodeDecodeUserImageAttachmentsRoundTrip(t *testing.T) {
	t.Parallel()

	raw, err := EncodeUserImageAttachments([]UserImageAttachment{
		{
			Name:        "diagram.png",
			ContentType: "image/png",
			DataURL:     "data:image/png;base64,ZmFrZQ==",
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
	if items[0].Name != "diagram.png" || items[0].ContentType != "image/png" || items[0].DataURL != "data:image/png;base64,ZmFrZQ==" {
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
	})
	if len(items) != 1 {
		t.Fatalf("expected only valid image attachment, got %#v", items)
	}
}

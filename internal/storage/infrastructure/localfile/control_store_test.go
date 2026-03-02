package localfile

import (
	"context"
	"testing"

	controldomain "alter0/internal/control/domain"
	shareddomain "alter0/internal/shared/domain"
)

func TestControlStoreJSONRoundTrip(t *testing.T) {
	store := NewControlStore(t.TempDir(), FormatJSON)
	err := store.Save(context.Background(),
		[]controldomain.Channel{
			{ID: "web-default", Type: shareddomain.ChannelTypeWeb, Enabled: true},
		},
		[]controldomain.Skill{
			{ID: "summary", Name: "summary", Enabled: true},
		},
	)
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	channels, skills, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(channels) != 1 || channels[0].ID != "web-default" {
		t.Fatalf("unexpected channels: %+v", channels)
	}
	if len(skills) != 1 || skills[0].ID != "summary" {
		t.Fatalf("unexpected skills: %+v", skills)
	}
}

func TestControlStoreMarkdownRoundTrip(t *testing.T) {
	store := NewControlStore(t.TempDir(), FormatMarkdown)
	err := store.Save(context.Background(),
		[]controldomain.Channel{
			{ID: "cli-default", Type: shareddomain.ChannelTypeCLI, Enabled: true},
		},
		[]controldomain.Skill{
			{ID: "default-nl", Name: "default-nl", Enabled: true},
		},
	)
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	channels, skills, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(channels) != 1 || channels[0].ID != "cli-default" {
		t.Fatalf("unexpected channels: %+v", channels)
	}
	if len(skills) != 1 || skills[0].ID != "default-nl" {
		t.Fatalf("unexpected skills: %+v", skills)
	}
}

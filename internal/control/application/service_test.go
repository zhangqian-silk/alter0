package application

import (
	"testing"

	controldomain "alter0/internal/control/domain"
	shareddomain "alter0/internal/shared/domain"
)

func TestChannelCRUD(t *testing.T) {
	service := NewService()

	err := service.UpsertChannel(controldomain.Channel{
		ID:      "web-default",
		Type:    shareddomain.ChannelTypeWeb,
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	channel, ok := service.ResolveChannel("web-default")
	if !ok {
		t.Fatalf("expected channel exists")
	}
	if channel.Type != shareddomain.ChannelTypeWeb {
		t.Fatalf("expected web type, got %s", channel.Type)
	}

	if !service.DeleteChannel("web-default") {
		t.Fatalf("expected delete success")
	}
}

func TestSkillCRUD(t *testing.T) {
	service := NewService()

	err := service.UpsertSkill(controldomain.Skill{
		ID:      "summary",
		Name:    "summary",
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	skill, ok := service.ResolveSkill("summary")
	if !ok {
		t.Fatalf("expected skill exists")
	}
	if skill.Name != "summary" {
		t.Fatalf("unexpected skill name %s", skill.Name)
	}

	if !service.DeleteSkill("summary") {
		t.Fatalf("expected delete success")
	}
}

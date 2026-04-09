package domain

import (
	"strings"
	"testing"
	"time"

	shareddomain "alter0/internal/shared/domain"
)

func TestMessageRoleIsValid(t *testing.T) {
	if !MessageRoleUser.IsValid() {
		t.Fatalf("user role IsValid() = false, want true")
	}
	if !MessageRoleAssistant.IsValid() {
		t.Fatalf("assistant role IsValid() = false, want true")
	}
	if MessageRole("system").IsValid() {
		t.Fatalf("system role IsValid() = true, want false")
	}
}

func TestMessageSourceValidateAllowsEmptyOrKnownSourceValues(t *testing.T) {
	validSources := []MessageSource{
		{},
		{TriggerType: shareddomain.TriggerTypeUser, ChannelType: shareddomain.ChannelTypeWeb},
		{TriggerType: shareddomain.TriggerTypeCron, ChannelType: shareddomain.ChannelTypeScheduler},
		{TriggerType: shareddomain.TriggerTypeSystem, ChannelType: shareddomain.ChannelTypeCLI},
	}
	for _, source := range validSources {
		if err := source.Validate(); err != nil {
			t.Fatalf("Validate(%+v) error = %v", source, err)
		}
	}

	cases := []struct {
		name    string
		source  MessageSource
		wantErr string
	}{
		{name: "trigger", source: MessageSource{TriggerType: "manual"}, wantErr: "source.trigger_type must be user/cron/system"},
		{name: "channel", source: MessageSource{ChannelType: "mobile"}, wantErr: "source.channel_type must be cli/web/scheduler"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.source.Validate()
			if err == nil {
				t.Fatalf("Validate() error = nil, want containing %q", tc.wantErr)
			}
			assertSessionErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestMessageRecordValidateRequiresIdentityContentTimestampAndValidSource(t *testing.T) {
	now := time.Date(2026, 4, 8, 9, 0, 0, 0, time.UTC)
	valid := MessageRecord{
		MessageID: "message-1",
		SessionID: "session-1",
		Role:      MessageRoleUser,
		Content:   "hello",
		Timestamp: now,
		Source: MessageSource{
			TriggerType: shareddomain.TriggerTypeUser,
			ChannelType: shareddomain.ChannelTypeWeb,
		},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	cases := []struct {
		name    string
		mutate  func(MessageRecord) MessageRecord
		wantErr string
	}{
		{name: "message id", mutate: func(r MessageRecord) MessageRecord { r.MessageID = " "; return r }, wantErr: "message_id is required"},
		{name: "session id", mutate: func(r MessageRecord) MessageRecord { r.SessionID = " "; return r }, wantErr: "session_id is required"},
		{name: "role", mutate: func(r MessageRecord) MessageRecord { r.Role = "system"; return r }, wantErr: "role must be user or assistant"},
		{name: "content", mutate: func(r MessageRecord) MessageRecord { r.Content = " "; return r }, wantErr: "content is required"},
		{name: "timestamp", mutate: func(r MessageRecord) MessageRecord { r.Timestamp = time.Time{}; return r }, wantErr: "timestamp is required"},
		{name: "source", mutate: func(r MessageRecord) MessageRecord { r.Source.TriggerType = "manual"; return r }, wantErr: "source.trigger_type must be user/cron/system"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.mutate(valid).Validate()
			if err == nil {
				t.Fatalf("Validate() error = nil, want containing %q", tc.wantErr)
			}
			assertSessionErrorContains(t, err, tc.wantErr)
		})
	}
}

func assertSessionErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want containing %q", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want containing %q", err.Error(), want)
	}
}

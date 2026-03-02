package domain

import (
	"testing"
	"time"
)

func TestUnifiedMessageValidate(t *testing.T) {
	msg := UnifiedMessage{
		MessageID:   "m1",
		SessionID:   "s1",
		ChannelID:   "web-default",
		ChannelType: ChannelTypeWeb,
		TriggerType: TriggerTypeUser,
		Content:     "hello",
		TraceID:     "t1",
		ReceivedAt:  time.Now().UTC(),
	}

	if err := msg.Validate(); err != nil {
		t.Fatalf("expected valid message, got %v", err)
	}
}

func TestUnifiedMessageValidateInvalidTrigger(t *testing.T) {
	msg := UnifiedMessage{
		MessageID:   "m1",
		SessionID:   "s1",
		ChannelID:   "web-default",
		ChannelType: ChannelTypeWeb,
		TriggerType: "invalid",
		Content:     "hello",
		TraceID:     "t1",
		ReceivedAt:  time.Now().UTC(),
	}

	if err := msg.Validate(); err == nil {
		t.Fatal("expected trigger validation error")
	}
}

package domain

import (
	"context"
	"testing"
	"time"

	shareddomain "alter0/internal/shared/domain"
)

type contractCommandHandler struct{}

func (contractCommandHandler) Name() string { return "echo" }

func (contractCommandHandler) Aliases() []string { return []string{"say"} }

func (contractCommandHandler) Execute(_ context.Context, req CommandRequest) (CommandResult, error) {
	return CommandResult{
		Output: req.Message.Content,
		Metadata: map[string]string{
			"command": req.Name,
		},
	}, nil
}

func TestIntentContractUsesStableRouteTypes(t *testing.T) {
	commandIntent := Intent{Type: IntentTypeCommand, CommandName: "echo", Args: []string{"hello"}}
	if commandIntent.Type != "command" || commandIntent.CommandName != "echo" || commandIntent.Args[0] != "hello" {
		t.Fatalf("command intent = %+v, want stable command contract", commandIntent)
	}

	nlIntent := Intent{Type: IntentTypeNL}
	if nlIntent.Type != "nl" {
		t.Fatalf("nl intent type = %q, want nl", nlIntent.Type)
	}
}

func TestCommandHandlerContractCarriesUnifiedMessageAndMetadata(t *testing.T) {
	handler := contractCommandHandler{}
	msg := shareddomain.UnifiedMessage{
		MessageID:   "message-1",
		SessionID:   "session-1",
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     "hello",
		TraceID:     "trace-1",
		ReceivedAt:  time.Date(2026, 4, 8, 9, 0, 0, 0, time.UTC),
	}

	result, err := handler.Execute(context.Background(), CommandRequest{
		Message: msg,
		Name:    handler.Name(),
		Args:    []string{"hello"},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Output != "hello" {
		t.Fatalf("Output = %q, want hello", result.Output)
	}
	if result.Metadata["command"] != "echo" {
		t.Fatalf("Metadata command = %q, want echo", result.Metadata["command"])
	}
}

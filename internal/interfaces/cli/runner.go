package cli

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	sharedapp "alter0/internal/shared/application"
	shareddomain "alter0/internal/shared/domain"
	"alter0/internal/shared/infrastructure/observability"
)

type Orchestrator interface {
	Handle(ctx context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error)
}

type Runner struct {
	orchestrator Orchestrator
	telemetry    *observability.Telemetry
	idGenerator  sharedapp.IDGenerator
	logger       *slog.Logger
}

func NewRunner(
	orchestrator Orchestrator,
	telemetry *observability.Telemetry,
	idGenerator sharedapp.IDGenerator,
	logger *slog.Logger,
) *Runner {
	return &Runner{
		orchestrator: orchestrator,
		telemetry:    telemetry,
		idGenerator:  idGenerator,
		logger:       logger,
	}
}

func (r *Runner) Run(ctx context.Context) error {
	scanner := bufio.NewScanner(os.Stdin)
	sessionID := r.idGenerator.NewID()
	channelID := "cli-default"

	fmt.Printf("alter0 cli session: %s\n", sessionID)
	fmt.Println("input /quit to exit.")

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if input == "/quit" || input == "/exit" {
			return nil
		}

		msg := shareddomain.UnifiedMessage{
			MessageID:   r.idGenerator.NewID(),
			SessionID:   sessionID,
			ChannelID:   channelID,
			ChannelType: shareddomain.ChannelTypeCLI,
			TriggerType: shareddomain.TriggerTypeUser,
			Content:     input,
			TraceID:     r.idGenerator.NewID(),
			ReceivedAt:  time.Now().UTC(),
		}

		r.telemetry.CountGateway(string(msg.ChannelType))
		result, err := r.orchestrator.Handle(ctx, msg)
		if err != nil {
			r.logger.Error("cli message failed",
				slog.String("trace_id", msg.TraceID),
				slog.String("session_id", msg.SessionID),
				slog.String("message_id", msg.MessageID),
				slog.String("error", err.Error()),
			)
			fmt.Printf("error: %s (%s)\n", err.Error(), result.ErrorCode)
			continue
		}
		fmt.Println(result.Output)
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

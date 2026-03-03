package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	orchdomain "alter0/internal/orchestration/domain"
	sharedapp "alter0/internal/shared/application"
	shareddomain "alter0/internal/shared/domain"
)

type ExecutionPort interface {
	ExecuteNaturalLanguage(ctx context.Context, msg shareddomain.UnifiedMessage) (string, error)
}

type Orchestrator interface {
	Handle(ctx context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error)
}

type Service struct {
	classifier orchdomain.IntentClassifier
	registry   orchdomain.CommandRegistry
	executor   ExecutionPort
	telemetry  sharedapp.Telemetry
	logger     *slog.Logger
}

func NewService(
	classifier orchdomain.IntentClassifier,
	registry orchdomain.CommandRegistry,
	executor ExecutionPort,
	telemetry sharedapp.Telemetry,
	logger *slog.Logger,
) *Service {
	return &Service{
		classifier: classifier,
		registry:   registry,
		executor:   executor,
		telemetry:  telemetry,
		logger:     logger,
	}
}

func (s *Service) Handle(ctx context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error) {
	startedAt := time.Now()
	result := shareddomain.OrchestrationResult{
		MessageID: msg.MessageID,
		SessionID: msg.SessionID,
	}

	if err := msg.Validate(); err != nil {
		s.telemetry.CountError("unknown")
		s.telemetry.ObserveDuration("unknown", time.Since(startedAt))
		return result, fmt.Errorf("invalid message: %w", err)
	}

	intent := s.classifier.Classify(msg.Content)
	switch intent.Type {
	case orchdomain.IntentTypeCommand:
		result.Route = shareddomain.RouteCommand
		s.telemetry.CountRoute(string(result.Route))
		handler, ok := s.registry.Resolve(intent.CommandName)
		if !ok {
			err := fmt.Errorf("command not found: %s", intent.CommandName)
			s.onError(msg, result.Route, startedAt, err)
			result.ErrorCode = "command_not_found"
			return result, err
		}

		s.telemetry.CountCommand(handler.Name())
		cmdResult, err := handler.Execute(ctx, orchdomain.CommandRequest{
			Message: msg,
			Name:    intent.CommandName,
			Args:    intent.Args,
		})
		if err != nil {
			s.onError(msg, result.Route, startedAt, err)
			result.ErrorCode = "command_failed"
			return result, err
		}

		result.Output = cmdResult.Output
		result.Metadata = cmdResult.Metadata
		s.onSuccess(msg, result.Route, startedAt)
		return result, nil
	case orchdomain.IntentTypeNL:
		result.Route = shareddomain.RouteNL
		s.telemetry.CountRoute(string(result.Route))
		output, err := s.executor.ExecuteNaturalLanguage(ctx, msg)
		if err != nil {
			s.onError(msg, result.Route, startedAt, err)
			result.ErrorCode = "nl_execution_failed"
			return result, err
		}

		result.Output = output
		s.onSuccess(msg, result.Route, startedAt)
		return result, nil
	default:
		err := errors.New("unsupported intent type")
		s.onError(msg, "unknown", startedAt, err)
		result.ErrorCode = "intent_unsupported"
		return result, err
	}
}

func (s *Service) onSuccess(msg shareddomain.UnifiedMessage, route shareddomain.Route, startedAt time.Time) {
	duration := time.Since(startedAt)
	s.telemetry.ObserveDuration(string(route), duration)
	s.logger.Info("message handled",
		slog.String("trace_id", msg.TraceID),
		slog.String("session_id", msg.SessionID),
		slog.String("message_id", msg.MessageID),
		slog.String("channel_id", msg.ChannelID),
		slog.String("channel_type", string(msg.ChannelType)),
		slog.String("trigger_type", string(msg.TriggerType)),
		slog.String("correlation_id", msg.CorrelationID),
		slog.String("route", string(route)),
		slog.Int64("duration_ms", duration.Milliseconds()),
	)
}

func (s *Service) onError(msg shareddomain.UnifiedMessage, route shareddomain.Route, startedAt time.Time, err error) {
	duration := time.Since(startedAt)
	s.telemetry.CountError(string(route))
	s.telemetry.ObserveDuration(string(route), duration)
	s.logger.Error("message failed",
		slog.String("trace_id", msg.TraceID),
		slog.String("session_id", msg.SessionID),
		slog.String("message_id", msg.MessageID),
		slog.String("channel_id", msg.ChannelID),
		slog.String("channel_type", string(msg.ChannelType)),
		slog.String("trigger_type", string(msg.TriggerType)),
		slog.String("correlation_id", msg.CorrelationID),
		slog.String("route", string(route)),
		slog.Int64("duration_ms", duration.Milliseconds()),
		slog.String("error", err.Error()),
	)
}

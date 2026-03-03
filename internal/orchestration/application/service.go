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
	ExecuteNaturalLanguage(ctx context.Context, msg shareddomain.UnifiedMessage) (shareddomain.ExecutionResult, error)
}

type Orchestrator interface {
	Handle(ctx context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error)
}

type Service struct {
	classifier orchdomain.IntentClassifier
	registry   orchdomain.CommandRegistry
	executor   ExecutionPort
	memory     sessionMemory
	longTerm   longTermMemory
	mandatory  mandatoryContext
	telemetry  sharedapp.Telemetry
	logger     *slog.Logger
}

type ServiceOption func(*Service)

type sessionMemory interface {
	Snapshot(sessionID string, input string, now time.Time) sessionMemorySnapshot
	Record(msg shareddomain.UnifiedMessage, route shareddomain.Route, output string)
}

type longTermMemory interface {
	Snapshot(msg shareddomain.UnifiedMessage, query string, now time.Time) longTermMemorySnapshot
	Record(msg shareddomain.UnifiedMessage, route shareddomain.Route, output string)
}

func NewService(
	classifier orchdomain.IntentClassifier,
	registry orchdomain.CommandRegistry,
	executor ExecutionPort,
	telemetry sharedapp.Telemetry,
	logger *slog.Logger,
) *Service {
	service := &Service{
		classifier: classifier,
		registry:   registry,
		executor:   executor,
		memory:     newSessionMemoryStore(SessionMemoryOptions{}),
		longTerm:   newLongTermMemoryStore(LongTermMemoryOptions{}),
		mandatory:  newMandatoryContextStore(MandatoryContextOptions{}),
		telemetry:  telemetry,
		logger:     logger,
	}
	return service
}

func NewServiceWithOptions(
	classifier orchdomain.IntentClassifier,
	registry orchdomain.CommandRegistry,
	executor ExecutionPort,
	telemetry sharedapp.Telemetry,
	logger *slog.Logger,
	options ...ServiceOption,
) *Service {
	service := NewService(classifier, registry, executor, telemetry, logger)
	for _, option := range options {
		if option == nil {
			continue
		}
		option(service)
	}
	if service.memory == nil {
		service.memory = newSessionMemoryStore(SessionMemoryOptions{})
	}
	if service.longTerm == nil {
		service.longTerm = newLongTermMemoryStore(LongTermMemoryOptions{})
	}
	if service.mandatory == nil {
		service.mandatory = newMandatoryContextStore(MandatoryContextOptions{})
	}
	return service
}

func WithSessionMemoryOptions(options SessionMemoryOptions) ServiceOption {
	return func(service *Service) {
		service.memory = newSessionMemoryStore(options)
	}
}

func WithSessionMemory(memory sessionMemory) ServiceOption {
	return func(service *Service) {
		service.memory = memory
	}
}

func WithLongTermMemoryOptions(options LongTermMemoryOptions) ServiceOption {
	return func(service *Service) {
		service.longTerm = newLongTermMemoryStore(options)
	}
}

func WithLongTermMemory(memory longTermMemory) ServiceOption {
	return func(service *Service) {
		service.longTerm = memory
	}
}

func WithMandatoryContextOptions(options MandatoryContextOptions) ServiceOption {
	return func(service *Service) {
		service.mandatory = newMandatoryContextStore(options)
	}
}

func WithMandatoryContext(mandatory mandatoryContext) ServiceOption {
	return func(service *Service) {
		service.mandatory = mandatory
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
		s.memory.Record(msg, result.Route, result.Output)
		s.longTerm.Record(msg, result.Route, result.Output)
		s.onSuccess(msg, result.Route, startedAt)
		return result, nil
	case orchdomain.IntentTypeNL:
		result.Route = shareddomain.RouteNL
		s.telemetry.CountRoute(string(result.Route))
		snapshot := s.memory.Snapshot(msg.SessionID, msg.Content, msg.ReceivedAt)
		longTermSnapshot := s.longTerm.Snapshot(msg, msg.Content, msg.ReceivedAt)
		mandatorySnapshot := s.mandatory.Snapshot(msg.ReceivedAt)
		snapshot, sessionConflicts := applyMandatoryContextToSessionMemory(snapshot, mandatorySnapshot)
		longTermSnapshot, longTermConflicts := applyMandatoryContextToLongTermMemory(longTermSnapshot, mandatorySnapshot)
		conflicts := append(sessionConflicts, longTermConflicts...)
		conflictMetadata := mandatoryContextConflictMetadata(conflicts)

		if len(conflicts) > 0 && s.logger != nil {
			s.logger.Warn("mandatory context overrides memory",
				slog.String("session_id", msg.SessionID),
				slog.String("message_id", msg.MessageID),
				slog.String("mandatory_context_version", mandatorySnapshot.Version),
				slog.String("mandatory_context_file", mandatorySnapshot.FilePath),
				slog.String("mandatory_conflicts", conflictMetadata[mandatoryContextConflictDetailMetadataKey]),
				slog.Int("mandatory_conflict_count", len(conflicts)),
			)
		}
		execMessage := msg
		execMessage.Content = buildSessionMemoryPrompt(msg.Content, snapshot)
		execMessage.Content = buildLongTermMemoryPrompt(execMessage.Content, longTermSnapshot)
		execMessage.Content = buildMandatoryContextPrompt(execMessage.Content, mandatorySnapshot)
		execMessage.Metadata = mergeStringMap(
			msg.Metadata,
			mergeStringMap(
				mergeStringMap(snapshot.Metadata(), longTermSnapshot.Metadata()),
				mergeStringMap(mandatorySnapshot.Metadata(), conflictMetadata),
			),
		)
		nlResult, err := s.executor.ExecuteNaturalLanguage(ctx, execMessage)
		if err != nil {
			s.onError(msg, result.Route, startedAt, err)
			result.ErrorCode = "nl_execution_failed"
			return result, err
		}

		result.Output = nlResult.Output
		result.Metadata = mergeStringMap(result.Metadata, snapshot.ResultMetadata())
		result.Metadata = mergeStringMap(result.Metadata, longTermSnapshot.ResultMetadata())
		result.Metadata = mergeStringMap(result.Metadata, mandatorySnapshot.ResultMetadata())
		result.Metadata = mergeStringMap(result.Metadata, conflictMetadata)
		result.Metadata = mergeStringMap(result.Metadata, nlResult.Metadata)
		s.memory.Record(msg, result.Route, nlResult.Output)
		s.longTerm.Record(msg, result.Route, nlResult.Output)
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

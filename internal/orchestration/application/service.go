package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	orchdomain "alter0/internal/orchestration/domain"
	sharedapp "alter0/internal/shared/application"
	shareddomain "alter0/internal/shared/domain"
	tasksummaryapp "alter0/internal/tasksummary/application"
)

const (
	terminalTaskInteractiveMetadataKey = "alter0.task.terminal_interactive"
	terminalTaskTypeMetadataKey        = "alter0.task.type"
	terminalTaskTypeValue              = "terminal"
	memoryHistoryBypassedMetadataKey   = "memory_history_bypassed"
	memoryHistoryScopeMetadataKey      = "memory_history_scope"
	memoryHistoryScopeTerminalSession  = "terminal_session"
)

type ExecutionPort interface {
	ExecuteAgent(ctx context.Context, msg shareddomain.UnifiedMessage) (shareddomain.ExecutionResult, error)
}

type streamExecutionPort interface {
	ExecuteAgentStream(
		ctx context.Context,
		msg shareddomain.UnifiedMessage,
		onEvent func(shareddomain.StreamEvent) error,
	) (shareddomain.ExecutionResult, error)
}

type Orchestrator interface {
	Handle(ctx context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error)
}

type Service struct {
	classifier orchdomain.IntentClassifier
	registry   orchdomain.CommandRegistry
	executor   ExecutionPort
	longTerm   longTermMemory
	taskMemory taskSummaryMemory
	mandatory  mandatoryContext
	telemetry  sharedapp.Telemetry
	logger     *slog.Logger
}

type ServiceOption func(*Service)

type longTermMemory interface {
	Snapshot(msg shareddomain.UnifiedMessage, query string, now time.Time) longTermMemorySnapshot
	Record(msg shareddomain.UnifiedMessage, route shareddomain.Route, output string)
}

type taskSummaryMemory interface {
	Snapshot(query string, now time.Time) tasksummaryapp.Snapshot
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
		longTerm:   newLongTermMemoryStore(LongTermMemoryOptions{}),
		taskMemory: tasksummaryapp.NewStore(tasksummaryapp.Options{}),
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
	if service.longTerm == nil {
		service.longTerm = newLongTermMemoryStore(LongTermMemoryOptions{})
	}
	if service.taskMemory == nil {
		service.taskMemory = tasksummaryapp.NewStore(tasksummaryapp.Options{})
	}
	if service.mandatory == nil {
		service.mandatory = newMandatoryContextStore(MandatoryContextOptions{})
	}
	return service
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

func WithTaskSummaryMemory(memory taskSummaryMemory) ServiceOption {
	return func(service *Service) {
		service.taskMemory = memory
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
	return s.handle(ctx, msg, nil)
}

func (s *Service) Classify(content string) orchdomain.Intent {
	if s == nil || s.classifier == nil {
		return orchdomain.Intent{Type: orchdomain.IntentTypeAgent}
	}
	return s.classifier.Classify(content)
}

func (s *Service) HandleStream(
	ctx context.Context,
	msg shareddomain.UnifiedMessage,
	onEvent func(shareddomain.StreamEvent) error,
) (shareddomain.OrchestrationResult, error) {
	return s.handle(ctx, msg, onEvent)
}

func (s *Service) handle(
	ctx context.Context,
	msg shareddomain.UnifiedMessage,
	onEvent func(shareddomain.StreamEvent) error,
) (shareddomain.OrchestrationResult, error) {
	startedAt := time.Now()
	terminalSessionOnly := isTerminalSessionContextOnly(msg.Metadata)
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
		result.ProcessSteps = append([]shareddomain.ProcessStep(nil), cmdResult.ProcessSteps...)
		if onEvent != nil && strings.TrimSpace(result.Output) != "" {
			if err := onEvent(shareddomain.StreamEvent{
				Type: shareddomain.StreamEventTypeOutput,
				Text: result.Output,
			}); err != nil {
				s.onError(msg, result.Route, startedAt, err)
				result.ErrorCode = "stream_write_failed"
				return result, err
			}
		}
		if !terminalSessionOnly {
			s.longTerm.Record(msg, result.Route, result.Output)
		}
		s.onSuccess(msg, result.Route, startedAt)
		return result, nil
	case orchdomain.IntentTypeAgent:
		result.Route = shareddomain.RouteAgent
		s.telemetry.CountRoute(string(result.Route))
		mandatorySnapshot := s.mandatory.Snapshot(msg.ReceivedAt)
		execMessage := msg
		conflictMetadata := map[string]string{}
		if terminalSessionOnly {
			execMessage.Content = buildMandatoryContextPrompt(msg.Content, mandatorySnapshot)
			execMessage.Metadata = mergeStringMap(
				msg.Metadata,
				mergeStringMap(mandatorySnapshot.Metadata(), terminalSessionOnlyMetadata()),
			)
		} else {
			longTermSnapshot := s.longTerm.Snapshot(msg, msg.Content, msg.ReceivedAt)
			taskSnapshot := s.taskMemory.Snapshot(msg.Content, msg.ReceivedAt)
			s.observeTaskSummarySnapshot(taskSnapshot)
			longTermSnapshot, longTermConflicts := applyMandatoryContextToLongTermMemory(longTermSnapshot, mandatorySnapshot)
			conflicts := longTermConflicts
			conflictMetadata = mandatoryContextConflictMetadata(conflicts)

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

			execMessage.Content = buildLongTermMemoryPrompt(msg.Content, longTermSnapshot)
			execMessage.Content = buildTaskSummaryPrompt(execMessage.Content, taskSnapshot)
			execMessage.Content = buildMandatoryContextPrompt(execMessage.Content, mandatorySnapshot)
			execMessage.Metadata = mergeStringMap(
				msg.Metadata,
				mergeStringMap(
					mergeStringMap(
						longTermSnapshot.Metadata(),
						taskSnapshot.Metadata(),
					),
					mergeStringMap(mandatorySnapshot.Metadata(), conflictMetadata),
				),
			)
			result.Metadata = mergeStringMap(result.Metadata, longTermSnapshot.ResultMetadata())
			result.Metadata = mergeStringMap(result.Metadata, taskSnapshot.ResultMetadata())
			result.Metadata = mergeStringMap(result.Metadata, conflictMetadata)
		}
		var nlResult shareddomain.ExecutionResult
		var err error
		if onEvent != nil {
			if streamExecutor, ok := s.executor.(streamExecutionPort); ok {
				nlResult, err = streamExecutor.ExecuteAgentStream(ctx, execMessage, onEvent)
			} else {
				nlResult, err = s.executor.ExecuteAgent(ctx, execMessage)
				if err == nil && strings.TrimSpace(nlResult.Output) != "" {
					if streamErr := onEvent(shareddomain.StreamEvent{
						Type: shareddomain.StreamEventTypeOutput,
						Text: nlResult.Output,
					}); streamErr != nil {
						s.onError(msg, result.Route, startedAt, streamErr)
						result.ErrorCode = "stream_write_failed"
						return result, streamErr
					}
				}
			}
		} else {
			nlResult, err = s.executor.ExecuteAgent(ctx, execMessage)
		}
		if err != nil {
			s.onError(msg, result.Route, startedAt, err)
			result.ErrorCode = "agent_execution_failed"
			return result, err
		}

		result.Output = nlResult.Output
		result.ProcessSteps = append([]shareddomain.ProcessStep(nil), nlResult.ProcessSteps...)
		if terminalSessionOnly {
			result.Metadata = mergeStringMap(result.Metadata, terminalSessionOnlyMetadata())
		}
		result.Metadata = mergeStringMap(result.Metadata, mandatorySnapshot.ResultMetadata())
		result.Metadata = mergeStringMap(result.Metadata, nlResult.Metadata)
		if !terminalSessionOnly {
			s.longTerm.Record(msg, result.Route, nlResult.Output)
		}
		s.onSuccess(msg, result.Route, startedAt)
		return result, nil
	default:
		err := errors.New("unsupported intent type")
		s.onError(msg, "unknown", startedAt, err)
		result.ErrorCode = "intent_unsupported"
		return result, err
	}
}

func isTerminalSessionContextOnly(metadata map[string]string) bool {
	if len(metadata) == 0 {
		return false
	}
	if parseTruthyString(metadata[terminalTaskInteractiveMetadataKey]) {
		return true
	}
	return strings.EqualFold(
		strings.TrimSpace(metadata[terminalTaskTypeMetadataKey]),
		terminalTaskTypeValue,
	)
}

func parseTruthyString(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func terminalSessionOnlyMetadata() map[string]string {
	return map[string]string{
		memoryHistoryBypassedMetadataKey: "true",
		memoryHistoryScopeMetadataKey:    memoryHistoryScopeTerminalSession,
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

func (s *Service) observeTaskSummarySnapshot(snapshot tasksummaryapp.Snapshot) {
	if s.telemetry == nil {
		return
	}
	if snapshot.DeepTriggered {
		s.telemetry.CountMemoryEvent("deep_retrieval_triggered")
	}
	if snapshot.DeepOverridden {
		s.telemetry.CountMemoryEvent("deep_retrieval_overridden")
	}
	if snapshot.DeepMiss {
		s.telemetry.CountMemoryEvent("deep_retrieval_miss")
	}
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

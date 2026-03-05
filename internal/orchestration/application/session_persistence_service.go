package application

import (
	"context"
	"log/slog"
	"strings"
	"time"

	sessionapp "alter0/internal/session/application"
	sessiondomain "alter0/internal/session/domain"
	sharedapp "alter0/internal/shared/application"
	shareddomain "alter0/internal/shared/domain"
)

const (
	sessionSourceMetadataJobIDKey   = "job_id"
	sessionSourceMetadataJobNameKey = "job_name"
	sessionSourceMetadataFiredAtKey = "fired_at"
)

type sessionRecorder interface {
	Append(records ...sessiondomain.MessageRecord) error
}

type SessionPersistenceService struct {
	downstream  Orchestrator
	recorder    sessionRecorder
	idGenerator sharedapp.IDGenerator
	logger      *slog.Logger
}

type streamPersistenceOrchestrator interface {
	HandleStream(
		ctx context.Context,
		msg shareddomain.UnifiedMessage,
		onDelta func(string) error,
	) (shareddomain.OrchestrationResult, error)
}

func NewSessionPersistenceService(
	downstream Orchestrator,
	recorder *sessionapp.Service,
	idGenerator sharedapp.IDGenerator,
	logger *slog.Logger,
) *SessionPersistenceService {
	if logger == nil {
		logger = slog.Default()
	}
	return &SessionPersistenceService{
		downstream:  downstream,
		recorder:    recorder,
		idGenerator: idGenerator,
		logger:      logger,
	}
}

func (s *SessionPersistenceService) Handle(ctx context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error) {
	result, err := s.downstream.Handle(ctx, msg)
	s.persistResult(msg, result, err)
	return result, err
}

func (s *SessionPersistenceService) HandleStream(
	ctx context.Context,
	msg shareddomain.UnifiedMessage,
	onDelta func(string) error,
) (shareddomain.OrchestrationResult, error) {
	var (
		result shareddomain.OrchestrationResult
		err    error
	)
	if downstream, ok := s.downstream.(streamPersistenceOrchestrator); ok {
		result, err = downstream.HandleStream(ctx, msg, onDelta)
	} else {
		result, err = s.downstream.Handle(ctx, msg)
		if err == nil && onDelta != nil && strings.TrimSpace(result.Output) != "" {
			if streamErr := onDelta(result.Output); streamErr != nil {
				err = streamErr
			}
		}
	}
	s.persistResult(msg, result, err)
	return result, err
}

func (s *SessionPersistenceService) persistResult(
	msg shareddomain.UnifiedMessage,
	result shareddomain.OrchestrationResult,
	err error,
) {
	if s.recorder == nil {
		return
	}

	userTimestamp := normalizePersistTimestamp(msg.ReceivedAt)
	assistantTimestamp := normalizePersistTimestamp(time.Now().UTC())
	if assistantTimestamp.Before(userTimestamp) {
		assistantTimestamp = userTimestamp
	}

	assistantContent := strings.TrimSpace(result.Output)
	if assistantContent == "" {
		if err != nil {
			assistantContent = err.Error()
		} else {
			assistantContent = "handled without output"
		}
	}

	assistantMessageID := s.newAssistantMessageID(msg.MessageID)
	taskID := ""
	if len(msg.Metadata) > 0 {
		taskID = strings.TrimSpace(msg.Metadata["task_id"])
	}
	source := buildSessionSource(msg)
	persistErr := s.recorder.Append(
		sessiondomain.MessageRecord{
			MessageID: msg.MessageID,
			SessionID: msg.SessionID,
			Role:      sessiondomain.MessageRoleUser,
			Content:   msg.Content,
			Timestamp: userTimestamp,
			Source:    source,
			RouteResult: sessiondomain.RouteResult{
				Route:     result.Route,
				ErrorCode: result.ErrorCode,
				TaskID:    taskID,
			},
		},
		sessiondomain.MessageRecord{
			MessageID: assistantMessageID,
			SessionID: msg.SessionID,
			Role:      sessiondomain.MessageRoleAssistant,
			Content:   assistantContent,
			Timestamp: assistantTimestamp,
			Source:    source,
			RouteResult: sessiondomain.RouteResult{
				Route:     result.Route,
				ErrorCode: result.ErrorCode,
				TaskID:    taskID,
			},
		},
	)
	if persistErr != nil && s.logger != nil {
		s.logger.Error("persist session content failed",
			slog.String("session_id", msg.SessionID),
			slog.String("message_id", msg.MessageID),
			slog.String("error", persistErr.Error()),
		)
	}
}

func (s *SessionPersistenceService) newAssistantMessageID(fallback string) string {
	if s.idGenerator != nil {
		id := strings.TrimSpace(s.idGenerator.NewID())
		if id != "" {
			return id
		}
	}
	base := strings.TrimSpace(fallback)
	if base == "" {
		base = "assistant"
	}
	return base + "-assistant"
}

func normalizePersistTimestamp(ts time.Time) time.Time {
	if ts.IsZero() {
		return time.Now().UTC()
	}
	return ts.UTC()
}

func buildSessionSource(msg shareddomain.UnifiedMessage) sessiondomain.MessageSource {
	source := sessiondomain.MessageSource{
		TriggerType:   msg.TriggerType,
		ChannelType:   msg.ChannelType,
		ChannelID:     strings.TrimSpace(msg.ChannelID),
		CorrelationID: strings.TrimSpace(msg.CorrelationID),
	}
	if msg.TriggerType != shareddomain.TriggerTypeCron {
		return source
	}
	source.JobID = strings.TrimSpace(msg.Metadata[sessionSourceMetadataJobIDKey])
	source.JobName = strings.TrimSpace(msg.Metadata[sessionSourceMetadataJobNameKey])
	if source.CorrelationID == "" {
		source.CorrelationID = source.JobID
	}
	if firedAt, err := parseSessionSourceFiredAt(msg.Metadata[sessionSourceMetadataFiredAtKey]); err == nil {
		source.FiredAt = firedAt
	}
	return source
}

func parseSessionSourceFiredAt(raw string) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

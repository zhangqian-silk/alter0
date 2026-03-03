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

type sessionRecorder interface {
	Append(records ...sessiondomain.MessageRecord) error
}

type SessionPersistenceService struct {
	downstream  Orchestrator
	recorder    sessionRecorder
	idGenerator sharedapp.IDGenerator
	logger      *slog.Logger
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
	if s.recorder == nil {
		return result, err
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
	persistErr := s.recorder.Append(
		sessiondomain.MessageRecord{
			MessageID: msg.MessageID,
			SessionID: msg.SessionID,
			Role:      sessiondomain.MessageRoleUser,
			Content:   msg.Content,
			Timestamp: userTimestamp,
			RouteResult: sessiondomain.RouteResult{
				Route:     result.Route,
				ErrorCode: result.ErrorCode,
			},
		},
		sessiondomain.MessageRecord{
			MessageID: assistantMessageID,
			SessionID: msg.SessionID,
			Role:      sessiondomain.MessageRoleAssistant,
			Content:   assistantContent,
			Timestamp: assistantTimestamp,
			RouteResult: sessiondomain.RouteResult{
				Route:     result.Route,
				ErrorCode: result.ErrorCode,
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

	return result, err
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

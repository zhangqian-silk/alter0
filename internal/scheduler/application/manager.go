package application

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	schedulerdomain "alter0/internal/scheduler/domain"
	sharedapp "alter0/internal/shared/application"
	shareddomain "alter0/internal/shared/domain"
)

type Orchestrator interface {
	Handle(ctx context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error)
}

type Store interface {
	Load(ctx context.Context) ([]schedulerdomain.Job, error)
	Save(ctx context.Context, jobs []schedulerdomain.Job) error
}

type Manager struct {
	orchestrator Orchestrator
	telemetry    sharedapp.Telemetry
	idGenerator  sharedapp.IDGenerator
	logger       *slog.Logger
	store        Store

	mu      sync.Mutex
	baseCtx context.Context
	started bool
	jobs    map[string]schedulerdomain.Job
	runners map[string]context.CancelFunc
}

func NewManager(
	orchestrator Orchestrator,
	telemetry sharedapp.Telemetry,
	idGenerator sharedapp.IDGenerator,
	logger *slog.Logger,
) *Manager {
	return newManager(orchestrator, telemetry, idGenerator, logger, nil)
}

func NewManagerWithStore(
	ctx context.Context,
	orchestrator Orchestrator,
	telemetry sharedapp.Telemetry,
	idGenerator sharedapp.IDGenerator,
	logger *slog.Logger,
	store Store,
) (*Manager, error) {
	manager := newManager(orchestrator, telemetry, idGenerator, logger, store)
	if store == nil {
		return manager, nil
	}

	jobs, err := store.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("load scheduler state: %w", err)
	}
	for _, job := range jobs {
		normalized, err := job.Normalize()
		if err != nil {
			return nil, fmt.Errorf("invalid job in store: %w", err)
		}
		manager.jobs[normalized.ID] = normalized
	}
	return manager, nil
}

func newManager(
	orchestrator Orchestrator,
	telemetry sharedapp.Telemetry,
	idGenerator sharedapp.IDGenerator,
	logger *slog.Logger,
	store Store,
) *Manager {
	return &Manager{
		orchestrator: orchestrator,
		telemetry:    telemetry,
		idGenerator:  idGenerator,
		logger:       logger,
		store:        store,
		jobs:         map[string]schedulerdomain.Job{},
		runners:      map[string]context.CancelFunc{},
	}
}

func (m *Manager) Start(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return
	}

	m.baseCtx = ctx
	m.started = true
	for _, job := range m.jobs {
		if !job.Enabled {
			continue
		}
		m.startRunnerLocked(job)
	}
}

func (m *Manager) Upsert(job schedulerdomain.Job) error {
	normalized, err := job.Normalize()
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	previous, existed := m.jobs[normalized.ID]
	m.jobs[normalized.ID] = normalized
	m.stopRunnerLocked(normalized.ID)
	if m.started && normalized.Enabled {
		m.startRunnerLocked(normalized)
	}
	if err := m.storeLocked(); err != nil {
		m.stopRunnerLocked(normalized.ID)
		if existed {
			m.jobs[normalized.ID] = previous
			if m.started && previous.Enabled {
				m.startRunnerLocked(previous)
			}
		} else {
			delete(m.jobs, normalized.ID)
		}
		return err
	}
	return nil
}

func (m *Manager) Delete(id string) bool {
	key := normalize(id)
	m.mu.Lock()
	defer m.mu.Unlock()

	previous, ok := m.jobs[key]
	if !ok {
		return false
	}
	delete(m.jobs, key)
	m.stopRunnerLocked(key)
	if err := m.storeLocked(); err != nil {
		m.jobs[key] = previous
		if m.started && previous.Enabled {
			m.startRunnerLocked(previous)
		}
		return false
	}
	return true
}

func (m *Manager) List() []schedulerdomain.Job {
	m.mu.Lock()
	defer m.mu.Unlock()

	ids := make([]string, 0, len(m.jobs))
	for id := range m.jobs {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	items := make([]schedulerdomain.Job, 0, len(ids))
	for _, id := range ids {
		items = append(items, m.jobs[id])
	}
	return items
}

func (m *Manager) startRunnerLocked(job schedulerdomain.Job) {
	if m.baseCtx == nil {
		return
	}
	ctx, cancel := context.WithCancel(m.baseCtx)
	m.runners[job.ID] = cancel

	go func(item schedulerdomain.Job) {
		for {
			nextRun, err := item.NextRun(time.Now().UTC())
			if err != nil {
				m.logger.Error("cron schedule parse failed",
					slog.String("job_id", item.ID),
					slog.String("job_name", item.Name),
					slog.String("error", err.Error()),
				)
				if waitContextDone(ctx, time.Minute) {
					return
				}
				continue
			}

			waitFor := time.Until(nextRun)
			if waitFor < 0 {
				waitFor = 0
			}
			timer := time.NewTimer(waitFor)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case firedAt := <-timer.C:
				m.triggerJob(ctx, item, firedAt.UTC())
			}
		}
	}(job)
}

func (m *Manager) stopRunnerLocked(jobID string) {
	cancel, ok := m.runners[jobID]
	if !ok {
		return
	}
	cancel()
	delete(m.runners, jobID)
}

func (m *Manager) triggerJob(ctx context.Context, job schedulerdomain.Job, firedAt time.Time) {
	channelID := strings.TrimSpace(job.ChannelID)
	if channelID == "" {
		channelID = "scheduler-default"
	}

	sessionID := m.newCronSessionID(job, firedAt)

	metadata := cloneMap(job.Metadata)
	metadata["job_id"] = job.ID
	metadata["job_name"] = job.Name
	metadata["fired_at"] = firedAt.Format(time.RFC3339)
	metadata["session_id"] = sessionID

	msg := shareddomain.UnifiedMessage{
		MessageID:     m.idGenerator.NewID(),
		SessionID:     sessionID,
		UserID:        job.UserID,
		ChannelID:     channelID,
		ChannelType:   shareddomain.ChannelTypeScheduler,
		TriggerType:   shareddomain.TriggerTypeCron,
		Content:       job.TaskConfig.Input,
		Metadata:      metadata,
		TraceID:       m.idGenerator.NewID(),
		CorrelationID: job.ID,
		ReceivedAt:    firedAt,
	}

	m.telemetry.CountGateway(string(msg.ChannelType))
	result, err := m.orchestrator.Handle(ctx, msg)
	if err != nil {
		m.logger.Error("cron job failed",
			slog.String("job_id", job.ID),
			slog.String("job_name", job.Name),
			slog.String("trace_id", msg.TraceID),
			slog.String("session_id", msg.SessionID),
			slog.String("message_id", msg.MessageID),
			slog.String("error", err.Error()),
			slog.String("error_code", result.ErrorCode),
		)
		return
	}

	m.logger.Info("cron job handled",
		slog.String("job_id", job.ID),
		slog.String("job_name", job.Name),
		slog.String("trace_id", msg.TraceID),
		slog.String("session_id", msg.SessionID),
		slog.String("message_id", msg.MessageID),
		slog.String("route", string(result.Route)),
	)
}

func (m *Manager) newCronSessionID(job schedulerdomain.Job, firedAt time.Time) string {
	prefix := normalize(job.ID)
	if prefix == "" {
		prefix = "job"
	}
	seed := fmt.Sprintf("%d", firedAt.UTC().UnixNano())
	if m.idGenerator != nil {
		if nextID := strings.TrimSpace(m.idGenerator.NewID()); nextID != "" {
			seed = normalize(nextID)
		}
	}
	return fmt.Sprintf("cron-%s-%s-%s", prefix, firedAt.UTC().Format("20060102t150405z"), seed)
}

func waitContextDone(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return true
	case <-timer.C:
		return false
	}
}

func cloneMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func (m *Manager) storeLocked() error {
	if m.store == nil {
		return nil
	}
	if err := m.store.Save(context.Background(), snapshotJobs(m.jobs)); err != nil {
		return fmt.Errorf("store scheduler state: %w", err)
	}
	return nil
}

func snapshotJobs(items map[string]schedulerdomain.Job) []schedulerdomain.Job {
	out := make([]schedulerdomain.Job, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		return normalize(out[i].ID) < normalize(out[j].ID)
	})
	return out
}

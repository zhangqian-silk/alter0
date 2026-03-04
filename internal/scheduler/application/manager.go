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
	runs    map[string][]schedulerdomain.Run
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
		normalized := job.Normalized()
		if err := normalized.Validate(); err != nil {
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
		runs:         map[string][]schedulerdomain.Run{},
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
	job = job.Normalized()
	if err := job.Validate(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	previous, existed := m.jobs[job.ID]
	m.jobs[job.ID] = job
	m.stopRunnerLocked(job.ID)
	if m.started && job.Enabled {
		m.startRunnerLocked(job)
	}
	if err := m.storeLocked(); err != nil {
		m.stopRunnerLocked(job.ID)
		if existed {
			m.jobs[job.ID] = previous
			if m.started && previous.Enabled {
				m.startRunnerLocked(previous)
			}
		} else {
			delete(m.jobs, job.ID)
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

func (m *Manager) ListRuns(jobID string) []schedulerdomain.Run {
	key := normalize(jobID)
	if key == "" {
		return []schedulerdomain.Run{}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	items := m.runs[key]
	if len(items) == 0 {
		return []schedulerdomain.Run{}
	}

	out := make([]schedulerdomain.Run, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].FiredAt.Equal(out[j].FiredAt) {
			return out[i].ID > out[j].ID
		}
		return out[i].FiredAt.After(out[j].FiredAt)
	})
	return out
}

func (m *Manager) startRunnerLocked(job schedulerdomain.Job) {
	if m.baseCtx == nil {
		return
	}
	ctx, cancel := context.WithCancel(m.baseCtx)
	m.runners[job.ID] = cancel

	go func() {
		if job.EffectiveScheduleMode() == schedulerdomain.ScheduleModeInterval {
			m.runIntervalLoop(ctx, job)
			return
		}
		m.runCronLoop(ctx, job)
	}()
}

func (m *Manager) runIntervalLoop(ctx context.Context, job schedulerdomain.Job) {
	ticker := time.NewTicker(job.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case firedAt := <-ticker.C:
			m.triggerJob(ctx, job, firedAt.UTC())
		}
	}
}

func (m *Manager) runCronLoop(ctx context.Context, job schedulerdomain.Job) {
	location, err := time.LoadLocation(job.EffectiveTimezone())
	if err != nil {
		m.logger.Error("cron job stopped due to invalid timezone",
			slog.String("job_id", job.ID),
			slog.String("timezone", job.EffectiveTimezone()),
			slog.String("error", err.Error()),
		)
		return
	}

	expression := job.EffectiveCronExpression()
	for {
		nextAt, nextErr := schedulerdomain.NextCronFireAt(expression, location, time.Now().UTC())
		if nextErr != nil {
			m.logger.Error("cron job stopped due to invalid expression",
				slog.String("job_id", job.ID),
				slog.String("cron_expression", expression),
				slog.String("error", nextErr.Error()),
			)
			return
		}
		wait := time.Until(nextAt)
		if wait < 0 {
			wait = 0
		}

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		case <-timer.C:
			m.triggerJob(ctx, job, nextAt.UTC())
		}
	}
}

func (m *Manager) cronSessionID(jobID string, firedAt time.Time) string {
	normalizedJobID := normalize(jobID)
	if normalizedJobID == "" {
		normalizedJobID = "cron"
	}
	suffix := ""
	if m.idGenerator != nil {
		suffix = strings.TrimSpace(m.idGenerator.NewID())
	}
	if suffix == "" {
		suffix = firedAt.Format("20060102150405")
	}
	return "cron-" + normalizedJobID + "-" + firedAt.Format("20060102t150405") + "-" + suffix
}

func (m *Manager) runID(firedAt time.Time) string {
	if m.idGenerator != nil {
		runID := strings.TrimSpace(m.idGenerator.NewID())
		if runID != "" {
			return runID
		}
	}
	return "run-" + firedAt.Format("20060102150405")
}

func (m *Manager) appendRun(run schedulerdomain.Run) {
	key := normalize(run.JobID)
	if key == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	items := append(m.runs[key], run)
	if len(items) > 200 {
		items = items[len(items)-200:]
	}
	m.runs[key] = items
}

func (m *Manager) triggerJob(ctx context.Context, job schedulerdomain.Job, firedAt time.Time) {
	sessionID := m.cronSessionID(job.ID, firedAt)
	run := schedulerdomain.Run{
		ID:        m.runID(firedAt),
		JobID:     job.ID,
		FiredAt:   firedAt.UTC(),
		SessionID: sessionID,
		Status:    schedulerdomain.RunStatusSuccess,
	}

	channelID := strings.TrimSpace(job.ChannelID)
	if channelID == "" {
		channelID = "scheduler-default"
	}

	metadata := cloneMap(job.Metadata)
	metadata["trigger_type"] = string(shareddomain.TriggerTypeCron)
	metadata["job_id"] = job.ID
	metadata["job_name"] = job.Name
	metadata["fired_at"] = firedAt.UTC().Format(time.RFC3339)
	metadata["schedule_mode"] = string(job.EffectiveScheduleMode())
	metadata["cron_expression"] = job.EffectiveCronExpression()
	if strings.TrimSpace(job.SessionID) != "" {
		metadata["job_session_id"] = strings.TrimSpace(job.SessionID)
	}

	msg := shareddomain.UnifiedMessage{
		MessageID:     m.idGenerator.NewID(),
		SessionID:     sessionID,
		UserID:        job.UserID,
		ChannelID:     channelID,
		ChannelType:   shareddomain.ChannelTypeScheduler,
		TriggerType:   shareddomain.TriggerTypeCron,
		Content:       job.Content,
		Metadata:      metadata,
		TraceID:       m.idGenerator.NewID(),
		CorrelationID: job.ID,
		ReceivedAt:    firedAt,
	}

	m.telemetry.CountGateway(string(msg.ChannelType))
	result, err := m.orchestrator.Handle(ctx, msg)
	if err != nil {
		run.Status = schedulerdomain.RunStatusFailed
		m.appendRun(run)
		m.logger.Error("cron job failed",
			slog.String("job_id", job.ID),
			slog.String("job_name", job.Name),
			slog.String("run_id", run.ID),
			slog.String("trace_id", msg.TraceID),
			slog.String("session_id", msg.SessionID),
			slog.String("message_id", msg.MessageID),
			slog.String("error", err.Error()),
			slog.String("error_code", result.ErrorCode),
		)
		return
	}

	m.appendRun(run)
	m.logger.Info("cron job handled",
		slog.String("job_id", job.ID),
		slog.String("job_name", job.Name),
		slog.String("run_id", run.ID),
		slog.String("trace_id", msg.TraceID),
		slog.String("session_id", msg.SessionID),
		slog.String("message_id", msg.MessageID),
		slog.String("route", string(result.Route)),
	)
}

func (m *Manager) stopRunnerLocked(jobID string) {
	cancel, ok := m.runners[jobID]
	if !ok {
		return
	}
	cancel()
	delete(m.runners, jobID)
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

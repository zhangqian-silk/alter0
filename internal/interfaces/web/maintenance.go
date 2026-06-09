package web

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	sessionapp "alter0/internal/session/application"
	sharedapp "alter0/internal/shared/application"
	shareddomain "alter0/internal/shared/domain"
	taskapp "alter0/internal/task/application"
	taskdomain "alter0/internal/task/domain"
)

const (
	defaultMaintenanceInactiveDuration = 7 * 24 * time.Hour
	defaultMemoryMaintenanceJobID      = "system-memory-maintenance"
	defaultSessionCleanupJobID         = "system-session-cleanup"
)

type sessionMaintenanceService interface {
	CleanupInactiveSessions(options sessionapp.CleanupInactiveSessionsOptions) (sessionapp.CleanupInactiveSessionsResult, error)
}

type sessionPinService interface {
	SetSessionPinned(sessionID string, pinned bool) error
}

type sessionTouchService interface {
	TouchSession(sessionID string, at time.Time) error
}

type maintenanceRunResponse struct {
	JobID                 string    `json:"job_id"`
	Status                string    `json:"status"`
	StartedAt             time.Time `json:"started_at,omitempty"`
	FinishedAt            time.Time `json:"finished_at,omitempty"`
	NextRunAt             time.Time `json:"next_run_at,omitempty"`
	DeletedCount          int       `json:"deleted_count,omitempty"`
	SkippedPinnedCount    int       `json:"skipped_pinned_count,omitempty"`
	SkippedProtectedCount int       `json:"skipped_protected_count,omitempty"`
	ScannedCount          int       `json:"scanned_count,omitempty"`
	ChangedFiles          []string  `json:"changed_files,omitempty"`
	Error                 string    `json:"error,omitempty"`
}

type maintenanceStatusResponse struct {
	Items []maintenanceRunResponse `json:"items"`
}

type maintenanceService struct {
	mu          sync.Mutex
	server      *Server
	idGenerator sharedapp.IDGenerator
	logger      *slog.Logger
	lastRuns    map[string]maintenanceRunResponse
}

func newMaintenanceService(server *Server, idGenerator sharedapp.IDGenerator, logger *slog.Logger) *maintenanceService {
	return &maintenanceService{
		server:      server,
		idGenerator: idGenerator,
		logger:      logger,
		lastRuns:    map[string]maintenanceRunResponse{},
	}
}

func (m *maintenanceService) Status(now time.Time) maintenanceStatusResponse {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	return maintenanceStatusResponse{Items: []maintenanceRunResponse{
		m.statusItemLocked(defaultMemoryMaintenanceJobID, now),
		m.statusItemLocked(defaultSessionCleanupJobID, now),
	}}
}

func (m *maintenanceService) RunMemoryMaintenance(ctx context.Context, now time.Time) maintenanceRunResponse {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	run := maintenanceRunResponse{
		JobID:     defaultMemoryMaintenanceJobID,
		Status:    "success",
		StartedAt: now,
		NextRunAt: nextDailyMaintenanceRun(now, 5, 10),
	}
	if m.server == nil || m.server.orchestrator == nil {
		run.Status = "failed"
		run.Error = "memory maintenance unavailable"
		run.FinishedAt = time.Now().UTC()
		m.storeRun(run)
		return run
	}
	messageID := m.newID("maintenance-memory")
	_, err := m.server.orchestrator.Handle(ctx, shareddomain.UnifiedMessage{
		MessageID:   messageID,
		SessionID:   "maintenance-memory-" + now.Format("20060102"),
		ChannelID:   "scheduler-default",
		ChannelType: shareddomain.ChannelTypeScheduler,
		TriggerType: shareddomain.TriggerTypeSystem,
		Content:     "Run system memory maintenance. Consolidate daily memory into long-term memory, remove duplicates, and report changed files.",
		Metadata: map[string]string{
			"alter0.skills.include":  `["memory-maintenance"]`,
			"alter0.memory.include":  `["memory_long_term","memory_daily_today","memory_daily_yesterday","user_md","soul_md"]`,
			"alter0.maintenance.job": defaultMemoryMaintenanceJobID,
		},
		TraceID:    m.newID("maintenance-trace"),
		ReceivedAt: now,
	})
	if err != nil {
		run.Status = "failed"
		run.Error = err.Error()
	}
	run.FinishedAt = time.Now().UTC()
	m.storeRun(run)
	return run
}

func (m *maintenanceService) RunSessionCleanup(now time.Time) maintenanceRunResponse {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	run := maintenanceRunResponse{
		JobID:     defaultSessionCleanupJobID,
		Status:    "success",
		StartedAt: now,
		NextRunAt: nextDailyMaintenanceRun(now, 5, 20),
	}
	sessionCleaner, ok := m.server.sessions.(sessionMaintenanceService)
	if !ok {
		run.Status = "failed"
		run.Error = "session cleanup unavailable"
		run.FinishedAt = time.Now().UTC()
		m.storeRun(run)
		return run
	}
	result, err := sessionCleaner.CleanupInactiveSessions(sessionapp.CleanupInactiveSessionsOptions{
		Now:                 now,
		InactiveDuration:    defaultMaintenanceInactiveDuration,
		ProtectedSessionIDs: m.activeTaskSessionIDs(),
	})
	if err != nil {
		run.Status = "failed"
		run.Error = err.Error()
		run.FinishedAt = time.Now().UTC()
		m.storeRun(run)
		return run
	}
	run.DeletedCount = result.DeletedCount
	run.SkippedPinnedCount = result.SkippedPinnedCount
	run.SkippedProtectedCount = result.SkippedProtectedCount
	run.ScannedCount = result.ScannedCount
	var cleanupErrors []string
	for _, sessionID := range result.DeletedSessionIDs {
		if m.server.tasks != nil {
			if err := m.server.tasks.DeleteBySession(sessionID); err != nil {
				cleanupErrors = append(cleanupErrors, fmt.Sprintf("delete tasks for %s: %v", sessionID, err))
			}
		}
		if err := m.server.deleteConversationRuntimeSessionRegistryEntry(sessionID); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Sprintf("delete runtime registry for %s: %v", sessionID, err))
		}
		if err := removeConversationSessionWorkspace(m.server.workspaceRoot, sessionID); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Sprintf("delete workspace for %s: %v", sessionID, err))
		}
	}
	if len(cleanupErrors) > 0 {
		run.Status = "failed"
		run.Error = strings.Join(cleanupErrors, "; ")
	}
	run.FinishedAt = time.Now().UTC()
	m.storeRun(run)
	return run
}

func (m *maintenanceService) activeTaskSessionIDs() []string {
	if m == nil || m.server == nil || m.server.tasks == nil {
		return nil
	}
	protected := map[string]struct{}{}
	for _, status := range []taskdomain.TaskStatus{taskdomain.TaskStatusQueued, taskdomain.TaskStatusRunning} {
		for page := 1; ; page++ {
			result := m.server.tasks.List(taskapp.ListQuery{Status: status, Page: page, PageSize: 200})
			for _, item := range result.Items {
				if sessionID := strings.TrimSpace(item.SessionID); sessionID != "" {
					protected[sessionID] = struct{}{}
				}
			}
			if !result.Pagination.HasNext {
				break
			}
		}
	}
	sessionIDs := make([]string, 0, len(protected))
	for sessionID := range protected {
		sessionIDs = append(sessionIDs, sessionID)
	}
	sort.Strings(sessionIDs)
	return sessionIDs
}

func (s *Server) startMaintenanceScheduler(ctx context.Context) {
	if s == nil {
		return
	}
	s.ensureMaintenanceService()
	go s.runDailyMaintenanceJob(ctx, defaultMemoryMaintenanceJobID, 5, 10)
	go s.runDailyMaintenanceJob(ctx, defaultSessionCleanupJobID, 5, 20)
}

func (s *Server) runDailyMaintenanceJob(ctx context.Context, jobID string, hour int, minute int) {
	for {
		now := time.Now().UTC()
		next := nextDailyMaintenanceRun(now, hour, minute)
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			if s.maintenance == nil {
				s.ensureMaintenanceService()
			}
			switch jobID {
			case defaultMemoryMaintenanceJobID:
				_ = s.maintenance.RunMemoryMaintenance(ctx, time.Now().UTC())
			case defaultSessionCleanupJobID:
				_ = s.maintenance.RunSessionCleanup(time.Now().UTC())
			}
		}
	}
}

func (m *maintenanceService) storeRun(run maintenanceRunResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if run.NextRunAt.IsZero() {
		switch run.JobID {
		case defaultMemoryMaintenanceJobID:
			run.NextRunAt = nextDailyMaintenanceRun(time.Now().UTC(), 5, 10)
		case defaultSessionCleanupJobID:
			run.NextRunAt = nextDailyMaintenanceRun(time.Now().UTC(), 5, 20)
		}
	}
	m.lastRuns[run.JobID] = run
}

func (m *maintenanceService) statusItemLocked(jobID string, now time.Time) maintenanceRunResponse {
	if item, ok := m.lastRuns[jobID]; ok {
		return item
	}
	minute := 10
	if jobID == defaultSessionCleanupJobID {
		minute = 20
	}
	return maintenanceRunResponse{
		JobID:     jobID,
		Status:    "idle",
		NextRunAt: nextDailyMaintenanceRun(now, 5, minute),
	}
}

func (m *maintenanceService) newID(prefix string) string {
	if m.idGenerator != nil {
		if id := strings.TrimSpace(m.idGenerator.NewID()); id != "" {
			return id
		}
	}
	return prefix + "-" + time.Now().UTC().Format("20060102t150405.000000000")
}

func nextDailyMaintenanceRun(now time.Time, hour int, minute int) time.Time {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	local := now.In(location)
	next := time.Date(local.Year(), local.Month(), local.Day(), hour, minute, 0, 0, location)
	if !next.After(local) {
		next = next.AddDate(0, 0, 1)
	}
	return next.UTC()
}

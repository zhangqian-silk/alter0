package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	sessiondomain "alter0/internal/session/domain"
	sharedapp "alter0/internal/shared/application"
	shareddomain "alter0/internal/shared/domain"
	taskdomain "alter0/internal/task/domain"
)

const (
	MetadataTaskIDKey           = "task_id"
	MetadataTaskStatusKey       = "task_status"
	MetadataTaskTypeKey         = "alter0.task.type"
	MetadataTaskTimeoutMS       = "alter0.task.timeout_ms"
	MetadataTaskAsyncMode       = "alter0.async.mode"
	MetadataTaskArtifact        = "alter0.task.artifact"
	MetadataTaskIdempotencyKey  = "alter0.task.idempotency_key"
	MetadataTaskSourceMessageID = "alter0.task.source_message_id"
	MetadataTaskChannelIDKey    = "alter0.task.channel_id"
	MetadataTaskChannelTypeKey  = "alter0.task.channel_type"
	MetadataTaskTriggerTypeKey  = "alter0.task.trigger_type"
	MetadataTaskUserIDKey       = "alter0.task.user_id"
	MetadataTaskTraceIDKey      = "alter0.task.trace_id"
	MetadataTaskCorrelationKey  = "alter0.task.correlation_id"

	defaultLongContentThreshold = 240
)

const (
	defaultWorkerCount = asyncExecutorMaxConcurrency
	defaultTimeout     = 90 * time.Second
	defaultMaxRetries  = 1
)

var (
	ErrTaskNotFound          = errors.New("task not found")
	ErrTaskStopped           = errors.New("task service stopped")
	ErrTaskConflict          = errors.New("task state transition conflict")
	ErrArtifactNotFound      = errors.New("artifact not found")
	ErrArtifactContentAbsent = errors.New("artifact content unavailable")
)

type Orchestrator interface {
	Handle(ctx context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error)
}

type Store interface {
	Load(ctx context.Context) ([]taskdomain.Task, error)
	Save(ctx context.Context, tasks []taskdomain.Task) error
}

type artifactStore interface {
	ReadArtifact(ctx context.Context, taskID string, artifactID string) (taskdomain.TaskArtifact, []byte, error)
}

type sessionRecorder interface {
	Append(records ...sessiondomain.MessageRecord) error
}

type TaskSummaryRecorder interface {
	Record(task taskdomain.Task)
}

type ListQuery struct {
	SessionID string
	Status    taskdomain.TaskStatus
	Page      int
	PageSize  int
}

type Pagination struct {
	Page     int  `json:"page"`
	PageSize int  `json:"page_size"`
	Total    int  `json:"total"`
	HasNext  bool `json:"has_next"`
}

type TaskPage struct {
	Items      []taskdomain.Task `json:"items"`
	Pagination Pagination        `json:"pagination"`
}

type TaskLogPage struct {
	Items      []taskdomain.TaskLog `json:"items"`
	Cursor     int                  `json:"cursor"`
	NextCursor int                  `json:"next_cursor"`
	HasMore    bool                 `json:"has_more"`
}

type Options struct {
	WorkerCount          int
	Timeout              time.Duration
	MaxRetries           int
	LongContentThreshold int
	ArtifactKeywords     []string
	SummaryMemory        TaskSummaryRecorder
}

type Service struct {
	downstream  Orchestrator
	recorder    sessionRecorder
	store       Store
	idGenerator sharedapp.IDGenerator
	logger      *slog.Logger
	options     Options
	summary     TaskSummaryRecorder

	mu           sync.RWMutex
	stopped      bool
	executor     *asyncExecutor
	tasks        map[string]taskdomain.Task
	sessionIndex map[string][]string
	idempotency  map[string]string
	requests     map[string]shareddomain.UnifiedMessage
	inflight     map[string]context.CancelFunc
}

func NewService(
	ctx context.Context,
	downstream Orchestrator,
	recorder sessionRecorder,
	idGenerator sharedapp.IDGenerator,
	logger *slog.Logger,
	store Store,
	options Options,
) (*Service, error) {
	if downstream == nil {
		return nil, errors.New("downstream orchestrator is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if logger == nil {
		logger = slog.Default()
	}

	service := &Service{
		downstream:   downstream,
		recorder:     recorder,
		store:        store,
		idGenerator:  idGenerator,
		logger:       logger,
		options:      normalizeOptions(options),
		summary:      options.SummaryMemory,
		tasks:        map[string]taskdomain.Task{},
		sessionIndex: map[string][]string{},
		idempotency:  map[string]string{},
		requests:     map[string]shareddomain.UnifiedMessage{},
		inflight:     map[string]context.CancelFunc{},
	}
	service.executor = newAsyncExecutor(service.options.WorkerCount, service.runQueuedTask)

	if err := service.loadStore(ctx); err != nil {
		return nil, err
	}
	service.executor.start(ctx)

	go func() {
		<-ctx.Done()
		service.mu.Lock()
		service.stopped = true
		service.mu.Unlock()
	}()

	return service, nil
}

func normalizeOptions(options Options) Options {
	if options.WorkerCount <= 0 {
		options.WorkerCount = defaultWorkerCount
	}
	if options.WorkerCount > asyncExecutorMaxConcurrency {
		options.WorkerCount = asyncExecutorMaxConcurrency
	}
	if options.Timeout <= 0 {
		options.Timeout = defaultTimeout
	}
	if options.MaxRetries < 0 {
		options.MaxRetries = defaultMaxRetries
	}
	if options.LongContentThreshold <= 0 {
		options.LongContentThreshold = defaultLongContentThreshold
	}
	if len(options.ArtifactKeywords) == 0 {
		options.ArtifactKeywords = []string{
			"artifact",
			"report",
			"export",
			"generate file",
			"output file",
			"生成文件",
			"导出",
			"产物",
			"报告",
			"压缩包",
		}
	}
	return options
}

func (s *Service) loadStore(ctx context.Context) error {
	if s.store == nil {
		return nil
	}

	items, err := s.store.Load(ctx)
	if err != nil {
		return fmt.Errorf("load task state: %w", err)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})

	changed := false
	for _, item := range items {
		normalized, itemChanged, err := s.normalizeStoredTask(item)
		if err != nil {
			return fmt.Errorf("invalid task in store: %w", err)
		}
		changed = changed || itemChanged
		if _, exists := s.tasks[normalized.ID]; exists {
			return fmt.Errorf("duplicate task id in store: %s", normalized.ID)
		}
		s.tasks[normalized.ID] = cloneTask(normalized)
		if normalized.Status.IsTerminal() && s.summary != nil {
			s.summary.Record(cloneTask(normalized))
		}
		key := normalizeKey(normalized.SessionID)
		s.sessionIndex[key] = append(s.sessionIndex[key], normalized.ID)
		idempotencyKey := normalizeIdempotencyKey(normalized.SessionID, normalized.RequestMetadata[MetadataTaskIdempotencyKey])
		if idempotencyKey != "" {
			s.idempotency[idempotencyKey] = normalized.ID
		}
	}

	if changed {
		s.mu.Lock()
		defer s.mu.Unlock()
		if err := s.storeLocked(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) normalizeStoredTask(task taskdomain.Task) (taskdomain.Task, bool, error) {
	now := time.Now().UTC()
	changed := false
	task.ID = strings.TrimSpace(task.ID)
	task.SessionID = strings.TrimSpace(task.SessionID)
	task.SourceMessageID = strings.TrimSpace(task.SourceMessageID)
	task.MessageID = strings.TrimSpace(task.MessageID)
	if task.SourceMessageID == "" && task.MessageID != "" {
		task.SourceMessageID = task.MessageID
		changed = true
	}
	if task.MessageID == "" && task.SourceMessageID != "" {
		task.MessageID = task.SourceMessageID
		changed = true
	}
	if strings.TrimSpace(task.TaskType) == "" {
		task.TaskType = resolveTaskType(task)
		changed = true
	}
	task.RequestContent = strings.TrimSpace(task.RequestContent)
	task.RequestMetadata = normalizeTaskRequestMetadata(task.RequestMetadata)
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
		changed = true
	}
	if task.UpdatedAt.IsZero() {
		task.UpdatedAt = task.CreatedAt
		changed = true
	}
	if task.AcceptedAt.IsZero() {
		task.AcceptedAt = task.CreatedAt
		changed = true
	}
	if strings.TrimSpace(task.Phase) == "" {
		task.Phase = string(task.Status)
		changed = true
	}
	if task.PhaseUpdatedAt.IsZero() {
		task.PhaseUpdatedAt = task.UpdatedAt
		changed = true
	}
	if task.QueueWaitMS < 0 {
		task.QueueWaitMS = 0
		changed = true
	}
	if task.Status != taskdomain.TaskStatusQueued && task.QueuePosition != 0 {
		task.QueuePosition = 0
		changed = true
	}
	if task.Progress < 0 {
		task.Progress = 0
		changed = true
	}
	if task.Progress > 100 {
		task.Progress = 100
		changed = true
	}
	if task.TimeoutMS <= 0 {
		task.TimeoutMS = s.options.Timeout.Milliseconds()
		changed = true
	}
	if task.TimeoutAt.IsZero() {
		base := task.CreatedAt
		if !task.StartedAt.IsZero() {
			base = task.StartedAt
		}
		task.TimeoutAt = base.Add(time.Duration(task.TimeoutMS) * time.Millisecond)
		changed = true
	}

	for idx := range task.Logs {
		logItem := task.Logs[idx]
		if logItem.Seq <= 0 {
			logItem.Seq = idx + 1
			changed = true
		}
		logItem.Stage = strings.TrimSpace(logItem.Stage)
		if logItem.Stage == "" {
			logItem.Stage = "runtime"
		}
		if logItem.CreatedAt.IsZero() && !logItem.Timestamp.IsZero() {
			logItem.CreatedAt = logItem.Timestamp
			changed = true
		}
		if logItem.Timestamp.IsZero() && !logItem.CreatedAt.IsZero() {
			logItem.Timestamp = logItem.CreatedAt
			changed = true
		}
		if logItem.Timestamp.IsZero() {
			logItem.Timestamp = task.UpdatedAt
			logItem.CreatedAt = task.UpdatedAt
			changed = true
		}
		if !logItem.Level.IsValid() {
			logItem.Level = taskdomain.TaskLogLevelInfo
			changed = true
		}
		logItem.Message = strings.TrimSpace(logItem.Message)
		if logItem.Message == "" {
			logItem.Message = "task log"
			changed = true
		}
		task.Logs[idx] = logItem
	}

	if task.Status == taskdomain.TaskStatusQueued || task.Status == taskdomain.TaskStatusRunning {
		task.Status = taskdomain.TaskStatusFailed
		task.Phase = string(taskdomain.TaskStatusFailed)
		task.Progress = 100
		task.FinishedAt = now
		task.UpdatedAt = now
		task.PhaseUpdatedAt = now
		task.QueuePosition = 0
		task.ErrorCode = "task_interrupted"
		task.ErrorMessage = "task interrupted by service restart"
		task.Result.ErrorCode = task.ErrorCode
		task.Logs = appendTaskLog(task.Logs, "recovery", taskdomain.TaskLogLevelWarn, "task marked failed after restart", now)
		task.Summary = buildTaskSummary(task, nil)
		changed = true
	}

	if task.Status.IsTerminal() && task.FinishedAt.IsZero() {
		task.FinishedAt = task.UpdatedAt
		changed = true
	}
	if task.Status.IsTerminal() && task.Phase != string(task.Status) {
		task.Phase = string(task.Status)
		changed = true
	}
	if task.Status.IsTerminal() && task.Progress < 100 {
		task.Progress = 100
		changed = true
	}
	if strings.TrimSpace(task.Summary) == "" && task.Status.IsTerminal() {
		task.Summary = buildTaskSummary(task, nil)
		changed = true
	}
	if task.Status.IsTerminal() && task.TaskSummary.IsZero() {
		task.TaskSummary = buildTaskStructuredSummary(task)
		changed = true
	}
	if strings.TrimSpace(task.RequestContent) == "" {
		task.RequestContent = "task payload unavailable"
		changed = true
	}
	if strings.TrimSpace(task.MessageLink.TaskID) == "" {
		task.MessageLink.TaskID = task.ID
		changed = true
	}
	if strings.TrimSpace(task.MessageLink.SessionID) == "" {
		task.MessageLink.SessionID = task.SessionID
		changed = true
	}
	if strings.TrimSpace(task.MessageLink.RequestMessageID) == "" {
		task.MessageLink.RequestMessageID = task.SourceMessageID
		changed = true
	}
	normalizedArtifacts, artifactsChanged := normalizeTaskArtifacts(task.ID, task.Artifacts)
	task.Artifacts = normalizedArtifacts
	if artifactsChanged {
		changed = true
	}
	if err := task.Validate(); err != nil {
		return taskdomain.Task{}, false, err
	}
	return task, changed, nil
}

func (s *Service) ShouldRunAsync(msg shareddomain.UnifiedMessage) bool {
	assessment := s.AssessComplexity(msg)
	return assessment.ExecutionMode == ExecutionModeAsync
}

func (s *Service) Submit(msg shareddomain.UnifiedMessage) (taskdomain.Task, error) {
	if err := msg.Validate(); err != nil {
		return taskdomain.Task{}, fmt.Errorf("invalid message: %w", err)
	}

	now := time.Now().UTC()
	metadata := normalizeTaskRequestMetadata(msg.Metadata)
	metadata[MetadataTaskChannelIDKey] = strings.TrimSpace(msg.ChannelID)
	metadata[MetadataTaskChannelTypeKey] = strings.TrimSpace(string(msg.ChannelType))
	metadata[MetadataTaskTriggerTypeKey] = strings.TrimSpace(string(msg.TriggerType))
	metadata[MetadataTaskUserIDKey] = strings.TrimSpace(msg.UserID)
	metadata[MetadataTaskTraceIDKey] = strings.TrimSpace(msg.TraceID)
	metadata[MetadataTaskCorrelationKey] = strings.TrimSpace(msg.CorrelationID)
	sourceMessageID := strings.TrimSpace(metadataValue(metadata, MetadataTaskSourceMessageID))
	if sourceMessageID == "" {
		sourceMessageID = strings.TrimSpace(msg.MessageID)
	}
	timeout := s.resolveTaskTimeoutByMetadata(metadata)
	taskType := normalizeTaskType(metadataValue(metadata, MetadataTaskTypeKey))
	idempotencyKey := normalizeIdempotencyKey(msg.SessionID, metadataValue(metadata, MetadataTaskIdempotencyKey))

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return taskdomain.Task{}, ErrTaskStopped
	}
	if idempotencyKey != "" {
		if taskID, exists := s.idempotency[idempotencyKey]; exists {
			if existingTask, ok := s.tasks[taskID]; ok {
				return cloneTask(existingTask), nil
			}
			delete(s.idempotency, idempotencyKey)
		}
	}

	task := taskdomain.Task{
		ID:              s.newTaskID(),
		SessionID:       strings.TrimSpace(msg.SessionID),
		SourceMessageID: sourceMessageID,
		MessageID:       sourceMessageID,
		TaskType:        taskType,
		Status:          taskdomain.TaskStatusQueued,
		Phase:           string(taskdomain.TaskStatusQueued),
		Progress:        0,
		RetryCount:      0,
		MaxRetries:      s.options.MaxRetries,
		TimeoutMS:       timeout.Milliseconds(),
		TimeoutAt:       now.Add(timeout),
		AcceptedAt:      now,
		PhaseUpdatedAt:  now,
		CreatedAt:       now,
		UpdatedAt:       now,
		RequestContent:  strings.TrimSpace(msg.Content),
		RequestMetadata: metadata,
		MessageLink: taskdomain.TaskMessageLink{
			TaskID:           "",
			SessionID:        strings.TrimSpace(msg.SessionID),
			RequestMessageID: sourceMessageID,
		},
	}
	task.MessageLink.TaskID = task.ID
	task.Logs = appendTaskLog(task.Logs, "accept", taskdomain.TaskLogLevelInfo, "task accepted", now)

	queuePosition := 0
	if s.executor != nil {
		queuePosition = s.executor.enqueue(task.ID)
	}
	if queuePosition <= 0 {
		return taskdomain.Task{}, ErrTaskStopped
	}
	task.QueuePosition = queuePosition

	s.tasks[task.ID] = cloneTask(task)
	sessionKey := normalizeKey(task.SessionID)
	s.sessionIndex[sessionKey] = append(s.sessionIndex[sessionKey], task.ID)
	s.requests[task.ID] = buildMessageFromTask(task)
	if idempotencyKey != "" {
		s.idempotency[idempotencyKey] = task.ID
	}
	if err := s.storeLocked(); err != nil {
		delete(s.tasks, task.ID)
		delete(s.requests, task.ID)
		if s.executor != nil {
			s.executor.remove(task.ID)
		}
		s.removeFromSessionIndexLocked(sessionKey, task.ID)
		if idempotencyKey != "" {
			delete(s.idempotency, idempotencyKey)
		}
		return taskdomain.Task{}, err
	}
	return cloneTask(task), nil
}

func (s *Service) Get(taskID string) (taskdomain.Task, bool) {
	key := strings.TrimSpace(taskID)
	if key == "" {
		return taskdomain.Task{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.tasks[key]
	if !ok {
		return taskdomain.Task{}, false
	}
	cloned := cloneTask(item)
	s.refreshTaskQueueState(&cloned)
	return cloned, true
}

func (s *Service) ListBySession(sessionID string) []taskdomain.Task {
	key := normalizeKey(sessionID)
	if key == "" {
		return []taskdomain.Task{}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := s.sessionIndex[key]
	if len(ids) == 0 {
		return []taskdomain.Task{}
	}
	items := make([]taskdomain.Task, 0, len(ids))
	for idx := len(ids) - 1; idx >= 0; idx-- {
		taskID := ids[idx]
		task, ok := s.tasks[taskID]
		if !ok {
			continue
		}
		cloned := cloneTask(task)
		s.refreshTaskQueueState(&cloned)
		items = append(items, cloned)
	}
	return items
}

func (s *Service) List(query ListQuery) TaskPage {
	pagination := normalizeTaskPagination(query.Page, query.PageSize)
	sessionKey := normalizeKey(query.SessionID)
	status := query.Status

	s.mu.RLock()
	defer s.mu.RUnlock()

	filtered := make([]taskdomain.Task, 0, len(s.tasks))
	for _, item := range s.tasks {
		if sessionKey != "" && normalizeKey(item.SessionID) != sessionKey {
			continue
		}
		if strings.TrimSpace(string(status)) != "" && item.Status != status {
			continue
		}
		cloned := cloneTask(item)
		s.refreshTaskQueueState(&cloned)
		filtered = append(filtered, cloned)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].CreatedAt.Equal(filtered[j].CreatedAt) {
			return filtered[i].ID > filtered[j].ID
		}
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
	})

	from, to := taskPageBounds(len(filtered), pagination.Page, pagination.PageSize)
	items := make([]taskdomain.Task, 0, to-from)
	items = append(items, filtered[from:to]...)
	pagination.Total = len(filtered)
	pagination.HasNext = to < len(filtered)
	return TaskPage{
		Items:      items,
		Pagination: pagination,
	}
}

func (s *Service) ListLogs(taskID string, cursor int, limit int) (TaskLogPage, error) {
	key := strings.TrimSpace(taskID)
	if key == "" {
		return TaskLogPage{}, ErrTaskNotFound
	}
	if cursor < 0 {
		cursor = 0
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[key]
	if !ok {
		return TaskLogPage{}, ErrTaskNotFound
	}
	logs := task.Logs
	if cursor > len(logs) {
		cursor = len(logs)
	}
	end := cursor + limit
	if end > len(logs) {
		end = len(logs)
	}
	items := make([]taskdomain.TaskLog, 0, end-cursor)
	items = append(items, logs[cursor:end]...)
	return TaskLogPage{
		Items:      items,
		Cursor:     cursor,
		NextCursor: end,
		HasMore:    end < len(logs),
	}, nil
}

func (s *Service) ListArtifacts(taskID string) ([]taskdomain.TaskArtifact, error) {
	key := strings.TrimSpace(taskID)
	if key == "" {
		return nil, ErrTaskNotFound
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[key]
	if !ok {
		return nil, ErrTaskNotFound
	}
	artifacts, _ := normalizeTaskArtifacts(task.ID, task.Artifacts)
	if len(artifacts) == 0 {
		return []taskdomain.TaskArtifact{}, nil
	}
	items := make([]taskdomain.TaskArtifact, 0, len(artifacts))
	items = append(items, artifacts...)
	return items, nil
}

func (s *Service) ReadArtifact(ctx context.Context, taskID string, artifactID string) (taskdomain.TaskArtifact, []byte, error) {
	key := strings.TrimSpace(taskID)
	artifactKey := strings.TrimSpace(artifactID)
	if key == "" || artifactKey == "" {
		return taskdomain.TaskArtifact{}, nil, ErrArtifactNotFound
	}

	s.mu.RLock()
	task, ok := s.tasks[key]
	s.mu.RUnlock()
	if !ok {
		return taskdomain.TaskArtifact{}, nil, ErrTaskNotFound
	}

	items, _ := normalizeTaskArtifacts(task.ID, task.Artifacts)
	artifact, exists := findArtifactByID(items, artifactKey)
	if !exists {
		return taskdomain.TaskArtifact{}, nil, ErrArtifactNotFound
	}

	if reader, ok := s.store.(artifactStore); ok {
		resolved, raw, err := reader.ReadArtifact(ctx, key, artifactKey)
		if err != nil {
			return taskdomain.TaskArtifact{}, nil, err
		}
		normalized, _ := normalizeTaskArtifacts(key, []taskdomain.TaskArtifact{resolved})
		if len(normalized) == 1 {
			resolved = normalized[0]
		}
		if resolved.Size <= 0 {
			resolved.Size = int64(len(raw))
		}
		return resolved, raw, nil
	}

	if strings.TrimSpace(artifact.Content) == "" {
		return taskdomain.TaskArtifact{}, nil, ErrArtifactContentAbsent
	}
	raw := []byte(artifact.Content)
	artifact.Size = int64(len(raw))
	return artifact, raw, nil
}

func (s *Service) Retry(taskID string) (taskdomain.Task, error) {
	key := strings.TrimSpace(taskID)
	if key == "" {
		return taskdomain.Task{}, ErrTaskNotFound
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return taskdomain.Task{}, ErrTaskStopped
	}
	task, ok := s.tasks[key]
	if !ok {
		return taskdomain.Task{}, ErrTaskNotFound
	}
	if task.Status != taskdomain.TaskStatusFailed && task.Status != taskdomain.TaskStatusCanceled {
		return taskdomain.Task{}, ErrTaskConflict
	}

	now := time.Now().UTC()
	timeout := s.resolveTaskTimeoutByMetadata(task.RequestMetadata)
	task.Status = taskdomain.TaskStatusQueued
	task.Phase = string(taskdomain.TaskStatusQueued)
	task.Progress = 0
	task.ErrorCode = ""
	task.ErrorMessage = ""
	task.Result.ErrorCode = ""
	task.Result.Output = ""
	task.FinishedAt = time.Time{}
	task.StartedAt = time.Time{}
	task.QueueWaitMS = 0
	task.AcceptedAt = now
	task.PhaseUpdatedAt = now
	task.UpdatedAt = now
	task.TimeoutMS = timeout.Milliseconds()
	task.TimeoutAt = now.Add(timeout)
	task.RetryCount++
	task.Logs = appendTaskLog(task.Logs, "retry", taskdomain.TaskLogLevelInfo, "task re-queued by retry action", now)
	task.Summary = ""
	task.TaskSummary = taskdomain.TaskSummary{}
	if strings.TrimSpace(task.MessageLink.TaskID) == "" {
		task.MessageLink.TaskID = task.ID
	}
	if strings.TrimSpace(task.MessageLink.SessionID) == "" {
		task.MessageLink.SessionID = task.SessionID
	}
	if strings.TrimSpace(task.MessageLink.RequestMessageID) == "" {
		task.MessageLink.RequestMessageID = task.SourceMessageID
	}

	queuePosition := 0
	if s.executor != nil {
		queuePosition = s.executor.enqueue(task.ID)
	}
	if queuePosition <= 0 {
		return taskdomain.Task{}, ErrTaskStopped
	}
	task.QueuePosition = queuePosition

	s.tasks[task.ID] = cloneTask(task)
	if _, exists := s.requests[task.ID]; !exists {
		s.requests[task.ID] = buildMessageFromTask(task)
	}
	if err := s.storeLocked(); err != nil {
		if s.executor != nil {
			s.executor.remove(task.ID)
		}
		return taskdomain.Task{}, err
	}
	return cloneTask(task), nil
}

func (s *Service) Cancel(taskID string) (taskdomain.Task, error) {
	key := strings.TrimSpace(taskID)
	if key == "" {
		return taskdomain.Task{}, ErrTaskNotFound
	}

	var cancel context.CancelFunc
	s.mu.Lock()
	task, ok := s.tasks[key]
	if !ok {
		s.mu.Unlock()
		return taskdomain.Task{}, ErrTaskNotFound
	}
	if task.Status.IsTerminal() {
		s.mu.Unlock()
		return taskdomain.Task{}, ErrTaskConflict
	}

	now := time.Now().UTC()
	task.Logs = appendTaskLog(task.Logs, "cancel", taskdomain.TaskLogLevelWarn, "task cancel requested", now)

	switch task.Status {
	case taskdomain.TaskStatusQueued:
		if s.executor != nil {
			s.executor.remove(task.ID)
		}
		delete(s.requests, task.ID)
		task.Status = taskdomain.TaskStatusCanceled
		task.Phase = string(taskdomain.TaskStatusCanceled)
		task.Progress = 100
		task.UpdatedAt = now
		task.FinishedAt = now
		task.PhaseUpdatedAt = now
		task.QueuePosition = 0
		task.ErrorCode = "task_canceled"
		task.ErrorMessage = "task canceled before execution"
		task.Result.ErrorCode = task.ErrorCode
		task.Summary = buildTaskSummary(task, nil)
		task.TaskSummary = buildTaskStructuredSummary(task)
		task.Logs = appendTaskLog(task.Logs, "cancel", taskdomain.TaskLogLevelInfo, "task canceled before worker picked it", now)
	case taskdomain.TaskStatusRunning:
		cancel = s.inflight[task.ID]
	default:
		s.mu.Unlock()
		return taskdomain.Task{}, ErrTaskConflict
	}

	s.tasks[task.ID] = cloneTask(task)
	if err := s.storeLocked(); err != nil {
		s.mu.Unlock()
		return taskdomain.Task{}, err
	}
	item := cloneTask(task)
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if item.Status == taskdomain.TaskStatusCanceled {
		s.appendSummaryToSession(item)
		s.recordTaskSummary(item)
	}
	return item, nil
}

func (s *Service) runQueuedTask(ctx context.Context, taskID string, workerID int) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return
	}
	msg, runCtx, cancel, ok := s.prepareQueuedTask(taskID, ctx, workerID)
	if !ok {
		s.finishRuntime(taskID)
		return
	}
	s.executeTask(taskID, msg, runCtx, workerID)
	cancel()
	s.finishRuntime(taskID)
}

func (s *Service) prepareQueuedTask(
	taskID string,
	ctx context.Context,
	workerID int,
) (shareddomain.UnifiedMessage, context.Context, context.CancelFunc, bool) {
	s.mu.Lock()
	if s.stopped || ctx.Err() != nil {
		s.mu.Unlock()
		return shareddomain.UnifiedMessage{}, nil, nil, false
	}

	task, ok := s.tasks[taskID]
	if !ok {
		s.mu.Unlock()
		return shareddomain.UnifiedMessage{}, nil, nil, false
	}
	if task.Status != taskdomain.TaskStatusQueued {
		s.mu.Unlock()
		return shareddomain.UnifiedMessage{}, nil, nil, false
	}

	msg, ok := s.requests[taskID]
	if !ok {
		msg = buildMessageFromTask(task)
	}
	if err := msg.Validate(); err != nil {
		now := time.Now().UTC()
		task.Status = taskdomain.TaskStatusFailed
		task.Phase = string(taskdomain.TaskStatusFailed)
		task.Progress = 100
		task.UpdatedAt = now
		task.FinishedAt = now
		task.PhaseUpdatedAt = now
		task.QueuePosition = 0
		task.ErrorCode = "task_payload_missing"
		task.ErrorMessage = "task request payload is unavailable"
		task.Result.ErrorCode = task.ErrorCode
		task.Summary = buildTaskSummary(task, nil)
		task.TaskSummary = buildTaskStructuredSummary(task)
		task.Logs = appendTaskLog(task.Logs, "prepare", taskdomain.TaskLogLevelError, "task failed before execution: payload unavailable", now)
		s.tasks[taskID] = cloneTask(task)
		_ = s.storeLocked()
		item := cloneTask(task)
		s.mu.Unlock()
		s.appendSummaryToSession(item)
		s.recordTaskSummary(item)
		return shareddomain.UnifiedMessage{}, nil, nil, false
	}
	s.requests[taskID] = cloneMessage(msg)

	now := time.Now().UTC()
	task.Status = taskdomain.TaskStatusRunning
	task.Phase = string(taskdomain.TaskStatusRunning)
	task.PhaseUpdatedAt = now
	if task.StartedAt.IsZero() {
		task.StartedAt = now
	}
	acceptedAt := task.AcceptedAt
	if acceptedAt.IsZero() {
		acceptedAt = task.CreatedAt
		task.AcceptedAt = acceptedAt
	}
	if !acceptedAt.IsZero() {
		waitDuration := now.Sub(acceptedAt)
		if waitDuration < 0 {
			waitDuration = 0
		}
		task.QueueWaitMS = waitDuration.Milliseconds()
	}
	task.QueuePosition = 0
	if task.Progress < 10 {
		task.Progress = 10
	}
	task.UpdatedAt = now
	task.Logs = appendTaskLog(task.Logs, "running", taskdomain.TaskLogLevelInfo, fmt.Sprintf("task started by worker-%d", workerID), now)
	task.Logs = appendTaskLog(task.Logs, "terminal", taskdomain.TaskLogLevelInfo, "已运行 "+task.RequestContent, now)
	timeout := s.resolveTaskTimeout(msg)
	task.TimeoutMS = timeout.Milliseconds()
	task.TimeoutAt = now.Add(timeout)
	s.tasks[taskID] = cloneTask(task)

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	s.inflight[taskID] = cancel
	if err := s.storeLocked(); err != nil && s.logger != nil {
		s.logger.Warn("persist running task state failed",
			slog.String("task_id", taskID),
			slog.String("error", err.Error()),
		)
	}
	s.mu.Unlock()
	return cloneMessage(msg), runCtx, cancel, true
}

func (s *Service) executeTask(taskID string, msg shareddomain.UnifiedMessage, runCtx context.Context, workerID int) {
	maxAttempts := s.options.MaxRetries + 1
	var lastResult shareddomain.OrchestrationResult
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			s.markTaskPhase(taskID, string(taskdomain.TaskStatusRunning))
		}
		execMessage := cloneMessage(msg)
		execMessage.Metadata = cloneStringMap(execMessage.Metadata)
		execMessage.Metadata[MetadataTaskIDKey] = taskID
		execMessage.Metadata[MetadataTaskStatusKey] = string(taskdomain.TaskStatusRunning)
		execMessage.Metadata["task_retry"] = strconv.Itoa(attempt)
		execMessage.Metadata["task_max_retries"] = strconv.Itoa(s.options.MaxRetries)
		lastResult, lastErr = s.downstream.Handle(runCtx, execMessage)
		if lastErr == nil {
			task := s.completeTask(taskID, taskdomain.TaskStatusSuccess, lastResult, nil, "task completed", taskdomain.TaskLogLevelInfo, 100)
			s.appendSummaryToSession(task)
			s.recordTaskSummary(task)
			return
		}

		if runCtx.Err() != nil {
			errorCode := "task_canceled"
			status := taskdomain.TaskStatusCanceled
			level := taskdomain.TaskLogLevelWarn
			progress := 100
			logMessage := "task canceled"
			if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
				errorCode = "task_timeout"
				status = taskdomain.TaskStatusFailed
				level = taskdomain.TaskLogLevelError
				logMessage = "task timed out"
			}
			task := s.completeTask(taskID, status, lastResult, lastErr, logMessage, level, progress, errorCode)
			s.appendSummaryToSession(task)
			s.recordTaskSummary(task)
			return
		}

		if attempt < s.options.MaxRetries {
			s.recordRetry(taskID, workerID, lastErr)
			continue
		}
		task := s.completeTask(taskID, taskdomain.TaskStatusFailed, lastResult, lastErr, "task execution failed", taskdomain.TaskLogLevelError, 100)
		s.appendSummaryToSession(task)
		s.recordTaskSummary(task)
		return
	}

	task := s.completeTask(taskID, taskdomain.TaskStatusFailed, lastResult, lastErr, "task execution failed", taskdomain.TaskLogLevelError, 100)
	s.appendSummaryToSession(task)
	s.recordTaskSummary(task)
}

func (s *Service) recordRetry(taskID string, workerID int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return
	}
	now := time.Now().UTC()
	task.RetryCount++
	task.UpdatedAt = now
	retryCount := task.RetryCount
	nextProgress := 10 + retryCount*20
	if nextProgress > 95 {
		nextProgress = 95
	}
	if task.Progress < nextProgress {
		task.Progress = nextProgress
	}
	task.Phase = "retrying"
	task.PhaseUpdatedAt = now
	task.Logs = appendTaskLog(task.Logs, "retry", taskdomain.TaskLogLevelWarn, fmt.Sprintf("worker-%d retry %d/%d: %s", workerID, retryCount, task.MaxRetries, strings.TrimSpace(err.Error())), now)
	s.tasks[taskID] = cloneTask(task)
	if saveErr := s.storeLocked(); saveErr != nil && s.logger != nil {
		s.logger.Warn("persist retry state failed",
			slog.String("task_id", task.ID),
			slog.String("error", saveErr.Error()),
		)
	}
}

func (s *Service) completeTask(
	taskID string,
	status taskdomain.TaskStatus,
	result shareddomain.OrchestrationResult,
	handleErr error,
	logMessage string,
	logLevel taskdomain.TaskLogLevel,
	progress int,
	overrideCode ...string,
) taskdomain.Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return taskdomain.Task{}
	}

	now := time.Now().UTC()
	task.Status = status
	task.Phase = string(status)
	task.Progress = progress
	task.UpdatedAt = now
	task.FinishedAt = now
	task.PhaseUpdatedAt = now
	task.QueuePosition = 0
	task.TimeoutAt = time.Time{}
	task.Result = toTaskResult(result)

	if status == taskdomain.TaskStatusSuccess {
		task.ErrorCode = ""
		task.ErrorMessage = ""
		task.Result.ErrorCode = ""
		if output := strings.TrimSpace(result.Output); output != "" {
			task.Artifacts = append(task.Artifacts, taskdomain.TaskArtifact{
				ArtifactID:   task.ID + "-result",
				ArtifactType: "result",
				Name:         "result.txt",
				ContentType:  "text/plain",
				Size:         int64(len([]byte(output))),
				Content:      output,
				URI:          "inline://result.txt",
				Summary:      summarySnippet(output, 120),
				CreatedAt:    now,
			})
		}
	} else {
		code := strings.TrimSpace(result.ErrorCode)
		if len(overrideCode) > 0 && strings.TrimSpace(overrideCode[0]) != "" {
			code = strings.TrimSpace(overrideCode[0])
		}
		if code == "" {
			switch status {
			case taskdomain.TaskStatusCanceled:
				code = "task_canceled"
			default:
				code = "task_failed"
			}
		}
		task.ErrorCode = code
		task.Result.ErrorCode = code
		if handleErr != nil {
			task.ErrorMessage = strings.TrimSpace(handleErr.Error())
		}
		if task.ErrorMessage == "" {
			task.ErrorMessage = code
		}
	}

	stage := "complete"
	switch status {
	case taskdomain.TaskStatusSuccess:
		stage = "success"
	case taskdomain.TaskStatusFailed:
		stage = "failed"
	case taskdomain.TaskStatusCanceled:
		stage = "canceled"
	}
	task.Logs = appendTaskTerminalLogs(task.Logs, result.Output, now)
	if handleErr != nil {
		task.Logs = appendTaskTerminalLogs(task.Logs, handleErr.Error(), now)
	}
	task.Logs = appendTaskLog(task.Logs, stage, logLevel, strings.TrimSpace(logMessage), now)
	task.Artifacts, _ = normalizeTaskArtifacts(task.ID, task.Artifacts)
	task.Summary = buildTaskSummary(task, handleErr)
	task.TaskSummary = buildTaskStructuredSummary(task)
	s.tasks[taskID] = cloneTask(task)

	if err := s.storeLocked(); err != nil && s.logger != nil {
		s.logger.Warn("persist completed task state failed",
			slog.String("task_id", taskID),
			slog.String("error", err.Error()),
		)
	}
	return cloneTask(task)
}

func (s *Service) appendSummaryToSession(task taskdomain.Task) {
	if s.recorder == nil {
		return
	}
	if strings.TrimSpace(task.Summary) == "" {
		return
	}
	record := sessiondomain.MessageRecord{
		MessageID: s.newMessageID(task.ID),
		SessionID: task.SessionID,
		Role:      sessiondomain.MessageRoleAssistant,
		Content:   task.Summary,
		Timestamp: time.Now().UTC(),
		RouteResult: sessiondomain.RouteResult{
			Route:     task.Result.Route,
			ErrorCode: task.ErrorCode,
			TaskID:    task.ID,
		},
	}
	if err := s.recorder.Append(record); err != nil && s.logger != nil {
		s.logger.Error("append task summary to session failed",
			slog.String("task_id", task.ID),
			slog.String("session_id", task.SessionID),
			slog.String("error", err.Error()),
		)
		return
	}
	if err := s.bindResultMessage(task.ID, record.MessageID); err != nil && s.logger != nil {
		s.logger.Warn("bind task result message failed",
			slog.String("task_id", task.ID),
			slog.String("message_id", record.MessageID),
			slog.String("error", err.Error()),
		)
	}
}

func (s *Service) recordTaskSummary(task taskdomain.Task) {
	if s.summary == nil {
		return
	}
	if !task.Status.IsTerminal() {
		return
	}
	s.summary.Record(task)
}

func (s *Service) bindResultMessage(taskID string, resultMessageID string) error {
	taskID = strings.TrimSpace(taskID)
	resultMessageID = strings.TrimSpace(resultMessageID)
	if taskID == "" || resultMessageID == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return ErrTaskNotFound
	}
	task.MessageLink.TaskID = task.ID
	task.MessageLink.SessionID = task.SessionID
	if strings.TrimSpace(task.MessageLink.RequestMessageID) == "" {
		task.MessageLink.RequestMessageID = task.SourceMessageID
	}
	task.MessageLink.ResultMessageID = resultMessageID
	task.UpdatedAt = time.Now().UTC()
	s.tasks[taskID] = cloneTask(task)
	if err := s.storeLocked(); err != nil {
		return err
	}
	return nil
}

func (s *Service) finishRuntime(taskID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.requests, taskID)
	delete(s.inflight, taskID)
}

func (s *Service) markTaskPhase(taskID string, phase string) {
	taskID = strings.TrimSpace(taskID)
	phase = strings.TrimSpace(phase)
	if taskID == "" || phase == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[taskID]
	if !ok {
		return
	}
	task.Phase = phase
	task.PhaseUpdatedAt = time.Now().UTC()
	s.tasks[taskID] = cloneTask(task)
	if err := s.storeLocked(); err != nil && s.logger != nil {
		s.logger.Warn("persist task phase failed",
			slog.String("task_id", taskID),
			slog.String("phase", phase),
			slog.String("error", err.Error()),
		)
	}
}

func (s *Service) resolveTaskTimeout(msg shareddomain.UnifiedMessage) time.Duration {
	return s.resolveTaskTimeoutByMetadata(msg.Metadata)
}

func (s *Service) resolveTaskTimeoutByMetadata(metadata map[string]string) time.Duration {
	timeout := s.options.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	raw := strings.TrimSpace(metadataValue(metadata, MetadataTaskTimeoutMS))
	if raw == "" {
		return timeout
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return timeout
	}
	return time.Duration(value) * time.Millisecond
}

func normalizeTaskArtifacts(taskID string, artifacts []taskdomain.TaskArtifact) ([]taskdomain.TaskArtifact, bool) {
	if len(artifacts) == 0 {
		return []taskdomain.TaskArtifact{}, false
	}
	normalized := make([]taskdomain.TaskArtifact, 0, len(artifacts))
	changed := false
	taskKey := strings.TrimSpace(taskID)
	for idx, item := range artifacts {
		artifact := item
		artifactID := strings.TrimSpace(artifact.ArtifactID)
		if artifactID == "" {
			artifactID = fmt.Sprintf("%s-artifact-%d", taskKey, idx+1)
			artifact.ArtifactID = artifactID
			changed = true
		}
		if artifact.Size <= 0 && strings.TrimSpace(artifact.Content) != "" {
			artifact.Size = int64(len([]byte(artifact.Content)))
			changed = true
		}
		downloadURL, previewURL := buildArtifactURLs(taskKey, artifactID)
		if strings.TrimSpace(artifact.DownloadURL) == "" {
			artifact.DownloadURL = downloadURL
			changed = true
		}
		if supportsArtifactPreview(artifact.ContentType) {
			if strings.TrimSpace(artifact.PreviewURL) == "" {
				artifact.PreviewURL = previewURL
				changed = true
			}
		} else if strings.TrimSpace(artifact.PreviewURL) != "" {
			artifact.PreviewURL = ""
			changed = true
		}
		normalized = append(normalized, artifact)
	}
	return normalized, changed
}

func findArtifactByID(items []taskdomain.TaskArtifact, artifactID string) (taskdomain.TaskArtifact, bool) {
	needle := strings.TrimSpace(artifactID)
	if needle == "" {
		return taskdomain.TaskArtifact{}, false
	}
	for _, item := range items {
		if strings.TrimSpace(item.ArtifactID) == needle {
			return item, true
		}
	}
	return taskdomain.TaskArtifact{}, false
}

func buildArtifactURLs(taskID string, artifactID string) (string, string) {
	taskKey := url.PathEscape(strings.TrimSpace(taskID))
	artifactKey := url.PathEscape(strings.TrimSpace(artifactID))
	downloadURL := "/api/tasks/" + taskKey + "/artifacts/" + artifactKey + "/download"
	previewURL := "/api/tasks/" + taskKey + "/artifacts/" + artifactKey + "/preview"
	return downloadURL, previewURL
}

func supportsArtifactPreview(contentType string) bool {
	lower := strings.ToLower(strings.TrimSpace(contentType))
	if lower == "" {
		return false
	}
	if strings.HasPrefix(lower, "text/") || strings.HasPrefix(lower, "image/") {
		return true
	}
	switch lower {
	case "application/json", "application/xml", "application/yaml", "application/x-yaml", "application/javascript", "application/pdf", "application/xhtml+xml":
		return true
	default:
		return false
	}
}

func buildTaskSummary(task taskdomain.Task, handleErr error) string {
	switch task.Status {
	case taskdomain.TaskStatusSuccess:
		if len(task.Artifacts) > 0 {
			refs := make([]string, 0, len(task.Artifacts))
			for _, artifact := range task.Artifacts {
				artifactID := strings.TrimSpace(artifact.ArtifactID)
				if artifactID == "" {
					continue
				}
				downloadURL := strings.TrimSpace(artifact.DownloadURL)
				previewURL := strings.TrimSpace(artifact.PreviewURL)
				if downloadURL == "" {
					downloadURL, previewURL = buildArtifactURLs(task.ID, artifactID)
				}
				reference := fmt.Sprintf("task_id=%s artifact_id=%s download_url=%s", task.ID, artifactID, downloadURL)
				if previewURL != "" {
					reference += " preview_url=" + previewURL
				}
				refs = append(refs, reference)
			}
			if len(refs) > 0 {
				return fmt.Sprintf("async task %s completed with artifacts: %s", task.ID, strings.Join(refs, "; "))
			}
		}
		snippet := summarySnippet(task.Result.Output, 160)
		if snippet == "" {
			snippet = "completed without textual output"
		}
		return fmt.Sprintf("async task %s completed: %s", task.ID, snippet)
	case taskdomain.TaskStatusCanceled:
		return fmt.Sprintf("async task %s canceled", task.ID)
	case taskdomain.TaskStatusFailed:
		if task.ErrorCode == "task_timeout" {
			return fmt.Sprintf("async task %s timed out", task.ID)
		}
		message := strings.TrimSpace(task.ErrorMessage)
		if message == "" && handleErr != nil {
			message = strings.TrimSpace(handleErr.Error())
		}
		if message == "" {
			message = "execution failed"
		}
		return fmt.Sprintf("async task %s failed: %s", task.ID, summarySnippet(message, 160))
	default:
		return fmt.Sprintf("async task %s status: %s", task.ID, task.Status)
	}
}

func buildTaskStructuredSummary(task taskdomain.Task) taskdomain.TaskSummary {
	finishedAt := task.FinishedAt
	if finishedAt.IsZero() {
		finishedAt = task.UpdatedAt
	}
	if finishedAt.IsZero() {
		finishedAt = time.Now().UTC()
	}
	taskType := resolveTaskType(task)
	goal := summarySnippet(task.RequestContent, 180)
	if goal == "" {
		goal = "task goal unavailable"
	}
	result := resolveTaskSummaryResult(task)
	return taskdomain.TaskSummary{
		TaskID:     strings.TrimSpace(task.ID),
		TaskType:   taskType,
		Goal:       goal,
		Result:     result,
		Status:     task.Status,
		FinishedAt: finishedAt,
		Tags:       buildTaskSummaryTags(task, taskType),
	}
}

func resolveTaskType(task taskdomain.Task) string {
	taskType := strings.TrimSpace(task.TaskType)
	if taskType != "" {
		return strings.ToLower(taskType)
	}
	taskType = strings.TrimSpace(metadataValue(task.RequestMetadata, MetadataTaskTypeKey))
	if taskType != "" {
		return strings.ToLower(taskType)
	}
	switch task.Result.Route {
	case shareddomain.RouteCommand:
		return "command"
	case shareddomain.RouteNL:
		return "natural_language"
	default:
		return "task"
	}
}

func resolveTaskSummaryResult(task taskdomain.Task) string {
	switch task.Status {
	case taskdomain.TaskStatusSuccess:
		if snippet := summarySnippet(task.Result.Output, 220); snippet != "" {
			return snippet
		}
		return "completed without textual output"
	case taskdomain.TaskStatusCanceled:
		if reason := summarySnippet(task.ErrorMessage, 220); reason != "" {
			return reason
		}
		return "task canceled"
	case taskdomain.TaskStatusFailed:
		if reason := summarySnippet(task.ErrorMessage, 220); reason != "" {
			return reason
		}
		if code := strings.TrimSpace(task.ErrorCode); code != "" {
			return code
		}
		return "task failed"
	default:
		return summarySnippet(task.Summary, 220)
	}
}

func buildTaskSummaryTags(task taskdomain.Task, taskType string) []string {
	tags := []string{
		"task",
		strings.ToLower(strings.TrimSpace(string(task.Status))),
		strings.ToLower(strings.TrimSpace(taskType)),
	}
	if route := strings.TrimSpace(string(task.Result.Route)); route != "" {
		tags = append(tags, "route:"+strings.ToLower(route))
	}
	if code := strings.TrimSpace(task.ErrorCode); code != "" {
		tags = append(tags, "error:"+strings.ToLower(code))
	}
	seen := map[string]struct{}{}
	cleaned := make([]string, 0, len(tags))
	for _, tag := range tags {
		normalized := strings.TrimSpace(tag)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		cleaned = append(cleaned, normalized)
	}
	if len(cleaned) == 0 {
		return []string{"task"}
	}
	return cleaned
}

func summarySnippet(content string, maxRunes int) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ""
	}
	if maxRunes <= 0 {
		maxRunes = 160
	}
	runes := []rune(trimmed)
	if len(runes) <= maxRunes {
		return trimmed
	}
	return string(runes[:maxRunes]) + "..."
}

func toTaskResult(result shareddomain.OrchestrationResult) taskdomain.TaskResult {
	return taskdomain.TaskResult{
		Route:     result.Route,
		Output:    result.Output,
		ErrorCode: result.ErrorCode,
		Metadata:  cloneStringMap(result.Metadata),
	}
}

func appendTaskLog(logs []taskdomain.TaskLog, stage string, level taskdomain.TaskLogLevel, message string, timestamp time.Time) []taskdomain.TaskLog {
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	if !level.IsValid() {
		level = taskdomain.TaskLogLevelInfo
	}
	stage = strings.TrimSpace(stage)
	if stage == "" {
		stage = "runtime"
	}
	messageValue := message
	if strings.TrimSpace(messageValue) == "" {
		messageValue = "task log"
	}
	seq := len(logs) + 1
	return append(logs, taskdomain.TaskLog{
		Seq:       seq,
		Stage:     stage,
		CreatedAt: timestamp,
		Timestamp: timestamp,
		Level:     level,
		Message:   messageValue,
	})
}

func appendTaskTerminalLogs(logs []taskdomain.TaskLog, output string, timestamp time.Time) []taskdomain.TaskLog {
	lines := splitTaskTerminalOutput(output)
	if len(lines) == 0 {
		return logs
	}
	for idx, line := range lines {
		logTime := timestamp.Add(time.Duration(idx) * time.Millisecond)
		logs = appendTaskLog(logs, "terminal", taskdomain.TaskLogLevelInfo, line, logTime)
	}
	return logs
}

func splitTaskTerminalOutput(content string) []string {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	rawLines := strings.Split(normalized, "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func normalizeTaskType(taskType string) string {
	trimmed := strings.ToLower(strings.TrimSpace(taskType))
	if trimmed == "" {
		return "task"
	}
	return trimmed
}

func normalizeTaskRequestMetadata(metadata map[string]string) map[string]string {
	items := cloneStringMap(metadata)
	items[MetadataTaskTypeKey] = normalizeTaskType(items[MetadataTaskTypeKey])
	return items
}

func normalizeIdempotencyKey(sessionID string, key string) string {
	sessionKey := normalizeKey(sessionID)
	idempotencyKey := normalizeKey(key)
	if sessionKey == "" || idempotencyKey == "" {
		return ""
	}
	return sessionKey + "::" + idempotencyKey
}

func buildMessageFromTask(task taskdomain.Task) shareddomain.UnifiedMessage {
	metadata := normalizeTaskRequestMetadata(task.RequestMetadata)
	messageID := strings.TrimSpace(task.SourceMessageID)
	if messageID == "" {
		messageID = strings.TrimSpace(task.MessageID)
	}
	if messageID == "" {
		messageID = strings.TrimSpace(task.ID) + "-source"
	}
	channelID := strings.TrimSpace(metadata[MetadataTaskChannelIDKey])
	if channelID == "" {
		channelID = "web-default"
	}
	channelType := parseChannelType(metadata[MetadataTaskChannelTypeKey])
	triggerType := parseTriggerType(metadata[MetadataTaskTriggerTypeKey])
	traceID := strings.TrimSpace(metadata[MetadataTaskTraceIDKey])
	if traceID == "" {
		traceID = strings.TrimSpace(task.ID) + "-trace"
	}
	receivedAt := task.CreatedAt
	if receivedAt.IsZero() {
		receivedAt = time.Now().UTC()
	}

	return shareddomain.UnifiedMessage{
		MessageID:     messageID,
		SessionID:     strings.TrimSpace(task.SessionID),
		UserID:        strings.TrimSpace(metadata[MetadataTaskUserIDKey]),
		ChannelID:     channelID,
		ChannelType:   channelType,
		TriggerType:   triggerType,
		Content:       strings.TrimSpace(task.RequestContent),
		Metadata:      metadata,
		TraceID:       traceID,
		CorrelationID: strings.TrimSpace(metadata[MetadataTaskCorrelationKey]),
		ReceivedAt:    receivedAt,
	}
}

func parseChannelType(raw string) shareddomain.ChannelType {
	channelType := shareddomain.ChannelType(strings.ToLower(strings.TrimSpace(raw)))
	switch channelType {
	case shareddomain.ChannelTypeCLI, shareddomain.ChannelTypeWeb, shareddomain.ChannelTypeScheduler:
		return channelType
	default:
		return shareddomain.ChannelTypeWeb
	}
}

func parseTriggerType(raw string) shareddomain.TriggerType {
	triggerType := shareddomain.TriggerType(strings.ToLower(strings.TrimSpace(raw)))
	switch triggerType {
	case shareddomain.TriggerTypeUser, shareddomain.TriggerTypeCron, shareddomain.TriggerTypeSystem:
		return triggerType
	default:
		return shareddomain.TriggerTypeUser
	}
}

func normalizeTaskPagination(page int, pageSize int) Pagination {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return Pagination{
		Page:     page,
		PageSize: pageSize,
	}
}

func taskPageBounds(total int, page int, pageSize int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	offset := (page - 1) * pageSize
	if offset >= total {
		return total, total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	return offset, end
}

func (s *Service) storeLocked() error {
	if s.store == nil {
		return nil
	}
	items := make([]taskdomain.Task, 0, len(s.tasks))
	for _, item := range s.tasks {
		items = append(items, cloneTask(item))
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	if err := s.store.Save(context.Background(), items); err != nil {
		return fmt.Errorf("save task state: %w", err)
	}
	return nil
}

func (s *Service) newTaskID() string {
	if s.idGenerator != nil {
		for range 3 {
			id := strings.TrimSpace(s.idGenerator.NewID())
			if id == "" {
				continue
			}
			if _, exists := s.tasks[id]; !exists {
				return id
			}
		}
	}
	return fmt.Sprintf("task-%d", time.Now().UTC().UnixNano())
}

func (s *Service) newMessageID(taskID string) string {
	if s.idGenerator != nil {
		id := strings.TrimSpace(s.idGenerator.NewID())
		if id != "" {
			return id
		}
	}
	return strings.TrimSpace(taskID) + "-summary"
}

func cloneTask(task taskdomain.Task) taskdomain.Task {
	out := task
	out.RequestMetadata = cloneStringMap(task.RequestMetadata)
	out.TaskSummary = taskdomain.TaskSummary{
		TaskID:     task.TaskSummary.TaskID,
		TaskType:   task.TaskSummary.TaskType,
		Goal:       task.TaskSummary.Goal,
		Result:     task.TaskSummary.Result,
		Status:     task.TaskSummary.Status,
		FinishedAt: task.TaskSummary.FinishedAt,
		Tags:       append([]string(nil), task.TaskSummary.Tags...),
	}
	out.Result = taskdomain.TaskResult{
		Route:     task.Result.Route,
		Output:    task.Result.Output,
		ErrorCode: task.Result.ErrorCode,
		Metadata:  cloneStringMap(task.Result.Metadata),
	}
	if len(task.Logs) == 0 {
		out.Logs = []taskdomain.TaskLog{}
	} else {
		out.Logs = make([]taskdomain.TaskLog, 0, len(task.Logs))
		out.Logs = append(out.Logs, task.Logs...)
	}
	if len(task.Artifacts) == 0 {
		out.Artifacts = []taskdomain.TaskArtifact{}
	} else {
		out.Artifacts = make([]taskdomain.TaskArtifact, 0, len(task.Artifacts))
		out.Artifacts = append(out.Artifacts, task.Artifacts...)
	}
	return out
}

func cloneMessage(msg shareddomain.UnifiedMessage) shareddomain.UnifiedMessage {
	return shareddomain.UnifiedMessage{
		MessageID:     msg.MessageID,
		SessionID:     msg.SessionID,
		UserID:        msg.UserID,
		ChannelID:     msg.ChannelID,
		ChannelType:   msg.ChannelType,
		TriggerType:   msg.TriggerType,
		Content:       msg.Content,
		Metadata:      cloneStringMap(msg.Metadata),
		TraceID:       msg.TraceID,
		CorrelationID: msg.CorrelationID,
		ReceivedAt:    msg.ReceivedAt,
	}
}

func cloneStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(source))
	for key, value := range source {
		out[key] = value
	}
	return out
}

func metadataValue(metadata map[string]string, key string) string {
	if len(metadata) == 0 {
		return ""
	}
	return metadata[key]
}

func normalizeKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isTruthy(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func (s *Service) removeFromSessionIndexLocked(key, taskID string) {
	items := s.sessionIndex[key]
	if len(items) == 0 {
		return
	}
	for idx := range items {
		if items[idx] != taskID {
			continue
		}
		items = append(items[:idx], items[idx+1:]...)
		if len(items) == 0 {
			delete(s.sessionIndex, key)
		} else {
			s.sessionIndex[key] = items
		}
		return
	}
}

func (s *Service) refreshTaskQueueState(task *taskdomain.Task) {
	if task == nil {
		return
	}
	if task.Status != taskdomain.TaskStatusQueued {
		task.QueuePosition = 0
		return
	}
	if s.executor == nil {
		if task.QueuePosition <= 0 {
			task.QueuePosition = 1
		}
		return
	}
	position := s.executor.position(task.ID)
	task.QueuePosition = position
}

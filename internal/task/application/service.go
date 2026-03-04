package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	sessiondomain "alter0/internal/session/domain"
	sharedapp "alter0/internal/shared/application"
	shareddomain "alter0/internal/shared/domain"
	taskdomain "alter0/internal/task/domain"
)

const (
	MetadataTaskIDKey     = "task_id"
	MetadataTaskStatusKey = "task_status"

	metadataAsyncModeKey        = "alter0.async.mode"
	metadataTaskArtifactKey     = "alter0.task.artifact"
	metadataTaskTypeKey         = "alter0.task.type"
	metadataTaskTimeoutMSKey    = "alter0.task.timeout_ms"
	defaultLongContentThreshold = 240
)

const (
	defaultWorkerCount = 2
	defaultTimeout     = 90 * time.Second
	defaultMaxRetries  = 1
)

var (
	ErrTaskNotFound = errors.New("task not found")
	ErrTaskStopped  = errors.New("task service stopped")
)

type Orchestrator interface {
	Handle(ctx context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error)
}

type Store interface {
	Load(ctx context.Context) ([]taskdomain.Task, error)
	Save(ctx context.Context, tasks []taskdomain.Task) error
}

type sessionRecorder interface {
	Append(records ...sessiondomain.MessageRecord) error
}

type Options struct {
	WorkerCount          int
	Timeout              time.Duration
	MaxRetries           int
	LongContentThreshold int
	ArtifactKeywords     []string
}

type Service struct {
	downstream  Orchestrator
	recorder    sessionRecorder
	store       Store
	idGenerator sharedapp.IDGenerator
	logger      *slog.Logger
	options     Options

	mu           sync.RWMutex
	cond         *sync.Cond
	stopped      bool
	tasks        map[string]taskdomain.Task
	queue        []string
	sessionIndex map[string][]string
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
		tasks:        map[string]taskdomain.Task{},
		queue:        []string{},
		sessionIndex: map[string][]string{},
		requests:     map[string]shareddomain.UnifiedMessage{},
		inflight:     map[string]context.CancelFunc{},
	}
	service.cond = sync.NewCond(&service.mu)

	if err := service.loadStore(ctx); err != nil {
		return nil, err
	}
	for workerID := 0; workerID < service.options.WorkerCount; workerID++ {
		go service.runWorker(ctx, workerID+1)
	}

	go func() {
		<-ctx.Done()
		service.mu.Lock()
		service.stopped = true
		service.mu.Unlock()
		service.cond.Broadcast()
	}()

	return service, nil
}

func normalizeOptions(options Options) Options {
	if options.WorkerCount <= 0 {
		options.WorkerCount = defaultWorkerCount
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
		key := normalizeKey(normalized.SessionID)
		s.sessionIndex[key] = append(s.sessionIndex[key], normalized.ID)
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
	task.MessageID = strings.TrimSpace(task.MessageID)
	task.RequestContent = strings.TrimSpace(task.RequestContent)
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
		changed = true
	}
	if task.UpdatedAt.IsZero() {
		task.UpdatedAt = task.CreatedAt
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

	for idx := range task.Logs {
		logItem := task.Logs[idx]
		if logItem.Timestamp.IsZero() {
			logItem.Timestamp = task.UpdatedAt
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
		task.Progress = 100
		task.FinishedAt = now
		task.UpdatedAt = now
		task.ErrorCode = "task_interrupted"
		task.ErrorMessage = "task interrupted by service restart"
		task.Result.ErrorCode = task.ErrorCode
		task.Logs = append(task.Logs, taskdomain.TaskLog{
			Timestamp: now,
			Level:     taskdomain.TaskLogLevelWarn,
			Message:   "task marked failed after restart",
		})
		task.Summary = buildTaskSummary(task, nil)
		changed = true
	}

	if task.Status.IsTerminal() && task.FinishedAt.IsZero() {
		task.FinishedAt = task.UpdatedAt
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
	if strings.TrimSpace(task.RequestContent) == "" {
		task.RequestContent = "task payload unavailable"
		changed = true
	}
	if err := task.Validate(); err != nil {
		return taskdomain.Task{}, false, err
	}
	return task, changed, nil
}

func (s *Service) ShouldRunAsync(msg shareddomain.UnifiedMessage) bool {
	mode := strings.ToLower(strings.TrimSpace(metadataValue(msg.Metadata, metadataAsyncModeKey)))
	switch mode {
	case "sync", "inline", "off", "disable":
		return false
	case "force", "async", "background":
		return true
	}

	if isTruthy(metadataValue(msg.Metadata, metadataTaskArtifactKey)) {
		return true
	}
	taskType := strings.ToLower(strings.TrimSpace(metadataValue(msg.Metadata, metadataTaskTypeKey)))
	if strings.Contains(taskType, "artifact") || strings.Contains(taskType, "export") {
		return true
	}

	trimmed := strings.TrimSpace(msg.Content)
	if utf8.RuneCountInString(trimmed) >= s.options.LongContentThreshold {
		return true
	}
	lowerContent := strings.ToLower(trimmed)
	for _, keyword := range s.options.ArtifactKeywords {
		if strings.Contains(lowerContent, strings.ToLower(strings.TrimSpace(keyword))) {
			return true
		}
	}
	return false
}

func (s *Service) Submit(msg shareddomain.UnifiedMessage) (taskdomain.Task, error) {
	if err := msg.Validate(); err != nil {
		return taskdomain.Task{}, fmt.Errorf("invalid message: %w", err)
	}

	now := time.Now().UTC()
	task := taskdomain.Task{
		ID:              s.newTaskID(),
		SessionID:       strings.TrimSpace(msg.SessionID),
		MessageID:       strings.TrimSpace(msg.MessageID),
		Status:          taskdomain.TaskStatusQueued,
		Progress:        0,
		RetryCount:      0,
		MaxRetries:      s.options.MaxRetries,
		TimeoutMS:       s.resolveTaskTimeout(msg).Milliseconds(),
		CreatedAt:       now,
		UpdatedAt:       now,
		RequestContent:  strings.TrimSpace(msg.Content),
		RequestMetadata: cloneStringMap(msg.Metadata),
		Logs: []taskdomain.TaskLog{
			{
				Timestamp: now,
				Level:     taskdomain.TaskLogLevelInfo,
				Message:   "task accepted",
			},
		},
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return taskdomain.Task{}, ErrTaskStopped
	}

	s.tasks[task.ID] = cloneTask(task)
	sessionKey := normalizeKey(task.SessionID)
	s.sessionIndex[sessionKey] = append(s.sessionIndex[sessionKey], task.ID)
	s.queue = append(s.queue, task.ID)
	s.requests[task.ID] = cloneMessage(msg)
	if err := s.storeLocked(); err != nil {
		delete(s.tasks, task.ID)
		delete(s.requests, task.ID)
		s.removeFromQueueLocked(task.ID)
		s.removeFromSessionIndexLocked(sessionKey, task.ID)
		return taskdomain.Task{}, err
	}
	s.cond.Signal()
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
	return cloneTask(item), true
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
		items = append(items, cloneTask(task))
	}
	return items
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
		item := cloneTask(task)
		s.mu.Unlock()
		return item, nil
	}

	now := time.Now().UTC()
	task.Logs = append(task.Logs, taskdomain.TaskLog{
		Timestamp: now,
		Level:     taskdomain.TaskLogLevelWarn,
		Message:   "task cancel requested",
	})

	switch task.Status {
	case taskdomain.TaskStatusQueued:
		s.removeFromQueueLocked(task.ID)
		delete(s.requests, task.ID)
		task.Status = taskdomain.TaskStatusCanceled
		task.Progress = 100
		task.UpdatedAt = now
		task.FinishedAt = now
		task.ErrorCode = "task_canceled"
		task.ErrorMessage = "task canceled before execution"
		task.Result.ErrorCode = task.ErrorCode
		task.Summary = buildTaskSummary(task, nil)
		task.Logs = append(task.Logs, taskdomain.TaskLog{
			Timestamp: now,
			Level:     taskdomain.TaskLogLevelInfo,
			Message:   "task canceled before worker picked it",
		})
	case taskdomain.TaskStatusRunning:
		cancel = s.inflight[task.ID]
	default:
		item := cloneTask(task)
		s.mu.Unlock()
		return item, nil
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
	}
	return item, nil
}

func (s *Service) runWorker(ctx context.Context, workerID int) {
	for {
		taskID, msg, runCtx, cancel, ok := s.nextTask(ctx, workerID)
		if !ok {
			return
		}
		s.executeTask(taskID, msg, runCtx, workerID)
		cancel()
		s.finishRuntime(taskID)
	}
}

func (s *Service) nextTask(ctx context.Context, workerID int) (string, shareddomain.UnifiedMessage, context.Context, context.CancelFunc, bool) {
	for {
		s.mu.Lock()
		for len(s.queue) == 0 && !s.stopped && ctx.Err() == nil {
			s.cond.Wait()
		}
		if s.stopped || ctx.Err() != nil {
			s.mu.Unlock()
			return "", shareddomain.UnifiedMessage{}, nil, nil, false
		}

		taskID := s.queue[0]
		s.queue = s.queue[1:]
		task, ok := s.tasks[taskID]
		if !ok {
			s.mu.Unlock()
			continue
		}
		msg, ok := s.requests[taskID]
		if !ok {
			now := time.Now().UTC()
			task.Status = taskdomain.TaskStatusFailed
			task.Progress = 100
			task.UpdatedAt = now
			task.FinishedAt = now
			task.ErrorCode = "task_payload_missing"
			task.ErrorMessage = "task request payload is unavailable"
			task.Result.ErrorCode = task.ErrorCode
			task.Summary = buildTaskSummary(task, nil)
			task.Logs = append(task.Logs, taskdomain.TaskLog{
				Timestamp: now,
				Level:     taskdomain.TaskLogLevelError,
				Message:   "task failed before execution: payload unavailable",
			})
			s.tasks[taskID] = cloneTask(task)
			_ = s.storeLocked()
			item := cloneTask(task)
			s.mu.Unlock()
			s.appendSummaryToSession(item)
			continue
		}

		now := time.Now().UTC()
		task.Status = taskdomain.TaskStatusRunning
		if task.StartedAt.IsZero() {
			task.StartedAt = now
		}
		if task.Progress < 10 {
			task.Progress = 10
		}
		task.UpdatedAt = now
		task.Logs = append(task.Logs, taskdomain.TaskLog{
			Timestamp: now,
			Level:     taskdomain.TaskLogLevelInfo,
			Message:   fmt.Sprintf("task started by worker-%d", workerID),
		})
		timeout := s.resolveTaskTimeout(msg)
		task.TimeoutMS = timeout.Milliseconds()
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
		return taskID, cloneMessage(msg), runCtx, cancel, true
	}
}

func (s *Service) executeTask(taskID string, msg shareddomain.UnifiedMessage, runCtx context.Context, workerID int) {
	maxAttempts := s.options.MaxRetries + 1
	var lastResult shareddomain.OrchestrationResult
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
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
			return
		}

		if attempt < s.options.MaxRetries {
			s.recordRetry(taskID, attempt+1, workerID, lastErr)
			continue
		}
		task := s.completeTask(taskID, taskdomain.TaskStatusFailed, lastResult, lastErr, "task execution failed", taskdomain.TaskLogLevelError, 100)
		s.appendSummaryToSession(task)
		return
	}

	task := s.completeTask(taskID, taskdomain.TaskStatusFailed, lastResult, lastErr, "task execution failed", taskdomain.TaskLogLevelError, 100)
	s.appendSummaryToSession(task)
}

func (s *Service) recordRetry(taskID string, retryCount int, workerID int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return
	}
	now := time.Now().UTC()
	task.RetryCount = retryCount
	task.UpdatedAt = now
	nextProgress := 10 + retryCount*20
	if nextProgress > 95 {
		nextProgress = 95
	}
	if task.Progress < nextProgress {
		task.Progress = nextProgress
	}
	task.Logs = append(task.Logs, taskdomain.TaskLog{
		Timestamp: now,
		Level:     taskdomain.TaskLogLevelWarn,
		Message:   fmt.Sprintf("worker-%d retry %d/%d: %s", workerID, retryCount, task.MaxRetries, strings.TrimSpace(err.Error())),
	})
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
	task.Progress = progress
	task.UpdatedAt = now
	task.FinishedAt = now
	task.Result = toTaskResult(result)

	if status == taskdomain.TaskStatusSuccess {
		task.ErrorCode = ""
		task.ErrorMessage = ""
		task.Result.ErrorCode = ""
		if output := strings.TrimSpace(result.Output); output != "" {
			task.Artifacts = append(task.Artifacts, taskdomain.TaskArtifact{
				ArtifactID:  task.ID + "-result",
				Name:        "result.txt",
				ContentType: "text/plain",
				Content:     output,
				CreatedAt:   now,
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

	task.Logs = append(task.Logs, taskdomain.TaskLog{
		Timestamp: now,
		Level:     logLevel,
		Message:   strings.TrimSpace(logMessage),
	})
	task.Summary = buildTaskSummary(task, handleErr)
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
	}
}

func (s *Service) finishRuntime(taskID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.requests, taskID)
	delete(s.inflight, taskID)
}

func (s *Service) resolveTaskTimeout(msg shareddomain.UnifiedMessage) time.Duration {
	timeout := s.options.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	raw := strings.TrimSpace(metadataValue(msg.Metadata, metadataTaskTimeoutMSKey))
	if raw == "" {
		return timeout
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return timeout
	}
	return time.Duration(value) * time.Millisecond
}

func buildTaskSummary(task taskdomain.Task, handleErr error) string {
	switch task.Status {
	case taskdomain.TaskStatusSuccess:
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

func (s *Service) removeFromQueueLocked(taskID string) {
	for idx := range s.queue {
		if s.queue[idx] != taskID {
			continue
		}
		s.queue = append(s.queue[:idx], s.queue[idx+1:]...)
		return
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

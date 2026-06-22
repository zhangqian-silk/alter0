package application

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"

	shareddomain "alter0/internal/shared/domain"
	taskdomain "alter0/internal/task/domain"
)

const MetadataTaskTypeKey = "alter0.task.type"

var (
	ErrTaskNotFound     = errors.New("task not found")
	ErrArtifactNotFound = errors.New("artifact not found")
)

type Store interface {
	Load(ctx context.Context) ([]taskdomain.Task, error)
	Save(ctx context.Context, tasks []taskdomain.Task) error
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
	SummaryMemory TaskSummaryRecorder
}

type Service struct {
	store   Store
	summary TaskSummaryRecorder

	mu           sync.RWMutex
	tasks        map[string]taskdomain.Task
	sessionIndex map[string][]string
}

func NewService(ctx context.Context, store Store, options Options) (*Service, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	service := &Service{
		store:        store,
		summary:      options.SummaryMemory,
		tasks:        map[string]taskdomain.Task{},
		sessionIndex: map[string][]string{},
	}
	if err := service.loadStore(ctx); err != nil {
		return nil, err
	}
	return service, nil
}

func (s *Service) loadStore(ctx context.Context) error {
	if s.store == nil {
		return nil
	}
	items, err := s.store.Load(ctx)
	if err != nil {
		return err
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	for _, item := range items {
		item.ID = strings.TrimSpace(item.ID)
		item.SessionID = strings.TrimSpace(item.SessionID)
		if item.ID == "" || !item.Status.IsTerminal() {
			continue
		}
		if item.Phase == "" {
			item.Phase = string(item.Status)
		}
		cloned := cloneTask(item)
		s.tasks[cloned.ID] = cloned
		key := normalizeKey(cloned.SessionID)
		if key != "" {
			s.sessionIndex[key] = append(s.sessionIndex[key], cloned.ID)
		}
		if s.summary != nil {
			s.summary.Record(cloneTask(cloned))
		}
	}
	return nil
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
		task, ok := s.tasks[ids[idx]]
		if ok {
			items = append(items, cloneTask(task))
		}
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
		filtered = append(filtered, cloneTask(item))
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].CreatedAt.Equal(filtered[j].CreatedAt) {
			return filtered[i].ID > filtered[j].ID
		}
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
	})

	from, to := taskPageBounds(len(filtered), pagination.Page, pagination.PageSize)
	items := append([]taskdomain.Task{}, filtered[from:to]...)
	pagination.Total = len(filtered)
	pagination.HasNext = to < len(filtered)
	return TaskPage{Items: items, Pagination: pagination}
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
	return TaskLogPage{
		Items:      append([]taskdomain.TaskLog{}, logs[cursor:end]...),
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
	if len(task.Artifacts) == 0 {
		return []taskdomain.TaskArtifact{}, nil
	}
	return append([]taskdomain.TaskArtifact{}, task.Artifacts...), nil
}

func (s *Service) DeleteBySession(sessionID string) error {
	key := normalizeKey(sessionID)
	if key == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	taskIDs := append([]string(nil), s.sessionIndex[key]...)
	if len(taskIDs) == 0 {
		return nil
	}
	previousTasks := cloneTaskMap(s.tasks)
	previousSessionIndex := cloneSessionIndex(s.sessionIndex)

	for _, taskID := range taskIDs {
		delete(s.tasks, taskID)
	}
	delete(s.sessionIndex, key)

	if err := s.storeLocked(); err != nil {
		s.tasks = previousTasks
		s.sessionIndex = previousSessionIndex
		return err
	}
	return nil
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
	return s.store.Save(context.Background(), items)
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
	return Pagination{Page: page, PageSize: pageSize}
}

func taskPageBounds(total int, page int, pageSize int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	from := (page - 1) * pageSize
	if from > total {
		from = total
	}
	to := from + pageSize
	if to > total {
		to = total
	}
	return from, to
}

func normalizeKey(value string) string {
	return strings.TrimSpace(value)
}

func cloneTaskMap(items map[string]taskdomain.Task) map[string]taskdomain.Task {
	out := make(map[string]taskdomain.Task, len(items))
	for key, item := range items {
		out[key] = cloneTask(item)
	}
	return out
}

func cloneSessionIndex(index map[string][]string) map[string][]string {
	out := make(map[string][]string, len(index))
	for key, ids := range index {
		out[key] = append([]string{}, ids...)
	}
	return out
}

func cloneTask(task taskdomain.Task) taskdomain.Task {
	out := task
	if task.RequestMetadata != nil {
		out.RequestMetadata = make(map[string]string, len(task.RequestMetadata))
		for key, value := range task.RequestMetadata {
			out.RequestMetadata[key] = value
		}
	}
	out.Logs = append([]taskdomain.TaskLog{}, task.Logs...)
	out.Artifacts = append([]taskdomain.TaskArtifact{}, task.Artifacts...)
	if task.Result.Metadata != nil {
		out.Result.Metadata = make(map[string]string, len(task.Result.Metadata))
		for key, value := range task.Result.Metadata {
			out.Result.Metadata[key] = value
		}
	}
	out.Result.ProcessSteps = append([]shareddomain.ProcessStep{}, task.Result.ProcessSteps...)
	out.TaskSummary.Tags = append([]string{}, task.TaskSummary.Tags...)
	return out
}

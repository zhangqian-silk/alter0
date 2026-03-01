package httpstate

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

type TaskResult struct {
	TaskID   string
	Response string
	Closed   bool
	Decision string
}

type AsyncRecord struct {
	RequestID    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Status       string
	Content      string
	UserID       string
	TaskID       string
	Result       *TaskResult
	CanceledAt   *time.Time
	CancelReason string
	cancel       context.CancelFunc
}

type AsyncStore struct {
	mu      sync.Mutex
	records map[string]*AsyncRecord
}

func NewAsyncStore() *AsyncStore {
	return &AsyncStore{records: map[string]*AsyncRecord{}}
}

func (s *AsyncStore) Create(requestID string, createdAt time.Time, content string, userID string, taskID string, cancel context.CancelFunc) AsyncRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := &AsyncRecord{
		RequestID: requestID,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		Status:    "pending",
		Content:   content,
		UserID:    userID,
		TaskID:    taskID,
		cancel:    cancel,
	}
	s.records[requestID] = rec
	return copyAsyncRecord(rec)
}

func (s *AsyncStore) MarkRunning(requestID string, now time.Time) (AsyncRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[requestID]
	if !ok {
		return AsyncRecord{}, false
	}
	if rec.Status != "pending" {
		return copyAsyncRecord(rec), false
	}
	rec.Status = "running"
	rec.UpdatedAt = now
	return copyAsyncRecord(rec), true
}

func (s *AsyncStore) Complete(requestID string, now time.Time, result TaskResult) (AsyncRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[requestID]
	if !ok {
		return AsyncRecord{}, false
	}
	if rec.Status != "pending" && rec.Status != "running" {
		return copyAsyncRecord(rec), false
	}
	copyResult := result
	rec.Status = "completed"
	rec.Result = &copyResult
	rec.UpdatedAt = now
	rec.cancel = nil
	return copyAsyncRecord(rec), true
}

func (s *AsyncStore) Timeout(requestID string, now time.Time) (AsyncRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[requestID]
	if !ok {
		return AsyncRecord{}, false
	}
	if rec.Status != "pending" && rec.Status != "running" {
		return copyAsyncRecord(rec), false
	}
	rec.Status = "timeout"
	rec.UpdatedAt = now
	rec.cancel = nil
	return copyAsyncRecord(rec), true
}

func (s *AsyncStore) Cancel(requestID string, now time.Time, reason string) (AsyncRecord, bool, context.CancelFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[requestID]
	if !ok {
		return AsyncRecord{}, false, nil
	}
	cancel := context.CancelFunc(nil)
	if rec.Status == "pending" || rec.Status == "running" {
		rec.Status = "canceled"
		rec.UpdatedAt = now
		rec.CanceledAt = &now
		rec.CancelReason = strings.TrimSpace(reason)
		cancel = rec.cancel
		rec.cancel = nil
	}
	return copyAsyncRecord(rec), true, cancel
}

func (s *AsyncStore) Get(requestID string) (AsyncRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[requestID]
	if !ok {
		return AsyncRecord{}, false
	}
	return copyAsyncRecord(rec), true
}

func (s *AsyncStore) List(statusFilter string, limit int) []AsyncRecord {
	s.mu.Lock()
	items := make([]AsyncRecord, 0, len(s.records))
	for _, rec := range s.records {
		if statusFilter != "" && statusFilter != "all" && rec.Status != statusFilter {
			continue
		}
		items = append(items, copyAsyncRecord(rec))
	}
	s.mu.Unlock()

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

func copyAsyncRecord(src *AsyncRecord) AsyncRecord {
	if src == nil {
		return AsyncRecord{}
	}
	cp := *src
	if src.Result != nil {
		resultCopy := *src.Result
		cp.Result = &resultCopy
	}
	if src.CanceledAt != nil {
		canceledAtCopy := *src.CanceledAt
		cp.CanceledAt = &canceledAtCopy
	}
	return cp
}

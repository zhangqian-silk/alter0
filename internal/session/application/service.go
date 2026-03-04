package application

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	sessiondomain "alter0/internal/session/domain"
	shareddomain "alter0/internal/shared/domain"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 200
	previewRuneSize = 72
)

type Store interface {
	Load(ctx context.Context) ([]sessiondomain.MessageRecord, error)
	Save(ctx context.Context, records []sessiondomain.MessageRecord) error
}

type Pagination struct {
	Page     int  `json:"page"`
	PageSize int  `json:"page_size"`
	Total    int  `json:"total"`
	HasNext  bool `json:"has_next"`
}

type SessionQuery struct {
	StartAt     time.Time
	EndAt       time.Time
	TriggerType shareddomain.TriggerType
	JobID       string
	Page        int
	PageSize    int
}

type MessageQuery struct {
	SessionID string
	StartAt   time.Time
	EndAt     time.Time
	Page      int
	PageSize  int
}

type SessionPage struct {
	Items      []sessiondomain.SessionSummary `json:"items"`
	Pagination Pagination                     `json:"pagination"`
}

type MessagePage struct {
	Items      []sessiondomain.MessageRecord `json:"items"`
	Pagination Pagination                    `json:"pagination"`
}

type Service struct {
	mu        sync.RWMutex
	store     Store
	records   []sessiondomain.MessageRecord
	bySession map[string][]int
}

func NewService() *Service {
	return newService(nil)
}

func NewServiceWithStore(ctx context.Context, store Store) (*Service, error) {
	service := newService(store)
	if store == nil {
		return service, nil
	}

	records, err := store.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("load session history: %w", err)
	}

	sort.SliceStable(records, func(i, j int) bool {
		if records[i].Timestamp.Equal(records[j].Timestamp) {
			if records[i].SessionID == records[j].SessionID {
				return records[i].MessageID < records[j].MessageID
			}
			return strings.ToLower(records[i].SessionID) < strings.ToLower(records[j].SessionID)
		}
		return records[i].Timestamp.Before(records[j].Timestamp)
	})

	for _, record := range records {
		normalized, err := sanitizeRecord(record)
		if err != nil {
			return nil, fmt.Errorf("invalid session message in store: %w", err)
		}
		service.appendLocked(normalized)
	}

	return service, nil
}

func newService(store Store) *Service {
	return &Service{
		store:     store,
		records:   []sessiondomain.MessageRecord{},
		bySession: map[string][]int{},
	}
}

func (s *Service) Append(records ...sessiondomain.MessageRecord) error {
	if len(records) == 0 {
		return nil
	}

	normalized := make([]sessiondomain.MessageRecord, 0, len(records))
	for _, record := range records {
		item, err := sanitizeRecord(record)
		if err != nil {
			return err
		}
		normalized = append(normalized, item)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	previousRecords := cloneRecords(s.records)
	previousIndexes := cloneSessionIndexes(s.bySession)

	for _, record := range normalized {
		s.appendLocked(record)
	}

	if err := s.storeLocked(); err != nil {
		s.records = previousRecords
		s.bySession = previousIndexes
		return err
	}

	return nil
}

func (s *Service) ListSessions(query SessionQuery) SessionPage {
	pagination := normalizePagination(query.Page, query.PageSize)
	startAt := normalizeTime(query.StartAt)
	endAt := normalizeTime(query.EndAt)

	s.mu.RLock()
	defer s.mu.RUnlock()

	summaries := make([]sessiondomain.SessionSummary, 0, len(s.bySession))
	for _, indexes := range s.bySession {
		if len(indexes) == 0 {
			continue
		}
		start, end := s.rangeBoundsLocked(indexes, startAt, endAt)
		if end <= start {
			continue
		}
		first := s.records[indexes[start]]
		last := s.records[indexes[end-1]]
		triggerType, jobID, firedAt := s.resolveSessionSource(indexes[start:end])
		if query.TriggerType != "" && triggerType != query.TriggerType {
			continue
		}
		if strings.TrimSpace(query.JobID) != "" && normalizeKey(jobID) != normalizeKey(query.JobID) {
			continue
		}
		summaries = append(summaries, sessiondomain.SessionSummary{
			SessionID:     first.SessionID,
			MessageCount:  end - start,
			StartedAt:     first.Timestamp,
			LastMessageAt: last.Timestamp,
			LastMessageID: last.MessageID,
			LastRoute:     string(last.RouteResult.Route),
			LastErrorCode: last.RouteResult.ErrorCode,
			LastPreview:   summarizePreview(last.Content),
			TriggerType:   triggerType,
			JobID:         jobID,
			FiredAt:       firedAt,
		})
	}

	sort.SliceStable(summaries, func(i, j int) bool {
		if summaries[i].LastMessageAt.Equal(summaries[j].LastMessageAt) {
			return strings.ToLower(summaries[i].SessionID) < strings.ToLower(summaries[j].SessionID)
		}
		return summaries[i].LastMessageAt.After(summaries[j].LastMessageAt)
	})

	from, to := pageBounds(len(summaries), pagination.Page, pagination.PageSize)
	items := make([]sessiondomain.SessionSummary, 0, to-from)
	for _, item := range summaries[from:to] {
		items = append(items, item)
	}

	pagination.Total = len(summaries)
	pagination.HasNext = to < len(summaries)
	return SessionPage{
		Items:      items,
		Pagination: pagination,
	}
}

func (s *Service) ListMessages(query MessageQuery) MessagePage {
	pagination := normalizePagination(query.Page, query.PageSize)
	startAt := normalizeTime(query.StartAt)
	endAt := normalizeTime(query.EndAt)
	sessionID := strings.TrimSpace(query.SessionID)
	if sessionID == "" {
		return MessagePage{
			Items: []sessiondomain.MessageRecord{},
			Pagination: Pagination{
				Page:     pagination.Page,
				PageSize: pagination.PageSize,
			},
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	indexes := s.bySession[normalizeKey(sessionID)]
	if len(indexes) == 0 {
		return MessagePage{
			Items: []sessiondomain.MessageRecord{},
			Pagination: Pagination{
				Page:     pagination.Page,
				PageSize: pagination.PageSize,
			},
		}
	}

	start, end := s.rangeBoundsLocked(indexes, startAt, endAt)
	total := end - start
	if total < 0 {
		total = 0
	}
	from, to := pageBounds(total, pagination.Page, pagination.PageSize)
	items := make([]sessiondomain.MessageRecord, 0, to-from)
	for _, idx := range indexes[start+from : start+to] {
		items = append(items, cloneRecord(s.records[idx]))
	}

	pagination.Total = total
	pagination.HasNext = to < total
	return MessagePage{
		Items:      items,
		Pagination: pagination,
	}
}

func sanitizeRecord(record sessiondomain.MessageRecord) (sessiondomain.MessageRecord, error) {
	record.MessageID = strings.TrimSpace(record.MessageID)
	record.SessionID = strings.TrimSpace(record.SessionID)
	record.JobID = strings.TrimSpace(record.JobID)
	record.Timestamp = normalizeTime(record.Timestamp)
	record.FiredAt = normalizeTime(record.FiredAt)
	if err := record.Validate(); err != nil {
		return sessiondomain.MessageRecord{}, err
	}
	return cloneRecord(record), nil
}

func (s *Service) appendLocked(record sessiondomain.MessageRecord) {
	idx := len(s.records)
	s.records = append(s.records, cloneRecord(record))

	key := normalizeKey(record.SessionID)
	indexes := s.bySession[key]
	insertAt := sort.Search(len(indexes), func(i int) bool {
		return s.records[indexes[i]].Timestamp.After(record.Timestamp)
	})
	indexes = append(indexes, 0)
	copy(indexes[insertAt+1:], indexes[insertAt:])
	indexes[insertAt] = idx
	s.bySession[key] = indexes
}

func (s *Service) rangeBoundsLocked(indexes []int, startAt, endAt time.Time) (int, int) {
	if len(indexes) == 0 {
		return 0, 0
	}
	if !startAt.IsZero() && !endAt.IsZero() && endAt.Before(startAt) {
		return 0, 0
	}

	start := 0
	if !startAt.IsZero() {
		start = sort.Search(len(indexes), func(i int) bool {
			return !s.records[indexes[i]].Timestamp.Before(startAt)
		})
	}
	end := len(indexes)
	if !endAt.IsZero() {
		end = sort.Search(len(indexes), func(i int) bool {
			return s.records[indexes[i]].Timestamp.After(endAt)
		})
	}
	if end < start {
		return start, start
	}
	return start, end
}

func (s *Service) storeLocked() error {
	if s.store == nil {
		return nil
	}
	if err := s.store.Save(context.Background(), cloneRecords(s.records)); err != nil {
		return fmt.Errorf("store session history: %w", err)
	}
	return nil
}

func summarizePreview(content string) string {
	runes := []rune(strings.TrimSpace(content))
	if len(runes) <= previewRuneSize {
		return string(runes)
	}
	return string(runes[:previewRuneSize-1]) + "..."
}

func normalizePagination(page, pageSize int) Pagination {
	if page <= 0 {
		page = defaultPage
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return Pagination{
		Page:     page,
		PageSize: pageSize,
	}
}

func pageBounds(total, page, pageSize int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	start := (page - 1) * pageSize
	if start >= total {
		return total, total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return start, end
}

func normalizeKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeTime(ts time.Time) time.Time {
	if ts.IsZero() {
		return time.Time{}
	}
	return ts.UTC()
}

func (s *Service) resolveSessionSource(indexes []int) (shareddomain.TriggerType, string, time.Time) {
	triggerType := shareddomain.TriggerType("")
	jobID := ""
	firedAt := time.Time{}
	for _, idx := range indexes {
		record := s.records[idx]
		if triggerType == "" && record.TriggerType != "" {
			triggerType = record.TriggerType
		}
		if jobID == "" && strings.TrimSpace(record.JobID) != "" {
			jobID = strings.TrimSpace(record.JobID)
		}
		if firedAt.IsZero() && !record.FiredAt.IsZero() {
			firedAt = record.FiredAt.UTC()
		}
		if triggerType != "" && (triggerType != shareddomain.TriggerTypeCron || (jobID != "" && !firedAt.IsZero())) {
			break
		}
	}
	return triggerType, jobID, firedAt
}

func cloneRecord(record sessiondomain.MessageRecord) sessiondomain.MessageRecord {
	return sessiondomain.MessageRecord{
		MessageID:   record.MessageID,
		SessionID:   record.SessionID,
		Role:        record.Role,
		Content:     record.Content,
		Timestamp:   record.Timestamp,
		TriggerType: record.TriggerType,
		JobID:       record.JobID,
		FiredAt:     record.FiredAt,
		RouteResult: sessiondomain.RouteResult{
			Route:     record.RouteResult.Route,
			ErrorCode: record.RouteResult.ErrorCode,
			TaskID:    record.RouteResult.TaskID,
		},
	}
}

func cloneRecords(records []sessiondomain.MessageRecord) []sessiondomain.MessageRecord {
	out := make([]sessiondomain.MessageRecord, 0, len(records))
	for _, record := range records {
		out = append(out, cloneRecord(record))
	}
	return out
}

func cloneSessionIndexes(source map[string][]int) map[string][]int {
	out := make(map[string][]int, len(source))
	for key, indexes := range source {
		copied := make([]int, len(indexes))
		copy(copied, indexes)
		out[key] = copied
	}
	return out
}

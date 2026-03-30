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

var ErrSessionNotFound = fmt.Errorf("session not found")

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
	ChannelType shareddomain.ChannelType
	ChannelID   string
	MessageID   string
	AgentID     string
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
	triggerType := shareddomain.TriggerType(strings.ToLower(strings.TrimSpace(string(query.TriggerType))))
	channelType := shareddomain.ChannelType(strings.ToLower(strings.TrimSpace(string(query.ChannelType))))
	channelID := strings.TrimSpace(query.ChannelID)
	messageID := strings.TrimSpace(query.MessageID)
	agentID := strings.TrimSpace(query.AgentID)
	jobID := strings.TrimSpace(query.JobID)

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
		source := s.resolveSessionSourceLocked(indexes)
		summary := sessiondomain.SessionSummary{
			SessionID:     first.SessionID,
			MessageCount:  end - start,
			StartedAt:     first.Timestamp,
			LastMessageAt: last.Timestamp,
			LastMessageID: last.MessageID,
			LastRoute:     string(last.RouteResult.Route),
			LastErrorCode: last.RouteResult.ErrorCode,
			LastPreview:   summarizePreview(last.Content),
			TriggerType:   source.TriggerType,
			ChannelType:   source.ChannelType,
			ChannelID:     source.ChannelID,
			CorrelationID: source.CorrelationID,
			AgentID:       source.AgentID,
			AgentName:     source.AgentName,
			JobID:         source.JobID,
			JobName:       source.JobName,
			FiredAt:       source.FiredAt,
		}
		if !matchSessionSourceFilters(summary, triggerType, channelType, channelID, agentID, jobID) {
			continue
		}
		if messageID != "" && !matchSessionMessageIDFilter(s.records, indexes[start:end], messageID) {
			continue
		}
		summaries = append(summaries, summary)
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

func (s *Service) DeleteSession(sessionID string) error {
	key := normalizeKey(sessionID)
	if key == "" {
		return ErrSessionNotFound
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	indexes, ok := s.bySession[key]
	if !ok || len(indexes) == 0 {
		return ErrSessionNotFound
	}

	previousRecords := cloneRecords(s.records)
	previousIndexes := cloneSessionIndexes(s.bySession)

	filtered := make([]sessiondomain.MessageRecord, 0, len(s.records)-len(indexes))
	for _, record := range s.records {
		if normalizeKey(record.SessionID) == key {
			continue
		}
		filtered = append(filtered, cloneRecord(record))
	}

	s.records = filtered
	s.bySession = buildSessionIndexes(filtered)
	if err := s.storeLocked(); err != nil {
		s.records = previousRecords
		s.bySession = previousIndexes
		return err
	}
	return nil
}

func sanitizeRecord(record sessiondomain.MessageRecord) (sessiondomain.MessageRecord, error) {
	record.MessageID = strings.TrimSpace(record.MessageID)
	record.SessionID = strings.TrimSpace(record.SessionID)
	record.Timestamp = normalizeTime(record.Timestamp)
	record.Source = normalizeMessageSource(record.Source)
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

func (s *Service) resolveSessionSourceLocked(indexes []int) sessiondomain.MessageSource {
	for _, idx := range indexes {
		source := normalizeMessageSource(s.records[idx].Source)
		if source.TriggerType != "" ||
			source.ChannelType != "" ||
			source.ChannelID != "" ||
			source.CorrelationID != "" ||
			source.AgentID != "" ||
			source.AgentName != "" ||
			source.JobID != "" ||
			source.JobName != "" ||
			!source.FiredAt.IsZero() {
			return source
		}
	}
	return sessiondomain.MessageSource{}
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

func cloneRecord(record sessiondomain.MessageRecord) sessiondomain.MessageRecord {
	return sessiondomain.MessageRecord{
		MessageID: record.MessageID,
		SessionID: record.SessionID,
		Role:      record.Role,
		Content:   record.Content,
		Timestamp: record.Timestamp,
		RouteResult: sessiondomain.RouteResult{
			Route:     record.RouteResult.Route,
			ErrorCode: record.RouteResult.ErrorCode,
			TaskID:    record.RouteResult.TaskID,
		},
		Source: sessiondomain.MessageSource{
			TriggerType:   record.Source.TriggerType,
			ChannelType:   record.Source.ChannelType,
			ChannelID:     record.Source.ChannelID,
			CorrelationID: record.Source.CorrelationID,
			AgentID:       record.Source.AgentID,
			AgentName:     record.Source.AgentName,
			JobID:         record.Source.JobID,
			JobName:       record.Source.JobName,
			FiredAt:       record.Source.FiredAt,
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

func buildSessionIndexes(records []sessiondomain.MessageRecord) map[string][]int {
	indexes := make(map[string][]int, len(records))
	for idx, record := range records {
		key := normalizeKey(record.SessionID)
		indexes[key] = append(indexes[key], idx)
	}
	return indexes
}

func normalizeMessageSource(source sessiondomain.MessageSource) sessiondomain.MessageSource {
	source.TriggerType = shareddomain.TriggerType(strings.ToLower(strings.TrimSpace(string(source.TriggerType))))
	source.ChannelType = shareddomain.ChannelType(strings.ToLower(strings.TrimSpace(string(source.ChannelType))))
	source.ChannelID = strings.TrimSpace(source.ChannelID)
	source.CorrelationID = strings.TrimSpace(source.CorrelationID)
	source.AgentID = strings.TrimSpace(source.AgentID)
	source.AgentName = strings.TrimSpace(source.AgentName)
	source.JobID = strings.TrimSpace(source.JobID)
	source.JobName = strings.TrimSpace(source.JobName)
	source.FiredAt = normalizeTime(source.FiredAt)
	return source
}

func matchSessionSourceFilters(
	summary sessiondomain.SessionSummary,
	triggerType shareddomain.TriggerType,
	channelType shareddomain.ChannelType,
	channelID string,
	agentID string,
	jobID string,
) bool {
	if triggerType != "" && summary.TriggerType != triggerType {
		return false
	}
	if channelType != "" && summary.ChannelType != channelType {
		return false
	}
	if channelID != "" && !strings.EqualFold(summary.ChannelID, channelID) {
		return false
	}
	if agentID != "" && !strings.EqualFold(summary.AgentID, agentID) {
		return false
	}
	if jobID != "" && !strings.EqualFold(summary.JobID, jobID) {
		return false
	}
	return true
}

func matchSessionMessageIDFilter(records []sessiondomain.MessageRecord, indexes []int, messageID string) bool {
	for _, idx := range indexes {
		record := records[idx]
		if strings.EqualFold(strings.TrimSpace(record.MessageID), messageID) {
			return true
		}
	}
	return false
}

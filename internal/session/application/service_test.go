package application

import (
	"context"
	"errors"
	"testing"
	"time"

	sessiondomain "alter0/internal/session/domain"
	shareddomain "alter0/internal/shared/domain"
)

type stubStore struct {
	loadRecords []sessiondomain.MessageRecord
	saveErr     error
	saveCalls   int
}

func (s *stubStore) Load(_ context.Context) ([]sessiondomain.MessageRecord, error) {
	out := make([]sessiondomain.MessageRecord, 0, len(s.loadRecords))
	for _, record := range s.loadRecords {
		out = append(out, record)
	}
	return out, nil
}

func (s *stubStore) Save(_ context.Context, _ []sessiondomain.MessageRecord) error {
	s.saveCalls++
	return s.saveErr
}

func TestServiceAppendAndListMessagesByTimeRange(t *testing.T) {
	service := NewService()
	base := time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC)

	if err := service.Append(
		newRecord("m-1", "s-1", sessiondomain.MessageRoleUser, "hello", base, shareddomain.RouteNL, ""),
		newRecord("m-2", "s-1", sessiondomain.MessageRoleAssistant, "hi", base.Add(1*time.Minute), shareddomain.RouteNL, ""),
		newRecord("m-3", "s-1", sessiondomain.MessageRoleAssistant, "done", base.Add(2*time.Minute), shareddomain.RouteCommand, "command_failed"),
		newRecord("m-4", "s-2", sessiondomain.MessageRoleUser, "other", base.Add(3*time.Minute), shareddomain.RouteNL, ""),
	); err != nil {
		t.Fatalf("append failed: %v", err)
	}

	page := service.ListMessages(MessageQuery{
		SessionID: "s-1",
		StartAt:   base.Add(1 * time.Minute),
		EndAt:     base.Add(2 * time.Minute),
		Page:      1,
		PageSize:  10,
	})
	if page.Pagination.Total != 2 {
		t.Fatalf("expected total 2, got %d", page.Pagination.Total)
	}
	if len(page.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(page.Items))
	}
	if page.Items[0].MessageID != "m-2" || page.Items[1].MessageID != "m-3" {
		t.Fatalf("unexpected message ids: %+v", page.Items)
	}
	if page.Items[1].RouteResult.ErrorCode != "command_failed" {
		t.Fatalf("expected command_failed, got %q", page.Items[1].RouteResult.ErrorCode)
	}
}

func TestServiceListSessionsPagination(t *testing.T) {
	service := NewService()
	base := time.Date(2026, 3, 3, 8, 0, 0, 0, time.UTC)

	if err := service.Append(
		newRecord("m-1", "s-1", sessiondomain.MessageRoleUser, "a", base, shareddomain.RouteNL, ""),
		newRecord("m-2", "s-2", sessiondomain.MessageRoleUser, "b", base.Add(1*time.Minute), shareddomain.RouteNL, ""),
		newRecord("m-3", "s-3", sessiondomain.MessageRoleUser, "c", base.Add(2*time.Minute), shareddomain.RouteNL, ""),
	); err != nil {
		t.Fatalf("append failed: %v", err)
	}

	first := service.ListSessions(SessionQuery{Page: 1, PageSize: 2})
	if len(first.Items) != 2 {
		t.Fatalf("expected 2 items on first page, got %d", len(first.Items))
	}
	if first.Pagination.Total != 3 {
		t.Fatalf("expected total 3, got %d", first.Pagination.Total)
	}
	if !first.Pagination.HasNext {
		t.Fatalf("expected has_next=true on first page")
	}
	if first.Items[0].SessionID != "s-3" || first.Items[1].SessionID != "s-2" {
		t.Fatalf("unexpected first page order: %+v", first.Items)
	}

	second := service.ListSessions(SessionQuery{Page: 2, PageSize: 2})
	if len(second.Items) != 1 {
		t.Fatalf("expected 1 item on second page, got %d", len(second.Items))
	}
	if second.Items[0].SessionID != "s-1" {
		t.Fatalf("unexpected second page item: %+v", second.Items[0])
	}
}

func TestServiceListSessionsSupportsSourceFilters(t *testing.T) {
	service := NewService()
	base := time.Date(2026, 3, 5, 8, 0, 0, 0, time.UTC)

	if err := service.Append(
		newRecord(
			"m-1",
			"s-cron-a",
			sessiondomain.MessageRoleUser,
			"cron task",
			base,
			shareddomain.RouteCommand,
			"",
			sessiondomain.MessageSource{
				TriggerType: shareddomain.TriggerTypeCron,
				ChannelType: shareddomain.ChannelTypeScheduler,
				ChannelID:   "scheduler-default",
				JobID:       "job-a",
			},
		),
		newRecord(
			"m-2",
			"s-user",
			sessiondomain.MessageRoleUser,
			"user task",
			base.Add(1*time.Minute),
			shareddomain.RouteNL,
			"",
			sessiondomain.MessageSource{
				TriggerType: shareddomain.TriggerTypeUser,
				ChannelType: shareddomain.ChannelTypeWeb,
				ChannelID:   "web-default",
				AgentID:     "researcher",
				AgentName:   "Researcher",
			},
		),
		newRecord(
			"m-3",
			"s-user",
			sessiondomain.MessageRoleAssistant,
			"done",
			base.Add(2*time.Minute),
			shareddomain.RouteNL,
			"",
			sessiondomain.MessageSource{
				TriggerType: shareddomain.TriggerTypeUser,
				ChannelType: shareddomain.ChannelTypeWeb,
				ChannelID:   "web-default",
				AgentID:     "researcher",
				AgentName:   "Researcher",
			},
		),
	); err != nil {
		t.Fatalf("append failed: %v", err)
	}

	cronOnly := service.ListSessions(SessionQuery{
		TriggerType: shareddomain.TriggerTypeCron,
		Page:        1,
		PageSize:    10,
	})
	if len(cronOnly.Items) != 1 {
		t.Fatalf("expected 1 cron session, got %d", len(cronOnly.Items))
	}
	if cronOnly.Items[0].SessionID != "s-cron-a" {
		t.Fatalf("expected cron session s-cron-a, got %s", cronOnly.Items[0].SessionID)
	}

	cronByJob := service.ListSessions(SessionQuery{
		TriggerType: shareddomain.TriggerTypeCron,
		JobID:       "job-a",
		Page:        1,
		PageSize:    10,
	})
	if len(cronByJob.Items) != 1 || cronByJob.Items[0].JobID != "job-a" {
		t.Fatalf("expected cron job job-a, got %+v", cronByJob.Items)
	}

	userByChannel := service.ListSessions(SessionQuery{
		ChannelType: shareddomain.ChannelTypeWeb,
		ChannelID:   "web-default",
		Page:        1,
		PageSize:    10,
	})
	if len(userByChannel.Items) != 1 || userByChannel.Items[0].SessionID != "s-user" {
		t.Fatalf("expected web session s-user, got %+v", userByChannel.Items)
	}

	userByMessage := service.ListSessions(SessionQuery{
		MessageID: "m-3",
		Page:      1,
		PageSize:  10,
	})
	if len(userByMessage.Items) != 1 || userByMessage.Items[0].SessionID != "s-user" {
		t.Fatalf("expected message filtered session s-user, got %+v", userByMessage.Items)
	}
	if userByMessage.Items[0].AgentID != "researcher" || userByMessage.Items[0].AgentName != "Researcher" {
		t.Fatalf("expected agent metadata on session summary, got %+v", userByMessage.Items[0])
	}

	agentOnly := service.ListSessions(SessionQuery{
		AgentID:  "researcher",
		Page:     1,
		PageSize: 10,
	})
	if len(agentOnly.Items) != 1 || agentOnly.Items[0].SessionID != "s-user" {
		t.Fatalf("expected agent filtered session s-user, got %+v", agentOnly.Items)
	}

	noMatch := service.ListSessions(SessionQuery{
		TriggerType: shareddomain.TriggerTypeCron,
		JobID:       "job-b",
		Page:        1,
		PageSize:    10,
	})
	if len(noMatch.Items) != 0 {
		t.Fatalf("expected no sessions, got %d", len(noMatch.Items))
	}
}

func TestServiceLoadsFromStoreAndBuildsIndex(t *testing.T) {
	base := time.Date(2026, 3, 3, 6, 0, 0, 0, time.UTC)
	store := &stubStore{
		loadRecords: []sessiondomain.MessageRecord{
			newRecord("m-2", "s-1", sessiondomain.MessageRoleAssistant, "second", base.Add(2*time.Minute), shareddomain.RouteNL, ""),
			newRecord("m-1", "s-1", sessiondomain.MessageRoleUser, "first", base.Add(1*time.Minute), shareddomain.RouteNL, ""),
		},
	}

	service, err := NewServiceWithStore(context.Background(), store)
	if err != nil {
		t.Fatalf("new service with store failed: %v", err)
	}

	page := service.ListMessages(MessageQuery{SessionID: "s-1", Page: 1, PageSize: 10})
	if len(page.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(page.Items))
	}
	if page.Items[0].MessageID != "m-1" || page.Items[1].MessageID != "m-2" {
		t.Fatalf("expected sorted by timestamp, got %+v", page.Items)
	}
}

func TestServiceAppendRollbackWhenStoreFails(t *testing.T) {
	store := &stubStore{saveErr: errors.New("disk full")}
	service, err := NewServiceWithStore(context.Background(), store)
	if err != nil {
		t.Fatalf("new service with store failed: %v", err)
	}

	base := time.Date(2026, 3, 3, 9, 0, 0, 0, time.UTC)
	err = service.Append(newRecord("m-1", "s-1", sessiondomain.MessageRoleUser, "hello", base, shareddomain.RouteNL, ""))
	if err == nil {
		t.Fatal("expected append error")
	}

	page := service.ListMessages(MessageQuery{SessionID: "s-1", Page: 1, PageSize: 10})
	if page.Pagination.Total != 0 {
		t.Fatalf("expected rollback on store failure, got total %d", page.Pagination.Total)
	}
	if store.saveCalls != 1 {
		t.Fatalf("expected 1 save call, got %d", store.saveCalls)
	}
}

func TestServiceDeleteSessionRemovesRecordsAndIndex(t *testing.T) {
	store := &stubStore{}
	service, err := NewServiceWithStore(context.Background(), store)
	if err != nil {
		t.Fatalf("new service with store failed: %v", err)
	}

	base := time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC)
	if err := service.Append(
		newRecord("m-1", "s-1", sessiondomain.MessageRoleUser, "hello", base, shareddomain.RouteNL, ""),
		newRecord("m-2", "s-1", sessiondomain.MessageRoleAssistant, "world", base.Add(time.Minute), shareddomain.RouteNL, ""),
		newRecord("m-3", "s-2", sessiondomain.MessageRoleUser, "other", base.Add(2*time.Minute), shareddomain.RouteNL, ""),
	); err != nil {
		t.Fatalf("append failed: %v", err)
	}

	if err := service.DeleteSession("s-1"); err != nil {
		t.Fatalf("delete session failed: %v", err)
	}

	page := service.ListMessages(MessageQuery{SessionID: "s-1", Page: 1, PageSize: 10})
	if page.Pagination.Total != 0 {
		t.Fatalf("expected deleted session to have no messages, got %d", page.Pagination.Total)
	}

	sessions := service.ListSessions(SessionQuery{Page: 1, PageSize: 10})
	if len(sessions.Items) != 1 || sessions.Items[0].SessionID != "s-2" {
		t.Fatalf("expected only s-2 to remain, got %+v", sessions.Items)
	}
}

func TestServiceDeleteSessionRollsBackOnStoreFailure(t *testing.T) {
	store := &stubStore{}
	service, err := NewServiceWithStore(context.Background(), store)
	if err != nil {
		t.Fatalf("new service with store failed: %v", err)
	}

	base := time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC)
	if err := service.Append(newRecord("m-1", "s-1", sessiondomain.MessageRoleUser, "hello", base, shareddomain.RouteNL, "")); err != nil {
		t.Fatalf("append failed: %v", err)
	}

	store.saveErr = errors.New("disk full")
	if err := service.DeleteSession("s-1"); err == nil {
		t.Fatal("expected delete error")
	}

	page := service.ListMessages(MessageQuery{SessionID: "s-1", Page: 1, PageSize: 10})
	if page.Pagination.Total != 1 {
		t.Fatalf("expected records restored after rollback, got %d", page.Pagination.Total)
	}
}

func newRecord(
	messageID string,
	sessionID string,
	role sessiondomain.MessageRole,
	content string,
	ts time.Time,
	route shareddomain.Route,
	errorCode string,
	source ...sessiondomain.MessageSource,
) sessiondomain.MessageRecord {
	recordSource := sessiondomain.MessageSource{}
	if len(source) > 0 {
		recordSource = source[0]
	}
	return sessiondomain.MessageRecord{
		MessageID: messageID,
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		Timestamp: ts,
		RouteResult: sessiondomain.RouteResult{
			Route:     route,
			ErrorCode: errorCode,
		},
		Source: recordSource,
	}
}

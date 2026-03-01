package schedule

import (
	"context"
	"database/sql"
	"fmt"

	orcschedule "alter0/app/core/orchestrator/schedule"
)

type (
	Job             = orcschedule.Job
	RunRecord       = orcschedule.RunRecord
	CreateRequest   = orcschedule.CreateRequest
	DeliveryPayload = orcschedule.DeliveryPayload
	Store           = orcschedule.Store
	Dispatcher      = orcschedule.Dispatcher
)

const (
	KindAt   = orcschedule.KindAt
	KindCron = orcschedule.KindCron

	StatusActive    = orcschedule.StatusActive
	StatusPaused    = orcschedule.StatusPaused
	StatusCanceled  = orcschedule.StatusCanceled
	StatusCompleted = orcschedule.StatusCompleted
	StatusFailed    = orcschedule.StatusFailed

	ModeDirect    = orcschedule.ModeDirect
	ModeAgentTurn = orcschedule.ModeAgentTurn
)

type Service struct {
	inner *orcschedule.Service
}

func NewStore(conn *sql.DB) (*Store, error) {
	return orcschedule.NewStore(conn)
}

func NewService(store *Store, dispatcher Dispatcher) *Service {
	return &Service{inner: orcschedule.NewService(store, dispatcher)}
}

func (s *Service) SetBatchSize(batchSize int) {
	if s == nil || s.inner == nil {
		return
	}
	s.inner.SetBatchSize(batchSize)
}

func (s *Service) DispatchDue(ctx context.Context) {
	if s == nil || s.inner == nil {
		return
	}
	s.inner.DispatchDue(ctx)
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Job, error) {
	if s == nil || s.inner == nil {
		return Job{}, fmt.Errorf("schedule service is not initialized")
	}
	return s.inner.Create(ctx, req)
}

func (s *Service) List(ctx context.Context, limit int) ([]Job, error) {
	if s == nil || s.inner == nil {
		return nil, fmt.Errorf("schedule service is not initialized")
	}
	return s.inner.List(ctx, limit)
}

func (s *Service) Get(ctx context.Context, id string) (Job, error) {
	if s == nil || s.inner == nil {
		return Job{}, fmt.Errorf("schedule service is not initialized")
	}
	return s.inner.Get(ctx, id)
}

func (s *Service) Pause(ctx context.Context, id string) error {
	if s == nil || s.inner == nil {
		return fmt.Errorf("schedule service is not initialized")
	}
	return s.inner.Pause(ctx, id)
}

func (s *Service) Resume(ctx context.Context, id string) error {
	if s == nil || s.inner == nil {
		return fmt.Errorf("schedule service is not initialized")
	}
	return s.inner.Resume(ctx, id)
}

func (s *Service) Cancel(ctx context.Context, id string) error {
	if s == nil || s.inner == nil {
		return fmt.Errorf("schedule service is not initialized")
	}
	return s.inner.Cancel(ctx, id)
}

func (s *Service) Runs(ctx context.Context, id string, limit int) ([]RunRecord, error) {
	if s == nil || s.inner == nil {
		return nil, fmt.Errorf("schedule service is not initialized")
	}
	return s.inner.Runs(ctx, id, limit)
}

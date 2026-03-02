package localfile

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	schedulerapp "alter0/internal/scheduler/application"
	schedulerdomain "alter0/internal/scheduler/domain"
)

type persistedJob struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Interval  string            `json:"interval"`
	Enabled   bool              `json:"enabled"`
	SessionID string            `json:"session_id"`
	UserID    string            `json:"user_id,omitempty"`
	ChannelID string            `json:"channel_id,omitempty"`
	Content   string            `json:"content"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type schedulerState struct {
	Jobs []persistedJob `json:"jobs"`
}

type SchedulerStore struct {
	path   string
	format Format
	mu     sync.Mutex
}

func NewSchedulerStore(baseDir string, format Format) *SchedulerStore {
	return &SchedulerStore{
		path:   filepath.Join(baseDir, "scheduler."+extension(format)),
		format: format,
	}
}

var _ schedulerapp.Persistence = (*SchedulerStore)(nil)

func (s *SchedulerStore) Load(_ context.Context) ([]schedulerdomain.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	raw, ok, err := readIfExists(s.path)
	if err != nil {
		return nil, err
	}
	if !ok {
		return []schedulerdomain.Job{}, nil
	}

	state := schedulerState{}
	if err := unmarshalPayload(s.format, raw, &state); err != nil {
		return nil, err
	}

	jobs := make([]schedulerdomain.Job, 0, len(state.Jobs))
	for _, item := range state.Jobs {
		interval, err := time.ParseDuration(item.Interval)
		if err != nil {
			return nil, fmt.Errorf("invalid interval for job %q: %w", item.ID, err)
		}
		jobs = append(jobs, schedulerdomain.Job{
			ID:        item.ID,
			Name:      item.Name,
			Interval:  interval,
			Enabled:   item.Enabled,
			SessionID: item.SessionID,
			UserID:    item.UserID,
			ChannelID: item.ChannelID,
			Content:   item.Content,
			Metadata:  item.Metadata,
		})
	}
	return jobs, nil
}

func (s *SchedulerStore) Save(_ context.Context, jobs []schedulerdomain.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]persistedJob, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, persistedJob{
			ID:        job.ID,
			Name:      job.Name,
			Interval:  job.Interval.String(),
			Enabled:   job.Enabled,
			SessionID: job.SessionID,
			UserID:    job.UserID,
			ChannelID: job.ChannelID,
			Content:   job.Content,
			Metadata:  job.Metadata,
		})
	}

	raw, err := marshalPayload(s.format, "alter0 scheduler state", schedulerState{Jobs: items})
	if err != nil {
		return err
	}
	return writeFile(s.path, raw)
}

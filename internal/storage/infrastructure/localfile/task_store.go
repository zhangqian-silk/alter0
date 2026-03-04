package localfile

import (
	"context"
	"path/filepath"
	"sync"

	taskapp "alter0/internal/task/application"
	taskdomain "alter0/internal/task/domain"
)

type taskState struct {
	Tasks []taskdomain.Task `json:"tasks"`
}

type TaskStore struct {
	path   string
	format Format
	mu     sync.Mutex
}

func NewTaskStore(baseDir string, format Format) *TaskStore {
	return &TaskStore{
		path:   filepath.Join(baseDir, "tasks."+extension(format)),
		format: format,
	}
}

var _ taskapp.Store = (*TaskStore)(nil)

func (s *TaskStore) Load(_ context.Context) ([]taskdomain.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	raw, ok, err := readIfExists(s.path)
	if err != nil {
		return nil, err
	}
	if !ok {
		return []taskdomain.Task{}, nil
	}

	state := taskState{}
	if err := unmarshalPayload(s.format, raw, &state); err != nil {
		return nil, err
	}
	if len(state.Tasks) == 0 {
		return []taskdomain.Task{}, nil
	}

	items := make([]taskdomain.Task, 0, len(state.Tasks))
	for _, item := range state.Tasks {
		items = append(items, cloneTask(item))
	}
	return items, nil
}

func (s *TaskStore) Save(_ context.Context, tasks []taskdomain.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]taskdomain.Task, 0, len(tasks))
	for _, item := range tasks {
		items = append(items, cloneTask(item))
	}

	raw, err := marshalPayload(s.format, "alter0 async task state", taskState{Tasks: items})
	if err != nil {
		return err
	}
	return writeFile(s.path, raw)
}

func cloneTask(task taskdomain.Task) taskdomain.Task {
	out := task
	if len(task.RequestMetadata) > 0 {
		out.RequestMetadata = make(map[string]string, len(task.RequestMetadata))
		for key, value := range task.RequestMetadata {
			out.RequestMetadata[key] = value
		}
	} else {
		out.RequestMetadata = map[string]string{}
	}
	if len(task.Result.Metadata) > 0 {
		out.Result.Metadata = make(map[string]string, len(task.Result.Metadata))
		for key, value := range task.Result.Metadata {
			out.Result.Metadata[key] = value
		}
	} else {
		out.Result.Metadata = map[string]string{}
	}
	if len(task.Logs) == 0 {
		out.Logs = []taskdomain.TaskLog{}
	} else {
		out.Logs = append([]taskdomain.TaskLog{}, task.Logs...)
	}
	if len(task.Artifacts) == 0 {
		out.Artifacts = []taskdomain.TaskArtifact{}
	} else {
		out.Artifacts = append([]taskdomain.TaskArtifact{}, task.Artifacts...)
	}
	return out
}

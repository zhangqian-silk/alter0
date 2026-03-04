package localfile

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	taskapp "alter0/internal/task/application"
	taskdomain "alter0/internal/task/domain"
)

const defaultTaskLogRetention = 90 * 24 * time.Hour

type taskState struct {
	Tasks []taskdomain.Task `json:"tasks"`
}

type taskIndexState struct {
	Items []taskIndexItem `json:"items"`
}

type taskIndexItem struct {
	TaskID          string                `json:"task_id"`
	SessionID       string                `json:"session_id"`
	SourceMessageID string                `json:"source_message_id"`
	TaskType        string                `json:"task_type,omitempty"`
	Status          taskdomain.TaskStatus `json:"status"`
	CreatedAt       time.Time             `json:"created_at"`
	FinishedAt      time.Time             `json:"finished_at,omitempty"`
}

type taskArtifactState struct {
	Items []taskdomain.TaskArtifact `json:"items"`
}

type TaskStoreOptions struct {
	LogRetention time.Duration
}

type TaskStore struct {
	tasksDir     string
	indexPath    string
	legacyPath   string
	format       Format
	logRetention time.Duration
	mu           sync.Mutex
}

func NewTaskStore(baseDir string, format Format) *TaskStore {
	return NewTaskStoreWithOptions(baseDir, format, TaskStoreOptions{})
}

func NewTaskStoreWithOptions(baseDir string, format Format, options TaskStoreOptions) *TaskStore {
	tasksDir := filepath.Join(baseDir, "tasks")
	retention := options.LogRetention
	if retention <= 0 {
		retention = defaultTaskLogRetention
	}
	return &TaskStore{
		tasksDir:     tasksDir,
		indexPath:    filepath.Join(tasksDir, "index.json"),
		legacyPath:   filepath.Join(baseDir, "tasks."+extension(format)),
		format:       format,
		logRetention: retention,
	}
}

var _ taskapp.Store = (*TaskStore)(nil)

func (s *TaskStore) Load(_ context.Context) ([]taskdomain.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if indexExists, err := fileExists(s.indexPath); err != nil {
		return nil, err
	} else if indexExists {
		return s.loadFromTaskLayout()
	}

	items, found, err := s.loadLegacyTaskState()
	if err != nil {
		return nil, err
	}
	if !found {
		return []taskdomain.Task{}, nil
	}
	if err := s.saveTaskLayout(items); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *TaskStore) Save(_ context.Context, tasks []taskdomain.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveTaskLayout(tasks)
}

func (s *TaskStore) loadFromTaskLayout() ([]taskdomain.Task, error) {
	raw, _, err := readIfExists(s.indexPath)
	if err != nil {
		return nil, err
	}
	index := taskIndexState{}
	if err := json.Unmarshal(raw, &index); err != nil {
		return nil, fmt.Errorf("decode task index: %w", err)
	}
	if len(index.Items) == 0 {
		return []taskdomain.Task{}, nil
	}

	items := make([]taskdomain.Task, 0, len(index.Items))
	for _, entry := range index.Items {
		taskID := strings.TrimSpace(entry.TaskID)
		if taskID == "" {
			continue
		}
		item, err := s.loadTaskEntry(entry)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	return items, nil
}

func (s *TaskStore) loadTaskEntry(index taskIndexItem) (taskdomain.Task, error) {
	taskDir := filepath.Join(s.tasksDir, index.TaskID)
	metaPath := filepath.Join(taskDir, "meta.json")
	metaRaw, _, err := readIfExists(metaPath)
	if err != nil {
		return taskdomain.Task{}, err
	}
	item := taskdomain.Task{}
	if err := json.Unmarshal(metaRaw, &item); err != nil {
		return taskdomain.Task{}, fmt.Errorf("decode task meta %s: %w", index.TaskID, err)
	}

	logs, err := readTaskLogs(filepath.Join(taskDir, "logs.jsonl"))
	if err != nil {
		return taskdomain.Task{}, err
	}
	item.Logs = applyTaskLogRetention(logs, s.logRetention, time.Now().UTC())

	artifacts, err := readTaskArtifacts(filepath.Join(taskDir, "artifacts.json"))
	if err != nil {
		return taskdomain.Task{}, err
	}
	item.Artifacts = artifacts
	item = enrichTaskWithIndex(item, index)
	return cloneTask(item), nil
}

func (s *TaskStore) loadLegacyTaskState() ([]taskdomain.Task, bool, error) {
	raw, ok, err := readIfExists(s.legacyPath)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return []taskdomain.Task{}, false, nil
	}
	state := taskState{}
	if err := unmarshalPayload(s.format, raw, &state); err != nil {
		return nil, false, err
	}
	if len(state.Tasks) == 0 {
		return []taskdomain.Task{}, true, nil
	}
	items := make([]taskdomain.Task, 0, len(state.Tasks))
	for _, item := range state.Tasks {
		item.Logs = applyTaskLogRetention(item.Logs, s.logRetention, time.Now().UTC())
		items = append(items, cloneTask(item))
	}
	return items, true, nil
}

func (s *TaskStore) saveTaskLayout(tasks []taskdomain.Task) error {
	items := make([]taskdomain.Task, 0, len(tasks))
	for _, item := range tasks {
		copyItem := cloneTask(item)
		copyItem.Logs = applyTaskLogRetention(copyItem.Logs, s.logRetention, time.Now().UTC())
		items = append(items, copyItem)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})

	if err := os.MkdirAll(s.tasksDir, 0o755); err != nil {
		return err
	}

	index := taskIndexState{Items: make([]taskIndexItem, 0, len(items))}
	for _, item := range items {
		if strings.TrimSpace(item.ID) == "" {
			continue
		}
		taskDir := filepath.Join(s.tasksDir, item.ID)
		if err := os.MkdirAll(taskDir, 0o755); err != nil {
			return err
		}

		if err := writeTaskMeta(filepath.Join(taskDir, "meta.json"), item); err != nil {
			return err
		}
		if err := appendTaskLogs(filepath.Join(taskDir, "logs.jsonl"), item.Logs); err != nil {
			return err
		}
		if err := writeTaskArtifacts(filepath.Join(taskDir, "artifacts.json"), item.Artifacts); err != nil {
			return err
		}
		index.Items = append(index.Items, taskIndexItem{
			TaskID:          item.ID,
			SessionID:       item.SessionID,
			SourceMessageID: item.SourceMessageID,
			TaskType:        strings.TrimSpace(item.TaskType),
			Status:          item.Status,
			CreatedAt:       item.CreatedAt,
			FinishedAt:      item.FinishedAt,
		})
	}

	raw, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	if err := writeFile(s.indexPath, append(raw, '\n')); err != nil {
		return err
	}
	_ = os.Remove(s.legacyPath)
	return nil
}

func writeTaskMeta(path string, task taskdomain.Task) error {
	item := cloneTask(task)
	item.Logs = []taskdomain.TaskLog{}
	item.Artifacts = []taskdomain.TaskArtifact{}
	raw, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return err
	}
	return writeFile(path, append(raw, '\n'))
}

func appendTaskLogs(path string, logs []taskdomain.TaskLog) error {
	if len(logs) == 0 {
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			return writeFile(path, []byte{})
		}
		return nil
	}

	existingSeq := map[int]struct{}{}
	raw, ok, err := readIfExists(path)
	if err != nil {
		return err
	}
	if ok {
		scanner := bufio.NewScanner(bytes.NewReader(raw))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			entry := taskdomain.TaskLog{}
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				continue
			}
			if entry.Seq > 0 {
				existingSeq[entry.Seq] = struct{}{}
			}
		}
		if err := scanner.Err(); err != nil {
			return err
		}
	}

	newLogs := make([]taskdomain.TaskLog, 0, len(logs))
	for _, item := range logs {
		if item.Seq > 0 {
			if _, exists := existingSeq[item.Seq]; exists {
				continue
			}
			existingSeq[item.Seq] = struct{}{}
		}
		newLogs = append(newLogs, item)
	}
	if len(newLogs) == 0 {
		if !ok {
			return writeFile(path, []byte{})
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	for _, item := range newLogs {
		line, err := json.Marshal(item)
		if err != nil {
			return err
		}
		if _, err := file.Write(append(line, '\n')); err != nil {
			return err
		}
	}
	return nil
}

func writeTaskArtifacts(path string, items []taskdomain.TaskArtifact) error {
	state := taskArtifactState{Items: []taskdomain.TaskArtifact{}}
	if len(items) > 0 {
		state.Items = append(state.Items, items...)
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return writeFile(path, append(raw, '\n'))
}

func readTaskLogs(path string) ([]taskdomain.TaskLog, error) {
	raw, ok, err := readIfExists(path)
	if err != nil {
		return nil, err
	}
	if !ok || len(bytes.TrimSpace(raw)) == 0 {
		return []taskdomain.TaskLog{}, nil
	}

	logs := []taskdomain.TaskLog{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		entry := taskdomain.TaskLog{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		logs = append(logs, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return logs, nil
}

func readTaskArtifacts(path string) ([]taskdomain.TaskArtifact, error) {
	raw, ok, err := readIfExists(path)
	if err != nil {
		return nil, err
	}
	if !ok {
		return []taskdomain.TaskArtifact{}, nil
	}
	state := taskArtifactState{}
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, fmt.Errorf("decode task artifacts: %w", err)
	}
	if len(state.Items) == 0 {
		return []taskdomain.TaskArtifact{}, nil
	}
	items := make([]taskdomain.TaskArtifact, 0, len(state.Items))
	items = append(items, state.Items...)
	return items, nil
}

func enrichTaskWithIndex(task taskdomain.Task, index taskIndexItem) taskdomain.Task {
	if strings.TrimSpace(task.ID) == "" {
		task.ID = strings.TrimSpace(index.TaskID)
	}
	if strings.TrimSpace(task.SessionID) == "" {
		task.SessionID = strings.TrimSpace(index.SessionID)
	}
	if strings.TrimSpace(task.SourceMessageID) == "" {
		task.SourceMessageID = strings.TrimSpace(index.SourceMessageID)
	}
	if strings.TrimSpace(task.TaskType) == "" {
		task.TaskType = strings.TrimSpace(index.TaskType)
	}
	if !task.Status.IsValid() && index.Status.IsValid() {
		task.Status = index.Status
	}
	if task.CreatedAt.IsZero() && !index.CreatedAt.IsZero() {
		task.CreatedAt = index.CreatedAt
	}
	if task.FinishedAt.IsZero() && !index.FinishedAt.IsZero() {
		task.FinishedAt = index.FinishedAt
	}
	return task
}

func applyTaskLogRetention(logs []taskdomain.TaskLog, retention time.Duration, now time.Time) []taskdomain.TaskLog {
	if len(logs) == 0 {
		return []taskdomain.TaskLog{}
	}
	if retention <= 0 {
		out := make([]taskdomain.TaskLog, 0, len(logs))
		out = append(out, logs...)
		return out
	}
	cutoff := now.Add(-retention)
	kept := make([]taskdomain.TaskLog, 0, len(logs))
	for _, item := range logs {
		ts := item.CreatedAt
		if ts.IsZero() {
			ts = item.Timestamp
		}
		if !ts.IsZero() && ts.Before(cutoff) {
			continue
		}
		kept = append(kept, item)
	}
	return kept
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
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

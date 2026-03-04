package application

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	taskdomain "alter0/internal/task/domain"
)

const (
	defaultRuntimeTaskDailyDir    = ".alter0/memory"
	defaultRuntimeTaskLongTermDir = ".alter0/memory/long-term"
)

type RuntimeMarkdownOptions struct {
	DailyDir    string
	LongTermDir string
}

type SummaryReference struct {
	TaskID string `json:"task_id"`
	Path   string `json:"path"`
	Date   string `json:"date,omitempty"`
	Tier   string `json:"tier"`
}

type RuntimeMarkdownStore struct {
	dailyDir    string
	longTermDir string
	mu          sync.Mutex
}

func NewRuntimeMarkdownStore(options RuntimeMarkdownOptions) *RuntimeMarkdownStore {
	dailyDir := strings.TrimSpace(options.DailyDir)
	if dailyDir == "" {
		dailyDir = defaultRuntimeTaskDailyDir
	}
	longTermDir := strings.TrimSpace(options.LongTermDir)
	if longTermDir == "" {
		longTermDir = defaultRuntimeTaskLongTermDir
	}
	return &RuntimeMarkdownStore{
		dailyDir:    filepath.Clean(dailyDir),
		longTermDir: filepath.Clean(longTermDir),
	}
}

func (s *RuntimeMarkdownStore) Record(task taskdomain.Task) {
	_, _ = s.Rebuild(task)
}

func (s *RuntimeMarkdownStore) Rebuild(task taskdomain.Task) ([]SummaryReference, error) {
	if !task.Status.IsTerminal() {
		return []SummaryReference{}, nil
	}
	summary := normalizeSummary(task)
	if summary.IsZero() {
		return []SummaryReference{}, nil
	}
	finishedAt := summary.FinishedAt
	if finishedAt.IsZero() {
		finishedAt = time.Now().UTC()
	}
	day := finishedAt.UTC().Format("2006-01-02")
	entry := buildTaskSummaryMarkdownEntry(task, summary)

	s.mu.Lock()
	defer s.mu.Unlock()

	dailyPath := filepath.Join(s.dailyDir, day+".md")
	if err := upsertTaskSummaryEntry(dailyPath, day, summary.TaskID, entry); err != nil {
		return nil, err
	}
	longTermPath := filepath.Join(s.longTermDir, day+".md")
	if err := upsertTaskSummaryEntry(longTermPath, day, summary.TaskID, entry); err != nil {
		return nil, err
	}
	return s.findSummaryRefsLocked(summary.TaskID), nil
}

func (s *RuntimeMarkdownStore) FindSummaryRefs(taskID string) []SummaryReference {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return []SummaryReference{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.findSummaryRefsLocked(taskID)
}

func (s *RuntimeMarkdownStore) findSummaryRefsLocked(taskID string) []SummaryReference {
	refs := make([]SummaryReference, 0, 4)
	refs = append(refs, scanTaskSummaryRefs(s.dailyDir, "daily", taskID)...)
	refs = append(refs, scanTaskSummaryRefs(s.longTermDir, "long_term", taskID)...)
	sort.SliceStable(refs, func(i, j int) bool {
		if refs[i].Date == refs[j].Date {
			if refs[i].Tier == refs[j].Tier {
				return refs[i].Path < refs[j].Path
			}
			return refs[i].Tier < refs[j].Tier
		}
		return refs[i].Date > refs[j].Date
	})
	return refs
}

func buildTaskSummaryMarkdownEntry(task taskdomain.Task, summary taskdomain.TaskSummary) string {
	tags := strings.Join(summary.Tags, ", ")
	if strings.TrimSpace(tags) == "" {
		tags = "task"
	}
	finished := summary.FinishedAt
	if finished.IsZero() {
		finished = task.FinishedAt
	}
	if finished.IsZero() {
		finished = time.Now().UTC()
	}
	builder := &strings.Builder{}
	builder.WriteString("### task_id: ")
	builder.WriteString(strings.TrimSpace(summary.TaskID))
	builder.WriteString("\n")
	builder.WriteString("- task_type: ")
	builder.WriteString(strings.TrimSpace(summary.TaskType))
	builder.WriteString("\n")
	builder.WriteString("- goal: ")
	builder.WriteString(strings.TrimSpace(summary.Goal))
	builder.WriteString("\n")
	builder.WriteString("- result: ")
	builder.WriteString(strings.TrimSpace(summary.Result))
	builder.WriteString("\n")
	builder.WriteString("- status: ")
	builder.WriteString(strings.TrimSpace(string(summary.Status)))
	builder.WriteString("\n")
	builder.WriteString("- finished_at: ")
	builder.WriteString(finished.UTC().Format(time.RFC3339))
	builder.WriteString("\n")
	builder.WriteString("- tags: ")
	builder.WriteString(tags)
	builder.WriteString("\n")
	builder.WriteString("- session_id: ")
	builder.WriteString(strings.TrimSpace(task.SessionID))
	builder.WriteString("\n")
	builder.WriteString("- source_message_id: ")
	builder.WriteString(strings.TrimSpace(task.SourceMessageID))
	builder.WriteString("\n")
	builder.WriteString("- task_meta: .alter0/tasks/")
	builder.WriteString(strings.TrimSpace(summary.TaskID))
	builder.WriteString("/meta.json\n")
	builder.WriteString("\n")
	return builder.String()
}

func upsertTaskSummaryEntry(path string, day string, taskID string, entry string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	content := ""
	if err == nil {
		content = string(raw)
	}
	marker := "### task_id: " + taskID + "\n"
	start := strings.Index(content, marker)
	if start >= 0 {
		end := len(content)
		tail := content[start+len(marker):]
		next := strings.Index(tail, "\n### task_id: ")
		if next >= 0 {
			end = start + len(marker) + next + 1
		}
		prefix := content[:start]
		suffix := content[end:]
		content = prefix + entry + suffix
	} else {
		if strings.TrimSpace(content) == "" {
			content = fmt.Sprintf("# Task Summaries %s\n\n", day)
		} else if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += entry
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func scanTaskSummaryRefs(dir string, tier string, taskID string) []SummaryReference {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []SummaryReference{}
	}
	needle := "task_id: " + taskID
	refs := make([]SummaryReference, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if !strings.HasSuffix(strings.ToLower(name), ".md") {
			continue
		}
		path := filepath.Join(dir, name)
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if !strings.Contains(string(raw), needle) {
			continue
		}
		date := strings.TrimSuffix(name, filepath.Ext(name))
		if _, err := time.Parse("2006-01-02", date); err != nil {
			date = ""
		}
		refs = append(refs, SummaryReference{
			TaskID: taskID,
			Path:   path,
			Date:   date,
			Tier:   tier,
		})
	}
	return refs
}

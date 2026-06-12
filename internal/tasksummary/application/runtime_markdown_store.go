package application

import (
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

func (s *RuntimeMarkdownStore) Rebuild(_ taskdomain.Task) ([]SummaryReference, error) {
	// Durable memory Markdown is agent-managed. Task summaries remain on the
	// task domain object and in task storage; memory files are updated only by
	// an agent through the resolved memory context.
	return []SummaryReference{}, nil
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

package web

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	taskdomain "alter0/internal/task/domain"
	tasksummaryapp "alter0/internal/tasksummary/application"
)

const (
	defaultWebLongTermMemoryPath = ".alter0/memory/long-term/MEMORY.md"
	defaultWebDailyMemoryDir     = ".alter0/memory"
	defaultWebRootInstructions   = "AGENTS.md"
	defaultWebMandatoryFilePath  = "SOUL.md"
	defaultWebSpecFilePath       = "docs/memory/persistent-memory-module-spec.md"
	defaultWebDailyMemoryLimit   = 30
)

type MemoryContextOptions struct {
	LongTermPath         string
	DailyDir             string
	RootInstructionsPath string
	MandatoryContextPath string
	SpecPath             string
	DailyLimit           int
	TaskSummaryRuntime   TaskSummaryRuntime
}

type TaskSummaryRuntime interface {
	FindSummaryRefs(taskID string) []tasksummaryapp.SummaryReference
	Rebuild(task taskdomain.Task) ([]tasksummaryapp.SummaryReference, error)
}

type memoryContextService struct {
	options            MemoryContextOptions
	taskSummaryRuntime TaskSummaryRuntime
}

type memoryContextResponse struct {
	LongTerm         memoryContextDocument `json:"long_term"`
	Daily            memoryContextDaily    `json:"daily"`
	RootInstructions memoryContextDocument `json:"root_instructions"`
	Mandatory        memoryContextDocument `json:"mandatory"`
	Specification    memoryContextDocument `json:"specification"`
}

type memoryContextDocument struct {
	Path      string `json:"path"`
	Exists    bool   `json:"exists"`
	UpdatedAt string `json:"updated_at,omitempty"`
	Content   string `json:"content,omitempty"`
	Error     string `json:"error,omitempty"`
}

type memoryContextDaily struct {
	Directory string                   `json:"directory"`
	Items     []memoryContextDailyItem `json:"items"`
	Error     string                   `json:"error,omitempty"`
}

type memoryContextDailyItem struct {
	Date      string `json:"date"`
	Path      string `json:"path"`
	UpdatedAt string `json:"updated_at,omitempty"`
	Content   string `json:"content,omitempty"`
	Error     string `json:"error,omitempty"`
}

func newMemoryContextService(options MemoryContextOptions) *memoryContextService {
	normalized := normalizeMemoryContextOptions(options)
	runtime := normalized.TaskSummaryRuntime
	if runtime == nil {
		runtime = tasksummaryapp.NewRuntimeMarkdownStore(tasksummaryapp.RuntimeMarkdownOptions{
			DailyDir:    normalized.DailyDir,
			LongTermDir: filepath.Join(normalized.DailyDir, "long-term"),
		})
	}
	return &memoryContextService{
		options:            normalized,
		taskSummaryRuntime: runtime,
	}
}

func normalizeMemoryContextOptions(options MemoryContextOptions) MemoryContextOptions {
	longTermPath := strings.TrimSpace(options.LongTermPath)
	if longTermPath == "" {
		longTermPath = defaultWebLongTermMemoryPath
	}
	dailyDir := strings.TrimSpace(options.DailyDir)
	if dailyDir == "" {
		dailyDir = defaultWebDailyMemoryDir
	}
	rootInstructionsPath := strings.TrimSpace(options.RootInstructionsPath)
	if rootInstructionsPath == "" {
		rootInstructionsPath = defaultWebRootInstructions
	}
	mandatoryContextPath := strings.TrimSpace(options.MandatoryContextPath)
	if mandatoryContextPath == "" {
		mandatoryContextPath = defaultWebMandatoryFilePath
	}
	specPath := strings.TrimSpace(options.SpecPath)
	if specPath == "" {
		specPath = defaultWebSpecFilePath
	}
	dailyLimit := options.DailyLimit
	if dailyLimit <= 0 {
		dailyLimit = defaultWebDailyMemoryLimit
	}
	return MemoryContextOptions{
		LongTermPath:         filepath.Clean(longTermPath),
		DailyDir:             filepath.Clean(dailyDir),
		RootInstructionsPath: filepath.Clean(rootInstructionsPath),
		MandatoryContextPath: filepath.Clean(mandatoryContextPath),
		SpecPath:             filepath.Clean(specPath),
		DailyLimit:           dailyLimit,
	}
}

func (s *memoryContextService) Snapshot() memoryContextResponse {
	return memoryContextResponse{
		LongTerm:         s.readDocument(s.options.LongTermPath),
		Daily:            s.readDailyMemory(),
		RootInstructions: s.readDocument(s.options.RootInstructionsPath),
		Mandatory:        s.readDocument(s.options.MandatoryContextPath),
		Specification:    s.readDocument(s.options.SpecPath),
	}
}

func (s *memoryContextService) TaskSummaryRefs(taskID string) []tasksummaryapp.SummaryReference {
	if s == nil || s.taskSummaryRuntime == nil {
		return []tasksummaryapp.SummaryReference{}
	}
	refs := s.taskSummaryRuntime.FindSummaryRefs(strings.TrimSpace(taskID))
	if len(refs) == 0 {
		return []tasksummaryapp.SummaryReference{}
	}
	items := make([]tasksummaryapp.SummaryReference, 0, len(refs))
	items = append(items, refs...)
	return items
}

func (s *memoryContextService) RebuildTaskSummary(task taskdomain.Task) ([]tasksummaryapp.SummaryReference, error) {
	if s == nil || s.taskSummaryRuntime == nil {
		return []tasksummaryapp.SummaryReference{}, nil
	}
	refs, err := s.taskSummaryRuntime.Rebuild(task)
	if err != nil {
		return nil, err
	}
	if len(refs) == 0 {
		return []tasksummaryapp.SummaryReference{}, nil
	}
	items := make([]tasksummaryapp.SummaryReference, 0, len(refs))
	items = append(items, refs...)
	return items, nil
}

func (s *memoryContextService) readDocument(path string) memoryContextDocument {
	path = strings.TrimSpace(path)
	if path == "" {
		return memoryContextDocument{
			Path:  path,
			Error: "path not configured",
		}
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return memoryContextDocument{Path: path}
		}
		return memoryContextDocument{
			Path:  path,
			Error: err.Error(),
		}
	}
	if info.IsDir() {
		return memoryContextDocument{
			Path:  path,
			Error: "path points to a directory",
		}
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return memoryContextDocument{
			Path:  path,
			Error: err.Error(),
		}
	}
	return memoryContextDocument{
		Path:      path,
		Exists:    true,
		UpdatedAt: info.ModTime().UTC().Format(time.RFC3339),
		Content:   string(raw),
	}
}

func (s *memoryContextService) readDailyMemory() memoryContextDaily {
	view := memoryContextDaily{
		Directory: s.options.DailyDir,
		Items:     []memoryContextDailyItem{},
	}
	entries, err := os.ReadDir(s.options.DailyDir)
	if err != nil {
		if os.IsNotExist(err) {
			return view
		}
		view.Error = err.Error()
		return view
	}

	items := make([]memoryContextDailyItem, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if !strings.HasSuffix(strings.ToLower(name), ".md") {
			continue
		}
		datePart := strings.TrimSuffix(name, filepath.Ext(name))
		day, err := time.Parse("2006-01-02", datePart)
		if err != nil {
			continue
		}

		path := filepath.Join(s.options.DailyDir, name)
		info, err := entry.Info()
		if err != nil {
			items = append(items, memoryContextDailyItem{
				Date:  day.UTC().Format("2006-01-02"),
				Path:  path,
				Error: err.Error(),
			})
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			items = append(items, memoryContextDailyItem{
				Date:  day.UTC().Format("2006-01-02"),
				Path:  path,
				Error: err.Error(),
			})
			continue
		}
		items = append(items, memoryContextDailyItem{
			Date:      day.UTC().Format("2006-01-02"),
			Path:      path,
			UpdatedAt: info.ModTime().UTC().Format(time.RFC3339),
			Content:   string(raw),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Date == items[j].Date {
			return items[i].Path > items[j].Path
		}
		return items[i].Date > items[j].Date
	})
	if s.options.DailyLimit > 0 && len(items) > s.options.DailyLimit {
		items = items[:s.options.DailyLimit]
	}
	view.Items = items
	return view
}

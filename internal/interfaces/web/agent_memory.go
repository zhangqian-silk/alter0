package web

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	defaultWebLongTermMemoryPath = ".alter0/memory/long-term/MEMORY.md"
	defaultWebDailyMemoryDir     = ".alter0/memory"
	defaultWebMandatoryFilePath  = "SOUL.md"
	defaultWebSpecFilePath       = "docs/memory/persistent-memory-module-spec.md"
	defaultWebDailyMemoryLimit   = 30
)

type AgentMemoryOptions struct {
	LongTermPath         string
	DailyDir             string
	MandatoryContextPath string
	SpecPath             string
	DailyLimit           int
}

type agentMemoryService struct {
	options AgentMemoryOptions
}

type agentMemoryResponse struct {
	LongTerm      agentMemoryDocument `json:"long_term"`
	Daily         agentMemoryDaily    `json:"daily"`
	Mandatory     agentMemoryDocument `json:"mandatory"`
	Specification agentMemoryDocument `json:"specification"`
}

type agentMemoryDocument struct {
	Path      string `json:"path"`
	Exists    bool   `json:"exists"`
	UpdatedAt string `json:"updated_at,omitempty"`
	Content   string `json:"content,omitempty"`
	Error     string `json:"error,omitempty"`
}

type agentMemoryDaily struct {
	Directory string                 `json:"directory"`
	Items     []agentMemoryDailyItem `json:"items"`
	Error     string                 `json:"error,omitempty"`
}

type agentMemoryDailyItem struct {
	Date      string `json:"date"`
	Path      string `json:"path"`
	UpdatedAt string `json:"updated_at,omitempty"`
	Content   string `json:"content,omitempty"`
	Error     string `json:"error,omitempty"`
}

func newAgentMemoryService(options AgentMemoryOptions) *agentMemoryService {
	return &agentMemoryService{
		options: normalizeAgentMemoryOptions(options),
	}
}

func normalizeAgentMemoryOptions(options AgentMemoryOptions) AgentMemoryOptions {
	longTermPath := strings.TrimSpace(options.LongTermPath)
	if longTermPath == "" {
		longTermPath = defaultWebLongTermMemoryPath
	}
	dailyDir := strings.TrimSpace(options.DailyDir)
	if dailyDir == "" {
		dailyDir = defaultWebDailyMemoryDir
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
	return AgentMemoryOptions{
		LongTermPath:         filepath.Clean(longTermPath),
		DailyDir:             filepath.Clean(dailyDir),
		MandatoryContextPath: filepath.Clean(mandatoryContextPath),
		SpecPath:             filepath.Clean(specPath),
		DailyLimit:           dailyLimit,
	}
}

func (s *agentMemoryService) Snapshot() agentMemoryResponse {
	return agentMemoryResponse{
		LongTerm:      s.readDocument(s.options.LongTermPath),
		Daily:         s.readDailyMemory(),
		Mandatory:     s.readDocument(s.options.MandatoryContextPath),
		Specification: s.readDocument(s.options.SpecPath),
	}
}

func (s *agentMemoryService) readDocument(path string) agentMemoryDocument {
	path = strings.TrimSpace(path)
	if path == "" {
		return agentMemoryDocument{
			Path:  path,
			Error: "path not configured",
		}
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return agentMemoryDocument{Path: path}
		}
		return agentMemoryDocument{
			Path:  path,
			Error: err.Error(),
		}
	}
	if info.IsDir() {
		return agentMemoryDocument{
			Path:  path,
			Error: "path points to a directory",
		}
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return agentMemoryDocument{
			Path:  path,
			Error: err.Error(),
		}
	}
	return agentMemoryDocument{
		Path:      path,
		Exists:    true,
		UpdatedAt: info.ModTime().UTC().Format(time.RFC3339),
		Content:   string(raw),
	}
}

func (s *agentMemoryService) readDailyMemory() agentMemoryDaily {
	view := agentMemoryDaily{
		Directory: s.options.DailyDir,
		Items:     []agentMemoryDailyItem{},
	}
	entries, err := os.ReadDir(s.options.DailyDir)
	if err != nil {
		if os.IsNotExist(err) {
			return view
		}
		view.Error = err.Error()
		return view
	}

	items := make([]agentMemoryDailyItem, 0, len(entries))
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
			items = append(items, agentMemoryDailyItem{
				Date:  day.UTC().Format("2006-01-02"),
				Path:  path,
				Error: err.Error(),
			})
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			items = append(items, agentMemoryDailyItem{
				Date:  day.UTC().Format("2006-01-02"),
				Path:  path,
				Error: err.Error(),
			})
			continue
		}
		items = append(items, agentMemoryDailyItem{
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

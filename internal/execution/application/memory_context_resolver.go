package application

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	execdomain "alter0/internal/execution/domain"
	shareddomain "alter0/internal/shared/domain"
)

const (
	memoryIncludeFilterKey       = "alter0.memory.include"
	memoryFileMaxChars           = 12000
	memoryFileTotalMaxChars      = 36000
	memoryFileTruncatedSuffix    = "\n...[truncated]"
	memorySelectionUserMD        = "user_md"
	memorySelectionSoulMD        = "soul_md"
	memorySelectionAgentsMD      = "agents_md"
	memorySelectionLongTerm      = "memory_long_term"
	memorySelectionDailyToday    = "memory_daily_today"
	memorySelectionDailyPrevious = "memory_daily_yesterday"
)

type memoryContextResolution struct {
	Context     execdomain.MemoryContext
	InjectedIDs []string
}

type memoryContextResolver struct{}

type memorySelectionSpec struct {
	ID    string
	Title string
	Paths func(msg shareddomain.UnifiedMessage, now time.Time) []string
}

type memoryFileRecord struct {
	ID        string
	Selection string
	Title     string
	Path      string
	Exists    bool
	UpdatedAt string
	Content   string
}

func newMemoryContextResolver() *memoryContextResolver {
	return &memoryContextResolver{}
}

func (r *memoryContextResolver) Resolve(msg shareddomain.UnifiedMessage) memoryContextResolution {
	resolution := memoryContextResolution{
		Context: execdomain.MemoryContext{Protocol: execdomain.MemoryContextProtocolVersion},
	}
	if r == nil {
		return resolution
	}

	selected := parseList(metadataValue(msg.Metadata, memoryIncludeFilterKey))
	if len(selected) == 0 {
		return resolution
	}

	repoRoot, err := resolveMemoryRepoRoot()
	if err != nil {
		return resolution
	}
	now := msg.ReceivedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}

	records := make([]memoryFileRecord, 0, len(selected))
	remainingChars := memoryFileTotalMaxChars
	for _, selectionID := range selected {
		spec, ok := memorySelectionByID(now)[strings.ToLower(strings.TrimSpace(selectionID))]
		if !ok {
			continue
		}
		files := loadSelectionMemoryFiles(spec, repoRoot, msg, now, &remainingChars)
		if len(files) == 0 {
			continue
		}
		for _, file := range files {
			records = append(records, file)
		}
		resolution.InjectedIDs = append(resolution.InjectedIDs, spec.ID)
	}

	if len(records) == 0 {
		return resolution
	}
	files := make([]execdomain.MemoryFileSpec, 0, len(records))
	for _, record := range records {
		files = append(files, execdomain.MemoryFileSpec{
			ID:        record.ID,
			Selection: record.Selection,
			Title:     record.Title,
			Path:      record.Path,
			Exists:    record.Exists,
			Writable:  true,
			UpdatedAt: record.UpdatedAt,
			Content:   record.Content,
		})
	}
	resolution.Context.Files = files
	return resolution
}

func memorySelectionByID(now time.Time) map[string]memorySelectionSpec {
	day := now.UTC().Format("2006-01-02")
	yesterday := now.UTC().Add(-24 * time.Hour).Format("2006-01-02")
	return map[string]memorySelectionSpec{
		memorySelectionUserMD: {
			ID:    memorySelectionUserMD,
			Title: "USER.md",
			Paths: func(_ shareddomain.UnifiedMessage, _ time.Time) []string { return []string{"USER.md"} },
		},
		memorySelectionSoulMD: {
			ID:    memorySelectionSoulMD,
			Title: "SOUL.md",
			Paths: func(_ shareddomain.UnifiedMessage, _ time.Time) []string { return []string{"SOUL.md"} },
		},
		memorySelectionAgentsMD: {
			ID:    memorySelectionAgentsMD,
			Title: "AGENTS.md",
			Paths: func(msg shareddomain.UnifiedMessage, _ time.Time) []string {
				return []string{
					filepath.ToSlash(filepath.Join(".alter0", "agents", normalizeMemoryAgentID(metadataValue(msg.Metadata, execdomain.AgentIDMetadataKey)), "AGENTS.md")),
				}
			},
		},
		memorySelectionLongTerm: {
			ID:    memorySelectionLongTerm,
			Title: "MEMORY.md",
			Paths: func(_ shareddomain.UnifiedMessage, _ time.Time) []string {
				return []string{
					filepath.ToSlash(filepath.Join(".alter0", "memory", "long-term", "MEMORY.md")),
					"MEMORY.md",
					"memory.md",
				}
			},
		},
		memorySelectionDailyToday: {
			ID:    memorySelectionDailyToday,
			Title: "daily memory (today)",
			Paths: func(_ shareddomain.UnifiedMessage, _ time.Time) []string {
				return []string{
					filepath.ToSlash(filepath.Join(".alter0", "memory", day+".md")),
					filepath.ToSlash(filepath.Join("memory", day+".md")),
				}
			},
		},
		memorySelectionDailyPrevious: {
			ID:    memorySelectionDailyPrevious,
			Title: "daily memory (yesterday)",
			Paths: func(_ shareddomain.UnifiedMessage, _ time.Time) []string {
				return []string{
					filepath.ToSlash(filepath.Join(".alter0", "memory", yesterday+".md")),
					filepath.ToSlash(filepath.Join("memory", yesterday+".md")),
				}
			},
		},
	}
}

var memoryAgentIDSanitizer = regexp.MustCompile(`[^a-z0-9._-]+`)

func normalizeMemoryAgentID(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	normalized = memoryAgentIDSanitizer.ReplaceAllString(normalized, "-")
	normalized = strings.Trim(normalized, "-.")
	if normalized == "" {
		return "unknown"
	}
	return normalized
}

func resolveMemoryRepoRoot() (string, error) {
	root, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve memory repo root: %w", err)
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve memory repo root: %w", err)
	}
	return absolute, nil
}

func loadSelectionMemoryFiles(spec memorySelectionSpec, repoRoot string, msg shareddomain.UnifiedMessage, now time.Time, remainingChars *int) []memoryFileRecord {
	paths := spec.Paths(msg, now)
	if len(paths) == 0 {
		return nil
	}
	records := make([]memoryFileRecord, 0, len(paths))
	seenRealpaths := map[string]struct{}{}
	existingCount := 0
	for _, relativePath := range paths {
		candidatePath := filepath.Join(repoRoot, filepath.FromSlash(relativePath))
		record := memoryFileRecord{
			ID:        spec.ID,
			Selection: spec.ID,
			Title:     spec.Title,
			Path:      filepath.ToSlash(candidatePath),
		}

		info, statErr := os.Stat(candidatePath)
		if statErr != nil {
			continue
		}
		if info.IsDir() {
			continue
		}
		realpath := candidatePath
		if resolved, err := filepath.EvalSymlinks(candidatePath); err == nil {
			realpath = resolved
		}
		if _, exists := seenRealpaths[realpath]; exists {
			continue
		}
		seenRealpaths[realpath] = struct{}{}

		record.Exists = true
		record.UpdatedAt = info.ModTime().UTC().Format(time.RFC3339)
		raw, readErr := os.ReadFile(candidatePath)
		if readErr == nil {
			record.Content = truncateMemoryContent(normalizeMemoryContent(string(raw)), remainingChars)
		}
		records = append(records, record)
		existingCount++
	}
	if existingCount == 0 && len(records) == 0 {
		records = append(records, memoryFileRecord{
			ID:        spec.ID,
			Selection: spec.ID,
			Title:     spec.Title,
			Path:      filepath.ToSlash(filepath.Join(repoRoot, filepath.FromSlash(paths[0]))),
		})
	}
	return records
}

func normalizeMemoryContent(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	return strings.TrimSpace(content)
}

func truncateMemoryContent(content string, remainingChars *int) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	limit := memoryFileMaxChars
	if remainingChars != nil && *remainingChars > 0 && *remainingChars < limit {
		limit = *remainingChars
	}
	if remainingChars != nil && *remainingChars <= 0 {
		return ""
	}
	if limit <= 0 {
		return ""
	}

	result := content
	if len(result) > limit {
		if limit > len(memoryFileTruncatedSuffix) {
			result = result[:limit-len(memoryFileTruncatedSuffix)] + memoryFileTruncatedSuffix
		} else {
			result = result[:limit]
		}
	}
	if remainingChars != nil {
		*remainingChars -= len(result)
		if *remainingChars < 0 {
			*remainingChars = 0
		}
	}
	return result
}

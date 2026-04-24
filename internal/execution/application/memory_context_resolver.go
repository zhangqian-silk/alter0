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
	memoryAutoRecallMaxHits      = 6
	memoryAutoRecallSnippetLines = 3
	memorySelectionUserMD        = "user_md"
	memorySelectionSoulMD        = "soul_md"
	memorySelectionAgentsMD      = "agents_md"
	memorySelectionLongTerm      = "memory_long_term"
	memorySelectionDailyToday    = "memory_daily_today"
	memorySelectionDailyPrevious = "memory_daily_yesterday"
	memorySelectionAgentSession  = "agent_session_profile"
)

type memoryContextResolution struct {
	Context     execdomain.MemoryContext
	InjectedIDs []string
}

type memoryContextResolver struct{}

type memorySelectionSpec struct {
	ID       string
	Title    string
	Writable bool
	Paths    func(msg shareddomain.UnifiedMessage, now time.Time) []string
}

type memoryFileRecord struct {
	ID        string
	Selection string
	Title     string
	Path      string
	Exists    bool
	Writable  bool
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

	selected := appendImplicitMemorySelections(
		parseList(metadataValue(msg.Metadata, memoryIncludeFilterKey)),
		msg,
	)
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
		if spec.ID == memorySelectionAgentSession {
			if _, err := ensureAgentSessionProfileFile(repoRoot, msg, now); err != nil {
				continue
			}
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
			Writable:  record.Writable,
			UpdatedAt: record.UpdatedAt,
			Content:   record.Content,
		})
	}
	resolution.Context.Files = files
	resolution.Context.Recall = buildMemoryAutoRecall(msg.Content, files, memoryAutoRecallMaxHits)
	return resolution
}

func memorySelectionByID(now time.Time) map[string]memorySelectionSpec {
	day := now.UTC().Format("2006-01-02")
	yesterday := now.UTC().Add(-24 * time.Hour).Format("2006-01-02")
	return map[string]memorySelectionSpec{
		memorySelectionUserMD: {
			ID:       memorySelectionUserMD,
			Title:    "USER.md",
			Writable: true,
			Paths:    func(_ shareddomain.UnifiedMessage, _ time.Time) []string { return []string{"USER.md"} },
		},
		memorySelectionSoulMD: {
			ID:       memorySelectionSoulMD,
			Title:    "SOUL.md",
			Writable: true,
			Paths:    func(_ shareddomain.UnifiedMessage, _ time.Time) []string { return []string{"SOUL.md"} },
		},
		memorySelectionAgentsMD: {
			ID:       memorySelectionAgentsMD,
			Title:    "AGENTS.md",
			Writable: true,
			Paths: func(msg shareddomain.UnifiedMessage, _ time.Time) []string {
				return agentPrivateRelativePaths(metadataValue(msg.Metadata, execdomain.AgentIDMetadataKey), "AGENTS.md")
			},
		},
		memorySelectionLongTerm: {
			ID:       memorySelectionLongTerm,
			Title:    "MEMORY.md",
			Writable: true,
			Paths: func(_ shareddomain.UnifiedMessage, _ time.Time) []string {
				return []string{
					filepath.ToSlash(filepath.Join(".alter0", "memory", "long-term", "MEMORY.md")),
					"MEMORY.md",
					"memory.md",
				}
			},
		},
		memorySelectionDailyToday: {
			ID:       memorySelectionDailyToday,
			Title:    "daily memory (today)",
			Writable: true,
			Paths: func(_ shareddomain.UnifiedMessage, _ time.Time) []string {
				return []string{
					filepath.ToSlash(filepath.Join(".alter0", "memory", day+".md")),
					filepath.ToSlash(filepath.Join("memory", day+".md")),
				}
			},
		},
		memorySelectionDailyPrevious: {
			ID:       memorySelectionDailyPrevious,
			Title:    "daily memory (yesterday)",
			Writable: true,
			Paths: func(_ shareddomain.UnifiedMessage, _ time.Time) []string {
				return []string{
					filepath.ToSlash(filepath.Join(".alter0", "memory", yesterday+".md")),
					filepath.ToSlash(filepath.Join("memory", yesterday+".md")),
				}
			},
		},
		memorySelectionAgentSession: {
			ID:       memorySelectionAgentSession,
			Title:    "Agent Session Profile",
			Writable: false,
			Paths: func(msg shareddomain.UnifiedMessage, _ time.Time) []string {
				path := agentSessionProfileRelativePath(msg)
				if path == "" {
					return nil
				}
				return []string{path}
			},
		},
	}
}

var memoryAgentIDSanitizer = regexp.MustCompile(`[^a-z0-9._-]+`)
var memoryRecallTokenPattern = regexp.MustCompile(`[\p{Han}]+|[A-Za-z0-9_./-]+`)

var memoryRecallStopWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "for": {}, "from": {}, "how": {}, "in": {}, "is": {}, "it": {}, "me": {}, "my": {}, "of": {}, "on": {}, "or": {}, "please": {}, "that": {}, "the": {}, "this": {}, "to": {}, "use": {}, "using": {}, "what": {}, "when": {}, "where": {}, "why": {}, "with": {},
	"什么": {}, "一下": {}, "一个": {}, "这个": {}, "那个": {}, "当前": {}, "现在": {}, "哪些": {}, "使用": {}, "帮忙": {}, "帮我": {}, "怎么": {}, "如何": {}, "为什么": {}, "感觉": {},
}

func buildMemoryAutoRecall(input string, files []execdomain.MemoryFileSpec, limit int) []execdomain.MemoryRecallHit {
	terms := memoryRecallTerms(input)
	if len(terms) == 0 || len(files) == 0 {
		return nil
	}
	if limit <= 0 {
		limit = memoryAutoRecallMaxHits
	}
	hits := make([]execdomain.MemoryRecallHit, 0, limit)
	seen := map[string]struct{}{}
	for _, file := range files {
		content := strings.TrimSpace(file.Content)
		if content == "" {
			continue
		}
		lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
		for idx, line := range lines {
			if !memoryLineMatchesRecallTerms(line, terms) {
				continue
			}
			key := strings.TrimSpace(file.Path) + ":" + fmt.Sprint(idx+1)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			hits = append(hits, execdomain.MemoryRecallHit{
				MemoryID: strings.TrimSpace(file.ID),
				Title:    strings.TrimSpace(file.Title),
				Path:     strings.TrimSpace(file.Path),
				Line:     idx + 1,
				Snippet:  buildMemoryRecallSnippet(lines, idx),
			})
			if len(hits) >= limit {
				return hits
			}
		}
	}
	return hits
}

func memoryRecallTerms(input string) []string {
	rawTerms := memoryRecallTokenPattern.FindAllString(strings.ToLower(strings.TrimSpace(input)), -1)
	terms := make([]string, 0, len(rawTerms))
	seen := map[string]struct{}{}
	appendTerm := func(term string) {
		term = strings.TrimSpace(strings.ToLower(term))
		if term == "" {
			return
		}
		if _, stop := memoryRecallStopWords[term]; stop {
			return
		}
		runes := []rune(term)
		if len(runes) < 2 {
			return
		}
		if isASCIIRecallTerm(term) && len(term) < 3 {
			return
		}
		if _, ok := seen[term]; ok {
			return
		}
		seen[term] = struct{}{}
		terms = append(terms, term)
	}
	for _, raw := range rawTerms {
		term := strings.TrimSpace(raw)
		if term == "" {
			continue
		}
		runes := []rune(term)
		if containsHan(term) && len(runes) > 8 {
			for idx := 0; idx+4 <= len(runes); idx += 2 {
				appendTerm(string(runes[idx : idx+4]))
			}
			continue
		}
		appendTerm(term)
	}
	return terms
}

func memoryLineMatchesRecallTerms(line string, terms []string) bool {
	if strings.TrimSpace(line) == "" || len(terms) == 0 {
		return false
	}
	lower := strings.ToLower(line)
	for _, term := range terms {
		if strings.Contains(lower, term) {
			return true
		}
	}
	return false
}

func buildMemoryRecallSnippet(lines []string, index int) string {
	if index < 0 || index >= len(lines) {
		return ""
	}
	start := index - 1
	if start < 0 {
		start = 0
	}
	end := start + memoryAutoRecallSnippetLines
	if end > len(lines) {
		end = len(lines)
	}
	parts := make([]string, 0, end-start)
	for idx := start; idx < end; idx++ {
		line := strings.TrimSpace(lines[idx])
		if line == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("L%d: %s", idx+1, line))
	}
	return strings.Join(parts, "\n")
}

func containsHan(value string) bool {
	for _, r := range value {
		if r >= '\u4e00' && r <= '\u9fff' {
			return true
		}
	}
	return false
}

func isASCIIRecallTerm(value string) bool {
	for _, r := range value {
		if r > 127 {
			return false
		}
	}
	return true
}

func appendImplicitMemorySelections(selected []string, msg shareddomain.UnifiedMessage) []string {
	items := make([]string, 0, len(selected)+1)
	seen := map[string]struct{}{}
	appendItem := func(value string) {
		trimmed := strings.ToLower(strings.TrimSpace(value))
		if trimmed == "" {
			return
		}
		if _, ok := seen[trimmed]; ok {
			return
		}
		seen[trimmed] = struct{}{}
		items = append(items, trimmed)
	}
	for _, item := range selected {
		appendItem(item)
	}
	if strings.TrimSpace(metadataValue(msg.Metadata, execdomain.AgentIDMetadataKey)) != "" {
		appendItem(memorySelectionAgentSession)
	}
	return items
}

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
			Writable:  spec.Writable,
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
			Writable:  spec.Writable,
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

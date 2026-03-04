package application

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	taskdomain "alter0/internal/task/domain"
)

const (
	defaultRecentWindow      = 5
	defaultDeepTopK          = 5
	defaultDetailLogLimit    = 6
	defaultDetailArtifactLim = 3
	minRecentWindow          = 3
	maxRecentWindow          = 5
)

type RetrievalMode string

const (
	RetrievalModeRecent   RetrievalMode = "recent"
	RetrievalModeDeep     RetrievalMode = "deep"
	RetrievalModeFallback RetrievalMode = "fallback"
)

type Options struct {
	RecentWindow       int
	DeepTopK           int
	DetailLogLimit     int
	DetailArtifactLimit int
}

type TaskDetail struct {
	TaskID    string               `json:"task_id"`
	Status    taskdomain.TaskStatus `json:"status"`
	Logs      []taskdomain.TaskLog  `json:"logs,omitempty"`
	Artifacts []taskdomain.TaskArtifact `json:"artifacts,omitempty"`
}

type Snapshot struct {
	Summaries      []taskdomain.TaskSummary
	Details        []TaskDetail
	Mode           RetrievalMode
	DeepTriggered  bool
	DeepOverridden bool
	DeepMiss       bool
	Truncated      bool
}

func (s Snapshot) Metadata() map[string]string {
	return map[string]string{
		"task_summary_injected_count": strconv.Itoa(len(s.Summaries)),
		"task_summary_retrieval_mode": string(s.Mode),
		"task_summary_detail_count":   strconv.Itoa(len(s.Details)),
		"task_summary_truncated":      strconv.FormatBool(s.Truncated),
		"deep_retrieval_triggered":    strconv.FormatBool(s.DeepTriggered),
		"deep_retrieval_overridden":   strconv.FormatBool(s.DeepOverridden),
		"deep_retrieval_miss":         strconv.FormatBool(s.DeepMiss),
	}
}

func (s Snapshot) ResultMetadata() map[string]string {
	return s.Metadata()
}

type Store struct {
	mu      sync.RWMutex
	options Options
	entries map[string]taskSummaryEntry
}

type taskSummaryEntry struct {
	summary taskdomain.TaskSummary
	task    taskdomain.Task
}

type retrievalIntent struct {
	historySignal bool
	taskSignal    bool
	timeSignal    bool
	negation      bool
	currentTask   bool
	detail        bool
	taskIDs       map[string]struct{}
	dates         map[string]struct{}
	tokens        []string
}

type scoredSummary struct {
	entry taskSummaryEntry
	score float64
}

var (
	historySignals = []string{
		"更早", "之前", "历史", "上次", "上周", "上个月", "最早", "第一条", "当时", "那次",
		"earlier", "previous", "history", "last time", "last week", "first task", "older",
	}
	taskSignals = []string{
		"任务", "需求", "pr", "执行", "task", "tasks", "日志", "log", "logs", "产物", "artifact", "artifacts",
	}
	negationSignals = []string{
		"不要查历史", "不看以前", "只看当前", "不用历史", "no history", "current only", "don't check history",
	}
	currentTaskSignals = []string{
		"这个任务", "当前任务", "现在进度", "this task", "current task", "latest task",
	}
	detailSignals = []string{
		"细节", "详情", "详细", "日志", "报错", "原因", "完整过程", "log", "logs", "artifact", "artifacts", "stack", "trace",
	}
	timeSignals = []string{
		"上周", "上个月", "昨天", "前天", "last week", "last month", "days ago", "weeks ago", "months ago",
	}
	taskIDPattern       = regexp.MustCompile(`(?i)\btask-[a-z0-9_-]+\b`)
	explicitDatePattern = regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}\b`)
	relativeTimePattern = regexp.MustCompile(`(?i)(\d+\s*(天前|周前|个月前)|\d+\s*(days?|weeks?|months?)\s+ago)`)
	tokenPattern        = regexp.MustCompile(`[\p{Han}]{1,}|[a-z0-9_./-]+`)
)

func NewStore(options Options) *Store {
	normalized := normalizeOptions(options)
	return &Store{
		options: normalized,
		entries: map[string]taskSummaryEntry{},
	}
}

func normalizeOptions(options Options) Options {
	if options.RecentWindow <= 0 {
		options.RecentWindow = defaultRecentWindow
	}
	if options.RecentWindow < minRecentWindow {
		options.RecentWindow = minRecentWindow
	}
	if options.RecentWindow > maxRecentWindow {
		options.RecentWindow = maxRecentWindow
	}
	if options.DeepTopK <= 0 {
		options.DeepTopK = defaultDeepTopK
	}
	if options.DetailLogLimit <= 0 {
		options.DetailLogLimit = defaultDetailLogLimit
	}
	if options.DetailArtifactLimit <= 0 {
		options.DetailArtifactLimit = defaultDetailArtifactLim
	}
	return options
}

func (s *Store) Record(task taskdomain.Task) {
	if !task.Status.IsTerminal() {
		return
	}
	summary := normalizeSummary(task)
	if summary.IsZero() {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.entries[summary.TaskID]
	if ok && !summary.FinishedAt.After(existing.summary.FinishedAt) {
		if existing.summary.Status == summary.Status && existing.summary.Result == summary.Result && existing.summary.Goal == summary.Goal {
			return
		}
	}
	s.entries[summary.TaskID] = taskSummaryEntry{
		summary: cloneSummary(summary),
		task:    cloneTask(task),
	}
}

func (s *Store) Snapshot(query string, now time.Time) Snapshot {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	entries := s.listEntries()
	if len(entries) == 0 {
		return Snapshot{Mode: RetrievalModeRecent}
	}

	intent := parseIntent(query)
	recent := selectRecentSummaries(entries, s.options.RecentWindow)
	historyRequested := intent.historySignal || intent.timeSignal || len(intent.taskIDs) > 0
	dualSignalPass := (intent.historySignal && intent.taskSignal) || intent.timeSignal || len(intent.taskIDs) > 0

	if historyRequested && (intent.negation || intent.currentTask || !dualSignalPass) {
		return Snapshot{
			Summaries:      recent,
			Mode:           RetrievalModeRecent,
			DeepTriggered:  false,
			DeepOverridden: true,
		}
	}

	if !dualSignalPass {
		return Snapshot{
			Summaries: recent,
			Mode:      RetrievalModeRecent,
		}
	}

	hits, truncated := deepRetrieve(entries, intent, s.options.DeepTopK, now)
	if len(hits) == 0 {
		return Snapshot{
			Summaries:     recent,
			Mode:          RetrievalModeFallback,
			DeepTriggered: true,
			DeepMiss:      true,
		}
	}

	details := []TaskDetail{}
	if intent.detail {
		details = buildDetails(hits, intent, s.options)
	}
	return Snapshot{
		Summaries:     summariesFromEntries(hits),
		Details:       details,
		Mode:          RetrievalModeDeep,
		DeepTriggered: true,
		Truncated:     truncated,
	}
}

func (s *Store) listEntries() []taskSummaryEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := make([]taskSummaryEntry, 0, len(s.entries))
	for _, entry := range s.entries {
		entries = append(entries, taskSummaryEntry{
			summary: cloneSummary(entry.summary),
			task:    cloneTask(entry.task),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].summary.FinishedAt.Equal(entries[j].summary.FinishedAt) {
			return entries[i].summary.TaskID > entries[j].summary.TaskID
		}
		return entries[i].summary.FinishedAt.After(entries[j].summary.FinishedAt)
	})
	return entries
}

func normalizeSummary(task taskdomain.Task) taskdomain.TaskSummary {
	summary := task.TaskSummary
	if summary.IsZero() {
		summary = taskdomain.TaskSummary{
			TaskID:     strings.TrimSpace(task.ID),
			TaskType:   resolveTaskType(task),
			Goal:       snippet(task.RequestContent, 180),
			Result:     resolveTaskResult(task),
			Status:     task.Status,
			FinishedAt: task.FinishedAt,
			Tags:       []string{"task", strings.ToLower(strings.TrimSpace(string(task.Status))), resolveTaskType(task)},
		}
	}
	if summary.FinishedAt.IsZero() {
		summary.FinishedAt = task.UpdatedAt
	}
	if summary.FinishedAt.IsZero() {
		summary.FinishedAt = time.Now().UTC()
	}
	if strings.TrimSpace(summary.TaskID) == "" {
		summary.TaskID = strings.TrimSpace(task.ID)
	}
	if strings.TrimSpace(summary.TaskType) == "" {
		summary.TaskType = resolveTaskType(task)
	}
	if strings.TrimSpace(summary.Goal) == "" {
		summary.Goal = snippet(task.RequestContent, 180)
	}
	if strings.TrimSpace(summary.Goal) == "" {
		summary.Goal = "task goal unavailable"
	}
	if strings.TrimSpace(summary.Result) == "" {
		summary.Result = resolveTaskResult(task)
	}
	if strings.TrimSpace(summary.Result) == "" {
		summary.Result = "task result unavailable"
	}
	if !summary.Status.IsValid() {
		summary.Status = task.Status
	}
	if len(summary.Tags) == 0 {
		summary.Tags = []string{"task", strings.ToLower(strings.TrimSpace(string(summary.Status))), strings.ToLower(strings.TrimSpace(summary.TaskType))}
	}
	summary.Tags = normalizeTags(summary.Tags)
	return summary
}

func resolveTaskType(task taskdomain.Task) string {
	if value := strings.ToLower(strings.TrimSpace(task.RequestMetadata["alter0.task.type"])); value != "" {
		return value
	}
	if route := strings.ToLower(strings.TrimSpace(string(task.Result.Route))); route != "" {
		return route
	}
	return "task"
}

func resolveTaskResult(task taskdomain.Task) string {
	switch task.Status {
	case taskdomain.TaskStatusSuccess:
		if out := snippet(task.Result.Output, 220); out != "" {
			return out
		}
		return "completed without textual output"
	case taskdomain.TaskStatusCanceled:
		if reason := snippet(task.ErrorMessage, 220); reason != "" {
			return reason
		}
		return "task canceled"
	case taskdomain.TaskStatusFailed:
		if reason := snippet(task.ErrorMessage, 220); reason != "" {
			return reason
		}
		if code := strings.TrimSpace(task.ErrorCode); code != "" {
			return code
		}
		return "task failed"
	default:
		return snippet(task.Summary, 220)
	}
}

func parseIntent(query string) retrievalIntent {
	trimmed := strings.TrimSpace(query)
	lower := strings.ToLower(trimmed)
	intent := retrievalIntent{
		historySignal: containsAny(trimmed, lower, historySignals),
		taskSignal:    containsAny(trimmed, lower, taskSignals),
		negation:      containsAny(trimmed, lower, negationSignals),
		currentTask:   containsAny(trimmed, lower, currentTaskSignals),
		detail:        containsAny(trimmed, lower, detailSignals),
		timeSignal:    containsAny(trimmed, lower, timeSignals) || relativeTimePattern.MatchString(trimmed) || explicitDatePattern.MatchString(trimmed),
		taskIDs:       map[string]struct{}{},
		dates:         map[string]struct{}{},
	}

	for _, taskID := range taskIDPattern.FindAllString(lower, -1) {
		intent.taskIDs[strings.TrimSpace(taskID)] = struct{}{}
	}
	for _, dateValue := range explicitDatePattern.FindAllString(trimmed, -1) {
		intent.dates[dateValue] = struct{}{}
	}
	intent.tokens = tokenize(lower)
	if len(intent.taskIDs) > 0 {
		intent.taskSignal = true
	}
	return intent
}

func containsAny(original string, lower string, tokens []string) bool {
	for _, token := range tokens {
		trimmed := strings.TrimSpace(token)
		if trimmed == "" {
			continue
		}
		if trimmed == strings.ToLower(trimmed) {
			if strings.Contains(lower, trimmed) {
				return true
			}
			continue
		}
		if strings.Contains(original, trimmed) {
			return true
		}
	}
	return false
}

func tokenize(content string) []string {
	matches := tokenPattern.FindAllString(strings.ToLower(content), -1)
	if len(matches) == 0 {
		return nil
	}
	tokens := make([]string, 0, len(matches))
	seen := map[string]struct{}{}
	for _, match := range matches {
		token := strings.TrimSpace(match)
		if token == "" {
			continue
		}
		if len([]rune(token)) <= 1 {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		tokens = append(tokens, token)
	}
	return tokens
}

func selectRecentSummaries(entries []taskSummaryEntry, limit int) []taskdomain.TaskSummary {
	if limit <= 0 {
		limit = defaultRecentWindow
	}
	if limit > len(entries) {
		limit = len(entries)
	}
	items := make([]taskdomain.TaskSummary, 0, limit)
	for idx := 0; idx < limit; idx++ {
		items = append(items, cloneSummary(entries[idx].summary))
	}
	return items
}

func deepRetrieve(entries []taskSummaryEntry, intent retrievalIntent, limit int, now time.Time) ([]taskSummaryEntry, bool) {
	if limit <= 0 {
		limit = defaultDeepTopK
	}
	scored := make([]scoredSummary, 0, len(entries))

	for _, entry := range entries {
		if len(intent.dates) > 0 {
			key := entry.summary.FinishedAt.UTC().Format("2006-01-02")
			if _, ok := intent.dates[key]; !ok {
				continue
			}
		}
		score, matched := scoreEntry(entry.summary, intent, now)
		if !matched {
			continue
		}
		scored = append(scored, scoredSummary{entry: entry, score: score})
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			if scored[i].entry.summary.FinishedAt.Equal(scored[j].entry.summary.FinishedAt) {
				return scored[i].entry.summary.TaskID > scored[j].entry.summary.TaskID
			}
			return scored[i].entry.summary.FinishedAt.After(scored[j].entry.summary.FinishedAt)
		}
		return scored[i].score > scored[j].score
	})

	truncated := len(scored) > limit
	if len(scored) > limit {
		scored = scored[:limit]
	}
	result := make([]taskSummaryEntry, 0, len(scored))
	for _, item := range scored {
		result = append(result, taskSummaryEntry{
			summary: cloneSummary(item.entry.summary),
			task:    cloneTask(item.entry.task),
		})
	}
	return result, truncated
}

func scoreEntry(summary taskdomain.TaskSummary, intent retrievalIntent, now time.Time) (float64, bool) {
	content := strings.ToLower(strings.Join([]string{
		summary.TaskID,
		summary.TaskType,
		summary.Goal,
		summary.Result,
		strings.Join(summary.Tags, " "),
	}, " "))
	if content == "" {
		return 0, false
	}

	score := 0.0
	if len(intent.taskIDs) > 0 {
		if _, ok := intent.taskIDs[strings.ToLower(strings.TrimSpace(summary.TaskID))]; !ok {
			return 0, false
		}
		score += 100
	}

	tokenMatch := 0
	for _, token := range intent.tokens {
		if strings.Contains(content, token) {
			tokenMatch++
			score += 3
		}
	}

	if len(intent.taskIDs) == 0 && len(intent.tokens) > 0 && tokenMatch == 0 {
		return 0, false
	}

	switch summary.Status {
	case taskdomain.TaskStatusSuccess:
		score += 0.8
	case taskdomain.TaskStatusFailed:
		score += 1.0
	}
	if strings.TrimSpace(summary.Result) != "" {
		score += 0.4
	}
	if !summary.FinishedAt.IsZero() {
		days := now.Sub(summary.FinishedAt).Hours() / 24
		if days < 0 {
			days = 0
		}
		if days <= 30 {
			score += (30 - days) / 30
		}
	}
	if len(intent.tokens) == 0 && len(intent.taskIDs) == 0 {
		score += 0.1
	}
	return score, score > 0
}

func buildDetails(entries []taskSummaryEntry, intent retrievalIntent, options Options) []TaskDetail {
	if len(entries) == 0 {
		return nil
	}
	details := make([]TaskDetail, 0, len(entries))
	for _, entry := range entries {
		taskID := strings.ToLower(strings.TrimSpace(entry.summary.TaskID))
		if len(intent.taskIDs) > 0 {
			if _, ok := intent.taskIDs[taskID]; !ok {
				continue
			}
		} else if len(details) >= 1 {
			break
		}
		detail := TaskDetail{
			TaskID: entry.summary.TaskID,
			Status: entry.summary.Status,
			Logs:   tailLogs(entry.task.Logs, options.DetailLogLimit),
			Artifacts: tailArtifacts(entry.task.Artifacts, options.DetailArtifactLimit),
		}
		details = append(details, detail)
	}
	return details
}

func summariesFromEntries(entries []taskSummaryEntry) []taskdomain.TaskSummary {
	items := make([]taskdomain.TaskSummary, 0, len(entries))
	for _, entry := range entries {
		items = append(items, cloneSummary(entry.summary))
	}
	return items
}

func tailLogs(items []taskdomain.TaskLog, limit int) []taskdomain.TaskLog {
	if len(items) == 0 || limit <= 0 {
		return nil
	}
	if len(items) <= limit {
		copied := make([]taskdomain.TaskLog, len(items))
		copy(copied, items)
		return copied
	}
	copied := make([]taskdomain.TaskLog, limit)
	copy(copied, items[len(items)-limit:])
	return copied
}

func tailArtifacts(items []taskdomain.TaskArtifact, limit int) []taskdomain.TaskArtifact {
	if len(items) == 0 || limit <= 0 {
		return nil
	}
	if len(items) <= limit {
		copied := make([]taskdomain.TaskArtifact, len(items))
		copy(copied, items)
		return copied
	}
	copied := make([]taskdomain.TaskArtifact, limit)
	copy(copied, items[len(items)-limit:])
	return copied
}

func snippet(content string, maxRunes int) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ""
	}
	if maxRunes <= 0 {
		maxRunes = 220
	}
	runes := []rune(trimmed)
	if len(runes) <= maxRunes {
		return trimmed
	}
	return string(runes[:maxRunes]) + "..."
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return []string{"task"}
	}
	seen := map[string]struct{}{}
	items := make([]string, 0, len(tags))
	for _, tag := range tags {
		normalized := strings.ToLower(strings.TrimSpace(tag))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		items = append(items, normalized)
	}
	if len(items) == 0 {
		return []string{"task"}
	}
	return items
}

func cloneSummary(summary taskdomain.TaskSummary) taskdomain.TaskSummary {
	return taskdomain.TaskSummary{
		TaskID:     summary.TaskID,
		TaskType:   summary.TaskType,
		Goal:       summary.Goal,
		Result:     summary.Result,
		Status:     summary.Status,
		FinishedAt: summary.FinishedAt,
		Tags:       append([]string(nil), summary.Tags...),
	}
}

func cloneTask(task taskdomain.Task) taskdomain.Task {
	copied := task
	copied.RequestMetadata = cloneStringMap(task.RequestMetadata)
	copied.TaskSummary = cloneSummary(task.TaskSummary)
	if len(task.Logs) == 0 {
		copied.Logs = nil
	} else {
		copied.Logs = make([]taskdomain.TaskLog, len(task.Logs))
		copy(copied.Logs, task.Logs)
	}
	if len(task.Artifacts) == 0 {
		copied.Artifacts = nil
	} else {
		copied.Artifacts = make([]taskdomain.TaskArtifact, len(task.Artifacts))
		copy(copied.Artifacts, task.Artifacts)
	}
	copied.Result = taskdomain.TaskResult{
		Route:     task.Result.Route,
		Output:    task.Result.Output,
		ErrorCode: task.Result.ErrorCode,
		Metadata:  cloneStringMap(task.Result.Metadata),
	}
	return copied
}

func cloneStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(source))
	for key, value := range source {
		out[key] = value
	}
	return out
}

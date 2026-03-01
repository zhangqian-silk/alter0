package riskbench

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const benchmarkVersion = "2026.03-n17"

var defaultRequiredCategories = []string{"provider_policy", "supply_chain"}

type Config struct {
	Now                time.Time
	MaxStaleHours      int
	RequiredCategories []string
	FailOnOverdue      bool
}

type Report struct {
	BenchmarkVersion string                     `json:"benchmark_version"`
	GeneratedAt      string                     `json:"generated_at"`
	WatchlistPath    string                     `json:"watchlist_path"`
	RunbookPath      string                     `json:"runbook_path"`
	Summary          Summary                    `json:"summary"`
	Categories       map[string]CategorySummary `json:"categories"`
	Drifts           []DriftEvent               `json:"drifts,omitempty"`
	Gate             GateResult                 `json:"gate"`
	NextSteps        []string                   `json:"next_steps"`
}

type Summary struct {
	Status            string   `json:"status"`
	UpdatedAt         string   `json:"updated_at,omitempty"`
	AgeHours          int      `json:"age_hours,omitempty"`
	TotalItems        int      `json:"total_items"`
	ActiveItems       int      `json:"active_items"`
	OverdueItems      int      `json:"overdue_items"`
	MissingCategories []string `json:"missing_categories,omitempty"`
}

type CategorySummary struct {
	TotalItems       int    `json:"total_items"`
	ActiveItems      int    `json:"active_items"`
	OverdueItems     int    `json:"overdue_items"`
	HighestSeverity  string `json:"highest_severity"`
	UpcomingReviewAt string `json:"upcoming_review_at,omitempty"`
}

type DriftEvent struct {
	ItemID        string `json:"item_id"`
	Category      string `json:"category"`
	Severity      string `json:"severity"`
	DriftLevel    string `json:"drift_level"`
	OverdueHours  int    `json:"overdue_hours"`
	NextReviewAt  string `json:"next_review_at,omitempty"`
	Owner         string `json:"owner,omitempty"`
	TriageChannel string `json:"triage_channel"`
}

type GateResult struct {
	Passed   bool     `json:"passed"`
	Failures []string `json:"failures,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

type watchlistDocument struct {
	UpdatedAt string          `json:"updated_at"`
	Items     []watchlistItem `json:"items"`
}

type watchlistItem struct {
	ID           string `json:"id"`
	Category     string `json:"category"`
	Severity     string `json:"severity"`
	Status       string `json:"status"`
	LastChecked  string `json:"last_checked_at"`
	NextReviewAt string `json:"next_review_at"`
	Owner        string `json:"owner"`
	Notes        string `json:"notes"`
}

func EvaluatePaths(watchlistPath string, runbookPath string, cfg Config) (Report, error) {
	normalized := normalizeConfig(cfg)
	now := normalized.Now.UTC()
	resolvedWatchlist := filepath.Clean(strings.TrimSpace(watchlistPath))
	resolvedRunbook := filepath.Clean(strings.TrimSpace(runbookPath))
	if resolvedWatchlist == "" {
		return Report{}, fmt.Errorf("watchlist path is required")
	}
	if resolvedRunbook == "" {
		return Report{}, fmt.Errorf("runbook path is required")
	}

	payload, err := os.ReadFile(resolvedWatchlist)
	if err != nil {
		return Report{}, fmt.Errorf("read watchlist: %w", err)
	}

	var doc watchlistDocument
	if err := json.Unmarshal(payload, &doc); err != nil {
		return Report{}, fmt.Errorf("decode watchlist: %w", err)
	}

	report := Report{
		BenchmarkVersion: benchmarkVersion,
		GeneratedAt:      now.Format(time.RFC3339),
		WatchlistPath:    resolvedWatchlist,
		RunbookPath:      resolvedRunbook,
		Categories:       map[string]CategorySummary{},
		Drifts:           make([]DriftEvent, 0),
		Gate: GateResult{
			Failures: make([]string, 0),
			Warnings: make([]string, 0),
		},
		NextSteps: []string{},
	}

	if _, err := os.Stat(resolvedRunbook); err != nil {
		report.Gate.Failures = append(report.Gate.Failures, fmt.Sprintf("drift triage runbook is unavailable: %s", resolvedRunbook))
	}

	report.Summary.TotalItems = len(doc.Items)
	if updatedAt, ok := parseRFC3339(doc.UpdatedAt); ok {
		report.Summary.UpdatedAt = updatedAt.Format(time.RFC3339)
		report.Summary.AgeHours = int(now.Sub(updatedAt).Hours())
		if report.Summary.AgeHours > normalized.MaxStaleHours {
			report.Gate.Failures = append(report.Gate.Failures, fmt.Sprintf("watchlist is stale: age=%dh max=%dh", report.Summary.AgeHours, normalized.MaxStaleHours))
		}
	} else {
		report.Gate.Failures = append(report.Gate.Failures, "watchlist updated_at is missing or invalid")
	}

	for _, item := range doc.Items {
		category := normalizeCategory(item.Category)
		entry := report.Categories[category]
		entry.TotalItems++

		severity := normalizeSeverity(item.Severity)
		if severityRank(severity) > severityRank(entry.HighestSeverity) {
			entry.HighestSeverity = severity
		}

		if !isClosedStatus(item.Status) {
			entry.ActiveItems++
			report.Summary.ActiveItems++
			nextReview, ok := parseRFC3339(item.NextReviewAt)
			if !ok {
				report.Gate.Warnings = append(report.Gate.Warnings, fmt.Sprintf("item %s missing valid next_review_at", displayItemID(item.ID)))
			} else {
				if entry.UpcomingReviewAt == "" || nextReview.Format(time.RFC3339) < entry.UpcomingReviewAt {
					entry.UpcomingReviewAt = nextReview.Format(time.RFC3339)
				}
				if nextReview.Before(now) {
					overdueHours := int(now.Sub(nextReview).Hours())
					report.Summary.OverdueItems++
					entry.OverdueItems++
					report.Drifts = append(report.Drifts, DriftEvent{
						ItemID:        displayItemID(item.ID),
						Category:      category,
						Severity:      severity,
						DriftLevel:    classifyDriftLevel(severity, overdueHours),
						OverdueHours:  overdueHours,
						NextReviewAt:  nextReview.Format(time.RFC3339),
						Owner:         strings.TrimSpace(item.Owner),
						TriageChannel: triageChannelForCategory(category),
					})
				}
			}
		}

		if entry.HighestSeverity == "" {
			entry.HighestSeverity = "unknown"
		}
		report.Categories[category] = entry
	}

	missing := missingCategories(report.Categories, normalized.RequiredCategories)
	if len(missing) > 0 {
		report.Summary.MissingCategories = missing
		report.Gate.Failures = append(report.Gate.Failures, fmt.Sprintf("missing required risk categories: %s", strings.Join(missing, ", ")))
	}
	if normalized.FailOnOverdue && report.Summary.OverdueItems > 0 {
		report.Gate.Failures = append(report.Gate.Failures, fmt.Sprintf("overdue risk items detected: %d", report.Summary.OverdueItems))
	}

	sort.Slice(report.Drifts, func(i, j int) bool {
		left := report.Drifts[i]
		right := report.Drifts[j]
		if left.DriftLevel == right.DriftLevel {
			return left.ItemID < right.ItemID
		}
		return severityRank(left.DriftLevel) > severityRank(right.DriftLevel)
	})

	report.NextSteps = buildNextSteps(report)
	report.Gate.Passed = len(report.Gate.Failures) == 0
	if report.Gate.Passed {
		report.Summary.Status = "pass"
	} else {
		report.Summary.Status = "fail"
	}
	return report, nil
}

func normalizeConfig(cfg Config) Config {
	normalized := cfg
	if normalized.Now.IsZero() {
		normalized.Now = time.Now().UTC()
	}
	if normalized.MaxStaleHours <= 0 {
		normalized.MaxStaleHours = 96
	}
	if len(normalized.RequiredCategories) == 0 {
		normalized.RequiredCategories = append([]string{}, defaultRequiredCategories...)
	}
	return normalized
}

func missingCategories(categories map[string]CategorySummary, required []string) []string {
	missing := make([]string, 0)
	for _, raw := range required {
		key := normalizeCategory(raw)
		if summary, ok := categories[key]; !ok || summary.TotalItems == 0 {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	return missing
}

func classifyDriftLevel(severity string, overdueHours int) string {
	switch {
	case severityRank(severity) >= severityRank("high") || overdueHours >= 72:
		return "critical"
	case severityRank(severity) >= severityRank("medium") || overdueHours >= 24:
		return "high"
	default:
		return "medium"
	}
}

func triageChannelForCategory(category string) string {
	switch category {
	case "provider_policy":
		return "provider-oncall"
	case "supply_chain":
		return "security-oncall"
	default:
		return "runtime-oncall"
	}
}

func buildNextSteps(report Report) []string {
	steps := []string{
		fmt.Sprintf("Follow runbook %s: detect -> classify -> notify -> recover -> postmortem.", report.RunbookPath),
	}
	if len(report.Drifts) == 0 {
		steps = append(steps, "No active drift. Keep periodic review cadence for provider_policy and supply_chain watchlist items.")
		return steps
	}

	highest := "medium"
	for _, item := range report.Drifts {
		if severityRank(item.DriftLevel) > severityRank(highest) {
			highest = item.DriftLevel
		}
	}

	switch highest {
	case "critical":
		steps = append(steps, "Execute L3 triage: notify on-call within 15 minutes and freeze impacted provider/channel or plugin source.")
	case "high":
		steps = append(steps, "Execute L2 triage: notify owners within 1 hour and start mitigation plan.")
	default:
		steps = append(steps, "Execute L1 triage: owner review within 24 hours and confirm recovery window.")
	}
	return steps
}

func parseRFC3339(raw string) (time.Time, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, false
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed.UTC(), true
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC(), true
	}
	return time.Time{}, false
}

func normalizeCategory(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return "uncategorized"
	}
	return value
}

func normalizeSeverity(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "critical", "high", "medium", "low":
		return value
	default:
		return "unknown"
	}
}

func severityRank(raw string) int {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low", "medium-low":
		return 1
	default:
		return 0
	}
}

func isClosedStatus(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "closed", "mitigated", "resolved":
		return true
	default:
		return false
	}
}

func displayItemID(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "(unnamed)"
	}
	return value
}

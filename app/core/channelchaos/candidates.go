package channelchaos

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type CandidateOptions struct {
	Now                  time.Time
	Window               time.Duration
	MinErrorEvents       int
	MaxCandidates        int
	MaxEventsPerScenario int
}

type CandidateReport struct {
	GeneratedAt       string     `json:"generated_at"`
	TraceBasePath     string     `json:"trace_base_path"`
	WindowMinutes     int        `json:"window_minutes"`
	TotalTraceEvents  int        `json:"total_trace_events"`
	EvaluatedChannels int        `json:"evaluated_channels"`
	CandidateCount    int        `json:"candidate_count"`
	Candidates        []Scenario `json:"candidates"`
	Notes             []string   `json:"notes,omitempty"`
}

type candidateTraceEntry struct {
	Timestamp time.Time
	ChannelID string
	Event     string
	Status    string
	Detail    string
}

type candidateChannelSummary struct {
	ChannelID     string
	TotalEvents   int
	ErrorEvents   int
	Disconnected  int
	LatestEventAt time.Time
	Events        []candidateTraceEntry
}

func BuildCandidatesFromTrace(basePath string, options CandidateOptions) (CandidateReport, error) {
	now := options.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	window := options.Window
	if window <= 0 {
		window = 6 * time.Hour
	}
	minErrorEvents := options.MinErrorEvents
	if minErrorEvents <= 0 {
		minErrorEvents = 2
	}
	maxCandidates := options.MaxCandidates
	if maxCandidates <= 0 {
		maxCandidates = 5
	}
	maxEventsPerScenario := options.MaxEventsPerScenario
	if maxEventsPerScenario <= 0 {
		maxEventsPerScenario = 8
	}

	tracePath := strings.TrimSpace(basePath)
	if tracePath == "" {
		tracePath = filepath.Join("output", "trace")
	}
	if abs, err := filepath.Abs(tracePath); err == nil {
		tracePath = abs
	}

	report := CandidateReport{
		GeneratedAt:   now.Format(time.RFC3339),
		TraceBasePath: tracePath,
		WindowMinutes: int(window / time.Minute),
		Candidates:    []Scenario{},
		Notes:         []string{},
	}

	since := now.Add(-window)
	entries, err := readCandidateTraceSince(tracePath, since)
	if err != nil {
		return report, err
	}
	report.TotalTraceEvents = len(entries)
	if len(entries) == 0 {
		report.Notes = append(report.Notes, "no trace events found in selected window")
		return report, nil
	}

	channelSummary := map[string]*candidateChannelSummary{}
	for _, entry := range entries {
		channelID := strings.ToLower(strings.TrimSpace(entry.ChannelID))
		if channelID == "" {
			continue
		}
		summary, ok := channelSummary[channelID]
		if !ok {
			summary = &candidateChannelSummary{ChannelID: channelID, Events: []candidateTraceEntry{}}
			channelSummary[channelID] = summary
		}
		summary.TotalEvents++
		if strings.EqualFold(entry.Status, "error") {
			summary.ErrorEvents++
			if strings.TrimSpace(entry.Event) == "channel_disconnected" {
				summary.Disconnected++
			}
		}
		if summary.LatestEventAt.IsZero() || entry.Timestamp.After(summary.LatestEventAt) {
			summary.LatestEventAt = entry.Timestamp
		}
		summary.Events = append(summary.Events, entry)
	}

	if len(channelSummary) == 0 {
		report.Notes = append(report.Notes, "trace events are missing channel_id values")
		return report, nil
	}
	report.EvaluatedChannels = len(channelSummary)

	healthy := make([]candidateChannelSummary, 0)
	degraded := make([]candidateChannelSummary, 0)
	for _, summary := range channelSummary {
		snapshot := *summary
		if snapshot.ErrorEvents == 0 && snapshot.Disconnected == 0 {
			healthy = append(healthy, snapshot)
			continue
		}
		if snapshot.Disconnected > 0 || snapshot.ErrorEvents >= minErrorEvents {
			degraded = append(degraded, snapshot)
		}
	}

	sort.Slice(healthy, func(i, j int) bool {
		if healthy[i].LatestEventAt.Equal(healthy[j].LatestEventAt) {
			return healthy[i].ChannelID < healthy[j].ChannelID
		}
		return healthy[i].LatestEventAt.After(healthy[j].LatestEventAt)
	})
	sort.Slice(degraded, func(i, j int) bool {
		if degraded[i].Disconnected != degraded[j].Disconnected {
			return degraded[i].Disconnected > degraded[j].Disconnected
		}
		if degraded[i].ErrorEvents != degraded[j].ErrorEvents {
			return degraded[i].ErrorEvents > degraded[j].ErrorEvents
		}
		if degraded[i].LatestEventAt.Equal(degraded[j].LatestEventAt) {
			return degraded[i].ChannelID < degraded[j].ChannelID
		}
		return degraded[i].LatestEventAt.After(degraded[j].LatestEventAt)
	})

	for idx, summary := range degraded {
		if len(report.Candidates) >= maxCandidates {
			break
		}

		events := selectCandidateEvents(summary.Events, maxEventsPerScenario)
		scenarioEvents := make([]TraceEvent, 0, len(events)+1)
		for _, event := range events {
			scenarioEvents = append(scenarioEvents, TraceEvent{
				OffsetSeconds: int(event.Timestamp.Sub(now).Seconds()),
				ChannelID:     event.ChannelID,
				Event:         event.Event,
				Status:        event.Status,
				Detail:        event.Detail,
			})
		}

		fallbackCandidates := 0
		if fallback := findFallbackHealthyEvent(healthy, summary.ChannelID); fallback != nil {
			fallbackCandidates = 1
			scenarioEvents = append(scenarioEvents, TraceEvent{
				OffsetSeconds: int(fallback.Timestamp.Sub(now).Seconds()),
				ChannelID:     fallback.ChannelID,
				Event:         fallback.Event,
				Status:        fallback.Status,
				Detail:        fallback.Detail,
			})
		}
		sort.Slice(scenarioEvents, func(i, j int) bool {
			return scenarioEvents[i].OffsetSeconds < scenarioEvents[j].OffsetSeconds
		})

		status := "degraded"
		if summary.Disconnected > 0 || summary.ErrorEvents >= 3 {
			status = "critical"
		}

		minDegraded := 1
		minFallback := fallbackCandidates
		expect := Expectation{
			Status:                status,
			MinDegradedChannels:   &minDegraded,
			MinFallbackCandidates: &minFallback,
			AlertCodes:            []string{"channel_degradation"},
		}
		if summary.Disconnected > 0 {
			expect.AlertCodes = append(expect.AlertCodes, "channel_disconnected")
		}
		if fallbackCandidates == 0 {
			expect.ReasonContains = "no healthy fallback"
		}

		scenarioID := fmt.Sprintf("trace-sample-%s-%s-%02d", sanitizeCandidateID(summary.ChannelID), now.Format("20060102t150405z"), idx+1)
		report.Candidates = append(report.Candidates, Scenario{
			ID:            scenarioID,
			Description:   fmt.Sprintf("auto-sampled from trace window: channel=%s errors=%d disconnected=%d", summary.ChannelID, summary.ErrorEvents, summary.Disconnected),
			WindowMinutes: report.WindowMinutes,
			Events:        scenarioEvents,
			Expect:        expect,
		})
	}

	report.CandidateCount = len(report.Candidates)
	if report.CandidateCount == 0 {
		report.Notes = append(report.Notes, fmt.Sprintf("no channels matched min_error_events=%d in selected window", minErrorEvents))
	}
	return report, nil
}

func selectCandidateEvents(events []candidateTraceEntry, maxEvents int) []candidateTraceEntry {
	if len(events) == 0 {
		return []candidateTraceEntry{}
	}
	sortedEvents := append([]candidateTraceEntry{}, events...)
	sort.Slice(sortedEvents, func(i, j int) bool {
		return sortedEvents[i].Timestamp.Before(sortedEvents[j].Timestamp)
	})
	if maxEvents > 0 && len(sortedEvents) > maxEvents {
		sortedEvents = sortedEvents[len(sortedEvents)-maxEvents:]
	}
	return sortedEvents
}

func findFallbackHealthyEvent(healthy []candidateChannelSummary, degradedChannel string) *candidateTraceEntry {
	for _, channel := range healthy {
		if channel.ChannelID == degradedChannel {
			continue
		}
		events := selectCandidateEvents(channel.Events, 1)
		if len(events) == 0 {
			continue
		}
		event := events[0]
		if strings.EqualFold(event.Status, "error") {
			continue
		}
		return &event
	}
	return nil
}

func sanitizeCandidateID(channelID string) string {
	trimmed := strings.ToLower(strings.TrimSpace(channelID))
	if trimmed == "" {
		return "unknown"
	}
	builder := strings.Builder{}
	for _, ch := range trimmed {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			builder.WriteRune(ch)
			continue
		}
		builder.WriteRune('-')
	}
	id := strings.Trim(builder.String(), "-")
	if id == "" {
		return "unknown"
	}
	for strings.Contains(id, "--") {
		id = strings.ReplaceAll(id, "--", "-")
	}
	return id
}

func readCandidateTraceSince(basePath string, since time.Time) ([]candidateTraceEntry, error) {
	dirs, err := os.ReadDir(basePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []candidateTraceEntry{}, nil
		}
		return nil, err
	}

	dayDirs := make([]string, 0, len(dirs))
	for _, entry := range dirs {
		if !entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if _, err := time.Parse("2006-01-02", name); err != nil {
			continue
		}
		dayDirs = append(dayDirs, name)
	}
	sort.Strings(dayDirs)

	entries := make([]candidateTraceEntry, 0)
	for _, day := range dayDirs {
		dayAt, err := time.Parse("2006-01-02", day)
		if err != nil {
			continue
		}
		if dayAt.Add(24 * time.Hour).Before(since) {
			continue
		}
		items, err := readCandidateTraceFile(filepath.Join(basePath, day, "gateway_events.jsonl"), since)
		if err != nil {
			continue
		}
		entries = append(entries, items...)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})
	return entries, nil
}

func readCandidateTraceFile(path string, since time.Time) ([]candidateTraceEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	type rawTrace struct {
		Timestamp string `json:"timestamp"`
		ChannelID string `json:"channel_id"`
		Event     string `json:"event"`
		Status    string `json:"status"`
		Detail    string `json:"detail"`
	}

	scanner := bufio.NewScanner(f)
	out := make([]candidateTraceEntry, 0, 32)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var item rawTrace
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			continue
		}
		ts, err := parseCandidateTimestamp(item.Timestamp)
		if err != nil || ts.Before(since) {
			continue
		}
		out = append(out, candidateTraceEntry{
			Timestamp: ts,
			ChannelID: strings.ToLower(strings.TrimSpace(item.ChannelID)),
			Event:     strings.TrimSpace(item.Event),
			Status:    strings.TrimSpace(item.Status),
			Detail:    strings.TrimSpace(item.Detail),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func parseCandidateTimestamp(raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, fmt.Errorf("timestamp is required")
	}
	if ts, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return ts.UTC(), nil
	}
	ts, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, err
	}
	return ts.UTC(), nil
}

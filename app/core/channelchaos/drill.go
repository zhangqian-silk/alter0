package channelchaos

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"alter0/app/core/runtime"
)

type Matrix struct {
	UpdatedAt string     `json:"updated_at,omitempty"`
	Scenarios []Scenario `json:"scenarios"`
}

type Scenario struct {
	ID            string       `json:"id"`
	Description   string       `json:"description,omitempty"`
	WindowMinutes int          `json:"window_minutes,omitempty"`
	Events        []TraceEvent `json:"events"`
	Expect        Expectation  `json:"expect"`
}

type TraceEvent struct {
	OffsetSeconds int    `json:"offset_seconds,omitempty"`
	ChannelID     string `json:"channel_id"`
	Event         string `json:"event"`
	Status        string `json:"status"`
	Detail        string `json:"detail,omitempty"`
}

type Expectation struct {
	Status                string   `json:"status,omitempty"`
	MinDegradedChannels   *int     `json:"min_degraded_channels,omitempty"`
	MaxDegradedChannels   *int     `json:"max_degraded_channels,omitempty"`
	MinFallbackCandidates *int     `json:"min_fallback_candidates,omitempty"`
	ReasonContains        string   `json:"reason_contains,omitempty"`
	AlertCodes            []string `json:"alert_codes,omitempty"`
}

type Report struct {
	GeneratedAt   string           `json:"generated_at"`
	MatrixUpdated string           `json:"matrix_updated_at,omitempty"`
	Passed        bool             `json:"passed"`
	ScenarioCount int              `json:"scenario_count"`
	FailedCount   int              `json:"failed_count"`
	Results       []ScenarioResult `json:"results"`
}

type ScenarioResult struct {
	ID          string   `json:"id"`
	Description string   `json:"description,omitempty"`
	Passed      bool     `json:"passed"`
	Failure     string   `json:"failure,omitempty"`
	Observed    Observed `json:"observed"`
}

type Observed struct {
	Status             string   `json:"status"`
	DegradedChannels   int      `json:"degraded_channels"`
	CriticalChannels   int      `json:"critical_channels"`
	FallbackCandidates []string `json:"fallback_candidates"`
	Reason             string   `json:"reason,omitempty"`
	AlertCodes         []string `json:"alert_codes"`
}

type snapshotPayload struct {
	ChannelDegradation struct {
		Status             string   `json:"status"`
		DegradedChannels   int      `json:"degraded_channels"`
		CriticalChannels   int      `json:"critical_channels"`
		FallbackCandidates []string `json:"fallback_candidates"`
		Reason             string   `json:"reason"`
	} `json:"channel_degradation"`
	Alerts []struct {
		Code string `json:"code"`
	} `json:"alerts"`
}

func LoadMatrix(path string) (Matrix, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return Matrix{}, fmt.Errorf("matrix path is required")
	}

	data, err := os.ReadFile(trimmed)
	if err != nil {
		return Matrix{}, err
	}

	var matrix Matrix
	if err := json.Unmarshal(data, &matrix); err != nil {
		return Matrix{}, err
	}
	if len(matrix.Scenarios) == 0 {
		return Matrix{}, fmt.Errorf("matrix has no scenarios")
	}
	return matrix, nil
}

func Run(ctx context.Context, matrix Matrix) Report {
	report := Report{
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		MatrixUpdated: strings.TrimSpace(matrix.UpdatedAt),
		Passed:        true,
		ScenarioCount: len(matrix.Scenarios),
		FailedCount:   0,
		Results:       make([]ScenarioResult, 0, len(matrix.Scenarios)),
	}

	for _, scenario := range matrix.Scenarios {
		result := runScenario(ctx, scenario)
		if !result.Passed {
			report.Passed = false
			report.FailedCount++
		}
		report.Results = append(report.Results, result)
	}

	return report
}

func runScenario(ctx context.Context, scenario Scenario) ScenarioResult {
	result := ScenarioResult{
		ID:          strings.TrimSpace(scenario.ID),
		Description: strings.TrimSpace(scenario.Description),
		Passed:      false,
		Observed: Observed{
			FallbackCandidates: []string{},
			AlertCodes:         []string{},
		},
	}

	if result.ID == "" {
		result.Failure = "scenario id is required"
		return result
	}
	if len(scenario.Events) == 0 {
		result.Failure = "scenario events are required"
		return result
	}

	basePath, err := writeScenarioTrace(scenario)
	if err != nil {
		result.Failure = fmt.Sprintf("write trace failed: %v", err)
		return result
	}
	defer os.RemoveAll(basePath)

	window := 30 * time.Minute
	if scenario.WindowMinutes > 0 {
		window = time.Duration(scenario.WindowMinutes) * time.Minute
	}
	collector := &runtime.StatusCollector{
		GatewayTraceBasePath: basePath,
		GatewayTraceWindow:   window,
	}
	observed, err := collectObserved(ctx, collector)
	if err != nil {
		result.Failure = fmt.Sprintf("collect snapshot failed: %v", err)
		return result
	}
	result.Observed = observed

	if err := evaluateExpectation(scenario.Expect, observed); err != nil {
		result.Failure = err.Error()
		return result
	}

	result.Passed = true
	return result
}

func writeScenarioTrace(scenario Scenario) (string, error) {
	basePath, err := os.MkdirTemp("", "alter0-channel-chaos-")
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	dayDir := filepath.Join(basePath, now.Format("2006-01-02"))
	if err := os.MkdirAll(dayDir, 0755); err != nil {
		return "", err
	}

	lines := make([]string, 0, len(scenario.Events))
	for idx, item := range scenario.Events {
		channelID := strings.TrimSpace(item.ChannelID)
		event := strings.TrimSpace(item.Event)
		status := strings.TrimSpace(item.Status)
		if channelID == "" || event == "" || status == "" {
			return "", fmt.Errorf("event[%d] requires channel_id/event/status", idx)
		}
		record := map[string]interface{}{
			"timestamp":  now.Add(time.Duration(item.OffsetSeconds) * time.Second).Format(time.RFC3339Nano),
			"channel_id": channelID,
			"event":      event,
			"status":     status,
		}
		if detail := strings.TrimSpace(item.Detail); detail != "" {
			record["detail"] = detail
		}
		payload, err := json.Marshal(record)
		if err != nil {
			return "", err
		}
		lines = append(lines, string(payload))
	}

	tracePath := filepath.Join(dayDir, "gateway_events.jsonl")
	if err := os.WriteFile(tracePath, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		return "", err
	}
	return basePath, nil
}

func collectObserved(ctx context.Context, collector *runtime.StatusCollector) (Observed, error) {
	snapshot := collector.Snapshot(ctx)
	payloadBytes, err := json.Marshal(snapshot)
	if err != nil {
		return Observed{}, err
	}

	var payload snapshotPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return Observed{}, err
	}

	alertCodes := make([]string, 0, len(payload.Alerts))
	seen := map[string]struct{}{}
	for _, item := range payload.Alerts {
		code := strings.TrimSpace(item.Code)
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		alertCodes = append(alertCodes, code)
	}
	sort.Strings(alertCodes)

	fallback := append([]string{}, payload.ChannelDegradation.FallbackCandidates...)
	sort.Strings(fallback)

	return Observed{
		Status:             strings.TrimSpace(payload.ChannelDegradation.Status),
		DegradedChannels:   payload.ChannelDegradation.DegradedChannels,
		CriticalChannels:   payload.ChannelDegradation.CriticalChannels,
		FallbackCandidates: fallback,
		Reason:             strings.TrimSpace(payload.ChannelDegradation.Reason),
		AlertCodes:         alertCodes,
	}, nil
}

func evaluateExpectation(expect Expectation, observed Observed) error {
	if status := strings.TrimSpace(expect.Status); status != "" && observed.Status != status {
		return fmt.Errorf("expected status %q, got %q", status, observed.Status)
	}
	if expect.MinDegradedChannels != nil && observed.DegradedChannels < *expect.MinDegradedChannels {
		return fmt.Errorf("expected degraded_channels >= %d, got %d", *expect.MinDegradedChannels, observed.DegradedChannels)
	}
	if expect.MaxDegradedChannels != nil && observed.DegradedChannels > *expect.MaxDegradedChannels {
		return fmt.Errorf("expected degraded_channels <= %d, got %d", *expect.MaxDegradedChannels, observed.DegradedChannels)
	}
	if expect.MinFallbackCandidates != nil && len(observed.FallbackCandidates) < *expect.MinFallbackCandidates {
		return fmt.Errorf("expected fallback_candidates >= %d, got %d", *expect.MinFallbackCandidates, len(observed.FallbackCandidates))
	}
	if reason := strings.TrimSpace(expect.ReasonContains); reason != "" && !strings.Contains(strings.ToLower(observed.Reason), strings.ToLower(reason)) {
		return fmt.Errorf("expected reason to contain %q, got %q", reason, observed.Reason)
	}

	if len(expect.AlertCodes) > 0 {
		missing := make([]string, 0)
		present := map[string]struct{}{}
		for _, code := range observed.AlertCodes {
			present[code] = struct{}{}
		}
		for _, required := range expect.AlertCodes {
			code := strings.TrimSpace(required)
			if code == "" {
				continue
			}
			if _, ok := present[code]; !ok {
				missing = append(missing, code)
			}
		}
		if len(missing) > 0 {
			sort.Strings(missing)
			return fmt.Errorf("expected alerts missing: %s", strings.Join(missing, ", "))
		}
	}
	return nil
}

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	config "alter0/app/configs"
)

const reconcileVersion = "2026.03-n25"

type thresholdHistoryDocument struct {
	GeneratedAt       string                 `json:"generated_at"`
	ThresholdGuidance map[string]interface{} `json:"threshold_guidance"`
}

type thresholdGuidance struct {
	Status            string `json:"status"`
	NeedsTuning       bool   `json:"needs_tuning"`
	RequiredMinTokens int    `json:"required_min_tokens"`
	Recommended       struct {
		SessionCostShare  float64 `json:"session_cost_share"`
		PromptOutputRatio float64 `json:"prompt_output_ratio"`
	} `json:"recommended"`
}

type thresholdValues struct {
	SessionCostShareAlertThreshold  float64 `json:"session_cost_share_alert_threshold"`
	PromptOutputRatioAlertThreshold float64 `json:"prompt_output_ratio_alert_threshold"`
	HeavySessionMinTokens           int     `json:"heavy_session_min_tokens"`
}

type reconcilePlan struct {
	Status      string          `json:"status"`
	Reason      string          `json:"reason,omitempty"`
	CappedDelta bool            `json:"capped_delta"`
	Current     thresholdValues `json:"current"`
	Recommended thresholdValues `json:"recommended"`
	Proposed    thresholdValues `json:"proposed"`
	Delta       thresholdValues `json:"delta"`
}

type reconcileReport struct {
	ReconcileVersion    string        `json:"reconcile_version"`
	GeneratedAt         string        `json:"generated_at"`
	HistoryPath         string        `json:"history_path"`
	HistoryGeneratedAt  string        `json:"history_generated_at,omitempty"`
	ConfigPath          string        `json:"config_path"`
	ApplyRequested      bool          `json:"apply_requested"`
	Applied             bool          `json:"applied"`
	ApplyError          string        `json:"apply_error,omitempty"`
	GuidanceStatus      string        `json:"guidance_status"`
	GuidanceNeedsTuning bool          `json:"guidance_needs_tuning"`
	Plan                reconcilePlan `json:"plan"`
}

type planOptions struct {
	MaxShareStep float64
	MaxRatioStep float64
}

func main() {
	historyPath := flag.String("history", filepath.Join("output", "cost", "threshold-history-latest.json"), "path to threshold-history report")
	configPath := flag.String("config", config.DefaultPath(), "path to runtime config json")
	outputPath := flag.String("output", filepath.Join("output", "cost", "threshold-reconcile-latest.json"), "path to write reconcile report (use - for stdout)")
	apply := flag.Bool("apply", false, "persist proposed thresholds into runtime config")
	maxShareStep := flag.Float64("max-share-step", 0.10, "max absolute delta for session_cost_share_alert_threshold per run")
	maxRatioStep := flag.Float64("max-ratio-step", 2.0, "max absolute delta for prompt_output_ratio_alert_threshold per run")
	flag.Parse()

	doc, guidance, err := loadGuidance(*historyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cost threshold reconcile failed: %v\n", err)
		os.Exit(2)
	}

	cfg, err := config.LoadConfigFile(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cost threshold reconcile failed: %v\n", err)
		os.Exit(2)
	}

	current := thresholdValues{
		SessionCostShareAlertThreshold:  cfg.Runtime.Observability.Cost.SessionCostShareAlertThreshold,
		PromptOutputRatioAlertThreshold: cfg.Runtime.Observability.Cost.PromptOutputRatioAlertThreshold,
		HeavySessionMinTokens:           cfg.Runtime.Observability.Cost.HeavySessionMinTokens,
	}

	plan := buildPlan(current, guidance, planOptions{MaxShareStep: *maxShareStep, MaxRatioStep: *maxRatioStep})
	report := reconcileReport{
		ReconcileVersion:    reconcileVersion,
		GeneratedAt:         time.Now().UTC().Format(time.RFC3339),
		HistoryPath:         strings.TrimSpace(*historyPath),
		HistoryGeneratedAt:  strings.TrimSpace(doc.GeneratedAt),
		ConfigPath:          strings.TrimSpace(*configPath),
		ApplyRequested:      *apply,
		GuidanceStatus:      strings.ToLower(strings.TrimSpace(guidance.Status)),
		GuidanceNeedsTuning: guidance.NeedsTuning,
		Plan:                plan,
	}

	if *apply && plan.Status == "ready" {
		if err := applyPlan(*configPath, plan.Proposed); err != nil {
			report.ApplyError = err.Error()
			report.Plan.Status = "apply_failed"
		} else {
			report.Applied = true
			report.Plan.Status = "applied"
			report.Plan.Reason = "thresholds persisted to config"
		}
	}

	if err := writeReport(*outputPath, report); err != nil {
		fmt.Fprintf(os.Stderr, "cost threshold reconcile failed: %v\n", err)
		os.Exit(2)
	}

	fmt.Printf("cost threshold reconcile finished; status=%s apply=%t report=%s\n", report.Plan.Status, report.Applied, *outputPath)
}

func loadGuidance(path string) (thresholdHistoryDocument, thresholdGuidance, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return thresholdHistoryDocument{}, thresholdGuidance{}, fmt.Errorf("history path is required")
	}
	payload, err := os.ReadFile(trimmed)
	if err != nil {
		return thresholdHistoryDocument{}, thresholdGuidance{}, fmt.Errorf("read threshold history: %w", err)
	}
	var doc thresholdHistoryDocument
	if err := json.Unmarshal(payload, &doc); err != nil {
		return thresholdHistoryDocument{}, thresholdGuidance{}, fmt.Errorf("decode threshold history: %w", err)
	}
	if doc.ThresholdGuidance == nil {
		return thresholdHistoryDocument{}, thresholdGuidance{}, fmt.Errorf("threshold history missing threshold_guidance")
	}
	raw, err := json.Marshal(doc.ThresholdGuidance)
	if err != nil {
		return thresholdHistoryDocument{}, thresholdGuidance{}, fmt.Errorf("encode threshold guidance: %w", err)
	}
	var guidance thresholdGuidance
	if err := json.Unmarshal(raw, &guidance); err != nil {
		return thresholdHistoryDocument{}, thresholdGuidance{}, fmt.Errorf("decode threshold guidance: %w", err)
	}
	return doc, guidance, nil
}

func buildPlan(current thresholdValues, guidance thresholdGuidance, opts planOptions) reconcilePlan {
	plan := reconcilePlan{
		Status:      "skipped",
		Current:     current,
		Recommended: current,
		Proposed:    current,
	}

	status := strings.ToLower(strings.TrimSpace(guidance.Status))
	if status != "ok" {
		plan.Reason = fmt.Sprintf("threshold guidance status is %q", status)
		return plan
	}
	if guidance.Recommended.SessionCostShare <= 0 || guidance.Recommended.PromptOutputRatio <= 0 {
		plan.Reason = "threshold guidance recommended values are missing"
		return plan
	}

	if opts.MaxShareStep <= 0 {
		opts.MaxShareStep = 0.10
	}
	if opts.MaxRatioStep <= 0 {
		opts.MaxRatioStep = 2.0
	}

	recommended := thresholdValues{
		SessionCostShareAlertThreshold:  clampFloat(guidance.Recommended.SessionCostShare, 0.20, 0.90),
		PromptOutputRatioAlertThreshold: clampFloat(guidance.Recommended.PromptOutputRatio, 2.0, 20.0),
		HeavySessionMinTokens:           current.HeavySessionMinTokens,
	}
	if guidance.RequiredMinTokens > 0 {
		recommended.HeavySessionMinTokens = guidance.RequiredMinTokens
	}

	shareRawDelta := recommended.SessionCostShareAlertThreshold - current.SessionCostShareAlertThreshold
	ratioRawDelta := recommended.PromptOutputRatioAlertThreshold - current.PromptOutputRatioAlertThreshold
	shareDelta := clampDelta(shareRawDelta, opts.MaxShareStep)
	ratioDelta := clampDelta(ratioRawDelta, opts.MaxRatioStep)

	proposed := thresholdValues{
		SessionCostShareAlertThreshold:  roundFloat(current.SessionCostShareAlertThreshold+shareDelta, 3),
		PromptOutputRatioAlertThreshold: roundFloat(current.PromptOutputRatioAlertThreshold+ratioDelta, 2),
		HeavySessionMinTokens:           recommended.HeavySessionMinTokens,
	}
	delta := thresholdValues{
		SessionCostShareAlertThreshold:  roundFloat(proposed.SessionCostShareAlertThreshold-current.SessionCostShareAlertThreshold, 3),
		PromptOutputRatioAlertThreshold: roundFloat(proposed.PromptOutputRatioAlertThreshold-current.PromptOutputRatioAlertThreshold, 2),
		HeavySessionMinTokens:           proposed.HeavySessionMinTokens - current.HeavySessionMinTokens,
	}

	plan.Recommended = recommended
	plan.Proposed = proposed
	plan.Delta = delta
	plan.CappedDelta = math.Abs(shareRawDelta-shareDelta) > 0.0005 || math.Abs(ratioRawDelta-ratioDelta) > 0.005

	if delta.SessionCostShareAlertThreshold == 0 && delta.PromptOutputRatioAlertThreshold == 0 && delta.HeavySessionMinTokens == 0 {
		plan.Status = "no_change"
		plan.Reason = "current thresholds already match recommendation"
		return plan
	}
	if !guidance.NeedsTuning && delta.HeavySessionMinTokens == 0 {
		plan.Status = "no_change"
		plan.Reason = "guidance indicates tuning is not required"
		return plan
	}

	plan.Status = "ready"
	if plan.CappedDelta {
		plan.Reason = "delta capped by per-run safety limits"
	} else {
		plan.Reason = "proposal generated from threshold guidance"
	}
	return plan
}

func applyPlan(configPath string, proposed thresholdValues) error {
	trimmed := strings.TrimSpace(configPath)
	if trimmed == "" {
		return fmt.Errorf("config path is required")
	}
	mgr, err := config.NewManager(trimmed)
	if err != nil {
		return fmt.Errorf("open config manager: %w", err)
	}
	_, err = mgr.Update(func(cfg *config.Config) {
		cfg.Runtime.Observability.Cost.SessionCostShareAlertThreshold = proposed.SessionCostShareAlertThreshold
		cfg.Runtime.Observability.Cost.PromptOutputRatioAlertThreshold = proposed.PromptOutputRatioAlertThreshold
		cfg.Runtime.Observability.Cost.HeavySessionMinTokens = proposed.HeavySessionMinTokens
	})
	if err != nil {
		return fmt.Errorf("update config thresholds: %w", err)
	}
	return nil
}

func writeReport(path string, payload reconcileReport) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("output path is required")
	}
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	encoded = append(encoded, '\n')
	if path == "-" {
		_, err := fmt.Fprintln(os.Stdout, string(encoded))
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	if err := os.WriteFile(path, encoded, 0644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	return nil
}

func clampDelta(value float64, maxAbs float64) float64 {
	if maxAbs <= 0 {
		return value
	}
	if value > maxAbs {
		return maxAbs
	}
	if value < -maxAbs {
		return -maxAbs
	}
	return value
}

func clampFloat(value float64, min float64, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func roundFloat(value float64, places int) float64 {
	if places < 0 {
		return value
	}
	factor := 1.0
	for i := 0; i < places; i++ {
		factor *= 10
	}
	if value >= 0 {
		return float64(int(value*factor+0.5)) / factor
	}
	return float64(int(value*factor-0.5)) / factor
}

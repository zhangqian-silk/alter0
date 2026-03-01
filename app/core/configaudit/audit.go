package configaudit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	config "alter0/app/configs"
)

const auditVersion = "2026.03-n19"

var requiredConfirmTools = []string{"browser", "canvas", "nodes", "message"}

type Options struct {
	Now time.Time
}

type Report struct {
	AuditVersion string                  `json:"audit_version"`
	GeneratedAt  string                  `json:"generated_at"`
	ConfigPath   string                  `json:"config_path"`
	Summary      Summary                 `json:"summary"`
	Domains      map[string]DomainReport `json:"domains"`
	Parameters   []ParameterCheck        `json:"parameters"`
	Gate         GateResult              `json:"gate"`
	NextSteps    []string                `json:"next_steps"`
}

type Summary struct {
	Status              string `json:"status"`
	TotalParameters     int    `json:"total_parameters"`
	HighRiskParameters  int    `json:"high_risk_parameters"`
	DefaultedParameters int    `json:"defaulted_parameters"`
	FailedChecks        int    `json:"failed_checks"`
	WarningChecks       int    `json:"warning_checks"`
}

type DomainReport struct {
	TotalParameters    int `json:"total_parameters"`
	HighRiskParameters int `json:"high_risk_parameters"`
	FailedChecks       int `json:"failed_checks"`
	WarningChecks      int `json:"warning_checks"`
}

type ParameterCheck struct {
	Path           string `json:"path"`
	Domain         string `json:"domain"`
	Risk           string `json:"risk"`
	Description    string `json:"description"`
	Value          any    `json:"value"`
	DefaultValue   any    `json:"default_value"`
	Explicit       bool   `json:"explicit"`
	Status         string `json:"status"`
	Message        string `json:"message,omitempty"`
	Recommendation string `json:"recommendation,omitempty"`
}

type GateResult struct {
	Passed   bool     `json:"passed"`
	Failures []string `json:"failures,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

type parameterRule struct {
	Path           string
	Domain         string
	Risk           string
	Description    string
	Recommendation string
	Current        func(config.Config) any
	Default        func(config.Config) any
	Validate       func(config.Config) (string, string)
}

func EvaluatePath(path string, opts Options) (Report, error) {
	resolvedPath := filepath.Clean(strings.TrimSpace(path))
	if resolvedPath == "" {
		return Report{}, fmt.Errorf("config path is required")
	}

	rawBytes, err := os.ReadFile(resolvedPath)
	if err != nil {
		return Report{}, fmt.Errorf("read config: %w", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(rawBytes, &raw); err != nil {
		return Report{}, fmt.Errorf("decode raw config: %w", err)
	}

	var fileCfg config.Config
	if err := json.Unmarshal(rawBytes, &fileCfg); err != nil {
		return Report{}, fmt.Errorf("decode config: %w", err)
	}
	normalized := config.NormalizeConfig(fileCfg)
	defaultCfg := config.DefaultConfig()

	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	report := Report{
		AuditVersion: auditVersion,
		GeneratedAt:  now.UTC().Format(time.RFC3339),
		ConfigPath:   resolvedPath,
		Domains:      map[string]DomainReport{},
		Parameters:   make([]ParameterCheck, 0),
		Gate: GateResult{
			Failures: make([]string, 0),
			Warnings: make([]string, 0),
		},
		NextSteps: make([]string, 0),
	}

	rules := parameterRules()
	for _, rule := range rules {
		domain := report.Domains[rule.Domain]
		domain.TotalParameters++

		risk := normalizeRisk(rule.Risk)
		if riskRank(risk) >= riskRank("high") {
			report.Summary.HighRiskParameters++
			domain.HighRiskParameters++
		}

		explicit := hasPath(raw, strings.Split(rule.Path, "."))
		if !explicit {
			report.Summary.DefaultedParameters++
		}

		status, message := "pass", ""
		if rule.Validate != nil {
			status, message = normalizeStatus(rule.Validate(normalized))
		}
		entry := ParameterCheck{
			Path:         rule.Path,
			Domain:       rule.Domain,
			Risk:         risk,
			Description:  rule.Description,
			Value:        rule.Current(normalized),
			DefaultValue: rule.Default(defaultCfg),
			Explicit:     explicit,
			Status:       status,
			Message:      strings.TrimSpace(message),
		}

		switch status {
		case "fail":
			report.Summary.FailedChecks++
			domain.FailedChecks++
			report.Gate.Failures = append(report.Gate.Failures, fmt.Sprintf("%s: %s", rule.Path, entry.Message))
			entry.Recommendation = rule.Recommendation
		case "warn":
			report.Summary.WarningChecks++
			domain.WarningChecks++
			report.Gate.Warnings = append(report.Gate.Warnings, fmt.Sprintf("%s: %s", rule.Path, entry.Message))
			entry.Recommendation = rule.Recommendation
		}

		report.Parameters = append(report.Parameters, entry)
		report.Domains[rule.Domain] = domain
	}

	report.Summary.TotalParameters = len(report.Parameters)
	report.Gate.Passed = len(report.Gate.Failures) == 0
	if report.Gate.Passed {
		report.Summary.Status = "pass"
	} else {
		report.Summary.Status = "fail"
	}
	report.NextSteps = buildNextSteps(report)

	return report, nil
}

func parameterRules() []parameterRule {
	return []parameterRule{
		{
			Path:           "agent.default_id",
			Domain:         "agents",
			Risk:           "high",
			Description:    "Primary default agent id used by runtime routing.",
			Recommendation: "Set agent.default_id to an existing entry in agent.registry.",
			Current: func(cfg config.Config) any {
				return cfg.Agent.DefaultID
			},
			Default: func(cfg config.Config) any {
				return cfg.Agent.DefaultID
			},
			Validate: func(cfg config.Config) (string, string) {
				defaultID := strings.TrimSpace(cfg.Agent.DefaultID)
				if defaultID == "" {
					return "fail", "default_id is empty"
				}
				for _, entry := range cfg.Agent.Registry {
					if entry.ID == defaultID {
						return "pass", ""
					}
				}
				return "fail", "default_id does not exist in agent.registry"
			},
		},
		{
			Path:           "agent.registry",
			Domain:         "agents",
			Risk:           "high",
			Description:    "Agent registry controls per-agent workspace and executor isolation.",
			Recommendation: "Keep at least one registry entry and avoid duplicate ids.",
			Current: func(cfg config.Config) any {
				return len(cfg.Agent.Registry)
			},
			Default: func(cfg config.Config) any {
				return len(cfg.Agent.Registry)
			},
			Validate: func(cfg config.Config) (string, string) {
				if len(cfg.Agent.Registry) == 0 {
					return "fail", "registry has no entries"
				}
				seen := map[string]struct{}{}
				for _, entry := range cfg.Agent.Registry {
					if _, ok := seen[entry.ID]; ok {
						return "fail", "registry contains duplicate agent ids"
					}
					seen[entry.ID] = struct{}{}
				}
				return "pass", ""
			},
		},
		{
			Path:           "executor.name",
			Domain:         "agents",
			Risk:           "medium",
			Description:    "Default executor backend for agent commands.",
			Recommendation: "Use a supported executor name (codex/claude_code/opencode) to reduce runtime drift.",
			Current: func(cfg config.Config) any {
				return cfg.Executor.Name
			},
			Default: func(cfg config.Config) any {
				return cfg.Executor.Name
			},
			Validate: func(cfg config.Config) (string, string) {
				switch strings.TrimSpace(cfg.Executor.Name) {
				case "codex", "claude_code", "opencode":
					return "pass", ""
				default:
					return "warn", "executor is not in the curated baseline set"
				}
			},
		},
		{
			Path:           "channels.telegram.enabled",
			Domain:         "bindings",
			Risk:           "high",
			Description:    "Telegram binding enable switch.",
			Recommendation: "When enabling Telegram, configure bot_token and default_chat_id to avoid runtime delivery gaps.",
			Current: func(cfg config.Config) any {
				return cfg.Channels.Telegram.Enabled
			},
			Default: func(cfg config.Config) any {
				return cfg.Channels.Telegram.Enabled
			},
			Validate: func(cfg config.Config) (string, string) {
				if !cfg.Channels.Telegram.Enabled {
					return "pass", ""
				}
				if strings.TrimSpace(cfg.Channels.Telegram.BotToken) == "" {
					return "fail", "telegram enabled but bot_token is empty"
				}
				if strings.TrimSpace(cfg.Channels.Telegram.DefaultChatID) == "" {
					return "warn", "telegram enabled without default_chat_id"
				}
				return "pass", ""
			},
		},
		{
			Path:           "channels.slack.enabled",
			Domain:         "bindings",
			Risk:           "high",
			Description:    "Slack binding enable switch.",
			Recommendation: "When enabling Slack, configure bot_token and app_id to avoid handshake failures.",
			Current: func(cfg config.Config) any {
				return cfg.Channels.Slack.Enabled
			},
			Default: func(cfg config.Config) any {
				return cfg.Channels.Slack.Enabled
			},
			Validate: func(cfg config.Config) (string, string) {
				if !cfg.Channels.Slack.Enabled {
					return "pass", ""
				}
				if strings.TrimSpace(cfg.Channels.Slack.BotToken) == "" {
					return "fail", "slack enabled but bot_token is empty"
				}
				if strings.TrimSpace(cfg.Channels.Slack.AppID) == "" {
					return "fail", "slack enabled but app_id is empty"
				}
				return "pass", ""
			},
		},
		{
			Path:           "task.routing_confidence_threshold",
			Domain:         "session",
			Risk:           "high",
			Description:    "Router confidence threshold for task dispatch.",
			Recommendation: "Keep routing_confidence_threshold in [0.45, 0.85] unless benchmark evidence justifies drift.",
			Current: func(cfg config.Config) any {
				return cfg.Task.RoutingConfidenceThreshold
			},
			Default: func(cfg config.Config) any {
				return cfg.Task.RoutingConfidenceThreshold
			},
			Validate: func(cfg config.Config) (string, string) {
				v := cfg.Task.RoutingConfidenceThreshold
				if v <= 0 || v >= 1 {
					return "fail", "threshold must be between 0 and 1"
				}
				if v < 0.45 || v > 0.85 {
					return "warn", "threshold is outside recommended operating window"
				}
				return "pass", ""
			},
		},
		{
			Path:           "task.close_confidence_threshold",
			Domain:         "session",
			Risk:           "medium",
			Description:    "Closer confidence threshold for auto-close decisions.",
			Recommendation: "Keep close_confidence_threshold in [0.55, 0.90] for safe closure behavior.",
			Current: func(cfg config.Config) any {
				return cfg.Task.CloseConfidenceThreshold
			},
			Default: func(cfg config.Config) any {
				return cfg.Task.CloseConfidenceThreshold
			},
			Validate: func(cfg config.Config) (string, string) {
				v := cfg.Task.CloseConfidenceThreshold
				if v <= 0 || v >= 1 {
					return "fail", "threshold must be between 0 and 1"
				}
				if v < 0.55 || v > 0.90 {
					return "warn", "threshold is outside recommended operating window"
				}
				return "pass", ""
			},
		},
		{
			Path:           "task.generation_timeout_sec",
			Domain:         "session",
			Risk:           "high",
			Description:    "Upper bound for generation turn execution time.",
			Recommendation: "Keep generation_timeout_sec in [30, 300] for predictable queue latency.",
			Current: func(cfg config.Config) any {
				return cfg.Task.GenerationTimeoutSec
			},
			Default: func(cfg config.Config) any {
				return cfg.Task.GenerationTimeoutSec
			},
			Validate: func(cfg config.Config) (string, string) {
				v := cfg.Task.GenerationTimeoutSec
				if v < 30 {
					return "fail", "generation timeout is too small (<30s)"
				}
				if v > 1800 {
					return "fail", "generation timeout is too large (>1800s)"
				}
				if v > 300 {
					return "warn", "generation timeout is high and may mask stuck turns"
				}
				return "pass", ""
			},
		},
		{
			Path:           "runtime.queue.workers",
			Domain:         "session",
			Risk:           "high",
			Description:    "Queue worker count affecting concurrency and throughput.",
			Recommendation: "Keep queue.workers in [1,8] unless host sizing is validated by benchmarks.",
			Current: func(cfg config.Config) any {
				return cfg.Runtime.Queue.Workers
			},
			Default: func(cfg config.Config) any {
				return cfg.Runtime.Queue.Workers
			},
			Validate: func(cfg config.Config) (string, string) {
				v := cfg.Runtime.Queue.Workers
				if v <= 0 {
					return "fail", "workers must be > 0"
				}
				if v > 32 {
					return "fail", "workers exceeds hard safety cap (32)"
				}
				if v > 8 {
					return "warn", "workers is above validated baseline"
				}
				return "pass", ""
			},
		},
		{
			Path:           "runtime.queue.max_retries",
			Domain:         "session",
			Risk:           "medium",
			Description:    "Retry count per queued attempt.",
			Recommendation: "Keep max_retries <= 3 to avoid retry storms and hidden failures.",
			Current: func(cfg config.Config) any {
				return cfg.Runtime.Queue.MaxRetries
			},
			Default: func(cfg config.Config) any {
				return cfg.Runtime.Queue.MaxRetries
			},
			Validate: func(cfg config.Config) (string, string) {
				v := cfg.Runtime.Queue.MaxRetries
				if v < 0 {
					return "fail", "max_retries must be >= 0"
				}
				if v > 10 {
					return "fail", "max_retries exceeds hard safety cap (10)"
				}
				if v > 3 {
					return "warn", "max_retries above recommended baseline"
				}
				return "pass", ""
			},
		},
		{
			Path:           "runtime.shutdown.drain_timeout_sec",
			Domain:         "session",
			Risk:           "high",
			Description:    "Graceful shutdown drain timeout.",
			Recommendation: "Use 5-60 seconds to balance graceful drain and restart responsiveness.",
			Current: func(cfg config.Config) any {
				return cfg.Runtime.Shutdown.DrainTimeoutSec
			},
			Default: func(cfg config.Config) any {
				return cfg.Runtime.Shutdown.DrainTimeoutSec
			},
			Validate: func(cfg config.Config) (string, string) {
				v := cfg.Runtime.Shutdown.DrainTimeoutSec
				if v < 3 {
					return "fail", "drain timeout is too small (<3s)"
				}
				if v > 180 {
					return "fail", "drain timeout is too large (>180s)"
				}
				if v > 60 {
					return "warn", "drain timeout above recommended ceiling"
				}
				return "pass", ""
			},
		},
		{
			Path:           "security.admin_user_ids",
			Domain:         "tools",
			Risk:           "high",
			Description:    "Admin principal allowlist for privileged operations.",
			Recommendation: "Keep at least one explicit admin id and avoid wildcard entries.",
			Current: func(cfg config.Config) any {
				return cfg.Security.AdminUserIDs
			},
			Default: func(cfg config.Config) any {
				return cfg.Security.AdminUserIDs
			},
			Validate: func(cfg config.Config) (string, string) {
				if len(cfg.Security.AdminUserIDs) == 0 {
					return "fail", "admin_user_ids cannot be empty"
				}
				for _, userID := range cfg.Security.AdminUserIDs {
					if strings.TrimSpace(userID) == "*" {
						return "fail", "wildcard admin_user_ids entry is not allowed"
					}
				}
				return "pass", ""
			},
		},
		{
			Path:           "security.tools.require_confirm",
			Domain:         "tools",
			Risk:           "critical",
			Description:    "High-risk tools requiring explicit confirmation.",
			Recommendation: "Ensure browser/canvas/nodes/message remain in require_confirm.",
			Current: func(cfg config.Config) any {
				return cfg.Security.Tools.RequireConfirm
			},
			Default: func(cfg config.Config) any {
				return cfg.Security.Tools.RequireConfirm
			},
			Validate: func(cfg config.Config) (string, string) {
				set := listToSet(cfg.Security.Tools.RequireConfirm)
				missing := make([]string, 0)
				for _, tool := range requiredConfirmTools {
					if _, ok := set[tool]; !ok {
						missing = append(missing, tool)
					}
				}
				if len(missing) > 0 {
					sort.Strings(missing)
					return "fail", fmt.Sprintf("missing required confirmation tools: %s", strings.Join(missing, ", "))
				}
				return "pass", ""
			},
		},
		{
			Path:           "security.tools.global_allow",
			Domain:         "tools",
			Risk:           "high",
			Description:    "Global tool allowlist baseline.",
			Recommendation: "Avoid overlap between global_allow and global_deny.",
			Current: func(cfg config.Config) any {
				return cfg.Security.Tools.GlobalAllow
			},
			Default: func(cfg config.Config) any {
				return cfg.Security.Tools.GlobalAllow
			},
			Validate: func(cfg config.Config) (string, string) {
				allow := listToSet(cfg.Security.Tools.GlobalAllow)
				overlap := make([]string, 0)
				for _, item := range cfg.Security.Tools.GlobalDeny {
					if _, ok := allow[normalizeName(item)]; ok {
						overlap = append(overlap, normalizeName(item))
					}
				}
				if len(overlap) > 0 {
					sort.Strings(overlap)
					return "fail", fmt.Sprintf("overlap with global_deny: %s", strings.Join(overlap, ", "))
				}
				return "pass", ""
			},
		},
		{
			Path:           "security.memory.trusted_channels",
			Domain:         "tools",
			Risk:           "high",
			Description:    "Channel allowlist for memory_search/memory_get access.",
			Recommendation: "Limit trusted channels to vetted internal surfaces and keep the allowlist non-empty.",
			Current: func(cfg config.Config) any {
				return cfg.Security.Memory.TrustedChannels
			},
			Default: func(cfg config.Config) any {
				return cfg.Security.Memory.TrustedChannels
			},
			Validate: func(cfg config.Config) (string, string) {
				if len(cfg.Security.Memory.TrustedChannels) == 0 {
					return "fail", "trusted_channels cannot be empty"
				}
				if !containsNormalized(cfg.Security.Memory.RestrictedPaths, "memory.md") {
					return "warn", "restricted_paths does not include memory.md"
				}
				return "pass", ""
			},
		},
	}
}

func buildNextSteps(report Report) []string {
	steps := []string{
		"Review parameter audit report before release and attach it to change notes.",
	}
	if report.Gate.Passed {
		steps = append(steps, "Parameter governance checks passed; keep monthly review cadence for agents/bindings/session/tools.")
		if report.Summary.DefaultedParameters > 0 {
			steps = append(steps, fmt.Sprintf("Config uses defaults for %d parameters; promote critical overrides explicitly when environment-specific.", report.Summary.DefaultedParameters))
		}
		return steps
	}
	steps = append(steps, "Fix failed high-risk parameters first, rerun make config-governance, then rerun make release-gate.")
	if report.Summary.WarningChecks > 0 {
		steps = append(steps, "Address warning-level parameters in the same milestone to keep governance drift bounded.")
	}
	return steps
}

func hasPath(root map[string]any, path []string) bool {
	if len(path) == 0 {
		return false
	}
	var current any = root
	for _, part := range path {
		obj, ok := current.(map[string]any)
		if !ok {
			return false
		}
		next, ok := obj[part]
		if !ok {
			return false
		}
		current = next
	}
	return true
}

func normalizeRisk(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "critical", "high", "medium", "low":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return "medium"
	}
}

func riskRank(raw string) int {
	switch normalizeRisk(raw) {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	default:
		return 1
	}
}

func normalizeStatus(status string, message string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "fail", "warn", "pass":
		return strings.ToLower(strings.TrimSpace(status)), strings.TrimSpace(message)
	default:
		if strings.TrimSpace(message) == "" {
			return "pass", ""
		}
		return "warn", strings.TrimSpace(message)
	}
}

func listToSet(items []string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, item := range items {
		name := normalizeName(item)
		if name == "" {
			continue
		}
		set[name] = struct{}{}
	}
	return set
}

func normalizeName(raw string) string {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	trimmed = strings.ReplaceAll(trimmed, "-", "_")
	trimmed = strings.ReplaceAll(trimmed, " ", "_")
	return trimmed
}

func containsNormalized(items []string, target string) bool {
	needle := normalizeName(target)
	for _, item := range items {
		if normalizeName(item) == needle {
			return true
		}
	}
	return false
}

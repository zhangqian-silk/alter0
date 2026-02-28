package builtins

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	config "alter0/app/configs"
	"alter0/app/core/agent"
	"alter0/app/pkg/types"
)

const (
	keyAgentName           = "agent.name"
	keyExecutorName        = "executor.name"
	keyTaskRouteTO         = "task.routing_timeout_sec"
	keyTaskCloseTO         = "task.close_timeout_sec"
	keyTaskGenTO           = "task.generation_timeout_sec"
	keyTaskRouteThres      = "task.routing_confidence_threshold"
	keyTaskCloseThres      = "task.close_confidence_threshold"
	keyTaskCLIUserID       = "task.cli_user_id"
	keyTaskCandLimit       = "task.open_task_candidate_limit"
	keyRuntimeMaintEnabled = "runtime.maintenance.enabled"
	keyRuntimeMaintIntvSec = "runtime.maintenance.task_memory_prune_interval_sec"
	keyRuntimeMaintTOsec   = "runtime.maintenance.task_memory_prune_timeout_sec"
	keyRuntimeMaintRetDays = "runtime.maintenance.task_memory_retention_days"
	keyAdminUserIDs        = "security.admin_user_ids"
)

type ConfigSkill struct {
	manager *config.Manager
	apply   func(config.Config) error
}

func NewConfigSkill(manager *config.Manager, apply func(config.Config) error) *ConfigSkill {
	return &ConfigSkill{manager: manager, apply: apply}
}

func (s *ConfigSkill) Manifest() types.SkillManifest {
	return types.SkillManifest{
		Name:        "config",
		Description: "Get or update runtime configuration",
		Parameters: map[string]interface{}{
			"action": "string - 'get' or 'set'",
			"key":    "string - config key",
			"value":  "string - config value",
		},
	}
}

func (s *ConfigSkill) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	action, _ := args["action"].(string)
	key, _ := args["key"].(string)
	value, _ := args["value"].(string)
	if action == "" {
		if cmd, ok := args["command"].(string); ok {
			pa, pk, pv, err := parseConfigCommand(cmd)
			if err != nil {
				return nil, err
			}
			action, key, value = pa, pk, pv
		}
	}
	if action == "" {
		return nil, fmt.Errorf("missing action")
	}

	switch strings.ToLower(action) {
	case "get":
		cfg := s.manager.Get()
		if strings.TrimSpace(key) == "" {
			return sanitizeConfig(cfg), nil
		}
		val, ok := getConfigValue(cfg, key)
		if !ok {
			return nil, fmt.Errorf("unknown key: %s", key)
		}
		return map[string]interface{}{key: val}, nil
	case "set":
		normalizedKey := normalizeConfigKey(key)
		if normalizedKey == "" {
			return nil, fmt.Errorf("unknown key: %s", key)
		}
		if normalizedKey == keyExecutorName {
			normalized := agent.NormalizeExecutorName(value)
			if normalized == "" {
				return nil, fmt.Errorf("invalid executor: %s", value)
			}
			if _, err := agent.EnsureExecutorInstalled(ctx, normalized); err != nil {
				return nil, err
			}
			value = normalized
		}
		updated, err := s.manager.Update(func(c *config.Config) {
			applyConfigValue(c, normalizedKey, value)
		})
		if err != nil {
			return nil, err
		}
		if err := s.apply(updated); err != nil {
			return nil, err
		}
		return sanitizeConfig(updated), nil
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

func parseConfigCommand(cmd string) (string, string, string, error) {
	tokens := strings.Fields(cmd)
	if len(tokens) == 0 {
		return "", "", "", fmt.Errorf("missing action")
	}
	action := strings.ToLower(tokens[0])
	if action == "get" {
		if len(tokens) > 1 {
			return action, tokens[1], "", nil
		}
		return action, "", "", nil
	}
	if action == "set" {
		if len(tokens) < 3 {
			return "", "", "", fmt.Errorf("usage: /config set <key> <value>")
		}
		return action, tokens[1], strings.Join(tokens[2:], " "), nil
	}
	return "", "", "", fmt.Errorf("unknown action: %s", action)
}

func normalizeConfigKey(key string) string {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "agent.name", "agent_name", "name":
		return keyAgentName
	case "executor.name", "executor":
		return keyExecutorName
	case "task.routing_timeout_sec":
		return keyTaskRouteTO
	case "task.close_timeout_sec":
		return keyTaskCloseTO
	case "task.generation_timeout_sec":
		return keyTaskGenTO
	case "task.routing_confidence_threshold":
		return keyTaskRouteThres
	case "task.close_confidence_threshold":
		return keyTaskCloseThres
	case "task.cli_user_id":
		return keyTaskCLIUserID
	case "task.open_task_candidate_limit":
		return keyTaskCandLimit
	case "runtime.maintenance.enabled", "maintenance.enabled":
		return keyRuntimeMaintEnabled
	case "runtime.maintenance.task_memory_prune_interval_sec", "maintenance.task_memory_prune_interval_sec":
		return keyRuntimeMaintIntvSec
	case "runtime.maintenance.task_memory_prune_timeout_sec", "maintenance.task_memory_prune_timeout_sec":
		return keyRuntimeMaintTOsec
	case "runtime.maintenance.task_memory_retention_days", "maintenance.task_memory_retention_days":
		return keyRuntimeMaintRetDays
	case "security.admin_user_ids", "security.admins":
		return keyAdminUserIDs
	default:
		return ""
	}
}

func getConfigValue(cfg config.Config, key string) (interface{}, bool) {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "agent":
		return map[string]interface{}{"name": cfg.Agent.Name}, true
	case "executor":
		return map[string]interface{}{"name": cfg.Executor.Name}, true
	case "task":
		return sanitizeTask(cfg.Task), true
	case "runtime":
		return sanitizeRuntime(cfg.Runtime), true
	case "security":
		return sanitizeSecurity(cfg.Security), true
	}
	switch normalizeConfigKey(key) {
	case keyAgentName:
		return cfg.Agent.Name, true
	case keyExecutorName:
		return cfg.Executor.Name, true
	case keyTaskRouteTO:
		return cfg.Task.RoutingTimeoutSec, true
	case keyTaskCloseTO:
		return cfg.Task.CloseTimeoutSec, true
	case keyTaskGenTO:
		return cfg.Task.GenerationTimeoutSec, true
	case keyTaskRouteThres:
		return cfg.Task.RoutingConfidenceThreshold, true
	case keyTaskCloseThres:
		return cfg.Task.CloseConfidenceThreshold, true
	case keyTaskCLIUserID:
		return cfg.Task.CLIUserID, true
	case keyTaskCandLimit:
		return cfg.Task.OpenTaskCandidateLimit, true
	case keyRuntimeMaintEnabled:
		return cfg.Runtime.Maintenance.Enabled, true
	case keyRuntimeMaintIntvSec:
		return cfg.Runtime.Maintenance.TaskMemoryPruneIntervalSec, true
	case keyRuntimeMaintTOsec:
		return cfg.Runtime.Maintenance.TaskMemoryPruneTimeoutSec, true
	case keyRuntimeMaintRetDays:
		return cfg.Runtime.Maintenance.TaskMemoryRetentionDays, true
	case keyAdminUserIDs:
		return cfg.Security.AdminUserIDs, true
	default:
		return nil, false
	}
}

func applyConfigValue(c *config.Config, key string, value string) {
	switch key {
	case keyAgentName:
		c.Agent.Name = value
	case keyExecutorName:
		c.Executor.Name = value
	case keyTaskRouteTO:
		if n, err := strconv.Atoi(value); err == nil {
			c.Task.RoutingTimeoutSec = n
		}
	case keyTaskCloseTO:
		if n, err := strconv.Atoi(value); err == nil {
			c.Task.CloseTimeoutSec = n
		}
	case keyTaskGenTO:
		if n, err := strconv.Atoi(value); err == nil {
			c.Task.GenerationTimeoutSec = n
		}
	case keyTaskRouteThres:
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			c.Task.RoutingConfidenceThreshold = f
		}
	case keyTaskCloseThres:
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			c.Task.CloseConfidenceThreshold = f
		}
	case keyTaskCLIUserID:
		c.Task.CLIUserID = value
	case keyTaskCandLimit:
		if n, err := strconv.Atoi(value); err == nil {
			c.Task.OpenTaskCandidateLimit = n
		}
	case keyRuntimeMaintEnabled:
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "1", "true", "yes", "on":
			c.Runtime.Maintenance.Enabled = true
		case "0", "false", "no", "off":
			c.Runtime.Maintenance.Enabled = false
		}
	case keyRuntimeMaintIntvSec:
		if n, err := strconv.Atoi(value); err == nil {
			c.Runtime.Maintenance.TaskMemoryPruneIntervalSec = n
		}
	case keyRuntimeMaintTOsec:
		if n, err := strconv.Atoi(value); err == nil {
			c.Runtime.Maintenance.TaskMemoryPruneTimeoutSec = n
		}
	case keyRuntimeMaintRetDays:
		if n, err := strconv.Atoi(value); err == nil {
			c.Runtime.Maintenance.TaskMemoryRetentionDays = n
		}
	case keyAdminUserIDs:
		items := strings.Split(value, ",")
		clean := make([]string, 0, len(items))
		for _, item := range items {
			trimmed := strings.TrimSpace(item)
			if trimmed == "" {
				continue
			}
			clean = append(clean, trimmed)
		}
		c.Security.AdminUserIDs = clean
	}
}

func sanitizeTask(t config.TaskConfig) map[string]interface{} {
	return map[string]interface{}{
		"routing_timeout_sec":          t.RoutingTimeoutSec,
		"close_timeout_sec":            t.CloseTimeoutSec,
		"generation_timeout_sec":       t.GenerationTimeoutSec,
		"routing_confidence_threshold": t.RoutingConfidenceThreshold,
		"close_confidence_threshold":   t.CloseConfidenceThreshold,
		"cli_user_id":                  t.CLIUserID,
		"open_task_candidate_limit":    t.OpenTaskCandidateLimit,
	}
}

func sanitizeRuntime(r config.RuntimeConfig) map[string]interface{} {
	return map[string]interface{}{
		"maintenance": map[string]interface{}{
			"enabled":                        r.Maintenance.Enabled,
			"task_memory_prune_interval_sec": r.Maintenance.TaskMemoryPruneIntervalSec,
			"task_memory_prune_timeout_sec":  r.Maintenance.TaskMemoryPruneTimeoutSec,
			"task_memory_retention_days":     r.Maintenance.TaskMemoryRetentionDays,
		},
	}
}

func sanitizeSecurity(s config.SecurityConfig) map[string]interface{} {
	return map[string]interface{}{
		"admin_user_ids": s.AdminUserIDs,
	}
}

func sanitizeConfig(cfg config.Config) map[string]interface{} {
	return map[string]interface{}{
		"agent": map[string]interface{}{
			"name": cfg.Agent.Name,
		},
		"executor": map[string]interface{}{
			"name": cfg.Executor.Name,
		},
		"task":     sanitizeTask(cfg.Task),
		"runtime":  sanitizeRuntime(cfg.Runtime),
		"security": sanitizeSecurity(cfg.Security),
	}
}

type ExecutorSkill struct {
	manager *config.Manager
	apply   func(config.Config) error
}

func NewExecutorSkill(manager *config.Manager, apply func(config.Config) error) *ExecutorSkill {
	return &ExecutorSkill{manager: manager, apply: apply}
}

func (s *ExecutorSkill) Manifest() types.SkillManifest {
	return types.SkillManifest{
		Name:        "executor",
		Description: "Switch executor",
		Parameters: map[string]interface{}{
			"name": "string - executor name",
		},
	}
}

func (s *ExecutorSkill) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	command, _ := args["command"].(string)
	executorName := strings.TrimSpace(command)
	if executorName != "" {
		if fields := strings.Fields(executorName); len(fields) > 0 {
			executorName = fields[0]
		}
	}
	cfg := s.manager.Get()
	if executorName == "" {
		return formatExecutors(cfg), nil
	}
	normalized := agent.NormalizeExecutorName(executorName)
	if normalized == "" {
		return nil, fmt.Errorf("unknown executor: %s", executorName)
	}
	if _, err := agent.EnsureExecutorInstalled(ctx, normalized); err != nil {
		return nil, err
	}
	updated, err := s.manager.Update(func(c *config.Config) {
		c.Executor.Name = normalized
	})
	if err != nil {
		return nil, err
	}
	if err := s.apply(updated); err != nil {
		return nil, err
	}
	return fmt.Sprintf("current executor: %s", updated.Executor.Name), nil
}

func formatExecutors(cfg config.Config) string {
	executors := []string{"claude_code", "codex"}
	var b strings.Builder
	b.WriteString("Executors:\n")
	for _, name := range executors {
		status := "missing"
		if _, _, err := agent.ResolveExecutorCommand(name); err == nil {
			status = "installed"
		}
		b.WriteString("  ")
		b.WriteString(name)
		b.WriteString(" (")
		b.WriteString(status)
		b.WriteString(")\n")
	}
	b.WriteString("Current: ")
	if strings.TrimSpace(cfg.Executor.Name) == "" {
		b.WriteString("<empty>")
	} else {
		b.WriteString(cfg.Executor.Name)
	}
	return strings.TrimSpace(b.String())
}

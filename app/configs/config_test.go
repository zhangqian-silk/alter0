package config

import "testing"

func TestApplyDefaultsSetsRuntimeMaintenanceDefaults(t *testing.T) {
	cfg := Config{}

	applyDefaults(&cfg)

	if !cfg.Runtime.Maintenance.Enabled {
		t.Fatalf("expected maintenance enabled by default")
	}
	if cfg.Runtime.Maintenance.TaskMemoryPruneIntervalSec != 6*60*60 {
		t.Fatalf("unexpected prune interval: %d", cfg.Runtime.Maintenance.TaskMemoryPruneIntervalSec)
	}
	if cfg.Runtime.Maintenance.TaskMemoryPruneTimeoutSec != 20 {
		t.Fatalf("unexpected prune timeout: %d", cfg.Runtime.Maintenance.TaskMemoryPruneTimeoutSec)
	}
	if cfg.Runtime.Maintenance.TaskMemoryRetentionDays != 30 {
		t.Fatalf("unexpected retention days: %d", cfg.Runtime.Maintenance.TaskMemoryRetentionDays)
	}
	if cfg.Runtime.Maintenance.TaskMemoryOpenRetentionDays != 0 {
		t.Fatalf("unexpected open retention days: %d", cfg.Runtime.Maintenance.TaskMemoryOpenRetentionDays)
	}
}

func TestApplyDefaultsKeepsExplicitMaintenanceDisable(t *testing.T) {
	cfg := Config{
		Runtime: RuntimeConfig{
			Maintenance: MaintenanceConfig{
				Enabled:                    false,
				TaskMemoryPruneIntervalSec: 3600,
				TaskMemoryPruneTimeoutSec:  9,
				TaskMemoryRetentionDays:    7,
			},
		},
	}

	applyDefaults(&cfg)

	if cfg.Runtime.Maintenance.Enabled {
		t.Fatalf("expected maintenance to remain disabled")
	}
}

func TestApplyDefaultsSanitizesOpenRetentionDays(t *testing.T) {
	cfg := Config{
		Runtime: RuntimeConfig{
			Maintenance: MaintenanceConfig{
				TaskMemoryOpenRetentionDays: -5,
			},
		},
	}

	applyDefaults(&cfg)

	if cfg.Runtime.Maintenance.TaskMemoryOpenRetentionDays != 0 {
		t.Fatalf("expected open retention to be clamped to 0, got %d", cfg.Runtime.Maintenance.TaskMemoryOpenRetentionDays)
	}
}

func TestApplyDefaultsSetsQueueDefaults(t *testing.T) {
	cfg := Config{}

	applyDefaults(&cfg)

	if !cfg.Runtime.Queue.Enabled {
		t.Fatalf("expected execution queue enabled by default")
	}
	if cfg.Runtime.Queue.Workers != 2 {
		t.Fatalf("unexpected workers: %d", cfg.Runtime.Queue.Workers)
	}
	if cfg.Runtime.Queue.Buffer != 128 {
		t.Fatalf("unexpected buffer: %d", cfg.Runtime.Queue.Buffer)
	}
	if cfg.Runtime.Queue.EnqueueTimeoutSec != 3 {
		t.Fatalf("unexpected enqueue timeout: %d", cfg.Runtime.Queue.EnqueueTimeoutSec)
	}
	if cfg.Runtime.Queue.AttemptTimeoutSec != 180 {
		t.Fatalf("unexpected attempt timeout: %d", cfg.Runtime.Queue.AttemptTimeoutSec)
	}
	if cfg.Runtime.Queue.MaxRetries != 1 {
		t.Fatalf("unexpected max retries: %d", cfg.Runtime.Queue.MaxRetries)
	}
	if cfg.Runtime.Queue.RetryDelaySec != 2 {
		t.Fatalf("unexpected retry delay: %d", cfg.Runtime.Queue.RetryDelaySec)
	}
	if cfg.Runtime.Shutdown.DrainTimeoutSec != 5 {
		t.Fatalf("unexpected shutdown drain timeout: %d", cfg.Runtime.Shutdown.DrainTimeoutSec)
	}
}

func TestApplyDefaultsSetsShutdownDrainTimeout(t *testing.T) {
	cfg := Config{
		Runtime: RuntimeConfig{
			Shutdown: ShutdownConfig{DrainTimeoutSec: 0},
		},
	}

	applyDefaults(&cfg)

	if cfg.Runtime.Shutdown.DrainTimeoutSec != 5 {
		t.Fatalf("expected default drain timeout 5, got %d", cfg.Runtime.Shutdown.DrainTimeoutSec)
	}
}

func TestApplyDefaultsSetsChannelDefaults(t *testing.T) {
	cfg := Config{}

	applyDefaults(&cfg)

	if cfg.Channels.Telegram.PollIntervalSec != 2 {
		t.Fatalf("unexpected telegram poll interval: %d", cfg.Channels.Telegram.PollIntervalSec)
	}
	if cfg.Channels.Telegram.TimeoutSec != 20 {
		t.Fatalf("unexpected telegram timeout: %d", cfg.Channels.Telegram.TimeoutSec)
	}
	if cfg.Channels.Slack.EventListenPort != 8091 {
		t.Fatalf("unexpected slack listen port: %d", cfg.Channels.Slack.EventListenPort)
	}
	if cfg.Channels.Slack.EventPath != "/events/slack" {
		t.Fatalf("unexpected slack event path: %s", cfg.Channels.Slack.EventPath)
	}
}

func TestApplyDefaultsBuildsAgentRegistryDefaults(t *testing.T) {
	cfg := Config{}

	applyDefaults(&cfg)

	if cfg.Agent.DefaultID != "default" {
		t.Fatalf("expected default id 'default', got %q", cfg.Agent.DefaultID)
	}
	if len(cfg.Agent.Registry) != 1 {
		t.Fatalf("expected 1 default agent entry, got %d", len(cfg.Agent.Registry))
	}
	entry := cfg.Agent.Registry[0]
	if entry.ID != "default" {
		t.Fatalf("unexpected agent id: %s", entry.ID)
	}
	if entry.Workspace != "." {
		t.Fatalf("unexpected workspace: %s", entry.Workspace)
	}
	if entry.AgentDir == "" {
		t.Fatal("expected non-empty agent dir")
	}
	if entry.Executor != cfg.Executor.Name {
		t.Fatalf("unexpected executor fallback: %s", entry.Executor)
	}
}

func TestApplyDefaultsAgentRegistryDedupAndFallbackDefault(t *testing.T) {
	cfg := Config{
		Agent: AgentConfig{
			Name:      "Alter0",
			DefaultID: "missing",
			Registry: []AgentRegistryEntry{
				{ID: "alpha", Workspace: "", AgentDir: "", Executor: ""},
				{ID: "alpha", Workspace: "dup"},
			},
		},
		Executor: ExecutorConfig{Name: "codex"},
	}

	applyDefaults(&cfg)

	if len(cfg.Agent.Registry) != 1 {
		t.Fatalf("expected deduped registry size 1, got %d", len(cfg.Agent.Registry))
	}
	entry := cfg.Agent.Registry[0]
	if entry.ID != "alpha" {
		t.Fatalf("unexpected agent id: %s", entry.ID)
	}
	if entry.Executor != "codex" {
		t.Fatalf("expected executor fallback codex, got %s", entry.Executor)
	}
	if cfg.Agent.DefaultID != "alpha" {
		t.Fatalf("expected default to fall back to alpha, got %s", cfg.Agent.DefaultID)
	}
}

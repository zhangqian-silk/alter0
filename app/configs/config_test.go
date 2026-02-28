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
}

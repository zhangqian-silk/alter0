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

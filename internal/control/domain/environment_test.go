package domain

import "testing"

func TestEnvironmentDefinitionNormalizeValue(t *testing.T) {
	definition := EnvironmentDefinition{
		Key:       "async_task_workers",
		Type:      EnvironmentValueTypeInteger,
		ApplyMode: EnvironmentApplyModeRestart,
		Validation: EnvironmentValidation{
			Required: true,
			Min:      "1",
			Max:      "5",
		},
	}

	normalized, err := definition.NormalizeValue(" 3 ")
	if err != nil {
		t.Fatalf("normalize failed: %v", err)
	}
	if normalized != "3" {
		t.Fatalf("expected 3, got %s", normalized)
	}

	if _, err := definition.NormalizeValue("0"); err == nil {
		t.Fatalf("expected min validation error")
	}
}

func TestEnvironmentDefinitionNormalizeDuration(t *testing.T) {
	definition := EnvironmentDefinition{
		Key:       "queue_timeout",
		Type:      EnvironmentValueTypeDuration,
		ApplyMode: EnvironmentApplyModeRestart,
		Validation: EnvironmentValidation{
			Required: true,
			Min:      "100ms",
			Max:      "5m",
		},
	}

	normalized, err := definition.NormalizeValue("2s")
	if err != nil {
		t.Fatalf("normalize failed: %v", err)
	}
	if normalized != "2s" {
		t.Fatalf("expected 2s, got %s", normalized)
	}

	if _, err := definition.NormalizeValue("10m"); err == nil {
		t.Fatalf("expected max validation error")
	}
}

func TestEnvironmentDefinitionNormalizeEnum(t *testing.T) {
	definition := EnvironmentDefinition{
		Key:       "long_term_memory_write_policy",
		Type:      EnvironmentValueTypeEnum,
		ApplyMode: EnvironmentApplyModeRestart,
		Validation: EnvironmentValidation{
			Required: true,
			Allowed:  []string{"write_through", "write_back"},
		},
	}

	normalized, err := definition.NormalizeValue("WRITE_BACK")
	if err != nil {
		t.Fatalf("normalize failed: %v", err)
	}
	if normalized != "write_back" {
		t.Fatalf("expected write_back, got %s", normalized)
	}
}

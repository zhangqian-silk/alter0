package domain

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type EnvironmentValueType string

const (
	EnvironmentValueTypeInteger  EnvironmentValueType = "integer"
	EnvironmentValueTypeDuration EnvironmentValueType = "duration"
	EnvironmentValueTypeString   EnvironmentValueType = "string"
	EnvironmentValueTypeEnum     EnvironmentValueType = "enum"
)

type EnvironmentApplyMode string

const (
	EnvironmentApplyModeImmediate EnvironmentApplyMode = "immediate"
	EnvironmentApplyModeRestart   EnvironmentApplyMode = "restart"
)

type EnvironmentValidation struct {
	Required bool     `json:"required"`
	Min      string   `json:"min,omitempty"`
	Max      string   `json:"max,omitempty"`
	Allowed  []string `json:"allowed,omitempty"`
}

type EnvironmentDefinition struct {
	Key          string               `json:"key"`
	Name         string               `json:"name"`
	Module       string               `json:"module"`
	Description  string               `json:"description,omitempty"`
	Type         EnvironmentValueType `json:"type"`
	DefaultValue string               `json:"default_value"`
	Sensitive    bool                 `json:"sensitive"`
	HotReload    bool                 `json:"hot_reload"`
	ApplyMode    EnvironmentApplyMode `json:"apply_mode"`
	Validation   EnvironmentValidation `json:"validation"`
}

type EnvironmentConfigItem struct {
	Definition     EnvironmentDefinition `json:"definition"`
	Value          string                `json:"value"`
	EffectiveValue string                `json:"effective_value"`
	ValueSource    string                `json:"value_source"`
	PendingRestart bool                  `json:"pending_restart"`
	Masked         bool                  `json:"masked"`
}

type EnvironmentAuditChange struct {
	Key       string               `json:"key"`
	OldValue  string               `json:"old_value"`
	NewValue  string               `json:"new_value"`
	ApplyMode EnvironmentApplyMode `json:"apply_mode"`
}

type EnvironmentAudit struct {
	Operator       string                   `json:"operator"`
	OccurredAt     time.Time                `json:"occurred_at"`
	Changes        []EnvironmentAuditChange `json:"changes"`
	RequiresRestart bool                    `json:"requires_restart"`
}

type EnvironmentUpdateResult struct {
	Changed      []EnvironmentAuditChange `json:"changed"`
	NeedsRestart bool                     `json:"needs_restart"`
	RestartKeys  []string                 `json:"restart_keys,omitempty"`
}

func (d EnvironmentDefinition) NormalizeValue(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		if d.Validation.Required {
			return "", fmt.Errorf("%s is required", d.Key)
		}
		return "", nil
	}

	switch d.Type {
	case EnvironmentValueTypeInteger:
		number, err := strconv.Atoi(value)
		if err != nil {
			return "", fmt.Errorf("%s must be integer", d.Key)
		}
		if err := d.validateInteger(number); err != nil {
			return "", err
		}
		return strconv.Itoa(number), nil
	case EnvironmentValueTypeDuration:
		duration, err := time.ParseDuration(value)
		if err != nil {
			return "", fmt.Errorf("%s must be duration", d.Key)
		}
		if err := d.validateDuration(duration); err != nil {
			return "", err
		}
		return duration.String(), nil
	case EnvironmentValueTypeEnum:
		if len(d.Validation.Allowed) == 0 {
			return value, nil
		}
		normalized := strings.ToLower(value)
		for _, candidate := range d.Validation.Allowed {
			option := strings.TrimSpace(candidate)
			if option == "" {
				continue
			}
			if normalized == strings.ToLower(option) {
				return option, nil
			}
		}
		return "", fmt.Errorf("%s must be one of: %s", d.Key, strings.Join(d.Validation.Allowed, ", "))
	case EnvironmentValueTypeString:
		return value, nil
	default:
		return "", fmt.Errorf("%s has unsupported type %s", d.Key, d.Type)
	}
}

func (d EnvironmentDefinition) validateInteger(value int) error {
	minRaw := strings.TrimSpace(d.Validation.Min)
	if minRaw != "" {
		minimum, err := strconv.Atoi(minRaw)
		if err == nil && value < minimum {
			return fmt.Errorf("%s must be >= %d", d.Key, minimum)
		}
	}
	maxRaw := strings.TrimSpace(d.Validation.Max)
	if maxRaw != "" {
		maximum, err := strconv.Atoi(maxRaw)
		if err == nil && value > maximum {
			return fmt.Errorf("%s must be <= %d", d.Key, maximum)
		}
	}
	return nil
}

func (d EnvironmentDefinition) validateDuration(value time.Duration) error {
	minRaw := strings.TrimSpace(d.Validation.Min)
	if minRaw != "" {
		minimum, err := time.ParseDuration(minRaw)
		if err == nil && value < minimum {
			return fmt.Errorf("%s must be >= %s", d.Key, minimum)
		}
	}
	maxRaw := strings.TrimSpace(d.Validation.Max)
	if maxRaw != "" {
		maximum, err := time.ParseDuration(maxRaw)
		if err == nil && value > maximum {
			return fmt.Errorf("%s must be <= %s", d.Key, maximum)
		}
	}
	return nil
}

func MaskEnvironmentValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) <= 4 {
		return "****"
	}
	return trimmed[:2] + "****" + trimmed[len(trimmed)-2:]
}

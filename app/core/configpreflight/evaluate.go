package configpreflight

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	config "alter0/app/configs"
	"alter0/app/core/runtime"
)

type Options struct {
	AllowMissingConfig bool
}

type Check struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message,omitempty"`
}

type Gate struct {
	Passed   bool     `json:"passed"`
	Failures []string `json:"failures,omitempty"`
}

type Report struct {
	GeneratedAt       string  `json:"generated_at"`
	ConfigPath        string  `json:"config_path"`
	ConfigExists      bool    `json:"config_exists"`
	UsedDefaultConfig bool    `json:"used_default_config"`
	Status            string  `json:"status"`
	Checks            []Check `json:"checks"`
	Gate              Gate    `json:"gate"`
}

func EvaluatePath(configPath string, opts Options) Report {
	report := Report{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		ConfigPath:  strings.TrimSpace(configPath),
		Status:      "failed",
		Checks:      make([]Check, 0, 2),
		Gate: Gate{
			Passed:   false,
			Failures: []string{},
		},
	}

	if report.ConfigPath == "" {
		appendFailure(&report, "config path is required")
		appendCheck(&report, "config_load", false, "config path is empty")
		return finalize(report)
	}

	cfg, exists, usedDefault, loadErr := loadEffectiveConfig(report.ConfigPath, opts.AllowMissingConfig)
	report.ConfigExists = exists
	report.UsedDefaultConfig = usedDefault
	if loadErr != nil {
		appendFailure(&report, loadErr.Error())
		appendCheck(&report, "config_load", false, loadErr.Error())
		return finalize(report)
	}
	appendCheck(&report, "config_load", true, "config loaded")

	if err := runtime.ValidateConfig(cfg); err != nil {
		message := fmt.Sprintf("config validation failed: %v", err)
		appendFailure(&report, message)
		appendCheck(&report, "config_validate", false, err.Error())
		return finalize(report)
	}
	appendCheck(&report, "config_validate", true, "runtime preflight config validation passed")

	return finalize(report)
}

func finalize(report Report) Report {
	if len(report.Gate.Failures) == 0 {
		report.Gate.Passed = true
		report.Status = "ok"
		return report
	}
	report.Gate.Passed = false
	report.Status = "failed"
	return report
}

func appendFailure(report *Report, failure string) {
	trimmed := strings.TrimSpace(failure)
	if trimmed == "" {
		return
	}
	report.Gate.Failures = append(report.Gate.Failures, trimmed)
}

func appendCheck(report *Report, name string, passed bool, message string) {
	report.Checks = append(report.Checks, Check{
		Name:    name,
		Passed:  passed,
		Message: strings.TrimSpace(message),
	})
}

func loadEffectiveConfig(configPath string, allowMissing bool) (config.Config, bool, bool, error) {
	stat, err := os.Stat(configPath)
	if err == nil {
		if stat.IsDir() {
			return config.Config{}, false, false, fmt.Errorf("config path is a directory: %s", configPath)
		}
		cfg, err := loadConfigFromFile(configPath)
		if err != nil {
			return config.Config{}, true, false, fmt.Errorf("load config failed: %w", err)
		}
		return cfg, true, false, nil
	}
	if !os.IsNotExist(err) {
		return config.Config{}, false, false, fmt.Errorf("stat config path failed: %w", err)
	}
	if !allowMissing {
		return config.Config{}, false, false, fmt.Errorf("config file not found: %s", configPath)
	}
	cfg, err := loadConfigFromBytes(nil)
	if err != nil {
		return config.Config{}, false, true, fmt.Errorf("load default config failed: %w", err)
	}
	return cfg, false, true, nil
}

func loadConfigFromFile(configPath string) (config.Config, error) {
	payload, err := os.ReadFile(configPath)
	if err != nil {
		return config.Config{}, err
	}
	return loadConfigFromBytes(payload)
}

func loadConfigFromBytes(payload []byte) (config.Config, error) {
	tempDir, err := os.MkdirTemp("", "alter0-config-preflight-")
	if err != nil {
		return config.Config{}, err
	}
	defer os.RemoveAll(tempDir)

	tempPath := filepath.Join(tempDir, "config.json")
	if payload != nil {
		if err := os.WriteFile(tempPath, payload, 0644); err != nil {
			return config.Config{}, err
		}
	}

	mgr, err := config.NewManager(tempPath)
	if err != nil {
		return config.Config{}, err
	}
	return mgr.Get(), nil
}

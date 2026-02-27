package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Config struct {
	Agent    AgentConfig    `json:"agent"`
	Executor ExecutorConfig `json:"executor"`
	Task     TaskConfig     `json:"task"`
}

type AgentConfig struct {
	Name string `json:"name"`
}

type ExecutorConfig struct {
	Name string `json:"name"`
}

type TaskConfig struct {
	RoutingTimeoutSec          int     `json:"routing_timeout_sec"`
	CloseTimeoutSec            int     `json:"close_timeout_sec"`
	GenerationTimeoutSec       int     `json:"generation_timeout_sec"`
	RoutingConfidenceThreshold float64 `json:"routing_confidence_threshold"`
	CloseConfidenceThreshold   float64 `json:"close_confidence_threshold"`
	CLIUserID                  string  `json:"cli_user_id"`
	OpenTaskCandidateLimit     int     `json:"open_task_candidate_limit"`
}

type Manager struct {
	path string
	mu   sync.RWMutex
	cfg  Config
}

func DefaultPath() string {
	return filepath.Join("config", "config.json")
}

func NewManager(path string) (*Manager, error) {
	cfg := defaultConfig()
	mgr := &Manager{
		path: path,
		cfg:  cfg,
	}
	if err := mgr.load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	if err := mgr.save(); err != nil {
		return nil, err
	}
	return mgr, nil
}

func (m *Manager) Get() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg
}

func (m *Manager) Update(apply func(*Config)) (Config, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	apply(&m.cfg)
	applyDefaults(&m.cfg)
	if err := m.saveLocked(); err != nil {
		return Config{}, err
	}
	return m.cfg, nil
}

func (m *Manager) load() error {
	data, err := os.ReadFile(m.path)
	if err != nil {
		return err
	}
	var fileCfg Config
	if err := json.Unmarshal(data, &fileCfg); err != nil {
		return err
	}
	m.cfg = fileCfg
	applyDefaults(&m.cfg)
	return nil
}

func (m *Manager) save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveLocked()
}

func (m *Manager) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(m.path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m.cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.path, data, 0644)
}

func defaultConfig() Config {
	return Config{
		Agent: AgentConfig{
			Name: "Alter0",
		},
		Executor: ExecutorConfig{
			Name: "claude_code",
		},
		Task: TaskConfig{
			RoutingTimeoutSec:          15,
			CloseTimeoutSec:            10,
			GenerationTimeoutSec:       120,
			RoutingConfidenceThreshold: 0.55,
			CloseConfidenceThreshold:   0.70,
			CLIUserID:                  "local_user",
			OpenTaskCandidateLimit:     8,
		},
	}
}

func applyDefaults(cfg *Config) {
	if strings.TrimSpace(cfg.Agent.Name) == "" {
		cfg.Agent.Name = "Alter0"
	}
	if strings.TrimSpace(cfg.Executor.Name) == "" {
		cfg.Executor.Name = "claude_code"
	}
	if cfg.Task.RoutingTimeoutSec <= 0 {
		cfg.Task.RoutingTimeoutSec = 15
	}
	if cfg.Task.CloseTimeoutSec <= 0 {
		cfg.Task.CloseTimeoutSec = 10
	}
	if cfg.Task.GenerationTimeoutSec <= 0 {
		cfg.Task.GenerationTimeoutSec = 120
	}
	if cfg.Task.RoutingConfidenceThreshold <= 0 || cfg.Task.RoutingConfidenceThreshold > 1 {
		cfg.Task.RoutingConfidenceThreshold = 0.55
	}
	if cfg.Task.CloseConfidenceThreshold <= 0 || cfg.Task.CloseConfidenceThreshold > 1 {
		cfg.Task.CloseConfidenceThreshold = 0.70
	}
	if strings.TrimSpace(cfg.Task.CLIUserID) == "" {
		cfg.Task.CLIUserID = "local_user"
	}
	if cfg.Task.OpenTaskCandidateLimit <= 0 {
		cfg.Task.OpenTaskCandidateLimit = 8
	}
}

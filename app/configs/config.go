package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	toolcore "alter0/app/core/tools"
)

type Config struct {
	Agent    AgentConfig    `json:"agent"`
	Executor ExecutorConfig `json:"executor"`
	Task     TaskConfig     `json:"task"`
	Runtime  RuntimeConfig  `json:"runtime"`
	Security SecurityConfig `json:"security"`
	Channels ChannelConfig  `json:"channels"`
}

type AgentConfig struct {
	Name      string               `json:"name"`
	DefaultID string               `json:"default_id"`
	Registry  []AgentRegistryEntry `json:"registry"`
}

type AgentRegistryEntry struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Workspace string `json:"workspace"`
	AgentDir  string `json:"agent_dir"`
	Executor  string `json:"executor"`
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

type SecurityConfig struct {
	AdminUserIDs []string         `json:"admin_user_ids"`
	Tools        ToolPolicyConfig `json:"tools"`
}

type ToolPolicyConfig struct {
	GlobalAllow    []string                   `json:"global_allow"`
	GlobalDeny     []string                   `json:"global_deny"`
	RequireConfirm []string                   `json:"require_confirm"`
	Agent          map[string]ToolAgentPolicy `json:"agent"`
}

type ToolAgentPolicy struct {
	Allow []string `json:"allow"`
	Deny  []string `json:"deny"`
}

type RuntimeConfig struct {
	Maintenance MaintenanceConfig    `json:"maintenance"`
	Queue       ExecutionQueueConfig `json:"queue"`
	Shutdown    ShutdownConfig       `json:"shutdown"`
}

type ShutdownConfig struct {
	DrainTimeoutSec int `json:"drain_timeout_sec"`
}

type MaintenanceConfig struct {
	Enabled                     bool `json:"enabled"`
	TaskMemoryPruneIntervalSec  int  `json:"task_memory_prune_interval_sec"`
	TaskMemoryPruneTimeoutSec   int  `json:"task_memory_prune_timeout_sec"`
	TaskMemoryRetentionDays     int  `json:"task_memory_retention_days"`
	TaskMemoryOpenRetentionDays int  `json:"task_memory_open_retention_days"`
}

type ExecutionQueueConfig struct {
	Enabled           bool `json:"enabled"`
	Workers           int  `json:"workers"`
	Buffer            int  `json:"buffer"`
	EnqueueTimeoutSec int  `json:"enqueue_timeout_sec"`
	AttemptTimeoutSec int  `json:"attempt_timeout_sec"`
	MaxRetries        int  `json:"max_retries"`
	RetryDelaySec     int  `json:"retry_delay_sec"`
}

type ChannelConfig struct {
	Telegram TelegramChannelConfig `json:"telegram"`
	Slack    SlackChannelConfig    `json:"slack"`
}

type TelegramChannelConfig struct {
	Enabled         bool   `json:"enabled"`
	BotToken        string `json:"bot_token"`
	DefaultChatID   string `json:"default_chat_id"`
	PollIntervalSec int    `json:"poll_interval_sec"`
	TimeoutSec      int    `json:"timeout_sec"`
	APIBaseURL      string `json:"api_base_url"`
}

type SlackChannelConfig struct {
	Enabled          bool   `json:"enabled"`
	BotToken         string `json:"bot_token"`
	AppID            string `json:"app_id"`
	EventListenPort  int    `json:"event_listen_port"`
	EventPath        string `json:"event_path"`
	DefaultChannelID string `json:"default_channel_id"`
	APIBaseURL       string `json:"api_base_url"`
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
			Name:      "Alter0",
			DefaultID: "default",
			Registry: []AgentRegistryEntry{
				{
					ID:        "default",
					Name:      "Alter0",
					Workspace: ".",
					AgentDir:  filepath.Join("output", "agents", "default"),
					Executor:  "claude_code",
				},
			},
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
		Runtime: RuntimeConfig{
			Maintenance: MaintenanceConfig{
				Enabled:                     true,
				TaskMemoryPruneIntervalSec:  6 * 60 * 60,
				TaskMemoryPruneTimeoutSec:   20,
				TaskMemoryRetentionDays:     30,
				TaskMemoryOpenRetentionDays: 0,
			},
			Queue: ExecutionQueueConfig{
				Enabled:           true,
				Workers:           2,
				Buffer:            128,
				EnqueueTimeoutSec: 3,
				AttemptTimeoutSec: 180,
				MaxRetries:        1,
				RetryDelaySec:     2,
			},
			Shutdown: ShutdownConfig{
				DrainTimeoutSec: 5,
			},
		},
		Security: SecurityConfig{
			AdminUserIDs: []string{"local_user"},
			Tools: ToolPolicyConfig{
				RequireConfirm: []string{"browser", "canvas", "nodes", "message"},
				Agent:          map[string]ToolAgentPolicy{},
			},
		},
		Channels: ChannelConfig{
			Telegram: TelegramChannelConfig{
				Enabled:         false,
				PollIntervalSec: 2,
				TimeoutSec:      20,
			},
			Slack: SlackChannelConfig{
				Enabled:         false,
				EventListenPort: 8091,
				EventPath:       "/events/slack",
			},
		},
	}
}

func applyDefaults(cfg *Config) {
	if strings.TrimSpace(cfg.Agent.Name) == "" {
		cfg.Agent.Name = "Alter0"
	}
	if strings.TrimSpace(cfg.Agent.DefaultID) == "" {
		cfg.Agent.DefaultID = "default"
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
	if !cfg.Runtime.Maintenance.Enabled && cfg.Runtime.Maintenance.TaskMemoryPruneIntervalSec == 0 && cfg.Runtime.Maintenance.TaskMemoryPruneTimeoutSec == 0 && cfg.Runtime.Maintenance.TaskMemoryRetentionDays == 0 && cfg.Runtime.Maintenance.TaskMemoryOpenRetentionDays == 0 {
		cfg.Runtime.Maintenance.Enabled = true
	}
	if cfg.Runtime.Maintenance.TaskMemoryPruneIntervalSec <= 0 {
		cfg.Runtime.Maintenance.TaskMemoryPruneIntervalSec = 6 * 60 * 60
	}
	if cfg.Runtime.Maintenance.TaskMemoryPruneTimeoutSec <= 0 {
		cfg.Runtime.Maintenance.TaskMemoryPruneTimeoutSec = 20
	}
	if cfg.Runtime.Maintenance.TaskMemoryRetentionDays <= 0 {
		cfg.Runtime.Maintenance.TaskMemoryRetentionDays = 30
	}
	if cfg.Runtime.Maintenance.TaskMemoryOpenRetentionDays < 0 {
		cfg.Runtime.Maintenance.TaskMemoryOpenRetentionDays = 0
	}
	if !cfg.Runtime.Queue.Enabled && cfg.Runtime.Queue.Workers == 0 && cfg.Runtime.Queue.Buffer == 0 && cfg.Runtime.Queue.EnqueueTimeoutSec == 0 && cfg.Runtime.Queue.AttemptTimeoutSec == 0 && cfg.Runtime.Queue.MaxRetries == 0 && cfg.Runtime.Queue.RetryDelaySec == 0 {
		cfg.Runtime.Queue.Enabled = true
	}
	if cfg.Runtime.Queue.Workers <= 0 {
		cfg.Runtime.Queue.Workers = 2
	}
	if cfg.Runtime.Queue.Buffer <= 0 {
		cfg.Runtime.Queue.Buffer = 128
	}
	if cfg.Runtime.Queue.EnqueueTimeoutSec <= 0 {
		cfg.Runtime.Queue.EnqueueTimeoutSec = 3
	}
	if cfg.Runtime.Queue.AttemptTimeoutSec <= 0 {
		cfg.Runtime.Queue.AttemptTimeoutSec = 180
	}
	if cfg.Runtime.Queue.MaxRetries <= 0 {
		cfg.Runtime.Queue.MaxRetries = 1
	}
	if cfg.Runtime.Queue.RetryDelaySec <= 0 {
		cfg.Runtime.Queue.RetryDelaySec = 2
	}
	if cfg.Runtime.Shutdown.DrainTimeoutSec <= 0 {
		cfg.Runtime.Shutdown.DrainTimeoutSec = 5
	}
	if cfg.Channels.Telegram.PollIntervalSec <= 0 {
		cfg.Channels.Telegram.PollIntervalSec = 2
	}
	if cfg.Channels.Telegram.TimeoutSec <= 0 {
		cfg.Channels.Telegram.TimeoutSec = 20
	}
	if cfg.Channels.Slack.EventListenPort <= 0 {
		cfg.Channels.Slack.EventListenPort = 8091
	}
	if strings.TrimSpace(cfg.Channels.Slack.EventPath) == "" {
		cfg.Channels.Slack.EventPath = "/events/slack"
	}

	registry := make([]AgentRegistryEntry, 0, len(cfg.Agent.Registry))
	seenAgentIDs := map[string]struct{}{}
	for _, entry := range cfg.Agent.Registry {
		id := strings.TrimSpace(entry.ID)
		if id == "" {
			continue
		}
		if _, exists := seenAgentIDs[id]; exists {
			continue
		}
		seenAgentIDs[id] = struct{}{}

		item := entry
		item.ID = id
		if strings.TrimSpace(item.Name) == "" {
			if id == "default" {
				item.Name = cfg.Agent.Name
			} else {
				item.Name = "Alter0-" + id
			}
		}
		if strings.TrimSpace(item.Workspace) == "" {
			item.Workspace = "."
		}
		if strings.TrimSpace(item.AgentDir) == "" {
			item.AgentDir = filepath.Join("output", "agents", id)
		}
		if strings.TrimSpace(item.Executor) == "" {
			item.Executor = cfg.Executor.Name
		}
		registry = append(registry, item)
	}

	if len(registry) == 0 {
		registry = append(registry, AgentRegistryEntry{
			ID:        "default",
			Name:      cfg.Agent.Name,
			Workspace: ".",
			AgentDir:  filepath.Join("output", "agents", "default"),
			Executor:  cfg.Executor.Name,
		})
	}

	requestedDefault := strings.TrimSpace(cfg.Agent.DefaultID)
	if requestedDefault == "" {
		requestedDefault = registry[0].ID
	}
	foundDefault := false
	for _, item := range registry {
		if item.ID == requestedDefault {
			foundDefault = true
			break
		}
	}
	if !foundDefault {
		requestedDefault = registry[0].ID
	}

	cfg.Agent.DefaultID = requestedDefault
	cfg.Agent.Registry = registry

	clean := make([]string, 0, len(cfg.Security.AdminUserIDs))
	seen := map[string]struct{}{}
	for _, userID := range cfg.Security.AdminUserIDs {
		trimmed := strings.TrimSpace(userID)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		clean = append(clean, trimmed)
	}
	if len(clean) == 0 {
		fallback := strings.TrimSpace(cfg.Task.CLIUserID)
		if fallback == "" {
			fallback = "local_user"
		}
		clean = append(clean, fallback)
	}
	cfg.Security.AdminUserIDs = clean
	cfg.Security.Tools = sanitizeToolPolicy(cfg.Security.Tools)
}

func sanitizeToolPolicy(policy ToolPolicyConfig) ToolPolicyConfig {
	policy.GlobalAllow = normalizeToolList(policy.GlobalAllow)
	policy.GlobalDeny = normalizeToolList(policy.GlobalDeny)
	policy.RequireConfirm = normalizeToolList(policy.RequireConfirm)
	if len(policy.RequireConfirm) == 0 {
		policy.RequireConfirm = []string{"browser", "canvas", "nodes", "message"}
	}

	if policy.Agent == nil {
		policy.Agent = map[string]ToolAgentPolicy{}
		return policy
	}
	cleanAgent := map[string]ToolAgentPolicy{}
	for rawID, rawPolicy := range policy.Agent {
		agentID := strings.TrimSpace(rawID)
		if agentID == "" {
			continue
		}
		cleanAgent[agentID] = ToolAgentPolicy{
			Allow: normalizeToolList(rawPolicy.Allow),
			Deny:  normalizeToolList(rawPolicy.Deny),
		}
	}
	policy.Agent = cleanAgent
	return policy
}

func normalizeToolList(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}
	set := map[string]struct{}{}
	for _, item := range items {
		name := toolcore.NormalizeToolName(item)
		if strings.TrimSpace(name) == "" {
			continue
		}
		set[name] = struct{}{}
	}
	if len(set) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(set))
	for item := range set {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

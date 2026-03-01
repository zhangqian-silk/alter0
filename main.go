package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	config "alter0/app/configs"
	"alter0/app/core/interaction/cli"
	coregateway "alter0/app/core/interaction/gateway"
	"alter0/app/core/interaction/http"
	"alter0/app/core/interaction/slack"
	"alter0/app/core/interaction/telegram"
	"alter0/app/core/orchestrator/agent"
	"alter0/app/core/orchestrator/skills"
	"alter0/app/core/orchestrator/skills/builtins"
	orctask "alter0/app/core/orchestrator/task"
	coreruntime "alter0/app/core/runtime"
	toolruntime "alter0/app/core/runtime/tools"
	schedulesvc "alter0/app/core/service/schedule"
	"alter0/app/pkg/logger"
	"alter0/app/pkg/queue"
	"alter0/app/pkg/scheduler"
)

type registeredAgent struct {
	ID        string
	Workspace string
	AgentDir  string
	Executor  string
	Brain     *agent.DefaultAgent
	TaskStore *orctask.Store
	Database  *orctask.DB
}

func main() {
	if err := logger.Init("output/logs"); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	logger.Info("Alter0 Kernel Starting...")

	cfgManager, err := config.NewManager(config.DefaultPath())
	if err != nil {
		logger.Error("Failed to load config: %v", err)
		os.Exit(1)
	}
	cfg := cfgManager.Get()

	if err := coreruntime.RunPreflight(context.Background(), cfg, "output/db"); err != nil {
		logger.Error("Startup preflight failed: %v", err)
		os.Exit(1)
	}
	logger.Info("Startup preflight checks passed")

	shutdownTimeout := time.Duration(cfg.Runtime.Shutdown.DrainTimeoutSec) * time.Second

	agents, defaultAgent, err := buildAgents(cfgManager, cfg)
	if err != nil {
		logger.Error("Failed to initialize agents: %v", err)
		os.Exit(1)
	}
	defer closeAgentDatabases(agents)
	logger.Info("Initialized %d agents, default=%s", len(agents), defaultAgent.ID)

	gw := coregateway.NewGateway(nil)
	traceRecorder, err := coregateway.NewTraceRecorder(filepath.Join("output", "trace"))
	if err != nil {
		logger.Error("Failed to initialize gateway trace recorder: %v", err)
		os.Exit(1)
	}
	gw.SetTraceRecorder(traceRecorder)
	for _, item := range agents {
		if err := gw.RegisterAgent(item.ID, item.Brain); err != nil {
			logger.Error("Failed to register agent %s: %v", item.ID, err)
			os.Exit(1)
		}
	}
	if err := gw.SetDefaultAgent(cfg.Agent.DefaultID); err != nil {
		logger.Error("Failed to set default agent %s: %v", cfg.Agent.DefaultID, err)
		os.Exit(1)
	}

	executionQueue := queue.New(cfg.Runtime.Queue.Buffer)
	if cfg.Runtime.Queue.Enabled {
		if err := executionQueue.Start(context.Background(), cfg.Runtime.Queue.Workers); err != nil {
			logger.Error("Failed to start execution queue: %v", err)
			os.Exit(1)
		}
		defer func() {
			report, err := executionQueue.StopWithReport(shutdownTimeout)
			logger.Info(
				"Execution queue shutdown drain report: pending=%d in_flight=%d drained=%d timed_out=%t remaining_depth=%d remaining_in_flight=%d elapsed=%s",
				report.PendingAtStart,
				report.InFlightAtStart,
				report.DrainedJobs,
				report.TimedOut,
				report.RemainingDepth,
				report.RemainingFlight,
				report.Elapsed,
			)
			if err != nil {
				logger.Error("Execution queue shutdown timeout: %v", err)
			}
		}()
	}
	gw.SetExecutionQueue(executionQueue, coregateway.QueueOptions{
		Enabled:        cfg.Runtime.Queue.Enabled,
		EnqueueTimeout: time.Duration(cfg.Runtime.Queue.EnqueueTimeoutSec) * time.Second,
		AttemptTimeout: time.Duration(cfg.Runtime.Queue.AttemptTimeoutSec) * time.Second,
		MaxRetries:     cfg.Runtime.Queue.MaxRetries,
		RetryDelay:     time.Duration(cfg.Runtime.Queue.RetryDelaySec) * time.Second,
	})

	cliChannel := cli.NewCLIChannel(cfg.Task.CLIUserID)
	gw.RegisterChannel(cliChannel)

	httpChannel := http.NewHTTPChannel(8080)
	httpChannel.SetShutdownTimeout(shutdownTimeout)
	httpChannel.SetSubagentAnnouncer(func(ctx context.Context, announcement http.SubagentAnnouncement) error {
		meta := map[string]interface{}{
			"subagent_id": announcement.SubagentID,
			"mode":        announcement.Mode,
			"turn":        announcement.Turn,
			"status":      announcement.Status,
		}
		return gw.DeliverDirect(ctx, announcement.ChannelID, announcement.To, announcement.Message, meta)
	})
	gw.RegisterChannel(httpChannel)

	if cfg.Channels.Telegram.Enabled {
		telegramChannel := telegram.NewChannel(telegram.Config{
			BotToken:       cfg.Channels.Telegram.BotToken,
			PollInterval:   time.Duration(cfg.Channels.Telegram.PollIntervalSec) * time.Second,
			TimeoutSeconds: cfg.Channels.Telegram.TimeoutSec,
			DefaultChatID:  cfg.Channels.Telegram.DefaultChatID,
			APIRoot:        cfg.Channels.Telegram.APIBaseURL,
		})
		gw.RegisterChannel(telegramChannel)
	}

	if cfg.Channels.Slack.Enabled {
		slackChannel := slack.NewChannel(slack.Config{
			BotToken:       cfg.Channels.Slack.BotToken,
			AppID:          cfg.Channels.Slack.AppID,
			ListenPort:     cfg.Channels.Slack.EventListenPort,
			EventPath:      cfg.Channels.Slack.EventPath,
			DefaultChannel: cfg.Channels.Slack.DefaultChannelID,
			APIRoot:        cfg.Channels.Slack.APIBaseURL,
		})
		gw.RegisterChannel(slackChannel)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobScheduler := scheduler.New()
	if err := coreruntime.RegisterMaintenanceJobs(jobScheduler, defaultAgent.TaskStore, coreruntime.MaintenanceOptions{
		Enabled:                     cfg.Runtime.Maintenance.Enabled,
		TaskMemoryRetentionDays:     cfg.Runtime.Maintenance.TaskMemoryRetentionDays,
		TaskMemoryOpenRetentionDays: cfg.Runtime.Maintenance.TaskMemoryOpenRetentionDays,
		PruneInterval:               time.Duration(cfg.Runtime.Maintenance.TaskMemoryPruneIntervalSec) * time.Second,
		PruneTimeout:                time.Duration(cfg.Runtime.Maintenance.TaskMemoryPruneTimeoutSec) * time.Second,
	}); err != nil {
		logger.Error("Failed to register maintenance jobs: %v", err)
		os.Exit(1)
	}
	scheduleStore, err := schedulesvc.NewStore(defaultAgent.Database.Conn())
	if err != nil {
		logger.Error("Failed to initialize schedule store: %v", err)
		os.Exit(1)
	}
	scheduleService := schedulesvc.NewService(scheduleStore, gw)
	if err := coreruntime.RegisterScheduleJobs(jobScheduler, scheduleService, coreruntime.ScheduleOptions{}); err != nil {
		logger.Error("Failed to register schedule jobs: %v", err)
		os.Exit(1)
	}
	if err := jobScheduler.Start(ctx); err != nil {
		logger.Error("Failed to start scheduler: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err := jobScheduler.Stop(shutdownTimeout); err != nil {
			logger.Error("Scheduler shutdown timeout: %v", err)
		}
	}()
	httpChannel.SetScheduleService(scheduleService)

	toolRuntime := toolruntime.NewRuntime(toolruntime.Config{
		GlobalAllow:    cfg.Security.Tools.GlobalAllow,
		GlobalDeny:     cfg.Security.Tools.GlobalDeny,
		RequireConfirm: cfg.Security.Tools.RequireConfirm,
		Agent:          toToolAgentPolicies(cfg.Security.Tools.Agent),
		Memory: toolruntime.MemoryPolicy{
			TrustedChannels: append([]string(nil), cfg.Security.Memory.TrustedChannels...),
			RestrictedPaths: append([]string(nil), cfg.Security.Memory.RestrictedPaths...),
		},
	}, filepath.Join("output", "audit"))

	statusCollector := &coreruntime.StatusCollector{
		GatewayStatusProvider:      func() interface{} { return gw.HealthStatus() },
		Scheduler:                  jobScheduler,
		Queue:                      executionQueue,
		TaskStore:                  defaultAgent.TaskStore,
		ScheduleService:            scheduleService,
		RepoPath:                   ".",
		GatewayTraceBasePath:       filepath.Join("output", "trace"),
		GatewayTraceWindow:         30 * time.Minute,
		AlertRetryStormWindow:      10 * time.Minute,
		AlertRetryStormMinimum:     3,
		AlertQueueBacklogRatio:     0.8,
		AlertQueueBacklogDepth:     8,
		AlertExecutorStrictMode:    true,
		AlertSessionCostShare:      0.35,
		AlertSessionCostMinTokens:  1200,
		AlertSessionPromptOutRatio: 6.0,
		RiskWatchlistPath:          filepath.Join("config", "risk-watchlist.json"),
		RiskWatchlistStaleAfter:    72 * time.Hour,
		AgentEntries:               runtimeAgentEntries(agents),
		ToolRuntime:                toolRuntime,
	}
	httpChannel.SetStatusProvider(statusCollector.Snapshot)
	for _, item := range agents {
		item.Brain.SetStatusProvider(statusCollector.Snapshot)
	}

	go runGatewayWithRetry(ctx, gw)

	logger.Info("Alter0 is ready to serve.")
	fmt.Println("- CLI Interface: Interactive")
	fmt.Println("- HTTP Interface: http://localhost:8080/api/message (POST), /api/tasks (POST)")
	fmt.Println("- Web Console:    http://localhost:8080/")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan
	logger.Info("Received signal: %v. Alter0 shutting down with drain timeout=%s", sig, shutdownTimeout)
	cancel()
}

func buildAgents(cfgManager *config.Manager, cfg config.Config) ([]registeredAgent, registeredAgent, error) {
	agents := make([]registeredAgent, 0, len(cfg.Agent.Registry))
	var defaultAgent registeredAgent

	for _, entry := range cfg.Agent.Registry {
		workspace := stringsOrDefault(entry.Workspace, ".")
		if !filepath.IsAbs(workspace) {
			workspace = filepath.Clean(filepath.Join(".", workspace))
		}
		agentDir := stringsOrDefault(entry.AgentDir, filepath.Join("output", "agents", entry.ID))
		if !filepath.IsAbs(agentDir) {
			agentDir = filepath.Clean(filepath.Join(".", agentDir))
		}
		executorName := stringsOrDefault(entry.Executor, cfg.Executor.Name)
		agentName := stringsOrDefault(entry.Name, cfg.Agent.Name)

		database, err := orctask.NewSQLiteDB(filepath.Join(agentDir, "db"))
		if err != nil {
			return nil, registeredAgent{}, fmt.Errorf("agent %s: %w", entry.ID, err)
		}
		taskStore := orctask.NewStore(database)
		skillMgr := skills.NewManager()
		brain := agent.NewAgent(agentName, skillMgr, taskStore, executorName, cfg.Task, cfg.Security)

		applyConfig := func(updated config.Config) error {
			brain.SetName(updated.Agent.Name)
			brain.SetExecutor(updated.Executor.Name)
			brain.SetTaskConfig(updated.Task)
			brain.SetSecurityConfig(updated.Security)
			return nil
		}
		skillMgr.Register(builtins.NewConfigSkill(cfgManager, applyConfig))
		skillMgr.Register(builtins.NewExecutorSkill(cfgManager, applyConfig))

		item := registeredAgent{
			ID:        entry.ID,
			Workspace: workspace,
			AgentDir:  agentDir,
			Executor:  executorName,
			Brain:     brain,
			TaskStore: taskStore,
			Database:  database,
		}
		agents = append(agents, item)
		if entry.ID == cfg.Agent.DefaultID {
			defaultAgent = item
		}
	}

	if defaultAgent.ID == "" {
		return nil, registeredAgent{}, fmt.Errorf("default agent not found in registry: %s", cfg.Agent.DefaultID)
	}
	return agents, defaultAgent, nil
}

func stringsOrDefault(v string, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func closeAgentDatabases(agents []registeredAgent) {
	for _, item := range agents {
		if item.Database == nil {
			continue
		}
		if err := item.Database.Close(); err != nil {
			logger.Error("Failed to close DB for agent=%s: %v", item.ID, err)
		}
	}
}

func runtimeAgentEntries(agents []registeredAgent) []coreruntime.AgentEntry {
	items := make([]coreruntime.AgentEntry, 0, len(agents))
	for _, item := range agents {
		items = append(items, coreruntime.AgentEntry{
			AgentID:   item.ID,
			Workspace: item.Workspace,
			AgentDir:  item.AgentDir,
			Executor:  item.Executor,
		})
	}
	return items
}

func toToolAgentPolicies(raw map[string]config.ToolAgentPolicy) map[string]toolruntime.AgentPolicy {
	if len(raw) == 0 {
		return map[string]toolruntime.AgentPolicy{}
	}
	out := make(map[string]toolruntime.AgentPolicy, len(raw))
	for agentID, policy := range raw {
		out[agentID] = toolruntime.AgentPolicy{
			Allow: append([]string(nil), policy.Allow...),
			Deny:  append([]string(nil), policy.Deny...),
		}
	}
	return out
}

func runGatewayWithRetry(ctx context.Context, gw *coregateway.DefaultGateway) {
	backoff := time.Second
	maxBackoff := 30 * time.Second
	for {
		err := gw.Start(ctx)
		if err == nil || ctx.Err() != nil {
			return
		}
		logger.Error("Gateway crashed, retrying in %s: %v", backoff, err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

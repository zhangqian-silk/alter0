package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	config "alter0/app/configs"
	"alter0/app/core/interaction/cli"
	"alter0/app/core/interaction/gateway"
	"alter0/app/core/interaction/http"
	"alter0/app/core/orchestrator/agent"
	"alter0/app/core/orchestrator/db"
	"alter0/app/core/orchestrator/skills"
	"alter0/app/core/orchestrator/skills/builtins"
	"alter0/app/core/orchestrator/task"
	"alter0/app/core/runtime"
	"alter0/app/core/scheduler"
	"alter0/app/pkg/logger"
)

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

	database, err := db.NewSQLiteDB("output/db")
	if err != nil {
		logger.Error("Failed to initialize DB: %v", err)
		os.Exit(1)
	}
	defer database.Close()
	logger.Info("Database initialized successfully")

	taskStore := task.NewStore(database)
	skillMgr := skills.NewManager()

	brain := agent.NewAgent(cfg.Agent.Name, skillMgr, taskStore, cfg.Executor.Name, cfg.Task, cfg.Security)

	applyConfig := func(updated config.Config) error {
		brain.SetName(updated.Agent.Name)
		brain.SetExecutor(updated.Executor.Name)
		brain.SetTaskConfig(updated.Task)
		brain.SetSecurityConfig(updated.Security)
		return nil
	}

	configSkill := builtins.NewConfigSkill(cfgManager, applyConfig)
	executorSkill := builtins.NewExecutorSkill(cfgManager, applyConfig)
	skillMgr.Register(configSkill)
	skillMgr.Register(executorSkill)

	gw := gateway.NewGateway(brain)

	cliChannel := cli.NewCLIChannel(cfg.Task.CLIUserID)
	gw.RegisterChannel(cliChannel)

	httpChannel := http.NewHTTPChannel(8080)
	gw.RegisterChannel(httpChannel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobScheduler := scheduler.New()
	if err := runtime.RegisterMaintenanceJobs(jobScheduler, taskStore, runtime.MaintenanceOptions{
		Enabled:                 cfg.Runtime.Maintenance.Enabled,
		TaskMemoryRetentionDays: cfg.Runtime.Maintenance.TaskMemoryRetentionDays,
		PruneInterval:           time.Duration(cfg.Runtime.Maintenance.TaskMemoryPruneIntervalSec) * time.Second,
		PruneTimeout:            time.Duration(cfg.Runtime.Maintenance.TaskMemoryPruneTimeoutSec) * time.Second,
	}); err != nil {
		logger.Error("Failed to register maintenance jobs: %v", err)
		os.Exit(1)
	}
	if err := jobScheduler.Start(ctx); err != nil {
		logger.Error("Failed to start scheduler: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err := jobScheduler.Stop(3 * time.Second); err != nil {
			logger.Error("Scheduler shutdown timeout: %v", err)
		}
	}()

	statusCollector := &runtime.StatusCollector{
		Gateway:   gw,
		Scheduler: jobScheduler,
		TaskStore: taskStore,
		RepoPath:  ".",
	}
	httpChannel.SetStatusProvider(statusCollector.Snapshot)
	brain.SetStatusProvider(statusCollector.Snapshot)

	go runGatewayWithRetry(ctx, gw)

	logger.Info("Alter0 is ready to serve.")
	fmt.Println("- CLI Interface: Interactive")
	fmt.Println("- HTTP Interface: http://localhost:8080/api/message (POST)")
	fmt.Println("- Web Console:    http://localhost:8080/")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan
	logger.Info("Received signal: %v. Alter0 Shutting Down...", sig)
	cancel()
}

func runGatewayWithRetry(ctx context.Context, gw *gateway.DefaultGateway) {
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

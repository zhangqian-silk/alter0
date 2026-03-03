package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	controlapp "alter0/internal/control/application"
	controldomain "alter0/internal/control/domain"
	execapp "alter0/internal/execution/application"
	execinfra "alter0/internal/execution/infrastructure"
	"alter0/internal/interfaces/cli"
	"alter0/internal/interfaces/web"
	orchapp "alter0/internal/orchestration/application"
	orchdomain "alter0/internal/orchestration/domain"
	orchinfra "alter0/internal/orchestration/infrastructure"
	schedulerapp "alter0/internal/scheduler/application"
	sharedapp "alter0/internal/shared/application"
	shareddomain "alter0/internal/shared/domain"
	sharedinfra "alter0/internal/shared/infrastructure/id"
	"alter0/internal/shared/infrastructure/observability"
	localstorage "alter0/internal/storage/infrastructure/localfile"
)

type storageProfile struct {
	Backend         string
	Dir             string
	ControlFormat   localstorage.Format
	SchedulerFormat localstorage.Format
}

var defaultStorageProfile = storageProfile{
	Backend:         "local",
	Dir:             ".alter0",
	ControlFormat:   localstorage.FormatJSON,
	SchedulerFormat: localstorage.FormatJSON,
}

const defaultWebAddr = "127.0.0.1:18088"

func main() {
	webAddr := flag.String("web-addr", defaultWebAddr, "web server listen address")
	workerPoolSize := flag.Int("worker-pool-size", 4, "global worker pool size")
	maxQueueSize := flag.Int("max-queue-size", 128, "max waiting queue size")
	queueTimeout := flag.Duration("queue-timeout", 5*time.Second, "max queue wait time")
	sessionMemoryTurns := flag.Int("session-memory-turns", 6, "short-term memory window size per session")
	sessionMemoryTTL := flag.Duration("session-memory-ttl", 20*time.Minute, "short-term memory ttl per session")
	flag.Parse()
	listenAddr := strings.TrimSpace(*webAddr)
	if listenAddr == "" {
		listenAddr = defaultWebAddr
	}

	rootCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logger := observability.NewLogger(slog.LevelInfo)
	telemetry := observability.NewTelemetry()
	idGen := sharedinfra.NewRandomIDGenerator()

	controlStore, schedulerStore, err := buildStorage(defaultStorageProfile)
	if err != nil {
		logger.Error("failed to initialize storage", slog.String("error", err.Error()))
		os.Exit(2)
	}

	control, err := newControlService(rootCtx, controlStore)
	if err != nil {
		logger.Error("failed to initialize control service", slog.String("error", err.Error()))
		os.Exit(2)
	}
	mustUpsertChannel(control, controldomain.Channel{
		ID:      "cli-default",
		Type:    shareddomain.ChannelTypeCLI,
		Enabled: true,
	})
	mustUpsertChannel(control, controldomain.Channel{
		ID:      "web-default",
		Type:    shareddomain.ChannelTypeWeb,
		Enabled: true,
	})
	mustUpsertChannel(control, controldomain.Channel{
		ID:      "scheduler-default",
		Type:    shareddomain.ChannelTypeScheduler,
		Enabled: true,
	})
	mustUpsertSkill(control, controldomain.Skill{
		ID:      "default-nl",
		Name:    "default-nl",
		Enabled: true,
	})

	registry := orchinfra.NewInMemoryCommandRegistry()
	helpHandler := orchinfra.NewHelpCommandHandler(registry)
	mustRegister(registry, helpHandler)
	mustRegister(registry, orchinfra.NewEchoCommandHandler())
	mustRegister(registry, orchinfra.NewTimeCommandHandler())

	classifier := orchinfra.NewSimpleIntentClassifier(registry)
	processor := execinfra.NewCodexCLIProcessor()
	executor := execapp.NewServiceWithSkills(processor, control, logger)
	baseOrchestrator := orchapp.NewServiceWithOptions(
		classifier,
		registry,
		executor,
		telemetry,
		logger,
		orchapp.WithSessionMemoryOptions(orchapp.SessionMemoryOptions{
			MaxTurns: *sessionMemoryTurns,
			TTL:      *sessionMemoryTTL,
		}),
	)
	orchestrator := orchapp.NewConcurrentService(
		rootCtx,
		baseOrchestrator,
		telemetry,
		logger,
		orchapp.ConcurrencyOptions{
			WorkerCount:    *workerPoolSize,
			MaxQueueSize:   *maxQueueSize,
			QueueTimeout:   *queueTimeout,
			OverloadPolicy: orchapp.OverloadPolicyRejectNew,
		},
	)

	scheduler, err := newSchedulerManager(rootCtx, orchestrator, telemetry, idGen, logger, schedulerStore)
	if err != nil {
		logger.Error("failed to initialize scheduler manager", slog.String("error", err.Error()))
		os.Exit(2)
	}
	scheduler.Start(rootCtx)

	server := web.NewServer(listenAddr, orchestrator, telemetry, idGen, control, scheduler, logger)
	webErrCh := make(chan error, 1)
	go func() {
		logger.Info("starting web server", slog.String("addr", listenAddr))
		webErrCh <- server.Run(rootCtx)
	}()

	go func() {
		runner := cli.NewRunner(orchestrator, telemetry, idGen, logger)
		if err := runner.Run(rootCtx); err != nil {
			logger.Error("cli exited with error", slog.String("error", err.Error()))
			return
		}
		logger.Info("cli adapter stopped")
	}()

	select {
	case err := <-webErrCh:
		if err != nil {
			logger.Error("web server exited with error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	case <-rootCtx.Done():
		if err := <-webErrCh; err != nil {
			logger.Error("web server exited with error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}
}

func mustRegister(registry *orchinfra.InMemoryCommandRegistry, handler orchdomain.CommandHandler) {
	if err := registry.Register(handler); err != nil {
		panic(err)
	}
}

func mustUpsertChannel(control *controlapp.Service, channel controldomain.Channel) {
	if err := control.UpsertChannel(channel); err != nil {
		panic(err)
	}
}

func mustUpsertSkill(control *controlapp.Service, skill controldomain.Skill) {
	if err := control.UpsertSkill(skill); err != nil {
		panic(err)
	}
}

func buildStorage(profile storageProfile) (controlapp.Store, schedulerapp.Store, error) {
	switch strings.ToLower(strings.TrimSpace(profile.Backend)) {
	case "none", "memory", "inmemory":
		return nil, nil, nil
	case "", "local":
		dir := strings.TrimSpace(profile.Dir)
		if dir == "" {
			dir = ".alter0"
		}
		return localstorage.NewControlStore(dir, profile.ControlFormat), localstorage.NewSchedulerStore(dir, profile.SchedulerFormat), nil
	default:
		return nil, nil, fmt.Errorf("unsupported storage backend %q", profile.Backend)
	}
}

func newControlService(ctx context.Context, store controlapp.Store) (*controlapp.Service, error) {
	if store == nil {
		return controlapp.NewService(), nil
	}
	return controlapp.NewServiceWithStore(ctx, store)
}

func newSchedulerManager(
	ctx context.Context,
	orchestrator schedulerapp.Orchestrator,
	telemetry *observability.Telemetry,
	idGen sharedapp.IDGenerator,
	logger *slog.Logger,
	store schedulerapp.Store,
) (*schedulerapp.Manager, error) {
	if store == nil {
		return schedulerapp.NewManager(orchestrator, telemetry, idGen, logger), nil
	}
	return schedulerapp.NewManagerWithStore(ctx, orchestrator, telemetry, idGen, logger, store)
}

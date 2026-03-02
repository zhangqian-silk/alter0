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
	localpersistence "alter0/internal/persistence/infrastructure/localfile"
	schedulerapp "alter0/internal/scheduler/application"
	schedulerdomain "alter0/internal/scheduler/domain"
	shareddomain "alter0/internal/shared/domain"
	sharedinfra "alter0/internal/shared/infrastructure/id"
	"alter0/internal/shared/infrastructure/observability"
)

func main() {
	mode := flag.String("mode", "web", "run mode: web | cli")
	addr := flag.String("addr", "127.0.0.1:8088", "web server listen address")
	cronEvery := flag.String("cron-every", "", "optional default cron interval, e.g. 30s or 5m")
	cronContent := flag.String("cron-content", "/time", "content to send for the default cron job")
	cronSession := flag.String("cron-session", "cron-system", "session id used by the default cron job")
	persistBackend := flag.String("persist-backend", "local", "persistence backend: local | none")
	persistDir := flag.String("persist-dir", ".alter0", "local persistence directory")
	persistFormat := flag.String("persist-format", "json", "local persistence format: json | markdown")
	flag.Parse()

	rootCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logger := observability.NewLogger(slog.LevelInfo)
	telemetry := observability.NewTelemetry()
	idGen := sharedinfra.NewRandomIDGenerator()

	controlStore, schedulerStore, err := buildPersistence(*persistBackend, *persistDir, *persistFormat)
	if err != nil {
		logger.Error("failed to initialize persistence", slog.String("error", err.Error()))
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
	processor := execinfra.NewTemplateNLProcessor()
	executor := execapp.NewService(processor)
	orchestrator := orchapp.NewService(
		classifier,
		registry,
		executor,
		telemetry,
		logger,
	)

	scheduler, err := newSchedulerManager(rootCtx, orchestrator, telemetry, idGen, logger, schedulerStore)
	if err != nil {
		logger.Error("failed to initialize scheduler manager", slog.String("error", err.Error()))
		os.Exit(2)
	}
	scheduler.Start(rootCtx)
	if err := maybeCreateDefaultCronJob(*cronEvery, *cronContent, *cronSession, scheduler); err != nil {
		logger.Error("failed to configure default cron job", slog.String("error", err.Error()))
		os.Exit(2)
	}

	switch *mode {
	case "web":
		server := web.NewServer(*addr, orchestrator, telemetry, idGen, control, scheduler, logger)
		logger.Info("starting web server", slog.String("addr", *addr))
		if err := server.Run(rootCtx); err != nil {
			logger.Error("web server exited with error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	case "cli":
		runner := cli.NewRunner(orchestrator, telemetry, idGen, logger)
		if err := runner.Run(rootCtx); err != nil {
			logger.Error("cli exited with error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unsupported mode %q, expected web or cli\n", *mode)
		os.Exit(2)
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

func maybeCreateDefaultCronJob(
	cronEvery string,
	cronContent string,
	cronSession string,
	scheduler *schedulerapp.Manager,
) error {
	cronEvery = strings.TrimSpace(cronEvery)
	if cronEvery == "" {
		return nil
	}

	interval, err := time.ParseDuration(cronEvery)
	if err != nil {
		return fmt.Errorf("invalid -cron-every: %w", err)
	}

	return scheduler.Upsert(schedulerdomain.Job{
		ID:        "default",
		Name:      "default",
		Interval:  interval,
		Enabled:   true,
		SessionID: strings.TrimSpace(cronSession),
		ChannelID: "scheduler-default",
		Content:   cronContent,
		Metadata: map[string]string{
			"managed_by": "startup_flag",
		},
	})
}

func buildPersistence(
	backend string,
	persistDir string,
	format string,
) (controlapp.Persistence, schedulerapp.Persistence, error) {
	switch strings.ToLower(strings.TrimSpace(backend)) {
	case "none", "memory", "inmemory":
		return nil, nil, nil
	case "", "local":
		parsedFormat, err := localpersistence.ParseFormat(format)
		if err != nil {
			return nil, nil, err
		}
		dir := strings.TrimSpace(persistDir)
		if dir == "" {
			dir = ".alter0"
		}
		return localpersistence.NewControlStore(dir, parsedFormat), localpersistence.NewSchedulerStore(dir, parsedFormat), nil
	default:
		return nil, nil, fmt.Errorf("unsupported persistence backend %q", backend)
	}
}

func newControlService(ctx context.Context, store controlapp.Persistence) (*controlapp.Service, error) {
	if store == nil {
		return controlapp.NewService(), nil
	}
	return controlapp.NewServiceWithPersistence(ctx, store)
}

func newSchedulerManager(
	ctx context.Context,
	orchestrator *orchapp.Service,
	telemetry *observability.Telemetry,
	idGen *sharedinfra.RandomIDGenerator,
	logger *slog.Logger,
	store schedulerapp.Persistence,
) (*schedulerapp.Manager, error) {
	if store == nil {
		return schedulerapp.NewManager(orchestrator, telemetry, idGen, logger), nil
	}
	return schedulerapp.NewManagerWithPersistence(ctx, orchestrator, telemetry, idGen, logger, store)
}

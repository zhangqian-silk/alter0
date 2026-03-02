package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	execapp "alter0/internal/execution/application"
	execinfra "alter0/internal/execution/infrastructure"
	"alter0/internal/interfaces/cli"
	"alter0/internal/interfaces/web"
	orchapp "alter0/internal/orchestration/application"
	orchdomain "alter0/internal/orchestration/domain"
	orchinfra "alter0/internal/orchestration/infrastructure"
	sharedinfra "alter0/internal/shared/infrastructure/id"
	"alter0/internal/shared/infrastructure/observability"
)

func main() {
	mode := flag.String("mode", "web", "run mode: web | cli")
	addr := flag.String("addr", "127.0.0.1:8088", "web server listen address")
	flag.Parse()

	logger := observability.NewLogger(slog.LevelInfo)
	telemetry := observability.NewTelemetry()
	idGen := sharedinfra.NewRandomIDGenerator()

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

	switch *mode {
	case "web":
		server := web.NewServer(*addr, orchestrator, telemetry, idGen, logger)
		logger.Info("starting web server", slog.String("addr", *addr))
		if err := server.Run(context.Background()); err != nil {
			logger.Error("web server exited with error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	case "cli":
		runner := cli.NewRunner(orchestrator, telemetry, idGen, logger)
		if err := runner.Run(context.Background()); err != nil {
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

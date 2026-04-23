package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	agentapp "alter0/internal/agent/application"
	codexapp "alter0/internal/codex/application"
	codexlocal "alter0/internal/codex/infrastructure/localfile"
	controlapp "alter0/internal/control/application"
	controldomain "alter0/internal/control/domain"
	execapp "alter0/internal/execution/application"
	execinfra "alter0/internal/execution/infrastructure"
	"alter0/internal/interfaces/cli"
	"alter0/internal/interfaces/web"
	llmapp "alter0/internal/llm/application"
	llminfra "alter0/internal/llm/infrastructure"
	orchapp "alter0/internal/orchestration/application"
	orchdomain "alter0/internal/orchestration/domain"
	orchinfra "alter0/internal/orchestration/infrastructure"
	schedulerapp "alter0/internal/scheduler/application"
	sessionapp "alter0/internal/session/application"
	sharedapp "alter0/internal/shared/application"
	shareddomain "alter0/internal/shared/domain"
	sharedinfra "alter0/internal/shared/infrastructure/id"
	"alter0/internal/shared/infrastructure/observability"
	localstorage "alter0/internal/storage/infrastructure/localfile"
	taskapp "alter0/internal/task/application"
	tasksummaryapp "alter0/internal/tasksummary/application"
	terminalapp "alter0/internal/terminal/application"
)

type storageProfile struct {
	Backend         string
	Dir             string
	ControlFormat   localstorage.Format
	SchedulerFormat localstorage.Format
	SessionFormat   localstorage.Format
	TaskFormat      localstorage.Format
}

var defaultStorageProfile = storageProfile{
	Backend:         "local",
	Dir:             ".alter0",
	ControlFormat:   localstorage.FormatJSON,
	SchedulerFormat: localstorage.FormatJSON,
	SessionFormat:   localstorage.FormatJSON,
	TaskFormat:      localstorage.FormatJSON,
}

const defaultWebAddr = "127.0.0.1:18088"

const defaultPublicCodexCommand = "/usr/local/bin/codex"
const defaultCodexWorkspaceModeEnvKey = "ALTER0_CODEX_WORKSPACE_MODE"
const defaultCodexWorkspaceMode = "session"

func main() {
	ensureDefaultRuntimePath()
	ensureLocalhostNoProxy()

	relaunchHelper := flag.Bool(relaunchHelperFlag, false, "internal relaunch helper")
	relaunchParentPID := flag.Int(relaunchParentPIDFlag, 0, "internal relaunch parent pid")
	relaunchExecPath := flag.String(relaunchExecPathFlag, "", "internal relaunch executable path")
	relaunchArgs := flag.String(relaunchArgsFlag, "", "internal relaunch encoded args")
	relaunchWorkingDir := flag.String(relaunchWorkingDirFlag, "", "internal relaunch working directory")
	runtimeChild := flag.Bool(runtimeChildFlag, false, "internal runtime child")
	webAddr := flag.String("web-addr", defaultWebAddr, "web server listen address")
	webBindLocalhostOnly := flag.Bool("web-bind-localhost-only", true, "force web server to bind loopback only")
	webLoginPasswordDefault := strings.TrimSpace(os.Getenv("ALTER0_WEB_LOGIN_PASSWORD"))
	webLoginPassword := flag.String("web-login-password", webLoginPasswordDefault, "web login password (empty to disable login page)")
	workerPoolSize := flag.Int("worker-pool-size", 4, "global worker pool size")
	maxQueueSize := flag.Int("max-queue-size", 128, "max waiting queue size")
	queueTimeout := flag.Duration("queue-timeout", 5*time.Second, "max queue wait time")
	codexCommand := flag.String("codex-command", strings.TrimSpace(os.Getenv("ALTER0_CODEX_COMMAND")), "Codex CLI executable path or command name")
	asyncTaskWorkers := flag.Int("async-task-workers", 5, "background async task worker count (max 5)")
	taskTerminalShell := flag.String("task-terminal-shell", "", "terminal Codex CLI executable path or command name")
	taskTerminalShellArgs := flag.String("task-terminal-shell-args", "", "terminal Codex CLI extra arguments")
	asyncTaskTimeout := flag.Duration("async-task-timeout", 90*time.Second, "background async task timeout")
	asyncTaskMaxRetries := flag.Int("async-task-max-retries", 1, "background async task max retries")
	asyncTaskTriggerThreshold := flag.Duration("async-task-trigger-threshold", 5*time.Minute, "estimated duration threshold to route request into async task")
	asyncLongContentThreshold := flag.Int("async-long-content-threshold", 240, "request content length threshold to trigger async task")
	sessionMemoryTurns := flag.Int("session-memory-turns", 6, "short-term memory window size per session")
	sessionMemoryTTL := flag.Duration("session-memory-ttl", 20*time.Minute, "short-term memory ttl per session")
	contextCompressionThreshold := flag.Int("context-compression-threshold", 1200, "estimated token threshold to trigger session context compression")
	contextCompressionSummaryTokens := flag.Int("context-compression-summary-tokens", 220, "estimated token budget per compressed summary fragment")
	contextCompressionRetainTurns := flag.Int("context-compression-retain-turns", 4, "recent turns retained before compressing historical turns")
	dailyMemoryDir := flag.String("daily-memory-dir", filepath.Join(defaultStorageProfile.Dir, "memory"), "day-level markdown memory directory")
	longTermMemoryPath := flag.String("long-term-memory-path", filepath.Join(defaultStorageProfile.Dir, "memory", "long-term", "MEMORY.md"), "tiered long-term memory persistence file path")
	longTermMemoryWritePolicy := flag.String("long-term-memory-write-policy", "write_through", "tiered long-term memory write policy: write_through/write_back")
	longTermMemoryWriteBackFlush := flag.Duration("long-term-memory-writeback-flush", 2*time.Second, "write-back flush interval for long-term memory persistence")
	longTermMemoryTokenBudget := flag.Int("long-term-memory-token-budget", 220, "long-term memory injection token budget")
	mandatoryContextFile := flag.String("mandatory-context-file", "SOUL.md", "mandatory context file path")
	flag.Parse()
	if *relaunchHelper {
		if err := runRelaunchHelper(*relaunchParentPID, *relaunchExecPath, *relaunchWorkingDir, *relaunchArgs); err != nil {
			fmt.Fprintf(os.Stderr, "alter0 relaunch helper failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if !*runtimeChild {
		logger := observability.NewLogger(slog.LevelInfo)
		supervisor, err := newRuntimeSupervisor(logger, filterInternalRuntimeArgs(os.Args[1:]), strings.TrimSpace(*webAddr), *webBindLocalhostOnly)
		if err != nil {
			logger.Error("failed to initialize runtime supervisor", slog.String("error", err.Error()))
			os.Exit(2)
		}

		rootCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		if err := supervisor.Run(rootCtx); err != nil {
			logger.Error("runtime supervisor exited with error", slog.String("error", err.Error()))
			os.Exit(1)
		}
		return
	}

	listenAddr := strings.TrimSpace(*webAddr)
	if listenAddr == "" {
		listenAddr = defaultWebAddr
	}

	rootCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logger := observability.NewLogger(slog.LevelInfo)
	runtimeInfo := newRuntimeInfoProvider(time.Now().UTC(), mustGetwd())
	telemetry := observability.NewTelemetry()
	idGen := sharedinfra.NewRandomIDGenerator()

	controlStore, schedulerStore, sessionStore, taskStore, err := buildStorage(defaultStorageProfile)
	if err != nil {
		logger.Error("failed to initialize storage", slog.String("error", err.Error()))
		os.Exit(2)
	}

	control, err := newControlService(rootCtx, controlStore)
	if err != nil {
		logger.Error("failed to initialize control service", slog.String("error", err.Error()))
		os.Exit(2)
	}

	listenAddr = control.ResolveEnvironmentString("web_addr", listenAddr)
	if strings.TrimSpace(listenAddr) == "" {
		listenAddr = defaultWebAddr
	}
	resolvedWebBindLocalhostOnly := resolveEnvironmentBool(control, "web_bind_localhost_only", *webBindLocalhostOnly)
	if resolvedWebBindLocalhostOnly {
		listenAddr = forceLoopbackListenAddr(listenAddr)
	}
	resolvedWebLoginPassword := resolveRuntimeChildWebLoginPassword(
		*runtimeChild,
		control.ResolveEnvironmentString("web_login_password", strings.TrimSpace(*webLoginPassword)),
	)
	ensureChildProcessWebLoginPassword(resolvedWebLoginPassword)

	resolvedWorkerPoolSize := control.ResolveEnvironmentInt("worker_pool_size", *workerPoolSize)
	resolvedMaxQueueSize := control.ResolveEnvironmentInt("max_queue_size", *maxQueueSize)
	resolvedQueueTimeout := control.ResolveEnvironmentDuration("queue_timeout", *queueTimeout)
	resolvedCodexCommand := resolveConfiguredCodexCommand(strings.TrimSpace(*codexCommand))
	ensureDefaultCodexWorkspaceMode()
	resolvedAsyncTaskWorkers := control.ResolveEnvironmentInt("async_task_workers", *asyncTaskWorkers)
	resolvedTaskTerminalShell := strings.TrimSpace(control.ResolveEnvironmentString("task_terminal_shell", strings.TrimSpace(*taskTerminalShell)))
	if resolvedTaskTerminalShell == "" {
		resolvedTaskTerminalShell = resolvedCodexCommand
	}
	resolvedTaskTerminalShellArgs := strings.TrimSpace(control.ResolveEnvironmentString("task_terminal_shell_args", strings.TrimSpace(*taskTerminalShellArgs)))
	resolvedAsyncTaskTimeout := control.ResolveEnvironmentDuration("async_task_timeout", *asyncTaskTimeout)
	resolvedAsyncTaskMaxRetries := control.ResolveEnvironmentInt("async_task_max_retries", *asyncTaskMaxRetries)
	resolvedAsyncTaskTriggerThreshold := control.ResolveEnvironmentDuration("async_task_trigger_threshold", *asyncTaskTriggerThreshold)
	resolvedAsyncLongContentThreshold := control.ResolveEnvironmentInt("async_long_content_threshold", *asyncLongContentThreshold)
	resolvedSessionMemoryTurns := control.ResolveEnvironmentInt("session_memory_turns", *sessionMemoryTurns)
	resolvedSessionMemoryTTL := control.ResolveEnvironmentDuration("session_memory_ttl", *sessionMemoryTTL)
	resolvedContextCompressionThreshold := control.ResolveEnvironmentInt("context_compression_threshold", *contextCompressionThreshold)
	resolvedContextCompressionSummaryTokens := control.ResolveEnvironmentInt("context_compression_summary_tokens", *contextCompressionSummaryTokens)
	resolvedContextCompressionRetainTurns := control.ResolveEnvironmentInt("context_compression_retain_turns", *contextCompressionRetainTurns)
	resolvedDailyMemoryDir := control.ResolveEnvironmentString("daily_memory_dir", strings.TrimSpace(*dailyMemoryDir))
	resolvedLongTermMemoryPath := control.ResolveEnvironmentString("long_term_memory_path", strings.TrimSpace(*longTermMemoryPath))
	resolvedLongTermMemoryWritePolicy := control.ResolveEnvironmentString("long_term_memory_write_policy", strings.TrimSpace(*longTermMemoryWritePolicy))
	resolvedLongTermMemoryWriteBackFlush := control.ResolveEnvironmentDuration("long_term_memory_writeback_flush", *longTermMemoryWriteBackFlush)
	resolvedLongTermMemoryTokenBudget := control.ResolveEnvironmentInt("long_term_memory_token_budget", *longTermMemoryTokenBudget)
	resolvedMandatoryContextFile := control.ResolveEnvironmentString("mandatory_context_file", strings.TrimSpace(*mandatoryContextFile))

	control.SetEnvironmentRuntime(map[string]string{
		"web_addr":                           listenAddr,
		"web_bind_localhost_only":            strconv.FormatBool(resolvedWebBindLocalhostOnly),
		"web_login_password":                 resolvedWebLoginPassword,
		"worker_pool_size":                   strconv.Itoa(resolvedWorkerPoolSize),
		"max_queue_size":                     strconv.Itoa(resolvedMaxQueueSize),
		"queue_timeout":                      resolvedQueueTimeout.String(),
		"async_task_workers":                 strconv.Itoa(resolvedAsyncTaskWorkers),
		"task_terminal_shell":                resolvedTaskTerminalShell,
		"task_terminal_shell_args":           resolvedTaskTerminalShellArgs,
		"async_task_timeout":                 resolvedAsyncTaskTimeout.String(),
		"async_task_max_retries":             strconv.Itoa(resolvedAsyncTaskMaxRetries),
		"async_task_trigger_threshold":       resolvedAsyncTaskTriggerThreshold.String(),
		"async_long_content_threshold":       strconv.Itoa(resolvedAsyncLongContentThreshold),
		"session_memory_turns":               strconv.Itoa(resolvedSessionMemoryTurns),
		"session_memory_ttl":                 resolvedSessionMemoryTTL.String(),
		"context_compression_threshold":      strconv.Itoa(resolvedContextCompressionThreshold),
		"context_compression_summary_tokens": strconv.Itoa(resolvedContextCompressionSummaryTokens),
		"context_compression_retain_turns":   strconv.Itoa(resolvedContextCompressionRetainTurns),
		"daily_memory_dir":                   resolvedDailyMemoryDir,
		"long_term_memory_path":              resolvedLongTermMemoryPath,
		"long_term_memory_write_policy":      resolvedLongTermMemoryWritePolicy,
		"long_term_memory_writeback_flush":   resolvedLongTermMemoryWriteBackFlush.String(),
		"long_term_memory_token_budget":      strconv.Itoa(resolvedLongTermMemoryTokenBudget),
		"mandatory_context_file":             resolvedMandatoryContextFile,
	})

	sessionHistory, err := newSessionHistory(rootCtx, sessionStore)
	if err != nil {
		logger.Error("failed to initialize session history service", slog.String("error", err.Error()))
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
	registerBuiltinSkills(control)
	if err := ensureBuiltinSkillFiles(); err != nil {
		logger.Error("failed to initialize builtin skill files", slog.String("error", err.Error()))
		os.Exit(2)
	}
	agentCatalog := agentapp.NewCatalog(control)

	registry := orchinfra.NewInMemoryCommandRegistry()
	helpHandler := orchinfra.NewHelpCommandHandler(registry)
	mustRegister(registry, helpHandler)
	mustRegister(registry, orchinfra.NewEchoCommandHandler())
	mustRegister(registry, orchinfra.NewTimeCommandHandler())

	llmStorage := llminfra.NewModelConfigStorage(".alter0/model_config.json")
	llmService := llmapp.NewModelConfigService(llmStorage)

	classifier := orchinfra.NewSimpleIntentClassifier(registry)
	processor := execinfra.NewHybridNLProcessorWithCatalog(execinfra.NewCodexCLIProcessorWithCommand(resolvedCodexCommand), llmService, agentCatalog, logger)
	executor := execapp.NewServiceWithSkills(processor, control, logger)
	taskSummaryMemory := tasksummaryapp.NewStore(tasksummaryapp.Options{})
	taskSummaryRuntime := tasksummaryapp.NewRuntimeMarkdownStore(tasksummaryapp.RuntimeMarkdownOptions{
		DailyDir:    resolvedDailyMemoryDir,
		LongTermDir: filepath.Join(resolvedDailyMemoryDir, "long-term"),
	})
	taskSummaryRecorder := tasksummaryapp.NewRecorderGroup(taskSummaryMemory, taskSummaryRuntime)
	baseOrchestrator := orchapp.NewServiceWithOptions(
		classifier,
		registry,
		executor,
		telemetry,
		logger,
		orchapp.WithSessionMemoryOptions(orchapp.SessionMemoryOptions{
			MaxTurns:                 resolvedSessionMemoryTurns,
			TTL:                      resolvedSessionMemoryTTL,
			CompressionTriggerTokens: resolvedContextCompressionThreshold,
			CompressionSummaryTokens: resolvedContextCompressionSummaryTokens,
			CompressionRetainTurns:   resolvedContextCompressionRetainTurns,
			DailyMemoryDir:           resolvedDailyMemoryDir,
		}),
		orchapp.WithSessionHistoryMemory(sessionHistory),
		orchapp.WithLongTermMemoryOptions(orchapp.LongTermMemoryOptions{
			InjectionTokenBudget: resolvedLongTermMemoryTokenBudget,
			PersistencePath:      resolvedLongTermMemoryPath,
			WritePolicy:          orchapp.LongTermMemoryWritePolicy(strings.ToLower(strings.TrimSpace(resolvedLongTermMemoryWritePolicy))),
			WriteBackFlush:       resolvedLongTermMemoryWriteBackFlush,
		}),
		orchapp.WithMandatoryContextOptions(orchapp.MandatoryContextOptions{
			FilePath: resolvedMandatoryContextFile,
		}),
		orchapp.WithTaskSummaryMemory(taskSummaryMemory),
	)
	persistentOrchestrator := orchapp.NewSessionPersistenceService(baseOrchestrator, sessionHistory, idGen, logger, mustGetwd())
	orchestrator := orchapp.NewConcurrentService(
		rootCtx,
		persistentOrchestrator,
		telemetry,
		logger,
		orchapp.ConcurrencyOptions{
			WorkerCount:    resolvedWorkerPoolSize,
			MaxQueueSize:   resolvedMaxQueueSize,
			QueueTimeout:   resolvedQueueTimeout,
			OverloadPolicy: orchapp.OverloadPolicyRejectNew,
		},
	)
	taskService, err := newTaskService(rootCtx, persistentOrchestrator, sessionHistory, idGen, logger, taskStore, taskapp.Options{
		WorkerCount:           resolvedAsyncTaskWorkers,
		Timeout:               resolvedAsyncTaskTimeout,
		MaxRetries:            resolvedAsyncTaskMaxRetries,
		AsyncTriggerThreshold: resolvedAsyncTaskTriggerThreshold,
		LongContentThreshold:  resolvedAsyncLongContentThreshold,
		SummaryMemory:         taskSummaryRecorder,
		ComplexityPredictor: taskapp.NewOpenAIComplexityPredictor(
			llmService,
			taskapp.NewCodexQuickComplexityPredictorWithCommand(resolvedCodexCommand),
			logger,
		),
	})
	if err != nil {
		logger.Error("failed to initialize task service", slog.String("error", err.Error()))
		os.Exit(2)
	}
	terminalService := terminalapp.NewService(rootCtx, idGen, logger, terminalapp.Options{
		Shell:         resolvedTaskTerminalShell,
		ShellArgsLine: resolvedTaskTerminalShellArgs,
	})

	scheduler, err := newSchedulerManager(rootCtx, orchestrator, telemetry, idGen, logger, schedulerStore)
	if err != nil {
		logger.Error("failed to initialize scheduler manager", slog.String("error", err.Error()))
		os.Exit(2)
	}
	scheduler.Start(rootCtx)

	codexAccounts := newCodexAccountService(logger, resolvedCodexCommand)

	server := web.NewServer(
		listenAddr,
		orchestrator,
		telemetry,
		idGen,
		control,
		scheduler,
		sessionHistory,
		taskService,
		terminalService,
		web.AgentMemoryOptions{
			LongTermPath:         resolvedLongTermMemoryPath,
			DailyDir:             resolvedDailyMemoryDir,
			MandatoryContextPath: resolvedMandatoryContextFile,
			TaskSummaryRuntime:   taskSummaryRuntime,
		},
		web.WebSecurityOptions{
			LoginPassword: resolvedWebLoginPassword,
			BindLocalhost: resolvedWebBindLocalhostOnly,
		},
		llmService,
		agentCatalog,
		logger,
	)
	server.SetCodexAccountService(codexAccounts)
	server.SetRuntimeInfoProvider(runtimeInfo)
	restarter, err := newRuntimeRestarter(cancel, logger, filterInternalRuntimeArgs(os.Args[1:]))
	if err != nil {
		logger.Error("failed to initialize service restarter", slog.String("error", err.Error()))
		os.Exit(2)
	}
	server.SetRuntimeRestarter(restarter)
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

func mustGetwd() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return dir
}

func ensureDefaultCodexWorkspaceMode() {
	if strings.TrimSpace(os.Getenv(defaultCodexWorkspaceModeEnvKey)) != "" {
		return
	}
	_ = os.Setenv(defaultCodexWorkspaceModeEnvKey, defaultCodexWorkspaceMode)
}

func ensureDefaultRuntimePath() {
	desiredPath := buildDefaultRuntimePath(strings.TrimSpace(os.Getenv("HOME")), os.Getenv("PATH"))
	if strings.TrimSpace(desiredPath) == "" {
		return
	}
	_ = os.Setenv("PATH", desiredPath)
}

func ensureLocalhostNoProxy() {
	existing := strings.TrimSpace(os.Getenv("NO_PROXY"))
	if existing == "" {
		existing = strings.TrimSpace(os.Getenv("no_proxy"))
	}
	merged := mergeNoProxyEntries(existing, "127.0.0.1", "localhost")
	if strings.TrimSpace(merged) == "" {
		return
	}
	_ = os.Setenv("NO_PROXY", merged)
	_ = os.Setenv("no_proxy", merged)
}

func ensureChildProcessWebLoginPassword(password string) {
	trimmed := strings.TrimSpace(password)
	if trimmed == "" {
		_ = os.Unsetenv("ALTER0_WEB_LOGIN_PASSWORD")
		return
	}
	_ = os.Setenv("ALTER0_WEB_LOGIN_PASSWORD", trimmed)
}

func resolveRuntimeChildWebLoginPassword(runtimeChild bool, password string) string {
	if runtimeChild {
		return ""
	}
	return strings.TrimSpace(password)
}

func mergeNoProxyEntries(existing string, required ...string) string {
	seen := make(map[string]struct{})
	merged := make([]string, 0, len(required)+4)
	appendEntry := func(value string) {
		entry := strings.TrimSpace(value)
		if entry == "" {
			return
		}
		lower := strings.ToLower(entry)
		if _, ok := seen[lower]; ok {
			return
		}
		seen[lower] = struct{}{}
		merged = append(merged, entry)
	}

	for _, value := range strings.Split(existing, ",") {
		appendEntry(value)
	}
	for _, value := range required {
		appendEntry(value)
	}
	return strings.Join(merged, ",")
}

func buildDefaultRuntimePath(home string, existing string) string {
	candidates := make([]string, 0, 8)
	if strings.TrimSpace(home) != "" {
		candidates = append(candidates,
			filepath.Join(home, ".local", "bin"),
			filepath.Join(home, ".local", "share", "pnpm"),
		)
	}
	candidates = append(candidates,
		"/usr/local/bin",
		"/usr/bin",
		"/bin",
		"/usr/local/sbin",
		"/usr/sbin",
		"/sbin",
	)

	seen := make(map[string]struct{})
	merged := make([]string, 0, len(candidates)+8)
	appendDir := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		seenKey := runtimePathSeenKey(path)
		if _, ok := seen[seenKey]; ok {
			return
		}
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			merged = append(merged, path)
			seen[seenKey] = struct{}{}
		}
	}

	for _, path := range candidates {
		appendDir(path)
	}
	for _, path := range filepath.SplitList(existing) {
		appendDir(path)
	}
	return strings.Join(merged, string(os.PathListSeparator))
}

func runtimePathSeenKey(path string) string {
	key := filepath.Clean(strings.TrimSpace(path))
	if runtime.GOOS == "windows" {
		key = strings.ToLower(key)
	}
	return key
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

func resolveEnvironmentBool(control *controlapp.Service, key string, fallback bool) bool {
	resolved := strings.TrimSpace(control.ResolveEnvironmentString(key, strconv.FormatBool(fallback)))
	if resolved == "" {
		return fallback
	}
	value, err := strconv.ParseBool(resolved)
	if err != nil {
		return fallback
	}
	return value
}

func forceLoopbackListenAddr(raw string) string {
	addr := strings.TrimSpace(raw)
	if addr == "" {
		return defaultWebAddr
	}
	if strings.HasPrefix(addr, ":") {
		return "127.0.0.1" + addr
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return defaultWebAddr
	}
	if !isLoopbackHost(host) {
		return net.JoinHostPort("127.0.0.1", port)
	}
	return addr
}

func resolveConfiguredCodexCommand(raw string) string {
	command := strings.TrimSpace(raw)
	if command != "" {
		return command
	}
	if isExecutableFile(defaultPublicCodexCommand) {
		return defaultPublicCodexCommand
	}
	return "codex"
}

func isExecutableFile(path string) bool {
	info, err := os.Stat(strings.TrimSpace(path))
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&0o111 != 0
}

func isLoopbackHost(rawHost string) bool {
	host := strings.Trim(strings.TrimSpace(rawHost), "[]")
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

func buildStorage(profile storageProfile) (controlapp.Store, schedulerapp.Store, sessionapp.Store, taskapp.Store, error) {
	switch strings.ToLower(strings.TrimSpace(profile.Backend)) {
	case "none", "memory", "inmemory":
		return nil, nil, nil, nil, nil
	case "", "local":
		dir := strings.TrimSpace(profile.Dir)
		if dir == "" {
			dir = ".alter0"
		}
		return localstorage.NewControlStore(dir, profile.ControlFormat), localstorage.NewSchedulerStore(dir, profile.SchedulerFormat), localstorage.NewSessionStore(dir, profile.SessionFormat), localstorage.NewTaskStore(dir, profile.TaskFormat), nil
	default:
		return nil, nil, nil, nil, fmt.Errorf("unsupported storage backend %q", profile.Backend)
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

func newSessionHistory(ctx context.Context, store sessionapp.Store) (*sessionapp.Service, error) {
	if store == nil {
		return sessionapp.NewService(), nil
	}
	return sessionapp.NewServiceWithStore(ctx, store)
}

func newTaskService(
	ctx context.Context,
	orchestrator taskapp.Orchestrator,
	recorder *sessionapp.Service,
	idGen sharedapp.IDGenerator,
	logger *slog.Logger,
	store taskapp.Store,
	options taskapp.Options,
) (*taskapp.Service, error) {
	return taskapp.NewService(ctx, orchestrator, recorder, idGen, logger, store, options)
}

func newCodexAccountService(logger *slog.Logger, command string) *codexapp.Service {
	activeHome, err := codexapp.ResolveActiveHome()
	if err != nil {
		if logger != nil {
			logger.Warn("failed to resolve active codex home for account manager", slog.String("error", err.Error()))
		}
		return nil
	}
	store, err := codexlocal.NewStore(filepath.Join(activeHome, "alter0-accounts"))
	if err != nil {
		if logger != nil {
			logger.Warn("failed to initialize codex account store", slog.String("error", err.Error()))
		}
		return nil
	}
	return codexapp.NewService(codexapp.ServiceOptions{
		Store:   store,
		Command: command,
	})
}

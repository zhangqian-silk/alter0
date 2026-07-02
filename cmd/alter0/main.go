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
	"strings"
	"syscall"
	"time"

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
const storageDirEnvKey = "ALTER0_STORAGE_DIR"

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
	webLoginPassword := flag.String("web-login-password", webLoginPasswordDefault, "required web login password for the shared gateway")
	codexCommand := flag.String("codex-command", strings.TrimSpace(os.Getenv("ALTER0_CODEX_COMMAND")), "Codex CLI executable path or command name")
	storageDirDefault := strings.TrimSpace(os.Getenv(storageDirEnvKey))
	if storageDirDefault == "" {
		storageDirDefault = defaultStorageProfile.Dir
	}
	storageDir := flag.String("storage-dir", storageDirDefault, "local storage directory for control, sessions, tasks, scheduler, model config, and runtime memory")
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

	storageProfile := defaultStorageProfile
	storageProfile.Dir = strings.TrimSpace(*storageDir)
	if storageProfile.Dir == "" {
		storageProfile.Dir = defaultStorageProfile.Dir
	}

	controlStore, schedulerStore, sessionStore, taskStore, err := buildStorage(storageProfile)
	if err != nil {
		logger.Error("failed to initialize storage", slog.String("error", err.Error()))
		os.Exit(2)
	}

	control, err := newControlService(rootCtx, controlStore)
	if err != nil {
		logger.Error("failed to initialize control service", slog.String("error", err.Error()))
		os.Exit(2)
	}

	if strings.TrimSpace(listenAddr) == "" {
		listenAddr = defaultWebAddr
	}
	resolvedWebBindLocalhostOnly := *webBindLocalhostOnly
	if resolvedWebBindLocalhostOnly {
		listenAddr = forceLoopbackListenAddr(listenAddr)
	}
	if err := validateRequiredWebLoginPassword(*runtimeChild, strings.TrimSpace(*webLoginPassword)); err != nil {
		logger.Error("invalid web login configuration", slog.String("error", err.Error()))
		os.Exit(2)
	}
	resolvedWebLoginPassword := resolveRuntimeChildWebLoginPassword(
		*runtimeChild,
		strings.TrimSpace(*webLoginPassword),
	)
	ensureChildProcessWebLoginPassword(resolvedWebLoginPassword)

	resolvedCodexCommand := resolveConfiguredCodexCommand(strings.TrimSpace(*codexCommand))
	ensureDefaultCodexWorkspaceMode()
	resolvedTaskTerminalShell := resolvedCodexCommand
	resolvedDailyMemoryDir := filepath.Join(storageProfile.Dir, "memory")
	resolvedLongTermMemoryPath := filepath.Join(storageProfile.Dir, "memory", "long-term", "MEMORY.md")
	resolvedMandatoryContextFile := "SOUL.md"

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
	registry := orchinfra.NewInMemoryCommandRegistry()
	helpHandler := orchinfra.NewHelpCommandHandler(registry)
	mustRegister(registry, helpHandler)
	mustRegister(registry, orchinfra.NewEchoCommandHandler())
	mustRegister(registry, orchinfra.NewTimeCommandHandler())

	llmStorage := llminfra.NewModelConfigStorage(filepath.Join(storageProfile.Dir, "model_config.json"))
	llmService := llmapp.NewModelConfigService(llmStorage)

	classifier := orchinfra.NewSimpleIntentClassifier(registry)
	codexProcessor := execinfra.NewCodexCLIProcessorWithCommand(resolvedCodexCommand)
	claudeProcessor := execinfra.NewClaudeCodeProcessor()
	processor := execinfra.NewRuntimeResolverProcessor(execinfra.RuntimeResolverOptions{
		ProviderSource: llmService,
		Claude:         claudeProcessor,
		Codex:          codexProcessor,
		Logger:         logger,
	})
	executor := execapp.NewServiceWithSkillsAndMemoryOptions(processor, control, logger, execapp.MemoryContextOptions{
		DailyDir:          resolvedDailyMemoryDir,
		LongTermPath:      resolvedLongTermMemoryPath,
		MandatoryFilePath: resolvedMandatoryContextFile,
	})
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
		orchapp.WithLongTermMemoryOptions(orchapp.LongTermMemoryOptions{
			PersistencePath: resolvedLongTermMemoryPath,
		}),
		orchapp.WithMandatoryContextOptions(orchapp.MandatoryContextOptions{
			FilePath: resolvedMandatoryContextFile,
		}),
		orchapp.WithTaskSummaryMemory(taskSummaryMemory),
	)
	persistentOrchestrator := orchapp.NewSessionPersistenceService(baseOrchestrator, sessionHistory, idGen, logger, mustGetwd())
	orchestrator := persistentOrchestrator
	taskService, err := newTaskService(rootCtx, taskStore, taskapp.Options{
		SummaryMemory: taskSummaryRecorder,
	})
	if err != nil {
		logger.Error("failed to initialize task service", slog.String("error", err.Error()))
		os.Exit(2)
	}
	terminalService := terminalapp.NewService(rootCtx, idGen, logger, terminalapp.Options{
		Shell:         resolvedTaskTerminalShell,
		ShellArgsLine: "",
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
		web.MemoryContextOptions{
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
	if runtimeChildSkipsWebLoginPassword(runtimeChild) {
		return ""
	}
	return strings.TrimSpace(password)
}

func validateRequiredWebLoginPassword(runtimeChild bool, password string) error {
	if runtimeChildSkipsWebLoginPassword(runtimeChild) {
		return nil
	}
	if strings.TrimSpace(password) == "" {
		return fmt.Errorf("web_login_password is required; anonymous web access is disabled")
	}
	return nil
}

func reuseGatewayAuthEnabled() bool {
	value := strings.TrimSpace(os.Getenv("ALTER0_WEB_REUSE_GATEWAY_AUTH"))
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func runtimeChildUsesSupervisorControl() bool {
	return strings.TrimSpace(os.Getenv(supervisorAddrEnv)) != "" && strings.TrimSpace(os.Getenv(supervisorTokenEnv)) != ""
}

func runtimeChildSkipsWebLoginPassword(runtimeChild bool) bool {
	return runtimeChild && (reuseGatewayAuthEnabled() || !runtimeChildUsesSupervisorControl())
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
	store taskapp.Store,
	options taskapp.Options,
) (*taskapp.Service, error) {
	return taskapp.NewService(ctx, store, options)
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

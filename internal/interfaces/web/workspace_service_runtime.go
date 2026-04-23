package web

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	workspaceServiceHealthProbeInterval = 500 * time.Millisecond
	workspaceServiceHealthProbeTimeout  = 20 * time.Second
)

type workspaceServiceRuntimeStatus struct {
	RuntimeDir string `json:"runtime_dir,omitempty"`
	LogPath    string `json:"log_path,omitempty"`
	PID        int    `json:"pid,omitempty"`
	Status     string `json:"status,omitempty"`
}

type workspaceServiceRuntime interface {
	EnsureStarted(entry workspaceServiceRegistration) (workspaceServiceRegistration, workspaceServiceRuntimeStatus, error)
	Stop(entry workspaceServiceRegistration) error
}

type managedWorkspaceServiceProcess struct {
	entry      workspaceServiceRegistration
	cmd        *exec.Cmd
	runtimeDir string
	logPath    string
	logFile    *os.File
}

type workspaceServiceProcessManager struct {
	logger        *slog.Logger
	client        *http.Client
	commandRunner func(name string, args ...string) *exec.Cmd

	mu        sync.Mutex
	processes map[string]*managedWorkspaceServiceProcess
}

func newWorkspaceServiceRuntime(logger *slog.Logger) workspaceServiceRuntime {
	return &workspaceServiceProcessManager{
		logger: logger,
		client: &http.Client{
			Timeout: 2 * time.Second,
		},
		commandRunner: exec.Command,
		processes:     map[string]*managedWorkspaceServiceProcess{},
	}
}

func (m *workspaceServiceProcessManager) EnsureStarted(entry workspaceServiceRegistration) (workspaceServiceRegistration, workspaceServiceRuntimeStatus, error) {
	if !isManagedWorkspaceService(entry) {
		return entry, workspaceServiceRuntimeStatus{Status: "external"}, nil
	}

	key := workspaceServiceKey(entry.SessionID, entry.ServiceID)

	m.mu.Lock()
	defer m.mu.Unlock()

	if current, ok := m.processes[key]; ok {
		if sameManagedWorkspaceServiceConfig(current.entry, entry) {
			if err := m.checkHealth(entry); err == nil {
				return entry, current.status("running"), nil
			}
		}
		m.stopProcessLocked(current)
		delete(m.processes, key)
	}

	process, err := m.startProcessLocked(key, entry)
	if err != nil {
		return entry, workspaceServiceRuntimeStatus{}, err
	}
	m.processes[key] = process
	return entry, process.status("running"), nil
}

func (m *workspaceServiceProcessManager) Stop(entry workspaceServiceRegistration) error {
	if !isManagedWorkspaceService(entry) {
		return nil
	}

	key := workspaceServiceKey(entry.SessionID, entry.ServiceID)
	m.mu.Lock()
	defer m.mu.Unlock()

	current, ok := m.processes[key]
	if !ok {
		return nil
	}
	m.stopProcessLocked(current)
	delete(m.processes, key)
	return nil
}

func (m *workspaceServiceProcessManager) startProcessLocked(key string, entry workspaceServiceRegistration) (*managedWorkspaceServiceProcess, error) {
	runtimeDir := workspaceServiceRuntimeDir(entry)
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir workspace service runtime dir: %w", err)
	}

	logPath := filepath.Join(runtimeDir, "service.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open workspace service log: %w", err)
	}

	cmd := m.commandRunner("bash", "-lc", entry.StartCommand)
	cmd.Dir = filepath.FromSlash(entry.Workdir)
	cmd.Env = workspaceServiceProcessEnv(entry.Port)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("start workspace service process: %w", err)
	}

	process := &managedWorkspaceServiceProcess{
		entry:      entry,
		cmd:        cmd,
		runtimeDir: runtimeDir,
		logPath:    logPath,
		logFile:    logFile,
	}

	go m.waitForExit(key, process)

	if err := m.waitUntilHealthy(entry); err != nil {
		m.stopProcessLocked(process)
		return nil, err
	}
	return process, nil
}

func (m *workspaceServiceProcessManager) waitForExit(key string, process *managedWorkspaceServiceProcess) {
	err := process.cmd.Wait()
	if process.logFile != nil {
		_ = process.logFile.Close()
	}
	if err != nil && m.logger != nil {
		m.logger.Error("workspace service process exited",
			"session_id", process.entry.SessionID,
			"service_id", process.entry.ServiceID,
			"error", err.Error(),
		)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	current, ok := m.processes[key]
	if ok && current == process {
		delete(m.processes, key)
	}
}

func (m *workspaceServiceProcessManager) waitUntilHealthy(entry workspaceServiceRegistration) error {
	deadline := time.Now().Add(workspaceServiceHealthProbeTimeout)
	var lastErr error
	for time.Now().Before(deadline) {
		lastErr = m.checkHealth(entry)
		if lastErr == nil {
			return nil
		}
		time.Sleep(workspaceServiceHealthProbeInterval)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("workspace service health probe timed out")
	}
	return fmt.Errorf("workspace service failed health probe: %w", lastErr)
}

func (m *workspaceServiceProcessManager) checkHealth(entry workspaceServiceRegistration) error {
	healthURL := strings.TrimRight(entry.UpstreamURL, "/") + normalizeWorkspaceServiceHealthPath(entry.HealthPath)
	resp, err := m.client.Get(healthURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("unexpected health status %d", resp.StatusCode)
	}
	return nil
}

func (m *workspaceServiceProcessManager) stopProcessLocked(process *managedWorkspaceServiceProcess) {
	if process == nil || process.cmd == nil || process.cmd.Process == nil {
		return
	}
	_ = process.cmd.Process.Signal(syscall.SIGTERM)
	time.Sleep(200 * time.Millisecond)
	_ = process.cmd.Process.Kill()
}

func (p *managedWorkspaceServiceProcess) status(state string) workspaceServiceRuntimeStatus {
	status := workspaceServiceRuntimeStatus{
		RuntimeDir: p.runtimeDir,
		LogPath:    p.logPath,
		Status:     state,
	}
	if p.cmd != nil && p.cmd.Process != nil {
		status.PID = p.cmd.Process.Pid
	}
	return status
}

func isManagedWorkspaceService(entry workspaceServiceRegistration) bool {
	return entry.ServiceType == workspaceServiceTypeHTTP && strings.TrimSpace(entry.StartCommand) != ""
}

func sameManagedWorkspaceServiceConfig(left workspaceServiceRegistration, right workspaceServiceRegistration) bool {
	return left.UpstreamURL == right.UpstreamURL &&
		left.StartCommand == right.StartCommand &&
		left.Workdir == right.Workdir &&
		left.HealthPath == right.HealthPath &&
		left.Port == right.Port
}

func workspaceServiceRuntimeDir(entry workspaceServiceRegistration) string {
	return filepath.Join(
		filepath.FromSlash(entry.Workdir),
		".alter0",
		"test-services",
		entry.SessionID,
		entry.ServiceID,
	)
}

func workspaceServiceProcessEnv(port int) []string {
	env := withoutEnvKey(os.Environ(), "ALTER0_WEB_LOGIN_PASSWORD")
	env = append(env, fmt.Sprintf("PORT=%d", port))
	env = append(env, fmt.Sprintf("ALTER0_SERVICE_PORT=%d", port))
	return env
}

func withoutEnvKey(env []string, key string) []string {
	prefix := strings.TrimSpace(key) + "="
	if prefix == "=" {
		return append([]string{}, env...)
	}
	filtered := make([]string, 0, len(env))
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

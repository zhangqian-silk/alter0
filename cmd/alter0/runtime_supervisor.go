package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"alter0/internal/interfaces/web"
)

const (
	runtimeChildFlag            = "internal-runtime-child"
	supervisorAddrEnv           = "ALTER0_INTERNAL_SUPERVISOR_ADDR"
	supervisorTokenEnv          = "ALTER0_INTERNAL_SUPERVISOR_TOKEN"
	supervisorTokenHeader       = "X-Alter0-Supervisor-Token"
	supervisorShutdownTimeout   = 5 * time.Second
	runtimeRestartStopTimeout   = 20 * time.Second
	runtimeReadyTimeout         = 45 * time.Second
	runtimeProbeInterval        = 1 * time.Second
	runtimeControlClientTimeout = 5 * time.Minute
)

type runtimeRestartClient interface {
	RequestRestart(options web.RuntimeRestartOptions) (bool, error)
}

type supervisorClientRestarter struct {
	addr   string
	token  string
	client *http.Client
}

func newRuntimeRestarter(cancel context.CancelFunc, logger *slog.Logger, args []string) (runtimeRestartClient, error) {
	addr := strings.TrimSpace(os.Getenv(supervisorAddrEnv))
	token := strings.TrimSpace(os.Getenv(supervisorTokenEnv))
	if addr != "" && token != "" {
		return &supervisorClientRestarter{
			addr:  addr,
			token: token,
			client: &http.Client{
				Timeout: runtimeControlClientTimeout,
			},
		}, nil
	}
	return newServiceRestarter(cancel, logger, args)
}

func (r *supervisorClientRestarter) RequestRestart(options web.RuntimeRestartOptions) (bool, error) {
	if r == nil {
		return false, errors.New("runtime restarter is required")
	}
	payload, err := json.Marshal(options)
	if err != nil {
		return false, fmt.Errorf("marshal runtime restart options: %w", err)
	}

	endpoint, err := url.Parse(r.addr)
	if err != nil {
		return false, fmt.Errorf("parse supervisor address: %w", err)
	}
	endpoint.Path = "/restart"
	req, err := http.NewRequest(http.MethodPost, endpoint.String(), bytes.NewReader(payload))
	if err != nil {
		return false, fmt.Errorf("create supervisor request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(supervisorTokenHeader, r.token)

	resp, err := r.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("request supervisor restart: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 32*1024))
	if readErr != nil {
		return false, fmt.Errorf("read supervisor response: %w", readErr)
	}

	var payloadResp struct {
		Accepted bool   `json:"accepted"`
		Error    string `json:"error"`
	}
	_ = json.Unmarshal(body, &payloadResp)
	if resp.StatusCode == http.StatusAccepted {
		return true, nil
	}
	if resp.StatusCode == http.StatusConflict {
		return false, nil
	}
	message := strings.TrimSpace(payloadResp.Error)
	if message == "" {
		message = strings.TrimSpace(string(body))
	}
	if message == "" {
		message = fmt.Sprintf("supervisor returned HTTP %d", resp.StatusCode)
	}
	return false, errors.New(message)
}

type managedChild struct {
	executable string
	args       []string
	cmd        *exec.Cmd
	done       chan struct{}

	mu      sync.RWMutex
	waitErr error
}

func (c *managedChild) setWaitErr(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.waitErr = err
}

func (c *managedChild) waitErrValue() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.waitErr
}

type runtimeSupervisor struct {
	logger             *slog.Logger
	executable         string
	workingDir         string
	appArgs            []string
	rawWebAddr         string
	rawBindLocalhost   bool
	supervisorAddr     string
	supervisorToken    string
	supervisorServer   *http.Server
	supervisorListener net.Listener
	probeClient        *http.Client
	childChanged       chan struct{}

	mu           sync.RWMutex
	child        *managedChild
	updating     bool
	shuttingDown bool
}

func newRuntimeSupervisor(logger *slog.Logger, appArgs []string, rawWebAddr string, rawBindLocalhost bool) (*runtimeSupervisor, error) {
	executable, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve supervisor executable: %w", err)
	}
	workingDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("resolve supervisor working directory: %w", err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen supervisor control: %w", err)
	}
	token, err := randomSupervisorToken()
	if err != nil {
		listener.Close()
		return nil, fmt.Errorf("create supervisor token: %w", err)
	}
	addr := "http://" + listener.Addr().String()
	supervisor := &runtimeSupervisor{
		logger:             logger,
		executable:         executable,
		workingDir:         workingDir,
		appArgs:            append([]string{}, appArgs...),
		rawWebAddr:         strings.TrimSpace(rawWebAddr),
		rawBindLocalhost:   rawBindLocalhost,
		supervisorAddr:     addr,
		supervisorToken:    token,
		supervisorListener: listener,
		probeClient: &http.Client{
			Timeout: 2 * time.Second,
		},
		childChanged: make(chan struct{}, 1),
	}
	supervisor.supervisorServer = &http.Server{
		Handler: supervisor.controlMux(),
	}
	return supervisor, nil
}

func (s *runtimeSupervisor) Run(ctx context.Context) error {
	go func() {
		if err := s.supervisorServer.Serve(s.supervisorListener); err != nil && !errors.Is(err, http.ErrServerClosed) && s.logger != nil {
			s.logger.Error("runtime supervisor control server exited", slog.String("error", err.Error()))
		}
	}()

	child, err := s.startChild(s.executable)
	if err != nil {
		_ = s.shutdownControlServer()
		return err
	}
	s.setChild(child)

	for {
		current := s.currentChild()
		if current == nil {
			_ = s.shutdownControlServer()
			return errors.New("runtime supervisor has no active child")
		}

		select {
		case <-ctx.Done():
			s.markShuttingDown()
			_ = s.stopChild(current, runtimeRestartStopTimeout)
			_ = s.shutdownControlServer()
			return nil
		case <-current.done:
			if !s.isCurrentChild(current) {
				continue
			}
			if s.isUpdating() {
				continue
			}
			_ = s.shutdownControlServer()
			waitErr := current.waitErrValue()
			if waitErr != nil {
				return fmt.Errorf("runtime child exited: %w", waitErr)
			}
			return errors.New("runtime child exited")
		case <-s.childChanged:
		}
	}
}

func (s *runtimeSupervisor) RequestRestart(options web.RuntimeRestartOptions) (bool, error) {
	s.mu.Lock()
	if s.shuttingDown || s.updating {
		s.mu.Unlock()
		return false, nil
	}
	current := s.child
	if current == nil {
		s.mu.Unlock()
		return false, errors.New("runtime child is unavailable")
	}
	s.updating = true
	s.mu.Unlock()

	candidate, err := s.prepareCandidate(options)
	if err != nil {
		s.finishUpdate()
		return false, err
	}

	go s.cutover(current, candidate)
	return true, nil
}

func (s *runtimeSupervisor) controlMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/restart", s.handleRestart)
	return mux
}

func (s *runtimeSupervisor) handleRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeSupervisorJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if strings.TrimSpace(r.Header.Get(supervisorTokenHeader)) != s.supervisorToken {
		writeSupervisorJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req web.RuntimeRestartOptions
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeSupervisorJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
	}

	accepted, err := s.RequestRestart(req)
	if err != nil {
		writeSupervisorJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if !accepted {
		writeSupervisorJSON(w, http.StatusConflict, map[string]string{"error": "runtime restart already in progress"})
		return
	}
	writeSupervisorJSON(w, http.StatusAccepted, map[string]any{
		"accepted":           true,
		"status":             "restarting",
		"sync_remote_master": req.SyncRemoteMaster,
	})
}

func (s *runtimeSupervisor) prepareCandidate(options web.RuntimeRestartOptions) (string, error) {
	if options.SyncRemoteMaster {
		if err := syncRemoteMasterBranch(s.workingDir); err != nil {
			return "", err
		}
	}
	candidate, err := buildRelaunchBinary(s.workingDir)
	if err != nil {
		return "", err
	}
	if s.logger != nil {
		s.logger.Info(
			"runtime candidate prepared",
			slog.String("candidate", candidate),
			slog.Bool("sync_remote_master", options.SyncRemoteMaster),
		)
	}
	return candidate, nil
}

func (s *runtimeSupervisor) cutover(previous *managedChild, candidateExecutable string) {
	defer s.finishUpdate()

	if err := s.stopChild(previous, runtimeRestartStopTimeout); err != nil && s.logger != nil {
		s.logger.Warn("failed to stop previous runtime child cleanly", slog.String("error", err.Error()))
	}

	next, err := s.startChild(candidateExecutable)
	if err != nil {
		s.restorePrevious(previous, fmt.Errorf("start candidate child: %w", err))
		return
	}

	readyAddr, err := s.resolveProbeAddr()
	if err != nil {
		_ = s.stopChild(next, runtimeRestartStopTimeout)
		s.restorePrevious(previous, fmt.Errorf("resolve runtime probe address: %w", err))
		return
	}
	if err := s.waitUntilReady(next, readyAddr, runtimeReadyTimeout); err != nil {
		_ = s.stopChild(next, runtimeRestartStopTimeout)
		s.restorePrevious(previous, err)
		return
	}

	s.setChild(next)
	if s.logger != nil {
		s.logger.Info(
			"runtime restart completed",
			slog.String("executable", candidateExecutable),
			slog.String("probe_addr", readyAddr),
		)
	}
}

func (s *runtimeSupervisor) restorePrevious(previous *managedChild, cause error) {
	if s.logger != nil && cause != nil {
		s.logger.Error("runtime restart failed, restoring previous child", slog.String("error", cause.Error()))
	}
	restored, err := s.startChild(previous.executable)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("failed to restore previous runtime child", slog.String("error", err.Error()))
		}
		s.clearChild()
		return
	}
	readyAddr, resolveErr := s.resolveProbeAddr()
	if resolveErr != nil {
		if s.logger != nil {
			s.logger.Error("failed to resolve runtime probe address during restore", slog.String("error", resolveErr.Error()))
		}
		s.setChild(restored)
		return
	}
	if waitErr := s.waitUntilReady(restored, readyAddr, runtimeReadyTimeout); waitErr != nil && s.logger != nil {
		s.logger.Error("restored runtime child did not become ready", slog.String("error", waitErr.Error()))
	}
	s.setChild(restored)
}

func (s *runtimeSupervisor) startChild(executable string) (*managedChild, error) {
	args := append([]string{"-" + runtimeChildFlag}, s.appArgs...)
	cmd := exec.Command(executable, args...)
	cmd.Dir = s.workingDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = append(os.Environ(),
		supervisorAddrEnv+"="+s.supervisorAddr,
		supervisorTokenEnv+"="+s.supervisorToken,
	)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start runtime child: %w", err)
	}

	child := &managedChild{
		executable: executable,
		args:       args,
		cmd:        cmd,
		done:       make(chan struct{}),
	}
	go func() {
		child.setWaitErr(cmd.Wait())
		close(child.done)
	}()
	if s.logger != nil {
		s.logger.Info(
			"runtime child started",
			slog.Int("pid", cmd.Process.Pid),
			slog.String("executable", executable),
		)
	}
	return child, nil
}

func (s *runtimeSupervisor) stopChild(child *managedChild, timeout time.Duration) error {
	if child == nil || child.cmd == nil || child.cmd.Process == nil {
		return nil
	}
	if err := child.cmd.Process.Signal(os.Interrupt); err != nil && s.logger != nil {
		s.logger.Warn("failed to send graceful stop signal to runtime child", slog.String("error", err.Error()))
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-child.done:
		return nil
	case <-timer.C:
		if err := child.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("kill runtime child: %w", err)
		}
		<-child.done
		return nil
	}
}

func (s *runtimeSupervisor) waitUntilReady(child *managedChild, probeAddr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-child.done:
			waitErr := child.waitErrValue()
			if waitErr != nil {
				return fmt.Errorf("candidate runtime exited before ready: %w", waitErr)
			}
			return errors.New("candidate runtime exited before ready")
		default:
		}

		req, err := http.NewRequest(http.MethodGet, probeAddr+"/readyz", nil)
		if err == nil {
			resp, reqErr := s.probeClient.Do(req)
			if reqErr == nil {
				_ = resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return nil
				}
			}
		}
		time.Sleep(runtimeProbeInterval)
	}
	return fmt.Errorf("runtime readiness check timed out after %s", timeout)
}

func (s *runtimeSupervisor) resolveProbeAddr() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	listenAddr, err := resolveConfiguredListenAddr(ctx, s.rawWebAddr, s.rawBindLocalhost)
	if err != nil {
		return "", err
	}
	return buildRuntimeProbeAddr(listenAddr)
}

func (s *runtimeSupervisor) setChild(child *managedChild) {
	s.mu.Lock()
	s.child = child
	s.mu.Unlock()
	s.notifyChildChanged()
}

func (s *runtimeSupervisor) clearChild() {
	s.mu.Lock()
	s.child = nil
	s.mu.Unlock()
	s.notifyChildChanged()
}

func (s *runtimeSupervisor) currentChild() *managedChild {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.child
}

func (s *runtimeSupervisor) isCurrentChild(child *managedChild) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.child == child
}

func (s *runtimeSupervisor) finishUpdate() {
	s.mu.Lock()
	s.updating = false
	s.mu.Unlock()
	s.notifyChildChanged()
}

func (s *runtimeSupervisor) isUpdating() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.updating
}

func (s *runtimeSupervisor) markShuttingDown() {
	s.mu.Lock()
	s.shuttingDown = true
	s.mu.Unlock()
}

func (s *runtimeSupervisor) notifyChildChanged() {
	select {
	case s.childChanged <- struct{}{}:
	default:
	}
}

func (s *runtimeSupervisor) shutdownControlServer() error {
	if s.supervisorServer == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), supervisorShutdownTimeout)
	defer cancel()
	return s.supervisorServer.Shutdown(ctx)
}

func writeSupervisorJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func resolveConfiguredListenAddr(ctx context.Context, requestedAddr string, requestedBindLocalhost bool) (string, error) {
	controlStore, _, _, _, err := buildStorage(defaultStorageProfile)
	if err != nil {
		return "", err
	}
	control, err := newControlService(ctx, controlStore)
	if err != nil {
		return "", err
	}
	listenAddr := strings.TrimSpace(control.ResolveEnvironmentString("web_addr", requestedAddr))
	if listenAddr == "" {
		listenAddr = defaultWebAddr
	}
	if resolveEnvironmentBool(control, "web_bind_localhost_only", requestedBindLocalhost) {
		listenAddr = forceLoopbackListenAddr(listenAddr)
	}
	return listenAddr, nil
}

func buildRuntimeProbeAddr(listenAddr string) (string, error) {
	addr := strings.TrimSpace(listenAddr)
	if addr == "" {
		addr = defaultWebAddr
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("parse runtime listen address %q: %w", listenAddr, err)
	}
	host = strings.Trim(strings.TrimSpace(host), "[]")
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		host = "127.0.0.1"
	case "localhost":
		host = "127.0.0.1"
	default:
		ip := net.ParseIP(host)
		if ip != nil && ip.IsUnspecified() {
			host = "127.0.0.1"
		}
	}
	return "http://" + net.JoinHostPort(host, port), nil
}

func filterInternalRuntimeArgs(args []string) []string {
	filtered := make([]string, 0, len(args))
	skipNextValue := false
	for _, arg := range args {
		if skipNextValue {
			skipNextValue = false
			continue
		}
		trimmed := strings.TrimSpace(arg)
		switch {
		case trimmed == "-"+runtimeChildFlag:
			continue
		case trimmed == "-"+relaunchHelperFlag:
			continue
		case strings.HasPrefix(trimmed, "-"+relaunchParentPIDFlag+"="):
			continue
		case trimmed == "-"+relaunchParentPIDFlag:
			skipNextValue = true
			continue
		case strings.HasPrefix(trimmed, "-"+relaunchExecPathFlag+"="):
			continue
		case trimmed == "-"+relaunchExecPathFlag:
			skipNextValue = true
			continue
		case strings.HasPrefix(trimmed, "-"+relaunchArgsFlag+"="):
			continue
		case trimmed == "-"+relaunchArgsFlag:
			skipNextValue = true
			continue
		case strings.HasPrefix(trimmed, "-"+relaunchWorkingDirFlag+"="):
			continue
		case trimmed == "-"+relaunchWorkingDirFlag:
			skipNextValue = true
			continue
		default:
			filtered = append(filtered, arg)
		}
	}
	return filtered
}

func randomSupervisorToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

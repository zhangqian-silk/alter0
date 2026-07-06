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
	GetRestartStatus() web.RuntimeRestartStatus
	ListRestartCandidates() (web.RuntimeRestartCandidateList, error)
}

type supervisorClientRestarter struct {
	addr   string
	token  string
	client *http.Client
}

func newRuntimeRestarter(cancel context.CancelFunc, logger *slog.Logger, args []string, runtimeRoot string) (runtimeRestartClient, error) {
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
	return newServiceRestarter(cancel, logger, args, runtimeRoot)
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
		Code     string `json:"code"`
		Error    string `json:"error"`
	}
	_ = json.Unmarshal(body, &payloadResp)
	if resp.StatusCode == http.StatusAccepted {
		return true, nil
	}
	message := strings.TrimSpace(payloadResp.Error)
	if message == "" {
		message = strings.TrimSpace(string(body))
	}
	if message == "" {
		message = fmt.Sprintf("supervisor returned HTTP %d", resp.StatusCode)
	}
	if code := strings.TrimSpace(payloadResp.Code); code != "" {
		return false, web.NewRuntimeRestartError(code, message)
	}
	if resp.StatusCode == http.StatusConflict {
		return false, nil
	}
	return false, errors.New(message)
}

func (r *supervisorClientRestarter) GetRestartStatus() web.RuntimeRestartStatus {
	if r == nil {
		return web.RuntimeRestartStatus{Status: "idle"}
	}
	endpoint, err := url.Parse(r.addr)
	if err != nil {
		return web.RuntimeRestartStatus{Status: "failed", Error: fmt.Sprintf("parse supervisor address: %v", err)}
	}
	endpoint.Path = "/restart"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return web.RuntimeRestartStatus{Status: "failed", Error: fmt.Sprintf("create supervisor status request: %v", err)}
	}
	req.Header.Set(supervisorTokenHeader, r.token)
	resp, err := r.client.Do(req)
	if err != nil {
		return web.RuntimeRestartStatus{Status: "failed", Error: fmt.Sprintf("request supervisor restart status: %v", err)}
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 32*1024))
	if readErr != nil {
		return web.RuntimeRestartStatus{Status: "failed", Error: fmt.Sprintf("read supervisor status response: %v", readErr)}
	}
	if resp.StatusCode != http.StatusOK {
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = fmt.Sprintf("supervisor returned HTTP %d", resp.StatusCode)
		}
		return web.RuntimeRestartStatus{Status: "failed", Error: message}
	}
	var status web.RuntimeRestartStatus
	if err := json.Unmarshal(body, &status); err != nil {
		return web.RuntimeRestartStatus{Status: "failed", Error: fmt.Sprintf("decode supervisor status response: %v", err)}
	}
	if strings.TrimSpace(status.Status) == "" {
		status.Status = "idle"
	}
	return status
}

func (r *supervisorClientRestarter) ListRestartCandidates() (web.RuntimeRestartCandidateList, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return web.RuntimeRestartCandidateList{}, fmt.Errorf("resolve working directory: %w", err)
	}
	return listRuntimeRestartCandidates(workingDir, resolveRuntimeCommitHash(workingDir))
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
	runtimeRoot        string
	appArgs            []string
	rawWebAddr         string
	rawBindLocalhost   bool
	supervisorAddr     string
	supervisorToken    string
	supervisorServer   *http.Server
	supervisorListener net.Listener
	probeClient        *http.Client
	childChanged       chan struct{}

	mu            sync.RWMutex
	child         *managedChild
	updating      bool
	shuttingDown  bool
	restartStatus web.RuntimeRestartStatus
}

func newRuntimeSupervisor(logger *slog.Logger, appArgs []string, rawWebAddr string, rawBindLocalhost bool, runtimeRoot string) (*runtimeSupervisor, error) {
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
		runtimeRoot:        cleanConfiguredPath(runtimeRoot),
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
	s.restartStatus = newRestartStatus("preparing", options, "")
	s.mu.Unlock()

	candidate, err := s.prepareCandidate(options)
	if err != nil {
		s.setRestartStatus("failed", options, err)
		s.finishUpdate()
		return false, err
	}
	s.setRestartStatus("switching", options, nil)

	go s.cutover(current, candidate, options)
	return true, nil
}

func (s *runtimeSupervisor) controlMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/restart", s.handleRestart)
	return mux
}

func (s *runtimeSupervisor) handleRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		writeSupervisorJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if strings.TrimSpace(r.Header.Get(supervisorTokenHeader)) != s.supervisorToken {
		writeSupervisorJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if r.Method == http.MethodGet {
		writeSupervisorJSON(w, http.StatusOK, s.GetRestartStatus())
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
		var restartErr *web.RuntimeRestartError
		if errors.As(err, &restartErr) && restartErr.Code != "" {
			status := http.StatusInternalServerError
			if restartErr.Code == web.RuntimeRestartDiscardConfirmationRequired {
				status = http.StatusConflict
			}
			writeSupervisorJSON(w, status, map[string]string{"code": restartErr.Code, "error": restartErr.Error()})
			return
		}
		writeSupervisorJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if !accepted {
		writeSupervisorJSON(w, http.StatusConflict, map[string]string{"error": "runtime restart already in progress"})
		return
	}
	writeSupervisorJSON(w, http.StatusAccepted, map[string]any{
		"accepted":                        true,
		"status":                          "restarting",
		"sync_remote_master":              req.SyncRemoteMaster,
		"confirm_discard_tracked_changes": req.ConfirmDiscardTrackedChanges,
		"target_commit":                   strings.TrimSpace(req.TargetCommit),
	})
}

func (s *runtimeSupervisor) prepareCandidate(options web.RuntimeRestartOptions) (string, error) {
	if options.SyncRemoteMaster {
		if err := syncRemoteMasterBranch(s.workingDir, options.ConfirmDiscardTrackedChanges, options.TargetCommit); err != nil {
			return "", err
		}
	}
	candidate, err := buildRelaunchBinary(s.workingDir, s.runtimeRoot)
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

func (s *runtimeSupervisor) cutover(previous *managedChild, candidateExecutable string, options web.RuntimeRestartOptions) {
	defer s.finishUpdate()

	if err := s.stopChild(previous, runtimeRestartStopTimeout); err != nil && s.logger != nil {
		s.logger.Warn("failed to stop previous runtime child cleanly", slog.String("error", err.Error()))
	}

	next, err := s.startChild(candidateExecutable)
	if err != nil {
		s.restorePrevious(previous, options, fmt.Errorf("start candidate child: %w", err))
		return
	}

	readyAddr, err := s.resolveProbeAddr()
	if err != nil {
		_ = s.stopChild(next, runtimeRestartStopTimeout)
		s.restorePrevious(previous, options, fmt.Errorf("resolve runtime probe address: %w", err))
		return
	}
	if err := s.waitUntilReady(next, readyAddr, runtimeReadyTimeout); err != nil {
		_ = s.stopChild(next, runtimeRestartStopTimeout)
		s.restorePrevious(previous, options, err)
		return
	}

	s.setChild(next)
	s.setRestartStatus("completed", options, nil)
	if s.logger != nil {
		s.logger.Info(
			"runtime restart completed",
			slog.String("executable", candidateExecutable),
			slog.String("probe_addr", readyAddr),
		)
	}
}

func (s *runtimeSupervisor) restorePrevious(previous *managedChild, options web.RuntimeRestartOptions, cause error) {
	s.setRestartStatus("failed", options, cause)
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

func (s *runtimeSupervisor) GetRestartStatus() web.RuntimeRestartStatus {
	if s == nil {
		return web.RuntimeRestartStatus{Status: "idle"}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.restartStatus.Status == "" {
		return web.RuntimeRestartStatus{Status: "idle"}
	}
	return s.restartStatus
}

func (s *runtimeSupervisor) setRestartStatus(status string, options web.RuntimeRestartOptions, cause error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.restartStatus.StartedAt.IsZero() {
		s.restartStatus = newRestartStatus(status, options, "")
	}
	s.restartStatus.Status = status
	s.restartStatus.SyncRemoteMaster = options.SyncRemoteMaster
	s.restartStatus.ConfirmDiscardTrackedChanges = options.ConfirmDiscardTrackedChanges
	s.restartStatus.TargetCommit = strings.TrimSpace(options.TargetCommit)
	s.restartStatus.UpdatedAt = time.Now().UTC()
	if cause != nil {
		s.restartStatus.Error = cause.Error()
	} else {
		s.restartStatus.Error = ""
	}
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

func resolveConfiguredListenAddr(_ context.Context, requestedAddr string, requestedBindLocalhost bool) (string, error) {
	listenAddr := strings.TrimSpace(requestedAddr)
	if listenAddr == "" {
		listenAddr = defaultWebAddr
	}
	if requestedBindLocalhost {
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
	return whitelistRuntimeArgs(args)
}

func whitelistRuntimeArgs(args []string) []string {
	allowedValueFlags := map[string]bool{
		"codex-command":      true,
		"web-addr":           true,
		"web-login-password": true,
	}
	allowedBoolFlags := map[string]bool{
		"web-bind-localhost-only": true,
	}
	filtered := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		if arg == "" {
			continue
		}
		name, hasValue := splitRuntimeFlagName(arg)
		if name == "" {
			continue
		}
		if allowedBoolFlags[name] {
			if hasValue || !strings.Contains(arg, "=") {
				filtered = append(filtered, arg)
			}
			continue
		}
		if !allowedValueFlags[name] {
			continue
		}
		if hasValue {
			filtered = append(filtered, arg)
			continue
		}
		if i+1 < len(args) && !strings.HasPrefix(strings.TrimSpace(args[i+1]), "-") {
			filtered = append(filtered, arg, args[i+1])
			i++
		}
	}
	return filtered
}

func splitRuntimeFlagName(arg string) (string, bool) {
	trimmed := strings.TrimSpace(arg)
	if !strings.HasPrefix(trimmed, "-") {
		return "", false
	}
	trimmed = strings.TrimLeft(trimmed, "-")
	if trimmed == "" {
		return "", false
	}
	if idx := strings.Index(trimmed, "="); idx >= 0 {
		return trimmed[:idx], true
	}
	return trimmed, false
}

func newRestartStatus(status string, options web.RuntimeRestartOptions, message string) web.RuntimeRestartStatus {
	now := time.Now().UTC()
	return web.RuntimeRestartStatus{
		Status:                       status,
		Error:                        strings.TrimSpace(message),
		SyncRemoteMaster:             options.SyncRemoteMaster,
		ConfirmDiscardTrackedChanges: options.ConfirmDiscardTrackedChanges,
		TargetCommit:                 strings.TrimSpace(options.TargetCommit),
		StartedAt:                    now,
		UpdatedAt:                    now,
	}
}

func randomSupervisorToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

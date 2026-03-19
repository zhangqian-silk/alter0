package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	relaunchHelperFlag     = "internal-relaunch-helper"
	relaunchParentPIDFlag  = "internal-relaunch-parent-pid"
	relaunchExecPathFlag   = "internal-relaunch-exec"
	relaunchArgsFlag       = "internal-relaunch-args"
	relaunchWorkingDirFlag = "internal-relaunch-cwd"
	relaunchWaitTimeout    = 20 * time.Second
)

type serviceRestarter struct {
	cancel context.CancelFunc
	logger *slog.Logger

	executable string
	workingDir string
	args       []string

	mu         sync.Mutex
	restarting bool
}

func newServiceRestarter(cancel context.CancelFunc, logger *slog.Logger, args []string) (*serviceRestarter, error) {
	if cancel == nil {
		return nil, errors.New("restart cancel function is required")
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve executable: %w", err)
	}
	workingDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("resolve working directory: %w", err)
	}
	return &serviceRestarter{
		cancel:     cancel,
		logger:     logger,
		executable: executable,
		workingDir: workingDir,
		args:       append([]string{}, args...),
	}, nil
}

func (r *serviceRestarter) RequestRestart() (bool, error) {
	r.mu.Lock()
	if r.restarting {
		r.mu.Unlock()
		return false, nil
	}
	r.restarting = true
	r.mu.Unlock()

	encodedArgs, err := encodeRelaunchArgs(r.args)
	if err != nil {
		r.reset()
		return false, err
	}

	helperArgs := []string{
		"-" + relaunchHelperFlag,
		fmt.Sprintf("-%s=%d", relaunchParentPIDFlag, os.Getpid()),
		fmt.Sprintf("-%s=%s", relaunchExecPathFlag, r.executable),
		fmt.Sprintf("-%s=%s", relaunchWorkingDirFlag, r.workingDir),
		fmt.Sprintf("-%s=%s", relaunchArgsFlag, encodedArgs),
	}
	helper := exec.Command(r.executable, helperArgs...)
	helper.Dir = r.workingDir
	if err := helper.Start(); err != nil {
		r.reset()
		return false, fmt.Errorf("start relaunch helper: %w", err)
	}

	if r.logger != nil {
		r.logger.Info("restart accepted", slog.Int("pid", os.Getpid()), slog.Int("helper_pid", helper.Process.Pid))
	}

	go func() {
		time.Sleep(150 * time.Millisecond)
		r.cancel()
	}()
	return true, nil
}

func (r *serviceRestarter) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.restarting = false
}

func runRelaunchHelper(parentPID int, executable string, workingDir string, encodedArgs string) error {
	if parentPID <= 0 {
		return errors.New("parent pid is required")
	}
	executable = strings.TrimSpace(executable)
	if executable == "" {
		return errors.New("relaunch executable path is required")
	}
	args, err := decodeRelaunchArgs(encodedArgs)
	if err != nil {
		return err
	}

	deadline := time.Now().Add(relaunchWaitTimeout)
	for processExists(parentPID) && time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
	}

	cmd := exec.Command(executable, args...)
	if strings.TrimSpace(workingDir) != "" {
		cmd.Dir = strings.TrimSpace(workingDir)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start relaunched service: %w", err)
	}
	return nil
}

func encodeRelaunchArgs(args []string) (string, error) {
	payload, err := json.Marshal(args)
	if err != nil {
		return "", fmt.Errorf("marshal relaunch args: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeRelaunchArgs(encoded string) ([]string, error) {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return []string{}, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode relaunch args: %w", err)
	}
	var args []string
	if err := json.Unmarshal(payload, &args); err != nil {
		return nil, fmt.Errorf("unmarshal relaunch args: %w", err)
	}
	return args, nil
}

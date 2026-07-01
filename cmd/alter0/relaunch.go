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
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"alter0/internal/interfaces/web"
)

const (
	relaunchHelperFlag     = "internal-relaunch-helper"
	relaunchParentPIDFlag  = "internal-relaunch-parent-pid"
	relaunchExecPathFlag   = "internal-relaunch-exec"
	relaunchArgsFlag       = "internal-relaunch-args"
	relaunchWorkingDirFlag = "internal-relaunch-cwd"
	relaunchWaitTimeout    = 20 * time.Second
	gitFetchTimeout        = 45 * time.Second
	gitMergeTimeout        = 15 * time.Second
	gitStatusTimeout       = 10 * time.Second
	gitBranchShowTimeout   = 10 * time.Second
	serviceBuildTimeout    = 10 * time.Minute
)

type serviceRestarter struct {
	cancel context.CancelFunc
	logger *slog.Logger

	executable string
	workingDir string
	args       []string

	mu         sync.Mutex
	restarting bool
	status     web.RuntimeRestartStatus
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

func (r *serviceRestarter) RequestRestart(options web.RuntimeRestartOptions) (bool, error) {
	r.mu.Lock()
	if r.restarting {
		r.mu.Unlock()
		return false, nil
	}
	r.restarting = true
	r.status = newRestartStatus("preparing", options, "")
	r.mu.Unlock()

	relaunchExecutable, err := r.resolveRelaunchExecutable(options)
	if err != nil {
		r.setRestartStatus("failed", options, err)
		r.reset()
		return false, err
	}

	encodedArgs, err := encodeRelaunchArgs(r.args)
	if err != nil {
		r.setRestartStatus("failed", options, err)
		r.reset()
		return false, err
	}

	helperArgs := []string{
		"-" + relaunchHelperFlag,
		fmt.Sprintf("-%s=%d", relaunchParentPIDFlag, os.Getpid()),
		fmt.Sprintf("-%s=%s", relaunchExecPathFlag, relaunchExecutable),
		fmt.Sprintf("-%s=%s", relaunchWorkingDirFlag, r.workingDir),
		fmt.Sprintf("-%s=%s", relaunchArgsFlag, encodedArgs),
	}
	helper := exec.Command(r.executable, helperArgs...)
	helper.Dir = r.workingDir
	if err := helper.Start(); err != nil {
		r.setRestartStatus("failed", options, err)
		r.reset()
		return false, fmt.Errorf("start relaunch helper: %w", err)
	}
	r.setRestartStatus("switching", options, nil)

	if r.logger != nil {
		r.logger.Info(
			"restart accepted",
			slog.Int("pid", os.Getpid()),
			slog.Int("helper_pid", helper.Process.Pid),
			slog.Bool("sync_remote_master", options.SyncRemoteMaster),
		)
	}

	go func() {
		time.Sleep(150 * time.Millisecond)
		r.cancel()
	}()
	return true, nil
}

func (r *serviceRestarter) GetRestartStatus() web.RuntimeRestartStatus {
	if r == nil {
		return web.RuntimeRestartStatus{Status: "idle"}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.status.Status == "" {
		return web.RuntimeRestartStatus{Status: "idle"}
	}
	return r.status
}

func (r *serviceRestarter) ListRestartCandidates() (web.RuntimeRestartCandidateList, error) {
	if r == nil {
		return web.RuntimeRestartCandidateList{}, errors.New("runtime restarter is required")
	}
	return listRuntimeRestartCandidates(r.workingDir, resolveRuntimeCommitHash(r.workingDir))
}

func (r *serviceRestarter) setRestartStatus(status string, options web.RuntimeRestartOptions, cause error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.status.StartedAt.IsZero() {
		r.status = newRestartStatus(status, options, "")
	}
	r.status.Status = status
	r.status.SyncRemoteMaster = options.SyncRemoteMaster
	r.status.ConfirmDiscardTrackedChanges = options.ConfirmDiscardTrackedChanges
	r.status.TargetCommit = strings.TrimSpace(options.TargetCommit)
	r.status.UpdatedAt = time.Now().UTC()
	if cause != nil {
		r.status.Error = cause.Error()
	} else {
		r.status.Error = ""
	}
}

func (r *serviceRestarter) resolveRelaunchExecutable(options web.RuntimeRestartOptions) (string, error) {
	if options.SyncRemoteMaster {
		if err := syncRemoteMasterBranch(r.workingDir, options.ConfirmDiscardTrackedChanges, options.TargetCommit); err != nil {
			return "", err
		}
	}
	return buildRelaunchBinary(r.workingDir)
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

func syncRemoteMasterBranch(workingDir string, confirmDiscardTrackedChanges bool, targetCommit string) error {
	repoDir := strings.TrimSpace(workingDir)
	if repoDir == "" {
		return errors.New("sync remote master requires a working directory")
	}

	branch, err := readCommandOutputWithTimeout(gitBranchShowTimeout, repoDir, "git", "branch", "--show-current")
	if err != nil {
		return fmt.Errorf("resolve current git branch: %w", err)
	}
	if branch != "master" {
		return fmt.Errorf("sync remote master requires the local branch to be master, current branch is %q", branch)
	}

	status, err := readCommandOutputWithTimeout(gitStatusTimeout, repoDir, "git", "status", "--porcelain", "--untracked-files=no")
	if err != nil {
		return fmt.Errorf("inspect git working tree: %w", err)
	}
	if status != "" {
		if !confirmDiscardTrackedChanges {
			return web.NewRuntimeRestartError(
				web.RuntimeRestartDiscardConfirmationRequired,
				fmt.Sprintf("sync remote master requires discard confirmation because tracked working tree changes exist: %s", status),
			)
		}
		if err := runCommandWithTimeout(gitStatusTimeout, repoDir, "git", "reset", "--hard", "HEAD"); err != nil {
			return fmt.Errorf("discard tracked working tree changes: %w", err)
		}
	}

	if err := runGitNetworkCommandWithRetry(repoDir, gitFetchTimeout, 2, "fetch", "--prune", "origin", "master"); err != nil {
		return fmt.Errorf("fetch origin/master: %w", err)
	}
	targetCommit = strings.TrimSpace(targetCommit)
	if targetCommit != "" {
		resolvedTarget, err := readCommandOutputWithTimeout(gitStatusTimeout, repoDir, "git", "rev-parse", "--verify", targetCommit+"^{commit}")
		if err != nil {
			return fmt.Errorf("resolve selected master commit: %w", err)
		}
		if err := runCommandWithTimeout(gitStatusTimeout, repoDir, "git", "merge-base", "--is-ancestor", resolvedTarget, "origin/master"); err != nil {
			return fmt.Errorf("selected commit %q is not reachable from origin/master: %w", targetCommit, err)
		}
		if err := runCommandWithTimeout(gitStatusTimeout, repoDir, "git", "reset", "--hard", resolvedTarget); err != nil {
			return fmt.Errorf("sync selected master commit: %w", err)
		}
		return nil
	}
	if err := runGitNetworkCommandWithRetry(repoDir, gitMergeTimeout, 1, "merge", "--ff-only", "FETCH_HEAD"); err != nil {
		return fmt.Errorf("sync origin/master: %w", err)
	}
	return nil
}

func listRuntimeRestartCandidates(workingDir string, currentCommit string) (web.RuntimeRestartCandidateList, error) {
	repoDir := strings.TrimSpace(workingDir)
	if repoDir == "" {
		return web.RuntimeRestartCandidateList{}, errors.New("list restart candidates requires a working directory")
	}
	if err := runGitNetworkCommandWithRetry(repoDir, gitFetchTimeout, 2, "fetch", "--prune", "origin", "master"); err != nil {
		return web.RuntimeRestartCandidateList{}, fmt.Errorf("fetch origin/master: %w", err)
	}

	currentCommit = strings.TrimSpace(currentCommit)
	if currentCommit == "" {
		var err error
		currentCommit, err = readCommandOutputWithTimeout(gitStatusTimeout, repoDir, "git", "rev-parse", "HEAD")
		if err != nil {
			return web.RuntimeRestartCandidateList{}, fmt.Errorf("resolve current runtime commit: %w", err)
		}
	}
	currentCommit, err := readCommandOutputWithTimeout(gitStatusTimeout, repoDir, "git", "rev-parse", "--verify", currentCommit+"^{commit}")
	if err != nil {
		return web.RuntimeRestartCandidateList{}, fmt.Errorf("resolve current runtime commit: %w", err)
	}

	result := web.RuntimeRestartCandidateList{CurrentCommit: currentCommit}
	seen := map[string]bool{}

	newerOutput, err := readCommandOutputWithTimeout(
		gitStatusTimeout,
		repoDir,
		"git",
		"log",
		"--reverse",
		"--ancestry-path",
		"--format=%H%x00%ct%x00%s",
		currentCommit+"..origin/master",
	)
	if err != nil {
		return result, nil
	}
	appendRuntimeRestartCandidates(&result, newerOutput, currentCommit, seen)

	historyOutput, err := readCommandOutputWithTimeout(
		gitStatusTimeout,
		repoDir,
		"git",
		"log",
		"-n",
		"11",
		"--format=%H%x00%ct%x00%s",
		currentCommit,
	)
	if err != nil {
		return result, nil
	}
	appendRuntimeRestartCandidates(&result, historyOutput, currentCommit, seen)
	return result, nil
}

func appendRuntimeRestartCandidates(result *web.RuntimeRestartCandidateList, output string, currentCommit string, seen map[string]bool) {
	if result == nil {
		return
	}
	for _, line := range strings.Split(output, "\n") {
		candidate, ok := parseRuntimeRestartCandidate(line, currentCommit)
		if !ok || seen[candidate.Hash] {
			continue
		}
		seen[candidate.Hash] = true
		result.Items = append(result.Items, candidate)
	}
}

func parseRuntimeRestartCandidate(line string, currentCommit string) (web.RuntimeRestartCandidate, bool) {
	parts := strings.SplitN(strings.TrimSpace(line), "\x00", 3)
	if len(parts) != 3 {
		return web.RuntimeRestartCandidate{}, false
	}
	hash := strings.TrimSpace(parts[0])
	if hash == "" {
		return web.RuntimeRestartCandidate{}, false
	}
	seconds, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil {
		return web.RuntimeRestartCandidate{}, false
	}
	shortHash := hash
	if len(shortHash) > 7 {
		shortHash = shortHash[:7]
	}
	return web.RuntimeRestartCandidate{
		Hash:        hash,
		ShortHash:   shortHash,
		Message:     strings.TrimSpace(parts[2]),
		CommittedAt: time.Unix(seconds, 0).UTC(),
		Current:     hash == strings.TrimSpace(currentCommit),
	}, true
}

func buildRelaunchBinary(workingDir string) (string, error) {
	repoDir := strings.TrimSpace(workingDir)
	if repoDir == "" {
		return "", errors.New("build relaunch binary requires a working directory")
	}

	outputDir := filepath.Join(repoDir, "output", "runtime")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf("create runtime output directory: %w", err)
	}

	extension := ""
	if runtime.GOOS == "windows" {
		extension = ".exe"
	}
	targetPath := filepath.Join(outputDir, fmt.Sprintf("alter0-relaunch-%d%s", time.Now().UnixNano(), extension))
	buildScript := filepath.Join(repoDir, "scripts", "build_alter0_service.sh")
	if err := runCommandWithTimeoutEnv(serviceBuildTimeout, repoDir, []string{"ALTER0_BUILD_OUTPUT=" + targetPath}, buildScript); err != nil {
		return "", fmt.Errorf("build relaunch binary: %w", err)
	}
	if _, err := os.Stat(targetPath); err != nil {
		return "", fmt.Errorf("inspect relaunch binary: %w", err)
	}
	return targetPath, nil
}

func readCommandOutput(dir string, name string, args ...string) (string, error) {
	return readCommandOutputWithTimeout(0, dir, name, args...)
}

func readCommandOutputWithTimeout(timeout time.Duration, dir string, name string, args ...string) (string, error) {
	cmd, cancel := prepareCommand(timeout, dir, name, args...)
	if cancel != nil {
		defer cancel()
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return "", err
		}
		return "", fmt.Errorf("%w: %s", err, message)
	}
	return strings.TrimSpace(string(output)), nil
}

func runCommand(dir string, name string, args ...string) error {
	return runCommandWithTimeout(0, dir, name, args...)
}

func runCommandWithTimeout(timeout time.Duration, dir string, name string, args ...string) error {
	return runCommandWithTimeoutEnv(timeout, dir, nil, name, args...)
}

func runCommandWithTimeoutEnv(timeout time.Duration, dir string, env []string, name string, args ...string) error {
	cmd, cancel := prepareCommand(timeout, dir, name, args...)
	if cancel != nil {
		defer cancel()
	}
	if len(env) > 0 {
		baseEnv := cmd.Env
		if baseEnv == nil {
			baseEnv = os.Environ()
		}
		cmd.Env = append(baseEnv, env...)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return err
		}
		return fmt.Errorf("%w: %s", err, message)
	}
	return nil
}

func prepareCommand(timeout time.Duration, dir string, name string, args ...string) (*exec.Cmd, context.CancelFunc) {
	ctx := context.Background()
	cancel := context.CancelFunc(nil)
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	if strings.EqualFold(strings.TrimSpace(name), "git") {
		cmd.Env = append(os.Environ(),
			"GIT_TERMINAL_PROMPT=0",
			"GIT_HTTP_LOW_SPEED_LIMIT=1",
			"GIT_HTTP_LOW_SPEED_TIME=30",
		)
	}
	return cmd, cancel
}

func runGitNetworkCommandWithRetry(dir string, timeout time.Duration, attempts int, args ...string) error {
	if attempts <= 0 {
		attempts = 1
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		lastErr = runCommandWithTimeout(timeout, dir, "git", args...)
		if lastErr == nil {
			return nil
		}
		if i == attempts-1 {
			break
		}
		time.Sleep(time.Duration(i+1) * 2 * time.Second)
	}
	return lastErr
}

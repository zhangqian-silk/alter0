package infrastructure

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

type stubWorkspaceServiceDeployer struct {
	lastRequest WorkspaceServiceDeployRequest
	result      WorkspaceServiceDeployResult
	err         error
}

func (s *stubWorkspaceServiceDeployer) Deploy(_ context.Context, req WorkspaceServiceDeployRequest) (WorkspaceServiceDeployResult, error) {
	s.lastRequest = req
	if s.err != nil {
		return WorkspaceServiceDeployResult{}, s.err
	}
	return s.result, nil
}

func TestWorkspaceServiceDeployerLeavesDefaultWebPreviewModeToScript(t *testing.T) {
	repoRoot, err := resolveToolRepoRoot()
	if err != nil {
		t.Fatalf("resolveToolRepoRoot() error = %v", err)
	}

	capturedArgs := []string{}
	deployer := &scriptWorkspaceServiceDeployer{
		commandRunner: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			capturedArgs = append([]string{name}, args...)
			return exec.CommandContext(ctx, "bash", "-lc", `printf '{"session_id":"session-default","service_id":"web","service_type":"http","host":"4e8f5f54.alter0.cn","url":"https://4e8f5f54.alter0.cn"}'`)
		},
	}

	_, err = deployer.Deploy(context.Background(), WorkspaceServiceDeployRequest{
		SessionID: "session-default",
		ServiceID: "web",
	})
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}

	if slices.Contains(capturedArgs, "--service-type") {
		t.Fatalf("expected default web preview deploy to let script infer service type, got %v", capturedArgs)
	}
	if !slices.Contains(capturedArgs, "--repo-path") {
		t.Fatalf("expected repo path to be forwarded for default web preview, got %v", capturedArgs)
	}
	if !slices.Contains(capturedArgs, filepath.ToSlash(repoRoot)) {
		t.Fatalf("expected repo root %q in args, got %v", filepath.ToSlash(repoRoot), capturedArgs)
	}
}

func TestWorkspaceServiceDeployerPreservesExplicitFrontendDistMode(t *testing.T) {
	capturedArgs := []string{}
	deployer := &scriptWorkspaceServiceDeployer{
		commandRunner: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			capturedArgs = append([]string{name}, args...)
			return exec.CommandContext(ctx, "bash", "-lc", `printf '{"session_id":"session-default","service_id":"web","service_type":"frontend_dist","host":"4e8f5f54.alter0.cn","url":"https://4e8f5f54.alter0.cn"}'`)
		},
	}

	_, err := deployer.Deploy(context.Background(), WorkspaceServiceDeployRequest{
		SessionID:   "session-default",
		ServiceID:   "web",
		ServiceType: workspaceServiceTypeFrontendDist,
		SkipBuild:   true,
	})
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}

	if !slices.Contains(capturedArgs, "--service-type") || !slices.Contains(capturedArgs, workspaceServiceTypeFrontendDist) {
		t.Fatalf("expected explicit frontend_dist args, got %v", capturedArgs)
	}
	if !slices.Contains(capturedArgs, "--skip-build") {
		t.Fatalf("expected skip-build to be forwarded, got %v", capturedArgs)
	}
}

func TestResolveWorkspaceServiceRepositoryPathDefaultsTravelToSessionWorkspace(t *testing.T) {
	repoRoot := t.TempDir()
	sessionWorkspace := buildSessionWorkspacePath(repoRoot, "travel-session")
	if err := os.MkdirAll(sessionWorkspace, 0o755); err != nil {
		t.Fatalf("mkdir session workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionWorkspace, "index.html"), []byte("<!doctype html><title>travel</title>"), 0o644); err != nil {
		t.Fatalf("write session index: %v", err)
	}

	repositoryPath, err := resolveWorkspaceServiceRepositoryPath(repoRoot, "travel-session", "travel", "")
	if err != nil {
		t.Fatalf("resolveWorkspaceServiceRepositoryPath() error = %v", err)
	}
	if repositoryPath != filepath.ToSlash(sessionWorkspace) {
		t.Fatalf("expected session workspace path %q, got %q", filepath.ToSlash(sessionWorkspace), repositoryPath)
	}
}

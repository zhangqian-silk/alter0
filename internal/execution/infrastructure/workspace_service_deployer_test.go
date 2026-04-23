package infrastructure

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	llmdomain "alter0/internal/llm/domain"
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

func TestHybridNLProcessorBuildAgentToolsIncludesDeployTestService(t *testing.T) {
	processor := NewHybridNLProcessor(newTestProcessor("success", mustBuildTestPrompt(t, "整理仓库", testRuntimeMetadata())), nil, nil)

	metadata := testRuntimeMetadata()
	metadata["alter0.agent.tools"] = `["deploy_test_service"]`

	tools := processor.buildAgentTools(metadata)
	toolNames := []string{}
	for _, item := range tools {
		toolNames = append(toolNames, item.Name)
	}
	if !strings.Contains(strings.Join(toolNames, ","), toolDeployTestService) {
		t.Fatalf("expected deploy_test_service in %+v", toolNames)
	}
}

func TestHybridNLProcessorExecutesDeployTestServiceTool(t *testing.T) {
	deployer := &stubWorkspaceServiceDeployer{
		result: WorkspaceServiceDeployResult{
			SessionID:   "session-default",
			ServiceID:   "docs",
			ServiceType: workspaceServiceTypeHTTP,
			Host:        "docs.4e8f5f54.alter0.cn",
			URL:         "https://docs.4e8f5f54.alter0.cn",
			UpstreamURL: "http://127.0.0.1:19191",
			Status:      "deployed",
		},
	}
	processor := &HybridNLProcessor{
		codex:           NewCodexCLIProcessor(),
		serviceDeployer: deployer,
	}

	metadata := testRuntimeMetadata()
	result, err := processor.executeModelTool(context.Background(), metadata, llmdomain.ToolCall{
		ID:   "deploy-1",
		Name: toolDeployTestService,
		Arguments: `{
			"service_name":"docs",
			"service_type":"http",
			"upstream_url":"http://127.0.0.1:19191",
			"health_path":"/healthz"
		}`,
	})
	if err != nil {
		t.Fatalf("deploy_test_service error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %+v", result)
	}
	if deployer.lastRequest.SessionID != "session-default" {
		t.Fatalf("expected session-default, got %+v", deployer.lastRequest)
	}
	if deployer.lastRequest.ServiceID != "docs" || deployer.lastRequest.ServiceType != workspaceServiceTypeHTTP {
		t.Fatalf("unexpected request %+v", deployer.lastRequest)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(result.Result), &payload); err != nil {
		t.Fatalf("unmarshal tool result: %v", err)
	}
	if payload["url"] != "https://docs.4e8f5f54.alter0.cn" {
		t.Fatalf("unexpected tool payload %+v", payload)
	}
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

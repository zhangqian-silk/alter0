package infrastructure

import (
	"context"
	"encoding/json"
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

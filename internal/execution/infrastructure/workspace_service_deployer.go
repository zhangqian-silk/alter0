package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	workspaceServiceTypeFrontendDist = "frontend_dist"
	workspaceServiceTypeHTTP         = "http"
)

type WorkspaceServiceDeployRequest struct {
	SessionID      string
	ServiceID      string
	ServiceType    string
	RepositoryPath string
	UpstreamURL    string
	StartCommand   string
	Workdir        string
	Port           int
	HealthPath     string
	SkipBuild      bool
}

type WorkspaceServiceDeployResult struct {
	SessionID      string `json:"session_id"`
	ServiceID      string `json:"service_id"`
	ServiceType    string `json:"service_type"`
	ShortHash      string `json:"short_hash,omitempty"`
	Host           string `json:"host"`
	URL            string `json:"url"`
	RepositoryPath string `json:"repository_path,omitempty"`
	DistPath       string `json:"dist_path,omitempty"`
	UpstreamURL    string `json:"upstream_url,omitempty"`
	RuntimeDir     string `json:"runtime_dir,omitempty"`
	LogPath        string `json:"log_path,omitempty"`
	PID            int    `json:"pid,omitempty"`
	Status         string `json:"status,omitempty"`
}

type workspaceServiceDeployer interface {
	Deploy(ctx context.Context, req WorkspaceServiceDeployRequest) (WorkspaceServiceDeployResult, error)
}

type scriptWorkspaceServiceDeployer struct {
	commandRunner func(context.Context, string, ...string) *exec.Cmd
}

func newWorkspaceServiceDeployer() workspaceServiceDeployer {
	return &scriptWorkspaceServiceDeployer{commandRunner: exec.CommandContext}
}

func (d *scriptWorkspaceServiceDeployer) Deploy(ctx context.Context, req WorkspaceServiceDeployRequest) (WorkspaceServiceDeployResult, error) {
	repoRoot, err := resolveToolRepoRoot()
	if err != nil {
		return WorkspaceServiceDeployResult{}, err
	}
	scriptPath := filepath.Join(repoRoot, "scripts", "deploy_test_service.sh")
	serviceID := normalizeWorkspaceServiceID(req.ServiceID)
	args := []string{scriptPath, strings.TrimSpace(req.SessionID), serviceID}

	repositoryPath, err := resolveWorkspaceServiceRepositoryPath(repoRoot, req.SessionID, req.RepositoryPath)
	if err != nil {
		return WorkspaceServiceDeployResult{}, err
	}
	args = append(args, "--repo-path", repositoryPath)

	serviceType := normalizeWorkspaceServiceType(req.ServiceType)
	if serviceType != "" {
		args = append(args, "--service-type", serviceType)
		if serviceType == workspaceServiceTypeFrontendDist && req.SkipBuild {
			args = append(args, "--skip-build")
		}
	} else if req.SkipBuild {
		args = append(args, "--skip-build")
	}
	if strings.TrimSpace(req.UpstreamURL) != "" {
		args = append(args, "--upstream-url", strings.TrimSpace(req.UpstreamURL))
	}
	if strings.TrimSpace(req.StartCommand) != "" {
		args = append(args, "--command", strings.TrimSpace(req.StartCommand))
	}
	if strings.TrimSpace(req.Workdir) != "" {
		args = append(args, "--workdir", strings.TrimSpace(req.Workdir))
	} else if (serviceType == workspaceServiceTypeHTTP || serviceType == "") && strings.TrimSpace(req.StartCommand) != "" {
		args = append(args, "--workdir", repositoryPath)
	}
	if req.Port > 0 {
		args = append(args, "--port", strconv.Itoa(req.Port))
	}
	if strings.TrimSpace(req.HealthPath) != "" {
		args = append(args, "--health-path", strings.TrimSpace(req.HealthPath))
	}

	cmd := d.commandRunner(ctx, "bash", args...)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return WorkspaceServiceDeployResult{}, fmt.Errorf("deploy test service: %w: %s", err, strings.TrimSpace(string(output)))
	}

	var result WorkspaceServiceDeployResult
	if err := json.Unmarshal(output, &result); err != nil {
		return WorkspaceServiceDeployResult{}, fmt.Errorf("decode deploy output: %w", err)
	}
	return result, nil
}

func normalizeWorkspaceServiceType(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case workspaceServiceTypeFrontendDist:
		return workspaceServiceTypeFrontendDist
	case workspaceServiceTypeHTTP:
		return workspaceServiceTypeHTTP
	}
	return ""
}

func normalizeWorkspaceServiceID(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return "web"
	}
	return trimmed
}

func resolveWorkspaceServiceRepositoryPath(repoRoot string, sessionID string, raw string) (string, error) {
	if trimmed := strings.TrimSpace(raw); trimmed != "" {
		absolute, err := filepath.Abs(trimmed)
		if err != nil {
			return "", err
		}
		return filepath.ToSlash(absolute), nil
	}
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return "", fmt.Errorf("repository root unavailable")
	}
	sessionRepo := buildCodingSessionRepoWorkspacePath(repoRoot, sessionID)
	if sessionRepo != "" {
		gitMarker := filepath.FromSlash(filepath.Join(sessionRepo, ".git"))
		if _, err := os.Stat(gitMarker); err == nil {
			return filepath.ToSlash(sessionRepo), nil
		}
	}
	return filepath.ToSlash(repoRoot), nil
}

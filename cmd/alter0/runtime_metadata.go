package main

import (
	"context"
	"os/exec"
	"runtime/debug"
	"strings"
	"time"

	"alter0/internal/interfaces/web"
)

const runtimeGitRevisionTimeout = 2 * time.Second

type staticRuntimeInfoProvider struct {
	info web.RuntimeInfo
}

func newRuntimeInfoProvider(startedAt time.Time, workingDir string) *staticRuntimeInfoProvider {
	return &staticRuntimeInfoProvider{
		info: web.RuntimeInfo{
			StartedAt:  startedAt.UTC(),
			CommitHash: resolveRuntimeCommitHash(workingDir),
		},
	}
}

func (p *staticRuntimeInfoProvider) GetRuntimeInfo() web.RuntimeInfo {
	if p == nil {
		return web.RuntimeInfo{}
	}
	return p.info
}

func resolveRuntimeCommitHash(workingDir string) string {
	if revision := readRuntimeBuildInfoSetting("vcs.revision"); revision != "" {
		return revision
	}

	dir := strings.TrimSpace(workingDir)
	if dir == "" {
		return ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), runtimeGitRevisionTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func readRuntimeBuildInfoSetting(key string) string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, setting := range info.Settings {
		if strings.TrimSpace(setting.Key) == key {
			return strings.TrimSpace(setting.Value)
		}
	}
	return ""
}

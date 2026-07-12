package application

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type NativeMemoriesFeatureStatus struct {
	Available  bool
	Diagnostic string
}

type RuntimeMemoriesStatus struct {
	Available         bool       `json:"available"`
	Enabled           bool       `json:"enabled"`
	GenerateMemories  bool       `json:"generate_memories"`
	UseMemories       bool       `json:"use_memories"`
	DirectoryExists   bool       `json:"directory_exists"`
	FileCount         int        `json:"file_count"`
	LastUpdatedAt     *time.Time `json:"last_updated_at,omitempty"`
	FeatureDiagnostic string     `json:"feature_diagnostic,omitempty"`
}

func inspectRuntimeMemories(activeHome string, feature NativeMemoriesFeatureStatus, enabled bool, generate bool, use bool) *RuntimeMemoriesStatus {
	status := &RuntimeMemoriesStatus{
		Available:         feature.Available,
		Enabled:           feature.Available && enabled,
		GenerateMemories:  feature.Available && generate,
		UseMemories:       feature.Available && use,
		FeatureDiagnostic: strings.TrimSpace(feature.Diagnostic),
	}
	if strings.TrimSpace(activeHome) == "" {
		return status
	}
	memoryDir := filepath.Join(activeHome, "memories")
	info, err := os.Stat(memoryDir)
	if err != nil || !info.IsDir() {
		return status
	}
	status.DirectoryExists = true
	_ = filepath.WalkDir(memoryDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return nil
		}
		status.FileCount++
		entryInfo, entryErr := entry.Info()
		if entryErr != nil {
			return nil
		}
		updatedAt := entryInfo.ModTime().UTC()
		if status.LastUpdatedAt == nil || updatedAt.After(*status.LastUpdatedAt) {
			status.LastUpdatedAt = &updatedAt
		}
		return nil
	})
	return status
}

func DetectNativeMemoriesFeature(ctx context.Context, command string) NativeMemoriesFeatureStatus {
	command = strings.TrimSpace(command)
	if command == "" {
		command = "codex"
	}
	output, err := exec.CommandContext(ctx, command, "features", "list").CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(output))
		if detail == "" {
			detail = err.Error()
		}
		return NativeMemoriesFeatureStatus{Diagnostic: fmt.Sprintf("unable to inspect Codex memories feature: %s", detail)}
	}
	return ParseNativeMemoriesFeatureList(string(output))
}

func ParseNativeMemoriesFeatureList(output string) NativeMemoriesFeatureStatus {
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) > 0 && fields[0] == "memories" {
			return NativeMemoriesFeatureStatus{Available: true}
		}
	}
	return NativeMemoriesFeatureStatus{Diagnostic: "installed Codex CLI does not expose the memories feature"}
}

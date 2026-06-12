package infrastructure

import (
	"fmt"
	"os"
	"path/filepath"
)

func resolveToolRepoRoot() (string, error) {
	root, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve tool repo root: %w", err)
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve tool repo root: %w", err)
	}
	return absolute, nil
}

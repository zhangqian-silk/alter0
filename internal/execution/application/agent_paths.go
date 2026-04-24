package application

import (
	"path/filepath"
)

func agentPrivatePathIDs(agentID string) []string {
	canonical := normalizeMemoryAgentID(agentID)
	if canonical == "" {
		return nil
	}
	return []string{canonical}
}

func agentPrivateRelativePaths(agentID string, parts ...string) []string {
	ids := agentPrivatePathIDs(agentID)
	if len(ids) == 0 {
		return nil
	}
	paths := make([]string, 0, len(ids))
	seen := map[string]struct{}{}
	for _, id := range ids {
		segments := append([]string{".alter0", "agents", id}, parts...)
		path := filepath.ToSlash(filepath.Join(segments...))
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	return paths
}

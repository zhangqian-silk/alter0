package application

import (
	"regexp"
	"strings"
)

var memoryTokenPattern = regexp.MustCompile(`[\\p{Han}]|[A-Za-z0-9_]+`)

func mergeStringMap(base map[string]string, overlay map[string]string) map[string]string {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}
	merged := map[string]string{}
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range overlay {
		merged[key] = value
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}

func normalizeSnippet(content string, maxSnippets int) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ""
	}
	normalized := strings.Join(strings.Fields(trimmed), " ")
	if maxSnippets <= 0 {
		return normalized
	}
	runes := []rune(normalized)
	if len(runes) <= maxSnippets {
		return normalized
	}
	return string(runes[:maxSnippets]) + "..."
}

func estimateTokenCount(content string) int {
	if strings.TrimSpace(content) == "" {
		return 0
	}
	return len(memoryTokenPattern.FindAllString(content, -1))
}

package infrastructure

import (
	"context"
	"strings"

	execdomain "alter0/internal/execution/domain"
)

func (p *HybridNLProcessor) finalizeDeliveryOutput(
	_ context.Context,
	_ string,
	output string,
	metadata map[string]string,
) (string, error) {
	if !isTravelSkillRun(metadata) {
		return output, nil
	}
	guideURL, err := resolveCurrentTravelGuideURL(metadata)
	if err != nil {
		return "", err
	}
	if guideURL == "" {
		return output, nil
	}
	return appendTravelGuideURL(output, guideURL), nil
}

func resolveCurrentTravelGuideURL(metadata map[string]string) (string, error) {
	sessionID := strings.TrimSpace(metadataValue(metadata, execdomain.RuntimeSessionIDMetadataKey))
	if sessionID == "" {
		return "", nil
	}
	repoRoot, err := resolveToolRepoRoot()
	if err != nil {
		return "", err
	}
	guideURL, ok, err := resolvePublishedWorkspaceServiceURL(repoRoot, sessionID, "travel")
	if err != nil {
		return "", err
	}
	if !ok {
		return "", nil
	}
	return strings.TrimSpace(guideURL), nil
}

func appendTravelGuideURL(output string, guideURL string) string {
	trimmedOutput := strings.TrimSpace(output)
	guideURL = strings.TrimSpace(guideURL)
	if guideURL == "" || trimmedOutput == "" || strings.Contains(trimmedOutput, guideURL) {
		return output
	}
	return trimmedOutput + "\n\nguide_html_url: " + guideURL
}

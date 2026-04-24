package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	execapp "alter0/internal/execution/application"
	execdomain "alter0/internal/execution/domain"
)

type SessionProfileCodexExtractor struct {
	processor execdomain.NLProcessor
}

func NewSessionProfileCodexExtractor(processor execdomain.NLProcessor) *SessionProfileCodexExtractor {
	if processor == nil {
		return nil
	}
	return &SessionProfileCodexExtractor{processor: processor}
}

func (e *SessionProfileCodexExtractor) ExtractPatch(
	ctx context.Context,
	request execapp.SessionProfileExtractionRequest,
) (map[string]string, error) {
	if e == nil || e.processor == nil {
		return nil, nil
	}
	prompt := buildSessionProfileExtractionPrompt(request)
	output, err := e.processor.Process(ctx, prompt, map[string]string{
		execdomain.CodexRuntimeStrategyMetadataKey: execdomain.CodexRuntimeStrategyPlain,
	})
	if err != nil {
		return nil, err
	}
	return parseSessionProfilePatch(output)
}

func buildSessionProfileExtractionPrompt(request execapp.SessionProfileExtractionRequest) string {
	lines := []string{
		"You extract session profile patches from one user message.",
		"Return only a JSON object.",
		"Only include keys that are writable schema fields.",
		"Do not include readonly fields.",
		"Do not invent values not supported by the message.",
		"If no field should change, return {}.",
		"",
		"Schema:",
	}
	for _, field := range request.Agent.SessionProfileFields {
		if field.ReadOnly {
			continue
		}
		line := fmt.Sprintf("- key=%s; label=%s", strings.TrimSpace(field.Key), strings.TrimSpace(field.Label))
		if description := strings.TrimSpace(field.Description); description != "" {
			line += "; description=" + description
		}
		lines = append(lines, line)
	}
	lines = append(lines, "", "Existing attributes:")
	if len(request.ExistingAttributes) == 0 {
		lines = append(lines, "{}")
	} else {
		encoded, _ := json.Marshal(request.ExistingAttributes)
		lines = append(lines, string(encoded))
	}
	lines = append(lines, "", "Latest user message:", strings.TrimSpace(request.Message.Content))
	return strings.Join(lines, "\n")
}

func parseSessionProfilePatch(raw string) (map[string]string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	if strings.HasPrefix(trimmed, "```") {
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "```json"))
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
		trimmed = strings.TrimSpace(strings.TrimSuffix(trimmed, "```"))
	}
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start >= 0 && end > start {
		trimmed = trimmed[start : end+1]
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return nil, err
	}
	patch := map[string]string{}
	for key, value := range payload {
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				patch[normalizedKey] = strings.TrimSpace(typed)
			}
		case float64:
			patch[normalizedKey] = strconv.FormatFloat(typed, 'f', -1, 64)
		case bool:
			patch[normalizedKey] = strconv.FormatBool(typed)
		}
	}
	if len(patch) == 0 {
		return nil, nil
	}
	return patch, nil
}

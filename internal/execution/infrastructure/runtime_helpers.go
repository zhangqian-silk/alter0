package infrastructure

import (
	"encoding/json"
	"strings"

	execdomain "alter0/internal/execution/domain"
)

func isTravelSkillRun(metadata map[string]string) bool {
	return hasSelectedSkill(metadata, "travel")
}

func hasSelectedSkill(metadata map[string]string, skillID string) bool {
	raw := strings.TrimSpace(metadataValue(metadata, execdomain.SkillContextMetadataKey))
	skillID = strings.TrimSpace(skillID)
	if raw == "" || skillID == "" {
		return false
	}
	var context execdomain.SkillContext
	if err := json.Unmarshal([]byte(raw), &context); err != nil {
		return false
	}
	for _, skill := range context.Skills {
		if strings.EqualFold(strings.TrimSpace(skill.ID), skillID) {
			return true
		}
	}
	return false
}

func renderDeliverablesInstruction(metadata map[string]string) string {
	raw := strings.TrimSpace(metadataValue(metadata, execdomain.DeliverablesMetadataKey))
	if raw == "" {
		return ""
	}
	var deliverables []execdomain.Deliverable
	if err := json.Unmarshal([]byte(raw), &deliverables); err != nil || len(deliverables) == 0 {
		return ""
	}
	lines := []string{
		"Current delivery contract:",
		"Do not finish with only a conversational answer when explicit deliverables are declared. Drive execution until the required deliverables are produced or you can clearly explain the blocker.",
	}
	for _, item := range deliverables {
		label := strings.TrimSpace(item.Label)
		if label == "" {
			continue
		}
		parts := make([]string, 0, 4)
		if item.Required {
			parts = append(parts, "required")
		} else {
			parts = append(parts, "optional")
		}
		if format := strings.TrimSpace(item.Format); format != "" {
			parts = append(parts, format)
		}
		if field := strings.TrimSpace(item.SessionAttributeKey); field != "" {
			parts = append(parts, "session attribute "+field)
		}
		line := "- " + label
		if description := strings.TrimSpace(item.Description); description != "" {
			line += ": " + description
		}
		if len(parts) > 0 {
			line += " (" + strings.Join(parts, ", ") + ")"
		}
		lines = append(lines, line)
	}
	if len(lines) == 2 {
		return ""
	}
	return strings.Join(lines, "\n")
}

func setExecutionSource(metadata map[string]string, source string) {
	if len(metadata) == 0 {
		return
	}
	metadata[execdomain.ExecutionSourceMetadataKey] = strings.TrimSpace(source)
}

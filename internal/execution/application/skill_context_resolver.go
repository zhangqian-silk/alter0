package application

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	controldomain "alter0/internal/control/domain"
	execdomain "alter0/internal/execution/domain"
	shareddomain "alter0/internal/shared/domain"
)

const (
	defaultSkillPriority = 100

	skillPriorityKey          = "skill.priority"
	skillDescriptionKey       = "skill.description"
	skillParameterTemplateKey = "skill.parameters"
	skillConstraintsKey       = "skill.constraints"

	skillIncludeFilterKey = "alter0.skills.include"
	skillExcludeFilterKey = "alter0.skills.exclude"
)

type SkillCapabilitySource interface {
	ListCapabilitiesByType(capabilityType controldomain.CapabilityType) []controldomain.Capability
}

type skillContextResolver struct {
	source SkillCapabilitySource
}

func newSkillContextResolver(source SkillCapabilitySource) *skillContextResolver {
	if source == nil {
		return nil
	}
	return &skillContextResolver{source: source}
}

func (r *skillContextResolver) Resolve(msg shareddomain.UnifiedMessage) (execdomain.SkillContext, []string) {
	if r == nil || r.source == nil {
		return execdomain.SkillContext{}, nil
	}

	includeFilter := parseLookupSet(metadataValue(msg.Metadata, skillIncludeFilterKey))
	excludeFilter := parseLookupSet(metadataValue(msg.Metadata, skillExcludeFilterKey))

	items := r.source.ListCapabilitiesByType(controldomain.CapabilityTypeSkill)
	skills := make([]execdomain.SkillSpec, 0, len(items))
	for _, item := range items {
		if !item.Enabled {
			continue
		}
		skill := buildSkillSpec(item)
		if len(includeFilter) > 0 && !matchesSkillFilter(skill, includeFilter) {
			continue
		}
		if matchesSkillFilter(skill, excludeFilter) {
			continue
		}
		skills = append(skills, skill)
	}

	sort.Slice(skills, func(i, j int) bool {
		if skills[i].Priority == skills[j].Priority {
			if strings.EqualFold(skills[i].Name, skills[j].Name) {
				return strings.ToLower(skills[i].ID) < strings.ToLower(skills[j].ID)
			}
			return strings.ToLower(skills[i].Name) < strings.ToLower(skills[j].Name)
		}
		return skills[i].Priority > skills[j].Priority
	})

	ids := make([]string, 0, len(skills))
	for _, skill := range skills {
		ids = append(ids, skill.ID)
	}

	return execdomain.SkillContext{
		Protocol: execdomain.SkillContextProtocolVersion,
		Skills:   skills,
	}, ids
}

func buildSkillSpec(capability controldomain.Capability) execdomain.SkillSpec {
	description := strings.TrimSpace(metadataValue(capability.Metadata, skillDescriptionKey))
	if description == "" {
		description = strings.TrimSpace(capability.Name)
	}
	return execdomain.SkillSpec{
		ID:                strings.TrimSpace(capability.ID),
		Name:              strings.TrimSpace(capability.Name),
		Description:       description,
		Priority:          parseSkillPriority(capability.Metadata),
		ParameterTemplate: parseParameterTemplate(capability.Metadata),
		Constraints:       parseList(metadataValue(capability.Metadata, skillConstraintsKey)),
	}
}

func parseSkillPriority(metadata map[string]string) int {
	raw := strings.TrimSpace(metadataValue(metadata, skillPriorityKey))
	if raw == "" {
		return defaultSkillPriority
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return defaultSkillPriority
	}
	return value
}

func parseParameterTemplate(metadata map[string]string) map[string]string {
	raw := strings.TrimSpace(metadataValue(metadata, skillParameterTemplateKey))
	if raw == "" {
		return nil
	}
	parsed := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil
	}
	if len(parsed) == 0 {
		return nil
	}
	return parsed
}

func parseList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	items := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		lookup := strings.ToLower(value)
		if _, exists := seen[lookup]; exists {
			continue
		}
		seen[lookup] = struct{}{}
		items = append(items, value)
	}
	return items
}

func parseLookupSet(raw string) map[string]struct{} {
	items := parseList(raw)
	if len(items) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(items))
	for _, item := range items {
		set[strings.ToLower(item)] = struct{}{}
	}
	return set
}

func matchesSkillFilter(skill execdomain.SkillSpec, filter map[string]struct{}) bool {
	if len(filter) == 0 {
		return false
	}
	if _, ok := filter[strings.ToLower(skill.ID)]; ok {
		return true
	}
	if _, ok := filter[strings.ToLower(skill.Name)]; ok {
		return true
	}
	return false
}

func metadataValue(metadata map[string]string, key string) string {
	if len(metadata) == 0 {
		return ""
	}
	return metadata[key]
}

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
	skillGuideKey             = "skill.guide"
	skillParameterTemplateKey = "skill.parameters"
	skillConstraintsKey       = "skill.constraints"
	skillAbilitiesKey         = "skill.abilities"

	skillIncludeFilterKey = "alter0.skills.include"
	skillExcludeFilterKey = "alter0.skills.exclude"
)

type SkillCapabilitySource interface {
	ListCapabilitiesByType(capabilityType controldomain.CapabilityType) []controldomain.Capability
}

type skillResolution struct {
	Context       execdomain.SkillContext
	InjectedIDs   []string
	ConflictTypes []string
}

type parameterOwner struct {
	skillID string
	value   string
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

func (r *skillContextResolver) Resolve(msg shareddomain.UnifiedMessage) skillResolution {
	resolution := skillResolution{
		Context: execdomain.SkillContext{Protocol: execdomain.SkillContextProtocolVersion},
	}
	if r == nil || r.source == nil {
		return resolution
	}

	includeFilter := parseLookupSet(metadataValue(msg.Metadata, skillIncludeFilterKey))
	excludeFilter := parseLookupSet(metadataValue(msg.Metadata, skillExcludeFilterKey))

	items := r.source.ListCapabilitiesByType(controldomain.CapabilityTypeSkill)
	candidates := make([]execdomain.SkillSpec, 0, len(items))
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
		candidates = append(candidates, skill)
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Priority == candidates[j].Priority {
			if strings.EqualFold(candidates[i].Name, candidates[j].Name) {
				return strings.ToLower(candidates[i].ID) < strings.ToLower(candidates[j].ID)
			}
			return strings.ToLower(candidates[i].Name) < strings.ToLower(candidates[j].Name)
		}
		return candidates[i].Priority > candidates[j].Priority
	})

	skills, conflicts := resolveNameConflicts(candidates)
	skills, parameters, conflictItems := resolveAbilityAndParameterConflicts(skills)
	conflicts = append(conflicts, conflictItems...)

	resolution.Context.Skills = skills
	if len(parameters) > 0 {
		resolution.Context.ResolvedParameters = parameters
	}
	if len(conflicts) > 0 {
		resolution.Context.Conflicts = conflicts
	}

	resolution.InjectedIDs = make([]string, 0, len(skills))
	for _, skill := range skills {
		resolution.InjectedIDs = append(resolution.InjectedIDs, skill.ID)
	}
	resolution.ConflictTypes = uniqueConflictTypes(conflicts)
	return resolution
}

func resolveNameConflicts(skills []execdomain.SkillSpec) ([]execdomain.SkillSpec, []execdomain.SkillConflict) {
	if len(skills) == 0 {
		return nil, nil
	}

	selected := make([]execdomain.SkillSpec, 0, len(skills))
	selectedByName := map[string]execdomain.SkillSpec{}
	conflicts := make([]execdomain.SkillConflict, 0)
	for _, skill := range skills {
		key := strings.ToLower(strings.TrimSpace(skill.Name))
		if key == "" {
			key = strings.ToLower(strings.TrimSpace(skill.ID))
		}
		if winner, exists := selectedByName[key]; exists {
			conflicts = append(conflicts, execdomain.SkillConflict{
				Type:           execdomain.SkillConflictTypeDuplicateName,
				Key:            key,
				WinnerSkillID:  winner.ID,
				DroppedSkillID: skill.ID,
				Detail:         "duplicate skill name, keep higher priority entry",
			})
			continue
		}
		selectedByName[key] = skill
		selected = append(selected, skill)
	}
	return selected, conflicts
}

func resolveAbilityAndParameterConflicts(skills []execdomain.SkillSpec) ([]execdomain.SkillSpec, map[string]string, []execdomain.SkillConflict) {
	if len(skills) == 0 {
		return nil, nil, nil
	}

	abilityOwners := map[string]string{}
	parameterOwners := map[string]parameterOwner{}
	resolvedParameters := map[string]string{}
	conflicts := make([]execdomain.SkillConflict, 0)

	resolvedSkills := make([]execdomain.SkillSpec, 0, len(skills))
	for _, skill := range skills {
		skillCopy := skill
		skillCopy.Abilities = resolveSkillAbilities(skill, abilityOwners, &conflicts)
		skillCopy.ParameterTemplate = resolveSkillParameters(skill, parameterOwners, resolvedParameters, &conflicts)
		resolvedSkills = append(resolvedSkills, skillCopy)
	}

	if len(resolvedParameters) == 0 {
		resolvedParameters = nil
	}
	return resolvedSkills, resolvedParameters, conflicts
}

func resolveSkillAbilities(
	skill execdomain.SkillSpec,
	abilityOwners map[string]string,
	conflicts *[]execdomain.SkillConflict,
) []string {
	if len(skill.Abilities) == 0 {
		return nil
	}

	abilities := make([]string, 0, len(skill.Abilities))
	for _, ability := range skill.Abilities {
		lookup := strings.ToLower(strings.TrimSpace(ability))
		if lookup == "" {
			continue
		}
		winner, exists := abilityOwners[lookup]
		if exists {
			*conflicts = append(*conflicts, execdomain.SkillConflict{
				Type:           execdomain.SkillConflictTypeDuplicateAbility,
				Key:            lookup,
				WinnerSkillID:  winner,
				DroppedSkillID: skill.ID,
				Detail:         "duplicate ability, keep higher priority owner",
			})
			continue
		}
		abilityOwners[lookup] = skill.ID
		abilities = append(abilities, ability)
	}
	if len(abilities) == 0 {
		return nil
	}
	return abilities
}

func resolveSkillParameters(
	skill execdomain.SkillSpec,
	parameterOwners map[string]parameterOwner,
	resolvedParameters map[string]string,
	conflicts *[]execdomain.SkillConflict,
) map[string]string {
	if len(skill.ParameterTemplate) == 0 {
		return nil
	}

	keys := make([]string, 0, len(skill.ParameterTemplate))
	for key := range skill.ParameterTemplate {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parameterTemplate := map[string]string{}
	for _, key := range keys {
		value := strings.TrimSpace(skill.ParameterTemplate[key])
		if value == "" {
			continue
		}
		lookup := strings.ToLower(strings.TrimSpace(key))
		if lookup == "" {
			continue
		}

		owner, exists := parameterOwners[lookup]
		if exists && owner.value != value {
			*conflicts = append(*conflicts, execdomain.SkillConflict{
				Type:           execdomain.SkillConflictTypeParameterConflict,
				Key:            lookup,
				WinnerSkillID:  owner.skillID,
				DroppedSkillID: skill.ID,
				Detail:         "parameter value conflict, keep higher priority setting",
			})
			continue
		}
		if !exists {
			parameterOwners[lookup] = parameterOwner{skillID: skill.ID, value: value}
			resolvedParameters[lookup] = value
		}
		parameterTemplate[key] = value
	}
	if len(parameterTemplate) == 0 {
		return nil
	}
	return parameterTemplate
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
		Guide:             strings.TrimSpace(metadataValue(capability.Metadata, skillGuideKey)),
		Priority:          parseSkillPriority(capability.Metadata),
		ParameterTemplate: parseParameterTemplate(capability.Metadata),
		Constraints:       parseList(metadataValue(capability.Metadata, skillConstraintsKey)),
		Abilities:         parseList(metadataValue(capability.Metadata, skillAbilitiesKey)),
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
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}

	if strings.HasPrefix(trimmed, "[") {
		var items []string
		if err := json.Unmarshal([]byte(trimmed), &items); err == nil {
			return normalizeList(items)
		}
	}

	return normalizeList(strings.Split(trimmed, ","))
}

func normalizeList(parts []string) []string {
	if len(parts) == 0 {
		return nil
	}
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
	if len(items) == 0 {
		return nil
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

func uniqueConflictTypes(conflicts []execdomain.SkillConflict) []string {
	if len(conflicts) == 0 {
		return nil
	}
	items := make([]string, 0, len(conflicts))
	seen := map[string]struct{}{}
	for _, conflict := range conflicts {
		item := strings.TrimSpace(conflict.Type)
		if item == "" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		items = append(items, item)
	}
	if len(items) == 0 {
		return nil
	}
	return items
}

func metadataValue(metadata map[string]string, key string) string {
	if len(metadata) == 0 {
		return ""
	}
	return metadata[key]
}

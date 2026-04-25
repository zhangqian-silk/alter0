package application

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	controldomain "alter0/internal/control/domain"
	execdomain "alter0/internal/execution/domain"
	shareddomain "alter0/internal/shared/domain"
)

const (
	defaultSkillPriority    = 100
	agentOwnedSkillPriority = 780

	skillPriorityKey          = "skill.priority"
	skillDescriptionKey       = "skill.description"
	skillGuideKey             = "skill.guide"
	skillParameterTemplateKey = "skill.parameters"
	skillConstraintsKey       = "skill.constraints"
	skillAbilitiesKey         = "skill.abilities"
	skillFilePathKey          = "skill.file_path"
	skillWritableKey          = "skill.writable"

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
	return &skillContextResolver{source: source}
}

func (r *skillContextResolver) Resolve(msg shareddomain.UnifiedMessage) skillResolution {
	resolution := skillResolution{
		Context: execdomain.SkillContext{Protocol: execdomain.SkillContextProtocolVersion},
	}
	if r == nil {
		return resolution
	}

	includeFilter := parseLookupSet(metadataValue(msg.Metadata, skillIncludeFilterKey))
	excludeFilter := parseLookupSet(metadataValue(msg.Metadata, skillExcludeFilterKey))
	agentOwnedSkill, hasAgentOwnedSkill := resolveAgentOwnedSkill(msg)

	var items []controldomain.Capability
	if r.source != nil {
		items = r.source.ListCapabilitiesByType(controldomain.CapabilityTypeSkill)
	}
	candidates := make([]execdomain.SkillSpec, 0, len(items)+1)
	for _, item := range items {
		if !item.Enabled {
			continue
		}
		skill := buildSkillSpec(item)
		if len(includeFilter) > 0 && !matchesSkillFilter(skill, includeFilter) && !isAgentOwnedSkill(skill, agentOwnedSkill) {
			continue
		}
		if matchesSkillFilter(skill, excludeFilter) {
			continue
		}
		candidates = append(candidates, skill)
	}
	if hasAgentOwnedSkill {
		candidates = append(candidates, agentOwnedSkill)
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

func resolveAgentOwnedSkill(msg shareddomain.UnifiedMessage) (execdomain.SkillSpec, bool) {
	agentID := strings.TrimSpace(metadataValue(msg.Metadata, execdomain.AgentIDMetadataKey))
	if agentID == "" {
		return execdomain.SkillSpec{}, false
	}
	normalizedAgentID := normalizeMemoryAgentID(agentID)
	if normalizedAgentID == "" {
		return execdomain.SkillSpec{}, false
	}
	agentName := strings.TrimSpace(metadataValue(msg.Metadata, execdomain.AgentNameMetadataKey))
	if agentName == "" {
		agentName = agentID
	}
	agentCapabilities := parseList(metadataValue(msg.Metadata, execdomain.AgentCapabilitiesMetadataKey))
	filePath := agentPrivateRelativePaths(agentID, "SKILL.md")[0]
	_ = ensureAgentOwnedSkillFile(agentID, filePath, agentOwnedSkillDocument(agentID, agentName, agentCapabilities))
	return execdomain.SkillSpec{
		ID:          agentOwnedSkillID(normalizedAgentID),
		Name:        agentName + " Skill",
		Description: agentOwnedSkillDescription(agentID, agentName, agentCapabilities),
		Guide:       agentOwnedSkillGuide(filePath, agentID, agentName, agentCapabilities),
		Priority:    agentOwnedSkillPriority,
		FilePath:    filePath,
		Writable:    true,
	}, true
}

func isAgentOwnedSkill(skill execdomain.SkillSpec, current execdomain.SkillSpec) bool {
	if strings.TrimSpace(current.ID) == "" {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(skill.ID), strings.TrimSpace(current.ID))
}

func agentOwnedSkillID(agentID string) string {
	trimmed := normalizeMemoryAgentID(agentID)
	if trimmed == "" {
		trimmed = "unknown"
	}
	return "agent-skill-" + trimmed
}

func agentOwnedSkillGuide(filePath string, agentID string, agentName string, capabilities []string) string {
	if isTravelAgent(agentID, agentName, capabilities) {
		return strings.Join([]string{
			"# travel agent-owned skill",
			"",
			"## Runtime contract",
			"",
			"- This skill is private to the current travel agent and is file-backed. Read `" + strings.TrimSpace(filePath) + "` before applying durable city-page, itinerary, transit, food, and map-output rules.",
			"- Treat the file as the canonical reusable rulebook for this travel agent's page structure, output sections, rendering conventions, and long-lived travel preferences requested by the user.",
			"- `AGENTS.md` remains the source of repository or workspace operating rules. Keep travel-domain rules and page conventions in this skill instead of repository policy files.",
			"",
			"## When to update",
			"",
			"- Update the skill when the user states a durable preference that should affect future travel pages or itineraries handled by this agent, such as section ordering, city-page density, transit-first defaults, food recommendation framing, or map-oriented output conventions.",
			"- Keep updates reusable across future cities and trips handled by the same travel agent instead of encoding one itinerary verbatim.",
			"",
			"## When not to update",
			"",
			"- Do not write one-off trip constraints, temporary dates, current companions, single-city event schedules, or session-only itinerary details into the skill.",
			"- Do not duplicate repository-wide operating rules that belong in `AGENTS.md`.",
			"- Do not rewrite the whole file for a small preference change. Preserve existing rules and apply focused edits.",
		}, "\n")
	}
	return strings.Join([]string{
		"# agent-owned skill",
		"",
		"## Runtime contract",
		"",
		"- This skill is private to the current agent and is file-backed. Read `" + strings.TrimSpace(filePath) + "` before applying durable agent-specific working rules.",
		"- Treat the file as the canonical rulebook for this agent's reusable execution patterns, output structure, domain heuristics, and long-lived user-requested preferences.",
		"- `AGENTS.md` remains the source of repository or workspace operating rules. Use this skill for agent-specific reusable behavior instead of repository policy.",
		"",
		"## When to update",
		"",
		"- Update the skill when the user states a durable preference that should affect future tasks handled by this agent, such as output shape, preferred workflow, naming style, review checklist, or reusable domain heuristics.",
		"- Keep updates reusable across future tasks for the same agent instead of encoding one current request verbatim.",
		"",
		"## When not to update",
		"",
		"- Do not write one-off task constraints, temporary deadlines, current ticket details, or session-specific facts into the skill.",
		"- Do not duplicate repository-wide operating rules that belong in `AGENTS.md`.",
		"- Do not rewrite the whole file for a small preference change. Preserve existing rules and apply focused edits.",
	}, "\n")
}

func agentOwnedSkillDescription(agentID string, agentName string, capabilities []string) string {
	if isTravelAgent(agentID, agentName, capabilities) {
		return "Private reusable rulebook for the current travel agent's city-page structure, itinerary composition, rendering conventions, and stable travel preferences."
	}
	return "Private reusable rulebook for the current agent's durable working patterns, output structure, and stable user-requested preferences."
}

func agentOwnedSkillDocument(agentID string, agentName string, capabilities []string) string {
	trimmedAgentID := strings.TrimSpace(agentID)
	if trimmedAgentID == "" {
		trimmedAgentID = "unknown"
	}
	trimmedAgentName := strings.TrimSpace(agentName)
	if trimmedAgentName == "" {
		trimmedAgentName = trimmedAgentID
	}
	if isTravelAgent(agentID, agentName, capabilities) {
		return strings.Join([]string{
			"# " + trimmedAgentName + " Skill",
			"",
			"## Purpose",
			"",
			"This file is the private reusable rulebook for the `" + trimmedAgentID + "` travel agent.",
			"",
			"## Stable travel contract",
			"",
			"- One city page represents one city-focused travel space.",
			"- The Workspace detail view and standalone HTML city page must stay aligned to the same guide content.",
			"- Page content should remain city-specific and must not mix multiple target cities into one page.",
			"- Preserve structured guide fields so later revisions can update city pages without rebuilding from scratch.",
			"",
			"## Default content expectations",
			"",
			"- Provide a clear city title and short summary.",
			"- Keep visible sections for highlights, day-by-day route planning, metro or transit guidance, food recommendations, practical notes, and map-oriented hints when available.",
			"- Prefer concise, scan-friendly sections that can be extended without breaking the page layout.",
			"",
			"## Store Here",
			"",
			"- Durable travel-page structure, section ordering, tone, naming conventions, and stable rendering preferences.",
			"- Reusable itinerary composition heuristics, transit defaults, food recommendation framing, and map-output conventions requested by the user.",
			"- Stable travel-agent defaults that should apply across future city pages handled by this agent.",
			"",
			"## Keep Out",
			"",
			"- Repository or workspace operating rules that belong in `.alter0/agents/" + normalizeMemoryAgentID(trimmedAgentID) + "/AGENTS.md`.",
			"- One-off trip constraints, temporary dates, current-session notes, or single-city exceptions that should stay in the target guide data.",
			"- Shared repository policy or non-travel reusable behavior that should live outside this travel-agent skill.",
			"",
			"## Editing Rules",
			"",
			"- Apply focused updates instead of replacing the whole file.",
			"- Preserve stable rules unless the user clearly changes a durable preference.",
			"- Promote only reusable travel guidance into this file.",
		}, "\n")
	}
	return strings.Join([]string{
		"# " + trimmedAgentName + " Skill",
		"",
		"## Purpose",
		"",
		"This file is the private reusable rulebook for the `" + trimmedAgentID + "` agent.",
		"",
		"## Store Here",
		"",
		"- Durable execution patterns that should persist across future tasks handled by this agent.",
		"- Stable output structures, review checklists, naming habits, or domain heuristics requested by the user.",
		"- Reusable agent-specific defaults that do not belong in repository-wide policy files.",
		"",
		"## Keep Out",
		"",
		"- Repository or workspace operating rules that belong in `.alter0/agents/" + normalizeMemoryAgentID(trimmedAgentID) + "/AGENTS.md`.",
		"- One-off task constraints, temporary facts, or current-session notes.",
		"- Cross-agent shared domain rules that should live in a global or product-level skill instead.",
		"",
		"## Editing Rules",
		"",
		"- Apply focused updates instead of replacing the whole file.",
		"- Preserve stable rules unless the user clearly changes a durable preference.",
		"- Promote only reusable guidance into this file.",
	}, "\n")
}

func isTravelAgent(agentID string, agentName string, capabilities []string) bool {
	for _, capability := range capabilities {
		if strings.EqualFold(strings.TrimSpace(capability), "travel") {
			return true
		}
	}
	lookup := strings.ToLower(strings.TrimSpace(agentID) + " " + strings.TrimSpace(agentName))
	return strings.Contains(lookup, "travel")
}

func ensureAgentOwnedSkillFile(agentID string, relativePath string, content string) error {
	repoRoot, err := resolveMemoryRepoRoot()
	if err != nil {
		return err
	}
	absolutePath := filepath.Join(repoRoot, filepath.FromSlash(strings.TrimSpace(relativePath)))
	if _, err := os.Stat(absolutePath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	for _, legacyPath := range agentPrivateRelativePaths(agentID, "SKILL.md")[1:] {
		legacyAbsolutePath := filepath.Join(repoRoot, filepath.FromSlash(legacyPath))
		if raw, err := os.ReadFile(legacyAbsolutePath); err == nil {
			content = string(raw)
			break
		}
	}
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(absolutePath, []byte(content), 0o644)
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
		FilePath:          strings.TrimSpace(metadataValue(capability.Metadata, skillFilePathKey)),
		Writable:          parseSkillWritable(capability.Metadata),
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

func parseSkillWritable(metadata map[string]string) bool {
	raw := strings.TrimSpace(metadataValue(metadata, skillWritableKey))
	if raw == "" {
		return false
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false
	}
	return value
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

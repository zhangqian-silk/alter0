package main

import (
	"os"
	"path/filepath"
	"strings"

	controlapp "alter0/internal/control/application"
	controldomain "alter0/internal/control/domain"
)

const (
	builtinSkillPriorityKey    = "skill.priority"
	builtinSkillDescriptionKey = "skill.description"
	builtinSkillGuideKey       = "skill.guide"
	builtinSkillFilePathKey    = "skill.file_path"
	builtinSkillWritableKey    = "skill.writable"
)

func registerBuiltinSkills(control *controlapp.Service) {
	for _, skill := range builtinSkills() {
		mustUpsertSkill(control, skill)
	}
}

func ensureBuiltinSkillFiles() error {
	for _, skill := range builtinSkills() {
		path := strings.TrimSpace(skill.Metadata[builtinSkillFilePathKey])
		if path == "" {
			continue
		}
		content := builtinSkillFileContent(skill.ID)
		if strings.TrimSpace(content) == "" {
			continue
		}
		if err := ensureBuiltinSkillFile(path, content); err != nil {
			return err
		}
	}
	return nil
}

func builtinSkills() []controldomain.Skill {
	return []controldomain.Skill{
		{
			ID:      "default-nl",
			Name:    "default-nl",
			Enabled: true,
		},
		{
			ID:      "memory",
			Name:    "Memory",
			Enabled: true,
			Scope:   controldomain.CapabilityScopeGlobal,
			Metadata: map[string]string{
				builtinSkillPriorityKey:    "800",
				builtinSkillDescriptionKey: "Introduce alter0 memory modules, file roles, and durable read/write policy for selected memory files.",
				builtinSkillGuideKey:       memorySkillGuide(),
			},
		},
		{
			ID:      "travel-page",
			Name:    "Travel Page",
			Enabled: true,
			Scope:   controldomain.CapabilityScopeGlobal,
			Metadata: map[string]string{
				builtinSkillPriorityKey:    "760",
				builtinSkillDescriptionKey: "Canonical reusable rulebook for travel city page generation, layout consistency, and stable page preferences.",
				builtinSkillGuideKey:       travelPageSkillGuide(),
				builtinSkillFilePathKey:    ".alter0/skills/travel-page.md",
				builtinSkillWritableKey:    "true",
			},
		},
	}
}

func memorySkillGuide() string {
	return strings.Join([]string{
		"# alter0 memory module",
		"",
		"## Runtime contract",
		"",
		"- If runtime metadata already includes `memory_context`, treat `memory_context.files[]` as the current snapshot of each selected memory file. It carries the canonical path, exists flag, writable flag, updated time, and file content.",
		"- Use the injected `path` as the canonical target. If `exists=false`, create the file at that path instead of inventing a new location.",
		"- If a snapshot ends with `...[truncated]`, or the planned edit is non-trivial, read the target file again before writing.",
		"- The skill defines routing and policy. The actual writable targets still come from `memory_context.files[]`.",
		"",
		"## File roles",
		"",
		"- `USER.md` stores stable user profile, collaboration style, output preferences, and recurring personal facts. Read it when personalization matters. Write only after the user gives a durable preference or identity fact that should persist.",
		"- `SOUL.md` stores hard, long-lived rules and non-negotiable constraints. Read it before making strategy, tone, or policy decisions. Write only when the user explicitly changes durable rules.",
		"- `AGENTS.md` stores repository or workspace operating rules for agents, including coding conventions, role boundaries, and delivery expectations. Read it when entering a repo and before making code or process decisions. Write only when maintainers update project rules.",
		"- `.alter0/memory/long-term/MEMORY.md`, `MEMORY.md`, or `memory.md` stores cross-session durable project memory: stable decisions, architecture facts, validated workflows, and long-lived plans. Read it when the task depends on history beyond the current session. Write after durable conclusions are confirmed.",
		"- `.alter0/memory/YYYY-MM-DD.md` or `memory/YYYY-MM-DD.md` for today stores same-day working memory: recent discoveries, active hypotheses, and task notes that may still change. Read it when continuing today's work. Write after meaningful progress, debugging findings, or short-lived decisions that may later be promoted.",
		"- Yesterday's daily memory is read-only by default. Read it only when today's work continues yesterday's thread or today's file points back to it. Prefer writing new notes to today's daily memory instead of editing yesterday.",
		"",
		"## Read logic",
		"",
		"- Read the smallest relevant set first. Do not open every memory file by default.",
		"- Before strategy, policy, tone, or priority decisions: read `SOUL.md` if available.",
		"- Before repository changes, workflow decisions, or delivery behavior changes: read `AGENTS.md` if available.",
		"- Before personalized wording or preference-sensitive output: read `USER.md` if available.",
		"- Before tasks that depend on prior project history, prior decisions, or multi-session continuity: read long-term memory.",
		"- Before continuing today's work: read today's daily memory first; read yesterday only when the thread clearly continues across days.",
		"- If the task is self-contained and current-turn only, do not read memory files just because they exist.",
		"",
		"## Write routing",
		"",
		"- Write to `USER.md` only for durable user preferences, identity facts, collaboration habits, or stable output expectations.",
		"- Write to `SOUL.md` only for explicit hard rules or non-negotiable constraints confirmed by the user. Do not use it for task notes or ordinary preferences.",
		"- Write to `AGENTS.md` only for repository or workspace operating rules, coding conventions, delivery expectations, or agent workflow rules maintained by the project.",
		"- Write to long-term memory only for stable project facts: confirmed architecture decisions, validated workflows, durable conventions, and reusable historical conclusions.",
		"- Write to today's daily memory for in-progress discoveries, debugging findings, active hypotheses, temporary plans, and same-day task context.",
		"- Treat yesterday's daily memory as read-only unless the user or maintainer explicitly asks to correct historical notes.",
		"- Promote information from daily memory into long-term memory only after it becomes stable, validated, or repeatedly useful across tasks.",
		"",
		"## Conflict rules",
		"",
		"- When files disagree, prefer `SOUL.md` over every other memory file.",
		"- Prefer `AGENTS.md` over long-term memory and daily memory for repository operating rules.",
		"- Prefer long-term memory over daily memory for stable facts.",
		"- If a conflict is unresolved or evidence is weak, do not write it into long-term memory. Keep it in today's daily memory or ask for clarification.",
		"",
		"## Write constraints",
		"",
		"- Keep each file focused on its role. Do not duplicate the same fact across every file.",
		"- Write only durable or high-signal information. Skip temporary shell output, one-off paths, secret tokens, ephemeral file paths, and unverified guesses.",
		"- For daily memory, append concise dated notes or update the most relevant section without rewriting the whole file.",
		"- Preserve existing structure when possible. Prefer surgical edits over full rewrites.",
		"- If the request is just to answer a question and there is no explicit or implicit need to persist memory, do not write memory files.",
	}, "\n")
}

func travelPageSkillGuide() string {
	return strings.Join([]string{
		"# travel page skill",
		"",
		"## Runtime contract",
		"",
		"- This skill is file-backed. Read `.alter0/skills/travel-page.md` before creating or revising reusable travel city page rules.",
		"- Treat the file as the canonical rulebook for `travel` Workspace pages and standalone HTML city pages.",
		"- Pass the current rulebook into `codex_exec` as reusable context, and only update the file when the user provides a durable, reusable page preference.",
		"",
		"## When to update",
		"",
		"- Update the skill when the user expresses a stable preference that should affect future travel city pages, such as page structure, tone, section ordering, field naming, or reusable rendering conventions.",
		"- Keep updates reusable across cities like Wuhan, Chengdu, and Beijing instead of encoding one specific itinerary.",
		"",
		"## When not to update",
		"",
		"- Do not write one-off trip constraints into the skill, such as a specific travel date, budget for a single trip, temporary companions, or one city's temporary event.",
		"- Do not rewrite the entire skill for a normal city-page revision. Keep edits focused and preserve existing stable rules.",
		"",
		"## Ownership",
		"",
		"- `travel-master` owns when to consult and maintain this skill during Workspace chat.",
		"- Concrete city content still belongs to each city page and guide record, not to the reusable skill file.",
	}, "\n")
}

func travelPageSkillDocument() string {
	return strings.Join([]string{
		"# Travel Page Rulebook",
		"",
		"## Purpose",
		"",
		"This file is the canonical reusable rulebook for generating `travel` city pages in Workspace and the corresponding standalone HTML pages.",
		"",
		"## Stable page contract",
		"",
		"- One page represents one city-focused travel space.",
		"- The Workspace detail view and `/products/travel/spaces/{space_id}.html` page must stay aligned to the same guide content.",
		"- Page content should remain city-specific and must not mix multiple target cities into one page.",
		"- Preserve structured guide fields so later revisions can update city pages without rebuilding from scratch.",
		"",
		"## Default content expectations",
		"",
		"- Provide a clear city title and short summary.",
		"- Keep visible sections for highlights, day-by-day route planning, metro or transit guidance, food recommendations, practical notes, and map-oriented hints when available.",
		"- Prefer concise, scan-friendly sections that can be extended without breaking the page layout.",
		"",
		"## Update policy",
		"",
		"- Write only durable, reusable preferences here.",
		"- Good examples: preferred section order, tone, page density, naming conventions, stable HTML presentation rules, and reusable city-page composition guidance.",
		"- Do not store one-off traveler constraints here. Put trip-specific requests into the target city page data instead.",
		"",
		"## Editing constraints",
		"",
		"- Preserve existing rules unless the user clearly changes a stable preference.",
		"- Apply focused updates instead of replacing the whole document.",
		"- If a preference only affects one current city request, keep it in that city page and do not promote it into this rulebook.",
	}, "\n")
}

func builtinSkillFileContent(skillID string) string {
	switch strings.TrimSpace(skillID) {
	case "travel-page":
		return travelPageSkillDocument()
	default:
		return ""
	}
}

func ensureBuiltinSkillFile(path string, content string) error {
	absolute, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil {
		return err
	}
	if _, err := os.Stat(absolute); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
		return err
	}
	return os.WriteFile(absolute, []byte(content), 0o644)
}

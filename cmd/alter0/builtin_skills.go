package main

import (
	"strings"

	controlapp "alter0/internal/control/application"
	controldomain "alter0/internal/control/domain"
)

const (
	builtinSkillPriorityKey    = "skill.priority"
	builtinSkillDescriptionKey = "skill.description"
	builtinSkillGuideKey       = "skill.guide"
)

func registerBuiltinSkills(control *controlapp.Service) {
	for _, skill := range builtinSkills() {
		mustUpsertSkill(control, skill)
	}
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
	}
}

func memorySkillGuide() string {
	return strings.Join([]string{
		"# alter0 memory module",
		"",
		"- If runtime metadata already includes `memory_context`, treat `memory_context.files[]` as the current snapshot of each selected memory file. It carries the canonical path, exists flag, writable flag, updated time, and file content.",
		"- If a snapshot ends with `...[truncated]` or the edit is non-trivial, read the target file again before writing. Use the injected path as the canonical location; if `exists=false`, create the file there.",
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
		"## Read policy",
		"",
		"- Read the smallest relevant set first. Do not open every memory file by default.",
		"- Start with `SOUL.md` and `USER.md` for durable behavior, `AGENTS.md` for repo rules, long-term memory for historical decisions, and daily memory for current work-in-progress.",
		"- When files disagree, prefer `SOUL.md` and `AGENTS.md` over ordinary memory notes. Prefer long-term memory over daily memory for stable facts.",
		"",
		"## Write policy",
		"",
		"- Keep each file focused on its role. Do not duplicate the same fact across every file.",
		"- Write only durable or high-signal information. Skip temporary shell output, one-off paths, and unverified guesses.",
		"- Promote validated facts from daily memory into long-term memory when they become stable or repeatedly useful.",
		"- Record user preferences in `USER.md`, hard rules in `SOUL.md`, repo conventions in `AGENTS.md`, and project history in long-term memory.",
		"- For daily memory, append concise dated notes or update the most relevant section without rewriting the whole file.",
		"- Preserve existing structure when possible. Prefer surgical edits over full rewrites.",
	}, "\n")
}

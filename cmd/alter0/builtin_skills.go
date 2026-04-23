package main

import (
	"fmt"
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
	repoRoot := resolveBuiltinSkillRepoRoot()
	for _, skill := range builtinSkills() {
		path := strings.TrimSpace(skill.Metadata[builtinSkillFilePathKey])
		if path == "" {
			continue
		}
		resolvedPath := path
		if !filepath.IsAbs(resolvedPath) {
			resolvedPath = filepath.Join(repoRoot, resolvedPath)
		}
		if _, err := os.Stat(resolvedPath); err != nil {
			return fmt.Errorf("builtin skill file %s: %w", path, err)
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
			ID:      "deploy-test-service",
			Name:    "Deploy Test Service",
			Enabled: true,
			Scope:   controldomain.CapabilityScopeGlobal,
			Metadata: map[string]string{
				builtinSkillPriorityKey:    "760",
				builtinSkillDescriptionKey: "Session-scoped preview and test-service deployment playbook for the shared alter0 gateway.",
				builtinSkillGuideKey:       deployTestServiceSkillGuide(),
				builtinSkillFilePathKey:    filepath.ToSlash(filepath.Join(".alter0", "skills", "deploy-test-service", "SKILL.md")),
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

func deployTestServiceSkillGuide() string {
	return strings.Join([]string{
		"# deploy test service",
		"",
		"- Use `deploy_test_service` when the user needs a session-scoped preview host or an additional routed test service without changing Nginx.",
		"- Registrations land on `/api/control/workspace-services` and are routed by the shared gateway rather than per-service Nginx edits.",
		"- Host routing is fixed at the gateway: `https://<short_hash>.alter0.cn` for the default `web` service, and `https://<service>.<short_hash>.alter0.cn` for additional services such as `api` or `docs`.",
		"- Default `web` deploys should register `service_type=http` and boot the current session backend so the preview host serves both the latest frontend build and `/api/*` from the same branch.",
		"- Use `service_type=frontend_dist` only for static-only UI previews. That mode serves built assets from `internal/interfaces/web/static/dist` but leaves `/api/*` on the shared runtime backend.",
		"- For custom services, deploy `service_type=http` with either an existing `upstream_url` or a `start_command` that boots the service inside the session workspace.",
		"- Keep test-service deployment session-scoped. Reuse the current session's short-hash namespace instead of inventing ad-hoc domains.",
		"- Prefer concise service labels and stable health paths so repeated redeploys land on the same routed host.",
	}, "\n")
}

func resolveBuiltinSkillRepoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	current := wd
	for {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return wd
		}
		current = parent
	}
}

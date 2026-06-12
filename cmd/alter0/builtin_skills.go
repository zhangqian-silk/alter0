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
		privateFileBackedBuiltinSkill(
			"memory-maintenance",
			"Memory Maintenance",
			780,
			"Consolidate daily memory into long-term memory and keep durable memory files concise, stable, and non-duplicative.",
			memoryMaintenanceSkillGuide(),
			filepath.ToSlash(filepath.Join("docs", "skills", "memory-maintenance", "SKILL.md")),
		),
		{
			ID:      "preview-publish",
			Name:    "Preview Publish",
			Enabled: true,
			Scope:   controldomain.CapabilityScopeGlobal,
			Metadata: map[string]string{
				builtinSkillPriorityKey:    "760",
				builtinSkillDescriptionKey: "Session-scoped preview and test-service deployment playbook for the shared alter0 gateway.",
				builtinSkillGuideKey:       previewPublishSkillGuide(),
				builtinSkillFilePathKey:    filepath.ToSlash(filepath.Join("docs", "skills", "preview-publish", "SKILL.md")),
			},
		},
		{
			ID:      "frontend-design",
			Name:    "Frontend Design",
			Enabled: true,
			Scope:   controldomain.CapabilityScopeGlobal,
			Metadata: map[string]string{
				builtinSkillPriorityKey:    "740",
				builtinSkillDescriptionKey: "Distinctive, production-grade frontend design rulebook for pages, components, and interface implementation.",
				builtinSkillGuideKey:       frontendDesignSkillGuide(),
				builtinSkillFilePathKey:    filepath.ToSlash(filepath.Join("docs", "skills", "frontend-design", "SKILL.md")),
			},
		},
		fileBackedBuiltinSkill(
			"doc-coauthoring",
			"Doc Coauthoring",
			700,
			"Document coauthoring guidance for collaborative drafting, editing, and review-ready long-form documents.",
			docCoauthoringSkillGuide(),
			filepath.ToSlash(filepath.Join("docs", "skills", "doc-coauthoring", "SKILL.md")),
		),
		fileBackedBuiltinSkill(
			"fullstack-developer",
			"Fullstack Developer",
			680,
			"Full-stack delivery workflow for coordinated frontend, backend, data, and deployment changes.",
			fullstackDeveloperSkillGuide(),
			filepath.ToSlash(filepath.Join("docs", "skills", "fullstack-developer", "SKILL.md")),
		),
		fileBackedBuiltinSkill(
			"code-reviewer",
			"Code Reviewer",
			660,
			"Structured code review guidance focused on bugs, risks, regressions, and review-ready findings.",
			codeReviewerSkillGuide(),
			filepath.ToSlash(filepath.Join("docs", "skills", "code-reviewer", "SKILL.md")),
		),
		fileBackedBuiltinSkill(
			"webapp-testing",
			"Webapp Testing",
			640,
			"Browser-driven web application testing workflow for reproducible validation, debugging, and regression checks.",
			webappTestingSkillGuide(),
			filepath.ToSlash(filepath.Join("docs", "skills", "webapp-testing", "SKILL.md")),
		),
		fileBackedBuiltinSkill(
			"find-skills",
			"Find Skills",
			620,
			"Skill discovery helper for locating relevant reusable skills when the current catalog is insufficient.",
			findSkillsSkillGuide(),
			filepath.ToSlash(filepath.Join("docs", "skills", "find-skills", "SKILL.md")),
		),
		fileBackedBuiltinSkill(
			"test-driven-development",
			"Test-Driven Development",
			600,
			"Test-driven development workflow that starts from failing tests and drives implementation through the smallest safe increments.",
			testDrivenDevelopmentSkillGuide(),
			filepath.ToSlash(filepath.Join("docs", "skills", "test-driven-development", "SKILL.md")),
		),
		fileBackedBuiltinSkill(
			"ui-ux-pro-max",
			"UI UX Pro Max",
			580,
			"High-fidelity UI and UX design guidance for stronger product interaction, layout, and visual execution.",
			uiUxProMaxSkillGuide(),
			filepath.ToSlash(filepath.Join("docs", "skills", "ui-ux-pro-max", "SKILL.md")),
		),
		fileBackedBuiltinSkill(
			"code-simplifier",
			"Code Simplifier",
			560,
			"Simplifies and refines code for clarity, consistency, and maintainability while preserving all functionality.",
			codeSimplifierSkillGuide(),
			filepath.ToSlash(filepath.Join("docs", "skills", "code-simplifier", "SKILL.md")),
		),
		fileBackedBuiltinSkill(
			"code-review",
			"Code Review",
			540,
			"Pull request review workflow that launches multiple review perspectives and filters findings by confidence.",
			codeReviewSkillGuide(),
			filepath.ToSlash(filepath.Join("docs", "skills", "code-review", "commands", "code-review.md")),
		),
		fileBackedBuiltinSkill(
			"brainstorming",
			"Brainstorming",
			520,
			"Brainstorming workflow for framing ambiguous problems, exploring options, and converging on promising directions.",
			brainstormingSkillGuide(),
			filepath.ToSlash(filepath.Join("docs", "skills", "brainstorming", "SKILL.md")),
		),
		fileBackedBuiltinSkill(
			"travel",
			"Travel",
			500,
			"Travel guide workflow for city itineraries, recommendation pools, route cards, generated map assets, and public HTML delivery.",
			travelSkillGuide(),
			filepath.ToSlash(filepath.Join("docs", "skills", "travel", "SKILL.md")),
		),
	}
}

func fileBackedBuiltinSkill(id string, name string, priority int, description string, guide string, filePath string) controldomain.Skill {
	return controldomain.Skill{
		ID:      id,
		Name:    name,
		Enabled: true,
		Scope:   controldomain.CapabilityScopeGlobal,
		Metadata: map[string]string{
			builtinSkillPriorityKey:    fmt.Sprintf("%d", priority),
			builtinSkillDescriptionKey: description,
			builtinSkillGuideKey:       guide,
			builtinSkillFilePathKey:    filePath,
		},
	}
}

func privateFileBackedBuiltinSkill(id string, name string, priority int, description string, guide string, filePath string) controldomain.Skill {
	skill := fileBackedBuiltinSkill(id, name, priority, description, guide, filePath)
	skill.Metadata["alter0.skill.visibility"] = "private"
	return skill
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
				"- `AGENTS.md` stores repository or workspace operating rules for automation runtimes, including coding conventions, role boundaries, and delivery expectations. Read it when entering a repo and before making code or process decisions. Write only when maintainers update project rules.",
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
		"- Write to `AGENTS.md` only for repository or workspace operating rules, coding conventions, delivery expectations, or runtime workflow rules maintained by the project.",
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

func memoryMaintenanceSkillGuide() string {
	return strings.Join([]string{
		"# alter0 memory maintenance",
		"",
		"- Use `docs/skills/memory-maintenance/SKILL.md` as the canonical maintenance workflow.",
		"- Consolidate daily memory into long-term memory only when a fact, preference, decision, or workflow has become stable.",
		"- Keep daily memory useful for active work and keep long-term memory concise, deduplicated, and directly reusable.",
	}, "\n")
}

func previewPublishSkillGuide() string {
	return strings.Join([]string{
		"# preview publish",
		"",
		"- Use this skill when the user needs a session-scoped preview host, a static artifact preview, or an additional routed test service without changing Nginx.",
		"- For text, image, code, markdown, JSON, screenshots, and standalone HTML artifact previews, stage files in the workspace and publish them with `bash docs/skills/preview-publish/scripts/publish_preview_artifact.sh <session_id> <service_name> --artifact <path> ...`.",
		"- For full applications, routed backends, or frontend build previews, use `deploy_test_service`.",
		"- Registrations land on `/api/control/workspace-services` and are routed by the shared gateway rather than per-service Nginx edits.",
		"- Host routing is fixed at the gateway: `https://<short_hash>.alter0.cn` for the default `web` service, and `https://<service>-<short_hash>.alter0.cn` for additional services such as `api` or `docs`.",
		"- Default `web` deploys should register `service_type=http` and boot the current session backend so the preview host serves both the latest frontend build and `/api/*` from the same branch.",
		"- Use `service_type=frontend_dist` only for static-only UI previews. That mode serves built assets from `internal/interfaces/web/static/dist` but leaves `/api/*` on the shared runtime backend.",
		"- For custom services, deploy `service_type=http` with either an existing `upstream_url` or a `start_command` that boots the service inside the session workspace.",
		"- Keep test-service deployment session-scoped. Reuse the current session's short-hash namespace instead of inventing ad-hoc domains.",
		"- Prefer concise service labels and stable health paths so repeated redeploys land on the same routed host.",
		"- Keep additional services on certificate-safe single-label subdomains under `*.alter0.cn`. Do not generate nested hosts such as `https://<service>.<short_hash>.alter0.cn` or `https://<short_hash>.travel.alter0.cn`.",
		"- Never report server-local artifact links such as `/srv/...`, `.alter0/workspaces/...`, `file://...`, `localhost`, or `127.0.0.1` as user-openable deliverables.",
		"- Do not finish with a local HTML/file path as the primary artifact. Publish it first, then return the deployed `https://*.alter0.cn` URL.",
		"- For public travel guides, deploy `service_name=travel` on `https://travel-<short_hash>.alter0.cn`. Publish only the current session workspace root after the current request's `index.html` has been generated there; do not publish a stale or unrelated page as a fallback.",
	}, "\n")
}

func frontendDesignSkillGuide() string {
	return strings.Join([]string{
		"# frontend design",
		"",
		"- Use this skill when the task is to build or revise a web page, component, application, or other frontend interface.",
		"- Commit to a BOLD aesthetic direction before coding and keep one memorable visual idea coherent across typography, color, motion, composition, and surface treatment.",
		"- Avoid generic fonts like Arial and Inter, cookie-cutter component layouts, and cliched purple-on-white AI styling.",
		"- The canonical file-backed skill lives at `docs/skills/frontend-design/SKILL.md`; treat it as the reusable design rulebook for production-grade frontend implementation.",
	}, "\n")
}

func docCoauthoringSkillGuide() string {
	return strings.Join([]string{
		"# doc coauthoring",
		"",
		"- Use this skill when the user needs collaborative drafting, restructuring, editing, or review passes for specs, proposals, READMEs, and other long-form documents.",
		"- The canonical file-backed skill lives at `docs/skills/doc-coauthoring/SKILL.md`; read it for the full coauthoring workflow, editing passes, and review checkpoints.",
	}, "\n")
}

func fullstackDeveloperSkillGuide() string {
	return strings.Join([]string{
		"# fullstack developer",
		"",
		"- Use this skill when the task spans frontend, backend, data, and delivery concerns and needs one coherent end-to-end implementation workflow.",
		"- The canonical file-backed skill lives at `docs/skills/fullstack-developer/SKILL.md`; read it for the full-stack delivery checklist and execution sequence.",
	}, "\n")
}

func codeReviewerSkillGuide() string {
	return strings.Join([]string{
		"# code reviewer",
		"",
		"- Use this skill when the user asks for a review and the primary goal is to find bugs, regressions, missing safeguards, and concrete engineering risks.",
		"- The canonical file-backed skill lives at `docs/skills/code-reviewer/SKILL.md`; read it for the review framing, severity bar, and reporting structure.",
	}, "\n")
}

func webappTestingSkillGuide() string {
	return strings.Join([]string{
		"# webapp testing",
		"",
		"- Use this skill when the task requires browser-driven validation, reproduction steps, regression checks, or systematic debugging of a web application.",
		"- The canonical file-backed skill lives at `docs/skills/webapp-testing/SKILL.md`; read it for the testing workflow, scripts, and example automation patterns.",
	}, "\n")
}

func findSkillsSkillGuide() string {
	return strings.Join([]string{
		"# find skills",
		"",
		"- Use this skill when the current runtime context is missing a reusable skill and you need to discover a better-fitting external skill or catalog entry first.",
		"- The canonical file-backed skill lives at `docs/skills/find-skills/SKILL.md`; read it for the discovery workflow and selection criteria.",
	}, "\n")
}

func testDrivenDevelopmentSkillGuide() string {
	return strings.Join([]string{
		"# test driven development",
		"",
		"- Use this skill when the task should be driven by failing tests first, followed by the smallest implementation that makes the tests pass.",
		"- The canonical file-backed skill lives at `docs/skills/test-driven-development/SKILL.md`; read it for the red-green-refactor workflow and anti-patterns.",
	}, "\n")
}

func uiUxProMaxSkillGuide() string {
	return strings.Join([]string{
		"# ui ux pro max",
		"",
		"- Use this skill when the task needs stronger product interaction design, higher-fidelity UI direction, or more deliberate UX execution than baseline implementation guidance.",
		"- The canonical file-backed skill lives at `docs/skills/ui-ux-pro-max/SKILL.md`; read it for the full design direction and interface heuristics.",
	}, "\n")
}

func codeSimplifierSkillGuide() string {
	return strings.Join([]string{
		"# code simplifier",
		"",
		"- Use this skill when the task is to simplify or refine recently modified code for clarity, consistency, and maintainability while preserving all functionality.",
		"- This catalog entry originates from a plugin-style package. The canonical file-backed instructions for alter0 live at `docs/skills/code-simplifier/SKILL.md`; plugin metadata remains in `docs/skills/code-simplifier/.claude-plugin/plugin.json`.",
	}, "\n")
}

func codeReviewSkillGuide() string {
	return strings.Join([]string{
		"# code review",
		"",
		"- Use this skill when the task is to review a pull request with a structured workflow that gathers context, launches multiple review perspectives, and filters issues by confidence.",
		"- This catalog entry originates from a plugin-style package. The canonical file-backed instructions for alter0 live at `docs/skills/code-review/commands/code-review.md`; plugin metadata remains in `docs/skills/code-review/.claude-plugin/plugin.json`.",
	}, "\n")
}

func brainstormingSkillGuide() string {
	return strings.Join([]string{
		"# brainstorming",
		"",
		"- Use this skill when the problem is still ambiguous and the immediate need is to frame the space, generate alternatives, compare directions, and converge on a practical plan.",
		"- The canonical file-backed skill lives at `docs/skills/brainstorming/SKILL.md`; read it for the full ideation workflow and companion materials.",
	}, "\n")
}

func travelSkillGuide() string {
	return strings.Join([]string{
		"# travel",
		"",
		"- Use this skill when creating or revising a city travel guide, itinerary, recommendation pool, route-map treatment, or public HTML travel page.",
		"- The canonical file-backed skill lives at `docs/skills/travel/SKILL.md`; read it for the city guide workflow, mobile-first page contract, route-card structure, generated itinerary image requirements, and travel service delivery checks.",
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

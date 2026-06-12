# Memory Maintenance

Use this skill when alter0 starts a scheduled memory maintenance run or when the user explicitly asks to organize cross-session memory.

## Scope

- Maintain durable memory files selected by the runtime memory context.
- Consolidate daily memory into long-term memory after facts, preferences, decisions, workflows, or constraints become stable.
- Keep daily memory focused on active work and recent discoveries.
- Keep long-term memory concise, deduplicated, current, and directly reusable by future sessions.

## Read Order

1. Read the injected memory context summary and writable file list.
2. Read today's daily memory when it exists.
3. Read recent daily memory only when today's file references it or the maintenance request covers a date range.
4. Read long-term memory before editing it.
5. Read `USER.md`, `SOUL.md`, or `AGENTS.md` only when the candidate update targets user preference, hard rule, or repository/workspace operating rule.

## Consolidation Rules

- Promote a note to long-term memory only when it is stable, validated, repeated, or explicitly confirmed.
- Merge duplicate facts instead of appending another equivalent bullet.
- Prefer concrete statements over raw transcript fragments.
- Preserve source meaning while removing temporary wording, uncertainty, and obsolete hypotheses.
- Keep unresolved findings in daily memory until they are verified.
- Do not promote secrets, one-off credentials, private tokens, or sensitive operational details.

## Target File Rules

- `USER.md`: stable user identity, preferences, terminology, response style, and repeated product direction.
- `SOUL.md`: mandatory hard rules that must override ordinary preferences.
- `AGENTS.md`: repository/workspace operating rules, tool discipline, execution boundaries, delivery workflow, and reusable runtime instructions.
- Daily Memory: active work context, unresolved findings, recent decisions awaiting validation, and candidates that need another signal.
- `MEMORY.md`: durable cross-session facts, confirmed decisions, reusable workflows, project invariants, and stable constraints.
- Task summaries: keep raw task outcomes in task storage. Promote only the durable implication when it will help future sessions.

## Summary Contract

Treat daily memory as candidate material, not as final memory.

For each candidate, classify it before editing:

- `promote`: stable user preference, durable project rule, confirmed decision, reusable workflow, environment invariant, or repeated correction.
- `keep_daily`: active task context, unresolved finding, temporary path, one-off request detail, or candidate that needs another signal.
- `drop`: raw transcript echo, duplicated wording, assistant acknowledgment, stale hypothesis, generated boilerplate, secret, credential, token, or private operational detail.

Promoted long-term entries must be short natural-language bullets with enough context to be useful later:

```markdown
- <subject/context>: <stable fact, preference, decision, workflow, or constraint>.
```

Do not paste `user:` / `assistant:` turns, long outputs, logs, stack traces, generated prose, or task implementation details into long-term memory. Summarize the durable implication instead.

When multiple candidates say the same thing, keep one canonical bullet and update it in place. When a newer candidate conflicts with older memory, prefer confirmed explicit user direction; otherwise keep the conflict in daily memory instead of silently overwriting long-term memory.

## Update Format

Long-term memory should use compact Markdown sections:

```markdown
## <domain or topic>

- <stable fact, preference, decision, workflow, or constraint>
```

Daily memory should use dated working notes:

```markdown
## Active Notes

- <recent finding or in-progress context>

## Candidates For Promotion

- <item to revisit after validation>
```

## Output

- Report which files were changed.
- Mention skipped candidates only when they affected the maintenance result.
- Keep the final response short; the updated files are the source of truth.

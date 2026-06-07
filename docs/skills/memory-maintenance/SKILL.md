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
5. Read `USER.md`, `SOUL.md`, or `AGENTS.md` only when the candidate update targets user preference, hard rule, or project operating rule.

## Consolidation Rules

- Promote a note to long-term memory only when it is stable, validated, repeated, or explicitly confirmed.
- Merge duplicate facts instead of appending another equivalent bullet.
- Prefer concrete statements over raw transcript fragments.
- Preserve source meaning while removing temporary wording, uncertainty, and obsolete hypotheses.
- Keep unresolved findings in daily memory until they are verified.
- Do not promote secrets, one-off credentials, private tokens, or sensitive operational details.

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

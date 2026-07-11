# Native Codex Runtime, Memories, and Skills

## Summary

Replace alter0's per-session Codex home, custom per-turn Skill selection, and Markdown-first memory path with Codex's native single-user runtime model:

- one shared active `CODEX_HOME` for authentication, configuration, threads, and local Memories;
- one isolated session workspace for attachments, artifacts, repository checkout, and alter0-owned state;
- the bound repository as the Codex working directory;
- globally installed native Codex Skills with implicit matching and explicit `$skill-name` invocation;
- Codex local Memories enabled for every alter0 Codex task.

The deployment is intentionally single-user and single-account. Multi-user, tenant, and account isolation are not part of this design.

## Goals

- Let Codex reuse local Memories across every alter0 conversation and project.
- Preserve isolated repositories, attachments, artifacts, and Codex threads per conversation.
- Let Codex load a repository's own `AGENTS.md` hierarchy through its native discovery rules.
- Keep alter0 Skill files out of user repositories.
- Remove the Composer's manual Skill picker and let Codex select Skills from their descriptions.
- Keep explicit Skill invocation available through `$skill-name`.
- Preserve global Skill enable/disable management in Settings.
- Remove the scheduled alter0 memory consolidation path that duplicates Codex Memories.
- Make native-memory availability and activity visible without treating Codex's generated files as a user-authored database.

## Non-goals

- Multiple alter0 users, tenants, or Codex accounts.
- Sharing memory between different servers or synchronizing `CODEX_HOME` across hosts.
- Defining a second alter0-owned entity graph, vector database, or Memory Curator.
- Guaranteeing that Codex extracts a specific fact from every turn.
- Replacing repository `AGENTS.md` with alter0-generated project instructions.
- Writing alter0 Skill directories, runtime manifests, or generated instruction files into a bound repository.
- Building per-session Skill allowlists or preserving the existing `skill_ids` API as a compatibility feature.

## Runtime State Model

The active Codex home resolved by `codexapp.ResolveActiveHome` is the only `CODEX_HOME` used by ChatRuntime. alter0 stops creating `<session workspace>/codex-home`, copying `auth.json`, and compiling a per-session `config.toml`.

```text
active CODEX_HOME/
  auth.json
  config.toml
  memories/
  sessions and other Codex state

runtime root/workspaces/chat/sessions/<session-id>/
  repo/                         optional bound repository
  input-attachments/            alter0-owned turn inputs
  artifacts and preview state
  persisted alter0 session state
```

Every Codex process receives the same resolved `CODEX_HOME`. Conversation isolation continues through:

- a distinct alter0 session workspace;
- a distinct bound repository checkout;
- a distinct Codex thread ID;
- distinct attachments, runtime events, and artifacts.

Deleting a conversation removes only its session workspace and alter0 session state. It never removes shared Codex configuration, authentication, Memories, or unrelated Codex threads.

## Native Memories

ChatRuntime invokes Codex with native Memories enabled:

```text
--enable memories
-c memories.generate_memories=true
-c memories.use_memories=true
```

These are invocation-scoped overrides. alter0 does not rewrite the user's base `config.toml` on every turn. A new and a resumed turn receive the same overrides.

At service startup, alter0 checks `codex features list` for the `memories` feature. The result is exposed in Runtime/Memory status:

- `available=true, enabled=true` when the installed CLI accepts native Memories;
- `available=false` with a diagnostic when the CLI is too old or the feature is missing.

Memory unavailability does not corrupt or silently replace Codex memory with the old alter0 store. Chat remains usable, but the UI reports that cross-session memory is unavailable. Invocation errors caused by unsupported memory configuration return a specific runtime compatibility error instead of an ordinary agent failure.

Codex decides when eligible idle tasks become memory inputs, performs extraction and consolidation in its own generated state, and injects relevant Memories into later tasks. alter0 does not run a second model pass per completed Turn.

## Repository Working Directory and Instructions

When a session has a ready repository binding, Codex starts with `<session workspace>/repo` as both `cmd.Dir` and its working root. Without a binding, Codex starts from the session workspace.

This makes Codex discover the repository's checked-in `AGENTS.md` and nested overrides natively. alter0 does not create, prepend, or update `repo/AGENTS.md`.

Stable alter0 runtime requirements are added to the provider-owned Turn prompt, alongside structured repository and attachment context. They cover:

- the session workspace and bound repository boundary;
- server-local link restrictions;
- preview publication requirements;
- the location of session attachments and artifacts;
- the fact that service-enforced permissions cannot be expanded by repository instructions.

Hard filesystem, credential, network, and publication boundaries remain service or sandbox responsibilities. Prompt instructions are not treated as the enforcement layer.

Attachment paths are calculated relative to the actual Codex working directory. A bound repository therefore receives paths such as `../input-attachments/<turn-id>/<file>`, while an unbound session keeps workspace-relative paths. Image arguments continue to use canonical filesystem paths.

The previous workspace-root generated `AGENTS.md` managed block is removed from ChatRuntime preparation. Existing repository files are never modified as a side effect of starting a Turn.

## Native Skill Model

### Global installation

Enabled public file-backed alter0 Skills are reconciled into Codex's documented user Skill location under the single server user's home directory:

```text
$HOME/.agents/skills/<managed-alter0-skill>/SKILL.md
```

The reconciler owns only destinations carrying alter0 management metadata and never deletes unrelated user Skills. It uses staging plus atomic rename so Codex never observes a partially copied Skill directory. Skill scripts, references, assets, and executable modes are preserved.

The Skill source remains `docs/skills/<skill-id>/`. Every native Skill must have a standard `SKILL.md` with `name` and `description` frontmatter. Existing nonstandard file-backed entries are normalized to that layout as part of this change.

Reconciliation runs:

- during service startup after built-in Skill registration;
- after a global Skill is enabled, disabled, created, updated, or deleted.

Disabled, private, missing, or invalid Skills are not installed. `memory-maintenance` remains private during migration and is then removed with the old maintenance path.

### Invocation

Codex receives its normal native Skill catalog and uses progressive disclosure:

- implicit invocation when the task matches a Skill description;
- explicit invocation when the user includes `$skill-name`;
- full `SKILL.md` loading only after Codex selects the Skill.

Skills with significant external side effects must set native metadata that disables implicit invocation. Mandatory runtime restrictions are not Skills.

### Removed selection path

The following behavior is removed rather than deprecated:

- Composer Skill multi-select UI and local `activeSkillIDs` state;
- `skill_ids` in Chat input requests and session payloads;
- `resolveChatRuntimeSkillContext`;
- Turn-level `SkillContext` persistence and retry snapshots;
- per-Turn copying to `.alter0/codex-runtime/skills/`;
- generated `.alter0/codex-runtime/skills.md`;
- the generated `AGENTS.md` instruction to read that manifest.

Settings > Skills remains the global lifecycle surface. Existing persisted Turn records containing `skill_context` continue to decode, but new records do not write the field. No migration rewrites historical session files.

## Legacy Memory Retirement

The primary Chat path stops exposing two competing memory systems.

- Remove the public guide-only `memory` Skill; native Codex Memories are runtime behavior, not an optional workflow.
- Remove the `system-memory-maintenance` Scheduler job and its private `memory-maintenance` Skill registration.
- Keep `system-session-cleanup`; it is unrelated to memory consolidation.
- Stop injecting alter0 Markdown `MemoryContext` into Codex execution.
- Stop writing or recalling orchestration `LongTermMemory` for Codex Agent routes.
- Preserve repository `AGENTS.md` and other explicit checked-in documentation as project instructions, not memory.

The existing Markdown data files are not deleted automatically. They may contain user-authored data and remain available for manual migration or archival. The implementation removes the legacy resolvers, stores, handlers, and tests after their callers move to native Codex state; no dormant compatibility path may continue writing, recalling, or maintaining the old memory model.

## Memory UI

Settings > Memory becomes a native Codex memory status surface instead of an editor-like view of alter0 Markdown files.

It displays:

- native Memories feature availability;
- whether alter0 invokes Codex with memory generation and use enabled;
- shared memory directory existence;
- last observed modification time and generated file count;
- a clear note that files are Codex-generated state and should not contain secrets.

The first version does not parse undocumented Codex memory schemas, edit generated entries, or promise stable filenames. It may list sanitized relative filenames and timestamps for diagnostics, but must not expose `CODEX_HOME`, server-local absolute paths, auth state, or raw secret-bearing evidence through the Web API.

## Data Flow

For a bound Chat Turn:

1. Resolve the single active Codex home.
2. Resolve the session workspace and ready repository checkout.
3. Prepare attachments in the session workspace.
4. Build an alter0 runtime context block with safe paths relative to the repository working directory.
5. Start or resume Codex from `repo/` with the shared `CODEX_HOME` and native Memories enabled.
6. Codex loads global native Skills and the repository's own `AGENTS.md` chain.
7. Codex selects relevant Skills implicitly or through an explicit `$skill-name` mention.
8. Codex emits normal JSONL runtime events; alter0 persists the Turn and thread ID as today.
9. Codex independently decides when the idle task is eligible for background memory extraction and consolidation.

An unbound Turn follows the same flow but uses the session workspace as its working directory.

## Error Handling

- **Active Home unavailable:** fail the Turn before starting Codex with `codex_home_unavailable`; do not create a fallback per-session Home.
- **Native Memories unsupported:** report `codex_memories_unavailable` in runtime status and Turn diagnostics; do not activate legacy alter0 memory as a fallback.
- **Skill source invalid:** keep the last valid installed copy, report the reconciliation error in Settings, and continue ordinary Chat without the broken Skill.
- **Skill name collision:** alter0 owns only its managed destination. A conflicting unmanaged user Skill is preserved and the alter0 Skill reports a collision instead of overwriting it.
- **Repository unavailable:** retain the existing preparation and retry behavior; Codex does not start until the binding is ready.
- **Attachment path failure:** fail before Codex starts and preserve the existing retryable Turn behavior.
- **Shared Home concurrency:** alter0 does not rewrite shared `auth.json` or base `config.toml` per Turn. Codex remains the owner of its concurrent thread and memory state.

## Security

- Single-user deployment is an explicit product constraint, not an inferred tenancy boundary.
- The browser never receives `CODEX_HOME`, memory absolute paths, auth files, or raw Codex configuration.
- Skill reconciliation accepts only configured file-backed Skill roots and prevents path traversal outside the resolved source directory.
- Repository checkout remains isolated per conversation.
- A repository cannot expand service permissions through `AGENTS.md` or Skill instructions.
- Memory and Skill generated state are never committed to the bound repository.

## Testing Strategy

### ChatRuntime application tests

- New and resumed Turns use the same resolved active `CODEX_HOME`.
- No session `codex-home`, generated Skill manifest, or generated workspace `AGENTS.md` is created.
- A ready repository becomes `cmd.Dir`; an unbound session keeps the session workspace.
- Bound attachment prompt paths are correct relative to `repo/`.
- New and resumed commands enable native Memories.
- Repository `AGENTS.md` remains byte-for-byte unchanged.
- Session deletion does not delete shared Codex state.

### Skill reconciliation tests

- Enabled public native Skills install atomically with their complete directory.
- Disabled/private/invalid Skills are absent.
- Unmanaged user Skills are preserved.
- Lifecycle changes reconcile the managed installation.
- Nonstandard built-in Skill sources are rejected until converted to `SKILL.md`.

### Web and frontend tests

- Chat requests no longer send `skill_ids`.
- Composer no longer renders a Skill multi-select.
- Settings > Skills continues global lifecycle management and displays reconciliation failures.
- Settings > Memory displays native availability and sanitized activity metadata.
- Schedules no longer lists Memory Maintenance and still lists Session Cleanup.

### Regression tests

- Repository preparation and retry remain idempotent.
- Existing persisted Turns with legacy `skill_context` still load.
- Attachments and image inputs work for bound and unbound sessions.
- Codex login, model selection, runtime restart, session resume, preview publication, and cleanup continue to work.
- Run focused Go, frontend component, and Playwright tests, followed by `go test ./...` and the production frontend build.

## Acceptance Criteria

- All Chat conversations on the server use the same active Codex Home.
- Codex Memories generated from one eligible conversation can be recalled in another conversation.
- A bound repository is the Codex working directory and its `AGENTS.md` hierarchy loads natively.
- Starting a Turn never writes alter0 runtime files into the bound repository.
- Users no longer choose Skills through the Composer.
- Enabled global Skills are available to Codex for implicit and explicit invocation.
- `$skill-name` can force an installed Skill.
- Disabled Skills are unavailable to new Codex processes.
- The old scheduled memory-maintenance path no longer runs.
- Memory status is observable without exposing server-local paths or relying on undocumented generated-file schemas.

# Coding Git Delivery Skill

## Purpose

- This private skill is the coding agent's reusable Git delivery playbook.
- Use it whenever the task requires pulling a repository, preparing a coding workspace, creating a branch, making a signed commit, pushing changes, opening a PR, or merging through GitHub.
- Keep task-specific implementation details out of this skill. Store only durable Git workflow rules here.

## Delivery Loop

- First understand the user's real engineering goal, acceptance boundary, and whether the task also includes preview deployment or GitHub delivery.
- Drive the implementation through repeated `codex_exec` calls until the requested result is actually achieved.
- Do not stop at "code changed". Continue until validation is complete, the preview page is deployed successfully when needed, and any requested Git delivery step is finished.
- After preview deployment succeeds, ask the user whether they want to perform manual testing before continuing with PR or merge delivery.

## Environment Parity Rule

- Keep the test repository configuration aligned with the production service for execution-critical settings.
- Model provider, Codex executable path, agent path, and similar runtime wiring must match production.
- Only session-scoped cache, session history, or other session-local runtime state may differ from production defaults.
- Do not treat a preview deployment as valid if it depends on ad-hoc config drift that would not exist in production.

## Workspace Rule

- For repository work, operate in the session repository clone at `.alter0/workspaces/sessions/<session_id>/repo`.
- Treat that path as a full clone with its own `.git` metadata.
- Do not rely on `git worktree` for coding delivery.
- Keep the source repository path as the upstream reference; do not edit or commit directly in the source repository when the session clone is available.

## Clone Rule

- If the session repository clone is missing, create a full clone before coding.
- Prefer cloning from the current source repository state so the session clone starts from the same codebase snapshot.
- After cloning from a local source repository, make sure `origin` points to the real remote repository URL when one is available.
- Reuse the existing session clone when it is already a valid Git repository.

## Commit Rule

- All commits must be authenticated and signed. Do not disable signing to bypass missing credentials.
- Before creating a commit, confirm the repository is on the intended working branch and only the intended changes are staged.
- After committing, verify the signature with `git log --show-signature -1`.
- Use Conventional Commits with the format `<type>(<scope>): <description>`.

## GitHub Rule

- Use `gh auth status` to confirm the GitHub CLI session before PR operations.
- Push the working branch with upstream tracking before creating a PR.
- Use `gh pr create` for PR creation and `gh pr merge` for merge execution when the user asks to open or merge a PR.
- After merge, report the PR URL, merge state, commit signature status, and validation commands that were run.
- When the task includes full delivery, move directly from successful preview deployment and any user-requested manual testing to signed commit, push, `gh pr create`, and `gh pr merge` without unnecessary pauses.

## Instruction Rule

- Rely on structured runtime context for stable facts such as workspace paths, repository identity, branch context, and preview URL.
- Use each `codex_exec` instruction for the incremental action of the current turn rather than repeating the full Git rulebook.
- Pass all necessary delivery information for the current step, but do not add unrelated repository history or unused workflow detail.

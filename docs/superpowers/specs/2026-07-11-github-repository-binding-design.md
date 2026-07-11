# Lightweight GitHub Repository Binding

## Summary

Add an optional GitHub repository selector to the chat composer. Selecting a repository associates the first user message and the resulting chat session with that repository. The server clones the repository into the session workspace before starting the agent, then supplies trusted repository context on every turn.

The feature is a convenience shortcut, not a permission or source-control workflow. It removes the need for the user to name the repository repeatedly or wait for the agent to discover and clone it.

## Goals

- Let the user select one accessible GitHub repository before sending the first message.
- Keep the New action and draft creation as lightweight as they are today.
- Clone the selected repository only after the user sends a message.
- Give the agent the repository name, branch, commit, and exact relative path as structured runtime context.
- Reuse the same checkout and binding for every later turn in the session.
- Use the server's personal GitHub credentials without exposing them to the browser.
- Preserve the existing chat and attachment behavior when no repository is selected.

## Non-goals

- GitHub App installation or per-user OAuth flows.
- Multiple repositories in one session.
- Branch selection, automatic pull, repository switching, or worktree management.
- Automatic commit, push, pull request creation, or source-control authorization rules.
- Changing the git author or committer identity configured for the agent runtime.
- Treating the binding as proof that every message must modify the repository.

## User Experience

### Composer entry point

The composer action row gains a dedicated GitHub icon next to the existing attachment button. It is a separate control and must not reuse the paperclip, attachment menu, attachment state, or attachment copy.

The GitHub button is available only while the conversation is an unsent draft. Activating it opens:

- a popover on desktop;
- a bottom sheet on mobile.

The selector shows recently updated accessible repositories and supports search. The first version is single-select. Choosing a repository closes the selector and adds a chip above the input:

`[GitHub icon] owner/repository  ×`

Removing the chip before send clears the draft selection. Merely opening the selector or choosing a repository must not create a server session, clone a repository, or start an agent. The selection lives in client-side draft state and follows the existing draft lifecycle.

### Sending and session state

When the user sends the first message:

1. The message appears immediately using the existing optimistic behavior.
2. The selected repository is submitted as structured data with the message.
3. The repository becomes the immutable binding for that session.
4. The chip remains visible as a session-level repository label without a remove action.
5. The conversation shows a lightweight `Preparing owner/repository…` process state.
6. The agent starts only after the checkout is ready.

If the first message is sent without a repository, the session remains unbound and behaves exactly as it does today. Binding is not added later to an active session; the user creates a new chat to target another repository.

## Architecture

The feature is divided into four independently testable units.

### Repository catalog

`GitHubRepositoryCatalog` lists and searches repositories available to the server's configured personal GitHub identity. It returns sanitized metadata only:

- stable repository ID;
- full name;
- visibility;
- default branch;
- updated timestamp.

The implementation uses the server's existing personal `gh`/Git credential source. Authentication material never enters API responses, frontend state, logs, clone URLs, or persisted session summaries. Catalog failures affect only the selector; ordinary chat remains available.

### Draft selection

The frontend owns an optional `RepositoryRef` in the unsent conversation draft. This state is independent from attachments and does not require a backend session. Once the first input succeeds, the frontend replaces the removable draft chip with the binding returned by the session API.

### Session repository binding

The chat runtime stores an optional immutable binding alongside the persisted session. The binding contains:

- provider (`github`);
- repository ID and full name;
- default branch and checked-out branch;
- preparation state (`preparing`, `ready`, or `failed`);
- workspace-relative path (`repo`);
- resolved HEAD SHA when ready;
- a sanitized failure code and message when failed.

The public session representation omits credentials, clone URLs, remote URLs, and absolute server paths.

### Repository workspace preparer

`RepositoryWorkspacePreparer` resolves the submitted repository ID through the trusted catalog, clones its default branch, and updates the binding state. It clones into a server-owned staging directory and atomically renames a successful checkout to `<session workspace>/repo`. Failed partial clones are removed from the staging directory, so `repo/` never represents a half-created checkout.

The preparer uses the configured personal GitHub credential helper and never embeds a token in a command argument or remote URL. A ready binding is reused as-is. Follow-up turns do not fetch, pull, reset, or clean the checkout because the agent may have uncommitted work.

## API Contracts

### Repository catalog

`GET /api/chat/repositories?query=<text>&cursor=<cursor>` returns a paginated result:

```json
{
  "repositories": [
    {
      "id": "123456789",
      "full_name": "owner/repository",
      "private": true,
      "default_branch": "main",
      "updated_at": "2026-07-11T10:00:00Z"
    }
  ],
  "next_cursor": "opaque-or-empty"
}
```

Repository IDs are strings at the API boundary to avoid JavaScript integer precision issues.

### First input

The existing input request gains an optional repository reference:

```json
{
  "input": "Update the retry behavior",
  "attachments": [],
  "skill_ids": [],
  "repository": {
    "provider": "github",
    "id": "123456789",
    "full_name": "owner/repository"
  }
}
```

The server treats the ID as the lookup key and `full_name` as a display hint. It re-resolves the repository and persists trusted metadata. On later inputs:

- omitting `repository` reuses the session binding;
- sending the same repository ID is idempotent;
- sending a different repository ID returns `409 repository_binding_conflict`.

### Retry

`POST /api/chat/sessions/{session_id}/repository/retry` retries only repository preparation for the already persisted first turn. It must not append another user message. On success, the pending turn continues into normal agent execution.

## Turn Data Flow

For a bound first message, the runtime performs these steps:

1. Create the backend session if the frontend draft is not yet server-backed.
2. Resolve and validate the repository through the trusted catalog.
3. Persist the original user message and trusted repository binding.
4. Mark the session busy and the repository binding `preparing`.
5. Clone the default branch into the staging directory and atomically install it as `repo/`.
6. Record the checked-out branch and HEAD SHA, then mark the binding `ready`.
7. Build provider-owned repository context and start the agent from the session workspace root.

The displayed and persisted user message remains unchanged. The provider adds a separate runtime context block to the agent input on every bound turn:

```text
Repository context:
- repository: owner/repository
- path: repo/
- branch: main
- head: <sha>

This user message is associated with the repository above. Treat it as the
default code target when the request relates to repository work.
```

The agent stays rooted at the session workspace so existing runtime files and attachments continue to work. The exact `repo/` path prevents repository discovery or ambiguous directory searches.

## Error Handling and Recovery

- **Catalog unavailable or unauthenticated:** the selector shows a retryable error. The composer and attachment flow remain usable.
- **Repository inaccessible at send time:** reject the input before message persistence, roll the optimistic message back into the composer, and keep the repository selection available for correction.
- **Clone authentication, network, or disk failure:** persist the user message, set the binding to `failed`, do not start the agent, and show a retry action.
- **Retry:** rerun preparation against the same trusted repository ID without duplicating the message. A successful retry resumes the pending turn.
- **Service interruption:** an interrupted preparation becomes retryable rather than being reported as ready. A valid existing `repo/.git` checkout is reused only when its repository identity matches the binding.
- **Binding conflict:** reject attempts to change repositories after the first message and direct the user to create a new conversation.

Errors exposed to the frontend are sanitized. Command output that could contain credential-helper details is not returned verbatim.

## Security and Identity

- This personal deployment uses one server-managed GitHub identity; it does not introduce GitHub App installation tokens or browser-side OAuth.
- The repository catalog is the authority for accessibility. The browser cannot supply arbitrary clone URLs or local paths.
- Git credentials are provided through the server credential helper, never persisted with the session.
- Existing git author, committer, and signing configuration remains authoritative. Binding a repository does not change commit attribution and does not authorize automatic push.
- The repository relative path is a fixed server value (`repo`), preventing path traversal.

## Testing Strategy

### Frontend tests

- The GitHub icon is distinct from the attachment button and opens the correct desktop/mobile selector.
- Opening and selecting from the repository picker does not create a session.
- Search, selection, removal, loading, empty, and retry states render correctly.
- A selected draft renders a removable chip; a persisted binding renders a non-removable label.
- The first input includes the repository reference once.
- Unbound conversations retain current behavior.
- A failed checkout exposes retry without duplicating the displayed user message.

### Backend unit tests

- Catalog responses are sanitized and repository IDs are handled as strings.
- The server resolves metadata by ID and ignores untrusted display metadata.
- Binding state transitions are valid and a different repository is rejected.
- Prompt construction preserves the original user text and adds repository context separately.
- Retry is idempotent and never appends a duplicate input event.
- Ready repositories are reused without fetch, reset, or pull.

### Integration tests

- Clone a local fixture repository through a fake provider into `repo/`, capture branch and HEAD, and start the fake agent only after readiness.
- Simulate clone failure and successful retry of the same pending turn.
- Verify partial staging directories never appear as a ready checkout.
- Verify session reload preserves sanitized repository status and reuses a ready checkout.
- Run the existing chat runtime, attachment, frontend, and Go regression suites.

## Acceptance Criteria

- Clicking New remains free of repository API writes, session creation caused by this feature, clone work, and agent startup.
- The composer has a dedicated GitHub icon independent of the upload control.
- Selecting a repository is no more than opening the picker and choosing one row.
- The first send binds and prepares the repository; later sends reuse it automatically.
- The agent receives the trusted repository identity and `repo/` path without the user repeating them.
- Checkout failures are visible and retryable without message duplication.
- No token, clone URL containing credentials, or absolute workspace path is exposed to the frontend.
- No automatic pull, commit, push, or repository switching is introduced.

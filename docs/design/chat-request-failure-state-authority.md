# Chat Request Failure and Session State Authority

Date: 2026-07-12

Status: Implemented

## Objective

Chat session state and history must come from backend session data. A browser request failure is a transport outcome, not evidence that the session or turn failed.

The frontend must distinguish three independent facts:

1. A request is currently being submitted by this browser.
2. The request failed to reach or receive a response from the backend.
3. The backend reports a session or turn business state such as `busy`, `ready`, `failed`, or `interrupted`.

Only the third fact can change persisted session state, session badges, timeline history, polling behavior, or Composer locking.

## State Model

### Backend session state

The following sources are authoritative:

- successful input responses containing a session snapshot;
- session collection and detail responses;
- typed session, turn, and runtime update events.

The frontend may merge these sources according to freshness and payload scope, but it must not synthesize a session failure from a request exception or an HTTP error alone.

### Browser submission state

The Composer owns a transient submission state while upload or input requests are in flight. This state may temporarily prevent duplicate submission in the same Composer, but it does not:

- change `session.status`;
- add a busy or failed message to session history;
- change the session list or workspace status indicator;
- enter the persisted session snapshot;
- start session recovery polling.

The draft and attachments remain intact until the backend acknowledges the input with an authoritative response.

### Request notices

Request failures produce a transient, non-blocking Composer notice. The notice is local UI state, is not part of the conversation timeline, and is not persisted.

Transport failures use a weak neutral message such as `Network request interrupted. Try again.` Backend HTTP errors use the backend's safe actionable message when available. Both restore the Composer immediately and retain the draft and attachments.

Authentication remains a global application concern. Existing authentication handling may redirect or request login, but it must not mutate the active Chat session.

## Request Flow

1. The user submits a non-empty draft or attachments.
2. The Composer enters local `submitting` state. The active session remains unchanged.
3. Attachments and input are submitted without adding optimistic session messages or statuses.
4. On an acknowledged input response, the frontend applies the returned session snapshot. The backend-provided state may then set the session to `busy`, add the accepted user turn, and activate normal updates polling.
5. The draft and attachments are cleared only after acknowledgment.
6. On a transport failure, the Composer exits `submitting`, retains the draft and attachments, and shows a weak transient notice.
7. On a structured backend error, the Composer exits `submitting`, retains the draft and attachments, and shows the backend error as a request notice. The error does not become session state unless an authoritative session payload or a subsequent authoritative read reports that state.

The frontend must not automatically retry the input POST because the backend may have accepted the request before the response was lost.

## Silent Reconciliation

After an ambiguous transport failure, the frontend performs a silent bounded detail refresh for the active session:

- if the backend reports an accepted running or completed turn, normal authoritative merging displays it;
- if the backend still reports the prior state, the session remains unchanged;
- if reconciliation also fails, no additional notice or session mutation occurs.

Reconciliation is read-only and must not recreate the input request. It may retry detail reads with the existing bounded recovery policy, but it must not keep the Composer locked while those reads run.

## Error Boundaries

| Condition | UI feedback | Session mutation | Draft |
|---|---|---|---|
| Fetch rejection, timeout, abort, connection loss | Weak transient network notice | None | Retained |
| Structured 4xx validation or conflict response | Transient actionable backend notice | None unless authoritative session data is present | Retained |
| Structured 5xx response | Weak backend-unavailable notice | None unless authoritative session data is present | Retained |
| Successful response with `session.status=busy` | Normal running UI | Apply backend session | Cleared |
| Detail/update reports failed turn or session | Normal backend failure presentation | Apply backend session | Unchanged |
| Authentication failure | Existing global auth handling | None | Retained where current storage permits |

## Frontend Changes

- Replace the pre-request `session.status = local_running` mutation with Composer-local submission state.
- Stop appending request exceptions such as `Load failed` as assistant messages.
- Stop setting `session.status = failed` in the input request catch path.
- Apply session state only from normalized backend payloads.
- Keep request notices outside `ChatSession.messages` and all session cache serializers.
- Ensure Composer disabled state is the union of backend-authoritative busy state and the current Composer's short-lived `submitting` flag, without exposing submission as session status.
- Preserve the current draft and attachments until an acknowledged response clears them.

## Verification

Frontend regression tests must prove:

1. A rejected input fetch produces only a transient request notice; session status, history, cache, draft, and attachments remain unchanged.
2. A delayed fetch rejection after the backend completed does not append `Load failed`, mark the session failed, or leave it busy; a silent detail read restores the backend turn.
3. A structured backend error presents its message without changing session state.
4. A successful input response with backend `busy` state locks the Composer and starts normal recovery polling.
5. A backend detail or update containing a failed turn remains authoritative and may render the existing backend failure state.
6. Refreshing from a stale browser cache cannot override a newer authoritative backend state with a request-level failure.

The implementation must run the focused Conversation runtime tests, frontend test suite, frontend build, and relevant Go tests for the Chat Web interface. README, requirements, requirements detail, and technical solution documentation must be updated in the implementation commit to reflect the final behavior.

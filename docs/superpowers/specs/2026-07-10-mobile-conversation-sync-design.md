# Mobile Conversation Sync Design

## Status

- Date: 2026-07-10
- Scope: Web Chat session list, active conversation detail, incremental runtime updates, history pagination, frontend merge rules, and browser persistence
- Decision: Approved design direction; implementation has not started

## 1. First-principles priorities

The design follows this order of priorities:

1. The user must always see the newest accepted conversation state. New server data must never later revert to an older browser snapshot.
2. Opening, switching, refreshing, backgrounding, and foregrounding Chat must preserve a continuous reading experience. Cached content stays visible while network requests run.
3. Long-running Agent tasks must recover reliably after mobile browsers suspend timers or move the page into the background.
4. Network request count, response size, browser storage size, and server work are optimized only after the correctness and continuity requirements are satisfied.

The primary environment is a personal, single-device mobile browser. Cross-device real-time consistency is not a goal. The user can use manual refresh when they explicitly suspect that displayed data is stale.

## 2. Problem statement

The current frontend keeps session data in several mutable locations: React state, multiple refs, a module-level cache, and localStorage snapshots. These sources are repeatedly merged into one another after startup.

The current merge path also merges messages before it finishes deciding whether the incoming session is older. When an older payload is detected, metadata such as title and status is protected, but messages that were already merged are not fully restored. An older state effect or cache snapshot can therefore run after a successful detail refresh and replace a newer turn with an older copy.

The target design must enforce these invariants:

- `updated_at` never moves backward in live conversation state.
- `last_applied_update_id` never moves backward.
- A summary payload can never modify turns or runtime events.
- A history payload can never modify the newest known turn.
- Browser persistence can never overwrite an already-loaded live state.
- An async response may only update the session for which it was requested.

## 3. Non-goals

This design does not introduce:

- SSE or WebSocket delivery;
- cross-device live synchronization;
- ETag-based session list requests;
- incremental or cursor-based session list loading;
- automatic loading of complete conversation history;
- background polling while the page is hidden;
- a server-side session revision field.

## 4. Canonical frontend state

React reducer state is the only mutable runtime source of truth. Refs may expose the latest reducer snapshot to async callbacks, but refs cannot independently merge, persist, or publish session state.

The canonical state is turn-based rather than rendered-message-based:

```text
ConversationState
  activeSessionID
  sessionsByID
    SessionState
      summary
      contentUpdatedAt
      turnsByID
      turnOrder
      turnsPaging
      optimisticInputs
      contentLoaded
      lastDetailFetchedAt
  sessionOrder
  lastAppliedUpdateID
  listRequestGeneration
  detailRequestGenerationBySession
```

Each turn owns its prompt, attachments, status, timestamps, final output, and runtime events. Runtime events are stored by canonical event ID with a stable order. User and assistant timeline messages are derived view models; they are never merged back into canonical state and are not persisted as a second business representation.

The reducer supports explicit actions:

- `hydrateCache`: initialize empty reducer state once from browser persistence;
- `replaceSummaries`: apply a successful full lightweight session list;
- `applyLatestDetail`: apply the active session's latest bounded turn page;
- `applyUpdate`: apply one ordered runtime update;
- `prependHistory`: add a page of older turns;
- `appendOptimisticInput`: show a locally submitted input immediately;
- `confirmOptimisticInput`: bind an optimistic input to its canonical server turn;
- `removeSession`: remove summary, content, optimistic input, and persistence for a session;
- `setActiveSession`: switch the visible session without mutating its content.

## 5. Server timestamp contract

`updated_at` is the content freshness clock. It advances for:

- accepted user input;
- turn creation or status changes;
- runtime event append or update;
- assistant output;
- completion, failure, cancellation, or interruption.

It does not advance for title, pinning, or other list-only configuration changes.

Because no session revision is introduced, content timestamps must be strictly monotonic even when several mutations occur in the same millisecond or the system clock moves backward:

```text
next_updated_at = max(current_wall_clock_ms, previous_updated_at_ms + 1)
```

The server persists the resulting timestamp. All Chat list, detail, and update payloads expose it as Unix milliseconds.

Title and pin state are authoritative in a successful full session list response and in the direct mutation response that changed them. Their freshness is controlled by request generation rather than `updated_at`.

## 6. API contracts

### 6.1 Lightweight full session list

`GET /api/chat/sessions` returns only fields required by the session rail and detail-fetch decision:

```json
{
  "items": [
    {
      "id": "c_x8k4p9m2q7vd3n6a",
      "title": "Review conversation sync",
      "status": "ready",
      "created_at": 1783650000000,
      "updated_at": 1783650300000,
      "pinned": false
    }
  ]
}
```

The list never returns turns, messages, runtime events, attachments, event detail blocks, model configuration, Skill selection, MCP selection, runtime paths, or provider metadata.

A successful list response is authoritative for server-backed membership. Missing server-backed sessions are removed. Browser-only drafts remain in the separate draft store. A failed or malformed list response must not delete any existing session.

The list updates summary fields only. It never modifies content, even when its `updated_at` is newer than the loaded content.

### 6.2 Active latest detail

`GET /api/chat/sessions/{session_id}` returns the session summary, the latest bounded turns page, and paging metadata. The default page remains 20 turns with an approximate maximum response size of 1 MiB.

The application service produces summary, `updated_at`, turns, and paging boundaries from one atomic session read snapshot. The Web layer must not assemble a detail response from separate `Get` and `ListTurns` reads that can observe different mutations.

### 6.3 Incremental updates

`POST /api/chat/sessions/updates` remains the visible-page incremental transport. Every public update has a stable envelope:

```json
{
  "update_id": 481,
  "type": "turn.completed",
  "session_id": "c_x8k4p9m2q7vd3n6a",
  "turn_id": "turn-8",
  "created_at": 1783650300000,
  "payload": {
    "session": {},
    "turn": {},
    "runtime_event": {}
  }
}
```

`update_id` is the ordered owner-level cursor. Duplicate or lower IDs are ignored. `has_more=true` causes an immediate bounded continuation request while the page remains visible. `resync_required=true` causes one active detail refresh.

`session.deleted` removes the canonical session and both persistence layers immediately. Runtime update filtering may omit unselected process events while still advancing the cursor. When a user explicitly enables a previously hidden process category, the frontend refreshes the active latest detail once to recover omitted event summaries.

Periodic detail fallback after a fixed number of empty update responses is removed. Empty responses use polling backoff; detail is fetched for explicit resync, an incomplete terminal payload, active-session switching rules, foreground calibration, or manual refresh.

### 6.4 Canonical IDs

Session, turn, and event IDs use the same original string form in every endpoint:

```text
c_x8k4p9m2q7vd3n6a
turn-8
event-15
```

The API must not return numeric turn IDs in turns while returning `turn-*` strings in paging cursors. List, detail, updates, history, acknowledgement manifests, and event-detail routes all use the canonical string IDs.

### 6.5 Optimistic input identity

The input request includes a client-generated request ID:

```json
{
  "client_request_id": "req_q7vd3n6ax8k4p9m2",
  "input": "Continue",
  "attachments": []
}
```

The input response and subsequent turn updates carry the same `client_request_id`. The frontend confirms the matching optimistic input by ID. It never reconciles optimistic inputs by text, because consecutive identical prompts are valid.

### 6.6 HTTP caching

Chat list, detail, history, update, mutation, and event-detail responses use:

```http
Cache-Control: no-store
```

Application-managed browser persistence is the only client cache for conversation data.

## 7. Merge and ordering rules

### 7.1 Full list

The newest list request generation is the only generation allowed to commit. A newer list request invalidates older in-flight list responses. Pin and title mutations also invalidate list requests started before the mutation.

On commit, the list:

- replaces server-backed membership;
- updates title, pin, status, created time, and content `updated_at`;
- preserves all canonical turns and optimistic inputs;
- marks active content as stale when the summary `updated_at` is newer than the loaded content timestamp;
- does not fetch non-active content.

### 7.2 Latest detail

A detail request records:

- its per-session request generation;
- the target session ID;
- `lastAppliedUpdateID` at request start.

The response is rejected when:

- a newer detail generation exists for that session;
- the target session has been deleted;
- `incoming.updated_at < current.contentUpdatedAt`;
- updates advanced after request start and the detail is not newer than current content.

An accepted latest detail replaces the canonical representation of turn IDs contained in the returned latest window. It preserves already-loaded turns strictly older than the returned oldest boundary. It reconciles optimistic inputs through `client_request_id` and derives timeline messages from the resulting turns.

### 7.3 Runtime updates

Updates are processed in ascending `update_id` order. An update with `update_id <= lastAppliedUpdateID` is ignored. A content patch whose `updated_at` is older than the current content timestamp is ignored before any turn or event merge occurs.

Summary fields and content fields are applied separately. A title or pin patch cannot touch turns. A turn patch only modifies its canonical turn. A runtime event patch only modifies its canonical event within that turn.

### 7.4 History pages

History responses only insert previously unknown turns before the current oldest turn. They do not update session summary fields, content freshness, the latest paging boundary, existing turn IDs, or runtime events already known from the latest detail or updates.

An invalid or unchanged `turn_before` boundary is reported as an explicit cursor error or no-progress result. The frontend releases the request lock and performs at most one latest-detail calibration; it does not leave a permanently nonfunctional "Load earlier" control.

## 8. Fetch timing

### 8.1 Initial page load

The frontend renders browser persistence immediately, then requests the full lightweight list. After the list commits, it requests the selected active session's latest detail. Network work never clears the cached timeline while loading.

### 8.2 Background and foreground

When `document.visibilityState` becomes hidden:

- polling timers stop;
- no new list or detail request starts;
- the background start time is recorded;
- dirty bounded content is flushed once to persistence.

When the page returns within five minutes:

- stable sessions make no list or detail request;
- locally unfinished sessions resume incremental updates.

When the page returns after at least five minutes:

1. request the full lightweight list;
2. commit it if successful;
3. retain the existing active session if it still exists, otherwise select the newest session;
4. request that active session's latest detail once;
5. resume updates if the resulting active session remains unfinished.

### 8.3 Manual refresh

Manual refresh invalidates older list and active-detail generations, requests a full lightweight list, then requests the resolved active latest detail. Existing content remains visible until replacements are accepted.

## 9. Session switching

Switching sessions never triggers a list request. It immediately updates the active ID and renders in-memory or persisted content.

The target detail is requested when any of these conditions holds:

- no content exists in memory or persistence;
- the list summary `updated_at` is newer than the content cache timestamp;
- the target is unfinished but has no valid incremental synchronization;
- persisted content is malformed;
- the user explicitly refreshes.

No detail request is made when a ready session has loaded content with the same `updated_at`, or when only title or pin state changed.

Only one detail request per session generation is active. Switching away cancels the prior active detail request when possible. If cancellation loses the race, the response may update only its own session cache and can never change the current timeline.

## 10. Browser persistence

Persistence is separated by responsibility:

### `SessionSummaryCache`

- active session ID;
- lightweight session items;
- last successful list time.

### `SessionContentCache:{sessionID}`

- canonical bounded turns and loaded older turns;
- lightweight runtime event summaries;
- paging boundaries;
- the content `updated_at` represented by the snapshot;
- last-used time for eviction.

### `OptimisticInputJournal`

- unconfirmed `client_request_id` values;
- small prompt text;
- stable attachment references.

The cache retains the active session plus up to four recently used session contents, with an approximate aggregate serialized limit of 4 MiB. Non-active least-recently-used content is evicted first. Large on-demand runtime event detail blocks and raw attachment data URLs are not persisted.

Bootstrap reads the summary cache and every retained v2 content entry before the reducer's single `hydrateCache` action. Session switching never performs a second persistence-to-live-state merge after bootstrap.

Summary mutations only write the summary cache. Content writes occur after accepted detail, terminal turn updates, successful history loads, and background transition. Streaming event patches do not synchronously rewrite the full content snapshot on every event. Writes are debounced and originate from one reducer-state persistence subscriber.

Legacy v1 cache may be imported once as bootstrap fallback. Valid `turn-id:role` messages are grouped into bootstrap-only canonical turns; malformed messages are ignored rather than allowed to compete with server data. After the first accepted server calibration, legacy data cannot dispatch again. A successful v2 write removes the old snapshot keys.

## 11. Error behavior

- List failure: retain current summaries and content; do not delete sessions.
- Detail failure: retain the existing timeline and expose retry through manual refresh.
- Update failure: retain the cursor and retry with bounded backoff while visible.
- Pin failure: roll back optimistic pin state and show a failure message.
- Delete failure: keep the session and show a failure message.
- Cache write failure or quota exhaustion: keep reducer state, evict non-active content, and continue without treating persistence failure as a live-data failure.
- Invalid live `updated_at`: reject content replacement when current content exists; do not guess freshness.
- Active-session deletion: choose the newest remaining session and load its detail under a new generation.

## 12. Code boundaries

The current provider is too large and allows transport, state, merge, persistence, and UI configuration to share mutable refs. The implementation introduces focused modules:

- `conversationSessionReducer`: canonical state and pure actions;
- `conversationSessionMerge`: detail, update, history, and optimistic-input rules;
- `conversationSessionCache`: bounded v2 persistence and migration;
- `conversationSessionSync`: list/detail/update request generations and visibility timing;
- `runtimeSessionApi`: stateless endpoint and transport contracts;
- `ConversationRuntimeProvider`: context composition and UI commands only.

The Chat path has exactly one session store. The shared runtime controller may delegate to this store or be reduced to transport commands, but it cannot retain an independent sessions state for Chat.

## 13. Verification strategy

### Reducer and merge tests

- old cache, then new detail, then stale persistence effect: new detail remains;
- new detail, then older list: turns remain untouched;
- updates advance while an older detail is in flight: old detail is rejected;
- newer detail after updates: terminal state is accepted;
- duplicate and descending update IDs: applied once or ignored;
- late history response: only older unknown turns are inserted;
- identical consecutive prompts: optimistic inputs reconcile by client request ID;
- summary-only pin and title changes: content cache remains byte-for-byte unchanged;
- missing server-backed list item: removed only after a successful list response.

### Request lifecycle tests

- initial cache renders before list/detail complete;
- foreground within five minutes with stable state: zero list/detail requests;
- foreground after five minutes: exactly one list and one resolved-active detail request;
- manual refresh: one current list generation followed by one current active detail generation;
- rapid session switching: stale responses never alter the active timeline;
- `has_more`: continuation pages drain immediately while visible;
- `resync_required`: one active detail request;
- hidden page: no new polling request starts.

### API tests

- lightweight list field allowlist;
- atomic detail snapshot;
- strictly monotonic `updated_at` under same-millisecond mutations and clock rollback;
- canonical string IDs across detail, updates, paging, acknowledgements, and event detail;
- `client_request_id` round trip;
- `session.deleted`, `has_more`, and `resync_required` contracts;
- `Cache-Control: no-store` on conversation APIs.

### Browser-level tests

- `visibilitychange`, `pagehide`, `pageshow`, and bfcache restoration;
- cached timeline continuity during refresh;
- mobile rapid switching and background recovery;
- cache quota failure without live-state loss.

## 14. Acceptance criteria

The design is complete when all of the following are true:

- A successfully accepted newer detail or update cannot later be replaced by older cache, state, ref, summary, history, or async response data.
- Switching to a fully cached ready session with matching `updated_at` makes no network request.
- Returning after five minutes performs one lightweight full-list request and one active latest-detail request.
- Title and pin changes never invalidate or rewrite conversation content.
- The session rail, active header, timeline, and persisted cache all derive from the same reducer state.
- Long-running Agent tasks recover from mobile background suspension through update continuation or one explicit resync detail request.
- Manual refresh always provides a deterministic server calibration without blanking the current timeline.

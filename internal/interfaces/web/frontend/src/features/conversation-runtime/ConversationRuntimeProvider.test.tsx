import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { memo, useState } from "react";
import {
  CHAT_RUNTIME_CACHE_SESSION_TTL_MS,
  buildRuntimeSessionUpdateAckManifest,
  ConversationRuntimeProvider,
  mergeRuntimeSessions,
  normalizeRuntimeSession,
  resetConversationRuntimeCache,
  resolveChatSessionPollPlan,
  resolveRuntimeResyncSessionIDs,
  shouldRefreshChatSessionDetailAfterEmptyUpdates,
  type ChatSession,
  useConversationRuntime,
  useConversationRuntimeComposer,
  useConversationRuntimeWorkspace,
} from "./ConversationRuntimeProvider";

const ACTIVE_SESSION_STORAGE_KEY = "alter0.web.session.active.v1";
const ACTIVE_SESSION_SNAPSHOT_STORAGE_KEY = "alter0.web.session.snapshot.v1";
const RECENT_SESSION_SNAPSHOT_STORAGE_KEY = "alter0.web.session.recent.v1";
const COMPOSER_ATTACHMENT_DRAFT_STORAGE_KEY = "alter0.web.composer.attachments.v1";
const LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY = "alter0.web.session.long_term_snapshot.v1";
const SESSION_INFO_SNAPSHOT_STORAGE_KEY = "alter0.web.session.info_snapshot.v1";
const RUNTIME_EVENT_FILTER_STORAGE_KEY = "alter0.web.runtime.event_filter.v1";

async function advanceRuntimePollTimers(count: number) {
  for (let index = 0; index < count; index += 1) {
    await act(async () => {
      await vi.advanceTimersToNextTimerAsync();
      await Promise.resolve();
    });
  }
}

const apiClientMock = {
  get: vi.fn(async () => ({ items: [] })),
  post: vi.fn(),
  patch: vi.fn(),
  delete: vi.fn(),
};

vi.mock("../../shared/api/client", () => ({
  createAPIClient: () => apiClientMock,
}));

function ActiveSessionTitleHarness() {
  const runtime = useConversationRuntimeWorkspace();
  return <output data-testid="active-session-title">{runtime.activeSession?.title || ""}</output>;
}

function RuntimeHarness() {
  const runtime = useConversationRuntime();
  const userMessage = runtime.activeSession?.messages.find((message) => message.role === "user");
  const assistantMessage = runtime.activeSession?.messages.find((message) => message.role === "assistant");

  return (
    <div>
      <button
        type="button"
        onClick={() => void runtime.addDraftAttachments([
          {
            id: "local-image-1",
            kind: "image",
            name: "trace.png",
            contentType: "image/png",
            size: 12,
            dataURL: "data:image/png;base64,ZmFrZS1pbWFnZS1kYXRh",
            previewDataURL: "data:image/webp;base64,c21hbGwtcHJldmlldw==",
          },
        ])}
      >
        attach
      </button>
      <button type="button" onClick={() => void runtime.sendPrompt("Inspect this image")}>
        send
      </button>
      <button type="button" onClick={() => runtime.setDraft("Retry this prompt")}>
        set draft
      </button>
      <button type="button" onClick={() => void runtime.sendPrompt()}>
        send draft
      </button>
      <output data-testid="user-text">{userMessage?.text || ""}</output>
      <output data-testid="assistant-text">{assistantMessage?.text || ""}</output>
      <output data-testid="assistant-texts">
        {runtime.activeSession?.messages
          .filter((message) => message.role === "assistant")
          .map((message) => message.text)
          .join("|") || ""}
      </output>
      <output data-testid="assistant-process-count">{assistantMessage?.processEvents.length || 0}</output>
      <output data-testid="assistant-process-ids">
        {assistantMessage?.processEvents.map((event) => event.id).join("|") || ""}
      </output>
      <output data-testid="assistant-process-status">{assistantMessage?.processEvents[0]?.status || ""}</output>
      <output data-testid="assistant-process-blocks">
        {JSON.stringify(assistantMessage?.processEvents[0]?.blocks || [])}
      </output>
      <button
        type="button"
        onClick={() => {
          const event = assistantMessage?.processEvents[0];
          if (assistantMessage && event) {
            void runtime.loadProcessEventDetail(assistantMessage.id, event.raw?.ref || event.id);
          }
        }}
      >
        load process detail
      </button>
      <button type="button" onClick={() => void runtime.refreshActiveSession()}>
        refresh active
      </button>
      <output data-testid="active-session-status">{runtime.activeSession?.status || ""}</output>
      <output data-testid="composer-submitting">{String(runtime.submitting)}</output>
      <output data-testid="composer-request-notice">{runtime.requestNotice}</output>
      <output data-testid="composer-draft">{runtime.draft}</output>
    </div>
  );
}

function MessageTextHarness() {
  const runtime = useConversationRuntime();
  const processEventCount = runtime.activeSession?.messages.reduce((count, message) => count + message.processEvents.length, 0) || 0;
  return (
    <div>
      <button type="button" onClick={() => void runtime.refreshActiveSession()}>
        refresh active
      </button>
      <button type="button" onClick={() => void runtime.loadEarlierHistory()}>
        load earlier history
      </button>
      <output data-testid="message-texts">
        {runtime.activeSession?.messages.map((message) => message.text).join("|") || ""}
      </output>
      <output data-testid="process-event-count">{processEventCount}</output>
    </div>
  );
}

let composerRenderCount = 0;

const ComposerRenderProbe = memo(function ComposerRenderProbe() {
  const runtime = useConversationRuntimeComposer();
  composerRenderCount += 1;
  return <output data-testid="composer-render-count">{composerRenderCount}:{runtime.draft}</output>;
});

function RuntimeEventFilterHarness() {
  const runtime = useConversationRuntimeComposer();
  return <output data-testid="runtime-event-filter">{runtime.runtimeEventFilter.join("|")}</output>;
}

function RepositoryBindingHarness() {
	const runtime = useConversationRuntimeComposer();
	const [listed, setListed] = useState("");
	return (
		<div>
			<button
				type="button"
				onClick={() => runtime.setDraftRepository({
					id: "123456789",
					fullName: "owner/repository",
					private: true,
					defaultBranch: "main",
					updatedAt: Date.parse("2026-07-11T10:00:00Z"),
				})}
			>
				select repository
			</button>
			<button type="button" onClick={() => void runtime.sendPrompt("Update retry behavior") }>
				send repository prompt
			</button>
			<button
				type="button"
				onClick={() => void runtime.listRepositories("alter0").then((items) => setListed(items.map((item) => item.fullName).join("|")))}
			>
				list repositories
			</button>
			<output data-testid="draft-repository">{runtime.draftRepository?.fullName || ""}</output>
			<output data-testid="bound-repository">{runtime.repositoryBinding?.fullName || ""}</output>
			<output data-testid="listed-repositories">{listed}</output>
		</div>
	);
}

function SendMessageTextHarness() {
  const runtime = useConversationRuntime();
  return (
    <div>
      <button type="button" onClick={() => void runtime.sendPrompt("new prompt")}>
        send prompt
      </button>
      <button type="button" onClick={() => void runtime.refreshActiveSession()}>
        refresh active
      </button>
      <output data-testid="message-texts">
        {runtime.activeSession?.messages.map((message) => message.text).join("|") || ""}
      </output>
    </div>
  );
}

function InspectorHarness() {
  const runtime = useConversationRuntime();

  return (
    <div>
      <button type="button" onClick={() => runtime.toggleInspector("model")}>
        model
      </button>
      <output data-testid="inspector-state">
        {[
          runtime.inspectorTab,
          runtime.inspectorOpen ? "details-open" : "details-closed",
          runtime.inspectorTabOpen ? "tab-open" : "tab-closed",
        ].join(":")}
      </output>
    </div>
  );
}

function ModelSelectionHarness() {
  const runtime = useConversationRuntime();

  return (
    <div>
      <button type="button" onClick={() => runtime.selectModel("alter0-codex", "codex")}>
        select codex
      </button>
      <button type="button" onClick={() => void runtime.sendPrompt("Run this with Codex")}>
        send with codex
      </button>
      <output data-testid="selected-model">{runtime.selectedModelLabel}</output>
      <output data-testid="provider-list">
        {runtime.providers.map((provider) => `${provider.name}:${provider.models.map((model) => model.name).join(",")}`).join("|")}
      </output>
    </div>
  );
}

function RuntimeConfigSelectionHarness() {
  const runtime = useConversationRuntime();
  const filesystemMCP = runtime.capabilities.find((item) => item.id === "filesystem");

  return (
    <div>
      <button type="button" onClick={() => runtime.selectModel("openrouter", "anthropic/claude-sonnet")}>
        select openrouter
      </button>
      <button type="button" onClick={() => runtime.toggleCapability("filesystem", "mcp", true)}>
        enable filesystem
      </button>
      <button type="button" onClick={() => void runtime.sendPrompt("Run with stored config")}>
        send with stored config
      </button>
      <output data-testid="selected-provider">{runtime.selectedProviderId}</output>
      <output data-testid="selected-model">{runtime.selectedModelId}</output>
      <output data-testid="filesystem-state">{filesystemMCP?.active ? "active" : "inactive"}</output>
    </div>
  );
}

function SessionListHarness() {
  const runtime = useConversationRuntimeWorkspace();
  return (
    <div>
      <button type="button" onClick={runtime.createSession}>
        new session
      </button>
      <button type="button" onClick={() => void runtime.setSessionPinned("c_51jttwiv4yggqagk", false)}>
        unpin active
      </button>
      <button type="button" onClick={() => void runtime.setSessionPinned(runtime.activeSession?.id || "", true)}>
        pin active
      </button>
      <button type="button" onClick={() => void runtime.setSessionPinned("c_olderchat0000000", true)}>
        pin older
      </button>
      <output data-testid="sessions">
        {runtime.sessionItems.map((session) => `${session.title}:${session.pinned ? "pinned" : "unpinned"}`).join("|")}
      </output>
    </div>
  );
}

function SessionStatusListHarness() {
  const runtime = useConversationRuntimeWorkspace();
  return (
    <output data-testid="session-statuses">
      {runtime.sessions.map((session) => `${session.id}:${session.status}`).join("|")}
    </output>
  );
}

function FocusSessionHarness() {
  const runtime = useConversationRuntimeWorkspace();
  return (
    <div>
      <button type="button" onClick={() => runtime.focusSession("c_stalechat0000000")}>
        focus stale
      </button>
      <button type="button" onClick={() => runtime.focusSession("c_otherchat0000000")}>
        focus other
      </button>
      <output data-testid="active-session-id">{runtime.activeSession?.id || ""}</output>
    </div>
  );
}

function ProcessToggleHarness() {
  const runtime = useConversationRuntimeWorkspace();
  const assistantMessage = runtime.activeSession?.messages.find((message) => message.role === "assistant");
  return (
    <div>
      <button type="button" onClick={() => assistantMessage ? runtime.toggleProcess(assistantMessage.id) : undefined}>
        toggle process
      </button>
      <button type="button" onClick={() => void runtime.refreshActiveSession()}>
        refresh active
      </button>
      <output data-testid="assistant-process-ready">{assistantMessage?.processEvents.length || 0}</output>
      <output data-testid="assistant-process-message-id">{assistantMessage?.id || ""}</output>
      <output data-testid="assistant-process-collapsed">{String(assistantMessage?.processCollapsed)}</output>
    </div>
  );
}

function setupDefaultAPI() {
  apiClientMock.get.mockImplementation(async (path: string) => {
    switch (path) {
      case "/api/chat/sessions":
        return {
          items: [
            {
              id: "c_51jttwiv4yggqagk",
              title: "Image session",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns: [],
            },
          ],
        };
      case "/api/chat/sessions/c_51jttwiv4yggqagk":
        return {
          session: {
            id: "c_51jttwiv4yggqagk",
            title: "Image session",
            status: "ready",
            created_at: "2026-04-23T03:30:00Z",
            turns: [],
          },
        };
      case "/api/control/llm/providers":
        return { items: [] };
      case "/api/control/skills":
        return { items: [] };
      case "/api/control/mcps":
        return { items: [] };
      default:
        return { items: [] };
    }
  });
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (error: unknown) => void;
  const promise = new Promise<T>((promiseResolve, promiseReject) => {
    resolve = promiseResolve;
    reject = promiseReject;
  });
  return { promise, resolve, reject };
}

function chatSessionFixture(overrides: Partial<ChatSession> = {}): ChatSession {
  return {
    id: "c_51jttwiv4yggqagk",
    sourceRoute: "chat",
    status: "ready",
    title: "Image session",
    titleAuto: true,
    titleScore: 0,
    createdAt: Date.parse("2026-04-23T03:30:00Z"),
    updatedAt: Date.parse("2026-04-23T03:30:00Z"),
    lastOutputAt: 0,
    activityAt: Date.parse("2026-04-23T03:30:00Z"),
    freshnessAt: Date.parse("2026-04-23T03:30:00Z"),
    detailFreshnessAt: Date.parse("2026-04-23T03:30:00Z"),
    pinned: false,
    target: { type: "model", id: "raw-model", name: "Raw Model" },
    modelProviderID: "",
    modelID: "",
    toolIDs: [],
    mcpIDs: [],
    messages: [],
    messagesLoaded: false,
    serverBacked: true,
    ...overrides,
  };
}

function chatTurnFixtures(count: number, outputPrefix = "cached answer") {
  return Array.from({ length: count }, (_, index) => {
    const value = index + 1;
    return {
      id: `turn-cache-${value}`,
      prompt: `cached prompt ${value}`,
      status: "success",
      started_at: `2026-04-23T03:${String(value).padStart(2, "0")}:00Z`,
      finished_at: `2026-04-23T03:${String(value).padStart(2, "0")}:02Z`,
      final_output: `${outputPrefix} ${value}`,
    };
  });
}

function mockMessageDone(output = "Done") {
  apiClientMock.post.mockImplementation(async (path: string, body?: Record<string, unknown>) => {
    if (path === "/api/chat/sessions/c_51jttwiv4yggqagk/input") {
      return {
        session: {
          id: "c_51jttwiv4yggqagk",
          title: "Image session",
          status: "ready",
          created_at: "2026-04-23T03:30:00Z",
          turns: [
            {
              id: "turn-1",
              prompt: typeof body?.input === "string" ? body.input : "Run this",
              status: "success",
              started_at: "2026-04-23T03:31:00Z",
              finished_at: "2026-04-23T03:31:01Z",
              final_output: output,
            },
          ],
        },
      };
    }
    return {};
  });
}

describe("ConversationRuntimeProvider", () => {
  beforeEach(() => {
    resetConversationRuntimeCache();
    composerRenderCount = 0;
    vi.clearAllMocks();
    window.sessionStorage.clear();
    window.localStorage.clear();
    window.history.replaceState({}, "", "/chat");
    window.sessionStorage.setItem(
      ACTIVE_SESSION_STORAGE_KEY,
      JSON.stringify({ chat: "c_51jttwiv4yggqagk" }),
    );
    setupDefaultAPI();
    apiClientMock.post.mockImplementation(async (path: string) => {
      if (path === "/api/chat/sessions") {
        return {
          session: {
            id: "c_newchat000000000",
            title: "New",
            status: "ready",
            created_at: "2026-04-23T04:00:00Z",
            turns: [],
          },
        };
      }
      if (path.endsWith("/attachments")) {
        return {
          items: [
            {
              id: "uploaded-image-1",
              name: "trace.png",
              content_type: "image/png",
              size: 12,
              asset_url: "/api/chat/sessions/c_51jttwiv4yggqagk/attachments/uploaded-image-1/original",
              preview_url: "/api/chat/sessions/c_51jttwiv4yggqagk/attachments/uploaded-image-1/preview",
            },
          ],
        };
      }
      return {};
    });
  });

  afterEach(() => {
    cleanup();
    vi.clearAllTimers();
    vi.useRealTimers();
    resetConversationRuntimeCache();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
    window.localStorage.clear();
    window.history.replaceState({}, "", "/");
  });

  it("pauses recoverable session polling while the page is hidden", () => {
    expect(resolveChatSessionPollPlan({ sessionCount: 0, pageHidden: false })).toEqual({
      enabled: false,
      interval: 0,
    });
    expect(resolveChatSessionPollPlan({ sessionCount: 1, pageHidden: false })).toEqual({
      enabled: true,
      interval: 2000,
    });
    expect(resolveChatSessionPollPlan({
      sessionCount: 1,
      pageHidden: false,
      fallbackAttempt: 1,
    })).toEqual({
      enabled: true,
      interval: 3000,
    });
    expect(resolveChatSessionPollPlan({
      sessionCount: 1,
      pageHidden: false,
      fallbackAttempt: 3,
    })).toEqual({
      enabled: true,
      interval: 5000,
    });
    expect(resolveChatSessionPollPlan({
      sessionCount: 1,
      pageHidden: false,
      fallbackAttempt: 6,
    })).toEqual({
      enabled: true,
      interval: 8000,
    });
    expect(resolveChatSessionPollPlan({ sessionCount: 1, pageHidden: true })).toEqual({
      enabled: false,
      interval: 0,
    });
  });

  it("keeps the Chat runtime cache alive for long single-device route gaps", () => {
    expect(CHAT_RUNTIME_CACHE_SESSION_TTL_MS).toBe(24 * 60 * 60 * 1000);
  });

  it("migrates the old default process disclosure filter to include reasoning", async () => {
    window.localStorage.setItem(RUNTIME_EVENT_FILTER_STORAGE_KEY, JSON.stringify(["important_text"]));

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeEventFilterHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("runtime-event-filter")).toHaveTextContent("important_text|reasoning"));
  });

  it("acks canonical runtime event ids without converting them to numbers", () => {
    const session = chatSessionFixture({
      id: "c_ackchat00000000",
      messages: [{
        id: "turn-1:assistant",
        role: "assistant",
        text: "",
        attachments: [],
        route: "chat",
        source: "runtime",
        error: false,
        status: "running",
        at: 1,
        processEvents: [{
          id: "event-3",
          turn_id: "turn-1",
          seq: 3,
          source: "adapter",
          provider: { engine: "codex", adapter: "codex_cli_json" },
          role: "assistant",
          kind: "reasoning",
          lifecycle: "completed",
          status: "completed",
          title: "Thinking",
          summary: "Thinking",
          blocks: [],
          visibility: "collapsed",
        }],
      }],
    });

    expect(buildRuntimeSessionUpdateAckManifest([session], [session.id])).toEqual([{
      id: session.id,
      turns: [{ id: "turn-1", event_ids: ["event-3"] }],
    }]);
  });

  it("keeps a locally running runtime session busy when a stale list summary returns ready", () => {
    const merged = mergeRuntimeSessions([
      chatSessionFixture({
        status: "ready",
        messages: [],
        messagesLoaded: false,
      }),
    ], [
      chatSessionFixture({
        status: "local_running",
        messages: [{
          id: "turn-1:user",
          role: "user",
          text: "new prompt",
          attachments: [],
          route: "chat",
          source: "runtime",
          error: false,
          status: "",
          at: Date.parse("2026-04-23T03:31:00Z"),
          processEvents: [],
        }],
        messagesLoaded: true,
      }),
    ]);

    expect(merged[0]?.status).toBe("local_running");
    expect(merged[0]?.messages.map((message) => message.text)).toEqual(["new prompt"]);
  });

  it("lets a ready server summary repair a stale cached busy session", () => {
    const merged = mergeRuntimeSessions([
      chatSessionFixture({
        status: "ready",
        updatedAt: Date.parse("2026-07-08T02:19:48Z"),
        freshnessAt: Date.parse("2026-07-08T02:19:48Z"),
        detailFreshnessAt: 0,
        messages: [],
        messagesLoaded: false,
      }),
    ], [
      chatSessionFixture({
        status: "busy",
        updatedAt: Date.parse("2026-07-08T02:19:48Z"),
        freshnessAt: Date.parse("2026-07-08T02:20:00Z"),
        detailFreshnessAt: Date.parse("2026-07-08T02:20:00Z"),
        messagesLoaded: true,
        messages: [{
          id: "turn-1:user",
          role: "user",
          text: "cached stale prompt",
          attachments: [],
          route: "chat",
          source: "runtime",
          error: false,
          status: "",
          at: Date.parse("2026-07-08T02:20:00Z"),
          processEvents: [],
        }],
      }),
    ]);

    expect(merged[0]?.status).toBe("ready");
    expect(merged[0]?.messages.map((message) => message.text)).toEqual(["cached stale prompt"]);
    expect(merged[0]?.messagesLoaded).toBe(true);
  });

  it("repairs a stale cached busy session from the session list while another session is active", async () => {
    window.history.replaceState(window.history.state, "", "/chat?session_id=c_otherchat0000000");
    window.sessionStorage.setItem(ACTIVE_SESSION_STORAGE_KEY, JSON.stringify({ chat: "c_otherchat0000000" }));
    window.localStorage.setItem(
      LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        cachedAt: Date.now(),
        activeSessionByRoute: { chat: "c_otherchat0000000" },
        sessionsByRoute: {
          chat: [
            {
              id: "c_stalechat0000000",
              status: "busy",
              title: "Stale cached busy",
              createdAt: Date.parse("2026-07-08T01:59:32Z"),
              updatedAt: Date.parse("2026-07-08T02:19:48Z"),
              freshnessAt: Date.parse("2026-07-08T02:20:00Z"),
              detailFreshnessAt: Date.parse("2026-07-08T02:20:00Z"),
              pinned: false,
              targetID: "codex",
              targetName: "Codex",
              messages: [{
                id: "turn-stale:user",
                role: "user",
                text: "cached stale prompt",
                attachments: [],
                route: "chat",
                source: "runtime",
                error: false,
                status: "",
                at: Date.parse("2026-07-08T02:20:00Z"),
                processEvents: [],
              }],
              messagesLoaded: true,
              serverBacked: true,
            },
            {
              id: "c_otherchat0000000",
              status: "ready",
              title: "Other chat",
              createdAt: Date.parse("2026-07-09T01:00:00Z"),
              updatedAt: Date.parse("2026-07-09T01:00:00Z"),
              freshnessAt: Date.parse("2026-07-09T01:00:00Z"),
              detailFreshnessAt: Date.parse("2026-07-09T01:00:00Z"),
              pinned: false,
              targetID: "codex",
              targetName: "Codex",
              messages: [],
              messagesLoaded: true,
              serverBacked: true,
            },
          ],
        },
      }),
    );

    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [
              {
                id: "c_stalechat0000000",
                title: "Stale cached busy",
                status: "ready",
                created_at: "2026-07-08T01:59:32Z",
                updated_at: "2026-07-08T02:19:48Z",
              },
              {
                id: "c_otherchat0000000",
                title: "Other chat",
                status: "ready",
                created_at: "2026-07-09T01:00:00Z",
                updated_at: "2026-07-09T01:00:00Z",
              },
            ],
          };
        case "/api/chat/sessions/c_otherchat0000000":
          return {
            session: {
              id: "c_otherchat0000000",
              title: "Other chat",
              status: "ready",
              created_at: "2026-07-09T01:00:00Z",
              updated_at: "2026-07-09T01:00:00Z",
              turns: [],
              turns_paging: { has_more_before: false },
            },
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <SessionStatusListHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("session-statuses")).toHaveTextContent("c_stalechat0000000:ready"));
  });

  it("unblocks input when a newer ready summary overtakes a cached running placeholder", async () => {
    const detailRead = deferred<{
      session: {
        id: string;
        title: string;
        status: string;
        created_at: string;
        updated_at: string;
        revision: number;
        turns: Array<{
          id: string;
          prompt: string;
          status: string;
          started_at: string;
          finished_at: string;
          final_output: string;
        }>;
      };
    }>();
    window.sessionStorage.setItem(ACTIVE_SESSION_STORAGE_KEY, JSON.stringify({ chat: "c_finishedchat0000" }));
    window.localStorage.setItem(
      LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        cachedAt: Date.now(),
        activeSessionByRoute: { chat: "c_finishedchat0000" },
        sessionsByRoute: {
          chat: [{
            id: "c_finishedchat0000",
            status: "busy",
            title: "Cached running chat",
            createdAt: Date.parse("2026-04-23T03:30:00Z"),
            updatedAt: Date.parse("2026-04-23T03:31:00Z"),
            activityAt: Date.parse("2026-04-23T03:31:01Z"),
            revision: Date.parse("2026-04-23T03:31:01Z"),
            detailRevision: Date.parse("2026-04-23T03:31:01Z"),
            pinned: false,
            targetID: "codex",
            targetName: "Codex",
            messages: [
              {
                id: "turn-running:user",
                role: "user",
                text: "cached prompt",
                attachments: [],
                route: "chat",
                source: "runtime",
                error: false,
                status: "queued",
                at: Date.parse("2026-04-23T03:31:00Z"),
                processEvents: [],
              },
              {
                id: "turn-running:assistant",
                role: "assistant",
                text: "",
                attachments: [],
                route: "chat",
                source: "runtime",
                error: false,
                status: "running",
                at: Date.parse("2026-04-23T03:31:01Z"),
                processEvents: [],
              },
            ],
            messagesLoaded: true,
            serverBacked: true,
          }],
        },
      }),
    );
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [{
              id: "c_finishedchat0000",
              title: "Cached running chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              updated_at: "2026-04-23T03:35:00Z",
              revision: Date.parse("2026-04-23T03:35:00Z"),
            }],
          };
        case "/api/chat/sessions/c_finishedchat0000":
          return detailRead.promise;
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });
    apiClientMock.post.mockImplementation(async (path: string, body?: Record<string, unknown>) => {
      if (path === "/api/chat/sessions/c_finishedchat0000/input") {
        return {
          session: {
            id: "c_finishedchat0000",
            title: "Cached running chat",
            status: "busy",
            created_at: "2026-04-23T03:30:00Z",
            updated_at: "2026-04-23T03:36:00Z",
            turns: [{
              id: "turn-next",
              prompt: typeof body?.input === "string" ? body.input : "new prompt",
              status: "running",
              started_at: "2026-04-23T03:36:00Z",
            }],
          },
        };
      }
      return {};
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions"));
    await waitFor(() => expect(screen.getByTestId("active-session-status")).toHaveTextContent("ready"));

    fireEvent.click(screen.getByRole("button", { name: "send" }));

    await waitFor(() => expect(apiClientMock.post).toHaveBeenCalledWith(
      "/api/chat/sessions/c_finishedchat0000/input",
      expect.objectContaining({ input: "Inspect this image" }),
    ));
    await waitFor(() => expect(screen.getByTestId("active-session-status")).toHaveTextContent("busy"));
    await act(async () => {
      detailRead.resolve({
        session: {
          id: "c_finishedchat0000",
          title: "Cached running chat",
          status: "ready",
          created_at: "2026-04-23T03:30:00Z",
          updated_at: "2026-04-23T03:35:00Z",
          revision: Date.parse("2026-04-23T03:35:00Z"),
          turns: [{
            id: "turn-running",
            prompt: "cached prompt",
            status: "success",
            started_at: "2026-04-23T03:31:00Z",
            finished_at: "2026-04-23T03:35:00Z",
            final_output: "cached final output",
          }],
        },
      });
      await detailRead.promise;
    });
  });

  it("backs off detail fallback for consecutive empty update polls", () => {
    expect(shouldRefreshChatSessionDetailAfterEmptyUpdates(0)).toBe(false);
    expect(shouldRefreshChatSessionDetailAfterEmptyUpdates(1)).toBe(false);
    expect(shouldRefreshChatSessionDetailAfterEmptyUpdates(5)).toBe(false);
    expect(shouldRefreshChatSessionDetailAfterEmptyUpdates(6)).toBe(true);
    expect(shouldRefreshChatSessionDetailAfterEmptyUpdates(13)).toBe(false);
    expect(shouldRefreshChatSessionDetailAfterEmptyUpdates(14)).toBe(true);
    expect(shouldRefreshChatSessionDetailAfterEmptyUpdates(21)).toBe(false);
    expect(shouldRefreshChatSessionDetailAfterEmptyUpdates(22)).toBe(true);
  });

  it("keeps restored ready detail authoritative when a stale busy summary arrives later", () => {
    const restoredDetail = chatSessionFixture({
      status: "ready",
      freshnessAt: Date.parse("2026-04-23T03:33:00Z"),
      detailFreshnessAt: Date.parse("2026-04-23T03:33:00Z"),
      updatedAt: Date.parse("2026-04-23T03:33:00Z"),
      lastOutputAt: Date.parse("2026-04-23T03:33:00Z"),
      activityAt: Date.parse("2026-04-23T03:33:00Z"),
      messagesLoaded: true,
      messages: [
        {
          id: "turn-1:user",
          role: "user",
          text: "new prompt",
          attachments: [],
          route: "chat",
          source: "runtime",
          error: false,
          status: "done",
          at: Date.parse("2026-04-23T03:32:00Z"),
          processEvents: [],
        },
        {
          id: "turn-1:assistant",
          role: "assistant",
          text: "restored answer remains visible",
          attachments: [],
          route: "chat",
          source: "runtime",
          error: false,
          status: "done",
          at: Date.parse("2026-04-23T03:33:00Z"),
          processEvents: [],
        },
      ],
    });
    const staleBusySummary = chatSessionFixture({
      status: "busy",
      freshnessAt: Date.parse("2026-04-23T03:33:00Z"),
      detailFreshnessAt: 0,
      updatedAt: Date.parse("2026-04-23T03:31:00Z"),
      lastOutputAt: 0,
      activityAt: Date.parse("2026-04-23T03:31:00Z"),
      messagesLoaded: false,
      messages: [],
    });

    const merged = mergeRuntimeSessions([staleBusySummary], [restoredDetail]);

    expect(merged[0]?.status).toBe("ready");
    expect(merged[0]?.messagesLoaded).toBe(true);
    expect(merged[0]?.detailFreshnessAt).toBe(restoredDetail.detailFreshnessAt);
    expect(merged[0]?.messages.map((message) => message.text)).toEqual([
      "new prompt",
      "restored answer remains visible",
    ]);
  });

  it("keeps restored ready detail authoritative when a stale busy partial carries old messages", () => {
    const restoredDetail = chatSessionFixture({
      status: "ready",
      freshnessAt: Date.parse("2026-07-08T10:20:00Z"),
      detailFreshnessAt: Date.parse("2026-07-08T10:20:00Z"),
      updatedAt: Date.parse("2026-07-08T10:20:00Z"),
      lastOutputAt: Date.parse("2026-07-08T10:20:00Z"),
      activityAt: Date.parse("2026-07-08T10:20:00Z"),
      messagesLoaded: true,
      messages: [
        {
          id: "turn-1:user",
          role: "user",
          text: "first prompt",
          attachments: [],
          route: "chat",
          source: "runtime",
          error: false,
          status: "done",
          at: Date.parse("2026-07-08T10:01:00Z"),
          processEvents: [],
        },
        {
          id: "turn-1:assistant",
          role: "assistant",
          text: "first answer",
          attachments: [],
          route: "chat",
          source: "runtime",
          error: false,
          status: "done",
          at: Date.parse("2026-07-08T10:02:00Z"),
          processEvents: [],
        },
        {
          id: "turn-2:user",
          role: "user",
          text: "second prompt",
          attachments: [],
          route: "chat",
          source: "runtime",
          error: false,
          status: "done",
          at: Date.parse("2026-07-08T10:19:00Z"),
          processEvents: [],
        },
        {
          id: "turn-2:assistant",
          role: "assistant",
          text: "second answer",
          attachments: [],
          route: "chat",
          source: "runtime",
          error: false,
          status: "done",
          at: Date.parse("2026-07-08T10:20:00Z"),
          processEvents: [],
        },
      ],
    });
    const staleBusyPartial = chatSessionFixture({
      status: "busy",
      freshnessAt: Date.parse("2026-07-08T10:01:00Z"),
      detailFreshnessAt: 0,
      updatedAt: Date.parse("2026-07-08T10:01:00Z"),
      lastOutputAt: 0,
      activityAt: Date.parse("2026-07-08T10:01:00Z"),
      messagesLoaded: false,
      messages: [
        {
          id: "turn-stale:user",
          role: "user",
          text: "stale prompt",
          attachments: [],
          route: "chat",
          source: "runtime",
          error: false,
          status: "queued",
          at: Date.parse("2026-07-08T10:01:00Z"),
          processEvents: [],
        },
        {
          id: "turn-stale:assistant",
          role: "assistant",
          text: "",
          attachments: [],
          route: "chat",
          source: "runtime",
          error: false,
          status: "running",
          at: Date.parse("2026-07-08T10:01:01Z"),
          processEvents: [],
        },
      ],
    });

    const merged = mergeRuntimeSessions([staleBusyPartial], [restoredDetail]);

    expect(merged[0]?.status).toBe("ready");
    expect(merged[0]?.messagesLoaded).toBe(true);
    expect(merged[0]?.messages.map((message) => message.text)).toEqual([
      "first prompt",
      "first answer",
      "second prompt",
      "second answer",
    ]);
  });

  it("unblocks a restored ready detail when a cached placeholder assistant has no turn id", () => {
    const restoredDetail = chatSessionFixture({
      status: "ready",
      revision: Date.parse("2026-04-23T03:33:00Z"),
      detailRevision: Date.parse("2026-04-23T03:33:00Z"),
      updatedAt: Date.parse("2026-04-23T03:33:00Z"),
      lastOutputAt: Date.parse("2026-04-23T03:33:00Z"),
      activityAt: Date.parse("2026-04-23T03:33:00Z"),
      messagesLoaded: true,
      messages: [
        {
          id: "turn-1:user",
          role: "user",
          text: "直接进入设置页时，侧边栏中的会话列表之类的是虚假的，排查修复下",
          attachments: [],
          route: "chat",
          source: "runtime",
          error: false,
          status: "completed",
          at: Date.parse("2026-04-23T03:32:00Z"),
          processEvents: [],
        },
        {
          id: "turn-1:assistant",
          role: "assistant",
          text: "已修复。",
          attachments: [],
          route: "chat",
          source: "runtime",
          error: false,
          status: "completed",
          at: Date.parse("2026-04-23T03:33:00Z"),
          processEvents: [],
        },
      ],
    });
    const cachedPlaceholder = chatSessionFixture({
      status: "busy",
      revision: Date.parse("2026-04-23T03:32:00Z"),
      detailRevision: Date.parse("2026-04-23T03:32:00Z"),
      updatedAt: Date.parse("2026-04-23T03:32:00Z"),
      activityAt: Date.parse("2026-04-23T03:32:00Z"),
      messagesLoaded: true,
      messages: [
        {
          id: "local-user",
          role: "user",
          text: "直接进入设置页时，侧边栏中的会话列表之类的是虚假的，排查修复下",
          attachments: [],
          route: "chat",
          source: "runtime",
          error: false,
          status: "queued",
          at: Date.parse("2026-04-23T03:32:00Z"),
          processEvents: [],
        },
        {
          id: "local-assistant",
          role: "assistant",
          text: "",
          attachments: [],
          route: "chat",
          source: "runtime",
          error: false,
          status: "running",
          at: Date.parse("2026-04-23T03:32:01Z"),
          processEvents: [],
        },
      ],
    });

    const merged = mergeRuntimeSessions([restoredDetail], [cachedPlaceholder]);

    expect(merged[0]?.status).toBe("ready");
    expect(merged[0]?.messages.map((message) => message.text)).toEqual([
      "直接进入设置页时，侧边栏中的会话列表之类的是虚假的，排查修复下",
      "已修复。",
    ]);
  });

  it("promotes a completed session back to busy when a newer bounded busy turn arrives", () => {
    const completedDetail = chatSessionFixture({
      status: "ready",
      freshnessAt: Date.parse("2026-04-23T03:33:00Z"),
      detailFreshnessAt: Date.parse("2026-04-23T03:33:00Z"),
      updatedAt: Date.parse("2026-04-23T03:33:00Z"),
      lastOutputAt: Date.parse("2026-04-23T03:33:00Z"),
      activityAt: Date.parse("2026-04-23T03:33:00Z"),
      messagesLoaded: true,
      messages: [
        {
          id: "turn-1:user",
          role: "user",
          text: "old prompt",
          attachments: [],
          route: "chat",
          source: "runtime",
          error: false,
          status: "done",
          at: Date.parse("2026-04-23T03:32:00Z"),
          processEvents: [],
        },
        {
          id: "turn-1:assistant",
          role: "assistant",
          text: "old answer",
          attachments: [],
          route: "chat",
          source: "runtime",
          error: false,
          status: "done",
          at: Date.parse("2026-04-23T03:33:00Z"),
          processEvents: [],
        },
      ],
    });
    const newerBusyBoundedTurn = chatSessionFixture({
      status: "busy",
      freshnessAt: Date.parse("2026-04-23T03:34:00Z"),
      detailFreshnessAt: Date.parse("2026-04-23T03:34:00Z"),
      updatedAt: Date.parse("2026-04-23T03:34:00Z"),
      lastOutputAt: 0,
      activityAt: Date.parse("2026-04-23T03:34:00Z"),
      messagesLoaded: true,
      messages: [{
        id: "turn-2:user",
        role: "user",
        text: "new prompt",
        attachments: [],
        route: "chat",
        source: "runtime",
        error: false,
        status: "running",
        at: Date.parse("2026-04-23T03:34:00Z"),
        processEvents: [],
      }],
      turnsPaging: {
        total: 2,
        limit: 1,
        has_more_before: true,
        oldest_turn_id: "turn-2",
        newest_turn_id: "turn-2",
        next_before_turn_id: "turn-2",
      },
    });

    const merged = mergeRuntimeSessions([newerBusyBoundedTurn], [completedDetail]);

    expect(merged[0]?.status).toBe("busy");
    expect(merged[0]?.messages.map((message) => message.text)).toEqual([
      "old prompt",
      "old answer",
      "new prompt",
    ]);
    expect(merged[0]?.detailFreshnessAt).toBe(newerBusyBoundedTurn.detailFreshnessAt);
  });

  it("keeps exhausted Chat history paging closed when a latest bounded update adds a new turn", () => {
    const previous = chatSessionFixture({
      status: "ready",
      messagesLoaded: true,
      messages: [
        {
          id: "turn-1:user",
          role: "user",
          text: "old prompt",
          attachments: [],
          route: "chat",
          source: "runtime",
          error: false,
          status: "done",
          at: Date.parse("2026-04-23T03:31:00Z"),
          processEvents: [],
        },
        {
          id: "turn-1:assistant",
          role: "assistant",
          text: "old answer",
          attachments: [],
          route: "chat",
          source: "runtime",
          error: false,
          status: "done",
          at: Date.parse("2026-04-23T03:31:01Z"),
          processEvents: [],
        },
      ],
      turnsPaging: {
        has_more_before: false,
        oldest_turn_id: "turn-1",
        newest_turn_id: "turn-1",
      },
    });
    const latestBoundedUpdate = chatSessionFixture({
      status: "ready",
      freshnessAt: Date.parse("2026-04-23T03:32:02Z"),
      detailFreshnessAt: Date.parse("2026-04-23T03:32:02Z"),
      updatedAt: Date.parse("2026-04-23T03:32:02Z"),
      lastOutputAt: Date.parse("2026-04-23T03:32:02Z"),
      activityAt: Date.parse("2026-04-23T03:32:02Z"),
      messagesLoaded: true,
      messages: [
        {
          id: "turn-2:user",
          role: "user",
          text: "new prompt",
          attachments: [],
          route: "chat",
          source: "runtime",
          error: false,
          status: "done",
          at: Date.parse("2026-04-23T03:32:00Z"),
          processEvents: [],
        },
        {
          id: "turn-2:assistant",
          role: "assistant",
          text: "new answer",
          attachments: [],
          route: "chat",
          source: "runtime",
          error: false,
          status: "done",
          at: Date.parse("2026-04-23T03:32:01Z"),
          processEvents: [],
        },
      ],
      turnsPaging: {
        total: 2,
        limit: 1,
        has_more_before: true,
        oldest_turn_id: "turn-2",
        newest_turn_id: "turn-2",
        next_before_turn_id: "turn-2",
      },
    });

    const merged = mergeRuntimeSessions([latestBoundedUpdate], [previous]);

    expect(merged[0]?.messages.map((message) => message.text)).toEqual([
      "old prompt",
      "old answer",
      "new prompt",
      "new answer",
    ]);
    expect(merged[0]?.turnsPaging?.has_more_before).toBe(false);
    expect(merged[0]?.turnsPaging?.next_before_turn_id).toBeUndefined();
  });

  it("keeps newer session detail authoritative when an older detail response arrives later", () => {
    const newerSession = chatSessionFixture({
      status: "ready",
      title: "New detail",
      updatedAt: Date.parse("2026-04-23T03:33:00Z"),
      lastOutputAt: Date.parse("2026-04-23T03:33:00Z"),
      activityAt: Date.parse("2026-04-23T03:33:00Z"),
      freshnessAt: Date.parse("2026-04-23T03:33:00Z"),
      detailFreshnessAt: Date.parse("2026-04-23T03:33:00Z"),
      messagesLoaded: true,
      messages: [{
        id: "turn-new:assistant",
        role: "assistant",
        text: "new answer remains visible",
        attachments: [],
        route: "chat",
        source: "runtime",
        error: false,
        status: "done",
        at: Date.parse("2026-04-23T03:33:01Z"),
        processEvents: [],
      }],
    });
    const olderLateDetail = chatSessionFixture({
      status: "busy",
      title: "Old detail",
      updatedAt: Date.parse("2026-04-23T03:31:00Z"),
      lastOutputAt: Date.parse("2026-04-23T03:31:00Z"),
      activityAt: Date.parse("2026-04-23T03:31:00Z"),
      freshnessAt: Date.parse("2026-04-23T03:31:00Z"),
      detailFreshnessAt: Date.parse("2026-04-23T03:31:00Z"),
      messagesLoaded: true,
      messages: [{
        id: "turn-old:assistant",
        role: "assistant",
        text: "old answer",
        attachments: [],
        route: "chat",
        source: "runtime",
        error: false,
        status: "running",
        at: Date.parse("2026-04-23T03:31:01Z"),
        processEvents: [],
      }],
    });

    const merged = mergeRuntimeSessions([olderLateDetail], [newerSession]);

    expect(merged[0]?.title).toBe("New detail");
    expect(merged[0]?.status).toBe("ready");
    expect(merged[0]?.freshnessAt).toBe(Date.parse("2026-04-23T03:33:00Z"));
    expect(merged[0]?.detailFreshnessAt).toBe(Date.parse("2026-04-23T03:33:00Z"));
    expect(merged[0]?.messages.map((message) => message.text)).toEqual(["new answer remains visible"]);
  });

  it("uses updatedAt rather than legacy freshnessAt when merging newer Chat session detail", () => {
    const previous = chatSessionFixture({
      status: "ready",
      title: "Old freshnessAt-heavy detail",
      updatedAt: Date.parse("2026-04-23T03:31:00Z"),
      activityAt: Date.parse("2026-04-23T03:31:00Z"),
      freshnessAt: Date.parse("2026-04-23T03:40:00Z"),
      detailFreshnessAt: Date.parse("2026-04-23T03:40:00Z"),
      messagesLoaded: true,
      messages: [],
    });
    const incoming = chatSessionFixture({
      status: "ready",
      title: "New updatedAt detail",
      updatedAt: Date.parse("2026-04-23T03:33:00Z"),
      activityAt: Date.parse("2026-04-23T03:33:00Z"),
      freshnessAt: Date.parse("2026-04-23T03:20:00Z"),
      detailFreshnessAt: Date.parse("2026-04-23T03:20:00Z"),
      messagesLoaded: true,
      messages: [{
        id: "turn-updated-at:assistant",
        role: "assistant",
        text: "new answer selected by updatedAt",
        attachments: [],
        route: "chat",
        source: "runtime",
        error: false,
        status: "done",
        at: Date.parse("2026-04-23T03:33:01Z"),
        processEvents: [],
      }],
    });

    const merged = mergeRuntimeSessions([incoming], [previous]);

    expect(merged[0]?.title).toBe("New updatedAt detail");
    expect(merged[0]?.updatedAt).toBe(Date.parse("2026-04-23T03:33:00Z"));
    expect(merged[0]?.messages.map((message) => message.text)).toContain("new answer selected by updatedAt");
  });

  it("orders sessions by conversation updatedAt instead of creation time", () => {
    const recentlyActiveOlderSession = {
      ...chatSessionFixture({
        id: "c_olderactivechat0",
        title: "Older but active",
        createdAt: Date.parse("2026-04-21T03:30:00Z"),
      }),
      updatedAt: Date.parse("2026-04-23T04:31:00Z"),
    };
    const newlyCreatedIdleSession = {
      ...chatSessionFixture({
        id: "c_newidlechat00000",
        title: "New but idle",
        createdAt: Date.parse("2026-04-23T03:30:00Z"),
      }),
      updatedAt: Date.parse("2026-04-23T03:30:00Z"),
    };

    const merged = mergeRuntimeSessions([], [
      newlyCreatedIdleSession,
      recentlyActiveOlderSession,
    ]);

    expect(merged.map((session) => session.id)).toEqual([
      "c_olderactivechat0",
      "c_newidlechat00000",
    ]);
  });

  it("keeps a turn user message before assistant process patches even when timestamps arrive out of order", () => {
    const merged = mergeRuntimeSessions([
      chatSessionFixture({
        status: "busy",
        messagesLoaded: true,
        messages: [{
          id: "turn-1:user",
          role: "user",
          text: "成都旅游攻略",
          attachments: [],
          route: "chat",
          source: "runtime",
          error: false,
          status: "running",
          at: Date.parse("2026-04-23T03:31:05Z"),
          processEvents: [],
        }],
      }),
    ], [
      chatSessionFixture({
        status: "busy",
        messagesLoaded: true,
        messages: [{
          id: "turn-1:assistant",
          role: "assistant",
          text: "",
          attachments: [],
          route: "chat",
          source: "runtime",
          error: false,
          status: "running",
          at: Date.parse("2026-04-23T03:31:00Z"),
          processEvents: [{
            id: "event-1",
            session_id: "c_51jttwiv4yggqagk",
            turn_id: "turn-1",
            seq: 1,
            source: "adapter",
            provider: { engine: "codex", adapter: "codex_cli_json" },
            role: "assistant",
            kind: "assistant_commentary",
            lifecycle: "running",
            status: "running",
            title: "Thinking",
            summary: "Working",
            blocks: [],
            visibility: "collapsed",
          }],
        }],
      }),
    ]);

    expect(merged[0]?.messages.map((message) => message.id)).toEqual([
      "turn-1:user",
      "turn-1:assistant",
    ]);
  });

  it("keeps an optimistic user message before the first busy assistant patch", () => {
    const merged = mergeRuntimeSessions([
      chatSessionFixture({
        status: "busy",
        messagesLoaded: true,
        messages: [{
          id: "turn-1:assistant",
          role: "assistant",
          text: "",
          attachments: [],
          route: "chat",
          source: "runtime",
          error: false,
          status: "running",
          at: Date.parse("2026-04-23T03:31:00Z"),
          processEvents: [{
            id: "event-1",
            session_id: "c_51jttwiv4yggqagk",
            turn_id: "turn-1",
            seq: 1,
            source: "adapter",
            provider: { engine: "codex", adapter: "codex_cli_json" },
            role: "assistant",
            kind: "assistant_commentary",
            lifecycle: "running",
            status: "running",
            title: "Thinking",
            summary: "Working",
            blocks: [],
            visibility: "collapsed",
          }],
        }],
      }),
    ], [
      chatSessionFixture({
        status: "busy",
        messagesLoaded: true,
        messages: [{
          id: "local-user-1",
          role: "user",
          text: "成都旅游攻略",
          attachments: [],
          route: "chat",
          source: "runtime",
          error: false,
          status: "queued",
          at: Date.parse("2026-04-23T03:31:05Z"),
          processEvents: [],
        }],
      }),
    ]);

    expect(merged[0]?.messages.map((message) => message.id)).toEqual([
      "local-user-1",
      "turn-1:assistant",
    ]);
  });

  it("selects only local sync intent sessions for owner-level polling resync", () => {
    expect(resolveRuntimeResyncSessionIDs([
      chatSessionFixture({
        id: "c_busychat00000000",
        status: "busy",
        messagesLoaded: true,
        serverBacked: true,
      }),
      chatSessionFixture({
        id: "c_readychat0000000",
        status: "ready",
        messagesLoaded: true,
        serverBacked: true,
      }),
      chatSessionFixture({
        id: "c_localrunningchat",
        status: "local_running",
        messagesLoaded: true,
        serverBacked: true,
      }),
      chatSessionFixture({
        id: "c_recoveringchat00",
        status: "recovering",
        messagesLoaded: true,
        serverBacked: true,
      }),
      chatSessionFixture({
        id: "local-busy",
        status: "busy",
        messagesLoaded: true,
        serverBacked: false,
      }),
    ])).toEqual(["c_localrunningchat", "c_recoveringchat00"]);
  });

  it("does not poll updates for restored sessions without local sync intent", async () => {
    vi.stubGlobal("fetch", vi.fn());
    apiClientMock.post.mockImplementation(async () => ({}));
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [
              {
                id: "c_readychat0000000",
                title: "Ready restored session",
                status: "ready",
                created_at: "2026-04-23T03:30:00Z",
                updated_at: "2026-04-23T03:35:00Z",
                turns: [],
              },
              {
                id: "c_emptystatuschat0",
                title: "Empty status restored session",
                created_at: "2026-04-23T03:20:00Z",
                updated_at: "2026-04-23T03:25:00Z",
                turns: [],
              },
              {
                id: "c_serverbusychat00",
                title: "Server busy from another device",
                status: "busy",
                created_at: "2026-04-23T03:10:00Z",
                updated_at: "2026-04-23T03:15:00Z",
                turns: [],
              },
            ],
          };
        case "/api/chat/sessions/c_readychat0000000":
        case "/api/chat/sessions/c_emptystatuschat0":
        case "/api/chat/sessions/c_serverbusychat00":
          return {
            session: {
              id: path.split("/").pop(),
              title: "Restored session",
              status: path.endsWith("c_serverbusychat00") ? "busy" : "ready",
              created_at: "2026-04-23T03:30:00Z",
              updated_at: "2026-04-23T03:35:00Z",
              turns: [],
            },
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    const view = render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions"));
    vi.useFakeTimers();
    await act(async () => {
      await vi.advanceTimersByTimeAsync(15000);
      await Promise.resolve();
    });

    expect(apiClientMock.post).not.toHaveBeenCalledWith(
      "/api/chat/sessions/updates",
      expect.anything(),
    );
    view.unmount();
    vi.clearAllTimers();
    vi.useRealTimers();
  });

  it("opens a default-collapsed completed process on the first toggle", async () => {
    const completedTurn = {
      id: "turn-1",
      prompt: "show process",
      status: "success",
      started_at: "2026-04-23T03:31:00Z",
      finished_at: "2026-04-23T03:31:01Z",
      final_output: "Done",
      runtime_trace_events: [
        {
          id: "event-1",
          turn_id: "turn-1",
          seq: 1,
          source: "adapter",
          provider: { engine: "codex", adapter: "codex_cli_json", event_type: "message", item_id: "event-1" },
          role: "assistant",
          kind: "assistant_commentary",
          lifecycle: "completed",
          status: "completed",
          title: "Progress",
          summary: "Progress detail",
          blocks: [{ type: "markdown", text: "Progress detail" }],
          visibility: "collapsed",
          raw: { ref: "event-1", type: "message", has_detail: true },
        },
      ],
    };
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
        case "/api/chat/sessions/c_51jttwiv4yggqagk":
          return {
            session: {
              id: "c_51jttwiv4yggqagk",
              title: "Process session",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns: [completedTurn],
            },
            items: [{
              id: "c_51jttwiv4yggqagk",
              title: "Process session",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns: [completedTurn],
            }],
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <ProcessToggleHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => {
      expect(screen.getByTestId("assistant-process-ready")).toHaveTextContent("1");
      expect(screen.getByTestId("assistant-process-message-id")).toHaveTextContent("turn-1:assistant");
      expect(screen.getByTestId("assistant-process-collapsed")).toHaveTextContent("true");
    });

    fireEvent.click(screen.getByRole("button", { name: "toggle process" }));
    await waitFor(() => expect(screen.getByTestId("assistant-process-collapsed")).toHaveTextContent("false"));

    fireEvent.click(screen.getByRole("button", { name: "refresh active" }));
    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions/c_51jttwiv4yggqagk"));
    expect(screen.getByTestId("assistant-process-collapsed")).toHaveTextContent("false");

    fireEvent.click(screen.getByRole("button", { name: "toggle process" }));
    await waitFor(() => expect(screen.getByTestId("assistant-process-collapsed")).toHaveTextContent("true"));
  });

  it("hydrates a fresh Chat runtime cache immediately and refreshes after the API returns", async () => {
    const cachedTurnCount = 18;
    const cachedTurns = chatTurnFixtures(cachedTurnCount);
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [{
              id: "c_51jttwiv4yggqagk",
              title: "Cached chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns: cachedTurns,
            }],
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    const firstView = render(
      <ConversationRuntimeProvider route="chat" language="en">
        <ActiveSessionTitleHarness />
        <MessageTextHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent(`cached answer ${cachedTurnCount}`));
    expect(screen.getByTestId("message-texts")).toHaveTextContent("cached answer 1");
    firstView.unmount();
    window.sessionStorage.removeItem(ACTIVE_SESSION_SNAPSHOT_STORAGE_KEY);
    window.sessionStorage.removeItem(RECENT_SESSION_SNAPSHOT_STORAGE_KEY);

    const listRequest = deferred<{ items?: unknown[] }>();
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return listRequest.promise;
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <ActiveSessionTitleHarness />
        <MessageTextHarness />
      </ConversationRuntimeProvider>,
    );

    expect(screen.getByTestId("active-session-title")).toHaveTextContent("Cached chat");
    expect(screen.getByTestId("message-texts")).toHaveTextContent(`cached answer ${cachedTurnCount}`);
    expect(screen.getByTestId("message-texts")).toHaveTextContent("cached answer 1");

    listRequest.resolve({
      items: [{
        id: "c_51jttwiv4yggqagk",
        title: "Server chat",
        status: "ready",
        created_at: "2026-04-23T03:30:00Z",
        turns: chatTurnFixtures(cachedTurnCount, "server answer"),
      }],
    });

    await waitFor(() => expect(screen.getByTestId("active-session-title")).toHaveTextContent("Server chat"));
    expect(screen.getByTestId("message-texts")).toHaveTextContent(`server answer ${cachedTurnCount}`);
  });

  it("hydrates all Chat messages from the long-lived browser cache before calibrating active detail", async () => {
    const cachedTurnCount = 18;
    window.sessionStorage.setItem(ACTIVE_SESSION_STORAGE_KEY, JSON.stringify({ chat: "c_51jttwiv4yggqagk" }));
    window.localStorage.setItem(
      LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        cachedAt: Date.now(),
        activeSessionByRoute: { chat: "c_51jttwiv4yggqagk" },
        sessionsByRoute: {
          chat: [{
            id: "c_51jttwiv4yggqagk",
            status: "ready",
            title: "Durable cached chat",
            createdAt: Date.parse("2026-04-23T03:30:00Z"),
            updatedAt: Date.parse("2026-04-23T03:48:00Z"),
            lastOutputAt: Date.parse("2026-04-23T03:48:00Z"),
            activityAt: Date.parse("2026-04-23T03:48:00Z"),
            freshnessAt: Date.parse("2026-04-23T03:48:00Z"),
            detailFreshnessAt: Date.parse("2026-04-23T03:48:00Z"),
            pinned: false,
            targetID: "codex",
            targetName: "Codex",
            messages: chatTurnFixtures(cachedTurnCount, "durable answer").flatMap((turn) => ([
              {
                id: `${turn.id}:user`,
                role: "user",
                text: turn.prompt,
                attachments: [],
                route: "chat",
                source: "runtime",
                error: false,
                status: "",
                at: Date.parse(turn.started_at),
                processEvents: [],
              },
              {
                id: `${turn.id}:assistant`,
                role: "assistant",
                text: turn.final_output,
                attachments: [],
                route: "chat",
                source: "runtime",
                error: false,
                status: "done",
                at: Date.parse(turn.finished_at),
                processEvents: [],
              },
            ])),
            messagesLoaded: true,
            serverBacked: true,
            turnsPaging: { has_more_before: false },
          }],
        },
      }),
    );

    const listRequest = deferred<{ items?: unknown[] }>();
    let detailReads = 0;
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return listRequest.promise;
        case "/api/chat/sessions/c_51jttwiv4yggqagk":
          detailReads += 1;
          return {
            session: {
              id: "c_51jttwiv4yggqagk",
              title: "Durable cached chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              updated_at: "2026-04-23T03:48:00Z",
              last_output_at: "2026-04-23T03:48:00Z",
              turns_paging: { has_more_before: false },
              turns: chatTurnFixtures(cachedTurnCount, "durable answer").map((turn, index) => (
                index === cachedTurnCount - 1
                  ? {
                      ...turn,
                      runtime_trace_events: [{
                        id: "event-reasoning-1",
                        turn_id: turn.id,
                        seq: 1,
                        source: "adapter",
                        provider: { engine: "codex", adapter: "codex_cli_json", event_type: "reasoning" },
                        role: "assistant",
                        kind: "reasoning",
                        lifecycle: "completed",
                        status: "completed",
                        title: "Thinking through the route",
                        summary: "Thinking through the route",
                        blocks: [{ type: "markdown", text: "Thinking through the route" }],
                        visibility: "collapsed",
                        raw: { ref: "event-reasoning-1", type: "reasoning", has_detail: true },
                      }],
                    }
                  : turn
              )),
            },
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <ActiveSessionTitleHarness />
        <MessageTextHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("active-session-title")).toHaveTextContent("Durable cached chat"));
    expect(screen.getByTestId("message-texts")).toHaveTextContent("durable answer 1");
    expect(screen.getByTestId("message-texts")).toHaveTextContent(`durable answer ${cachedTurnCount}`);
    expect(screen.getByTestId("process-event-count")).toHaveTextContent("0");

    listRequest.resolve({
      items: [{
        id: "c_51jttwiv4yggqagk",
        title: "Durable cached chat",
        status: "ready",
        created_at: "2026-04-23T03:30:00Z",
        updated_at: "2026-04-23T03:48:00Z",
        last_output_at: "2026-04-23T03:48:00Z",
        turns_paging: { has_more_before: false },
        turns: [],
      }],
    });

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions/c_51jttwiv4yggqagk"));
    expect(detailReads).toBeGreaterThan(0);
    await waitFor(() => expect(screen.getByTestId("process-event-count")).toHaveTextContent("1"));
    expect(screen.getByTestId("message-texts")).toHaveTextContent("durable answer 1");
    expect(screen.getByTestId("message-texts")).toHaveTextContent(`durable answer ${cachedTurnCount}`);
  });

  it("drops legacy cached Chat session ids before requesting session detail", async () => {
    window.sessionStorage.setItem(ACTIVE_SESSION_STORAGE_KEY, JSON.stringify({ chat: "alter0-chat" }));
    window.localStorage.setItem(
      LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        cachedAt: Date.now(),
        activeSessionByRoute: { chat: "alter0-chat" },
        sessionsByRoute: {
          chat: [chatSessionFixture({ id: "alter0-chat", title: "Legacy cached chat" })],
        },
      }),
    );

    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [{
              id: "c_51jttwiv4yggqagk",
              title: "Canonical chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns: [],
            }],
          };
        case "/api/chat/sessions/c_51jttwiv4yggqagk":
          return {
            session: {
              id: "c_51jttwiv4yggqagk",
              title: "Canonical chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns: [],
            },
          };
        case "/api/chat/sessions/alter0-chat":
          throw new Error("legacy detail should not be requested");
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <ActiveSessionTitleHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("active-session-title")).toHaveTextContent("Canonical chat"));
    expect(apiClientMock.get).not.toHaveBeenCalledWith("/api/chat/sessions/alter0-chat");
  });

  it("shows cached Chat messages immediately, then accepts an authoritative empty session list", async () => {
    const cachedTurnCount = 3;
    window.sessionStorage.setItem(ACTIVE_SESSION_STORAGE_KEY, JSON.stringify({ chat: "c_51jttwiv4yggqagk" }));
    window.localStorage.setItem(
      LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        cachedAt: Date.now(),
        activeSessionByRoute: { chat: "c_51jttwiv4yggqagk" },
        sessionsByRoute: {
          chat: [chatSessionFixture({
            id: "c_51jttwiv4yggqagk",
            title: "Cached resilient chat",
            messages: chatTurnFixtures(cachedTurnCount, "resilient answer").flatMap((turn) => ([{
              id: `${turn.id}:user`,
              role: "user",
              text: turn.prompt,
              attachments: [],
              route: "chat",
              source: "runtime",
              error: false,
              status: "",
              at: Date.parse(turn.started_at),
              processEvents: [],
            }, {
              id: `${turn.id}:assistant`,
              role: "assistant",
              text: turn.final_output,
              attachments: [],
              route: "chat",
              source: "runtime",
              error: false,
              status: "done",
              at: Date.parse(turn.finished_at),
              processEvents: [],
            }])),
            messagesLoaded: true,
            turnsPaging: { has_more_before: false },
          })],
        },
      }),
    );

    const listRequest = deferred<{ items?: unknown[] }>();
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return listRequest.promise;
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <ActiveSessionTitleHarness />
        <MessageTextHarness />
      </ConversationRuntimeProvider>,
    );

    expect(screen.getByTestId("active-session-title")).toHaveTextContent("Cached resilient chat");
    expect(screen.getByTestId("message-texts")).toHaveTextContent("resilient answer 1");
    expect(screen.getByTestId("message-texts")).toHaveTextContent(`resilient answer ${cachedTurnCount}`);

    await act(async () => {
      listRequest.resolve({ items: [] });
      await Promise.resolve();
    });

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions"));
    await act(async () => {
      await Promise.resolve();
    });
    expect(screen.getByTestId("active-session-title")).toHaveTextContent("");
    expect(screen.getByTestId("message-texts")).toHaveTextContent("");
  });

  it("keeps cached Chat messages loaded when a newer summary arrives without turns", async () => {
    window.history.replaceState(window.history.state, "", "/chat?session_id=c_otherchat0000000");
    window.sessionStorage.setItem(ACTIVE_SESSION_STORAGE_KEY, JSON.stringify({ chat: "c_otherchat0000000" }));
    window.localStorage.setItem(
      LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        cachedAt: Date.now(),
        activeSessionByRoute: { chat: "c_otherchat0000000" },
        sessionsByRoute: {
          chat: [
            {
              id: "c_stalechat0000000",
              status: "ready",
              title: "Stale chat",
              createdAt: Date.parse("2026-04-23T03:20:00Z"),
              updatedAt: Date.parse("2026-04-23T03:30:00Z"),
              lastOutputAt: Date.parse("2026-04-23T03:30:00Z"),
              activityAt: Date.parse("2026-04-23T03:30:00Z"),
              freshnessAt: Date.parse("2026-04-23T03:30:00Z"),
              detailFreshnessAt: Date.parse("2026-04-23T03:30:00Z"),
              pinned: false,
              targetID: "codex",
              targetName: "Codex",
              messages: [{
                id: "turn-old:assistant",
                role: "assistant",
                text: "old cached answer",
                attachments: [],
                route: "chat",
                source: "runtime",
                error: false,
                status: "done",
                at: Date.parse("2026-04-23T03:30:00Z"),
                processEvents: [],
              }],
              messagesLoaded: true,
              serverBacked: true,
              turnsPaging: { has_more_before: false },
            },
            {
              id: "c_otherchat0000000",
              status: "ready",
              title: "Other chat",
              createdAt: Date.parse("2026-04-23T03:25:00Z"),
              updatedAt: Date.parse("2026-04-23T03:31:00Z"),
              lastOutputAt: Date.parse("2026-04-23T03:31:00Z"),
              activityAt: Date.parse("2026-04-23T03:31:00Z"),
              freshnessAt: Date.parse("2026-04-23T03:31:00Z"),
              detailFreshnessAt: Date.parse("2026-04-23T03:31:00Z"),
              pinned: false,
              targetID: "codex",
              targetName: "Codex",
              messages: [{
                id: "turn-other:assistant",
                role: "assistant",
                text: "other answer",
                attachments: [],
                route: "chat",
                source: "runtime",
                error: false,
                status: "done",
                at: Date.parse("2026-04-23T03:31:00Z"),
                processEvents: [],
              }],
              messagesLoaded: true,
              serverBacked: true,
              turnsPaging: { has_more_before: false },
            },
          ],
        },
      }),
    );

    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [
              {
                id: "c_stalechat0000000",
                title: "Stale chat",
                status: "ready",
                created_at: "2026-04-23T03:20:00Z",
                updated_at: "2026-04-23T03:40:00Z",
                last_output_at: "2026-04-23T03:40:00Z",
                activity_at: "2026-04-23T03:40:00Z",
              },
              {
                id: "c_otherchat0000000",
                title: "Other chat",
                status: "ready",
                created_at: "2026-04-23T03:25:00Z",
                updated_at: "2026-04-23T03:31:00Z",
                last_output_at: "2026-04-23T03:31:00Z",
                activity_at: "2026-04-23T03:31:00Z",
              },
            ],
          };
        case "/api/chat/sessions/c_otherchat0000000":
          return {
            session: {
              id: "c_otherchat0000000",
              title: "Other chat",
              status: "ready",
              created_at: "2026-04-23T03:25:00Z",
              updated_at: "2026-04-23T03:31:00Z",
              last_output_at: "2026-04-23T03:31:00Z",
              turns: [{
                id: "turn-other",
                prompt: "other prompt",
                status: "success",
                started_at: "2026-04-23T03:31:00Z",
                finished_at: "2026-04-23T03:31:02Z",
                final_output: "other answer",
              }],
              turns_paging: { has_more_before: false },
            },
          };
        case "/api/chat/sessions/c_stalechat0000000":
          return {
            session: {
              id: "c_stalechat0000000",
              title: "Stale chat",
              status: "ready",
              created_at: "2026-04-23T03:20:00Z",
              updated_at: "2026-04-23T03:40:00Z",
              turns: [],
              turns_paging: { has_more_before: false },
            },
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <FocusSessionHarness />
        <MessageTextHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("active-session-id")).toHaveTextContent("c_otherchat0000000"));
    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("other answer"));
    apiClientMock.get.mockClear();

    fireEvent.click(screen.getByRole("button", { name: "focus stale" }));

    await waitFor(() => expect(screen.getByTestId("active-session-id")).toHaveTextContent("c_stalechat0000000"));
    expect(screen.getByTestId("message-texts")).toHaveTextContent("old cached answer");
    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions/c_stalechat0000000"));
  });

  it("merges Chat event turn patches without marking detail messages loaded", () => {
    const previous = chatSessionFixture({
      id: "c_eventpatch000000",
      status: "local_running",
      updatedAt: Date.parse("2026-04-23T03:30:00Z"),
      activityAt: Date.parse("2026-04-23T03:30:00Z"),
      freshnessAt: Date.parse("2026-04-23T03:30:00Z"),
      detailFreshnessAt: 0,
      messages: [],
      messagesLoaded: false,
      turnsPaging: undefined,
    });

    const normalized = normalizeRuntimeSession({
      id: "c_eventpatch000000",
      title: "Event patch",
      status: "ready",
      created_at: "2026-04-23T03:20:00Z",
      updated_at: "2026-04-23T03:31:00Z",
      turns: [{
        id: "turn-event",
        prompt: "event prompt",
        status: "success",
        started_at: "2026-04-23T03:31:00Z",
        finished_at: "2026-04-23T03:31:01Z",
        final_output: "event answer",
      }],
    }, previous, "chat", { source: "event" });

    expect(normalized?.messages.map((message) => message.text)).toEqual(["event prompt", "event answer"]);
    expect(normalized?.messagesLoaded).toBe(false);
    expect(normalized?.detailFreshnessAt).toBe(0);
  });

  it("drops stale recoverable Chat placeholders when a ready detail page returns a newer turn", () => {
    const previous = chatSessionFixture({
      id: "c_jwq2bz6wjw3lyusw",
      status: "busy",
      updatedAt: Date.parse("2026-07-08T10:01:00Z"),
      activityAt: Date.parse("2026-07-08T10:01:00Z"),
      freshnessAt: Date.parse("2026-07-08T10:01:00Z"),
      detailFreshnessAt: 0,
      messagesLoaded: false,
      messages: [
        {
          id: "turn-1:user",
          role: "user",
          text: "old prompt",
          attachments: [],
          route: "chat",
          source: "runtime",
          error: false,
          status: "running",
          at: Date.parse("2026-07-08T10:01:00Z"),
          processEvents: [],
        },
        {
          id: "turn-1:assistant",
          role: "assistant",
          text: "",
          attachments: [],
          route: "chat",
          source: "runtime",
          error: false,
          status: "running",
          at: Date.parse("2026-07-08T10:01:01Z"),
          processEvents: [],
        },
      ],
      turnsPaging: undefined,
    });

    const normalized = normalizeRuntimeSession({
      id: "c_jwq2bz6wjw3lyusw",
      title: "Ready detail",
      status: "ready",
      created_at: "2026-07-08T09:59:32Z",
      updated_at: "2026-07-08T10:19:48Z",
      last_output_at: "2026-07-08T10:19:40Z",
      activity_at: "2026-07-08T10:19:48Z",
      turns_paging: {
        total: 2,
        has_more_before: true,
        oldest_turn_id: "turn-2",
        newest_turn_id: "turn-2",
        next_before_turn_id: "turn-2",
      },
      turns: [{
        id: "turn-2",
        prompt: "new prompt",
        status: "success",
        started_at: "2026-07-08T10:19:00Z",
        finished_at: "2026-07-08T10:19:40Z",
        final_output: "new ready answer",
      }],
    }, previous, "chat", { source: "detail" });

    expect(normalized?.status).toBe("ready");
    expect(normalized?.messages.map((message) => message.id)).toEqual(["turn-2:user", "turn-2:assistant"]);
    expect(normalized?.messages.map((message) => message.text)).toEqual(["new prompt", "new ready answer"]);
    expect(normalized?.messagesLoaded).toBe(true);
    expect(normalized?.turnsPaging?.has_more_before).toBe(true);
  });

  it("keeps runtime trace events when detail refresh fills a cached assistant message", () => {
    const cached = chatSessionFixture({
      messagesLoaded: true,
      messages: [{
        id: "turn-1:assistant",
        role: "assistant",
        text: "cached answer",
        attachments: [],
        route: "chat",
        source: "runtime",
        error: false,
        status: "done",
        at: Date.parse("2026-04-23T03:30:01Z"),
        processEvents: [],
      }],
    });
    const detailed = chatSessionFixture({
      messagesLoaded: true,
      messages: [{
        id: "turn-1:assistant",
        role: "assistant",
        text: "cached answer",
        attachments: [],
        route: "chat",
        source: "runtime",
        error: false,
        status: "done",
        at: Date.parse("2026-04-23T03:30:01Z"),
        processEvents: [{
          id: "event-1",
          turn_id: "turn-1",
          seq: 1,
          source: "adapter",
          provider: { engine: "codex", adapter: "codex_cli_json", event_type: "reasoning" },
          role: "assistant",
          kind: "reasoning",
          lifecycle: "completed",
          status: "completed",
          title: "Thinking",
          summary: "Thinking",
          blocks: [{ type: "markdown", text: "Thinking" }],
          visibility: "collapsed",
          raw: { ref: "event-1", type: "reasoning", has_detail: true },
        }],
      }],
    });

    const merged = mergeRuntimeSessions([detailed], [cached]);

    expect(merged[0]?.messages[0]?.processEvents).toHaveLength(1);
  });

  it("keeps runtime trace events when a later summary carries cached messages without events", () => {
    const detailed = chatSessionFixture({
      messagesLoaded: true,
      messages: [{
        id: "turn-1:assistant",
        role: "assistant",
        text: "cached answer",
        attachments: [],
        route: "chat",
        source: "runtime",
        error: false,
        status: "done",
        at: Date.parse("2026-04-23T03:30:01Z"),
        processEvents: [{
          id: "event-1",
          turn_id: "turn-1",
          seq: 1,
          source: "adapter",
          provider: { engine: "codex", adapter: "codex_cli_json", event_type: "reasoning" },
          role: "assistant",
          kind: "reasoning",
          lifecycle: "completed",
          status: "completed",
          title: "Thinking",
          summary: "Thinking",
          blocks: [{ type: "markdown", text: "Thinking" }],
          visibility: "collapsed",
          raw: { ref: "event-1", type: "reasoning", has_detail: true },
        }],
      }],
    });
    const summary = chatSessionFixture({
      messagesLoaded: true,
      messages: [{
        id: "turn-1:assistant",
        role: "assistant",
        text: "cached answer",
        attachments: [],
        route: "chat",
        source: "runtime",
        error: false,
        status: "done",
        at: Date.parse("2026-04-23T03:30:01Z"),
        processEvents: [],
      }],
    });

    const merged = mergeRuntimeSessions([summary], [detailed]);

    expect(merged[0]?.messages[0]?.processEvents).toHaveLength(1);
  });

  it("does not refresh a full stable Chat session on page activation", async () => {
    window.sessionStorage.setItem(ACTIVE_SESSION_STORAGE_KEY, JSON.stringify({ chat: "c_cachedchat000000" }));
    window.localStorage.setItem(
      LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        cachedAt: Date.now(),
        activeSessionByRoute: { chat: "c_cachedchat000000" },
        sessionsByRoute: {
          chat: [{
            id: "c_cachedchat000000",
            status: "ready",
            title: "Fully cached chat",
            createdAt: Date.parse("2026-04-23T03:30:00Z"),
            pinned: false,
            targetID: "codex",
            targetName: "Codex",
            messages: [
              {
                id: "cached-turn:user",
                role: "user",
                text: "cached prompt",
                attachments: [],
                route: "chat",
                source: "runtime",
                error: false,
                status: "",
                at: Date.parse("2026-04-23T03:30:00Z"),
                processEvents: [],
              },
              {
                id: "cached-turn:assistant",
                role: "assistant",
                text: "cached answer",
                attachments: [],
                route: "chat",
                source: "runtime",
                error: false,
                status: "done",
                at: Date.parse("2026-04-23T03:30:02Z"),
                processEvents: [],
              },
            ],
            messagesLoaded: true,
            serverBacked: true,
            turnsPaging: { has_more_before: false },
          }],
        },
      }),
    );
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [{
              id: "c_cachedchat000000",
              title: "Fully cached chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns: [],
              turns_paging: { has_more_before: false },
            }],
          };
        case "/api/chat/sessions/c_cachedchat000000":
          return {
            session: {
              id: "c_cachedchat000000",
              title: "Fully cached chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns_paging: { has_more_before: false },
              turns: [{
                id: "cached-turn",
                prompt: "cached prompt",
                status: "success",
                started_at: "2026-04-23T03:30:00Z",
                finished_at: "2026-04-23T03:30:02Z",
                final_output: "cached answer",
              }],
            },
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <ActiveSessionTitleHarness />
        <MessageTextHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("cached answer"));
    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions"));
    apiClientMock.get.mockClear();

    await act(async () => {
      window.dispatchEvent(new PageTransitionEvent("pageshow", { persisted: true }));
      window.dispatchEvent(new Event("focus"));
      await Promise.resolve();
    });

    expect(apiClientMock.get).not.toHaveBeenCalledWith("/api/chat/sessions");
    expect(apiClientMock.get).not.toHaveBeenCalledWith("/api/chat/sessions/c_cachedchat000000");
    expect(apiClientMock.get).not.toHaveBeenCalledWith(
      "/api/chat/sessions/c_cachedchat000000?turn_before=cached-turn&turn_limit=20",
    );
    expect(screen.getByTestId("message-texts")).toHaveTextContent("cached answer");
  });

  it("does not refresh a stable latest Chat page cache on page activation", async () => {
    window.sessionStorage.setItem(ACTIVE_SESSION_STORAGE_KEY, JSON.stringify({ chat: "c_longchat00000000" }));
    window.localStorage.setItem(
      LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        cachedAt: Date.now(),
        activeSessionByRoute: { chat: "c_longchat00000000" },
        sessionsByRoute: {
          chat: [{
            id: "c_longchat00000000",
            status: "ready",
            title: "Long stable chat",
            createdAt: Date.parse("2026-04-23T03:30:00Z"),
            updatedAt: Date.parse("2026-04-23T04:30:00Z"),
            lastOutputAt: Date.parse("2026-04-23T04:30:00Z"),
            activityAt: Date.parse("2026-04-23T04:30:00Z"),
            pinned: false,
            targetID: "codex",
            targetName: "Codex",
            messages: [
              {
                id: "turn-latest:user",
                role: "user",
                text: "latest prompt",
                attachments: [],
                route: "chat",
                source: "runtime",
                error: false,
                status: "",
                at: Date.parse("2026-04-23T04:30:00Z"),
                processEvents: [],
              },
              {
                id: "turn-latest:assistant",
                role: "assistant",
                text: "latest stable answer",
                attachments: [],
                route: "chat",
                source: "runtime",
                error: false,
                status: "done",
                at: Date.parse("2026-04-23T04:30:02Z"),
                processEvents: [],
              },
            ],
            messagesLoaded: true,
            serverBacked: true,
            turnsPaging: {
              has_more_before: true,
              oldest_turn_id: "turn-latest",
              newest_turn_id: "turn-latest",
              next_before_turn_id: "turn-latest",
            },
          }],
        },
      }),
    );
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [{
              id: "c_longchat00000000",
              title: "Long stable chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              updated_at: "2026-04-23T04:30:00Z",
              last_output_at: "2026-04-23T04:30:00Z",
              turns_paging: {
                has_more_before: true,
                oldest_turn_id: "turn-latest",
                newest_turn_id: "turn-latest",
                next_before_turn_id: "turn-latest",
              },
              turns: [],
            }],
          };
        case "/api/chat/sessions/c_longchat00000000":
          return {
            session: {
              id: "c_longchat00000000",
              title: "Long stable chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              updated_at: "2026-04-23T04:30:00Z",
              last_output_at: "2026-04-23T04:30:00Z",
              turns_paging: {
                has_more_before: true,
                oldest_turn_id: "turn-latest",
                newest_turn_id: "turn-latest",
                next_before_turn_id: "turn-latest",
              },
              turns: [{
                id: "turn-latest",
                prompt: "latest prompt",
                status: "success",
                started_at: "2026-04-23T04:30:00Z",
                finished_at: "2026-04-23T04:30:02Z",
                final_output: "latest stable answer",
              }],
            },
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <MessageTextHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("latest stable answer"));
    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions"));
    apiClientMock.get.mockClear();

    await act(async () => {
      window.dispatchEvent(new PageTransitionEvent("pageshow", { persisted: true }));
      window.dispatchEvent(new Event("focus"));
      await Promise.resolve();
    });

    expect(apiClientMock.get).not.toHaveBeenCalledWith("/api/chat/sessions");
    expect(apiClientMock.get).not.toHaveBeenCalledWith("/api/chat/sessions/c_longchat00000000");
    expect(apiClientMock.get).not.toHaveBeenCalledWith(
      "/api/chat/sessions/c_longchat00000000?turn_before=turn-latest&turn_limit=20",
    );
    expect(screen.getByTestId("message-texts")).toHaveTextContent("latest stable answer");
  });

  it("reloads Chat detail when a newer list summary accidentally includes empty turns", async () => {
    window.sessionStorage.setItem(ACTIVE_SESSION_STORAGE_KEY, JSON.stringify({ chat: "c_stalechat0000000" }));
    window.localStorage.setItem(
      LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        cachedAt: Date.now(),
        activeSessionByRoute: { chat: "c_stalechat0000000" },
        sessionsByRoute: {
          chat: [{
            id: "c_stalechat0000000",
            status: "ready",
            title: "Stale chat",
            createdAt: Date.parse("2026-04-23T03:30:00Z"),
            updatedAt: Date.parse("2026-04-23T03:31:00Z"),
            lastOutputAt: Date.parse("2026-04-23T03:31:00Z"),
            activityAt: Date.parse("2026-04-23T03:31:00Z"),
            freshnessAt: Date.parse("2026-04-23T03:31:00Z"),
            detailFreshnessAt: Date.parse("2026-04-23T03:31:00Z"),
            pinned: false,
            targetID: "codex",
            targetName: "Codex",
            messages: [
              {
                id: "turn-old:user",
                role: "user",
                text: "old prompt",
                attachments: [],
                route: "chat",
                source: "runtime",
                error: false,
                status: "",
                at: Date.parse("2026-04-23T03:31:00Z"),
                processEvents: [],
              },
              {
                id: "turn-old:assistant",
                role: "assistant",
                text: "old answer",
                attachments: [],
                route: "chat",
                source: "runtime",
                error: false,
                status: "done",
                at: Date.parse("2026-04-23T03:31:01Z"),
                processEvents: [],
              },
            ],
            messagesLoaded: true,
            serverBacked: true,
            turnsPaging: { has_more_before: false },
          }],
        },
      }),
    );
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [{
              id: "c_stalechat0000000",
              title: "Stale chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              updated_at: "2026-04-23T03:32:00Z",
              last_output_at: "2026-04-23T03:32:00Z",
              activity_at: "2026-04-23T03:32:00Z",
              freshnessAt: Date.parse("2026-04-23T03:32:00Z"),
              turns: [],
            }],
          };
        case "/api/chat/sessions/c_stalechat0000000":
          return {
            session: {
              id: "c_stalechat0000000",
              title: "Stale chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              updated_at: "2026-04-23T03:32:00Z",
              last_output_at: "2026-04-23T03:32:00Z",
              activity_at: "2026-04-23T03:32:00Z",
              freshnessAt: Date.parse("2026-04-23T03:32:00Z"),
              turns_paging: { has_more_before: false },
              turns: [{
                id: "turn-new",
                prompt: "new prompt",
                status: "success",
                started_at: "2026-04-23T03:32:00Z",
                finished_at: "2026-04-23T03:32:01Z",
                final_output: "new answer",
              }],
            },
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <MessageTextHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions/c_stalechat0000000"));
    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("new answer"));
  });

  it("reloads Chat detail when a list summary omits freshnessAt for an existing cached session", async () => {
    window.sessionStorage.setItem(ACTIVE_SESSION_STORAGE_KEY, JSON.stringify({ chat: "c_stalechat0000000" }));
    window.localStorage.setItem(
      LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        cachedAt: Date.now(),
        activeSessionByRoute: { chat: "c_stalechat0000000" },
        sessionsByRoute: {
          chat: [{
            id: "c_stalechat0000000",
            status: "ready",
            title: "Stale chat",
            createdAt: Date.parse("2026-04-23T03:30:00Z"),
            updatedAt: Date.parse("2026-04-23T03:31:00Z"),
            lastOutputAt: Date.parse("2026-04-23T03:31:00Z"),
            activityAt: Date.parse("2026-04-23T03:31:00Z"),
            freshnessAt: Date.parse("2026-04-23T03:31:00Z"),
            detailFreshnessAt: Date.parse("2026-04-23T03:31:00Z"),
            pinned: false,
            targetID: "codex",
            targetName: "Codex",
            messages: [
              {
                id: "turn-old:user",
                role: "user",
                text: "old prompt",
                attachments: [],
                route: "chat",
                source: "runtime",
                error: false,
                status: "",
                at: Date.parse("2026-04-23T03:31:00Z"),
                processEvents: [],
              },
              {
                id: "turn-old:assistant",
                role: "assistant",
                text: "old answer",
                attachments: [],
                route: "chat",
                source: "runtime",
                error: false,
                status: "done",
                at: Date.parse("2026-04-23T03:31:01Z"),
                processEvents: [],
              },
            ],
            messagesLoaded: true,
            serverBacked: true,
            turnsPaging: { has_more_before: false },
          }],
        },
      }),
    );
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [{
              id: "c_stalechat0000000",
              title: "Stale chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              updated_at: "2026-04-23T03:32:00Z",
              last_output_at: "2026-04-23T03:32:00Z",
              activity_at: "2026-04-23T03:32:00Z",
            }],
          };
        case "/api/chat/sessions/c_stalechat0000000":
          return {
            session: {
              id: "c_stalechat0000000",
              title: "Stale chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              updated_at: "2026-04-23T03:32:00Z",
              last_output_at: "2026-04-23T03:32:00Z",
              activity_at: "2026-04-23T03:32:00Z",
              turns_paging: { has_more_before: false },
              turns: [{
                id: "turn-new",
                prompt: "new prompt",
                status: "success",
                started_at: "2026-04-23T03:32:00Z",
                finished_at: "2026-04-23T03:32:01Z",
                final_output: "new answer without summary freshnessAt",
              }],
            },
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <MessageTextHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions/c_stalechat0000000"));
    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("new answer without summary freshnessAt"));
  });

  it("does not hydrate expired Chat browser caches", async () => {
    const nowSpy = vi.spyOn(Date, "now").mockReturnValue(1000);
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [{
              id: "c_51jttwiv4yggqagk",
              title: "Expired chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns: chatTurnFixtures(1, "expired answer"),
            }],
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    const firstView = render(
      <ConversationRuntimeProvider route="chat" language="en">
        <ActiveSessionTitleHarness />
        <MessageTextHarness />
      </ConversationRuntimeProvider>,
    );
    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("expired answer 1"));
    firstView.unmount();
    window.sessionStorage.removeItem(ACTIVE_SESSION_SNAPSHOT_STORAGE_KEY);
    window.sessionStorage.removeItem(RECENT_SESSION_SNAPSHOT_STORAGE_KEY);

    nowSpy.mockReturnValue(1000 + (24 * 60 * 60 * 1000) + 1);
    const listRequest = deferred<{ items?: unknown[] }>();
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return listRequest.promise;
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <ActiveSessionTitleHarness />
        <MessageTextHarness />
      </ConversationRuntimeProvider>,
    );

    expect(screen.getByTestId("active-session-title")).toHaveTextContent("");
    expect(screen.getByTestId("message-texts")).not.toHaveTextContent("expired answer 1");

    listRequest.resolve({
      items: [{
        id: "c_51jttwiv4yggqagk",
        title: "Server chat after expiry",
        status: "ready",
        created_at: "2026-04-23T04:00:00Z",
        turns: chatTurnFixtures(1, "server answer after expiry"),
      }],
    });

    await waitFor(() => expect(screen.getByTestId("active-session-title")).toHaveTextContent("Server chat after expiry"));
    expect(screen.getByTestId("message-texts")).toHaveTextContent("server answer after expiry 1");
  });

  it("does not create local blank Chat sessions before the user starts a ChatRuntime-backed session", async () => {
    window.sessionStorage.clear();
    apiClientMock.get.mockImplementation(async () => ({ items: [] }));

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <SessionListHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions"));
    expect(screen.getByTestId("sessions")).toHaveTextContent("");

    fireEvent.click(screen.getByRole("button", { name: "new session" }));

    await waitFor(() => expect(apiClientMock.post).toHaveBeenCalledWith("/api/chat/sessions", {}));
    await waitFor(() => expect(screen.getByTestId("sessions")).toHaveTextContent(/^New:/));
  });

  it("uses the underlying ChatRuntime session title for first-turn Chat input", async () => {
    window.sessionStorage.clear();
    const inputRequest = deferred<Record<string, unknown>>();
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });
    apiClientMock.post.mockImplementation(async (path: string) => {
      if (path === "/api/chat/sessions") {
        return {
          session: {
            id: "c_underlyingtitle0",
            title: "Underlying runtime title",
            status: "ready",
            created_at: "2026-04-23T04:00:00Z",
            turns: [],
          },
        };
      }
      if (path === "/api/chat/sessions/c_underlyingtitle0/input") {
        return inputRequest.promise;
      }
      return {};
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <ActiveSessionTitleHarness />
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions"));

    fireEvent.click(screen.getByRole("button", { name: "send" }));

    await waitFor(() => expect(apiClientMock.post).toHaveBeenCalledWith("/api/chat/sessions", {}));
    await waitFor(() => expect(screen.getByTestId("composer-submitting")).toHaveTextContent("true"));
    expect(screen.getByTestId("user-text")).toHaveTextContent("");
    expect(screen.getByTestId("active-session-title")).toHaveTextContent("Underlying runtime title");

    inputRequest.resolve({
      session: {
        id: "c_underlyingtitle0",
        title: "Underlying runtime title",
        status: "ready",
        created_at: "2026-04-23T04:00:00Z",
        turns: [
          {
            id: "turn-underlying-title",
            prompt: "Inspect this image",
            status: "success",
            started_at: "2026-04-23T04:00:01Z",
            finished_at: "2026-04-23T04:00:02Z",
            final_output: "Done",
          },
        ],
      },
    });
    await waitFor(() => expect(screen.getByTestId("assistant-text")).toHaveTextContent("Done"));
    expect(screen.getByTestId("user-text")).toHaveTextContent("Inspect this image");
  });

  it("loads Chat sessions through the isolated chat ChatRuntime scope", async () => {
    window.sessionStorage.clear();
    apiClientMock.get.mockImplementation(async () => ({ items: [] }));

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <SessionListHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions"));
  });

  it("creates ChatRuntime-backed Chat sessions when New is pressed repeatedly", async () => {
    window.sessionStorage.clear();
    apiClientMock.get.mockImplementation(async () => ({ items: [] }));

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <SessionListHarness />
      </ConversationRuntimeProvider>,
    );

    fireEvent.click(screen.getByRole("button", { name: "new session" }));
    fireEvent.click(screen.getByRole("button", { name: "new session" }));

    await waitFor(() => expect(apiClientMock.post).toHaveBeenCalledWith("/api/chat/sessions", {}));
  });

  it("updates Chat session pin state through the session history pin endpoint", async () => {
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [
              {
                id: "c_51jttwiv4yggqagk",
                title: "Pinned session",
                created_at: "2026-04-23T03:30:00Z",
                target_type: "model",
                target_id: "raw-model",
                target_name: "Raw Model",
                pinned: true,
              },
            ],
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });
    apiClientMock.post.mockResolvedValueOnce({ session_id: "c_51jttwiv4yggqagk", pinned: false });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <SessionListHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("sessions")).toHaveTextContent("Pinned session"));
    expect(screen.getByTestId("sessions")).toHaveTextContent("pinned");

    fireEvent.click(screen.getByRole("button", { name: "unpin active" }));

    await waitFor(() => {
      expect(apiClientMock.post).toHaveBeenCalledWith(
        "/api/chat/sessions/c_51jttwiv4yggqagk/pin",
        { pinned: false },
      );
    });
    expect(screen.getByTestId("sessions")).toHaveTextContent("unpinned");
  });

  it("keeps the confirmed pin state when the pin request fails", async () => {
    apiClientMock.get.mockImplementation(async (path: string) => {
      if (path === "/api/chat/sessions") {
        return {
          items: [{
            id: "c_51jttwiv4yggqagk",
            title: "Pinned session",
            created_at: "2026-04-23T03:30:00Z",
            pinned: true,
          }],
        };
      }
      return { items: [] };
    });
    apiClientMock.post.mockRejectedValueOnce(new Error("pin failed"));

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <SessionListHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("sessions")).toHaveTextContent("Pinned session:pinned"));
    fireEvent.click(screen.getByRole("button", { name: "unpin active" }));
    await waitFor(() => expect(apiClientMock.post).toHaveBeenCalledWith(
      "/api/chat/sessions/c_51jttwiv4yggqagk/pin",
      { pinned: false },
    ));
    expect(screen.getByTestId("sessions")).toHaveTextContent("Pinned session:pinned");
  });

  it("moves pinned Chat sessions ahead of newer unpinned sessions", async () => {
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [
              {
                id: "c_newerchat0000000",
                title: "Newer session",
                created_at: "2026-04-23T04:30:00Z",
                target_type: "model",
                target_id: "raw-model",
                target_name: "Raw Model",
                pinned: false,
              },
              {
                id: "c_olderchat0000000",
                title: "Older session",
                created_at: "2026-04-23T03:30:00Z",
                target_type: "model",
                target_id: "raw-model",
                target_name: "Raw Model",
                pinned: false,
              },
            ],
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });
    apiClientMock.post.mockResolvedValueOnce({ session_id: "c_olderchat0000000", pinned: true });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <SessionListHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("sessions")).toHaveTextContent(/^Newer session:/));

    fireEvent.click(screen.getByRole("button", { name: "pin older" }));

    await waitFor(() => {
      expect(apiClientMock.post).toHaveBeenCalledWith(
        "/api/chat/sessions/c_olderchat0000000/pin",
        { pinned: true },
      );
    });
    expect(screen.getByTestId("sessions")).toHaveTextContent(/^Older session:pinned\|Newer session:unpinned$/);
  });

  it("pins a newly created ChatRuntime-backed Chat session", async () => {
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });
    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <SessionListHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions"));

    fireEvent.click(screen.getByRole("button", { name: "new session" }));
    await waitFor(() => expect(screen.getByTestId("sessions")).toHaveTextContent(/^New:/));

    fireEvent.click(screen.getByRole("button", { name: "pin active" }));

    await waitFor(() => expect(apiClientMock.post).toHaveBeenCalledWith(
      "/api/chat/sessions/c_newchat000000000/pin",
      { pinned: true },
    ));
    expect(screen.getByTestId("sessions")).toHaveTextContent(/^New:pinned$/);
  });

  it("loads Chat sessions from the Chat route and hydrates them as Chat sessions", async () => {
    window.sessionStorage.setItem(
      ACTIVE_SESSION_STORAGE_KEY,
      JSON.stringify({ chat: "c_skillsession0000" }),
    );
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [
              {
                id: "c_skillsession0000",
                title: "Travel Plan",
                created_at: "2026-04-23T09:00:00Z",
                target_type: "skill",
                target_id: "travel",
                target_name: "Travel Skill",
                messages: [],
              },
            ],
          };
        case "/api/chat/sessions/c_skillsession0000":
          return {
            session: {
              id: "c_skillsession0000",
              title: "Travel Plan",
              created_at: "2026-04-23T09:00:00Z",
              target_type: "skill",
              target_id: "travel",
              target_name: "Travel Skill",
              messages: [
                { id: "m-1", role: "user", content: "Plan Wuhan", created_at: "2026-04-23T09:00:00Z" },
              ],
            },
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <SessionListHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("sessions")).toHaveTextContent("Travel Plan"));
    expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions");
  });

  it("hydrates Chat detail turns whose ids arrive as numbers", async () => {
    window.sessionStorage.setItem(
      ACTIVE_SESSION_STORAGE_KEY,
      JSON.stringify({ chat: "c_51jttwiv4yggqagk" }),
    );
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [
              {
                id: "c_51jttwiv4yggqagk",
                title: "Smoke compact id",
                status: "running",
                created_at: 1783435585330,
                updated_at: 1783437647729,
                turns: [],
              },
            ],
          };
        case "/api/chat/sessions/c_51jttwiv4yggqagk":
          return {
            session: {
              id: "c_51jttwiv4yggqagk",
              title: "Smoke compact id",
              status: "running",
              created_at: 1783435585330,
              updated_at: 1783437647729,
              turns: [
                {
                  id: 1,
                  prompt: "成都旅游攻略",
                  status: "running",
                  started_at: 1783437617847,
                  runtime_trace_events: [
                    {
                      id: 1,
                      kind: "important_text",
                      status: "completed",
                      text: "我先确认你的出行环境要求。",
                      created_at: 1783437627484,
                    },
                  ],
                },
              ],
              turns_paging: { has_more_before: false },
            },
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("user-text")).toHaveTextContent("成都旅游攻略"));
    expect(screen.getByTestId("assistant-process-count")).toHaveTextContent("1");
    expect(screen.getByTestId("assistant-process-ids")).toHaveTextContent("1");
  });

  it("opens the latest ChatRuntime-backed Chat session when the route has no explicit session query", async () => {
    window.history.replaceState({}, "", "/chat");
    window.sessionStorage.setItem(
      ACTIVE_SESSION_STORAGE_KEY,
      JSON.stringify({ chat: "c_olderchatsession" }),
    );
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [
              {
                id: "c_latestchat000000",
                title: "Latest chat",
                created_at: "2026-06-11T05:40:00Z",
                target_type: "model",
                target_id: "raw-model",
                target_name: "Raw Model",
                messages: [],
              },
              {
                id: "c_olderchatsession",
                title: "Older chat",
                created_at: "2026-06-10T05:40:00Z",
                target_type: "model",
                target_id: "raw-model",
                target_name: "Raw Model",
                messages: [],
              },
            ],
          };
        case "/api/chat/sessions/c_latestchat000000":
          return {
            session: {
              id: "c_latestchat000000",
              title: "Latest chat",
              created_at: "2026-06-11T05:40:00Z",
              target_type: "model",
              target_id: "raw-model",
              target_name: "Raw Model",
              messages: [],
            },
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <ActiveSessionTitleHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("active-session-title")).toHaveTextContent("Latest chat"));
    expect(window.location.search).toBe("");
  });

  it("falls back to the stored active Chat session when the session list cannot provide a latest session", async () => {
    window.history.replaceState({}, "", "/chat");
    window.sessionStorage.setItem(
      ACTIVE_SESSION_STORAGE_KEY,
      JSON.stringify({ chat: "c_olderchatsession" }),
    );
    window.localStorage.setItem(
      LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        cachedAt: Date.now(),
        activeSessionByRoute: { chat: "c_olderchatsession" },
        sessionsByRoute: {
          chat: [chatSessionFixture({
            id: "c_olderchatsession",
            title: "Older chat",
            createdAt: Date.parse("2026-06-10T05:40:00Z"),
            activityAt: Date.parse("2026-06-10T05:40:00Z"),
            freshnessAt: Date.parse("2026-06-10T05:40:00Z"),
            detailFreshnessAt: Date.parse("2026-06-10T05:40:00Z"),
            messagesLoaded: true,
            messages: [{
              id: "turn-old:assistant",
              role: "assistant",
              text: "old answer",
              at: Date.parse("2026-06-10T05:40:01Z"),
              route: "chat",
              source: "runtime",
              status: "done",
              error: false,
              attachments: [],
              processEvents: [],
            }],
          })],
        },
      }),
    );
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return { items: [] };
        case "/api/chat/sessions/c_olderchatsession":
          return {
            session: {
              id: "c_olderchatsession",
              title: "Older chat",
              created_at: "2026-06-10T05:40:00Z",
              target_type: "model",
              target_id: "raw-model",
              target_name: "Raw Model",
              turns: [{
                id: "turn-old",
                prompt: "old prompt",
                status: "success",
                started_at: "2026-06-10T05:40:00Z",
                finished_at: "2026-06-10T05:40:01Z",
                final_output: "old answer",
              }],
            },
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <ActiveSessionTitleHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("active-session-title")).toHaveTextContent("Older chat"));
    expect(window.location.search).toBe("");
  });

  it("clears legacy local chat snapshots without restoring them as Chat sessions", async () => {
    window.sessionStorage.clear();
    window.sessionStorage.setItem(
      ACTIVE_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        "chat": {
          id: "c_legacy0000000000",
          title: "Legacy local chat",
          createdAt: Date.parse("2026-04-23T09:00:00Z"),
          target: { type: "skill", id: "travel", name: "Travel Skill" },
          messages: [],
        },
      }),
    );
    window.sessionStorage.setItem(
      RECENT_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        "chat": [
          {
            id: "c_recent0000000000",
            title: "Recent local chat",
            createdAt: Date.parse("2026-04-22T09:00:00Z"),
            target: { type: "skill", id: "writing", name: "Writing Skill" },
            messages: [],
          },
        ],
      }),
    );
    apiClientMock.get.mockImplementation(async (path: string) => {
      if (path === "/api/chat/sessions") {
        return { items: [] };
      }
      return { items: [] };
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <SessionListHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions"));
    expect(screen.getByTestId("sessions")).not.toHaveTextContent("Legacy local chat");
    expect(screen.getByTestId("sessions")).not.toHaveTextContent("Recent local chat");
    expect(window.sessionStorage.getItem(ACTIVE_SESSION_SNAPSHOT_STORAGE_KEY)).toBeNull();
    expect(window.sessionStorage.getItem(RECENT_SESSION_SNAPSHOT_STORAGE_KEY)).toBeNull();
    expect(window.localStorage.getItem(LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY)).not.toContain("Legacy local chat");
    expect(window.localStorage.getItem(LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY)).not.toContain("Recent local chat");
  });

  it("uploads draft images into the active Chat session workspace", async () => {
    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions"));
    fireEvent.click(screen.getByRole("button", { name: "attach" }));

    await waitFor(() => expect(apiClientMock.post).toHaveBeenCalledWith(
      "/api/chat/sessions/c_51jttwiv4yggqagk/attachments",
      {
        attachments: [
          {
            name: "trace.png",
            content_type: "image/png",
            data_url: "data:image/png;base64,ZmFrZS1pbWFnZS1kYXRh",
            preview_data_url: "data:image/webp;base64,c21hbGwtcHJldmlldw==",
          },
        ],
      },
    ));
  });

	it("keeps repository selection client-side until send and submits a structured reference", async () => {
		apiClientMock.get.mockImplementation(async (path: string) => {
			if (path === "/api/chat/repositories?query=alter0") {
				return {
					repositories: [{
						id: "123456789",
						full_name: "owner/repository",
						private: true,
						default_branch: "main",
						updated_at: "2026-07-11T10:00:00Z",
					}],
				};
			}
			if (path === "/api/chat/sessions") {
				return {
					items: [{
						id: "c_51jttwiv4yggqagk",
						title: "Image session",
						status: "ready",
						created_at: "2026-04-23T03:30:00Z",
						turns: [],
					}],
				};
			}
			return { items: [] };
		});
		apiClientMock.post.mockImplementation(async (path: string, body?: Record<string, unknown>) => {
			if (path === "/api/chat/sessions/c_51jttwiv4yggqagk/input") {
				return {
					session: {
						id: "c_51jttwiv4yggqagk",
						title: "Update retry behavior",
						status: "busy",
						created_at: "2026-04-23T03:30:00Z",
						repository: {
							provider: "github",
							id: "123456789",
							full_name: "owner/repository",
							private: true,
							default_branch: "main",
							status: "preparing",
							workspace_path: "repo",
						},
						turns: [{
							id: "turn-1",
							prompt: typeof body?.input === "string" ? body.input : "Update retry behavior",
							status: "running",
						}],
					},
				};
			}
			return {};
		});

		render(
			<ConversationRuntimeProvider route="chat" language="en">
				<RepositoryBindingHarness />
			</ConversationRuntimeProvider>,
		);

		await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions"));
		fireEvent.click(screen.getByRole("button", { name: "list repositories" }));
		await waitFor(() => expect(screen.getByTestId("listed-repositories")).toHaveTextContent("owner/repository"));
		fireEvent.click(screen.getByRole("button", { name: "select repository" }));
		expect(screen.getByTestId("draft-repository")).toHaveTextContent("owner/repository");
		expect(apiClientMock.post).not.toHaveBeenCalledWith("/api/chat/sessions", expect.anything());

		fireEvent.click(screen.getByRole("button", { name: "send repository prompt" }));
		await waitFor(() => expect(apiClientMock.post).toHaveBeenCalledWith(
			"/api/chat/sessions/c_51jttwiv4yggqagk/input",
			expect.objectContaining({
				input: "Update retry behavior",
				repository: {
					provider: "github",
					id: "123456789",
					full_name: "owner/repository",
				},
			}),
		));
		await waitFor(() => expect(screen.getByTestId("bound-repository")).toHaveTextContent("owner/repository"));
	});

  it("keeps submission state local without changing the session before backend acknowledgement", async () => {
    vi.stubGlobal("fetch", vi.fn());
    apiClientMock.post.mockImplementation(async (path: string) => {
      if (path === "/api/chat/sessions/c_51jttwiv4yggqagk/input") {
        return new Promise(() => undefined);
      }
      return {};
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions"));
    fireEvent.click(screen.getByRole("button", { name: "send" }));

    await waitFor(() => expect(screen.getByTestId("composer-submitting")).toHaveTextContent("true"));
    expect(screen.getByTestId("active-session-status")).toHaveTextContent("ready");
    expect(screen.getByTestId("user-text")).toHaveTextContent("");
    expect(screen.getByTestId("assistant-text")).toHaveTextContent("");
    expect(screen.getByTestId("assistant-process-count")).toHaveTextContent("0");
    expect(screen.getByTestId("assistant-process-status")).toHaveTextContent("");
    expect(window.localStorage.getItem(LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY) || "").not.toContain("local_running");
    expect(apiClientMock.post).toHaveBeenCalledWith("/api/chat/sessions/c_51jttwiv4yggqagk/input", expect.any(Object));
    expect(fetch).not.toHaveBeenCalled();
  });

  it("waits for backend acknowledgement before adding a submitted prompt to session history", async () => {
    const inputResponse = deferred<unknown>();
    apiClientMock.post.mockImplementation(async (path: string) => {
      if (path === "/api/chat/sessions/c_51jttwiv4yggqagk/input") {
        return inputResponse.promise;
      }
      return {};
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions"));
    fireEvent.click(screen.getByRole("button", { name: "send" }));

    await waitFor(() => expect(screen.getByTestId("composer-submitting")).toHaveTextContent("true"));
    expect(screen.getByTestId("user-text")).toHaveTextContent("");
    expect(screen.getByTestId("active-session-status")).toHaveTextContent("ready");
    expect(window.localStorage.getItem(LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY) || "").not.toContain("local_running");

    inputResponse.resolve({
      session: {
        id: "c_51jttwiv4yggqagk",
        title: "Inspect this image",
        status: "ready",
        created_at: "2026-04-23T03:30:00Z",
        turns: [{
          id: "turn-cached-optimistic",
          prompt: "Inspect this image",
          status: "success",
          started_at: "2026-04-23T04:00:00Z",
          finished_at: "2026-04-23T04:00:02Z",
          final_output: "Stored response",
        }],
      },
    });

    await waitFor(() => expect(screen.getByTestId("assistant-text")).toHaveTextContent("Stored response"));
    expect(screen.getByTestId("user-text")).toHaveTextContent("Inspect this image");
    expect(screen.getByTestId("composer-submitting")).toHaveTextContent("false");
  });

  it("treats a transport failure as a weak composer notice without mutating the session", async () => {
    apiClientMock.post.mockImplementation(async (path: string) => {
      if (path === "/api/chat/sessions/c_51jttwiv4yggqagk/input") {
        throw new Error("Load failed");
      }
      return {};
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions"));
    fireEvent.click(screen.getByRole("button", { name: "send" }));

    await waitFor(() => expect(screen.getByTestId("composer-request-notice")).toHaveTextContent("Network request interrupted"));
    expect(screen.getByTestId("composer-submitting")).toHaveTextContent("false");
    expect(screen.getByTestId("active-session-status")).toHaveTextContent("ready");
    expect(screen.getByTestId("user-text")).toHaveTextContent("");
    expect(screen.getByTestId("assistant-texts")).not.toHaveTextContent("Load failed");
    expect(window.localStorage.getItem(LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY) || "").not.toContain("Load failed");
    expect(window.localStorage.getItem(LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY) || "").not.toContain("local_running");
  });

  it("shows a backend rejection message only in the composer without mutating the session", async () => {
    apiClientMock.post.mockImplementation(async (path: string) => {
      if (path === "/api/chat/sessions/c_51jttwiv4yggqagk/input") {
        throw Object.assign(new Error("Prompt rejected by policy"), { status: 422, code: "policy_rejected" });
      }
      return {};
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions"));
    fireEvent.click(screen.getByRole("button", { name: "send" }));

    await waitFor(() => expect(screen.getByTestId("composer-request-notice")).toHaveTextContent("Prompt rejected by policy"));
    expect(screen.getByTestId("active-session-status")).toHaveTextContent("ready");
    expect(screen.getByTestId("assistant-texts")).toHaveTextContent("");
    expect(window.localStorage.getItem(LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY) || "").not.toContain("Prompt rejected by policy");
  });

  it("keeps a first-message draft visible after session creation succeeds but input transport fails", async () => {
    window.sessionStorage.clear();
    apiClientMock.get.mockImplementation(async () => ({ items: [] }));
    apiClientMock.post.mockImplementation(async (path: string) => {
      if (path === "/api/chat/sessions") {
        return {
          session: {
            id: "c_createdbeforerr0",
            title: "New",
            status: "ready",
            created_at: "2026-04-23T04:00:00Z",
            turns: [],
          },
        };
      }
      if (path === "/api/chat/sessions/c_createdbeforerr0/input") {
        throw new Error("Load failed");
      }
      return {};
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions"));
    fireEvent.click(screen.getByRole("button", { name: "set draft" }));
    fireEvent.click(screen.getByRole("button", { name: "send draft" }));

    await waitFor(() => expect(screen.getByTestId("composer-request-notice")).toHaveTextContent("Network request interrupted"));
    expect(screen.getByTestId("composer-draft")).toHaveTextContent("Retry this prompt");
    expect(screen.getByTestId("active-session-status")).toHaveTextContent("ready");
    expect(screen.getByTestId("user-text")).toHaveTextContent("");
  });

  it("compacts the queued optimistic Chat user message when the input response only confirms a busy turn", async () => {
    apiClientMock.post.mockImplementation(async (path: string) => {
      if (path === "/api/chat/sessions/c_51jttwiv4yggqagk/input") {
        return {
          session: {
            id: "c_51jttwiv4yggqagk",
            title: "New prompt",
            status: "busy",
            created_at: "2026-04-23T03:30:00Z",
            turns: [{
              id: "turn-busy",
              prompt: "new prompt",
              status: "running",
              started_at: "2026-04-23T04:00:00Z",
              final_output: "",
              runtime_trace_events: [{
                id: "event-busy",
                turn_id: "turn-busy",
                seq: 1,
                source: "adapter",
                provider: { engine: "codex", adapter: "codex_cli_json" },
                role: "assistant",
                kind: "assistant_commentary",
                lifecycle: "running",
                status: "running",
                title: "Thinking",
                summary: "Working",
                blocks: [],
                visibility: "collapsed",
              }],
            }],
          },
        };
      }
      return {};
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <SendMessageTextHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions"));
    fireEvent.click(screen.getByRole("button", { name: "send prompt" }));

    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent(/^new prompt\|$/));
    const texts = screen.getByTestId("message-texts").textContent || "";
    expect((texts.match(/new prompt/g) || []).length).toBe(1);
  });

  it("appends a returned new Chat turn without replacing loaded history", async () => {
    const existingTurns = chatTurnFixtures(3, "existing answer");
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [{
              id: "c_51jttwiv4yggqagk",
              title: "History session",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns: existingTurns,
            }],
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });
    apiClientMock.post.mockImplementation(async (path: string) => {
      if (path === "/api/chat/sessions/c_51jttwiv4yggqagk/input") {
        return {
          session: {
            id: "c_51jttwiv4yggqagk",
            title: "History session",
            status: "ready",
            created_at: "2026-04-23T03:30:00Z",
            turns: [{
              id: "turn-new",
              prompt: "new prompt",
              status: "success",
              started_at: "2026-04-23T04:00:00Z",
              finished_at: "2026-04-23T04:00:02Z",
              final_output: "new answer",
            }],
          },
        };
      }
      return {};
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <SendMessageTextHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("existing answer 3"));

    fireEvent.click(screen.getByRole("button", { name: "send prompt" }));

    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("new answer"));
    expect(screen.getByTestId("message-texts")).toHaveTextContent("existing answer 1");
    expect(screen.getByTestId("message-texts")).toHaveTextContent("existing answer 3");
  });

  it("merges the final earlier Chat history page without dropping the latest page", async () => {
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [{
              id: "c_51jttwiv4yggqagk",
              title: "Progressive history",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns: [{
                id: "turn-3",
                prompt: "latest prompt",
                status: "success",
                started_at: "2026-04-23T03:03:00Z",
                finished_at: "2026-04-23T03:03:02Z",
                final_output: "latest answer",
              }],
            }],
          };
        case "/api/chat/sessions/c_51jttwiv4yggqagk":
          return {
            session: {
              id: "c_51jttwiv4yggqagk",
              title: "Progressive history",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns_paging: {
                has_more_before: false,
                oldest_turn_id: "turn-1",
                newest_turn_id: "turn-2",
              },
              turns: [
                {
                  id: "turn-1",
                  prompt: "older prompt",
                  status: "success",
                  started_at: "2026-04-23T03:01:00Z",
                  finished_at: "2026-04-23T03:01:02Z",
                  final_output: "older answer",
                },
                {
                  id: "turn-2",
                  prompt: "middle prompt",
                  status: "success",
                  started_at: "2026-04-23T03:02:00Z",
                  finished_at: "2026-04-23T03:02:02Z",
                  final_output: "middle answer",
                },
              ],
            },
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <MessageTextHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("latest answer"));

    fireEvent.click(screen.getByRole("button", { name: "refresh active" }));

    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("older answer"));
    expect(screen.getByTestId("message-texts")).toHaveTextContent("middle answer");
    expect(screen.getByTestId("message-texts")).toHaveTextContent("latest answer");
  });

  it("keeps the Chat composer stable when explicitly loading earlier history", async () => {
    const earlierPage = deferred<{
      session: {
        id: string;
        title: string;
        status: string;
        created_at: string;
        turns_paging: {
          has_more_before: boolean;
          oldest_turn_id: string;
          newest_turn_id: string;
        };
        turns: Array<{
          id: string;
          prompt: string;
          status: string;
          started_at: string;
          finished_at: string;
          final_output: string;
        }>;
      };
    }>();
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [{
              id: "c_51jttwiv4yggqagk",
              title: "Progressive history",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns_paging: {
                has_more_before: true,
                oldest_turn_id: "turn-3",
                newest_turn_id: "turn-3",
                next_before_turn_id: "turn-3",
              },
              turns: [{
                id: "turn-3",
                prompt: "latest prompt",
                status: "success",
                started_at: "2026-04-23T03:03:00Z",
                finished_at: "2026-04-23T03:03:02Z",
                final_output: "latest answer",
              }],
            }],
          };
        case "/api/chat/sessions/c_51jttwiv4yggqagk?turn_before=turn-3&turn_limit=20":
          return earlierPage.promise;
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <MessageTextHarness />
        <ComposerRenderProbe />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("latest answer"));
    const renderCountBeforeHistoryMerge = Number(screen.getByTestId("composer-render-count").textContent?.split(":")[0] || "0");

    fireEvent.click(screen.getByRole("button", { name: "load earlier history" }));

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith(
      "/api/chat/sessions/c_51jttwiv4yggqagk?turn_before=turn-3&turn_limit=20",
    ));
    await act(async () => {
      earlierPage.resolve({
        session: {
          id: "c_51jttwiv4yggqagk",
          title: "Progressive history",
          status: "ready",
          created_at: "2026-04-23T03:30:00Z",
          turns_paging: {
            has_more_before: false,
            oldest_turn_id: "turn-1",
            newest_turn_id: "turn-2",
          },
          turns: [
            {
              id: "turn-1",
              prompt: "older prompt",
              status: "success",
              started_at: "2026-04-23T03:01:00Z",
              finished_at: "2026-04-23T03:01:02Z",
              final_output: "older answer",
            },
            {
              id: "turn-2",
              prompt: "middle prompt",
              status: "success",
              started_at: "2026-04-23T03:02:00Z",
              finished_at: "2026-04-23T03:02:02Z",
              final_output: "middle answer",
            },
          ],
        },
      });
      await earlierPage.promise;
    });

    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("older answer"));
    expect(screen.getByTestId("message-texts")).toHaveTextContent("middle answer");
    expect(screen.getByTestId("message-texts")).toHaveTextContent("latest answer");
    expect(Number(screen.getByTestId("composer-render-count").textContent?.split(":")[0] || "0"))
      .toBe(renderCountBeforeHistoryMerge);
  });

  it("appends a returned user-only Chat turn without replacing loaded history", async () => {
    const existingTurns = chatTurnFixtures(3, "existing answer");
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [{
              id: "c_51jttwiv4yggqagk",
              title: "History session",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns: existingTurns,
            }],
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });
    apiClientMock.post.mockImplementation(async (path: string) => {
      if (path === "/api/chat/sessions/c_51jttwiv4yggqagk/input") {
        return {
          session: {
            id: "c_51jttwiv4yggqagk",
            title: "History session",
            status: "busy",
            created_at: "2026-04-23T03:30:00Z",
            turns: [{
              id: "turn-user-only",
              prompt: "new prompt",
              status: "running",
              started_at: "2026-04-23T04:00:00Z",
              final_output: "",
            }],
          },
        };
      }
      return {};
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <SendMessageTextHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("existing answer 3"));

    fireEvent.click(screen.getByRole("button", { name: "send prompt" }));

    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("new prompt"));
    expect(screen.getByTestId("message-texts")).toHaveTextContent("existing answer 1");
    expect(screen.getByTestId("message-texts")).toHaveTextContent("existing answer 3");
  });

  it("keeps polling a busy ChatRuntime-backed Chat session until its final output is restored", async () => {
    vi.stubGlobal("fetch", vi.fn());
    const EventSourceMock = vi.fn(() => {
      throw new Error("EventSource should not be used for conversation updates");
    });
    vi.stubGlobal("EventSource", EventSourceMock);
    let inputAccepted = false;
    let updateCallsAfterInput = 0;
    const updateRequestBodies: Record<string, unknown>[] = [];
    apiClientMock.post.mockImplementation(async (path: string, body?: Record<string, unknown>) => {
      if (path === "/api/chat/sessions/updates") {
        updateRequestBodies.push(body || {});
        if (inputAccepted) {
          updateCallsAfterInput += 1;
        }
        return {
          latest_update_id: updateCallsAfterInput * 2,
          resync_required: false,
          updates: inputAccepted ? [
            {
              update_id: updateCallsAfterInput * 2 - 1,
              session_id: "c_51jttwiv4yggqagk",
              turn_id: "turn-running",
              type: "turn.event.appended",
              created_at: "2026-04-23T03:31:01Z",
              runtime_event: {
                id: updateCallsAfterInput * 2 + 100,
                kind: "reasoning",
                status: "completed",
                text: "Validated update protocol.",
                detail_available: true,
                created_at: "2026-04-23T03:31:01Z",
              },
              payload: {
                session: {
                  id: "c_51jttwiv4yggqagk",
                  title: "Image session",
                  status: "local_running",
                  created_at: "2026-04-23T03:30:00Z",
                },
                turn: {
                  id: "turn-running",
                  prompt: "Inspect this image",
                  status: "running",
                  started_at: "2026-04-23T03:31:00Z",
                },
              },
            },
            {
              update_id: updateCallsAfterInput * 2,
              session_id: "c_51jttwiv4yggqagk",
              turn_id: "turn-running",
              type: "turn.completed",
              created_at: "2026-04-23T03:31:03Z",
              payload: {
                session: {
                  id: "c_51jttwiv4yggqagk",
                  title: "Image session",
                  status: "ready",
                  created_at: "2026-04-23T03:30:00Z",
                },
                turn: {
                  id: "turn-running",
                  prompt: "Inspect this image",
                  status: "success",
                  started_at: "2026-04-23T03:31:00Z",
                  finished_at: "2026-04-23T03:31:03Z",
                  final_output: "Restored final output",
                },
              },
            },
          ] : [],
        };
      }
      if (path === "/api/chat/sessions/c_51jttwiv4yggqagk/input") {
        inputAccepted = true;
        return {
          session: {
            id: "c_51jttwiv4yggqagk",
            title: "Image session",
            status: "local_running",
            created_at: "2026-04-23T03:30:00Z",
            turns: [
              {
                id: "turn-running",
                prompt: typeof body?.input === "string" ? body.input : "Inspect this image",
                status: "running",
                started_at: "2026-04-23T03:31:00Z",
                runtime_trace_events: [
                  {
                    id: 1,
                    kind: "reasoning",
                    status: "completed",
                    blocks: [],
                  },
                  {
                    id: 2,
                    kind: "reasoning",
                    status: "completed",
                    blocks: [],
                  },
                ],
              },
            ],
          },
        };
      }
      return {};
    });
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [
              {
                id: "c_51jttwiv4yggqagk",
                title: "Image session",
                status: inputAccepted ? "busy" : "ready",
                created_at: "2026-04-23T03:30:00Z",
                turns: [],
              },
            ],
          };
        case "/api/chat/sessions/c_51jttwiv4yggqagk":
          return {
            session: inputAccepted
              ? updateCallsAfterInput > 0
                ? {
                  id: "c_51jttwiv4yggqagk",
                  title: "Image session",
                  status: "ready",
                  created_at: "2026-04-23T03:30:00Z",
                  turns: [
                    {
                      id: "turn-running",
                      prompt: "Inspect this image",
                      status: "success",
                      started_at: "2026-04-23T03:31:00Z",
                      finished_at: "2026-04-23T03:31:03Z",
                      final_output: "Restored final output",
                    },
                  ],
                }
                : {
                    id: "c_51jttwiv4yggqagk",
                    title: "Image session",
                    status: "busy",
                    created_at: "2026-04-23T03:30:00Z",
                    turns: [
                      {
                        id: "turn-running",
                        prompt: "Inspect this image",
                        status: "running",
                        started_at: "2026-04-23T03:31:00Z",
                      },
                    ],
                  }
              : {
                  id: "c_51jttwiv4yggqagk",
                  title: "Image session",
                  status: "ready",
                  created_at: "2026-04-23T03:30:00Z",
                  turns: [],
                },
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions"));
    vi.useFakeTimers();
    fireEvent.click(screen.getByRole("button", { name: "send" }));

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });
    expect(screen.getByTestId("active-session-status")).toHaveTextContent("local_running");

    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
      await Promise.resolve();
    });

    expect(EventSourceMock).not.toHaveBeenCalled();
    expect(apiClientMock.post).toHaveBeenCalledWith(
      "/api/chat/sessions/updates",
      expect.objectContaining({
        after_update_id: "0",
        limit: 50,
        byte_limit: 1048576,
        visible_event_kinds: ["important_text", "reasoning"],
      }),
    );
    expect(updateRequestBodies[0]).not.toHaveProperty("since_event_id");
    expect(updateRequestBodies).toContainEqual(expect.objectContaining({
      sessions: expect.arrayContaining([
        expect.objectContaining({
          id: "c_51jttwiv4yggqagk",
        }),
      ]),
    }));
    expect(screen.getByTestId("assistant-text")).toHaveTextContent("Restored final output");
    expect(screen.getByTestId("assistant-process-count")).toHaveTextContent("3");
    expect(screen.getByTestId("assistant-process-ids")).toHaveTextContent("1|2|102");
    expect(screen.getByTestId("active-session-status")).toHaveTextContent("ready");
    vi.useRealTimers();
  });

  it("falls back to Chat session detail when busy update polling returns no events", async () => {
    vi.stubGlobal("fetch", vi.fn());
    let inputAccepted = false;
    let updateCallsAfterInput = 0;
    let detailReadsAfterInput = 0;
    apiClientMock.post.mockImplementation(async (path: string, body?: Record<string, unknown>) => {
      if (path === "/api/chat/sessions/updates") {
        if (inputAccepted) {
          updateCallsAfterInput += 1;
        }
        return {
          latest_update_id: updateCallsAfterInput,
          resync_required: false,
          updates: [],
        };
      }
      if (path === "/api/chat/sessions/c_51jttwiv4yggqagk/input") {
        inputAccepted = true;
        return {
          session: {
            id: "c_51jttwiv4yggqagk",
            title: "Fallback session",
            status: "busy",
            created_at: "2026-04-23T03:30:00Z",
            turns: [
              {
                id: "turn-running",
                prompt: typeof body?.input === "string" ? body.input : "Recover from empty updates",
                status: "running",
                started_at: "2026-04-23T03:31:00Z",
              },
            ],
          },
        };
      }
      return {};
    });
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [
              {
                id: "c_51jttwiv4yggqagk",
                title: "Fallback session",
                status: inputAccepted ? "busy" : "ready",
                created_at: "2026-04-23T03:30:00Z",
                turns: [],
              },
            ],
          };
        case "/api/chat/sessions/c_51jttwiv4yggqagk":
          if (inputAccepted) {
            detailReadsAfterInput += 1;
          }
          return {
            session: inputAccepted
              ? {
                  id: "c_51jttwiv4yggqagk",
                  title: "Fallback session",
                  status: "ready",
                  created_at: "2026-04-23T03:30:00Z",
                  turns: [
                    {
                      id: "turn-running",
                      prompt: "Recover from empty updates",
                      status: "success",
                      started_at: "2026-04-23T03:31:00Z",
                      finished_at: "2026-04-23T03:31:03Z",
                      final_output: "Recovered through detail fallback",
                    },
                  ],
                }
              : {
                  id: "c_51jttwiv4yggqagk",
                  title: "Fallback session",
                  status: "ready",
                  created_at: "2026-04-23T03:30:00Z",
                  turns: [],
                },
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    const view = render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions"));
    vi.useFakeTimers();
    fireEvent.click(screen.getByRole("button", { name: "send" }));

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });
    expect(screen.getByTestId("active-session-status")).toHaveTextContent("busy");

    await advanceRuntimePollTimers(9);

    expect(updateCallsAfterInput).toBeGreaterThanOrEqual(6);
    expect(detailReadsAfterInput).toBeGreaterThan(0);
    expect(screen.getByTestId("assistant-text")).toHaveTextContent("Recovered through detail fallback");
    expect(screen.getByTestId("active-session-status")).toHaveTextContent("ready");
    view.unmount();
    vi.clearAllTimers();
    vi.useRealTimers();
  });

  it("does not count advancing busy update events as empty polling for detail fallback", async () => {
    vi.stubGlobal("fetch", vi.fn());
    let inputAccepted = false;
    let updateCallsAfterInput = 0;
    let detailReadsAfterInput = 0;
    apiClientMock.post.mockImplementation(async (path: string, body?: Record<string, unknown>) => {
      if (path === "/api/chat/sessions/updates") {
        if (inputAccepted) {
          updateCallsAfterInput += 1;
        }
        return {
          latest_update_id: updateCallsAfterInput,
          resync_required: false,
          updates: inputAccepted
            ? [{
                update_id: updateCallsAfterInput,
                session_id: "c_51jttwiv4yggqagk",
                turn_id: "turn-running",
                type: "turn.started",
                payload: {
                  session: {
                    id: "c_51jttwiv4yggqagk",
                    title: "Long running session",
                    status: "busy",
                    created_at: "2026-04-23T03:30:00Z",
                    updated_at: Date.now() + updateCallsAfterInput,
                  },
                  turn: {
                    id: "turn-running",
                    prompt: typeof body?.input === "string" ? body.input : "Still running",
                    status: "running",
                    started_at: "2026-04-23T03:31:00Z",
                  },
                },
              }]
            : [],
        };
      }
      if (path === "/api/chat/sessions/c_51jttwiv4yggqagk/input") {
        inputAccepted = true;
        return {
          session: {
            id: "c_51jttwiv4yggqagk",
            title: "Long running session",
            status: "busy",
            created_at: "2026-04-23T03:30:00Z",
            turns: [
              {
                id: "turn-running",
                prompt: typeof body?.input === "string" ? body.input : "Still running",
                status: "running",
                started_at: "2026-04-23T03:31:00Z",
              },
            ],
          },
        };
      }
      return {};
    });
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [
              {
                id: "c_51jttwiv4yggqagk",
                title: "Long running session",
                status: inputAccepted ? "busy" : "ready",
                created_at: "2026-04-23T03:30:00Z",
                turns: [],
              },
            ],
          };
        case "/api/chat/sessions/c_51jttwiv4yggqagk":
          if (inputAccepted) {
            detailReadsAfterInput += 1;
          }
          return {
            session: {
              id: "c_51jttwiv4yggqagk",
              title: "Long running session",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns: [{
                id: "turn-old",
                prompt: "old prompt",
                status: "success",
                started_at: "2026-04-23T03:30:00Z",
                finished_at: "2026-04-23T03:30:01Z",
                final_output: "old answer",
              }],
            },
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    const view = render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions"));
    vi.useFakeTimers();
    fireEvent.click(screen.getByRole("button", { name: "send" }));

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });
    expect(screen.getByTestId("active-session-status")).toHaveTextContent("busy");

    await advanceRuntimePollTimers(12);

    expect(updateCallsAfterInput).toBeGreaterThanOrEqual(10);
    expect(detailReadsAfterInput).toBe(0);
    expect(screen.getByTestId("active-session-status")).toHaveTextContent("busy");
    view.unmount();
    vi.clearAllTimers();
    vi.useRealTimers();
  });

  it("falls back to Chat session detail when busy update polling returns only unrelated backlog events", async () => {
    vi.stubGlobal("fetch", vi.fn());
    let inputAccepted = false;
    let updateCallsAfterInput = 0;
    let detailReadsAfterInput = 0;
    apiClientMock.post.mockImplementation(async (path: string, body?: Record<string, unknown>) => {
      if (path === "/api/chat/sessions/updates") {
        if (inputAccepted) {
          updateCallsAfterInput += 1;
        }
        return {
          latest_update_id: updateCallsAfterInput,
          resync_required: false,
          has_more: true,
          updates: inputAccepted
            ? [{
                update_id: updateCallsAfterInput,
                session_id: "c_stalechat0000000",
                type: "session.updated",
                payload: {
                  session: {
                    id: "c_stalechat0000000",
                    title: "Stale backlog",
                    status: "ready",
                    created_at: "2026-04-23T03:00:00Z",
                  },
                },
              }]
            : [],
        };
      }
      if (path === "/api/chat/sessions/c_51jttwiv4yggqagk/input") {
        inputAccepted = true;
        return {
          session: {
            id: "c_51jttwiv4yggqagk",
            title: "Backlog fallback session",
            status: "busy",
            created_at: "2026-04-23T03:30:00Z",
            turns: [
              {
                id: "turn-running",
                prompt: typeof body?.input === "string" ? body.input : "Recover from unrelated backlog",
                status: "running",
                started_at: "2026-04-23T03:31:00Z",
              },
            ],
          },
        };
      }
      return {};
    });
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [
              {
                id: "c_51jttwiv4yggqagk",
                title: "Backlog fallback session",
                status: inputAccepted ? "busy" : "ready",
                created_at: "2026-04-23T03:30:00Z",
                turns: [],
              },
            ],
          };
        case "/api/chat/sessions/c_51jttwiv4yggqagk":
          if (inputAccepted) {
            detailReadsAfterInput += 1;
          }
          return {
            session: inputAccepted
              ? {
                  id: "c_51jttwiv4yggqagk",
                  title: "Backlog fallback session",
                  status: "ready",
                  created_at: "2026-04-23T03:30:00Z",
                  turns: [
                    {
                      id: "turn-running",
                      prompt: "Recover from unrelated backlog",
                      status: "success",
                      started_at: "2026-04-23T03:31:00Z",
                      finished_at: "2026-04-23T03:31:03Z",
                      final_output: "Recovered despite unrelated backlog",
                    },
                  ],
                }
              : {
                  id: "c_51jttwiv4yggqagk",
                  title: "Backlog fallback session",
                  status: "ready",
                  created_at: "2026-04-23T03:30:00Z",
                  turns: [],
                },
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    const view = render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions"));
    vi.useFakeTimers();
    fireEvent.click(screen.getByRole("button", { name: "send" }));

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });
    expect(screen.getByTestId("active-session-status")).toHaveTextContent("busy");

    await advanceRuntimePollTimers(9);

    expect(updateCallsAfterInput).toBeGreaterThanOrEqual(6);
    expect(detailReadsAfterInput).toBeGreaterThan(0);
    expect(screen.getByTestId("assistant-text")).toHaveTextContent("Recovered despite unrelated backlog");
    expect(screen.getByTestId("active-session-status")).toHaveTextContent("ready");
    view.unmount();
    vi.clearAllTimers();
    vi.useRealTimers();
  });

  it("falls back to Chat session detail when current-session update polling only repeats busy backlog", async () => {
    vi.stubGlobal("fetch", vi.fn());
    let inputAccepted = false;
    let updateCallsAfterInput = 0;
    let detailReadsAfterInput = 0;
    apiClientMock.post.mockImplementation(async (path: string, body?: Record<string, unknown>) => {
      if (path === "/api/chat/sessions/updates") {
        if (inputAccepted) {
          updateCallsAfterInput += 1;
        }
        return {
          latest_update_id: updateCallsAfterInput,
          resync_required: false,
          has_more: true,
          updates: inputAccepted
            ? [{
                update_id: updateCallsAfterInput,
                session_id: "c_51jttwiv4yggqagk",
                turn_id: "turn-running",
                type: "turn.started",
                payload: {
                  session: {
                    id: "c_51jttwiv4yggqagk",
                    title: "Repeated busy backlog",
                    status: "busy",
                    created_at: "2026-04-23T03:30:00Z",
                  },
                  turn: {
                    id: "turn-running",
                    prompt: typeof body?.input === "string" ? body.input : "Recover from repeated busy",
                    status: "running",
                    started_at: "2026-04-23T03:31:00Z",
                  },
                },
              }]
            : [],
        };
      }
      if (path === "/api/chat/sessions/c_51jttwiv4yggqagk/input") {
        inputAccepted = true;
        return {
          session: {
            id: "c_51jttwiv4yggqagk",
            title: "Repeated busy backlog",
            status: "busy",
            created_at: "2026-04-23T03:30:00Z",
            updated_at: "2026-04-23T03:34:00Z",
            turns: [
              {
                id: "turn-running",
                prompt: typeof body?.input === "string" ? body.input : "Recover from repeated busy",
                status: "running",
                started_at: "2026-04-23T03:31:00Z",
              },
            ],
          },
        };
      }
      return {};
    });
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [
              {
                id: "c_51jttwiv4yggqagk",
                title: "Repeated busy backlog",
                status: inputAccepted ? "ready" : "ready",
                created_at: "2026-04-23T03:30:00Z",
                updated_at: "2026-04-23T03:33:00Z",
                last_output_at: inputAccepted ? "2026-04-23T03:33:00Z" : "",
                turns: [],
              },
            ],
          };
        case "/api/chat/sessions/c_51jttwiv4yggqagk":
          if (inputAccepted) {
            detailReadsAfterInput += 1;
          }
          return {
            session: inputAccepted && updateCallsAfterInput >= 6
              ? {
                  id: "c_51jttwiv4yggqagk",
                  title: "Repeated busy backlog",
                  status: "ready",
                  created_at: "2026-04-23T03:30:00Z",
                  updated_at: "2026-04-23T03:35:00Z",
                  last_output_at: "2026-04-23T03:35:00Z",
                  turns: [
                    {
                      id: "turn-running",
                      prompt: "Recover from repeated busy",
                      status: "success",
                      started_at: "2026-04-23T03:31:00Z",
                      finished_at: "2026-04-23T03:35:00Z",
                      final_output: "Recovered despite repeated busy backlog",
                    },
                  ],
                }
              : inputAccepted
                ? {
                    id: "c_51jttwiv4yggqagk",
                    title: "Repeated busy backlog",
                    status: "busy",
                    created_at: "2026-04-23T03:30:00Z",
                    updated_at: "2026-04-23T03:34:00Z",
                    turns: [{
                      id: "turn-running",
                      prompt: "Recover from repeated busy",
                      status: "running",
                      started_at: "2026-04-23T03:31:00Z",
                    }],
                  }
              : {
                  id: "c_51jttwiv4yggqagk",
                  title: "Repeated busy backlog",
                  status: "ready",
                  created_at: "2026-04-23T03:30:00Z",
                  turns: [],
                },
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    const view = render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions"));
    vi.useFakeTimers();
    fireEvent.click(screen.getByRole("button", { name: "send" }));

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });
    expect(screen.getByTestId("active-session-status")).toHaveTextContent("busy");

    await advanceRuntimePollTimers(9);

    expect(updateCallsAfterInput).toBeGreaterThanOrEqual(6);
    expect(detailReadsAfterInput).toBeGreaterThan(0);
    expect(screen.getByTestId("assistant-text")).toHaveTextContent("Recovered despite repeated busy backlog");
    expect(screen.getByTestId("active-session-status")).toHaveTextContent("ready");
    view.unmount();
    vi.clearAllTimers();
    vi.useRealTimers();
  });

  it("does not submit another Chat input while the ChatRuntime-backed session is busy", async () => {
    vi.stubGlobal("fetch", vi.fn());
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
        case "/api/chat/sessions/c_51jttwiv4yggqagk":
          return {
            items: [
              {
                id: "c_51jttwiv4yggqagk",
                title: "Image session",
                status: "busy",
                created_at: "2026-04-23T03:30:00Z",
                turns: [
                  {
                    id: "turn-running",
                    prompt: "成都旅游攻略",
                    status: "running",
                    started_at: "2026-04-23T03:31:00Z",
                  },
                ],
              },
            ],
            session: {
              id: "c_51jttwiv4yggqagk",
              title: "Image session",
              status: "busy",
              created_at: "2026-04-23T03:30:00Z",
              turns: [
                {
                  id: "turn-running",
                  prompt: "成都旅游攻略",
                  status: "running",
                  started_at: "2026-04-23T03:31:00Z",
                },
              ],
            },
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("active-session-status")).toHaveTextContent("busy"));

    fireEvent.click(screen.getByRole("button", { name: "send" }));

    await new Promise((resolve) => window.setTimeout(resolve, 0));

    expect(apiClientMock.post).not.toHaveBeenCalledWith("/api/chat/sessions/c_51jttwiv4yggqagk/input", expect.any(Object));
    expect(screen.getByTestId("user-text")).toHaveTextContent("成都旅游攻略");
    expect(screen.getByTestId("assistant-text")).toHaveTextContent("");
  });

  it("allows sending another Chat input when a ready summary contains a terminal failed turn", async () => {
    vi.stubGlobal("fetch", vi.fn());
    const failedAt = "2026-04-23T03:31:02Z";
    const failedTurn = {
      id: "turn-failed",
      prompt: "previous prompt",
      status: "failed",
      started_at: "2026-04-23T03:31:00Z",
      finished_at: failedAt,
      runtime_trace_events: [{
        id: "event-failed",
        turn_id: "turn-failed",
        seq: 1,
        source: "adapter",
        provider: { engine: "codex", adapter: "codex_cli_json" },
        role: "assistant",
        kind: "system_event",
        lifecycle: "failed",
        status: "failed",
        title: "Request failed",
        summary: "codex command failed",
        blocks: [],
        visibility: "developer",
        completed_at: failedAt,
      }],
    };
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
        case "/api/chat/sessions/c_51jttwiv4yggqagk":
          return {
            items: [
              {
                id: "c_51jttwiv4yggqagk",
                title: "Failed session",
                status: "ready",
                created_at: "2026-04-23T03:30:00Z",
                updated_at: failedAt,
                turns: [failedTurn],
              },
            ],
            session: {
              id: "c_51jttwiv4yggqagk",
              title: "Failed session",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              updated_at: failedAt,
              turns: [failedTurn],
            },
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });
    apiClientMock.post.mockImplementation(async (path: string, body?: Record<string, unknown>) => {
      if (path === "/api/chat/sessions/c_51jttwiv4yggqagk/input") {
        return {
          session: {
            id: "c_51jttwiv4yggqagk",
            title: "Failed session",
            status: "ready",
            created_at: "2026-04-23T03:30:00Z",
            updated_at: "2026-04-23T03:32:02Z",
            turns: [
              {
                id: "turn-recovered",
                prompt: typeof body?.input === "string" ? body.input : "Inspect this image",
                status: "success",
                started_at: "2026-04-23T03:32:00Z",
                finished_at: "2026-04-23T03:32:02Z",
                final_output: "Recovered reply",
              },
            ],
          },
        };
      }
      return {};
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("active-session-status")).toHaveTextContent("ready"));

    fireEvent.click(screen.getByRole("button", { name: "send" }));

    await waitFor(() => expect(apiClientMock.post).toHaveBeenCalledWith(
      "/api/chat/sessions/c_51jttwiv4yggqagk/input",
      expect.objectContaining({ input: "Inspect this image" }),
    ));
    await waitFor(() => expect(screen.getByTestId("assistant-texts")).toHaveTextContent("Recovered reply"));
  });

  it("allows clicking the active inspector tab again to collapse only that tab content", async () => {
    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <InspectorHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("inspector-state")).toHaveTextContent("model:details-closed:tab-open"));

    fireEvent.click(screen.getByRole("button", { name: "model" }));
    expect(screen.getByTestId("inspector-state")).toHaveTextContent("model:details-open:tab-closed");

    fireEvent.click(screen.getByRole("button", { name: "model" }));
    expect(screen.getByTestId("inspector-state")).toHaveTextContent("model:details-open:tab-open");
  });

  it("adds a Codex option for Chat model selection and submits through ChatRuntime input", async () => {
    vi.stubGlobal("fetch", vi.fn());
    let requestBody: Record<string, unknown> | null = null;
    apiClientMock.post.mockImplementation(async (path: string, body?: Record<string, unknown>) => {
      if (path === "/api/chat/sessions/c_51jttwiv4yggqagk/input") {
        requestBody = body || null;
        return {
          session: {
            id: "c_51jttwiv4yggqagk",
            title: "Image session",
            status: "ready",
            created_at: "2026-04-23T03:30:00Z",
            turns: [
              {
                id: "turn-codex",
                prompt: "Run this with Codex",
                status: "success",
                final_output: "Codex done",
              },
            ],
          },
        };
      }
      return {};
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <ModelSelectionHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("provider-list")).toHaveTextContent("Codex:Codex"));
    fireEvent.click(screen.getByRole("button", { name: "select codex" }));
    await waitFor(() => expect(screen.getByTestId("selected-model")).toHaveTextContent("Codex"));

    fireEvent.click(screen.getByRole("button", { name: "send with codex" }));

    await waitFor(() => expect(requestBody?.input).toBe("Run this with Codex"));
    expect(requestBody).not.toHaveProperty("metadata");
    expect(requestBody?.execution_engine).toBe("codex");
    expect(requestBody).not.toHaveProperty("model_provider_id");
    expect(apiClientMock.post).toHaveBeenCalledWith("/api/chat/sessions/c_51jttwiv4yggqagk/input", expect.any(Object));
    expect(fetch).not.toHaveBeenCalled();
  });

  it("defaults the executor selection to Codex and restores changed runtime config from browser storage", async () => {
    let requestBody: Record<string, unknown> | null = null;
    apiClientMock.post.mockImplementation(async (path: string, body?: Record<string, unknown>) => {
      if (path === "/api/chat/sessions/c_51jttwiv4yggqagk/input") {
        requestBody = body || null;
        return {
          session: {
            id: "c_51jttwiv4yggqagk",
            title: "Stored config session",
            status: "ready",
            created_at: "2026-04-23T03:30:00Z",
            turns: [{ id: "turn-stored-config", prompt: "Run with stored config", status: "success", final_output: "Done" }],
          },
        };
      }
      return {};
    });
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [
              {
                id: "c_51jttwiv4yggqagk",
                title: "Stored config session",
                status: "ready",
                created_at: "2026-04-23T03:30:00Z",
                turns: [],
              },
            ],
          };
        case "/api/chat/sessions/c_51jttwiv4yggqagk":
          return {
            session: {
              id: "c_51jttwiv4yggqagk",
              title: "Stored config session",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns: [],
            },
          };
        case "/api/control/llm/providers":
          return {
            items: [
              {
                id: "openrouter",
                name: "OpenRouter",
                is_enabled: true,
                is_default: true,
                default_model: "anthropic/claude-sonnet",
                models: [
                  { id: "anthropic/claude-sonnet", name: "Claude Sonnet", is_enabled: true },
                ],
              },
            ],
          };
        case "/api/control/mcps":
          return {
            items: [
              { id: "filesystem", name: "Filesystem", description: "Read files", enabled: true },
            ],
          };
        default:
          return { items: [] };
      }
    });

    const firstView = render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeConfigSelectionHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("selected-provider")).toHaveTextContent("alter0-codex"));
    expect(screen.getByTestId("selected-model")).toHaveTextContent("codex");

    fireEvent.click(screen.getByRole("button", { name: "select openrouter" }));
    fireEvent.click(screen.getByRole("button", { name: "enable filesystem" }));

    await waitFor(() => expect(screen.getByTestId("selected-provider")).toHaveTextContent("openrouter"));
    expect(screen.getByTestId("filesystem-state")).toHaveTextContent("active");

    firstView.unmount();

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeConfigSelectionHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("selected-provider")).toHaveTextContent("openrouter"));
    expect(screen.getByTestId("selected-model")).toHaveTextContent("anthropic/claude-sonnet");
    expect(screen.getByTestId("filesystem-state")).toHaveTextContent("active");

    fireEvent.click(screen.getByRole("button", { name: "send with stored config" }));

    await waitFor(() => expect(requestBody?.input).toBe("Run with stored config"));
    expect(requestBody?.model_provider_id).toBe("openrouter");
    expect(requestBody?.model_id).toBe("anthropic/claude-sonnet");
    expect(requestBody?.mcp_ids).toEqual(["filesystem"]);
    expect(requestBody).not.toHaveProperty("skill_ids");
  });

  it("keeps locally appended messages when a session collection refresh returns a shorter history", async () => {
    mockMessageDone("Remote completion");

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions"));

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "send" }));
    });

    await waitFor(() => expect(screen.getByTestId("assistant-text")).toHaveTextContent("Remote completion"));
  });

  it("keeps existing Chat history when a send response returns a non-overlapping paged turn", async () => {
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [{
              id: "c_51jttwiv4yggqagk",
              title: "Paged send",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns_paging: { has_more_before: true },
              turns: [{
                id: "turn-1",
                prompt: "older prompt",
                status: "success",
                started_at: "2026-04-23T03:31:00Z",
                finished_at: "2026-04-23T03:31:01Z",
                final_output: "older answer",
              }],
            }],
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });
    apiClientMock.post.mockImplementation(async (path: string) => {
      if (path === "/api/chat/sessions/c_51jttwiv4yggqagk/input") {
        return {
          session: {
            id: "c_51jttwiv4yggqagk",
            title: "Paged send",
            status: "ready",
            created_at: "2026-04-23T03:30:00Z",
            turns_paging: { has_more_before: true },
            turns: [{
              id: "turn-2",
              prompt: "new prompt",
              status: "success",
              started_at: "2026-04-23T03:32:00Z",
              finished_at: "2026-04-23T03:32:01Z",
              final_output: "new answer",
            }],
          },
        };
      }
      return {};
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <MessageTextHarness />
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("older answer"));

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "send" }));
    });

    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("new answer"));
    expect(screen.getByTestId("message-texts")).toHaveTextContent("older answer");
  });

  it("rejects a stale paged detail before applying a newer latest-page refresh", async () => {
    let detailReads = 0;
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [{
              id: "c_51jttwiv4yggqagk",
              title: "Paged chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              updated_at: "2026-04-23T03:32:03Z",
              last_output_at: "2026-04-23T03:32:03Z",
              activity_at: "2026-04-23T03:32:03Z",
              freshnessAt: Date.parse("2026-04-23T03:32:03Z"),
              turns_paging: { has_more_before: true },
            }],
          };
        case "/api/chat/sessions/c_51jttwiv4yggqagk":
          detailReads += 1;
          if (detailReads === 1) {
            return {
              session: {
                id: "c_51jttwiv4yggqagk",
                title: "Paged chat",
                status: "ready",
                created_at: "2026-04-23T03:30:00Z",
                updated_at: "2026-04-23T03:32:01Z",
                last_output_at: "2026-04-23T03:32:01Z",
                activity_at: "2026-04-23T03:32:01Z",
                freshnessAt: Date.parse("2026-04-23T03:32:01Z"),
                turns_paging: { has_more_before: true },
                turns: [
                  {
                    id: "turn-1",
                    prompt: "older",
                    status: "success",
                    started_at: "2026-04-23T03:31:00Z",
                    finished_at: "2026-04-23T03:31:01Z",
                    final_output: "older answer",
                  },
                  {
                    id: "turn-2",
                    prompt: "newer",
                    status: "success",
                    started_at: "2026-04-23T03:32:00Z",
                    finished_at: "2026-04-23T03:32:01Z",
                    final_output: "newer answer",
                  },
                ],
              },
            };
          }
          return {
            session: {
              id: "c_51jttwiv4yggqagk",
              title: "Paged chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              updated_at: "2026-04-23T03:32:03Z",
              last_output_at: "2026-04-23T03:32:03Z",
              activity_at: "2026-04-23T03:32:03Z",
              freshnessAt: Date.parse("2026-04-23T03:32:03Z"),
              turns_paging: { has_more_before: true },
              turns: [{
                id: "turn-2",
                prompt: "newer",
                status: "success",
                started_at: "2026-04-23T03:32:00Z",
                finished_at: "2026-04-23T03:32:02Z",
                final_output: "newer answer refreshed",
              }],
            },
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <MessageTextHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("newer answer"));

    fireEvent.click(screen.getByRole("button", { name: "refresh active" }));

    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("newer answer refreshed"));
    expect(screen.getByTestId("message-texts")).not.toHaveTextContent("older answer");
  });

  it("refreshes the active Chat session on demand so paged history can continue loading", async () => {
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [{
              id: "c_51jttwiv4yggqagk",
              title: "Paged chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns_paging: { has_more_before: true },
              turns: [{
                id: "turn-2",
                prompt: "newer",
                status: "success",
                started_at: "2026-04-23T03:32:00Z",
                finished_at: "2026-04-23T03:32:01Z",
                final_output: "newer answer",
              }],
            }],
          };
        case "/api/chat/sessions/c_51jttwiv4yggqagk":
          return {
            session: {
              id: "c_51jttwiv4yggqagk",
              title: "Paged chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns_paging: { has_more_before: true },
              turns: [
                {
                  id: "turn-1",
                  prompt: "older",
                  status: "success",
                  started_at: "2026-04-23T03:31:00Z",
                  finished_at: "2026-04-23T03:31:01Z",
                  final_output: "older answer",
                },
                {
                  id: "turn-2",
                  prompt: "newer",
                  status: "success",
                  started_at: "2026-04-23T03:32:00Z",
                  finished_at: "2026-04-23T03:32:01Z",
                  final_output: "newer answer",
                },
              ],
            },
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <MessageTextHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("newer answer"));
    fireEvent.click(screen.getByRole("button", { name: "refresh active" }));

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions/c_51jttwiv4yggqagk"));
    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("older answer"));
    expect(screen.getByTestId("message-texts")).toHaveTextContent("newer answer");
  });

  it("does not visibly refresh Chat with earlier turn pages until the user asks for history", async () => {
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [{
              id: "c_51jttwiv4yggqagk",
              title: "Progressive chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns_paging: {
                has_more_before: true,
                oldest_turn_id: "turn-3",
                newest_turn_id: "turn-3",
                next_before_turn_id: "turn-3",
                total: 3,
              },
              turns: [{
                id: "turn-3",
                prompt: "latest",
                status: "success",
                started_at: "2026-04-23T03:33:00Z",
                finished_at: "2026-04-23T03:33:01Z",
                final_output: "latest answer",
              }],
            }],
          };
        case "/api/chat/sessions/c_51jttwiv4yggqagk?turn_before=turn-3&turn_limit=20":
          return {
            session: {
              id: "c_51jttwiv4yggqagk",
              title: "Progressive chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns_paging: {
                has_more_before: false,
                has_more_after: true,
                oldest_turn_id: "turn-1",
                newest_turn_id: "turn-2",
                next_before_turn_id: "turn-1",
                total: 3,
              },
              turns: [
                {
                  id: "turn-1",
                  prompt: "oldest",
                  status: "success",
                  started_at: "2026-04-23T03:31:00Z",
                  finished_at: "2026-04-23T03:31:01Z",
                  final_output: "oldest answer",
                },
                {
                  id: "turn-2",
                  prompt: "middle",
                  status: "success",
                  started_at: "2026-04-23T03:32:00Z",
                  finished_at: "2026-04-23T03:32:01Z",
                  final_output: "middle answer",
                },
              ],
            },
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <MessageTextHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("latest answer"));
    await act(async () => {
      await new Promise((resolve) => window.setTimeout(resolve, 50));
    });

    expect(apiClientMock.get).not.toHaveBeenCalledWith(
      "/api/chat/sessions/c_51jttwiv4yggqagk?turn_before=turn-3&turn_limit=20",
    );
    expect(screen.getByTestId("message-texts")).not.toHaveTextContent("oldest answer");

    fireEvent.click(screen.getByRole("button", { name: "load earlier history" }));

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith(
      "/api/chat/sessions/c_51jttwiv4yggqagk?turn_before=turn-3&turn_limit=20",
    ));
    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("oldest answer"));
    expect(screen.getByTestId("message-texts")).toHaveTextContent("middle answer");
    expect(screen.getByTestId("message-texts")).toHaveTextContent("latest answer");
  });

  it("does not repeatedly reload the same Chat history page when a requested page makes no progress", async () => {
    let backgroundPageLoads = 0;
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [{
              id: "c_51jttwiv4yggqagk",
              title: "Loop guard",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns_paging: {
                has_more_before: true,
                oldest_turn_id: "turn-3",
                newest_turn_id: "turn-3",
                next_before_turn_id: "turn-3",
              },
              turns: [{
                id: "turn-3",
                prompt: "latest",
                status: "success",
                started_at: "2026-04-23T03:33:00Z",
                finished_at: "2026-04-23T03:33:01Z",
                final_output: "latest answer",
                runtime_trace_events: [],
              }],
            }],
          };
        case "/api/chat/sessions/c_51jttwiv4yggqagk?turn_before=turn-3&turn_limit=20":
          backgroundPageLoads += 1;
          if (backgroundPageLoads > 1) {
            throw new Error("reloaded the same Chat history page");
          }
          return {
            session: {
              id: "c_51jttwiv4yggqagk",
              title: "Loop guard",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns: [],
            },
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <MessageTextHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("latest answer"));
    fireEvent.click(screen.getByRole("button", { name: "load earlier history" }));
    await waitFor(() => expect(backgroundPageLoads).toBe(1));
    fireEvent.click(screen.getByRole("button", { name: "load earlier history" }));
    await act(async () => {
      await new Promise((resolve) => window.setTimeout(resolve, 50));
    });

    expect(backgroundPageLoads).toBe(1);
  });

  it("restores cached Chat session info when the full long-term message cache is unavailable", async () => {
    window.sessionStorage.setItem(ACTIVE_SESSION_STORAGE_KEY, JSON.stringify({ chat: "c_cachedchat000000" }));
    window.localStorage.setItem(
      SESSION_INFO_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        cachedAt: Date.now(),
        activeSessionByRoute: { chat: "c_cachedchat000000" },
        sessionsByRoute: {
          chat: [{
            id: "c_cachedchat000000",
            status: "ready",
            title: "Cached session info",
            createdAt: Date.parse("2026-04-23T03:30:00Z"),
            pinned: true,
            targetID: "codex",
            targetName: "Codex",
            messages: [],
            messagesLoaded: false,
            serverBacked: true,
          }],
        },
      }),
    );

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <ActiveSessionTitleHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("active-session-title")).toHaveTextContent("Cached session info"));
  });

  it("merges cached Chat session info with full long-term message caches", async () => {
    window.sessionStorage.setItem(ACTIVE_SESSION_STORAGE_KEY, JSON.stringify({ chat: "c_fullchat00000000" }));
    window.localStorage.setItem(
      LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        cachedAt: Date.now(),
        activeSessionByRoute: { chat: "c_fullchat00000000" },
        sessionsByRoute: {
          chat: [{
            id: "c_fullchat00000000",
            status: "ready",
            title: "Full cached chat",
            createdAt: Date.parse("2026-04-23T03:30:00Z"),
            pinned: false,
            targetID: "codex",
            targetName: "Codex",
            messages: [
              {
                id: "turn-full:user",
                role: "user",
                text: "full prompt",
                attachments: [],
                route: "chat",
                source: "runtime",
                error: false,
                status: "",
                at: Date.parse("2026-04-23T03:30:00Z"),
                processEvents: [],
              },
              {
                id: "turn-full:assistant",
                role: "assistant",
                text: "full cached answer",
                attachments: [],
                route: "chat",
                source: "runtime",
                error: false,
                status: "done",
                at: Date.parse("2026-04-23T03:30:02Z"),
                processEvents: [],
              },
            ],
            messagesLoaded: true,
            serverBacked: true,
            turnsPaging: { has_more_before: false },
          }],
        },
      }),
    );
    window.localStorage.setItem(
      SESSION_INFO_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        cachedAt: Date.now(),
        activeSessionByRoute: { chat: "c_fullchat00000000" },
        sessionsByRoute: {
          chat: [
            {
              id: "c_fullchat00000000",
              status: "ready",
              title: "Full cached chat",
              createdAt: Date.parse("2026-04-23T03:30:00Z"),
              pinned: false,
              targetID: "codex",
              targetName: "Codex",
              messages: [],
              messagesLoaded: false,
              serverBacked: true,
            },
            {
              id: "c_infochat00000000",
              status: "ready",
              title: "Info cached chat",
              createdAt: Date.parse("2026-04-23T04:30:00Z"),
              pinned: false,
              targetID: "codex",
              targetName: "Codex",
              messages: [],
              messagesLoaded: false,
              serverBacked: true,
            },
          ],
        },
      }),
    );

    const listRequest = deferred<{ items?: unknown[] }>();
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return listRequest.promise;
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <MessageTextHarness />
        <SessionListHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("full cached answer"));
    expect(screen.getByTestId("sessions")).toHaveTextContent("Full cached chat");
    expect(screen.getByTestId("sessions")).toHaveTextContent("Info cached chat");
  });

  it("loads Chat runtime process event details on demand without persisting heavy detail blocks", async () => {
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [{
              id: "c_51jttwiv4yggqagk",
              title: "Detail chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns: [{
                id: "turn-1",
                prompt: "show thinking",
                status: "success",
                started_at: "2026-04-23T03:31:00Z",
                finished_at: "2026-04-23T03:31:01Z",
                final_output: "done",
                runtime_trace_events: [{
                  id: 101,
                  kind: "reasoning",
                  status: "completed",
                  text: "Thinking",
                  detail_available: true,
                  created_at: "2026-04-23T03:31:01Z",
                }],
              }],
            }],
          };
        case "/api/chat/sessions/c_51jttwiv4yggqagk":
          return {
            session: {
              id: "c_51jttwiv4yggqagk",
              title: "Detail chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns: [{
                id: "turn-1",
                prompt: "show thinking",
                status: "success",
                started_at: "2026-04-23T03:31:00Z",
                finished_at: "2026-04-23T03:31:01Z",
                final_output: "done",
                runtime_trace_events: [{
                  id: 101,
                  kind: "reasoning",
                  status: "completed",
                  text: "Thinking",
                  detail_available: true,
                  created_at: "2026-04-23T03:31:01Z",
                }],
              }],
              turns_paging: { has_more_before: false },
            },
          };
        case "/api/chat/sessions/c_51jttwiv4yggqagk/turns/turn-1/events/101":
          return {
            event: {
              turn_id: "turn-1",
              event: {
                id: 101,
                kind: "reasoning",
                status: "completed",
                text: "Thinking",
                detail_available: false,
              },
              blocks: [{ type: "thinking", text: "full thinking detail" }],
            },
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("assistant-process-count")).toHaveTextContent("1"));
    expect(screen.getByTestId("assistant-process-blocks")).not.toHaveTextContent("full thinking detail");

    fireEvent.click(screen.getByRole("button", { name: "load process detail" }));

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith(
      "/api/chat/sessions/c_51jttwiv4yggqagk/turns/turn-1/events/101",
    ));
    await waitFor(() => expect(screen.getByTestId("assistant-process-blocks")).toHaveTextContent("full thinking detail"));

    apiClientMock.get.mockClear();
    fireEvent.click(screen.getByRole("button", { name: "refresh active" }));
    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions/c_51jttwiv4yggqagk"));
    expect(screen.getByTestId("assistant-process-blocks")).toHaveTextContent("full thinking detail");
    expect(window.localStorage.getItem("alter0.web.session.long_term_snapshot.v1")).not.toContain("full thinking detail");
  });
});

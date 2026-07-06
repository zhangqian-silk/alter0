import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { memo } from "react";
import {
  CHAT_RUNTIME_CACHE_SESSION_TTL_MS,
  ConversationRuntimeProvider,
  mergeRuntimeSessions,
  resetConversationRuntimeCache,
  resolveChatSessionPollPlan,
  resolveRuntimeResyncSessionIDs,
  shouldRefreshChatSessionDetailAfterEmptyUpdates,
  type ChatSession,
  useConversationRuntime,
  useConversationRuntimeComposer,
  useConversationRuntimeWorkspace,
} from "./ConversationRuntimeProvider";
import { hashSessionIDShort } from "../../shared/session/sessionHash";

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
      <output data-testid="user-text">{userMessage?.text || ""}</output>
      <output data-testid="assistant-text">{assistantMessage?.text || ""}</output>
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
      <button type="button" onClick={() => runtime.toggleInspector("capabilities")}>
        capabilities
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

function SkillSelectionHarness() {
  const runtime = useConversationRuntime();
  const memorySkill = runtime.skills.find((item) => item.id === "memory");

  return (
    <div>
      <button type="button" onClick={() => runtime.toggleSkill("memory", true)}>
        enable memory
      </button>
      <button type="button" onClick={() => runtime.toggleSkill("memory", false)}>
        disable memory
      </button>
      <button type="button" onClick={() => void runtime.sendPrompt("Run with selected skills")}>
        send with skills
      </button>
      <output data-testid="memory-skill-state">{memorySkill?.active ? "active" : "inactive"}</output>
      <output data-testid="skill-count">{runtime.skillCount}</output>
    </div>
  );
}

function RuntimeConfigSelectionHarness() {
  const runtime = useConversationRuntime();
  const filesystemMCP = runtime.capabilities.find((item) => item.id === "filesystem");
  const memorySkill = runtime.skills.find((item) => item.id === "memory");

  return (
    <div>
      <button type="button" onClick={() => runtime.selectModel("openrouter", "anthropic/claude-sonnet")}>
        select openrouter
      </button>
      <button type="button" onClick={() => runtime.toggleCapability("filesystem", "mcp", true)}>
        enable filesystem
      </button>
      <button type="button" onClick={() => runtime.toggleSkill("memory", false)}>
        disable memory
      </button>
      <button type="button" onClick={() => void runtime.sendPrompt("Run with stored config")}>
        send with stored config
      </button>
      <output data-testid="selected-provider">{runtime.selectedProviderId}</output>
      <output data-testid="selected-model">{runtime.selectedModelId}</output>
      <output data-testid="filesystem-state">{filesystemMCP?.active ? "active" : "inactive"}</output>
      <output data-testid="memory-skill-state">{memorySkill?.active ? "active" : "inactive"}</output>
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
      <button type="button" onClick={() => void runtime.setSessionPinned("alter0-chat", false)}>
        unpin active
      </button>
      <button type="button" onClick={() => void runtime.setSessionPinned(runtime.activeSession?.id || "", true)}>
        pin active
      </button>
      <button type="button" onClick={() => void runtime.setSessionPinned("older-chat", true)}>
        pin older
      </button>
      <output data-testid="sessions">
        {runtime.sessionItems.map((session) => `${session.title}:${session.shortHash}:${session.pinned ? "pinned" : "unpinned"}`).join("|")}
      </output>
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
              id: "alter0-chat",
              title: "Image session",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns: [],
            },
          ],
        };
      case "/api/chat/sessions/alter0-chat":
        return {
          session: {
            id: "alter0-chat",
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
    id: "alter0-chat",
    sourceRoute: "chat",
    status: "ready",
    title: "Image session",
    titleAuto: true,
    titleScore: 0,
    createdAt: Date.parse("2026-04-23T03:30:00Z"),
    updatedAt: Date.parse("2026-04-23T03:30:00Z"),
    lastOutputAt: 0,
    activityAt: Date.parse("2026-04-23T03:30:00Z"),
    revision: Date.parse("2026-04-23T03:30:00Z"),
    detailRevision: Date.parse("2026-04-23T03:30:00Z"),
    pinned: false,
    target: { type: "model", id: "raw-model", name: "Raw Model" },
    modelProviderID: "",
    modelID: "",
    toolIDs: [],
    skillIDs: [],
    skillIDsExplicit: false,
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
    if (path === "/api/chat/sessions/alter0-chat/input") {
      return {
        session: {
          id: "alter0-chat",
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
      JSON.stringify({ chat: "alter0-chat" }),
    );
    setupDefaultAPI();
    apiClientMock.post.mockImplementation(async (path: string) => {
      if (path === "/api/chat/sessions") {
        return {
          session: {
            id: "new-chatRuntime-chat",
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
              asset_url: "/api/chat/sessions/alter0-chat/attachments/uploaded-image-1/original",
              preview_url: "/api/chat/sessions/alter0-chat/attachments/uploaded-image-1/preview",
            },
          ],
        };
      }
      return {};
    });
  });

  afterEach(() => {
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
      interval: 1000,
    });
    expect(resolveChatSessionPollPlan({
      sessionCount: 1,
      pageHidden: false,
      fallbackAttempt: 8,
    })).toEqual({
      enabled: true,
      interval: 5000,
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

  it("keeps a locally busy runtime session busy when a stale list summary returns ready", () => {
    const merged = mergeRuntimeSessions([
      chatSessionFixture({
        status: "ready",
        messages: [],
        messagesLoaded: false,
      }),
    ], [
      chatSessionFixture({
        status: "busy",
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

    expect(merged[0]?.status).toBe("busy");
    expect(merged[0]?.messages.map((message) => message.text)).toEqual(["new prompt"]);
  });

  it("unblocks input when a newer ready summary overtakes a cached running placeholder", async () => {
    const detailRead = deferred<{
      session: {
        id: string;
        title: string;
        status: string;
        created_at: string;
        updated_at: string;
        last_output_at: string;
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
    window.sessionStorage.setItem(ACTIVE_SESSION_STORAGE_KEY, JSON.stringify({ chat: "finished-chat" }));
    window.localStorage.setItem(
      LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        cachedAt: Date.now(),
        activeSessionByRoute: { chat: "finished-chat" },
        sessionsByRoute: {
          chat: [{
            id: "finished-chat",
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
              id: "finished-chat",
              title: "Cached running chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              updated_at: "2026-04-23T03:35:00Z",
              last_output_at: "2026-04-23T03:35:00Z",
              activity_at: "2026-04-23T03:35:00Z",
              revision: Date.parse("2026-04-23T03:35:00Z"),
            }],
          };
        case "/api/chat/sessions/finished-chat":
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
      if (path === "/api/chat/sessions/finished-chat/input") {
        return {
          session: {
            id: "finished-chat",
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
      "/api/chat/sessions/finished-chat/input",
      expect.objectContaining({ input: "Inspect this image" }),
    ));
    await waitFor(() => expect(screen.getByTestId("active-session-status")).toHaveTextContent("busy"));
    await act(async () => {
      detailRead.resolve({
        session: {
          id: "finished-chat",
          title: "Cached running chat",
          status: "ready",
          created_at: "2026-04-23T03:30:00Z",
          updated_at: "2026-04-23T03:35:00Z",
          last_output_at: "2026-04-23T03:35:00Z",
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
    expect(shouldRefreshChatSessionDetailAfterEmptyUpdates(9)).toBe(false);
    expect(shouldRefreshChatSessionDetailAfterEmptyUpdates(10)).toBe(true);
    expect(shouldRefreshChatSessionDetailAfterEmptyUpdates(19)).toBe(false);
    expect(shouldRefreshChatSessionDetailAfterEmptyUpdates(20)).toBe(true);
    expect(shouldRefreshChatSessionDetailAfterEmptyUpdates(49)).toBe(false);
    expect(shouldRefreshChatSessionDetailAfterEmptyUpdates(50)).toBe(true);
    expect(shouldRefreshChatSessionDetailAfterEmptyUpdates(75)).toBe(false);
    expect(shouldRefreshChatSessionDetailAfterEmptyUpdates(100)).toBe(true);
  });

  it("keeps restored ready detail authoritative when a stale busy summary arrives later", () => {
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
      revision: Date.parse("2026-04-23T03:33:00Z"),
      detailRevision: 0,
      updatedAt: Date.parse("2026-04-23T03:31:00Z"),
      lastOutputAt: 0,
      activityAt: Date.parse("2026-04-23T03:31:00Z"),
      messagesLoaded: false,
      messages: [],
    });

    const merged = mergeRuntimeSessions([staleBusySummary], [restoredDetail]);

    expect(merged[0]?.status).toBe("ready");
    expect(merged[0]?.messagesLoaded).toBe(true);
    expect(merged[0]?.detailRevision).toBe(restoredDetail.detailRevision);
    expect(merged[0]?.messages.map((message) => message.text)).toEqual([
      "new prompt",
      "restored answer remains visible",
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
      revision: Date.parse("2026-04-23T03:34:00Z"),
      detailRevision: Date.parse("2026-04-23T03:34:00Z"),
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
    expect(merged[0]?.detailRevision).toBe(newerBusyBoundedTurn.detailRevision);
  });

  it("keeps newer session detail authoritative when an older detail response arrives later", () => {
    const newerSession = chatSessionFixture({
      status: "ready",
      title: "New detail",
      updatedAt: Date.parse("2026-04-23T03:33:00Z"),
      lastOutputAt: Date.parse("2026-04-23T03:33:00Z"),
      activityAt: Date.parse("2026-04-23T03:33:00Z"),
      revision: Date.parse("2026-04-23T03:33:00Z"),
      detailRevision: Date.parse("2026-04-23T03:33:00Z"),
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
      revision: Date.parse("2026-04-23T03:31:00Z"),
      detailRevision: Date.parse("2026-04-23T03:31:00Z"),
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
    expect(merged[0]?.revision).toBe(Date.parse("2026-04-23T03:33:00Z"));
    expect(merged[0]?.detailRevision).toBe(Date.parse("2026-04-23T03:33:00Z"));
    expect(merged[0]?.messages.map((message) => message.text)).toContain("new answer remains visible");
  });

  it("orders sessions by the latest runtime activity instead of creation time", () => {
    const recentlyActiveOlderSession = {
      ...chatSessionFixture({
        id: "older-active-chat",
        title: "Older but active",
        createdAt: Date.parse("2026-04-21T03:30:00Z"),
      }),
      lastOutputAt: Date.parse("2026-04-23T04:30:00Z"),
      updatedAt: Date.parse("2026-04-23T04:31:00Z"),
      activityAt: Date.parse("2026-04-23T04:31:00Z"),
    } as ChatSession & { lastOutputAt: number; updatedAt: number; activityAt: number };
    const newlyCreatedIdleSession = {
      ...chatSessionFixture({
        id: "new-idle-chat",
        title: "New but idle",
        createdAt: Date.parse("2026-04-23T03:30:00Z"),
      }),
      lastOutputAt: 0,
      updatedAt: Date.parse("2026-04-23T03:30:00Z"),
      activityAt: Date.parse("2026-04-23T03:30:00Z"),
    } as ChatSession & { lastOutputAt: number; updatedAt: number; activityAt: number };

    const merged = mergeRuntimeSessions([], [
      newlyCreatedIdleSession,
      recentlyActiveOlderSession,
    ]);

    expect(merged.map((session) => session.id)).toEqual([
      "older-active-chat",
      "new-idle-chat",
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
            session_id: "alter0-chat",
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
            session_id: "alter0-chat",
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

  it("selects only recoverable runtime sessions for owner-level polling resync", () => {
    expect(resolveRuntimeResyncSessionIDs([
      chatSessionFixture({
        id: "busy-chat",
        status: "busy",
        messagesLoaded: true,
        serverBacked: true,
      }),
      chatSessionFixture({
        id: "ready-chat",
        status: "ready",
        messagesLoaded: true,
        serverBacked: true,
      }),
      chatSessionFixture({
        id: "local-busy",
        status: "busy",
        messagesLoaded: true,
        serverBacked: false,
      }),
    ])).toEqual(["busy-chat"]);
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
        case "/api/chat/sessions/alter0-chat":
          return {
            session: {
              id: "alter0-chat",
              title: "Process session",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns: [completedTurn],
            },
            items: [{
              id: "alter0-chat",
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

    await waitFor(() => expect(screen.getByTestId("assistant-process-message-id")).toHaveTextContent("turn-1:assistant"));

    fireEvent.click(screen.getByRole("button", { name: "toggle process" }));
    await waitFor(() => expect(screen.getByTestId("assistant-process-collapsed")).toHaveTextContent("false"));

    fireEvent.click(screen.getByRole("button", { name: "refresh active" }));
    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions/alter0-chat"));
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
              id: "alter0-chat",
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
        id: "alter0-chat",
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
    window.sessionStorage.setItem(ACTIVE_SESSION_STORAGE_KEY, JSON.stringify({ chat: "alter0-chat" }));
    window.localStorage.setItem(
      LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        cachedAt: Date.now(),
        activeSessionByRoute: { chat: "alter0-chat" },
        sessionsByRoute: {
          chat: [{
            id: "alter0-chat",
            status: "ready",
            title: "Durable cached chat",
            createdAt: Date.parse("2026-04-23T03:30:00Z"),
            updatedAt: Date.parse("2026-04-23T03:48:00Z"),
            lastOutputAt: Date.parse("2026-04-23T03:48:00Z"),
            activityAt: Date.parse("2026-04-23T03:48:00Z"),
            revision: Date.parse("2026-04-23T03:48:00Z"),
            detailRevision: Date.parse("2026-04-23T03:48:00Z"),
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
        case "/api/chat/sessions/alter0-chat":
          detailReads += 1;
          return {
            session: {
              id: "alter0-chat",
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
        id: "alter0-chat",
        title: "Durable cached chat",
        status: "ready",
        created_at: "2026-04-23T03:30:00Z",
        updated_at: "2026-04-23T03:48:00Z",
        last_output_at: "2026-04-23T03:48:00Z",
        turns_paging: { has_more_before: false },
        turns: [],
      }],
    });

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions/alter0-chat"));
    expect(detailReads).toBeGreaterThan(0);
    await waitFor(() => expect(screen.getByTestId("process-event-count")).toHaveTextContent("1"));
    expect(screen.getByTestId("message-texts")).toHaveTextContent("durable answer 1");
    expect(screen.getByTestId("message-texts")).toHaveTextContent(`durable answer ${cachedTurnCount}`);
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

  it("calibrates a full cached Chat session on page activation without reloading earlier history", async () => {
    window.sessionStorage.setItem(ACTIVE_SESSION_STORAGE_KEY, JSON.stringify({ chat: "cached-chat" }));
    window.localStorage.setItem(
      LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        cachedAt: Date.now(),
        activeSessionByRoute: { chat: "cached-chat" },
        sessionsByRoute: {
          chat: [{
            id: "cached-chat",
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
              id: "cached-chat",
              title: "Fully cached chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns: [],
              turns_paging: { has_more_before: false },
            }],
          };
        case "/api/chat/sessions/cached-chat":
          return {
            session: {
              id: "cached-chat",
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

    window.dispatchEvent(new PageTransitionEvent("pageshow", { persisted: true }));

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions"));
    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions/cached-chat"));
    expect(apiClientMock.get).not.toHaveBeenCalledWith(
      "/api/chat/sessions/cached-chat?turn_before=cached-turn&turn_limit=20",
    );
    expect(screen.getByTestId("message-texts")).toHaveTextContent("cached answer");
  });

  it("calibrates a stable latest Chat page cache on page activation without reloading incomplete history", async () => {
    window.sessionStorage.setItem(ACTIVE_SESSION_STORAGE_KEY, JSON.stringify({ chat: "long-chat" }));
    window.localStorage.setItem(
      LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        cachedAt: Date.now(),
        activeSessionByRoute: { chat: "long-chat" },
        sessionsByRoute: {
          chat: [{
            id: "long-chat",
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
              id: "long-chat",
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
        case "/api/chat/sessions/long-chat":
          return {
            session: {
              id: "long-chat",
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

    window.dispatchEvent(new PageTransitionEvent("pageshow", { persisted: true }));

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions"));
    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions/long-chat"));
    expect(apiClientMock.get).not.toHaveBeenCalledWith(
      "/api/chat/sessions/long-chat?turn_before=turn-latest&turn_limit=20",
    );
    expect(screen.getByTestId("message-texts")).toHaveTextContent("latest stable answer");
  });

  it("reloads Chat detail when a newer list summary accidentally includes empty turns", async () => {
    window.sessionStorage.setItem(ACTIVE_SESSION_STORAGE_KEY, JSON.stringify({ chat: "stale-chat" }));
    window.localStorage.setItem(
      LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        cachedAt: Date.now(),
        activeSessionByRoute: { chat: "stale-chat" },
        sessionsByRoute: {
          chat: [{
            id: "stale-chat",
            status: "ready",
            title: "Stale chat",
            createdAt: Date.parse("2026-04-23T03:30:00Z"),
            updatedAt: Date.parse("2026-04-23T03:31:00Z"),
            lastOutputAt: Date.parse("2026-04-23T03:31:00Z"),
            activityAt: Date.parse("2026-04-23T03:31:00Z"),
            revision: Date.parse("2026-04-23T03:31:00Z"),
            detailRevision: Date.parse("2026-04-23T03:31:00Z"),
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
              id: "stale-chat",
              title: "Stale chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              updated_at: "2026-04-23T03:32:00Z",
              last_output_at: "2026-04-23T03:32:00Z",
              activity_at: "2026-04-23T03:32:00Z",
              revision: Date.parse("2026-04-23T03:32:00Z"),
              turns: [],
            }],
          };
        case "/api/chat/sessions/stale-chat":
          return {
            session: {
              id: "stale-chat",
              title: "Stale chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              updated_at: "2026-04-23T03:32:00Z",
              last_output_at: "2026-04-23T03:32:00Z",
              activity_at: "2026-04-23T03:32:00Z",
              revision: Date.parse("2026-04-23T03:32:00Z"),
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

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions/stale-chat"));
    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("new answer"));
  });

  it("reloads Chat detail when a list summary omits revision for an existing cached session", async () => {
    window.sessionStorage.setItem(ACTIVE_SESSION_STORAGE_KEY, JSON.stringify({ chat: "stale-chat" }));
    window.localStorage.setItem(
      LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        cachedAt: Date.now(),
        activeSessionByRoute: { chat: "stale-chat" },
        sessionsByRoute: {
          chat: [{
            id: "stale-chat",
            status: "ready",
            title: "Stale chat",
            createdAt: Date.parse("2026-04-23T03:30:00Z"),
            updatedAt: Date.parse("2026-04-23T03:31:00Z"),
            lastOutputAt: Date.parse("2026-04-23T03:31:00Z"),
            activityAt: Date.parse("2026-04-23T03:31:00Z"),
            revision: Date.parse("2026-04-23T03:31:00Z"),
            detailRevision: Date.parse("2026-04-23T03:31:00Z"),
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
              id: "stale-chat",
              title: "Stale chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              updated_at: "2026-04-23T03:32:00Z",
              last_output_at: "2026-04-23T03:32:00Z",
              activity_at: "2026-04-23T03:32:00Z",
            }],
          };
        case "/api/chat/sessions/stale-chat":
          return {
            session: {
              id: "stale-chat",
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
                final_output: "new answer without summary revision",
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

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions/stale-chat"));
    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("new answer without summary revision"));
  });

  it("does not hydrate expired Chat browser caches", async () => {
    const nowSpy = vi.spyOn(Date, "now").mockReturnValue(1000);
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [{
              id: "alter0-chat",
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
        id: "alter0-chat",
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
                id: "alter0-chat",
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
    apiClientMock.post.mockResolvedValueOnce({ session_id: "alter0-chat", pinned: false });

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
        "/api/chat/sessions/alter0-chat/pin",
        { pinned: false },
      );
    });
    expect(screen.getByTestId("sessions")).toHaveTextContent("unpinned");
  });

  it("moves pinned Chat sessions ahead of newer unpinned sessions", async () => {
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [
              {
                id: "newer-chat",
                title: "Newer session",
                created_at: "2026-04-23T04:30:00Z",
                target_type: "model",
                target_id: "raw-model",
                target_name: "Raw Model",
                pinned: false,
              },
              {
                id: "older-chat",
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
    apiClientMock.post.mockResolvedValueOnce({ session_id: "older-chat", pinned: true });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <SessionListHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("sessions")).toHaveTextContent(/^Newer session:/));

    fireEvent.click(screen.getByRole("button", { name: "pin older" }));

    await waitFor(() => {
      expect(apiClientMock.post).toHaveBeenCalledWith(
        "/api/chat/sessions/older-chat/pin",
        { pinned: true },
      );
    });
    expect(screen.getByTestId("sessions")).toHaveTextContent(/^Older session:[^|]*:pinned\|Newer session:[^|]*:unpinned$/);
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
      "/api/chat/sessions/new-chatRuntime-chat/pin",
      { pinned: true },
    ));
    expect(screen.getByTestId("sessions")).toHaveTextContent(/^New:[^|]*:pinned$/);
  });

  it("selects all public skills by default for a new blank Chat session", async () => {
    window.sessionStorage.clear();
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
        case "/api/control/llm/providers":
        case "/api/control/mcps":
          return { items: [] };
        case "/api/control/skills":
          return {
            items: [
              {
                id: "memory",
                name: "Memory",
                description: "Use workspace memory",
                enabled: true,
              },
              {
                id: "frontend-design",
                name: "Frontend Design",
                description: "UI guidance",
                enabled: true,
              },
              {
                id: "private",
                name: "Private",
                description: "Private skill",
                enabled: true,
                metadata: { "alter0.skill.visibility": "private" },
              },
              {
                id: "disabled-skill",
                name: "Disabled Skill",
                description: "Disabled",
                enabled: false,
              },
            ],
          };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <SkillSelectionHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("memory-skill-state")).toHaveTextContent(/^active$/));
    expect(screen.getByTestId("skill-count")).toHaveTextContent(/^2$/);
  });

  it("selects all public skills by default when a ChatRuntime-backed Chat session has no skill_ids field", async () => {
    let requestBody: Record<string, unknown> | null = null;
    apiClientMock.post.mockImplementation(async (path: string, body?: Record<string, unknown>) => {
      if (path === "/api/chat/sessions/alter0-chat/input") {
        requestBody = body || null;
        return {
          session: {
            id: "alter0-chat",
            title: "Image session",
            status: "ready",
            created_at: "2026-04-23T03:30:00Z",
            turns: [{ id: "turn-default-skills", prompt: "Run with selected skills", status: "success", final_output: "Done" }],
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
                id: "alter0-chat",
                title: "Image session",
                status: "ready",
                created_at: "2026-04-23T03:30:00Z",
                turns: [],
              },
            ],
          };
        case "/api/chat/sessions/alter0-chat":
          return {
            session: {
              id: "alter0-chat",
              title: "Image session",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns: [],
            },
          };
        case "/api/control/skills":
          return {
            items: [
              {
                id: "memory",
                name: "Memory",
                description: "Use workspace memory",
                enabled: true,
              },
              {
                id: "frontend-design",
                name: "Frontend Design",
                description: "UI guidance",
                enabled: true,
              },
            ],
          };
        case "/api/control/llm/providers":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <SkillSelectionHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("memory-skill-state")).toHaveTextContent(/^active$/));
    expect(screen.getByTestId("skill-count")).toHaveTextContent(/^2$/);

    fireEvent.click(screen.getByRole("button", { name: "send with skills" }));

    await waitFor(() => expect(requestBody?.skill_ids).toEqual(["memory", "frontend-design"]));
  });

  it("loads Chat sessions from the Chat route and hydrates them as Chat sessions", async () => {
    window.sessionStorage.setItem(
      ACTIVE_SESSION_STORAGE_KEY,
      JSON.stringify({ chat: hashSessionIDShort("skill-session-2") }),
    );
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [
              {
                id: "skill-session-2",
                title: "Travel Plan",
                created_at: "2026-04-23T09:00:00Z",
                target_type: "skill",
                target_id: "travel",
                target_name: "Travel Skill",
                skill_ids: ["travel-map"],
                messages: [],
              },
            ],
          };
        case "/api/chat/sessions/skill-session-2":
          return {
            session: {
              id: "skill-session-2",
              title: "Travel Plan",
              created_at: "2026-04-23T09:00:00Z",
              target_type: "skill",
              target_id: "travel",
              target_name: "Travel Skill",
              skill_ids: ["travel-map"],
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

  it("opens the latest ChatRuntime-backed Chat session when the route has no explicit session query", async () => {
    window.history.replaceState({}, "", "/chat");
    window.sessionStorage.setItem(
      ACTIVE_SESSION_STORAGE_KEY,
      JSON.stringify({ chat: "older-chat-session" }),
    );
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [
              {
                id: "latest-chat-session",
                title: "Latest chat",
                created_at: "2026-06-11T05:40:00Z",
                target_type: "model",
                target_id: "raw-model",
                target_name: "Raw Model",
                messages: [],
              },
              {
                id: "older-chat-session",
                title: "Older chat",
                created_at: "2026-06-10T05:40:00Z",
                target_type: "model",
                target_id: "raw-model",
                target_name: "Raw Model",
                messages: [],
              },
            ],
          };
        case "/api/chat/sessions/latest-chat-session":
          return {
            session: {
              id: "latest-chat-session",
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
      JSON.stringify({ chat: "older-chat-session" }),
    );
    window.localStorage.setItem(
      LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        cachedAt: Date.now(),
        activeSessionByRoute: { chat: "older-chat-session" },
        sessionsByRoute: {
          chat: [chatSessionFixture({
            id: "older-chat-session",
            title: "Older chat",
            createdAt: Date.parse("2026-06-10T05:40:00Z"),
            activityAt: Date.parse("2026-06-10T05:40:00Z"),
            revision: Date.parse("2026-06-10T05:40:00Z"),
            detailRevision: Date.parse("2026-06-10T05:40:00Z"),
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
        case "/api/chat/sessions/older-chat-session":
          return {
            session: {
              id: "older-chat-session",
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

  it("merges legacy local chat snapshots into the Chat session bucket", async () => {
    window.sessionStorage.clear();
    window.sessionStorage.setItem(
      ACTIVE_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        "chat": {
          id: "legacy-skill-local-1",
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
            id: "legacy-skill-recent-1",
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

    await waitFor(() => expect(screen.getByTestId("sessions")).toHaveTextContent("Legacy local chat"));
    expect(screen.getByTestId("sessions")).toHaveTextContent("Recent local chat");
    expect(window.sessionStorage.getItem(ACTIVE_SESSION_SNAPSHOT_STORAGE_KEY)).toBeNull();
    expect(window.sessionStorage.getItem(RECENT_SESSION_SNAPSHOT_STORAGE_KEY)).toBeNull();
    expect(window.localStorage.getItem(LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY)).toContain("Legacy local chat");
    expect(window.localStorage.getItem(LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY)).toContain("Recent local chat");
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
      "/api/chat/sessions/alter0-chat/attachments",
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

  it("marks the chat session busy without creating local stream process events after sending a prompt", async () => {
    vi.stubGlobal("fetch", vi.fn());
    apiClientMock.post.mockImplementation(async (path: string) => {
      if (path === "/api/chat/sessions/alter0-chat/input") {
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

    await waitFor(() => expect(screen.getByTestId("active-session-status")).toHaveTextContent("busy"));
    expect(screen.getByTestId("assistant-text")).toHaveTextContent("");
    expect(screen.getByTestId("assistant-process-count")).toHaveTextContent("0");
    expect(screen.getByTestId("assistant-process-status")).toHaveTextContent("");
    expect(apiClientMock.post).toHaveBeenCalledWith("/api/chat/sessions/alter0-chat/input", expect.any(Object));
    expect(fetch).not.toHaveBeenCalled();
  });

  it("caches and appends the optimistic Chat user message before the input response completes", async () => {
    const inputResponse = deferred<unknown>();
    apiClientMock.post.mockImplementation(async (path: string) => {
      if (path === "/api/chat/sessions/alter0-chat/input") {
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

    await waitFor(() => expect(screen.getByTestId("user-text")).toHaveTextContent("Inspect this image"));
    expect(screen.getByTestId("active-session-status")).toHaveTextContent("busy");
    expect(window.localStorage.getItem(LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY) || "").toContain("Inspect this image");

    inputResponse.resolve({
      session: {
        id: "alter0-chat",
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
  });

  it("compacts the queued optimistic Chat user message when the input response only confirms a busy turn", async () => {
    apiClientMock.post.mockImplementation(async (path: string) => {
      if (path === "/api/chat/sessions/alter0-chat/input") {
        return {
          session: {
            id: "alter0-chat",
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
              id: "alter0-chat",
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
      if (path === "/api/chat/sessions/alter0-chat/input") {
        return {
          session: {
            id: "alter0-chat",
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
              id: "alter0-chat",
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
        case "/api/chat/sessions/alter0-chat":
          return {
            session: {
              id: "alter0-chat",
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
              id: "alter0-chat",
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
        case "/api/chat/sessions/alter0-chat?turn_before=turn-3&turn_limit=20":
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
      "/api/chat/sessions/alter0-chat?turn_before=turn-3&turn_limit=20",
    ));
    await act(async () => {
      earlierPage.resolve({
        session: {
          id: "alter0-chat",
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
              id: "alter0-chat",
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
      if (path === "/api/chat/sessions/alter0-chat/input") {
        return {
          session: {
            id: "alter0-chat",
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
        const session = inputAccepted && updateCallsAfterInput > 0
          ? {
              id: "alter0-chat",
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
                  runtime_trace_events_partial: true,
                  runtime_trace_events: [
                    {
                      id: "step-3",
                      seq: 3,
                      turn_id: "turn-running",
                      kind: "reasoning",
                      status: "completed",
                      blocks: [],
                    },
                  ],
                },
              ],
            }
          : {
              id: "alter0-chat",
              title: "Image session",
              status: inputAccepted ? "busy" : "ready",
              created_at: "2026-04-23T03:30:00Z",
              turns: inputAccepted
                ? [
                    {
                      id: "turn-running",
                      prompt: "Inspect this image",
                      status: "running",
                      started_at: "2026-04-23T03:31:00Z",
                    },
                  ]
                : [],
            };
        return {
          owner_id: "chat",
          cursor: updateCallsAfterInput,
          resync_required: false,
          events: inputAccepted ? [{
            event_id: updateCallsAfterInput,
            owner_id: "chat",
            session_id: "alter0-chat",
            event_type: "session.updated",
            revision: updateCallsAfterInput,
            created_at: "2026-04-23T03:31:00Z",
            payload: { session },
          }] : [],
        };
      }
      if (path === "/api/chat/sessions/alter0-chat/input") {
        inputAccepted = true;
        return {
          session: {
            id: "alter0-chat",
            title: "Image session",
            status: "busy",
            created_at: "2026-04-23T03:30:00Z",
            turns: [
              {
                id: "turn-running",
                prompt: typeof body?.input === "string" ? body.input : "Inspect this image",
                status: "running",
                started_at: "2026-04-23T03:31:00Z",
                runtime_trace_events: [
                  {
                    id: "step-1",
                    seq: 1,
                    turn_id: "turn-running",
                    kind: "reasoning",
                    status: "completed",
                    blocks: [],
                  },
                  {
                    id: "step-2",
                    seq: 2,
                    turn_id: "turn-running",
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
                id: "alter0-chat",
                title: "Image session",
                status: inputAccepted ? "busy" : "ready",
                created_at: "2026-04-23T03:30:00Z",
                turns: [],
              },
            ],
          };
        case "/api/chat/sessions/alter0-chat":
          return {
            session: inputAccepted
              ? updateCallsAfterInput > 0
                ? {
                  id: "alter0-chat",
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
                    id: "alter0-chat",
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
                  id: "alter0-chat",
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
    expect(screen.getByTestId("active-session-status")).toHaveTextContent("busy");

    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
      await Promise.resolve();
    });

    expect(EventSourceMock).not.toHaveBeenCalled();
    expect(apiClientMock.post).toHaveBeenCalledWith(
      "/api/chat/sessions/updates",
      expect.objectContaining({
        limit: 50,
        byte_limit: 65536,
      }),
    );
    expect(updateRequestBodies).toContainEqual(expect.objectContaining({
      sessions: expect.arrayContaining([
        expect.objectContaining({
          id: "alter0-chat",
          turns: expect.arrayContaining([
            expect.objectContaining({
              id: "turn-running",
              event_seq_ranges: [[1, 2]],
            }),
          ]),
        }),
      ]),
    }));
    expect(screen.getByTestId("assistant-text")).toHaveTextContent("Restored final output");
    expect(screen.getByTestId("assistant-process-count")).toHaveTextContent("3");
    expect(screen.getByTestId("assistant-process-ids")).toHaveTextContent("step-1|step-2|step-3");
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
          owner_id: "chat",
          cursor: updateCallsAfterInput,
          resync_required: false,
          events: [],
        };
      }
      if (path === "/api/chat/sessions/alter0-chat/input") {
        inputAccepted = true;
        return {
          session: {
            id: "alter0-chat",
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
                id: "alter0-chat",
                title: "Fallback session",
                status: inputAccepted ? "busy" : "ready",
                created_at: "2026-04-23T03:30:00Z",
                turns: [],
              },
            ],
          };
        case "/api/chat/sessions/alter0-chat":
          if (inputAccepted) {
            detailReadsAfterInput += 1;
          }
          return {
            session: inputAccepted
              ? {
                  id: "alter0-chat",
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
                  id: "alter0-chat",
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

    expect(updateCallsAfterInput).toBeGreaterThanOrEqual(8);
    expect(detailReadsAfterInput).toBe(0);

    await advanceRuntimePollTimers(2);

    expect(updateCallsAfterInput).toBeGreaterThanOrEqual(10);
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
          owner_id: "chat",
          cursor: updateCallsAfterInput,
          resync_required: false,
          events: inputAccepted
            ? [{
                event_id: updateCallsAfterInput,
                owner_id: "chat",
                session_id: "alter0-chat",
                event_type: "session.updated",
                revision: Date.parse("2026-04-23T03:31:00Z") + updateCallsAfterInput,
                created_at: "2026-04-23T03:31:00Z",
                payload: {
                  session: {
                    id: "alter0-chat",
                    title: "Long running session",
                    status: "busy",
                    created_at: "2026-04-23T03:30:00Z",
                    updated_at: `2026-04-23T03:31:${String(updateCallsAfterInput).padStart(2, "0")}Z`,
                    revision: Date.parse("2026-04-23T03:31:00Z") + updateCallsAfterInput,
                    turns: [
                      {
                        id: "turn-running",
                        prompt: typeof body?.input === "string" ? body.input : "Still running",
                        status: "running",
                        started_at: "2026-04-23T03:31:00Z",
                      },
                    ],
                  },
                },
              }]
            : [],
        };
      }
      if (path === "/api/chat/sessions/alter0-chat/input") {
        inputAccepted = true;
        return {
          session: {
            id: "alter0-chat",
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
                id: "alter0-chat",
                title: "Long running session",
                status: inputAccepted ? "busy" : "ready",
                created_at: "2026-04-23T03:30:00Z",
                turns: [],
              },
            ],
          };
        case "/api/chat/sessions/alter0-chat":
          if (inputAccepted) {
            detailReadsAfterInput += 1;
          }
          return {
            session: {
              id: "alter0-chat",
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
          owner_id: "chat",
          cursor: updateCallsAfterInput,
          resync_required: false,
          has_more: true,
          events: inputAccepted
            ? [{
                event_id: updateCallsAfterInput,
                owner_id: "chat",
                session_id: "stale-chat",
                event_type: "session.updated",
                revision: updateCallsAfterInput,
                created_at: "2026-04-23T03:29:00Z",
                payload: {
                  session: {
                    id: "stale-chat",
                    title: "Stale backlog",
                    status: "ready",
                    created_at: "2026-04-23T03:00:00Z",
                    turns: [{
                      id: "turn-stale",
                      prompt: "old prompt",
                      status: "success",
                      final_output: "old answer",
                    }],
                  },
                },
              }]
            : [],
        };
      }
      if (path === "/api/chat/sessions/alter0-chat/input") {
        inputAccepted = true;
        return {
          session: {
            id: "alter0-chat",
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
                id: "alter0-chat",
                title: "Backlog fallback session",
                status: inputAccepted ? "busy" : "ready",
                created_at: "2026-04-23T03:30:00Z",
                turns: [],
              },
            ],
          };
        case "/api/chat/sessions/alter0-chat":
          if (inputAccepted) {
            detailReadsAfterInput += 1;
          }
          return {
            session: inputAccepted
              ? {
                  id: "alter0-chat",
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
                  id: "alter0-chat",
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

    expect(updateCallsAfterInput).toBeGreaterThanOrEqual(8);
    expect(detailReadsAfterInput).toBe(0);

    await advanceRuntimePollTimers(2);

    expect(updateCallsAfterInput).toBeGreaterThanOrEqual(10);
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
          owner_id: "chat",
          cursor: updateCallsAfterInput,
          resync_required: false,
          has_more: true,
          events: inputAccepted
            ? [{
                event_id: updateCallsAfterInput,
                owner_id: "chat",
                session_id: "alter0-chat",
                event_type: "session.updated",
                revision: updateCallsAfterInput,
                created_at: "2026-04-23T03:31:00Z",
                payload: {
                  session: {
                    id: "alter0-chat",
                    title: "Repeated busy backlog",
                    status: "busy",
                    created_at: "2026-04-23T03:30:00Z",
                    turns: [
                      {
                        id: "turn-running",
                        prompt: typeof body?.input === "string" ? body.input : "Recover from repeated busy",
                        status: "running",
                        started_at: "2026-04-23T03:31:00Z",
                      },
                    ],
                  },
                },
              }]
            : [],
        };
      }
      if (path === "/api/chat/sessions/alter0-chat/input") {
        inputAccepted = true;
        return {
          session: {
            id: "alter0-chat",
            title: "Repeated busy backlog",
            status: "busy",
            created_at: "2026-04-23T03:30:00Z",
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
                id: "alter0-chat",
                title: "Repeated busy backlog",
                status: inputAccepted ? "ready" : "ready",
                created_at: "2026-04-23T03:30:00Z",
                updated_at: "2026-04-23T03:33:00Z",
                last_output_at: inputAccepted ? "2026-04-23T03:33:00Z" : "",
                turns: [],
              },
            ],
          };
        case "/api/chat/sessions/alter0-chat":
          if (inputAccepted) {
            detailReadsAfterInput += 1;
          }
          return {
            session: inputAccepted
              ? {
                  id: "alter0-chat",
                  title: "Repeated busy backlog",
                  status: "ready",
                  created_at: "2026-04-23T03:30:00Z",
                  updated_at: "2026-04-23T03:33:00Z",
                  last_output_at: "2026-04-23T03:33:00Z",
                  turns: [
                    {
                      id: "turn-running",
                      prompt: "Recover from repeated busy",
                      status: "success",
                      started_at: "2026-04-23T03:31:00Z",
                      finished_at: "2026-04-23T03:33:00Z",
                      final_output: "Recovered despite repeated busy backlog",
                    },
                  ],
                }
              : {
                  id: "alter0-chat",
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

    expect(updateCallsAfterInput).toBeGreaterThanOrEqual(8);
    expect(detailReadsAfterInput).toBe(0);

    await advanceRuntimePollTimers(2);

    expect(updateCallsAfterInput).toBeGreaterThanOrEqual(10);
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
        case "/api/chat/sessions/alter0-chat":
          return {
            items: [
              {
                id: "alter0-chat",
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
              id: "alter0-chat",
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

    expect(apiClientMock.post).not.toHaveBeenCalledWith("/api/chat/sessions/alter0-chat/input", expect.any(Object));
    expect(screen.getByTestId("user-text")).toHaveTextContent("成都旅游攻略");
    expect(screen.getByTestId("assistant-text")).toHaveTextContent("");
  });

  it("allows sending another Chat input after a ChatRuntime-backed session fails without assistant output", async () => {
    vi.stubGlobal("fetch", vi.fn());
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
        case "/api/chat/sessions/alter0-chat":
          return {
            items: [
              {
                id: "alter0-chat",
                title: "Failed session",
                status: "failed",
                created_at: "2026-04-23T03:30:00Z",
                turns: [
                  {
                    id: "turn-failed",
                    prompt: "previous prompt",
                    status: "failed",
                    started_at: "2026-04-23T03:31:00Z",
                    finished_at: "2026-04-23T03:31:02Z",
                  },
                ],
              },
            ],
            session: {
              id: "alter0-chat",
              title: "Failed session",
              status: "failed",
              created_at: "2026-04-23T03:30:00Z",
              turns: [
                {
                  id: "turn-failed",
                  prompt: "previous prompt",
                  status: "failed",
                  started_at: "2026-04-23T03:31:00Z",
                  finished_at: "2026-04-23T03:31:02Z",
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
    apiClientMock.post.mockImplementation(async (path: string, body?: Record<string, unknown>) => {
      if (path === "/api/chat/sessions/alter0-chat/input") {
        return {
          session: {
            id: "alter0-chat",
            title: "Failed session",
            status: "ready",
            created_at: "2026-04-23T03:30:00Z",
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

    await waitFor(() => expect(screen.getByTestId("active-session-status")).toHaveTextContent("failed"));

    fireEvent.click(screen.getByRole("button", { name: "send" }));

    await waitFor(() => expect(apiClientMock.post).toHaveBeenCalledWith(
      "/api/chat/sessions/alter0-chat/input",
      expect.objectContaining({ input: "Inspect this image" }),
    ));
    await waitFor(() => expect(screen.getByTestId("assistant-text")).toHaveTextContent("Recovered reply"));
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

    fireEvent.click(screen.getByRole("button", { name: "capabilities" }));
    expect(screen.getByTestId("inspector-state")).toHaveTextContent("capabilities:details-open:tab-open");
  });

  it("adds a Codex option for Chat model selection and submits through ChatRuntime input", async () => {
    vi.stubGlobal("fetch", vi.fn());
    let requestBody: Record<string, unknown> | null = null;
    apiClientMock.post.mockImplementation(async (path: string, body?: Record<string, unknown>) => {
      if (path === "/api/chat/sessions/alter0-chat/input") {
        requestBody = body || null;
        return {
          session: {
            id: "alter0-chat",
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
    expect(apiClientMock.post).toHaveBeenCalledWith("/api/chat/sessions/alter0-chat/input", expect.any(Object));
    expect(fetch).not.toHaveBeenCalled();
  });

  it("defaults the executor selection to Codex and restores changed runtime config from browser storage", async () => {
    let requestBody: Record<string, unknown> | null = null;
    apiClientMock.post.mockImplementation(async (path: string, body?: Record<string, unknown>) => {
      if (path === "/api/chat/sessions/alter0-chat/input") {
        requestBody = body || null;
        return {
          session: {
            id: "alter0-chat",
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
                id: "alter0-chat",
                title: "Stored config session",
                status: "ready",
                created_at: "2026-04-23T03:30:00Z",
                turns: [],
              },
            ],
          };
        case "/api/chat/sessions/alter0-chat":
          return {
            session: {
              id: "alter0-chat",
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
        case "/api/control/skills":
          return {
            items: [
              { id: "memory", name: "Memory", description: "Use workspace memory", enabled: true },
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
    expect(screen.getByTestId("memory-skill-state")).toHaveTextContent("active");

    fireEvent.click(screen.getByRole("button", { name: "select openrouter" }));
    fireEvent.click(screen.getByRole("button", { name: "enable filesystem" }));
    fireEvent.click(screen.getByRole("button", { name: "disable memory" }));

    await waitFor(() => expect(screen.getByTestId("selected-provider")).toHaveTextContent("openrouter"));
    expect(screen.getByTestId("filesystem-state")).toHaveTextContent("active");
    expect(screen.getByTestId("memory-skill-state")).toHaveTextContent("inactive");

    firstView.unmount();

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeConfigSelectionHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("selected-provider")).toHaveTextContent("openrouter"));
    expect(screen.getByTestId("selected-model")).toHaveTextContent("anthropic/claude-sonnet");
    expect(screen.getByTestId("filesystem-state")).toHaveTextContent("active");
    expect(screen.getByTestId("memory-skill-state")).toHaveTextContent("inactive");

    fireEvent.click(screen.getByRole("button", { name: "send with stored config" }));

    await waitFor(() => expect(requestBody?.input).toBe("Run with stored config"));
    expect(requestBody?.model_provider_id).toBe("openrouter");
    expect(requestBody?.model_id).toBe("anthropic/claude-sonnet");
    expect(requestBody?.mcp_ids).toEqual(["filesystem"]);
    expect(requestBody?.skill_ids).toEqual([]);
  });

  it("persists Chat skill selections to the runtime session before the next message is sent", async () => {
    let requestBody: Record<string, unknown> | null = null;
    apiClientMock.post.mockImplementation(async (path: string, body?: Record<string, unknown>) => {
      if (path === "/api/chat/sessions/alter0-chat/input") {
        requestBody = body || null;
        return {
          session: {
            id: "alter0-chat",
            title: "Configurable session",
            status: "ready",
            created_at: "2026-04-23T03:30:00Z",
            skill_ids: ["memory"],
            turns: [{ id: "turn-memory", prompt: "Run with selected skills", status: "success", final_output: "Done" }],
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
                id: "alter0-chat",
                title: "Configurable session",
                created_at: "2026-04-23T03:30:00Z",
                target_type: "model",
                target_id: "raw-model",
                target_name: "Raw Model",
                skill_ids: [],
              },
            ],
          };
        case "/api/chat/sessions/alter0-chat":
          return {
            session: {
              id: "alter0-chat",
              title: "Configurable session",
              created_at: "2026-04-23T03:30:00Z",
              target_type: "model",
              target_id: "raw-model",
              target_name: "Raw Model",
              skill_ids: [],
              messages: [],
            },
          };
        case "/api/control/skills":
          return {
            items: [
              {
                id: "memory",
                name: "Memory",
                description: "Use workspace memory",
                enabled: true,
              },
            ],
          };
        case "/api/control/llm/providers":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });
    apiClientMock.patch.mockResolvedValue({
      session: {
        id: "alter0-chat",
        title: "Configurable session",
        created_at: "2026-04-23T03:30:00Z",
        target_type: "model",
        target_id: "raw-model",
        target_name: "Raw Model",
        skill_ids: ["memory"],
        messages: [],
      },
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <SkillSelectionHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("memory-skill-state")).toHaveTextContent("inactive"));

    fireEvent.click(screen.getByRole("button", { name: "enable memory" }));

    await waitFor(() => expect(screen.getByTestId("memory-skill-state")).toHaveTextContent("active"));
    expect(screen.getByTestId("skill-count")).toHaveTextContent("1");
    expect(apiClientMock.patch).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole("button", { name: "send with skills" }));

    await waitFor(() => expect(requestBody?.skill_ids).toEqual(["memory"]));
  });

  it("keeps cleared Chat skill selections cleared after the runtime session patch response", async () => {
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
        case "/api/chat/sessions/alter0-chat":
          return {
            session: {
              id: "alter0-chat",
              title: "Configurable session",
              created_at: "2026-04-23T03:30:00Z",
              target_type: "model",
              target_id: "raw-model",
              target_name: "Raw Model",
              skill_ids: ["memory"],
              messages: [],
            },
            items: [
              {
                id: "alter0-chat",
                title: "Configurable session",
                created_at: "2026-04-23T03:30:00Z",
                target_type: "model",
                target_id: "raw-model",
                target_name: "Raw Model",
                skill_ids: ["memory"],
              },
            ],
          };
        case "/api/control/skills":
          return {
            items: [
              {
                id: "memory",
                name: "Memory",
                description: "Use workspace memory",
                enabled: true,
              },
            ],
          };
        case "/api/control/llm/providers":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });
    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <SkillSelectionHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("memory-skill-state")).toHaveTextContent("active"));

    fireEvent.click(screen.getByRole("button", { name: "disable memory" }));

    await waitFor(() => expect(screen.getByTestId("memory-skill-state")).toHaveTextContent("inactive"));
    expect(screen.getByTestId("skill-count")).toHaveTextContent("0");
    expect(apiClientMock.patch).not.toHaveBeenCalled();
  });

  it("drops unavailable skills from historical Chat sessions before the next message is sent", async () => {
    let requestBody: Record<string, unknown> | null = null;
    apiClientMock.post.mockImplementation(async (path: string, body?: Record<string, unknown>) => {
      if (path === "/api/chat/sessions/alter0-chat/input") {
        requestBody = body || null;
        return {
          session: {
            id: "alter0-chat",
            title: "Historical session",
            status: "ready",
            created_at: "2026-04-23T03:30:00Z",
            skill_ids: ["memory"],
            turns: [{ id: "turn-filtered-skill", prompt: "Run with selected skills", status: "success", final_output: "Done" }],
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
                id: "alter0-chat",
                title: "Historical session",
                created_at: "2026-04-23T03:30:00Z",
                target_type: "model",
                target_id: "raw-model",
                target_name: "Raw Model",
                skill_ids: ["memory", "deleted-skill"],
              },
            ],
          };
        case "/api/chat/sessions/alter0-chat":
          return {
            session: {
              id: "alter0-chat",
              title: "Historical session",
              created_at: "2026-04-23T03:30:00Z",
              target_type: "model",
              target_id: "raw-model",
              target_name: "Raw Model",
              skill_ids: ["memory", "deleted-skill"],
              messages: [],
            },
          };
        case "/api/control/skills":
          return {
            items: [
              {
                id: "memory",
                name: "Memory",
                description: "Use workspace memory",
                enabled: true,
              },
            ],
          };
        case "/api/control/llm/providers":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <SkillSelectionHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("memory-skill-state")).toHaveTextContent(/^active$/));
    expect(screen.getByTestId("skill-count")).toHaveTextContent(/^1$/);

    fireEvent.click(screen.getByRole("button", { name: "send with skills" }));

    await waitFor(() => expect(requestBody?.skill_ids).toEqual(["memory"]));
  });

  it("sends newly checked skills from historical Chat sessions without a reload", async () => {
    let requestBody: Record<string, unknown> | null = null;
    apiClientMock.post.mockImplementation(async (path: string, body?: Record<string, unknown>) => {
      if (path === "/api/chat/sessions/alter0-chat/input") {
        requestBody = body || null;
        return {
          session: {
            id: "alter0-chat",
            title: "Historical session",
            status: "ready",
            created_at: "2026-04-23T03:30:00Z",
            skill_ids: ["memory"],
            turns: [{ id: "turn-new-skill", prompt: "Run with selected skills", status: "success", final_output: "Done" }],
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
                id: "alter0-chat",
                title: "Historical session",
                created_at: "2026-04-23T03:30:00Z",
                target_type: "model",
                target_id: "raw-model",
                target_name: "Raw Model",
                skill_ids: [],
              },
            ],
          };
        case "/api/chat/sessions/alter0-chat":
          return {
            session: {
              id: "alter0-chat",
              title: "Historical session",
              created_at: "2026-04-23T03:30:00Z",
              target_type: "model",
              target_id: "raw-model",
              target_name: "Raw Model",
              skill_ids: [],
              messages: [],
            },
          };
        case "/api/control/skills":
          return {
            items: [
              {
                id: "memory",
                name: "Memory",
                description: "Use workspace memory",
                enabled: true,
              },
            ],
          };
        case "/api/control/llm/providers":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });
    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <SkillSelectionHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("memory-skill-state")).toHaveTextContent(/^inactive$/));
    fireEvent.click(screen.getByRole("button", { name: "enable memory" }));
    await waitFor(() => expect(screen.getByTestId("memory-skill-state")).toHaveTextContent(/^active$/));

    fireEvent.click(screen.getByRole("button", { name: "send with skills" }));

    await waitFor(() => expect(requestBody?.skill_ids).toEqual(["memory"]));
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
              id: "alter0-chat",
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
      if (path === "/api/chat/sessions/alter0-chat/input") {
        return {
          session: {
            id: "alter0-chat",
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

  it("merges paged Chat session detail refreshes into existing messages", async () => {
    let detailReads = 0;
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [{
              id: "alter0-chat",
              title: "Paged chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              updated_at: "2026-04-23T03:32:03Z",
              last_output_at: "2026-04-23T03:32:03Z",
              activity_at: "2026-04-23T03:32:03Z",
              revision: Date.parse("2026-04-23T03:32:03Z"),
              turns_paging: { has_more_before: true },
            }],
          };
        case "/api/chat/sessions/alter0-chat":
          detailReads += 1;
          if (detailReads === 1) {
            return {
              session: {
                id: "alter0-chat",
                title: "Paged chat",
                status: "ready",
                created_at: "2026-04-23T03:30:00Z",
                updated_at: "2026-04-23T03:32:01Z",
                last_output_at: "2026-04-23T03:32:01Z",
                activity_at: "2026-04-23T03:32:01Z",
                revision: Date.parse("2026-04-23T03:32:01Z"),
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
              id: "alter0-chat",
              title: "Paged chat",
              status: "ready",
              created_at: "2026-04-23T03:30:00Z",
              updated_at: "2026-04-23T03:32:03Z",
              last_output_at: "2026-04-23T03:32:03Z",
              activity_at: "2026-04-23T03:32:03Z",
              revision: Date.parse("2026-04-23T03:32:03Z"),
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

    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("older answer"));
    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("newer answer"));

    window.dispatchEvent(new PageTransitionEvent("pageshow", { persisted: true }));

    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("newer answer refreshed"));
    expect(screen.getByTestId("message-texts")).toHaveTextContent("older answer");
  });

  it("refreshes the active Chat session on demand so paged history can continue loading", async () => {
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [{
              id: "alter0-chat",
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
        case "/api/chat/sessions/alter0-chat":
          return {
            session: {
              id: "alter0-chat",
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

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions/alter0-chat"));
    await waitFor(() => expect(screen.getByTestId("message-texts")).toHaveTextContent("older answer"));
    expect(screen.getByTestId("message-texts")).toHaveTextContent("newer answer");
  });

  it("does not visibly refresh Chat with earlier turn pages until the user asks for history", async () => {
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [{
              id: "alter0-chat",
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
        case "/api/chat/sessions/alter0-chat?turn_before=turn-3&turn_limit=20":
          return {
            session: {
              id: "alter0-chat",
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
      "/api/chat/sessions/alter0-chat?turn_before=turn-3&turn_limit=20",
    );
    expect(screen.getByTestId("message-texts")).not.toHaveTextContent("oldest answer");

    fireEvent.click(screen.getByRole("button", { name: "load earlier history" }));

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith(
      "/api/chat/sessions/alter0-chat?turn_before=turn-3&turn_limit=20",
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
              id: "alter0-chat",
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
        case "/api/chat/sessions/alter0-chat?turn_before=turn-3&turn_limit=20":
          backgroundPageLoads += 1;
          if (backgroundPageLoads > 1) {
            throw new Error("reloaded the same Chat history page");
          }
          return {
            session: {
              id: "alter0-chat",
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
    window.sessionStorage.setItem(ACTIVE_SESSION_STORAGE_KEY, JSON.stringify({ chat: "cached-chat" }));
    window.localStorage.setItem(
      SESSION_INFO_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        cachedAt: Date.now(),
        activeSessionByRoute: { chat: "cached-chat" },
        sessionsByRoute: {
          chat: [{
            id: "cached-chat",
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
    window.sessionStorage.setItem(ACTIVE_SESSION_STORAGE_KEY, JSON.stringify({ chat: "full-chat" }));
    window.localStorage.setItem(
      LONG_TERM_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        cachedAt: Date.now(),
        activeSessionByRoute: { chat: "full-chat" },
        sessionsByRoute: {
          chat: [{
            id: "full-chat",
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
        activeSessionByRoute: { chat: "full-chat" },
        sessionsByRoute: {
          chat: [
            {
              id: "full-chat",
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
              id: "info-chat",
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

  it("loads Chat runtime process event details on demand and keeps them in the message cache", async () => {
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/chat/sessions":
          return {
            items: [{
              id: "alter0-chat",
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
                  id: "step-1",
                  turn_id: "turn-1",
                  seq: 1,
                  source: "adapter",
                  provider: { engine: "codex", adapter: "codex_cli_json" },
                  role: "assistant",
                  kind: "reasoning",
                  lifecycle: "completed",
                  status: "completed",
                  title: "Thinking",
                  blocks: [],
                  visibility: "collapsed",
                  raw: { ref: "event-ref-1", has_detail: true },
                }],
              }],
            }],
          };
        case "/api/chat/sessions/alter0-chat":
          return {
            session: {
              id: "alter0-chat",
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
                  id: "step-1",
                  turn_id: "turn-1",
                  seq: 1,
                  source: "adapter",
                  provider: { engine: "codex", adapter: "codex_cli_json" },
                  role: "assistant",
                  kind: "reasoning",
                  lifecycle: "completed",
                  status: "completed",
                  title: "Thinking",
                  blocks: [],
                  visibility: "collapsed",
                  raw: { ref: "event-ref-1", has_detail: true },
                }],
              }],
              turns_paging: { has_more_before: false },
            },
          };
        case "/api/chat/sessions/alter0-chat/turns/turn-1/events/event-ref-1":
          return {
            event: {
              turn_id: "turn-1",
              event: {
                id: "step-1",
                turn_id: "turn-1",
                seq: 1,
                source: "adapter",
                provider: { engine: "codex", adapter: "codex_cli_json" },
                role: "assistant",
                kind: "reasoning",
                lifecycle: "completed",
                status: "completed",
                title: "Thinking",
                blocks: [{ type: "thinking", text: "full thinking detail" }],
                visibility: "collapsed",
                raw: { ref: "event-ref-1", has_detail: true },
              },
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
      "/api/chat/sessions/alter0-chat/turns/turn-1/events/event-ref-1",
    ));
    await waitFor(() => expect(screen.getByTestId("assistant-process-blocks")).toHaveTextContent("full thinking detail"));

    fireEvent.click(screen.getByRole("button", { name: "refresh active" }));
    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/chat/sessions/alter0-chat"));
    expect(screen.getByTestId("assistant-process-blocks")).toHaveTextContent("full thinking detail");
    await waitFor(() =>
      expect(window.localStorage.getItem("alter0.web.session.long_term_snapshot.v1")).toContain("full thinking detail"),
    );
  });
});

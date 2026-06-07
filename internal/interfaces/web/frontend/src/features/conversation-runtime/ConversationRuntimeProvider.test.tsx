import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import {
  ConversationRuntimeProvider,
  useConversationRuntime,
  useConversationRuntimeComposer,
  useConversationRuntimeWorkspace,
} from "./ConversationRuntimeProvider";
import { hashSessionIDShort } from "../../shared/session/sessionHash";

const ACTIVE_SESSION_STORAGE_KEY = "alter0.web.session.active.v1";
const ACTIVE_SESSION_SNAPSHOT_STORAGE_KEY = "alter0.web.session.snapshot.v1";
const RECENT_SESSION_SNAPSHOT_STORAGE_KEY = "alter0.web.session.recent.v1";
const COMPOSER_ATTACHMENT_DRAFT_STORAGE_KEY = "alter0.web.composer.attachments.v1";

const apiClientMock = {
  get: vi.fn(async () => ({ items: [] })),
  post: vi.fn(),
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
      <output data-testid="assistant-text">{assistantMessage?.text || ""}</output>
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

function SessionListHarness() {
  const runtime = useConversationRuntimeWorkspace();
  return (
    <output data-testid="sessions">
      {runtime.sessionItems.map((session) => `${session.title}:${session.shortHash}`).join("|")}
    </output>
  );
}

function setupDefaultAPI() {
  apiClientMock.get.mockImplementation(async (path: string) => {
    switch (path) {
      case "/api/conversation-runtime/sessions?route=chat":
        return {
          items: [
            {
              id: "alter0-chat",
              title: "Image session",
              title_auto: false,
              title_score: 1,
              created_at: "2026-04-23T03:30:00Z",
              target_type: "model",
              target_id: "raw-model",
              target_name: "Raw Model",
              model_provider_id: "",
              model_id: "",
              tool_ids: [],
              skill_ids: [],
              mcp_ids: [],
            },
          ],
        };
      case "/api/conversation-runtime/sessions/alter0-chat?route=chat":
        return {
          session: {
            id: "alter0-chat",
            title: "Image session",
            title_auto: false,
            title_score: 1,
            created_at: "2026-04-23T03:30:00Z",
            target_type: "model",
            target_id: "raw-model",
            target_name: "Raw Model",
            model_provider_id: "",
            model_id: "",
            tool_ids: [],
            skill_ids: [],
            mcp_ids: [],
            messages: [],
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

function mockStreamDone(output = "Done") {
  vi.stubGlobal("fetch", vi.fn(async () => ({
    ok: true,
    body: {
      getReader: () => {
        const events = [
          new TextEncoder().encode(`event: done\ndata: ${JSON.stringify({ result: { output } })}\n\n`),
        ];
        return {
          read: vi.fn(async () => {
            const value = events.shift();
            return value ? { done: false, value } : { done: true, value: undefined };
          }),
        };
      },
    },
  })));
}

describe("ConversationRuntimeProvider", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.sessionStorage.clear();
    window.history.replaceState({}, "", "/chat");
    window.sessionStorage.setItem(
      ACTIVE_SESSION_STORAGE_KEY,
      JSON.stringify({ chat: "alter0-chat", "agent-runtime": "" }),
    );
    setupDefaultAPI();
    apiClientMock.post.mockImplementation(async (path: string) => {
      if (path.endsWith("/attachments")) {
        return {
          items: [
            {
              id: "uploaded-image-1",
              name: "trace.png",
              content_type: "image/png",
              size: 12,
              asset_url: "/api/sessions/alter0-chat/attachments/uploaded-image-1/original",
              preview_url: "/api/sessions/alter0-chat/attachments/uploaded-image-1/preview",
            },
          ],
        };
      }
      return {};
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    window.history.replaceState({}, "", "/");
  });

  it("creates blank Chat sessions and normalizes legacy agent-runtime route props to Chat", async () => {
    window.sessionStorage.clear();
    apiClientMock.get.mockImplementation(async () => ({ items: [] }));

    const chatView = render(
      <ConversationRuntimeProvider route="chat" language="en">
        <ActiveSessionTitleHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("active-session-title")).toHaveTextContent("New"));
    chatView.unmount();

    render(
      <ConversationRuntimeProvider route="agent-runtime" language="en">
        <ActiveSessionTitleHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("active-session-title")).toHaveTextContent("New"));
    expect(apiClientMock.get).toHaveBeenCalledWith("/api/conversation-runtime/sessions?route=chat");
    expect(apiClientMock.get).not.toHaveBeenCalledWith("/api/conversation-runtime/sessions?route=agent-runtime");
  });

  it("loads migrated legacy agent sessions from the Chat route and hydrates them as Chat sessions", async () => {
    window.sessionStorage.setItem(
      ACTIVE_SESSION_STORAGE_KEY,
      JSON.stringify({ chat: hashSessionIDShort("agent-session-2"), "agent-runtime": "agent-session-2" }),
    );
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/conversation-runtime/sessions?route=chat":
          return {
            items: [
              {
                id: "agent-session-2",
                title: "Travel Plan",
                created_at: "2026-04-23T09:00:00Z",
                target_type: "agent",
                target_id: "travel",
                target_name: "Travel Planner",
                skill_ids: ["travel-map"],
                messages: [],
              },
            ],
          };
        case "/api/conversation-runtime/sessions/agent-session-2?route=chat":
          return {
            session: {
              id: "agent-session-2",
              title: "Travel Plan",
              created_at: "2026-04-23T09:00:00Z",
              target_type: "agent",
              target_id: "travel",
              target_name: "Travel Planner",
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
    expect(apiClientMock.get).not.toHaveBeenCalledWith("/api/conversation-runtime/sessions?route=agent-runtime");
  });

  it("merges legacy local agent-runtime snapshots into the Chat session bucket", async () => {
    window.sessionStorage.clear();
    window.sessionStorage.setItem(
      ACTIVE_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        "agent-runtime": {
          id: "legacy-agent-local-1",
          title: "Legacy local agent",
          createdAt: Date.parse("2026-04-23T09:00:00Z"),
          target: { type: "agent", id: "travel", name: "Travel Planner" },
          messages: [],
        },
      }),
    );
    window.sessionStorage.setItem(
      RECENT_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        "agent-runtime": [
          {
            id: "legacy-agent-recent-1",
            title: "Recent local agent",
            createdAt: Date.parse("2026-04-22T09:00:00Z"),
            target: { type: "agent", id: "writing", name: "Writing Agent" },
            messages: [],
          },
        ],
      }),
    );
    apiClientMock.get.mockImplementation(async (path: string) => {
      if (path === "/api/conversation-runtime/sessions?route=chat") {
        return { items: [] };
      }
      return { items: [] };
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <SessionListHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("sessions")).toHaveTextContent("Legacy local agent"));
    expect(screen.getByTestId("sessions")).toHaveTextContent("Recent local agent");
  });

  it("uploads draft images into the active Chat session workspace", async () => {
    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/conversation-runtime/sessions/alter0-chat?route=chat"));
    fireEvent.click(screen.getByRole("button", { name: "attach" }));

    await waitFor(() => expect(apiClientMock.post).toHaveBeenCalledWith(
      "/api/sessions/alter0-chat/attachments",
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

  it("adds a Codex option for Chat model selection and sends codex execution metadata", async () => {
    let requestBody: Record<string, unknown> | null = null;
    vi.stubGlobal("fetch", vi.fn(async (_url: string, init?: RequestInit) => {
      requestBody = JSON.parse(String(init?.body || "{}"));
      return {
        ok: true,
        body: {
          getReader: () => {
            const events = [
              new TextEncoder().encode(`event: done\ndata: ${JSON.stringify({ result: { output: "Codex done" } })}\n\n`),
            ];
            return {
              read: vi.fn(async () => {
                const value = events.shift();
                return value ? { done: false, value } : { done: true, value: undefined };
              }),
            };
          },
        },
      };
    }));

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <ModelSelectionHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("provider-list")).toHaveTextContent("Codex:Codex"));
    fireEvent.click(screen.getByRole("button", { name: "select codex" }));
    await waitFor(() => expect(screen.getByTestId("selected-model")).toHaveTextContent("Codex"));

    fireEvent.click(screen.getByRole("button", { name: "send with codex" }));

    await waitFor(() => expect(requestBody?.metadata).toMatchObject({
      "alter0.execution.engine": "codex",
    }));
    expect(requestBody?.metadata).not.toHaveProperty("alter0.llm.provider_id");
    expect(fetch).toHaveBeenCalledWith("/api/messages/stream", expect.any(Object));
  });

  it("keeps locally appended messages when a session collection refresh returns a shorter history", async () => {
    mockStreamDone("Remote completion");

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/conversation-runtime/sessions/alter0-chat?route=chat"));

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "send" }));
    });

    await waitFor(() => expect(screen.getByTestId("assistant-text")).toHaveTextContent("Remote completion"));
  });
});

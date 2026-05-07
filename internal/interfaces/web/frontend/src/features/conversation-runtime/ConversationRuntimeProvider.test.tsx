import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { ConversationRuntimeProvider, useConversationRuntime } from "./ConversationRuntimeProvider";

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
      <button
        type="button"
        onClick={() => void runtime.addDraftAttachments([
          {
            id: "local-file-1",
            kind: "file",
            name: "notes.md",
            contentType: "text/markdown",
            size: 14,
            dataURL: "data:text/markdown;base64,IyBub3Rlcw==",
          },
        ])}
      >
        attach file
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
      <button type="button" onClick={() => runtime.toggleInspector("target")}>
        target
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
      <output data-testid="target-list">
        {runtime.targetOptions.map((agent) => agent.name).join("|")}
      </output>
    </div>
  );
}

function AgentOptionsHarness() {
  const runtime = useConversationRuntime();

  return (
    <output data-testid="agent-options">
      {runtime.targetOptions.map((agent) => agent.name).join("|")}
    </output>
  );
}

function SkillOptionsHarness() {
  const runtime = useConversationRuntime();

  return (
    <div>
      <output data-testid="skill-options">
        {JSON.stringify(runtime.skills.map((skill) => ({
          id: skill.id,
          name: skill.name,
          active: skill.active,
          locked: skill.locked,
          visibility: skill.visibility,
        })))}
      </output>
      <output data-testid="skill-count">{runtime.skillCount}</output>
    </div>
  );
}

function AgentSessionProfileHarness() {
  const runtime = useConversationRuntime();
  return (
    <output data-testid="session-profile-state">
      {JSON.stringify({
        agentID: runtime.activeAgent?.id || "",
        fields: runtime.activeSessionProfile?.fields || [],
        attributes: runtime.activeSessionProfile?.attributes || {},
      })}
    </output>
  );
}

function AgentDeliverablesHarness() {
  const runtime = useConversationRuntime();
  return (
    <output data-testid="agent-deliverables">
      {JSON.stringify(runtime.activeAgent?.deliverables || [])}
    </output>
  );
}

function ActiveSessionTitleHarness() {
  const runtime = useConversationRuntime();
  return <output data-testid="active-session-title">{runtime.activeSession?.title || ""}</output>;
}

function ActiveSessionStatusHarness() {
  const runtime = useConversationRuntime();
  return <output data-testid="active-session-status">{runtime.activeSession?.status || ""}</output>;
}

function SessionItemsHarness() {
  const runtime = useConversationRuntime();
  return (
    <output data-testid="session-items">
      {JSON.stringify(runtime.sessionItems.map((item) => ({
        id: item.id,
        title: item.title,
        active: item.active,
      })))}
    </output>
  );
}

function MessageListHarness() {
  const runtime = useConversationRuntime();
  return (
    <div>
      <button type="button" onClick={() => void runtime.sendPrompt("Follow up prompt")}>
        send followup
      </button>
      <output data-testid="message-list">
        {JSON.stringify((runtime.activeSession?.messages || []).map((message) => ({
          id: message.id,
          role: message.role,
          text: message.text,
          status: message.status,
        })))}
      </output>
    </div>
  );
}

describe("ConversationRuntimeProvider", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.sessionStorage.clear();
    window.sessionStorage.setItem(
      ACTIVE_SESSION_STORAGE_KEY,
      JSON.stringify({ chat: "session-1", "agent-runtime": "" }),
    );
    window.sessionStorage.setItem(
      COMPOSER_ATTACHMENT_DRAFT_STORAGE_KEY,
      JSON.stringify({
        "session-1": [
          {
            id: "image-1",
            name: "trace.png",
            content_type: "image/png",
            size: 12,
            asset_url: "/api/sessions/session-1/attachments/image-1/original",
            preview_url: "/api/sessions/session-1/attachments/image-1/preview",
          },
        ],
      }),
    );
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/conversation-runtime/sessions?route=chat":
          return {
            items: [
              {
                id: "session-1",
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
        case "/api/conversation-runtime/sessions/session-1?route=chat":
          return {
            session: {
              id: "session-1",
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
        case "/api/conversation-runtime/sessions?route=agent-runtime":
          return { items: [] };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
        case "/api/agents":
          return { items: [] };
        default:
          return { items: [] };
      }
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("creates blank chat and agent runtime sessions with the shared New title", async () => {
    window.sessionStorage.clear();
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/conversation-runtime/sessions?route=chat":
        case "/api/conversation-runtime/sessions?route=agent-runtime":
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
        case "/api/agents":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

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
  });

  it("does not rewrite stored sessions for streaming deltas after an image message is queued", async () => {
    const encoder = new TextEncoder();
    let streamController: ReadableStreamDefaultController<Uint8Array> | null = null;
    const fetchMock = vi.fn(async () => new Response(new ReadableStream({
      start(controller) {
        streamController = controller;
      },
    }), {
      status: 200,
      headers: {
        "Content-Type": "text/event-stream",
      },
    }));
    vi.stubGlobal("fetch", fetchMock);

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );
    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/conversation-runtime/sessions?route=chat"));

    fireEvent.click(screen.getByRole("button", { name: "send" }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
    const request = JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body)) as {
      attachments?: Array<Record<string, string>>;
    };
    expect(request.attachments?.[0]).toMatchObject({
      id: "image-1",
      asset_url: "/api/sessions/session-1/attachments/image-1/original",
      preview_url: "/api/sessions/session-1/attachments/image-1/preview",
    });
    expect(request.attachments?.[0]?.data_url).toBeUndefined();

    await act(async () => {
      streamController?.enqueue(encoder.encode('event: delta\ndata: {"delta":"Analyzing"}\n\n'));
      await Promise.resolve();
    });

    await waitFor(() => expect(screen.getByTestId("assistant-text")).toHaveTextContent("Analyzing"));

    await act(async () => {
      streamController?.enqueue(encoder.encode('event: done\ndata: {"result":{"output":"Analyzing complete"}}\n\n'));
      streamController?.close();
      await Promise.resolve();
    });

    await waitFor(() => expect(screen.getByTestId("assistant-text")).toHaveTextContent("Analyzing complete"));
  });

  it("keeps append-style history stable after a streamed reply is finalized", async () => {
    window.sessionStorage.setItem(
      ACTIVE_SESSION_STORAGE_KEY,
      JSON.stringify({ chat: "session-append-style", "agent-runtime": "" }),
    );
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/conversation-runtime/sessions?route=chat":
          return {
            items: [
              {
                id: "session-append-style",
                title: "Append style session",
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
        case "/api/conversation-runtime/sessions/session-append-style?route=chat":
          return {
            session: {
              id: "session-append-style",
              title: "Append style session",
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
              messages: [
                {
                  id: "message-user-old",
                  role: "user",
                  text: "Initial prompt",
                  attachments: [],
                  route: "nl",
                  source: "web-default",
                  error: false,
                  status: "",
                  at: "2026-04-23T03:31:00Z",
                  process_steps: [],
                },
                {
                  id: "message-assistant-old",
                  role: "assistant",
                  text: "Initial answer",
                  attachments: [],
                  route: "nl",
                  source: "codex_exec",
                  error: false,
                  status: "done",
                  at: "2026-04-23T03:32:00Z",
                  process_steps: [],
                },
              ],
            },
          };
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
        case "/api/agents":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    const encoder = new TextEncoder();
    let streamController: ReadableStreamDefaultController<Uint8Array> | null = null;
    const fetchMock = vi.fn(async () => new Response(new ReadableStream({
      start(controller) {
        streamController = controller;
      },
    }), {
      status: 200,
      headers: {
        "Content-Type": "text/event-stream",
      },
    }));
    vi.stubGlobal("fetch", fetchMock);

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <MessageListHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => {
      const messages = JSON.parse(screen.getByTestId("message-list").textContent || "[]") as Array<{ role: string }>;
      expect(messages).toHaveLength(2);
    });

    fireEvent.click(screen.getByRole("button", { name: "send followup" }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));

    await act(async () => {
      streamController?.enqueue(encoder.encode('event: done\ndata: {"result":{"output":"Fresh answer"}}\n\n'));
      streamController?.enqueue(encoder.encode('event: process\ndata: {"process_step":{"kind":"tool","title":"late tool step"}}\n\n'));
      streamController?.enqueue(encoder.encode('event: delta\ndata: {"delta":" should not reopen"}\n\n'));
      streamController?.close();
      await Promise.resolve();
    });

    await waitFor(() => {
      const messages = JSON.parse(screen.getByTestId("message-list").textContent || "[]") as Array<{
        role: string;
        text: string;
        status: string;
      }>;
      expect(messages).toHaveLength(4);
      expect(messages[1]).toMatchObject({ role: "assistant", text: "Initial answer", status: "done" });
      expect(messages[3]).toMatchObject({ role: "assistant", text: "Fresh answer", status: "done" });
    });
  });

  it("uploads draft images into the session workspace before they are persisted locally", async () => {
    apiClientMock.post.mockResolvedValueOnce({
      items: [
        {
          id: "image-1",
          name: "trace.png",
          content_type: "image/png",
          size: 12,
          asset_url: "/api/sessions/session-1/attachments/image-1/original",
          preview_url: "/api/sessions/session-1/attachments/image-1/preview",
        },
      ],
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/conversation-runtime/sessions?route=chat"));

    fireEvent.click(screen.getByRole("button", { name: "attach" }));

    await waitFor(() => expect(apiClientMock.post).toHaveBeenCalledWith(
      "/api/sessions/session-1/attachments",
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

    const persistedDrafts = JSON.parse(
      window.sessionStorage.getItem(COMPOSER_ATTACHMENT_DRAFT_STORAGE_KEY) || "{}",
    ) as Record<string, Array<Record<string, string>>>;
    expect(persistedDrafts["session-1"]?.[0]).toMatchObject({
      id: "image-1",
      assetURL: "/api/sessions/session-1/attachments/image-1/original",
      previewURL: "/api/sessions/session-1/attachments/image-1/preview",
    });
    expect(persistedDrafts["session-1"]?.[0]?.dataURL).toBeUndefined();
  });

  it("uploads draft files into the session workspace through the same attachment draft API", async () => {
    apiClientMock.post.mockResolvedValueOnce({
      items: [
        {
          id: "file-1",
          name: "notes.md",
          content_type: "text/markdown",
          size: 14,
          asset_url: "/api/sessions/session-1/attachments/file-1/original",
        },
      ],
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/conversation-runtime/sessions?route=chat"));

    fireEvent.click(screen.getByRole("button", { name: "attach file" }));

    await waitFor(() => expect(apiClientMock.post).toHaveBeenCalledWith(
      "/api/sessions/session-1/attachments",
      {
        attachments: [
          {
            name: "notes.md",
            content_type: "text/markdown",
            data_url: "data:text/markdown;base64,IyBub3Rlcw==",
            preview_data_url: "data:text/markdown;base64,IyBub3Rlcw==",
          },
        ],
      },
    ));

    const persistedDrafts = JSON.parse(
      window.sessionStorage.getItem(COMPOSER_ATTACHMENT_DRAFT_STORAGE_KEY) || "{}",
    ) as Record<string, Array<Record<string, string>>>;
    const storedFile = persistedDrafts["session-1"]?.find((item) => item.id === "file-1");
    expect(storedFile).toMatchObject({
      id: "file-1",
      kind: "file",
      assetURL: "/api/sessions/session-1/attachments/file-1/original",
    });
    expect(storedFile?.previewURL).toBeUndefined();
    expect(storedFile?.dataURL).toBeUndefined();
  });

  it("allows clicking the active inspector tab again to collapse only that tab content", async () => {
    render(
      <ConversationRuntimeProvider route="agent-runtime" language="en">
        <InspectorHarness />
      </ConversationRuntimeProvider>,
    );

    expect(screen.getByTestId("inspector-state")).toHaveTextContent("model:details-closed:tab-open");

    fireEvent.click(screen.getByRole("button", { name: "target" }));
    await waitFor(() => expect(screen.getByTestId("inspector-state")).toHaveTextContent("target:details-open:tab-open"));

    fireEvent.click(screen.getByRole("button", { name: "target" }));
    await waitFor(() => expect(screen.getByTestId("inspector-state")).toHaveTextContent("target:details-open:tab-closed"));

    fireEvent.click(screen.getByRole("button", { name: "model" }));
    await waitFor(() => expect(screen.getByTestId("inspector-state")).toHaveTextContent("model:details-open:tab-open"));
  });

  it("adds a Codex option for chat model selection and sends codex execution metadata", async () => {
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/control/llm/providers":
          return {
            items: [
              {
                id: "openai",
                name: "OpenAI",
                is_default: true,
                default_model: "gpt-5.4",
                models: [
                  {
                    id: "gpt-5.4",
                    name: "GPT-5.4",
                    is_enabled: true,
                    supports_vision: true,
                  },
                ],
              },
            ],
          };
        case "/api/control/skills":
        case "/api/control/mcps":
        case "/api/agents":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    const encoder = new TextEncoder();
    const fetchMock = vi.fn(async () => new Response(new ReadableStream({
      start(controller) {
        controller.enqueue(encoder.encode('event: done\ndata: {"result":{"output":"done"}}\n\n'));
        controller.close();
      },
    }), {
      status: 200,
      headers: {
        "Content-Type": "text/event-stream",
      },
    }));
    vi.stubGlobal("fetch", fetchMock);

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <ModelSelectionHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("provider-list")).toHaveTextContent("Codex:Codex"));
    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/conversation-runtime/sessions?route=chat"));

    fireEvent.click(screen.getByRole("button", { name: "select codex" }));
    await waitFor(() => expect(screen.getByTestId("selected-model")).toHaveTextContent("Codex"));

    fireEvent.click(screen.getByRole("button", { name: "send with codex" }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));

    const request = JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body)) as {
      metadata?: Record<string, string>;
    };
    expect(request.metadata?.["alter0.execution.engine"]).toBe("codex");
    expect(request.metadata?.["alter0.llm.provider_id"]).toBeUndefined();
    expect(request.metadata?.["alter0.llm.model"]).toBeUndefined();
  });

  it("adds a Codex option for agent-runtime model selection and sends codex execution metadata", async () => {
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/control/llm/providers":
          return {
            items: [
              {
                id: "openai",
                name: "OpenAI",
                is_default: true,
                default_model: "gpt-5.4",
                models: [
                  {
                    id: "gpt-5.4",
                    name: "GPT-5.4",
                    is_enabled: true,
                    supports_vision: true,
                  },
                ],
              },
            ],
          };
        case "/api/agents":
          return {
            items: [
              {
                id: "coding",
                name: "Coding Agent",
                enabled: true,
              },
            ],
          };
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        default:
          return { items: [] };
      }
    });

    const encoder = new TextEncoder();
    const fetchMock = vi.fn(async () => new Response(new ReadableStream({
      start(controller) {
        controller.enqueue(encoder.encode('event: done\ndata: {"result":{"output":"done"}}\n\n'));
        controller.close();
      },
    }), {
      status: 200,
      headers: {
        "Content-Type": "text/event-stream",
      },
    }));
    vi.stubGlobal("fetch", fetchMock);

    render(
      <ConversationRuntimeProvider route="agent-runtime" language="en">
        <ModelSelectionHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("provider-list")).toHaveTextContent("Codex:Codex"));
    await waitFor(() => expect(screen.getByTestId("target-list")).toHaveTextContent("Coding Agent"));

    fireEvent.click(screen.getByRole("button", { name: "select codex" }));
    await waitFor(() => expect(screen.getByTestId("selected-model")).toHaveTextContent("Codex"));

    fireEvent.click(screen.getByRole("button", { name: "send with codex" }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));

    expect(fetchMock.mock.calls[0]?.[0]).toBe("/api/agent/messages/stream");
    const request = JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body)) as {
      agent_id?: string;
      metadata?: Record<string, string>;
    };
    expect(request.agent_id).toBe("coding");
    expect(request.metadata?.["alter0.execution.engine"]).toBe("codex");
    expect(request.metadata?.["alter0.llm.provider_id"]).toBeUndefined();
    expect(request.metadata?.["alter0.llm.model"]).toBeUndefined();
  });

  it("recovers agent-runtime responses from session detail when the stream stops after start", async () => {
    window.sessionStorage.setItem(
      ACTIVE_SESSION_STORAGE_KEY,
      JSON.stringify({ chat: "", "agent-runtime": "agent-session-1" }),
    );
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        case "/api/agents":
          return {
            items: [
              {
                id: "travel",
                name: "Travel Agent",
                enabled: true,
              },
            ],
          };
        case "/api/conversation-runtime/sessions?route=agent-runtime":
          return {
            items: [
              {
                id: "agent-session-1",
                title: "Travel runtime",
                title_auto: false,
                title_score: 1,
                created_at: "2026-04-23T03:30:00Z",
                target_type: "agent",
                target_id: "travel",
                target_name: "Travel Agent",
                model_provider_id: "",
                model_id: "",
                tool_ids: [],
                skill_ids: [],
                mcp_ids: [],
                messages: [],
              },
            ],
          };
        case "/api/conversation-runtime/sessions/agent-session-1?route=agent-runtime":
          return {
            session: {
              id: "agent-session-1",
              title: "Travel runtime",
              title_auto: false,
              title_score: 1,
              created_at: "2026-04-23T03:30:00Z",
              target_type: "agent",
              target_id: "travel",
              target_name: "Travel Agent",
              model_provider_id: "",
              model_id: "",
              tool_ids: [],
              skill_ids: [],
              mcp_ids: [],
              messages: [
                {
                  id: "server-user-1",
                  role: "user",
                  text: "Inspect this image",
                  status: "done",
                  at: "2026-04-23T03:31:00Z",
                },
                {
                  id: "server-assistant-1",
                  role: "assistant",
                  text: "Recovered response",
                  status: "done",
                  route: "nl",
                  at: "2026-04-23T03:31:05Z",
                },
              ],
            },
          };
        default:
          return { items: [] };
      }
    });

    const encoder = new TextEncoder();
    const fetchMock = vi.fn(async () => new Response(new ReadableStream({
      start(controller) {
        controller.enqueue(encoder.encode('event: start\ndata: {"message_id":"server-assistant-1","session_id":"agent-session-1","channel_id":"web-default","trace_id":"trace-1"}\n\n'));
        controller.close();
      },
    }), {
      status: 200,
      headers: {
        "Content-Type": "text/event-stream",
      },
    }));
    vi.stubGlobal("fetch", fetchMock);

    render(
      <ConversationRuntimeProvider route="agent-runtime" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/agents"));

    fireEvent.click(screen.getByRole("button", { name: "send" }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith(
      "/api/conversation-runtime/sessions/agent-session-1?route=agent-runtime",
    ));
    await waitFor(() => expect(screen.getByTestId("assistant-text")).toHaveTextContent("Recovered response"));
    expect(apiClientMock.post).not.toHaveBeenCalled();
  });

  it("recovers persisted agent-runtime responses when the stream reader throws after start", async () => {
    window.sessionStorage.setItem(
      ACTIVE_SESSION_STORAGE_KEY,
      JSON.stringify({ chat: "", "agent-runtime": "agent-session-throw-1" }),
    );
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/control/llm/providers":
        case "/api/control/skills":
        case "/api/control/mcps":
          return { items: [] };
        case "/api/agents":
          return {
            items: [
              {
                id: "coding",
                name: "Coding Agent",
                enabled: true,
              },
            ],
          };
        case "/api/conversation-runtime/sessions?route=agent-runtime":
          return {
            items: [
              {
                id: "agent-session-throw-1",
                title: "Coding runtime",
                title_auto: false,
                title_score: 1,
                created_at: "2026-04-23T03:30:00Z",
                target_type: "agent",
                target_id: "coding",
                target_name: "Coding Agent",
                model_provider_id: "",
                model_id: "",
                tool_ids: [],
                skill_ids: [],
                mcp_ids: [],
                messages: [],
              },
            ],
          };
        case "/api/conversation-runtime/sessions/agent-session-throw-1?route=agent-runtime":
          return {
            session: {
              id: "agent-session-throw-1",
              title: "Coding runtime",
              title_auto: false,
              title_score: 1,
              created_at: "2026-04-23T03:30:00Z",
              target_type: "agent",
              target_id: "coding",
              target_name: "Coding Agent",
              model_provider_id: "",
              model_id: "",
              tool_ids: [],
              skill_ids: [],
              mcp_ids: [],
              messages: [
                {
                  id: "server-user-throw-1",
                  role: "user",
                  text: "Fix this flow",
                  status: "done",
                  at: "2026-04-23T03:31:00Z",
                },
                {
                  id: "server-assistant-throw-1",
                  role: "assistant",
                  text: "Recovered after stream read failure",
                  status: "done",
                  route: "nl",
                  at: "2026-04-23T03:31:06Z",
                },
              ],
            },
          };
        default:
          return { items: [] };
      }
    });

    const encoder = new TextEncoder();
    let sentStart = false;
    const fetchMock = vi.fn(async () => new Response(new ReadableStream({
      pull(controller) {
        if (!sentStart) {
          sentStart = true;
          controller.enqueue(encoder.encode('event: start\ndata: {"message_id":"server-assistant-throw-1","session_id":"agent-session-throw-1","channel_id":"web-default","trace_id":"trace-throw-1"}\n\n'));
          return;
        }
        controller.error(new TypeError("Load failed"));
      },
    }), {
      status: 200,
      headers: {
        "Content-Type": "text/event-stream",
      },
    }));
    vi.stubGlobal("fetch", fetchMock);

    render(
      <ConversationRuntimeProvider route="agent-runtime" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/agents"));

    fireEvent.click(screen.getByRole("button", { name: "send" }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith(
      "/api/conversation-runtime/sessions/agent-session-throw-1?route=agent-runtime",
    ));
    await waitFor(() => expect(screen.getByTestId("assistant-text")).toHaveTextContent("Recovered after stream read failure"));
    expect(screen.getByTestId("assistant-text")).not.toHaveTextContent("Load failed");
    expect(apiClientMock.post).not.toHaveBeenCalled();
  });

  it("excludes the main Alter0 assistant from agent-runtime target options", async () => {
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/agents":
          return {
            items: [
              {
                id: "main",
                name: "Alter0",
                enabled: true,
              },
              {
                id: "coding",
                name: "Coding Agent",
                enabled: true,
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

    render(
      <ConversationRuntimeProvider route="agent-runtime" language="en">
        <AgentOptionsHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("agent-options")).toHaveTextContent("Coding Agent"));
    expect(screen.getByTestId("agent-options")).not.toHaveTextContent("Alter0");
  });

  it("shows the active agent private skill as locked and keeps selectable skills public", async () => {
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/conversation-runtime/sessions?route=agent-runtime":
          return {
            items: [
              {
                id: "travel-session-1",
                title: "Travel runtime",
                title_auto: false,
                title_score: 1,
                created_at: "2026-04-23T03:30:00Z",
                target_type: "agent",
                target_id: "travel",
                target_name: "Travel Agent",
                model_provider_id: "",
                model_id: "",
                tool_ids: [],
                skill_ids: ["deploy-test-service"],
                mcp_ids: [],
              },
            ],
          };
        case "/api/conversation-runtime/sessions/travel-session-1?route=agent-runtime":
          return {
            session: {
              id: "travel-session-1",
              title: "Travel runtime",
              title_auto: false,
              title_score: 1,
              created_at: "2026-04-23T03:30:00Z",
              target_type: "agent",
              target_id: "travel",
              target_name: "Travel Agent",
              model_provider_id: "",
              model_id: "",
              tool_ids: [],
              skill_ids: ["deploy-test-service"],
              mcp_ids: [],
              messages: [],
            },
          };
        case "/api/agents":
          return {
            items: [
              {
                id: "travel",
                name: "Travel Agent",
                enabled: true,
                capabilities: ["travel"],
                skills: ["deploy-test-service"],
              },
            ],
          };
        case "/api/control/skills":
          return {
            items: [
              {
                id: "deploy-test-service",
                name: "Deploy Test Service",
                description: "Deploy verification workflow",
                enabled: true,
              },
              {
                id: "travel-city-rules",
                name: "Travel City Rules",
                description: "Private travel planning rules",
                enabled: true,
                metadata: { "skill.visibility": "agent-private" },
              },
              {
                id: "frontend-design",
                name: "Frontend Design",
                description: "Shared frontend delivery standards",
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
      <ConversationRuntimeProvider route="agent-runtime" language="en">
        <SkillOptionsHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("skill-options")).toHaveTextContent("agent-skill-travel"));

    const skills = JSON.parse(screen.getByTestId("skill-options").textContent || "[]") as Array<{
      id: string;
      name: string;
      active: boolean;
      locked?: boolean;
      visibility?: string;
    }>;

    expect(skills).toEqual(expect.arrayContaining([
      expect.objectContaining({
        id: "agent-skill-travel",
        name: "Travel Agent Skill",
        active: true,
        locked: true,
        visibility: "agent-private",
      }),
      expect.objectContaining({
        id: "deploy-test-service",
        active: true,
        locked: false,
        visibility: "public",
      }),
      expect.objectContaining({
        id: "frontend-design",
        active: false,
        locked: false,
        visibility: "public",
      }),
    ]));
    expect(skills.map((skill) => skill.id)).not.toContain("travel-city-rules");
    expect(screen.getByTestId("skill-count")).toHaveTextContent("2");
  });

  it("loads agent session profile details for the active runtime session", async () => {
    window.sessionStorage.setItem(
      ACTIVE_SESSION_STORAGE_KEY,
      JSON.stringify({ chat: "", "agent-runtime": "agent-session-1" }),
    );
    apiClientMock.get.mockImplementation(async (path: string) => {
      if (path === "/api/control/llm/providers") {
        return { items: [] };
      }
      if (path === "/api/control/skills") {
        return { items: [] };
      }
      if (path === "/api/control/mcps") {
        return { items: [] };
      }
      if (path === "/api/conversation-runtime/sessions?route=agent-runtime") {
        return {
          items: [
            {
              id: "agent-session-1",
              title: "Coding runtime",
              title_auto: false,
              title_score: 1,
              created_at: "2026-04-23T03:30:00Z",
              target_type: "agent",
              target_id: "coding",
              target_name: "Coding Agent",
              model_provider_id: "",
              model_id: "",
              tool_ids: [],
              skill_ids: [],
              mcp_ids: [],
            },
          ],
        };
      }
      if (path === "/api/conversation-runtime/sessions/agent-session-1?route=agent-runtime") {
        return {
          session: {
            id: "agent-session-1",
            title: "Coding runtime",
            title_auto: false,
            title_score: 1,
            created_at: "2026-04-23T03:30:00Z",
            target_type: "agent",
            target_id: "coding",
            target_name: "Coding Agent",
            model_provider_id: "",
            model_id: "",
            tool_ids: [],
            skill_ids: [],
            mcp_ids: [],
            messages: [],
          },
        };
      }
      if (path === "/api/agents") {
        return {
          items: [
            {
              id: "coding",
              name: "Coding Agent",
              enabled: true,
              session_profile_fields: [
                { key: "repository_path", label: "Repository", readonly: true },
                { key: "branch", label: "Branch", readonly: true },
              ],
            },
          ],
        };
      }
      if (path === "/api/agent/session-profile?agent_id=coding&session_id=agent-session-1") {
        return {
          agent_id: "coding",
          session_id: "agent-session-1",
          path: ".alter0/agents/coding/sessions/agent-session-1.md",
          exists: true,
          fields: [
            { key: "repository_path", label: "Repository", readonly: true },
            { key: "branch", label: "Branch", readonly: true },
          ],
          attributes: {
            repository_path: "/workspace/alter0-remote",
            branch: "feature/session-profile",
          },
        };
      }
      return { items: [] };
    });

    render(
      <ConversationRuntimeProvider route="agent-runtime" language="en">
        <AgentSessionProfileHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith(
      "/api/agent/session-profile?agent_id=coding&session_id=agent-session-1",
    ));

    await waitFor(() => {
      const payload = JSON.parse(screen.getByTestId("session-profile-state").textContent || "{}") as {
        agentID?: string;
        fields?: Array<{ key?: string }>;
        attributes?: Record<string, string>;
      };
      expect(payload.agentID).toBe("coding");
      expect(payload.fields?.[0]?.key).toBe("repository_path");
      expect(payload.attributes?.repository_path).toBe("/workspace/alter0-remote");
      expect(payload.attributes?.branch).toBe("feature/session-profile");
    });
  });

  it("loads explicit deliverable contracts for runtime agents", async () => {
    window.sessionStorage.setItem(
      ACTIVE_SESSION_STORAGE_KEY,
      JSON.stringify([
        {
          id: "travel-session-1",
          title: "Travel runtime",
          titleAuto: false,
          titleScore: 1,
          createdAt: Date.parse("2026-04-23T03:30:00Z"),
          targetType: "agent",
          targetID: "travel",
          targetName: "Travel Agent",
          modelProviderID: "",
          modelID: "",
          toolIDs: [],
          skillIDs: [],
          mcpIDs: [],
          messages: [],
        },
      ]),
    );
    window.sessionStorage.setItem(
      ACTIVE_SESSION_STORAGE_KEY,
      JSON.stringify({ chat: "", "agent-runtime": "travel-session-1" }),
    );
    apiClientMock.get.mockImplementation(async (path: string) => {
      if (path === "/api/control/llm/providers") {
        return { items: [] };
      }
      if (path === "/api/control/skills") {
        return { items: [] };
      }
      if (path === "/api/control/mcps") {
        return { items: [] };
      }
      if (path === "/api/agents") {
        return {
          items: [
            {
              id: "travel",
              name: "Travel Agent",
              enabled: true,
              deliverables: [
                {
                  id: "guide-markdown",
                  label: "Travel Guide",
                  format: "markdown",
                  required: true,
                },
                {
                  id: "guide-html",
                  label: "HTML Guide",
                  format: "html",
                  required: true,
                  session_attribute_key: "guide_html_url",
                },
              ],
            },
          ],
        };
      }
      if (path === "/api/agent/session-profile?agent_id=travel&session_id=travel-session-1") {
        return {
          agent_id: "travel",
          session_id: "travel-session-1",
          path: ".alter0/agents/travel/sessions/travel-session-1.md",
          exists: true,
          fields: [
            { key: "guide_html_url", label: "Guide HTML URL", readonly: true },
          ],
          attributes: {
            guide_html_url: "https://travel-travel-session.alter0.cn",
          },
        };
      }
      return { items: [] };
    });

    render(
      <ConversationRuntimeProvider route="agent-runtime" language="en">
        <AgentDeliverablesHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => {
      const payload = JSON.parse(screen.getByTestId("agent-deliverables").textContent || "[]") as Array<{
        id?: string;
        session_attribute_key?: string;
      }>;
      expect(payload).toHaveLength(2);
      expect(payload[0]?.id).toBe("guide-markdown");
      expect(payload[1]?.session_attribute_key).toBe("guide_html_url");
    });
  });

  it("restores the active runtime session snapshot while the remote list is temporarily empty", async () => {
    window.sessionStorage.setItem(
      ACTIVE_SESSION_STORAGE_KEY,
      JSON.stringify({ chat: "", "agent-runtime": "agent-pending-1" }),
    );
    window.sessionStorage.setItem(
      ACTIVE_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        "agent-runtime": {
          id: "agent-pending-1",
          title: "Pending runtime",
          titleAuto: false,
          titleScore: 1,
          createdAt: Date.parse("2026-04-23T03:30:00Z"),
          targetType: "agent",
          targetID: "coding",
          targetName: "Coding Agent",
          messagesLoaded: true,
          serverBacked: false,
          messages: [
            {
              id: "msg-user",
              role: "user",
              text: "Fix the regression",
              attachments: [],
              at: Date.parse("2026-04-23T03:30:01Z"),
            },
            {
              id: "msg-assistant",
              role: "assistant",
              text: "Thinking...",
              attachments: [],
              status: "streaming",
              at: Date.parse("2026-04-23T03:30:02Z"),
            },
          ],
        },
      }),
    );
    apiClientMock.get.mockImplementation(async (path: string) => {
      if (path === "/api/conversation-runtime/sessions?route=agent-runtime") {
        return { items: [] };
      }
      if (path === "/api/control/llm/providers") {
        return { items: [] };
      }
      if (path === "/api/control/skills") {
        return { items: [] };
      }
      if (path === "/api/control/mcps") {
        return { items: [] };
      }
      if (path === "/api/agents") {
        return {
          items: [
            { id: "coding", name: "Coding Agent", enabled: true },
          ],
        };
      }
      return { items: [] };
    });

    render(
      <ConversationRuntimeProvider route="agent-runtime" language="en">
        <ActiveSessionTitleHarness />
        <MessageListHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("active-session-title").textContent).toBe("Pending runtime"));
    await waitFor(() => {
      const payload = JSON.parse(screen.getByTestId("message-list").textContent || "[]") as Array<{ text?: string }>;
      expect(payload).toHaveLength(2);
      expect(payload[1]?.text).toBe("Thinking...");
    });
  });

  it("hydrates the stored active runtime session by id when the collection response still misses it", async () => {
    window.sessionStorage.setItem(
      ACTIVE_SESSION_STORAGE_KEY,
      JSON.stringify({ chat: "", "agent-runtime": "agent-pending-2" }),
    );
    window.sessionStorage.setItem(
      ACTIVE_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        "agent-runtime": {
          id: "agent-pending-2",
          title: "Pending runtime",
          titleAuto: false,
          titleScore: 1,
          createdAt: Date.parse("2026-04-23T03:30:00Z"),
          targetType: "agent",
          targetID: "coding",
          targetName: "Coding Agent",
          messagesLoaded: true,
          serverBacked: false,
          messages: [
            {
              id: "msg-user",
              role: "user",
              text: "Fix the regression",
              attachments: [],
              at: Date.parse("2026-04-23T03:30:01Z"),
            },
          ],
        },
      }),
    );
    apiClientMock.get.mockImplementation(async (path: string) => {
      if (path === "/api/conversation-runtime/sessions?route=agent-runtime") {
        return { items: [] };
      }
      if (path === "/api/conversation-runtime/sessions/agent-pending-2?route=agent-runtime") {
        return {
          session: {
            id: "agent-pending-2",
            title: "Recovered runtime",
            title_auto: false,
            title_score: 2,
            created_at: "2026-04-23T03:30:00Z",
            target_type: "agent",
            target_id: "coding",
            target_name: "Coding Agent",
            messages: [
              {
                id: "msg-user",
                role: "user",
                text: "Fix the regression",
                at: "2026-04-23T03:30:01Z",
              },
              {
                id: "msg-assistant",
                role: "assistant",
                text: "Recovered from server",
                status: "done",
                at: "2026-04-23T03:30:04Z",
              },
            ],
          },
        };
      }
      if (path === "/api/control/llm/providers") {
        return { items: [] };
      }
      if (path === "/api/control/skills") {
        return { items: [] };
      }
      if (path === "/api/control/mcps") {
        return { items: [] };
      }
      if (path === "/api/agents") {
        return {
          items: [
            { id: "coding", name: "Coding Agent", enabled: true },
          ],
        };
      }
      return { items: [] };
    });

    render(
      <ConversationRuntimeProvider route="agent-runtime" language="en">
        <ActiveSessionTitleHarness />
        <MessageListHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith(
      "/api/conversation-runtime/sessions/agent-pending-2?route=agent-runtime",
    ));
    await waitFor(() => expect(screen.getByTestId("active-session-title").textContent).toBe("Recovered runtime"));
    await waitFor(() => {
      const payload = JSON.parse(screen.getByTestId("message-list").textContent || "[]") as Array<{ text?: string }>;
      expect(payload).toHaveLength(2);
      expect(payload[1]?.text).toBe("Recovered from server");
    });
  });

  it("keeps recently restored runtime sessions in the list when another refresh temporarily misses them", async () => {
    window.sessionStorage.setItem(
      ACTIVE_SESSION_STORAGE_KEY,
      JSON.stringify({ chat: "", "agent-runtime": "agent-visible-2" }),
    );
    window.sessionStorage.setItem(
      RECENT_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        "agent-runtime": [
          {
            id: "agent-visible-2",
            title: "Visible current session",
            titleAuto: false,
            titleScore: 2,
            createdAt: Date.parse("2026-04-23T03:40:00Z"),
            targetType: "agent",
            targetID: "coding",
            targetName: "Coding Agent",
            messagesLoaded: true,
            serverBacked: true,
            messages: [
              {
                id: "msg-visible-2",
                role: "assistant",
                text: "Current session reply",
                attachments: [],
                status: "done",
                at: Date.parse("2026-04-23T03:40:02Z"),
              },
            ],
          },
          {
            id: "agent-visible-1",
            title: "Recently created session",
            titleAuto: false,
            titleScore: 2,
            createdAt: Date.parse("2026-04-23T03:35:00Z"),
            targetType: "agent",
            targetID: "coding",
            targetName: "Coding Agent",
            messagesLoaded: true,
            serverBacked: true,
            messages: [
              {
                id: "msg-visible-1",
                role: "assistant",
                text: "Fresh HTML guide",
                attachments: [],
                status: "done",
                at: Date.parse("2026-04-23T03:35:02Z"),
              },
            ],
          },
        ],
      }),
    );
    apiClientMock.get.mockImplementation(async (path: string) => {
      if (path === "/api/conversation-runtime/sessions?route=agent-runtime") {
        return {
          items: [
            {
              id: "agent-visible-2",
              title: "Visible current session",
              title_auto: false,
              title_score: 2,
              created_at: "2026-04-23T03:40:00Z",
              target_type: "agent",
              target_id: "coding",
              target_name: "Coding Agent",
            },
          ],
        };
      }
      if (path === "/api/control/llm/providers") {
        return { items: [] };
      }
      if (path === "/api/control/skills") {
        return { items: [] };
      }
      if (path === "/api/control/mcps") {
        return { items: [] };
      }
      if (path === "/api/agents") {
        return {
          items: [
            { id: "coding", name: "Coding Agent", enabled: true },
          ],
        };
      }
      return { items: [] };
    });

    render(
      <ConversationRuntimeProvider route="agent-runtime" language="en">
        <SessionItemsHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => {
      const payload = JSON.parse(screen.getByTestId("session-items").textContent || "[]") as Array<{
        id?: string;
        active?: boolean;
      }>;
      expect(payload).toHaveLength(2);
      expect(payload.map((item) => item.id)).toEqual(["agent-visible-2", "agent-visible-1"]);
      expect(payload.find((item) => item.id === "agent-visible-2")?.active).toBe(true);
    });
  });

  it("hydrates the active runtime session from recent snapshots when the collection is still empty", async () => {
    window.sessionStorage.setItem(
      ACTIVE_SESSION_STORAGE_KEY,
      JSON.stringify({ chat: "", "agent-runtime": "agent-visible-3" }),
    );
    window.sessionStorage.setItem(
      RECENT_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        "agent-runtime": [
          {
            id: "agent-visible-3",
            title: "Recovered from recent",
            titleAuto: false,
            titleScore: 2,
            createdAt: Date.parse("2026-04-23T03:45:00Z"),
            targetType: "agent",
            targetID: "coding",
            targetName: "Coding Agent",
            messagesLoaded: false,
            serverBacked: true,
            messages: [],
          },
        ],
      }),
    );
    apiClientMock.get.mockImplementation(async (path: string) => {
      if (path === "/api/conversation-runtime/sessions?route=agent-runtime") {
        return { items: [] };
      }
      if (path === "/api/conversation-runtime/sessions/agent-visible-3?route=agent-runtime") {
        return {
          session: {
            id: "agent-visible-3",
            title: "Recovered from server",
            title_auto: false,
            title_score: 3,
            created_at: "2026-04-23T03:45:00Z",
            target_type: "agent",
            target_id: "coding",
            target_name: "Coding Agent",
            messages: [
              {
                id: "msg-user-3",
                role: "user",
                text: "Build the travel page",
                at: "2026-04-23T03:45:01Z",
              },
              {
                id: "msg-assistant-3",
                role: "assistant",
                text: "HTML generated",
                status: "done",
                at: "2026-04-23T03:45:05Z",
              },
            ],
          },
        };
      }
      if (path === "/api/control/llm/providers") {
        return { items: [] };
      }
      if (path === "/api/control/skills") {
        return { items: [] };
      }
      if (path === "/api/control/mcps") {
        return { items: [] };
      }
      if (path === "/api/agents") {
        return {
          items: [
            { id: "coding", name: "Coding Agent", enabled: true },
          ],
        };
      }
      return { items: [] };
    });

    render(
      <ConversationRuntimeProvider route="agent-runtime" language="en">
        <ActiveSessionTitleHarness />
        <MessageListHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("active-session-title").textContent).toBe("Recovered from server"));
    await waitFor(() => {
      const payload = JSON.parse(screen.getByTestId("message-list").textContent || "[]") as Array<{ text?: string }>;
      expect(payload).toHaveLength(2);
      expect(payload[1]?.text).toBe("HTML generated");
    });
  });

  it("reconciles a stored failed agent snapshot from session detail when the collection still only has summary data", async () => {
    window.sessionStorage.setItem(
      ACTIVE_SESSION_STORAGE_KEY,
      JSON.stringify({ chat: "", "agent-runtime": "agent-recover-1" }),
    );
    window.sessionStorage.setItem(
      ACTIVE_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        "agent-runtime": {
          id: "agent-recover-1",
          status: "failed",
          title: "Recover pending",
          titleAuto: false,
          titleScore: 1,
          createdAt: Date.parse("2026-04-23T03:30:00Z"),
          targetType: "agent",
          targetID: "coding",
          targetName: "Coding Agent",
          messagesLoaded: true,
          serverBacked: false,
          messages: [
            {
              id: "local-user-1",
              role: "user",
              text: "Recover this response",
              attachments: [],
              at: Date.parse("2026-04-23T03:30:01Z"),
            },
            {
              id: "local-assistant-1",
              role: "assistant",
              text: "Load failed",
              attachments: [],
              status: "error",
              error: true,
              at: Date.parse("2026-04-23T03:30:04Z"),
            },
          ],
        },
      }),
    );
    apiClientMock.get.mockImplementation(async (path: string) => {
      if (path === "/api/conversation-runtime/sessions?route=agent-runtime") {
        return {
          items: [
            {
              id: "agent-recover-1",
              status: "ready",
              title: "Recover pending",
              title_auto: false,
              title_score: 1,
              created_at: "2026-04-23T03:30:00Z",
              target_type: "agent",
              target_id: "coding",
              target_name: "Coding Agent",
            },
          ],
        };
      }
      if (path === "/api/conversation-runtime/sessions/agent-recover-1?route=agent-runtime") {
        return {
          session: {
            id: "agent-recover-1",
            status: "ready",
            title: "Recover pending",
            title_auto: false,
            title_score: 1,
            created_at: "2026-04-23T03:30:00Z",
            target_type: "agent",
            target_id: "coding",
            target_name: "Coding Agent",
            messages: [
              {
                id: "server-user-1",
                role: "user",
                text: "Recover this response",
                at: "2026-04-23T03:30:01Z",
              },
              {
                id: "server-assistant-1",
                role: "assistant",
                text: "Recovered persisted agent result",
                status: "done",
                at: "2026-04-23T03:30:05Z",
              },
            ],
          },
        };
      }
      if (path === "/api/control/llm/providers") {
        return { items: [] };
      }
      if (path === "/api/control/skills") {
        return { items: [] };
      }
      if (path === "/api/control/mcps") {
        return { items: [] };
      }
      if (path === "/api/agents") {
        return {
          items: [
            { id: "coding", name: "Coding Agent", enabled: true },
          ],
        };
      }
      return { items: [] };
    });

    render(
      <ConversationRuntimeProvider route="agent-runtime" language="en">
        <ActiveSessionStatusHarness />
        <MessageListHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith(
      "/api/conversation-runtime/sessions/agent-recover-1?route=agent-runtime",
    ));
    await waitFor(() => expect(screen.getByTestId("active-session-status")).toHaveTextContent("ready"));
    await waitFor(() => {
      const payload = JSON.parse(screen.getByTestId("message-list").textContent || "[]") as Array<{ text?: string; status?: string }>;
      expect(payload).toHaveLength(2);
      expect(payload[1]?.text).toBe("Recovered persisted agent result");
      expect(payload[1]?.status).toBe("done");
    });
  });

  it("keeps retrying active agent recovery when session detail initially returns only registry summary", async () => {
    vi.useFakeTimers();
    try {
      window.sessionStorage.setItem(
        ACTIVE_SESSION_STORAGE_KEY,
        JSON.stringify({ chat: "", "agent-runtime": "agent-retry-1" }),
      );
      window.sessionStorage.setItem(
        ACTIVE_SESSION_SNAPSHOT_STORAGE_KEY,
        JSON.stringify({
          "agent-runtime": {
            id: "agent-retry-1",
            status: "failed",
            title: "Retry pending",
            titleAuto: false,
            titleScore: 1,
            createdAt: Date.parse("2026-04-23T03:30:00Z"),
            targetType: "agent",
            targetID: "coding",
            targetName: "Coding Agent",
            messagesLoaded: true,
            serverBacked: false,
            messages: [
              {
                id: "local-user-retry-1",
                role: "user",
                text: "Recover immediately after refresh",
                attachments: [],
                at: Date.parse("2026-04-23T03:30:01Z"),
              },
              {
                id: "local-assistant-retry-1",
                role: "assistant",
                text: "Load failed",
                attachments: [],
                status: "error",
                error: true,
                at: Date.parse("2026-04-23T03:30:03Z"),
              },
            ],
          },
        }),
      );

      let detailCalls = 0;
      apiClientMock.get.mockImplementation(async (path: string) => {
        if (path === "/api/conversation-runtime/sessions?route=agent-runtime") {
          return {
            items: [
              {
                id: "agent-retry-1",
                status: "ready",
                title: "Retry pending",
                title_auto: false,
                title_score: 1,
                created_at: "2026-04-23T03:30:00Z",
                target_type: "agent",
                target_id: "coding",
                target_name: "Coding Agent",
              },
            ],
          };
        }
        if (path === "/api/conversation-runtime/sessions/agent-retry-1?route=agent-runtime") {
          detailCalls += 1;
          if (detailCalls === 1) {
            return {
              session: {
                id: "agent-retry-1",
                status: "ready",
                title: "Retry pending",
                title_auto: false,
                title_score: 1,
                created_at: "2026-04-23T03:30:00Z",
                target_type: "agent",
                target_id: "coding",
                target_name: "Coding Agent",
              },
            };
          }
          return {
            session: {
              id: "agent-retry-1",
              status: "ready",
              title: "Retry pending",
              title_auto: false,
              title_score: 1,
              created_at: "2026-04-23T03:30:00Z",
              target_type: "agent",
              target_id: "coding",
              target_name: "Coding Agent",
            },
          };
        }
        if (path === "/api/control/llm/providers") {
          return { items: [] };
        }
        if (path === "/api/control/skills") {
          return { items: [] };
        }
        if (path === "/api/control/mcps") {
          return { items: [] };
        }
        if (path === "/api/agents") {
          return {
            items: [
              { id: "coding", name: "Coding Agent", enabled: true },
            ],
          };
        }
        return { items: [] };
      });

      render(
        <ConversationRuntimeProvider route="agent-runtime" language="en">
          <MessageListHarness />
        </ConversationRuntimeProvider>,
      );

      await act(async () => {
        await Promise.resolve();
        await Promise.resolve();
      });
      expect(detailCalls).toBe(1);

      await act(async () => {
        await vi.advanceTimersByTimeAsync(3000);
      });

      expect(detailCalls).toBeGreaterThanOrEqual(2);
    } finally {
      vi.useRealTimers();
    }
  });

  it("preserves failed runtime status when recovery falls back to a registry-backed session detail", async () => {
    window.sessionStorage.setItem(
      ACTIVE_SESSION_STORAGE_KEY,
      JSON.stringify({ chat: "", "agent-runtime": "agent-failed-1" }),
    );
    window.sessionStorage.setItem(
      RECENT_SESSION_SNAPSHOT_STORAGE_KEY,
      JSON.stringify({
        "agent-runtime": [
          {
            id: "agent-failed-1",
            status: "failed",
            title: "Travel failed",
            titleAuto: false,
            titleScore: 2,
            createdAt: Date.parse("2026-04-23T03:45:00Z"),
            targetType: "agent",
            targetID: "travel",
            targetName: "Travel Agent",
            messagesLoaded: false,
            serverBacked: true,
            messages: [],
          },
        ],
      }),
    );
    apiClientMock.get.mockImplementation(async (path: string) => {
      if (path === "/api/conversation-runtime/sessions?route=agent-runtime") {
        return { items: [] };
      }
      if (path === "/api/conversation-runtime/sessions/agent-failed-1?route=agent-runtime") {
        return {
          session: {
            id: "agent-failed-1",
            status: "failed",
            title: "Travel failed",
            title_auto: false,
            title_score: 3,
            created_at: "2026-04-23T03:45:00Z",
            target_type: "agent",
            target_id: "travel",
            target_name: "Travel Agent",
          },
        };
      }
      if (path === "/api/control/llm/providers") {
        return { items: [] };
      }
      if (path === "/api/control/skills") {
        return { items: [] };
      }
      if (path === "/api/control/mcps") {
        return { items: [] };
      }
      if (path === "/api/agents") {
        return {
          items: [
            { id: "travel", name: "Travel Agent", enabled: true },
          ],
        };
      }
      return { items: [] };
    });

    render(
      <ConversationRuntimeProvider route="agent-runtime" language="en">
        <ActiveSessionTitleHarness />
        <ActiveSessionStatusHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("active-session-title").textContent).toBe("Travel failed"));
    await waitFor(() => expect(screen.getByTestId("active-session-status").textContent).toBe("failed"));
  });
});

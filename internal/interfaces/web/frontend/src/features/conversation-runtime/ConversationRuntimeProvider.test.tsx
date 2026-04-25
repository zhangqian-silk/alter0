import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { ConversationRuntimeProvider, useConversationRuntime } from "./ConversationRuntimeProvider";

const SESSION_STORAGE_KEY = "alter0.web.sessions.v3";
const ACTIVE_SESSION_STORAGE_KEY = "alter0.web.session.active.v1";
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
        {runtime.inspectorOpen ? `${runtime.inspectorTab}:open` : `${runtime.inspectorTab}:closed`}
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

describe("ConversationRuntimeProvider", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.sessionStorage.clear();
    window.sessionStorage.setItem(
      SESSION_STORAGE_KEY,
      JSON.stringify([
        {
          id: "session-1",
          title: "Image session",
          titleAuto: false,
          titleScore: 1,
          createdAt: Date.parse("2026-04-23T03:30:00Z"),
          targetType: "model",
          targetID: "raw-model",
          targetName: "Raw Model",
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
  });

  afterEach(() => {
    vi.unstubAllGlobals();
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

    const setItemSpy = vi.spyOn(Storage.prototype, "setItem");

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
      await new Promise((resolve) => window.setTimeout(resolve, 220));
    });

    const sessionWritesAfterQueue = setItemSpy.mock.calls.filter(([key]) => key === SESSION_STORAGE_KEY).length;
    expect(sessionWritesAfterQueue).toBe(1);
    const latestQueuedSessionPayload = setItemSpy.mock.calls
      .filter(([key]) => key === SESSION_STORAGE_KEY)
      .at(-1)?.[1];
    expect(typeof latestQueuedSessionPayload).toBe("string");
    const queuedSessions = JSON.parse(String(latestQueuedSessionPayload)) as Array<{
      messages?: Array<{ attachments?: Array<{ preview_url?: string; data_url?: string }> }>;
    }>;
    expect(queuedSessions[0]?.messages?.[0]?.attachments?.[0]?.preview_url).toBe("/api/sessions/session-1/attachments/image-1/preview");
    expect(queuedSessions[0]?.messages?.[0]?.attachments?.[0]?.data_url).toBeUndefined();

    await act(async () => {
      streamController?.enqueue(encoder.encode('event: delta\ndata: {"delta":"Analyzing"}\n\n'));
      await Promise.resolve();
      await new Promise((resolve) => window.setTimeout(resolve, 220));
    });

    await waitFor(() => expect(screen.getByTestId("assistant-text")).toHaveTextContent("Analyzing"));

    const sessionWritesAfterDelta = setItemSpy.mock.calls.filter(([key]) => key === SESSION_STORAGE_KEY).length;
    expect(sessionWritesAfterDelta).toBe(1);

    await act(async () => {
      streamController?.enqueue(encoder.encode('event: done\ndata: {"result":{"output":"Analyzing complete"}}\n\n'));
      streamController?.close();
      await Promise.resolve();
      await new Promise((resolve) => window.setTimeout(resolve, 220));
    });

    await waitFor(() => expect(screen.getByTestId("assistant-text")).toHaveTextContent("Analyzing complete"));

    const sessionWritesAfterDone = setItemSpy.mock.calls.filter(([key]) => key === SESSION_STORAGE_KEY).length;
    expect(sessionWritesAfterDone).toBe(2);
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

  it("allows clicking the active inspector tab again to collapse the details body", async () => {
    render(
      <ConversationRuntimeProvider route="agent-runtime" language="en">
        <InspectorHarness />
      </ConversationRuntimeProvider>,
    );

    expect(screen.getByTestId("inspector-state")).toHaveTextContent("model:closed");

    fireEvent.click(screen.getByRole("button", { name: "target" }));
    await waitFor(() => expect(screen.getByTestId("inspector-state")).toHaveTextContent("target:open"));

    fireEvent.click(screen.getByRole("button", { name: "target" }));
    await waitFor(() => expect(screen.getByTestId("inspector-state")).toHaveTextContent("target:closed"));

    fireEvent.click(screen.getByRole("button", { name: "model" }));
    await waitFor(() => expect(screen.getByTestId("inspector-state")).toHaveTextContent("model:open"));
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

  it("loads agent session profile details for the active runtime session", async () => {
    window.sessionStorage.setItem(
      SESSION_STORAGE_KEY,
      JSON.stringify([
        {
          id: "agent-session-1",
          title: "Coding runtime",
          titleAuto: false,
          titleScore: 1,
          createdAt: Date.parse("2026-04-23T03:30:00Z"),
          targetType: "agent",
          targetID: "coding",
          targetName: "Coding Agent",
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
});

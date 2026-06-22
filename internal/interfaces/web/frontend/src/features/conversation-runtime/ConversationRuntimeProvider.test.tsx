import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import {
  ConversationRuntimeProvider,
  resolveChatTaskPollPlan,
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
      <output data-testid="assistant-process-count">{assistantMessage?.processSteps.length || 0}</output>
      <output data-testid="assistant-process-status">{assistantMessage?.processSteps[0]?.status || ""}</output>
      <output data-testid="active-session-status">{runtime.activeSession?.status || ""}</output>
    </div>
  );
}

function MessageTextHarness() {
  const runtime = useConversationRuntime();
  return (
    <output data-testid="message-texts">
      {runtime.activeSession?.messages.map((message) => message.text).join("|") || ""}
    </output>
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

function setupDefaultAPI() {
  apiClientMock.get.mockImplementation(async (path: string) => {
    switch (path) {
      case "/api/terminal/sessions?scope=chat":
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
      case "/api/terminal/sessions/alter0-chat?scope=chat":
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

function mockMessageDone(output = "Done") {
  apiClientMock.post.mockImplementation(async (path: string, body?: Record<string, unknown>) => {
    if (path === "/api/terminal/sessions/alter0-chat/input?scope=chat") {
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
    vi.clearAllMocks();
    window.sessionStorage.clear();
    window.history.replaceState({}, "", "/chat");
    window.sessionStorage.setItem(
      ACTIVE_SESSION_STORAGE_KEY,
      JSON.stringify({ chat: "alter0-chat" }),
    );
    setupDefaultAPI();
    apiClientMock.post.mockImplementation(async (path: string) => {
      if (path === "/api/terminal/sessions?scope=chat") {
        return {
          session: {
            id: "new-terminal-chat",
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
    vi.useRealTimers();
    vi.unstubAllGlobals();
    window.history.replaceState({}, "", "/");
  });

  it("pauses pending task polling while the page is hidden", () => {
    expect(resolveChatTaskPollPlan({ pendingCount: 0, pageHidden: false })).toEqual({
      enabled: false,
      interval: 0,
    });
    expect(resolveChatTaskPollPlan({ pendingCount: 1, pageHidden: false })).toEqual({
      enabled: true,
      interval: 3000,
    });
    expect(resolveChatTaskPollPlan({ pendingCount: 1, pageHidden: true })).toEqual({
      enabled: false,
      interval: 0,
    });
  });

  it("does not create local blank Chat sessions before the user starts a Terminal-backed session", async () => {
    window.sessionStorage.clear();
    apiClientMock.get.mockImplementation(async () => ({ items: [] }));

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <SessionListHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/terminal/sessions?scope=chat"));
    expect(screen.getByTestId("sessions")).toHaveTextContent("");

    fireEvent.click(screen.getByRole("button", { name: "new session" }));

    await waitFor(() => expect(apiClientMock.post).toHaveBeenCalledWith("/api/terminal/sessions?scope=chat", {}));
    await waitFor(() => expect(screen.getByTestId("sessions")).toHaveTextContent(/^New:/));
  });

  it("loads Chat sessions through the isolated chat Terminal scope", async () => {
    window.sessionStorage.clear();
    apiClientMock.get.mockImplementation(async () => ({ items: [] }));

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <SessionListHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/terminal/sessions?scope=chat"));
  });

  it("creates Terminal-backed Chat sessions when New is pressed repeatedly", async () => {
    window.sessionStorage.clear();
    apiClientMock.get.mockImplementation(async () => ({ items: [] }));

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <SessionListHarness />
      </ConversationRuntimeProvider>,
    );

    fireEvent.click(screen.getByRole("button", { name: "new session" }));
    fireEvent.click(screen.getByRole("button", { name: "new session" }));

    await waitFor(() => expect(apiClientMock.post).toHaveBeenCalledWith("/api/terminal/sessions?scope=chat", {}));
  });

  it("updates Chat session pin state through the session history pin endpoint", async () => {
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/terminal/sessions?scope=chat":
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
        "/api/terminal/sessions/alter0-chat/pin?scope=chat",
        { pinned: false },
      );
    });
    expect(screen.getByTestId("sessions")).toHaveTextContent("unpinned");
  });

  it("moves pinned Chat sessions ahead of newer unpinned sessions", async () => {
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/terminal/sessions?scope=chat":
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
        "/api/terminal/sessions/older-chat/pin?scope=chat",
        { pinned: true },
      );
    });
    expect(screen.getByTestId("sessions")).toHaveTextContent(/^Older session:[^|]*:pinned\|Newer session:[^|]*:unpinned$/);
  });

  it("pins a newly created Terminal-backed Chat session", async () => {
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/terminal/sessions?scope=chat":
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

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/terminal/sessions?scope=chat"));

    fireEvent.click(screen.getByRole("button", { name: "new session" }));
    await waitFor(() => expect(screen.getByTestId("sessions")).toHaveTextContent(/^New:/));

    fireEvent.click(screen.getByRole("button", { name: "pin active" }));

    await waitFor(() => expect(apiClientMock.post).toHaveBeenCalledWith(
      "/api/terminal/sessions/new-terminal-chat/pin?scope=chat",
      { pinned: true },
    ));
    expect(screen.getByTestId("sessions")).toHaveTextContent(/^New:[^|]*:pinned$/);
  });

  it("selects all public skills by default for a new blank Chat session", async () => {
    window.sessionStorage.clear();
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/terminal/sessions?scope=chat":
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

  it("selects all public skills by default when a Terminal-backed Chat session has no skill_ids field", async () => {
    let requestBody: Record<string, unknown> | null = null;
    apiClientMock.post.mockImplementation(async (path: string, body?: Record<string, unknown>) => {
      if (path === "/api/terminal/sessions/alter0-chat/input?scope=chat") {
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
        case "/api/terminal/sessions?scope=chat":
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
        case "/api/terminal/sessions/alter0-chat?scope=chat":
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
        case "/api/terminal/sessions?scope=chat":
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
        case "/api/terminal/sessions/skill-session-2?scope=chat":
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
    expect(apiClientMock.get).toHaveBeenCalledWith("/api/terminal/sessions?scope=chat");
  });

  it("keeps the stored active Terminal-backed Chat session when the route has no explicit session query", async () => {
    window.history.replaceState({}, "", "/chat");
    window.sessionStorage.setItem(
      ACTIVE_SESSION_STORAGE_KEY,
      JSON.stringify({ chat: "older-chat-session" }),
    );
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/terminal/sessions?scope=chat":
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
        case "/api/terminal/sessions/latest-chat-session?scope=chat":
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
      if (path === "/api/terminal/sessions?scope=chat") {
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
  });

  it("uploads draft images into the active Chat session workspace", async () => {
    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/terminal/sessions?scope=chat"));
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

  it("marks the chat session busy without creating local stream process steps after sending a prompt", async () => {
    vi.stubGlobal("fetch", vi.fn());
    apiClientMock.post.mockImplementation(async (path: string) => {
      if (path === "/api/terminal/sessions/alter0-chat/input?scope=chat") {
        return new Promise(() => undefined);
      }
      return {};
    });

    render(
      <ConversationRuntimeProvider route="chat" language="en">
        <RuntimeHarness />
      </ConversationRuntimeProvider>,
    );

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/terminal/sessions?scope=chat"));
    fireEvent.click(screen.getByRole("button", { name: "send" }));

    await waitFor(() => expect(screen.getByTestId("active-session-status")).toHaveTextContent("busy"));
    expect(screen.getByTestId("assistant-text")).toHaveTextContent("");
    expect(screen.getByTestId("assistant-process-count")).toHaveTextContent("0");
    expect(screen.getByTestId("assistant-process-status")).toHaveTextContent("");
    expect(apiClientMock.post).toHaveBeenCalledWith("/api/terminal/sessions/alter0-chat/input?scope=chat", expect.any(Object));
    expect(fetch).not.toHaveBeenCalled();
  });

  it("keeps polling a busy Terminal-backed Chat session until its final output is restored", async () => {
    vi.stubGlobal("fetch", vi.fn());
    let inputAccepted = false;
    let detailCallsAfterInput = 0;
    apiClientMock.post.mockImplementation(async (path: string, body?: Record<string, unknown>) => {
      if (path === "/api/terminal/sessions/alter0-chat/input?scope=chat") {
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
              },
            ],
          },
        };
      }
      return {};
    });
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/terminal/sessions?scope=chat":
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
        case "/api/terminal/sessions/alter0-chat?scope=chat":
          if (inputAccepted) {
            detailCallsAfterInput += 1;
          }
          return {
            session: inputAccepted
              ? detailCallsAfterInput > 3
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

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/terminal/sessions?scope=chat"));
    vi.useFakeTimers();
    fireEvent.click(screen.getByRole("button", { name: "send" }));

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });
    expect(screen.getByTestId("active-session-status")).toHaveTextContent("busy");

    await act(async () => {
      await vi.advanceTimersByTimeAsync(3000);
    });

    expect(screen.getByTestId("assistant-text")).toHaveTextContent("Restored final output");
    expect(screen.getByTestId("active-session-status")).toHaveTextContent("ready");
    vi.useRealTimers();
  });

  it("does not submit another Chat input while the Terminal-backed session is busy", async () => {
    vi.stubGlobal("fetch", vi.fn());
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/terminal/sessions?scope=chat":
        case "/api/terminal/sessions/alter0-chat?scope=chat":
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

    expect(apiClientMock.post).not.toHaveBeenCalledWith("/api/terminal/sessions/alter0-chat/input?scope=chat", expect.any(Object));
    expect(screen.getByTestId("assistant-text")).toHaveTextContent("");
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

  it("adds a Codex option for Chat model selection and submits through Terminal input", async () => {
    vi.stubGlobal("fetch", vi.fn());
    let requestBody: Record<string, unknown> | null = null;
    apiClientMock.post.mockImplementation(async (path: string, body?: Record<string, unknown>) => {
      if (path === "/api/terminal/sessions/alter0-chat/input?scope=chat") {
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
    expect(apiClientMock.post).toHaveBeenCalledWith("/api/terminal/sessions/alter0-chat/input?scope=chat", expect.any(Object));
    expect(fetch).not.toHaveBeenCalled();
  });

  it("persists Chat skill selections to the runtime session before the next message is sent", async () => {
    let requestBody: Record<string, unknown> | null = null;
    apiClientMock.post.mockImplementation(async (path: string, body?: Record<string, unknown>) => {
      if (path === "/api/terminal/sessions/alter0-chat/input?scope=chat") {
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
        case "/api/terminal/sessions?scope=chat":
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
        case "/api/terminal/sessions/alter0-chat?scope=chat":
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
        case "/api/terminal/sessions?scope=chat":
        case "/api/terminal/sessions/alter0-chat?scope=chat":
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
      if (path === "/api/terminal/sessions/alter0-chat/input?scope=chat") {
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
        case "/api/terminal/sessions?scope=chat":
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
        case "/api/terminal/sessions/alter0-chat?scope=chat":
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
      if (path === "/api/terminal/sessions/alter0-chat/input?scope=chat") {
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
        case "/api/terminal/sessions?scope=chat":
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
        case "/api/terminal/sessions/alter0-chat?scope=chat":
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

    await waitFor(() => expect(apiClientMock.get).toHaveBeenCalledWith("/api/terminal/sessions?scope=chat"));

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "send" }));
    });

    await waitFor(() => expect(screen.getByTestId("assistant-text")).toHaveTextContent("Remote completion"));
  });

  it("merges paged Chat session detail refreshes into existing messages", async () => {
    apiClientMock.get.mockImplementation(async (path: string) => {
      switch (path) {
        case "/api/terminal/sessions?scope=chat":
          return {
            items: [{
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
            }],
          };
        case "/api/terminal/sessions/alter0-chat?scope=chat":
          return {
            session: {
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
});

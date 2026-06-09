import { fireEvent, render, screen, within } from "@testing-library/react";
import { useState } from "react";
import { ConversationWorkspace } from "./ConversationWorkspace";
import { WorkbenchContext, type WorkbenchContextValue } from "../../app/WorkbenchContext";
import { conversationMarkdownSyntaxFixture } from "../shell/components/MessageMarkdownSyntaxFixture";

const { buildChatTimelineItemsMock } = vi.hoisted(() => ({
  buildChatTimelineItemsMock: vi.fn(({ messages }: { messages: Array<{ id: string }> }) =>
    messages.map((message) => ({
      id: message.id,
      className: "msg assistant",
      articleProps: { "data-message-id": message.id },
      bubbleClassName: "msg-bubble",
      blocks: [],
    })),
  ),
}));

const runtimeMock = {
  route: "chat" as const,
  compact: true,
  inspectorOpen: false,
  inspectorTab: "model" as const,
  inspectorTabOpen: true,
  sessions: [],
  activeSession: {
    id: "session-1",
    status: "ready",
    title: "New",
    messages: [],
  },
  sessionItems: [
    {
      id: "session-1",
      title: "New",
      meta: "now",
      shortHash: "abcd1234",
      createdAt: Date.parse("2026-04-23T09:00:00Z"),
      active: true,
    },
  ],
  draft: "",
  target: { type: "model" as const, id: "raw-model", name: "Raw Model" },
  lockedTarget: false,
  selectedProviderId: "",
  selectedModelId: "",
  selectedModelLabel: "DeepSeek V3.2",
  selectedModelSupportsVision: true,
  providers: [],
  capabilities: [] as Array<{
    id: string;
    name: string;
    description: string;
    kind: "tool" | "mcp" | "skill";
    active: boolean;
  }>,
  skills: [] as Array<{
    id: string;
    name: string;
    description: string;
    kind: "tool" | "mcp" | "skill";
    active: boolean;
    visibility?: "public" | "agent-private";
    locked?: boolean;
  }>,
  toolCount: 0,
  skillCount: 0,
  createSession: vi.fn(),
  focusSession: vi.fn(),
  removeSession: vi.fn().mockResolvedValue(undefined),
  setDraft: vi.fn(),
  draftAttachments: [],
  addDraftAttachments: vi.fn().mockResolvedValue(undefined),
  removeDraftAttachment: vi.fn(),
  clearDraftAttachments: vi.fn(),
  sendPrompt: vi.fn().mockResolvedValue(undefined),
  toggleInspector: vi.fn(),
  closeInspector: vi.fn(),
  selectModel: vi.fn(),
  toggleCapability: vi.fn(),
  toggleSkill: vi.fn(),
  toggleAgentProcess: vi.fn(),
};

vi.mock("./ConversationRuntimeProvider", () => ({
  useConversationRuntime: () => runtimeMock,
  useConversationRuntimeWorkspace: () => runtimeMock,
  useConversationRuntimeComposer: () => runtimeMock,
}));

vi.mock("../shell/components/ChatMessageRegion", () => ({
  ChatMessageRegion: () => <div data-testid="chat-message-region">messages</div>,
  buildChatTimelineItems: buildChatTimelineItemsMock,
}));

function WorkspaceTestFrame({ overrides = {} }: { overrides?: Partial<WorkbenchContextValue> }) {
  const baseContextValue: WorkbenchContextValue = {
    route: "chat",
    language: "en",
    navigate: vi.fn(),
    isMobileViewport: true,
    mobileNavOpen: false,
    mobileSessionPaneOpen: false,
    toggleMobileNav: vi.fn(),
    toggleMobileSessionPane: vi.fn(),
    closeMobileNav: vi.fn(),
    closeMobileSessionPane: vi.fn(),
    ...overrides,
  };

  const [mobilePanel, setMobilePanel] = useState<"nav" | "sessions" | null>(() => {
    if (baseContextValue.mobileNavOpen) {
      return "nav";
    }
    if (baseContextValue.mobileSessionPaneOpen) {
      return "sessions";
    }
    return null;
  });
  const contextValue: WorkbenchContextValue = {
    ...baseContextValue,
    mobileNavOpen: mobilePanel === "nav",
    mobileSessionPaneOpen: mobilePanel === "sessions",
    toggleMobileNav: () => {
      baseContextValue.toggleMobileNav();
      setMobilePanel((current) => current === "nav" ? null : "nav");
    },
    toggleMobileSessionPane: () => {
      baseContextValue.toggleMobileSessionPane();
      setMobilePanel((current) => current === "sessions" ? null : "sessions");
    },
    closeMobileNav: () => {
      baseContextValue.closeMobileNav();
      setMobilePanel((current) => current === "nav" ? null : current);
    },
    closeMobileSessionPane: () => {
      baseContextValue.closeMobileSessionPane();
      setMobilePanel((current) => current === "sessions" ? null : current);
    },
  };

  return (
    <WorkbenchContext.Provider value={contextValue}>
      <ConversationWorkspace language="en" />
    </WorkbenchContext.Provider>
  );
}

function renderWorkspace(overrides: Partial<WorkbenchContextValue> = {}) {
  return render(<WorkspaceTestFrame overrides={overrides} />);
}

describe("ConversationWorkspace", () => {
  beforeEach(() => {
    window.history.pushState({}, "", "/chat");
    runtimeMock.route = "chat";
    runtimeMock.compact = true;
    runtimeMock.inspectorOpen = false;
    runtimeMock.inspectorTab = "model";
    runtimeMock.inspectorTabOpen = true;
    runtimeMock.activeSession = {
      id: "session-1",
      status: "ready",
      title: "New",
      messages: [],
    };
    runtimeMock.sessions = [runtimeMock.activeSession];
    runtimeMock.sessionItems = [
      {
        id: "session-1",
        title: "New",
        meta: "now",
        shortHash: "abcd1234",
        createdAt: Date.parse("2026-04-23T09:00:00Z"),
        active: true,
      },
    ];
    runtimeMock.target = { type: "model", id: "raw-model", name: "Raw Model" };
    runtimeMock.selectedProviderId = "";
    runtimeMock.selectedModelId = "";
    runtimeMock.selectedModelLabel = "DeepSeek V3.2";
    runtimeMock.selectedModelSupportsVision = true;
    runtimeMock.providers = [];
    runtimeMock.capabilities = [];
    runtimeMock.skills = [];
    runtimeMock.toolCount = 0;
    runtimeMock.skillCount = 0;
    runtimeMock.draft = "";
    runtimeMock.draftAttachments = [];
    runtimeMock.createSession.mockClear();
    runtimeMock.focusSession.mockClear();
    runtimeMock.removeSession.mockClear();
    runtimeMock.addDraftAttachments.mockClear();
    runtimeMock.removeDraftAttachment.mockClear();
    runtimeMock.clearDraftAttachments.mockClear();
    runtimeMock.sendPrompt.mockClear();
    runtimeMock.toggleInspector.mockClear();
    runtimeMock.closeInspector.mockClear();
    runtimeMock.selectModel.mockClear();
    runtimeMock.toggleSkill.mockClear();
    buildChatTimelineItemsMock.mockClear();
  });

  it("renders the markdown syntax demo timeline when the chat demo query is present", () => {
    window.history.pushState({}, "", "/chat?markdown_demo=1");

    renderWorkspace({ isMobileViewport: false });

    expect(buildChatTimelineItemsMock).toHaveBeenCalledWith(expect.objectContaining({
      cacheScope: "session-1:markdown-demo",
      language: "en",
      messages: [
        expect.objectContaining({
          id: "markdown-syntax-demo-assistant",
          role: "assistant",
          text: conversationMarkdownSyntaxFixture.markdown,
          status: "done",
        }),
      ],
    }));
    expect(document.querySelector(".conversation-empty-state")).not.toBeInTheDocument();
  });

  it("uses the markdown syntax demo query as an explicit non-persistent timeline preview", () => {
    window.history.pushState({}, "", "/chat?markdown_demo=1");
    runtimeMock.activeSession = {
      id: "session-1",
      title: "Existing",
      messages: [
        {
          id: "real-message-1",
          role: "assistant",
          text: "Existing session message",
          attachments: [],
          route: "chat",
          source: "chat",
          error: false,
          status: "done",
          at: Date.parse("2026-04-23T09:00:00Z"),
          processSteps: [],
        },
      ],
    };

    renderWorkspace({ isMobileViewport: false });

    expect(buildChatTimelineItemsMock).toHaveBeenCalledWith(expect.objectContaining({
      cacheScope: "session-1:markdown-demo",
      messages: [
        expect.objectContaining({
          id: "markdown-syntax-demo-assistant",
          text: conversationMarkdownSyntaxFixture.markdown,
        }),
      ],
    }));
  });

  it("renders the Chat workspace as the only conversation runtime route", () => {
    const toggleMobileNav = vi.fn();
    const toggleMobileSessionPane = vi.fn();
    renderWorkspace({ toggleMobileNav, toggleMobileSessionPane });

    expect(document.querySelector("[data-runtime-view='conversation']")).toHaveAttribute("data-runtime-route", "chat");
    expect(screen.getByTestId("conversation-session-pane")).toHaveAttribute("data-session-pane-placement", "navigation");
    expect(screen.getByTestId("conversation-session-pane")).toHaveAttribute("aria-hidden", "true");
    expect(screen.getByTestId("conversation-session-pane").querySelector(".runtime-session-summary-row")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add attachment" })).toHaveAttribute("data-runtime-composer-upload", "chat");
    expect(screen.getByRole("button", { name: "Send" })).toHaveAttribute("data-runtime-submit", "chat");
    const sessionButton = screen.getByRole("button", { name: "Session" });
    expect(sessionButton).toHaveClass("runtime-composer-utility");
    expect(sessionButton).toHaveAttribute("data-runtime-composer-utility", "session");

    fireEvent.click(screen.getByRole("button", { name: "Menu" }));
    expect(toggleMobileNav).toHaveBeenCalledTimes(1);
    expect(toggleMobileSessionPane).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole("button", { name: "New" }));
    expect(runtimeMock.createSession).toHaveBeenCalledTimes(1);
  });

  it("renders Chat jump controls and timeline blocks", () => {
    runtimeMock.activeSession = {
      id: "session-1",
      status: "ready",
      title: "Session with messages",
      messages: [
        { id: "msg-1", role: "assistant", text: "One", at: Date.now(), status: "done" },
        { id: "msg-2", role: "assistant", text: "Two", at: Date.now(), status: "done" },
      ],
    };
    runtimeMock.sessions = [runtimeMock.activeSession];

    renderWorkspace();

    expect(screen.getByRole("button", { name: "Top" })).toHaveAttribute("data-scroll-jump-top", "chat");
    expect(screen.getByRole("button", { name: "Previous" })).toHaveAttribute("data-scroll-jump-prev", "chat");
    expect(screen.getByRole("button", { name: "Next" })).toHaveAttribute("data-scroll-jump-next", "chat");
    expect(screen.getByRole("button", { name: "Latest" })).toHaveAttribute("data-scroll-jump-bottom", "chat");
    expect(buildChatTimelineItemsMock).toHaveBeenCalledWith(expect.objectContaining({
      messages: runtimeMock.activeSession.messages,
    }));
  });

  it("keeps Details focused on Chat metadata without Agent Runtime panels", () => {
    runtimeMock.providers = [
      {
        id: "alter0-codex",
        name: "Codex",
        models: [{ id: "codex", name: "Codex", supportsVision: true, active: false }],
      },
    ];
    runtimeMock.capabilities = [
      { id: "filesystem", name: "Filesystem", description: "Read workspace files", kind: "mcp", active: true },
    ];
    runtimeMock.skills = [
      { id: "frontend-design", name: "Frontend Design", description: "UI guidance", kind: "skill", active: true },
    ];
    renderWorkspace({ isMobileViewport: false });

    fireEvent.click(screen.getByRole("button", { name: "Details" }));
    const detailsPanel = document.querySelector("[data-runtime-details-panel='conversation']") as HTMLElement;
    expect(detailsPanel).toBeInTheDocument();
    expect(within(detailsPanel).getByText("Session")).toBeInTheDocument();
    expect(within(detailsPanel).getByText("Chat")).toBeInTheDocument();
    expect(within(detailsPanel).queryByRole("tab", { name: "Agent" })).not.toBeInTheDocument();
    expect(within(detailsPanel).queryByRole("tab", { name: "Deliverables" })).not.toBeInTheDocument();
    expect(within(detailsPanel).queryByRole("tab", { name: "Session Profile" })).not.toBeInTheDocument();
    expect(within(detailsPanel).queryByRole("tab", { name: "Skills" })).not.toBeInTheDocument();
    expect(detailsPanel.querySelector("[data-runtime-config-panel='conversation-details']")).not.toBeInTheDocument();
  });

  it("lets Chat composer Session update public skill selections", () => {
    runtimeMock.skills = [
      { id: "frontend-design", name: "Frontend Design", description: "UI guidance", kind: "skill", active: false },
    ];
    renderWorkspace({ isMobileViewport: false });

    fireEvent.click(screen.getByRole("button", { name: "Session" }));
    expect(runtimeMock.toggleInspector).toHaveBeenCalledWith();
    runtimeMock.inspectorOpen = true;
    runtimeMock.inspectorTab = "skills";
    renderWorkspace({ isMobileViewport: false });
    const configPanel = document.querySelector("[data-runtime-config-panel='conversation']") as HTMLElement;
    expect(configPanel).toBeInTheDocument();
    fireEvent.click(within(configPanel).getByLabelText(/Frontend Design/));

    expect(runtimeMock.toggleSkill).toHaveBeenCalledWith("frontend-design", true);
  });

  it("keeps Chat skill selections reachable from the mobile composer Session button", () => {
    runtimeMock.skills = [
      { id: "frontend-design", name: "Frontend Design", description: "UI guidance", kind: "skill", active: false },
    ];
    renderWorkspace({ isMobileViewport: true });

    fireEvent.click(screen.getByRole("button", { name: "Session" }));
    expect(runtimeMock.toggleInspector).toHaveBeenCalledWith();
    runtimeMock.inspectorOpen = true;
    runtimeMock.inspectorTab = "skills";
    renderWorkspace({ isMobileViewport: true });
    const configPanel = document.querySelector("[data-runtime-config-panel='conversation']") as HTMLElement;
    expect(configPanel).toBeInTheDocument();
    fireEvent.click(within(configPanel).getByLabelText(/Frontend Design/));

    expect(runtimeMock.toggleSkill).toHaveBeenCalledWith("frontend-design", true);
  });

  it("shows only public skill selections provided by the runtime context", () => {
    runtimeMock.skills = [
      { id: "frontend-design", name: "Frontend Design", description: "UI guidance", kind: "skill", active: true },
    ];
    runtimeMock.inspectorOpen = true;
    runtimeMock.inspectorTab = "skills";

    renderWorkspace({ isMobileViewport: false });

    expect(screen.getByText("Frontend Design")).toBeInTheDocument();
    expect(screen.queryByText("Writing Agent Skill")).not.toBeInTheDocument();
  });

  it("keeps legacy agent sessions as normal Chat session rows", () => {
    runtimeMock.activeSession = {
      id: "legacy-agent-1",
      status: "ready",
      title: "Travel Plan",
      messages: [],
      target: { type: "agent", id: "travel", name: "Travel Planner" },
    };
    runtimeMock.sessions = [runtimeMock.activeSession];
    runtimeMock.sessionItems = [
      {
        id: "legacy-agent-1",
        title: "Travel Plan",
        meta: "2026-04-23 17:00",
        shortHash: "zzzz1111",
        createdAt: Date.parse("2026-04-23T09:00:00Z"),
        active: true,
      },
    ];

    renderWorkspace();

    const sessionPane = screen.getByTestId("conversation-session-pane");
    expect(within(sessionPane).getByText("Travel Plan")).toBeInTheDocument();
    expect(sessionPane.querySelector(".runtime-session-context")).not.toBeInTheDocument();
    expect(screen.queryByText("Travel Planner")).not.toBeInTheDocument();
  });

  it("removes the Agent Runtime empty-state picker while keeping the composer session control", () => {
    renderWorkspace({ isMobileViewport: true });

    expect(screen.queryByRole("radiogroup", { name: "Choose agent" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Session" })).toHaveAttribute("data-runtime-composer-utility", "session");
    expect(screen.getByRole("button", { name: "Add attachment" })).toBeInTheDocument();
  });

  it("submits the current Chat draft on the first send action", () => {
    runtimeMock.draft = "Run this";
    renderWorkspace();

    fireEvent.click(screen.getByRole("button", { name: "Send" }));

    expect(runtimeMock.sendPrompt).toHaveBeenCalledWith("Run this");
  });
});

import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { useState } from "react";
import { ConversationWorkspace } from "./ConversationWorkspace";
import { WorkbenchContext, type WorkbenchContextValue } from "../../app/WorkbenchContext";

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
  activeAgent: null,
  activeSessionProfile: null,
  lockedTarget: false,
  targetOptions: [],
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
  selectTarget: vi.fn(),
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

function renderWorkspace(overrides: Partial<WorkbenchContextValue> = {}) {
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

  function ConversationWorkspaceHarness() {
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

  return render(
    <ConversationWorkspaceHarness />,
  );
}

describe("ConversationWorkspace", () => {
  beforeEach(() => {
    runtimeMock.route = "chat";
    runtimeMock.inspectorOpen = false;
    runtimeMock.inspectorTab = "model";
    runtimeMock.inspectorTabOpen = true;
    runtimeMock.activeSession = {
      id: "session-1",
      title: "New",
      messages: [],
    };
    runtimeMock.sessions = [
      {
        id: "session-1",
        title: "New",
        messages: [],
      },
    ];
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
    runtimeMock.activeAgent = null;
    runtimeMock.activeSessionProfile = null;
    runtimeMock.inspectorOpen = false;
    runtimeMock.inspectorTab = "model";
    runtimeMock.inspectorTabOpen = true;
    runtimeMock.selectedModelLabel = "DeepSeek V3.2";
    runtimeMock.selectedModelSupportsVision = true;
    runtimeMock.providers = [];
    runtimeMock.capabilities = [];
    runtimeMock.skills = [];
    runtimeMock.toolCount = 0;
    runtimeMock.skillCount = 0;
    runtimeMock.draft = "";
    runtimeMock.createSession.mockClear();
    runtimeMock.focusSession.mockClear();
    runtimeMock.removeSession.mockClear();
    runtimeMock.draftAttachments = [];
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

  it("keeps the shared workspace header visible alongside terminal-style mobile actions for an empty chat workspace", () => {
    const toggleMobileNav = vi.fn();
    const toggleMobileSessionPane = vi.fn();
    renderWorkspace({ toggleMobileNav, toggleMobileSessionPane });

    expect(document.querySelector("[data-runtime-view='conversation']")).toHaveClass("runtime-workspace-view");
    expect(document.querySelector("[data-runtime-view='conversation']")).toHaveAttribute("data-runtime-route", "chat");
    expect(Array.from(document.querySelector("[data-runtime-view='conversation']")?.children || []).map((node) =>
      (node as HTMLElement).tagName.toLowerCase(),
    )).toEqual(["aside", "section"]);
    expect(screen.getByTestId("conversation-session-pane")).toHaveClass("runtime-workspace-session-pane");
    expect(screen.getByTestId("conversation-session-pane")).not.toHaveClass("terminal-session-pane");
    expect(screen.getByTestId("conversation-session-pane")).not.toHaveClass("conversation-session-pane");
    expect(within(screen.getByTestId("conversation-session-pane")).getByRole("list")).toHaveAttribute(
      "data-runtime-session-list",
      "conversation",
    );
    expect(within(screen.getByTestId("conversation-session-pane")).getAllByRole("listitem")).toHaveLength(1);
    expect(document.querySelector(".runtime-session-main")).toBeInTheDocument();
    expect(document.querySelector(".runtime-session-title-row")).toBeInTheDocument();
    expect(document.querySelector(".runtime-session-title-row")?.textContent).toContain("New");
    expect(document.querySelector(".runtime-session-summary-row")).toBeInTheDocument();
    expect(document.querySelector(".runtime-session-summary-row")).toHaveTextContent("now");
    expect(document.querySelector(".runtime-session-summary-row")).toHaveTextContent("#abcd1234");
    expect(document.querySelector(".runtime-session-bottomline")).not.toBeInTheDocument();
    expect(document.querySelector(".runtime-session-badge")).not.toBeInTheDocument();
    expect(document.querySelector("[data-runtime-session-list='conversation']")).toHaveClass("runtime-session-list");
    expect(document.querySelector("[data-runtime-workspace='conversation']")).toHaveClass("runtime-workspace");
    expect(document.querySelector("[data-runtime-workspace-page='true']")).toBeInTheDocument();
    expect(document.querySelector(".runtime-workspace-body")).not.toHaveClass("terminal-workspace-body");
    expect(document.querySelector(".runtime-workspace-body")).not.toHaveClass("conversation-workspace-body");
    expect(document.querySelector("[data-runtime-screen='conversation']")).toHaveClass("runtime-workspace-screen");
    expect(document.querySelector("[data-runtime-timeline='true']")).toBeInTheDocument();
    expect(document.querySelector("[data-runtime-session-pane-head='true']")).toBeInTheDocument();
    const sessionPaneHead = document.querySelector("[data-runtime-session-pane-head='true']") as HTMLElement;
    expect(Array.from(sessionPaneHead.children).map((node) => (node as HTMLElement).className)).toEqual([
      "runtime-workspace-session-pane-copy",
      "runtime-workspace-session-pane-actions",
    ]);
    const sessionPaneCopy = sessionPaneHead.querySelector(".runtime-workspace-session-pane-copy") as HTMLElement;
    expect(sessionPaneCopy).toBeInTheDocument();
    expect(within(sessionPaneCopy).getByText("Sessions")).toBeInTheDocument();
    expect(within(sessionPaneCopy).getByText("1 sessions")).toBeInTheDocument();
    expect(sessionPaneHead.textContent).toContain("New");
    expect(sessionPaneHead.textContent).toContain("Hide");
    expect(document.querySelector("[data-runtime-screen='conversation']")?.closest(".conversation-console-panel"))
      .toBe(document.querySelector(".runtime-workspace-body > .conversation-console-panel"));

    const mobileHeader = document.querySelector(".runtime-workspace-body > [data-runtime-mobile-variant='conversation']") as HTMLElement;
    expect(mobileHeader).toBeInTheDocument();
    expect(mobileHeader).toHaveAttribute("data-runtime-mobile-header", "body");
    expect(within(mobileHeader).getByRole("button", { name: "Menu" })).toHaveClass(
      "runtime-workspace-mobile-action",
    );
    expect(within(mobileHeader).getByRole("button", { name: "Sessions" })).toHaveClass(
      "runtime-workspace-mobile-action",
    );
    expect(within(mobileHeader).getByRole("button", { name: "New" })).toHaveClass(
      "runtime-workspace-mobile-action",
    );
    const hideSessionsAction = within(screen.getByTestId("conversation-session-pane"))
      .getAllByRole("button", { name: "Hide" })
      .find((button) => button.classList.contains("runtime-workspace-session-pane-action"));
    expect(hideSessionsAction).toHaveClass("runtime-workspace-session-pane-action");
    const workspaceHeader = document.querySelector(".runtime-workspace-head") as HTMLElement;
    expect(workspaceHeader).toHaveAttribute("data-runtime-workspace-header", "true");
    expect(workspaceHeader).toHaveClass("is-sticky");
    expect(screen.getByRole("heading", { name: "New" })).toBeInTheDocument();
    const statusIndicator = within(workspaceHeader).getByLabelText("Ready");
    expect(statusIndicator).toHaveClass(
      "workspace-header-status",
      "is-ready",
    );
    expect(statusIndicator).toHaveAttribute("data-runtime-header-signal-container", "ready");
    expect(statusIndicator.querySelector("[data-runtime-header-signal='ready']")).toBeInTheDocument();
    const titleLeading = workspaceHeader.querySelector(".runtime-workspace-title-leading") as HTMLElement;
    expect(titleLeading.firstElementChild).toBe(statusIndicator);
    expect(within(workspaceHeader).getByRole("button", { name: "Details" })).toHaveClass("workspace-header-details");
    expect(within(workspaceHeader).queryByRole("button", { name: "Workspace Flow" })).not.toBeInTheDocument();
    expect(within(workspaceHeader).queryByRole("button", { name: "Model" })).not.toBeInTheDocument();
    expect(within(workspaceHeader).queryByRole("button", { name: "Tools / MCP" })).not.toBeInTheDocument();
    expect(document.querySelector(".runtime-composer-shell")).toBeInTheDocument();
    expect(document.querySelector(".runtime-composer-form[data-runtime-composer-kind='chat']")).toHaveClass("runtime-composer-form");
    expect(document.querySelector(".runtime-composer-body")).toBeInTheDocument();
    expect(document.querySelector(".runtime-composer-toolbar")).toBeInTheDocument();
    const composerToolbarStart = document.querySelector(".runtime-composer-toolbar-start") as HTMLElement;
    const composerToolbarEnd = document.querySelector(".runtime-composer-toolbar-end") as HTMLElement;
    expect(composerToolbarStart).toBeInTheDocument();
    expect(composerToolbarEnd).toBeInTheDocument();
    expect(document.querySelector("[data-runtime-composer-input='chat']")).toHaveClass("runtime-composer-input");
    expect(document.querySelector("[data-runtime-composer-submit='chat']")).toHaveClass("runtime-composer-submit");
    expect(screen.queryByRole("button", { name: "Quick tools" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Mention" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Workspace tools" })).not.toBeInTheDocument();
    const sessionButton = screen.getByRole("button", { name: "Session" });
    const addAttachmentButton = screen.getByRole("button", { name: "Add attachment" });
    expect(sessionButton).toHaveClass("runtime-composer-utility");
    expect(sessionButton).not.toHaveClass("is-pill");
    expect(addAttachmentButton).toHaveClass("runtime-composer-upload");
    expect(addAttachmentButton.querySelector(".runtime-composer-upload-icon svg")).toBeInTheDocument();
    expect(addAttachmentButton.querySelector(".runtime-composer-upload-label")).toHaveClass("sr-only");
    expect(composerToolbarStart).toContainElement(addAttachmentButton);
    expect(composerToolbarEnd).toContainElement(screen.getByRole("button", { name: "Send" }));
    expect(document.querySelector(".terminal-composer-shell")).not.toBeInTheDocument();
    expect(document.querySelector(".conversation-composer-shell")).not.toBeInTheDocument();
    expect(document.querySelector(".terminal-chat-form")).not.toBeInTheDocument();
    expect(document.querySelector(".conversation-chat-form")).not.toBeInTheDocument();
    expect(document.querySelector(".terminal-composer-input")).not.toBeInTheDocument();
    expect(document.querySelector(".conversation-composer-input")).not.toBeInTheDocument();
    expect(document.querySelector(".terminal-chat-submit")).not.toBeInTheDocument();
    expect(document.querySelector(".conversation-chat-submit")).not.toBeInTheDocument();
    expect(screen.getByTestId("conversation-session-pane")).toHaveAttribute("data-mobile-open", "false");

    fireEvent.click(within(workspaceHeader).getByRole("button", { name: "Details" }));
    const detailsPanel = document.querySelector("[data-runtime-details-panel='conversation']") as HTMLElement;
    expect(detailsPanel).toBeInTheDocument();
    expect(within(detailsPanel).getByText("Session")).toBeInTheDocument();
    expect(within(detailsPanel).queryByText("OpenRouter")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Session" }));
    expect(runtimeMock.toggleInspector).toHaveBeenLastCalledWith("model");

    fireEvent.click(within(mobileHeader).getByRole("button", { name: "Menu" }));
    expect(toggleMobileNav).toHaveBeenCalledTimes(1);

    fireEvent.click(within(mobileHeader).getByRole("button", { name: "Sessions" }));
    expect(screen.getByTestId("conversation-session-pane")).toHaveAttribute("data-mobile-open", "true");
    expect(toggleMobileSessionPane).toHaveBeenCalledTimes(1);

    fireEvent.click(within(mobileHeader).getByRole("button", { name: "New" }));
    expect(runtimeMock.createSession).toHaveBeenCalledTimes(1);
  });

  it("renders the four shared jump buttons for chat timelines", () => {
    runtimeMock.activeSession = {
      id: "session-1",
      title: "New",
      messages: [
        { id: "message-1" },
        { id: "message-2" },
      ],
    } as typeof runtimeMock.activeSession;

    renderWorkspace({ isMobileViewport: false });

    expect(document.querySelector("[data-scroll-jump-top='chat']")).toBeInTheDocument();
    expect(document.querySelector("[data-scroll-jump-prev='chat']")).toBeInTheDocument();
    expect(document.querySelector("[data-scroll-jump-next='chat']")).toBeInTheDocument();
    expect(document.querySelector("[data-scroll-jump-bottom='chat']")).toBeInTheDocument();
  });

  it("renders the four shared jump buttons for agent timelines", () => {
    runtimeMock.route = "agent-runtime";
    runtimeMock.activeSession = {
      id: "session-1",
      title: "Agent session",
      messages: [
        { id: "message-1" },
        { id: "message-2" },
      ],
    } as typeof runtimeMock.activeSession;

    renderWorkspace({ isMobileViewport: false });

    expect(document.querySelector("[data-scroll-jump-top='agent']")).toBeInTheDocument();
    expect(document.querySelector("[data-scroll-jump-prev='agent']")).toBeInTheDocument();
    expect(document.querySelector("[data-scroll-jump-next='agent']")).toBeInTheDocument();
    expect(document.querySelector("[data-scroll-jump-bottom='agent']")).toBeInTheDocument();
  });

  it("hides shared jump buttons while the mobile composer is focused", () => {
    runtimeMock.activeSession = {
      id: "session-1",
      title: "New",
      messages: [
        { id: "message-1" },
        { id: "message-2" },
      ],
    } as typeof runtimeMock.activeSession;

    renderWorkspace({ isMobileViewport: true });

    const composerInput = screen.getByLabelText("Type a message to continue this workspace...") as HTMLTextAreaElement;
    fireEvent.focus(composerInput);

    expect(document.querySelector("[data-scroll-jump-top='chat']")).not.toBeInTheDocument();
    expect(document.querySelector("[data-scroll-jump-prev='chat']")).not.toBeInTheDocument();
    expect(document.querySelector("[data-scroll-jump-next='chat']")).not.toBeInTheDocument();
    expect(document.querySelector("[data-scroll-jump-bottom='chat']")).not.toBeInTheDocument();

    fireEvent.blur(composerInput);

    expect(document.querySelector("[data-scroll-jump-top='chat']")).toBeInTheDocument();
    expect(document.querySelector("[data-scroll-jump-prev='chat']")).toBeInTheDocument();
    expect(document.querySelector("[data-scroll-jump-next='chat']")).toBeInTheDocument();
    expect(document.querySelector("[data-scroll-jump-bottom='chat']")).toBeInTheDocument();
  });

  it("groups session rows into recency sections like a workspace sidebar", () => {
    const todayStart = new Date();
    todayStart.setHours(0, 0, 0, 0);
    const yesterdayStart = new Date(todayStart);
    yesterdayStart.setDate(todayStart.getDate() - 1);
    const earlierStart = new Date(todayStart);
    earlierStart.setDate(todayStart.getDate() - 5);
    runtimeMock.sessionItems = [
      {
        id: "session-1",
        title: "Ship session sidebar refresh",
        meta: "just now",
        shortHash: "abcd1234",
        createdAt: todayStart.getTime() + (2 * 60 * 60 * 1000),
        active: true,
      },
      {
        id: "session-2",
        title: "Review Gemini layout notes",
        meta: "2 hr ago",
        shortHash: "efgh5678",
        createdAt: yesterdayStart.getTime() + (2 * 60 * 60 * 1000),
        active: false,
      },
      {
        id: "session-3",
        title: "Archive older shell ideas",
        meta: "36 hr ago",
        shortHash: "ijkl9012",
        createdAt: earlierStart.getTime() + (2 * 60 * 60 * 1000),
        active: false,
      },
    ];

    renderWorkspace({ isMobileViewport: false });

    const sessionPane = screen.getByTestId("conversation-session-pane");
    expect(within(sessionPane).getByText("Sessions")).toBeInTheDocument();
    expect(within(sessionPane).getByRole("button", { name: "New" })).toBeInTheDocument();
    expect(within(sessionPane).queryByRole("button", { name: "New Chat" })).not.toBeInTheDocument();
    expect(within(sessionPane).queryByRole("button", { name: "New Agent Run" })).not.toBeInTheDocument();
    expect(within(sessionPane).getByText("Today")).toBeInTheDocument();
    expect(within(sessionPane).getByText("Yesterday")).toBeInTheDocument();
    expect(within(sessionPane).getByText("Earlier")).toBeInTheDocument();
    expect(within(sessionPane).getAllByRole("button", { name: "Delete session" })).toHaveLength(3);
    expect(within(sessionPane).getAllByRole("listitem")).toHaveLength(3);
    const targetCard = within(sessionPane)
      .getAllByRole("listitem")
      .find((item) => item.textContent?.includes("Review Gemini layout notes")) as HTMLElement;
    expect(targetCard).toBeInTheDocument();
    expect(targetCard.querySelector(".runtime-session-title-row")).toBeInTheDocument();
    expect(targetCard.querySelector(".runtime-session-summary-row")).toHaveTextContent("2 hr ago");
    expect(targetCard.querySelector(".runtime-session-summary-row")).toHaveTextContent("#efgh5678");
    expect(targetCard.querySelector(".runtime-session-bottomline")).not.toBeInTheDocument();
    expect(targetCard.querySelector(".runtime-session-badge")).not.toBeInTheDocument();
  });

  it("shows the concrete agent name inside agent-runtime session cards", () => {
    runtimeMock.route = "agent-runtime";
    runtimeMock.target = { type: "agent", id: "coding", name: "Coding Agent" };
    runtimeMock.activeAgent = {
      id: "coding",
      name: "Coding Agent",
      description: "Handles repository work",
      deliverables: [],
      session_profile_fields: [],
    };
    runtimeMock.sessionItems = [
      {
        id: "agent-session-1",
        title: "Refine preview deployment flow",
        meta: "18 min ago",
        contextLabel: "Coding Agent",
        shortHash: "code7788",
        createdAt: Date.parse("2026-04-23T11:18:00Z"),
        active: true,
      },
    ];

    renderWorkspace({ route: "agent-runtime", isMobileViewport: false });

    const sessionPane = screen.getByTestId("conversation-session-pane");
    const card = within(sessionPane).getAllByRole("listitem")[0] as HTMLElement;
    expect(card.querySelector(".runtime-session-context")).toHaveTextContent("Coding Agent");
    expect(card.querySelector(".runtime-session-summary-row")).toHaveTextContent("18 min ago");
    expect(card.querySelector(".runtime-session-summary-row")).toHaveTextContent("#code7788");
    expect(card.querySelector(".runtime-session-bottomline")).not.toBeInTheDocument();
    expect(card.querySelector(".runtime-session-title")).toHaveTextContent("Refine preview deployment flow");
    const titleRow = card.querySelector(".runtime-session-title-row") as HTMLElement;
    const titleCopy = card.querySelector(".runtime-session-title-copy") as HTMLElement;
    expect(titleRow).toBeInTheDocument();
    expect(titleCopy).toBeInTheDocument();
    expect(titleRow.querySelector(".runtime-session-signal")).toBeInTheDocument();
    expect(Array.from(titleCopy.children).map((node) => (node as HTMLElement).className)).toEqual([
      "runtime-session-title",
      "runtime-session-context",
    ]);
  });

  it("shows a Codex chip in the chat model selector and forwards selection", () => {
    runtimeMock.inspectorOpen = true;
    runtimeMock.inspectorTab = "model";
    runtimeMock.inspectorTabOpen = true;
    runtimeMock.providers = [
      {
        id: "openai",
        name: "OpenAI",
        models: [
          { id: "gpt-5.4", name: "GPT-5.4", supportsVision: true, active: false },
        ],
      },
      {
        id: "alter0-codex",
        name: "Codex",
        models: [
          { id: "codex", name: "Codex", supportsVision: true, active: true },
        ],
      },
    ];
    runtimeMock.selectedProviderId = "alter0-codex";
    runtimeMock.selectedModelId = "codex";
    runtimeMock.selectedModelLabel = "Codex";

    renderWorkspace({ isMobileViewport: false });

    const codexButton = screen.getByRole("button", { name: "Codex" });
    expect(codexButton).toBeInTheDocument();

    fireEvent.click(codexButton);
    expect(runtimeMock.selectModel).toHaveBeenCalledWith("alter0-codex", "codex");
  });

  it("keeps session details separate from composer configuration panels", () => {
    runtimeMock.inspectorOpen = true;
    runtimeMock.inspectorTab = "model";
    runtimeMock.inspectorTabOpen = true;
    runtimeMock.providers = [
      {
        id: "openrouter",
        name: "OpenRouter",
        models: [
          { id: "deepseek-v3.2", name: "DeepSeek V3.2", supportsVision: true, active: true },
        ],
      },
    ];

    const view = renderWorkspace({ isMobileViewport: false });

    expect(document.querySelector("[data-runtime-details-panel='conversation']")).not.toBeInTheDocument();
    expect(screen.getByText("OpenRouter")).toBeInTheDocument();

    const workspaceHeader = document.querySelector(".runtime-workspace-head") as HTMLElement;
    fireEvent.click(within(workspaceHeader).getByRole("button", { name: "Details" }));

    const detailsPanel = document.querySelector("[data-runtime-details-panel='conversation']") as HTMLElement;
    expect(detailsPanel).toBeInTheDocument();
    expect(within(detailsPanel).getByText("Session")).toBeInTheDocument();
    expect(within(detailsPanel).queryByText("OpenRouter")).not.toBeInTheDocument();

    view.unmount();
  });

  it("does not rebuild the timeline when only the composer draft changes", () => {
    runtimeMock.activeSession = {
      id: "session-1",
      title: "New",
      messages: [
        {
          id: "message-1",
          role: "assistant",
          text: "Existing reply",
          attachments: [],
          route: "nl",
          source: "codex_exec",
          status: "done",
          error: false,
          at: Date.parse("2026-04-23T09:01:00Z"),
          processSteps: [],
          agentProcessCollapsed: false,
          taskID: "",
          taskStatus: "",
          taskPending: false,
          taskResultDelivered: false,
          taskResultFor: "",
        },
      ],
    };
    runtimeMock.sessions = [
      {
        id: "session-1",
        title: "New",
        status: "ready",
        messages: runtimeMock.activeSession.messages,
      },
    ];

    const contextValue: WorkbenchContextValue = {
      route: "chat",
      language: "en",
      navigate: vi.fn(),
      isMobileViewport: false,
      mobileNavOpen: false,
      mobileSessionPaneOpen: false,
      toggleMobileNav: vi.fn(),
      toggleMobileSessionPane: vi.fn(),
      closeMobileNav: vi.fn(),
      closeMobileSessionPane: vi.fn(),
    };
    const tree = (
      <WorkbenchContext.Provider value={contextValue}>
        <ConversationWorkspace language="en" />
      </WorkbenchContext.Provider>
    );
    const view = render(tree);
    expect(buildChatTimelineItemsMock).toHaveBeenCalledTimes(1);

    runtimeMock.draft = "draft update";
    view.rerender(tree);

    expect(buildChatTimelineItemsMock).toHaveBeenCalledTimes(1);
  });

  it("keeps agent private skills locked and only lists public skills as available", () => {
    runtimeMock.route = "agent-runtime";
    runtimeMock.inspectorOpen = true;
    runtimeMock.inspectorTab = "skills";
    runtimeMock.inspectorTabOpen = true;
    runtimeMock.skills = [
      {
        id: "agent-skill-travel",
        name: "Travel Agent Skill",
        description: "Private reusable rulebook for travel pages",
        kind: "skill",
        active: true,
        visibility: "agent-private",
        locked: true,
      },
      {
        id: "deploy-test-service",
        name: "Deploy Test Service",
        description: "Deploy verification workflow",
        kind: "skill",
        active: true,
        visibility: "public",
        locked: false,
      },
      {
        id: "frontend-design",
        name: "Frontend Design",
        description: "Shared frontend delivery standards",
        kind: "skill",
        active: false,
        visibility: "public",
        locked: false,
      },
      {
        id: "agent-skill-writing",
        name: "Writing Agent Skill",
        description: "Private reusable writing rules",
        kind: "skill",
        active: false,
        visibility: "agent-private",
        locked: true,
      },
    ];

    renderWorkspace({ isMobileViewport: false });

    const travelLabel = screen.getByText("Travel Agent Skill").closest("label") as HTMLElement;
    const privateCheckbox = within(travelLabel).getByRole("checkbox") as HTMLInputElement;
    expect(privateCheckbox).toBeChecked();
    expect(privateCheckbox).toBeDisabled();

    fireEvent.click(privateCheckbox);
    expect(runtimeMock.toggleSkill).not.toHaveBeenCalledWith("agent-skill-travel", false);

    expect(screen.getByText("Deploy Test Service")).toBeInTheDocument();
    expect(screen.getByText("Frontend Design")).toBeInTheDocument();
    expect(screen.queryByText("Writing Agent Skill")).not.toBeInTheDocument();
  });

  it("focuses the mobile composer on first touch so keyboard handling matches terminal", () => {
    renderWorkspace();

    const composerInput = screen.getByLabelText("Type a message to continue this workspace...") as HTMLTextAreaElement;
    const focusSpy = vi.spyOn(composerInput, "focus");

    fireEvent.pointerDown(composerInput, { pointerType: "touch" });

    expect(focusSpy).toHaveBeenCalled();
  });

  it("dismisses the session panel when the composer input is pressed", () => {
    runtimeMock.inspectorOpen = true;
    runtimeMock.inspectorTab = "model";
    runtimeMock.inspectorTabOpen = true;
    runtimeMock.providers = [
      {
        id: "openrouter",
        name: "OpenRouter",
        models: [
          { id: "deepseek-v3.2", name: "DeepSeek V3.2", supportsVision: true, active: true },
        ],
      },
    ];

    renderWorkspace({ isMobileViewport: false });

    const composerInput = screen.getByLabelText("Type a message to continue this workspace...") as HTMLTextAreaElement;
    fireEvent.pointerDown(composerInput, { pointerType: "mouse" });

    expect(runtimeMock.closeInspector).toHaveBeenCalledTimes(1);
  });

  it("opens the session panel on the first mobile touch while the composer is focused", () => {
    renderWorkspace({ isMobileViewport: true });

    const composerInput = screen.getByLabelText("Type a message to continue this workspace...") as HTMLTextAreaElement;
    fireEvent.focus(composerInput);
    fireEvent.touchStart(screen.getByRole("button", { name: "Session" }));

    expect(runtimeMock.toggleInspector).toHaveBeenLastCalledWith("model");
  });

  it("marks the shared runtime composer input as plain text so mobile autofill bars stay off", () => {
    renderWorkspace();

    const composerInput = screen.getByLabelText("Type a message to continue this workspace...") as HTMLTextAreaElement;

    expect(composerInput).toHaveAttribute("autocomplete", "off");
    expect(composerInput).toHaveAttribute("autocorrect", "off");
    expect(composerInput).toHaveAttribute("autocapitalize", "off");
    expect(composerInput).toHaveAttribute("enterkeyhint", "send");
    expect(composerInput).toHaveAttribute("spellcheck", "false");
  });

  it("renders draft image thumbnails with preview and remove actions", () => {
    runtimeMock.draftAttachments = [
      {
        id: "image-1",
        kind: "image",
        name: "diagram.png",
        contentType: "image/png",
        size: 1024,
        assetURL: "/api/sessions/session-1/attachments/image-1/original",
        previewURL: "/api/sessions/session-1/attachments/image-1/preview",
      },
    ];
    renderWorkspace({ isMobileViewport: false });

    expect(screen.getByRole("button", { name: "Add attachment" })).toHaveClass("runtime-composer-upload");
    expect(screen.getByRole("button", { name: "Preview diagram.png" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Remove diagram.png" })).toBeInTheDocument();
    expect(screen.getAllByRole("img", { name: "diagram.png" })[0]).toHaveAttribute(
      "src",
      "/api/sessions/session-1/attachments/image-1/preview",
    );

    fireEvent.click(screen.getByRole("button", { name: "Preview diagram.png" }));
    expect(document.querySelector("[data-runtime-attachment-preview='true']")).toBeInTheDocument();
    expect(screen.getByRole("dialog", { name: "diagram.png" })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Close preview" }));
    expect(screen.queryByRole("dialog", { name: "diagram.png" })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Remove diagram.png" }));
    expect(runtimeMock.removeDraftAttachment).toHaveBeenCalledWith("image-1");
  });

  it("forwards selected files to the draft attachment handler", async () => {
    renderWorkspace({ isMobileViewport: false });

    const fileInput = document.querySelector('input[type="file"][accept*=".md"]') as HTMLInputElement;
    expect(fileInput).toBeInTheDocument();

    const file = new File(["workspace notes"], "notes.md", { type: "text/markdown" });
    fireEvent.change(fileInput, { target: { files: [file] } });

    await waitFor(() => {
      expect(runtimeMock.addDraftAttachments).toHaveBeenCalledTimes(1);
    });
  });

  it("shows only the draft character count in composer meta", () => {
    renderWorkspace({ isMobileViewport: false });

    expect(document.querySelector(".runtime-composer-meta")).not.toBeInTheDocument();
  });

  it("closes the mobile session pane after selecting a session", () => {
    renderWorkspace();

    fireEvent.click(screen.getByRole("button", { name: "Sessions" }));
    expect(screen.getByTestId("conversation-session-pane")).toHaveAttribute("data-mobile-open", "true");

    const sessionSelect = screen.getByTestId("conversation-session-pane").querySelector(".runtime-session-select") as HTMLButtonElement;
    fireEvent.click(sessionSelect);
    expect(runtimeMock.focusSession).toHaveBeenCalledWith("session-1");
    expect(screen.getByTestId("conversation-session-pane")).toHaveAttribute("data-mobile-open", "false");
  });

  it("maps a streaming conversation session to a busy status signal", () => {
    runtimeMock.activeSession = {
      id: "session-1",
      title: "Run deployment check",
      messages: [
        {
          id: "message-1",
          role: "assistant",
          text: "Thinking...",
          attachments: [],
          route: "",
          source: "",
          error: false,
          status: "streaming",
          at: Date.parse("2026-04-23T09:02:00Z"),
          processSteps: [],
          taskID: "",
          taskStatus: "",
          taskPending: false,
          taskResultDelivered: false,
          taskResultFor: "",
        },
      ],
    };
    runtimeMock.sessions = [runtimeMock.activeSession];
    runtimeMock.sessionItems = [
      {
        id: "session-1",
        title: "Run deployment check",
        meta: "now",
        shortHash: "abcd1234",
        createdAt: Date.parse("2026-04-23T09:00:00Z"),
        active: true,
      },
    ];

    renderWorkspace({ isMobileViewport: false });

    expect(document.querySelector("[data-runtime-session-card='session-1']")).toHaveAttribute("data-runtime-session-tone", "busy");
    expect(document.querySelector("[data-runtime-session-signal='busy']")).toBeInTheDocument();
    expect(screen.getByLabelText("Busy")).toBeInTheDocument();
  });

  it("maps a failed conversation session to a failed status signal", () => {
    runtimeMock.activeSession = {
      id: "session-1",
      title: "Fix failing request",
      messages: [
        {
          id: "message-1",
          role: "assistant",
          text: "Request failed",
          attachments: [],
          route: "",
          source: "",
          error: true,
          status: "error",
          at: Date.parse("2026-04-23T09:02:00Z"),
          processSteps: [],
          taskID: "",
          taskStatus: "",
          taskPending: false,
          taskResultDelivered: false,
          taskResultFor: "",
        },
      ],
    };
    runtimeMock.sessions = [runtimeMock.activeSession];
    runtimeMock.sessionItems = [
      {
        id: "session-1",
        title: "Fix failing request",
        meta: "now",
        shortHash: "abcd1234",
        createdAt: Date.parse("2026-04-23T09:00:00Z"),
        active: true,
      },
    ];

    renderWorkspace({ isMobileViewport: false });

    expect(document.querySelector("[data-runtime-session-card='session-1']")).toHaveAttribute("data-runtime-session-tone", "failed");
    expect(document.querySelector("[data-runtime-session-signal='failed']")).toBeInTheDocument();
    expect(screen.getByLabelText("Failed")).toBeInTheDocument();
  });

  it("maps a registry-backed failed status to a failed signal even before assistant messages load", () => {
    runtimeMock.activeSession = {
      id: "session-1",
      status: "failed",
      title: "Travel publish blocked",
      messages: [],
    };
    runtimeMock.sessions = [runtimeMock.activeSession];
    runtimeMock.sessionItems = [
      {
        id: "session-1",
        title: "Travel publish blocked",
        meta: "now",
        shortHash: "abcd1234",
        createdAt: Date.parse("2026-04-23T09:00:00Z"),
        active: true,
      },
    ];

    renderWorkspace({ isMobileViewport: false });

    expect(document.querySelector("[data-runtime-session-card='session-1']")).toHaveAttribute("data-runtime-session-tone", "failed");
    expect(document.querySelector("[data-runtime-session-signal='failed']")).toBeInTheDocument();
    expect(screen.getByLabelText("Failed")).toBeInTheDocument();
  });

  it("keeps the mobile session pane mutually exclusive with the menu overlay", () => {
    const toggleMobileNav = vi.fn();

    renderWorkspace({ toggleMobileNav });

    fireEvent.click(screen.getByRole("button", { name: "Sessions" }));
    expect(screen.getByTestId("conversation-session-pane")).toHaveAttribute("data-mobile-open", "true");

    fireEvent.click(screen.getByRole("button", { name: "Menu" }));

    expect(toggleMobileNav).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("conversation-session-pane")).toHaveAttribute("data-mobile-open", "false");
  });

  it("keeps the workspace title row when the conversation already has messages", () => {
    runtimeMock.activeSession = {
      id: "session-1",
      title: "Fix runtime shell",
      messages: [
        {
          id: "message-1",
          role: "assistant",
          content: "done",
          html: "<p>done</p>",
        },
      ],
    };
    runtimeMock.sessionItems = [
      {
        id: "session-1",
        title: "Fix runtime shell",
        meta: "now",
        shortHash: "abcd1234",
        createdAt: Date.parse("2026-04-23T09:00:00Z"),
        active: true,
      },
    ];

    renderWorkspace();

    const workspaceHeader = document.querySelector(".runtime-workspace-head") as HTMLElement;
    expect(document.querySelector(".runtime-workspace-shell")).toBeInTheDocument();
    expect(document.querySelector("[data-testid='conversation-session-pane']")).toHaveClass("runtime-workspace-session-pane");
    expect(document.querySelector(".runtime-workspace-body")).toBeInTheDocument();
    expect(document.querySelector("[data-runtime-screen='conversation']")).toHaveClass("runtime-workspace-screen");
    expect(document.querySelector(".runtime-workspace-head")).toHaveClass("is-compact");
    expect(document.querySelector(".runtime-composer-form")).toHaveAttribute("data-runtime-composer", "true");
    expect(document.querySelector(".runtime-composer-form")).toHaveAttribute("data-runtime-composer-kind", "chat");
    expect(document.querySelector(".runtime-composer-form")).toBeInTheDocument();
    expect(document.querySelector(".runtime-composer-input")).toBeInTheDocument();
    expect(document.querySelector(".runtime-composer-submit")).toBeInTheDocument();
    expect(document.querySelector(".runtime-composer-tools")).toBeInTheDocument();
    expect(document.querySelector(".runtime-composer-meta")).not.toBeInTheDocument();
    expect(document.querySelector("[data-runtime-attachment-strip='true']")).not.toBeInTheDocument();
    expect(document.querySelector(".runtime-composer-submit .runtime-composer-submit-icon svg")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Fix runtime shell" })).toBeInTheDocument();
    expect(screen.getByLabelText("Ready")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Details" })).toBeInTheDocument();
    expect(within(workspaceHeader).queryByRole("button", { name: "Model" })).not.toBeInTheDocument();
    expect(within(workspaceHeader).queryByRole("button", { name: "Tools" })).not.toBeInTheDocument();
    expect(within(workspaceHeader).queryByRole("button", { name: "Tools / MCP" })).not.toBeInTheDocument();
    expect(screen.queryByText("Chat")).not.toBeInTheDocument();
    expect(screen.queryByText("DeepSeek V3.2 · 0 / 0")).not.toBeInTheDocument();
  });

  it("keeps the agent-runtime compact header visible on mobile empty state", () => {
    runtimeMock.route = "agent-runtime";
    runtimeMock.activeSession = {
      id: "session-1",
      title: "New",
      messages: [],
    };
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
    runtimeMock.target = { type: "agent", id: "alter0", name: "Alter0" };

    renderWorkspace({ route: "agent-runtime" });

    const mobileHeader = document.querySelector(".runtime-workspace-body > [data-runtime-mobile-variant='conversation']") as HTMLElement;
    const workspaceHeader = document.querySelector(".runtime-workspace-head") as HTMLElement;
    expect(mobileHeader).toBeInTheDocument();
    expect(within(mobileHeader).getByRole("button", { name: "Menu" })).toBeInTheDocument();
    expect(within(mobileHeader).getByRole("button", { name: "Sessions" })).toBeInTheDocument();
    expect(within(mobileHeader).getByRole("button", { name: "New" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "New" })).toBeInTheDocument();
    expect(screen.getByLabelText("Ready")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Details" })).toBeInTheDocument();
    expect(within(workspaceHeader).queryByRole("button", { name: "Model" })).not.toBeInTheDocument();
    expect(within(workspaceHeader).queryByRole("button", { name: "Agent" })).not.toBeInTheDocument();
    expect(within(workspaceHeader).queryByRole("button", { name: "Choose Agent" })).not.toBeInTheDocument();
    expect(document.querySelector(".runtime-composer-form")).toHaveAttribute("data-runtime-composer-kind", "agent");
    expect(document.querySelector("[data-runtime-composer-input='agent']")).toHaveClass("runtime-composer-input");
    expect(document.querySelector("[data-runtime-composer-submit='agent']")).toHaveClass("runtime-composer-submit");
  });

  it("keeps the agent-runtime workspace row compact when messages already exist on mobile", () => {
    runtimeMock.route = "agent-runtime";
    runtimeMock.activeSession = {
      id: "session-1",
      title: "Investigate release drift",
      messages: [
        {
          id: "message-1",
          role: "assistant",
          content: "done",
          html: "<p>done</p>",
        },
      ],
    };
    runtimeMock.sessionItems = [
      {
        id: "session-1",
        title: "Investigate release drift",
        meta: "now",
        shortHash: "abcd1234",
        createdAt: Date.parse("2026-04-23T09:00:00Z"),
        active: true,
      },
    ];
    runtimeMock.target = { type: "agent", id: "alter0", name: "Alter0" };

    renderWorkspace({ route: "agent-runtime" });

    const workspaceHeader = document.querySelector(".runtime-workspace-head") as HTMLElement;
    expect(screen.getByRole("heading", { name: "Investigate release drift" })).toBeInTheDocument();
    expect(screen.getByLabelText("Ready")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Details" })).toBeInTheDocument();
    expect(within(workspaceHeader).queryByRole("button", { name: "Model" })).not.toBeInTheDocument();
    expect(within(workspaceHeader).queryByRole("button", { name: "Agent" })).not.toBeInTheDocument();
    expect(within(workspaceHeader).queryByRole("button", { name: "Choose Agent" })).not.toBeInTheDocument();
  });

  it("keeps the desktop empty-state workspace summary visible", () => {
    renderWorkspace({ isMobileViewport: false });

    const workspaceHeader = document.querySelector(".runtime-workspace-head") as HTMLElement;
    expect(document.querySelector("[data-runtime-mobile-variant='conversation']")).not.toBeInTheDocument();
    expect(document.querySelector(".runtime-composer-shell")).toBeInTheDocument();
    expect(document.querySelector(".conversation-console-panel")).toHaveClass("is-empty");
    expect(document.querySelector("[data-runtime-screen='conversation']")).toHaveClass("runtime-workspace-screen", "is-empty");
    expect(screen.getByRole("heading", { name: "New" })).toBeInTheDocument();
    expect(screen.getByLabelText("Ready")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Details" })).toBeInTheDocument();
    expect(within(workspaceHeader).queryByRole("button", { name: "Model" })).not.toBeInTheDocument();
    expect(within(workspaceHeader).queryByRole("button", { name: "Tools / MCP" })).not.toBeInTheDocument();
    expect(screen.queryByText("Chat")).not.toBeInTheDocument();
    expect(screen.queryByText("DeepSeek V3.2 · 0 / 0")).not.toBeInTheDocument();
  });

  it("submits the current draft value on the first send action", () => {
    runtimeMock.draft = "ship the runtime refactor";

    renderWorkspace({ isMobileViewport: false });

    fireEvent.click(screen.getByRole("button", { name: "Send" }));
    expect(runtimeMock.sendPrompt).toHaveBeenCalledWith("ship the runtime refactor");
  });

  it("submits immediately when the mobile send button is tapped", () => {
    runtimeMock.draft = "ship the mobile tap path";

    renderWorkspace({ isMobileViewport: true });

    fireEvent.touchStart(screen.getByRole("button", { name: "Send" }));
    expect(runtimeMock.sendPrompt).toHaveBeenCalledWith("ship the mobile tap path");
  });

  it("submits when the mobile send button is pressed through touch pointer while the composer stays focused", () => {
    runtimeMock.draft = "ship with keyboard open";

    renderWorkspace({ isMobileViewport: true });

    const composerInput = screen.getByLabelText("Type a message to continue this workspace...") as HTMLTextAreaElement;
    fireEvent.focus(composerInput);
    fireEvent.pointerDown(screen.getByRole("button", { name: "Send" }), { pointerType: "touch" });

    expect(runtimeMock.sendPrompt).toHaveBeenCalledWith("ship with keyboard open");
  });

  it("blurs the agent-runtime composer when the mobile send button is tapped", async () => {
    runtimeMock.route = "agent-runtime";
    runtimeMock.target = { type: "agent", id: "alter0", name: "Alter0" };
    runtimeMock.draft = "close the mobile keyboard";

    renderWorkspace({ route: "agent-runtime", isMobileViewport: true });

    const composerInput = screen.getByLabelText("Type a message to continue this workspace...") as HTMLTextAreaElement;
    await act(async () => {
      composerInput.focus();
    });
    expect(composerInput).toHaveFocus();

    await act(async () => {
      fireEvent.touchStart(screen.getByRole("button", { name: "Send" }));
    });

    expect(runtimeMock.sendPrompt).toHaveBeenCalledWith("close the mobile keyboard");
    expect(composerInput).not.toHaveFocus();
  });

  it("keeps the agent-runtime header summary visible outside the mobile empty state", () => {
    runtimeMock.route = "agent-runtime";
    runtimeMock.activeSession = {
      id: "session-1",
      title: "Investigate release drift",
      messages: [],
    };
    runtimeMock.sessionItems = [
      {
        id: "session-1",
        title: "Investigate release drift",
        meta: "now",
        shortHash: "abcd1234",
        createdAt: Date.parse("2026-04-23T09:00:00Z"),
        active: true,
      },
    ];
    runtimeMock.target = { type: "agent", id: "alter0", name: "Alter0" };
    runtimeMock.inspectorOpen = true;
    runtimeMock.inspectorTab = "model";

    renderWorkspace({ route: "agent-runtime", isMobileViewport: false });

    const workspaceHeader = document.querySelector(".runtime-workspace-head") as HTMLElement;
    expect(screen.getByRole("heading", { name: "Investigate release drift" })).toBeInTheDocument();
    expect(within(workspaceHeader).getByLabelText("Ready")).toBeInTheDocument();
    expect(within(workspaceHeader).getByRole("button", { name: "Details" })).toBeInTheDocument();
    expect(within(workspaceHeader).queryByRole("button", { name: "Model" })).not.toBeInTheDocument();
    expect(within(workspaceHeader).queryByRole("button", { name: "Choose Agent" })).not.toBeInTheDocument();

    expect(screen.getByRole("button", { name: "Session" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Agent" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Deliverables" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Model" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Tools" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Skills" })).toBeInTheDocument();

    fireEvent.click(within(workspaceHeader).getByRole("button", { name: "Details" }));
    const detailsPanel = document.querySelector("[data-runtime-details-panel='conversation']") as HTMLElement;
    expect(detailsPanel).toBeInTheDocument();
    expect(document.querySelector(".runtime-workspace-head")?.contains(detailsPanel)).toBe(false);
    expect(within(detailsPanel).queryByRole("button", { name: "Choose Agent" })).not.toBeInTheDocument();
    expect(within(detailsPanel).queryByRole("button", { name: "Model" })).not.toBeInTheDocument();
  });

  it("shows agent session profile fields inside details for runtime sessions", () => {
    runtimeMock.route = "agent-runtime";
    runtimeMock.inspectorOpen = true;
    runtimeMock.inspectorTab = "session-profile";
    runtimeMock.activeSession = {
      id: "session-agent-1",
      title: "Coding run",
      messages: [],
    };
    runtimeMock.sessionItems = [
      {
        id: "session-agent-1",
        title: "Coding run",
        meta: "now",
        shortHash: "ff12aa45",
        createdAt: Date.parse("2026-04-23T09:00:00Z"),
        active: true,
      },
    ];
    runtimeMock.target = { type: "agent", id: "coding", name: "Coding Agent" };
    runtimeMock.activeAgent = {
      id: "coding",
      name: "Coding Agent",
      description: "Dedicated coding agent",
      session_profile_fields: [
        { key: "repository_path", label: "Repository", readonly: true },
        { key: "branch", label: "Branch", readonly: true },
        { key: "preview_subdomain", label: "Preview Subdomain", readonly: true },
      ],
    };
    runtimeMock.activeSessionProfile = {
      agent_id: "coding",
      session_id: "session-agent-1",
      path: ".alter0/agents/coding/sessions/session-agent-1.md",
      exists: true,
      fields: [
        { key: "repository_path", label: "Repository", readonly: true },
        { key: "branch", label: "Branch", readonly: true },
        { key: "preview_subdomain", label: "Preview Subdomain", readonly: true },
      ],
      attributes: {
        repository_path: "/workspace/alter0-remote",
        branch: "feature/session-profile-schema",
        preview_subdomain: "coding-run-42",
      },
    };

    renderWorkspace({ isMobileViewport: false });

    fireEvent.click(screen.getByRole("button", { name: "Details" }));

    const detailsPanel = document.querySelector("[data-runtime-details-panel='conversation']") as HTMLElement;
    expect(within(detailsPanel).getAllByText("Repository")[0]).toBeInTheDocument();
    expect(within(detailsPanel).getByText("/workspace/alter0-remote")).toBeInTheDocument();
    expect(within(detailsPanel).getByText("feature/session-profile-schema")).toBeInTheDocument();
    expect(within(detailsPanel).getByText("coding-run-42")).toBeInTheDocument();
  });

  it("shows the active agent delivery contract with resolved session outputs", () => {
    runtimeMock.route = "agent-runtime";
    runtimeMock.inspectorOpen = true;
    runtimeMock.inspectorTab = "deliverables";
    runtimeMock.activeSession = {
      id: "session-travel-1",
      title: "Wuhan Guide",
      messages: [],
    };
    runtimeMock.sessionItems = [
      {
        id: "session-travel-1",
        title: "Wuhan Guide",
        meta: "now",
        shortHash: "wu42aa18",
        createdAt: Date.parse("2026-04-23T09:00:00Z"),
        active: true,
      },
    ];
    runtimeMock.target = { type: "agent", id: "travel", name: "Travel Agent" };
    runtimeMock.activeAgent = {
      id: "travel",
      name: "Travel Agent",
      description: "Travel planning specialist",
      deliverables: [
        {
          id: "guide-markdown",
          label: "Travel Guide",
          description: "Structured city guide aligned with the current trip request.",
          format: "markdown",
          required: true,
        },
        {
          id: "guide-html",
          label: "HTML Guide",
          description: "Published HTML guide for the current session.",
          format: "html",
          required: true,
          session_attribute_key: "guide_html_url",
        },
      ],
      session_profile_fields: [
        { key: "guide_html_url", label: "Guide HTML URL", readonly: true },
      ],
    };
    runtimeMock.activeSessionProfile = {
      agent_id: "travel",
      session_id: "session-travel-1",
      path: ".alter0/agents/travel/sessions/session-travel-1.md",
      exists: true,
      fields: [
        { key: "guide_html_url", label: "Guide HTML URL", readonly: true },
      ],
      attributes: {
        guide_html_url: "https://travel-wu42aa18.alter0.cn",
      },
    };

    renderWorkspace({ isMobileViewport: false });

    expect(screen.getByRole("tab", { name: "Deliverables" })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Details" }));

    const detailsPanel = document.querySelector("[data-runtime-details-panel='conversation']") as HTMLElement;
    expect(within(detailsPanel).getAllByText("Delivery Contract")[0]).toBeInTheDocument();
    expect(within(detailsPanel).getByText("Travel Guide")).toBeInTheDocument();
    expect(within(detailsPanel).getByText("HTML Guide")).toBeInTheDocument();
    expect(within(detailsPanel).getByText("https://travel-wu42aa18.alter0.cn")).toBeInTheDocument();
  });

  it("copies readonly session profile values from the details panel", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, {
      clipboard: {
        writeText,
      },
    });

    runtimeMock.route = "agent-runtime";
    runtimeMock.inspectorOpen = true;
    runtimeMock.inspectorTab = "session-profile";
    runtimeMock.activeSession = {
      id: "session-travel-1",
      title: "Wuhan Guide",
      messages: [],
    };
    runtimeMock.sessionItems = [
      {
        id: "session-travel-1",
        title: "Wuhan Guide",
        meta: "now",
        shortHash: "wu42aa18",
        createdAt: Date.parse("2026-04-23T09:00:00Z"),
        active: true,
      },
    ];
    runtimeMock.target = { type: "agent", id: "travel", name: "Travel Agent" };
    runtimeMock.activeAgent = {
      id: "travel",
      name: "Travel Agent",
      description: "Travel planning specialist",
      deliverables: [],
      session_profile_fields: [
        { key: "guide_html_url", label: "Guide HTML URL", readonly: true },
      ],
    };
    runtimeMock.activeSessionProfile = {
      agent_id: "travel",
      session_id: "session-travel-1",
      path: ".alter0/agents/travel/sessions/session-travel-1.md",
      exists: true,
      fields: [
        { key: "guide_html_url", label: "Guide HTML URL", readonly: true },
      ],
      attributes: {
        guide_html_url: "https://travel-wu42aa18.alter0.cn",
      },
    };

    renderWorkspace({ isMobileViewport: false });
    fireEvent.click(screen.getByRole("button", { name: "Details" }));

    const detailsPanel = document.querySelector("[data-runtime-details-panel='conversation']") as HTMLElement;
    const copyButtons = within(detailsPanel).getAllByRole("button", { name: "Copy value" });

    fireEvent.click(copyButtons.at(-1) as HTMLButtonElement);

    await waitFor(() => {
      expect(writeText).toHaveBeenCalledWith("https://travel-wu42aa18.alter0.cn");
    });
  });
});

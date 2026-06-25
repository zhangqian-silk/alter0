import { act, fireEvent, render, screen, within } from "@testing-library/react";
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
      draft: true,
      pinned: false,
      pinning: false,
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
    visibility?: "public" | "private";
    locked?: boolean;
  }>,
  runtimeEventFilter: ["important_text"] as Array<"important_text" | "plan" | "reasoning" | "tools" | "commands" | "system">,
  toolCount: 0,
  skillCount: 0,
  busy: false,
  createSession: vi.fn(),
  focusSession: vi.fn(),
  removeSession: vi.fn().mockResolvedValue(undefined),
  setSessionPinned: vi.fn().mockResolvedValue(undefined),
  refreshActiveSession: vi.fn().mockResolvedValue(undefined),
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
  toggleRuntimeEventFilter: vi.fn(),
  toggleProcess: vi.fn(),
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
        draft: true,
        pinned: false,
        pinning: false,
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
    runtimeMock.runtimeEventFilter = ["important_text"];
    runtimeMock.toolCount = 0;
    runtimeMock.skillCount = 0;
    runtimeMock.busy = false;
    runtimeMock.draft = "";
    runtimeMock.draftAttachments = [];
    runtimeMock.createSession.mockClear();
    runtimeMock.focusSession.mockClear();
    runtimeMock.removeSession.mockClear();
    runtimeMock.setSessionPinned.mockClear();
    runtimeMock.refreshActiveSession.mockClear();
    runtimeMock.addDraftAttachments.mockClear();
    runtimeMock.removeDraftAttachment.mockClear();
    runtimeMock.clearDraftAttachments.mockClear();
    runtimeMock.sendPrompt.mockClear();
    runtimeMock.toggleInspector.mockClear();
    runtimeMock.closeInspector.mockClear();
    runtimeMock.selectModel.mockClear();
    runtimeMock.toggleSkill.mockClear();
    runtimeMock.toggleRuntimeEventFilter.mockClear();
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

  it("passes the selected runtime event disclosure filter to the chat timeline", () => {
    runtimeMock.runtimeEventFilter = ["important_text", "reasoning"];

    renderWorkspace({ isMobileViewport: false });

    expect(buildChatTimelineItemsMock).toHaveBeenCalledWith(expect.objectContaining({
      runtimeEventFilter: ["important_text", "reasoning"],
    }));
  });

  it("clears opened process event details when toggling the outer Thinking disclosure", () => {
    runtimeMock.activeSession = {
      ...runtimeMock.activeSession,
      messages: [
        {
          id: "assistant-1",
          role: "assistant",
          text: "",
          attachments: [],
          route: "chat",
          source: "terminal",
          error: false,
          status: "running",
          at: Date.parse("2026-04-23T09:01:00Z"),
          processEvents: [
            { id: "step-1", title: "Message", detail: "Progress note." },
            { id: "step-2", title: "Shell", detail: "git status" },
          ],
          processCollapsed: false,
        },
      ],
    };
    runtimeMock.sessions = [runtimeMock.activeSession];

    renderWorkspace({ isMobileViewport: true });

    let timelineOptions = buildChatTimelineItemsMock.mock.calls[buildChatTimelineItemsMock.mock.calls.length - 1][0];
    expect(timelineOptions.expandedProcessEvents).toEqual({});

    act(() => {
      timelineOptions.onToggleProcessEvent("assistant-1", "step-1");
    });

    timelineOptions = buildChatTimelineItemsMock.mock.calls[buildChatTimelineItemsMock.mock.calls.length - 1][0];
    expect(timelineOptions.expandedProcessEvents).toEqual({ "assistant-1:step-1": true });

    act(() => {
      timelineOptions.onToggleProcess("assistant-1");
    });

    expect(runtimeMock.toggleProcess).toHaveBeenCalledWith("assistant-1");
    timelineOptions = buildChatTimelineItemsMock.mock.calls[buildChatTimelineItemsMock.mock.calls.length - 1][0];
    expect(timelineOptions.expandedProcessEvents).toEqual({});
  });

  it("renders process disclosure checkboxes in the model inspector", () => {
    runtimeMock.inspectorOpen = true;
    runtimeMock.inspectorTab = "model";

    renderWorkspace({ isMobileViewport: false });

    expect(screen.getByRole("checkbox", { name: /Important text/i })).toBeChecked();
    expect(screen.getByRole("checkbox", { name: /Reasoning/i })).not.toBeChecked();

    fireEvent.click(screen.getByRole("checkbox", { name: /Reasoning/i }));

    expect(runtimeMock.toggleRuntimeEventFilter).toHaveBeenCalledWith("reasoning", true);
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
          processEvents: [],
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

  it("disables the Chat composer while the active Terminal-backed session is busy", () => {
    runtimeMock.busy = true;
    runtimeMock.activeSession = {
      ...runtimeMock.activeSession,
      status: "busy",
    };

    renderWorkspace({ isMobileViewport: false });

    expect(screen.getByRole("textbox", { name: /Type a message/i })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Add attachment" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Send" })).toBeDisabled();
  });

  it("opens Chat session details from the session row without deleting the session", () => {
    runtimeMock.activeSession = {
      ...runtimeMock.activeSession,
      messages: [{ id: "msg-1", role: "user", text: "hello", at: Date.now(), status: "done" }],
      serverBacked: true,
    };
    runtimeMock.sessions = [runtimeMock.activeSession];
    runtimeMock.sessionItems = [{ ...runtimeMock.sessionItems[0], draft: false }];
    const closeMobileSessionPane = vi.fn();
    renderWorkspace({ isMobileViewport: false, closeMobileSessionPane });

    const sessionPane = screen.getByTestId("conversation-session-pane");
    closeMobileSessionPane.mockClear();
    fireEvent.click(within(sessionPane).getByRole("button", { name: "Session actions", hidden: true }));
    fireEvent.click(within(sessionPane).getByRole("menuitem", {
      name: "View session details",
      hidden: true,
    }));

    expect(runtimeMock.focusSession).toHaveBeenCalledWith("session-1");
    expect(runtimeMock.removeSession).not.toHaveBeenCalled();
    expect(closeMobileSessionPane).not.toHaveBeenCalled();
    expect(document.querySelector("[data-runtime-details-panel='conversation']")).toBeInTheDocument();
  });

  it("does not show real session actions for a draft Chat New placeholder", () => {
    const closeMobileSessionPane = vi.fn();
    renderWorkspace({ isMobileViewport: true, mobileSessionPaneOpen: true, closeMobileSessionPane });

    const sessionPane = screen.getByTestId("conversation-session-pane");
    expect(within(sessionPane).queryByRole("button", { name: "Session actions", hidden: true })).not.toBeInTheDocument();
    const draftCard = sessionPane.querySelector("[data-runtime-session-card='session-1']") as HTMLElement;
    fireEvent.click(within(draftCard).getByRole("button", { name: /New/, hidden: true }));

    expect(runtimeMock.focusSession).toHaveBeenCalledWith("session-1");
    expect(closeMobileSessionPane).toHaveBeenCalled();
    expect(document.querySelector("[data-runtime-details-panel='conversation']")).not.toBeInTheDocument();
  });

  it("pins Chat sessions directly from the session row action", () => {
    runtimeMock.activeSession = {
      ...runtimeMock.activeSession,
      messages: [{ id: "msg-1", role: "user", text: "hello", at: Date.now(), status: "done" }],
      serverBacked: true,
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
        draft: false,
        pinned: false,
        pinning: false,
      },
    ];
    const closeMobileSessionPane = vi.fn();
    renderWorkspace({ isMobileViewport: false, closeMobileSessionPane });

    const sessionPane = screen.getByTestId("conversation-session-pane");
    closeMobileSessionPane.mockClear();
    fireEvent.click(within(sessionPane).getByRole("button", { name: "Session actions", hidden: true }));
    fireEvent.click(within(sessionPane).getByRole("menuitem", {
      name: "Pin session",
      hidden: true,
    }));

    expect(runtimeMock.setSessionPinned).toHaveBeenCalledWith("session-1", true);
    expect(runtimeMock.focusSession).not.toHaveBeenCalled();
    expect(runtimeMock.removeSession).not.toHaveBeenCalled();
    expect(closeMobileSessionPane).not.toHaveBeenCalled();
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
    runtimeMock.sessionItems = [{ ...runtimeMock.sessionItems[0], draft: false }];

    renderWorkspace();

    expect(screen.getByRole("button", { name: "Top" })).toHaveAttribute("data-scroll-jump-top", "chat");
    expect(screen.getByRole("button", { name: "Previous" })).toHaveAttribute("data-scroll-jump-prev", "chat");
    expect(screen.getByRole("button", { name: "Next" })).toHaveAttribute("data-scroll-jump-next", "chat");
    expect(screen.getByRole("button", { name: "Latest" })).toHaveAttribute("data-scroll-jump-bottom", "chat");
    expect(buildChatTimelineItemsMock).toHaveBeenCalledWith(expect.objectContaining({
      messages: runtimeMock.activeSession.messages,
    }));
  });

  it("keeps the already visible Chat history in the render window after a new turn is appended", () => {
    const initialMessages = Array.from({ length: 32 }, (_value, index) => ({
      id: `msg-${index + 1}`,
      role: index % 2 === 0 ? "user" : "assistant",
      text: `Message ${index + 1}`,
      attachments: [],
      route: "chat",
      source: "chat",
      error: false,
      status: "done",
      at: Date.parse("2026-04-23T09:00:00Z") + index,
      processEvents: [],
    }));
    runtimeMock.activeSession = {
      id: "session-1",
      status: "ready",
      title: "Session with full visible history",
      messages: initialMessages,
    };
    runtimeMock.sessions = [runtimeMock.activeSession];
    runtimeMock.sessionItems = [{ ...runtimeMock.sessionItems[0], draft: false }];

    const { rerender } = render(<WorkspaceTestFrame overrides={{ isMobileViewport: false }} />);
    expect(buildChatTimelineItemsMock).toHaveBeenLastCalledWith(expect.objectContaining({
      messages: initialMessages,
    }));

    const nextMessages = [
      ...initialMessages,
      {
        id: "msg-33",
        role: "user",
        text: "New prompt",
        attachments: [],
        route: "chat",
        source: "chat",
        error: false,
        status: "",
        at: Date.parse("2026-04-23T09:10:00Z"),
        processEvents: [],
      },
      {
        id: "msg-34",
        role: "assistant",
        text: "New response",
        attachments: [],
        route: "chat",
        source: "chat",
        error: false,
        status: "done",
        at: Date.parse("2026-04-23T09:10:01Z"),
        processEvents: [],
      },
    ];
    runtimeMock.activeSession = {
      ...runtimeMock.activeSession,
      messages: nextMessages,
    };
    runtimeMock.sessions = [runtimeMock.activeSession];

    rerender(<WorkspaceTestFrame overrides={{ isMobileViewport: false }} />);

    expect(buildChatTimelineItemsMock).toHaveBeenLastCalledWith(expect.objectContaining({
      messages: nextMessages,
    }));
  });

  it("refreshes the active Chat session when the user pulls the timeline back to the top", () => {
    runtimeMock.activeSession = {
      id: "session-1",
      status: "ready",
      title: "Session with loaded messages",
      messages: [
        { id: "msg-1", role: "assistant", text: "One", at: Date.now(), status: "done" },
        { id: "msg-2", role: "assistant", text: "Two", at: Date.now(), status: "done" },
      ],
    };
    runtimeMock.sessions = [runtimeMock.activeSession];
    runtimeMock.sessionItems = [{ ...runtimeMock.sessionItems[0], draft: false }];

    renderWorkspace({ isMobileViewport: false });

    const screenNode = document.querySelector("[data-runtime-screen='conversation']") as HTMLDivElement;
    expect(screenNode).toBeInTheDocument();
    fireEvent.scroll(screenNode, { target: { scrollTop: 0 } });

    expect(runtimeMock.refreshActiveSession).toHaveBeenCalledTimes(1);
  });

  it("keeps Details focused on Chat metadata without Chat panels", () => {
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
    runtimeMock.sessionItems = [{ ...runtimeMock.sessionItems[0], draft: false }];
    renderWorkspace({ isMobileViewport: false });

    fireEvent.click(screen.getByRole("button", { name: "Details" }));
    const detailsPanel = document.querySelector("[data-runtime-details-panel='conversation']") as HTMLElement;
    expect(detailsPanel).toBeInTheDocument();
    expect(within(detailsPanel).getByText("Session")).toBeInTheDocument();
    expect(within(detailsPanel).getByText("Chat")).toBeInTheDocument();
    expect(within(detailsPanel).queryByRole("tab", { name: "Skill" })).not.toBeInTheDocument();
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

  it("preserves the native mobile keyboard gesture when the Chat composer input is pressed", () => {
    renderWorkspace({ isMobileViewport: true });

    const input = document.querySelector("[data-runtime-composer-input='chat']") as HTMLTextAreaElement;
    const focusMock = vi.fn();
    Object.defineProperty(input, "focus", {
      configurable: true,
      value: focusMock,
    });
    const pointerEvent = new Event("pointerdown", {
      bubbles: true,
      cancelable: true,
    });
    Object.defineProperty(pointerEvent, "pointerType", {
      configurable: true,
      value: "touch",
    });
    const touchEvent = new Event("touchstart", {
      bubbles: true,
      cancelable: true,
    });

    input.dispatchEvent(pointerEvent);
    input.dispatchEvent(touchEvent);

    expect(pointerEvent.defaultPrevented).toBe(false);
    expect(touchEvent.defaultPrevented).toBe(false);
    expect(focusMock).not.toHaveBeenCalled();
  });

  it("shows only public skill selections provided by the runtime context", () => {
    runtimeMock.skills = [
      { id: "frontend-design", name: "Frontend Design", description: "UI guidance", kind: "skill", active: true },
    ];
    runtimeMock.inspectorOpen = true;
    runtimeMock.inspectorTab = "skills";

    renderWorkspace({ isMobileViewport: false });

    expect(screen.getByText("Frontend Design")).toBeInTheDocument();
    expect(screen.queryByText("Writing Skill Skill")).not.toBeInTheDocument();
  });

  it("keeps legacy chat sessions as normal Chat session rows", () => {
    runtimeMock.activeSession = {
      id: "legacy-skill-1",
      status: "ready",
      title: "Travel Plan",
      messages: [],
      target: { type: "skill", id: "travel", name: "Travel Skill" },
    };
    runtimeMock.sessions = [runtimeMock.activeSession];
    runtimeMock.sessionItems = [
      {
        id: "legacy-skill-1",
        title: "Travel Plan",
        meta: "2026-04-23 17:00",
        shortHash: "zzzz1111",
        createdAt: Date.parse("2026-04-23T09:00:00Z"),
        active: true,
        draft: false,
        pinned: false,
        pinning: false,
      },
    ];

    renderWorkspace();

    const sessionPane = screen.getByTestId("conversation-session-pane");
    expect(within(sessionPane).getByText("Travel Plan")).toBeInTheDocument();
    expect(sessionPane.querySelector(".runtime-session-context")).not.toBeInTheDocument();
    expect(screen.queryByText("Travel Skill")).not.toBeInTheDocument();
  });

  it("removes the Chat empty-state picker while keeping the composer session control", () => {
    renderWorkspace({ isMobileViewport: true });

    expect(screen.queryByRole("radiogroup", { name: "Choose skill" })).not.toBeInTheDocument();
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

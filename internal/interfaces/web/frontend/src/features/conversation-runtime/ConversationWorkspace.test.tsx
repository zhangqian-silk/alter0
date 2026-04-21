import { fireEvent, render, screen, within } from "@testing-library/react";
import { ConversationWorkspace } from "./ConversationWorkspace";
import { WorkbenchContext, type WorkbenchContextValue } from "../../app/WorkbenchContext";

const runtimeMock = {
  route: "chat" as const,
  compact: true,
  inspectorOpen: false,
  inspectorTab: "model" as const,
  sessions: [],
  activeSession: {
    id: "session-1",
    title: "New Chat",
    messages: [],
  },
  sessionItems: [
    {
      id: "session-1",
      title: "New Chat",
      meta: "now",
      shortHash: "abcd1234",
      active: true,
    },
  ],
  draft: "",
  target: { type: "model" as const, id: "raw-model", name: "Raw Model" },
  lockedTarget: false,
  targetOptions: [],
  selectedProviderId: "",
  selectedModelId: "",
  selectedModelLabel: "DeepSeek V3.2",
  providers: [],
  capabilities: [],
  skills: [],
  toolCount: 0,
  skillCount: 0,
  createSession: vi.fn(),
  focusSession: vi.fn(),
  removeSession: vi.fn().mockResolvedValue(undefined),
  setDraft: vi.fn(),
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
}));

vi.mock("../shell/components/ChatMessageRegion", () => ({
  ChatMessageRegion: () => <div data-testid="chat-message-region">messages</div>,
}));

function renderWorkspace(overrides: Partial<WorkbenchContextValue> = {}) {
  const contextValue: WorkbenchContextValue = {
    route: "chat",
    language: "en",
    navigate: vi.fn(),
    isMobileViewport: true,
    mobileNavOpen: false,
    toggleMobileNav: vi.fn(),
    closeMobileNav: vi.fn(),
    ...overrides,
  };

  return render(
    <WorkbenchContext.Provider value={contextValue}>
      <ConversationWorkspace language="en" />
    </WorkbenchContext.Provider>,
  );
}

describe("ConversationWorkspace", () => {
  beforeEach(() => {
    runtimeMock.route = "chat";
    runtimeMock.activeSession = {
      id: "session-1",
      title: "New Chat",
      messages: [],
    };
    runtimeMock.sessionItems = [
      {
        id: "session-1",
        title: "New Chat",
        meta: "now",
        shortHash: "abcd1234",
        active: true,
      },
    ];
    runtimeMock.target = { type: "model", id: "raw-model", name: "Raw Model" };
    runtimeMock.selectedModelLabel = "DeepSeek V3.2";
    runtimeMock.toolCount = 0;
    runtimeMock.skillCount = 0;
    runtimeMock.createSession.mockClear();
    runtimeMock.focusSession.mockClear();
    runtimeMock.removeSession.mockClear();
  });

  it("renders only terminal-style mobile actions for an empty chat workspace", () => {
    const toggleMobileNav = vi.fn();
    renderWorkspace({ toggleMobileNav });

    const mobileHeader = document.querySelector(".conversation-workspace-body > [data-conversation-mobile-header]") as HTMLElement;
    expect(mobileHeader).toBeInTheDocument();
    expect(within(mobileHeader).getByRole("button", { name: "Menu" })).toHaveClass(
      "conversation-mobile-action",
      "terminal-inline-button",
      "is-quiet",
    );
    expect(within(mobileHeader).getByRole("button", { name: "Sessions" })).toHaveClass(
      "conversation-mobile-action",
      "terminal-inline-button",
      "is-quiet",
    );
    expect(within(mobileHeader).getByRole("button", { name: "New" })).toHaveClass(
      "conversation-mobile-action",
      "terminal-inline-button",
      "is-primary",
    );
    expect(within(mobileHeader).getByRole("button", { name: "Menu" })).toBeInTheDocument();
    expect(within(mobileHeader).getByRole("button", { name: "Sessions" })).toBeInTheDocument();
    expect(within(mobileHeader).getByRole("button", { name: "New" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "New Chat" })).not.toBeInTheDocument();
    expect(screen.queryByText("DeepSeek V3.2 · 0 / 0")).not.toBeInTheDocument();
    expect(screen.getByTestId("conversation-session-pane")).toHaveAttribute("data-mobile-open", "false");

    fireEvent.click(within(mobileHeader).getByRole("button", { name: "Menu" }));
    expect(toggleMobileNav).toHaveBeenCalledTimes(1);

    fireEvent.click(within(mobileHeader).getByRole("button", { name: "Sessions" }));
    expect(screen.getByTestId("conversation-session-pane")).toHaveAttribute("data-mobile-open", "true");

    fireEvent.click(within(mobileHeader).getByRole("button", { name: "New" }));
    expect(runtimeMock.createSession).toHaveBeenCalledTimes(1);
  });

  it("closes the mobile session pane after selecting a session", () => {
    renderWorkspace();

    fireEvent.click(screen.getByRole("button", { name: "Sessions" }));
    expect(screen.getByTestId("conversation-session-pane")).toHaveAttribute("data-mobile-open", "true");

    fireEvent.click(within(screen.getByTestId("conversation-session-pane")).getByRole("button", { name: /New Chat/ }));
    expect(runtimeMock.focusSession).toHaveBeenCalledWith("session-1");
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
        active: true,
      },
    ];

    renderWorkspace();

    expect(screen.getByRole("heading", { name: "Fix runtime shell" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Model" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Tools" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Tools / MCP" })).not.toBeInTheDocument();
    expect(screen.queryByText("Chat")).not.toBeInTheDocument();
    expect(screen.queryByText("DeepSeek V3.2 · 0 / 0")).not.toBeInTheDocument();
  });

  it("hides the agent-runtime workspace summary row on mobile empty state", () => {
    runtimeMock.route = "agent-runtime";
    runtimeMock.activeSession = {
      id: "session-1",
      title: "New Agent Session",
      messages: [],
    };
    runtimeMock.sessionItems = [
      {
        id: "session-1",
        title: "New Agent Session",
        meta: "now",
        shortHash: "abcd1234",
        active: true,
      },
    ];
    runtimeMock.target = { type: "agent", id: "alter0", name: "Alter0" };

    renderWorkspace({ route: "agent-runtime" });

    const mobileHeader = document.querySelector(".conversation-workspace-body > [data-conversation-mobile-header]") as HTMLElement;
    expect(mobileHeader).toBeInTheDocument();
    expect(within(mobileHeader).getByRole("button", { name: "Menu" })).toBeInTheDocument();
    expect(within(mobileHeader).getByRole("button", { name: "Sessions" })).toBeInTheDocument();
    expect(within(mobileHeader).getByRole("button", { name: "New" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "New Agent Session" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Choose Agent" })).not.toBeInTheDocument();
    expect(screen.queryByText("Agent: Alter0")).not.toBeInTheDocument();
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
        active: true,
      },
    ];
    runtimeMock.target = { type: "agent", id: "alter0", name: "Alter0" };

    renderWorkspace({ route: "agent-runtime" });

    expect(screen.getByRole("heading", { name: "Investigate release drift" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Model" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Agent" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Choose Agent" })).not.toBeInTheDocument();
    expect(screen.queryByText("Agent: Alter0")).not.toBeInTheDocument();
  });

  it("keeps the desktop empty-state workspace summary visible", () => {
    renderWorkspace({ isMobileViewport: false });

    expect(document.querySelector("[data-conversation-mobile-header]")).not.toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "New Chat" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Model" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Tools / MCP" })).toBeInTheDocument();
    expect(screen.queryByText("Chat")).not.toBeInTheDocument();
    expect(screen.queryByText("DeepSeek V3.2 · 0 / 0")).not.toBeInTheDocument();
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
        active: true,
      },
    ];
    runtimeMock.target = { type: "agent", id: "alter0", name: "Alter0" };

    renderWorkspace({ route: "agent-runtime", isMobileViewport: false });

    expect(screen.getByRole("heading", { name: "Investigate release drift" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Model" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Choose Agent" })).toBeInTheDocument();
    expect(screen.queryByText("Agent: Alter0")).not.toBeInTheDocument();
  });
});

import { fireEvent, render, screen, within } from "@testing-library/react";
import { useState } from "react";
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
    runtimeMock.draft = "";
    runtimeMock.createSession.mockClear();
    runtimeMock.focusSession.mockClear();
    runtimeMock.removeSession.mockClear();
    runtimeMock.sendPrompt.mockClear();
  });

  it("keeps the compact header visible alongside terminal-style mobile actions for an empty chat workspace", () => {
    const toggleMobileNav = vi.fn();
    const toggleMobileSessionPane = vi.fn();
    renderWorkspace({ toggleMobileNav, toggleMobileSessionPane });

    expect(document.querySelector("[data-conversation-view='chat']")).toHaveClass("terminal-runtime-view");
    expect(Array.from(document.querySelector("[data-conversation-view='chat']")?.children || []).map((node) =>
      (node as HTMLElement).tagName.toLowerCase(),
    )).toEqual(["aside", "section"]);
    expect(screen.getByTestId("conversation-session-pane")).toHaveClass(
      "terminal-session-pane",
      "conversation-session-pane",
    );
    expect(within(screen.getByTestId("conversation-session-pane")).getByRole("list")).toHaveAttribute(
      "data-conversation-session-list",
      "true",
    );
    expect(within(screen.getByTestId("conversation-session-pane")).getAllByRole("listitem")).toHaveLength(1);
    expect(document.querySelector(".conversation-session-pane-shell")).toHaveClass(
      "terminal-session-pane-shell",
      "conversation-session-pane-shell",
    );
    expect(document.querySelector("[data-conversation-workspace]")).toHaveClass(
      "terminal-workspace",
      "conversation-workspace",
    );
    expect(document.querySelector(".conversation-workspace-body")).toHaveClass(
      "terminal-workspace-body",
      "conversation-workspace-body",
    );
    expect(document.querySelector("[data-conversation-chat-screen]")?.closest(".conversation-console-panel"))
      .toBe(document.querySelector(".conversation-workspace-body > .conversation-console-panel"));

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
    expect(screen.getByRole("heading", { name: "New Chat" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Model" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Tools" })).toBeInTheDocument();
    expect(screen.queryByText("DeepSeek V3.2 · 0 / 0")).not.toBeInTheDocument();
    expect(screen.getByTestId("conversation-session-pane")).toHaveAttribute("data-mobile-open", "false");

    fireEvent.click(within(mobileHeader).getByRole("button", { name: "Menu" }));
    expect(toggleMobileNav).toHaveBeenCalledTimes(1);

    fireEvent.click(within(mobileHeader).getByRole("button", { name: "Sessions" }));
    expect(screen.getByTestId("conversation-session-pane")).toHaveAttribute("data-mobile-open", "true");
    expect(toggleMobileSessionPane).toHaveBeenCalledTimes(1);

    fireEvent.click(within(mobileHeader).getByRole("button", { name: "New" }));
    expect(runtimeMock.createSession).toHaveBeenCalledTimes(1);
  });

  it("focuses the mobile composer on first touch so keyboard handling matches terminal", () => {
    renderWorkspace();

    const composerInput = screen.getByLabelText("Type a message to continue this workspace...") as HTMLTextAreaElement;
    const focusSpy = vi.spyOn(composerInput, "focus");

    fireEvent.pointerDown(composerInput, { pointerType: "touch" });

    expect(focusSpy).toHaveBeenCalled();
  });

  it("closes the mobile session pane after selecting a session", () => {
    renderWorkspace();

    fireEvent.click(screen.getByRole("button", { name: "Sessions" }));
    expect(screen.getByTestId("conversation-session-pane")).toHaveAttribute("data-mobile-open", "true");

    fireEvent.click(within(screen.getByTestId("conversation-session-pane")).getByRole("button", { name: /New Chat/ }));
    expect(runtimeMock.focusSession).toHaveBeenCalledWith("session-1");
    expect(screen.getByTestId("conversation-session-pane")).toHaveAttribute("data-mobile-open", "false");
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
        active: true,
      },
    ];

    renderWorkspace();

    expect(document.querySelector("[data-conversation-chat-screen]")).toHaveClass(
      "terminal-chat-screen",
      "conversation-chat-screen",
    );
    expect(document.querySelector(".conversation-workspace-head")).toHaveClass(
      "terminal-workspace-head",
      "conversation-workspace-head",
      "is-compact",
    );
    expect(document.querySelector(".conversation-chat-form")).toHaveClass("terminal-chat-form");
    expect(document.querySelector(".conversation-chat-submit")).toHaveClass(
      "terminal-chat-submit",
      "conversation-chat-submit",
    );
    expect(document.querySelector(".conversation-chat-form .terminal-composer-tools")).toBeInTheDocument();
    expect(document.querySelector(".conversation-chat-form .terminal-composer-meta")).toBeInTheDocument();
    expect(document.querySelector(".conversation-chat-submit .terminal-chat-form-button-icon svg")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Fix runtime shell" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Model" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Tools" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Tools / MCP" })).not.toBeInTheDocument();
    expect(screen.queryByText("Chat")).not.toBeInTheDocument();
    expect(screen.queryByText("DeepSeek V3.2 · 0 / 0")).not.toBeInTheDocument();
  });

  it("keeps the agent-runtime compact header visible on mobile empty state", () => {
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
    expect(screen.getByRole("heading", { name: "New Agent Session" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Model" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Agent" })).toBeInTheDocument();
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
    expect(document.querySelector(".conversation-composer-shell")).toHaveClass("terminal-composer-shell");
    expect(document.querySelector(".conversation-console-panel")).toHaveClass("is-empty");
    expect(document.querySelector("[data-conversation-chat-screen]")).toHaveClass(
      "terminal-chat-screen",
      "conversation-chat-screen",
      "is-empty",
    );
    expect(screen.getByRole("heading", { name: "New Chat" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Model" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Tools / MCP" })).toBeInTheDocument();
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

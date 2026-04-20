import { fireEvent, render, screen } from "@testing-library/react";
import { ConversationWorkspace } from "./ConversationWorkspace";
import { WorkbenchContext, type WorkbenchContextValue } from "../../app/WorkbenchContext";

const runtimeMock = {
  route: "chat" as const,
  compact: true,
  inspectorOpen: false,
  inspectorTab: "model" as const,
  sessions: [],
  activeSession: null,
  sessionItems: [
    {
      id: "session-1",
      title: "Session 1",
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
  selectedModelLabel: "Default",
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
    runtimeMock.createSession.mockClear();
    runtimeMock.focusSession.mockClear();
    runtimeMock.removeSession.mockClear();
  });

  it("renders mobile workspace actions and toggles the session pane", () => {
    const toggleMobileNav = vi.fn();
    renderWorkspace({ toggleMobileNav });

    expect(screen.getByRole("button", { name: "Menu" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Sessions" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "New Chat" })).toBeInTheDocument();
    expect(screen.getByTestId("conversation-session-pane")).toHaveAttribute("data-mobile-open", "false");

    fireEvent.click(screen.getByRole("button", { name: "Menu" }));
    expect(toggleMobileNav).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByRole("button", { name: "Sessions" }));
    expect(screen.getByTestId("conversation-session-pane")).toHaveAttribute("data-mobile-open", "true");

    fireEvent.click(screen.getByRole("button", { name: "New Chat" }));
    expect(runtimeMock.createSession).toHaveBeenCalledTimes(1);
  });

  it("closes the mobile session pane after selecting a session", () => {
    renderWorkspace();

    fireEvent.click(screen.getByRole("button", { name: "Sessions" }));
    expect(screen.getByTestId("conversation-session-pane")).toHaveAttribute("data-mobile-open", "true");

    fireEvent.click(screen.getByRole("button", { name: /Session 1/ }));
    expect(runtimeMock.focusSession).toHaveBeenCalledWith("session-1");
    expect(screen.getByTestId("conversation-session-pane")).toHaveAttribute("data-mobile-open", "false");
  });
});

import { fireEvent, render, screen, waitFor } from "@testing-library/react";

const mockIsLegacyShellMobileViewport = vi.fn(() => false);
const mockViewportSyncDestroy = vi.fn();
const mockCreateMobileViewportSyncController = vi.fn(() => ({
  sync: vi.fn(),
  destroy: mockViewportSyncDestroy,
}));

vi.mock("../features/shell/legacyShellState", () => ({
  isLegacyShellMobileViewport: () => mockIsLegacyShellMobileViewport(),
}));

vi.mock("../shared/viewport/mobileViewportSync", () => ({
  createMobileViewportSyncController: () => mockCreateMobileViewportSyncController(),
}));

vi.mock("../features/conversation-runtime/ConversationRuntimeProvider", () => ({
  ConversationRuntimeProvider: ({
    route,
    language,
    children,
  }: {
    route: string;
    language: string;
    children: React.ReactNode;
  }) => (
    <div data-testid="conversation-provider" data-route={route} data-language={language}>
      {children}
    </div>
  ),
}));

vi.mock("../features/conversation-runtime/ConversationWorkspace", () => ({
  ConversationWorkspace: ({ language }: { language: string }) => (
    <div data-testid="conversation-workspace" data-language={language}>
      conversation workspace
    </div>
  ),
}));

vi.mock("../features/shell/components/ReactManagedRouteBody", () => ({
  ReactManagedRouteBody: ({ route, language }: { route: string; language: string }) => (
    <div data-testid="route-body" data-route={route} data-language={language}>
      {route}:{language}
    </div>
  ),
}));

vi.mock("../features/shell/components/PrimaryNav", () => ({
  PrimaryNav: ({
    currentRoute,
    language,
    onNavigate,
    onToggleLanguage,
    onToggleNavCollapsed,
  }: {
    currentRoute: string;
    language: string;
    onNavigate: (route: string) => void;
    onToggleLanguage: () => void;
    onToggleNavCollapsed: () => void;
  }) => (
    <div data-testid="primary-nav" data-route={currentRoute} data-language={language}>
      <button type="button" onClick={() => onNavigate("chat")}>
        go chat
      </button>
      <button type="button" onClick={() => onNavigate("tasks")}>
        go tasks
      </button>
      <button type="button" onClick={() => onToggleLanguage()}>
        toggle language
      </button>
      <button type="button" onClick={() => onToggleNavCollapsed()}>
        toggle nav
      </button>
    </div>
  ),
}));

import { WorkbenchApp } from "./WorkbenchApp";

describe("WorkbenchApp", () => {
  beforeEach(() => {
    window.location.hash = "#chat";
    document.documentElement.lang = "en";
    mockIsLegacyShellMobileViewport.mockReturnValue(false);
    mockCreateMobileViewportSyncController.mockClear();
    mockViewportSyncDestroy.mockClear();
  });

  afterEach(() => {
    vi.clearAllMocks();
    window.location.hash = "";
    document.documentElement.lang = "en";
  });

  it("renders conversation routes through the conversation runtime and syncs language changes", async () => {
    const { container } = render(<WorkbenchApp />);

    expect(screen.getByTestId("conversation-provider")).toHaveAttribute("data-route", "chat");
    expect(screen.getByTestId("conversation-workspace")).toHaveAttribute("data-language", "en");
    expect(screen.queryByTestId("route-body")).not.toBeInTheDocument();
    expect(container.querySelector(".app-shell")).toHaveAttribute("data-workbench-route", "chat");
    expect(container.querySelector(".chat-pane")).toHaveAttribute("data-route", "chat");

    fireEvent.click(screen.getByRole("button", { name: "toggle language" }));

    await waitFor(() => {
      expect(document.documentElement.lang).toBe("zh-CN");
    });
    expect(screen.getByTestId("primary-nav")).toHaveAttribute("data-language", "zh");

    fireEvent.click(screen.getByRole("button", { name: "go tasks" }));

    await waitFor(() => {
      expect(screen.getByTestId("route-body")).toHaveAttribute("data-route", "tasks");
    });
    expect(screen.getByTestId("route-body")).toHaveAttribute("data-language", "zh");
    expect(container.querySelector(".app-shell")).toHaveAttribute("data-workbench-route", "tasks");
    expect(container.querySelector(".chat-pane")).toHaveAttribute("data-route", "tasks");
    expect(screen.queryByTestId("conversation-provider")).not.toBeInTheDocument();
  });

  it("uses an overlay on mobile nav and closes it after route navigation", async () => {
    mockIsLegacyShellMobileViewport.mockReturnValue(true);
    const { container } = render(<WorkbenchApp />);
    const shell = container.querySelector(".app-shell");
    expect(shell).not.toHaveClass("nav-open");

    fireEvent.click(screen.getByRole("button", { name: "toggle nav" }));
    expect(shell).toHaveClass("nav-open");
    expect(shell).toHaveClass("overlay-open");
    expect(shell).not.toHaveClass("nav-collapsed");

    fireEvent.click(screen.getByRole("button", { name: "go tasks" }));

    await waitFor(() => {
      expect(screen.getByTestId("route-body")).toHaveAttribute("data-route", "tasks");
    });
    expect(shell).not.toHaveClass("nav-open");
    expect(shell).not.toHaveClass("overlay-open");
  });

  it("renders the terminal route without a duplicate page hero", async () => {
    window.location.hash = "#terminal";
    const { container } = render(<WorkbenchApp />);

    await waitFor(() => {
      expect(screen.getByTestId("route-body")).toHaveAttribute("data-route", "terminal");
    });

    expect(container.querySelector(".route-view.terminal-route")).toBeInTheDocument();
    expect(container.querySelector(".route-view.terminal-route > .route-head")).not.toBeInTheDocument();
    expect(container.querySelector(".route-body.terminal-route-body")).toBeInTheDocument();
  });

  it("installs and cleans up the mobile viewport sync controller at the app root", () => {
    const { unmount } = render(<WorkbenchApp />);

    expect(mockCreateMobileViewportSyncController).toHaveBeenCalledTimes(1);

    unmount();

    expect(mockViewportSyncDestroy).toHaveBeenCalledTimes(1);
  });
});

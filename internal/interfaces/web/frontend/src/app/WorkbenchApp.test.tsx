import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";

const mockIsLegacyShellMobileViewport = vi.fn(() => false);
const mockViewportSyncDestroy = vi.fn();
const mockCreateMobileViewportSyncController = vi.fn(() => ({
  sync: vi.fn(),
  destroy: mockViewportSyncDestroy,
}));
let mockRuntimeRouteHostRegistersRail = true;

vi.mock("../features/shell/legacyShellState", () => ({
  isLegacyShellMobileViewport: () => mockIsLegacyShellMobileViewport(),
}));

vi.mock("../shared/viewport/mobileViewportSync", () => ({
  createMobileViewportSyncController: () => mockCreateMobileViewportSyncController(),
}));

vi.mock("../features/shell/components/ReactManagedRouteBody", () => ({
  ReactManagedRouteBody: ({ route, language }: { route: string; language: string }) => (
    <div data-testid="route-body" data-route={route} data-language={language}>
      {route}:{language}
    </div>
  ),
}));

vi.mock("../features/shell/components/RuntimeRouteHost", async () => {
  const React = await vi.importActual<typeof import("react")>("react");
  const { useWorkbenchContext } = await vi.importActual<typeof import("./WorkbenchContext")>("./WorkbenchContext");

  return {
    RuntimeRouteHost: ({ route, language }: { route: string; language: string }) => {
      const {
        setRuntimeSessionRail,
        toggleMobileSessionPane,
      } = useWorkbenchContext();

      React.useEffect(() => {
        if (!mockRuntimeRouteHostRegistersRail) {
          return;
        }
        setRuntimeSessionRail?.({
          route,
          countLabel: "1 sessions",
          onPrimaryAction: vi.fn(),
          body: <div data-testid="mock-session-rail-body">session rail body:{route}</div>,
        });
        return () => setRuntimeSessionRail?.(null);
      }, [route, setRuntimeSessionRail]);

      return (
        <div data-testid="runtime-route-host" data-route={route} data-language={language}>
          runtime:{route}:{language}
          <button type="button" onClick={toggleMobileSessionPane}>
            open sessions
          </button>
        </div>
      );
    },
  };
});

vi.mock("../features/shell/components/PrimaryNav", () => ({
  PrimaryNav: ({
    currentRoute,
    language,
    onNavigate,
    onToggleLanguage,
    onToggleNavCollapsed,
    sessionRail,
  }: {
    currentRoute: string;
    language: string;
    sessionRail?: { route: string } | null;
    onNavigate: (route: string) => void;
    onToggleLanguage: () => void;
    onToggleNavCollapsed: () => void;
  }) => (
    <div
      data-testid="primary-nav"
      data-route={currentRoute}
      data-language={language}
      data-session-rail-route={sessionRail?.route || ""}
    >
      <div data-testid="primary-nav-session-rail-body">
        {sessionRail?.body || null}
      </div>
      <button type="button" onClick={() => onNavigate("chat")}>
        go chat
      </button>
      <button type="button" onClick={() => onNavigate("tasks")}>
        go tasks
      </button>
      <button type="button" onClick={() => onNavigate("settings")}>
        go settings
      </button>
      <button type="button" onClick={() => onNavigate("terminal")}>
        go terminal
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
    window.history.replaceState({}, "", "/chat");
    document.documentElement.lang = "en";
    mockIsLegacyShellMobileViewport.mockReturnValue(false);
    mockRuntimeRouteHostRegistersRail = true;
    mockCreateMobileViewportSyncController.mockClear();
    mockViewportSyncDestroy.mockClear();
  });

  afterEach(() => {
    vi.clearAllMocks();
    window.history.replaceState({}, "", "/");
    document.documentElement.lang = "en";
  });

  it("renders conversation routes through the conversation runtime and syncs language changes", async () => {
    const { container } = render(<WorkbenchApp />);

    expect(screen.getByTestId("runtime-route-host")).toHaveAttribute("data-route", "chat");
    expect(screen.getByTestId("runtime-route-host")).toHaveAttribute("data-language", "en");
    expect(screen.queryByTestId("route-body")).not.toBeInTheDocument();
    expect(container.querySelector(".app-shell")).toHaveAttribute("data-workbench-route", "chat");
    expect(container.querySelector(".chat-pane")).toHaveAttribute("data-route", "chat");

    fireEvent.click(screen.getByRole("button", { name: "toggle language" }));

    await waitFor(() => {
      expect(document.documentElement.lang).toBe("zh-CN");
    });
    expect(screen.getByTestId("primary-nav")).toHaveAttribute("data-language", "zh");

    fireEvent.click(screen.getByRole("button", { name: "go settings" }));

    await waitFor(() => {
      expect(screen.getByTestId("route-body")).toHaveAttribute("data-route", "settings");
    });
    expect(screen.getByTestId("route-body")).toHaveAttribute("data-language", "zh");
    expect(container.querySelector(".app-shell")).toHaveAttribute("data-workbench-route", "settings");
    expect(container.querySelector(".chat-pane")).toHaveAttribute("data-route", "settings");
    expect(screen.queryByTestId("runtime-route-host")).not.toBeInTheDocument();
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

    fireEvent.click(screen.getByRole("button", { name: "go settings" }));

    await waitFor(() => {
      expect(screen.getByTestId("route-body")).toHaveAttribute("data-route", "settings");
    });
    expect(shell).not.toHaveClass("nav-open");
    expect(shell).not.toHaveClass("overlay-open");
  });

  it("opens the mobile primary nav when runtime sessions are owned by the left rail", async () => {
    mockIsLegacyShellMobileViewport.mockReturnValue(true);
    const { container } = render(<WorkbenchApp />);
    const shell = container.querySelector(".app-shell");

    await waitFor(() => {
      expect(screen.getByTestId("primary-nav")).toHaveAttribute("data-session-rail-route", "chat");
    });
    expect(shell).not.toHaveClass("nav-open");

    fireEvent.click(screen.getByRole("button", { name: "open sessions" }));

    expect(shell).toHaveClass("nav-open");
    expect(shell).toHaveClass("overlay-open");
    expect(screen.getByTestId("primary-nav")).toHaveAttribute("data-session-rail-route", "chat");
  });

  it("keeps a stable session rail shell before the runtime route registers its list", () => {
    mockRuntimeRouteHostRegistersRail = false;
    window.history.replaceState({}, "", "/terminal");

    render(<WorkbenchApp />);

    expect(screen.getByTestId("primary-nav")).toHaveAttribute("data-route", "terminal");
    expect(screen.getByTestId("primary-nav")).toHaveAttribute("data-session-rail-route", "terminal");
  });

  it("uses one active runtime session rail when switching between chat and terminal", async () => {
    render(<WorkbenchApp />);

    await waitFor(() => {
      expect(screen.getByTestId("primary-nav")).toHaveAttribute("data-session-rail-route", "chat");
    });
    expect(screen.getByTestId("primary-nav-session-rail-body")).toHaveTextContent("session rail body:chat");

    fireEvent.click(screen.getByRole("button", { name: "go terminal" }));

    await waitFor(() => {
      expect(screen.getByTestId("primary-nav")).toHaveAttribute("data-session-rail-route", "terminal");
    });
    expect(screen.getByTestId("primary-nav-session-rail-body")).toHaveTextContent("session rail body:terminal");

    fireEvent.click(screen.getByRole("button", { name: "go chat" }));

    await waitFor(() => {
      expect(screen.getByTestId("primary-nav")).toHaveAttribute("data-session-rail-route", "chat");
    });
    expect(screen.getByTestId("primary-nav-session-rail-body")).toHaveTextContent("session rail body:chat");
    expect(screen.getByTestId("primary-nav-session-rail-body")).not.toHaveTextContent("session rail body:terminal");
  });

  it("renders settings inside the unified settings page frame", async () => {
    window.history.replaceState({}, "", "/settings");
    mockIsLegacyShellMobileViewport.mockReturnValue(true);
    const { container } = render(<WorkbenchApp />);
    const shell = container.querySelector(".app-shell");

    await waitFor(() => {
      expect(screen.getByTestId("route-body")).toHaveAttribute("data-route", "settings");
    });

    const mobileHeader = container.querySelector("[data-route-mobile-head]") as HTMLElement;
    const routeHead = container.querySelector(".route-head") as HTMLElement;
    expect(mobileHeader).toBeInTheDocument();
    expect(container.querySelector(".route-view")).toHaveAttribute("data-route-family", "settings");
    expect(container.querySelector(".route-view")).toHaveClass("workbench-route-frame");
    expect(routeHead).toHaveClass("workbench-title-head", "is-compact");
    expect(routeHead).toHaveAttribute("data-workbench-title-head", "route");
    expect(routeHead.querySelector(".workbench-title-leading")).toBeInTheDocument();
    expect(routeHead.querySelector(".route-title-marker")).toBeInTheDocument();
    expect(container.querySelector(".route-head h3")?.textContent).toBe("Settings");
    expect(mobileHeader.querySelector(".route-mobile-title h3")?.textContent).toBe("Settings");
    expect(mobileHeader.querySelector(".route-mobile-head-spacer")).toBeInTheDocument();
    expect(within(mobileHeader).getByRole("button", { name: "Menu" }).querySelector("[data-route-mobile-icon='menu']")).toBeInTheDocument();
    expect(within(mobileHeader).getByRole("button", { name: "Menu" }).querySelector(".route-mobile-action-label")).toHaveClass("sr-only");

    fireEvent.click(within(mobileHeader).getByRole("button", { name: "Menu" }));

    expect(shell).toHaveClass("nav-open");
    expect(shell).toHaveClass("overlay-open");
  });

  it("renders the terminal route as a direct runtime workspace frame without route-page wrappers", async () => {
    const { container } = render(<WorkbenchApp />);

    fireEvent.click(screen.getByRole("button", { name: "go terminal" }));

    await waitFor(() => {
      expect(screen.getByTestId("runtime-route-host")).toHaveAttribute("data-route", "terminal");
    });

    const paneShell = container.querySelector("[data-workbench-pane-shell]") as HTMLElement;
    expect(paneShell).toBeInTheDocument();
    expect(paneShell.firstElementChild).toBe(screen.getByTestId("runtime-route-host"));
    expect(screen.getByTestId("runtime-route-host")).toHaveAttribute("data-route", "terminal");
    expect(container.querySelector(".route-view.terminal-route")).not.toBeInTheDocument();
    expect(container.querySelector(".route-body.terminal-route-body")).not.toBeInTheDocument();
  });

  it("installs and cleans up the mobile viewport sync controller at the app root", () => {
    const { unmount } = render(<WorkbenchApp />);

    expect(mockCreateMobileViewportSyncController).toHaveBeenCalledTimes(1);

    unmount();

    expect(mockViewportSyncDestroy).toHaveBeenCalledTimes(1);
  });
});

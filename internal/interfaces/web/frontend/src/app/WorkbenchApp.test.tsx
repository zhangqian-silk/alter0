import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { act } from "react";

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
  ReactManagedRouteBody: ({
    route,
    language,
    onToggleLanguage,
  }: {
    route: string;
    language: string;
    onToggleLanguage: () => void;
  }) => (
    <div data-testid="route-body" data-route={route} data-language={language}>
      {route}:{language}
      <button type="button" onClick={onToggleLanguage}>
        Language
      </button>
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
    onDismiss,
    onNavigate,
    sessionRail,
  }: {
    currentRoute: string;
    language: string;
    onDismiss?: () => void;
    sessionRail?: { route: string } | null;
    onNavigate: (route: string) => void;
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
      <button type="button" onClick={() => onNavigate("chatRuntime")}>
        go removed chatRuntime
      </button>
      {onDismiss ? (
        <button type="button" onClick={() => onDismiss()}>
          close nav
        </button>
      ) : null}
    </div>
  ),
}));

import { WorkbenchApp } from "./WorkbenchApp";
import { resetConversationRuntimeCache } from "../features/conversation-runtime/ConversationRuntimeProvider";

function jsonResponse(payload: unknown) {
  return new Response(JSON.stringify(payload), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

function stubChatRuntimeFetch(items: unknown[] = []) {
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const path = input.toString();
    if (path === "/api/chat/sessions") {
      return jsonResponse({ items });
    }
    return jsonResponse({ items: [] });
  });
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

describe("WorkbenchApp", () => {
  beforeEach(() => {
    window.history.replaceState({}, "", "/chat");
    document.documentElement.lang = "en";
    mockIsLegacyShellMobileViewport.mockReturnValue(false);
    mockRuntimeRouteHostRegistersRail = true;
    mockCreateMobileViewportSyncController.mockClear();
    mockViewportSyncDestroy.mockClear();
    resetConversationRuntimeCache();
    window.localStorage.clear();
    window.sessionStorage.clear();
  });

  afterEach(() => {
    vi.clearAllMocks();
    vi.unstubAllGlobals();
    resetConversationRuntimeCache();
    window.localStorage.clear();
    window.sessionStorage.clear();
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

    fireEvent.click(screen.getByRole("button", { name: "go settings" }));
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Language" })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Language" }));

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

  it("loads the real Chat session rail when Settings is opened directly", async () => {
    window.history.replaceState({}, "", "/settings");
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const path = input.toString();
      if (path === "/api/chat/sessions") {
        return jsonResponse({
          items: [
            {
              id: "c_51jttwiv4yggqagk",
              title: "Real settings session",
              status: "ready",
              created_at: "2026-07-08T10:00:00Z",
              updated_at: "2026-07-08T10:05:00Z",
              activity_at: "2026-07-08T10:05:00Z",
              turns: [],
            },
          ],
        });
      }
      return jsonResponse({ items: [] });
    });
    vi.stubGlobal("fetch", fetchMock);

    render(<WorkbenchApp />);

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "/api/chat/sessions",
        expect.objectContaining({ method: "GET" }),
      );
    });
    await waitFor(() => {
      expect(screen.getByTestId("primary-nav-session-rail-body")).toHaveTextContent("Real settings session");
    });
    expect(screen.getByTestId("primary-nav-session-rail-body")).not.toHaveTextContent(/^New$/);
  });

  it("uses an overlay on mobile nav and closes it after route navigation", async () => {
    mockIsLegacyShellMobileViewport.mockReturnValue(true);
    const { container } = render(<WorkbenchApp />);
    const shell = container.querySelector(".app-shell");
    expect(shell).not.toHaveClass("nav-open");
    expect(screen.queryByRole("button", { name: "Close panels" })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "open sessions" }));
    expect(shell).toHaveClass("nav-open");
    expect(shell).toHaveClass("overlay-open");
    expect(shell).not.toHaveClass("nav-collapsed");
    expect(screen.getByRole("button", { name: "Close panels" })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "go settings" }));

    await waitFor(() => {
      expect(screen.getByTestId("route-body")).toHaveAttribute("data-route", "settings");
    });
    expect(shell).not.toHaveClass("nav-open");
    expect(shell).not.toHaveClass("overlay-open");
    expect(screen.queryByRole("button", { name: "Close panels" })).not.toBeInTheDocument();
  });

  it("uses the drawer dismiss action only in the shared mobile shell", async () => {
    mockIsLegacyShellMobileViewport.mockReturnValue(false);
    const { container } = render(<WorkbenchApp />);
    const shell = container.querySelector(".app-shell");

    expect(screen.queryByRole("button", { name: "close nav" })).not.toBeInTheDocument();
    expect(shell).not.toHaveClass("nav-collapsed");

    mockIsLegacyShellMobileViewport.mockReturnValue(true);
    act(() => {
      window.dispatchEvent(new Event("resize"));
    });

    await waitFor(() => {
      expect(shell).not.toHaveClass("nav-collapsed");
    });
    expect(screen.getByRole("button", { name: "open sessions" })).toBeInTheDocument();
  });

  it("falls back to Chat without confirming when navigating to the removed ChatRuntime route", async () => {
    render(<WorkbenchApp />);
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    document.body.setAttribute("data-composer-unsaved-state", "dirty");

    try {
      fireEvent.click(screen.getByRole("button", { name: "go removed chatRuntime" }));

      await waitFor(() => {
        expect(screen.getByTestId("runtime-route-host")).toHaveAttribute("data-route", "chat");
      });
      expect(confirmSpy).not.toHaveBeenCalled();
      expect(document.body).not.toHaveAttribute("data-composer-unsaved-confirm");
    } finally {
      document.body.removeAttribute("data-composer-unsaved-state");
      document.body.removeAttribute("data-composer-unsaved-confirm");
      confirmSpy.mockRestore();
    }
  });

  it("blurs the active input before opening a mobile drawer", async () => {
    mockIsLegacyShellMobileViewport.mockReturnValue(true);
    render(<WorkbenchApp />);
    const input = document.createElement("textarea");
    document.body.appendChild(input);
    input.focus();
    expect(document.activeElement).toBe(input);

    try {
      fireEvent.click(screen.getByRole("button", { name: "open sessions" }));
      expect(document.activeElement).not.toBe(input);

      fireEvent.click(screen.getByRole("button", { name: "Close panels" }));
      expect(screen.queryByRole("button", { name: "Close panels" })).not.toBeInTheDocument();
      input.focus();
      expect(document.activeElement).toBe(input);
      fireEvent.click(screen.getByRole("button", { name: "open sessions" }));
      expect(document.activeElement).not.toBe(input);
    } finally {
      input.remove();
    }
  });

  it("opens the mobile primary nav when runtime sessions are owned by the left rail", async () => {
    mockIsLegacyShellMobileViewport.mockReturnValue(true);
    const { container } = render(<WorkbenchApp />);
    const shell = container.querySelector(".app-shell");

    await waitFor(() => {
      expect(screen.getByTestId("runtime-route-host")).toHaveAttribute("data-route", "chat");
    });
    expect(screen.queryByTestId("primary-nav")).not.toBeInTheDocument();
    expect(shell).not.toHaveClass("nav-open");

    fireEvent.click(screen.getByRole("button", { name: "open sessions" }));

    expect(shell).toHaveClass("nav-open");
    expect(shell).toHaveClass("overlay-open");
    expect(screen.getByTestId("primary-nav")).toHaveAttribute("data-session-rail-route", "chat");
  });

  it("keeps a stable Chat session rail shell before the runtime route registers its list", () => {
    mockRuntimeRouteHostRegistersRail = false;

    render(<WorkbenchApp />);

    expect(screen.getByTestId("primary-nav")).toHaveAttribute("data-route", "chat");
    expect(screen.getByTestId("primary-nav")).toHaveAttribute("data-session-rail-route", "chat");
  });

  it("keeps the Chat session rail visible when opening Settings from the sidebar shortcut", async () => {
    const fetchMock = stubChatRuntimeFetch([]);
    render(<WorkbenchApp />);

    await waitFor(() => {
      expect(screen.getByTestId("primary-nav")).toHaveAttribute("data-session-rail-route", "chat");
    });
    expect(screen.getByTestId("primary-nav-session-rail-body")).toHaveTextContent("session rail body:chat");

    fireEvent.click(screen.getByRole("button", { name: "go settings" }));

    await waitFor(() => {
      expect(screen.getByTestId("route-body")).toHaveAttribute("data-route", "settings");
    });
    expect(screen.getByTestId("primary-nav")).toHaveAttribute("data-route", "settings");
    expect(screen.getByTestId("primary-nav")).toHaveAttribute("data-session-rail-route", "chat");
    expect(fetchMock).not.toHaveBeenCalled();
    expect(screen.getByTestId("primary-nav-session-rail-body")).toHaveTextContent("session rail body:chat");
    expect(screen.queryByTestId("runtime-route-host")).not.toBeInTheDocument();
  });

  it("does not show a selected fallback conversation while Settings owns the right pane", async () => {
    mockRuntimeRouteHostRegistersRail = false;
    stubChatRuntimeFetch([]);

    render(<WorkbenchApp />);

    fireEvent.click(screen.getByRole("button", { name: "go settings" }));

    await waitFor(() => {
      expect(screen.getByTestId("route-body")).toHaveAttribute("data-route", "settings");
    });
    expect(screen.getByTestId("primary-nav")).toHaveAttribute("data-session-rail-route", "chat");
    expect(screen.getByTestId("primary-nav-session-rail-body").querySelector("[data-runtime-session-list-placeholder]")).not.toBeInTheDocument();
    expect(screen.getByTestId("primary-nav-session-rail-body").querySelector(".runtime-session-card.is-active")).not.toBeInTheDocument();
    expect(screen.getByTestId("primary-nav-session-rail-body").querySelector(".runtime-session-select.active")).not.toBeInTheDocument();
  });

  it("keeps the Chat runtime session rail when navigating to the removed ChatRuntime route", async () => {
    render(<WorkbenchApp />);

    await waitFor(() => {
      expect(screen.getByTestId("primary-nav")).toHaveAttribute("data-session-rail-route", "chat");
    });
    expect(screen.getByTestId("primary-nav-session-rail-body")).toHaveTextContent("session rail body:chat");

    fireEvent.click(screen.getByRole("button", { name: "go removed chatRuntime" }));

    await waitFor(() => {
      expect(screen.getByTestId("primary-nav")).toHaveAttribute("data-session-rail-route", "chat");
    });
    expect(screen.getByTestId("primary-nav-session-rail-body")).toHaveTextContent("session rail body:chat");

    fireEvent.click(screen.getByRole("button", { name: "go chat" }));

    await waitFor(() => {
      expect(screen.getByTestId("primary-nav")).toHaveAttribute("data-session-rail-route", "chat");
    });
    expect(screen.getByTestId("primary-nav-session-rail-body")).toHaveTextContent("session rail body:chat");
    expect(screen.getByTestId("primary-nav-session-rail-body")).not.toHaveTextContent("session rail body:chatRuntime");
  });

  it("renders settings inside the same runtime workspace shell as Chat", async () => {
    window.history.replaceState({}, "", "/settings");
    mockIsLegacyShellMobileViewport.mockReturnValue(true);
    const { container } = render(<WorkbenchApp />);
    const shell = container.querySelector(".app-shell");

    await waitFor(() => {
      expect(container.querySelector("[data-runtime-workspace-page='settings']")).toBeInTheDocument();
    });

    const mobileHeader = container.querySelector("[data-runtime-mobile-header]") as HTMLElement;
    const routeHead = container.querySelector("[data-runtime-workspace-page='settings'] .runtime-workspace-head") as HTMLElement;
    expect(mobileHeader).toBeInTheDocument();
    expect(container.querySelector("[data-runtime-workspace-page='settings']")).toHaveClass("runtime-workspace-shell", "runtime-workspace-view");
    expect(container.querySelector("[data-runtime-workspace-page='settings']")).toHaveAttribute("data-runtime-view", "settings");
    expect(container.querySelector("[data-runtime-workspace-page='settings'] .runtime-workspace-body")).toHaveAttribute("data-runtime-view", "settings");
    expect(container.querySelector(".route-view")).not.toBeInTheDocument();
    expect(container.querySelector(".route-body")).not.toBeInTheDocument();
    // On mobile, desktop header is not rendered — mobile header provides nav toggle
    expect(routeHead).not.toBeInTheDocument();
    expect(container.querySelector(".route-head")).not.toBeInTheDocument();
    expect(mobileHeader.querySelector(".runtime-workspace-mobile-title-text")?.textContent).toBe("Settings");
    expect(mobileHeader.querySelector(".route-title-marker")).not.toBeInTheDocument();
    expect(mobileHeader.querySelector("[data-runtime-header-signal]")).not.toBeInTheDocument();
    expect(within(mobileHeader).getByRole("button", { name: "Menu" }).querySelector("[data-runtime-mobile-icon='menu']")).toBeInTheDocument();
    expect(within(mobileHeader).getByRole("button", { name: "Menu" }).querySelector(".runtime-workspace-mobile-action-label")).toHaveClass("sr-only");

    fireEvent.click(within(mobileHeader).getByRole("button", { name: "Menu" }));

    expect(shell).toHaveClass("nav-open");
    expect(shell).toHaveClass("overlay-open");
  });

  it("does not render a ChatRuntime runtime workspace for the removed ChatRuntime route", async () => {
    const { container } = render(<WorkbenchApp />);

    fireEvent.click(screen.getByRole("button", { name: "go removed chatRuntime" }));

    await waitFor(() => {
      expect(screen.getByTestId("runtime-route-host")).toHaveAttribute("data-route", "chat");
    });

    const paneShell = container.querySelector("[data-workbench-pane-shell]") as HTMLElement;
    expect(paneShell).toBeInTheDocument();
    expect(paneShell.firstElementChild).toBe(screen.getByTestId("runtime-route-host"));
    expect(screen.getByTestId("runtime-route-host")).toHaveAttribute("data-route", "chat");
    expect(container.querySelector(".route-view.chatRuntime-route")).not.toBeInTheDocument();
    expect(container.querySelector(".route-body.chatRuntime-route-body")).not.toBeInTheDocument();
  });

  it("installs and cleans up the mobile viewport sync controller at the app root", () => {
    const { unmount } = render(<WorkbenchApp />);

    expect(mockCreateMobileViewportSyncController).toHaveBeenCalledTimes(1);

    unmount();

    expect(mockViewportSyncDestroy).toHaveBeenCalledTimes(1);
  });
});

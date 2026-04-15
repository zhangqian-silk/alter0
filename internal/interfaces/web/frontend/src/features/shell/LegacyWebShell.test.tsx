import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import {
  LEGACY_SHELL_CREATE_SESSION_EVENT,
  LEGACY_SHELL_FOCUS_SESSION_EVENT,
  LEGACY_SHELL_NAVIGATE_EVENT,
  LEGACY_SHELL_QUICK_PROMPT_EVENT,
  LEGACY_SHELL_REMOVE_SESSION_EVENT,
  LEGACY_SHELL_SYNC_CHAT_WORKSPACE_EVENT,
  LEGACY_SHELL_SYNC_CHAT_RUNTIME_EVENT,
  LEGACY_SHELL_SYNC_MESSAGE_REGION_EVENT,
  LEGACY_SHELL_SYNC_NAV_COLLAPSED_EVENT,
  LEGACY_SHELL_SYNC_SESSION_PANE_EVENT,
  LEGACY_SHELL_SYNC_SESSION_HISTORY_EVENT,
  LEGACY_SHELL_TOGGLE_LANGUAGE_EVENT,
} from "./legacyShellBridge";
import { LegacyWebShell } from "./LegacyWebShell";
import { LEGACY_SHELL_IDS } from "./legacyDomContract";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: init.status ?? 200,
    headers: {
      "Content-Type": "application/json",
      ...(init.headers ?? {}),
    },
  });
}

describe("LegacyWebShell", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn((input: RequestInfo | URL) => {
        const url = String(input);
        if (url === "/api/control/channels" || url === "/api/control/skills" || url === "/api/control/mcps") {
          return Promise.resolve(jsonResponse({ items: [] }));
        }
        if (url.startsWith("/api/sessions?")) {
          return Promise.resolve(jsonResponse({ items: [] }));
        }
        return Promise.reject(new Error(`Unhandled fetch: ${url}`));
      }),
    );
    window.location.hash = "";
    document.documentElement.lang = "en";
    window.sessionStorage.clear();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders the legacy shell contract for the runtime bridge", () => {
    render(<LegacyWebShell />);

    expect(document.getElementById(LEGACY_SHELL_IDS.appShell)).toBeInTheDocument();
    expect(document.getElementById(LEGACY_SHELL_IDS.navCollapseButton)).toBeInTheDocument();
    expect(document.getElementById(LEGACY_SHELL_IDS.sessionList)).toBeInTheDocument();
    expect(document.getElementById(LEGACY_SHELL_IDS.messageArea)).toBeInTheDocument();
    expect(document.getElementById(LEGACY_SHELL_IDS.chatRuntimePanel)).toBeInTheDocument();
    expect(document.getElementById(LEGACY_SHELL_IDS.chatRuntimeSheetHost)).toBeInTheDocument();
    expect(document.getElementById(LEGACY_SHELL_IDS.mobileBackdrop)).toBeInTheDocument();
    expect(document.getElementById("welcomeTargetList")).toBeInTheDocument();
    expect(document.querySelector("[data-runtime-controls-root]")).toBeInTheDocument();
    expect(document.querySelector("[data-runtime-note-root]")).toBeInTheDocument();
    expect(document.querySelector("[data-runtime-sheet-root]")).toBeInTheDocument();
  });

  it("renders the workspace navigation and composer entrypoints", () => {
    render(<LegacyWebShell />);

    expect(screen.getByRole("navigation", { name: "Primary workspace navigation" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Chat" })).toHaveAttribute("data-route", "chat");
    expect(screen.getByRole("button", { name: "Agent" })).toHaveAttribute("data-route", "agent-runtime");
    expect(screen.getByRole("button", { name: "Terminal" })).toHaveAttribute("data-route", "terminal");
    expect(screen.getByRole("button", { name: "Send message" })).toHaveAttribute("data-composer-submit", "chat-main");
    expect(screen.getByLabelText("Input your message")).toHaveAttribute("data-composer-input", "chat-main");
  });

  it("renders the new shell information architecture for chat mode", () => {
    render(<LegacyWebShell />);

    expect(document.querySelector('[data-shell-section="brand-panel"]')).toBeInTheDocument();
    expect(document.querySelector('[data-shell-section="session-context"]')).toBeInTheDocument();
    expect(document.querySelector('[data-shell-section="chat-hero"]')).toBeInTheDocument();
    expect(document.querySelector('[data-shell-section="prompt-deck"]')).toBeInTheDocument();
    expect(document.querySelector('[data-shell-section="composer-panel"]')).toBeInTheDocument();
    expect(document.querySelector('[data-shell-section="route-hero"]')).not.toBeInTheDocument();
  });

  it("renders a dedicated route hero for page-mode routes", () => {
    window.location.hash = "#channels";

    render(<LegacyWebShell />);

    expect(document.querySelector('[data-shell-section="route-hero"]')).toBeInTheDocument();
    expect(document.querySelector('[data-shell-section="chat-hero"]')).not.toBeInTheDocument();
  });

  it("marks React-managed route bodies so the legacy runtime can skip DOM ownership", () => {
    window.location.hash = "#channels";

    render(<LegacyWebShell />);

    expect(document.getElementById(LEGACY_SHELL_IDS.routeBody)).toHaveAttribute(
      "data-react-managed-route",
      "true",
    );
  });

  it("keeps legacy-managed route bodies unmarked", () => {
    window.location.hash = "#terminal";

    render(<LegacyWebShell />);

    expect(document.getElementById(LEGACY_SHELL_IDS.routeBody)).toHaveAttribute(
      "data-react-managed-route",
      "false",
    );
  });

  it("renders the shell navigation groups and route placement", () => {
    const { container } = render(<LegacyWebShell />);

    expect(screen.getByRole("heading", { name: "Workspace" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Agent Studio" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Control" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Settings" })).toBeInTheDocument();

    const routeButtons = [...container.querySelectorAll<HTMLButtonElement>(".menu-item")];
    const routes = routeButtons.map((button) => button.dataset.route);

    expect(routes).toEqual([
      "chat",
      "agent-runtime",
      "terminal",
      "agent",
      "products",
      "memory",
      "skills",
      "mcp",
      "sessions",
      "tasks",
      "cron-jobs",
      "channels",
      "models",
      "environments"
    ]);
  });

  it("renders the session pane and mobile shell entrypoints", () => {
    render(<LegacyWebShell />);

    expect(document.getElementById("mobileNewChatButton")).toBeInTheDocument();
    expect(document.getElementById("sessionHistoryToggle")).toBeInTheDocument();
    expect(document.getElementById("sessionHistoryPanel")).toBeInTheDocument();
    expect(document.getElementById("sessionLoadError")).toBeInTheDocument();
    expect(document.getElementById(LEGACY_SHELL_IDS.newChatButton)).toHaveTextContent("New Chat");
  });

  it("hydrates the session history collapse state from session storage", () => {
    window.sessionStorage.setItem(
      "alter0.web.session-history-panel.v1",
      JSON.stringify({ collapsed_state: true }),
    );

    render(<LegacyWebShell />);

    expect(screen.getByRole("button", { name: "Expand" })).toHaveAttribute("aria-expanded", "false");
    expect(document.getElementById("sessionHistoryPanel")).toHaveAttribute("hidden");
    expect(document.querySelector(".session-pane")).toHaveClass("history-collapsed");
  });

  it("keeps the active navigation item aligned with the hash route", async () => {
    window.location.hash = "#terminal";

    render(<LegacyWebShell />);

    const appShell = document.getElementById(LEGACY_SHELL_IDS.appShell);
    expect(appShell).toHaveClass("info-mode");
    expect(screen.getByRole("button", { name: "Terminal" })).toHaveClass("active");
    expect(screen.getByRole("button", { name: "Chat" })).not.toHaveClass("active");

    await act(async () => {
      window.location.hash = "#chat";
      window.dispatchEvent(new HashChangeEvent("hashchange"));
    });

    await waitFor(() => {
      expect(appShell).not.toHaveClass("info-mode");
      expect(screen.getByRole("button", { name: "Chat" })).toHaveClass("active");
      expect(screen.getByRole("button", { name: "Terminal" })).not.toHaveClass("active");
    });
  });

  it("keeps the workspace route shell state aligned with the hash route without clearing route mounts", async () => {
    render(<LegacyWebShell />);

    const chatPane = document.querySelector(".chat-pane");
    const chatView = document.getElementById("chatView");
    const routeView = document.getElementById(LEGACY_SHELL_IDS.routeView);
    const routeBody = document.getElementById(LEGACY_SHELL_IDS.routeBody);
    routeBody?.insertAdjacentHTML("beforeend", '<div data-runtime-node="route">runtime route</div>');

    expect(chatPane).toHaveAttribute("data-route", "chat");
    expect(chatPane).not.toHaveClass("page-mode");
    expect(chatView).not.toHaveAttribute("hidden");
    expect(routeView).toHaveAttribute("hidden");

    await act(async () => {
      window.location.hash = "#terminal";
      window.dispatchEvent(new HashChangeEvent("hashchange"));
    });

    await waitFor(() => {
      expect(chatPane).toHaveAttribute("data-route", "terminal");
      expect(chatPane).toHaveClass("page-mode");
      expect(chatView).toHaveAttribute("hidden");
      expect(routeView).not.toHaveAttribute("hidden");
      expect(routeView).toHaveAttribute("data-route", "terminal");
      expect(routeView).toHaveClass("terminal-route");
      expect(routeBody).toHaveAttribute("data-route", "terminal");
      expect(routeBody).toHaveClass("terminal-route-body");
      expect(screen.getByText("runtime route")).toBeInTheDocument();
    });

    await act(async () => {
      window.location.hash = "#agent-runtime";
      window.dispatchEvent(new HashChangeEvent("hashchange"));
    });

    await waitFor(() => {
      expect(chatPane).toHaveAttribute("data-route", "agent-runtime");
      expect(chatPane).not.toHaveClass("page-mode");
      expect(chatView).not.toHaveAttribute("hidden");
      expect(routeView).toHaveAttribute("hidden");
      expect(routeView).toHaveAttribute("data-route", "agent-runtime");
      expect(routeView).not.toHaveClass("terminal-route");
      expect(routeBody).toHaveAttribute("data-route", "agent-runtime");
      expect(routeBody).not.toHaveClass("terminal-route-body");
      expect(screen.getByText("runtime route")).toBeInTheDocument();
    });
  });

  it("keeps the route view heading copy aligned with route and language without clearing route mounts", async () => {
    window.location.hash = "#terminal";

    render(<LegacyWebShell />);

    const routeBody = document.getElementById(LEGACY_SHELL_IDS.routeBody);
    routeBody?.insertAdjacentHTML("beforeend", '<div data-runtime-node="route">runtime route</div>');

    expect(document.getElementById("routeTitle")).toHaveTextContent("Terminal");
    expect(document.getElementById("routeSubtitle")).toHaveTextContent(
      "Persistent Codex CLI sessions with runtime-aligned status",
    );

    await act(async () => {
      document.documentElement.lang = "zh-CN";
    });

    await waitFor(() => {
      expect(document.getElementById("routeTitle")).toHaveTextContent("终端");
      expect(document.getElementById("routeSubtitle")).toHaveTextContent(
        "独立终端会话，状态与实际 shell 进程保持一致",
      );
      expect(screen.getByText("runtime route")).toBeInTheDocument();
    });

    await act(async () => {
      window.location.hash = "#tasks";
      window.dispatchEvent(new HashChangeEvent("hashchange"));
    });

    await waitFor(() => {
      expect(document.getElementById("routeTitle")).toHaveTextContent("任务观测");
      expect(document.getElementById("routeSubtitle")).toHaveTextContent(
        "基于来源、状态和时间范围观测运行任务",
      );
      expect(screen.getByText("runtime route")).toBeInTheDocument();
    });
  });

  it("toggles navigation collapse without clearing legacy runtime mounts", () => {
    const navCollapsedEvents: boolean[] = [];
    document.addEventListener(
      LEGACY_SHELL_SYNC_NAV_COLLAPSED_EVENT,
      ((event: Event) => {
        navCollapsedEvents.push(Boolean((event as CustomEvent<{ collapsed: boolean }>).detail?.collapsed));
      }) as EventListener,
    );

    render(<LegacyWebShell />);

    const runtimeMount = document.getElementById(LEGACY_SHELL_IDS.sessionList);
    runtimeMount?.insertAdjacentHTML("beforeend", '<button type="button" data-runtime-node="session">runtime session</button>');

    fireEvent.click(screen.getByRole("button", { name: "Collapse navigation" }));

    expect(document.getElementById(LEGACY_SHELL_IDS.appShell)).toHaveClass("nav-collapsed");
    expect(screen.getByRole("button", { name: "Expand navigation" })).toHaveAttribute("aria-expanded", "false");
    expect(screen.getByText("runtime session")).toBeInTheDocument();
    expect(navCollapsedEvents).toEqual([true]);
  });

  it("dispatches route navigation bridge events from the primary navigation", () => {
    const routes: string[] = [];
    document.addEventListener(
      LEGACY_SHELL_NAVIGATE_EVENT,
      ((event: Event) => {
        routes.push(String((event as CustomEvent<{ route: string }>).detail?.route || ""));
      }) as EventListener,
    );

    render(<LegacyWebShell />);

    fireEvent.click(screen.getByRole("button", { name: "Tasks" }));

    expect(routes).toEqual(["tasks"]);
  });

  it("shows a delayed tooltip for collapsed navigation route items", async () => {
    vi.useFakeTimers();

    render(<LegacyWebShell />);

    fireEvent.click(screen.getByRole("button", { name: "Collapse navigation" }));
    fireEvent.mouseEnter(screen.getByRole("button", { name: "Tasks" }));

    expect(document.querySelector(".nav-tooltip")).toBeNull();

    await act(async () => {
      vi.advanceTimersByTime(90);
    });

    const tooltip = document.querySelector(".nav-tooltip");
    expect(tooltip).toHaveClass("visible");
    expect(tooltip).toHaveAttribute("aria-hidden", "false");
    expect(tooltip).toHaveTextContent("Tasks");

    fireEvent.mouseLeave(screen.getByRole("button", { name: "Tasks" }));

    await act(async () => {
      vi.advanceTimersByTime(40);
    });

    expect(tooltip).not.toHaveClass("visible");
    expect(tooltip).toHaveAttribute("aria-hidden", "true");

    vi.useRealTimers();
  });

  it("does not show route tooltips while the navigation is expanded", async () => {
    vi.useFakeTimers();

    render(<LegacyWebShell />);

    fireEvent.mouseEnter(screen.getByRole("button", { name: "Tasks" }));

    await act(async () => {
      vi.advanceTimersByTime(90);
    });

    expect(document.querySelector(".nav-tooltip")).toBeNull();

    vi.useRealTimers();
  });

  it("shows an immediate tooltip for the navigation collapse button on focus", () => {
    render(<LegacyWebShell />);

    const collapseButton = screen.getByRole("button", { name: "Collapse navigation" });
    fireEvent.focus(collapseButton);

    const tooltip = document.querySelector(".nav-tooltip");
    expect(tooltip).toHaveClass("visible");
    expect(tooltip).toHaveAttribute("aria-hidden", "false");
    expect(tooltip).toHaveTextContent("Collapse navigation");

    fireEvent.blur(collapseButton);

    expect(tooltip).not.toHaveClass("visible");
    expect(tooltip).toHaveAttribute("aria-hidden", "true");
  });

  it("dispatches route navigation bridge events from the route action button", async () => {
    const routes: string[] = [];
    document.addEventListener(
      LEGACY_SHELL_NAVIGATE_EVENT,
      ((event: Event) => {
        routes.push(String((event as CustomEvent<{ route: string }>).detail?.route || ""));
      }) as EventListener,
    );

    window.location.hash = "#channels";
    render(<LegacyWebShell />);

    const routeActionButton = document.getElementById("routeActionButton") as HTMLButtonElement;
    await act(async () => {
      routeActionButton.hidden = false;
      routeActionButton.dataset.route = "channels";
      fireEvent.click(routeActionButton);
    });

    expect(routes).toEqual(["channels"]);
  });

  it("dispatches a session creation bridge event from both session entrypoints", () => {
    let creationCount = 0;
    document.addEventListener(
      LEGACY_SHELL_CREATE_SESSION_EVENT,
      (() => {
        creationCount += 1;
      }) as EventListener,
    );

    render(<LegacyWebShell />);

    fireEvent.click(document.getElementById(LEGACY_SHELL_IDS.newChatButton)!);
    fireEvent.click(document.getElementById("mobileNewChatButton")!);

    expect(creationCount).toBe(2);
  });

  it("dispatches the language toggle bridge event from the locale button", () => {
    let toggleCount = 0;
    document.addEventListener(
      LEGACY_SHELL_TOGGLE_LANGUAGE_EVENT,
      (() => {
        toggleCount += 1;
      }) as EventListener,
    );

    render(<LegacyWebShell />);

    fireEvent.click(screen.getByRole("button", { name: "Language" }));

    expect(toggleCount).toBe(1);
  });

  it("dispatches quick prompt bridge events from the welcome prompt grid", () => {
    const prompts: string[] = [];
    document.addEventListener(
      LEGACY_SHELL_QUICK_PROMPT_EVENT,
      ((event: Event) => {
        prompts.push(String((event as CustomEvent<{ prompt: string }>).detail?.prompt || ""));
      }) as EventListener,
    );

    render(<LegacyWebShell />);

    fireEvent.click(screen.getByRole("button", { name: "Let's start a new journey!" }));

    expect(prompts).toEqual(["Let's start a new journey!"]);
  });

  it("toggles the mobile navigation drawer through the React shell controls", async () => {
    render(<LegacyWebShell />);

    const appShell = document.getElementById(LEGACY_SHELL_IDS.appShell);
    const navToggle = document.getElementById(LEGACY_SHELL_IDS.navToggle);

    await act(async () => {
      fireEvent.click(navToggle!);
    });
    await waitFor(() => {
      expect(appShell).toHaveClass("nav-open");
      expect(appShell).toHaveClass("overlay-open");
    });

    await act(async () => {
      fireEvent.click(navToggle!);
    });
    await waitFor(() => {
      expect(appShell).not.toHaveClass("nav-open");
      expect(appShell).not.toHaveClass("overlay-open");
    });
  });

  it("toggles the chat session drawer through the React shell controls", async () => {
    render(<LegacyWebShell />);

    const appShell = document.getElementById(LEGACY_SHELL_IDS.appShell);
    const sessionToggle = document.getElementById(LEGACY_SHELL_IDS.sessionToggle);

    await act(async () => {
      fireEvent.click(sessionToggle!);
    });
    await waitFor(() => {
      expect(appShell).toHaveClass("panel-open");
      expect(appShell).toHaveClass("overlay-open");
    });

    await act(async () => {
      fireEvent.click(document.getElementById("togglePaneButton")!);
    });
    await waitFor(() => {
      expect(appShell).not.toHaveClass("panel-open");
      expect(appShell).not.toHaveClass("overlay-open");
    });
  });

  it("delegates terminal session drawer toggles to the terminal route mount", async () => {
    window.location.hash = "#terminal";

    render(<LegacyWebShell />);

    const routeBody = document.getElementById(LEGACY_SHELL_IDS.routeBody);
    const appShell = document.getElementById(LEGACY_SHELL_IDS.appShell);
    const terminalToggle = document.createElement("button");
    terminalToggle.type = "button";
    terminalToggle.setAttribute("data-terminal-session-pane-toggle", "");
    routeBody?.appendChild(terminalToggle);

    const terminalToggleSpy = vi.fn();
    terminalToggle.addEventListener("click", terminalToggleSpy);

    fireEvent.click(document.getElementById(LEGACY_SHELL_IDS.sessionToggle)!);

    expect(terminalToggleSpy).toHaveBeenCalledTimes(1);
    expect(appShell).not.toHaveClass("panel-open");
    expect(appShell).not.toHaveClass("overlay-open");
  });

  it("closes transient drawers from the mobile backdrop without clearing runtime mounts", async () => {
    render(<LegacyWebShell />);

    const appShell = document.getElementById(LEGACY_SHELL_IDS.appShell);
    const runtimeMount = document.getElementById(LEGACY_SHELL_IDS.sessionList);
    runtimeMount?.insertAdjacentHTML("beforeend", '<button type="button" data-runtime-node="session">runtime session</button>');

    await act(async () => {
      appShell?.classList.add("nav-open", "panel-open", "overlay-open");
    });

    await act(async () => {
      fireEvent.click(document.getElementById(LEGACY_SHELL_IDS.mobileBackdrop)!);
    });

    await waitFor(() => {
      expect(appShell).not.toHaveClass("nav-open");
      expect(appShell).not.toHaveClass("panel-open");
      expect(appShell).not.toHaveClass("overlay-open");
    });
    expect(screen.getByText("runtime session")).toBeInTheDocument();
  });

  it("follows document language changes for the primary navigation copy", async () => {
    render(<LegacyWebShell />);

    await act(async () => {
      document.documentElement.lang = "zh-CN";
    });

    await waitFor(() => {
      expect(screen.getByRole("heading", { name: "工作区" })).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "终端代理" })).toHaveAttribute("data-route", "terminal");
      expect(screen.getByRole("button", { name: "语言" })).toHaveTextContent("中文");
    });
  });

  it("toggles the session history collapse state without clearing runtime session mounts", () => {
    const collapsedEvents: boolean[] = [];
    document.addEventListener(
      LEGACY_SHELL_SYNC_SESSION_HISTORY_EVENT,
      ((event: Event) => {
        collapsedEvents.push(Boolean((event as CustomEvent<{ collapsed: boolean }>).detail?.collapsed));
      }) as EventListener,
    );

    render(<LegacyWebShell />);

    const runtimeMount = document.getElementById(LEGACY_SHELL_IDS.sessionList);
    runtimeMount?.insertAdjacentHTML("beforeend", '<button type="button" data-runtime-node="session">runtime session</button>');

    fireEvent.click(screen.getByRole("button", { name: "Collapse" }));

    expect(screen.getByRole("button", { name: "Expand" })).toHaveAttribute("aria-expanded", "false");
    expect(document.getElementById("sessionHistoryPanel")).toHaveAttribute("hidden");
    expect(window.sessionStorage.getItem("alter0.web.session-history-panel.v1")).toBe(
      JSON.stringify({ collapsed_state: true }),
    );
    expect(screen.getByText("runtime session")).toBeInTheDocument();
    expect(collapsedEvents).toEqual([true]);
  });

  it("keeps the session pane copy aligned with the route and language", async () => {
    window.location.hash = "#agent-runtime";

    render(<LegacyWebShell />);

    expect(document.getElementById(LEGACY_SHELL_IDS.newChatButton)).toHaveTextContent("New Agent Run");
    expect(document.getElementById("sessionEmpty")).toHaveTextContent("No Agent sessions yet. Open Agent to start.");
    expect(document.getElementById(LEGACY_SHELL_IDS.sessionList)).toHaveAttribute("aria-label", "Agent conversation sessions");

    await act(async () => {
      document.documentElement.lang = "zh-CN";
    });

    await waitFor(() => {
      expect(screen.getByText("与 alter0 协作")).toBeInTheDocument();
      expect(document.getElementById(LEGACY_SHELL_IDS.newChatButton)).toHaveTextContent("新 Agent 会话");
      expect(document.getElementById("sessionEmpty")).toHaveTextContent("当前还没有 Agent 会话。请前往 Agent 页面开始。");
      expect(document.getElementById(LEGACY_SHELL_IDS.sessionList)).toHaveAttribute("aria-label", "Agent 会话列表");
      expect(screen.getByRole("button", { name: "折叠" })).toHaveAttribute("aria-controls", "sessionHistoryPanel");
    });
  });

  it("preserves legacy shell transient classes across React rerenders", async () => {
    render(<LegacyWebShell />);

    const appShell = document.getElementById(LEGACY_SHELL_IDS.appShell);
    appShell?.classList.add("nav-open", "panel-open", "overlay-open", "runtime-sheet-open");

    await act(async () => {
      window.location.hash = "#terminal";
      window.dispatchEvent(new HashChangeEvent("hashchange"));
    });

    await waitFor(() => {
      expect(appShell).toHaveClass("info-mode");
      expect(appShell).toHaveClass("nav-open");
      expect(appShell).toHaveClass("panel-open");
      expect(appShell).toHaveClass("overlay-open");
      expect(appShell).toHaveClass("runtime-sheet-open");
    });
  });

  it("tracks legacy shell transient class removals after external cleanup", async () => {
    render(<LegacyWebShell />);

    const appShell = document.getElementById(LEGACY_SHELL_IDS.appShell);
    appShell?.classList.add("nav-open", "overlay-open");

    appShell?.classList.remove("nav-open", "overlay-open");

    await act(async () => {
      window.location.hash = "#terminal";
      window.dispatchEvent(new HashChangeEvent("hashchange"));
    });

    await waitFor(() => {
      expect(appShell).toHaveClass("info-mode");
      expect(appShell).not.toHaveClass("nav-open");
      expect(appShell).not.toHaveClass("overlay-open");
    });
  });

  it("keeps the chat workspace action labels aligned with route and language", async () => {
    render(<LegacyWebShell />);

    const navToggle = document.getElementById(LEGACY_SHELL_IDS.navToggle);
    const sessionToggle = document.getElementById(LEGACY_SHELL_IDS.sessionToggle);
    const mobileNewChatButton = document.getElementById("mobileNewChatButton");

    expect(navToggle).toHaveTextContent("Menu");
    expect(navToggle).toHaveAttribute("aria-label", "Menu");
    expect(sessionToggle).toHaveTextContent("Sessions");
    expect(sessionToggle).toHaveAttribute("aria-label", "Sessions");
    expect(mobileNewChatButton).toHaveTextContent("New Chat");
    expect(mobileNewChatButton).toHaveAttribute("aria-label", "New Chat");

    await act(async () => {
      window.location.hash = "#terminal";
      window.dispatchEvent(new HashChangeEvent("hashchange"));
    });

    await waitFor(() => {
      expect(sessionToggle).toHaveTextContent("Sessions");
      expect(sessionToggle).toHaveAttribute("aria-label", "Sessions");
      expect(mobileNewChatButton).toHaveTextContent("New");
      expect(mobileNewChatButton).toHaveAttribute("aria-label", "New");
    });

    await act(async () => {
      document.documentElement.lang = "zh-CN";
    });

    await waitFor(() => {
      expect(navToggle).toHaveTextContent("菜单");
      expect(navToggle).toHaveAttribute("aria-label", "菜单");
      expect(sessionToggle).toHaveTextContent("会话列表");
      expect(sessionToggle).toHaveAttribute("aria-label", "会话列表");
      expect(mobileNewChatButton).toHaveTextContent("新建");
      expect(mobileNewChatButton).toHaveAttribute("aria-label", "新建");
    });
  });

  it("keeps runtime session pane snapshot state across shell rerenders", async () => {
    render(<LegacyWebShell />);

    const sessionList = document.getElementById(LEGACY_SHELL_IDS.sessionList);
    sessionList?.insertAdjacentHTML("beforeend", '<button type="button" data-runtime-node="session">runtime session</button>');

    await act(async () => {
      document.dispatchEvent(
        new CustomEvent(LEGACY_SHELL_SYNC_SESSION_PANE_EVENT, {
          detail: {
            route: "chat",
            hasSessions: true,
            loadError: "Runtime load error",
          },
        }),
      );
    });

    expect(document.getElementById("sessionEmpty")).toHaveAttribute("hidden");
    expect(document.getElementById("sessionLoadError")).not.toHaveAttribute("hidden");
    expect(document.getElementById("sessionLoadError")).toHaveTextContent("Runtime load error");
    expect(screen.getByText("runtime session")).toBeInTheDocument();

    await act(async () => {
      document.documentElement.lang = "zh-CN";
    });

    await waitFor(() => {
      expect(document.getElementById("sessionEmpty")).toHaveAttribute("hidden");
      expect(document.getElementById("sessionLoadError")).not.toHaveAttribute("hidden");
      expect(document.getElementById("sessionLoadError")).toHaveTextContent("Runtime load error");
      expect(screen.getByText("runtime session")).toBeInTheDocument();
    });
  });

  it("renders session snapshot cards and dispatches session item bridge events", async () => {
    const focusedSessions: string[] = [];
    const removedSessions: string[] = [];
    document.addEventListener(
      LEGACY_SHELL_FOCUS_SESSION_EVENT,
      ((event: Event) => {
        focusedSessions.push(String((event as CustomEvent<{ sessionId: string }>).detail?.sessionId || ""));
      }) as EventListener,
    );
    document.addEventListener(
      LEGACY_SHELL_REMOVE_SESSION_EVENT,
      ((event: Event) => {
        removedSessions.push(String((event as CustomEvent<{ sessionId: string }>).detail?.sessionId || ""));
      }) as EventListener,
    );

    window.location.hash = "#agent-runtime";
    render(<LegacyWebShell />);

    await act(async () => {
      document.dispatchEvent(
        new CustomEvent(LEGACY_SHELL_SYNC_SESSION_PANE_EVENT, {
          detail: {
            route: "agent-runtime",
            hasSessions: true,
            loadError: "",
            items: [
              {
                id: "session-runtime-1",
                title: "Runtime session",
                meta: "Agent · provider / model · 3 messages · just now",
                active: true,
                shortHash: "1a2b3c4d",
                copyValue: "1a2b3c4d",
                copyLabel: "Copy session id",
                deleteLabel: "Delete session",
              },
            ],
          },
        }),
      );
    });

    expect(screen.getByText("Runtime session")).toBeInTheDocument();
    expect(screen.getByText("Agent · provider / model · 3 messages · just now")).toBeInTheDocument();
    expect(screen.getByText("#1a2b3c4d")).toBeInTheDocument();

    fireEvent.click(screen.getByText("Runtime session").closest("button")!);
    fireEvent.click(screen.getByRole("button", { name: "Delete session" }));

    expect(focusedSessions).toEqual(["session-runtime-1"]);
    expect(removedSessions).toEqual(["session-runtime-1"]);
  });

  it("keeps runtime message region snapshot state across shell rerenders", async () => {
    render(<LegacyWebShell />);

    const messageArea = document.getElementById(LEGACY_SHELL_IDS.messageArea);
    messageArea?.insertAdjacentHTML("beforeend", '<div data-runtime-node="message">runtime message</div>');

    await act(async () => {
      document.dispatchEvent(
        new CustomEvent(LEGACY_SHELL_SYNC_MESSAGE_REGION_EVENT, {
          detail: {
            route: "chat",
            hasMessages: true,
          },
        }),
      );
    });

    expect(document.querySelector(".chat-pane")).not.toHaveClass("empty-state");
    expect(document.getElementById(LEGACY_SHELL_IDS.welcomeScreen)).toHaveAttribute("hidden");
    expect(document.getElementById(LEGACY_SHELL_IDS.messageArea)).not.toHaveAttribute("hidden");
    expect(screen.getByText("runtime message")).toBeInTheDocument();

    await act(async () => {
      document.documentElement.lang = "zh-CN";
    });

    await waitFor(() => {
      expect(document.querySelector(".chat-pane")).not.toHaveClass("empty-state");
      expect(document.getElementById(LEGACY_SHELL_IDS.welcomeScreen)).toHaveAttribute("hidden");
      expect(document.getElementById(LEGACY_SHELL_IDS.messageArea)).not.toHaveAttribute("hidden");
      expect(screen.getByText("runtime message")).toBeInTheDocument();
    });
  });

  it("renders runtime message region snapshot content across shell rerenders", async () => {
    render(<LegacyWebShell />);

    await act(async () => {
      document.dispatchEvent(
        new CustomEvent(LEGACY_SHELL_SYNC_MESSAGE_REGION_EVENT, {
          detail: {
            route: "chat",
            hasMessages: true,
            sessionId: "session-runtime-1",
            html: '<div class="message-list" data-message-session-id="session-runtime-1"><article class="msg user" data-message-id="msg-runtime-1"><div class="msg-bubble"><p>Runtime user message</p></div><div class="msg-meta"><span>just now</span></div></article></div>',
          },
        }),
      );
    });

    expect(screen.getByText("Runtime user message")).toBeInTheDocument();
    expect(document.querySelector('[data-message-session-id="session-runtime-1"]')).toBeInTheDocument();

    await act(async () => {
      document.documentElement.lang = "zh-CN";
    });

    await waitFor(() => {
      expect(screen.getByText("Runtime user message")).toBeInTheDocument();
      expect(document.querySelector('[data-message-session-id="session-runtime-1"]')).toBeInTheDocument();
    });
  });

  it("keeps runtime chat workspace snapshot copy across shell rerenders", async () => {
    render(<LegacyWebShell />);

    const messageArea = document.getElementById(LEGACY_SHELL_IDS.messageArea);
    messageArea?.insertAdjacentHTML("beforeend", '<div data-runtime-node="message">runtime message</div>');

    await act(async () => {
      document.dispatchEvent(
        new CustomEvent(LEGACY_SHELL_SYNC_CHAT_WORKSPACE_EVENT, {
          detail: {
            route: "chat",
            heading: "Runtime heading",
            subheading: "Runtime subheading",
            welcomeHeading: "Runtime welcome heading",
            welcomeDescription: "Runtime welcome description",
            welcomeTargetHTML: '<button class="welcome-target-card active" type="button" data-chat-target-type="agent" data-chat-target-id="agent-runtime" data-chat-target-name="Runtime Agent"><strong>Runtime Agent</strong><span>agent-runtime</span></button>',
          },
        }),
      );
    });

    expect(document.getElementById("sessionHeading")).toHaveTextContent("Runtime heading");
    expect(document.getElementById("sessionSubheading")).toHaveTextContent("Runtime subheading");
    expect(document.getElementById("welcomeHeading")).toHaveTextContent("Runtime welcome heading");
    expect(document.getElementById("welcomeDescription")).toHaveTextContent("Runtime welcome description");
    expect(screen.getByText("Runtime Agent")).toBeInTheDocument();
    expect(screen.getByText("runtime message")).toBeInTheDocument();

    await act(async () => {
      document.documentElement.lang = "zh-CN";
    });

    await waitFor(() => {
      expect(document.getElementById("sessionHeading")).toHaveTextContent("Runtime heading");
      expect(document.getElementById("sessionSubheading")).toHaveTextContent("Runtime subheading");
      expect(document.getElementById("welcomeHeading")).toHaveTextContent("Runtime welcome heading");
      expect(document.getElementById("welcomeDescription")).toHaveTextContent("Runtime welcome description");
      expect(screen.getByText("Runtime Agent")).toBeInTheDocument();
      expect(screen.getByText("runtime message")).toBeInTheDocument();
    });
  });

  it("keeps runtime panel snapshot content across shell rerenders", async () => {
    render(<LegacyWebShell />);

    await act(async () => {
      document.dispatchEvent(
        new CustomEvent(LEGACY_SHELL_SYNC_CHAT_RUNTIME_EVENT, {
          detail: {
            route: "chat",
            controlsHTML: '<div class="composer-runtime-group"><button class="composer-runtime-trigger is-open" type="button" data-runtime-toggle="model"><span class="composer-runtime-trigger-label">Runtime model</span></button></div>',
            noteHTML: '<p class="chat-runtime-note chat-runtime-error">Runtime provider warning</p>',
            sheetHTML: '<div class="composer-runtime-popover-mobile-body" data-runtime-scroll-container="mobile"><label class="composer-runtime-checkbox"><input type="checkbox" data-runtime-toggle-item="skills" value="runtime-skill" checked><span class="composer-runtime-checkbox-copy"><strong>Runtime skill</strong><span>Enabled in runtime</span></span></label></div>',
          },
        }),
      );
    });

    expect(screen.getByText("Runtime model")).toBeInTheDocument();
    expect(screen.getByText("Runtime provider warning")).toBeInTheDocument();
    expect(screen.getByText("Runtime skill")).toBeInTheDocument();

    await act(async () => {
      document.documentElement.lang = "zh-CN";
    });

    await waitFor(() => {
      expect(screen.getByText("Runtime model")).toBeInTheDocument();
      expect(screen.getByText("Runtime provider warning")).toBeInTheDocument();
      expect(screen.getByText("Runtime skill")).toBeInTheDocument();
    });
  });

  it("restores runtime panel scroll snapshots after React rerenders", async () => {
    render(<LegacyWebShell />);

    await act(async () => {
      document.dispatchEvent(
        new CustomEvent(LEGACY_SHELL_SYNC_CHAT_RUNTIME_EVENT, {
          detail: {
            route: "chat",
            controlsHTML: "",
            noteHTML: "",
            sheetHTML: '<div class="composer-runtime-popover-mobile-body" data-runtime-scroll-container="mobile"><div style="height: 400px;">Runtime sheet body</div></div>',
            scrollPopover: "mobile",
            scrollTop: 96,
          },
        }),
      );
    });

    await waitFor(() => {
      const scrollContainer = document.querySelector<HTMLElement>('[data-runtime-scroll-container="mobile"]');
      expect(scrollContainer).not.toBeNull();
      expect(scrollContainer?.scrollTop).toBe(96);
    });
  });

  it("renders React-managed control route cards inside the route body", async () => {
    window.location.hash = "#channels";
    vi.mocked(fetch).mockResolvedValueOnce(
      jsonResponse({
        items: [
          {
            id: "channel-runtime-1",
            type: "web",
            description: "Primary web entry for the cockpit shell.",
            enabled: true,
          },
        ],
      }),
    );

    render(<LegacyWebShell />);

    await waitFor(() => {
      expect(screen.getAllByText("channel-runtime-1")).toHaveLength(2);
    });

    expect(screen.getByText("Primary web entry for the cockpit shell.")).toBeInTheDocument();
    expect(document.getElementById(LEGACY_SHELL_IDS.routeBody)?.querySelector(".route-card")).toBeInTheDocument();

    await act(async () => {
      document.documentElement.lang = "zh-CN";
    });

    await waitFor(() => {
      expect(screen.getAllByText("channel-runtime-1")).toHaveLength(2);
      expect(screen.getByText("启用")).toBeInTheDocument();
    });
  });

  it("renders the React-managed sessions route body inside the route view", async () => {
    window.location.hash = "#sessions";
    vi.mocked(fetch).mockResolvedValueOnce(
      jsonResponse({
        items: [
          {
            session_id: "session-runtime-1",
            channel_type: "web",
            channel_id: "web-default",
            last_message_id: "msg-runtime-1",
            updated_at: "2026-04-13T01:02:03Z",
            created_at: "2026-04-13T00:00:00Z",
            message_count: 3,
            trigger_type: "user",
            job_id: "",
            job_name: "",
            fired_at: "",
          },
        ],
      }),
    );

    render(<LegacyWebShell />);

    await waitFor(() => {
      expect(screen.getAllByText("session-runtime-1")).toHaveLength(2);
    });

    expect(screen.getByText("View Detail")).toBeInTheDocument();
    expect(screen.getAllByText("Web").length).toBeGreaterThan(0);
    expect(document.getElementById(LEGACY_SHELL_IDS.routeBody)?.querySelector(".session-route-card")).toBeInTheDocument();
  });

  it("renders the React-managed products route body inside the route view", async () => {
    window.location.hash = "#products";
    vi.mocked(fetch)
      .mockResolvedValueOnce(
        jsonResponse({
          items: [
            {
              id: "travel",
              name: "Travel",
              summary: "City guides with workspace-backed detail pages.",
              status: "active",
              visibility: "public",
              owner_type: "builtin",
              version: "v1.2.0",
            },
          ],
        }),
      )
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
      .mockResolvedValueOnce(
        jsonResponse({
          product: {
            id: "travel",
            name: "Travel",
            summary: "City guides with workspace-backed detail pages.",
            status: "active",
            visibility: "public",
            version: "v1.2.0",
          },
          master_agent: {
            agent_id: "travel-master",
            name: "Travel Master",
            description: "Maintains city guide spaces.",
            tools: ["codex_exec"],
          },
          spaces: [
            {
              space_id: "wuhan",
              title: "Wuhan",
              summary: "Metro-first night food city guide.",
              html_path: "/products/travel/spaces/wuhan.html",
              status: "active",
              tags: ["metro"],
            },
          ],
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          space: {
            space_id: "wuhan",
            title: "Wuhan",
            summary: "Metro-first night food city guide.",
          },
          guide: {
            id: "wuhan",
            city: "Wuhan",
            days: 3,
            travel_style: "metro-first",
            budget: "mid-range",
            companions: [],
            must_visit: [],
            avoid: [],
            additional_requirements: [],
            keep_conditions: [],
            replace_conditions: [],
            notes: [],
            daily_routes: [],
            map_layers: [],
            content: "Initial content",
            revision: 1,
            updated_at: "2026-04-15T01:02:03Z",
          },
        }),
      );

    render(<LegacyWebShell />);

    await waitFor(() => {
      expect(screen.getAllByText("Travel").length).toBeGreaterThan(0);
    });

    expect(screen.getByText("Product Workspace")).toBeInTheDocument();
    expect(document.getElementById(LEGACY_SHELL_IDS.routeBody)?.querySelector(".product-workspace-grid")).toBeInTheDocument();
  });
});

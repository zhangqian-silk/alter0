import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import {
  LEGACY_SHELL_CREATE_SESSION_EVENT,
  LEGACY_SHELL_FOCUS_SESSION_EVENT,
  LEGACY_SHELL_NAVIGATE_EVENT,
  LEGACY_SHELL_QUICK_PROMPT_EVENT,
  LEGACY_SHELL_REMOVE_SESSION_EVENT,
  LEGACY_SHELL_SYNC_NAV_COLLAPSED_EVENT,
  LEGACY_SHELL_SYNC_SESSION_HISTORY_EVENT,
  LEGACY_SHELL_TOGGLE_LANGUAGE_EVENT,
} from "./legacyShellBridge";
import { LegacyWebShell } from "./LegacyWebShell";
import { LEGACY_SHELL_IDS } from "./legacyDomContract";
import { resetLegacyRuntimeSnapshotStore } from "./legacyRuntimeSnapshotStore";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: init.status ?? 200,
    headers: {
      "Content-Type": "application/json",
      ...(init.headers ?? {}),
    },
  });
}

function applyScrollableMetrics(
  container: HTMLElement,
  targets: HTMLElement[],
  targetTops: number[],
  options: {
    clientHeight: number;
    scrollHeight: number;
  },
) {
  Object.defineProperty(container, "clientHeight", {
    configurable: true,
    value: options.clientHeight,
  });
  Object.defineProperty(container, "scrollHeight", {
    configurable: true,
    value: options.scrollHeight,
  });
  Object.defineProperty(container, "scrollTop", {
    configurable: true,
    writable: true,
    value: 0,
  });
  container.getBoundingClientRect = () => ({
    x: 0,
    y: 0,
    width: 640,
    height: options.clientHeight,
    top: 0,
    right: 640,
    bottom: options.clientHeight,
    left: 0,
    toJSON: () => ({}),
  });
  container.scrollTo = vi.fn(({ top }: ScrollToOptions) => {
    container.scrollTop = Math.max(Number(top || 0), 0);
    fireEvent.scroll(container);
  });

  targets.forEach((target, index) => {
    target.getBoundingClientRect = () => {
      const top = targetTops[index] - container.scrollTop;
      return {
        x: 0,
        y: top,
        width: 520,
        height: 96,
        top,
        right: 520,
        bottom: top + 96,
        left: 0,
        toJSON: () => ({}),
      };
    };
  });
}

function runtime() {
  if (!window.__alter0LegacyRuntime) {
    throw new Error("legacy runtime bridge unavailable");
  }
  return window.__alter0LegacyRuntime;
}

function runtimeSnapshot(overrides: Record<string, unknown> = {}) {
  return {
    route: "chat",
    compact: false,
    openPopover: "",
    note: "",
    agentRuntime: false,
    locked: false,
    target: {
      type: "model",
      id: "raw-model",
      name: "Raw Model",
    },
    targetOptions: [],
    selectedProviderId: "openai",
    selectedModelId: "gpt-5.4",
    selectedModelLabel: "GPT-5.4",
    toolCount: 0,
    skillCount: 0,
    providers: [
      {
        id: "openai",
        name: "OpenAI",
        models: [
          {
            id: "gpt-5.4",
            name: "GPT-5.4",
            active: true,
          },
        ],
      },
    ],
    capabilities: [],
    skills: [],
    ...overrides,
  };
}

async function renderChannelsRouteShell() {
  window.location.hash = "#channels";
  render(<LegacyWebShell />);
  await screen.findByText("No Channels available.");
}

async function renderAgentRuntimeShell() {
  window.location.hash = "#agent-runtime";
  render(<LegacyWebShell />);
  await screen.findByRole("button", { name: "Send message" });
}

async function renderTerminalRouteShell() {
  window.location.hash = "#terminal";
  render(<LegacyWebShell />);
  await screen.findByText("No terminal sessions yet.");
}

describe("LegacyWebShell", () => {
  beforeEach(() => {
    resetLegacyRuntimeSnapshotStore();
    delete window.__alter0LegacyRuntime;
    vi.stubGlobal(
      "fetch",
      vi.fn((input: RequestInfo | URL) => {
        const url = String(input);
        if (url === "/api/control/channels" || url === "/api/control/skills" || url === "/api/control/mcps") {
          return Promise.resolve(jsonResponse({ items: [] }));
        }
        if (url === "/api/terminal/sessions") {
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

  it("renders a simplified chat shell without duplicated context surfaces", () => {
    render(<LegacyWebShell />);

    expect(document.querySelector('[data-shell-section="brand-panel"]')).toBeInTheDocument();
    expect(document.querySelector('[data-shell-section="brand-panel"] .brand-mark')).not.toBeInTheDocument();
    expect(document.querySelector('[data-shell-section="brand-panel"] .nav-collapse')).not.toBeInTheDocument();
    expect(document.querySelector('[data-shell-section="brand-panel"] .brand strong')).toHaveTextContent("Alter0");
    expect(document.querySelector('[data-shell-section="brand-panel"] .brand-kicker')).not.toBeInTheDocument();
    expect(document.querySelector('[data-shell-section="brand-panel"] .brand-description')).not.toBeInTheDocument();
    expect(document.querySelector('[data-shell-section="brand-panel"] .brand-status-strip')).not.toBeInTheDocument();
    expect(document.querySelector('[data-shell-section="session-context"]')).not.toBeInTheDocument();
    expect(document.querySelector('[data-shell-section="chat-hero"]')).not.toBeInTheDocument();
    expect(document.querySelector('[data-shell-section="prompt-deck"]')).toBeInTheDocument();
    expect(document.querySelector('[data-shell-section="composer-panel"]')).toBeInTheDocument();
    expect(document.querySelector('[data-shell-section="route-hero"]')).not.toBeInTheDocument();
  });

  it("centers chat reading surfaces inside a shared content frame", () => {
    render(<LegacyWebShell />);

    expect(document.querySelector(".welcome-screen .chat-content-frame")).toBeInTheDocument();
    expect(document.querySelector(".composer-shell .chat-content-frame")).toBeInTheDocument();
  });

  it("renders a single route heading surface for page-mode routes", async () => {
    await renderChannelsRouteShell();

    expect(document.querySelector('[data-shell-section="route-hero"]')).toBeInTheDocument();
    expect(document.querySelector('[data-shell-section="chat-hero"]')).not.toBeInTheDocument();
    expect(document.querySelector(".route-head")).not.toBeInTheDocument();
  });

  it("marks React-managed route bodies so the legacy runtime can skip DOM ownership", async () => {
    await renderChannelsRouteShell();

    expect(document.getElementById(LEGACY_SHELL_IDS.routeBody)).toHaveAttribute(
      "data-react-managed-route",
      "true",
    );
  });

  it("publishes the stable set of React-managed routes on the shell root", () => {
    render(<LegacyWebShell />);

    expect(document.getElementById(LEGACY_SHELL_IDS.appShell)).toHaveAttribute(
      "data-react-managed-routes",
      "agent,terminal,products,memory,sessions,tasks,codex-accounts,channels,skills,mcp,models,environments,cron-jobs",
    );
  });

  it("marks the terminal route body as React-managed and renders the React terminal shell", async () => {
    await renderTerminalRouteShell();

    const routeBody = document.getElementById(LEGACY_SHELL_IDS.routeBody);

    expect(routeBody).toHaveAttribute(
      "data-react-managed-route",
      "true",
    );
    expect(routeBody?.querySelector("[data-terminal-view]")).toBeInTheDocument();
    expect(routeBody?.querySelector("[data-terminal-session-pane]")).toBeInTheDocument();
    expect(routeBody?.querySelector(".terminal-workspace")).toBeInTheDocument();
    expect(document.querySelector('[data-shell-section="route-hero"]')).not.toBeInTheDocument();
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
      "environments",
      "codex-accounts"
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

    expect(document.getElementById("routeTitle")).toBeNull();
    expect(document.getElementById("routeSubtitle")).toBeNull();

    await act(async () => {
      document.documentElement.lang = "zh-CN";
    });

    await waitFor(() => {
      expect(document.getElementById("routeTitle")).toBeNull();
      expect(document.getElementById("routeSubtitle")).toBeNull();
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

  it("does not render the retired route action button shell mount", async () => {
    await renderChannelsRouteShell();

    expect(document.getElementById("routeActionButton")).toBeNull();
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
    await renderTerminalRouteShell();

    const appShell = document.getElementById(LEGACY_SHELL_IDS.appShell);
    const terminalToggle = document.querySelector("[data-terminal-session-pane-toggle]") as HTMLButtonElement | null;
    expect(terminalToggle).toBeInTheDocument();

    const terminalToggleSpy = vi.fn();
    terminalToggle?.addEventListener("click", terminalToggleSpy);

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
      runtime().publishSessionPaneSnapshot?.({
        route: "chat",
        hasSessions: true,
        loadError: "Runtime load error",
      });
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
      runtime().publishSessionPaneSnapshot?.({
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
      });
    });

    expect(screen.getByText("Runtime session")).toBeInTheDocument();
    expect(screen.getByText("Agent · provider / model · 3 messages · just now")).toBeInTheDocument();
    expect(screen.getByText("#1a2b3c4d")).toBeInTheDocument();

    fireEvent.click(screen.getByText("Runtime session").closest("button")!);
    fireEvent.click(screen.getByRole("button", { name: "Delete session" }));

    expect(focusedSessions).toEqual(["session-runtime-1"]);
    expect(removedSessions).toEqual(["session-runtime-1"]);
  });

  it("keeps structured runtime message region state across shell rerenders", async () => {
    render(<LegacyWebShell />);

    await act(async () => {
      runtime().publishMessageRegionSnapshot?.({
        route: "chat",
        hasMessages: true,
        sessionId: "session-runtime-keep",
        messages: [
          {
            id: "msg-runtime-keep",
            role: "assistant",
            text: "Runtime message",
            status: "done",
            error: false,
            at: Date.UTC(2026, 3, 15, 11, 0, 0),
            source: "model",
          },
        ],
      });
    });

    expect(document.querySelector(".chat-pane")).not.toHaveClass("empty-state");
    expect(document.getElementById(LEGACY_SHELL_IDS.welcomeScreen)).toHaveAttribute("hidden");
    expect(document.getElementById(LEGACY_SHELL_IDS.messageArea)).not.toHaveAttribute("hidden");
    expect(screen.getByText("Runtime message")).toBeInTheDocument();
    expect(document.querySelector('[data-message-session-id="session-runtime-keep"]')).toBeInTheDocument();

    await act(async () => {
      document.documentElement.lang = "zh-CN";
    });

    await waitFor(() => {
      expect(document.querySelector(".chat-pane")).not.toHaveClass("empty-state");
      expect(document.getElementById(LEGACY_SHELL_IDS.welcomeScreen)).toHaveAttribute("hidden");
      expect(document.getElementById(LEGACY_SHELL_IDS.messageArea)).not.toHaveAttribute("hidden");
      expect(screen.getByText("Runtime message")).toBeInTheDocument();
      expect(document.querySelector('[data-message-session-id="session-runtime-keep"]')).toBeInTheDocument();
    });
  });

  it("ignores runtime message region snapshots for other routes", async () => {
    render(<LegacyWebShell />);

    await act(async () => {
      runtime().publishMessageRegionSnapshot?.({
        route: "terminal",
        hasMessages: true,
        messages: [
          {
            id: "msg-terminal-scope",
            role: "assistant",
            text: "terminal message should stay scoped",
            status: "done",
            error: false,
            at: Date.UTC(2026, 3, 15, 12, 0, 0),
          },
        ],
      });
    });

    expect(document.getElementById(LEGACY_SHELL_IDS.welcomeScreen)).not.toHaveAttribute("hidden");
    expect(document.getElementById(LEGACY_SHELL_IDS.messageArea)).toHaveAttribute("hidden");
    expect(screen.queryByText("terminal message should stay scoped")).toBeNull();
  });

  it("renders runtime message region snapshot content across shell rerenders", async () => {
    render(<LegacyWebShell />);

    await act(async () => {
      runtime().publishMessageRegionSnapshot?.({
        route: "chat",
        hasMessages: true,
        sessionId: "session-runtime-1",
        messages: [
          {
            id: "msg-runtime-1",
            role: "user",
            text: "Runtime user message",
            at: Date.UTC(2026, 3, 15, 12, 30, 0),
          },
        ],
      });
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

  it("shows arrow jump controls only in the agent runtime message area", async () => {
    const rafSpy = vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback: FrameRequestCallback) => {
      window.setTimeout(() => callback(16), 0);
      return 1;
    });
    const cancelSpy = vi.spyOn(window, "cancelAnimationFrame").mockImplementation(() => undefined);

    try {
      await renderAgentRuntimeShell();

      await act(async () => {
        runtime().publishMessageRegionSnapshot?.({
          route: "agent-runtime",
          hasMessages: true,
          sessionId: "session-runtime-jump",
          messages: Array.from({ length: 6 }, (_value, index) => ({
            id: `msg-runtime-jump-${index + 1}`,
            role: index % 2 === 0 ? "user" : "assistant",
            text: `Runtime jump message ${index + 1}`,
            status: "done",
            error: false,
            at: Date.UTC(2026, 3, 15, 13, index, 0),
            source: "model",
          })),
        });
      });

      const messageArea = document.getElementById(LEGACY_SHELL_IDS.messageArea) as HTMLElement;
      const messages = [...messageArea.querySelectorAll<HTMLElement>("[data-message-id]")];
      applyScrollableMetrics(messageArea, messages, [0, 120, 280, 440, 600, 760], {
        clientHeight: 260,
        scrollHeight: 980,
      });

      messageArea.scrollTop = 340;
      fireEvent.scroll(messageArea);

      await waitFor(() => {
        expect(document.querySelector("[data-scroll-jump-top='agent']")).toHaveClass("is-visible");
        expect(document.querySelector("[data-scroll-jump-prev='agent']")).toHaveClass("is-visible");
        expect(document.querySelector("[data-scroll-jump-next='agent']")).toHaveClass("is-visible");
        expect(document.querySelector("[data-scroll-jump-bottom='agent']")).toHaveClass("is-visible");
      });

      expect(document.querySelector("[data-scroll-jump-top='agent']")).toHaveAttribute("aria-label", "Top");
      expect(document.querySelector("[data-scroll-jump-prev='agent']")).toHaveTextContent("↑");
      expect(document.querySelector("[data-scroll-jump-next='agent']")).toHaveTextContent("↓");
      expect(document.querySelector("[data-scroll-jump-bottom='agent']")).toHaveAttribute("aria-label", "Latest");
    } finally {
      rafSpy.mockRestore();
      cancelSpy.mockRestore();
    }
  });

  it("does not render jump controls for the regular chat message area", async () => {
    const rafSpy = vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback: FrameRequestCallback) => {
      window.setTimeout(() => callback(16), 0);
      return 1;
    });
    const cancelSpy = vi.spyOn(window, "cancelAnimationFrame").mockImplementation(() => undefined);

    try {
      render(<LegacyWebShell />);

      await act(async () => {
        runtime().publishMessageRegionSnapshot?.({
          route: "chat",
          hasMessages: true,
          sessionId: "session-runtime-chat",
          messages: Array.from({ length: 5 }, (_value, index) => ({
            id: `msg-runtime-chat-${index + 1}`,
            role: index % 2 === 0 ? "user" : "assistant",
            text: `Runtime chat message ${index + 1}`,
            status: "done",
            error: false,
            at: Date.UTC(2026, 3, 15, 14, index, 0),
            source: "model",
          })),
        });
      });

      const messageArea = document.getElementById(LEGACY_SHELL_IDS.messageArea) as HTMLElement;
      const messages = [...messageArea.querySelectorAll<HTMLElement>("[data-message-id]")];
      applyScrollableMetrics(messageArea, messages, [0, 120, 280, 440, 600], {
        clientHeight: 260,
        scrollHeight: 820,
      });

      messageArea.scrollTop = 280;
      fireEvent.scroll(messageArea);

      await waitFor(() => {
        expect(messages).toHaveLength(5);
      });

      expect(document.querySelector("[data-scroll-jump-scope='agent']")).toBeNull();
      expect(document.querySelector("[data-scroll-jump-top='chat']")).toBeNull();
    } finally {
      rafSpy.mockRestore();
      cancelSpy.mockRestore();
    }
  });

  it("does not render route-level jump controls on page-mode routes", async () => {
    await renderChannelsRouteShell();

    expect(document.querySelector("[data-scroll-jump-scope='route']")).toBeNull();
    expect(document.querySelector("[data-scroll-jump-top='route']")).toBeNull();
  });

  it("renders structured assistant process snapshots with final answer", async () => {
    render(<LegacyWebShell />);

    await act(async () => {
      runtime().publishMessageRegionSnapshot?.({
        route: "chat",
        hasMessages: true,
        sessionId: "session-runtime-process",
        messages: [
          {
            id: "msg-runtime-process",
            role: "assistant",
            text: "Runtime final answer",
            source: "codex_cli",
            status: "done",
            error: false,
            at: Date.UTC(2026, 3, 15, 13, 0, 0),
            process_steps: [
              {
                id: "step-1",
                title: "Inspect repository",
                detail: "Checked the current shell layout.",
                status: "success",
              },
              {
                id: "step-2",
                title: "Apply patch",
                detail: "Moved message rendering into React.",
                status: "success",
              },
            ],
          },
        ],
      });
    });

    expect(screen.getByRole("button", { name: /Process/i })).toBeInTheDocument();
    expect(screen.getByText("Inspect repository")).toBeInTheDocument();
    expect(screen.getByText("Moved message rendering into React.")).toBeInTheDocument();
    expect(screen.getByText("Runtime final answer")).toBeInTheDocument();
    expect(screen.getByText("CODEX CLI")).toBeInTheDocument();
  });

  it("keeps runtime chat workspace snapshot copy across shell rerenders", async () => {
    render(<LegacyWebShell />);

    const messageArea = document.getElementById(LEGACY_SHELL_IDS.messageArea);
    messageArea?.insertAdjacentHTML("beforeend", '<div data-runtime-node="message">runtime message</div>');

    await act(async () => {
      runtime().publishChatWorkspaceSnapshot?.({
        route: "chat",
        heading: "Runtime heading",
        subheading: "Runtime subheading",
        welcomeHeading: "Runtime welcome heading",
        welcomeDescription: "Runtime welcome description",
        welcomeTargets: [
          {
            type: "agent",
            id: "agent-runtime",
            name: "Runtime Agent",
            active: true,
            interactive: true,
          },
        ],
      });
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
      expect(screen.getByRole("button", { name: "Runtime Agent agent-runtime" })).toHaveAttribute(
        "data-chat-target-id",
        "agent-runtime",
      );
      expect(screen.getByText("runtime message")).toBeInTheDocument();
    });
  });

  it("renders structured welcome target picker cards and errors from runtime snapshots", async () => {
    window.location.hash = "#agent-runtime";

    render(<LegacyWebShell />);

    await act(async () => {
      runtime().publishChatWorkspaceSnapshot?.({
        route: "agent-runtime",
        heading: "Agent runtime",
        subheading: "Runtime subheading",
        welcomeHeading: "Choose target",
        welcomeDescription: "Pick an agent to continue.",
        welcomeTargets: [
          {
            type: "agent",
            id: "planner",
            name: "Planner",
            active: true,
            interactive: true,
          },
          {
            type: "agent",
            id: "reviewer",
            name: "Reviewer",
            active: false,
            interactive: true,
          },
          {
            type: "agent",
            id: "pinned-agent",
            name: "Pinned Agent",
            active: false,
            interactive: false,
          },
        ],
        welcomeTargetError: "Catalog unavailable",
      });
    });

    expect(screen.getByRole("button", { name: "Planner planner" })).toHaveClass("active");
    expect(screen.getByRole("button", { name: "Reviewer reviewer" })).toHaveAttribute(
      "data-chat-target-name",
      "Reviewer",
    );
    expect(screen.getByText("Pinned Agent")).toBeInTheDocument();
    expect(document.querySelector(".welcome-target-card.is-static")).toHaveTextContent("Pinned Agent");
    expect(screen.getByText("Catalog unavailable")).toBeInTheDocument();
  });

  it("keeps runtime panel snapshot content across shell rerenders", async () => {
    render(<LegacyWebShell />);

    await act(async () => {
      runtime().publishChatRuntimeSnapshot?.(
        runtimeSnapshot({
          openPopover: "model",
          note: "Runtime provider warning",
          providers: [
            {
              id: "openai",
              name: "OpenAI",
              models: [
                {
                  id: "gpt-5.4",
                  name: "GPT-5.4",
                  active: true,
                },
                {
                  id: "gpt-5.4-mini",
                  name: "GPT-5.4 Mini",
                  active: false,
                },
              ],
            },
          ],
        }),
      );
    });

    const runtimePanel = document.getElementById(LEGACY_SHELL_IDS.chatRuntimePanel);
    const runtimeControls = document.querySelector('[data-runtime-controls-root=""]');
    const runtimeNote = document.querySelector('[data-runtime-note-root=""]');

    expect(runtimePanel).toBeInTheDocument();
    expect(runtimeControls).toBeInTheDocument();
    expect(runtimeNote).toBeInTheDocument();
    expect(within(runtimeControls as HTMLElement).getByText("Model · GPT-5.4")).toBeInTheDocument();
    expect(runtimeNote).toHaveTextContent("Runtime provider warning");
    expect(within(runtimeControls as HTMLElement).getByText("GPT-5.4 Mini")).toBeInTheDocument();

    await act(async () => {
      document.documentElement.lang = "zh-CN";
    });

    await waitFor(() => {
      expect(within(runtimeControls as HTMLElement).getByText("模型 · GPT-5.4")).toBeInTheDocument();
      expect(runtimeNote).toHaveTextContent("Runtime provider warning");
      expect(within(runtimeControls as HTMLElement).getByText("GPT-5.4 Mini")).toBeInTheDocument();
    });
  });

  it("renders structured runtime sheet content in the detached sheet host", async () => {
    render(<LegacyWebShell />);

    await act(async () => {
      runtime().publishChatRuntimeSnapshot?.(
        runtimeSnapshot({
          compact: true,
          openPopover: "mobile",
          note: "Runtime mobile hint",
          capabilities: [
            {
              id: "memory",
              name: "Memory",
              description: "Search memory files",
              kind: "tool",
              active: true,
            },
          ],
          skills: [
            {
              id: "runtime-skill",
              name: "Runtime skill",
              description: "Enabled in runtime",
              kind: "skill",
              active: true,
            },
          ],
        }),
      );
    });

    const sheetHost = document.getElementById(LEGACY_SHELL_IDS.chatRuntimeSheetHost);

    await waitFor(() => {
      expect(sheetHost?.querySelector(".composer-runtime-sheet-backdrop")).toBeInTheDocument();
      expect(sheetHost?.querySelector(".composer-runtime-popover-mobile")).toBeInTheDocument();
      expect(sheetHost?.querySelector('[data-runtime-scroll-container="mobile"]')).toBeInTheDocument();
      expect(within(sheetHost as HTMLElement).getByText("Runtime skill")).toBeInTheDocument();
      expect(within(sheetHost as HTMLElement).getByText("Memory")).toBeInTheDocument();
    });
  });

  it("dispatches runtime control actions through the shared runtime bridge", async () => {
    render(<LegacyWebShell />);

    const toggleChatRuntimePopover = vi.fn(() => true);
    const selectChatRuntimeModel = vi.fn(() => true);
    const toggleChatRuntimeItem = vi.fn(() => true);
    const closeChatRuntimePopover = vi.fn(() => true);

    Object.assign(runtime(), {
      toggleChatRuntimePopover,
      selectChatRuntimeModel,
      toggleChatRuntimeItem,
      closeChatRuntimePopover,
    });

    await act(async () => {
      runtime().publishChatRuntimeSnapshot?.(
        runtimeSnapshot({
          openPopover: "model",
          compact: false,
          providers: [
            {
              id: "openai",
              name: "OpenAI",
              models: [
                { id: "gpt-5.4", name: "GPT-5.4", active: true },
                { id: "gpt-5.4-mini", name: "GPT-5.4 Mini", active: false },
              ],
            },
          ],
          capabilities: [
            {
              id: "memory",
              name: "Memory",
              description: "Search memory files",
              kind: "tool",
              active: true,
            },
          ],
        }),
      );
    });

    fireEvent.click(document.querySelector('[data-runtime-toggle="model"]') as HTMLElement);
    fireEvent.click(screen.getByRole("button", { name: "GPT-5.4 Mini OpenAI" }));

    await act(async () => {
      runtime().publishChatRuntimeSnapshot?.(
        runtimeSnapshot({
          openPopover: "capabilities",
          capabilities: [
            {
              id: "memory",
              name: "Memory",
              description: "Search memory files",
              kind: "tool",
              active: true,
            },
          ],
        }),
      );
    });

    fireEvent.click(screen.getByRole("checkbox", { name: "Memory Search memory files" }));

    await act(async () => {
      runtime().publishChatRuntimeSnapshot?.(
        runtimeSnapshot({
          compact: true,
          openPopover: "mobile",
        }),
      );
    });

    fireEvent.click(document.querySelector(".composer-runtime-sheet-backdrop") as HTMLElement);

    expect(toggleChatRuntimePopover).toHaveBeenCalledWith("model");
    expect(selectChatRuntimeModel).toHaveBeenCalledWith({
      providerId: "openai",
      modelId: "gpt-5.4-mini",
    });
    expect(toggleChatRuntimeItem).toHaveBeenCalledWith({
      group: "capabilities",
      id: "memory",
      checked: false,
      kind: "tool",
    });
    expect(closeChatRuntimePopover).toHaveBeenCalledTimes(1);
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

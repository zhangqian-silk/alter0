import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { LegacyWebShell } from "./LegacyWebShell";
import { LEGACY_SHELL_IDS } from "./legacyDomContract";

describe("LegacyWebShell", () => {
  beforeEach(() => {
    window.location.hash = "";
    document.documentElement.lang = "en";
    window.sessionStorage.clear();
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
    expect(document.querySelector("[data-runtime-controls-root]")).toBeInTheDocument();
    expect(document.querySelector("[data-runtime-note-root]")).toBeInTheDocument();
    expect(document.querySelector("[data-runtime-sheet-root]")).toBeInTheDocument();
  });

  it("renders the workspace navigation and composer entrypoints", () => {
    render(<LegacyWebShell />);

    expect(screen.getByRole("button", { name: "Chat" })).toHaveAttribute("data-route", "chat");
    expect(screen.getByRole("button", { name: "Agent" })).toHaveAttribute("data-route", "agent-runtime");
    expect(screen.getByRole("button", { name: "Terminal" })).toHaveAttribute("data-route", "terminal");
    expect(screen.getByRole("button", { name: "Send message" })).toHaveAttribute("data-composer-submit", "chat-main");
    expect(screen.getByLabelText("Input your message")).toHaveAttribute("data-composer-input", "chat-main");
  });

  it("renders the shell navigation groups and route placement", () => {
    const { container } = render(<LegacyWebShell />);

    expect(screen.getByText("Workspace")).toBeInTheDocument();
    expect(screen.getByText("Agent Studio")).toBeInTheDocument();
    expect(screen.getByText("Control")).toBeInTheDocument();
    expect(screen.getByText("Settings")).toBeInTheDocument();

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
    render(<LegacyWebShell />);

    const runtimeMount = document.getElementById(LEGACY_SHELL_IDS.sessionList);
    runtimeMount?.insertAdjacentHTML("beforeend", '<button type="button" data-runtime-node="session">runtime session</button>');

    fireEvent.click(screen.getByRole("button", { name: "Collapse navigation" }));

    expect(document.getElementById(LEGACY_SHELL_IDS.appShell)).toHaveClass("nav-collapsed");
    expect(screen.getByRole("button", { name: "Expand navigation" })).toHaveAttribute("aria-expanded", "false");
    expect(screen.getByText("runtime session")).toBeInTheDocument();
  });

  it("follows document language changes for the primary navigation copy", async () => {
    render(<LegacyWebShell />);

    await act(async () => {
      document.documentElement.lang = "zh-CN";
    });

    await waitFor(() => {
      expect(screen.getByText("工作区")).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "终端代理" })).toHaveAttribute("data-route", "terminal");
      expect(screen.getByRole("button", { name: "语言" })).toHaveTextContent("中文");
    });
  });

  it("toggles the session history collapse state without clearing runtime session mounts", () => {
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

  it("does not overwrite legacy-managed chat headings during shell rerenders", async () => {
    render(<LegacyWebShell />);

    const heading = document.getElementById("sessionHeading");
    const subheading = document.getElementById("sessionSubheading");
    heading!.textContent = "Runtime heading";
    subheading!.textContent = "Runtime subheading";

    await act(async () => {
      window.location.hash = "#terminal";
      window.dispatchEvent(new HashChangeEvent("hashchange"));
      document.documentElement.lang = "zh-CN";
    });

    await waitFor(() => {
      expect(heading).toHaveTextContent("Runtime heading");
      expect(subheading).toHaveTextContent("Runtime subheading");
    });
  });
});

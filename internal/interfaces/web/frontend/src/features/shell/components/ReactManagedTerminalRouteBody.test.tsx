import { act, cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { useState } from "react";
import {
  ReactManagedTerminalRouteBody,
  resetTerminalRuntimeCache,
  resolveTerminalPollPlan,
  TERMINAL_RUNTIME_CACHE_SESSION_TTL_MS,
} from "./ReactManagedTerminalRouteBody";
import { WorkbenchContext, type WorkbenchContextValue } from "../../../app/WorkbenchContext";
import { hashSessionIDShort } from "../../../shared/session/sessionHash";
import type { RuntimeTraceEvent } from "./runtimeTraceEvents";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: init.status ?? 200,
    headers: {
      "Content-Type": "application/json",
      ...(init.headers ?? {}),
    },
  });
}

type TerminalTurnFixture = {
  id: string;
  prompt?: string;
  final_output?: string;
};

function terminalRuntimeEventFixture(overrides: Partial<RuntimeTraceEvent> = {}): RuntimeTraceEvent {
  const kind = overrides.kind || "shell_command";
  return {
    id: "step-1",
    turn_id: "turn-1",
    seq: 1,
    source: "adapter",
    provider: { engine: "codex", adapter: "codex_cli_json", event_type: "command", item_id: "step-1" },
    role: "assistant",
    kind,
    lifecycle: "completed",
    status: "completed",
    title: "Inspect workspace",
    summary: "pwd",
    blocks: [],
    action: kind === "shell_command" ? { family: "shell", name: "shell" } : undefined,
    visibility: "collapsed",
    duration_ms: 1000,
    raw: { ref: "step-1", type: "command", has_detail: true },
    ...overrides,
  };
}

function installImmediateAnimationFrame() {
  vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback: FrameRequestCallback) => {
    window.setTimeout(() => callback(16), 0);
    return 1;
  });
  vi.spyOn(window, "cancelAnimationFrame").mockImplementation(() => undefined);
}

function stubTerminalTurnsFetch(turns: TerminalTurnFixture[], shell = "codex exec") {
  vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    const method = String(init?.method || "GET").toUpperCase();
    if (url === "/api/terminal/sessions" && method === "GET") {
      return Promise.resolve(jsonResponse({
        items: [
          {
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell,
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
          },
        ],
      }));
    }
    if (url === "/api/control/skills" && method === "GET") {
      return Promise.resolve(jsonResponse({ items: [] }));
    }
    if (url === "/api/terminal/sessions/terminal-1" && method === "GET") {
      return Promise.resolve(jsonResponse({
        session: {
          id: "terminal-1",
          title: "Workspace shell",
          terminal_session_id: "terminal-1",
          status: "ready",
          shell,
          working_dir: "/workspace/alter0",
          created_at: "2026-04-15T10:00:00Z",
          updated_at: "2026-04-15T10:10:00Z",
          turns: turns.map((turn, index) => ({
            id: turn.id,
            prompt: turn.prompt || `prompt-${index + 1}`,
            status: "completed",
            started_at: `2026-04-15T10:0${index}:00Z`,
            finished_at: `2026-04-15T10:0${index}:02Z`,
            duration_ms: 2000,
            final_output: turn.final_output || `output-${index + 1}`,
            runtime_trace_events: [],
          })),
        },
      }));
    }
    return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
  }));
}

function applyTerminalTurnMetrics(
  chatScreen: HTMLDivElement,
  layouts: Array<{ id: string; top: number; height: number }>,
  options: {
    clientHeight: number;
    scrollHeight: number;
    scrollTop: number;
  },
) {
  Object.defineProperty(chatScreen, "clientHeight", {
    configurable: true,
    value: options.clientHeight,
  });
  Object.defineProperty(chatScreen, "scrollHeight", {
    configurable: true,
    value: options.scrollHeight,
  });
  Object.defineProperty(chatScreen, "scrollTop", {
    configurable: true,
    writable: true,
    value: options.scrollTop,
  });
  chatScreen.scrollTo = vi.fn((value?: ScrollToOptions | number, y?: number) => {
    const top = typeof value === "number" ? y : value?.top;
    chatScreen.scrollTop = Number(top || 0);
  }) as HTMLElement["scrollTo"];

  layouts.forEach((layout) => {
    const node = document.querySelector(`[data-terminal-turn="${layout.id}"]`) as HTMLElement | null;
    if (!node) {
      return;
    }
    Object.defineProperty(node, "offsetTop", {
      configurable: true,
      get: () => layout.top,
    });
    Object.defineProperty(node, "offsetHeight", {
      configurable: true,
      get: () => layout.height,
    });
  });
}

function openTerminalSessionActions(sessionID: string): HTMLElement {
  const card = document.querySelector(`[data-runtime-session-card='${sessionID}']`) as HTMLElement;
  fireEvent.click(within(card).getByRole("button", { name: "Session actions", hidden: true }));
  return card;
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (error: unknown) => void;
  const promise = new Promise<T>((promiseResolve, promiseReject) => {
    resolve = promiseResolve;
    reject = promiseReject;
  });
  return { promise, resolve, reject };
}

function terminalSessionFixture(overrides: Record<string, unknown> = {}) {
  return {
    id: "terminal-cache",
    title: "Cached shell",
    terminal_session_id: "terminal-cache",
    status: "ready",
    shell: "codex exec",
    working_dir: "/workspace/alter0",
    created_at: "2026-04-15T10:00:00Z",
    updated_at: "2026-04-15T10:10:00Z",
    ...overrides,
  };
}

function terminalTurnFixtures(count: number, outputPrefix = "cached output"): TerminalTurnFixture[] {
  return Array.from({ length: count }, (_, index) => {
    const value = index + 1;
    return {
      id: `turn-${value}`,
      prompt: `prompt-${value}`,
      final_output: `${outputPrefix} ${value}`,
    };
  });
}

describe("ReactManagedTerminalRouteBody", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return Promise.resolve(jsonResponse({
          items: [
            {
              id: "terminal-1",
              title: "Workspace shell",
              terminal_session_id: "terminal-1",
              status: "ready",
              shell: "codex exec",
              working_dir: "/workspace/alter0",
              created_at: "2026-04-15T10:00:00Z",
              updated_at: "2026-04-15T10:10:00Z",
            },
          ],
        }));
      }
      if (url === "/api/control/skills" && method === "GET") {
        return Promise.resolve(jsonResponse({
          items: [
            {
              id: "preview-publish",
              name: "Preview Publish",
              enabled: true,
              metadata: {
                "skill.description": "Publish the current session to the shared preview gateway.",
              },
            },
            {
              id: "frontend-design",
              name: "Frontend Design",
              enabled: true,
              metadata: {
                "skill.description": "Apply production-grade frontend design rules.",
              },
            },
            {
              id: "summary",
              name: "Summary",
              enabled: true,
              metadata: {
                "skill.description": "Summarize terminal work.",
              },
            },
            {
              id: "memory",
              name: "Memory",
              enabled: true,
              metadata: {
                "skill.description": "Memory routing rules.",
              },
            },
            {
              id: "private",
              name: "Private",
              enabled: true,
              metadata: {
                "alter0.skill.visibility": "private",
              },
            },
          ],
        }));
      }
      if (url === "/api/terminal/sessions/terminal-1" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
            turns: [
              {
                id: "turn-1",
                prompt: "pwd",
                status: "completed",
                started_at: "2026-04-15T10:05:00Z",
                finished_at: "2026-04-15T10:05:02Z",
                duration_ms: 2000,
                final_output: [
                  "# Workspace",
                  "",
                  "- /workspace/alter0",
                  "- ready",
                  "",
                  "Use `pwd` to inspect the repo.",
                ].join("\n"),
                runtime_trace_events: [
                  terminalRuntimeEventFixture({ title: "Inspect workspace", summary: "pwd" }),
                ],
              },
            ],
          },
        }));
      }
      if (url === "/api/terminal/sessions/terminal-1/turns/turn-1/events/step-1" && method === "GET") {
        return Promise.resolve(jsonResponse({
          event: {
            turn_id: "turn-1",
            blocks: [
              {
                type: "terminal",
                title: "Shell",
                command: "pwd",
                output: "/workspace/alter0",
                language: "shell",
              },
            ],
          },
        }));
      }
      if (url === "/api/terminal/sessions" && method === "POST") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-2",
            title: "terminal-2",
            terminal_session_id: "terminal-2",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0/.alter0/workspaces/terminal/sessions/terminal-2",
            created_at: "2026-04-15T10:20:00Z",
            updated_at: "2026-04-15T10:20:00Z",
          },
        }, { status: 201 }));
      }
      if (url === "/api/terminal/sessions/terminal-2" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-2",
            title: "terminal-2",
            terminal_session_id: "terminal-2",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0/.alter0/workspaces/terminal/sessions/terminal-2",
            created_at: "2026-04-15T10:20:00Z",
            updated_at: "2026-04-15T10:20:00Z",
            turns: [],
          },
        }));
      }
      if (url === "/api/terminal/sessions/terminal-1/input" && method === "POST") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "busy",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:30Z",
          },
        }));
      }
      if (url === "/api/terminal/sessions/terminal-1/attachments" && method === "POST") {
        return Promise.resolve(jsonResponse({
          items: [
            {
              id: "asset-terminal-1",
              name: "terminal-shot.svg",
              content_type: "image/svg+xml",
              size: 32,
              asset_url: "/api/terminal/sessions/terminal-1/attachments/asset-terminal-1/original",
              preview_url: "/api/terminal/sessions/terminal-1/attachments/asset-terminal-1/preview",
            },
          ],
        }));
      }
      if (url === "/api/terminal/sessions/terminal-2/attachments" && method === "POST") {
        return Promise.resolve(jsonResponse({
          items: [
            {
              id: "asset-terminal-2",
              name: "diagram.svg",
              content_type: "image/svg+xml",
              size: 32,
              asset_url: "/api/terminal/sessions/terminal-2/attachments/asset-terminal-2/original",
              preview_url: "/api/terminal/sessions/terminal-2/attachments/asset-terminal-2/preview",
            },
          ],
        }));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));
  });

  afterEach(() => {
    cleanup();
    resetTerminalRuntimeCache();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
    vi.useRealTimers();
    window.localStorage.clear();
    window.history.replaceState({}, "", "/");
  });

  function renderTerminalRouteBody(overrides: Partial<WorkbenchContextValue> = {}) {
    const baseContextValue: WorkbenchContextValue = {
      route: "terminal",
      language: "en",
      navigate: vi.fn(),
      isMobileViewport: false,
      mobileNavOpen: false,
      mobileSessionPaneOpen: false,
      toggleMobileNav: vi.fn(),
      toggleMobileSessionPane: vi.fn(),
      openMobileSessionPane: vi.fn(),
      closeMobileNav: vi.fn(),
      closeMobileSessionPane: vi.fn(),
      ...overrides,
    };

    function TerminalRouteBodyHarness() {
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
        openMobileSessionPane: () => {
          baseContextValue.openMobileSessionPane();
          setMobilePanel("sessions");
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
          <ReactManagedTerminalRouteBody />
        </WorkbenchContext.Provider>
      );
    }

    return render(
      <TerminalRouteBodyHarness />,
    );
  }

  it("prefers the terminal session query parameter on load and syncs short hashes after session switches", async () => {
    window.history.replaceState({}, "", "/terminal?session_id=terminal-2");
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return Promise.resolve(jsonResponse({
          items: [
            {
              id: "terminal-1",
              title: "Workspace shell",
              terminal_session_id: "terminal-1",
              status: "ready",
              shell: "codex exec",
              working_dir: "/workspace/alter0",
              created_at: "2026-04-15T10:00:00Z",
              updated_at: "2026-04-15T10:10:00Z",
            },
            {
              id: "terminal-2",
              title: "Review shell",
              terminal_session_id: "terminal-2",
              status: "ready",
              shell: "codex exec",
              working_dir: "/workspace/alter0/review",
              created_at: "2026-04-15T11:00:00Z",
              updated_at: "2026-04-15T11:10:00Z",
            },
          ],
        }));
      }
      if (url === "/api/control/skills" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/terminal/sessions/terminal-1" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
            turns: [],
          },
        }));
      }
      if (url === "/api/terminal/sessions/terminal-2" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-2",
            title: "Review shell",
            terminal_session_id: "terminal-2",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0/review",
            created_at: "2026-04-15T11:00:00Z",
            updated_at: "2026-04-15T11:10:00Z",
            turns: [],
          },
        }));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    const view = renderTerminalRouteBody();

    await waitFor(() => {
      expect(view.container.querySelector('[data-runtime-session-select="terminal-2"]')).toHaveClass("active");
    });
    expect(window.location.search).toContain(`session_id=${hashSessionIDShort("terminal-2")}`);
    expect(window.location.search).not.toContain("session_id=terminal-2");

    fireEvent.click(view.container.querySelector('[data-runtime-session-select="terminal-1"]') as HTMLElement);

    await waitFor(() => {
      expect(view.container.querySelector('[data-runtime-session-select="terminal-1"]')).toHaveClass("active");
    });
    expect(window.location.search).toContain(`session_id=${hashSessionIDShort("terminal-1")}`);
    expect(window.location.search).not.toContain("session_id=terminal-1");
  });

  it("restores a terminal session from its short hash query parameter", async () => {
    window.history.replaceState({}, "", `/terminal?session_id=${hashSessionIDShort("terminal-2")}`);
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return Promise.resolve(jsonResponse({
          items: [
            {
              id: "terminal-1",
              title: "Workspace shell",
              terminal_session_id: "terminal-1",
              status: "ready",
              shell: "codex exec",
              working_dir: "/workspace/alter0",
              created_at: "2026-04-15T10:00:00Z",
              updated_at: "2026-04-15T10:10:00Z",
            },
            {
              id: "terminal-2",
              title: "Review shell",
              terminal_session_id: "terminal-2",
              status: "ready",
              shell: "codex exec",
              working_dir: "/workspace/alter0/review",
              created_at: "2026-04-15T11:00:00Z",
              updated_at: "2026-04-15T11:10:00Z",
            },
          ],
        }));
      }
      if (url === "/api/control/skills" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/terminal/sessions/terminal-2" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-2",
            title: "Review shell",
            terminal_session_id: "terminal-2",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0/review",
            created_at: "2026-04-15T11:00:00Z",
            updated_at: "2026-04-15T11:10:00Z",
            turns: [],
          },
        }));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    const view = renderTerminalRouteBody();

    await waitFor(() => {
      expect(view.container.querySelector('[data-runtime-session-select="terminal-2"]')).toHaveClass("active");
    });
    expect(window.location.search).toContain(`session_id=${hashSessionIDShort("terminal-2")}`);
  });

  it("adapts terminal polling cadence to runtime status and interaction state", () => {
    expect(
      resolveTerminalPollPlan({
        status: "ready",
        pageHidden: false,
        scrollingActive: false,
        inputFocused: false,
      }),
    ).toEqual({
      enabled: false,
      interval: 0,
      refreshActiveSession: false,
    });

    expect(
      resolveTerminalPollPlan({
        status: "ready",
        pageHidden: true,
        scrollingActive: false,
        inputFocused: false,
      }),
    ).toEqual({
      enabled: false,
      interval: 0,
      refreshActiveSession: false,
    });

    expect(
      resolveTerminalPollPlan({
        status: "busy",
        pageHidden: false,
        scrollingActive: false,
        inputFocused: false,
      }),
    ).toEqual({
      enabled: true,
      interval: 2000,
      refreshActiveSession: true,
    });

    expect(
      resolveTerminalPollPlan({
        status: "busy",
        pageHidden: true,
        scrollingActive: false,
        inputFocused: false,
      }),
    ).toEqual({
      enabled: true,
      interval: 12000,
      refreshActiveSession: true,
    });

    expect(
      resolveTerminalPollPlan({
        status: "busy",
        pageHidden: false,
        scrollingActive: true,
        inputFocused: false,
      }),
    ).toEqual({
      enabled: false,
      interval: 0,
      refreshActiveSession: true,
    });
  });

  it("keeps the Terminal runtime cache alive for longer single-device route gaps", () => {
    expect(TERMINAL_RUNTIME_CACHE_SESSION_TTL_MS).toBe(24 * 60 * 60 * 1000);
  });

  it("hydrates a fresh Terminal runtime cache immediately and refreshes after the API returns", async () => {
    const cachedTurnCount = 8;
    const cachedTurns = terminalTurnFixtures(cachedTurnCount);
    const cachedTurnPayloads = cachedTurns.map((turn, index) => ({
      id: turn.id,
      prompt: turn.prompt,
      status: "completed",
      started_at: `2026-04-15T10:${String(index + 1).padStart(2, "0")}:00Z`,
      finished_at: `2026-04-15T10:${String(index + 1).padStart(2, "0")}:02Z`,
      duration_ms: 2000,
      final_output: turn.final_output,
      runtime_trace_events: [],
    }));

    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return Promise.resolve(jsonResponse({
          items: [terminalSessionFixture()],
        }));
      }
      if (url === "/api/control/skills" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/terminal/sessions/terminal-cache" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: terminalSessionFixture({ turns: cachedTurnPayloads }),
        }));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    const firstView = renderTerminalRouteBody();

    await waitFor(() => {
      expect(screen.getByText(`cached output ${cachedTurnCount}`)).toBeInTheDocument();
    });
    expect(screen.getByText("cached output 1")).toBeInTheDocument();
    firstView.unmount();

    const listRequest = deferred<{ items?: unknown[] }>();
    const sessionRequest = deferred<{ session?: unknown }>();
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return listRequest.promise.then((payload) => jsonResponse(payload));
      }
      if (url === "/api/control/skills" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/terminal/sessions/terminal-cache" && method === "GET") {
        return sessionRequest.promise.then((payload) => jsonResponse(payload));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    renderTerminalRouteBody();

    expect(screen.getByRole("heading", { name: "Cached shell" })).toBeInTheDocument();
    expect(screen.getByText(`cached output ${cachedTurnCount}`)).toBeInTheDocument();
    expect(screen.getByText("cached output 1")).toBeInTheDocument();

    listRequest.resolve({
      items: [terminalSessionFixture({
        title: "Server shell",
        updated_at: "2026-04-15T10:20:00Z",
      })],
    });
    sessionRequest.resolve({
      session: terminalSessionFixture({
        title: "Server shell",
        updated_at: "2026-04-15T10:20:00Z",
        turns: [{
          id: "turn-server",
          prompt: "server prompt",
          status: "completed",
          started_at: "2026-04-15T10:20:00Z",
          finished_at: "2026-04-15T10:20:02Z",
          duration_ms: 2000,
          final_output: "server output",
          runtime_trace_events: [],
        }],
      }),
    });

    await waitFor(() => {
      expect(screen.getByText("server output")).toBeInTheDocument();
    });
    expect(screen.getByRole("heading", { name: "Server shell" })).toBeInTheDocument();
  });

  it("hydrates Terminal sessions from the 24 hour localStorage runtime snapshot", async () => {
    const nowSpy = vi.spyOn(Date, "now").mockReturnValue(1_000_000);
    const cachedTurns = terminalTurnFixtures(2, "stored output").map((turn, index) => ({
      id: turn.id,
      prompt: turn.prompt,
      status: "completed",
      started_at: `2026-04-15T10:${String(index + 1).padStart(2, "0")}:00Z`,
      finished_at: `2026-04-15T10:${String(index + 1).padStart(2, "0")}:02Z`,
      duration_ms: 2000,
      final_output: turn.final_output,
      runtime_trace_events: [],
    }));
    window.localStorage.setItem("alter0.web.terminal.runtime_snapshot.v1", JSON.stringify({
      cachedAt: 1_000_000,
      activeSessionID: "terminal-cache",
      sessions: [terminalSessionFixture({ turns: cachedTurns })],
    }));

    const listRequest = deferred<{ items?: unknown[] }>();
    const sessionRequest = deferred<{ session?: unknown }>();
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return listRequest.promise.then((payload) => jsonResponse(payload));
      }
      if (url === "/api/control/skills" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/terminal/sessions/terminal-cache" && method === "GET") {
        return sessionRequest.promise.then((payload) => jsonResponse(payload));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    renderTerminalRouteBody();

    expect(screen.getByRole("heading", { name: "Cached shell" })).toBeInTheDocument();
    expect(screen.getByText("stored output 1")).toBeInTheDocument();
    expect(screen.getByText("stored output 2")).toBeInTheDocument();

    listRequest.resolve({ items: [terminalSessionFixture({ title: "Server shell" })] });
    sessionRequest.resolve({
      session: terminalSessionFixture({
        title: "Server shell",
        turns: [{
          id: "turn-server",
          prompt: "server prompt",
          status: "completed",
          started_at: "2026-04-15T10:20:00Z",
          finished_at: "2026-04-15T10:20:02Z",
          duration_ms: 2000,
          final_output: "server output",
          runtime_trace_events: [],
        }],
      }),
    });

    await waitFor(() => {
      expect(screen.getByText("server output")).toBeInTheDocument();
    });
    nowSpy.mockRestore();
  });

  it("does not hydrate an expired Terminal runtime cache", async () => {
    const nowSpy = vi.spyOn(Date, "now").mockReturnValue(1000);
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return Promise.resolve(jsonResponse({
          items: [terminalSessionFixture()],
        }));
      }
      if (url === "/api/control/skills" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/terminal/sessions/terminal-cache" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: terminalSessionFixture({
            turns: [{
              id: "turn-cache",
              prompt: "cached prompt",
              status: "completed",
              started_at: "2026-04-15T10:00:00Z",
              finished_at: "2026-04-15T10:00:02Z",
              duration_ms: 2000,
              final_output: "cached output before expiry",
              runtime_trace_events: [],
            }],
          }),
        }));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    const firstView = renderTerminalRouteBody();

    await waitFor(() => {
      expect(screen.getByText("cached output before expiry")).toBeInTheDocument();
    });
    firstView.unmount();

    nowSpy.mockReturnValue(1000 + TERMINAL_RUNTIME_CACHE_SESSION_TTL_MS + 1);
    const listRequest = deferred<{ items?: unknown[] }>();
    const sessionRequest = deferred<{ session?: unknown }>();
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return listRequest.promise.then((payload) => jsonResponse(payload));
      }
      if (url === "/api/control/skills" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/terminal/sessions/terminal-cache" && method === "GET") {
        return sessionRequest.promise.then((payload) => jsonResponse(payload));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    renderTerminalRouteBody();

    expect(screen.queryByText("cached output before expiry")).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Cached shell" })).not.toBeInTheDocument();

    listRequest.resolve({
      items: [terminalSessionFixture({ title: "Server shell after expiry" })],
    });
    sessionRequest.resolve({
      session: terminalSessionFixture({
        title: "Server shell after expiry",
        turns: [{
          id: "turn-server",
          prompt: "server prompt",
          status: "completed",
          started_at: "2026-04-15T10:20:00Z",
          finished_at: "2026-04-15T10:20:02Z",
          duration_ms: 2000,
          final_output: "server output after expiry",
          runtime_trace_events: [],
        }],
      }),
    });

    await waitFor(() => {
      expect(screen.getByText("server output after expiry")).toBeInTheDocument();
    });
  });

  it("renders the terminal session list and active workspace in React", async () => {
    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-session-select='terminal-1']")).toBeInTheDocument();
    });

    expect(document.querySelector("[data-runtime-view='terminal']")).toBeInTheDocument();
    expect(document.querySelector("[data-runtime-session-pane='terminal']")).toBeInTheDocument();
    expect(document.querySelector("[data-runtime-workspace='terminal']")).toHaveAttribute(
      "data-runtime-session-id",
      "terminal-1",
    );
    await waitFor(() => {
      expect(document.querySelector("[data-terminal-turn='turn-1']")).toBeInTheDocument();
    });
    expect(document.querySelector("[data-terminal-turn='turn-1']")).toBeInTheDocument();
    const terminalTurn = document.querySelector("[data-terminal-turn='turn-1']") as HTMLElement;
    const terminalPrompt = terminalTurn.querySelector(".terminal-turn-prompt") as HTMLElement;
    const terminalFinal = terminalTurn.querySelector("[data-terminal-final-output='turn-1']") as HTMLElement;
    expect(terminalPrompt).toHaveClass("runtime-message", "runtime-message-user");
    expect(terminalPrompt.querySelector(".runtime-message-bubble")).toBeInTheDocument();
    expect(terminalPrompt.querySelector(".runtime-message-user-shell")).toBeInTheDocument();
    expect(terminalFinal).toHaveClass("runtime-message", "runtime-message-assistant");
    expect(terminalFinal).not.toHaveClass("msg", "assistant");
    expect(terminalFinal.querySelector(".runtime-message-bubble")).toBeInTheDocument();
    expect(terminalFinal.querySelector(".runtime-message-bubble")).not.toHaveClass("msg-bubble");
    expect(terminalFinal.querySelector(".runtime-message-assistant-shell")).toBeInTheDocument();
    expect(terminalTurn.querySelector(".terminal-log-time")).not.toBeInTheDocument();
    expect(document.querySelector("[data-runtime-view='terminal']")).toHaveClass("runtime-workspace-view");
    expect(document.querySelector("[data-runtime-workspace-page='true']")).toBeInTheDocument();
    expect(document.querySelector(".runtime-session-select")).toBeInTheDocument();
    const sessionActionCard = openTerminalSessionActions("terminal-1");
    expect(sessionActionCard.querySelector(".runtime-session-more")).toBeInTheDocument();
    expect(within(sessionActionCard).getByRole("menuitem", { name: "Pin session", hidden: true })).toBeInTheDocument();
    expect(within(sessionActionCard).getByRole("menuitem", { name: "Details", hidden: true })).toBeInTheDocument();
    expect(within(sessionActionCard).getByRole("menuitem", { name: "Delete", hidden: true })).toBeInTheDocument();
    expect(document.querySelector(".runtime-session-card")).not.toHaveClass("route-card");
    expect(document.querySelector(".runtime-session-select")).not.toHaveClass("route-card-button");
    expect(document.querySelector(".runtime-session-topline .task-summary-status")).not.toBeInTheDocument();
    expect(document.querySelector(".runtime-session-main")).toBeInTheDocument();
    expect(document.querySelector(".runtime-session-title-row")).toBeInTheDocument();
    expect(document.querySelector(".runtime-session-title-row")?.textContent).toContain("Workspace shell");
    expect(document.querySelector(".runtime-session-summary-row")).not.toBeInTheDocument();
    expect(document.querySelector(".runtime-session-context")).not.toBeInTheDocument();
    expect(document.querySelector(".runtime-session-bottomline")).not.toBeInTheDocument();
    expect(document.querySelector(".runtime-session-badge")).not.toBeInTheDocument();
    expect(document.querySelector("[data-runtime-session-card='terminal-1']")).toHaveAttribute("data-runtime-session-tone", "ready");
    expect(document.querySelector("[data-runtime-session-card='terminal-1'] .runtime-session-signal")).not.toBeInTheDocument();
    expect(document.querySelector("[data-runtime-session-card='terminal-1'] .runtime-session-loading")).not.toBeInTheDocument();
    expect(document.querySelector(".runtime-session-hash")).not.toBeInTheDocument();
    const sessionPane = document.querySelector("[data-runtime-session-pane='terminal']") as HTMLElement;
    expect(sessionPane).toHaveClass("is-navigation-owned");
    expect(sessionPane).toHaveAttribute("aria-hidden", "true");
    expect(sessionPane).toHaveAttribute("data-session-pane-placement", "navigation");
    expect(within(sessionPane).getByRole("list", { hidden: true })).toHaveAttribute(
      "data-runtime-session-list",
      "terminal",
    );
    expect(within(sessionPane).getAllByRole("listitem", { hidden: true })).toHaveLength(1);
    expect(document.querySelector("[data-terminal-delete]")).not.toBeInTheDocument();
    expect(document.querySelector(".runtime-workspace-shell")).toBeInTheDocument();
    expect(document.querySelector("[data-runtime-session-list='terminal']")).toHaveClass(
      "runtime-session-list",
    );
    expect(document.querySelector("[data-runtime-session-pane='terminal']")).toHaveClass(
      "runtime-workspace-session-pane",
    );
    expect(document.querySelector("[data-runtime-session-pane-head='true']")).toBeInTheDocument();
    expect(document.querySelector("[data-runtime-workspace='terminal']")).toHaveClass("runtime-workspace");
    expect(document.querySelector(".runtime-workspace-body")).not.toHaveClass("terminal-workspace-body");
    expect(document.querySelector(".runtime-workspace-body")).not.toHaveClass("conversation-workspace-body");
    expect(document.querySelector(".runtime-workspace-head")).toBeInTheDocument();
    expect(document.querySelector(".runtime-workspace-head")).toHaveClass("is-compact");
    expect(document.querySelector(".runtime-workspace-head")).toHaveClass("is-sticky");
    expect(document.querySelector(".runtime-workspace-head")).toHaveAttribute("data-runtime-workspace-header", "true");
    expect(document.querySelector(".runtime-workspace-head")).toHaveAttribute("data-runtime-header-kind", "conversation");
    expect(document.querySelector("[data-runtime-screen='terminal']")).toHaveClass("runtime-workspace-screen");
    expect(document.querySelector("[data-runtime-timeline='true']")).toBeInTheDocument();
    expect(document.querySelector(".runtime-workspace-row")).toHaveClass("runtime-workspace-title-row", "is-compact");
    expect(document.querySelector(".runtime-workspace-copy")).toHaveClass("is-compact");
    const workspaceHeader = document.querySelector(".runtime-workspace-head") as HTMLElement;
    expect(within(workspaceHeader).getByLabelText("Ready")).toBeInTheDocument();
    expect(within(workspaceHeader).getByRole("button", { name: "Details" })).toBeInTheDocument();
    expect(within(workspaceHeader).queryByRole("button", { name: "Workspace Flow" })).not.toBeInTheDocument();
    expect(within(workspaceHeader).queryByRole("button", { name: "Sessions" })).not.toBeInTheDocument();
    expect(document.querySelector("[data-terminal-close]")).not.toBeInTheDocument();
    expect(document.querySelector("[data-runtime-panel='terminal-console']")).toBeInTheDocument();
    expect(document.querySelector(".runtime-composer-form")).toBeInTheDocument();
    expect(document.querySelector(".runtime-composer-form")).toHaveAttribute("data-runtime-composer", "true");
    expect(document.querySelector(".runtime-composer-body")).toBeInTheDocument();
    expect(document.querySelector(".runtime-composer-toolbar")).toBeInTheDocument();
    const composerToolbarStart = document.querySelector(".runtime-composer-toolbar-start") as HTMLElement;
    const composerToolbarEnd = document.querySelector(".runtime-composer-toolbar-end") as HTMLElement;
    expect(composerToolbarStart).toBeInTheDocument();
    expect(composerToolbarEnd).toBeInTheDocument();
    expect(document.querySelector(".runtime-composer-input")).toBeInTheDocument();
    expect(document.querySelector(".runtime-composer-submit")).toBeInTheDocument();
    expect(document.querySelector(".runtime-composer-tools")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Quick tools" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Mention" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Workspace tools" })).not.toBeInTheDocument();
    const sessionButton = screen.getByRole("button", { name: "Session" });
    const addAttachmentButton = screen.getByRole("button", { name: "Add attachment" });
    expect(sessionButton).toHaveClass("runtime-composer-utility");
    expect(sessionButton).not.toHaveClass("is-pill");
    expect(addAttachmentButton).toHaveClass("runtime-composer-upload");
    expect(addAttachmentButton.querySelector(".runtime-composer-upload-icon svg")).toBeInTheDocument();
    expect(addAttachmentButton.querySelector(".runtime-composer-upload-label")).toHaveClass("sr-only");
    expect(composerToolbarStart).toContainElement(addAttachmentButton);
    expect(composerToolbarEnd).toContainElement(screen.getByRole("button", { name: "Send" }));
    expect(document.querySelector(".runtime-composer-meta")).not.toBeInTheDocument();
    expect(document.querySelector(".runtime-composer-form[data-runtime-composer-kind='terminal']")).toHaveClass("runtime-composer-form");
    expect(document.querySelector("[data-runtime-composer-input='terminal']")).toHaveClass("runtime-composer-input");
    expect(document.querySelector("[data-runtime-composer-submit='terminal']")).toHaveClass("runtime-composer-submit");
    expect(document.querySelector(".terminal-composer-shell")).not.toBeInTheDocument();
    expect(document.querySelector(".terminal-chat-form")).not.toBeInTheDocument();
    expect(document.querySelector(".terminal-composer-input")).not.toBeInTheDocument();
    expect(document.querySelector(".terminal-chat-submit")).not.toBeInTheDocument();
    expect(document.querySelector(".terminal-composer-tools")).not.toBeInTheDocument();
    expect(document.querySelector(".terminal-composer-meta")).not.toBeInTheDocument();
    expect(document.querySelector("[data-runtime-attachment-strip='true']")).not.toBeInTheDocument();
    expect(document.querySelector("[data-runtime-composer-submit='terminal'] svg")).toBeInTheDocument();
    expect(document.querySelector("[data-terminal-final-output='turn-1'] .message-markdown-shell")).toBeInTheDocument();
    expect(document.querySelector("[data-terminal-final-output='turn-1'] .message-markdown-toolbar")).toBeInTheDocument();
    expect(document.querySelector("[data-terminal-final-output='turn-1'] .message-markdown-copy")).toBeInTheDocument();
    expect(document.querySelector("[data-terminal-final-output='turn-1'] .message-markdown-body")).toBeInTheDocument();
    expect(document.querySelector("[data-terminal-final-output='turn-1'] .terminal-final-rendered > .message-markdown-rendered")).toBeInTheDocument();
    expect(document.querySelector("[data-terminal-final-output='turn-1'] .terminal-final-rendered")).toContainHTML(
      "<h1>Workspace</h1>",
    );
    expect(document.querySelector("[data-terminal-final-output='turn-1'] .terminal-final-rendered")).toContainHTML(
      "<li>/workspace/alter0</li>",
    );
    const code = document.querySelector("[data-terminal-final-output='turn-1'] .chat-md-inline-code") as HTMLElement;
    expect(code).toBeInTheDocument();
    expect(code.textContent).toBe("pwd");

    fireEvent.click(within(workspaceHeader).getByRole("button", { name: "Details" }));
    const metaPanel = document.querySelector("[data-runtime-details-panel='terminal']") as HTMLElement;
    expect(metaPanel).toBeInTheDocument();
    expect(workspaceHeader.contains(metaPanel)).toBe(false);
    expect(within(metaPanel).getByText("/workspace/alter0")).toBeInTheDocument();
    expect(within(metaPanel).queryByText("Summary")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Session" }));
    expect(screen.getByTestId("terminal-skill-selector")).toBeInTheDocument();
  });

  it("shows a default New terminal session placeholder when the server list is empty", async () => {
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/control/skills" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/terminal/sessions" && method === "POST") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-new-1",
            title: "New",
            terminal_session_id: "terminal-new-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0/.alter0/workspaces/terminal/sessions/terminal-new-1",
            created_at: "2026-04-15T10:20:00Z",
            updated_at: "2026-04-15T10:20:00Z",
          },
        }, { status: 201 }));
      }
      if (url === "/api/terminal/sessions/terminal-new-1/input" && method === "POST") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-new-1",
            title: "New",
            terminal_session_id: "terminal-new-1",
            status: "busy",
            shell: "codex exec",
            working_dir: "/workspace/alter0/.alter0/workspaces/terminal/sessions/terminal-new-1",
            created_at: "2026-04-15T10:20:00Z",
            updated_at: "2026-04-15T10:21:00Z",
          },
        }));
      }
      if (url === "/api/terminal/sessions/terminal-new-1" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-new-1",
            title: "Server New",
            terminal_session_id: "terminal-new-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0/.alter0/workspaces/terminal/sessions/terminal-new-1",
            created_at: "2026-04-15T10:20:00Z",
            updated_at: "2026-04-15T10:20:00Z",
            turns: [],
          },
        }));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    const view = renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-session-select='terminal-new-placeholder']")).toBeInTheDocument();
    });

    const sessionPane = document.querySelector("[data-runtime-session-pane='terminal']") as HTMLElement;
    const placeholderItems = within(sessionPane).getAllByRole("listitem", { hidden: true });
    expect(placeholderItems).toHaveLength(1);
    expect(placeholderItems[0].querySelector(".runtime-session-title")).toHaveTextContent("New");
    expect(within(sessionPane).queryByText("No terminal sessions yet.")).not.toBeInTheDocument();
    expect(document.querySelector("[data-runtime-session-card='terminal-new-placeholder']")).toHaveClass("is-active");
    expect(document.querySelector("[data-runtime-workspace='terminal']")).toHaveAttribute("data-runtime-session-id", "");
    expect(screen.getByRole("heading", { name: "New" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Details" })).toBeDisabled();
    expect(within(sessionPane).queryByRole("button", { name: "Session actions", hidden: true })).not.toBeInTheDocument();
    expect(document.querySelector("[data-runtime-composer-input='terminal']")).toHaveAttribute("placeholder", "Type command or prompt...");

    fireEvent.click(within(sessionPane).getByRole("button", { name: "New", hidden: true }));
    const fetchMock = vi.mocked(fetch);
    expect(fetchMock.mock.calls.filter(([request, init]) =>
      String(request) === "/api/terminal/sessions"
      && String(init?.method || "GET").toUpperCase() === "POST")).toHaveLength(0);

    fireEvent.change(document.querySelector("[data-runtime-composer-input='terminal']") as HTMLTextAreaElement, {
      target: { value: "pwd" },
    });
    fireEvent.click(document.querySelector("[data-runtime-composer-submit='terminal']") as HTMLButtonElement);

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-session-select='terminal-new-1']")).toBeInTheDocument();
    });

    expect(fetchMock.mock.calls.some(([request, init]) =>
      String(request) === "/api/terminal/sessions"
      && String(init?.method || "GET").toUpperCase() === "POST")).toBe(true);
    expect(fetchMock.mock.calls.some(([request, init]) =>
      String(request) === "/api/terminal/sessions/terminal-new-1/input"
      && String(init?.method || "GET").toUpperCase() === "POST")).toBe(true);
    await waitFor(() => {
      expect(fetchMock.mock.calls.some(([request, init]) =>
        String(request) === "/api/terminal/sessions/terminal-new-1"
        && String(init?.method || "GET").toUpperCase() === "GET")).toBe(true);
    });
    await waitFor(() => {
      expect(screen.getByRole("heading", { name: "Server New" })).toBeInTheDocument();
    });
    expect(view.container.querySelector("[data-runtime-session-select='terminal-new-placeholder']")).not.toBeInTheDocument();
  });

  it("pins terminal sessions through the shared session action button set", async () => {
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return Promise.resolve(jsonResponse({
          items: [{
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            pinned: false,
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
          }],
        }));
      }
      if (url === "/api/control/skills" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/terminal/sessions/terminal-1" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            pinned: false,
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
            turns: [],
          },
        }));
      }
      if (url === "/api/terminal/sessions/terminal-1/pin" && method === "POST") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            pinned: true,
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
            turns: [],
          },
        }));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-session-card='terminal-1']")).toBeInTheDocument();
    });
    const card = openTerminalSessionActions("terminal-1");
    fireEvent.click(within(card).getByRole("menuitem", { name: "Pin session", hidden: true }));

    await waitFor(() => {
      expect(vi.mocked(fetch).mock.calls.some(([request, init]) =>
        String(request) === "/api/terminal/sessions/terminal-1/pin"
        && String(init?.method || "GET").toUpperCase() === "POST"
        && init?.body === JSON.stringify({ pinned: true })
      )).toBe(true);
    });
    const updatedCard = openTerminalSessionActions("terminal-1");
    expect(within(updatedCard).getByRole("menuitem", { name: "Unpin session", hidden: true })).toBeInTheDocument();
  });

  it("keeps terminal status copy inside the shared composer form instead of adding an outer note row", async () => {
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return Promise.resolve(jsonResponse({
          items: [{
            id: "terminal-failed",
            title: "Failed shell",
            terminal_session_id: "terminal-failed",
            status: "failed",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            error_message: "Command exited with code 1",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
          }],
        }));
      }
      if (url === "/api/control/skills" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/terminal/sessions/terminal-failed" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-failed",
            title: "Failed shell",
            terminal_session_id: "terminal-failed",
            status: "failed",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            error_message: "Command exited with code 1",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
            turns: [],
          },
        }));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    const view = renderTerminalRouteBody();

    await waitFor(() => {
      expect(screen.getByRole("heading", { name: "Failed shell" })).toBeInTheDocument();
    });

    expect(view.container.querySelector("[data-runtime-note='terminal']")).not.toBeInTheDocument();
    const toolbarMeta = view.container.querySelector(".runtime-composer-form .runtime-composer-meta");
    expect(toolbarMeta).toBeInTheDocument();
    expect(toolbarMeta).toHaveTextContent("The last runtime failed");
    expect(toolbarMeta).toHaveTextContent("Command exited with code 1");
  });

  it("shows the default New terminal session placeholder while the server list is loading", async () => {
    let resolveSessions: (() => void) | null = null;
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return new Promise<Response>((resolve) => {
          resolveSessions = () => resolve(jsonResponse({ items: [] }));
        });
      }
      if (url === "/api/control/skills" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    renderTerminalRouteBody();

    expect(document.querySelector("[data-runtime-session-select='terminal-new-placeholder']")).toBeInTheDocument();
    expect(document.querySelector("[data-runtime-session-card='terminal-new-placeholder']")).toHaveClass("is-active");
    expect(screen.getByRole("heading", { name: "New" })).toBeInTheDocument();

    resolveSessions?.();
    await waitFor(() => {
      expect(document.querySelector("[data-runtime-session-select='terminal-new-placeholder']")).toBeInTheDocument();
    });
  });

  it("treats the empty terminal New placeholder as an interactive draft session like Chat", async () => {
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/control/skills" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));
    const closeMobileSessionPane = vi.fn();

    renderTerminalRouteBody({
      isMobileViewport: true,
      mobileSessionPaneOpen: true,
      closeMobileSessionPane,
    });

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-session-select='terminal-new-placeholder']")).toBeInTheDocument();
    });
    const input = document.querySelector("[data-runtime-composer-input='terminal']") as HTMLTextAreaElement;
    expect(document.activeElement).not.toBe(input);

    const placeholderCard = document.querySelector("[data-runtime-session-card='terminal-new-placeholder']") as HTMLElement;
    fireEvent.click(within(placeholderCard).getByRole("button", { name: /New/, hidden: true }));

    const fetchMock = vi.mocked(fetch);
    expect(fetchMock.mock.calls.filter(([request, init]) =>
      String(request) === "/api/terminal/sessions"
      && String(init?.method || "GET").toUpperCase() === "POST")).toHaveLength(0);
    expect(closeMobileSessionPane).toHaveBeenCalled();
    expect(document.querySelector("[data-runtime-session-pane='terminal']")).not.toHaveClass("is-open");
    await waitFor(() => {
      expect(document.activeElement).toBe(input);
    });
  });

  it("shows Codex slash command candidates for an explicit Codex terminal session", async () => {
    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-composer-input='terminal']")).toBeInTheDocument();
      expect(document.querySelector("[data-runtime-workspace='terminal']")).toHaveAttribute("data-runtime-session-id", "terminal-1");
    });

    fireEvent.change(document.querySelector("[data-runtime-composer-input='terminal']") as HTMLTextAreaElement, {
      target: { value: "/" },
    });

    const commandList = screen.getByRole("listbox", { name: "Codex slash commands" });
    expect(commandList).toHaveAttribute("data-runtime-composer-command-list", "codex");
    expect(within(commandList).getAllByRole("option").length).toBeGreaterThan(10);
    expect(within(commandList).getByRole("option", { name: /\/goal/i })).toBeInTheDocument();
    expect(within(commandList).getByRole("option", { name: /\/model/i })).toBeInTheDocument();
    expect(within(commandList).getByRole("option", { name: /\/status/i })).toBeInTheDocument();
    expect(within(commandList).queryByRole("option", { name: /\/permissions/i })).not.toBeInTheDocument();
  });

  it("applies a Codex slash command candidate in the terminal composer", async () => {
    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-composer-input='terminal']")).toBeInTheDocument();
      expect(document.querySelector("[data-runtime-workspace='terminal']")).toHaveAttribute("data-runtime-session-id", "terminal-1");
    });

    const input = document.querySelector("[data-runtime-composer-input='terminal']") as HTMLTextAreaElement;
    fireEvent.change(input, {
      target: { value: "/g ship terminal candidates" },
    });
    const goalOption = await screen.findByRole("option", { name: /\/goal/i });
    fireEvent.click(goalOption);

    expect(input.value).toBe("/goal ship terminal candidates");
  });

  it("does not show Codex slash command candidates for non-Codex terminal sessions", async () => {
    stubTerminalTurnsFetch([], "bash");
    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-composer-input='terminal']")).toBeInTheDocument();
      expect(document.querySelector("[data-runtime-workspace='terminal']")).toHaveAttribute("data-runtime-session-id", "terminal-1");
    });

    fireEvent.change(document.querySelector("[data-runtime-composer-input='terminal']") as HTMLTextAreaElement, {
      target: { value: "/" },
    });

    expect(screen.queryByRole("listbox", { name: "Codex slash commands" })).not.toBeInTheDocument();
  });

  it("renders terminal inline code without leaking HTML entities", async () => {
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return Promise.resolve(jsonResponse({
          items: [
            {
              id: "terminal-1",
              title: "Workspace shell",
              terminal_session_id: "terminal-1",
              status: "ready",
              shell: "codex exec",
              working_dir: "/workspace/alter0",
              created_at: "2026-04-15T10:00:00Z",
              updated_at: "2026-04-15T10:10:00Z",
            },
          ],
        }));
      }
      if (url === "/api/terminal/sessions/terminal-1" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
            turns: [
              {
                id: "turn-1",
                prompt: "explain",
                status: "completed",
                started_at: "2026-04-15T10:05:00Z",
                finished_at: "2026-04-15T10:05:02Z",
                duration_ms: 2000,
                final_output: "链路：`请求接入 -> 召回 -> 粗排 -> 精排 -> 返回广告`",
                runtime_trace_events: [],
              },
            ],
          },
        }));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-turn='turn-1']")).toBeInTheDocument();
    });

    const code = document.querySelector("[data-terminal-final-output='turn-1'] .chat-md-inline-code") as HTMLElement;
    expect(code).toBeInTheDocument();
    expect(code.textContent).toBe("请求接入 -> 召回 -> 粗排 -> 精排 -> 返回广告");
    expect(code.innerHTML).toContain("-&gt;");
    expect(code.innerHTML).not.toContain("&amp;gt;");
  });

  it("decodes html entities in terminal final output", async () => {
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return Promise.resolve(jsonResponse({
          items: [
            {
              id: "terminal-1",
              title: "Workspace shell",
              terminal_session_id: "terminal-1",
              status: "ready",
              shell: "codex exec",
              working_dir: "/workspace/alter0",
              created_at: "2026-04-15T10:00:00Z",
              updated_at: "2026-04-15T10:10:00Z",
            },
          ],
        }));
      }
      if (url === "/api/terminal/sessions/terminal-1" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
            turns: [
              {
                id: "turn-1",
                prompt: "explain",
                status: "completed",
                started_at: "2026-04-15T10:05:00Z",
                finished_at: "2026-04-15T10:05:02Z",
                duration_ms: 2000,
                final_output: "Use Chat &gt; Details &gt; Model to switch runtime.",
                runtime_trace_events: [],
              },
            ],
          },
        }));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-turn='turn-1']")).toBeInTheDocument();
    });

    const output = document.querySelector("[data-terminal-final-output='turn-1']") as HTMLElement;
    expect(output).toBeInTheDocument();
    expect(output.textContent).toContain("Use Chat > Details > Model to switch runtime.");
    expect(output.textContent).not.toContain("&gt;");
  });

  it("copies long terminal output without mirroring the payload into DOM attributes", async () => {
    const finalOutput = "terminal copy output\n".repeat(512);
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });
    stubTerminalTurnsFetch([
      {
        id: "turn-long-copy",
        prompt: "print long output",
        final_output: finalOutput,
      },
    ]);

    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-final-output='turn-long-copy']")).toBeInTheDocument();
    });

    const copyButton = document.querySelector("[data-terminal-final-output='turn-long-copy'] .message-markdown-copy") as HTMLButtonElement;
    expect(copyButton).toBeInTheDocument();
    expect(copyButton).not.toHaveAttribute("data-copy-value");

    fireEvent.click(copyButton);

    await waitFor(() => {
      expect(writeText).toHaveBeenCalledWith(finalOutput);
    });
  });

  it("groups terminal sessions into recency sections in the shared sidebar", async () => {
    const todayStart = new Date();
    todayStart.setHours(0, 0, 0, 0);
    const yesterdayStart = new Date(todayStart);
    yesterdayStart.setDate(todayStart.getDate() - 1);
    const earlierStart = new Date(todayStart);
    earlierStart.setDate(todayStart.getDate() - 5);
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return Promise.resolve(jsonResponse({
          items: [
            {
              id: "terminal-1",
              title: "Workspace shell",
              terminal_session_id: "terminal-1",
              status: "ready",
              shell: "codex exec",
              working_dir: "/workspace/alter0",
              created_at: new Date(todayStart.getTime() + (60 * 60 * 1000)).toISOString(),
              updated_at: new Date(todayStart.getTime() + (2 * 60 * 60 * 1000)).toISOString(),
            },
            {
              id: "terminal-2",
              title: "Gemini parity review",
              terminal_session_id: "terminal-2",
              status: "busy",
              shell: "codex exec",
              working_dir: "/workspace/alter0",
              created_at: new Date(yesterdayStart.getTime() + (60 * 60 * 1000)).toISOString(),
              updated_at: new Date(yesterdayStart.getTime() + (2 * 60 * 60 * 1000)).toISOString(),
            },
            {
              id: "terminal-3",
              title: "Pinned archival session",
              terminal_session_id: "terminal-3",
              status: "ready",
              shell: "codex exec",
              working_dir: "/workspace/alter0",
              pinned: true,
              created_at: new Date(earlierStart.getTime() + (60 * 60 * 1000)).toISOString(),
              updated_at: new Date(earlierStart.getTime() + (2 * 60 * 60 * 1000)).toISOString(),
            },
            {
              id: "terminal-4",
              title: "Older archival session",
              terminal_session_id: "terminal-4",
              status: "exited",
              shell: "codex exec",
              working_dir: "/workspace/alter0",
              created_at: new Date(earlierStart.getTime() + (60 * 60 * 1000)).toISOString(),
              updated_at: new Date(earlierStart.getTime() + (2 * 60 * 60 * 1000)).toISOString(),
            },
          ],
        }));
      }
      if (url === "/api/terminal/sessions/terminal-1" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: new Date(todayStart.getTime() + (60 * 60 * 1000)).toISOString(),
            updated_at: new Date(todayStart.getTime() + (2 * 60 * 60 * 1000)).toISOString(),
            turns: [],
          },
        }));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-session-select='terminal-1']")).toBeInTheDocument();
    });

    const sessionPane = document.querySelector("[data-runtime-session-pane='terminal']") as HTMLElement;
    expect(sessionPane).toHaveClass("is-navigation-owned");
    expect(sessionPane).toHaveAttribute("aria-hidden", "true");
    expect(sessionPane).toHaveAttribute("data-session-pane-placement", "navigation");
    expect(Array.from(sessionPane.querySelectorAll(".runtime-session-group-label")).map((item) => item.textContent)).toEqual([
      "Pinned",
      "Today",
      "Yesterday",
      "Earlier",
    ]);
    expect(within(sessionPane).getByText("Pinned")).toBeInTheDocument();
    expect(within(sessionPane).getByText("Today")).toBeInTheDocument();
    expect(within(sessionPane).getByText("Yesterday")).toBeInTheDocument();
    expect(within(sessionPane).getByText("Earlier")).toBeInTheDocument();
    expect(within(sessionPane).getAllByRole("listitem", { hidden: true })).toHaveLength(4);
    const firstCard = within(sessionPane).getAllByRole("listitem", { hidden: true })[0] as HTMLElement;
    expect(firstCard.querySelector(".runtime-session-summary-row")).not.toBeInTheDocument();
    expect(firstCard.querySelector(".runtime-session-context")).not.toBeInTheDocument();
    expect(firstCard).not.toHaveTextContent("Last output");
    expect(firstCard).not.toHaveTextContent("最近输出");
    expect(firstCard).not.toHaveTextContent("#");
    expect(firstCard.querySelector(".runtime-session-bottomline")).not.toBeInTheDocument();
  });

  it("keeps a deleted terminal session hidden when a later list refresh returns stale data", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(true);

    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return Promise.resolve(jsonResponse({
          items: [
            {
              id: "terminal-1",
              title: "Workspace shell",
              terminal_session_id: "terminal-1",
              status: "ready",
              shell: "codex exec",
              working_dir: "/workspace/alter0",
              created_at: "2026-04-15T10:00:00Z",
              updated_at: "2026-04-15T10:10:00Z",
            },
            {
              id: "terminal-2",
              title: "Older shell",
              terminal_session_id: "terminal-2",
              status: "ready",
              shell: "codex exec",
              working_dir: "/workspace/alter0",
              created_at: "2026-04-15T09:00:00Z",
              updated_at: "2026-04-15T09:10:00Z",
            },
          ],
        }));
      }
      if (url === "/api/control/skills" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/terminal/sessions/terminal-1" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
            turns: [],
          },
        }));
      }
      if (url === "/api/terminal/sessions/terminal-2" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-2",
            title: "Older shell",
            terminal_session_id: "terminal-2",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T09:00:00Z",
            updated_at: "2026-04-15T09:10:00Z",
            turns: [],
          },
        }));
      }
      if (url === "/api/terminal/sessions/terminal-2" && method === "DELETE") {
        return Promise.resolve(new Response(null, { status: 204 }));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-session-select='terminal-2']")).toBeInTheDocument();
    });

    const card = openTerminalSessionActions("terminal-2");
    fireEvent.click(within(card).getByRole("menuitem", { name: "Delete", hidden: true }));

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-session-select='terminal-2']")).not.toBeInTheDocument();
    });

    await act(async () => {
      window.dispatchEvent(new FocusEvent("focus"));
      await Promise.resolve();
    });

    await waitFor(() => {
      expect(document.querySelectorAll("[data-runtime-session-select]")).toHaveLength(1);
    });
    expect(document.querySelector("[data-runtime-session-select='terminal-2']")).not.toBeInTheDocument();
    expect(document.querySelector("[data-runtime-workspace='terminal']")).toHaveAttribute("data-runtime-session-id", "terminal-1");
  });

  it("attaches images in terminal composer and submits them with the terminal input payload", async () => {
    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-composer-input='terminal']")).toBeInTheDocument();
    });

    const fileInput = document.querySelector('input[type="file"][accept="image/*,.txt,.md,.json,.yaml,.yml,.csv,.log,.pdf"]') as HTMLInputElement;
    expect(fileInput).toBeInTheDocument();

    const image = new File(['<svg xmlns="http://www.w3.org/2000/svg"></svg>'], "terminal-shot.svg", { type: "image/svg+xml" });
    fireEvent.change(fileInput, { target: { files: [image] } });

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Preview terminal-shot.svg" })).toBeInTheDocument();
      expect(document.querySelector("[data-runtime-attachment-strip='true']")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Preview terminal-shot.svg" }));
    expect(document.querySelector("[data-runtime-attachment-preview='true']")).toBeInTheDocument();
    expect(screen.getByRole("dialog", { name: "terminal-shot.svg" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Close preview" }));

    fireEvent.change(document.querySelector("[data-runtime-composer-input='terminal']") as HTMLTextAreaElement, {
      target: { value: "inspect screenshot" },
    });
    fireEvent.click(document.querySelector("[data-runtime-composer-submit='terminal']") as HTMLButtonElement);

    await waitFor(() => {
      const fetchMock = vi.mocked(fetch);
      expect(fetchMock.mock.calls.some(([request, init]) =>
        String(request) === "/api/terminal/sessions/terminal-1/attachments"
        && String(init?.method || "GET").toUpperCase() === "POST")).toBe(true);
      expect(fetchMock.mock.calls.some(([request, init]) =>
        String(request) === "/api/terminal/sessions/terminal-1/input"
        && String(init?.method || "GET").toUpperCase() === "POST"
        && JSON.parse(String(init?.body || "{}")).attachments?.length === 1)).toBe(true);
    });

    const fetchMock = vi.mocked(fetch);
    const inputCall = fetchMock.mock.calls.filter(([request, init]) =>
      String(request) === "/api/terminal/sessions/terminal-1/input"
      && String(init?.method || "GET").toUpperCase() === "POST").at(-1);
    const payload = JSON.parse(String((inputCall?.[1] as RequestInit | undefined)?.body || "{}"));
    expect(payload.attachments[0]).toMatchObject({
      id: "asset-terminal-1",
      asset_url: "/api/terminal/sessions/terminal-1/attachments/asset-terminal-1/original",
      preview_url: "/api/terminal/sessions/terminal-1/attachments/asset-terminal-1/preview",
    });
    expect(payload.attachments[0].data_url).toBeUndefined();
  });

  it("adds pasted image files from the terminal composer input to attachments", async () => {
    renderTerminalRouteBody();

    const composerInput = await waitFor(() => {
      const input = document.querySelector("[data-runtime-composer-input='terminal']") as HTMLTextAreaElement | null;
      expect(input).toBeInTheDocument();
      expect(input).not.toBeDisabled();
      return input as HTMLTextAreaElement;
    });
    const image = new File(['<svg xmlns="http://www.w3.org/2000/svg"></svg>'], "terminal-paste.svg", {
      type: "image/svg+xml",
    });
    const pasteEvent = new Event("paste", { bubbles: true, cancelable: true });
    Object.defineProperty(pasteEvent, "clipboardData", {
      value: {
        files: [image],
      },
    });

    fireEvent(composerInput, pasteEvent);

    await waitFor(() => {
      const fetchMock = vi.mocked(fetch);
      expect(fetchMock.mock.calls.some(([request, init]) =>
        String(request) === "/api/terminal/sessions/terminal-1/attachments"
        && String(init?.method || "GET").toUpperCase() === "POST")).toBe(true);
      expect(screen.getByRole("button", { name: "Preview terminal-shot.svg" })).toBeInTheDocument();
    });
    expect(pasteEvent.defaultPrevented).toBe(true);
  });

  it("attaches files in terminal composer and submits them with stable asset references", async () => {
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return Promise.resolve(jsonResponse({
          items: [
            {
              id: "terminal-1",
              title: "Workspace shell",
              terminal_session_id: "terminal-1",
              status: "ready",
              shell: "codex exec",
              working_dir: "/workspace/alter0",
              created_at: "2026-04-15T10:00:00Z",
              updated_at: "2026-04-15T10:10:00Z",
            },
          ],
        }));
      }
      if (url === "/api/terminal/sessions/terminal-1" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
            turns: [],
          },
        }));
      }
      if (url === "/api/terminal/sessions/terminal-1/attachments" && method === "POST") {
        return Promise.resolve(jsonResponse({
          items: [
            {
              id: "asset-terminal-file-1",
              name: "requirements.md",
              content_type: "text/markdown",
              size: 20,
              asset_url: "/api/terminal/sessions/terminal-1/attachments/asset-terminal-file-1/original",
            },
          ],
        }));
      }
      if (url === "/api/terminal/sessions/terminal-1/input" && method === "POST") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "busy",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:30Z",
          },
        }));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-input]")).toBeInTheDocument();
    });

    const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement;
    const file = new File(["# Requirements"], "requirements.md", { type: "text/markdown" });
    fireEvent.change(fileInput, { target: { files: [file] } });

    await waitFor(() => {
      expect(screen.getByText("requirements.md")).toBeInTheDocument();
    });

    fireEvent.change(document.querySelector("[data-terminal-input]") as HTMLTextAreaElement, {
      target: { value: "review the attached docs" },
    });
    fireEvent.click(document.querySelector("[data-terminal-submit]") as HTMLButtonElement);

    await waitFor(() => {
      const fetchMock = vi.mocked(fetch);
      expect(fetchMock.mock.calls.some(([request, init]) =>
        String(request) === "/api/terminal/sessions/terminal-1/input"
        && String(init?.method || "GET").toUpperCase() === "POST")).toBe(true);
    });

    const fetchMock = vi.mocked(fetch);
    const inputCall = fetchMock.mock.calls.filter(([request, init]) =>
      String(request) === "/api/terminal/sessions/terminal-1/input"
      && String(init?.method || "GET").toUpperCase() === "POST").at(-1);
    const payload = JSON.parse(String((inputCall?.[1] as RequestInit | undefined)?.body || "{}"));
    expect(payload.attachments[0]).toMatchObject({
      id: "asset-terminal-file-1",
      asset_url: "/api/terminal/sessions/terminal-1/attachments/asset-terminal-file-1/original",
    });
    expect(payload.attachments[0].preview_url).toBeUndefined();
    expect(payload.attachments[0].data_url).toBeUndefined();
  });

  it("loads event detail when expanding a process event", async () => {
    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-step-toggle='step-1']")).toBeInTheDocument();
    });

    fireEvent.click(document.querySelector("[data-terminal-step-toggle='step-1']")!);

    await waitFor(() => {
      expect(document.querySelector(".terminal-step-content code")?.textContent).toBe(
        "pwd\n\n/workspace/alter0",
      );
    });
  });

  it("renders terminal markdown event detail as readable rich text", async () => {
    const defaultFetch = vi.mocked(fetch).getMockImplementation();
    vi.mocked(fetch).mockImplementation((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions/terminal-1" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
            turns: [
              {
                id: "turn-1",
                prompt: "summarize",
                status: "completed",
                started_at: "2026-04-15T10:05:00Z",
                finished_at: "2026-04-15T10:05:02Z",
                duration_ms: 2000,
                final_output: "done",
                runtime_trace_events: [
                  terminalRuntimeEventFixture({
                    kind: "assistant_commentary",
                    title: "Markdown detail",
                    summary: "Markdown detail",
                    raw: { ref: "step-1", type: "message", has_detail: true },
                  }),
                ],
              },
            ],
          },
        }));
      }
      if (url === "/api/terminal/sessions/terminal-1/turns/turn-1/events/step-1" && method === "GET") {
        return Promise.resolve(jsonResponse({
          event: {
            turn_id: "turn-1",
            blocks: [
              {
                type: "markdown",
                title: "Markdown contract",
                text: "Render **markdown** as readable text.",
              },
            ],
          },
        }));
      }
      return defaultFetch?.(input, init) ?? Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    });

    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-step-toggle='step-1']")).toBeInTheDocument();
    });

    fireEvent.click(document.querySelector("[data-terminal-step-toggle='step-1']")!);

    await waitFor(() => {
      const step = document.querySelector("[data-terminal-step-item='step-1']") as HTMLElement;
      const block = step.querySelector(".terminal-rich-block.type-markdown") as HTMLElement;
      expect(block).toBeInTheDocument();
      expect(block.querySelector(".terminal-rich-head")).toHaveTextContent("Markdown contract");
      expect(block.querySelector(".message-markdown-rendered")).toHaveTextContent("Render markdown as readable text.");
      expect(block.querySelector(".terminal-step-content code")).not.toBeInTheDocument();
    });
  });

  it("does not repeat the step status inside expanded process detail blocks", async () => {
    const defaultFetch = vi.mocked(fetch).getMockImplementation();
    vi.mocked(fetch).mockImplementation((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions/terminal-1" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
            turns: [
              {
                id: "turn-1",
                prompt: "validate",
                status: "completed",
                started_at: "2026-04-15T10:05:00Z",
                finished_at: "2026-04-15T10:05:02Z",
                duration_ms: 2000,
                final_output: "done",
                runtime_trace_events: [
                  terminalRuntimeEventFixture({
                    kind: "system_event",
                    lifecycle: "failed",
                    status: "failed",
                    title: "Error log",
                    summary: "Simulated validation error",
                    raw: { ref: "step-1", type: "log", has_detail: true },
                  }),
                ],
              },
            ],
          },
        }));
      }
      if (url === "/api/terminal/sessions/terminal-1/turns/turn-1/events/step-1" && method === "GET") {
        return Promise.resolve(jsonResponse({
          event: {
            turn_id: "turn-1",
            blocks: [
              {
                type: "text",
                title: "Error",
                text: "Simulated validation error.",
              },
            ],
          },
        }));
      }
      return defaultFetch?.(input, init) ?? Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    });

    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-step-toggle='step-1']")).toBeInTheDocument();
    });

    fireEvent.click(document.querySelector("[data-terminal-step-toggle='step-1']")!);

    await waitFor(() => {
      const step = document.querySelector("[data-terminal-step-item='step-1']") as HTMLElement;
      expect(step).toBeInTheDocument();
      expect(within(step).getAllByText("Failed")).toHaveLength(1);
      expect(step.querySelector(".terminal-rich-head")).toHaveTextContent("Error");
      expect(step.querySelector(".terminal-rich-meta")).not.toBeInTheDocument();
    });
  });

  it("discloses runtime event categories on terminal process events", async () => {
    renderTerminalRouteBody();

    await waitFor(() => {
      const step = document.querySelector("[data-terminal-step-item='step-1']") as HTMLElement;
      expect(step).toBeInTheDocument();
      expect(step.querySelector(".terminal-step-kind")).toHaveTextContent("Commands");
    });
  });

  it("waits for shell event detail before expanding the step body", async () => {
    let resolveEventDetail: ((value: Response) => void) | undefined;
    const eventDetail = new Promise<Response>((resolve) => {
      resolveEventDetail = resolve;
    });
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return Promise.resolve(jsonResponse({
          items: [
            {
              id: "terminal-1",
              title: "Workspace shell",
              terminal_session_id: "terminal-1",
              status: "ready",
              shell: "codex exec",
              working_dir: "/workspace/alter0",
              created_at: "2026-04-15T10:00:00Z",
              updated_at: "2026-04-15T10:10:00Z",
            },
          ],
        }));
      }
      if (url === "/api/control/skills" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/terminal/sessions/terminal-1" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
            turns: [
              {
                id: "turn-1",
                prompt: "inspect",
                status: "completed",
                started_at: "2026-04-15T10:05:00Z",
                finished_at: "2026-04-15T10:05:02Z",
                duration_ms: 2000,
                final_output: "done",
                runtime_trace_events: [
                  terminalRuntimeEventFixture({
                    title: "Shell",
                    summary: "sed -n '1,120p' AGENTS.md",
                  }),
                ],
              },
            ],
          },
        }));
      }
      if (url === "/api/terminal/sessions/terminal-1/turns/turn-1/events/step-1" && method === "GET") {
        return eventDetail;
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-step-toggle='step-1']")).toBeInTheDocument();
    });

    fireEvent.click(document.querySelector("[data-terminal-step-toggle='step-1']")!);

    expect(document.querySelector("[data-terminal-step-toggle='step-1']")).toHaveAttribute("aria-expanded", "false");
    expect(document.querySelector("[data-terminal-step-item='step-1'] .terminal-step-body")).toHaveAttribute("hidden");

    await act(async () => {
      resolveEventDetail?.(jsonResponse({
        event: {
          turn_id: "turn-1",
          blocks: [
            {
              type: "terminal",
              title: "Shell",
              command: "sed -n '1,120p' AGENTS.md",
              output: "# Rule",
              language: "shell",
            },
          ],
        },
      }));
      await eventDetail;
    });

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-step-toggle='step-1']")).toHaveAttribute("aria-expanded", "true");
      expect(document.querySelector(".terminal-step-content code")?.textContent).toBe(
        "sed -n '1,120p' AGENTS.md\n\n# Rule",
      );
    });
  });

  it("renders terminal process as a compact thought disclosure", async () => {
    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-process-toggle='turn-1']")).toBeInTheDocument();
    });

    const process = document.querySelector("[data-terminal-process-shell='turn-1']") as HTMLElement;
    const toggle = document.querySelector("[data-terminal-process-toggle='turn-1']") as HTMLButtonElement;
    expect(process).toHaveClass("is-collapsed");
    expect(process).toHaveClass("runtime-thinking-shell");
    expect(toggle).toHaveClass("runtime-thinking-toggle");
    expect(toggle).toHaveTextContent("Thinking");
    expect(toggle).not.toHaveTextContent("2s");
    expect(toggle).not.toHaveTextContent("Process");
    expect(toggle.querySelector(".terminal-process-meta")).not.toBeInTheDocument();
    expect(toggle.querySelector(".terminal-step-toggle-icon")).toHaveTextContent(">");
  });

  it("renders terminal events with runtime event metadata", async () => {
    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-step-item='step-1']")).toBeInTheDocument();
    });

    const step = document.querySelector("[data-terminal-step-item='step-1']") as HTMLElement;
    expect(step).toHaveAttribute("data-runtime-event-kind", "shell_command");
    expect(step).toHaveAttribute("data-runtime-event-source", "adapter");
  });

  it("renders a dedicated terminal step toggle icon so the step title stays in the readable content column", async () => {
    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-step-toggle='step-1']")).toBeInTheDocument();
    });

    const toggle = document.querySelector("[data-terminal-step-toggle='step-1']") as HTMLButtonElement;
    const icon = toggle.querySelector(".terminal-step-toggle-icon");
    const title = toggle.querySelector(".terminal-step-title");

    expect(icon).toBeInTheDocument();
    expect(icon).toHaveTextContent(">");
    expect(title).toHaveTextContent("pwd");

    fireEvent.click(toggle);

    await waitFor(() => {
      expect(toggle.querySelector(".terminal-step-toggle-icon")).toHaveTextContent("v");
      expect(toggle.querySelector(".terminal-step-title")).toHaveTextContent("pwd");
    });
  });

  it("renders terminal narrative event detail as readable text instead of preserving pathological per-glyph line breaks", async () => {
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return Promise.resolve(jsonResponse({
          items: [
            {
              id: "terminal-1",
              title: "Workspace shell",
              terminal_session_id: "terminal-1",
              status: "ready",
              shell: "codex exec",
              working_dir: "/workspace/alter0",
              created_at: "2026-04-15T10:00:00Z",
              updated_at: "2026-04-15T10:10:00Z",
            },
          ],
        }));
      }
      if (url === "/api/terminal/sessions/terminal-1" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
            turns: [
              {
                id: "turn-1",
                prompt: "summarize",
                status: "completed",
                started_at: "2026-04-15T10:05:00Z",
                finished_at: "2026-04-15T10:05:02Z",
                duration_ms: 2000,
                final_output: "done",
                runtime_trace_events: [
                  terminalRuntimeEventFixture({
                    kind: "reasoning",
                    title: "Explain local runtime constraints",
                    summary: "先\n读\n取\n本\n地\n运\n行\n约\n束",
                    raw: { ref: "step-1", type: "reasoning", has_detail: true },
                  }),
                ],
              },
            ],
          },
        }));
      }
      if (url === "/api/terminal/sessions/terminal-1/turns/turn-1/events/step-1" && method === "GET") {
        return Promise.resolve(jsonResponse({
          event: {
            turn_id: "turn-1",
            blocks: [
              {
                type: "text",
                title: "Reasoning",
                text: "先\n读\n取\n本\n地\n运\n行\n约\n束，\n然\n后\n直\n接\n给\n出\n方\n案",
              },
            ],
          },
        }));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-step-toggle='step-1']")).toBeInTheDocument();
    });

    fireEvent.click(document.querySelector("[data-terminal-step-toggle='step-1']")!);

    await waitFor(() => {
      const detail = document.querySelector(".terminal-step-detail .message-markdown-rendered") as HTMLElement;
      expect(detail).toBeInTheDocument();
      expect(detail.textContent).toContain("先读取本地运行约束，然后直接给出方案");
      expect(document.querySelector(".terminal-step-content code")).not.toBeInTheDocument();
    });
  });

  it("creates a new terminal session through the React action bar", async () => {
    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-session-select='terminal-1']")).toBeInTheDocument();
    });

    fireEvent.click(document.querySelector("[data-runtime-create-session='terminal']")!);

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-workspace='terminal']")).toHaveAttribute(
        "data-runtime-session-id",
        "terminal-2",
      );
    });
  });

  it("submits the first terminal input on the first click even when no session exists yet", async () => {
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/control/skills" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/terminal/sessions" && method === "POST") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-2",
            title: "terminal-2",
            terminal_session_id: "terminal-2",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0/.alter0/workspaces/terminal/sessions/terminal-2",
            created_at: "2026-04-15T10:20:00Z",
            updated_at: "2026-04-15T10:20:00Z",
          },
        }, { status: 201 }));
      }
      if (url === "/api/terminal/sessions/terminal-2" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-2",
            title: "terminal-2",
            terminal_session_id: "terminal-2",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0/.alter0/workspaces/terminal/sessions/terminal-2",
            created_at: "2026-04-15T10:20:00Z",
            updated_at: "2026-04-15T10:20:00Z",
            turns: [
              {
                id: "turn-2",
                prompt: "pwd",
                status: "completed",
                started_at: "2026-04-15T10:20:01Z",
                finished_at: "2026-04-15T10:20:03Z",
                duration_ms: 2000,
                final_output: "/workspace/alter0/.alter0/workspaces/terminal/sessions/terminal-2",
                runtime_trace_events: [],
              },
            ],
          },
        }));
      }
      if (url === "/api/terminal/sessions/terminal-2/input" && method === "POST") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-2",
            title: "terminal-2",
            terminal_session_id: "terminal-2",
            status: "busy",
            shell: "codex exec",
            working_dir: "/workspace/alter0/.alter0/workspaces/terminal/sessions/terminal-2",
            created_at: "2026-04-15T10:20:00Z",
            updated_at: "2026-04-15T10:20:01Z",
            turns: [],
          },
        }));
      }
      if (url === "/api/terminal/sessions/terminal-2/attachments" && method === "POST") {
        return Promise.resolve(jsonResponse({
          items: [
            {
              id: "asset-terminal-2",
              name: "diagram.svg",
              content_type: "image/svg+xml",
              size: 32,
              asset_url: "/api/terminal/sessions/terminal-2/attachments/asset-terminal-2/original",
              preview_url: "/api/terminal/sessions/terminal-2/attachments/asset-terminal-2/preview",
            },
          ],
        }));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-composer-input='terminal']")).toBeInTheDocument();
    });

    fireEvent.change(document.querySelector("[data-runtime-composer-input='terminal']") as HTMLTextAreaElement, {
      target: { value: "pwd" },
    });
    fireEvent.click(document.querySelector("[data-runtime-composer-submit='terminal']") as HTMLButtonElement);

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-workspace='terminal']")).toHaveAttribute(
        "data-runtime-session-id",
        "terminal-2",
      );
    });
    await waitFor(() => {
      expect(document.querySelector("[data-terminal-turn='turn-2']")).toBeInTheDocument();
    });
    expect(document.querySelector("[data-runtime-composer-input='terminal']")).toHaveValue("");

    const fetchMock = vi.mocked(fetch);
    await waitFor(() => {
      expect(fetchMock.mock.calls.some(([request, init]) =>
        String(request) === "/api/terminal/sessions/terminal-2/input"
        && String(init?.method || "GET").toUpperCase() === "POST")).toBe(true);
    });
  });

  it("keeps image attachments on the first terminal input when no session exists yet", async () => {
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/control/skills" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/terminal/sessions" && method === "POST") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-2",
            title: "terminal-2",
            terminal_session_id: "terminal-2",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0/.alter0/workspaces/terminal/sessions/terminal-2",
            created_at: "2026-04-15T10:20:00Z",
            updated_at: "2026-04-15T10:20:00Z",
          },
        }, { status: 201 }));
      }
      if (url === "/api/terminal/sessions/terminal-2" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-2",
            title: "terminal-2",
            terminal_session_id: "terminal-2",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0/.alter0/workspaces/terminal/sessions/terminal-2",
            created_at: "2026-04-15T10:20:00Z",
            updated_at: "2026-04-15T10:20:00Z",
            turns: [
              {
                id: "turn-2",
                prompt: "describe this image",
                attachments: [
                  {
                    name: "diagram.svg",
                    content_type: "image/svg+xml",
                    data_url: "data:image/svg+xml;base64,PHN2Zy8+",
                  },
                ],
                status: "completed",
                started_at: "2026-04-15T10:20:01Z",
                finished_at: "2026-04-15T10:20:03Z",
                duration_ms: 2000,
                final_output: "done",
                runtime_trace_events: [],
              },
            ],
          },
        }));
      }
      if (url === "/api/terminal/sessions/terminal-2/input" && method === "POST") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-2",
            title: "terminal-2",
            terminal_session_id: "terminal-2",
            status: "busy",
            shell: "codex exec",
            working_dir: "/workspace/alter0/.alter0/workspaces/terminal/sessions/terminal-2",
            created_at: "2026-04-15T10:20:00Z",
            updated_at: "2026-04-15T10:20:01Z",
            turns: [],
          },
        }));
      }
      if (url === "/api/terminal/sessions/terminal-2/attachments" && method === "POST") {
        return Promise.resolve(jsonResponse({
          items: [
            {
              id: "asset-terminal-2",
              name: "diagram.svg",
              content_type: "image/svg+xml",
              size: 32,
              asset_url: "/api/terminal/sessions/terminal-2/attachments/asset-terminal-2/original",
              preview_url: "/api/terminal/sessions/terminal-2/attachments/asset-terminal-2/preview",
            },
          ],
        }));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    const { container } = renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-composer-input='terminal']")).toBeInTheDocument();
    });

    const fileInput = container.querySelector('input[type="file"]') as HTMLInputElement | null;
    expect(fileInput).not.toBeNull();
    const file = new File(
      ['<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12"><rect width="12" height="12" fill="#000"/></svg>'],
      "diagram.svg",
      { type: "image/svg+xml" },
    );

    fireEvent.change(fileInput!, { target: { files: [file] } });
    await waitFor(() => {
      expect(screen.getByAltText("diagram.svg")).toBeInTheDocument();
    });

    fireEvent.change(document.querySelector("[data-runtime-composer-input='terminal']") as HTMLTextAreaElement, {
      target: { value: "describe this image" },
    });
    fireEvent.click(document.querySelector("[data-runtime-composer-submit='terminal']") as HTMLButtonElement);

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-turn='turn-2']")).toBeInTheDocument();
    });

    const fetchMock = vi.mocked(fetch);
    const inputCall = fetchMock.mock.calls.find(([request, init]) =>
      String(request) === "/api/terminal/sessions/terminal-2/input"
      && String(init?.method || "GET").toUpperCase() === "POST");
    expect(inputCall).toBeTruthy();
    const payload = JSON.parse(String((inputCall?.[1] as RequestInit | undefined)?.body || "{}"));
    expect(payload.attachments).toHaveLength(1);
    expect(payload.attachments[0]).toMatchObject({
      id: "asset-terminal-2",
      asset_url: "/api/terminal/sessions/terminal-2/attachments/asset-terminal-2/original",
      preview_url: "/api/terminal/sessions/terminal-2/attachments/asset-terminal-2/preview",
    });
    expect(payload.attachments[0].data_url).toBeUndefined();
  });

  it("marks the first send as pending immediately while the terminal session is still being created", async () => {
    let resolveCreateSession: ((value: Response) => void) | null = null;

    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/terminal/sessions" && method === "POST") {
        return new Promise<Response>((resolve) => {
          resolveCreateSession = resolve;
        });
      }
      if (url === "/api/terminal/sessions/terminal-2/input" && method === "POST") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-2",
            title: "terminal-2",
            terminal_session_id: "terminal-2",
            status: "busy",
            shell: "codex exec",
            working_dir: "/workspace/alter0/.alter0/workspaces/terminal/sessions/terminal-2",
            created_at: "2026-04-15T10:20:00Z",
            updated_at: "2026-04-15T10:20:01Z",
            turns: [],
          },
        }));
      }
      if (url === "/api/terminal/sessions/terminal-2" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-2",
            title: "terminal-2",
            terminal_session_id: "terminal-2",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0/.alter0/workspaces/terminal/sessions/terminal-2",
            created_at: "2026-04-15T10:20:00Z",
            updated_at: "2026-04-15T10:20:01Z",
            turns: [],
          },
        }));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-composer-input='terminal']")).toBeInTheDocument();
    });

    fireEvent.change(document.querySelector("[data-runtime-composer-input='terminal']") as HTMLTextAreaElement, {
      target: { value: "pwd" },
    });
    fireEvent.click(document.querySelector("[data-runtime-composer-submit='terminal']") as HTMLButtonElement);

    expect(document.querySelector("[data-runtime-composer-submit='terminal']")).toBeDisabled();
    expect(document.querySelector("[data-runtime-composer-submit='terminal']")).toHaveAttribute("aria-label", "Sending...");

    resolveCreateSession?.(jsonResponse({
      session: {
        id: "terminal-2",
        title: "terminal-2",
        terminal_session_id: "terminal-2",
        status: "ready",
        shell: "codex exec",
        working_dir: "/workspace/alter0/.alter0/workspaces/terminal/sessions/terminal-2",
        created_at: "2026-04-15T10:20:00Z",
        updated_at: "2026-04-15T10:20:00Z",
      },
    }, { status: 201 }));

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-workspace='terminal']")).toHaveAttribute(
        "data-runtime-session-id",
        "terminal-2",
      );
    });
  });

  it("submits immediately when the mobile send button is tapped", async () => {
    renderTerminalRouteBody({
      isMobileViewport: true,
    });

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-composer-input='terminal']")).toBeInTheDocument();
    });

    const input = document.querySelector("[data-runtime-composer-input='terminal']") as HTMLTextAreaElement;
    fireEvent.focus(input);
    fireEvent.change(input, {
      target: { value: "pwd" },
    });
    fireEvent.touchStart(document.querySelector("[data-runtime-composer-submit='terminal']") as HTMLButtonElement);

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-composer-input='terminal']")).toHaveValue("");
    });

    const fetchMock = vi.mocked(fetch);
    expect(fetchMock.mock.calls.some(([request, init]) =>
      String(request) === "/api/terminal/sessions/terminal-1/input"
      && String(init?.method || "GET").toUpperCase() === "POST")).toBe(true);
  });

  it("refreshes the terminal list and active session when the page becomes visible again", async () => {
    let visibilityState = "visible";
    Object.defineProperty(document, "visibilityState", {
      configurable: true,
      get: () => visibilityState,
    });
    Object.defineProperty(document, "hidden", {
      configurable: true,
      get: () => visibilityState !== "visible",
    });

    let listLoads = 0;
    let sessionLoads = 0;
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        listLoads += 1;
        return Promise.resolve(jsonResponse({
          items: [
            {
              id: "terminal-1",
              title: listLoads > 1 ? "Workspace shell refreshed" : "Workspace shell",
              terminal_session_id: "terminal-1",
              status: "ready",
              shell: "codex exec",
              working_dir: "/workspace/alter0",
              created_at: "2026-04-15T10:00:00Z",
              updated_at: "2026-04-15T10:10:00Z",
            },
          ],
        }));
      }
      if (url === "/api/control/skills" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/terminal/sessions/terminal-1" && method === "GET") {
        sessionLoads += 1;
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-1",
            title: sessionLoads > 1 ? "Workspace shell refreshed" : "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
            turns: [],
          },
        }));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    renderTerminalRouteBody();

    await waitFor(() => {
      expect(screen.getAllByText("Workspace shell").length).toBeGreaterThan(0);
    });

    visibilityState = "hidden";
    fireEvent(document, new Event("visibilitychange"));
    visibilityState = "visible";
    fireEvent(document, new Event("visibilitychange"));

    await waitFor(() => expect(listLoads).toBeGreaterThan(1));
    await waitFor(() => expect(sessionLoads).toBeGreaterThan(1));
    await waitFor(() => {
      expect(screen.getAllByText("Workspace shell refreshed").length).toBeGreaterThan(0);
    });
  });

  it("merges a paged active-session refresh into existing terminal turns", async () => {
    let sessionLoads = 0;
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return Promise.resolve(jsonResponse({
          items: [{
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
          }],
        }));
      }
      if (url === "/api/control/skills" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/terminal/sessions/terminal-1" && method === "GET") {
        sessionLoads += 1;
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
            turns_paging: {
              has_more_before: true,
              oldest_turn_id: sessionLoads > 1 ? "turn-2" : "turn-1",
              newest_turn_id: "turn-2",
            },
            turns: sessionLoads > 1
              ? [{ id: "turn-2", prompt: "newer", status: "completed", final_output: "newer updated", runtime_trace_events: [] }]
              : [
                  { id: "turn-1", prompt: "older", status: "completed", final_output: "older output", runtime_trace_events: [] },
                  { id: "turn-2", prompt: "newer", status: "completed", final_output: "newer output", runtime_trace_events: [] },
                ],
          },
        }));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    renderTerminalRouteBody();

    await waitFor(() => expect(screen.getByText("older output")).toBeInTheDocument());
    await waitFor(() => expect(screen.getByText("newer output")).toBeInTheDocument());

    window.dispatchEvent(new PageTransitionEvent("pageshow", { persisted: true }));

    await waitFor(() => expect(screen.getByText("newer updated")).toBeInTheDocument());
    expect(screen.getByText("older output")).toBeInTheDocument();
  });

  it("progressively loads earlier Terminal turn pages after the latest session detail", async () => {
    const requests: string[] = [];
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      requests.push(`${method} ${url}`);
      if (url === "/api/terminal/sessions" && method === "GET") {
        return Promise.resolve(jsonResponse({
          items: [{
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
          }],
        }));
      }
      if (url === "/api/control/skills" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/terminal/sessions/terminal-1" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
            turns_paging: {
              has_more_before: true,
              oldest_turn_id: "turn-3",
              newest_turn_id: "turn-3",
              next_before_turn_id: "turn-3",
            },
            turns: [{
              id: "turn-3",
              prompt: "latest",
              status: "completed",
              started_at: "2026-04-15T10:03:00Z",
              finished_at: "2026-04-15T10:03:02Z",
              duration_ms: 2000,
              final_output: "latest output",
              runtime_trace_events: [],
            }],
          },
        }));
      }
      if (url === "/api/terminal/sessions/terminal-1?turn_before=turn-3&turn_limit=20" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
            turns_paging: {
              has_more_before: false,
              oldest_turn_id: "turn-1",
              newest_turn_id: "turn-2",
            },
            turns: [
              {
                id: "turn-1",
                prompt: "older",
                status: "completed",
                started_at: "2026-04-15T10:01:00Z",
                finished_at: "2026-04-15T10:01:02Z",
                duration_ms: 2000,
                final_output: "older output",
                runtime_trace_events: [],
              },
              {
                id: "turn-2",
                prompt: "middle",
                status: "completed",
                started_at: "2026-04-15T10:02:00Z",
                finished_at: "2026-04-15T10:02:02Z",
                duration_ms: 2000,
                final_output: "middle output",
                runtime_trace_events: [],
              },
            ],
          },
        }));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    renderTerminalRouteBody();

    await waitFor(() => expect(screen.getByText("latest output")).toBeInTheDocument());
    await waitFor(() => expect(screen.getByText("older output")).toBeInTheDocument());
    expect(screen.getByText("middle output")).toBeInTheDocument();
    expect(screen.getByText("latest output")).toBeInTheDocument();
    expect(requests).toContain("GET /api/terminal/sessions/terminal-1?turn_before=turn-3&turn_limit=20");
  });

  it("does not repeatedly reload the same Terminal history page when a background page makes no progress", async () => {
    let backgroundPageLoads = 0;
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return Promise.resolve(jsonResponse({
          items: [{
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
          }],
        }));
      }
      if (url === "/api/control/skills" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/terminal/sessions/terminal-1" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
            turns_paging: {
              has_more_before: true,
              oldest_turn_id: "turn-3",
              newest_turn_id: "turn-3",
              next_before_turn_id: "turn-3",
            },
            turns: [{
              id: "turn-3",
              prompt: "latest",
              status: "completed",
              started_at: "2026-04-15T10:03:00Z",
              finished_at: "2026-04-15T10:03:02Z",
              duration_ms: 2000,
              final_output: "latest output",
              runtime_trace_events: [],
            }],
          },
        }));
      }
      if (url === "/api/terminal/sessions/terminal-1?turn_before=turn-3&turn_limit=20" && method === "GET") {
        backgroundPageLoads += 1;
        if (backgroundPageLoads > 1) {
          return Promise.reject(new Error("reloaded the same Terminal history page"));
        }
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
            turns: [],
          },
        }));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    renderTerminalRouteBody();

    await waitFor(() => expect(screen.getByText("latest output")).toBeInTheDocument());
    await waitFor(() => expect(backgroundPageLoads).toBe(1));
    await act(async () => {
      await new Promise((resolve) => window.setTimeout(resolve, 50));
    });

    expect(backgroundPageLoads).toBe(1);
  });

  it("keeps loaded Terminal history when an input response returns only the new turn", async () => {
    let sessionLoads = 0;
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return Promise.resolve(jsonResponse({
          items: [{
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
          }],
        }));
      }
      if (url === "/api/control/skills" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/terminal/sessions/terminal-1" && method === "GET") {
        sessionLoads += 1;
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:11:00Z",
            turns: sessionLoads > 1
              ? [{
                  id: "turn-3",
                  prompt: "continue",
                  status: "completed",
                  started_at: "2026-04-15T10:11:00Z",
                  finished_at: "2026-04-15T10:11:02Z",
                  duration_ms: 2000,
                  final_output: "new output",
                  runtime_trace_events: [],
                }]
              : [
                  {
                    id: "turn-1",
                    prompt: "older",
                    status: "completed",
                    started_at: "2026-04-15T10:01:00Z",
                    finished_at: "2026-04-15T10:01:02Z",
                    duration_ms: 2000,
                    final_output: "older output",
                    runtime_trace_events: [],
                  },
                  {
                    id: "turn-2",
                    prompt: "latest",
                    status: "completed",
                    started_at: "2026-04-15T10:02:00Z",
                    finished_at: "2026-04-15T10:02:02Z",
                    duration_ms: 2000,
                    final_output: "latest output",
                    runtime_trace_events: [],
                  },
                ],
          },
        }));
      }
      if (url === "/api/terminal/sessions/terminal-1/input" && method === "POST") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:11:00Z",
            turns: [{
              id: "turn-3",
              prompt: "continue",
              status: "completed",
              started_at: "2026-04-15T10:11:00Z",
              finished_at: "2026-04-15T10:11:02Z",
              duration_ms: 2000,
              final_output: "new output",
              runtime_trace_events: [],
            }],
          },
        }));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    renderTerminalRouteBody();

    await waitFor(() => expect(screen.getByText("older output")).toBeInTheDocument());
    await waitFor(() => expect(screen.getByText("latest output")).toBeInTheDocument());

    fireEvent.change(document.querySelector("[data-runtime-composer-input='terminal']") as HTMLTextAreaElement, {
      target: { value: "continue" },
    });
    fireEvent.click(document.querySelector("[data-runtime-composer-submit='terminal']") as HTMLButtonElement);

    await waitFor(() => expect(screen.getByText("new output")).toBeInTheDocument());
    expect(screen.getByText("older output")).toBeInTheDocument();
    expect(screen.getByText("latest output")).toBeInTheDocument();
  });

  it("marks the terminal composer input as plain text so mobile autofill bars stay off", async () => {
    renderTerminalRouteBody({
      isMobileViewport: true,
    });

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-composer-input='terminal']")).toBeInTheDocument();
    });

    const input = document.querySelector("[data-runtime-composer-input='terminal']") as HTMLTextAreaElement;

    expect(input).toHaveAttribute("autocomplete", "off");
    expect(input).toHaveAttribute("autocorrect", "off");
    expect(input).toHaveAttribute("autocapitalize", "off");
    expect(input).toHaveAttribute("enterkeyhint", "send");
    expect(input).toHaveAttribute("spellcheck", "false");
  });

  it("submits when the mobile send button is pressed through touch pointer while the composer stays focused", async () => {
    renderTerminalRouteBody({
      isMobileViewport: true,
    });

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-composer-input='terminal']")).toBeInTheDocument();
    });

    const input = document.querySelector("[data-runtime-composer-input='terminal']") as HTMLTextAreaElement;
    fireEvent.focus(input);
    fireEvent.change(input, {
      target: { value: "pwd" },
    });
    fireEvent.pointerDown(
      document.querySelector("[data-runtime-composer-submit='terminal']") as HTMLButtonElement,
      { pointerType: "touch" },
    );

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-composer-input='terminal']")).toHaveValue("");
    });

    const fetchMock = vi.mocked(fetch);
    expect(fetchMock.mock.calls.some(([request, init]) =>
      String(request) === "/api/terminal/sessions/terminal-1/input"
      && String(init?.method || "GET").toUpperCase() === "POST")).toBe(true);
  });

  it("lets terminal users choose public skills for the next input", async () => {
    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-session-select='terminal-1']")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Session" }));

    const configPanel = await screen.findByTestId("terminal-skill-selector");
    expect(within(configPanel).getByLabelText("Preview Publish")).toBeChecked();
    expect(within(configPanel).getByLabelText("Frontend Design")).toBeChecked();
    expect(within(configPanel).getByLabelText("Summary")).toBeChecked();
    expect(within(configPanel).getByLabelText("Memory")).toBeChecked();
    expect(within(configPanel).getByText("Summary")).toBeInTheDocument();
    expect(within(configPanel).queryByText("Private")).not.toBeInTheDocument();

    fireEvent.change(document.querySelector("[data-runtime-composer-input='terminal']") as HTMLTextAreaElement, {
      target: { value: "summarize this workspace" },
    });
    fireEvent.click(document.querySelector("[data-runtime-composer-submit='terminal']") as HTMLButtonElement);

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-composer-input='terminal']")).toHaveValue("");
    });

    const fetchMock = vi.mocked(fetch);
    const inputCall = fetchMock.mock.calls.find(([request, init]) =>
      String(request) === "/api/terminal/sessions/terminal-1/input"
      && String(init?.method || "GET").toUpperCase() === "POST");
    expect(inputCall).toBeTruthy();
    const payload = JSON.parse(String((inputCall?.[1] as RequestInit | undefined)?.body || "{}"));
    expect(payload.skill_ids).toEqual(["preview-publish", "frontend-design", "summary", "memory"]);
  });

  it("dismisses the terminal session panel when the composer input is pressed", async () => {
    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-composer-input='terminal']")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Session" }));
    expect(await screen.findByTestId("terminal-skill-selector")).toBeInTheDocument();

    fireEvent.pointerDown(
      document.querySelector("[data-runtime-composer-input='terminal']") as HTMLTextAreaElement,
      { pointerType: "mouse" },
    );

    await waitFor(() => {
      expect(screen.queryByTestId("terminal-skill-selector")).not.toBeInTheDocument();
    });
  });

  it("opens the terminal session panel on the first mobile touch while the composer is focused", async () => {
    renderTerminalRouteBody({
      isMobileViewport: true,
    });

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-composer-input='terminal']")).toBeInTheDocument();
    });

    const input = document.querySelector("[data-runtime-composer-input='terminal']") as HTMLTextAreaElement;
    fireEvent.focus(input);
    fireEvent.touchStart(screen.getByRole("button", { name: "Session" }));

    await waitFor(() => {
      expect(screen.getByTestId("terminal-skill-selector")).toBeInTheDocument();
    });
  });

  it("does not refresh a ready session while the terminal output is being scrolled", async () => {
    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-turn='turn-1']")).toBeInTheDocument();
    });

    vi.useFakeTimers();

    const fetchMock = vi.mocked(fetch);
    const initialCallCount = fetchMock.mock.calls.length;
    const chatScreen = document.querySelector("[data-runtime-screen='terminal']") as HTMLDivElement;

    Object.defineProperty(chatScreen, "scrollHeight", {
      configurable: true,
      value: 1200,
    });
    Object.defineProperty(chatScreen, "clientHeight", {
      configurable: true,
      value: 360,
    });
    Object.defineProperty(chatScreen, "scrollTop", {
      configurable: true,
      value: 240,
      writable: true,
    });

    await act(async () => {
      fireEvent.scroll(chatScreen);
    });

    await act(async () => {
      await vi.advanceTimersByTimeAsync(6100);
    });
    expect(fetchMock.mock.calls).toHaveLength(initialCallCount);
    expect(chatScreen.scrollTop).toBe(240);
    expect(chatScreen.scrollTop).toBe(240);
    expect(document.querySelector("[data-terminal-turn='turn-1']")).toBeInTheDocument();
  });

  it("opens an existing terminal session pinned to the output bottom", async () => {
    const originalScrollHeight = Object.getOwnPropertyDescriptor(HTMLElement.prototype, "scrollHeight");
    const originalScrollTop = Object.getOwnPropertyDescriptor(HTMLElement.prototype, "scrollTop");
    const scrollTopByElement = new WeakMap<Element, number>();
    Object.defineProperty(HTMLElement.prototype, "scrollHeight", {
      configurable: true,
      get() {
        return this instanceof HTMLElement && this.getAttribute("data-runtime-screen") === "terminal" ? 920 : 0;
      },
    });
    Object.defineProperty(HTMLElement.prototype, "scrollTop", {
      configurable: true,
      get() {
        return scrollTopByElement.get(this) || 0;
      },
      set(value) {
        scrollTopByElement.set(this, Number(value || 0));
      },
    });

    try {
      renderTerminalRouteBody();

      await waitFor(() => {
        expect(document.querySelector("[data-terminal-turn='turn-1']")).toBeInTheDocument();
      });

      const chatScreen = document.querySelector("[data-runtime-screen='terminal']") as HTMLDivElement;
      await waitFor(() => expect(chatScreen.scrollTop).toBe(920));
    } finally {
      if (originalScrollHeight) {
        Object.defineProperty(HTMLElement.prototype, "scrollHeight", originalScrollHeight);
      } else {
        delete (HTMLElement.prototype as { scrollHeight?: number }).scrollHeight;
      }
      if (originalScrollTop) {
        Object.defineProperty(HTMLElement.prototype, "scrollTop", originalScrollTop);
      } else {
        delete (HTMLElement.prototype as { scrollTop?: number }).scrollTop;
      }
    }
  });

  it("targets the visible turn for previous and the real next turn when only one terminal turn is visible", async () => {
    installImmediateAnimationFrame();
    stubTerminalTurnsFetch([
      { id: "turn-1" },
      { id: "turn-2" },
      { id: "turn-3" },
    ]);

    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-turn='turn-3']")).toBeInTheDocument();
    });

    const chatScreen = document.querySelector("[data-runtime-screen='terminal']") as HTMLDivElement;
    applyTerminalTurnMetrics(chatScreen, [
      { id: "turn-1", top: 0, height: 180 },
      { id: "turn-2", top: 200, height: 180 },
      { id: "turn-3", top: 400, height: 180 },
    ], {
      clientHeight: 150,
      scrollHeight: 620,
      scrollTop: 220,
    });

    fireEvent.scroll(chatScreen);

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-jump-prev]")).toHaveAttribute("data-terminal-jump-target", "turn-2");
      expect(document.querySelector("[data-terminal-jump-next]")).toHaveAttribute("data-terminal-jump-target", "turn-3");
    });
  });

  it("targets the first and last visible turns when multiple terminal turns share the viewport", async () => {
    installImmediateAnimationFrame();
    stubTerminalTurnsFetch([
      { id: "turn-1" },
      { id: "turn-2" },
      { id: "turn-3" },
    ]);

    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-turn='turn-3']")).toBeInTheDocument();
    });

    const chatScreen = document.querySelector("[data-runtime-screen='terminal']") as HTMLDivElement;
    applyTerminalTurnMetrics(chatScreen, [
      { id: "turn-1", top: 0, height: 180 },
      { id: "turn-2", top: 200, height: 180 },
      { id: "turn-3", top: 400, height: 180 },
    ], {
      clientHeight: 140,
      scrollHeight: 620,
      scrollTop: 170,
    });

    fireEvent.scroll(chatScreen);

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-jump-prev]")).toHaveAttribute("data-terminal-jump-target", "turn-1");
      expect(document.querySelector("[data-terminal-jump-next]")).toHaveAttribute("data-terminal-jump-target", "turn-2");
    });
  });

  it("hides the next jump control when the last terminal turn is the only visible turn", async () => {
    installImmediateAnimationFrame();
    stubTerminalTurnsFetch([
      { id: "turn-1" },
      { id: "turn-2" },
      { id: "turn-3" },
    ]);

    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-turn='turn-3']")).toBeInTheDocument();
    });

    const chatScreen = document.querySelector("[data-runtime-screen='terminal']") as HTMLDivElement;
    applyTerminalTurnMetrics(chatScreen, [
      { id: "turn-1", top: 0, height: 180 },
      { id: "turn-2", top: 200, height: 180 },
      { id: "turn-3", top: 400, height: 180 },
    ], {
      clientHeight: 140,
      scrollHeight: 620,
      scrollTop: 430,
    });

    fireEvent.scroll(chatScreen);

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-jump-prev]")).toHaveAttribute("data-terminal-jump-target", "turn-3");
    });
    expect(document.querySelector("[data-terminal-jump-next]")).toHaveAttribute("data-terminal-jump-target", "");
    expect(document.querySelector("[data-terminal-jump-next]")).not.toHaveClass("is-visible");
  });

  it("hides the next jump control when the viewport is already pinned to the terminal bottom", async () => {
    installImmediateAnimationFrame();
    stubTerminalTurnsFetch([
      { id: "turn-1" },
      { id: "turn-2" },
      { id: "turn-3" },
    ]);

    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-turn='turn-3']")).toBeInTheDocument();
    });

    const chatScreen = document.querySelector("[data-runtime-screen='terminal']") as HTMLDivElement;
    applyTerminalTurnMetrics(chatScreen, [
      { id: "turn-1", top: 0, height: 180 },
      { id: "turn-2", top: 200, height: 180 },
      { id: "turn-3", top: 400, height: 180 },
    ], {
      clientHeight: 220,
      scrollHeight: 580,
      scrollTop: 360,
    });

    fireEvent.scroll(chatScreen);

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-jump-prev]")).toHaveAttribute("data-terminal-jump-target", "turn-2");
    });
    expect(document.querySelector("[data-terminal-jump-next]")).toHaveAttribute("data-terminal-jump-target", "");
    expect(document.querySelector("[data-terminal-jump-next]")).not.toHaveClass("is-visible");
  });

  it("hides the next jump control once the last turn is already visible even if bottom remains available", async () => {
    installImmediateAnimationFrame();
    stubTerminalTurnsFetch([
      { id: "turn-1" },
      { id: "turn-2" },
      { id: "turn-3" },
    ]);

    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-turn='turn-3']")).toBeInTheDocument();
    });

    const chatScreen = document.querySelector("[data-runtime-screen='terminal']") as HTMLDivElement;
    applyTerminalTurnMetrics(chatScreen, [
      { id: "turn-1", top: 0, height: 180 },
      { id: "turn-2", top: 200, height: 180 },
      { id: "turn-3", top: 400, height: 180 },
    ], {
      clientHeight: 180,
      scrollHeight: 720,
      scrollTop: 290,
    });

    fireEvent.scroll(chatScreen);

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-jump-prev]")).toHaveAttribute("data-terminal-jump-target", "turn-2");
    });
    expect(document.querySelector("[data-terminal-jump-bottom]")).toHaveClass("is-visible");
    expect(document.querySelector("[data-terminal-jump-next]")).toHaveAttribute("data-terminal-jump-target", "");
    expect(document.querySelector("[data-terminal-jump-next]")).not.toHaveClass("is-visible");
  });

  it("renders terminal jump controls with the original arrow glyphs", async () => {
    installImmediateAnimationFrame();
    stubTerminalTurnsFetch([
      { id: "turn-1" },
      { id: "turn-2" },
      { id: "turn-3" },
    ]);

    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-turn='turn-3']")).toBeInTheDocument();
    });

    const chatScreen = document.querySelector("[data-runtime-screen='terminal']") as HTMLDivElement;
    applyTerminalTurnMetrics(chatScreen, [
      { id: "turn-1", top: 0, height: 180 },
      { id: "turn-2", top: 200, height: 180 },
      { id: "turn-3", top: 400, height: 180 },
    ], {
      clientHeight: 150,
      scrollHeight: 620,
      scrollTop: 220,
    });

    fireEvent.scroll(chatScreen);

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-jump-prev]")).toHaveAttribute("data-terminal-jump-target", "turn-2");
      expect(document.querySelector("[data-terminal-jump-next]")).toHaveAttribute("data-terminal-jump-target", "turn-3");
    });

    [
      "[data-terminal-jump-top]",
      "[data-terminal-jump-prev]",
      "[data-terminal-jump-next]",
      "[data-terminal-jump-bottom]",
    ].forEach((selector) => {
      const button = document.querySelector(selector) as HTMLButtonElement | null;
      expect(button).not.toBeNull();
      expect(button?.textContent?.trim().length || 0).toBeGreaterThan(0);
    });
  });

  it("hides terminal jump controls while the mobile composer is focused", async () => {
    installImmediateAnimationFrame();
    stubTerminalTurnsFetch([
      { id: "turn-1" },
      { id: "turn-2" },
      { id: "turn-3" },
    ]);

    renderTerminalRouteBody({
      isMobileViewport: true,
    });

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-composer-input='terminal']")).toBeInTheDocument();
    });

    const input = document.querySelector("[data-runtime-composer-input='terminal']") as HTMLTextAreaElement;
    fireEvent.focus(input);

    expect(document.querySelector("[data-terminal-jump-top]")).not.toBeInTheDocument();
    expect(document.querySelector("[data-terminal-jump-prev]")).not.toBeInTheDocument();
    expect(document.querySelector("[data-terminal-jump-next]")).not.toBeInTheDocument();
    expect(document.querySelector("[data-terminal-jump-bottom]")).not.toBeInTheDocument();

    fireEvent.blur(input);

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-jump-top]")).toBeInTheDocument();
      expect(document.querySelector("[data-terminal-jump-prev]")).toBeInTheDocument();
      expect(document.querySelector("[data-terminal-jump-next]")).toBeInTheDocument();
      expect(document.querySelector("[data-terminal-jump-bottom]")).toBeInTheDocument();
    });
  });

  it("hides the next jump control while a newly submitted terminal turn is still in flight", async () => {
    installImmediateAnimationFrame();
    let resolveInput: ((value: Response) => void) | null = null;

    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return Promise.resolve(jsonResponse({
          items: [
            {
              id: "terminal-1",
              title: "Workspace shell",
              terminal_session_id: "terminal-1",
              status: "ready",
              shell: "codex exec",
              working_dir: "/workspace/alter0",
              created_at: "2026-04-15T10:00:00Z",
              updated_at: "2026-04-15T10:10:00Z",
            },
          ],
        }));
      }
      if (url === "/api/control/skills" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/terminal/sessions/terminal-1" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
            turns: [
              {
                id: "turn-1",
                prompt: "prompt-1",
                status: "completed",
                started_at: "2026-04-15T10:00:00Z",
                finished_at: "2026-04-15T10:00:02Z",
                duration_ms: 2000,
                final_output: "output-1",
                runtime_trace_events: [],
              },
              {
                id: "turn-2",
                prompt: "prompt-2",
                status: "completed",
                started_at: "2026-04-15T10:01:00Z",
                finished_at: "2026-04-15T10:01:02Z",
                duration_ms: 2000,
                final_output: "output-2",
                runtime_trace_events: [],
              },
              {
                id: "turn-3",
                prompt: "prompt-3",
                status: "completed",
                started_at: "2026-04-15T10:02:00Z",
                finished_at: "2026-04-15T10:02:02Z",
                duration_ms: 2000,
                final_output: "output-3",
                runtime_trace_events: [],
              },
            ],
          },
        }));
      }
      if (url === "/api/terminal/sessions/terminal-1/input" && method === "POST") {
        return new Promise<Response>((resolve) => {
          resolveInput = resolve;
        });
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-turn='turn-3']")).toBeInTheDocument();
    });

    const chatScreen = document.querySelector("[data-runtime-screen='terminal']") as HTMLDivElement;
    applyTerminalTurnMetrics(chatScreen, [
      { id: "turn-1", top: 0, height: 180 },
      { id: "turn-2", top: 200, height: 180 },
      { id: "turn-3", top: 400, height: 180 },
    ], {
      clientHeight: 150,
      scrollHeight: 720,
      scrollTop: 220,
    });

    fireEvent.change(document.querySelector("[data-runtime-composer-input='terminal']") as HTMLTextAreaElement, {
      target: { value: "continue" },
    });
    fireEvent.click(document.querySelector("[data-runtime-composer-submit='terminal']") as HTMLButtonElement);
    fireEvent.scroll(chatScreen);

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-jump-prev]")).toHaveAttribute("data-terminal-jump-target", "turn-2");
    });
    expect(document.querySelector("[data-terminal-jump-bottom]")).toHaveClass("is-visible");
    expect(document.querySelector("[data-terminal-jump-next]")).toHaveAttribute("data-terminal-jump-target", "");
    expect(document.querySelector("[data-terminal-jump-next]")).not.toHaveClass("is-visible");

    resolveInput?.(jsonResponse({
      session: {
        id: "terminal-1",
        title: "Workspace shell",
        terminal_session_id: "terminal-1",
        status: "busy",
        shell: "codex exec",
        working_dir: "/workspace/alter0",
        created_at: "2026-04-15T10:00:00Z",
        updated_at: "2026-04-15T10:10:30Z",
      },
    }));
  });

  it("reuses cached terminal turn anchors across scroll-only updates", async () => {
    installImmediateAnimationFrame();
    stubTerminalTurnsFetch([
      { id: "turn-1" },
      { id: "turn-2" },
      { id: "turn-3" },
    ]);

    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-turn='turn-3']")).toBeInTheDocument();
    });

    const chatScreen = document.querySelector("[data-runtime-screen='terminal']") as HTMLDivElement;
    const offsetReadCount = { top: 0, height: 0 };
    Object.defineProperty(chatScreen, "clientHeight", {
      configurable: true,
      value: 150,
    });
    Object.defineProperty(chatScreen, "scrollHeight", {
      configurable: true,
      value: 620,
    });
    Object.defineProperty(chatScreen, "scrollTop", {
      configurable: true,
      writable: true,
      value: 220,
    });

    [
      { id: "turn-1", top: 0, height: 180 },
      { id: "turn-2", top: 200, height: 180 },
      { id: "turn-3", top: 400, height: 180 },
    ].forEach((layout) => {
      const node = document.querySelector(`[data-terminal-turn="${layout.id}"]`) as HTMLElement;
      Object.defineProperty(node, "offsetTop", {
        configurable: true,
        get: () => {
          offsetReadCount.top += 1;
          return layout.top;
        },
      });
      Object.defineProperty(node, "offsetHeight", {
        configurable: true,
        get: () => {
          offsetReadCount.height += 1;
          return layout.height;
        },
      });
    });

    fireEvent.scroll(chatScreen);

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-jump-next]")).toHaveAttribute("data-terminal-jump-target", "turn-3");
    });

    const readsAfterFirstSync = { ...offsetReadCount };
    chatScreen.scrollTop = 240;
    fireEvent.scroll(chatScreen);
    await waitFor(() => {
      expect(document.querySelector("[data-terminal-jump-next]")).toHaveAttribute("data-terminal-jump-target", "turn-3");
    });

    expect(offsetReadCount).toEqual(readsAfterFirstSync);
  });

  it("renders a single mobile runtime bar and links its controls to terminal navigation", async () => {
    const toggleMobileNav = vi.fn();
    const toggleMobileSessionPane = vi.fn();
    renderTerminalRouteBody({
      isMobileViewport: true,
      toggleMobileNav,
      toggleMobileSessionPane,
    });

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-session-select='terminal-1']")).toBeInTheDocument();
    });

    const mobileHeader = document.querySelector("[data-runtime-mobile-variant='terminal']") as HTMLElement;
    expect(mobileHeader).toHaveAttribute("data-runtime-mobile-header", "body");
    expect(within(mobileHeader).getByRole("button", { name: "Menu" })).toHaveClass(
      "runtime-workspace-mobile-action",
    );
    expect(within(mobileHeader).queryByRole("button", { name: "Sessions" })).not.toBeInTheDocument();
    expect(mobileHeader.querySelector("[data-runtime-mobile-primary='terminal']")).toHaveClass(
      "runtime-workspace-mobile-action",
    );
    const mobileTitle = mobileHeader.querySelector("[data-runtime-mobile-title='terminal']") as HTMLButtonElement;
    expect(mobileTitle).toBeInTheDocument();
    expect(mobileTitle).toHaveTextContent("Workspace shell");
    expect(mobileTitle.querySelector("[data-runtime-header-signal='ready']")).toBeInTheDocument();
    expect(document.querySelector(".runtime-workspace-head")).toHaveClass("is-mobile-collapsed");

    fireEvent.click(within(mobileHeader).getByRole("button", { name: "Menu" }));
    expect(toggleMobileNav).toHaveBeenCalledTimes(1);

    fireEvent.click(mobileTitle);
    const metaPanel = document.querySelector("[data-runtime-details-panel='terminal']") as HTMLElement;
    expect(metaPanel).toBeInTheDocument();
    expect(within(metaPanel).getByText("/workspace/alter0")).toBeInTheDocument();

    expect(document.querySelector("[data-runtime-session-pane='terminal']")).not.toHaveClass("is-open");
    expect(toggleMobileSessionPane).not.toHaveBeenCalled();
  });

  it("keeps the mobile session pane open after deleting a session from the list", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(true);
    const closeMobileSessionPane = vi.fn();

    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return Promise.resolve(jsonResponse({
          items: [
            {
              id: "terminal-1",
              title: "Workspace shell",
              terminal_session_id: "terminal-1",
              status: "ready",
              shell: "codex exec",
              working_dir: "/workspace/alter0",
              created_at: "2026-04-15T10:00:00Z",
              updated_at: "2026-04-15T10:10:00Z",
            },
            {
              id: "terminal-2",
              title: "Older shell",
              terminal_session_id: "terminal-2",
              status: "ready",
              shell: "codex exec",
              working_dir: "/workspace/alter0",
              created_at: "2026-04-15T09:00:00Z",
              updated_at: "2026-04-15T09:10:00Z",
            },
          ],
        }));
      }
      if (url === "/api/control/skills" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/terminal/sessions/terminal-1" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
            turns: [],
          },
        }));
      }
      if (url === "/api/terminal/sessions/terminal-2" && method === "DELETE") {
        return Promise.resolve(new Response(null, { status: 204 }));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    renderTerminalRouteBody({
      isMobileViewport: true,
      mobileSessionPaneOpen: true,
      closeMobileSessionPane,
    });

    await waitFor(() => {
      expect(document.querySelectorAll("[data-runtime-session-select]")).toHaveLength(2);
    });
    expect(document.querySelector("[data-runtime-session-pane='terminal']")).toHaveClass("is-open");

    const card = openTerminalSessionActions("terminal-2");
    fireEvent.click(within(card).getByRole("menuitem", { name: "Delete", hidden: true }));

    await waitFor(() => {
      expect(document.querySelectorAll("[data-runtime-session-select]")).toHaveLength(1);
    });
    expect(document.querySelector("[data-runtime-session-pane='terminal']")).toHaveClass("is-open");
    expect(document.querySelector("[data-runtime-session-select='terminal-2']")).not.toBeInTheDocument();
    expect(closeMobileSessionPane).not.toHaveBeenCalled();
  });

  it("keeps the mobile session pane open after deleting the active session from the list", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(true);
    const closeMobileSessionPane = vi.fn();

    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
        return Promise.resolve(jsonResponse({
          items: [
            {
              id: "terminal-1",
              title: "Workspace shell",
              terminal_session_id: "terminal-1",
              status: "ready",
              shell: "codex exec",
              working_dir: "/workspace/alter0",
              created_at: "2026-04-15T10:00:00Z",
              updated_at: "2026-04-15T10:10:00Z",
            },
            {
              id: "terminal-2",
              title: "Older shell",
              terminal_session_id: "terminal-2",
              status: "ready",
              shell: "codex exec",
              working_dir: "/workspace/alter0",
              created_at: "2026-04-15T09:00:00Z",
              updated_at: "2026-04-15T09:10:00Z",
            },
          ],
        }));
      }
      if (url === "/api/control/skills" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/terminal/sessions/terminal-1" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-1",
            title: "Workspace shell",
            terminal_session_id: "terminal-1",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T10:00:00Z",
            updated_at: "2026-04-15T10:10:00Z",
            turns: [],
          },
        }));
      }
      if (url === "/api/terminal/sessions/terminal-2" && method === "GET") {
        return Promise.resolve(jsonResponse({
          session: {
            id: "terminal-2",
            title: "Older shell",
            terminal_session_id: "terminal-2",
            status: "ready",
            shell: "codex exec",
            working_dir: "/workspace/alter0",
            created_at: "2026-04-15T09:00:00Z",
            updated_at: "2026-04-15T09:10:00Z",
            turns: [],
          },
        }));
      }
      if (url === "/api/terminal/sessions/terminal-1" && method === "DELETE") {
        return Promise.resolve(new Response(null, { status: 204 }));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    renderTerminalRouteBody({
      isMobileViewport: true,
      mobileSessionPaneOpen: true,
      closeMobileSessionPane,
    });

    await waitFor(() => {
      expect(document.querySelectorAll("[data-runtime-session-select]")).toHaveLength(2);
    });
    expect(document.querySelector("[data-runtime-session-pane='terminal']")).toHaveClass("is-open");

    const card = openTerminalSessionActions("terminal-1");
    fireEvent.click(within(card).getByRole("menuitem", { name: "Delete", hidden: true }));

    await waitFor(() => {
      expect(document.querySelectorAll("[data-runtime-session-select]")).toHaveLength(1);
    });
    expect(document.querySelector("[data-runtime-session-pane='terminal']")).toHaveClass("is-open");
    expect(document.querySelector("[data-runtime-session-select='terminal-1']")).not.toBeInTheDocument();
    expect(document.querySelector("[data-runtime-workspace='terminal']")).toHaveAttribute("data-runtime-session-id", "terminal-2");
    expect(closeMobileSessionPane).not.toHaveBeenCalled();
  });

  it("keeps the mobile session pane mutually exclusive with the menu overlay", async () => {
    const toggleMobileNav = vi.fn();

    renderTerminalRouteBody({
      isMobileViewport: true,
      toggleMobileNav,
    });

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-session-select='terminal-1']")).toBeInTheDocument();
    });

    const mobileHeader = document.querySelector("[data-runtime-mobile-variant='terminal']") as HTMLElement;

    expect(within(mobileHeader).queryByRole("button", { name: "Sessions" })).not.toBeInTheDocument();
    fireEvent.click(within(mobileHeader).getByRole("button", { name: "Menu" }));
    expect(toggleMobileNav).toHaveBeenCalledTimes(1);
    expect(document.querySelector("[data-runtime-session-pane='terminal']")).not.toHaveClass("is-open");

    fireEvent.click(within(mobileHeader).getByRole("button", { name: "Menu" }));

    expect(toggleMobileNav).toHaveBeenCalledTimes(2);
    expect(document.querySelector("[data-runtime-session-pane='terminal']")).not.toHaveClass("is-open");
  });

  it("preserves the native mobile keyboard gesture when the terminal composer input is pressed", async () => {
    renderTerminalRouteBody({
      isMobileViewport: true,
    });

    await waitFor(() => {
      expect(document.querySelector("[data-runtime-composer-input='terminal']")).toBeInTheDocument();
    });

    const input = document.querySelector("[data-runtime-composer-input='terminal']") as HTMLTextAreaElement;
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
});

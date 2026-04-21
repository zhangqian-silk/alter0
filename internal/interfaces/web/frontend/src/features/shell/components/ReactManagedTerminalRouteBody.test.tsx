import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { ReactManagedTerminalRouteBody, resolveTerminalPollPlan } from "./ReactManagedTerminalRouteBody";
import { WorkbenchContext, type WorkbenchContextValue } from "../../../app/WorkbenchContext";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: init.status ?? 200,
    headers: {
      "Content-Type": "application/json",
      ...(init.headers ?? {}),
    },
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
                steps: [
                  {
                    id: "step-1",
                    title: "Inspect workspace",
                    type: "command",
                    status: "completed",
                    duration_ms: 1000,
                    preview: "pwd",
                    has_detail: true,
                  },
                ],
              },
            ],
          },
        }));
      }
      if (url === "/api/terminal/sessions/terminal-1/turns/turn-1/steps/step-1" && method === "GET") {
        return Promise.resolve(jsonResponse({
          step: {
            turn_id: "turn-1",
            blocks: [
              {
                type: "terminal",
                title: "Shell",
                content: "pwd\n/workspace/alter0",
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
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.useRealTimers();
    window.localStorage.clear();
  });

  function renderTerminalRouteBody(overrides: Partial<WorkbenchContextValue> = {}) {
    const contextValue: WorkbenchContextValue = {
      route: "terminal",
      language: "en",
      navigate: vi.fn(),
      isMobileViewport: false,
      mobileNavOpen: false,
      toggleMobileNav: vi.fn(),
      closeMobileNav: vi.fn(),
      ...overrides,
    };

    return render(
      <WorkbenchContext.Provider value={contextValue}>
        <ReactManagedTerminalRouteBody />
      </WorkbenchContext.Provider>,
    );
  }

  it("adapts terminal polling cadence to runtime status and interaction state", () => {
    expect(
      resolveTerminalPollPlan({
        status: "ready",
        pageHidden: false,
        scrollingActive: false,
        inputFocused: false,
      }),
    ).toEqual({
      enabled: true,
      interval: 12000,
      refreshActiveSession: false,
    });

    expect(
      resolveTerminalPollPlan({
        status: "ready",
        pageHidden: false,
        scrollingActive: true,
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

  it("renders the terminal session list and active workspace in React", async () => {
    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-session-select='terminal-1']")).toBeInTheDocument();
    });

    expect(document.querySelector("[data-terminal-view]")).toBeInTheDocument();
    expect(document.querySelector("[data-terminal-session-pane]")).toBeInTheDocument();
    expect(document.querySelector("[data-terminal-workspace]")).toHaveAttribute(
      "data-terminal-session-id",
      "terminal-1",
    );
    await waitFor(() => {
      expect(document.querySelector("[data-terminal-turn='turn-1']")).toBeInTheDocument();
    });
    expect(document.querySelector("[data-terminal-turn='turn-1']")).toBeInTheDocument();
    expect(document.querySelector(".terminal-session-select")).toBeInTheDocument();
    expect(document.querySelector(".terminal-session-list-delete")).toBeInTheDocument();
    expect(document.querySelector("[data-terminal-delete]")).not.toBeInTheDocument();
    expect(document.querySelector("[data-terminal-session-pane]")).toHaveClass("conversation-session-pane");
    expect(document.querySelector(".terminal-session-pane-shell")).toHaveClass("conversation-session-pane-shell");
    expect(document.querySelector("[data-terminal-workspace]")).toHaveClass("conversation-workspace");
    expect(document.querySelector(".terminal-workspace-body")).toBeInTheDocument();
    expect(document.querySelector(".terminal-workspace-body")).toHaveClass("conversation-workspace-body");
    expect(document.querySelector(".terminal-workspace-head")).toBeInTheDocument();
    expect(document.querySelector(".terminal-workspace-head")).toHaveClass("conversation-workspace-head");
    expect(document.querySelector(".terminal-workspace-head")).toHaveClass("is-compact");
    expect(document.querySelector(".terminal-workspace-row")).toHaveClass("conversation-workspace-row", "is-compact");
    expect(document.querySelector(".terminal-workspace-copy")).toHaveClass("conversation-workspace-copy", "is-compact");
    expect(document.querySelector("[data-terminal-console-panel]")).toBeInTheDocument();
    expect(document.querySelector(".terminal-chat-form")).toBeInTheDocument();
    expect(document.querySelector(".terminal-chat-form .terminal-composer-tools")).toBeInTheDocument();
    expect(document.querySelector(".terminal-chat-form .terminal-composer-meta")).toBeInTheDocument();
    expect(document.querySelector("[data-terminal-submit] svg")).toBeInTheDocument();
    expect(document.querySelector("[data-terminal-final-output='turn-1'] .terminal-final-rendered")).toContainHTML(
      "<h1>Workspace</h1>",
    );
    expect(document.querySelector("[data-terminal-final-output='turn-1'] .terminal-final-rendered")).toContainHTML(
      "<li>/workspace/alter0</li>",
    );
  });

  it("loads step detail when expanding a process step", async () => {
    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-step-toggle='step-1']")).toBeInTheDocument();
    });

    fireEvent.click(document.querySelector("[data-terminal-step-toggle='step-1']")!);

    await waitFor(() => {
      expect(document.querySelector(".terminal-step-content code")?.textContent).toBe(
        "pwd\n/workspace/alter0",
      );
    });
  });

  it("creates a new terminal session through the React action bar", async () => {
    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-session-select='terminal-1']")).toBeInTheDocument();
    });

    fireEvent.click(document.querySelector("[data-terminal-create]")!);

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-workspace]")).toHaveAttribute(
        "data-terminal-session-id",
        "terminal-2",
      );
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
    const chatScreen = document.querySelector("[data-terminal-chat-screen]") as HTMLDivElement;

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

  it("renders mobile menu actions and links them to workbench navigation", async () => {
    const toggleMobileNav = vi.fn();
    const closeMobileNav = vi.fn();
    renderTerminalRouteBody({
      isMobileViewport: true,
      toggleMobileNav,
      closeMobileNav,
    });

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-session-select='terminal-1']")).toBeInTheDocument();
    });

    const mobileHeader = document.querySelector("[data-terminal-mobile-header]") as HTMLElement;
    expect(mobileHeader.querySelector(".nav-toggle")).toHaveClass(
      "conversation-mobile-action",
      "terminal-inline-button",
      "is-quiet",
    );
    expect(mobileHeader.querySelector(".panel-toggle")).toHaveClass(
      "conversation-mobile-action",
      "terminal-inline-button",
      "is-quiet",
    );
    expect(mobileHeader.querySelector(".mobile-new-chat")).toHaveClass(
      "conversation-mobile-action",
      "terminal-inline-button",
      "is-primary",
    );
    expect(within(mobileHeader).getByRole("button", { name: "Menu" })).toBeInTheDocument();
    expect(within(mobileHeader).getByRole("button", { name: "Sessions" })).toBeInTheDocument();
    expect(within(mobileHeader).getByRole("button", { name: "New" })).toBeInTheDocument();

    fireEvent.click(within(mobileHeader).getByRole("button", { name: "Menu" }));
    expect(toggleMobileNav).toHaveBeenCalledTimes(1);

    fireEvent.click(within(mobileHeader).getByRole("button", { name: "Sessions" }));
    expect(document.querySelector("[data-terminal-session-pane]")).toHaveClass("is-open");
    expect(closeMobileNav).toHaveBeenCalledTimes(1);
  });
});

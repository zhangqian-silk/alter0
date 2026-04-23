import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { useState } from "react";
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
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.useRealTimers();
    window.localStorage.clear();
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
    expect(document.querySelector(".conversation-session-main")).toBeInTheDocument();
    expect(document.querySelector(".conversation-session-title-row")).toBeInTheDocument();
    expect(document.querySelector(".conversation-session-title-row")?.textContent).toContain("Workspace shell");
    expect(document.querySelector(".conversation-session-summary-row")).toBeInTheDocument();
    expect(document.querySelector(".conversation-session-bottomline")).toBeInTheDocument();
    expect(within(document.querySelector("[data-terminal-session-pane]") as HTMLElement).getByRole("list")).toHaveAttribute(
      "data-terminal-session-list",
      "true",
    );
    expect(within(document.querySelector("[data-terminal-session-pane]") as HTMLElement).getAllByRole("listitem")).toHaveLength(1);
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
    expect(document.querySelector("[data-terminal-close]")).not.toBeInTheDocument();
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
              title: "Older archival session",
              terminal_session_id: "terminal-3",
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
      expect(document.querySelector("[data-terminal-session-select='terminal-1']")).toBeInTheDocument();
    });

    const sessionPane = document.querySelector("[data-terminal-session-pane]") as HTMLElement;
    expect(within(sessionPane).getByText("Today")).toBeInTheDocument();
    expect(within(sessionPane).getByText("Yesterday")).toBeInTheDocument();
    expect(within(sessionPane).getByText("Earlier")).toBeInTheDocument();
    expect(within(sessionPane).getAllByRole("listitem")).toHaveLength(3);
  });

  it("attaches images in terminal composer and submits them with the terminal input payload", async () => {
    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-input]")).toBeInTheDocument();
    });

    const fileInput = document.querySelector('input[type="file"][accept="image/*"]') as HTMLInputElement;
    expect(fileInput).toBeInTheDocument();

    const image = new File(['<svg xmlns="http://www.w3.org/2000/svg"></svg>'], "terminal-shot.svg", { type: "image/svg+xml" });
    fireEvent.change(fileInput, { target: { files: [image] } });

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Preview terminal-shot.svg" })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Preview terminal-shot.svg" }));
    expect(screen.getByRole("dialog", { name: "terminal-shot.svg" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Close preview" }));

    fireEvent.change(document.querySelector("[data-terminal-input]") as HTMLTextAreaElement, {
      target: { value: "inspect screenshot" },
    });
    fireEvent.click(document.querySelector("[data-terminal-submit]") as HTMLButtonElement);

    await waitFor(() => {
      const fetchMock = vi.mocked(fetch);
      expect(fetchMock.mock.calls.some(([request, init]) =>
        String(request) === "/api/terminal/sessions/terminal-1/input"
        && String(init?.method || "GET").toUpperCase() === "POST"
        && JSON.parse(String(init?.body || "{}")).attachments?.length === 1)).toBe(true);
    });
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

  it("submits the first terminal input on the first click even when no session exists yet", async () => {
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
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
                steps: [],
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
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-input]")).toBeInTheDocument();
    });

    fireEvent.change(document.querySelector("[data-terminal-input]") as HTMLTextAreaElement, {
      target: { value: "pwd" },
    });
    fireEvent.click(document.querySelector("[data-terminal-submit]") as HTMLButtonElement);

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-workspace]")).toHaveAttribute(
        "data-terminal-session-id",
        "terminal-2",
      );
    });
    await waitFor(() => {
      expect(document.querySelector("[data-terminal-turn='turn-2']")).toBeInTheDocument();
    });
    expect(document.querySelector("[data-terminal-input]")).toHaveValue("");

    const fetchMock = vi.mocked(fetch);
    expect(fetchMock.mock.calls.some(([request, init]) =>
      String(request) === "/api/terminal/sessions/terminal-2/input"
      && String(init?.method || "GET").toUpperCase() === "POST")).toBe(true);
  });

  it("keeps image attachments on the first terminal input when no session exists yet", async () => {
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/terminal/sessions" && method === "GET") {
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
                steps: [],
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
      return Promise.reject(new Error(`Unhandled fetch: ${method} ${url}`));
    }));

    const { container } = renderTerminalRouteBody();

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-input]")).toBeInTheDocument();
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

    fireEvent.change(document.querySelector("[data-terminal-input]") as HTMLTextAreaElement, {
      target: { value: "describe this image" },
    });
    fireEvent.click(document.querySelector("[data-terminal-submit]") as HTMLButtonElement);

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
      name: "diagram.svg",
      content_type: "image/svg+xml",
    });
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
      expect(document.querySelector("[data-terminal-input]")).toBeInTheDocument();
    });

    fireEvent.change(document.querySelector("[data-terminal-input]") as HTMLTextAreaElement, {
      target: { value: "pwd" },
    });
    fireEvent.click(document.querySelector("[data-terminal-submit]") as HTMLButtonElement);

    expect(document.querySelector("[data-terminal-submit]")).toBeDisabled();
    expect(document.querySelector("[data-terminal-submit]")).toHaveAttribute("aria-label", "Sending...");

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
      expect(document.querySelector("[data-terminal-workspace]")).toHaveAttribute(
        "data-terminal-session-id",
        "terminal-2",
      );
    });
  });

  it("submits immediately when the mobile send button is tapped", async () => {
    renderTerminalRouteBody({
      isMobileViewport: true,
    });

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-input]")).toBeInTheDocument();
    });

    const input = document.querySelector("[data-terminal-input]") as HTMLTextAreaElement;
    fireEvent.focus(input);
    fireEvent.change(input, {
      target: { value: "pwd" },
    });
    fireEvent.touchStart(document.querySelector("[data-terminal-submit]") as HTMLButtonElement);

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-input]")).toHaveValue("");
    });

    const fetchMock = vi.mocked(fetch);
    expect(fetchMock.mock.calls.some(([request, init]) =>
      String(request) === "/api/terminal/sessions/terminal-1/input"
      && String(init?.method || "GET").toUpperCase() === "POST")).toBe(true);
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
    const toggleMobileSessionPane = vi.fn();
    renderTerminalRouteBody({
      isMobileViewport: true,
      toggleMobileNav,
      toggleMobileSessionPane,
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
    expect(toggleMobileSessionPane).toHaveBeenCalledTimes(1);
  });

  it("keeps the mobile session pane mutually exclusive with the menu overlay", async () => {
    const toggleMobileNav = vi.fn();

    renderTerminalRouteBody({
      isMobileViewport: true,
      toggleMobileNav,
    });

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-session-select='terminal-1']")).toBeInTheDocument();
    });

    const mobileHeader = document.querySelector("[data-terminal-mobile-header]") as HTMLElement;

    fireEvent.click(within(mobileHeader).getByRole("button", { name: "Sessions" }));
    expect(document.querySelector("[data-terminal-session-pane]")).toHaveClass("is-open");

    fireEvent.click(within(mobileHeader).getByRole("button", { name: "Menu" }));

    expect(toggleMobileNav).toHaveBeenCalledTimes(1);
    expect(document.querySelector("[data-terminal-session-pane]")).not.toHaveClass("is-open");
  });

  it("uses preventScroll focus when the mobile composer is touched", async () => {
    renderTerminalRouteBody({
      isMobileViewport: true,
    });

    await waitFor(() => {
      expect(document.querySelector("[data-terminal-input]")).toBeInTheDocument();
    });

    const input = document.querySelector("[data-terminal-input]") as HTMLTextAreaElement;
    const focusMock = vi.fn();
    Object.defineProperty(input, "focus", {
      configurable: true,
      value: focusMock,
    });

    fireEvent.pointerDown(input, { pointerType: "touch" });

    expect(focusMock).toHaveBeenCalledWith({ preventScroll: true });
  });
});

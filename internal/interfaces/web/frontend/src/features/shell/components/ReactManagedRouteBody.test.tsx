import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import {
  getReactManagedRouteBodyRoutes,
  isReactManagedRouteBody,
  ReactManagedRouteBody,
} from "./ReactManagedRouteBody";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: init.status ?? 200,
    headers: {
      "Content-Type": "application/json",
      ...(init.headers ?? {}),
    },
  });
}

describe("ReactManagedRouteBody", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders settings as the single management route and switches compact sections without changing paths", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(
        jsonResponse({
          items: [],
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          items: [],
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          items: [],
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          long_term: { exists: false },
          daily: { items: [] },
          mandatory: { exists: false },
          specification: { exists: false },
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          items: [],
          pagination: { page: 1, total: 0, has_next: false },
        }),
      );

    expect(isReactManagedRouteBody("settings")).toBe(true);
    expect(isReactManagedRouteBody("management")).toBe(true);
    expect(isReactManagedRouteBody("memory")).toBe(false);

    window.history.replaceState({}, "", "/settings");
    const { container } = render(<ReactManagedRouteBody route="settings" language="en" />);

    expect(container.querySelector(".management-route-body")).toHaveAttribute("data-management-route", "runtime");
    expect(container.querySelectorAll("[data-management-route-group]")).toHaveLength(1);
    expect(screen.getByRole("button", { name: "Runtime" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("button", { name: "Runtime" }).querySelector(".management-route-tab-icon")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Runtime" }).querySelector(".management-route-tab-shortcut")).toHaveTextContent("RU");

    fireEvent.click(screen.getByRole("button", { name: "Memory" }));

    await waitFor(() => {
      expect(screen.getByRole("tab", { name: "Task History" })).toBeInTheDocument();
    });
    expect(window.location.pathname).toBe("/settings");
    expect(container.querySelector(".management-route-body")).toHaveAttribute("data-management-route", "memory");
    expect(screen.getByRole("button", { name: "Memory" })).toHaveAttribute("aria-current", "page");
    expect(container.querySelector(".management-route-content")).toHaveAttribute("data-management-route-content", "memory");

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/agent/memory",
      expect.objectContaining({ method: "GET" }),
    );
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/memory/tasks?page=1&page_size=10",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("keeps the original service restart and update flow reachable from settings", async () => {
    vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL) => {
      const url = String(input);
      if (url === "/api/control/llm/providers") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/codex/accounts") {
        return Promise.resolve(jsonResponse({
          current: {
            account_name: "local",
            runtime_ready: true,
            runtime_model: "gpt-5.5",
            runtime_reasoning: "high",
          },
          accounts: [],
          models: [{ id: "gpt-5.5", name: "GPT-5.5", enabled: true }],
          reasoning_modes: [{ id: "high", name: "High", enabled: true }],
        }));
      }
      if (url === "/api/control/environments") {
        return Promise.resolve(jsonResponse({
          items: [
            {
              definition: {
                key: "web_addr",
                name: "Web Listen Address",
                module: "Web & Queue",
                description: "Controls the HTTP listen address.",
                type: "string",
                apply_mode: "restart",
                default_value: "127.0.0.1:18088",
                hot_reload: false,
                sensitive: false,
                validation: { required: true },
              },
              value: "127.0.0.1:18088",
              effective_value: "127.0.0.1:18088",
              value_source: "runtime",
              pending_restart: false,
              masked: false,
            },
          ],
        }));
      }
      if (url === "/api/control/environments/audits") {
        return Promise.resolve(jsonResponse({ items: [] }));
      }
      if (url === "/api/control/runtime") {
        return Promise.resolve(jsonResponse({
          started_at: "2026-04-11T05:13:37Z",
          commit_hash: "14f7f84b602f0000000000000000000000000000",
        }));
      }
      return Promise.resolve(jsonResponse({ items: [] }));
    }));

    window.history.replaceState({}, "", "/settings");
    const { container } = render(<ReactManagedRouteBody route="settings" language="en" />);

    fireEvent.click(screen.getByRole("button", { name: "Environments" }));

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Restart Service" })).toBeInTheDocument();
    });

    expect(container.querySelector(".management-route-body")).toHaveAttribute("data-management-route", "environments");
    expect(container.querySelector(".management-route-content")).toHaveAttribute("data-management-route-content", "environments");
    expect(screen.getByRole("button", { name: "Save Changes" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Live Configuration Control" })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Restart Service" }));

    expect(screen.getByRole("dialog", { name: "Restart the service now?" })).toBeInTheDocument();
    expect(screen.getByLabelText("Sync remote master changes before restart")).toBeChecked();
  });

  it("does not treat old management subpages as react-managed route bodies", () => {
    expect(isReactManagedRouteBody("tasks")).toBe(false);
    expect(isReactManagedRouteBody("agent")).toBe(false);
    expect(isReactManagedRouteBody("codex-accounts")).toBe(false);
  });

  it("tracks the full set of routes now owned by React", () => {
    expect(getReactManagedRouteBodyRoutes()).toEqual([
      "settings",
      "management",
      "terminal",
    ]);
    expect(isReactManagedRouteBody("settings")).toBe(true);
    expect(isReactManagedRouteBody("management")).toBe(true);
    expect(isReactManagedRouteBody("agent")).toBe(false);
    expect(isReactManagedRouteBody("memory")).toBe(false);
    expect(isReactManagedRouteBody("products")).toBe(false);
    expect(isReactManagedRouteBody("terminal")).toBe(true);
  });
});

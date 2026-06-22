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

  it("renders settings as the single settings route and switches compact sections without changing paths", async () => {
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
    expect(isReactManagedRouteBody("overview")).toBe(false);
    expect(isReactManagedRouteBody("memory")).toBe(false);

    window.history.replaceState({}, "", "/settings");
    const { container } = render(<ReactManagedRouteBody route="settings" language="en" />);

    expect(container.querySelector(".settings-route-body")).toHaveAttribute("data-settings-route", "runtime");
    expect(container.querySelectorAll("[data-settings-route-group]")).toHaveLength(1);
    expect(screen.getByRole("button", { name: "Runtime" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("button", { name: "Runtime" }).querySelector(".settings-route-tab-icon")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Runtime" }).querySelector(".settings-route-tab-shortcut")).toHaveTextContent("RU");

    fireEvent.click(screen.getByRole("button", { name: "Memory" }));

    await waitFor(() => {
      expect(screen.getByRole("tab", { name: "Task History" })).toBeInTheDocument();
    });
    expect(window.location.pathname).toBe("/settings");
    expect(container.querySelector(".settings-route-body")).toHaveAttribute("data-settings-route", "memory");
    expect(screen.getByRole("button", { name: "Memory" })).toHaveAttribute("aria-current", "page");
    expect(container.querySelector(".settings-route-content")).toHaveAttribute("data-settings-route-content", "memory");

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/memory/context",
      expect.objectContaining({ method: "GET" }),
    );
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/memory/tasks?page=1&page_size=10",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("keeps the service restart flow reachable from runtime settings", async () => {
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
      return Promise.resolve(jsonResponse({ items: [] }));
    }));

    window.history.replaceState({}, "", "/settings");
    render(<ReactManagedRouteBody route="settings" language="en" />);

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Restart service" })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Restart service" }));

    expect(screen.getByRole("dialog", { name: "Restart service?" })).toBeInTheDocument();
    const updateCheckbox = screen.getByRole("checkbox");
    expect(updateCheckbox).not.toBeChecked();

    fireEvent.click(updateCheckbox);
    fireEvent.click(screen.getByRole("button", { name: "Restart" }));

    expect(screen.getByRole("dialog", { name: "Discard local tracked changes?" })).toBeInTheDocument();
  });

  it("does not treat old settings subpages as react-managed route bodies", () => {
    expect(isReactManagedRouteBody("tasks")).toBe(false);
    expect(isReactManagedRouteBody("skill")).toBe(false);
    expect(isReactManagedRouteBody("codex-accounts")).toBe(false);
  });

  it("tracks the full set of routes now owned by React", () => {
    expect(getReactManagedRouteBodyRoutes()).toEqual([
      "settings",
      "terminal",
    ]);
    expect(isReactManagedRouteBody("settings")).toBe(true);
    expect(isReactManagedRouteBody("overview")).toBe(false);
    expect(isReactManagedRouteBody("skill")).toBe(false);
    expect(isReactManagedRouteBody("memory")).toBe(false);
    expect(isReactManagedRouteBody("products")).toBe(false);
    expect(isReactManagedRouteBody("terminal")).toBe(true);
  });
});

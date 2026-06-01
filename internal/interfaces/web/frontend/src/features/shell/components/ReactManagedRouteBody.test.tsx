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

  it("renders management as the single react-managed management route and switches sections without changing paths", async () => {
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

    expect(isReactManagedRouteBody("management")).toBe(true);
    expect(isReactManagedRouteBody("memory")).toBe(false);

    window.history.replaceState({}, "", "/management");
    render(<ReactManagedRouteBody route="management" language="en" />);

    expect(screen.getByRole("button", { name: "Profiles" })).toHaveAttribute("aria-current", "page");

    fireEvent.click(screen.getByRole("button", { name: "Memory" }));

    await waitFor(() => {
      expect(screen.getByRole("tab", { name: "Task History" })).toBeInTheDocument();
    });
    expect(window.location.pathname).toBe("/management");

    expect(fetchMock).toHaveBeenNthCalledWith(
      4,
      "/api/agent/memory",
      expect.objectContaining({ method: "GET" }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      5,
      "/api/memory/tasks?page=1&page_size=10",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("does not treat old management subpages as react-managed route bodies", () => {
    expect(isReactManagedRouteBody("tasks")).toBe(false);
    expect(isReactManagedRouteBody("agent")).toBe(false);
    expect(isReactManagedRouteBody("codex-accounts")).toBe(false);
  });

  it("tracks the full set of routes now owned by React", () => {
    expect(getReactManagedRouteBodyRoutes()).toEqual([
      "management",
      "terminal",
    ]);
    expect(isReactManagedRouteBody("management")).toBe(true);
    expect(isReactManagedRouteBody("agent")).toBe(false);
    expect(isReactManagedRouteBody("memory")).toBe(false);
    expect(isReactManagedRouteBody("products")).toBe(false);
    expect(isReactManagedRouteBody("terminal")).toBe(true);
  });
});

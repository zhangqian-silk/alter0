import { render, screen, waitFor } from "@testing-library/react";
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

  it("treats memory as a react-managed route body", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
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

    expect(isReactManagedRouteBody("memory")).toBe(true);

    render(<ReactManagedRouteBody route="memory" language="en" />);

    await waitFor(() => {
      expect(screen.getByRole("tab", { name: "Task History" })).toBeInTheDocument();
    });

    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      "/api/agent/memory",
      expect.objectContaining({ method: "GET" }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "/api/memory/tasks?page=1&page_size=10",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("treats tasks as a react-managed route body", () => {
    expect(isReactManagedRouteBody("tasks")).toBe(true);
  });

  it("tracks the full set of routes now owned by React", () => {
    expect(getReactManagedRouteBodyRoutes()).toEqual([
      "agent",
      "terminal",
      "products",
      "memory",
      "sessions",
      "tasks",
      "codex-accounts",
      "channels",
      "skills",
      "mcp",
      "models",
      "environments",
      "cron-jobs",
    ]);
    expect(isReactManagedRouteBody("agent")).toBe(true);
    expect(isReactManagedRouteBody("memory")).toBe(true);
    expect(isReactManagedRouteBody("products")).toBe(true);
    expect(isReactManagedRouteBody("codex-accounts")).toBe(true);
    expect(isReactManagedRouteBody("channels")).toBe(true);
    expect(isReactManagedRouteBody("skills")).toBe(true);
    expect(isReactManagedRouteBody("mcp")).toBe(true);
    expect(isReactManagedRouteBody("models")).toBe(true);
    expect(isReactManagedRouteBody("environments")).toBe(true);
    expect(isReactManagedRouteBody("cron-jobs")).toBe(true);
    expect(isReactManagedRouteBody("sessions")).toBe(true);
    expect(isReactManagedRouteBody("tasks")).toBe(true);
    expect(isReactManagedRouteBody("terminal")).toBe(true);
  });
});

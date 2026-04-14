import { render, screen, waitFor } from "@testing-library/react";
import {
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
});

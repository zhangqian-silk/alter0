import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { ReactManagedMemoryRouteBody } from "./ReactManagedMemoryRouteBody";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: init.status ?? 200,
    headers: {
      "Content-Type": "application/json",
      ...(init.headers ?? {}),
    },
  });
}

describe("ReactManagedMemoryRouteBody", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("loads task history and long-term memory tabs from the unified memory APIs", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(
        jsonResponse({
          long_term: {
            exists: true,
            path: "/memory/MEMORY.md",
            updated_at: "2026-03-04T08:00:00Z",
            content: "# Long-Term Memory\n- key: value",
          },
          daily: {
            directory: "/memory/daily",
            items: [],
          },
          mandatory: {
            exists: false,
          },
          specification: {
            exists: false,
          },
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          items: [
            {
              task_id: "task-1",
              task_type: "release",
              goal: "Ship the release notes",
              result: "Published release notes",
              status: "success",
              updated_at: "2026-03-04T09:00:00Z",
              finished_at: "2026-03-04T09:05:00Z",
              tags: ["task", "release"],
            },
          ],
          pagination: {
            page: 1,
            total: 1,
            has_next: false,
          },
        }),
      );

    render(<ReactManagedMemoryRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getByText("task-1")).toBeInTheDocument();
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

    fireEvent.click(screen.getByRole("tab", { name: "Long-Term" }));

    expect(document.querySelector(".memory-content")).toHaveTextContent(
      /# Long-Term Memory\s+- key: value/,
    );
    expect(screen.getByText("/memory/MEMORY.md")).toBeInTheDocument();
  });

  it("opens task detail and loads logs and artifacts on demand", async () => {
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
          items: [
            {
              task_id: "task-1",
              task_type: "release",
              goal: "Ship the release notes",
              result: "Published release notes",
              status: "success",
              updated_at: "2026-03-04T09:00:00Z",
              finished_at: "2026-03-04T09:05:00Z",
              tags: ["task", "release"],
            },
          ],
          pagination: {
            page: 1,
            total: 1,
            has_next: false,
          },
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          meta: {
            task_id: "task-1",
            task_type: "release",
            session_id: "session-task-1",
            source_message_id: "msg-1",
            status: "success",
            progress: 100,
            retry_count: 0,
            updated_at: "2026-03-04T09:00:00Z",
            finished_at: "2026-03-04T09:05:00Z",
          },
          summary_refs: [
            {
              tier: "daily",
              date: "2026-03-04",
              path: "/memory/daily/2026-03-04.md",
            },
          ],
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          items: [
            {
              seq: 1,
              stage: "run",
              level: "success",
              created_at: "2026-03-04T09:00:00Z",
              message: "completed release generation",
            },
          ],
          has_more: false,
          next_cursor: 1,
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          items: [
            {
              artifact_id: "artifact-1",
              artifact_type: "markdown",
              summary: "Release note markdown",
              created_at: "2026-03-04T09:06:00Z",
            },
          ],
        }),
      );

    render(<ReactManagedMemoryRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getByText("task-1")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: /View Detail/i }));

    await waitFor(() => {
      expect(screen.getByText("Task Detail")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Load Logs" }));

    await waitFor(() => {
      expect(screen.getByText("completed release generation")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Load Artifacts" }));

    await waitFor(() => {
      expect(screen.getByText("Release note markdown")).toBeInTheDocument();
    });
  });

  it("renders specification markdown sections inside the specification tab", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(
        jsonResponse({
          long_term: { exists: false },
          daily: { items: [] },
          mandatory: { exists: false },
          specification: {
            exists: true,
            path: "/docs/memory/spec.md",
            updated_at: "2026-03-04T08:00:00Z",
            content: "# Mapping\n- USER.md\n## Rules\n- Keep memory concise",
          },
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          items: [],
          pagination: { page: 1, total: 0, has_next: false },
        }),
      );

    render(<ReactManagedMemoryRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getByRole("tab", { name: "Specification" })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("tab", { name: "Specification" }));

    expect(screen.getByText("Mapping")).toBeInTheDocument();
    expect(screen.getByText("Rules")).toBeInTheDocument();
    expect(screen.getByText("- Keep memory concise")).toBeInTheDocument();
  });
});

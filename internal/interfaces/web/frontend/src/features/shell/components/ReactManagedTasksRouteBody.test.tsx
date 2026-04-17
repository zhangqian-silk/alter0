import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { ReactManagedTasksRouteBody } from "./ReactManagedTasksRouteBody";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: init.status ?? 200,
    headers: {
      "Content-Type": "application/json",
      ...(init.headers ?? {}),
    },
  });
}

describe("ReactManagedTasksRouteBody", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
    vi.stubGlobal(
      "EventSource",
      class {
        close() {}
        addEventListener() {}
      },
    );
    Object.assign(navigator, {
      clipboard: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
    });
    vi.stubGlobal("alert", vi.fn());
    window.sessionStorage.clear();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("loads tasks, reapplies submitted filters, and renders master-detail with logs", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(
        jsonResponse({
          items: [
            {
              task_id: "task-runtime-1",
              session_id: "session-runtime-1",
              status: "running",
              progress: 45,
              trigger_type: "cron",
              channel_type: "scheduler",
              channel_id: "scheduler-default",
              source_message_id: "msg-user-1",
              updated_at: "2026-04-11T01:02:03Z",
              last_heartbeat_at: "2026-04-11T01:00:00Z",
              timeout_at: "2026-04-11T01:10:00Z",
              job_id: "job-nightly",
            },
          ],
          pagination: { page: 1, total: 1, has_next: false },
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          items: [
            {
              task_id: "task-runtime-1",
              session_id: "session-runtime-1",
              status: "running",
              progress: 45,
              trigger_type: "cron",
              channel_type: "scheduler",
              channel_id: "scheduler-default",
              source_message_id: "msg-user-1",
              updated_at: "2026-04-11T01:02:03Z",
            },
          ],
          pagination: { page: 1, total: 1, has_next: false },
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          task: {
            id: "task-runtime-1",
            session_id: "session-runtime-1",
            source_message_id: "msg-user-1",
            message_id: "msg-assistant-1",
            task_type: "artifact",
            status: "running",
            phase: "executing",
            progress: 45,
            retry_count: 1,
            queue_position: 0,
            queue_wait_ms: 0,
            created_at: "2026-04-11T00:59:00Z",
            updated_at: "2026-04-11T01:02:03Z",
            started_at: "2026-04-11T01:00:00Z",
            last_heartbeat_at: "2026-04-11T01:01:00Z",
            timeout_at: "2026-04-11T01:10:00Z",
            request_content: "generate timeline",
            result: {
              output: "- first point\n- second point",
            },
          },
          source: {
            trigger_type: "cron",
            channel_type: "scheduler",
            channel_id: "scheduler-default",
            correlation_id: "corr-runtime-1",
            job_id: "job-nightly",
            job_name: "Nightly Build",
            fired_at: "2026-04-11T00:58:00Z",
          },
          actions: {
            retry: { enabled: true, reason: "", allowed_statuses: ["failed", "canceled"] },
            cancel: { enabled: true, reason: "", allowed_statuses: ["queued", "running"] },
          },
          link: {
            task_id: "task-runtime-1",
            session_id: "session-runtime-1",
            terminal_session_id: "terminal-runtime-1",
            request_message_id: "msg-user-1",
            result_message_id: "msg-assistant-1",
            task_detail_path: "/api/control/tasks/task-runtime-1",
            session_tasks_path: "/api/sessions/session-runtime-1/tasks",
            session_messages_path: "/api/sessions/session-runtime-1/messages",
          },
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          items: [
            { seq: 1, stage: "running", message: "fetching dependencies" },
          ],
          has_more: false,
          next_cursor: 2,
        }),
      );

    const { container } = render(<ReactManagedTasksRouteBody language="en" />);

    await waitFor(() => {
      expect(fetchMock).toHaveBeenNthCalledWith(
        1,
        "/api/control/tasks?page=1&page_size=20",
        expect.objectContaining({ method: "GET" }),
      );
    });

    expect(screen.getByRole("button", { name: "task-runtime-1" })).toBeInTheDocument();
    expect(container.querySelector(".task-summary-list-compact")).not.toBeInTheDocument();
    expect(container.querySelector(".task-summary-list")).toBeInTheDocument();
    expect(screen.getByText("Select a task from the list to inspect runtime detail.")).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Status"), {
      target: { value: "running" },
    });
    fireEvent.change(screen.getByLabelText("Session ID"), {
      target: { value: "session-runtime-1" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Apply" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenNthCalledWith(
        2,
        "/api/control/tasks?page=1&page_size=20&session_id=session-runtime-1&status=running",
        expect.objectContaining({ method: "GET" }),
      );
    });

    fireEvent.click(screen.getByRole("button", { name: "task-runtime-1" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenNthCalledWith(
        3,
        "/api/control/tasks/task-runtime-1",
        expect.objectContaining({ method: "GET" }),
      );
    });

    await waitFor(() => {
      expect(fetchMock).toHaveBeenNthCalledWith(
        4,
        "/api/control/tasks/task-runtime-1/logs?cursor=0&limit=200",
        expect.objectContaining({ method: "GET" }),
      );
    });

    expect(screen.getAllByText("task-runtime-1").length).toBeGreaterThan(1);
    expect(screen.getByText("fetching dependencies")).toBeInTheDocument();
    expect(screen.getByText("Generate timeline version")).toBeInTheDocument();
    expect(screen.getByText("first point")).toBeInTheDocument();
  });

  it("converts datetime-local filters to RFC3339 query values", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(jsonResponse({ items: [], pagination: { page: 1, total: 0, has_next: false } }))
      .mockResolvedValueOnce(jsonResponse({ items: [], pagination: { page: 1, total: 0, has_next: false } }));

    render(<ReactManagedTasksRouteBody language="en" />);

    await waitFor(() => {
      expect(fetchMock).toHaveBeenNthCalledWith(
        1,
        "/api/control/tasks?page=1&page_size=20",
        expect.objectContaining({ method: "GET" }),
      );
    });

    fireEvent.click(screen.getByRole("button", { name: "Show Advanced Filters" }));
    fireEvent.change(screen.getByLabelText("Start"), {
      target: { value: "2026-04-11T09:00" },
    });
    fireEvent.change(screen.getByLabelText("End"), {
      target: { value: "2026-04-11T11:30" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Apply" }));

    const expectedTimeRange = `${new Date("2026-04-11T09:00").toISOString()},${new Date("2026-04-11T11:30").toISOString()}`;

    await waitFor(() => {
      expect(fetchMock).toHaveBeenNthCalledWith(
        2,
        `/api/control/tasks?page=1&page_size=20&time_range=${encodeURIComponent(expectedTimeRange)}`,
        expect.objectContaining({ method: "GET" }),
      );
    });
  });

  it("submits terminal follow-up input and reloads the returned task detail", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(
        jsonResponse({
          items: [
            {
              task_id: "task-root-1",
              session_id: "session-runtime-1",
              status: "running",
              progress: 45,
              trigger_type: "user",
              channel_type: "web",
              channel_id: "web-default",
              updated_at: "2026-04-11T01:02:03Z",
            },
          ],
          pagination: { page: 1, total: 1, has_next: false },
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          task: {
            id: "task-root-1",
            session_id: "session-runtime-1",
            status: "running",
            phase: "executing",
            progress: 45,
            retry_count: 0,
            created_at: "2026-04-11T00:59:00Z",
            updated_at: "2026-04-11T01:02:03Z",
            request_content: "generate timeline",
          },
          source: {
            trigger_type: "user",
            channel_type: "web",
            channel_id: "web-default",
          },
          actions: {
            retry: { enabled: false, reason: "", allowed_statuses: [] },
            cancel: { enabled: true, reason: "", allowed_statuses: ["queued", "running"] },
          },
          link: {
            task_id: "task-root-1",
            session_id: "session-runtime-1",
            terminal_session_id: "terminal-runtime-1",
            task_detail_path: "/api/control/tasks/task-root-1",
          },
        }),
      )
      .mockResolvedValueOnce(jsonResponse({ items: [], has_more: false, next_cursor: 0 }))
      .mockResolvedValueOnce(
        jsonResponse({
          task_id: "task-follow-up-1",
          anchor_task_id: "task-root-1",
        }, { status: 202 }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          task: {
            id: "task-follow-up-1",
            session_id: "session-runtime-1",
            status: "queued",
            phase: "queued",
            progress: 0,
            retry_count: 0,
            created_at: "2026-04-11T01:03:00Z",
            updated_at: "2026-04-11T01:03:00Z",
            request_content: "continue with next step",
          },
          source: {
            trigger_type: "user",
            channel_type: "web",
            channel_id: "web-default",
          },
          actions: {
            retry: { enabled: false, reason: "", allowed_statuses: [] },
            cancel: { enabled: true, reason: "", allowed_statuses: ["queued", "running"] },
          },
          link: {
            task_id: "task-follow-up-1",
            session_id: "session-runtime-1",
            terminal_session_id: "terminal-runtime-1",
            task_detail_path: "/api/control/tasks/task-follow-up-1",
          },
        }),
      )
      .mockResolvedValueOnce(jsonResponse({ items: [], has_more: false, next_cursor: 0 }));

    render(<ReactManagedTasksRouteBody language="en" />);

    await screen.findByRole("button", { name: "task-root-1" });
    fireEvent.click(screen.getByRole("button", { name: "task-root-1" }));

    await screen.findByRole("button", { name: "Send" });
    fireEvent.change(screen.getByPlaceholderText("Type command or prompt..."), {
      target: { value: "continue with next step" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Send" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenNthCalledWith(
        4,
        "/api/control/tasks/task-root-1/terminal/input",
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({
            input: "continue with next step",
            reuse_task: true,
            anchor_task_id: "task-root-1",
          }),
        }),
      );
    });

    await waitFor(() => {
      expect(fetchMock).toHaveBeenNthCalledWith(
        5,
        "/api/control/tasks/task-follow-up-1",
        expect.objectContaining({ method: "GET" }),
      );
    });
  });
});

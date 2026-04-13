import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { ReactManagedSessionsRouteBody } from "./ReactManagedSessionsRouteBody";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: init.status ?? 200,
    headers: {
      "Content-Type": "application/json",
      ...(init.headers ?? {}),
    },
  });
}

describe("ReactManagedSessionsRouteBody", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
    window.sessionStorage.clear();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("loads session history and reapplies submitted filters", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(
        jsonResponse({
          items: [
            {
              session_id: "session-runtime-1",
              channel_type: "web",
              channel_id: "web-default",
              last_message_id: "msg-runtime-1",
              updated_at: "2026-04-13T01:02:03Z",
              created_at: "2026-04-13T00:00:00Z",
              message_count: 3,
              trigger_type: "user",
              job_id: "",
              job_name: "",
              fired_at: "",
            },
          ],
        }),
      )
      .mockResolvedValueOnce(jsonResponse({ items: [] }));

    render(<ReactManagedSessionsRouteBody language="en" />);

    await waitFor(() => {
      expect(fetchMock).toHaveBeenNthCalledWith(
        1,
        "/api/sessions?page=1&page_size=50",
        expect.objectContaining({ method: "GET" }),
      );
    });

    expect(screen.getAllByText("session-runtime-1")).toHaveLength(2);
    expect(screen.getByText("View Detail")).toBeInTheDocument();
    expect(screen.getAllByText("Web").length).toBeGreaterThan(0);

    fireEvent.change(screen.getByLabelText("Trigger Type"), {
      target: { value: "cron" },
    });
    fireEvent.change(screen.getByLabelText("Channel ID"), {
      target: { value: "web-default" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Apply" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenNthCalledWith(
        2,
        "/api/sessions?page=1&page_size=50&trigger_type=cron&channel_id=web-default",
        expect.objectContaining({ method: "GET" }),
      );
    });
  });

  it("restores persisted filters, supports reset, and switches labels with language", async () => {
    const fetchMock = vi.mocked(fetch);
    window.sessionStorage.setItem(
      "alter0.web.sessions.route-filters.v1",
      JSON.stringify({
        triggerType: "system",
        channelType: "scheduler",
        channelID: "sched-default",
        messageID: "msg-123",
        jobID: "job-daily",
      }),
    );
    fetchMock
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
      .mockResolvedValueOnce(jsonResponse({ items: [] }));

    render(<ReactManagedSessionsRouteBody language="zh" />);

    await waitFor(() => {
      expect(fetchMock).toHaveBeenNthCalledWith(
        1,
        "/api/sessions?page=1&page_size=50&trigger_type=system&channel_type=scheduler&channel_id=sched-default&message_id=msg-123&job_id=job-daily",
        expect.objectContaining({ method: "GET" }),
      );
    });

    expect(screen.getByText("暂无会话记录。")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "重置" })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "重置" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenNthCalledWith(
        2,
        "/api/sessions?page=1&page_size=50",
        expect.objectContaining({ method: "GET" }),
      );
    });
  });

  it("renders request failures as route errors", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValueOnce(
      jsonResponse({ error: "session history unavailable" }, { status: 503 }),
    );

    render(<ReactManagedSessionsRouteBody language="en" />);

    await waitFor(() => {
      expect(
        screen.getByText("Load failed: session history unavailable"),
      ).toBeInTheDocument();
    });
  });
});

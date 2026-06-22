import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import {
  ReactManagedControlRouteBody,
  isReactManagedControlRoute,
} from "./ReactManagedControlRouteBody";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: init.status ?? 200,
    headers: {
      "Content-Type": "application/json",
      ...(init.headers ?? {}),
    },
  });
}

describe("ReactManagedControlRouteBody", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("identifies only the settings control routes still owned by React", () => {
    expect(isReactManagedControlRoute("skills")).toBe(true);
    expect(isReactManagedControlRoute("cron-jobs")).toBe(true);
    expect(isReactManagedControlRoute("channels")).toBe(false);
    expect(isReactManagedControlRoute("mcp")).toBe(false);
    expect(isReactManagedControlRoute("models")).toBe(false);
    expect(isReactManagedControlRoute("unknown")).toBe(false);
    expect(isReactManagedControlRoute("tasks")).toBe(false);
  });

  it("renders skills from the control API", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValue(
      jsonResponse({
        items: [
          {
            id: "skill-runtime-1",
            type: "skill",
            name: "Structured Writer",
            description: "Produces concise structured output.",
            scope: "builtin",
            version: "v2",
            enabled: false,
          },
        ],
      }),
    );

    const { rerender } = render(
      <ReactManagedControlRouteBody route="skills" language="en" />,
    );

    await waitFor(() => {
      expect(screen.getByText("Structured Writer")).toBeInTheDocument();
    });

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/control/skills",
      expect.objectContaining({ method: "GET" }),
    );
    expect(screen.getByText("Disabled")).toBeInTheDocument();
    expect(screen.getByText("Produces concise structured output.")).toBeInTheDocument();

    rerender(<ReactManagedControlRouteBody route="skills" language="zh" />);

    expect(screen.getByText("Structured Writer")).toBeInTheDocument();
    expect(screen.getByText("停用")).toBeInTheDocument();
    expect(screen.getByText("名称")).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("renders route-specific empty and error states", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
      .mockResolvedValueOnce(
        jsonResponse({ error: "service unavailable" }, { status: 503 }),
      );

    const { rerender } = render(
      <ReactManagedControlRouteBody route="cron-jobs" language="zh" />,
    );

    await waitFor(() => {
      expect(screen.getByText("暂无定时任务。")).toBeInTheDocument();
    });

    rerender(<ReactManagedControlRouteBody route="skills" language="en" />);

    await waitFor(() => {
      expect(screen.getByText("Load failed: service unavailable")).toBeInTheDocument();
    });
  });

  it("renders cron jobs from the scheduler control API", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValue(
      jsonResponse({
        items: [
          {
            id: "job-daily",
            name: "Daily Summary",
            enabled: true,
            schedule_mode: "daily",
            cron_expression: "30 9 * * *",
            timezone: "Asia/Shanghai",
            task_config: {
              input: "summarize latest tasks",
              retry_limit: 2,
            },
          },
        ],
      }),
    );

    render(<ReactManagedControlRouteBody route="cron-jobs" language="en" />);

    await waitFor(() => {
      expect(screen.getByText("Daily Summary")).toBeInTheDocument();
    });

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/control/cron/jobs",
      expect.objectContaining({ method: "GET" }),
    );
    expect(screen.getByText("30 9 * * *")).toBeInTheDocument();
    expect(screen.getByText("Asia/Shanghai")).toBeInTheDocument();
    expect(screen.getByText("summarize latest tasks")).toBeInTheDocument();
  });

  it("shows builtin cron jobs as protected and allows disabling them", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(
        jsonResponse({
          items: [
            {
              id: "system-memory-maintenance",
              name: "Memory Maintenance",
              enabled: true,
              builtin: true,
              schedule_mode: "daily",
              cron_expression: "10 5 * * *",
              timezone: "Asia/Shanghai",
              task_config: {
                input: "Run system memory maintenance.",
              },
            },
          ],
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          id: "system-memory-maintenance",
          name: "Memory Maintenance",
          enabled: false,
          builtin: true,
          schedule_mode: "daily",
          cron_expression: "10 5 * * *",
          timezone: "Asia/Shanghai",
          task_config: {
            input: "Run system memory maintenance.",
          },
        }),
      );

    render(<ReactManagedControlRouteBody route="cron-jobs" language="en" />);

    await waitFor(() => {
      expect(screen.getByText("Memory Maintenance")).toBeInTheDocument();
    });
    expect(screen.getByText("Built-in")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Disable job" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "/api/control/cron/jobs/system-memory-maintenance",
        expect.objectContaining({
          method: "PUT",
          body: JSON.stringify({ enabled: false }),
        }),
      );
    });
    await waitFor(() => {
      expect(screen.getByText("Disabled")).toBeInTheDocument();
    });
  });
});

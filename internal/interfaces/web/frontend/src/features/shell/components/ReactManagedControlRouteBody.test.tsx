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

  it("separates alter0 built-in skills from the Codex catalog", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValue(
      jsonResponse({
        project_skills: [
          {
            id: "preview-publish",
            name: "Preview Publish",
            description: "Publishes session-scoped previews.",
            configured_enabled: true,
            codex_visible: true,
            sync_status: "ready",
            duplicate: false,
          },
        ],
        codex_skills: [
          {
            name: "frontend-design",
            display_name: "Frontend Design",
            description: "User-installed frontend workflow.",
            enabled: true,
            scope: "user",
            location: "user_agents",
            duplicate: true,
            duplicate_group: "frontend-design",
            dependencies: [{ type: "command", value: "node" }],
          },
          {
            name: "frontend-design",
            description: "Codex home copy.",
            enabled: false,
            scope: "user",
            location: "codex_home",
            duplicate: true,
            duplicate_group: "frontend-design",
            dependencies: [],
          },
        ],
        errors: [
          {
            code: "parse_error",
            message: "Codex could not load a Skill from this location.",
            location: "repo",
          },
        ],
      }),
    );

    const { rerender } = render(
      <ReactManagedControlRouteBody route="skills" language="en" />,
    );

    await waitFor(() => {
      expect(screen.getByText("Preview Publish")).toBeInTheDocument();
    });

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/control/skill-catalog",
      expect.objectContaining({ method: "GET" }),
    );
    expect(screen.getByRole("heading", { name: "Alter0 Built-in" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Codex Skills" })).toBeInTheDocument();
    expect(screen.getByText("Frontend Design")).toBeInTheDocument();
    expect(screen.getAllByText("frontend-design").length).toBeGreaterThanOrEqual(2);
    expect(screen.getByText("~/.agents/skills")).toBeInTheDocument();
    expect(screen.getByText("$CODEX_HOME/skills")).toBeInTheDocument();
    expect(screen.getAllByText("Duplicate name")).toHaveLength(2);
    expect(screen.getByText("Codex could not load a Skill from this location.")).toBeInTheDocument();

    rerender(<ReactManagedControlRouteBody route="skills" language="zh" />);

    expect(screen.getByRole("heading", { name: "Alter0 内置" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Codex Skills" })).toBeInTheDocument();
    expect(screen.getAllByText("名称重复")).toHaveLength(2);
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

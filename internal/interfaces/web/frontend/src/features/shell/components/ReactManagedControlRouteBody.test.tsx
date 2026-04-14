import { render, screen, waitFor } from "@testing-library/react";
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

  it("identifies the control routes now owned by React", () => {
    expect(isReactManagedControlRoute("channels")).toBe(true);
    expect(isReactManagedControlRoute("skills")).toBe(true);
    expect(isReactManagedControlRoute("mcp")).toBe(true);
    expect(isReactManagedControlRoute("models")).toBe(true);
    expect(isReactManagedControlRoute("environments")).toBe(true);
    expect(isReactManagedControlRoute("cron-jobs")).toBe(true);
    expect(isReactManagedControlRoute("tasks")).toBe(false);
  });

  it("fetches and renders the channels card list with legacy-compatible markup", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValue(
      jsonResponse({
        items: [
          {
            id: "channel-runtime-1",
            type: "web",
            description: "Primary web entry for the cockpit shell.",
            enabled: true,
          },
        ],
      }),
    );

    const { container } = render(
      <ReactManagedControlRouteBody route="channels" language="en" />,
    );

    expect(screen.getByText("Loading...")).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getAllByText("channel-runtime-1")).toHaveLength(2);
    });

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/control/channels",
      expect.objectContaining({ method: "GET" }),
    );
    expect(screen.getByText("Enabled")).toBeInTheDocument();
    expect(screen.getByText("Primary web entry for the cockpit shell.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Copy value" })).toHaveAttribute(
      "data-copy-value",
      "channel-runtime-1",
    );
    expect(container.querySelector(".route-card")).toBeInTheDocument();
    expect(container.querySelector(".route-field-value.is-mono")).toHaveTextContent(
      "channel-runtime-1",
    );
  });

  it("keeps fetched control data stable across language rerenders while switching labels", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValue(
      jsonResponse({
        items: [
          {
            id: "skill-runtime-1",
            type: "skill",
            name: "Structured Writer",
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

    rerender(<ReactManagedControlRouteBody route="skills" language="zh" />);

    expect(screen.getByText("Structured Writer")).toBeInTheDocument();
    expect(screen.getByText("停用")).toBeInTheDocument();
    expect(screen.getByText("名称")).toBeInTheDocument();
    expect(screen.getByText("范围")).toBeInTheDocument();
    expect(screen.getByText("版本")).toBeInTheDocument();
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
      <ReactManagedControlRouteBody route="mcp" language="zh" />,
    );

    await waitFor(() => {
      expect(screen.getByText("暂无 MCP 配置。")).toBeInTheDocument();
    });

    rerender(<ReactManagedControlRouteBody route="channels" language="en" />);

    await waitFor(() => {
      expect(screen.getByText("Load failed: service unavailable")).toBeInTheDocument();
    });
  });

  it("renders model providers from the LLM control API", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValue(
      jsonResponse({
        items: [
          {
            id: "openrouter",
            name: "OpenRouter",
            provider_type: "openrouter",
            api_type: "openai-completions",
            base_url: "https://openrouter.ai/api/v1",
            default_model: "openai/gpt-5.4",
            models: [
              { id: "openai/gpt-5.4", is_enabled: true },
              { id: "anthropic/claude-3.7-sonnet", is_enabled: false },
            ],
            is_enabled: true,
          },
        ],
      }),
    );

    render(<ReactManagedControlRouteBody route="models" language="en" />);

    await waitFor(() => {
      expect(screen.getByText("OpenRouter")).toBeInTheDocument();
    });

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/control/llm/providers",
      expect.objectContaining({ method: "GET" }),
    );
    expect(screen.getByText("openai-completions")).toBeInTheDocument();
    expect(screen.getAllByText("openai/gpt-5.4").length).toBeGreaterThan(0);
    expect(screen.getByText("https://openrouter.ai/api/v1")).toBeInTheDocument();
  });

  it("renders environment config items with pending restart status", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValue(
      jsonResponse({
        items: [
          {
            definition: {
              key: "worker_pool_size",
              name: "Worker Pool Size",
              module: "runtime",
              description: "Controls concurrent task workers.",
              type: "integer",
              apply_mode: "restart",
            },
            value: "8",
            effective_value: "4",
            value_source: "control",
            pending_restart: true,
            masked: false,
          },
        ],
      }),
    );

    render(<ReactManagedControlRouteBody route="environments" language="en" />);

    await waitFor(() => {
      expect(screen.getByText("Worker Pool Size")).toBeInTheDocument();
    });

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/control/environments",
      expect.objectContaining({ method: "GET" }),
    );
    expect(screen.getByText("Pending restart")).toBeInTheDocument();
    expect(screen.getByText("worker_pool_size")).toBeInTheDocument();
    expect(screen.getByText("Controls concurrent task workers.")).toBeInTheDocument();
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
});

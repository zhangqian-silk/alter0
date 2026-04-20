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
    expect(container.querySelector('[data-control-route-grid="channels"]')).toBeInTheDocument();
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

  it("renders environments with runtime toolbar, grouped cards, and audits", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValue(
      jsonResponse({ items: [] }),
    );
    fetchMock
      .mockResolvedValueOnce(
        jsonResponse({
          items: [
            {
              definition: {
                key: "web_addr",
                name: "Web Listen Address",
                module: "Web & Queue",
                description: "Controls the HTTP listen address.",
                type: "string",
                apply_mode: "restart",
                default_value: "127.0.0.1:18088",
                hot_reload: false,
                sensitive: false,
                validation: { required: true },
              },
              value: "127.0.0.1:18088",
              effective_value: "127.0.0.1:18088",
              value_source: "runtime",
              pending_restart: false,
              masked: false,
            },
            {
              definition: {
                key: "web_login_password",
                name: "Web Login Password",
                module: "Web & Queue",
                description: "Controls the console password.",
                type: "string",
                apply_mode: "restart",
                default_value: "",
                hot_reload: false,
                sensitive: true,
                validation: { required: false },
              },
              value: "68****7f",
              effective_value: "68****7f",
              value_source: "persisted",
              pending_restart: false,
              masked: true,
            },
          ],
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          items: [
            {
              operator: "tester",
              occurred_at: "2026-04-11T05:13:37Z",
              requires_restart: true,
              changes: [
                {
                  key: "web_addr",
                  old_value: "127.0.0.1:18088",
                  new_value: "0.0.0.0:18088",
                  apply_mode: "restart",
                },
              ],
            },
          ],
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          started_at: "2026-04-11T05:13:37Z",
          commit_hash: "14f7f84b602f0000000000000000000000000000",
        }),
      );

    render(<ReactManagedControlRouteBody route="environments" language="en" />);

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Reveal Sensitive" })).toBeInTheDocument();
    });

    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      "/api/control/environments",
      expect.objectContaining({ method: "GET" }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "/api/control/environments/audits",
      expect.objectContaining({ method: "GET" }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      3,
      "/api/control/runtime",
      expect.objectContaining({ method: "GET" }),
    );

    expect(screen.getByRole("button", { name: "Reload" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Restart Service" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save Changes" })).toBeInTheDocument();
    expect(screen.getByText("Last Restart")).toBeInTheDocument();
    expect(screen.getByText("Commit Hash")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Web & Queue" })).toBeInTheDocument();
    expect(screen.getByText("Web Listen Address")).toBeInTheDocument();
    expect(screen.getByDisplayValue("127.0.0.1:18088")).toBeInTheDocument();
    expect(screen.getByText("Change Audits")).toBeInTheDocument();
    expect(screen.getAllByText("tester").length).toBeGreaterThan(0);
    expect(screen.queryByRole("table", { name: "Environment Config" })).not.toBeInTheDocument();
  });

  it("reloads environments with revealed sensitive values", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(
        jsonResponse({
          items: [
            {
              definition: {
                key: "web_login_password",
                name: "Web Login Password",
                module: "Web & Queue",
                description: "Controls the console password.",
                type: "string",
                apply_mode: "restart",
                default_value: "",
                hot_reload: false,
                sensitive: true,
                validation: { required: false },
              },
              value: "68****7f",
              effective_value: "68****7f",
              value_source: "persisted",
              pending_restart: false,
              masked: true,
            },
          ],
        }),
      )
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
      .mockResolvedValueOnce(
        jsonResponse({
          started_at: "2026-04-11T05:13:37Z",
          commit_hash: "14f7f84b602f0000000000000000000000000000",
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          items: [
            {
              definition: {
                key: "web_login_password",
                name: "Web Login Password",
                module: "Web & Queue",
                description: "Controls the console password.",
                type: "string",
                apply_mode: "restart",
                default_value: "",
                hot_reload: false,
                sensitive: true,
                validation: { required: false },
              },
              value: "super-secret",
              effective_value: "super-secret",
              value_source: "persisted",
              pending_restart: false,
              masked: false,
            },
          ],
        }),
      )
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
      .mockResolvedValueOnce(
        jsonResponse({
          started_at: "2026-04-11T05:13:37Z",
          commit_hash: "14f7f84b602f0000000000000000000000000000",
        }),
      );

    render(<ReactManagedControlRouteBody route="environments" language="en" />);

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Reveal Sensitive" })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Reveal Sensitive" }));

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Hide Sensitive" })).toBeInTheDocument();
    });

    expect(fetchMock).toHaveBeenNthCalledWith(
      4,
      "/api/control/environments?reveal_sensitive=true",
      expect.objectContaining({ method: "GET" }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      5,
      "/api/control/environments/audits?reveal_sensitive=true",
      expect.objectContaining({ method: "GET" }),
    );
    expect(screen.getByDisplayValue("super-secret")).toBeInTheDocument();
  });

  it("submits the runtime restart request from the environments toolbar", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(
        jsonResponse({
          items: [
            {
              definition: {
                key: "web_addr",
                name: "Web Listen Address",
                module: "Web & Queue",
                description: "Controls the HTTP listen address.",
                type: "string",
                apply_mode: "restart",
                default_value: "127.0.0.1:18088",
                hot_reload: false,
                sensitive: false,
                validation: { required: true },
              },
              value: "127.0.0.1:18088",
              effective_value: "127.0.0.1:18088",
              value_source: "runtime",
              pending_restart: false,
              masked: false,
            },
          ],
        }),
      )
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
      .mockResolvedValueOnce(
        jsonResponse({
          started_at: "2026-04-11T05:13:37Z",
          commit_hash: "14f7f84b602f0000000000000000000000000000",
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          accepted: true,
          status: "restarting",
          sync_remote_master: true,
        }),
      );

    render(<ReactManagedControlRouteBody route="environments" language="en" />);

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Restart Service" })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Restart Service" }));

    expect(
      screen.getByLabelText("Sync remote master changes before restart"),
    ).toBeChecked();

    fireEvent.click(screen.getByRole("button", { name: "Confirm Restart" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenNthCalledWith(
        4,
        "/api/control/runtime/restart",
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({ sync_remote_master: true }),
        }),
      );
    });

    expect(
      screen.getByText("Syncing remote master and restarting service..."),
    ).toBeInTheDocument();
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

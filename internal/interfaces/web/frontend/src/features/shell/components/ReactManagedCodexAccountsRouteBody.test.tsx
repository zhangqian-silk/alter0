import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { ReactManagedCodexAccountsRouteBody } from "./ReactManagedCodexAccountsRouteBody";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: init.status ?? 200,
    headers: {
      "Content-Type": "application/json",
      ...(init.headers ?? {}),
    },
  });
}

function runtimeFixture(overrides: Record<string, unknown> = {}) {
  return {
    command: "codex",
    auth_path: "/var/lib/alter0/.codex/auth.json",
    config_path: "/var/lib/alter0/.codex/config.toml",
    has_auth: true,
    has_config: true,
    current: {
      live: {
        auth_mode: "oauth",
        account_name: "qian zhang",
        email: "qian@example.com",
        user_id: "user-work",
        account_id: "acct-work",
        plan: "prolite",
        last_refresh_at: "2026-06-07T14:29:00Z",
      },
      managed: {
        name: "work",
      },
      quota: {
        hourly: { remaining_percent: 78, reset_at: "2026-06-07T15:30:00Z" },
        weekly: { remaining_percent: 61, reset_at: "2026-06-11T02:10:00Z" },
        plan: "prolite",
      },
    },
    profile: "auto-max",
    model: "gpt-5.4",
    reasoning_effort: "high",
    model_origin: {
      key_path: "model",
      file_path: "/var/lib/alter0/.codex/config.toml",
      version: "user",
    },
    reasoning_origin: {
      key_path: "model_reasoning_effort",
      file_path: "/var/lib/alter0/.codex/config.toml",
      version: "user",
    },
    models: [
      {
        id: "gpt-5.4",
        model: "gpt-5.4",
        display_name: "GPT-5.4",
        description: "Balanced coding model",
        is_default: true,
        default_reasoning_effort: "high",
        supported_reasoning_effort: [
          { reasoning_effort: "medium", description: "Faster responses" },
          { reasoning_effort: "high", description: "Balanced depth" },
        ],
      },
      {
        id: "gpt-5.4-mini",
        model: "gpt-5.4-mini",
        display_name: "GPT-5.4 Mini",
        description: "Lower latency model",
        default_reasoning_effort: "medium",
        supported_reasoning_effort: [
          { reasoning_effort: "low", description: "Lowest latency" },
          { reasoning_effort: "medium", description: "Recommended default" },
        ],
      },
    ],
    ...overrides,
  };
}

describe("ReactManagedCodexAccountsRouteBody", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("keeps the runtime shell visible while loading without account switching surfaces", () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockImplementation(() => new Promise(() => {}));

    render(<ReactManagedCodexAccountsRouteBody language="en" />);

    expect(screen.getByText("Codex Runtime")).toBeInTheDocument();
    expect(screen.queryByText("Runtime Configuration")).not.toBeInTheDocument();
    expect(screen.queryByText("Readiness")).not.toBeInTheDocument();
    expect(screen.queryByText("Runtime Diagnostics")).not.toBeInTheDocument();
    expect(screen.queryByText("Managed Accounts")).not.toBeInTheDocument();
    expect(screen.queryByText("Import or Add Account")).not.toBeInTheDocument();
    expect(screen.queryByText("Login Session")).not.toBeInTheDocument();
  });

  it("loads the single codex runtime and shows unregistered LLM provider state", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(jsonResponse(runtimeFixture()))
      .mockResolvedValueOnce(jsonResponse({ items: [] }));

    render(<ReactManagedCodexAccountsRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getByText("Codex Runtime")).toBeInTheDocument();
      expect(screen.getByText("No LLM providers registered. Codex Direct remains available.")).toBeInTheDocument();
    });

    const statusBand = screen.getByText("Codex Runtime").closest(".codex-runtime-status-band");
    expect(statusBand).not.toBeNull();
    const statusQueries = within(statusBand as HTMLElement);
    expect(statusQueries.queryByText("Status")).not.toBeInTheDocument();
    expect(statusQueries.queryByText("Ready")).not.toBeInTheDocument();
    expect(statusQueries.getByText("qian zhang")).toBeInTheDocument();
    expect(statusQueries.getByText("qian@example.com")).toBeInTheDocument();
    expect(statusQueries.getByText("prolite")).toBeInTheDocument();
    expect(statusQueries.getByText("oauth")).toBeInTheDocument();
    expect(statusQueries.getByText("Profile")).toBeInTheDocument();
    expect(statusQueries.getByText("auto-max")).toBeInTheDocument();
    expect(statusQueries.getByText("78%")).toBeInTheDocument();
    expect(statusQueries.getByText("61%")).toBeInTheDocument();
    expect(statusQueries.getByText("2026-06-07 23:30")).toBeInTheDocument();
    expect(statusQueries.getByText("2026-06-11 10:10")).toBeInTheDocument();
    expect(statusQueries.getByLabelText("Model")).toHaveValue("gpt-5.4");
    expect(statusQueries.getByLabelText("Reasoning Depth")).toHaveValue("high");
    expect(statusQueries.getByRole("option", { name: "High" })).toBeInTheDocument();
    expect(statusQueries.queryByText("Balanced depth")).not.toBeInTheDocument();
    expect(statusQueries.queryByRole("button", { name: "Save" })).not.toBeInTheDocument();
    expect(statusBand?.querySelector(".codex-runtime-identity-card")).not.toBeNull();
    expect(statusBand?.querySelector(".codex-runtime-status-pane")).toBeNull();
    expect(statusBand?.querySelector(".codex-runtime-account-pane")).not.toBeNull();
    expect(statusBand?.querySelector(".codex-runtime-account-strip")).toBeNull();
    expect(statusBand?.querySelector(".codex-runtime-summary-item")).toBeNull();
    expect(statusBand?.querySelector(".codex-runtime-kv-select")).not.toBeNull();
    expect(screen.queryByText("Runtime Configuration")).not.toBeInTheDocument();
    expect(document.querySelector(".codex-runtime-inline-config")).toBeNull();
    expect(statusQueries.queryByText("Saved Name")).not.toBeInTheDocument();
    expect(statusQueries.queryByText("Account ID")).not.toBeInTheDocument();
    expect(statusQueries.queryByText("User ID")).not.toBeInTheDocument();
    expect(statusQueries.queryByText("acct-work")).not.toBeInTheDocument();
    expect(statusQueries.queryByText("user-work")).not.toBeInTheDocument();
    expect(screen.queryByText("Codex Identity")).not.toBeInTheDocument();
    expect(document.querySelector(".codex-runtime-overview")).toBeNull();
    expect(document.querySelector(".codex-runtime-workspace")).toBeNull();
    expect(screen.queryByText("Readiness")).not.toBeInTheDocument();
    expect(screen.queryByText("Runtime Diagnostics")).not.toBeInTheDocument();

    expect(screen.getByLabelText("Model")).toHaveValue("gpt-5.4");
    expect(screen.getByLabelText("Reasoning Depth")).toHaveValue("high");
    expect(screen.queryByRole("button", { name: "Import auth.json" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Start Codex Login" })).not.toBeInTheDocument();

    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      "/api/control/codex/runtime",
      expect.objectContaining({ method: "GET" }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "/api/control/llm/providers",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("loads runtime status and provider state in parallel", async () => {
    const fetchMock = vi.mocked(fetch);
    let resolveRuntime: ((value: Response) => void) | null = null;
    let resolveProviders: ((value: Response) => void) | null = null;
    fetchMock.mockImplementation((input: RequestInfo | URL) => {
      const url = String(input);
      if (url === "/api/control/codex/runtime") {
        return new Promise<Response>((resolve) => {
          resolveRuntime = resolve;
        });
      }
      if (url === "/api/control/llm/providers") {
        return new Promise<Response>((resolve) => {
          resolveProviders = resolve;
        });
      }
      if (url === "/api/control/runtime/restart") {
        return Promise.reject(new Error("restart status unavailable"));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${url}`));
    });

    render(<ReactManagedCodexAccountsRouteBody language="en" />);

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(3));
    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      "/api/control/codex/runtime",
      expect.objectContaining({ method: "GET" }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "/api/control/llm/providers",
      expect.objectContaining({ method: "GET" }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      3,
      "/api/control/runtime/restart",
      expect.objectContaining({ method: "GET" }),
    );

    resolveRuntime?.(jsonResponse(runtimeFixture()));
    resolveProviders?.(jsonResponse({ items: [] }));

    await waitFor(() => {
      expect(screen.getByText("No LLM providers registered. Codex Direct remains available.")).toBeInTheDocument();
    });
  });

  it("defaults to updating from remote master and restarts immediately when no tracked changes require confirmation", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(jsonResponse(runtimeFixture()))
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
      .mockResolvedValueOnce(jsonResponse({ status: "idle" }))
      .mockResolvedValueOnce(jsonResponse({ accepted: true, status: "restarting", sync_remote_master: false }, { status: 202 }))
      .mockResolvedValueOnce(jsonResponse({ accepted: true, status: "restarting", sync_remote_master: true }, { status: 202 }));

    render(<ReactManagedCodexAccountsRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getByText("Service controls")).toBeInTheDocument();
    });
    expect(screen.getByText("Restart the running service when runtime settings or deployment state need to be reapplied.")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Restart service" }));
    expect(document.querySelector(".runtime-restart-overlay")).toBeInTheDocument();
    expect(screen.getByRole("dialog", { name: "Restart service?" })).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: /Update from remote master/ })).toBeChecked();
    fireEvent.click(screen.getByRole("checkbox", { name: /Update from remote master/ }));
    fireEvent.click(screen.getByRole("button", { name: "Restart" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenNthCalledWith(
        4,
        "/api/control/runtime/restart",
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({ sync_remote_master: false, confirm_discard_tracked_changes: false }),
        }),
      );
    });
    expect(screen.getByText("Restart accepted. The service will come back online shortly.")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Restart service" }));
    expect(screen.getByRole("checkbox", { name: /Update from remote master/ })).toBeChecked();
    expect(screen.getByText("Fetch and fast-forward when the working tree has no tracked changes, rebuild, then restart.")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Restart" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenNthCalledWith(
        5,
        "/api/control/runtime/restart",
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({ sync_remote_master: true, confirm_discard_tracked_changes: false }),
        }),
      );
    });
    expect(screen.queryByRole("dialog", { name: "Discard local tracked changes?" })).not.toBeInTheDocument();
  });

  it("asks to discard tracked changes only after the restart API requires confirmation", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(jsonResponse(runtimeFixture()))
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
      .mockResolvedValueOnce(jsonResponse({ status: "idle" }))
      .mockResolvedValueOnce(
        jsonResponse(
          {
            code: "runtime_restart_discard_confirmation_required",
            error: "sync remote master requires discard confirmation because tracked working tree changes exist: M README.md",
          },
          { status: 409 },
        ),
      )
      .mockResolvedValueOnce(jsonResponse({ accepted: true, status: "restarting", sync_remote_master: true }, { status: 202 }));

    render(<ReactManagedCodexAccountsRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getByText("Service controls")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Restart service" }));
    expect(screen.getByRole("checkbox", { name: /Update from remote master/ })).toBeChecked();
    fireEvent.click(screen.getByRole("button", { name: "Restart" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenNthCalledWith(
        4,
        "/api/control/runtime/restart",
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({ sync_remote_master: true, confirm_discard_tracked_changes: false }),
        }),
      );
    });
    expect(screen.getByRole("dialog", { name: "Discard local tracked changes?" })).toBeInTheDocument();
    expect(screen.getByText("Updating from remote master will discard tracked local changes before rebuilding. Untracked files are kept.")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Discard and restart" }));
    await waitFor(() => {
      expect(fetchMock).toHaveBeenNthCalledWith(
        5,
        "/api/control/runtime/restart",
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({ sync_remote_master: true, confirm_discard_tracked_changes: true }),
        }),
      );
    });
  });

  it("shows the latest runtime restart failure reason", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(jsonResponse(runtimeFixture()))
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
      .mockResolvedValueOnce(
        jsonResponse({
          status: "failed",
          error: "candidate runtime exited before ready: flag provided but not defined",
          sync_remote_master: true,
          updated_at: "2026-06-23T05:20:00Z",
        }),
      );

    render(<ReactManagedCodexAccountsRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getByText("Last restart")).toBeInTheDocument();
    });
    expect(screen.getByText("Failed and rolled back")).toBeInTheDocument();
    expect(screen.getByText("Remote master sync requested")).toBeInTheDocument();
    expect(screen.getByText("Failure reason: candidate runtime exited before ready: flag provided but not defined")).toBeInTheDocument();
  });

  it("starts a Codex device-code login session and shows the key handoff details", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(jsonResponse(runtimeFixture({ has_auth: false, auth_path: "" })))
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
      .mockResolvedValueOnce(jsonResponse({ status: "idle" }))
      .mockResolvedValueOnce(
        jsonResponse(
          {
            id: "login-device",
            account_name: "runtime",
            auth_method: "device_auth",
            status: "running",
            logs: "Open https://login.openai.com/activate and enter code WDJB-MJHT",
            device: {
              verification_uri: "https://login.openai.com/activate",
              user_code: "WDJB-MJHT",
              expires_in: 900,
              interval: 5,
            },
          },
          { status: 202 },
        ),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          id: "login-device",
          account_name: "runtime",
          auth_method: "device_auth",
          status: "running",
          logs: "Open https://login.openai.com/activate and enter code WDJB-MJHT",
          device: {
            verification_uri: "https://login.openai.com/activate",
            user_code: "WDJB-MJHT",
            expires_in: 900,
            interval: 5,
          },
        }),
      );

    render(<ReactManagedCodexAccountsRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Start device login" })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Start device login" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenNthCalledWith(
        4,
        "/api/control/codex/accounts/login-sessions",
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({ name: "runtime-device", overwrite: true, auth_method: "device_auth" }),
        }),
      );
    });
    await waitFor(() => {
      expect(fetchMock).toHaveBeenNthCalledWith(
        5,
        "/api/control/codex/accounts/login-sessions/login-device",
        expect.objectContaining({ method: "GET" }),
      );
    });

    expect(screen.getByText("Device Code Login")).toBeInTheDocument();
    expect(screen.getByText("https://login.openai.com/activate")).toBeInTheDocument();
    expect(screen.getByText("WDJB-MJHT")).toBeInTheDocument();
    expect(screen.getByText("Expires in 900s")).toBeInTheDocument();
    expect(screen.getByText("Poll every 5s")).toBeInTheDocument();
    expect(screen.getByText("Open https://login.openai.com/activate and enter code WDJB-MJHT")).toBeInTheDocument();
  });

  it("registers an OpenAI-compatible provider with multiple Claude Code models from the runtime page", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(jsonResponse(runtimeFixture()))
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
      .mockResolvedValueOnce(jsonResponse({ status: "idle" }))
      .mockResolvedValueOnce(
        jsonResponse(
          {
            id: "provider-claude-code",
            name: "Claude Code",
            provider_type: "openai-compatible",
            api_type: "openai-completions",
            base_url: "https://gateway.example.com/v1",
            api_key: "sk-****cdef",
            default_model: "claude-sonnet-4-20260601",
            models: [
              {
                id: "claude-sonnet-4-20260601",
                name: "claude-sonnet-4-20260601",
                supports_tools: true,
                supports_streaming: true,
                is_enabled: true,
              },
              {
                id: "claude-opus-4-20260601",
                name: "claude-opus-4-20260601",
                supports_tools: true,
                supports_streaming: true,
                is_enabled: true,
              },
              {
                id: "claude-haiku-4-20260601",
                name: "claude-haiku-4-20260601",
                supports_tools: true,
                supports_streaming: true,
                is_enabled: true,
              },
            ],
            is_enabled: true,
          },
          { status: 201 },
        ),
      )
      .mockResolvedValueOnce(jsonResponse(runtimeFixture()))
      .mockResolvedValueOnce(
        jsonResponse({
          items: [{ id: "provider-claude-code", name: "Claude Code", is_enabled: true }],
        }),
      )
      .mockResolvedValueOnce(jsonResponse({ status: "idle" }));

    render(<ReactManagedCodexAccountsRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Register provider" })).toBeInTheDocument();
    });

    fireEvent.change(screen.getByLabelText("Provider Name"), { target: { value: "Claude Code" } });
    fireEvent.change(screen.getByLabelText("Base URL"), { target: { value: "https://gateway.example.com/v1" } });
    fireEvent.change(screen.getByLabelText("API Key"), { target: { value: "sk-test-abcdef" } });
    fireEvent.change(screen.getByLabelText("Provider Models"), {
      target: { value: "claude-sonnet-4-20260601\nclaude-opus-4-20260601, claude-haiku-4-20260601" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Register provider" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenNthCalledWith(
        4,
        "/api/control/llm/providers",
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({
            name: "Claude Code",
            provider_type: "openai-compatible",
            api_type: "openai-completions",
            base_url: "https://gateway.example.com/v1",
            api_key: "sk-test-abcdef",
            default_model: "claude-sonnet-4-20260601",
            models: [
              {
                id: "claude-sonnet-4-20260601",
                name: "claude-sonnet-4-20260601",
                supports_tools: true,
                supports_vision: true,
                supports_streaming: true,
                is_enabled: true,
              },
              {
                id: "claude-opus-4-20260601",
                name: "claude-opus-4-20260601",
                supports_tools: true,
                supports_vision: true,
                supports_streaming: true,
                is_enabled: true,
              },
              {
                id: "claude-haiku-4-20260601",
                name: "claude-haiku-4-20260601",
                supports_tools: true,
                supports_vision: true,
                supports_streaming: true,
                is_enabled: true,
              },
            ],
            is_enabled: true,
          }),
        }),
      );
    });

    await waitFor(() => {
      expect(screen.getByText("Provider registered for Claude Code.")).toBeInTheDocument();
      expect(screen.getByText("1 registered provider")).toBeInTheDocument();
    });
    expect(screen.getByLabelText("API Key")).toHaveValue("");
    expect(fetchMock).toHaveBeenNthCalledWith(
      5,
      "/api/control/codex/runtime",
      expect.objectContaining({ method: "GET" }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      6,
      "/api/control/llm/providers",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("keeps the runtime provider form ready for registering additional Claude Code providers", async () => {
    const fetchMock = vi.mocked(fetch);
    const providers: Array<{ id: string; name: string; is_enabled: boolean }> = [];
    const providerPosts: Array<Record<string, unknown>> = [];
    fetchMock.mockImplementation((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/control/codex/runtime") {
        return Promise.resolve(jsonResponse(runtimeFixture()));
      }
      if (url === "/api/control/llm/providers" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: providers }));
      }
      if (url === "/api/control/runtime/restart" && method === "GET") {
        return Promise.resolve(jsonResponse({ status: "idle" }));
      }
      if (url === "/api/control/llm/providers" && method === "POST") {
        const payload = JSON.parse(String(init?.body || "{}")) as Record<string, unknown>;
        providerPosts.push(payload);
        providers.push({
          id: `provider-${providerPosts.length}`,
          name: String(payload.name || ""),
          is_enabled: true,
        });
        return Promise.resolve(
          jsonResponse({ ...payload, id: `provider-${providerPosts.length}`, is_enabled: true }, { status: 201 }),
        );
      }
      return Promise.reject(new Error(`Unhandled fetch: ${url}`));
    });

    render(<ReactManagedCodexAccountsRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Register provider" })).toBeInTheDocument();
    });

    fireEvent.change(screen.getByLabelText("Base URL"), { target: { value: "https://gateway-a.example.com/v1" } });
    fireEvent.change(screen.getByLabelText("API Key"), { target: { value: "sk-provider-a" } });
    fireEvent.change(screen.getByLabelText("Provider Models"), { target: { value: "claude-sonnet-4-a" } });
    fireEvent.click(screen.getByRole("button", { name: "Register provider" }));

    await waitFor(() => {
      expect(screen.getByText("1 registered provider")).toBeInTheDocument();
      expect(screen.getByLabelText("Provider Name")).toHaveValue("Claude Code 2");
    });
    expect(screen.getByLabelText("Base URL")).toHaveValue("");
    expect(screen.getByLabelText("API Key")).toHaveValue("");
    expect(screen.getByLabelText("Provider Models")).toHaveValue("");

    fireEvent.change(screen.getByLabelText("Base URL"), { target: { value: "https://gateway-b.example.com/v1" } });
    fireEvent.change(screen.getByLabelText("API Key"), { target: { value: "sk-provider-b" } });
    fireEvent.change(screen.getByLabelText("Provider Models"), { target: { value: "claude-sonnet-4-b" } });
    fireEvent.click(screen.getByRole("button", { name: "Register provider" }));

    await waitFor(() => {
      expect(screen.getByText("2 registered providers")).toBeInTheDocument();
    });
    expect(providerPosts).toHaveLength(2);
    expect(providerPosts[0]).toEqual(expect.objectContaining({ name: "Claude Code" }));
    expect(providerPosts[1]).toEqual(expect.objectContaining({ name: "Claude Code 2" }));
  });

  it("shows registered provider details and loads one into the runtime provider edit form", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(jsonResponse(runtimeFixture()))
      .mockResolvedValueOnce(
        jsonResponse({
          items: [
            {
              id: "provider-claude-code",
              name: "Claude Code",
              provider_type: "openai-compatible",
              api_type: "openai-completions",
              base_url: "https://gateway.example.com/v1",
              api_key: "sk-****cdef",
              default_model: "claude-sonnet-4",
              models: [
                { id: "claude-sonnet-4", name: "claude-sonnet-4", is_enabled: true },
                { id: "claude-opus-4", name: "claude-opus-4", is_enabled: true },
              ],
              is_enabled: true,
              is_default: true,
            },
          ],
        }),
      )
      .mockResolvedValueOnce(jsonResponse({ status: "idle" }));

    const { container } = render(<ReactManagedCodexAccountsRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getByText("Configured Providers")).toBeInTheDocument();
      expect(screen.getByText("https://gateway.example.com/v1")).toBeInTheDocument();
      expect(screen.getByText("Default: claude-sonnet-4")).toBeInTheDocument();
      expect(screen.getByText("2 models")).toBeInTheDocument();
      expect(screen.getByText("claude-sonnet-4, claude-opus-4")).toBeInTheDocument();
    });
    const consolePanel = container.querySelector(".codex-runtime-provider-console");
    expect(consolePanel).not.toBeNull();
    expect(consolePanel?.querySelector(".codex-runtime-provider-registry")).not.toBeNull();
    expect(consolePanel?.querySelector(".codex-runtime-provider-editor")).not.toBeNull();
    expect(screen.getByLabelText("Provider Models")).toHaveAttribute("rows", "4");
    expect(screen.getByLabelText("Provider Models").closest(".codex-runtime-text-field")).toHaveClass("codex-runtime-provider-models-field");
    expect(screen.getByText("Use one model per line or separate models with commas. The first model becomes the default.")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Edit Claude Code" }));

    expect(screen.getByLabelText("Provider Name")).toHaveValue("Claude Code");
    expect(screen.getByLabelText("Base URL")).toHaveValue("https://gateway.example.com/v1");
    expect(screen.getByLabelText("API Key")).toHaveValue("");
    expect(screen.getByText("Leave blank to keep the stored API key.")).toBeInTheDocument();
    expect(screen.getByLabelText("Provider Models")).toHaveValue("claude-sonnet-4\nclaude-opus-4");
    expect(screen.getByRole("button", { name: "Update provider" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "New provider" })).toBeInTheDocument();
  });

  it("updates an existing runtime provider with multiple models while preserving the stored api key", async () => {
    const fetchMock = vi.mocked(fetch);
    const providers = [
      {
        id: "provider-claude-code",
        name: "Claude Code",
        provider_type: "openai-compatible",
        api_type: "openai-completions",
        base_url: "https://gateway.example.com/v1",
        api_key: "sk-****cdef",
        default_model: "claude-sonnet-4",
        models: [
          { id: "claude-sonnet-4", name: "claude-sonnet-4", is_enabled: true },
          { id: "claude-opus-4", name: "claude-opus-4", is_enabled: true },
        ],
        is_enabled: true,
        is_default: true,
      },
    ];
    const updates: Array<Record<string, unknown>> = [];
    fetchMock.mockImplementation((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = String(init?.method || "GET").toUpperCase();
      if (url === "/api/control/codex/runtime") {
        return Promise.resolve(jsonResponse(runtimeFixture()));
      }
      if (url === "/api/control/llm/providers" && method === "GET") {
        return Promise.resolve(jsonResponse({ items: providers }));
      }
      if (url === "/api/control/runtime/restart" && method === "GET") {
        return Promise.resolve(jsonResponse({ status: "idle" }));
      }
      if (url === "/api/control/llm/providers/provider-claude-code" && method === "PUT") {
        const payload = JSON.parse(String(init?.body || "{}")) as Record<string, unknown>;
        updates.push(payload);
        Object.assign(providers[0], {
          ...payload,
          api_key: "sk-****cdef",
          is_default: true,
        });
        return Promise.resolve(jsonResponse(providers[0]));
      }
      return Promise.reject(new Error(`Unhandled fetch: ${url}`));
    });

    render(<ReactManagedCodexAccountsRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Edit Claude Code" })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Edit Claude Code" }));
    fireEvent.change(screen.getByLabelText("Base URL"), { target: { value: "https://gateway-updated.example.com/v1" } });
    fireEvent.change(screen.getByLabelText("Provider Models"), {
      target: { value: "claude-sonnet-4\nclaude-opus-4, claude-haiku-4" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Update provider" }));

    await waitFor(() => {
      expect(screen.getByText("Provider updated for Claude Code.")).toBeInTheDocument();
      expect(screen.getByText("3 models")).toBeInTheDocument();
    });
    expect(updates).toHaveLength(1);
    expect(updates[0]).toEqual(expect.objectContaining({
      id: "provider-claude-code",
      name: "Claude Code",
      provider_type: "openai-compatible",
      api_type: "openai-completions",
      base_url: "https://gateway-updated.example.com/v1",
      api_key: "",
      default_model: "claude-sonnet-4",
      is_enabled: true,
    }));
    expect(updates[0].models).toEqual([
      {
        id: "claude-sonnet-4",
        name: "claude-sonnet-4",
        supports_tools: true,
        supports_vision: true,
        supports_streaming: true,
        is_enabled: true,
      },
      {
        id: "claude-opus-4",
        name: "claude-opus-4",
        supports_tools: true,
        supports_vision: true,
        supports_streaming: true,
        is_enabled: true,
      },
      {
        id: "claude-haiku-4",
        name: "claude-haiku-4",
        supports_tools: true,
        supports_vision: true,
        supports_streaming: true,
        is_enabled: true,
      },
    ]);
    expect(screen.getByLabelText("Provider Name")).toHaveValue("Claude Code 2");
    expect(screen.getByLabelText("Base URL")).toHaveValue("");
    expect(screen.getByLabelText("API Key")).toHaveValue("");
    expect(screen.getByLabelText("Provider Models")).toHaveValue("");
  });

  it("refreshes the runtime identity after a device-code login succeeds", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(jsonResponse(runtimeFixture({ has_auth: false, auth_path: "" })))
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
      .mockResolvedValueOnce(jsonResponse({ status: "idle" }))
      .mockResolvedValueOnce(
        jsonResponse(
          {
            id: "login-device",
            auth_method: "device_auth",
            status: "running",
            device: {
              verification_uri: "https://login.openai.com/activate",
              user_code: "WDJB-MJHT",
            },
          },
          { status: 202 },
        ),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          id: "login-device",
          auth_method: "device_auth",
          status: "succeeded",
          device: {
            verification_uri: "https://login.openai.com/activate",
            user_code: "WDJB-MJHT",
          },
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse(
          runtimeFixture({
            current: {
              live: {
                auth_mode: "oauth",
                account_name: "Device Account",
                email: "device@example.com",
                plan: "team",
              },
              managed: { name: "runtime-device" },
              quota: {
                hourly: { remaining_percent: 88 },
                weekly: { remaining_percent: 91 },
                plan: "team",
              },
            },
          }),
        ),
      )
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
      .mockResolvedValueOnce(jsonResponse({ status: "idle" }));

    render(<ReactManagedCodexAccountsRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Start device login" })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Start device login" }));

    await waitFor(() => {
      expect(screen.getAllByText("Codex login succeeded. Runtime identity has been refreshed.")).toHaveLength(2);
    });
    expect(screen.getByText("Device Account")).toBeInTheDocument();
    expect(screen.getByText("device@example.com")).toBeInTheDocument();
    expect(fetchMock).toHaveBeenNthCalledWith(
      6,
      "/api/control/codex/runtime",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("updates the active model and reasoning depth from the runtime-only endpoint", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(jsonResponse(runtimeFixture()))
      .mockResolvedValueOnce(jsonResponse({ items: [{ id: "anthropic", name: "Anthropic", is_enabled: true }] }))
      .mockResolvedValueOnce(jsonResponse({ status: "idle" }))
      .mockResolvedValueOnce(
        jsonResponse(
          runtimeFixture({
            model: "gpt-5.4-mini",
            reasoning_effort: "medium",
          }),
        ),
      )
      .mockResolvedValueOnce(
        jsonResponse(
          runtimeFixture({
            model: "gpt-5.4-mini",
            reasoning_effort: "low",
          }),
        ),
      )
      .mockResolvedValueOnce(jsonResponse({ items: [{ id: "anthropic", name: "Anthropic", is_enabled: true }] }));

    render(<ReactManagedCodexAccountsRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getByLabelText("Model")).toHaveValue("gpt-5.4");
      expect(screen.getByLabelText("Reasoning Depth")).toHaveValue("high");
      expect(screen.getByText("1 registered provider")).toBeInTheDocument();
    });

    fireEvent.change(screen.getByLabelText("Model"), {
      target: { value: "gpt-5.4-mini" },
    });

    await waitFor(() => {
      expect(screen.getByLabelText("Reasoning Depth")).toHaveValue("medium");
    });

    await waitFor(() => {
      expect(fetchMock).toHaveBeenNthCalledWith(
        4,
        "/api/control/codex/runtime",
        expect.objectContaining({
          method: "PUT",
          body: JSON.stringify({ model: "gpt-5.4-mini", reasoning_effort: "medium" }),
        }),
      );
    });

    fireEvent.change(screen.getByLabelText("Reasoning Depth"), {
      target: { value: "low" },
    });

    await waitFor(() => {
      expect(screen.getByLabelText("Reasoning Depth")).toHaveValue("low");
    });

    expect(fetchMock).toHaveBeenNthCalledWith(
      5,
      "/api/control/codex/runtime",
      expect.objectContaining({
        method: "PUT",
        body: JSON.stringify({ model: "gpt-5.4-mini", reasoning_effort: "low" }),
      }),
    );
    expect(fetchMock).not.toHaveBeenCalledWith(
      expect.stringContaining("/api/control/codex/accounts/"),
      expect.anything(),
    );
    expect(screen.getByLabelText("Model")).toHaveValue("gpt-5.4-mini");
    expect(screen.getByLabelText("Reasoning Depth")).toHaveValue("low");
  });

  it("renders missing auth without account login actions, status copy, or side diagnostics", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(jsonResponse(runtimeFixture({ has_auth: false, auth_path: "" })))
      .mockResolvedValueOnce(jsonResponse({ items: [] }));

    render(<ReactManagedCodexAccountsRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getByText("Codex Runtime")).toBeInTheDocument();
    });

    expect(screen.queryByText("Auth missing")).not.toBeInTheDocument();
    expect(screen.queryByText("Status")).not.toBeInTheDocument();
    expect(document.querySelector(".codex-runtime-status-pane")).toBeNull();
    expect(screen.queryByText("Codex auth.json is not loaded.")).not.toBeInTheDocument();
    expect(screen.queryByText("Runtime Diagnostics")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Account Name")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Auth File")).not.toBeInTheDocument();
  });
});

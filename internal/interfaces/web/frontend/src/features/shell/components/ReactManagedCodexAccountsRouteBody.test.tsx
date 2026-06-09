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

  it("keeps the runtime shell visible while loading without account management surfaces", () => {
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
      return Promise.reject(new Error(`Unhandled fetch: ${url}`));
    });

    render(<ReactManagedCodexAccountsRouteBody language="en" />);

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
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

    resolveRuntime?.(jsonResponse(runtimeFixture()));
    resolveProviders?.(jsonResponse({ items: [] }));

    await waitFor(() => {
      expect(screen.getByText("No LLM providers registered. Codex Direct remains available.")).toBeInTheDocument();
    });
  });

  it("updates the active model and reasoning depth from the runtime-only endpoint", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(jsonResponse(runtimeFixture()))
      .mockResolvedValueOnce(jsonResponse({ items: [{ id: "anthropic", name: "Anthropic", is_enabled: true }] }))
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
        3,
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
      4,
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

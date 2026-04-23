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

  it("keeps the dashboard shell visible while loading", () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockImplementation(() => new Promise(() => {}));

    render(<ReactManagedCodexAccountsRouteBody language="en" />);

    expect(screen.getByText("Runtime Overview")).toBeInTheDocument();
    expect(screen.getByText("Current Codex")).toBeInTheDocument();
    expect(screen.getByText("Managed Accounts")).toBeInTheDocument();
    expect(screen.getByText("Import or Add Account")).toBeInTheDocument();
    expect(screen.getByText("Login Session")).toBeInTheDocument();
    expect(screen.getAllByText("Loading...").length).toBeGreaterThan(0);
  });

  it("loads codex account statuses and switches the active account", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(
        jsonResponse({
          items: [
            {
              record: {
                name: "work",
                snapshot: { account_name: "Work Account", email: "work@example.com", plan: "plus" },
              },
              current: true,
              quota: {
                hourly: { remaining_percent: 80, reset_at: "2026-05-01T14:30:00Z" },
                weekly: { remaining_percent: 92, reset_at: "2026-05-08T14:30:00Z" },
                plan: "plus",
              },
            },
            {
              record: {
                name: "personal",
                snapshot: { account_name: "Personal Account", email: "personal@example.com", plan: "pro" },
              },
              current: false,
              quota: {
                hourly: { remaining_percent: 70, reset_at: "2026-05-01T16:00:00Z" },
                weekly: { remaining_percent: 88, reset_at: "2026-05-08T16:00:00Z" },
                plan: "pro",
              },
            },
          ],
          active: {
            managed: {
              name: "work",
            },
            auth_path: "/var/lib/alter0/.codex/auth.json",
          },
          runtime: {
            ...runtimeFixture(),
          },
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          account: {
            name: "personal",
            snapshot: { account_name: "Personal Account" },
          },
          backup_path: "/tmp/auth-backup.json",
          active: {
            managed: {
              name: "personal",
            },
          },
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          items: [
            {
              record: {
                name: "personal",
                snapshot: { account_name: "Personal Account", email: "personal@example.com", plan: "pro" },
              },
              current: true,
              quota: {
                hourly: { remaining_percent: 70, reset_at: "2026-05-01T16:00:00Z" },
                weekly: { remaining_percent: 88, reset_at: "2026-05-08T16:00:00Z" },
                plan: "pro",
              },
            },
          ],
          active: {
            managed: {
              name: "personal",
            },
            auth_path: "/var/lib/alter0/.codex/auth.json",
          },
          runtime: {
            ...runtimeFixture(),
          },
        }),
      );

    render(<ReactManagedCodexAccountsRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getByText("Managed Accounts")).toBeInTheDocument();
      expect(screen.getByText("Runtime Overview")).toBeInTheDocument();
      expect(screen.getAllByText("Work Account").length).toBeGreaterThan(0);
    });

    expect(screen.getAllByTestId("codex-account-card")).toHaveLength(2);
    const overview = screen.getByText("Runtime Overview").closest(".codex-accounts-overview");
    expect(overview).not.toBeNull();
    const overviewQueries = within(overview as HTMLElement);
    expect(overviewQueries.getByText("Work Account")).toBeInTheDocument();
    expect(overviewQueries.getByText("plus")).toBeInTheDocument();
    expect(overviewQueries.getByText("80%")).toBeInTheDocument();
    expect(overviewQueries.getByText("92%")).toBeInTheDocument();
    expect(overviewQueries.queryByText("/var/lib/alter0/.codex/auth.json")).not.toBeInTheDocument();
    expect(overviewQueries.queryByText("Active Auth Path")).not.toBeInTheDocument();
    expect(overviewQueries.getByRole("progressbar", { name: "Hourly Remaining" })).toHaveAttribute("aria-valuenow", "80");
    expect(overviewQueries.getByRole("progressbar", { name: "Weekly Remaining" })).toHaveAttribute("aria-valuenow", "92");
    expect(overviewQueries.getByText("2026-05-01 14:30 UTC")).toBeInTheDocument();
    expect(overviewQueries.getByText("2026-05-08 14:30 UTC")).toBeInTheDocument();

    const accountCards = screen.getAllByTestId("codex-account-card");
    const firstCardQueries = within(accountCards[0] as HTMLElement);
    expect(firstCardQueries.getByRole("progressbar", { name: "Hourly Remaining" })).toHaveAttribute("aria-valuenow", "80");
    expect(firstCardQueries.getByText("2026-05-01 14:30 UTC")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Switch to Personal Account" }));

    await waitFor(() => {
      expect(screen.getByText("Switched to personal.")).toBeInTheDocument();
    });

    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "/api/control/codex/accounts/personal/switch",
      expect.objectContaining({ method: "POST" }),
    );
    expect(screen.getAllByText("Personal Account").length).toBeGreaterThan(0);
  });

  it("imports an auth file and creates the managed account", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(
        jsonResponse({
          items: [],
          active: {
            live: {
              account_name: "CLI Account",
              email: "cli@example.com",
              plan: "plus",
            },
            quota: {
              hourly: { remaining_percent: 61, reset_at: "2026-05-02T08:00:00Z" },
              weekly: { remaining_percent: 84, reset_at: "2026-05-09T08:00:00Z" },
              plan: "plus",
            },
            auth_path: "/var/lib/alter0/.codex/auth.json",
          },
          runtime: {
            ...runtimeFixture(),
          },
        }),
      )
      .mockResolvedValueOnce(jsonResponse({ name: "work", snapshot: { account_name: "Work Account" } }, { status: 201 }))
      .mockResolvedValueOnce(
        jsonResponse({
          items: [
            {
              record: {
                name: "work",
                snapshot: { account_name: "Work Account", email: "work@example.com" },
              },
              current: false,
            },
          ],
          active: null,
          runtime: {
            ...runtimeFixture(),
          },
        }),
      );

    render(<ReactManagedCodexAccountsRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Choose auth.json" })).toBeInTheDocument();
      expect(screen.getByText("No managed Codex accounts yet.")).toBeInTheDocument();
      expect(screen.getByText("CLI Account")).toBeInTheDocument();
      expect(
        screen.getAllByText("Current auth.json is active but not managed yet. Import it to create the first managed snapshot.")
          .length,
      ).toBeGreaterThan(0);
    });
    const overview = screen.getByText("Runtime Overview").closest(".codex-accounts-overview");
    expect(overview).not.toBeNull();
    const overviewQueries = within(overview as HTMLElement);
    expect(overviewQueries.getByText("plus")).toBeInTheDocument();
    expect(overviewQueries.getByText("61%")).toBeInTheDocument();
    expect(overviewQueries.getByText("84%")).toBeInTheDocument();
    expect(overviewQueries.queryByText("/var/lib/alter0/.codex/auth.json")).not.toBeInTheDocument();
    expect(overviewQueries.getByText("2026-05-02 08:00 UTC")).toBeInTheDocument();
    expect(overviewQueries.getByRole("progressbar", { name: "Hourly Remaining" })).toHaveAttribute("aria-valuenow", "61");

    const authFile = new File([`{"auth_mode":"apikey","OPENAI_API_KEY":"sk-test"}`], "auth.json", { type: "application/json" });
    Object.defineProperty(authFile, "text", {
      value: () => Promise.resolve(`{"auth_mode":"apikey","OPENAI_API_KEY":"sk-test"}`),
    });

    fireEvent.change(screen.getByLabelText("Account Name"), {
      target: { value: "work" },
    });
    fireEvent.change(screen.getByLabelText("Auth File"), {
      target: {
        files: [authFile],
      },
    });

    expect(screen.getByText("auth.json")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Import auth.json" }));

    await waitFor(() => {
      expect(screen.getByText("Account imported.")).toBeInTheDocument();
    });

    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "/api/control/codex/accounts",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          name: "work",
          overwrite: false,
          auth_file_content: `{"auth_mode":"apikey","OPENAI_API_KEY":"sk-test"}`,
        }),
      }),
    );
  });

  it("starts a login session and refreshes the session state", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(jsonResponse({ items: [], active: null, runtime: null }))
      .mockResolvedValueOnce(
        jsonResponse({
          id: "login-1",
          account_name: "fresh",
          status: "running",
        }, { status: 202 }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          id: "login-1",
          account_name: "fresh",
          status: "succeeded",
          logs: "open browser",
        }),
      )
      .mockResolvedValueOnce(jsonResponse({ items: [], active: null, runtime: null }));

    render(<ReactManagedCodexAccountsRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getByText("Login Session")).toBeInTheDocument();
      expect(screen.getByText("No login session started.")).toBeInTheDocument();
      expect(screen.getByText("Runtime Overview")).toBeInTheDocument();
    });

    fireEvent.change(screen.getByLabelText("Account Name"), {
      target: { value: "fresh" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Start Codex Login" }));

    await waitFor(() => {
      expect(screen.getByText("Login completed.")).toBeInTheDocument();
    });

    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "/api/control/codex/accounts/login-sessions",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          name: "fresh",
          overwrite: false,
        }),
      }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      3,
      "/api/control/codex/accounts/login-sessions/login-1",
      expect.objectContaining({ method: "GET" }),
    );
    expect(screen.getByText("open browser")).toBeInTheDocument();
  });

  it("shows current codex runtime status and updates the active model and reasoning depth", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(
        jsonResponse({
          items: [],
          active: {
            managed: {
              name: "work",
              snapshot: { account_name: "Work Account" },
            },
            auth_path: "/var/lib/alter0/.codex/auth.json",
          },
          runtime: {
            ...runtimeFixture(),
          },
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse(
          runtimeFixture({
            model: "gpt-5.4-mini",
            reasoning_effort: "low",
          }),
        ),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          items: [],
          active: {
            managed: {
              name: "work",
              snapshot: { account_name: "Work Account" },
            },
            auth_path: "/var/lib/alter0/.codex/auth.json",
          },
          runtime: {
            ...runtimeFixture({
              model: "gpt-5.4-mini",
              reasoning_effort: "low",
            }),
          },
        }),
      );

    render(<ReactManagedCodexAccountsRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getByText("Current Codex")).toBeInTheDocument();
      expect(screen.getByLabelText("Model")).toHaveValue("gpt-5.4");
      expect(screen.getByLabelText("Reasoning Depth")).toHaveValue("high");
      expect(screen.getByText("Runtime Details")).toBeInTheDocument();
    });
    expect(screen.getAllByText("GPT-5.4 (gpt-5.4)")).toHaveLength(1);
    expect(document.querySelectorAll(".codex-accounts-runtime-current")).toHaveLength(0);

    fireEvent.change(screen.getByLabelText("Model"), {
      target: { value: "gpt-5.4-mini" },
    });

    await waitFor(() => {
      expect(screen.getByLabelText("Reasoning Depth")).toHaveValue("medium");
    });

    fireEvent.change(screen.getByLabelText("Reasoning Depth"), {
      target: { value: "low" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Apply Runtime Settings" }));

    await waitFor(() => {
      expect(screen.getByText("Runtime settings updated.")).toBeInTheDocument();
    });

    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "/api/control/codex/runtime",
      expect.objectContaining({
        method: "PUT",
        body: JSON.stringify({ model: "gpt-5.4-mini", reasoning_effort: "low" }),
      }),
    );
    expect(screen.getByLabelText("Model")).toHaveValue("gpt-5.4-mini");
    expect(screen.getByLabelText("Reasoning Depth")).toHaveValue("low");
  });
});

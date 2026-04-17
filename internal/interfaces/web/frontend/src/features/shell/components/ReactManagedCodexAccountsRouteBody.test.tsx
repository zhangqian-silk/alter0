import { fireEvent, render, screen, waitFor } from "@testing-library/react";
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
                hourly: { remaining_percent: 80 },
                weekly: { remaining_percent: 92 },
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
                hourly: { remaining_percent: 70 },
                weekly: { remaining_percent: 88 },
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
                hourly: { remaining_percent: 70 },
                weekly: { remaining_percent: 88 },
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
        }),
      );

    render(<ReactManagedCodexAccountsRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getByText("Managed Accounts")).toBeInTheDocument();
      expect(screen.getByText("Runtime Overview")).toBeInTheDocument();
      expect(screen.getByText("Work Account")).toBeInTheDocument();
    });

    expect(screen.getAllByTestId("codex-account-card")).toHaveLength(2);
    expect(screen.getByText("80%")).toBeInTheDocument();
    expect(screen.getByText("92%")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Switch to Personal Account" }));

    await waitFor(() => {
      expect(screen.getByText("Switched to personal.")).toBeInTheDocument();
    });

    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "/api/control/codex/accounts/personal/switch",
      expect.objectContaining({ method: "POST" }),
    );
    expect(screen.getByText("Personal Account")).toBeInTheDocument();
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
            },
            auth_path: "/var/lib/alter0/.codex/auth.json",
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
      .mockResolvedValueOnce(jsonResponse({ items: [], active: null }))
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
      .mockResolvedValueOnce(jsonResponse({ items: [], active: null }));

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
});

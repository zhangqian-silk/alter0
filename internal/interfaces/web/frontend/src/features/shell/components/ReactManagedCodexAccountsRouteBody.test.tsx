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
      expect(screen.getByText("Work Account")).toBeInTheDocument();
    });

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
      .mockResolvedValueOnce(jsonResponse({ items: [], active: null }))
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
      expect(screen.getByText("No managed Codex accounts yet.")).toBeInTheDocument();
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
      expect(screen.getByText("No managed Codex accounts yet.")).toBeInTheDocument();
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

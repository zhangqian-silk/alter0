import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { ReactManagedMaintenanceRouteBody } from "./ReactManagedMaintenanceRouteBody";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: init.status ?? 200,
    headers: {
      "Content-Type": "application/json",
      ...(init.headers ?? {}),
    },
  });
}

describe("ReactManagedMaintenanceRouteBody", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders maintenance status and runs session cleanup manually", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(
        jsonResponse({
          items: [
            {
              job_id: "system-memory-maintenance",
              status: "success",
              finished_at: "2026-04-14T03:04:05Z",
              next_run_at: "2026-04-15T05:10:00Z",
              changed_files: ["memories/user.md"],
            },
            {
              job_id: "system-session-cleanup",
              status: "idle",
              next_run_at: "2026-04-15T05:20:00Z",
            },
          ],
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          job_id: "system-session-cleanup",
          status: "success",
          deleted_count: 2,
          skipped_pinned_count: 1,
          skipped_protected_count: 3,
          scanned_count: 5,
          finished_at: "2026-04-14T03:10:00Z",
          next_run_at: "2026-04-15T05:20:00Z",
        }),
      );

    render(<ReactManagedMaintenanceRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getByText("Memory Maintenance")).toBeInTheDocument();
    });

    expect(screen.getByText("memories/user.md")).toBeInTheDocument();
    expect(screen.getByText("Session Cleanup")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Clean up now" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenNthCalledWith(
        2,
        "/api/maintenance/sessions/cleanup",
        expect.objectContaining({ method: "POST" }),
      );
    });
    expect(screen.getByText("Deleted 2 · pinned 1 · protected 3 · scanned 5")).toBeInTheDocument();
  });
});

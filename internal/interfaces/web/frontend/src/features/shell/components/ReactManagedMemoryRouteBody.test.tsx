import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { ReactManagedMemoryRouteBody } from "./ReactManagedMemoryRouteBody";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: init.status ?? 200,
    headers: {
      "Content-Type": "application/json",
      ...(init.headers ?? {}),
    },
  });
}

describe("ReactManagedMemoryRouteBody", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("loads memory document tabs from the unified memory API", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValueOnce(
      jsonResponse({
        long_term: {
          exists: true,
          path: "/memory/MEMORY.md",
          updated_at: "2026-03-04T08:00:00Z",
          content: "# Long-Term Memory\n- key: value",
        },
        root_instructions: {
          exists: true,
          path: "/workspace/AGENTS.md",
          updated_at: "2026-03-04T08:00:00Z",
          content: "# AGENTS\n- run tests",
        },
        daily: {
          directory: "/memory/daily",
          items: [],
        },
        mandatory: {
          exists: false,
        },
        specification: {
          exists: false,
        },
      }),
    );

    render(<ReactManagedMemoryRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getByRole("tab", { name: "Long-Term" })).toBeInTheDocument();
    });

    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      "/api/memory/context",
      expect.objectContaining({ method: "GET" }),
    );
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock).not.toHaveBeenCalledWith(expect.stringContaining("/api/memory/tasks"), expect.anything());

    fireEvent.click(screen.getByRole("tab", { name: "Long-Term" }));

    expect(document.querySelector(".memory-content h1")).toHaveTextContent("Long-Term Memory");
    expect(document.querySelector(".memory-content li")).toHaveTextContent("key: value");
    expect(screen.getByText("/memory/MEMORY.md")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("tab", { name: "AGENTS.md" }));

    expect(document.querySelector(".memory-content h1")).toHaveTextContent("AGENTS");
    expect(document.querySelector(".memory-content li")).toHaveTextContent("run tests");
    expect(screen.getByText("/workspace/AGENTS.md")).toBeInTheDocument();
  });

  it("renders specification markdown sections inside the specification tab", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValueOnce(
      jsonResponse({
        long_term: { exists: false },
        daily: { items: [] },
        mandatory: { exists: false },
        specification: {
          exists: true,
          path: "/docs/memory/spec.md",
          updated_at: "2026-03-04T08:00:00Z",
          content: "# Mapping\n- USER.md\n## Rules\n- Keep memory concise",
        },
      }),
    );

    render(<ReactManagedMemoryRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getByRole("tab", { name: "Specification" })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("tab", { name: "Specification" }));

    expect(screen.getByText("Mapping")).toBeInTheDocument();
    expect(screen.getByText("Rules")).toBeInTheDocument();
    expect(screen.getByText("Keep memory concise")).toBeInTheDocument();
  });

  it("renders memory documents as markdown instead of raw preformatted text", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValueOnce(
      jsonResponse({
        long_term: {
          exists: true,
          path: "/memory/MEMORY.md",
          updated_at: "2026-03-04T08:00:00Z",
          content: "## Memory Rules\n- **Ship** durable notes\n- [Workspace](/chat)",
        },
        daily: {
          directory: "/memory/daily",
          items: [
            {
              date: "2026-03-04",
              path: "/memory/daily/2026-03-04.md",
              content: "### Daily\n- Follow up with [Tasks](/tasks)",
            },
          ],
        },
        mandatory: {
          exists: false,
        },
        specification: {
          exists: false,
        },
      }),
    );

    const { container } = render(<ReactManagedMemoryRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getByRole("tab", { name: "Long-Term" })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("tab", { name: "Long-Term" }));

    expect(container.querySelector(".memory-content h2")).toHaveTextContent("Memory Rules");
    expect(container.querySelector(".memory-content strong")).toHaveTextContent("Ship");
    expect(container.querySelector(".memory-content a")).toHaveAttribute("href", "/chat");
    expect(container.querySelector(".memory-content pre")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("tab", { name: "Daily" }));

    expect(container.querySelector(".memory-content h3")).toHaveTextContent("Daily");
    expect(container.querySelector(".memory-content a")).toHaveAttribute("href", "/tasks");
  });
});

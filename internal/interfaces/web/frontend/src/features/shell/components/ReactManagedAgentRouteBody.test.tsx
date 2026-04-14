import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { ReactManagedAgentRouteBody } from "./ReactManagedAgentRouteBody";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: init.status ?? 200,
    headers: {
      "Content-Type": "application/json",
      ...(init.headers ?? {}),
    },
  });
}

describe("ReactManagedAgentRouteBody", () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("loads managed agents with their skill and MCP options", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(
        jsonResponse({
          items: [
            {
              id: "researcher",
              name: "Researcher",
              enabled: true,
              scope: "global",
              version: "v1.0.0",
              system_prompt: "Execute first, summarize later.",
              max_iterations: 8,
              tools: ["codex_exec"],
              skills: ["summary"],
              mcps: ["github"],
              memory_files: ["user_md"],
            },
          ],
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          items: [{ id: "summary", name: "Summary", scope: "builtin", version: "v1" }],
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          items: [{ id: "github", name: "GitHub", scope: "builtin", version: "v1" }],
        }),
      );

    render(<ReactManagedAgentRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getByText("Researcher")).toBeInTheDocument();
    });

    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      "/api/control/agents",
      expect.objectContaining({ method: "GET" }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "/api/control/skills",
      expect.objectContaining({ method: "GET" }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      3,
      "/api/control/mcps",
      expect.objectContaining({ method: "GET" }),
    );

    expect(screen.getByDisplayValue("Researcher")).toBeInTheDocument();
    expect(screen.getByDisplayValue("Execute first, summarize later.")).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: /Summary/ })).toBeChecked();
    expect(screen.getByRole("checkbox", { name: /GitHub/ })).toBeChecked();
  });

  it("creates a managed agent and reloads the route state", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
      .mockResolvedValueOnce(
        jsonResponse({
          id: "researcher",
          name: "Researcher",
          enabled: true,
          scope: "global",
          version: "v1.0.0",
          system_prompt: "Execute first, summarize later.",
          max_iterations: 8,
          tools: ["codex_exec"],
          skills: [],
          mcps: [],
          memory_files: [],
        }),
        { status: 201 },
      )
      .mockResolvedValueOnce(
        jsonResponse({
          items: [
            {
              id: "researcher",
              name: "Researcher",
              enabled: true,
              scope: "global",
              version: "v1.0.0",
              system_prompt: "Execute first, summarize later.",
              max_iterations: 8,
              tools: ["codex_exec"],
              skills: [],
              mcps: [],
              memory_files: [],
            },
          ],
        }),
      )
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
      .mockResolvedValueOnce(jsonResponse({ items: [] }));

    render(<ReactManagedAgentRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getByText("Unsaved Agent")).toBeInTheDocument();
    });

    fireEvent.change(screen.getByLabelText("Name"), {
      target: { value: "Researcher" },
    });
    fireEvent.change(screen.getByLabelText("Max Iterations"), {
      target: { value: "8" },
    });
    fireEvent.change(screen.getByLabelText("System Prompt"), {
      target: { value: "Execute first, summarize later." },
    });

    fireEvent.click(screen.getByRole("button", { name: "Save Agent" }));

    await waitFor(() => {
      expect(screen.getByText("Agent saved.")).toBeInTheDocument();
    });

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/control/agents",
      expect.objectContaining({
        method: "POST",
      }),
    );
    expect(screen.getByDisplayValue("Researcher")).toBeInTheDocument();
    await waitFor(() => {
      expect(window.localStorage.getItem("alter0.web.agent-route.v1")).toContain("researcher");
    });
  });

  it("deletes the selected agent and returns to the draft form", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(
        jsonResponse({
          items: [
            {
              id: "researcher",
              name: "Researcher",
              enabled: true,
              scope: "global",
              version: "v1.0.0",
              system_prompt: "Execute first, summarize later.",
              max_iterations: 8,
              tools: ["codex_exec"],
              skills: [],
              mcps: [],
              memory_files: [],
            },
          ],
        }),
      )
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
      .mockResolvedValueOnce(jsonResponse({ status: "deleted" }))
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
      .mockResolvedValueOnce(jsonResponse({ items: [] }));

    render(<ReactManagedAgentRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("Researcher")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Delete Agent" }));

    await waitFor(() => {
      expect(screen.getByText("Agent deleted.")).toBeInTheDocument();
    });

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/control/agents/researcher",
      expect.objectContaining({ method: "DELETE" }),
    );
    expect(screen.getByText("Unsaved Agent")).toBeInTheDocument();
  });
});

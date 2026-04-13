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
});

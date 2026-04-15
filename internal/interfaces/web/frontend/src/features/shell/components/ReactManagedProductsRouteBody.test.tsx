import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { ReactManagedProductsRouteBody } from "./ReactManagedProductsRouteBody";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: init.status ?? 200,
    headers: {
      "Content-Type": "application/json",
      ...(init.headers ?? {}),
    },
  });
}

describe("ReactManagedProductsRouteBody", () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("loads the selected product workspace and draft studio data", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(
        jsonResponse({
          items: [
            {
              id: "travel",
              name: "Travel",
              summary: "City guides with workspace-backed detail pages.",
              status: "active",
              visibility: "public",
              owner_type: "builtin",
              version: "v1.2.0",
              master_agent_id: "travel-master",
              artifact_types: ["city_guide"],
              knowledge_sources: ["poi_catalog"],
              worker_agents: [],
              tags: ["travel"],
            },
          ],
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          items: [
            {
              draft_id: "draft-travel-1",
              mode: "bootstrap",
              review_status: "draft",
              generated_at: "2026-04-15T01:02:03Z",
              updated_at: "2026-04-15T01:02:03Z",
              goal: "Launch premium city guides.",
              product: {
                id: "travel",
                name: "Travel Premium",
                summary: "Premium travel planning.",
                owner_type: "managed",
              },
              master_agent: {
                agent_id: "travel-master",
                name: "Travel Master",
                tools: ["codex_exec"],
                skills: ["city-guides"],
                allowed_delegate_targets: [],
              },
              worker_matrix: [],
            },
          ],
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          product: {
            id: "travel",
            name: "Travel",
            summary: "City guides with workspace-backed detail pages.",
            status: "active",
            visibility: "public",
            version: "v1.2.0",
            artifact_types: ["city_guide"],
            knowledge_sources: ["poi_catalog"],
            tags: ["travel"],
            worker_agents: [],
          },
          master_agent: {
            agent_id: "travel-master",
            name: "Travel Master",
            description: "Maintains city guide spaces.",
            capabilities: ["city-guide"],
            tools: ["codex_exec"],
          },
          workspace_hint: "Talk to the product master agent and maintain product detail pages.",
          space_label: "Detail Pages",
          spaces: [
            {
              space_id: "wuhan",
              title: "Wuhan",
              slug: "wuhan",
              html_path: "/products/travel/spaces/wuhan.html",
              summary: "Metro-first night food city guide.",
              status: "active",
              revision: 3,
              updated_at: "2026-04-15T01:02:03Z",
              tags: ["metro", "food"],
            },
          ],
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          space: {
            space_id: "wuhan",
            title: "Wuhan",
            summary: "Metro-first night food city guide.",
            html_path: "/products/travel/spaces/wuhan.html",
          },
          guide: {
            id: "wuhan",
            city: "Wuhan",
            days: 3,
            travel_style: "metro-first",
            budget: "mid-range",
            companions: ["friends"],
            must_visit: ["Yellow Crane Tower"],
            avoid: [],
            additional_requirements: ["late-night food"],
            keep_conditions: [],
            replace_conditions: [],
            notes: ["Keep walking sections short."],
            daily_routes: [
              {
                day: 1,
                theme: "Riverfront warm-up",
                stops: ["Yellow Crane Tower", "Hubu Alley"],
                transit: ["Metro Line 4"],
              },
            ],
            map_layers: [
              {
                id: "food",
                label: "Food Layer",
                description: "Night food stops.",
              },
            ],
            content: "Day 1 focuses on riverfront landmarks and late-night snacks.",
            revision: 3,
            updated_at: "2026-04-15T01:02:03Z",
          },
        }),
      );

    render(<ReactManagedProductsRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getAllByText("Travel").length).toBeGreaterThan(0);
    });

    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      "/api/control/products",
      expect.objectContaining({ method: "GET" }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "/api/control/products/drafts",
      expect.objectContaining({ method: "GET" }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      3,
      "/api/products/travel/workspace",
      expect.objectContaining({ method: "GET" }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      4,
      "/api/products/travel/workspace/spaces/wuhan",
      expect.objectContaining({ method: "GET" }),
    );

    expect(screen.getByText("Product Workspace")).toBeInTheDocument();
    expect(screen.getByText("Master Agent Conversation")).toBeInTheDocument();
    expect(screen.getAllByText("Metro-first night food city guide.").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Open HTML Page").length).toBeGreaterThan(0);
  });

  it("sends workspace messages through the product master endpoint and refreshes the selected space", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(
        jsonResponse({
          items: [
            {
              id: "travel",
              name: "Travel",
              summary: "City guides with workspace-backed detail pages.",
              status: "active",
              visibility: "public",
              owner_type: "builtin",
              version: "v1.2.0",
            },
          ],
        }),
      )
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
      .mockResolvedValueOnce(
        jsonResponse({
          product: {
            id: "travel",
            name: "Travel",
            summary: "City guides with workspace-backed detail pages.",
            status: "active",
            visibility: "public",
            version: "v1.2.0",
          },
          master_agent: {
            agent_id: "travel-master",
            name: "Travel Master",
            description: "Maintains city guide spaces.",
            tools: ["codex_exec"],
          },
          spaces: [
            {
              space_id: "wuhan",
              title: "Wuhan",
              summary: "Metro-first night food city guide.",
              html_path: "/products/travel/spaces/wuhan.html",
              status: "active",
              tags: ["metro"],
            },
          ],
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          space: {
            space_id: "wuhan",
            title: "Wuhan",
            summary: "Metro-first night food city guide.",
          },
          guide: {
            id: "wuhan",
            city: "Wuhan",
            days: 3,
            travel_style: "metro-first",
            budget: "mid-range",
            companions: [],
            must_visit: [],
            avoid: [],
            additional_requirements: [],
            keep_conditions: [],
            replace_conditions: [],
            notes: [],
            daily_routes: [],
            map_layers: [],
            content: "Initial content",
            revision: 1,
            updated_at: "2026-04-15T01:02:03Z",
          },
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          reply: "Wuhan page updated with a stronger snack crawl.",
          guide: {
            id: "wuhan-night-food",
          },
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          product: {
            id: "travel",
            name: "Travel",
            summary: "City guides with workspace-backed detail pages.",
            status: "active",
            visibility: "public",
            version: "v1.2.0",
          },
          master_agent: {
            agent_id: "travel-master",
            name: "Travel Master",
            description: "Maintains city guide spaces.",
            tools: ["codex_exec"],
          },
          spaces: [
            {
              space_id: "wuhan-night-food",
              title: "Wuhan Night Food",
              summary: "Late-night crawl guide.",
              html_path: "/products/travel/spaces/wuhan-night-food.html",
              status: "active",
              tags: ["food"],
            },
          ],
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          space: {
            space_id: "wuhan-night-food",
            title: "Wuhan Night Food",
            summary: "Late-night crawl guide.",
          },
          guide: {
            id: "wuhan-night-food",
            city: "Wuhan",
            days: 2,
            travel_style: "snack crawl",
            budget: "mid-range",
            companions: [],
            must_visit: [],
            avoid: [],
            additional_requirements: [],
            keep_conditions: [],
            replace_conditions: [],
            notes: ["Add late-night noodle stops."],
            daily_routes: [],
            map_layers: [],
            content: "Updated content",
            revision: 2,
            updated_at: "2026-04-15T02:02:03Z",
          },
        }),
      );

    render(<ReactManagedProductsRouteBody language="en" />);

    await waitFor(() => {
      expect(screen.getByText("Master Agent Conversation")).toBeInTheDocument();
    });

    fireEvent.change(screen.getByLabelText("Workspace message"), {
      target: { value: "Add more late-night noodle stops." },
    });
    fireEvent.click(screen.getByRole("button", { name: "Send to Master" }));

    await waitFor(() => {
      expect(screen.getByText("Wuhan page updated with a stronger snack crawl.")).toBeInTheDocument();
    });

    expect(fetchMock).toHaveBeenNthCalledWith(
      5,
      "/api/products/travel/workspace/chat",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          session_id: "",
          space_id: "wuhan",
          content: "Add more late-night noodle stops.",
        }),
      }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      6,
      "/api/products/travel/workspace",
      expect.objectContaining({ method: "GET" }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      7,
      "/api/products/travel/workspace/spaces/wuhan-night-food",
      expect.objectContaining({ method: "GET" }),
    );
    expect(screen.getByText("Workspace synced.")).toBeInTheDocument();
    expect(screen.getAllByText("Wuhan Night Food").length).toBeGreaterThan(0);
  });
});

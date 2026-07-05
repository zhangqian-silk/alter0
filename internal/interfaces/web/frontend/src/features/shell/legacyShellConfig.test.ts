import { SETTINGS_ROUTE_GROUPS, NAV_GROUPS, PROMPTS } from "./legacyShellConfig";

describe("legacyShellConfig", () => {
  it("keeps the navigation groups in the shell order", () => {
    expect(NAV_GROUPS.map((group) => group.heading)).toEqual([
      "Workspace",
    ]);
  });

  it("keeps chat as the default workspace route", () => {
    expect(NAV_GROUPS[0].items[0]).toMatchObject({
      route: "chat",
      label: "Chat"
    });
  });

  it("keeps navigation routes unique across groups", () => {
    const routes = NAV_GROUPS.flatMap((group) => group.items.map((item) => item.route));
    const uniqueRoutes = new Set(routes);

    expect(uniqueRoutes.size).toBe(routes.length);
  });

  it("keeps only chat in the primary navigation route list", () => {
    const workspaceRoutes = NAV_GROUPS[0].items.map((item) => item.route);
    const allRoutes = NAV_GROUPS.flatMap((group) => group.items.map((item) => item.route));

    expect(workspaceRoutes).toEqual(["chat"]);
    expect(allRoutes).toEqual(["chat"]);
    expect(allRoutes).not.toContain("settings");
    expect(allRoutes).not.toContain("chatRuntime");
    expect(allRoutes).not.toContain("memory");
    expect(allRoutes).not.toContain("tasks");
    expect(allRoutes).not.toContain("codex-accounts");
  });

  it("keeps service restart and update controls reachable inside settings", () => {
    expect(SETTINGS_ROUTE_GROUPS[0].items.map((item) => item.route)).toEqual([
      "runtime",
      "skills",
      "memory",
      "schedules",
      "general",
    ]);
  });

  it("does not configure bottom quick suggestions for the Chat empty state", () => {
    expect(PROMPTS).toEqual([]);
  });
});

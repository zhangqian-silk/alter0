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

  it("keeps only chat, terminal, and settings as primary navigation routes", () => {
    const workspaceRoutes = NAV_GROUPS[0].items.map((item) => item.route);
    const allRoutes = NAV_GROUPS.flatMap((group) => group.items.map((item) => item.route));

    expect(workspaceRoutes).toEqual(["chat", "terminal", "settings"]);
    expect(allRoutes).toEqual(["chat", "terminal", "settings"]);
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
    ]);
  });

  it("keeps welcome prompts available for the empty state", () => {
    expect(PROMPTS).toEqual([
      {
        i18n: "prompt.journey",
        prompt: "Let's start a new journey!",
        label: "Let's start a new journey!"
      },
      {
        i18n: "prompt.skills",
        prompt: "Can you tell me what skills you have?",
        label: "Can you tell me what skills you have?"
      }
    ]);
  });
});

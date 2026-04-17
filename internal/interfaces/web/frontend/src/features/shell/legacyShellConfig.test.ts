import { NAV_GROUPS, PROMPTS } from "./legacyShellConfig";

describe("legacyShellConfig", () => {
  it("keeps the navigation groups in the shell order", () => {
    expect(NAV_GROUPS.map((group) => group.heading)).toEqual([
      "Workspace",
      "Agent Studio",
      "Control",
      "Settings"
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

  it("keeps workspace and settings routes in their intended groups", () => {
    const workspaceRoutes = NAV_GROUPS[0].items.map((item) => item.route);
    const settingsRoutes = NAV_GROUPS[3].items.map((item) => item.route);

    expect(workspaceRoutes).toEqual(["chat", "agent-runtime", "terminal"]);
    expect(settingsRoutes).toEqual(["channels", "models", "environments", "codex-accounts"]);
  });

  it("keeps agent and memory routes under agent studio only", () => {
    const agentStudioRoutes = NAV_GROUPS[1].items.map((item) => item.route);
    const otherRoutes = NAV_GROUPS.filter((_, index) => index !== 1).flatMap((group) =>
      group.items.map((item) => item.route),
    );

    expect(agentStudioRoutes).toContain("agent");
    expect(agentStudioRoutes).toContain("memory");
    expect(agentStudioRoutes).not.toContain("configuration");
    expect(otherRoutes).not.toContain("configuration");
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

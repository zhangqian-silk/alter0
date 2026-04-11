import { NAV_GROUPS, PROMPTS } from "./legacyShellConfig";

describe("legacyShellConfig", () => {
  it("keeps chat as the only active workspace entry", () => {
    const activeItems = NAV_GROUPS.flatMap((group) => group.items).filter((item) => item.active);

    expect(activeItems).toHaveLength(1);
    expect(activeItems[0]).toMatchObject({
      route: "chat",
      label: "Chat"
    });
  });

  it("keeps navigation routes unique across groups", () => {
    const routes = NAV_GROUPS.flatMap((group) => group.items.map((item) => item.route));
    const uniqueRoutes = new Set(routes);

    expect(uniqueRoutes.size).toBe(routes.length);
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

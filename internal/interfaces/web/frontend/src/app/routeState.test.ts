import {
  DEFAULT_WORKBENCH_ROUTE,
  isConversationRoute,
  navigateWorkbenchRoute,
  parseWorkbenchHashRoute,
} from "./routeState";

describe("routeState", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    window.location.hash = "";
  });

  it("keeps chat as the fallback route for empty and unknown hashes", () => {
    expect(parseWorkbenchHashRoute("")).toBe(DEFAULT_WORKBENCH_ROUTE);
    expect(parseWorkbenchHashRoute("#unknown")).toBe(DEFAULT_WORKBENCH_ROUTE);
  });

  it("writes the normalized route hash for known and unknown routes", () => {
    navigateWorkbenchRoute("tasks");
    expect(window.location.hash).toBe("#tasks");

    navigateWorkbenchRoute("not-a-route");
    expect(window.location.hash).toBe("#chat");
  });

  it("emits a hashchange event when navigating to the already active route", () => {
    window.location.hash = "#tasks";
    const dispatchEventSpy = vi.spyOn(window, "dispatchEvent");

    navigateWorkbenchRoute("tasks");

    expect(dispatchEventSpy).toHaveBeenCalledWith(expect.any(HashChangeEvent));
  });

  it("identifies conversation routes explicitly", () => {
    expect(isConversationRoute("chat")).toBe(true);
    expect(isConversationRoute("agent-runtime")).toBe(true);
    expect(isConversationRoute("tasks")).toBe(false);
  });
});

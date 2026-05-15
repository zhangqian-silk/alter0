import {
  DEFAULT_WORKBENCH_ROUTE,
  isConversationRoute,
  navigateWorkbenchRoute,
  parseWorkbenchHashRoute,
  readWorkbenchRouteSessionID,
  writeWorkbenchRouteSessionID,
} from "./routeState";

describe("routeState", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    window.location.hash = "";
    window.history.replaceState({}, "", "/");
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

  it("reads and writes route-specific session query parameters without clobbering the rest of the URL", () => {
    window.history.replaceState({}, "", "/workspace?foo=bar&chat_session_id=session-chat-1#terminal");

    expect(readWorkbenchRouteSessionID("chat")).toBe("session-chat-1");
    expect(readWorkbenchRouteSessionID("terminal")).toBe("");

    writeWorkbenchRouteSessionID("terminal", "terminal-9");

    expect(window.location.search).toContain("foo=bar");
    expect(window.location.search).toContain("chat_session_id=session-chat-1");
    expect(window.location.search).toContain("terminal_session_id=terminal-9");
    expect(window.location.hash).toBe("#terminal");

    writeWorkbenchRouteSessionID("chat", "");

    expect(window.location.search).not.toContain("chat_session_id=");
    expect(window.location.search).toContain("terminal_session_id=terminal-9");
  });
});

import {
  DEFAULT_WORKBENCH_ROUTE,
  isConversationRoute,
  navigateWorkbenchRoute,
  parseWorkbenchRoute,
  readWorkbenchRouteSessionID,
  writeWorkbenchRouteSessionID,
} from "./routeState";

describe("routeState", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    window.location.hash = "";
    window.history.replaceState({}, "", "/");
  });

  it("keeps chat as the fallback route for root and unknown paths", () => {
    expect(parseWorkbenchRoute("/")).toBe(DEFAULT_WORKBENCH_ROUTE);
    expect(parseWorkbenchRoute("/unknown")).toBe(DEFAULT_WORKBENCH_ROUTE);
  });

  it("maps only the stable top-level workbench routes to canonical paths", () => {
    expect(parseWorkbenchRoute("/chat")).toBe("chat");
    expect(parseWorkbenchRoute("/settings")).toBe("settings");
    expect(parseWorkbenchRoute("/chatRuntime")).toBe(DEFAULT_WORKBENCH_ROUTE);
    expect(parseWorkbenchRoute("/skill")).toBe(DEFAULT_WORKBENCH_ROUTE);
    expect(parseWorkbenchRoute("/memory")).toBe(DEFAULT_WORKBENCH_ROUTE);
    expect(parseWorkbenchRoute("/skills")).toBe(DEFAULT_WORKBENCH_ROUTE);
    expect(parseWorkbenchRoute("/mcp")).toBe(DEFAULT_WORKBENCH_ROUTE);
    expect(parseWorkbenchRoute("/sessions")).toBe(DEFAULT_WORKBENCH_ROUTE);
    expect(parseWorkbenchRoute("/tasks")).toBe(DEFAULT_WORKBENCH_ROUTE);
    expect(parseWorkbenchRoute("/cron-jobs")).toBe(DEFAULT_WORKBENCH_ROUTE);
    expect(parseWorkbenchRoute("/channels")).toBe(DEFAULT_WORKBENCH_ROUTE);
    expect(parseWorkbenchRoute("/models")).toBe(DEFAULT_WORKBENCH_ROUTE);
    expect(parseWorkbenchRoute("/unknown")).toBe(DEFAULT_WORKBENCH_ROUTE);
    expect(parseWorkbenchRoute("/codex-accounts")).toBe(DEFAULT_WORKBENCH_ROUTE);
  });

  it("writes canonical paths for top-level workspace routes only", () => {
    navigateWorkbenchRoute("settings");
    expect(window.location.pathname).toBe("/settings");
    expect(window.location.search).toBe("");
    expect(window.location.hash).toBe("");

    navigateWorkbenchRoute("chatRuntime");
    expect(window.location.pathname).toBe("/chat");
    expect(window.location.search).toBe("");
    expect(window.location.hash).toBe("");

    navigateWorkbenchRoute("not-a-route");
    expect(window.location.pathname).toBe("/chat");
    expect(window.location.search).toBe("");
    expect(window.location.hash).toBe("");
  });

  it("clears stale Chat session query when re-entering Chat from the primary route", () => {
    window.history.replaceState({}, "", "/chat?foo=bar&session_id=c_olderchat0000001");

    navigateWorkbenchRoute("chat");

    expect(window.location.pathname).toBe("/chat");
    expect(window.location.search).toBe("?foo=bar");
  });

  it("does not preserve old settings subpage paths as workbench routes", () => {
    navigateWorkbenchRoute("codex-accounts");

    expect(window.location.pathname).toBe("/chat");
    expect(parseWorkbenchRoute("/codex-accounts")).toBe(DEFAULT_WORKBENCH_ROUTE);
  });

  it("emits a route change event when navigating to the already active route", () => {
    window.history.replaceState({}, "", "/settings");
    const dispatchEventSpy = vi.spyOn(window, "dispatchEvent");

    navigateWorkbenchRoute("settings");

    expect(dispatchEventSpy).toHaveBeenCalledWith(expect.any(PopStateEvent));
  });

  it("identifies conversation routes explicitly", () => {
    expect(isConversationRoute("chat")).toBe(true);
    expect(isConversationRoute("settings")).toBe(false);
    expect(isConversationRoute("tasks")).toBe(false);
  });

  it("reads and writes canonical Chat session ids without clobbering other filters", () => {
    window.history.replaceState({}, "", "/chat?foo=bar&session_id=c_sessionchat00001");

    expect(readWorkbenchRouteSessionID("chat")).toBe("c_sessionchat00001");

    writeWorkbenchRouteSessionID("chat", "c_1234567890abcdef");

    expect(window.location.search).toContain("foo=bar");
    expect(window.location.search).toContain("session_id=c_1234567890abcdef");
    expect(window.location.pathname).toBe("/chat");
    expect(window.location.hash).toBe("");

    writeWorkbenchRouteSessionID("chat", "");

    expect(window.location.search).not.toContain("session_id=");
    expect(window.location.search).toContain("foo=bar");
  });

  it("rejects legacy and malformed Chat session ids in route state", () => {
    expect(readWorkbenchRouteSessionID("chat", "?session_id=alter0-chat")).toBe("");
    expect(readWorkbenchRouteSessionID("chat", "?session_id=chat-20260707T051709.110973500-f01ec2b780bbdb0d")).toBe("");
    expect(readWorkbenchRouteSessionID("chat", "?session_id=c_short")).toBe("");
    expect(readWorkbenchRouteSessionID("chat", "?session_id=c_51jttwiv4yggqagk")).toBe("c_51jttwiv4yggqagk");

    window.history.replaceState({}, "", "/chat?foo=bar&session_id=c_51jttwiv4yggqagk");

    writeWorkbenchRouteSessionID("chat", "alter0-chat");

    expect(window.location.pathname).toBe("/chat");
    expect(window.location.search).toBe("?foo=bar");
  });
});

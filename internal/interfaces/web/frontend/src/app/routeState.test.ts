import {
  DEFAULT_WORKBENCH_ROUTE,
  isConversationRoute,
  navigateWorkbenchRoute,
  parseWorkbenchRoute,
  readWorkbenchRouteSessionID,
  writeWorkbenchRouteSessionID,
} from "./routeState";
import { hashSessionIDShort } from "../shared/session/sessionHash";

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
    expect(parseWorkbenchRoute("/agent-runtime")).toBe("chat");
    expect(parseWorkbenchRoute("/terminal")).toBe("terminal");
    expect(parseWorkbenchRoute("/settings")).toBe("settings");
    expect(parseWorkbenchRoute("/management")).toBe("settings");
    expect(parseWorkbenchRoute("/agent")).toBe(DEFAULT_WORKBENCH_ROUTE);
    expect(parseWorkbenchRoute("/memory")).toBe(DEFAULT_WORKBENCH_ROUTE);
    expect(parseWorkbenchRoute("/skills")).toBe(DEFAULT_WORKBENCH_ROUTE);
    expect(parseWorkbenchRoute("/mcp")).toBe(DEFAULT_WORKBENCH_ROUTE);
    expect(parseWorkbenchRoute("/sessions")).toBe(DEFAULT_WORKBENCH_ROUTE);
    expect(parseWorkbenchRoute("/tasks")).toBe(DEFAULT_WORKBENCH_ROUTE);
    expect(parseWorkbenchRoute("/cron-jobs")).toBe(DEFAULT_WORKBENCH_ROUTE);
    expect(parseWorkbenchRoute("/channels")).toBe(DEFAULT_WORKBENCH_ROUTE);
    expect(parseWorkbenchRoute("/models")).toBe(DEFAULT_WORKBENCH_ROUTE);
    expect(parseWorkbenchRoute("/environments")).toBe(DEFAULT_WORKBENCH_ROUTE);
    expect(parseWorkbenchRoute("/codex-accounts")).toBe(DEFAULT_WORKBENCH_ROUTE);
  });

  it("writes canonical paths for top-level workspace routes only", () => {
    navigateWorkbenchRoute("settings");
    expect(window.location.pathname).toBe("/settings");
    expect(window.location.search).toBe("");
    expect(window.location.hash).toBe("");

    navigateWorkbenchRoute("terminal");
    expect(window.location.pathname).toBe("/terminal");
    expect(window.location.search).toBe("");
    expect(window.location.hash).toBe("");

    navigateWorkbenchRoute("not-a-route");
    expect(window.location.pathname).toBe("/chat");
    expect(window.location.search).toBe("");
    expect(window.location.hash).toBe("");
  });

  it("clears stale Chat session query when re-entering Chat from the primary route", () => {
    window.history.replaceState({}, "", `/chat?foo=bar&session_id=${hashSessionIDShort("older-chat-session")}`);

    navigateWorkbenchRoute("chat");

    expect(window.location.pathname).toBe("/chat");
    expect(window.location.search).toBe("?foo=bar");
  });

  it("does not preserve old management subpage paths as workbench routes", () => {
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
    expect(isConversationRoute("agent-runtime")).toBe(false);
    expect(isConversationRoute("settings")).toBe(false);
    expect(isConversationRoute("tasks")).toBe(false);
  });

  it("reads legacy agent-runtime session query parameters through Chat and writes compact short hashes without clobbering other filters", () => {
    window.history.replaceState({}, "", "/agent-runtime?foo=bar&session_id=session-agent-1");

    expect(readWorkbenchRouteSessionID("chat")).toBe("session-agent-1");

    writeWorkbenchRouteSessionID("terminal", "terminal-9");

    expect(window.location.search).toContain("foo=bar");
    expect(window.location.search).toContain(`session_id=${hashSessionIDShort("terminal-9")}`);
    expect(window.location.search).not.toContain("session_id=terminal-9");
    expect(window.location.pathname).toBe("/terminal");
    expect(window.location.hash).toBe("");

    writeWorkbenchRouteSessionID("terminal", "");

    expect(window.location.search).not.toContain("session_id=");
    expect(window.location.search).toContain("foo=bar");
  });
});

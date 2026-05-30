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

  it("maps every workbench page to its canonical path", () => {
    expect(parseWorkbenchRoute("/chat")).toBe("chat");
    expect(parseWorkbenchRoute("/agent-runtime")).toBe("agent-runtime");
    expect(parseWorkbenchRoute("/terminal")).toBe("terminal");
    expect(parseWorkbenchRoute("/agent")).toBe("agent");
    expect(parseWorkbenchRoute("/memory")).toBe("memory");
    expect(parseWorkbenchRoute("/skills")).toBe("skills");
    expect(parseWorkbenchRoute("/mcp")).toBe("mcp");
    expect(parseWorkbenchRoute("/sessions")).toBe("sessions");
    expect(parseWorkbenchRoute("/tasks")).toBe("tasks");
    expect(parseWorkbenchRoute("/cron-jobs")).toBe("cron-jobs");
    expect(parseWorkbenchRoute("/channels")).toBe("channels");
    expect(parseWorkbenchRoute("/models")).toBe("models");
    expect(parseWorkbenchRoute("/environments")).toBe("environments");
    expect(parseWorkbenchRoute("/codex-accounts")).toBe("codex-accounts");
  });

  it("writes canonical paths for all workspace routes", () => {
    navigateWorkbenchRoute("tasks");
    expect(window.location.pathname).toBe("/tasks");
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

  it("emits a route change event when navigating to the already active route", () => {
    window.history.replaceState({}, "", "/tasks");
    const dispatchEventSpy = vi.spyOn(window, "dispatchEvent");

    navigateWorkbenchRoute("tasks");

    expect(dispatchEventSpy).toHaveBeenCalledWith(expect.any(PopStateEvent));
  });

  it("identifies conversation routes explicitly", () => {
    expect(isConversationRoute("chat")).toBe(true);
    expect(isConversationRoute("agent-runtime")).toBe(true);
    expect(isConversationRoute("tasks")).toBe(false);
  });

  it("reads shared session query parameters and writes compact short hashes without clobbering other filters", () => {
    window.history.replaceState({}, "", "/agent-runtime?foo=bar&session_id=session-agent-1");

    expect(readWorkbenchRouteSessionID("agent-runtime")).toBe("session-agent-1");

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

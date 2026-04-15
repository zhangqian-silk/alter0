import {
  LEGACY_SHELL_CREATE_SESSION_EVENT,
  LEGACY_SHELL_FOCUS_SESSION_EVENT,
  LEGACY_SHELL_REMOVE_SESSION_EVENT,
  requestLegacyShellSessionCreation,
  requestLegacyShellSessionFocus,
  requestLegacyShellSessionRemoval,
} from "./legacyShellBridge";

declare global {
  interface Window {
    __alter0LegacyRuntime?: {
      createSession?: () => boolean | void;
      focusSession?: (sessionId: string) => boolean | void;
      removeSession?: (sessionId: string) => boolean | void;
    };
  }
}

describe("legacyShellBridge", () => {
  beforeEach(() => {
    delete window.__alter0LegacyRuntime;
  });

  it("falls back to the DOM event bridge for session creation when no runtime api is available", () => {
    const events: string[] = [];
    document.addEventListener(
      LEGACY_SHELL_CREATE_SESSION_EVENT,
      (() => {
        events.push("create");
      }) as EventListener,
      { once: true },
    );

    expect(requestLegacyShellSessionCreation()).toBe(true);
    expect(events).toEqual(["create"]);
  });

  it("prefers the runtime api for session actions when available", () => {
    const events: string[] = [];
    document.addEventListener(
      LEGACY_SHELL_CREATE_SESSION_EVENT,
      (() => {
        events.push("create");
      }) as EventListener,
    );
    document.addEventListener(
      LEGACY_SHELL_FOCUS_SESSION_EVENT,
      (() => {
        events.push("focus");
      }) as EventListener,
    );
    document.addEventListener(
      LEGACY_SHELL_REMOVE_SESSION_EVENT,
      (() => {
        events.push("remove");
      }) as EventListener,
    );

    const createSession = vi.fn(() => true);
    const focusSession = vi.fn(() => true);
    const removeSession = vi.fn(() => true);
    window.__alter0LegacyRuntime = {
      createSession,
      focusSession,
      removeSession,
    };

    expect(requestLegacyShellSessionCreation()).toBe(true);
    expect(requestLegacyShellSessionFocus("session-1")).toBe(true);
    expect(requestLegacyShellSessionRemoval("session-1")).toBe(true);

    expect(createSession).toHaveBeenCalledTimes(1);
    expect(focusSession).toHaveBeenCalledWith("session-1");
    expect(removeSession).toHaveBeenCalledWith("session-1");
    expect(events).toEqual([]);
  });
});

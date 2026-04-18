import {
  LEGACY_SHELL_CREATE_SESSION_EVENT,
  LEGACY_SHELL_FOCUS_SESSION_EVENT,
  LEGACY_SHELL_REMOVE_SESSION_EVENT,
  requestLegacyChatRuntimeClose,
  requestLegacyChatRuntimeItem,
  requestLegacyChatRuntimeModel,
  requestLegacyChatRuntimePopover,
  requestLegacyChatRuntimeTarget,
  requestLegacyShellSessionCreation,
  requestLegacyShellSessionFocus,
  requestLegacyShellSessionRemoval,
} from "./legacyShellBridge";
import {
  ensureLegacyRuntimeSnapshotBridge,
  readLegacyRuntimeSnapshot,
  resetLegacyRuntimeSnapshotStore,
} from "./legacyRuntimeSnapshotStore";

describe("legacyShellBridge", () => {
  beforeEach(() => {
    delete window.__alter0LegacyRuntime;
    resetLegacyRuntimeSnapshotStore();
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

  it("routes chat runtime actions through the shared runtime api when available", () => {
    const toggleChatRuntimePopover = vi.fn(() => true);
    const closeChatRuntimePopover = vi.fn(() => true);
    const selectChatRuntimeTarget = vi.fn(() => true);
    const selectChatRuntimeModel = vi.fn(() => true);
    const toggleChatRuntimeItem = vi.fn(() => true);

    window.__alter0LegacyRuntime = {
      toggleChatRuntimePopover,
      closeChatRuntimePopover,
      selectChatRuntimeTarget,
      selectChatRuntimeModel,
      toggleChatRuntimeItem,
    };

    expect(requestLegacyChatRuntimePopover("model")).toBe(true);
    expect(requestLegacyChatRuntimeClose()).toBe(true);
    expect(
      requestLegacyChatRuntimeTarget({
        type: "agent",
        id: "planner",
        name: "Planner",
      }),
    ).toBe(true);
    expect(
      requestLegacyChatRuntimeModel({
        providerId: "openai",
        modelId: "gpt-5.4",
      }),
    ).toBe(true);
    expect(
      requestLegacyChatRuntimeItem({
        group: "capabilities",
        id: "memory",
        checked: true,
        kind: "tool",
      }),
    ).toBe(true);

    expect(toggleChatRuntimePopover).toHaveBeenCalledWith("model");
    expect(closeChatRuntimePopover).toHaveBeenCalledTimes(1);
    expect(selectChatRuntimeTarget).toHaveBeenCalledWith({
      type: "agent",
      id: "planner",
      name: "Planner",
    });
    expect(selectChatRuntimeModel).toHaveBeenCalledWith({
      providerId: "openai",
      modelId: "gpt-5.4",
    });
    expect(toggleChatRuntimeItem).toHaveBeenCalledWith({
      group: "capabilities",
      id: "memory",
      checked: true,
      kind: "tool",
    });
  });

  it("installs runtime snapshot publishers onto the shared legacy runtime bridge", () => {
    ensureLegacyRuntimeSnapshotBridge();

    expect(
      window.__alter0LegacyRuntime?.publishSessionPaneSnapshot?.({
        route: "chat",
        hasSessions: true,
        loadError: "",
        items: [],
      }),
    ).toBe(true);

    expect(readLegacyRuntimeSnapshot("alter0:legacy-shell:sync-session-pane")).toMatchObject({
      route: "chat",
      hasSessions: true,
    });
  });
});

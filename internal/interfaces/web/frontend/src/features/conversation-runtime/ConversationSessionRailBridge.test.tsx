import { render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { WorkbenchContext, type WorkbenchContextValue, type WorkbenchSessionRail } from "../../app/WorkbenchContext";
import { ConversationSessionRailBridge } from "./ConversationSessionRailBridge";

const { runtimeMock } = vi.hoisted(() => ({
  runtimeMock: {
    route: "chat" as const,
    sessionItems: [] as Array<{
      id: string;
      title: string;
      meta: string;
      createdAt: number;
      updatedAt: number;
      active: boolean;
      draft: boolean;
      pinned: boolean;
      pinning: boolean;
    }>,
    sessions: [] as Array<{
      id: string;
      status: string;
      messages: Array<{ role?: string; status?: string; error?: boolean }>;
    }>,
    createSession: vi.fn(),
    focusSession: vi.fn(),
    removeSession: vi.fn().mockResolvedValue(undefined),
    setSessionPinned: vi.fn().mockResolvedValue(undefined),
  },
}));

vi.mock("./ConversationRuntimeProvider", () => ({
  useConversationRuntimeWorkspace: () => runtimeMock,
}));

function buildWorkbenchContext(setRuntimeSessionRail: (rail: WorkbenchSessionRail | null) => void): WorkbenchContextValue {
  return {
    route: "chat",
    language: "en",
    navigate: vi.fn(),
    isMobileViewport: true,
    mobileNavOpen: true,
    mobileSessionPaneOpen: false,
    toggleMobileNav: vi.fn(),
    toggleMobileSessionPane: vi.fn(),
    openMobileSessionPane: vi.fn(),
    closeMobileNav: vi.fn(),
    closeMobileSessionPane: vi.fn(),
    setRuntimeSessionRail,
  };
}

function renderBridge(setRuntimeSessionRail: (rail: WorkbenchSessionRail | null) => void) {
  const contextValue = buildWorkbenchContext(setRuntimeSessionRail);
  return render(
    <WorkbenchContext.Provider value={contextValue}>
      <ConversationSessionRailBridge language="en" />
    </WorkbenchContext.Provider>,
  );
}

describe("ConversationSessionRailBridge", () => {
  afterEach(() => {
    vi.useRealTimers();
    runtimeMock.createSession.mockClear();
    runtimeMock.focusSession.mockClear();
    runtimeMock.removeSession.mockClear();
    runtimeMock.setSessionPinned.mockClear();
  });

  it("registers navigation session groups by updated time instead of creation time", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-04-23T12:00:00Z"));
    runtimeMock.sessionItems = [
      {
        id: "session-updated-today",
        title: "Updated Today",
        meta: "2026-04-23 11:00",
        createdAt: Date.parse("2026-04-22T09:00:00Z"),
        updatedAt: Date.parse("2026-04-23T11:00:00Z"),
        active: true,
        draft: false,
        pinned: false,
        pinning: false,
      },
    ];
    runtimeMock.sessions = [
      {
        id: "session-updated-today",
        status: "ready",
        messages: [],
      },
    ];
    const setRuntimeSessionRail = vi.fn();

    renderBridge(setRuntimeSessionRail);

    expect(setRuntimeSessionRail).toHaveBeenCalledWith(expect.objectContaining({
      route: "chat",
    }));
    const rail = setRuntimeSessionRail.mock.calls.at(-1)?.[0] as WorkbenchSessionRail;
    render(
      <WorkbenchContext.Provider value={buildWorkbenchContext(setRuntimeSessionRail)}>
        {rail.body}
      </WorkbenchContext.Provider>,
    );

    const sessionList = screen.getByRole("list");
    expect(within(sessionList).getByText("Today")).toBeInTheDocument();
    expect(within(sessionList).queryByText("Yesterday")).not.toBeInTheDocument();
    expect(within(sessionList).getByText("Updated Today")).toBeInTheDocument();
  });

  it("registers sessions nearing cleanup under an expiring soon group", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-04-23T12:00:00Z"));
    runtimeMock.sessionItems = [
      {
        id: "session-expiring-soon",
        title: "Review Before Cleanup",
        meta: "2026-04-18 20:00",
        createdAt: Date.parse("2026-04-10T12:00:00Z"),
        updatedAt: Date.parse("2026-04-18T12:00:00Z"),
        active: false,
        draft: false,
        pinned: false,
        pinning: false,
      },
    ];
    runtimeMock.sessions = [
      {
        id: "session-expiring-soon",
        status: "ready",
        messages: [],
      },
    ];
    const setRuntimeSessionRail = vi.fn();

    renderBridge(setRuntimeSessionRail);

    const rail = setRuntimeSessionRail.mock.calls.at(-1)?.[0] as WorkbenchSessionRail;
    render(
      <WorkbenchContext.Provider value={buildWorkbenchContext(setRuntimeSessionRail)}>
        {rail.body}
      </WorkbenchContext.Provider>,
    );

    const sessionList = screen.getByRole("list");
    expect(within(sessionList).getByText("Expiring Soon")).toBeInTheDocument();
    expect(within(sessionList).queryByText("Earlier")).not.toBeInTheDocument();
    expect(within(sessionList).getByText("Review Before Cleanup")).toBeInTheDocument();
  });
});

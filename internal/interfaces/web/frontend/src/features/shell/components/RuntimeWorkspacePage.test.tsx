import { render, waitFor, within } from "@testing-library/react";
import { WorkbenchContext, type WorkbenchContextValue } from "../../../app/WorkbenchContext";
import { RuntimeWorkspacePage, type RuntimeWorkspacePageController } from "./RuntimeWorkspacePage";

function buildController(): RuntimeWorkspacePageController {
  return {
    shell: {
      rootClassName: "runtime-workspace-view",
      sessionPaneBackdrop: {
        ariaLabel: "Hide",
        onClick: vi.fn(),
      },
      sessionPaneTitle: "Sessions",
      sessionPaneCountLabel: "1 sessions",
      sessionPanePrimaryActionLabel: "New",
      onSessionPanePrimaryAction: vi.fn(),
    },
    sessionList: {
      groups: [
        {
          key: "today",
          label: "Today",
          items: [
            {
              id: "session-1",
              active: true,
              title: "Design Review",
              meta: "now",
              shortHash: "a1b2c3d4",
              activeLabel: "Active",
              idleLabel: "Open",
              statusTone: "busy",
              statusLabel: "Busy",
              onSelect: vi.fn(),
            },
            {
              id: "session-2",
              active: false,
              title: "Ready Notes",
              contextLabel: "Research Agent",
              meta: "1 hr ago",
              shortHash: "b2c3d4e5",
              activeLabel: "Active",
              idleLabel: "Open",
              statusTone: "ready",
              statusLabel: "Ready",
              onSelect: vi.fn(),
            },
          ],
        },
      ],
    },
    header: {
      title: "Design Review",
      statusLabel: "Ready",
      statusTone: "ready",
      detailsLabel: "Details",
      detailsOpen: false,
      onToggleDetails: vi.fn(),
    },
    screen: {},
    timeline: {
      items: [],
    },
  };
}

function renderRuntimeWorkspacePage(setRuntimeSessionRail = vi.fn()) {
  const contextValue: WorkbenchContextValue = {
    route: "chat",
    language: "en",
    navigate: vi.fn(),
    isMobileViewport: false,
    mobileNavOpen: false,
    mobileSessionPaneOpen: false,
    toggleMobileNav: vi.fn(),
    toggleMobileSessionPane: vi.fn(),
    openMobileSessionPane: vi.fn(),
    closeMobileNav: vi.fn(),
    closeMobileSessionPane: vi.fn(),
    setRuntimeSessionRail,
  };

  const view = render(
    <WorkbenchContext.Provider value={contextValue}>
      <RuntimeWorkspacePage controller={buildController()} />
    </WorkbenchContext.Provider>,
  );

  return { ...view, setRuntimeSessionRail };
}

describe("RuntimeWorkspacePage", () => {
  it("registers the runtime session list for the primary navigation rail", async () => {
    const { container, setRuntimeSessionRail } = renderRuntimeWorkspacePage();

    await waitFor(() => {
      expect(setRuntimeSessionRail).toHaveBeenCalledWith(expect.objectContaining({
        route: "chat",
        title: "Sessions",
        countLabel: "1 sessions",
        primaryActionLabel: "New",
      }));
    });
    const rail = setRuntimeSessionRail.mock.calls.at(-1)?.[0];
    const railView = render(<>{rail.body}</>);

    expect(within(railView.container).getByText("Design Review")).toBeInTheDocument();
    expect(within(railView.container).getByText("Ready Notes")).toBeInTheDocument();
    expect(railView.container.querySelector(".runtime-session-summary-row")).not.toBeInTheDocument();
    expect(railView.container.querySelector(".runtime-session-context")).not.toBeInTheDocument();
    expect(railView.container.querySelector(".runtime-session-signal")).not.toBeInTheDocument();
    expect(railView.container.querySelectorAll(".runtime-session-loading")).toHaveLength(1);
    expect(within(railView.container).queryByText("1 hr ago")).not.toBeInTheDocument();
    expect(within(railView.container).queryByText("#b2c3d4e5")).not.toBeInTheDocument();
    expect(container.querySelector("[data-runtime-session-pane='chat']")).toHaveAttribute(
      "data-session-pane-placement",
      "navigation",
    );
  });

  it("clears the registered primary navigation session list on unmount", async () => {
    const { unmount, setRuntimeSessionRail } = renderRuntimeWorkspacePage();

    await waitFor(() => {
      expect(setRuntimeSessionRail).toHaveBeenCalled();
    });

    unmount();

    expect(setRuntimeSessionRail).toHaveBeenLastCalledWith(null);
  });
});

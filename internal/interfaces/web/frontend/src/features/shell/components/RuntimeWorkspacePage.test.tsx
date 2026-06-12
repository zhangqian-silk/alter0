import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
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
              contextLabel: "Research Runtime",
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

function renderRuntimeWorkspacePage(
  setRuntimeSessionRail = vi.fn(),
  controller = buildController(),
) {
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
      <RuntimeWorkspacePage controller={controller} />
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
        countLabel: "1 sessions",
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

  it("opens session actions from a compact more menu and confirms delete", () => {
    const controller = buildController();
    const onPinnedChange = vi.fn();
    const onViewDetails = vi.fn();
    const onDelete = vi.fn();
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    Object.assign(controller.sessionList.groups[0].items[1], {
      pinned: true,
      pinLabel: "Pin",
      unpinLabel: "Unpin",
      pinAriaLabel: "Pin session",
      unpinAriaLabel: "Unpin session",
      onPinnedChange,
      onViewDetails,
      viewDetailsLabel: "Details",
      viewDetailsAriaLabel: "View session details",
      onDelete,
      deleteLabel: "Delete",
      deleteAriaLabel: "Delete session",
      deleteConfirmLabel: "Delete this session?",
    });

    try {
      renderRuntimeWorkspacePage(vi.fn(), controller);

      expect(screen.queryByRole("button", { name: "Unpin session", hidden: true })).not.toBeInTheDocument();
      expect(screen.queryByRole("button", { name: "View session details", hidden: true })).not.toBeInTheDocument();
      expect(screen.queryByRole("button", { name: "Delete session", hidden: true })).not.toBeInTheDocument();

      fireEvent.click(screen.getByRole("button", { name: "Session actions", hidden: true }));

      fireEvent.click(screen.getByRole("menuitem", { name: "Unpin session", hidden: true }));
      expect(onPinnedChange).toHaveBeenCalledWith(false);
      expect(onViewDetails).not.toHaveBeenCalled();
      expect(onDelete).not.toHaveBeenCalled();

      fireEvent.click(screen.getByRole("button", { name: "Session actions", hidden: true }));
      fireEvent.click(screen.getByRole("menuitem", { name: "View session details", hidden: true }));
      expect(onViewDetails).toHaveBeenCalledTimes(1);
      expect(onDelete).not.toHaveBeenCalled();

      fireEvent.click(screen.getByRole("button", { name: "Session actions", hidden: true }));
      fireEvent.click(screen.getByRole("menuitem", { name: "Delete session", hidden: true }));
      expect(confirmSpy).toHaveBeenCalledWith("Delete this session?");
      expect(onDelete).not.toHaveBeenCalled();

      confirmSpy.mockReturnValue(true);
      fireEvent.click(screen.getByRole("button", { name: "Session actions", hidden: true }));
      fireEvent.click(screen.getByRole("menuitem", { name: "Delete session", hidden: true }));
      expect(onDelete).toHaveBeenCalledTimes(1);
    } finally {
      confirmSpy.mockRestore();
    }
  });

});

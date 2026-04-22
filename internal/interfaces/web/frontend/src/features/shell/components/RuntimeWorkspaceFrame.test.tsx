import { fireEvent, render, screen } from "@testing-library/react";
import { RuntimeWorkspaceFrame } from "./RuntimeWorkspaceFrame";

describe("RuntimeWorkspaceFrame", () => {
  it("renders a shared runtime scaffold with slot-based session, header, body, and footer regions", () => {
    const onClosePane = vi.fn();

    render(
      <RuntimeWorkspaceFrame
        rootClassName="runtime-view"
        rootProps={{ "data-testid": "runtime-view" }}
        sessionPaneClassName="runtime-session-pane is-open"
        sessionPaneProps={{ "data-testid": "runtime-session-pane" }}
        sessionPaneBackdrop={{
          ariaLabel: "Close sessions",
          onClick: onClosePane,
        }}
        sessionPaneShellClassName="runtime-session-pane-shell"
        sessionPaneContent={<div data-testid="runtime-session-content">sessions</div>}
        workspaceClassName="runtime-workspace"
        workspaceProps={{ "data-testid": "runtime-workspace" }}
        workspaceBodyClassName="runtime-workspace-body"
        mobileHeader={<div data-testid="runtime-mobile-header">mobile actions</div>}
        workspaceHeader={<div data-testid="runtime-header">header</div>}
        workspaceContent={<div data-testid="runtime-content">content</div>}
        workspaceFooter={<div data-testid="runtime-footer">footer</div>}
      />,
    );

    expect(screen.getByTestId("runtime-view")).toHaveClass("runtime-view");
    expect(screen.getByTestId("runtime-session-pane")).toHaveClass("runtime-session-pane", "is-open");
    expect(screen.getByTestId("runtime-session-content")).toBeInTheDocument();
    expect(screen.getByTestId("runtime-workspace")).toHaveClass("runtime-workspace");
    expect(screen.getByTestId("runtime-mobile-header")).toBeInTheDocument();
    expect(screen.getByTestId("runtime-header")).toBeInTheDocument();
    expect(screen.getByTestId("runtime-content")).toBeInTheDocument();
    expect(screen.getByTestId("runtime-footer")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Close sessions" }));
    expect(onClosePane).toHaveBeenCalledTimes(1);
  });

  it("omits optional regions when slots are not provided", () => {
    render(
      <RuntimeWorkspaceFrame
        rootClassName="runtime-view"
        sessionPaneClassName="runtime-session-pane"
        sessionPaneShellClassName="runtime-session-pane-shell"
        sessionPaneContent={<div>sessions</div>}
        workspaceClassName="runtime-workspace"
        workspaceBodyClassName="runtime-workspace-body"
        workspaceContent={<div>content</div>}
      />,
    );

    expect(screen.queryByRole("button")).not.toBeInTheDocument();
    expect(screen.getByText("sessions")).toBeInTheDocument();
    expect(screen.getByText("content")).toBeInTheDocument();
  });
});

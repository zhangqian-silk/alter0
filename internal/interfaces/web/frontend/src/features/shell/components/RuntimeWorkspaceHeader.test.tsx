import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { RuntimeWorkspaceHeader } from "./RuntimeWorkspaceHeader";

function RuntimeWorkspaceHeaderHarness() {
  const [detailsOpen, setDetailsOpen] = useState(false);

  return (
    <RuntimeWorkspaceHeader
      title="Runtime session"
      statusLabel="Ready"
      statusTone="ready"
      detailsLabel="Details"
      detailsOpen={detailsOpen}
      onToggleDetails={() => setDetailsOpen((current) => !current)}
      detailsContent={<div>Session metadata</div>}
      detailsPanelProps={{ "data-testid": "runtime-details-panel" }}
    />
  );
}

describe("RuntimeWorkspaceHeader", () => {
  it("does not render a workspace flow button even if legacy props are still passed in", () => {
    render(
      <RuntimeWorkspaceHeader
        {...({
          title: "Runtime session",
          statusLabel: "Ready",
          statusTone: "ready",
          detailsLabel: "Details",
          detailsOpen: false,
          onToggleDetails: () => undefined,
          detailsContent: <div>Session metadata</div>,
          flowLabel: "Workspace Flow",
        } as never)}
      />,
    );

    expect(screen.queryByRole("button", { name: "Workspace Flow" })).not.toBeInTheDocument();
  });

  it("can render a static page title without a status signal or details action", () => {
    const { container } = render(
      <RuntimeWorkspaceHeader
        title="Settings"
        statusLabel="Ready"
        statusTone="ready"
        showStatusSignal={false}
        detailsLabel="Details"
        detailsOpen={false}
        onToggleDetails={() => undefined}
        detailsDisabled
        showDetailsAction={false}
      />,
    );

    expect(screen.getByRole("heading", { name: "Settings" })).toBeInTheDocument();
    expect(container.querySelector("[data-runtime-header-signal]")).not.toBeInTheDocument();
    expect(container.querySelector("[data-runtime-header-signal-slot='empty']")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Details" })).not.toBeInTheDocument();
  });

  it("keeps the title text in the same column when the status signal is hidden", () => {
    const { container, rerender } = render(
      <RuntimeWorkspaceHeader
        title="New"
        statusLabel="Ready"
        statusTone="ready"
        detailsLabel="Details"
        detailsOpen={false}
        onToggleDetails={() => undefined}
      />,
    );

    const activeSlot = container.querySelector("[data-runtime-header-signal-slot='ready']");
    const activeTitle = container.querySelector(".runtime-workspace-title-leading h4");
    expect(activeSlot).toBeInTheDocument();
    expect(activeTitle?.previousElementSibling).toBe(activeSlot);

    rerender(
      <RuntimeWorkspaceHeader
        title="Settings"
        statusLabel="Ready"
        statusTone="ready"
        detailsLabel="Details"
        detailsOpen={false}
        onToggleDetails={() => undefined}
        showStatusSignal={false}
        showDetailsAction={false}
      />,
    );

    const emptySlot = container.querySelector("[data-runtime-header-signal-slot='empty']");
    const staticTitle = container.querySelector(".runtime-workspace-title-leading h4");
    expect(emptySlot).toBeInTheDocument();
    expect(staticTitle?.previousElementSibling).toBe(emptySlot);
  });

  it("opens details from the title and does not render a separate details button", () => {
    render(<RuntimeWorkspaceHeaderHarness />);

    const statusIndicator = screen.getByLabelText("Ready");
    expect(statusIndicator).toHaveClass("workspace-header-status", "is-ready");
    expect(statusIndicator).toHaveAttribute("data-runtime-header-signal-container", "ready");
    expect(statusIndicator.querySelector("[data-runtime-header-signal='ready']")).toBeInTheDocument();
    const titleLeading = document.querySelector(".runtime-workspace-title-leading") as HTMLElement;
    expect(titleLeading.firstElementChild).toHaveAttribute("data-runtime-header-signal-slot", "ready");
    expect(titleLeading.firstElementChild?.firstElementChild).toBe(statusIndicator);

    expect(screen.queryByRole("button", { name: "Details" })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Runtime session" }));

    expect(screen.getByRole("dialog", { name: "Details" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Dismiss Details" })).toBeInTheDocument();
    expect(screen.getByTestId("runtime-details-panel")).toHaveTextContent("Session metadata");

    fireEvent.click(screen.getByRole("button", { name: "Close Details" }));

    expect(screen.queryByRole("dialog", { name: "Details" })).not.toBeInTheDocument();
  });
});

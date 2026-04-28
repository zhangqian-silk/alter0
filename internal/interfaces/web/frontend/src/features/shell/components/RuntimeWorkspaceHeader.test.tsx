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

  it("opens details in a dialog layer and closes it from the backdrop", () => {
    render(<RuntimeWorkspaceHeaderHarness />);

    const statusIndicator = screen.getByLabelText("Ready");
    expect(statusIndicator).toHaveClass("workspace-header-status", "is-ready");
    expect(statusIndicator).toHaveAttribute("data-runtime-header-signal-container", "ready");
    expect(statusIndicator.querySelector("[data-runtime-header-signal='ready']")).toBeInTheDocument();
    const titleLeading = document.querySelector(".runtime-workspace-title-leading") as HTMLElement;
    expect(titleLeading.firstElementChild).toBe(statusIndicator);

    fireEvent.click(screen.getByRole("button", { name: "Details" }));

    expect(screen.getByRole("dialog", { name: "Details" })).toBeInTheDocument();
    expect(screen.getByTestId("runtime-details-panel")).toHaveTextContent("Session metadata");

    fireEvent.click(screen.getByRole("button", { name: "Close Details" }));

    expect(screen.queryByRole("dialog", { name: "Details" })).not.toBeInTheDocument();
  });
});

import { render, screen } from "@testing-library/react";
import { RuntimeWorkspaceScreen } from "./RuntimeWorkspaceScreen";

describe("RuntimeWorkspaceScreen", () => {
  it("renders overlay controls outside the scrollable screen body", () => {
    render(
      <RuntimeWorkspaceScreen
        screenProps={{ "data-testid": "runtime-screen" }}
        overlay={<div data-testid="runtime-overlay">jump controls</div>}
      >
        <div data-testid="runtime-content">timeline</div>
      </RuntimeWorkspaceScreen>,
    );

    const screenBody = screen.getByTestId("runtime-screen");
    const overlay = screen.getByTestId("runtime-overlay");

    expect(screenBody).toContainElement(screen.getByTestId("runtime-content"));
    expect(screenBody).not.toContainElement(overlay);
    expect(overlay.parentElement).toBe(screenBody.parentElement);
  });
});

import { render, screen } from "@testing-library/react";
import { RuntimeRouteHost } from "./RuntimeRouteHost";

vi.mock("../../conversation-runtime/ConversationRuntimeProvider", () => ({
  ConversationRuntimeProvider: ({
    route,
    children,
  }: {
    route: string;
    children: React.ReactNode;
  }) => (
    <section data-testid="conversation-runtime-provider" data-route={route}>
      {children}
    </section>
  ),
}));

vi.mock("../../conversation-runtime/ConversationWorkspace", () => ({
  ConversationWorkspace: () => <div data-testid="conversation-workspace"></div>,
}));

vi.mock("./ReactManagedTerminalRouteBody", () => ({
  ReactManagedTerminalRouteBody: () => <div data-testid="terminal-runtime-body"></div>,
}));

describe("RuntimeRouteHost", () => {
  it("keeps the terminal path as a compatibility entry backed by Chat runtime", () => {
    render(<RuntimeRouteHost route="terminal" language="en" />);

    expect(screen.getByTestId("conversation-runtime-provider")).toHaveAttribute("data-route", "terminal");
    expect(screen.getByTestId("conversation-workspace")).toBeInTheDocument();
    expect(screen.queryByTestId("terminal-runtime-body")).not.toBeInTheDocument();
  });
});

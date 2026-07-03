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

describe("RuntimeRouteHost", () => {
  it("mounts the Chat runtime workspace", () => {
    render(<RuntimeRouteHost route="chat" language="en" />);

    expect(screen.getByTestId("conversation-runtime-provider")).toHaveAttribute("data-route", "chat");
    expect(screen.getByTestId("conversation-workspace")).toBeInTheDocument();
  });
});

import { fireEvent, render, screen } from "@testing-library/react";
import { PrimaryNav } from "./PrimaryNav";

describe("PrimaryNav", () => {
  it("renders text-only brand chrome and locale controls in the expanded sidebar", () => {
    const { container } = render(
      <PrimaryNav
        currentRoute="chat"
        language="en"
        navCollapsed={false}
        onNavigate={vi.fn()}
        onToggleLanguage={vi.fn()}
        onToggleNavCollapsed={vi.fn()}
      />,
    );

    expect(container.querySelector(".brand-mark")).not.toBeInTheDocument();
    expect(screen.getByText("Alter0")).toBeInTheDocument();
    expect(container.querySelector(".nav-profile")).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Workspace" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Language" })).toBeInTheDocument();
  });

  it("keeps route navigation and the collapse control interactive", () => {
    const onNavigate = vi.fn();
    const onToggleNavCollapsed = vi.fn();

    render(
      <PrimaryNav
        currentRoute="chat"
        language="en"
        navCollapsed={false}
        onNavigate={onNavigate}
        onToggleLanguage={vi.fn()}
        onToggleNavCollapsed={onToggleNavCollapsed}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Terminal" }));
    fireEvent.click(screen.getByRole("button", { name: "Collapse navigation" }));

    expect(onNavigate).toHaveBeenCalledWith("terminal");
    expect(onToggleNavCollapsed).toHaveBeenCalledTimes(1);
  });

  it("limits the primary route surface to chat, agent, and terminal", () => {
    render(
      <PrimaryNav
        currentRoute="chat"
        language="en"
        navCollapsed={false}
        onNavigate={vi.fn()}
        onToggleLanguage={vi.fn()}
        onToggleNavCollapsed={vi.fn()}
      />,
    );

    expect(screen.getByRole("button", { name: "Chat" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Agent" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Terminal" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Memory" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Tasks" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Codex Accounts" })).not.toBeInTheDocument();
  });

  it("keeps management visible as a utility entry without expanding every management route", () => {
    const onNavigate = vi.fn();

    render(
      <PrimaryNav
        currentRoute="management"
        language="en"
        navCollapsed={false}
        onNavigate={onNavigate}
        onToggleLanguage={vi.fn()}
        onToggleNavCollapsed={vi.fn()}
      />,
    );

    const management = screen.getByRole("button", { name: "Management" });

    expect(management).toBeInTheDocument();
    expect(management).toHaveClass("active");

    fireEvent.click(management);

    expect(onNavigate).toHaveBeenCalledWith("management");
  });

  it("renders the active runtime conversation list directly in the left navigation", () => {
    const onCreate = vi.fn();

    const { container } = render(
      <PrimaryNav
        currentRoute="chat"
        language="en"
        navCollapsed={false}
        onNavigate={vi.fn()}
        onToggleLanguage={vi.fn()}
        onToggleNavCollapsed={vi.fn()}
        sessionRail={{
          route: "chat",
          title: "Sessions",
          countLabel: "2 sessions",
          primaryActionLabel: "New",
          onPrimaryAction: onCreate,
          body: (
            <div role="list">
              <div role="listitem">Design Review</div>
              <div role="listitem">Launch Plan</div>
            </div>
          ),
        }}
      />,
    );

    const rail = container.querySelector("[data-nav-session-rail='chat']") as HTMLElement;

    expect(rail).toBeInTheDocument();
    expect(rail).toHaveTextContent("Sessions");
    expect(rail).toHaveTextContent("2 sessions");
    expect(screen.getByText("Design Review")).toBeInTheDocument();
    expect(screen.getByText("Launch Plan")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "New" }));

    expect(onCreate).toHaveBeenCalledTimes(1);
  });

  it("hides the runtime conversation list when the sidebar is collapsed", () => {
    const { container } = render(
      <PrimaryNav
        currentRoute="chat"
        language="en"
        navCollapsed={true}
        onNavigate={vi.fn()}
        onToggleLanguage={vi.fn()}
        onToggleNavCollapsed={vi.fn()}
        sessionRail={{
          route: "chat",
          title: "Sessions",
          countLabel: "2 sessions",
          primaryActionLabel: "New",
          onPrimaryAction: vi.fn(),
          body: <div>Design Review</div>,
        }}
      />,
    );

    expect(container.querySelector("[data-nav-session-rail='chat']")).not.toBeInTheDocument();
  });
});

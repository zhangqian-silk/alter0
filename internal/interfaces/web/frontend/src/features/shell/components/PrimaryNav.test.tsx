import { fireEvent, render, screen } from "@testing-library/react";
import { PrimaryNav } from "./PrimaryNav";

describe("PrimaryNav", () => {
  it("renders text-only brand chrome and opens Settings from the lower-left shortcut", () => {
    const onNavigate = vi.fn();
    const { container } = render(
      <PrimaryNav
        currentRoute="chat"
        language="en"
        navCollapsed={false}
        onNavigate={onNavigate}
        onToggleNavCollapsed={vi.fn()}
      />,
    );

    expect(container.querySelector(".primary-nav")).toHaveAttribute("data-shell-design", "light-tech");
    expect(container.querySelector(".brand-mark")).not.toBeInTheDocument();
    expect(screen.getByText("Alter0")).toBeInTheDocument();
    expect(container.querySelector(".nav-profile")).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Workspace" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Settings" }));
    expect(onNavigate).toHaveBeenCalledWith("settings");
    expect(screen.queryByRole("button", { name: "Language" })).not.toBeInTheDocument();
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
        onToggleNavCollapsed={onToggleNavCollapsed}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Settings" }));
    fireEvent.click(screen.getByRole("button", { name: "Collapse navigation" }));

    expect(onNavigate).toHaveBeenCalledWith("settings");
    expect(onToggleNavCollapsed).toHaveBeenCalledTimes(1);
  });

  it("keeps Settings out of the primary route surface", () => {
    render(
      <PrimaryNav
        currentRoute="chat"
        language="en"
        navCollapsed={false}
        onNavigate={vi.fn()}
        onToggleNavCollapsed={vi.fn()}
      />,
    );

    expect(screen.getByRole("button", { name: "Chat" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Settings" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Settings" })).toHaveClass("nav-settings-shortcut");
    expect(screen.queryByRole("button", { name: "Language" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "ChatRuntime" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Skill" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Memory" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Tasks" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Codex Accounts" })).not.toBeInTheDocument();
  });

  it("opens settings from the lower-left shortcut without replacing the Chat session rail", () => {
    const onNavigate = vi.fn();
    const { container } = render(
      <PrimaryNav
        currentRoute="settings"
        language="en"
        navCollapsed={false}
        onNavigate={onNavigate}
        onToggleNavCollapsed={vi.fn()}
        sessionRail={{
          route: "chat",
          countLabel: "2 sessions",
          onPrimaryAction: vi.fn(),
          body: <div role="list"><div role="listitem">Design Review</div></div>,
        }}
      />,
    );

    const settings = screen.getByRole("button", { name: "Settings" });

    expect(settings).toBeInTheDocument();
    expect(settings).toHaveClass("active");
    expect(settings).toHaveClass("nav-settings-shortcut");
    expect(screen.queryByRole("button", { name: "Language" })).not.toBeInTheDocument();
    expect(container.querySelector("[data-nav-session-rail='chat']")).toBeInTheDocument();
    expect(screen.getByText("Design Review")).toBeInTheDocument();

    fireEvent.click(settings);

    expect(onNavigate).toHaveBeenCalledWith("settings");
  });

  it("renders the active runtime conversation list directly in the left navigation", () => {
    const onCreate = vi.fn();

    const { container } = render(
      <PrimaryNav
        currentRoute="chat"
        language="en"
        navCollapsed={false}
        onNavigate={vi.fn()}
        onToggleNavCollapsed={vi.fn()}
        sessionRail={{
          route: "chat",
          countLabel: "2 sessions",
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
    expect(rail.closest(".primary-nav")).toHaveAttribute("data-shell-design", "light-tech");
    expect(rail).toHaveTextContent("Sessions");
    expect(rail).toHaveTextContent("2 sessions");
    expect(screen.getByText("Design Review")).toBeInTheDocument();
    expect(screen.getByText("Launch Plan")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "New" }));

    expect(onCreate).toHaveBeenCalledTimes(1);
  });

  it("keeps the runtime session rail chrome owned by the primary navigation", () => {
    const onCreate = vi.fn();

    render(
      <PrimaryNav
        currentRoute="chat"
        language="zh"
        navCollapsed={false}
        onNavigate={vi.fn()}
        onToggleNavCollapsed={vi.fn()}
        sessionRail={{
          route: "chat",
          countLabel: "2 个会话",
          onPrimaryAction: onCreate,
          body: <div role="list"><div role="listitem">New</div></div>,
        }}
      />,
    );

    expect(screen.getByText("会话列表")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "新建" }));

    expect(onCreate).toHaveBeenCalledTimes(1);
  });

  it("hides the runtime conversation list when the sidebar is collapsed", () => {
    const { container } = render(
      <PrimaryNav
        currentRoute="chat"
        language="en"
        navCollapsed={true}
        onNavigate={vi.fn()}
        onToggleNavCollapsed={vi.fn()}
        sessionRail={{
          route: "chat",
          countLabel: "2 sessions",
          onPrimaryAction: vi.fn(),
          body: <div>Design Review</div>,
        }}
      />,
    );

    expect(container.querySelector("[data-nav-session-rail='chat']")).not.toBeInTheDocument();
  });
});

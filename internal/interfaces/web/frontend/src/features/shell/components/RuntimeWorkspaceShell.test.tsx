import { fireEvent, render, screen, within, type RenderResult } from "@testing-library/react";
import type { ComponentPropsWithoutRef } from "react";
import { RuntimeWorkspaceShell } from "./RuntimeWorkspaceShell";
import type { RuntimeMobileLayoutState } from "./runtimeMobileLayout";

type RuntimeWorkspaceShellHarnessProps = ComponentPropsWithoutRef<typeof RuntimeWorkspaceShell> & {
  mobileLayoutState?: RuntimeMobileLayoutState;
};

const RuntimeWorkspaceShellHarness = RuntimeWorkspaceShell as unknown as (
  props: RuntimeWorkspaceShellHarnessProps
) => React.ReactElement;

describe("RuntimeWorkspaceShell", () => {
  function renderMobileShell(actions: {
    onMobileNav?: () => void;
    onMobileTitle?: () => void;
    onMobilePrimary?: () => void;
  }, props: Partial<RuntimeWorkspaceShellHarnessProps> = {}): RenderResult {
    return render(
      <RuntimeWorkspaceShellHarness
        sessionPaneBackdrop={{ ariaLabel: "Hide sessions", onClick: vi.fn() }}
        sessionPaneTitle="Sessions"
        sessionPaneCountLabel="1 session"
        sessionPanePrimaryActionLabel="New"
        onSessionPanePrimaryAction={vi.fn()}
        sessionPaneBody={<div>sessions</div>}
        workspaceContent={<div>content</div>}
        mobileHeaderPlacement="body"
        mobileNavButtonLabel="Menu"
        onMobileNav={actions.onMobileNav}
        mobileTitleButtonLabel="Current session"
        onMobileTitle={actions.onMobileTitle}
        mobilePrimaryButtonLabel="New"
        onMobilePrimary={actions.onMobilePrimary}
        {...props}
      />,
    );
  }

  it("runs mobile header actions from the first touchstart gesture", () => {
    const onMobileNav = vi.fn();
    const onMobileTitle = vi.fn();
    const onMobilePrimary = vi.fn();

    renderMobileShell({ onMobileNav, onMobileTitle, onMobilePrimary });

    const mobileHeader = screen.getByRole("banner");

    fireEvent.touchStart(within(mobileHeader).getByRole("button", { name: "Menu" }));
    fireEvent.touchStart(within(mobileHeader).getByRole("button", { name: "Current session" }));
    fireEvent.touchStart(within(mobileHeader).getByRole("button", { name: "New" }));

    expect(onMobileNav).toHaveBeenCalledTimes(1);
    expect(onMobileTitle).toHaveBeenCalledTimes(1);
    expect(onMobilePrimary).toHaveBeenCalledTimes(1);
  });

  it("keeps mobile edge actions as icon buttons with accessible labels", () => {
    renderMobileShell({});

    const mobileHeader = screen.getByRole("banner");
    const menuButton = within(mobileHeader).getByRole("button", { name: "Menu" });
    const newButton = within(mobileHeader).getByRole("button", { name: "New" });

    expect(menuButton.querySelector("[data-runtime-mobile-icon='menu']")).toBeInTheDocument();
    expect(menuButton.querySelector(".runtime-workspace-mobile-action-label")).toHaveClass("sr-only");
    expect(newButton.querySelector("[data-runtime-mobile-icon='plus']")).toBeInTheDocument();
    expect(newButton.querySelector(".runtime-workspace-mobile-action-label")).toHaveClass("sr-only");
  });

  it("does not rerun a touch-triggered mobile header action from the follow-up click", () => {
    vi.useFakeTimers();
    const onMobilePrimary = vi.fn();
    try {
      renderMobileShell({ onMobilePrimary });

      const newButton = within(screen.getByRole("banner")).getByRole("button", { name: "New" });
      fireEvent.touchStart(newButton);
      expect(onMobilePrimary).toHaveBeenCalledTimes(1);
      vi.advanceTimersByTime(300);
      fireEvent.click(newButton);

      expect(onMobilePrimary).toHaveBeenCalledTimes(1);
      vi.advanceTimersByTime(700);
      fireEvent.click(newButton);
      expect(onMobilePrimary).toHaveBeenCalledTimes(2);
    } finally {
      vi.useRealTimers();
    }
  });

  it("blurs the active composer input before running mobile header actions", () => {
    const input = document.createElement("textarea");
    document.body.appendChild(input);
    input.focus();
    const blur = vi.spyOn(input, "blur");
    const onMobileNav = vi.fn();

    try {
      renderMobileShell({ onMobileNav });

      fireEvent.click(within(screen.getByRole("banner")).getByRole("button", { name: "Menu" }));

      expect(blur).toHaveBeenCalledTimes(1);
      expect(onMobileNav).toHaveBeenCalledTimes(1);
    } finally {
      input.remove();
    }
  });

  it("publishes mobile drawer layout state and suspends the runtime composer layer", () => {
    renderMobileShell({}, {
      mobileLayoutState: "mobile-session-drawer",
      sessionPaneClassName: "is-open",
      workspaceFooter: <div data-testid="runtime-composer-spacer" />,
    });

    const shell = document.querySelector(".runtime-workspace-shell") as HTMLElement;
    const workspaceBody = document.querySelector(".runtime-workspace-body") as HTMLElement;
    const sessionPane = document.querySelector(".runtime-workspace-session-pane") as HTMLElement;

    expect(shell.dataset.runtimeMobileLayout).toBe("mobile-session-drawer");
    expect(shell.dataset.runtimeSessionPaneOpen).toBe("true");
    expect(shell.dataset.runtimeComposerSuspended).toBe("true");
    expect(workspaceBody.dataset.runtimeMobileLayout).toBe("mobile-session-drawer");
    expect(workspaceBody.dataset.runtimeComposerSuspended).toBe("true");
    expect(sessionPane).toHaveClass("is-open");
  });

  it("blurs the active composer input when the mobile session drawer opens", () => {
    const input = document.createElement("textarea");
    document.body.appendChild(input);
    input.focus();
    const blur = vi.spyOn(input, "blur");

    try {
      renderMobileShell({}, {
        mobileLayoutState: "mobile-session-drawer",
        sessionPaneClassName: "is-open",
      });

      expect(blur).toHaveBeenCalledTimes(1);
    } finally {
      input.remove();
    }
  });

  it("suspends the composer layer when the primary navigation drawer owns mobile layout", () => {
    renderMobileShell({}, {
      mobileLayoutState: "mobile-primary-nav-drawer",
      workspaceFooter: <div data-testid="runtime-composer-spacer" />,
    });

    const shell = document.querySelector(".runtime-workspace-shell") as HTMLElement;
    const workspaceBody = document.querySelector(".runtime-workspace-body") as HTMLElement;

    expect(shell.dataset.runtimeMobileLayout).toBe("mobile-primary-nav-drawer");
    expect(shell.dataset.runtimeSessionPaneOpen).toBe("false");
    expect(shell.dataset.runtimeComposerSuspended).toBe("true");
    expect(workspaceBody.dataset.runtimeComposerSuspended).toBe("true");
  });
});

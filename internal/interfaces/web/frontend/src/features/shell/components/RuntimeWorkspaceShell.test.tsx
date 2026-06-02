import { fireEvent, render, screen, within } from "@testing-library/react";
import { RuntimeWorkspaceShell } from "./RuntimeWorkspaceShell";

describe("RuntimeWorkspaceShell", () => {
  function renderMobileShell(actions: {
    onMobileNav?: () => void;
    onMobileTitle?: () => void;
    onMobilePrimary?: () => void;
  }) {
    return render(
      <RuntimeWorkspaceShell
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
});

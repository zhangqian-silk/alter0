import { render, screen } from "@testing-library/react";
import { RuntimeMobileComposerPortal } from "./RuntimeMobileComposerPortal";

describe("RuntimeMobileComposerPortal", () => {
  it("renders the composer into a mobile portal host", () => {
    render(
      <RuntimeMobileComposerPortal
        isMobileViewport
        route="terminal"
        composerNode={<button type="button">Send</button>}
        mobileLayoutState="mobile-rest"
      />,
    );

    const portalHost = document.body.querySelector("[data-runtime-composer-portal-host='terminal']");
    expect(portalHost).toHaveAttribute("data-runtime-composer-view", "terminal");
    expect(portalHost).toHaveAttribute("data-runtime-composer-suspended", "false");
    expect(screen.getByRole("button", { name: "Send" })).toBeInTheDocument();
  });

  it("does not render outside mobile viewports", () => {
    render(
      <RuntimeMobileComposerPortal
        isMobileViewport={false}
        route="terminal"
        composerNode={<button type="button">Send</button>}
      />,
    );

    expect(document.body.querySelector("[data-runtime-composer-portal-host]")).not.toBeInTheDocument();
  });

  it("marks the composer portal suspended while a mobile drawer owns layout", () => {
    render(
      <RuntimeMobileComposerPortal
        isMobileViewport
        route="chat"
        composerNode={<button type="button">Send</button>}
        mobileLayoutState="mobile-primary-nav-drawer"
      />,
    );

    const portalHost = document.body.querySelector("[data-runtime-composer-portal-host='chat']");
    expect(portalHost).toHaveAttribute("data-runtime-composer-view", "conversation");
    expect(portalHost).toHaveAttribute("data-runtime-composer-suspended", "true");
    expect(portalHost).toHaveAttribute("aria-hidden", "true");
  });
});

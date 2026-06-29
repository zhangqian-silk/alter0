import {
  resolveRuntimeMobileLayoutState,
  runtimeMobileLayoutAllowsComposerInteraction,
} from "./runtimeMobileLayout";

describe("resolveRuntimeMobileLayoutState", () => {
  it("keeps desktop layouts outside the mobile viewport", () => {
    expect(resolveRuntimeMobileLayoutState({
      isMobileViewport: false,
      inputFocused: true,
      primaryNavOpen: true,
      sessionPaneOpen: true,
      composerPanelOpen: true,
    })).toBe("desktop");
  });

  it("prioritizes the primary navigation drawer over every runtime overlay", () => {
    expect(resolveRuntimeMobileLayoutState({
      isMobileViewport: true,
      inputFocused: true,
      primaryNavOpen: true,
      sessionPaneOpen: true,
      composerPanelOpen: true,
    })).toBe("mobile-primary-nav-drawer");
  });

  it("prioritizes the session drawer over keyboard and composer panels", () => {
    expect(resolveRuntimeMobileLayoutState({
      isMobileViewport: true,
      inputFocused: true,
      primaryNavOpen: false,
      sessionPaneOpen: true,
      composerPanelOpen: true,
    })).toBe("mobile-session-drawer");
  });

  it("keeps composer panels above the focused keyboard state", () => {
    expect(resolveRuntimeMobileLayoutState({
      isMobileViewport: true,
      inputFocused: true,
      primaryNavOpen: false,
      sessionPaneOpen: false,
      composerPanelOpen: true,
    })).toBe("mobile-composer-panel");
  });

  it("reports the keyboard and rest states for ordinary mobile composer focus", () => {
    expect(resolveRuntimeMobileLayoutState({
      isMobileViewport: true,
      inputFocused: true,
      primaryNavOpen: false,
      sessionPaneOpen: false,
      composerPanelOpen: false,
    })).toBe("mobile-keyboard");
    expect(resolveRuntimeMobileLayoutState({
      isMobileViewport: true,
      inputFocused: false,
      primaryNavOpen: false,
      sessionPaneOpen: false,
      composerPanelOpen: false,
    })).toBe("mobile-rest");
  });

  it("keeps drawer-owned composers visible but non-interactive", () => {
    expect(runtimeMobileLayoutAllowsComposerInteraction("mobile-rest")).toBe(true);
    expect(runtimeMobileLayoutAllowsComposerInteraction("mobile-keyboard")).toBe(true);
    expect(runtimeMobileLayoutAllowsComposerInteraction("mobile-composer-panel")).toBe(true);
    expect(runtimeMobileLayoutAllowsComposerInteraction("mobile-primary-nav-drawer")).toBe(false);
    expect(runtimeMobileLayoutAllowsComposerInteraction("mobile-session-drawer")).toBe(false);
  });
});

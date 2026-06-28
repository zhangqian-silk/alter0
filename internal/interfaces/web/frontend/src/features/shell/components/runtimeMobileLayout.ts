export type RuntimeMobileLayoutState =
  | "desktop"
  | "mobile-rest"
  | "mobile-keyboard"
  | "mobile-primary-nav-drawer"
  | "mobile-session-drawer"
  | "mobile-composer-panel";

export type RuntimeMobileLayoutInput = {
  isMobileViewport: boolean;
  inputFocused: boolean;
  primaryNavOpen: boolean;
  sessionPaneOpen: boolean;
  composerPanelOpen: boolean;
};

export function resolveRuntimeMobileLayoutState({
  isMobileViewport,
  inputFocused,
  primaryNavOpen,
  sessionPaneOpen,
  composerPanelOpen,
}: RuntimeMobileLayoutInput): RuntimeMobileLayoutState {
  if (!isMobileViewport) {
    return "desktop";
  }
  if (primaryNavOpen) {
    return "mobile-primary-nav-drawer";
  }
  if (sessionPaneOpen) {
    return "mobile-session-drawer";
  }
  if (composerPanelOpen) {
    return "mobile-composer-panel";
  }
  return inputFocused ? "mobile-keyboard" : "mobile-rest";
}

export function runtimeMobileLayoutSuspendsComposer(state: RuntimeMobileLayoutState | undefined): boolean {
  switch (state) {
    case "mobile-primary-nav-drawer":
    case "mobile-session-drawer":
      return true;
    default:
      return false;
  }
}

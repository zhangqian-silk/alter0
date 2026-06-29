import {
  MOBILE_KEYBOARD_MIN_OFFSET_PX,
  MOBILE_VIEWPORT_ALIGN_COOLDOWN_MS,
  createDefaultMobileViewportState,
  deriveMobileViewportState,
  isMobileViewportWidth,
} from "./mobileViewport";

type MobileViewportSyncOptions = {
  doc?: Document;
  root?: HTMLElement;
  win?: Window;
  hasActiveInput?: () => boolean;
};

export type MobileViewportSyncController = {
  sync: () => void;
  destroy: () => void;
};

function defaultHasActiveInput(doc: Document): boolean {
  const active = doc.activeElement;
  return active instanceof HTMLInputElement || active instanceof HTMLTextAreaElement;
}

export function createMobileViewportSyncController(
  options: MobileViewportSyncOptions = {},
): MobileViewportSyncController {
  const doc = options.doc ?? document;
  const root = options.root ?? doc.documentElement;
  const win = options.win ?? window;
  const hasActiveInput = options.hasActiveInput ?? (() => defaultHasActiveInput(doc));
  const visualViewport = win.visualViewport;
  let state = createDefaultMobileViewportState();
  let cooldownSyncTimeoutID = 0;

  const clearCooldownSync = () => {
    if (!cooldownSyncTimeoutID) {
      return;
    }
    win.clearTimeout(cooldownSyncTimeoutID);
    cooldownSyncTimeoutID = 0;
  };
  const scheduleCooldownSync = (delayMS: number) => {
    cooldownSyncTimeoutID = win.setTimeout(() => {
      cooldownSyncTimeoutID = 0;
      syncWhenVisible();
    }, delayMS);
  };

  const sync = () => {
    const activeInput = hasActiveInput();
    const result = deriveMobileViewportState(state, {
      mobileViewport: isMobileViewportWidth(win.innerWidth),
      windowWidth: win.innerWidth,
      windowHeight: win.innerHeight,
      viewportWidth: visualViewport?.width,
      viewportHeight: visualViewport?.height,
      viewportOffsetTop: visualViewport?.offsetTop,
      hasActiveInput: activeInput,
    });
    state = result.state;
    root.style.setProperty("--mobile-viewport-height", result.cssVars.mobileViewportHeight);
    root.style.setProperty("--mobile-viewport-offset-top", result.cssVars.mobileViewportOffsetTop);
    root.style.setProperty("--keyboard-offset", result.cssVars.keyboardOffset);
    clearCooldownSync();
    const reportedViewportHeight = Math.round(visualViewport?.height ?? win.innerHeight);
    const focusedFullHeightReport =
      activeInput
      && result.state.keyboardOffset >= MOBILE_KEYBOARD_MIN_OFFSET_PX
      && result.state.baselineHeight > 0
      && reportedViewportHeight >= result.state.baselineHeight - 2;
    if (focusedFullHeightReport) {
      scheduleCooldownSync(MOBILE_VIEWPORT_ALIGN_COOLDOWN_MS + 16);
    }
  };
  const syncWhenVisible = () => {
    if (doc.visibilityState === "hidden") {
      return;
    }
    sync();
  };

  sync();
  win.addEventListener("resize", sync);
  win.addEventListener("focus", syncWhenVisible);
  win.addEventListener("pageshow", syncWhenVisible);
  visualViewport?.addEventListener("resize", sync);
  visualViewport?.addEventListener("scroll", sync);
  doc.addEventListener("visibilitychange", syncWhenVisible);
  doc.addEventListener("focusin", sync);
  doc.addEventListener("focusout", sync);

  return {
    sync,
    destroy: () => {
      clearCooldownSync();
      win.removeEventListener("resize", sync);
      win.removeEventListener("focus", syncWhenVisible);
      win.removeEventListener("pageshow", syncWhenVisible);
      visualViewport?.removeEventListener("resize", sync);
      visualViewport?.removeEventListener("scroll", sync);
      doc.removeEventListener("visibilitychange", syncWhenVisible);
      doc.removeEventListener("focusin", sync);
      doc.removeEventListener("focusout", sync);
      root.style.setProperty("--mobile-viewport-height", "100dvh");
      root.style.setProperty("--mobile-viewport-offset-top", "0px");
      root.style.setProperty("--keyboard-offset", "0px");
    },
  };
}

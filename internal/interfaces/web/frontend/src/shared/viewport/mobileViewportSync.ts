import {
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

  const sync = () => {
    const result = deriveMobileViewportState(state, {
      mobileViewport: isMobileViewportWidth(win.innerWidth),
      windowWidth: win.innerWidth,
      windowHeight: win.innerHeight,
      viewportWidth: visualViewport?.width,
      viewportHeight: visualViewport?.height,
      viewportOffsetTop: visualViewport?.offsetTop,
      hasActiveInput: hasActiveInput(),
    });
    state = result.state;
    root.style.setProperty("--mobile-viewport-height", result.cssVars.mobileViewportHeight);
    root.style.setProperty("--keyboard-offset", result.cssVars.keyboardOffset);
  };

  sync();
  win.addEventListener("resize", sync);
  visualViewport?.addEventListener("resize", sync);
  visualViewport?.addEventListener("scroll", sync);
  doc.addEventListener("focusin", sync);
  doc.addEventListener("focusout", sync);

  return {
    sync,
    destroy: () => {
      win.removeEventListener("resize", sync);
      visualViewport?.removeEventListener("resize", sync);
      visualViewport?.removeEventListener("scroll", sync);
      doc.removeEventListener("focusin", sync);
      doc.removeEventListener("focusout", sync);
      root.style.setProperty("--mobile-viewport-height", "100dvh");
      root.style.setProperty("--keyboard-offset", "0px");
    },
  };
}

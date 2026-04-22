export const MOBILE_VIEWPORT_BREAKPOINT_PX = 1100;
export const TERMINAL_SESSION_SHEET_BREAKPOINT_PX = 760;
export const MOBILE_VIEWPORT_SYNC_THRESHOLD_PX = 8;
export const MOBILE_KEYBOARD_MIN_OFFSET_PX = 120;
export const MOBILE_VIEWPORT_ALIGN_COOLDOWN_MS = 240;
export const MOBILE_VIEWPORT_WIDTH_RESET_DELTA_PX = 48;

export type MobileViewportState = {
  baselineHeight: number;
  width: number;
  height: number;
  keyboardOffset: number;
  lastAlignedAt?: number;
};

export type DeriveMobileViewportInput = {
  mobileViewport: boolean;
  windowWidth: number;
  windowHeight: number;
  viewportWidth?: number;
  viewportHeight?: number;
  viewportOffsetTop?: number;
  hasActiveInput: boolean;
};

export type DeriveMobileViewportResult = {
  state: MobileViewportState;
  cssVars: {
    mobileViewportHeight: string;
    keyboardOffset: string;
  };
  changed: {
    width: boolean;
    height: boolean;
    keyboardOffset: boolean;
  };
};

export function createDefaultMobileViewportState(): MobileViewportState {
  return {
    baselineHeight: 0,
    width: 0,
    height: 0,
    keyboardOffset: 0,
    lastAlignedAt: 0
  };
}

export function isMobileViewportWidth(width: number): boolean {
  return Number(width) <= MOBILE_VIEWPORT_BREAKPOINT_PX;
}

export function isTerminalSessionSheetViewportWidth(width: number): boolean {
  return Number(width) <= TERMINAL_SESSION_SHEET_BREAKPOINT_PX;
}

export function deriveMobileViewportState(
  previous: MobileViewportState,
  input: DeriveMobileViewportInput,
): DeriveMobileViewportResult {
  const previousState = previous ?? createDefaultMobileViewportState();
  if (!input.mobileViewport) {
    return {
      state: createDefaultMobileViewportState(),
      cssVars: {
        mobileViewportHeight: "100dvh",
        keyboardOffset: "0px"
      },
      changed: {
        width: previousState.width !== 0,
        height: previousState.height !== 0,
        keyboardOffset: previousState.keyboardOffset !== 0
      }
    };
  }

  const effectiveHeight = Math.max(
    0,
    Math.round((input.viewportHeight ?? input.windowHeight) + Math.max(input.viewportOffsetTop ?? 0, 0)),
  );
  const viewportWidth = Math.max(
    0,
    Math.round(input.viewportWidth ?? input.windowWidth),
  );
  const widthChanged = Math.abs(viewportWidth - previousState.width) > MOBILE_VIEWPORT_WIDTH_RESET_DELTA_PX;
  const keyboardClosing =
    !input.hasActiveInput
    && previousState.keyboardOffset >= MOBILE_VIEWPORT_SYNC_THRESHOLD_PX
    && previousState.baselineHeight > 0
    && effectiveHeight < previousState.baselineHeight - 2;

  let baselineHeight = previousState.baselineHeight;
  if (!baselineHeight || widthChanged) {
    baselineHeight = effectiveHeight;
  }
  if (keyboardClosing) {
    baselineHeight = Math.max(previousState.baselineHeight, effectiveHeight);
  } else if (!input.hasActiveInput || effectiveHeight >= baselineHeight - 2) {
    baselineHeight = effectiveHeight;
  } else {
    baselineHeight = Math.max(baselineHeight, effectiveHeight);
  }

  const rawKeyboardOffset = input.hasActiveInput || keyboardClosing
    ? Math.max(0, baselineHeight - effectiveHeight)
    : 0;
  const keyboardOffset = rawKeyboardOffset >= MOBILE_KEYBOARD_MIN_OFFSET_PX
    ? rawKeyboardOffset
    : 0;
  const heightChanged = Math.abs(effectiveHeight - previousState.height) >= MOBILE_VIEWPORT_SYNC_THRESHOLD_PX;
  const offsetChanged = Math.abs(keyboardOffset - previousState.keyboardOffset) >= MOBILE_VIEWPORT_SYNC_THRESHOLD_PX;

  return {
    state: {
      ...previousState,
      baselineHeight,
      width: viewportWidth,
      height: effectiveHeight,
      keyboardOffset,
    },
    cssVars: {
      mobileViewportHeight: `${effectiveHeight}px`,
      keyboardOffset: `${keyboardOffset}px`
    },
    changed: {
      width: widthChanged,
      height: heightChanged || previousState.height === 0,
      keyboardOffset: offsetChanged || previousState.height === 0
    }
  };
}

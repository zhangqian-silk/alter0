export const MOBILE_VIEWPORT_BREAKPOINT_PX = 1100;
export const TERMINAL_SESSION_SHEET_BREAKPOINT_PX = 760;
export const MOBILE_VIEWPORT_SYNC_THRESHOLD_PX = 8;
export const MOBILE_KEYBOARD_MIN_OFFSET_PX = 120;
export const MOBILE_VIEWPORT_ALIGN_COOLDOWN_MS = 240;
export const MOBILE_VIEWPORT_COMPOSER_OFFSET_HOLD_MS = 360;
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
  currentTimeMS?: number;
};

export type DeriveMobileViewportResult = {
  state: MobileViewportState;
  cssVars: {
    mobileViewportHeight: string;
    keyboardOffset: string;
    keyboardComposerOffset: string;
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

function resolveKeyboardComposerOffset(
  keyboardOffset: number,
  viewportOffsetTop: number,
  holdViewportOffsetTop: boolean,
): number {
  return Math.max(0, keyboardOffset - (holdViewportOffsetTop ? 0 : viewportOffsetTop));
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
        keyboardOffset: "0px",
        keyboardComposerOffset: "0px"
      },
      changed: {
        width: previousState.width !== 0,
        height: previousState.height !== 0,
        keyboardOffset: previousState.keyboardOffset !== 0
      }
    };
  }

  const reportedViewportHeight = Math.max(
    0,
    Math.round(input.viewportHeight ?? input.windowHeight),
  );
  const viewportOffsetTop = Math.max(input.viewportOffsetTop ?? 0, 0);
  const viewportBottomHeight = Math.max(
    0,
    Math.round(reportedViewportHeight + viewportOffsetTop),
  );
  const viewportWidth = Math.max(
    0,
    Math.round(input.viewportWidth ?? input.windowWidth),
  );
  const widthChanged = Math.abs(viewportWidth - previousState.width) > MOBILE_VIEWPORT_WIDTH_RESET_DELTA_PX;
  const currentTimeMS = Number.isFinite(input.currentTimeMS) ? Number(input.currentTimeMS) : Date.now();
  const previousKeyboardActive =
    previousState.keyboardOffset >= MOBILE_KEYBOARD_MIN_OFFSET_PX
    && previousState.baselineHeight > 0;
  const reportedViewportKeyboardOffset = previousState.baselineHeight > 0
    ? Math.max(0, previousState.baselineHeight - reportedViewportHeight)
    : 0;
  const focusedViewportIsShrinking =
    input.hasActiveInput
    && !widthChanged
    && previousState.baselineHeight > 0
    && reportedViewportKeyboardOffset >= MOBILE_VIEWPORT_SYNC_THRESHOLD_PX;
  const viewportReportsKeyboard =
    focusedViewportIsShrinking
    || (
      reportedViewportKeyboardOffset >= MOBILE_KEYBOARD_MIN_OFFSET_PX
      && (input.hasActiveInput || previousKeyboardActive)
    );
  const effectiveHeight = viewportReportsKeyboard
    ? reportedViewportHeight
    : viewportBottomHeight;
  const keyboardRecentlyAligned =
    previousState.keyboardOffset >= MOBILE_KEYBOARD_MIN_OFFSET_PX
    && (currentTimeMS - (previousState.lastAlignedAt || 0)) < MOBILE_VIEWPORT_ALIGN_COOLDOWN_MS;
  const shouldHoldComposerOffset =
    input.hasActiveInput
    && !widthChanged
    && viewportOffsetTop > 0
    && previousState.keyboardOffset >= MOBILE_KEYBOARD_MIN_OFFSET_PX
    && (currentTimeMS - (previousState.lastAlignedAt || 0)) < MOBILE_VIEWPORT_COMPOSER_OFFSET_HOLD_MS;
  if (
    input.hasActiveInput
    && !widthChanged
    && keyboardRecentlyAligned
    && previousState.baselineHeight > 0
    && previousState.height > 0
    && effectiveHeight >= previousState.baselineHeight - 2
  ) {
    return {
      state: {
        ...previousState,
        width: viewportWidth
      },
      cssVars: {
        mobileViewportHeight: `${previousState.height}px`,
        keyboardOffset: `${previousState.keyboardOffset}px`,
        keyboardComposerOffset: `${resolveKeyboardComposerOffset(previousState.keyboardOffset, viewportOffsetTop, shouldHoldComposerOffset)}px`
      },
      changed: {
        width: false,
        height: false,
        keyboardOffset: false
      }
    };
  }
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
  const keyboardOffsetThreshold = input.hasActiveInput || keyboardClosing
    ? MOBILE_VIEWPORT_SYNC_THRESHOLD_PX
    : MOBILE_KEYBOARD_MIN_OFFSET_PX;
  const keyboardOffset = rawKeyboardOffset >= keyboardOffsetThreshold
    ? rawKeyboardOffset
    : 0;
  const holdComposerOffset =
    shouldHoldComposerOffset
    && keyboardOffset >= MOBILE_KEYBOARD_MIN_OFFSET_PX;
  const heightChanged = Math.abs(effectiveHeight - previousState.height) >= MOBILE_VIEWPORT_SYNC_THRESHOLD_PX;
  const offsetChanged = Math.abs(keyboardOffset - previousState.keyboardOffset) >= MOBILE_VIEWPORT_SYNC_THRESHOLD_PX;

  return {
    state: {
      ...previousState,
      baselineHeight,
      width: viewportWidth,
      height: effectiveHeight,
      keyboardOffset,
      lastAlignedAt: keyboardOffset >= MOBILE_KEYBOARD_MIN_OFFSET_PX
        ? currentTimeMS
        : previousState.lastAlignedAt
    },
    cssVars: {
      mobileViewportHeight: `${effectiveHeight}px`,
      keyboardOffset: `${keyboardOffset}px`,
      keyboardComposerOffset: `${resolveKeyboardComposerOffset(keyboardOffset, viewportOffsetTop, holdComposerOffset)}px`
    },
    changed: {
      width: widthChanged,
      height: heightChanged || previousState.height === 0,
      keyboardOffset: offsetChanged || previousState.height === 0
    }
  };
}

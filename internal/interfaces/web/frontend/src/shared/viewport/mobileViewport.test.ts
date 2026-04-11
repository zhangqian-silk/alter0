import {
  MOBILE_KEYBOARD_MIN_OFFSET_PX,
  MOBILE_VIEWPORT_ALIGN_COOLDOWN_MS,
  MOBILE_VIEWPORT_BREAKPOINT_PX,
  MOBILE_VIEWPORT_SYNC_THRESHOLD_PX,
  TERMINAL_SESSION_SHEET_BREAKPOINT_PX,
  createDefaultMobileViewportState,
  deriveMobileViewportState,
  isMobileViewportWidth,
  isTerminalSessionSheetViewportWidth
} from "./mobileViewport";

describe("shared viewport mobileViewport", () => {
  it("keeps viewport breakpoints aligned with the legacy shell", () => {
    expect(MOBILE_VIEWPORT_BREAKPOINT_PX).toBe(1100);
    expect(TERMINAL_SESSION_SHEET_BREAKPOINT_PX).toBe(760);
    expect(MOBILE_VIEWPORT_SYNC_THRESHOLD_PX).toBe(8);
    expect(MOBILE_KEYBOARD_MIN_OFFSET_PX).toBe(120);
    expect(MOBILE_VIEWPORT_ALIGN_COOLDOWN_MS).toBe(240);
  });

  it("matches the legacy mobile breakpoint helpers", () => {
    expect(isMobileViewportWidth(1100)).toBe(true);
    expect(isMobileViewportWidth(1101)).toBe(false);
    expect(isTerminalSessionSheetViewportWidth(760)).toBe(true);
    expect(isTerminalSessionSheetViewportWidth(761)).toBe(false);
  });

  it("resets viewport state and css values outside mobile layouts", () => {
    const result = deriveMobileViewportState(createDefaultMobileViewportState(), {
      mobileViewport: false,
      windowWidth: 1440,
      windowHeight: 900,
      hasActiveInput: false
    });

    expect(result.state).toEqual(createDefaultMobileViewportState());
    expect(result.cssVars).toEqual({
      mobileViewportHeight: "100dvh",
      keyboardOffset: "0px"
    });
  });

  it("computes keyboard offset only when the focused input exceeds the threshold", () => {
    const previous = {
      ...createDefaultMobileViewportState(),
      baselineHeight: 800,
      width: 390,
      height: 800,
      keyboardOffset: 0
    };

    const next = deriveMobileViewportState(previous, {
      mobileViewport: true,
      windowWidth: 390,
      windowHeight: 800,
      viewportWidth: 390,
      viewportHeight: 650,
      viewportOffsetTop: 0,
      hasActiveInput: true
    });

    expect(next.state.baselineHeight).toBe(800);
    expect(next.state.height).toBe(650);
    expect(next.state.keyboardOffset).toBe(150);
    expect(next.cssVars).toEqual({
      mobileViewportHeight: "650px",
      keyboardOffset: "150px"
    });
  });

  it("drops keyboard offset below the minimum threshold", () => {
    const previous = {
      ...createDefaultMobileViewportState(),
      baselineHeight: 800,
      width: 390,
      height: 800,
      keyboardOffset: 0
    };

    const next = deriveMobileViewportState(previous, {
      mobileViewport: true,
      windowWidth: 390,
      windowHeight: 800,
      viewportWidth: 390,
      viewportHeight: 720,
      viewportOffsetTop: 0,
      hasActiveInput: true
    });

    expect(next.state.keyboardOffset).toBe(0);
    expect(next.cssVars.keyboardOffset).toBe("0px");
  });

  it("resets the baseline when viewport width changes substantially", () => {
    const previous = {
      ...createDefaultMobileViewportState(),
      baselineHeight: 800,
      width: 390,
      height: 800,
      keyboardOffset: 0
    };

    const next = deriveMobileViewportState(previous, {
      mobileViewport: true,
      windowWidth: 520,
      windowHeight: 760,
      viewportWidth: 520,
      viewportHeight: 760,
      viewportOffsetTop: 0,
      hasActiveInput: false
    });

    expect(next.state.width).toBe(520);
    expect(next.state.baselineHeight).toBe(760);
    expect(next.state.keyboardOffset).toBe(0);
  });
});

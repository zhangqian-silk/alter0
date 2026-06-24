import {
  MOBILE_KEYBOARD_MIN_OFFSET_PX,
  MOBILE_VIEWPORT_COMPOSER_OFFSET_HOLD_MS,
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
    expect(MOBILE_VIEWPORT_COMPOSER_OFFSET_HOLD_MS).toBe(360);
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
      keyboardOffset: "0px",
      keyboardComposerOffset: "0px"
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
      keyboardOffset: "150px",
      keyboardComposerOffset: "150px"
    });
  });

  it("keeps the focused keyboard offset below the full keyboard threshold to avoid shell height jumps", () => {
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

    expect(next.state.keyboardOffset).toBe(80);
    expect(next.cssVars).toEqual({
      mobileViewportHeight: "720px",
      keyboardOffset: "80px",
      keyboardComposerOffset: "80px"
    });
  });

  it("keeps the active keyboard inset through transient full-height visual viewport reports", () => {
    const previous = {
      ...createDefaultMobileViewportState(),
      baselineHeight: 932,
      width: 430,
      height: 620,
      keyboardOffset: 312,
      lastAlignedAt: 1000
    };

    const transient = deriveMobileViewportState(previous, {
      mobileViewport: true,
      windowWidth: 430,
      windowHeight: 932,
      viewportWidth: 430,
      viewportHeight: 932,
      viewportOffsetTop: 0,
      hasActiveInput: true,
      currentTimeMS: 1120
    });

    expect(transient.state.height).toBe(620);
    expect(transient.state.keyboardOffset).toBe(312);
    expect(transient.cssVars).toEqual({
      mobileViewportHeight: "620px",
      keyboardOffset: "312px",
      keyboardComposerOffset: "312px"
    });

    const settled = deriveMobileViewportState(previous, {
      mobileViewport: true,
      windowWidth: 430,
      windowHeight: 932,
      viewportWidth: 430,
      viewportHeight: 932,
      viewportOffsetTop: 0,
      hasActiveInput: true,
      currentTimeMS: 1260
    });

    expect(settled.state.height).toBe(932);
    expect(settled.state.keyboardOffset).toBe(0);
    expect(settled.cssVars.keyboardOffset).toBe("0px");
  });

  it("keeps the active keyboard inset when a focused viewport reports a shifted visual offset", () => {
    const previous = {
      ...createDefaultMobileViewportState(),
      baselineHeight: 932,
      width: 430,
      height: 620,
      keyboardOffset: 312,
      lastAlignedAt: 1000
    };

    const shifted = deriveMobileViewportState(previous, {
      mobileViewport: true,
      windowWidth: 430,
      windowHeight: 932,
      viewportWidth: 430,
      viewportHeight: 620,
      viewportOffsetTop: 312,
      hasActiveInput: true,
      currentTimeMS: 1600
    });

    expect(shifted.state.height).toBe(620);
    expect(shifted.state.keyboardOffset).toBe(312);
    expect(shifted.cssVars).toEqual({
      mobileViewportHeight: "620px",
      keyboardOffset: "312px",
      keyboardComposerOffset: "0px"
    });
  });

  it("holds the composer offset through transient visual offset reports during keyboard opening", () => {
    const previous = {
      ...createDefaultMobileViewportState(),
      baselineHeight: 932,
      width: 430,
      height: 620,
      keyboardOffset: 312,
      lastAlignedAt: 1000
    };

    const transitioning = deriveMobileViewportState(previous, {
      mobileViewport: true,
      windowWidth: 430,
      windowHeight: 932,
      viewportWidth: 430,
      viewportHeight: 620,
      viewportOffsetTop: 312,
      hasActiveInput: true,
      currentTimeMS: 1300
    });

    expect(transitioning.state.height).toBe(620);
    expect(transitioning.state.keyboardOffset).toBe(312);
    expect(transitioning.cssVars).toEqual({
      mobileViewportHeight: "620px",
      keyboardOffset: "312px",
      keyboardComposerOffset: "312px"
    });
  });

  it("keeps the keyboard inset after blur while a shifted visual viewport is still recovering", () => {
    const previous = {
      ...createDefaultMobileViewportState(),
      baselineHeight: 932,
      width: 430,
      height: 620,
      keyboardOffset: 312,
      lastAlignedAt: 1000
    };

    const recovering = deriveMobileViewportState(previous, {
      mobileViewport: true,
      windowWidth: 430,
      windowHeight: 932,
      viewportWidth: 430,
      viewportHeight: 620,
      viewportOffsetTop: 312,
      hasActiveInput: false,
      currentTimeMS: 1600
    });

    expect(recovering.state.height).toBe(620);
    expect(recovering.state.keyboardOffset).toBe(312);
    expect(recovering.cssVars).toEqual({
      mobileViewportHeight: "620px",
      keyboardOffset: "312px",
      keyboardComposerOffset: "0px"
    });

    const recovered = deriveMobileViewportState(recovering.state, {
      mobileViewport: true,
      windowWidth: 430,
      windowHeight: 932,
      viewportWidth: 430,
      viewportHeight: 932,
      viewportOffsetTop: 0,
      hasActiveInput: false,
      currentTimeMS: 1800
    });

    expect(recovered.state.keyboardOffset).toBe(0);
    expect(recovered.cssVars).toEqual({
      mobileViewportHeight: "932px",
      keyboardOffset: "0px",
      keyboardComposerOffset: "0px"
    });
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

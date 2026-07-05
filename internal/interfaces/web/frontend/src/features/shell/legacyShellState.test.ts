import {
  LEGACY_SHELL_MOBILE_BREAKPOINT_PX,
  isLegacyShellMobileViewport,
  parseLegacyShellPathRoute,
} from "./legacyShellState";

describe("legacyShellState", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("keeps chat as the fallback route for root and unknown paths", () => {
    expect(parseLegacyShellPathRoute("/unknown")).toBe("chat");
    expect(parseLegacyShellPathRoute("/")).toBe("chat");
    expect(parseLegacyShellPathRoute("/tasks")).toBe("chat");
  });

  it("keeps regular desktop widths on the desktop sidebar before switching to the shared top bar shell", () => {
    const matchMedia = vi.fn((query: string) => ({
      matches: query === `(max-width: ${LEGACY_SHELL_MOBILE_BREAKPOINT_PX}px)`,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }));

    vi.stubGlobal("window", {
      ...window,
      matchMedia,
    });

    expect(isLegacyShellMobileViewport()).toBe(true);
    expect(LEGACY_SHELL_MOBILE_BREAKPOINT_PX).toBe(1280);
    expect(matchMedia).toHaveBeenCalledWith("(max-width: 1280px)");
  });
});

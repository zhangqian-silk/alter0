import {
  isLegacyShellMobileViewport,
  parseLegacyShellHashRoute,
} from "./legacyShellState";

describe("legacyShellState", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("keeps chat as the fallback route for unknown hash fragments", () => {
    expect(parseLegacyShellHashRoute("#unknown")).toBe("chat");
    expect(parseLegacyShellHashRoute("")).toBe("chat");
  });

  it("treats only 960px-and-below viewports as mobile shell mode", () => {
    const matchMedia = vi.fn((query: string) => ({
      matches: query === "(max-width: 960px)",
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
    expect(matchMedia).toHaveBeenCalledWith("(max-width: 960px)");
  });
});

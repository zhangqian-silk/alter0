import { createMobileViewportSyncController } from "./mobileViewportSync";
import {
  MOBILE_VIEWPORT_ALIGN_COOLDOWN_MS,
} from "./mobileViewport";

class MockVisualViewport extends EventTarget {
  width: number;
  height: number;
  offsetTop: number;

  constructor(width: number, height: number) {
    super();
    this.width = width;
    this.height = height;
    this.offsetTop = 0;
  }
}

function setWindowSize(width: number, height: number) {
  Object.defineProperty(window, "innerWidth", {
    configurable: true,
    writable: true,
    value: width,
  });
  Object.defineProperty(window, "innerHeight", {
    configurable: true,
    writable: true,
    value: height,
  });
}

describe("shared viewport mobileViewportSync", () => {
  beforeEach(() => {
    document.documentElement.style.removeProperty("--mobile-viewport-height");
    document.documentElement.style.removeProperty("--mobile-viewport-offset-top");
    document.documentElement.style.removeProperty("--keyboard-offset");
  });

  it("writes keyboard offset css vars when the mobile visual viewport shrinks around a focused input", () => {
    const visualViewport = new MockVisualViewport(393, 852);
    const input = document.createElement("textarea");
    document.body.appendChild(input);
    setWindowSize(393, 852);
    Object.defineProperty(window, "visualViewport", {
      configurable: true,
      value: visualViewport,
    });

    const controller = createMobileViewportSyncController();

    input.focus();
    visualViewport.height = 520;
    visualViewport.dispatchEvent(new Event("resize"));

    expect(document.documentElement.style.getPropertyValue("--mobile-viewport-height")).toBe("520px");
    expect(document.documentElement.style.getPropertyValue("--mobile-viewport-offset-top")).toBe("0px");
    expect(document.documentElement.style.getPropertyValue("--keyboard-offset")).toBe("332px");

    controller.destroy();
    input.remove();
  });

  it("publishes visual viewport top shifts without moving root chrome", () => {
    const visualViewport = new MockVisualViewport(430, 932);
    const input = document.createElement("textarea");
    document.body.appendChild(input);
    setWindowSize(430, 932);
    Object.defineProperty(window, "visualViewport", {
      configurable: true,
      value: visualViewport,
    });

    const controller = createMobileViewportSyncController();

    input.focus();
    visualViewport.height = 620;
    visualViewport.dispatchEvent(new Event("resize"));
    expect(document.documentElement.style.getPropertyValue("--keyboard-offset")).toBe("312px");
    expect(document.documentElement.style.getPropertyValue("--mobile-viewport-offset-top")).toBe("0px");

    visualViewport.offsetTop = 312;
    visualViewport.dispatchEvent(new Event("scroll"));

    expect(document.documentElement.style.getPropertyValue("--mobile-viewport-height")).toBe("932px");
    expect(document.documentElement.style.getPropertyValue("--mobile-viewport-offset-top")).toBe("312px");
    expect(document.documentElement.style.getPropertyValue("--keyboard-offset")).toBe("0px");

    controller.destroy();
    input.remove();
  });

  it("clears keyboard offset when the focused input blurs or the layout leaves mobile mode", () => {
    const visualViewport = new MockVisualViewport(760, 980);
    const input = document.createElement("textarea");
    document.body.appendChild(input);
    setWindowSize(760, 980);
    Object.defineProperty(window, "visualViewport", {
      configurable: true,
      value: visualViewport,
    });

    const controller = createMobileViewportSyncController();

    input.focus();
    visualViewport.height = 620;
    visualViewport.dispatchEvent(new Event("resize"));
    expect(document.documentElement.style.getPropertyValue("--keyboard-offset")).toBe("360px");

    input.blur();
    expect(document.documentElement.style.getPropertyValue("--keyboard-offset")).toBe("360px");

    visualViewport.height = 980;
    visualViewport.dispatchEvent(new Event("resize"));
    expect(document.documentElement.style.getPropertyValue("--keyboard-offset")).toBe("0px");

    setWindowSize(1280, 900);
    window.dispatchEvent(new Event("resize"));
    expect(document.documentElement.style.getPropertyValue("--mobile-viewport-height")).toBe("100dvh");
    expect(document.documentElement.style.getPropertyValue("--mobile-viewport-offset-top")).toBe("0px");
    expect(document.documentElement.style.getPropertyValue("--keyboard-offset")).toBe("0px");

    controller.destroy();
    input.remove();
  });

  it("keeps the keyboard offset until the mobile viewport actually recovers after blur", () => {
    const visualViewport = new MockVisualViewport(760, 980);
    const input = document.createElement("textarea");
    document.body.appendChild(input);
    setWindowSize(760, 980);
    Object.defineProperty(window, "visualViewport", {
      configurable: true,
      value: visualViewport,
    });

    const controller = createMobileViewportSyncController();

    input.focus();
    visualViewport.height = 620;
    visualViewport.dispatchEvent(new Event("resize"));
    expect(document.documentElement.style.getPropertyValue("--keyboard-offset")).toBe("360px");

    input.blur();
    expect(document.documentElement.style.getPropertyValue("--mobile-viewport-height")).toBe("620px");
    expect(document.documentElement.style.getPropertyValue("--keyboard-offset")).toBe("360px");

    visualViewport.height = 760;
    visualViewport.dispatchEvent(new Event("resize"));
    expect(document.documentElement.style.getPropertyValue("--keyboard-offset")).toBe("220px");

    visualViewport.height = 980;
    visualViewport.dispatchEvent(new Event("resize"));
    expect(document.documentElement.style.getPropertyValue("--keyboard-offset")).toBe("0px");

    controller.destroy();
    input.remove();
  });

  it("rechecks focused full-height viewport reports after the keyboard alignment cooldown", () => {
    vi.useFakeTimers();
    const visualViewport = new MockVisualViewport(430, 932);
    const input = document.createElement("textarea");
    document.body.appendChild(input);
    setWindowSize(430, 932);
    Object.defineProperty(window, "visualViewport", {
      configurable: true,
      value: visualViewport,
    });

    const controller = createMobileViewportSyncController();

    try {
      input.focus();
      visualViewport.height = 620;
      visualViewport.dispatchEvent(new Event("resize"));
      expect(document.documentElement.style.getPropertyValue("--keyboard-offset")).toBe("312px");

      visualViewport.height = 932;
      visualViewport.dispatchEvent(new Event("resize"));
      expect(document.documentElement.style.getPropertyValue("--keyboard-offset")).toBe("312px");

      vi.advanceTimersByTime(MOBILE_VIEWPORT_ALIGN_COOLDOWN_MS + 16);
      expect(document.documentElement.style.getPropertyValue("--mobile-viewport-height")).toBe("932px");
      expect(document.documentElement.style.getPropertyValue("--keyboard-offset")).toBe("0px");
    } finally {
      controller.destroy();
      input.remove();
      vi.useRealTimers();
    }
  });

  it("re-syncs the recovered mobile viewport when the page returns to the foreground", () => {
    const visualViewport = new MockVisualViewport(430, 932);
    const input = document.createElement("textarea");
    let visibilityState: DocumentVisibilityState = "visible";
    document.body.appendChild(input);
    setWindowSize(430, 932);
    Object.defineProperty(window, "visualViewport", {
      configurable: true,
      value: visualViewport,
    });
    Object.defineProperty(document, "visibilityState", {
      configurable: true,
      get: () => visibilityState,
    });

    const controller = createMobileViewportSyncController();

    input.focus();
    visualViewport.height = 620;
    visualViewport.dispatchEvent(new Event("resize"));
    expect(document.documentElement.style.getPropertyValue("--keyboard-offset")).toBe("312px");

    input.blur();
    visualViewport.height = 932;
    visibilityState = "hidden";
    document.dispatchEvent(new Event("visibilitychange"));
    visibilityState = "visible";
    document.dispatchEvent(new Event("visibilitychange"));

    expect(document.documentElement.style.getPropertyValue("--mobile-viewport-height")).toBe("932px");
    expect(document.documentElement.style.getPropertyValue("--keyboard-offset")).toBe("0px");

    controller.destroy();
    input.remove();
  });
});

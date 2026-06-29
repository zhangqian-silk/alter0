import { act, render } from "@testing-library/react";
import { useRef } from "react";
import { useViewportScrollAnchor } from "./useViewportScrollAnchor";

type ScrollMetrics = {
  clientHeight: number;
  scrollHeight: number;
  scrollTop: number;
};

function defineScrollMetrics(node: HTMLElement, metrics: ScrollMetrics): void {
  Object.defineProperty(node, "clientHeight", {
    configurable: true,
    get: () => metrics.clientHeight,
  });
  Object.defineProperty(node, "scrollHeight", {
    configurable: true,
    get: () => metrics.scrollHeight,
  });
  Object.defineProperty(node, "scrollTop", {
    configurable: true,
    get: () => metrics.scrollTop,
    set: (value) => {
      metrics.scrollTop = Number(value || 0);
    },
  });
}

describe("useViewportScrollAnchor", () => {
  let captureAnchor: (() => void) | null = null;
  let frameID = 0;

  beforeEach(() => {
    captureAnchor = null;
    frameID = 0;
    vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback: FrameRequestCallback) => {
      frameID += 1;
      callback(frameID);
      return frameID;
    });
    vi.spyOn(window, "cancelAnimationFrame").mockImplementation(() => undefined);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  function TestHarness({
    active = true,
    enabled = true,
  }: {
    active?: boolean;
    enabled?: boolean;
  }) {
    const containerRef = useRef<HTMLDivElement | null>(null);
    captureAnchor = useViewportScrollAnchor({
      active,
      containerRef,
      enabled,
      focusSelector: "[data-scroll-anchor-input='true']",
    });

    return (
      <>
        <input data-scroll-anchor-input="true" />
        <div data-testid="scroll-container" ref={containerRef}></div>
      </>
    );
  }

  it("preserves the bottom distance when an active viewport resize shortens the container", () => {
    const metrics = { clientHeight: 500, scrollHeight: 1000, scrollTop: 500 };
    const { getByTestId } = render(<TestHarness />);
    const container = getByTestId("scroll-container");
    defineScrollMetrics(container, metrics);

    act(() => {
      captureAnchor?.();
      metrics.clientHeight = 300;
      window.dispatchEvent(new Event("resize"));
    });

    expect(metrics.scrollTop).toBe(700);
  });

  it("does not restore when the anchor is inactive", () => {
    const metrics = { clientHeight: 500, scrollHeight: 1000, scrollTop: 500 };
    const { getByTestId } = render(<TestHarness active={false} />);
    const container = getByTestId("scroll-container");
    defineScrollMetrics(container, metrics);

    act(() => {
      captureAnchor?.();
      metrics.clientHeight = 300;
      window.dispatchEvent(new Event("resize"));
    });

    expect(metrics.scrollTop).toBe(500);
  });

  it("does not pull a scrolled-away container back to the bottom", () => {
    const metrics = { clientHeight: 500, scrollHeight: 1000, scrollTop: 100 };
    const { getByTestId } = render(<TestHarness />);
    const container = getByTestId("scroll-container");
    defineScrollMetrics(container, metrics);

    act(() => {
      captureAnchor?.();
      metrics.clientHeight = 300;
      window.dispatchEvent(new Event("resize"));
    });

    expect(metrics.scrollTop).toBe(100);
  });
});

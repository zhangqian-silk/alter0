import { fireEvent, render, waitFor } from "@testing-library/react";
import { createRef } from "react";
import { ScrollJumpStrip } from "./ScrollJumpStrip";

function applyScrollableMetrics(
  container: HTMLElement,
  targets: HTMLElement[],
  targetTops: number[],
  options: {
    clientHeight: number;
    scrollHeight: number;
  },
) {
  Object.defineProperty(container, "clientHeight", {
    configurable: true,
    value: options.clientHeight,
  });
  Object.defineProperty(container, "scrollHeight", {
    configurable: true,
    value: options.scrollHeight,
  });
  Object.defineProperty(container, "scrollTop", {
    configurable: true,
    writable: true,
    value: 0,
  });
  container.getBoundingClientRect = () => ({
    x: 0,
    y: 0,
    width: 320,
    height: options.clientHeight,
    top: 0,
    right: 320,
    bottom: options.clientHeight,
    left: 0,
    toJSON: () => ({}),
  });
  container.scrollTo = vi.fn(({ top }: ScrollToOptions) => {
    container.scrollTop = Math.max(Number(top || 0), 0);
    fireEvent.scroll(container);
  });

  targets.forEach((target, index) => {
    target.getBoundingClientRect = () => {
      const top = targetTops[index] - container.scrollTop;
      return {
        x: 0,
        y: top,
        width: 280,
        height: 120,
        top,
        right: 280,
        bottom: top + 120,
        left: 0,
        toJSON: () => ({}),
      };
    };
  });
}

describe("ScrollJumpStrip", () => {
  beforeEach(() => {
    vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback: FrameRequestCallback) => {
      window.setTimeout(() => callback(16), 0);
      return 1;
    });
    vi.spyOn(window, "cancelAnimationFrame").mockImplementation(() => undefined);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("shows chat jump controls and scrolls to the next message block", async () => {
    const containerRef = createRef<HTMLElement>();
    const { container } = render(
      <section ref={containerRef}>
        <article data-message-id="message-1">message-1</article>
        <article data-message-id="message-2">message-2</article>
        <article data-message-id="message-3">message-3</article>
        <ScrollJumpStrip
          scope="chat"
          language="zh"
          containerRef={containerRef}
          itemSelector="[data-message-id]"
          itemAttribute="data-message-id"
        />
      </section>,
    );

    const scrollContainer = container.firstElementChild as HTMLElement;
    const targets = [...scrollContainer.querySelectorAll<HTMLElement>("[data-message-id]")];
    applyScrollableMetrics(scrollContainer, targets, [0, 320, 640], {
      clientHeight: 280,
      scrollHeight: 920,
    });

    scrollContainer.scrollTop = 360;
    fireEvent.scroll(scrollContainer);

    await waitFor(() => {
      expect(container.querySelector("[data-scroll-jump-top='chat']")).toHaveClass("is-visible");
      expect(container.querySelector("[data-scroll-jump-prev='chat']")).toHaveClass("is-visible");
      expect(container.querySelector("[data-scroll-jump-next='chat']")).toHaveClass("is-visible");
      expect(container.querySelector("[data-scroll-jump-bottom='chat']")).toHaveClass("is-visible");
    });

    fireEvent.click(container.querySelector("[data-scroll-jump-next='chat']") as HTMLElement);

    expect(scrollContainer.scrollTo).toHaveBeenCalledWith({
      top: 628,
      behavior: "smooth",
    });
  });

  it("assigns route anchor ids to visible sections before jumping", async () => {
    const containerRef = createRef<HTMLElement>();
    const { container } = render(
      <section ref={containerRef}>
        <div className="route-hero">hero</div>
        <div id="routeBody">
          <section>filters</section>
          <section>content</section>
          <section>details</section>
        </div>
        <ScrollJumpStrip
          scope="route"
          language="en"
          containerRef={containerRef}
          itemSelector=".route-hero, #routeBody > section"
          itemAttribute="data-scroll-jump-anchor"
        />
      </section>,
    );

    const scrollContainer = container.firstElementChild as HTMLElement;
    const targets = [
      ...scrollContainer.querySelectorAll<HTMLElement>(".route-hero, #routeBody > section"),
    ];
    applyScrollableMetrics(scrollContainer, targets, [0, 180, 420, 720], {
      clientHeight: 260,
      scrollHeight: 980,
    });

    scrollContainer.scrollTop = 260;
    fireEvent.scroll(scrollContainer);

    await waitFor(() => {
      expect(container.querySelector("[data-scroll-jump-next='route']")).toHaveClass("is-visible");
    });

    const assignedTargets = targets.map((node) => node.getAttribute("data-scroll-jump-anchor"));
    expect(assignedTargets.every(Boolean)).toBe(true);
  });
});

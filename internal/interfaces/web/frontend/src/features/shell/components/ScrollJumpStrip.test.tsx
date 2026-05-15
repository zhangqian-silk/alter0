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
    scrollTop?: number;
    targetHeights?: number[];
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
    value: options.scrollTop ?? 0,
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
  const scrollToMock = vi.fn((options?: ScrollToOptions | number, y?: number) => {
    const top = typeof options === "number" ? y : options?.top;
    container.scrollTop = Math.max(Number(top || 0), 0);
    fireEvent.scroll(container);
  }) as HTMLElement["scrollTo"];
  container.scrollTo = scrollToMock;

  targets.forEach((target, index) => {
    const height = options.targetHeights?.[index] ?? 120;
    Object.defineProperty(target, "offsetTop", {
      configurable: true,
      value: targetTops[index],
    });
    Object.defineProperty(target, "offsetHeight", {
      configurable: true,
      value: height,
    });
    target.getBoundingClientRect = () => {
      const top = targetTops[index] - container.scrollTop;
      return {
        x: 0,
        y: top,
        width: 280,
        height,
        top,
        right: 280,
        bottom: top + height,
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

  it("shows agent jump controls as arrow buttons and scrolls to the next message block", async () => {
    const containerRef = createRef<HTMLElement>();
    const { container } = render(
      <section ref={containerRef}>
        <article data-message-id="message-1">message-1</article>
        <article data-message-id="message-2">message-2</article>
        <article data-message-id="message-3">message-3</article>
        <ScrollJumpStrip
          scope="agent"
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
      expect(container.querySelector("[data-scroll-jump-top='agent']")).toHaveClass("is-visible");
      expect(container.querySelector("[data-scroll-jump-prev='agent']")).toHaveClass("is-visible");
      expect(container.querySelector("[data-scroll-jump-next='agent']")).toHaveClass("is-visible");
      expect(container.querySelector("[data-scroll-jump-bottom='agent']")).toHaveClass("is-visible");
    });

    expect(container.querySelector("[data-scroll-jump-top='agent']")).toHaveAttribute("aria-label", "回到顶部");
    expect(container.querySelector("[data-scroll-jump-prev='agent']")).toHaveTextContent("↑");
    expect(container.querySelector("[data-scroll-jump-next='agent']")).toHaveTextContent("↓");
    expect(container.querySelector("[data-scroll-jump-bottom='agent']")).toHaveAttribute("aria-label", "回到底部");

    fireEvent.click(container.querySelector("[data-scroll-jump-next='agent']") as HTMLElement);

    expect(scrollContainer.scrollTo).toHaveBeenCalledWith({
      top: 628,
      behavior: "smooth",
    });
  });

  it("applies terminal-style visible-range targeting for conversation timelines", async () => {
    const containerRef = createRef<HTMLElement>();
    const { container } = render(
      <section ref={containerRef}>
        <article data-message-id="message-1">message-1</article>
        <article data-message-id="message-2">message-2</article>
        <article data-message-id="message-3">message-3</article>
        <article data-message-id="message-4">message-4</article>
        <article data-message-id="message-5">message-5</article>
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
    applyScrollableMetrics(scrollContainer, targets, [0, 160, 320, 480, 640], {
      clientHeight: 360,
      scrollHeight: 920,
    });

    scrollContainer.scrollTop = 170;
    fireEvent.scroll(scrollContainer);

    await waitFor(() => {
      expect(container.querySelector("[data-scroll-jump-prev='chat']")).toHaveAttribute(
        "data-scroll-jump-target",
        "message-2",
      );
      expect(container.querySelector("[data-scroll-jump-next='chat']")).toHaveAttribute(
        "data-scroll-jump-target",
        "message-4",
      );
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
          scope="agent"
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
      expect(container.querySelector("[data-scroll-jump-next='agent']")).toHaveClass("is-visible");
    });

    const assignedTargets = targets.map((node) => node.getAttribute("data-scroll-jump-anchor"));
    expect(assignedTargets.every(Boolean)).toBe(true);
  });

  it("can render terminal jump controls with terminal-specific selectors", async () => {
    const containerRef = createRef<HTMLElement>();
    const { container } = render(
      <section ref={containerRef}>
        <article data-terminal-turn="turn-1">turn-1</article>
        <article data-terminal-turn="turn-2">turn-2</article>
        <article data-terminal-turn="turn-3">turn-3</article>
        <ScrollJumpStrip
          scope="terminal"
          namespace="terminal"
          language="en"
          containerRef={containerRef}
          itemSelector="[data-terminal-turn]"
          itemAttribute="data-terminal-turn"
        />
      </section>,
    );

    const scrollContainer = container.firstElementChild as HTMLElement;
    const targets = [...scrollContainer.querySelectorAll<HTMLElement>("[data-terminal-turn]")];
    applyScrollableMetrics(scrollContainer, targets, [0, 300, 620], {
      clientHeight: 260,
      scrollHeight: 900,
    });

    scrollContainer.scrollTop = 320;
    fireEvent.scroll(scrollContainer);

    await waitFor(() => {
      expect(container.querySelector("[data-terminal-jump-top]")).toHaveClass("is-visible");
      expect(container.querySelector("[data-terminal-jump-prev]")).toHaveAttribute("data-terminal-jump-target", "turn-2");
      expect(container.querySelector("[data-terminal-jump-next]")).toHaveAttribute("data-terminal-jump-target", "turn-3");
      expect(container.querySelector("[data-terminal-jump-bottom]")).toHaveClass("is-visible");
    });
  });

  it("keeps all jump controls hidden when every message already fits in the viewport", async () => {
    const containerRef = createRef<HTMLElement>();
    const { container } = render(
      <section ref={containerRef}>
        <article data-message-id="message-1">message-1</article>
        <article data-message-id="message-2">message-2</article>
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
    applyScrollableMetrics(scrollContainer, targets, [0, 140], {
      clientHeight: 520,
      scrollHeight: 520,
    });

    fireEvent.scroll(scrollContainer);

    await waitFor(() => {
      expect(container.querySelector("[data-scroll-jump-top='chat']")).not.toHaveClass("is-visible");
      expect(container.querySelector("[data-scroll-jump-prev='chat']")).not.toHaveClass("is-visible");
      expect(container.querySelector("[data-scroll-jump-next='chat']")).not.toHaveClass("is-visible");
      expect(container.querySelector("[data-scroll-jump-bottom='chat']")).not.toHaveClass("is-visible");
    });
  });

  it("keeps the upward jump controls available while the first message is only partially visible", async () => {
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
    applyScrollableMetrics(scrollContainer, targets, [0, 430, 620], {
      clientHeight: 320,
      scrollHeight: 900,
      scrollTop: 220,
      targetHeights: [400, 120, 120],
    });

    fireEvent.scroll(scrollContainer);

    await waitFor(() => {
      expect(container.querySelector("[data-scroll-jump-top='chat']")).toHaveClass("is-visible");
      expect(container.querySelector("[data-scroll-jump-prev='chat']")).toHaveClass("is-visible");
      expect(container.querySelector("[data-scroll-jump-next='chat']")).toHaveClass("is-visible");
      expect(container.querySelector("[data-scroll-jump-bottom='chat']")).toHaveClass("is-visible");
    });
  });

  it("hides the downward jump controls once the last message is fully visible and only blank space remains below", async () => {
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
    applyScrollableMetrics(scrollContainer, targets, [0, 220, 440], {
      clientHeight: 220,
      scrollHeight: 980,
      scrollTop: 360,
      targetHeights: [120, 120, 120],
    });

    fireEvent.scroll(scrollContainer);

    await waitFor(() => {
      expect(container.querySelector("[data-scroll-jump-top='chat']")).toHaveClass("is-visible");
      expect(container.querySelector("[data-scroll-jump-prev='chat']")).toHaveClass("is-visible");
      expect(container.querySelector("[data-scroll-jump-next='chat']")).not.toHaveClass("is-visible");
      expect(container.querySelector("[data-scroll-jump-bottom='chat']")).not.toHaveClass("is-visible");
    });
  });

  it("hides jump controls while the timeline has an active text selection and restores them after selection clears", async () => {
    const selectionState = {
      anchorNode: null as Node | null,
      focusNode: null as Node | null,
      rangeCount: 0,
      isCollapsed: true,
      text: "",
    };
    const rangeMock = {
      commonAncestorContainer: null as Node | null,
    } as Range;
    const selectionMock = {
      get anchorNode() {
        return selectionState.anchorNode;
      },
      get focusNode() {
        return selectionState.focusNode;
      },
      get rangeCount() {
        return selectionState.rangeCount;
      },
      get isCollapsed() {
        return selectionState.isCollapsed;
      },
      getRangeAt: vi.fn(() => rangeMock),
      toString: () => selectionState.text,
    } as unknown as Selection;
    vi.spyOn(document, "getSelection").mockImplementation(() => selectionMock);

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
      scrollTop: 360,
    });

    fireEvent.scroll(scrollContainer);

    await waitFor(() => {
      expect(container.querySelector("[data-scroll-jump-top='chat']")).toHaveClass("is-visible");
      expect(container.querySelector("[data-scroll-jump-prev='chat']")).toHaveClass("is-visible");
      expect(container.querySelector("[data-scroll-jump-next='chat']")).toHaveClass("is-visible");
      expect(container.querySelector("[data-scroll-jump-bottom='chat']")).toHaveClass("is-visible");
    });

    const selectedNode = targets[1]?.firstChild;
    selectionState.anchorNode = selectedNode;
    selectionState.focusNode = selectedNode;
    selectionState.rangeCount = 1;
    selectionState.isCollapsed = false;
    selectionState.text = "message-2";
    rangeMock.commonAncestorContainer = selectedNode;
    fireEvent(document, new Event("selectionchange"));

    await waitFor(() => {
      expect(container.querySelector("[data-scroll-jump-top='chat']")).not.toHaveClass("is-visible");
      expect(container.querySelector("[data-scroll-jump-prev='chat']")).not.toHaveClass("is-visible");
      expect(container.querySelector("[data-scroll-jump-next='chat']")).not.toHaveClass("is-visible");
      expect(container.querySelector("[data-scroll-jump-bottom='chat']")).not.toHaveClass("is-visible");
      expect(container.querySelector("[data-scroll-jump-scope='chat']")).toHaveAttribute(
        "data-scroll-jump-selection-active",
        "true",
      );
    });

    selectionState.anchorNode = null;
    selectionState.focusNode = null;
    selectionState.rangeCount = 0;
    selectionState.isCollapsed = true;
    selectionState.text = "";
    rangeMock.commonAncestorContainer = null;
    fireEvent(document, new Event("selectionchange"));

    await waitFor(() => {
      expect(container.querySelector("[data-scroll-jump-top='chat']")).toHaveClass("is-visible");
      expect(container.querySelector("[data-scroll-jump-prev='chat']")).toHaveClass("is-visible");
      expect(container.querySelector("[data-scroll-jump-next='chat']")).toHaveClass("is-visible");
      expect(container.querySelector("[data-scroll-jump-bottom='chat']")).toHaveClass("is-visible");
      expect(container.querySelector("[data-scroll-jump-scope='chat']")).toHaveAttribute(
        "data-scroll-jump-selection-active",
        "false",
      );
    });
  });
});

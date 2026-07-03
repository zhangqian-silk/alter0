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

  it("shows chat jump controls as arrow buttons and scrolls to the next message block", async () => {
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

    expect(container.querySelector("[data-scroll-jump-top='chat']")).toHaveAttribute("aria-label", "回到顶部");
    expect(container.querySelector("[data-scroll-jump-prev='chat']")).toHaveTextContent("↑");
    expect(container.querySelector("[data-scroll-jump-next='chat']")).toHaveTextContent("↓");
    expect(container.querySelector("[data-scroll-jump-bottom='chat']")).toHaveAttribute("aria-label", "回到底部");

    fireEvent.click(container.querySelector("[data-scroll-jump-next='chat']") as HTMLElement);

    expect(scrollContainer.scrollTo).toHaveBeenCalledWith({
      top: 628,
      behavior: "smooth",
    });
  });

  it("applies chatRuntime-style visible-range targeting for conversation timelines", async () => {
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

  it("continues moving upward after the current visible item has been aligned", async () => {
    const containerRef = createRef<HTMLElement>();
    const { container } = render(
      <section ref={containerRef}>
        <article data-message-id="message-1">message-1</article>
        <article data-message-id="message-2">message-2</article>
        <article data-message-id="message-3">message-3</article>
        <article data-message-id="message-4">message-4</article>
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
    applyScrollableMetrics(scrollContainer, targets, [0, 160, 320, 480], {
      clientHeight: 260,
      scrollHeight: 760,
    });

    scrollContainer.scrollTop = 170;
    fireEvent.scroll(scrollContainer);

    await waitFor(() => {
      expect(container.querySelector("[data-scroll-jump-prev='chat']")).toHaveAttribute(
        "data-scroll-jump-target",
        "message-2",
      );
    });

    fireEvent.click(container.querySelector("[data-scroll-jump-prev='chat']") as HTMLElement);

    expect(scrollContainer.scrollTop).toBe(148);

    await waitFor(() => {
      expect(container.querySelector("[data-scroll-jump-prev='chat']")).toHaveAttribute(
        "data-scroll-jump-target",
        "message-1",
      );
    });

    fireEvent.click(container.querySelector("[data-scroll-jump-prev='chat']") as HTMLElement);

    expect(scrollContainer.scrollTop).toBe(0);
  });

  it("remeasures jump targets after earlier messages are prepended", async () => {
    const containerRef = createRef<HTMLElement>();
    const initialMessages = Array.from({ length: 41 }, (_value, index) => `message-${index + 40}`);
    const renderStrip = (messages: string[]) => (
      <section ref={containerRef}>
        {messages.map((id) => (
          <article key={id} data-message-id={id}>{id}</article>
        ))}
        <ScrollJumpStrip
          scope="chat"
          language="zh"
          containerRef={containerRef}
          itemSelector="[data-message-id]"
          itemAttribute="data-message-id"
        />
      </section>
    );
    const { container, rerender } = render(renderStrip(initialMessages));

    const scrollContainer = container.firstElementChild as HTMLElement;
    let targets = [...scrollContainer.querySelectorAll<HTMLElement>("[data-message-id]")];
    applyScrollableMetrics(
      scrollContainer,
      targets,
      targets.map((_target, index) => index * 100),
      {
        clientHeight: 260,
        scrollHeight: 4300,
        scrollTop: 3921,
      },
    );
    fireEvent.scroll(scrollContainer);

    await waitFor(() => {
      expect(container.querySelector("[data-scroll-jump-prev='chat']")).toHaveAttribute(
        "data-scroll-jump-target",
        "message-79",
      );
    });

    const expandedMessages = Array.from({ length: 65 }, (_value, index) => `message-${index + 16}`);
    rerender(renderStrip(expandedMessages));
    targets = [...scrollContainer.querySelectorAll<HTMLElement>("[data-message-id]")];
    applyScrollableMetrics(
      scrollContainer,
      targets,
      targets.map((_target, index) => index * 100),
      {
        clientHeight: 260,
        scrollHeight: 6700,
        scrollTop: 2421,
      },
    );
    fireEvent.scroll(scrollContainer);

    await waitFor(() => {
      expect(container.querySelector("[data-scroll-jump-prev='chat']")).toHaveAttribute(
        "data-scroll-jump-target",
        "message-40",
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
          scope="chat"
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
      expect(container.querySelector("[data-scroll-jump-next='chat']")).toHaveClass("is-visible");
    });

    const assignedTargets = targets.map((node) => node.getAttribute("data-scroll-jump-anchor"));
    expect(assignedTargets.every(Boolean)).toBe(true);
  });

  it("can render chatRuntime jump controls with chatRuntime-specific selectors", async () => {
    const containerRef = createRef<HTMLElement>();
    const { container } = render(
      <section ref={containerRef}>
        <article data-chat-runtime-turn="turn-1">turn-1</article>
        <article data-chat-runtime-turn="turn-2">turn-2</article>
        <article data-chat-runtime-turn="turn-3">turn-3</article>
        <ScrollJumpStrip
          scope="chatRuntime"
          namespace="chatRuntime"
          language="en"
          containerRef={containerRef}
          itemSelector="[data-chat-runtime-turn]"
          itemAttribute="data-chat-runtime-turn"
        />
      </section>,
    );

    const scrollContainer = container.firstElementChild as HTMLElement;
    const targets = [...scrollContainer.querySelectorAll<HTMLElement>("[data-chat-runtime-turn]")];
    applyScrollableMetrics(scrollContainer, targets, [0, 300, 620], {
      clientHeight: 260,
      scrollHeight: 900,
    });

    scrollContainer.scrollTop = 320;
    fireEvent.scroll(scrollContainer);

    await waitFor(() => {
      expect(container.querySelector("[data-chat-runtime-jump-top]")).toHaveClass("is-visible");
      expect(container.querySelector("[data-chat-runtime-jump-prev]")).toHaveAttribute("data-chat-runtime-jump-target", "turn-2");
      expect(container.querySelector("[data-chat-runtime-jump-next]")).toHaveAttribute("data-chat-runtime-jump-target", "turn-3");
      expect(container.querySelector("[data-chat-runtime-jump-bottom]")).toHaveClass("is-visible");
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

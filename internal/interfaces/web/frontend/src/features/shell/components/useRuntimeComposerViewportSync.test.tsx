import { fireEvent, render } from "@testing-library/react";
import { useRef } from "react";
import { useRuntimeComposerViewportSync } from "./useRuntimeComposerViewportSync";

function setRect(node: HTMLElement, rect: Pick<DOMRect, "top" | "bottom" | "height">) {
  Object.defineProperty(node, "getBoundingClientRect", {
    configurable: true,
    value: () => ({
      x: 0,
      y: rect.top,
      width: 0,
      left: 0,
      right: 0,
      toJSON: () => ({}),
      ...rect,
    }),
  });
}

function Harness({
  isMobileViewport,
  inputFocused,
  workspaceRect = { top: 0, bottom: 640, height: 640 },
  composerRect = { top: 520, bottom: 640, height: 120 },
}: {
  isMobileViewport: boolean;
  inputFocused: boolean;
  workspaceRect?: Pick<DOMRect, "top" | "bottom" | "height">;
  composerRect?: Pick<DOMRect, "top" | "bottom" | "height">;
}) {
  const workspaceBodyRef = useRef<HTMLDivElement | null>(null);
  const composerShellRef = useRef<HTMLElement | null>(null);

  useRuntimeComposerViewportSync({
    isMobileViewport,
    inputFocused,
    workspaceBodyRef,
    composerShellRef,
  });

  return (
    <div
      ref={(node) => {
        workspaceBodyRef.current = node;
        if (node) {
          setRect(node, workspaceRect);
        }
      }}
      data-testid="workspace-body"
    >
      <section className="runtime-workspace-screen" data-testid="workspace-screen" />
      <footer
        ref={(node) => {
          composerShellRef.current = node;
          if (node) {
            setRect(node, composerRect);
          }
        }}
        data-testid="composer-shell"
      >
        <textarea data-runtime-composer-input="terminal" data-testid="composer-input" />
      </footer>
    </div>
  );
}

describe("useRuntimeComposerViewportSync", () => {
  beforeEach(() => {
    document.documentElement.style.setProperty("--keyboard-offset", "0px");
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("keeps the resting inset equal to the visible composer height even when the keyboard offset is larger", () => {
    document.documentElement.style.setProperty("--keyboard-offset", "332px");

    render(<Harness isMobileViewport inputFocused={false} />);

    const workspaceBody = document.querySelector("[data-testid='workspace-body']") as HTMLDivElement;
    expect(workspaceBody.style.getPropertyValue("--runtime-composer-rest-inset")).toBe("120px");
    expect(workspaceBody.style.getPropertyValue("--runtime-composer-inset")).toBe("0px");
  });

  it("keeps the workspace inset static while the composer follows the mobile keyboard", () => {
    render(
      <Harness
        isMobileViewport
        inputFocused
        workspaceRect={{ top: 0, bottom: 932, height: 932 }}
        composerRect={{ top: 468, bottom: 620, height: 152 }}
      />,
    );

    const workspaceBody = document.querySelector("[data-testid='workspace-body']") as HTMLDivElement;
    expect(workspaceBody.style.getPropertyValue("--runtime-composer-rest-inset")).toBe("152px");
    expect(workspaceBody.style.getPropertyValue("--runtime-composer-inset")).toBe("0px");
  });

  it("does not override browser page scroll while the mobile composer input remains focused", () => {
    const originalScrollX = Object.getOwnPropertyDescriptor(window, "scrollX");
    const originalScrollY = Object.getOwnPropertyDescriptor(window, "scrollY");
    const input = document.createElement("textarea");
    document.body.appendChild(input);
    input.focus();
    Object.defineProperty(window, "scrollX", {
      configurable: true,
      value: 0,
    });
    Object.defineProperty(window, "scrollY", {
      configurable: true,
      value: 48,
    });
    const scrollTo = vi.spyOn(window, "scrollTo").mockImplementation(() => undefined);
    vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback: FrameRequestCallback) => {
      callback(0);
      return 1;
    });
    vi.spyOn(window, "cancelAnimationFrame").mockImplementation(() => undefined);

    try {
      render(<Harness isMobileViewport inputFocused />);
      window.dispatchEvent(new Event("scroll"));

      expect(scrollTo).not.toHaveBeenCalled();
    } finally {
      if (originalScrollX) {
        Object.defineProperty(window, "scrollX", originalScrollX);
      }
      if (originalScrollY) {
        Object.defineProperty(window, "scrollY", originalScrollY);
      }
      input.remove();
    }
  });

  it("leaves workspace scroll positions to the active scroll container during mobile keyboard focus", () => {
    render(<Harness isMobileViewport inputFocused />);

    const workspaceBody = document.querySelector("[data-testid='workspace-body']") as HTMLDivElement;
    const workspaceScreen = document.querySelector("[data-testid='workspace-screen']") as HTMLElement;
    workspaceBody.scrollTop = 84;
    workspaceScreen.scrollTop = 132;

    window.dispatchEvent(new Event("scroll"));

    expect(workspaceBody.scrollTop).toBe(84);
    expect(workspaceScreen.scrollTop).toBe(132);
  });

  it("does not lock background workspace scroll during mobile keyboard focus", () => {
    const { rerender } = render(<Harness isMobileViewport inputFocused={false} />);
    const workspaceBody = document.querySelector("[data-testid='workspace-body']") as HTMLDivElement;
    const workspaceScreen = document.querySelector("[data-testid='workspace-screen']") as HTMLElement;
    const scrollTo = vi.spyOn(window, "scrollTo").mockImplementation(() => undefined);
    workspaceBody.scrollTop = 84;
    workspaceScreen.scrollTop = 132;

    rerender(<Harness isMobileViewport inputFocused />);
    workspaceBody.scrollTop = 160;
    workspaceScreen.scrollTop = 240;
    workspaceBody.dispatchEvent(new Event("scroll"));
    workspaceScreen.dispatchEvent(new Event("scroll"));

    expect(scrollTo).not.toHaveBeenCalled();
    expect(workspaceBody.scrollTop).toBe(160);
    expect(workspaceScreen.scrollTop).toBe(240);
  });

  it("leaves the first mobile composer touch to native focus without locking page scroll", () => {
    vi.useFakeTimers();
    const originalScrollX = Object.getOwnPropertyDescriptor(window, "scrollX");
    const originalScrollY = Object.getOwnPropertyDescriptor(window, "scrollY");
    let scrollX = 0;
    let scrollY = 0;
    Object.defineProperty(window, "scrollX", {
      configurable: true,
      get: () => scrollX,
    });
    Object.defineProperty(window, "scrollY", {
      configurable: true,
      get: () => scrollY,
    });
    const scrollTo = vi.spyOn(window, "scrollTo").mockImplementation((options?: ScrollToOptions | number) => {
      if (typeof options === "object") {
        scrollX = Number(options.left || 0);
        scrollY = Number(options.top || 0);
      }
    });
    vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback: FrameRequestCallback) => {
      callback(0);
      return 1;
    });
    vi.spyOn(window, "cancelAnimationFrame").mockImplementation(() => undefined);

    try {
      render(<Harness isMobileViewport inputFocused={false} />);
      const input = document.querySelector("[data-testid='composer-input']") as HTMLTextAreaElement;
      const focus = vi.spyOn(input, "focus").mockImplementation(() => undefined);
      const workspaceBody = document.querySelector("[data-testid='workspace-body']") as HTMLDivElement;
      const workspaceScreen = document.querySelector("[data-testid='workspace-screen']") as HTMLElement;
      workspaceBody.scrollTop = 84;
      workspaceScreen.scrollTop = 132;

      fireEvent.pointerDown(input, { pointerType: "touch" });
      scrollY = 96;
      window.dispatchEvent(new Event("scroll"));
      vi.advanceTimersByTime(200);

      expect(focus).not.toHaveBeenCalled();
      expect(scrollTo).not.toHaveBeenCalled();
      expect(scrollY).toBe(96);
      expect(workspaceBody.scrollTop).toBe(84);
      expect(workspaceScreen.scrollTop).toBe(132);
    } finally {
      vi.useRealTimers();
      if (originalScrollX) {
        Object.defineProperty(window, "scrollX", originalScrollX);
      }
      if (originalScrollY) {
        Object.defineProperty(window, "scrollY", originalScrollY);
      }
    }
  });

  it("does not run delayed page-level scroll corrections after mobile keyboard focus", () => {
    vi.useFakeTimers();
    const originalScrollX = Object.getOwnPropertyDescriptor(window, "scrollX");
    const originalScrollY = Object.getOwnPropertyDescriptor(window, "scrollY");
    const input = document.createElement("textarea");
    let scrollY = 0;
    document.body.appendChild(input);
    input.focus();
    Object.defineProperty(window, "scrollX", {
      configurable: true,
      value: 0,
    });
    Object.defineProperty(window, "scrollY", {
      configurable: true,
      get: () => scrollY,
    });
    const scrollTo = vi.spyOn(window, "scrollTo").mockImplementation(() => undefined);
    vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback: FrameRequestCallback) => {
      callback(0);
      return 1;
    });
    vi.spyOn(window, "cancelAnimationFrame").mockImplementation(() => undefined);

    try {
      render(<Harness isMobileViewport inputFocused />);
      scrollY = 64;
      vi.advanceTimersByTime(96);

      expect(scrollTo).not.toHaveBeenCalled();
    } finally {
      vi.useRealTimers();
      if (originalScrollX) {
        Object.defineProperty(window, "scrollX", originalScrollX);
      }
      if (originalScrollY) {
        Object.defineProperty(window, "scrollY", originalScrollY);
      }
      input.remove();
    }
  });

  it("does not replay scroll anchors captured before the mobile composer input receives focus", () => {
    const originalScrollX = Object.getOwnPropertyDescriptor(window, "scrollX");
    const originalScrollY = Object.getOwnPropertyDescriptor(window, "scrollY");
    let scrollY = 0;
    Object.defineProperty(window, "scrollX", {
      configurable: true,
      value: 0,
    });
    Object.defineProperty(window, "scrollY", {
      configurable: true,
      get: () => scrollY,
    });
    const scrollTo = vi.spyOn(window, "scrollTo").mockImplementation(() => undefined);
    vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback: FrameRequestCallback) => {
      callback(0);
      return 1;
    });
    vi.spyOn(window, "cancelAnimationFrame").mockImplementation(() => undefined);

    try {
      const { rerender } = render(<Harness isMobileViewport inputFocused={false} />);
      const workspaceBody = document.querySelector("[data-testid='workspace-body']") as HTMLDivElement;
      const workspaceScreen = document.querySelector("[data-testid='workspace-screen']") as HTMLElement;

      scrollY = 72;
      workspaceBody.scrollTop = 96;
      workspaceScreen.scrollTop = 144;
      rerender(<Harness isMobileViewport inputFocused />);

      expect(scrollTo).not.toHaveBeenCalled();
      expect(workspaceBody.scrollTop).toBe(96);
      expect(workspaceScreen.scrollTop).toBe(144);
    } finally {
      if (originalScrollX) {
        Object.defineProperty(window, "scrollX", originalScrollX);
      }
      if (originalScrollY) {
        Object.defineProperty(window, "scrollY", originalScrollY);
      }
    }
  });
});

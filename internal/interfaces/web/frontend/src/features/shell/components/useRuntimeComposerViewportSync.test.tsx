import { render } from "@testing-library/react";
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
      />
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

  it("anchors page-level scroll while the mobile composer input remains focused", () => {
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

      expect(scrollTo).toHaveBeenCalledWith({ left: 0, top: 0, behavior: "auto" });
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

  it("restores workspace scroll positions changed by mobile keyboard focus alignment", () => {
    render(<Harness isMobileViewport inputFocused />);

    const workspaceBody = document.querySelector("[data-testid='workspace-body']") as HTMLDivElement;
    const workspaceScreen = document.querySelector("[data-testid='workspace-screen']") as HTMLElement;
    workspaceBody.scrollTop = 84;
    workspaceScreen.scrollTop = 132;

    window.dispatchEvent(new Event("scroll"));

    expect(workspaceBody.scrollTop).toBe(0);
    expect(workspaceScreen.scrollTop).toBe(0);
  });

  it("rechecks page-level scroll after delayed mobile keyboard focus alignment", () => {
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

      expect(scrollTo).toHaveBeenCalledWith({ left: 0, top: 0, behavior: "auto" });
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
});

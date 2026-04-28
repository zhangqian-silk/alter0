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
}: {
  isMobileViewport: boolean;
  inputFocused: boolean;
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
          setRect(node, { top: 0, bottom: 640, height: 640 });
        }
      }}
      data-testid="workspace-body"
    >
      <footer
        ref={(node) => {
          composerShellRef.current = node;
          if (node) {
            setRect(node, { top: 520, bottom: 640, height: 120 });
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

  it("keeps the resting inset equal to the visible composer height even when the keyboard offset is larger", () => {
    document.documentElement.style.setProperty("--keyboard-offset", "332px");

    render(<Harness isMobileViewport inputFocused={false} />);

    const workspaceBody = document.querySelector("[data-testid='workspace-body']") as HTMLDivElement;
    expect(workspaceBody.style.getPropertyValue("--runtime-composer-rest-inset")).toBe("120px");
    expect(workspaceBody.style.getPropertyValue("--runtime-composer-inset")).toBe("0px");
  });
});

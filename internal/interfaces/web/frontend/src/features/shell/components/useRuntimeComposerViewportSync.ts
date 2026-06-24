import { useLayoutEffect, useRef, type RefObject } from "react";

type UseRuntimeComposerViewportSyncProps = {
  isMobileViewport: boolean;
  inputFocused: boolean;
  workspaceBodyRef: RefObject<HTMLDivElement | null>;
  composerShellRef: RefObject<HTMLElement | null>;
};

type ScrollAnchor = {
  node: HTMLElement;
  top: number;
  left: number;
};

function collectScrollAnchors(workspaceBodyNode: HTMLElement | null): ScrollAnchor[] {
  const nodes = new Set<HTMLElement>();
  const scrollingElement = document.scrollingElement;
  if (scrollingElement instanceof HTMLElement) {
    nodes.add(scrollingElement);
  }
  if (document.body) {
    nodes.add(document.body);
  }
  if (workspaceBodyNode) {
    nodes.add(workspaceBodyNode);
    let ancestor = workspaceBodyNode.parentElement;
    while (ancestor) {
      if (
        ancestor.matches(
          ".app-shell, .workbench-main, .workbench-pane-shell, .chat-pane, .runtime-workspace, .runtime-workspace-view",
        )
      ) {
        nodes.add(ancestor);
      }
      ancestor = ancestor.parentElement;
    }
    workspaceBodyNode
      .querySelectorAll<HTMLElement>(".runtime-workspace-panel, .runtime-workspace-screen")
      .forEach((node) => nodes.add(node));
  }
  return Array.from(nodes, (node) => ({
    node,
    top: node.scrollTop,
    left: node.scrollLeft,
  }));
}

function restoreScrollAnchors(scrollAnchors: ScrollAnchor[]) {
  if (window.scrollX !== 0 || window.scrollY !== 0) {
    window.scrollTo({ left: 0, top: 0, behavior: "auto" });
  }
  scrollAnchors.forEach(({ node, top, left }) => {
    if (node.scrollTop !== top) {
      node.scrollTop = top;
    }
    if (node.scrollLeft !== left) {
      node.scrollLeft = left;
    }
  });
}

export function useRuntimeComposerViewportSync({
  isMobileViewport,
  inputFocused,
  workspaceBodyRef,
  composerShellRef,
}: UseRuntimeComposerViewportSyncProps) {
  const pendingInputFocusAnchorsRef = useRef<ScrollAnchor[] | null>(null);

  useLayoutEffect(() => {
    const composerShellNode = composerShellRef.current;
    if (!isMobileViewport || !composerShellNode) {
      pendingInputFocusAnchorsRef.current = null;
      return;
    }
    const captureInputFocusAnchor = (event: Event) => {
      const target = event.target;
      if (!(target instanceof Element) || !target.closest("[data-runtime-composer-input]")) {
        return;
      }
      pendingInputFocusAnchorsRef.current = collectScrollAnchors(workspaceBodyRef.current);
    };
    composerShellNode.addEventListener("pointerdown", captureInputFocusAnchor, true);
    composerShellNode.addEventListener("touchstart", captureInputFocusAnchor, true);
    return () => {
      composerShellNode.removeEventListener("pointerdown", captureInputFocusAnchor, true);
      composerShellNode.removeEventListener("touchstart", captureInputFocusAnchor, true);
    };
  }, [composerShellRef, isMobileViewport, workspaceBodyRef]);

  useLayoutEffect(() => {
    if (!isMobileViewport || !inputFocused) {
      return;
    }
    const scrollAnchors = pendingInputFocusAnchorsRef.current ?? collectScrollAnchors(workspaceBodyRef.current);
    pendingInputFocusAnchorsRef.current = null;
    const keepViewportAnchored = () => {
      restoreScrollAnchors(scrollAnchors);
    };
    const frameID = window.requestAnimationFrame(keepViewportAnchored);
    const lateFrameIDs = [96, 180, 280].map((delayMS) =>
      window.setTimeout(keepViewportAnchored, delayMS),
    );
    const visualViewport = window.visualViewport;
    window.addEventListener("scroll", keepViewportAnchored, { passive: true });
    visualViewport?.addEventListener("resize", keepViewportAnchored);
    visualViewport?.addEventListener("scroll", keepViewportAnchored);
    return () => {
      window.cancelAnimationFrame(frameID);
      lateFrameIDs.forEach((timeoutID) => window.clearTimeout(timeoutID));
      window.removeEventListener("scroll", keepViewportAnchored);
      visualViewport?.removeEventListener("resize", keepViewportAnchored);
      visualViewport?.removeEventListener("scroll", keepViewportAnchored);
    };
  }, [inputFocused, isMobileViewport, workspaceBodyRef]);

  useLayoutEffect(() => {
    const workspaceBodyNode = workspaceBodyRef.current;
    const composerShellNode = composerShellRef.current;
    if (!workspaceBodyNode) {
      return;
    }
    if (!isMobileViewport || !composerShellNode) {
      workspaceBodyNode.style.removeProperty("--runtime-composer-inset");
      workspaceBodyNode.style.removeProperty("--runtime-composer-rest-inset");
      return;
    }

    const syncComposerInset = () => {
      const composerRect = composerShellNode.getBoundingClientRect();
      const restInset = Math.max(0, Math.ceil(composerRect.height));
      workspaceBodyNode.style.setProperty(
        "--runtime-composer-rest-inset",
        `${restInset}px`,
      );
      workspaceBodyNode.style.setProperty(
        "--runtime-composer-inset",
        "0px",
      );
    };

    let settleFrameID = 0;
    let settleLateFrameID = 0;
    let settleTimeoutID = 0;
    const clearScheduledSync = () => {
      if (settleFrameID) {
        window.cancelAnimationFrame(settleFrameID);
        settleFrameID = 0;
      }
      if (settleLateFrameID) {
        window.cancelAnimationFrame(settleLateFrameID);
        settleLateFrameID = 0;
      }
      if (settleTimeoutID) {
        window.clearTimeout(settleTimeoutID);
        settleTimeoutID = 0;
      }
    };
    const scheduleComposerInsetSync = () => {
      syncComposerInset();
      clearScheduledSync();
      settleFrameID = window.requestAnimationFrame(() => {
        settleFrameID = 0;
        syncComposerInset();
        settleLateFrameID = window.requestAnimationFrame(() => {
          settleLateFrameID = 0;
          syncComposerInset();
        });
      });
      settleTimeoutID = window.setTimeout(() => {
        settleTimeoutID = 0;
        syncComposerInset();
      }, 260);
    };

    scheduleComposerInsetSync();

    const resizeObserver = typeof ResizeObserver === "undefined"
      ? null
      : new ResizeObserver(() => scheduleComposerInsetSync());
    resizeObserver?.observe(composerShellNode);
    window.addEventListener("resize", scheduleComposerInsetSync);
    window.visualViewport?.addEventListener("resize", scheduleComposerInsetSync);
    window.visualViewport?.addEventListener("scroll", scheduleComposerInsetSync);
    composerShellNode.addEventListener("transitionend", scheduleComposerInsetSync);
    return () => {
      clearScheduledSync();
      resizeObserver?.disconnect();
      window.removeEventListener("resize", scheduleComposerInsetSync);
      window.visualViewport?.removeEventListener("resize", scheduleComposerInsetSync);
      window.visualViewport?.removeEventListener("scroll", scheduleComposerInsetSync);
      composerShellNode.removeEventListener("transitionend", scheduleComposerInsetSync);
      workspaceBodyNode.style.removeProperty("--runtime-composer-inset");
      workspaceBodyNode.style.removeProperty("--runtime-composer-rest-inset");
    };
  }, [composerShellRef, inputFocused, isMobileViewport, workspaceBodyRef]);
}

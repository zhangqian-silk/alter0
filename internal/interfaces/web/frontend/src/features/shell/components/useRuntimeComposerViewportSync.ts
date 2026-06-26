import { useLayoutEffect, type RefObject } from "react";

type UseRuntimeComposerViewportSyncProps = {
  isMobileViewport: boolean;
  inputFocused: boolean;
  workspaceBodyRef: RefObject<HTMLDivElement | null>;
  composerShellRef: RefObject<HTMLElement | null>;
};

export function useRuntimeComposerViewportSync({
  isMobileViewport,
  workspaceBodyRef,
  composerShellRef,
}: UseRuntimeComposerViewportSyncProps) {
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
  }, [composerShellRef, isMobileViewport, workspaceBodyRef]);
}

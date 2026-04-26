import { useLayoutEffect, type RefObject } from "react";

type UseRuntimeComposerViewportSyncProps = {
  isMobileViewport: boolean;
  inputFocused: boolean;
  workspaceBodyRef: RefObject<HTMLDivElement | null>;
  composerShellRef: RefObject<HTMLElement | null>;
};

export function useRuntimeComposerViewportSync({
  isMobileViewport,
  inputFocused,
  workspaceBodyRef,
  composerShellRef,
}: UseRuntimeComposerViewportSyncProps) {
  useLayoutEffect(() => {
    if (!isMobileViewport || !inputFocused) {
      return;
    }
    const keepViewportAnchored = () => {
      if (window.scrollX !== 0 || window.scrollY !== 0) {
        window.scrollTo({ left: 0, top: 0, behavior: "auto" });
      }
    };
    const frameID = window.requestAnimationFrame(keepViewportAnchored);
    const visualViewport = window.visualViewport;
    window.addEventListener("scroll", keepViewportAnchored, { passive: true });
    visualViewport?.addEventListener("resize", keepViewportAnchored);
    visualViewport?.addEventListener("scroll", keepViewportAnchored);
    return () => {
      window.cancelAnimationFrame(frameID);
      window.removeEventListener("scroll", keepViewportAnchored);
      visualViewport?.removeEventListener("resize", keepViewportAnchored);
      visualViewport?.removeEventListener("scroll", keepViewportAnchored);
    };
  }, [inputFocused, isMobileViewport]);

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
      const workspaceRect = workspaceBodyNode.getBoundingClientRect();
      const composerRect = composerShellNode.getBoundingClientRect();
      const keyboardOffsetValue = Number.parseFloat(
        window.getComputedStyle(document.documentElement).getPropertyValue("--keyboard-offset"),
      );
      const keyboardOffset = Number.isFinite(keyboardOffsetValue) ? keyboardOffsetValue : 0;
      const restInset = Math.max(0, Math.ceil(composerRect.height - keyboardOffset));
      const activeInset = inputFocused
        ? Math.max(0, Math.ceil(workspaceRect.bottom - composerRect.top))
        : 0;
      workspaceBodyNode.style.setProperty(
        "--runtime-composer-rest-inset",
        `${restInset}px`,
      );
      workspaceBodyNode.style.setProperty(
        "--runtime-composer-inset",
        `${activeInset}px`,
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

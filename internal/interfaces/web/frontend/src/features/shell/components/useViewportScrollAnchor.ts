import { useCallback, useEffect, useRef } from "react";

type ScrollAnchorContainerRef = {
  current: HTMLElement | null;
};

type ViewportScrollAnchorSnapshot = {
  scrollHeight: number;
  clientHeight: number;
  scrollTop: number;
  bottomDistance: number;
  anchoredToBottom: boolean;
};

type UseViewportScrollAnchorOptions = {
  active: boolean;
  containerRef: ScrollAnchorContainerRef;
  enabled: boolean;
  focusSelector?: string;
  maxBottomThreshold?: number;
  minBottomThreshold?: number;
};

const DEFAULT_MIN_BOTTOM_THRESHOLD = 96;
const DEFAULT_MAX_BOTTOM_THRESHOLD = 240;

function resolveBottomAnchorThreshold(
  clientHeight: number,
  minBottomThreshold: number,
  maxBottomThreshold: number,
): number {
  return Math.max(
    minBottomThreshold,
    Math.min(maxBottomThreshold, clientHeight * 0.25),
  );
}

function readScrollAnchorSnapshot(
  node: HTMLElement,
  minBottomThreshold: number,
  maxBottomThreshold: number,
): ViewportScrollAnchorSnapshot {
  const bottomDistance = Math.max(node.scrollHeight - node.clientHeight - node.scrollTop, 0);
  const scrollable = node.scrollHeight > node.clientHeight + 1;
  return {
    scrollHeight: node.scrollHeight,
    clientHeight: node.clientHeight,
    scrollTop: node.scrollTop,
    bottomDistance,
    anchoredToBottom: scrollable && bottomDistance <= resolveBottomAnchorThreshold(
      node.clientHeight,
      minBottomThreshold,
      maxBottomThreshold,
    ),
  };
}

export function useViewportScrollAnchor({
  active,
  containerRef,
  enabled,
  focusSelector,
  maxBottomThreshold = DEFAULT_MAX_BOTTOM_THRESHOLD,
  minBottomThreshold = DEFAULT_MIN_BOTTOM_THRESHOLD,
}: UseViewportScrollAnchorOptions): () => void {
  const activeRef = useRef(active);
  const snapshotRef = useRef<ViewportScrollAnchorSnapshot | null>(null);

  useEffect(() => {
    activeRef.current = active;
  }, [active]);

  const capture = useCallback(() => {
    const node = containerRef.current;
    snapshotRef.current = node
      ? readScrollAnchorSnapshot(node, minBottomThreshold, maxBottomThreshold)
      : null;
  }, [containerRef, maxBottomThreshold, minBottomThreshold]);

  useEffect(() => {
    const node = containerRef.current;
    if (!enabled || !node) {
      return undefined;
    }

    const isFocusSelectorActive = () => {
      if (!focusSelector) {
        return false;
      }
      const activeElement = document.activeElement;
      return activeElement instanceof HTMLElement && activeElement.matches(focusSelector);
    };
    const shouldRestoreAnchor = () => activeRef.current || isFocusSelectorActive();
    const restoreAnchor = () => {
      if (!shouldRestoreAnchor()) {
        return;
      }
      const snapshot = snapshotRef.current;
      if (!snapshot?.anchoredToBottom) {
        capture();
        return;
      }
      const current = readScrollAnchorSnapshot(node, minBottomThreshold, maxBottomThreshold);
      if (
        current.clientHeight === snapshot.clientHeight
        && current.scrollHeight === snapshot.scrollHeight
      ) {
        snapshotRef.current = current;
        return;
      }
      node.scrollTop = Math.max(0, node.scrollHeight - node.clientHeight - snapshot.bottomDistance);
      snapshotRef.current = readScrollAnchorSnapshot(node, minBottomThreshold, maxBottomThreshold);
    };
    let frame = 0;
    let settleFrame = 0;
    const scheduleRestore = () => {
      if (frame) {
        return;
      }
      frame = window.requestAnimationFrame(() => {
        frame = 0;
        restoreAnchor();
        settleFrame = window.requestAnimationFrame(() => {
          settleFrame = 0;
          restoreAnchor();
        });
      });
    };
    const handleFocusIn = (event: FocusEvent) => {
      if (!focusSelector || !(event.target instanceof HTMLElement) || !event.target.matches(focusSelector)) {
        return;
      }
      capture();
    };

    if (shouldRestoreAnchor()) {
      capture();
    }
    document.addEventListener("focusin", handleFocusIn);
    window.addEventListener("resize", scheduleRestore);
    window.visualViewport?.addEventListener("resize", scheduleRestore);
    window.visualViewport?.addEventListener("scroll", scheduleRestore);
    const resizeObserver = typeof ResizeObserver === "undefined"
      ? null
      : new ResizeObserver(scheduleRestore);
    resizeObserver?.observe(node);

    return () => {
      document.removeEventListener("focusin", handleFocusIn);
      window.removeEventListener("resize", scheduleRestore);
      window.visualViewport?.removeEventListener("resize", scheduleRestore);
      window.visualViewport?.removeEventListener("scroll", scheduleRestore);
      resizeObserver?.disconnect();
      if (frame) {
        window.cancelAnimationFrame(frame);
      }
      if (settleFrame) {
        window.cancelAnimationFrame(settleFrame);
      }
    };
  }, [
    capture,
    containerRef,
    enabled,
    focusSelector,
    maxBottomThreshold,
    minBottomThreshold,
  ]);

  return capture;
}

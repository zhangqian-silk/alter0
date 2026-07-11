import { memo, useEffect, useId, useRef, useState, type ComponentPropsWithoutRef, type RefObject } from "react";
import type { LegacyShellLanguage } from "../legacyShellCopy";

type ScrollJumpStripProps = {
  scope: "chat" | "chatRuntime";
  namespace?: "scroll" | "chatRuntime";
  language: LegacyShellLanguage;
  containerRef: RefObject<HTMLElement | null>;
  itemSelector: string;
  itemAttribute: string;
  targetOffset?: number;
  watchKey?: string | number | boolean | null;
  suppressNextTarget?: boolean;
  onControlPointerDown?: ComponentPropsWithoutRef<"button">["onPointerDown"];
};

type ScrollJumpState = {
  previousID: string;
  nextID: string;
  showTop: boolean;
  showBottom: boolean;
};

type ScrollJumpCopy = {
  top: string;
  prev: string;
  next: string;
  bottom: string;
};

const SCROLL_JUMP_TOP_THRESHOLD = 180;
const SCROLL_JUMP_BOTTOM_THRESHOLD = 220;
const SCROLL_BOTTOM_ANCHOR_THRESHOLD = 24;

const SCROLL_JUMP_COPY: Record<LegacyShellLanguage, ScrollJumpCopy> = {
  en: {
    top: "Top",
    prev: "Previous",
    next: "Next",
    bottom: "Latest",
  },
  zh: {
    top: "回到顶部",
    prev: "上一条",
    next: "下一条",
    bottom: "回到底部",
  },
};

const EMPTY_SCROLL_JUMP_STATE: ScrollJumpState = {
  previousID: "",
  nextID: "",
  showTop: false,
  showBottom: false,
};

function normalizeSelectionNode(node: Node | null): Node | null {
  if (!node) {
    return null;
  }
  return node.nodeType === Node.TEXT_NODE ? node.parentNode : node;
}

function hasActiveSelectionInsideContainer(container: HTMLElement | null): boolean {
  if (!container || typeof document === "undefined" || typeof document.getSelection !== "function") {
    return false;
  }
  const selection = document.getSelection();
  if (!selection || selection.rangeCount === 0 || selection.isCollapsed) {
    return false;
  }

  const selectionText = typeof selection.toString === "function"
    ? selection.toString().trim()
    : "";
  if (!selectionText) {
    return false;
  }

  const anchorNode = normalizeSelectionNode(selection.anchorNode);
  const focusNode = normalizeSelectionNode(selection.focusNode);
  if ((anchorNode && container.contains(anchorNode)) || (focusNode && container.contains(focusNode))) {
    return true;
  }

  for (let index = 0; index < selection.rangeCount; index += 1) {
    const commonAncestor = normalizeSelectionNode(selection.getRangeAt(index).commonAncestorContainer);
    if (commonAncestor && container.contains(commonAncestor)) {
      return true;
    }
  }
  return false;
}

function isVisibleJumpTarget(node: HTMLElement): boolean {
  if (node.hidden) {
    return false;
  }
  const style = window.getComputedStyle(node);
  if (style.display === "none" || style.visibility === "hidden") {
    return false;
  }
  return true;
}

type ChatRuntimeJumpMeasurement = {
  id: string;
  top: number;
  bottom: number;
};

function measureChatRuntimeJumpEntries(
  container: HTMLElement,
  itemSelector: string,
  itemAttribute: string,
  idPrefix: string,
): ChatRuntimeJumpMeasurement[] {
  return [...container.querySelectorAll<HTMLElement>(itemSelector)]
    .filter((node) => isVisibleJumpTarget(node))
    .map((node, index) => {
      const existing = node.getAttribute(itemAttribute)?.trim();
      const id = existing || `${idPrefix}-${index + 1}`;
      if (!existing) {
        node.setAttribute(itemAttribute, id);
      }
      const top = node.offsetTop;
      const height = Math.max(node.offsetHeight, 0);
      return {
        id,
        top,
        bottom: top + height,
      };
    })
    .filter((entry) => entry.id);
}

function resolveChatRuntimeJumpState(
  container: HTMLElement | null,
  itemSelector: string,
  itemAttribute: string,
  idPrefix: string,
  measurementCacheRef: RefObject<ChatRuntimeJumpMeasurement[] | null>,
  measurementDirtyRef: RefObject<boolean>,
  targetOffset: number,
  suppressNextTarget: boolean,
): ScrollJumpState {
  if (!container) {
    measurementCacheRef.current = null;
    measurementDirtyRef.current = true;
    return EMPTY_SCROLL_JUMP_STATE;
  }

  const scrollTop = Math.max(container.scrollTop, 0);
  const viewportBottom = scrollTop + container.clientHeight;
  const remaining = Math.max(container.scrollHeight - scrollTop - container.clientHeight, 0);
  const measureEntries = () => measureChatRuntimeJumpEntries(container, itemSelector, itemAttribute, idPrefix);
  let entries = !measurementDirtyRef.current && measurementCacheRef.current
    ? measurementCacheRef.current
    : measureEntries();
  const cachedLastBottom = entries[entries.length - 1]?.bottom ?? 0;

  if (!measurementDirtyRef.current && entries.length > 0 && scrollTop > cachedLastBottom) {
    entries = measureEntries();
    measurementCacheRef.current = entries;
  }
  if (measurementDirtyRef.current) {
    measurementCacheRef.current = entries;
    measurementDirtyRef.current = false;
  }
  if (!entries.length) {
    return {
      previousID: "",
      nextID: "",
      showTop: scrollTop > SCROLL_JUMP_TOP_THRESHOLD,
      showBottom: remaining > SCROLL_JUMP_BOTTOM_THRESHOLD,
    };
  }

  let visibleEntries = entries.filter((entry) => entry.bottom > scrollTop && entry.top < viewportBottom);
  if (!visibleEntries.length && !measurementDirtyRef.current) {
    const remeasuredEntries = measureEntries();
    if (remeasuredEntries.length > 0) {
      entries = remeasuredEntries;
      measurementCacheRef.current = remeasuredEntries;
      visibleEntries = remeasuredEntries.filter((entry) => entry.bottom > scrollTop && entry.top < viewportBottom);
    }
  }
  if (!visibleEntries.length) {
    const previousID = [...entries]
      .reverse()
      .find((entry) => entry.bottom <= scrollTop)?.id || "";
    const nextID = suppressNextTarget
      ? ""
      : entries.find((entry) => entry.top >= viewportBottom)?.id || "";
    return {
      previousID,
      nextID,
      showTop: scrollTop > SCROLL_JUMP_TOP_THRESHOLD,
      showBottom: remaining > SCROLL_JUMP_BOTTOM_THRESHOLD,
    };
  }

  const previousID = visibleEntries[0]?.id || "";
  const lastVisibleID = visibleEntries[visibleEntries.length - 1]?.id || "";
  const firstEntryID = entries[0]?.id || "";
  const lastEntryID = entries[entries.length - 1]?.id || "";
  const firstEntryVisible = visibleEntries.some((entry) => entry.id === firstEntryID);
  const lastEntryVisible = visibleEntries.some((entry) => entry.id === lastEntryID);
  const firstEntryTop = entries[0]?.top ?? 0;
  const lastEntryBottom = entries[entries.length - 1]?.bottom ?? 0;
  const hasHiddenContentAbove = firstEntryTop < scrollTop;
  const hasHiddenContentBelow = lastEntryBottom > viewportBottom;
  const firstVisibleID = visibleEntries[0]?.id || "";
  const firstVisibleIndex = firstVisibleID
    ? entries.findIndex((entry) => entry.id === firstVisibleID)
    : -1;
  const lastVisibleIndex = lastVisibleID
    ? entries.findIndex((entry) => entry.id === lastVisibleID)
    : -1;
  let nextID = "";
  let previousTargetID = "";

  if (hasHiddenContentAbove) {
    const firstVisibleTop = visibleEntries[0]?.top ?? scrollTop;
    const firstVisibleAligned = firstVisibleTop >= scrollTop + targetOffset - 2;
    previousTargetID = firstVisibleAligned && firstVisibleIndex > 0
      ? entries[firstVisibleIndex - 1]?.id || ""
      : previousID;
  }

  if (suppressNextTarget) {
    nextID = "";
  } else if (remaining <= SCROLL_BOTTOM_ANCHOR_THRESHOLD) {
    nextID = "";
  } else if (lastEntryVisible) {
    nextID = "";
  } else if (lastVisibleIndex >= 0 && lastVisibleIndex === entries.length - 1) {
    nextID = "";
  } else if (visibleEntries.length > 1) {
    nextID = lastVisibleID;
  } else {
    const visibleID = visibleEntries[0]?.id || "";
    const visibleIndex = entries.findIndex((entry) => entry.id === visibleID);
    nextID = visibleIndex >= 0 ? entries[visibleIndex + 1]?.id || "" : "";
  }

  return {
    previousID: previousTargetID,
    nextID,
    showTop: hasHiddenContentAbove && scrollTop > SCROLL_JUMP_TOP_THRESHOLD,
    showBottom: hasHiddenContentBelow && remaining > SCROLL_JUMP_BOTTOM_THRESHOLD,
  };
}

function scrollContainerToTarget(
  container: HTMLElement | null,
  itemAttribute: string,
  targetID: string,
  targetOffset: number,
) {
  if (!container || !targetID) {
    return;
  }

  const target = container.querySelector<HTMLElement>(`[${itemAttribute}="${targetID}"]`);
  if (!target) {
    return;
  }

  const containerRect = container.getBoundingClientRect();
  const targetRect = target.getBoundingClientRect();
  const top = Math.max(container.scrollTop + targetRect.top - containerRect.top - targetOffset, 0);
  container.scrollTo({ top, behavior: "smooth" });
}

export const ScrollJumpStrip = memo(function ScrollJumpStrip({
  scope,
  namespace = "scroll",
  language,
  containerRef,
  itemSelector,
  itemAttribute,
  targetOffset = 12,
  watchKey = null,
  suppressNextTarget = false,
  onControlPointerDown,
}: ScrollJumpStripProps) {
  const copy = SCROLL_JUMP_COPY[language];
  const idPrefix = useId().replace(/:/g, "");
  const [state, setState] = useState<ScrollJumpState>(EMPTY_SCROLL_JUMP_STATE);
  const [selectionActive, setSelectionActive] = useState(false);
  const measurementCacheRef = useRef<ChatRuntimeJumpMeasurement[] | null>(null);
  const measurementDirtyRef = useRef(true);
  const clusterClassName = namespace === "chatRuntime" ? "chatRuntime-jump-cluster" : "scroll-jump-strip";
  const controlClassName = namespace === "chatRuntime" ? "chatRuntime-jump-control" : "scroll-jump-control";
  const iconClassName = namespace === "chatRuntime" ? "chatRuntime-jump-control-icon" : "scroll-jump-control-icon";
  const topDataAttr = namespace === "chatRuntime" ? "data-chat-runtime-jump-top" : "data-scroll-jump-top";
  const prevDataAttr = namespace === "chatRuntime" ? "data-chat-runtime-jump-prev" : "data-scroll-jump-prev";
  const nextDataAttr = namespace === "chatRuntime" ? "data-chat-runtime-jump-next" : "data-scroll-jump-next";
  const bottomDataAttr = namespace === "chatRuntime" ? "data-chat-runtime-jump-bottom" : "data-scroll-jump-bottom";
  const targetDataAttr = namespace === "chatRuntime" ? "data-chat-runtime-jump-target" : "data-scroll-jump-target";

  useEffect(() => {
    const container = containerRef.current;
    if (!container) {
      setState(EMPTY_SCROLL_JUMP_STATE);
      measurementCacheRef.current = null;
      measurementDirtyRef.current = true;
      return;
    }

    let frame = 0;
    const sync = () => {
      frame = 0;
      setState(
        resolveChatRuntimeJumpState(
          container,
          itemSelector,
          itemAttribute,
          `${scope}-${idPrefix}`,
          measurementCacheRef,
          measurementDirtyRef,
          targetOffset,
          suppressNextTarget,
        ),
      );
    };
    const scheduleSync = () => {
      if (frame) {
        return;
      }
      frame = window.requestAnimationFrame(sync);
    };
    const scheduleDirtySync = () => {
      measurementDirtyRef.current = true;
      scheduleSync();
    };

    scheduleSync();
    container.addEventListener("scroll", scheduleSync, { passive: true });
    window.addEventListener("resize", scheduleDirtySync);
    const observer = new MutationObserver(scheduleDirtySync);
    observer.observe(container, {
      childList: true,
      subtree: true,
      attributes: true,
      attributeFilter: ["hidden", "class", itemAttribute],
    });

    return () => {
      container.removeEventListener("scroll", scheduleSync);
      window.removeEventListener("resize", scheduleDirtySync);
      observer.disconnect();
      if (frame) {
        window.cancelAnimationFrame(frame);
      }
    };
  }, [containerRef, idPrefix, itemAttribute, itemSelector, scope, suppressNextTarget, targetOffset, watchKey]);

  useEffect(() => {
    measurementDirtyRef.current = true;
  }, [itemAttribute, itemSelector, watchKey]);

  useEffect(() => {
    const syncSelectionState = () => {
      setSelectionActive(hasActiveSelectionInsideContainer(containerRef.current));
    };

    syncSelectionState();
    document.addEventListener("selectionchange", syncSelectionState);
    return () => {
      document.removeEventListener("selectionchange", syncSelectionState);
    };
  }, [containerRef]);

  const showTop = !selectionActive && state.showTop;
  const showPrevious = !selectionActive && Boolean(state.previousID);
  const showNext = !selectionActive && Boolean(state.nextID);
  const showBottom = !selectionActive && state.showBottom;

  return (
    <div
      className={clusterClassName}
      data-scroll-jump-scope={namespace === "scroll" ? scope : undefined}
      data-scroll-jump-selection-active={selectionActive ? "true" : "false"}
      aria-label="Turn navigation"
    >
      <button
        className={showTop ? `${controlClassName} ${namespace === "chatRuntime" ? "chatRuntime-jump-top" : "scroll-jump-top"} is-visible` : `${controlClassName} ${namespace === "chatRuntime" ? "chatRuntime-jump-top" : "scroll-jump-top"}`}
        type="button"
        {...{ [topDataAttr]: scope }}
        aria-label={copy.top}
        title={copy.top}
        onPointerDown={onControlPointerDown}
        onClick={() => {
          const container = containerRef.current;
          if (!container) {
            return;
          }
          container.scrollTo({ top: 0, behavior: "smooth" });
        }}
      >
        <span className={iconClassName} aria-hidden="true">↑↑</span>
      </button>
      <button
        className={showPrevious ? `${controlClassName} ${namespace === "chatRuntime" ? "chatRuntime-jump-prev" : "scroll-jump-prev"} is-visible` : `${controlClassName} ${namespace === "chatRuntime" ? "chatRuntime-jump-prev" : "scroll-jump-prev"}`}
        type="button"
        {...{ [prevDataAttr]: scope, [targetDataAttr]: state.previousID }}
        aria-label={copy.prev}
        title={copy.prev}
        onPointerDown={onControlPointerDown}
        onClick={() => {
          scrollContainerToTarget(containerRef.current, itemAttribute, state.previousID, targetOffset);
        }}
      >
        <span className={iconClassName} aria-hidden="true">↑</span>
      </button>
      <button
        className={showNext ? `${controlClassName} ${namespace === "chatRuntime" ? "chatRuntime-jump-next" : "scroll-jump-next"} is-visible` : `${controlClassName} ${namespace === "chatRuntime" ? "chatRuntime-jump-next" : "scroll-jump-next"}`}
        type="button"
        {...{ [nextDataAttr]: scope, [targetDataAttr]: state.nextID }}
        aria-label={copy.next}
        title={copy.next}
        onPointerDown={onControlPointerDown}
        onClick={() => {
          scrollContainerToTarget(containerRef.current, itemAttribute, state.nextID, targetOffset);
        }}
      >
        <span className={iconClassName} aria-hidden="true">↓</span>
      </button>
      <button
        className={showBottom ? `${controlClassName} ${namespace === "chatRuntime" ? "chatRuntime-jump-bottom" : "scroll-jump-bottom"} is-visible` : `${controlClassName} ${namespace === "chatRuntime" ? "chatRuntime-jump-bottom" : "scroll-jump-bottom"}`}
        type="button"
        {...{ [bottomDataAttr]: scope }}
        aria-label={copy.bottom}
        title={copy.bottom}
        onPointerDown={onControlPointerDown}
        onClick={() => {
          const container = containerRef.current;
          if (!container) {
            return;
          }
          container.scrollTo({ top: container.scrollHeight, behavior: "smooth" });
        }}
      >
        <span className={iconClassName} aria-hidden="true">↓↓</span>
      </button>
    </div>
  );
});

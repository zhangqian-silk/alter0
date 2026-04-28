import { memo, useEffect, useId, useRef, useState, type ComponentPropsWithoutRef, type RefObject } from "react";
import type { LegacyShellLanguage } from "../legacyShellCopy";

type ScrollJumpStripProps = {
  scope: "chat" | "agent" | "terminal";
  namespace?: "scroll" | "terminal";
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

type TerminalJumpMeasurement = {
  id: string;
  top: number;
  bottom: number;
};

function measureTerminalJumpEntries(
  container: HTMLElement,
  itemSelector: string,
  itemAttribute: string,
  idPrefix: string,
): TerminalJumpMeasurement[] {
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

function resolveTerminalJumpState(
  container: HTMLElement | null,
  itemSelector: string,
  itemAttribute: string,
  idPrefix: string,
  measurementCacheRef: RefObject<TerminalJumpMeasurement[] | null>,
  measurementDirtyRef: RefObject<boolean>,
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
  const measureEntries = () => measureTerminalJumpEntries(container, itemSelector, itemAttribute, idPrefix);
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
    return {
      previousID: "",
      nextID: "",
      showTop: scrollTop > SCROLL_JUMP_TOP_THRESHOLD,
      showBottom: remaining > SCROLL_JUMP_BOTTOM_THRESHOLD,
    };
  }

  const previousID = visibleEntries[0]?.id || "";
  const lastVisibleID = visibleEntries[visibleEntries.length - 1]?.id || "";
  const lastVisibleIndex = lastVisibleID
    ? entries.findIndex((entry) => entry.id === lastVisibleID)
    : -1;
  let nextID = "";

  if (suppressNextTarget) {
    nextID = "";
  } else if (remaining <= SCROLL_BOTTOM_ANCHOR_THRESHOLD) {
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
    previousID,
    nextID,
    showTop: scrollTop > SCROLL_JUMP_TOP_THRESHOLD,
    showBottom: remaining > SCROLL_JUMP_BOTTOM_THRESHOLD,
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
  const measurementCacheRef = useRef<TerminalJumpMeasurement[] | null>(null);
  const measurementDirtyRef = useRef(true);
  const clusterClassName = namespace === "terminal" ? "terminal-jump-cluster" : "scroll-jump-strip";
  const controlClassName = namespace === "terminal" ? "terminal-jump-control" : "scroll-jump-control";
  const iconClassName = namespace === "terminal" ? "terminal-jump-control-icon" : "scroll-jump-control-icon";
  const topDataAttr = namespace === "terminal" ? "data-terminal-jump-top" : "data-scroll-jump-top";
  const prevDataAttr = namespace === "terminal" ? "data-terminal-jump-prev" : "data-scroll-jump-prev";
  const nextDataAttr = namespace === "terminal" ? "data-terminal-jump-next" : "data-scroll-jump-next";
  const bottomDataAttr = namespace === "terminal" ? "data-terminal-jump-bottom" : "data-scroll-jump-bottom";
  const targetDataAttr = namespace === "terminal" ? "data-terminal-jump-target" : "data-scroll-jump-target";

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
        resolveTerminalJumpState(
          container,
          itemSelector,
          itemAttribute,
          `${scope}-${idPrefix}`,
          measurementCacheRef,
          measurementDirtyRef,
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

    scheduleSync();
    container.addEventListener("scroll", scheduleSync, { passive: true });
    window.addEventListener("resize", scheduleSync);
    const observer = new MutationObserver(scheduleSync);
    observer.observe(container, {
      childList: true,
      subtree: true,
      attributes: true,
      attributeFilter: ["hidden", "class", itemAttribute],
    });

    return () => {
      container.removeEventListener("scroll", scheduleSync);
      window.removeEventListener("resize", scheduleSync);
      observer.disconnect();
      if (frame) {
        window.cancelAnimationFrame(frame);
      }
    };
  }, [containerRef, idPrefix, itemAttribute, itemSelector, scope, suppressNextTarget, watchKey]);

  useEffect(() => {
    measurementDirtyRef.current = true;
  }, [itemAttribute, itemSelector, watchKey]);

  return (
    <div
      className={clusterClassName}
      data-scroll-jump-scope={namespace === "scroll" ? scope : undefined}
      aria-label="Turn navigation"
    >
      <button
        className={state.showTop ? `${controlClassName} ${namespace === "terminal" ? "terminal-jump-top" : "scroll-jump-top"} is-visible` : `${controlClassName} ${namespace === "terminal" ? "terminal-jump-top" : "scroll-jump-top"}`}
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
        className={state.previousID ? `${controlClassName} ${namespace === "terminal" ? "terminal-jump-prev" : "scroll-jump-prev"} is-visible` : `${controlClassName} ${namespace === "terminal" ? "terminal-jump-prev" : "scroll-jump-prev"}`}
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
        className={state.nextID ? `${controlClassName} ${namespace === "terminal" ? "terminal-jump-next" : "scroll-jump-next"} is-visible` : `${controlClassName} ${namespace === "terminal" ? "terminal-jump-next" : "scroll-jump-next"}`}
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
        className={state.showBottom ? `${controlClassName} ${namespace === "terminal" ? "terminal-jump-bottom" : "scroll-jump-bottom"} is-visible` : `${controlClassName} ${namespace === "terminal" ? "terminal-jump-bottom" : "scroll-jump-bottom"}`}
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

import { memo, useEffect, useId, useState, type RefObject } from "react";
import type { LegacyShellLanguage } from "../legacyShellCopy";

type ScrollJumpStripProps = {
  scope: "chat" | "route";
  language: LegacyShellLanguage;
  containerRef: RefObject<HTMLElement | null>;
  itemSelector: string;
  itemAttribute: string;
  targetOffset?: number;
};

type ScrollJumpEntry = {
  id: string;
  top: number;
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

const SCROLL_JUMP_COPY: Record<LegacyShellLanguage, ScrollJumpCopy> = {
  en: {
    top: "Top",
    prev: "Prev",
    next: "Next",
    bottom: "Bottom",
  },
  zh: {
    top: "顶部",
    prev: "上一个",
    next: "下一个",
    bottom: "底部",
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

function collectScrollJumpEntries(
  container: HTMLElement,
  itemSelector: string,
  itemAttribute: string,
  idPrefix: string,
): ScrollJumpEntry[] {
  const containerRect = container.getBoundingClientRect();

  return [...container.querySelectorAll<HTMLElement>(itemSelector)]
    .filter((node) => isVisibleJumpTarget(node))
    .map((node, index) => {
      const existing = node.getAttribute(itemAttribute)?.trim();
      const id = existing || `${idPrefix}-${index + 1}`;
      if (!existing) {
        node.setAttribute(itemAttribute, id);
      }

      return {
        id,
        top: Math.max(
          container.scrollTop + node.getBoundingClientRect().top - containerRect.top,
          0,
        ),
      };
    });
}

function resolveScrollJumpState(
  container: HTMLElement | null,
  itemSelector: string,
  itemAttribute: string,
  idPrefix: string,
): ScrollJumpState {
  if (!container) {
    return EMPTY_SCROLL_JUMP_STATE;
  }

  const entries = collectScrollJumpEntries(container, itemSelector, itemAttribute, idPrefix);
  const scrollTop = Math.max(container.scrollTop, 0);
  const remaining = Math.max(container.scrollHeight - scrollTop - container.clientHeight, 0);

  if (!entries.length) {
    return {
      previousID: "",
      nextID: "",
      showTop: scrollTop > SCROLL_JUMP_TOP_THRESHOLD,
      showBottom: remaining > SCROLL_JUMP_BOTTOM_THRESHOLD,
    };
  }

  const viewportAnchor = scrollTop + 24;
  let currentIndex = 0;

  for (let index = 0; index < entries.length; index += 1) {
    const current = entries[index];
    const next = entries[index + 1];
    if (viewportAnchor < current.top) {
      currentIndex = Math.max(index - 1, 0);
      break;
    }
    currentIndex = index;
    if (!next || viewportAnchor < next.top) {
      break;
    }
  }

  return {
    previousID: currentIndex > 0 ? entries[currentIndex - 1]?.id || "" : "",
    nextID: currentIndex < entries.length - 1 ? entries[currentIndex + 1]?.id || "" : "",
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
  language,
  containerRef,
  itemSelector,
  itemAttribute,
  targetOffset = 12,
}: ScrollJumpStripProps) {
  const copy = SCROLL_JUMP_COPY[language];
  const idPrefix = useId().replace(/:/g, "");
  const [state, setState] = useState<ScrollJumpState>(EMPTY_SCROLL_JUMP_STATE);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) {
      setState(EMPTY_SCROLL_JUMP_STATE);
      return;
    }

    let frame = 0;
    const sync = () => {
      frame = 0;
      setState(resolveScrollJumpState(container, itemSelector, itemAttribute, `${scope}-${idPrefix}`));
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
  }, [containerRef, idPrefix, itemAttribute, itemSelector, scope]);

  return (
    <div className="scroll-jump-strip" data-scroll-jump-scope={scope}>
      <button
        className={state.showTop ? "pane-action is-visible" : "pane-action"}
        type="button"
        data-scroll-jump-top={scope}
        onClick={() => {
          const container = containerRef.current;
          if (!container) {
            return;
          }
          container.scrollTo({ top: 0, behavior: "smooth" });
        }}
      >
        {copy.top}
      </button>
      <button
        className={state.previousID ? "pane-action is-visible" : "pane-action"}
        type="button"
        data-scroll-jump-prev={scope}
        data-scroll-jump-target={state.previousID}
        onClick={() => {
          scrollContainerToTarget(containerRef.current, itemAttribute, state.previousID, targetOffset);
        }}
      >
        {copy.prev}
      </button>
      <button
        className={state.nextID ? "pane-action is-visible" : "pane-action"}
        type="button"
        data-scroll-jump-next={scope}
        data-scroll-jump-target={state.nextID}
        onClick={() => {
          scrollContainerToTarget(containerRef.current, itemAttribute, state.nextID, targetOffset);
        }}
      >
        {copy.next}
      </button>
      <button
        className={state.showBottom ? "pane-action is-visible" : "pane-action"}
        type="button"
        data-scroll-jump-bottom={scope}
        onClick={() => {
          const container = containerRef.current;
          if (!container) {
            return;
          }
          container.scrollTo({ top: container.scrollHeight, behavior: "smooth" });
        }}
      >
        {copy.bottom}
      </button>
    </div>
  );
});

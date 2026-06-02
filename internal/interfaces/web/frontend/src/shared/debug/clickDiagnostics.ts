export const CLICK_DIAGNOSTICS_STORAGE_KEY = "alter0.debug.clicks";

const ENABLED_VALUES = new Set(["1", "true", "on", "yes"]);
const DISABLED_VALUES = new Set(["0", "false", "off", "no"]);
const CLICK_EVENT_TYPES = ["pointerdown", "pointerup", "touchstart", "touchend", "click"] as const;
const BLOCKING_LAYER_SELECTOR = [
  ".mobile-backdrop",
  ".runtime-workspace-session-pane-backdrop",
  ".workspace-details-layer",
  ".workspace-details-backdrop",
  ".runtime-image-preview-backdrop",
  "[data-runtime-details-backdrop='true']",
].join(",");

type ClickDiagnosticsLogger = Pick<Console, "debug" | "warn">;

type ClickDiagnosticsOptions = {
  document?: Document;
  window?: Window & typeof globalThis;
  logger?: ClickDiagnosticsLogger;
};

type EventPoint = {
  x: number;
  y: number;
};

function normalizeToggleValue(value: string | null): boolean | null {
  if (value === null) {
    return null;
  }
  const normalized = value.trim().toLowerCase();
  if (ENABLED_VALUES.has(normalized)) {
    return true;
  }
  if (DISABLED_VALUES.has(normalized)) {
    return false;
  }
  return null;
}

export function isClickDiagnosticsEnabled(
  search = window.location.search,
  storage: Storage | null = window.localStorage,
) {
  const params = new URLSearchParams(search);
  const queryValue = normalizeToggleValue(params.get("debug_clicks"))
    ?? normalizeToggleValue(params.get("debugClicks"));
  if (queryValue !== null) {
    return queryValue;
  }

  try {
    return normalizeToggleValue(storage?.getItem(CLICK_DIAGNOSTICS_STORAGE_KEY) ?? null) === true;
  } catch {
    return false;
  }
}

function describeElement(target: EventTarget | null | undefined): string {
  if (!(target instanceof Element)) {
    if (target instanceof Window) {
      return "window";
    }
    return target ? target.constructor.name : "null";
  }

  const id = target.id ? `#${target.id}` : "";
  const className = typeof target.className === "string"
    ? target.className.trim().replace(/\s+/g, ".")
    : "";
  const classes = className ? `.${className}` : "";
  const role = target.getAttribute("role");
  const ariaLabel = target.getAttribute("aria-label");
  const disabled = target instanceof HTMLButtonElement && target.disabled ? "[disabled]" : "";
  const roleLabel = role ? `[role=${role}]` : "";
  const label = ariaLabel ? `[aria-label="${ariaLabel}"]` : "";
  return `${target.tagName.toLowerCase()}${id}${classes}${roleLabel}${label}${disabled}`;
}

function eventPoint(event: Event): EventPoint | null {
  const touchEvent = event as TouchEvent;
  const touch = touchEvent.touches?.[0] ?? touchEvent.changedTouches?.[0];
  if (touch) {
    return { x: touch.clientX, y: touch.clientY };
  }

  const mouseEvent = event as MouseEvent;
  if (typeof mouseEvent.clientX === "number" && typeof mouseEvent.clientY === "number") {
    return { x: mouseEvent.clientX, y: mouseEvent.clientY };
  }
  return null;
}

function isBlockingLayer(element: Element, win: Window & typeof globalThis) {
  const style = win.getComputedStyle(element);
  if (style.display === "none" || style.visibility === "hidden" || style.pointerEvents === "none") {
    return false;
  }
  if (style.opacity === "0") {
    return false;
  }
  return true;
}

function blockingLayers(doc: Document, win: Window & typeof globalThis) {
  return Array.from(doc.querySelectorAll(BLOCKING_LAYER_SELECTOR))
    .filter((element) => isBlockingLayer(element, win))
    .map(describeElement);
}

function eventPath(event: Event) {
  if (!event.composedPath) {
    return [];
  }
  return event.composedPath().slice(0, 10).map(describeElement);
}

export function installClickDiagnostics(options: ClickDiagnosticsOptions = {}) {
  const doc = options.document ?? document;
  const win = options.window ?? window;
  const logger = options.logger ?? console;

  const handleEvent = (event: Event) => {
    const point = eventPoint(event);
    win.setTimeout(() => {
      const topElement = point && doc.elementFromPoint
        ? doc.elementFromPoint(point.x, point.y)
        : null;
      const shell = doc.querySelector(".app-shell");
      const targetElement = event.target instanceof Element ? event.target : null;
      logger.debug("[alter0:click]", {
        type: event.type,
        point,
        target: describeElement(event.target),
        topElement: describeElement(topElement),
        activeElement: describeElement(doc.activeElement),
        defaultPrevented: event.defaultPrevented,
        cancelable: event.cancelable,
        disabled: targetElement instanceof HTMLButtonElement ? targetElement.disabled : undefined,
        shellClassName: shell?.className || "",
        blockingLayers: blockingLayers(doc, win),
        path: eventPath(event),
      });
    }, 0);
  };

  for (const type of CLICK_EVENT_TYPES) {
    doc.addEventListener(type, handleEvent, true);
  }

  let observer: PerformanceObserver | null = null;
  if (typeof win.PerformanceObserver === "function") {
    try {
      observer = new win.PerformanceObserver((list) => {
        for (const entry of list.getEntries()) {
          logger.warn("[alter0:longtask]", {
            duration: Math.round(entry.duration),
            name: entry.name,
            startTime: Math.round(entry.startTime),
          });
        }
      });
      observer.observe({ entryTypes: ["longtask"] });
    } catch {
      observer = null;
    }
  }

  return () => {
    for (const type of CLICK_EVENT_TYPES) {
      doc.removeEventListener(type, handleEvent, true);
    }
    observer?.disconnect();
  };
}

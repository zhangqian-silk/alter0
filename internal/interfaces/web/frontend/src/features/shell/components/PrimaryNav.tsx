import { useEffect, useRef, useState } from "react";
import { LEGACY_SHELL_IDS } from "../legacyDomContract";
import { NAV_GROUPS, toI18nKey } from "../legacyShellConfig";
import { getLegacyShellCopy, type LegacyShellLanguage } from "../legacyShellCopy";
import { isLegacyShellMobileViewport } from "../legacyShellState";
import { NavIcon } from "./NavIcon";

const NAV_TOOLTIP_SHOW_DELAY = 90;
const NAV_TOOLTIP_HIDE_DELAY = 40;
const NAV_TOOLTIP_OFFSET = 12;

type NavTooltipTarget =
  | { kind: "collapse" }
  | { kind: "route"; route: string };

type NavTooltipState = {
  target: NavTooltipTarget;
  text: string;
  top: number;
  left: number;
  visible: boolean;
};

function isSameNavTooltipTarget(left: NavTooltipTarget, right: NavTooltipTarget): boolean {
  if (left.kind !== right.kind) {
    return false;
  }
  if (left.kind === "collapse" && right.kind === "collapse") {
    return true;
  }
  return left.kind === "route" && right.kind === "route" && left.route === right.route;
}

type PrimaryNavProps = {
  currentRoute: string;
  language: LegacyShellLanguage;
  navCollapsed: boolean;
  onNavigate: (route: string) => void;
  onToggleLanguage: () => void;
  onToggleNavCollapsed: () => void;
};

export function PrimaryNav({
  currentRoute,
  language,
  navCollapsed,
  onNavigate,
  onToggleLanguage,
  onToggleNavCollapsed,
}: PrimaryNavProps) {
  const copy = getLegacyShellCopy(language);
  const navToggleLabel = navCollapsed ? copy.navExpandLabel : copy.navCollapseLabel;
  const collapseButtonRef = useRef<HTMLButtonElement | null>(null);
  const routeButtonRefs = useRef<Record<string, HTMLButtonElement | null>>({});
  const tooltipRef = useRef<HTMLDivElement | null>(null);
  const showTimerRef = useRef(0);
  const hideTimerRef = useRef(0);
  const [tooltip, setTooltip] = useState<NavTooltipState | null>(null);
  const tooltipId = "legacy-shell-nav-tooltip";

  const clearShowTimer = () => {
    if (!showTimerRef.current) {
      return;
    }
    window.clearTimeout(showTimerRef.current);
    showTimerRef.current = 0;
  };

  const clearHideTimer = () => {
    if (!hideTimerRef.current) {
      return;
    }
    window.clearTimeout(hideTimerRef.current);
    hideTimerRef.current = 0;
  };

  const resolveTooltipText = (target: NavTooltipTarget): string => {
    if (target.kind === "collapse") {
      return navToggleLabel;
    }
    return copy.routes[target.route] ?? target.route;
  };

  const resolveTooltipElement = (target: NavTooltipTarget): HTMLButtonElement | null => {
    if (target.kind === "collapse") {
      return collapseButtonRef.current;
    }
    return routeButtonRefs.current[target.route] ?? null;
  };

  const shouldShowTooltip = (target: NavTooltipTarget | null): boolean => {
    if (!target || isLegacyShellMobileViewport()) {
      return false;
    }
    return target.kind === "collapse" || navCollapsed;
  };

  const computeTooltipState = (target: NavTooltipTarget): NavTooltipState | null => {
    if (!shouldShowTooltip(target)) {
      return null;
    }

    const element = resolveTooltipElement(target);
    const text = resolveTooltipText(target).trim();
    if (!element || !text) {
      return null;
    }

    const rect = element.getBoundingClientRect();
    const viewportMargin = 8;
    const tooltipWidth = tooltipRef.current?.offsetWidth ?? 0;
    const top = Math.min(
      Math.max(rect.top + (rect.height / 2), viewportMargin),
      window.innerHeight - viewportMargin,
    );
    const maxLeft = Math.max(viewportMargin, window.innerWidth - tooltipWidth - viewportMargin);
    const left = Math.min(
      Math.max(rect.right + NAV_TOOLTIP_OFFSET, viewportMargin),
      maxLeft,
    );

    return {
      target,
      text,
      top,
      left,
      visible: true,
    };
  };

  const syncTooltipState = (target: NavTooltipTarget) => {
    const next = computeTooltipState(target);
    if (!next) {
      setTooltip((current) => (current ? { ...current, visible: false } : null));
      return;
    }
    setTooltip((current) => {
      if (
        current &&
        isSameNavTooltipTarget(current.target, next.target) &&
        current.text === next.text &&
        current.top === next.top &&
        current.left === next.left &&
        current.visible === next.visible
      ) {
        return current;
      }
      return next;
    });
  };

  const hideTooltip = (immediate = false) => {
    clearShowTimer();

    const close = () => {
      hideTimerRef.current = 0;
      setTooltip((current) => (current ? { ...current, visible: false } : null));
    };

    if (immediate) {
      clearHideTimer();
      close();
      return;
    }

    clearHideTimer();
    hideTimerRef.current = window.setTimeout(close, NAV_TOOLTIP_HIDE_DELAY);
  };

  const queueTooltip = (target: NavTooltipTarget, immediate = false) => {
    if (!shouldShowTooltip(target)) {
      hideTooltip(true);
      return;
    }

    clearHideTimer();
    if (immediate) {
      clearShowTimer();
      syncTooltipState(target);
      return;
    }

    clearShowTimer();
    showTimerRef.current = window.setTimeout(() => {
      showTimerRef.current = 0;
      syncTooltipState(target);
    }, NAV_TOOLTIP_SHOW_DELAY);
  };

  useEffect(() => {
    return () => {
      clearShowTimer();
      clearHideTimer();
    };
  }, []);

  useEffect(() => {
    if (!tooltip) {
      return;
    }
    if (!shouldShowTooltip(tooltip.target)) {
      hideTooltip(true);
      return;
    }
    syncTooltipState(tooltip.target);
  }, [currentRoute, language, navCollapsed]);

  useEffect(() => {
    if (!tooltip?.visible) {
      return;
    }

    const syncPosition = () => {
      if (!shouldShowTooltip(tooltip.target)) {
        hideTooltip(true);
        return;
      }
      syncTooltipState(tooltip.target);
    };

    const frame = window.requestAnimationFrame(syncPosition);
    window.addEventListener("resize", syncPosition);
    window.addEventListener("scroll", syncPosition, true);

    return () => {
      window.cancelAnimationFrame(frame);
      window.removeEventListener("resize", syncPosition);
      window.removeEventListener("scroll", syncPosition, true);
    };
  }, [tooltip]);

  return (
    <aside className="primary-nav">
      <div className="brand">
        <div className="brand-copy">
          <strong>Alter0</strong>
        </div>
        <button
          className="nav-collapse"
          id={LEGACY_SHELL_IDS.navCollapseButton}
          type="button"
          aria-label={navToggleLabel}
          aria-expanded={navCollapsed ? "false" : "true"}
          aria-describedby={tooltip?.visible && tooltip.target.kind === "collapse" ? tooltipId : undefined}
          ref={collapseButtonRef}
          onMouseEnter={() => queueTooltip({ kind: "collapse" })}
          onMouseLeave={() => hideTooltip()}
          onFocus={() => queueTooltip({ kind: "collapse" }, true)}
          onBlur={() => hideTooltip(true)}
          onPointerDown={() => hideTooltip(true)}
          onClick={onToggleNavCollapsed}
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <polyline points="15 18 9 12 15 6"></polyline>
          </svg>
        </button>
      </div>

      <nav className="menu" aria-label={copy.primaryNavLabel}>
        {NAV_GROUPS.map((group) => (
          <section
            key={group.heading}
            className={group.bottom ? "menu-group menu-group-bottom" : "menu-group"}
          >
            <h2 data-i18n={`nav.${toI18nKey(group.heading)}`}>{copy.headings[group.heading] ?? group.heading}</h2>
            {group.items.map((item) => (
              <button
                key={item.route}
                className={item.route === currentRoute ? "menu-item active" : "menu-item"}
                type="button"
                data-route={item.route}
                data-abbr={item.abbr}
                aria-label={copy.routes[item.route] ?? item.label}
                aria-describedby={
                  tooltip?.visible &&
                  tooltip.target.kind === "route" &&
                  tooltip.target.route === item.route
                    ? tooltipId
                    : undefined
                }
                ref={(node) => {
                  routeButtonRefs.current[item.route] = node;
                }}
                onMouseEnter={() => queueTooltip({ kind: "route", route: item.route })}
                onMouseLeave={() => hideTooltip()}
                onFocus={() => queueTooltip({ kind: "route", route: item.route }, true)}
                onBlur={() => hideTooltip(true)}
                onPointerDown={() => hideTooltip(true)}
                onClick={() => onNavigate(item.route)}
              >
                <span className="menu-icon" aria-hidden="true">
                  <NavIcon icon={item.icon} />
                </span>
                <span className="menu-label" data-i18n={`nav.${toI18nKey(item.route)}`}>
                  {copy.routes[item.route] ?? item.label}
                </span>
              </button>
            ))}
          </section>
        ))}
      </nav>

      <div className="nav-locale">
        <button
          className="locale nav-locale-button"
          type="button"
          aria-label={copy.localeAriaLabel}
          data-short-lang={copy.localeShort}
          onClick={onToggleLanguage}
        >
          {copy.localeButton}
        </button>
      </div>
      {tooltip ? (
        <div
          ref={tooltipRef}
          className={tooltip.visible ? "nav-tooltip visible" : "nav-tooltip"}
          id={tooltipId}
          role="tooltip"
          aria-hidden={tooltip.visible ? "false" : "true"}
          style={{
            top: `${tooltip.top}px`,
            left: `${tooltip.left}px`,
          }}
        >
          {tooltip.text}
        </div>
      ) : null}
    </aside>
  );
}

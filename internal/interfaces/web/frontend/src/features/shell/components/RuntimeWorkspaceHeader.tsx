import {
  useEffect,
  useId,
  useLayoutEffect,
  useRef,
  useState,
  type ComponentPropsWithoutRef,
  type CSSProperties,
  type ReactNode,
} from "react";
import { createPortal } from "react-dom";

type RuntimeWorkspaceHeaderStatusTone = "ready" | "busy" | "failed" | "interrupted" | "exited";

type RuntimeWorkspaceHeaderProps = {
  title: string;
  statusLabel: string;
  statusTone: RuntimeWorkspaceHeaderStatusTone;
  detailsLabel: string;
  detailsOpen: boolean;
  onToggleDetails: () => void;
  detailsDisabled?: boolean;
  mobileEmpty?: boolean;
  detailsContent?: ReactNode;
  headerProps?: ComponentPropsWithoutRef<"header">;
  statusButtonProps?: Omit<ComponentPropsWithoutRef<"button">, "type" | "children" | "disabled">;
  detailsButtonProps?: Omit<ComponentPropsWithoutRef<"button">, "type" | "children" | "aria-expanded" | "aria-controls" | "disabled" | "onClick">;
  detailsPanelProps?: ComponentPropsWithoutRef<"section">;
};

function joinClassNames(...values: Array<string | undefined>) {
  return values.filter(Boolean).join(" ");
}

export function RuntimeWorkspaceHeader({
  title,
  statusLabel,
  statusTone,
  detailsLabel,
  detailsOpen,
  onToggleDetails,
  detailsDisabled = false,
  mobileEmpty = false,
  detailsContent,
  headerProps,
  statusButtonProps,
  detailsButtonProps,
  detailsPanelProps,
}: RuntimeWorkspaceHeaderProps) {
  const detailsID = useId();
  const headerRef = useRef<HTMLElement | null>(null);
  const [detailsPanelStyle, setDetailsPanelStyle] = useState<CSSProperties>({});
  const {
    className: headerClassName,
    ...headerRestProps
  } = headerProps || {};
  const {
    className: detailsPanelClassName,
    style: detailsPanelInlineStyle,
    ...detailsPanelRestProps
  } = detailsPanelProps || {};
  const {
    className: statusButtonClassName,
    ...statusButtonRestProps
  } = statusButtonProps || {};
  const {
    className: detailsButtonClassName,
    ...detailsButtonRestProps
  } = detailsButtonProps || {};

  useEffect(() => {
    if (!detailsOpen) {
      return;
    }
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onToggleDetails();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [detailsOpen, onToggleDetails]);

  useLayoutEffect(() => {
    if (!detailsOpen || !detailsContent) {
      return;
    }

    const syncDetailsPanelPosition = () => {
      const headerNode = headerRef.current;
      if (!headerNode) {
        return;
      }

      const rect = headerNode.getBoundingClientRect();
      const viewportWidth = window.innerWidth || document.documentElement.clientWidth || 0;
      const viewportHeight = window.innerHeight || document.documentElement.clientHeight || 0;
      const availableWidth = Math.max(320, viewportWidth - 24);
      const panelWidth = Math.min(560, availableWidth);
      const fallbackRight = Math.max(12, viewportWidth - 12);
      const rectRight = rect.right > 0 ? rect.right : fallbackRight;
      const left = Math.max(12, Math.min(rectRight - panelWidth, viewportWidth - panelWidth - 12));
      const fallbackTop = 72;
      const top = Math.max(
        12,
        Math.min(rect.bottom > 0 ? rect.bottom + 8 : fallbackTop, Math.max(12, viewportHeight - 120)),
      );

      setDetailsPanelStyle({
        top: `${top}px`,
        left: `${left}px`,
        width: `${panelWidth}px`,
      });
    };

    syncDetailsPanelPosition();

    const resizeObserver = typeof ResizeObserver === "undefined"
      ? null
      : new ResizeObserver(() => syncDetailsPanelPosition());
    if (headerRef.current) {
      resizeObserver?.observe(headerRef.current);
    }
    window.addEventListener("resize", syncDetailsPanelPosition);
    window.addEventListener("scroll", syncDetailsPanelPosition, true);
    window.visualViewport?.addEventListener("resize", syncDetailsPanelPosition);
    window.visualViewport?.addEventListener("scroll", syncDetailsPanelPosition);
    return () => {
      resizeObserver?.disconnect();
      window.removeEventListener("resize", syncDetailsPanelPosition);
      window.removeEventListener("scroll", syncDetailsPanelPosition, true);
      window.visualViewport?.removeEventListener("resize", syncDetailsPanelPosition);
      window.visualViewport?.removeEventListener("scroll", syncDetailsPanelPosition);
    };
  }, [detailsContent, detailsOpen]);

  const detailsPanel = detailsOpen && detailsContent && typeof document !== "undefined"
    ? createPortal(
      <div className="workspace-details-layer">
        <button
          type="button"
          className="workspace-details-backdrop"
          aria-label={detailsLabel}
          data-runtime-details-backdrop="true"
          onClick={onToggleDetails}
        ></button>
        <section
          id={detailsID}
          className={joinClassNames("runtime-workspace-details-panel workspace-details-panel", detailsPanelClassName)}
          style={{
            ...detailsPanelInlineStyle,
            ...detailsPanelStyle,
          }}
          {...detailsPanelRestProps}
        >
          {detailsContent}
        </section>
      </div>,
      document.body,
    )
    : null;

  return (
    <header
      ref={headerRef}
      className={joinClassNames(
        "runtime-workspace-head is-compact is-sticky",
        mobileEmpty ? "is-mobile-empty" : undefined,
        headerClassName,
      )}
      data-runtime-workspace-header="true"
      {...headerRestProps}
    >
      <div
        className={joinClassNames(
          "runtime-workspace-row runtime-workspace-title-row is-compact",
          mobileEmpty ? "is-mobile-empty" : undefined,
        )}
      >
        <div
          className={joinClassNames(
            "runtime-workspace-copy is-compact",
            mobileEmpty ? "is-mobile-empty" : undefined,
          )}
        >
          <h4 title={title}>{title}</h4>
        </div>
        <div
          className={joinClassNames(
            "runtime-workspace-actions workspace-header-actions",
            mobileEmpty ? "is-mobile-empty" : undefined,
          )}
        >
          <button
            className={joinClassNames(
              "runtime-workspace-button",
              "is-quiet",
              "workspace-header-status",
              `is-${statusTone}`,
              statusButtonClassName,
            )}
            type="button"
            disabled
            {...statusButtonRestProps}
          >
            {statusLabel}
          </button>
          <button
            className={joinClassNames(
              "runtime-workspace-button",
              "workspace-header-details",
              detailsOpen ? "is-active" : undefined,
              detailsButtonClassName,
            )}
            type="button"
            aria-expanded={detailsOpen}
            aria-controls={detailsID}
            disabled={detailsDisabled}
            onClick={onToggleDetails}
            {...detailsButtonRestProps}
          >
            {detailsLabel}
          </button>
        </div>
      </div>
      {detailsPanel}
    </header>
  );
}

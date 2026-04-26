import { useEffect, useId, useState, type ComponentPropsWithoutRef, type ReactNode, type Ref } from "react";
import { RuntimeWorkspaceFrame } from "./RuntimeWorkspaceFrame";

function joinClassNames(...values: Array<string | undefined>) {
  return values.filter(Boolean).join(" ");
}

function RuntimeWorkspaceMobileLauncherIcon() {
  return (
    <svg viewBox="0 0 20 20" fill="none" focusable="false" aria-hidden="true">
      <rect x="3" y="3" width="5" height="5" rx="1.4" stroke="currentColor" strokeWidth="1.6" />
      <rect x="12" y="3" width="5" height="5" rx="1.4" stroke="currentColor" strokeWidth="1.6" />
      <rect x="3" y="12" width="5" height="5" rx="1.4" stroke="currentColor" strokeWidth="1.6" />
      <rect x="12" y="12" width="5" height="5" rx="1.4" stroke="currentColor" strokeWidth="1.6" />
    </svg>
  );
}

type RuntimeWorkspaceShellProps = {
  rootClassName?: string;
  rootProps?: ComponentPropsWithoutRef<"section">;
  sessionPaneClassName?: string;
  sessionPaneProps?: ComponentPropsWithoutRef<"aside">;
  sessionPaneBackdrop: {
    className?: string;
    ariaLabel: string;
    onClick: () => void;
    buttonProps?: Omit<ComponentPropsWithoutRef<"button">, "type" | "className" | "aria-label" | "onClick">;
  };
  sessionPaneShellClassName?: string;
  sessionPaneHeaderClassName?: string;
  sessionPaneCopyClassName?: string;
  sessionPaneActionsClassName?: string;
  sessionPanePrimaryActionClassName?: string;
  sessionPaneSecondaryActionClassName?: string;
  sessionPanePrimaryActionProps?: Omit<ComponentPropsWithoutRef<"button">, "type" | "className" | "children" | "onClick">;
  sessionPaneSecondaryActionProps?: Omit<ComponentPropsWithoutRef<"button">, "type" | "className" | "children" | "onClick">;
  sessionPaneTitle: string;
  sessionPaneCountLabel: string;
  sessionPanePrimaryActionLabel: string;
  onSessionPanePrimaryAction: () => void;
  sessionPaneSecondaryActionLabel?: string;
  onSessionPaneSecondaryAction?: () => void;
  sessionPaneBody: ReactNode;
  workspaceClassName?: string;
  workspaceProps?: ComponentPropsWithoutRef<"section">;
  workspaceBodyClassName?: string;
  workspaceBodyRef?: Ref<HTMLDivElement>;
  mobileHeaderPlacement?: "leading" | "body";
  mobileHeaderClassName?: string;
  mobileHeaderProps?: Omit<ComponentPropsWithoutRef<"header">, "className" | "children">;
  mobileLauncherLabel?: string;
  mobileNavButtonClassName?: string;
  mobileNavButtonLabel?: string;
  mobileNavButtonProps?: Omit<ComponentPropsWithoutRef<"button">, "type" | "className" | "children" | "onClick">;
  onMobileNav?: () => void;
  mobileSessionButtonClassName?: string;
  mobileSessionButtonLabel?: string;
  mobileSessionButtonProps?: Omit<ComponentPropsWithoutRef<"button">, "type" | "className" | "children" | "onClick">;
  onMobileSession?: () => void;
  mobilePrimaryButtonClassName?: string;
  mobilePrimaryButtonLabel?: string;
  mobilePrimaryButtonProps?: Omit<ComponentPropsWithoutRef<"button">, "type" | "className" | "children" | "onClick">;
  onMobilePrimary?: () => void;
  mobileHeaderActionsClassName?: string;
  workspaceHeader?: ReactNode;
  workspaceContent: ReactNode;
  workspaceFooter?: ReactNode;
};

export function RuntimeWorkspaceShell({
  rootClassName,
  rootProps,
  sessionPaneClassName,
  sessionPaneProps,
  sessionPaneBackdrop,
  sessionPaneShellClassName,
  sessionPaneHeaderClassName,
  sessionPaneCopyClassName,
  sessionPaneActionsClassName,
  sessionPanePrimaryActionClassName,
  sessionPaneSecondaryActionClassName,
  sessionPanePrimaryActionProps,
  sessionPaneSecondaryActionProps,
  sessionPaneTitle,
  sessionPaneCountLabel,
  sessionPanePrimaryActionLabel,
  onSessionPanePrimaryAction,
  sessionPaneSecondaryActionLabel,
  onSessionPaneSecondaryAction,
  sessionPaneBody,
  workspaceClassName,
  workspaceProps,
  workspaceBodyClassName,
  workspaceBodyRef,
  mobileHeaderPlacement,
  mobileHeaderClassName,
  mobileHeaderProps,
  mobileLauncherLabel,
  mobileNavButtonClassName,
  mobileNavButtonLabel,
  mobileNavButtonProps,
  onMobileNav,
  mobileSessionButtonClassName,
  mobileSessionButtonLabel,
  mobileSessionButtonProps,
  onMobileSession,
  mobilePrimaryButtonClassName,
  mobilePrimaryButtonLabel,
  mobilePrimaryButtonProps,
  onMobilePrimary,
  mobileHeaderActionsClassName,
  workspaceHeader,
  workspaceContent,
  workspaceFooter,
}: RuntimeWorkspaceShellProps) {
  const [mobileActionsOpen, setMobileActionsOpen] = useState(false);
  const mobileActionsID = useId();
  const resolvedMobileLauncherLabel = mobileLauncherLabel || "Workspace actions";
  const hasMobileHeaderActions = Boolean(
    mobileNavButtonLabel || mobileSessionButtonLabel || mobilePrimaryButtonLabel,
  );

  useEffect(() => {
    if (!mobileHeaderPlacement || !hasMobileHeaderActions) {
      setMobileActionsOpen(false);
    }
  }, [hasMobileHeaderActions, mobileHeaderPlacement]);

  useEffect(() => {
    if (!mobileActionsOpen) {
      return;
    }
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setMobileActionsOpen(false);
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [mobileActionsOpen]);

  const triggerMobileAction = (action?: () => void) => {
    setMobileActionsOpen(false);
    action?.();
  };

  const mobileHeader = mobileHeaderPlacement ? (
    <header
      className={joinClassNames("runtime-workspace-mobile-header", mobileHeaderClassName)}
      data-runtime-mobile-header={mobileHeaderPlacement}
      {...mobileHeaderProps}
    >
      {hasMobileHeaderActions ? (
        <button
          className={joinClassNames(
            "runtime-workspace-mobile-action",
            "runtime-workspace-mobile-launcher",
            mobileActionsOpen ? "is-active" : undefined,
          )}
          type="button"
          aria-label={resolvedMobileLauncherLabel}
          aria-expanded={mobileActionsOpen}
          aria-controls={mobileActionsID}
          onClick={() => setMobileActionsOpen((current) => !current)}
        >
          <RuntimeWorkspaceMobileLauncherIcon />
        </button>
      ) : null}
      {mobileActionsOpen ? (
        <>
          <button
            type="button"
            className="runtime-workspace-mobile-sheet-backdrop"
            aria-label={resolvedMobileLauncherLabel}
            onClick={() => setMobileActionsOpen(false)}
          ></button>
          <div
            id={mobileActionsID}
            className={joinClassNames(
              "runtime-workspace-mobile-sheet",
              mobileHeaderActionsClassName,
            )}
            role="dialog"
            aria-modal="true"
            aria-label={resolvedMobileLauncherLabel}
          >
            {mobileNavButtonLabel ? (
              <button
                className={joinClassNames(
                  "runtime-workspace-mobile-sheet-action",
                  mobileNavButtonClassName,
                )}
                type="button"
                onClick={() => triggerMobileAction(onMobileNav)}
                {...mobileNavButtonProps}
              >
                {mobileNavButtonLabel}
              </button>
            ) : null}
            {mobileSessionButtonLabel ? (
              <button
                className={joinClassNames(
                  "runtime-workspace-mobile-sheet-action",
                  mobileSessionButtonClassName,
                )}
                type="button"
                onClick={() => triggerMobileAction(onMobileSession)}
                {...mobileSessionButtonProps}
              >
                {mobileSessionButtonLabel}
              </button>
            ) : null}
            {mobilePrimaryButtonLabel ? (
              <button
                className={joinClassNames(
                  "runtime-workspace-mobile-sheet-action",
                  "is-primary",
                  mobilePrimaryButtonClassName,
                )}
                type="button"
                onClick={() => triggerMobileAction(onMobilePrimary)}
                {...mobilePrimaryButtonProps}
              >
                {mobilePrimaryButtonLabel}
              </button>
            ) : null}
          </div>
        </>
      ) : null}
    </header>
  ) : null;

  return (
    <RuntimeWorkspaceFrame
      rootClassName={joinClassNames("runtime-workspace-shell", rootClassName)}
      rootProps={rootProps}
      leadingContent={mobileHeaderPlacement === "leading" ? mobileHeader : undefined}
      sessionPaneClassName={joinClassNames(
        "runtime-workspace-session-pane",
        sessionPaneClassName,
      )}
      sessionPaneProps={sessionPaneProps}
      sessionPaneBackdrop={{
        ...sessionPaneBackdrop,
        className: joinClassNames(
          "runtime-workspace-session-pane-backdrop",
          sessionPaneBackdrop.className,
        ),
      }}
      sessionPaneShellClassName={joinClassNames(
        "runtime-workspace-session-pane-shell",
        "route-surface",
        sessionPaneShellClassName,
      )}
      sessionPaneContent={(
        <>
          <div
            className={joinClassNames(
              "runtime-workspace-session-pane-head",
              sessionPaneHeaderClassName,
            )}
            data-runtime-session-pane-head="true"
          >
            <div
              className={joinClassNames(
                "runtime-workspace-session-pane-copy",
                sessionPaneCopyClassName,
              )}
            >
              <strong>{sessionPaneTitle}</strong>
              <span>{sessionPaneCountLabel}</span>
            </div>
            <div
              className={joinClassNames(
                "runtime-workspace-session-pane-actions",
                sessionPaneActionsClassName,
              )}
            >
              <button
                className={joinClassNames(
                  "runtime-workspace-session-pane-action",
                  sessionPanePrimaryActionClassName,
                )}
                type="button"
                onClick={onSessionPanePrimaryAction}
                {...sessionPanePrimaryActionProps}
              >
                {sessionPanePrimaryActionLabel}
              </button>
              {sessionPaneSecondaryActionLabel && onSessionPaneSecondaryAction ? (
                <button
                  className={joinClassNames(
                    "runtime-workspace-session-pane-action",
                    sessionPaneSecondaryActionClassName,
                  )}
                  type="button"
                  onClick={onSessionPaneSecondaryAction}
                  {...sessionPaneSecondaryActionProps}
                >
                  {sessionPaneSecondaryActionLabel}
                </button>
              ) : null}
            </div>
          </div>
          {sessionPaneBody}
        </>
      )}
      workspaceClassName={joinClassNames(
        "runtime-workspace",
        workspaceClassName,
      )}
      workspaceProps={workspaceProps}
      workspaceBodyClassName={joinClassNames(
        "runtime-workspace-body",
        workspaceBodyClassName,
      )}
      workspaceBodyRef={workspaceBodyRef}
      mobileHeader={mobileHeaderPlacement === "body" ? mobileHeader : undefined}
      workspaceHeader={workspaceHeader}
      workspaceContent={workspaceContent}
      workspaceFooter={workspaceFooter}
    />
  );
}

import type { ComponentPropsWithoutRef, ReactNode, Ref } from "react";
import { RuntimeWorkspaceFrame } from "./RuntimeWorkspaceFrame";

function joinClassNames(...values: Array<string | undefined>) {
  return values.filter(Boolean).join(" ");
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
  const mobileHeader = mobileHeaderPlacement ? (
    <header
      className={joinClassNames("runtime-workspace-mobile-header", mobileHeaderClassName)}
      data-runtime-mobile-header={mobileHeaderPlacement}
      {...mobileHeaderProps}
    >
      {mobileNavButtonLabel ? (
        <button
          className={joinClassNames(
            "runtime-workspace-mobile-action",
            mobileNavButtonClassName,
          )}
          type="button"
          onClick={onMobileNav}
          {...mobileNavButtonProps}
        >
          {mobileNavButtonLabel}
        </button>
      ) : null}
      {(mobileSessionButtonLabel || mobilePrimaryButtonLabel) ? (
        <div
          className={joinClassNames(
            "runtime-workspace-mobile-actions",
            mobileHeaderActionsClassName,
          )}
        >
          {mobileSessionButtonLabel ? (
            <button
              className={joinClassNames(
                "runtime-workspace-mobile-action",
                mobileSessionButtonClassName,
              )}
              type="button"
              onClick={onMobileSession}
              {...mobileSessionButtonProps}
            >
              {mobileSessionButtonLabel}
            </button>
          ) : null}
          {mobilePrimaryButtonLabel ? (
            <button
              className={joinClassNames(
                "runtime-workspace-mobile-action",
                mobilePrimaryButtonClassName,
              )}
              type="button"
              onClick={onMobilePrimary}
              {...mobilePrimaryButtonProps}
            >
              {mobilePrimaryButtonLabel}
            </button>
          ) : null}
        </div>
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

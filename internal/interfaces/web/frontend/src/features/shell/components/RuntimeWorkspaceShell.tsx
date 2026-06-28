import { useEffect, type ComponentPropsWithoutRef, type ReactNode, type Ref } from "react";
import { RuntimeWorkspaceFrame } from "./RuntimeWorkspaceFrame";
import { runtimeMobileLayoutSuspendsComposer, type RuntimeMobileLayoutState } from "./runtimeMobileLayout";
import { runWithKeyboardDismissal } from "./runtimeKeyboardDismissal";
import { useRuntimeMobilePressAction, type RuntimeMobilePressButtonProps } from "./useRuntimeMobilePressAction";

function joinClassNames(...values: Array<string | undefined>) {
  return values.filter(Boolean).join(" ");
}

function RuntimeSessionPaneAddIcon() {
  return (
    <svg viewBox="0 0 20 20" fill="none" focusable="false" aria-hidden="true">
      <path d="M10 5.25v9.5" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" />
      <path d="M5.25 10h9.5" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" />
    </svg>
  );
}

function RuntimeSessionPaneHideIcon() {
  return (
    <svg viewBox="0 0 20 20" fill="none" focusable="false" aria-hidden="true">
      <path
        d="M2.5 10c1.9-3.2 4.42-4.8 7.5-4.8s5.6 1.6 7.5 4.8c-1.9 3.2-4.42 4.8-7.5 4.8S4.4 13.2 2.5 10Z"
        stroke="currentColor"
        strokeWidth="1.7"
        strokeLinejoin="round"
      />
      <circle cx="10" cy="10" r="2.2" stroke="currentColor" strokeWidth="1.7" />
      <path d="M4 16 16 4" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
    </svg>
  );
}

function RuntimeMobileMenuIcon() {
  return (
    <svg viewBox="0 0 20 20" fill="none" focusable="false" aria-hidden="true" data-runtime-mobile-icon="menu">
      <path d="M4.5 6.25h11" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
      <path d="M4.5 10h11" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
      <path d="M4.5 13.75h11" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
    </svg>
  );
}

function RuntimeMobilePlusIcon() {
  return (
    <svg viewBox="0 0 20 20" fill="none" focusable="false" aria-hidden="true" data-runtime-mobile-icon="plus">
      <path d="M10 4.75v10.5" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" />
      <path d="M4.75 10h10.5" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" />
    </svg>
  );
}

function RuntimeMobileActionLabel({ label }: { label: string }) {
  return <span className="runtime-workspace-mobile-action-label sr-only">{label}</span>;
}

type RuntimeWorkspaceMobileTitleTone = "ready" | "busy" | "failed" | "interrupted" | "exited";
type RuntimeWorkspaceMobileActionKey = "nav" | "title" | "session" | "primary";
const MOBILE_ACTION_CLICK_SUPPRESS_MS = 700;

type RuntimeWorkspaceShellProps = {
  rootClassName?: string;
  rootProps?: ComponentPropsWithoutRef<"section">;
  mobileLayoutState?: RuntimeMobileLayoutState;
  sessionPanePlacement?: "workspace" | "navigation";
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
  workspaceBodyProps?: ComponentPropsWithoutRef<"div">;
  mobileHeaderPlacement?: "leading" | "body";
  mobileHeaderClassName?: string;
  mobileHeaderProps?: Omit<ComponentPropsWithoutRef<"header">, "className" | "children">;
  mobileNavButtonClassName?: string;
  mobileNavButtonLabel?: string;
  mobileNavButtonProps?: Omit<ComponentPropsWithoutRef<"button">, "type" | "className" | "children" | "onClick">;
  onMobileNav?: () => void;
  mobileTitleButtonClassName?: string;
  mobileTitleButtonLabel?: string;
  mobileTitleStatusLabel?: string;
  mobileTitleTone?: RuntimeWorkspaceMobileTitleTone;
  mobileTitleButtonProps?: Omit<ComponentPropsWithoutRef<"button">, "type" | "className" | "children" | "onClick">;
  onMobileTitle?: () => void;
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

function classNameIncludes(value: string | undefined, className: string): boolean {
  return (value || "").split(/\s+/).includes(className);
}

export function RuntimeWorkspaceShell({
  rootClassName,
  rootProps,
  mobileLayoutState,
  sessionPanePlacement,
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
  workspaceBodyProps,
  mobileHeaderPlacement,
  mobileHeaderClassName,
  mobileHeaderProps,
  mobileNavButtonClassName,
  mobileNavButtonLabel,
  mobileNavButtonProps,
  onMobileNav,
  mobileTitleButtonClassName,
  mobileTitleButtonLabel,
  mobileTitleStatusLabel,
  mobileTitleTone,
  mobileTitleButtonProps,
  onMobileTitle,
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
  const { createPressHandlers, triggerFromClick } = useRuntimeMobilePressAction({
    suppressClickMs: MOBILE_ACTION_CLICK_SUPPRESS_MS,
  });
  const createMobilePressHandlers = (
    key: RuntimeWorkspaceMobileActionKey,
    action: (() => void) | undefined,
    props: RuntimeMobilePressButtonProps | undefined,
  ) => createPressHandlers(key, action, props);
  const mobileNavPressProps = createMobilePressHandlers("nav", onMobileNav, mobileNavButtonProps);
  const mobileTitlePressProps = createMobilePressHandlers("title", onMobileTitle, mobileTitleButtonProps);
  const mobileSessionPressProps = createMobilePressHandlers("session", onMobileSession, mobileSessionButtonProps);
  const mobilePrimaryPressProps = createMobilePressHandlers("primary", onMobilePrimary, mobilePrimaryButtonProps);
  const sessionPaneOpen = classNameIncludes(sessionPaneClassName, "is-open")
    || mobileLayoutState === "mobile-session-drawer";
  const composerSuspendedState = runtimeMobileLayoutSuspendsComposer(mobileLayoutState);
  useEffect(() => {
    if (!composerSuspendedState) {
      return;
    }
    runWithKeyboardDismissal();
  }, [composerSuspendedState]);
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
          {...mobileNavPressProps}
          onClick={() => triggerFromClick("nav", onMobileNav)}
        >
          <RuntimeMobileMenuIcon />
          <RuntimeMobileActionLabel label={mobileNavButtonLabel} />
        </button>
      ) : null}
      {mobileTitleButtonLabel ? (
        <button
          className={joinClassNames(
            "runtime-workspace-mobile-title",
            mobileTitleButtonClassName,
          )}
          type="button"
          {...mobileTitlePressProps}
          onClick={() => triggerFromClick("title", onMobileTitle)}
        >
          <span className="runtime-workspace-mobile-title-copy">
            {mobileTitleTone ? (
              <span
                className={joinClassNames(
                  "workspace-header-status",
                  `is-${mobileTitleTone}`,
                )}
                role="img"
                aria-label={mobileTitleStatusLabel}
                title={mobileTitleStatusLabel}
                data-runtime-header-signal-container={mobileTitleTone}
              >
                <span
                  className={`runtime-session-signal is-${mobileTitleTone}`}
                  data-runtime-header-signal={mobileTitleTone}
                  aria-hidden="true"
                ></span>
              </span>
            ) : null}
            <span className="runtime-workspace-mobile-title-text">{mobileTitleButtonLabel}</span>
          </span>
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
              {...mobileSessionPressProps}
              onClick={() => triggerFromClick("session", onMobileSession)}
            >
              <RuntimeMobileMenuIcon />
              <RuntimeMobileActionLabel label={mobileSessionButtonLabel} />
            </button>
          ) : null}
          {mobilePrimaryButtonLabel ? (
            <button
              className={joinClassNames(
                "runtime-workspace-mobile-action",
                mobilePrimaryButtonClassName,
              )}
              type="button"
              {...mobilePrimaryPressProps}
              onClick={() => triggerFromClick("primary", onMobilePrimary)}
            >
              <RuntimeMobilePlusIcon />
              <RuntimeMobileActionLabel label={mobilePrimaryButtonLabel} />
            </button>
          ) : null}
        </div>
      ) : null}
    </header>
  ) : null;

  return (
    <RuntimeWorkspaceFrame
      rootClassName={joinClassNames("runtime-workspace-shell", rootClassName)}
      rootProps={{
        ...rootProps,
        "data-runtime-mobile-layout": mobileLayoutState,
        "data-runtime-session-pane-open": sessionPaneOpen ? "true" : "false",
        "data-runtime-composer-suspended": composerSuspendedState ? "true" : "false",
      }}
      leadingContent={mobileHeaderPlacement === "leading" ? mobileHeader : undefined}
      sessionPaneClassName={joinClassNames(
        "runtime-workspace-session-pane",
        sessionPanePlacement === "navigation" ? "is-navigation-owned" : undefined,
        sessionPaneClassName,
      )}
      sessionPaneProps={{
        ...sessionPaneProps,
        "data-session-pane-placement": sessionPanePlacement || "workspace",
        "aria-hidden": sessionPanePlacement === "navigation" ? "true" : sessionPaneProps?.["aria-hidden"],
      }}
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
                <span className="runtime-workspace-session-pane-action-icon">
                  <RuntimeSessionPaneAddIcon />
                </span>
                <span>{sessionPanePrimaryActionLabel}</span>
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
                  <span className="runtime-workspace-session-pane-action-icon">
                    <RuntimeSessionPaneHideIcon />
                  </span>
                  <span>{sessionPaneSecondaryActionLabel}</span>
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
      workspaceBodyProps={{
        ...workspaceBodyProps,
        "data-runtime-mobile-layout": mobileLayoutState,
        "data-runtime-session-pane-open": sessionPaneOpen ? "true" : "false",
        "data-runtime-composer-suspended": composerSuspendedState ? "true" : "false",
      }}
      mobileHeader={mobileHeaderPlacement === "body" ? mobileHeader : undefined}
      workspaceHeader={workspaceHeader}
      workspaceContent={workspaceContent}
      workspaceFooter={workspaceFooter}
    />
  );
}

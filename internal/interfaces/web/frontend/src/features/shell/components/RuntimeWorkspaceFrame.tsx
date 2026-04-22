import type { ComponentPropsWithoutRef, ReactNode, Ref } from "react";

type RuntimeWorkspaceFrameProps = {
  rootClassName: string;
  rootProps?: ComponentPropsWithoutRef<"section">;
  leadingContent?: ReactNode;
  sessionPaneClassName: string;
  sessionPaneProps?: ComponentPropsWithoutRef<"aside">;
  sessionPaneBackdrop?: {
    className?: string;
    ariaLabel: string;
    onClick: () => void;
    buttonProps?: Omit<ComponentPropsWithoutRef<"button">, "type" | "className" | "aria-label" | "onClick">;
  };
  sessionPaneShellClassName: string;
  sessionPaneShellProps?: ComponentPropsWithoutRef<"div">;
  sessionPaneContent: ReactNode;
  workspaceClassName: string;
  workspaceProps?: ComponentPropsWithoutRef<"section">;
  workspaceBodyClassName: string;
  workspaceBodyRef?: Ref<HTMLDivElement>;
  workspaceBodyProps?: ComponentPropsWithoutRef<"div">;
  mobileHeader?: ReactNode;
  workspaceHeader?: ReactNode;
  workspaceContent: ReactNode;
  workspaceFooter?: ReactNode;
};

function joinClassNames(...values: Array<string | undefined>) {
  return values.filter(Boolean).join(" ");
}

export function RuntimeWorkspaceFrame({
  rootClassName,
  rootProps,
  leadingContent,
  sessionPaneClassName,
  sessionPaneProps,
  sessionPaneBackdrop,
  sessionPaneShellClassName,
  sessionPaneShellProps,
  sessionPaneContent,
  workspaceClassName,
  workspaceProps,
  workspaceBodyClassName,
  workspaceBodyRef,
  workspaceBodyProps,
  mobileHeader,
  workspaceHeader,
  workspaceContent,
  workspaceFooter,
}: RuntimeWorkspaceFrameProps) {
  const {
    className: rootPropsClassName,
    ...rootRestProps
  } = rootProps || {};
  const {
    className: sessionPanePropsClassName,
    ...sessionPaneRestProps
  } = sessionPaneProps || {};
  const {
    className: sessionPaneShellPropsClassName,
    ...sessionPaneShellRestProps
  } = sessionPaneShellProps || {};
  const {
    className: workspacePropsClassName,
    ...workspaceRestProps
  } = workspaceProps || {};
  const {
    className: workspaceBodyPropsClassName,
    ...workspaceBodyRestProps
  } = workspaceBodyProps || {};

  return (
    <section className={joinClassNames(rootClassName, rootPropsClassName)} {...rootRestProps}>
      {leadingContent}
      <aside className={joinClassNames(sessionPaneClassName, sessionPanePropsClassName)} {...sessionPaneRestProps}>
        {sessionPaneBackdrop ? (
          <button
            type="button"
            aria-label={sessionPaneBackdrop.ariaLabel}
            onClick={sessionPaneBackdrop.onClick}
            className={sessionPaneBackdrop.className}
            {...sessionPaneBackdrop.buttonProps}
          ></button>
        ) : null}
        <div
          className={joinClassNames(sessionPaneShellClassName, sessionPaneShellPropsClassName)}
          {...sessionPaneShellRestProps}
        >
          {sessionPaneContent}
        </div>
      </aside>
      <section className={joinClassNames(workspaceClassName, workspacePropsClassName)} {...workspaceRestProps}>
        <div
          ref={workspaceBodyRef}
          className={joinClassNames(workspaceBodyClassName, workspaceBodyPropsClassName)}
          {...workspaceBodyRestProps}
        >
          {mobileHeader}
          {workspaceHeader}
          {workspaceContent}
          {workspaceFooter}
        </div>
      </section>
    </section>
  );
}

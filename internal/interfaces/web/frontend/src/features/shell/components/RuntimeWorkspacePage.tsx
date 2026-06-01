import { useEffect, useMemo, type ComponentPropsWithoutRef, type MouseEvent, type ReactNode, type TouchEvent } from "react";
import { useWorkbenchContext, type WorkbenchSessionRail } from "../../../app/WorkbenchContext";
import { RuntimeComposer } from "./RuntimeComposer";
import { RuntimeSessionList, type RuntimeSessionListGroup } from "./RuntimeSessionList";
import { RouteFieldRow } from "./RouteBodyPrimitives";
import { RuntimeTimeline, type RuntimeTimelineItem } from "./RuntimeTimeline";
import { RuntimeWorkspaceHeader } from "./RuntimeWorkspaceHeader";
import { RuntimeWorkspaceScreen } from "./RuntimeWorkspaceScreen";
import { RuntimeWorkspaceShell } from "./RuntimeWorkspaceShell";

function RuntimeSessionMoreIcon() {
  return (
    <svg viewBox="0 0 20 20" fill="currentColor" focusable="false" aria-hidden="true">
      <circle cx="5" cy="10" r="1.35" />
      <circle cx="10" cy="10" r="1.35" />
      <circle cx="15" cy="10" r="1.35" />
    </svg>
  );
}

function swallowSessionDeleteGesture(
  event: MouseEvent<HTMLButtonElement> | TouchEvent<HTMLButtonElement>,
) {
  event.preventDefault();
  event.stopPropagation();
}

export type RuntimeWorkspaceDetailsField = {
  label: string;
  value: unknown;
  copyLabel: string;
  copyable?: boolean;
  mono?: boolean;
  multiline?: boolean;
  markdown?: boolean;
};

export type RuntimeWorkspaceSessionTone = "ready" | "busy" | "failed";

export type RuntimeWorkspaceSessionItem = {
  id: string;
  active: boolean;
  title: string;
  contextLabel?: string;
  meta: string;
  shortHash: string;
  statusTone?: RuntimeWorkspaceSessionTone;
  statusLabel?: string;
  activeLabel: string;
  idleLabel: string;
  onSelect: () => void;
  onDelete?: () => void;
  deleteLabel?: string;
  deleteAriaLabel?: string;
  deleting?: boolean;
  deleteClassName?: string;
  deleteProps?: Omit<ComponentPropsWithoutRef<"button">, "type" | "className" | "children" | "aria-label" | "disabled" | "onClick">;
  shellClassName?: string;
  buttonClassName?: string;
  shellProps?: ComponentPropsWithoutRef<"div">;
  buttonProps?: Omit<ComponentPropsWithoutRef<"button">, "type" | "className" | "children" | "onClick">;
};

export type RuntimeWorkspacePageController = {
  shell: Omit<ComponentPropsWithoutRef<typeof RuntimeWorkspaceShell>, "sessionPaneBody" | "workspaceHeader" | "workspaceContent" | "workspaceFooter"> & {
    rootProps?: ComponentPropsWithoutRef<"section">;
  };
  sessionList: {
    groups: Array<RuntimeSessionListGroup<RuntimeWorkspaceSessionItem>>;
    emptyState?: ReactNode;
    listClassName?: string;
    listProps?: Omit<ComponentPropsWithoutRef<"div">, "children" | "className"> & {
      className?: string;
    };
  };
  header: Omit<ComponentPropsWithoutRef<typeof RuntimeWorkspaceHeader>, "detailsContent"> & {
    customHeaderContent?: ReactNode;
    detailsSummary?: RuntimeWorkspaceDetailsField[];
    detailsBody?: ReactNode;
    detailsClassName?: string;
  };
  screen: Omit<ComponentPropsWithoutRef<typeof RuntimeWorkspaceScreen>, "children">;
  timeline: {
    className?: string;
    timelineProps?: Omit<ComponentPropsWithoutRef<"div">, "children" | "className">;
    emptyState?: ReactNode;
    topContent?: ReactNode;
    items: RuntimeTimelineItem[];
    overlay?: ReactNode;
  };
  composer?: ComponentPropsWithoutRef<typeof RuntimeComposer>;
  composerNode?: ReactNode;
};

export function RuntimeWorkspacePage({ controller }: { controller: RuntimeWorkspacePageController }) {
  const {
    route,
    setRuntimeSessionRail,
  } = useWorkbenchContext();
  const detailsSummary = controller.header.detailsSummary || [];
  const workspaceHeader = useMemo(() => controller.header.customHeaderContent ?? (
    <RuntimeWorkspaceHeader
      {...controller.header}
      detailsContent={controller.header.detailsBody || detailsSummary.length > 0 ? (
        <section className={controller.header.detailsClassName || "workspace-details-content"}>
          {detailsSummary.length > 0 ? (
            <div className="workspace-details-summary">
              {detailsSummary.map((field) => (
                <RouteFieldRow
                  key={`${field.label}:${String(field.value)}`}
                  label={field.label}
                  value={field.value}
                  copyLabel={field.copyLabel}
                  copyable={field.copyable}
                  mono={field.mono}
                  multiline={field.multiline}
                  markdown={field.markdown}
                />
              ))}
            </div>
          ) : null}
          {controller.header.detailsBody ? (
            <div className="workspace-details-body">
              {controller.header.detailsBody}
            </div>
          ) : null}
        </section>
      ) : null}
    />
  ), [controller.header, detailsSummary]);
  const sessionPaneBody = useMemo(() => (
    <RuntimeSessionList
      groups={controller.sessionList.groups}
      emptyState={controller.sessionList.emptyState}
      listClassName={controller.sessionList.listClassName}
      listProps={controller.sessionList.listProps}
      renderItem={(item) => {
        const busy = item.statusTone === "busy";
        return (
          <div
            key={item.id}
            role="listitem"
            className={item.shellClassName}
            {...item.shellProps}
          >
            <button
              className={item.buttonClassName}
              type="button"
              aria-current={item.active ? "true" : undefined}
              onClick={item.onSelect}
              {...item.buttonProps}
            >
              <span className="runtime-session-main">
                <span className="runtime-session-title-row">
                  <span className="sr-only">
                    {item.active ? item.activeLabel : item.idleLabel}
                    {busy && item.statusLabel ? `, ${item.statusLabel}` : ""}
                  </span>
                  <span className="runtime-session-title-copy">
                    <span className="runtime-session-title">{item.title}</span>
                  </span>
                  {busy ? (
                    <span
                      className="runtime-session-loading"
                      data-runtime-session-loading="busy"
                      aria-hidden="true"
                    ></span>
                  ) : null}
                </span>
              </span>
            </button>
            {item.onDelete ? (
              <button
                className={item.deleteClassName || "runtime-session-delete"}
                type="button"
                aria-label={item.deleteAriaLabel || item.deleteLabel}
                disabled={item.deleting}
                onMouseDown={swallowSessionDeleteGesture}
                onTouchStart={swallowSessionDeleteGesture}
                onClick={(event) => {
                  swallowSessionDeleteGesture(event);
                  item.onDelete?.();
                }}
                {...item.deleteProps}
              >
                <span className="runtime-session-delete-icon" aria-hidden="true">
                  <RuntimeSessionMoreIcon />
                </span>
                <span className="sr-only">{item.deleteLabel}</span>
              </button>
            ) : null}
          </div>
        );
      }}
    />
  ), [controller.sessionList]);
  const runtimeSessionRail = useMemo<WorkbenchSessionRail>(() => ({
    route,
    title: controller.shell.sessionPaneTitle,
    countLabel: controller.shell.sessionPaneCountLabel,
    primaryActionLabel: controller.shell.sessionPanePrimaryActionLabel,
    onPrimaryAction: controller.shell.onSessionPanePrimaryAction,
    primaryActionClassName: controller.shell.sessionPanePrimaryActionClassName,
    primaryActionProps: controller.shell.sessionPanePrimaryActionProps,
    body: sessionPaneBody,
  }), [
    controller.shell.onSessionPanePrimaryAction,
    controller.shell.sessionPaneCountLabel,
    controller.shell.sessionPanePrimaryActionClassName,
    controller.shell.sessionPanePrimaryActionLabel,
    controller.shell.sessionPanePrimaryActionProps,
    controller.shell.sessionPaneTitle,
    route,
    sessionPaneBody,
  ]);
  useEffect(() => {
    setRuntimeSessionRail?.(runtimeSessionRail);
    return () => {
      setRuntimeSessionRail?.(null);
    };
  }, [runtimeSessionRail, setRuntimeSessionRail]);
  const workspaceContent = useMemo(() => (
    <RuntimeWorkspaceScreen {...controller.screen} overlay={controller.timeline.overlay}>
      <RuntimeTimeline
        className={controller.timeline.className}
        timelineProps={controller.timeline.timelineProps}
        emptyState={controller.timeline.emptyState}
        topContent={controller.timeline.topContent}
        items={controller.timeline.items}
      />
    </RuntimeWorkspaceScreen>
  ), [controller.screen, controller.timeline]);

  return (
    <RuntimeWorkspaceShell
      {...controller.shell}
      sessionPanePlacement="navigation"
      sessionPaneProps={{
        "data-runtime-session-pane": route,
        ...controller.shell.sessionPaneProps,
      }}
      rootProps={{
        ...controller.shell.rootProps,
        "data-runtime-workspace-page": "true",
      }}
      sessionPaneBody={sessionPaneBody}
      workspaceHeader={workspaceHeader}
      workspaceContent={workspaceContent}
      workspaceFooter={controller.composerNode ?? (controller.composer ? <RuntimeComposer {...controller.composer} /> : null)}
    />
  );
}

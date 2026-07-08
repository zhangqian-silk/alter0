import { useEffect, useMemo, useState, type ComponentPropsWithoutRef, type MouseEvent, type ReactNode, type TouchEvent } from "react";
import { useWorkbenchContext, type WorkbenchSessionRail } from "../../../app/WorkbenchContext";
import { RuntimeComposer } from "./RuntimeComposer";
import { RuntimeSessionList, type RuntimeSessionListGroup } from "./RuntimeSessionList";
import { RouteFieldRow } from "./RouteBodyPrimitives";
import { RuntimeTimeline, type RuntimeTimelineItem } from "./RuntimeTimeline";
import { RuntimeWorkspaceHeader } from "./RuntimeWorkspaceHeader";
import { RuntimeWorkspaceScreen } from "./RuntimeWorkspaceScreen";
import { RuntimeWorkspaceShell } from "./RuntimeWorkspaceShell";

function RuntimeSessionDetailsIcon() {
  return (
    <svg viewBox="0 0 20 20" fill="none" focusable="false" aria-hidden="true">
      <circle cx="10" cy="10" r="7.1" stroke="currentColor" strokeWidth="1.6" />
      <path d="M10 9.2v4.35" stroke="currentColor" strokeLinecap="round" strokeWidth="1.7" />
      <circle cx="10" cy="6.35" r="0.9" fill="currentColor" />
    </svg>
  );
}

function RuntimeSessionDeleteIcon() {
  return (
    <svg viewBox="0 0 20 20" fill="none" focusable="false" aria-hidden="true">
      <path d="M6.2 7.6h7.6l-.55 7.3a1.6 1.6 0 0 1-1.6 1.48h-3.3a1.6 1.6 0 0 1-1.6-1.48L6.2 7.6Z" stroke="currentColor" strokeWidth="1.55" strokeLinejoin="round" />
      <path d="M5 7.6h10" stroke="currentColor" strokeLinecap="round" strokeWidth="1.55" />
      <path d="M8.2 5.1h3.6" stroke="currentColor" strokeLinecap="round" strokeWidth="1.55" />
      <path d="M8.85 10.1v3.45M11.15 10.1v3.45" stroke="currentColor" strokeLinecap="round" strokeWidth="1.35" />
    </svg>
  );
}

function RuntimeSessionMoreIcon() {
  return (
    <svg viewBox="0 0 20 20" fill="none" focusable="false" aria-hidden="true">
      <circle cx="4.75" cy="10" r="1.45" fill="currentColor" />
      <circle cx="10" cy="10" r="1.45" fill="currentColor" />
      <circle cx="15.25" cy="10" r="1.45" fill="currentColor" />
    </svg>
  );
}

function RuntimeSessionPinIcon({ pinned }: { pinned: boolean }) {
  return (
    <svg viewBox="0 0 20 20" fill="none" focusable="false" aria-hidden="true">
      <path
        d="M8.25 3.75h5.5l-1.35 4.1 2.35 2.45v1.45H11.1L10 16.25l-1.1-4.5H5.25V10.3L7.6 7.85 6.25 3.75h2"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="1.55"
        fill={pinned ? "currentColor" : "none"}
        fillOpacity={pinned ? "0.14" : undefined}
      />
      <path d="M10 11.75v4.5" stroke="currentColor" strokeLinecap="round" strokeWidth="1.55" />
    </svg>
  );
}

function swallowSessionActionGesture(
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
  statusTone?: RuntimeWorkspaceSessionTone;
  statusLabel?: string;
  activeLabel: string;
  idleLabel: string;
  onSelect: () => void;
  onViewDetails?: () => void;
  viewDetailsLabel?: string;
  viewDetailsAriaLabel?: string;
  viewDetailsClassName?: string;
  viewDetailsProps?: Omit<ComponentPropsWithoutRef<"button">, "type" | "className" | "children" | "aria-label" | "onClick">;
  pinned?: boolean;
  pinning?: boolean;
  onPinnedChange?: (pinned: boolean) => void;
  pinLabel?: string;
  unpinLabel?: string;
  pinAriaLabel?: string;
  unpinAriaLabel?: string;
  pinClassName?: string;
  pinProps?: Omit<ComponentPropsWithoutRef<"button">, "type" | "className" | "children" | "aria-label" | "aria-pressed" | "disabled" | "onClick">;
  onDelete?: () => void;
  deleteLabel?: string;
  deleteAriaLabel?: string;
  deleteConfirmLabel?: string;
  deleting?: boolean;
  deleteClassName?: string;
  deleteProps?: Omit<ComponentPropsWithoutRef<"button">, "type" | "className" | "children" | "aria-label" | "disabled" | "onClick">;
  actionsLabel?: string;
  actionsAriaLabel?: string;
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

export function RuntimeWorkspaceNavigationSessionList({
  sessionList,
}: {
  sessionList: RuntimeWorkspacePageController["sessionList"];
}) {
  const {
    route,
    navigate,
  } = useWorkbenchContext();
  const [openActionMenuID, setOpenActionMenuID] = useState("");
  const routeOwnsSessionSelection = route === "chat" || route === "chatRuntime";

  return (
    <RuntimeSessionList
      groups={sessionList.groups}
      emptyState={sessionList.emptyState}
      listClassName={sessionList.listClassName}
      listProps={sessionList.listProps}
      renderItem={(item) => {
        const busy = item.statusTone === "busy";
        const itemActive = routeOwnsSessionSelection && item.active;
        const shellClassName = itemActive
          ? item.shellClassName
          : (item.shellClassName || "runtime-session-card").replace(/\bis-active\b/g, "").trim() || "runtime-session-card";
        const buttonClassName = itemActive
          ? item.buttonClassName
          : (item.buttonClassName || "runtime-session-select").replace(/\bactive\b/g, "").trim() || "runtime-session-select";
        const shellProps = {
          ...item.shellProps,
          "data-runtime-session-state": itemActive ? "active" : "idle",
        };

        return (
          <div
            key={item.id}
            role="listitem"
            className={openActionMenuID === item.id ? `${shellClassName} is-menu-open` : shellClassName}
            {...shellProps}
          >
            <button
              className={buttonClassName}
              type="button"
              aria-current={itemActive ? "true" : undefined}
              onClick={() => {
                setOpenActionMenuID("");
                if (!routeOwnsSessionSelection) {
                  navigate("chat");
                }
                item.onSelect();
              }}
              {...item.buttonProps}
            >
              <span className="runtime-session-main">
                <span className="runtime-session-title-row">
                  <span className="sr-only">
                    {itemActive ? item.activeLabel : item.idleLabel}
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
            {item.onPinnedChange || item.onViewDetails || item.onDelete ? (
              <span className={openActionMenuID === item.id ? "runtime-session-actions is-menu-open" : "runtime-session-actions"}>
                <button
                  className="runtime-session-action runtime-session-more"
                  type="button"
                  aria-label={item.actionsAriaLabel || item.actionsLabel || "Session actions"}
                  aria-haspopup="menu"
                  aria-expanded={openActionMenuID === item.id ? "true" : "false"}
                  aria-controls={`runtime-session-actions-${item.id}`}
                  onMouseDown={swallowSessionActionGesture}
                  onTouchStart={swallowSessionActionGesture}
                  onClick={(event) => {
                    swallowSessionActionGesture(event);
                    setOpenActionMenuID((current) => current === item.id ? "" : item.id);
                  }}
                >
                  <span className="runtime-session-action-icon runtime-session-more-icon" aria-hidden="true">
                    <RuntimeSessionMoreIcon />
                  </span>
                  <span className="sr-only">{item.actionsLabel || "Session actions"}</span>
                </button>
                {openActionMenuID === item.id ? (
                  <span
                    id={`runtime-session-actions-${item.id}`}
                    className="runtime-session-action-menu"
                    role="menu"
                    aria-label={item.actionsLabel || "Session actions"}
                  >
                    {item.onPinnedChange ? (
                      <button
                        className={item.pinClassName || [
                          "runtime-session-menu-item",
                          "runtime-session-pin",
                          item.pinned ? "is-pinned" : undefined,
                        ].filter(Boolean).join(" ")}
                        type="button"
                        role="menuitem"
                        aria-label={
                          item.pinned
                            ? item.unpinAriaLabel || item.unpinLabel
                            : item.pinAriaLabel || item.pinLabel
                        }
                        disabled={item.pinning}
                        onMouseDown={swallowSessionActionGesture}
                        onTouchStart={swallowSessionActionGesture}
                        onClick={(event) => {
                          swallowSessionActionGesture(event);
                          setOpenActionMenuID("");
                          item.onPinnedChange?.(!item.pinned);
                        }}
                        {...item.pinProps}
                      >
                        <span className="runtime-session-action-icon runtime-session-pin-icon" aria-hidden="true">
                          <RuntimeSessionPinIcon pinned={Boolean(item.pinned)} />
                        </span>
                        <span className="runtime-session-menu-label">{item.pinned ? item.unpinLabel : item.pinLabel}</span>
                      </button>
                    ) : null}
                    {item.onViewDetails ? (
                      <button
                        className={item.viewDetailsClassName || "runtime-session-menu-item runtime-session-details"}
                        type="button"
                        role="menuitem"
                        aria-label={item.viewDetailsAriaLabel || item.viewDetailsLabel}
                        onMouseDown={swallowSessionActionGesture}
                        onTouchStart={swallowSessionActionGesture}
                        onClick={(event) => {
                          swallowSessionActionGesture(event);
                          setOpenActionMenuID("");
                          item.onViewDetails?.();
                        }}
                        {...item.viewDetailsProps}
                      >
                        <span className="runtime-session-action-icon" aria-hidden="true">
                          <RuntimeSessionDetailsIcon />
                        </span>
                        <span className="runtime-session-menu-label">{item.viewDetailsLabel}</span>
                      </button>
                    ) : null}
                    {item.onDelete ? (
                      <button
                        className={item.deleteClassName || ["runtime-session-menu-item", "runtime-session-delete"].join(" ")}
                        type="button"
                        role="menuitem"
                        aria-label={item.deleteAriaLabel || item.deleteLabel}
                        disabled={item.deleting}
                        onMouseDown={swallowSessionActionGesture}
                        onTouchStart={swallowSessionActionGesture}
                        onClick={(event) => {
                          swallowSessionActionGesture(event);
                          setOpenActionMenuID("");
                          if (item.deleteConfirmLabel && typeof window !== "undefined" && !window.confirm(item.deleteConfirmLabel)) {
                            return;
                          }
                          item.onDelete?.();
                        }}
                        {...item.deleteProps}
                      >
                        <span className="runtime-session-action-icon runtime-session-delete-icon" aria-hidden="true">
                          <RuntimeSessionDeleteIcon />
                        </span>
                        <span className="runtime-session-menu-label">{item.deleteLabel}</span>
                      </button>
                    ) : null}
                  </span>
                ) : null}
              </span>
            ) : null}
          </div>
        );
      }}
    />
  );
}

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
    <RuntimeWorkspaceNavigationSessionList sessionList={controller.sessionList} />
  ), [controller.sessionList]);
  const runtimeSessionRail = useMemo<WorkbenchSessionRail>(() => ({
    route: "chat",
    countLabel: controller.shell.sessionPaneCountLabel,
    onPrimaryAction: controller.shell.onSessionPanePrimaryAction,
    primaryActionClassName: controller.shell.sessionPanePrimaryActionClassName,
    primaryActionProps: controller.shell.sessionPanePrimaryActionProps,
    body: sessionPaneBody,
  }), [
    controller.shell.onSessionPanePrimaryAction,
    controller.shell.sessionPaneCountLabel,
    controller.shell.sessionPanePrimaryActionClassName,
    controller.shell.sessionPanePrimaryActionProps,
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
  const composerNode = controller.composerNode ?? (controller.composer ? <RuntimeComposer {...controller.composer} /> : null);
  const workspaceFooter = composerNode;

  return (
    <>
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
        workspaceFooter={workspaceFooter}
      />
    </>
  );
}

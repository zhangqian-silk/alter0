import type { ComponentPropsWithoutRef, ReactNode } from "react";
import { RuntimeComposer } from "./RuntimeComposer";
import { RuntimeSessionList, type RuntimeSessionListGroup } from "./RuntimeSessionList";
import { RouteFieldRow } from "./RouteBodyPrimitives";
import { RuntimeTimeline, type RuntimeTimelineItem } from "./RuntimeTimeline";
import { RuntimeWorkspaceHeader } from "./RuntimeWorkspaceHeader";
import { RuntimeWorkspaceScreen } from "./RuntimeWorkspaceScreen";
import { RuntimeWorkspaceShell } from "./RuntimeWorkspaceShell";

export type RuntimeWorkspaceDetailsField = {
  label: string;
  value: unknown;
  copyLabel: string;
  copyable?: boolean;
  mono?: boolean;
  multiline?: boolean;
};

export type RuntimeWorkspaceSessionItem = {
  id: string;
  active: boolean;
  title: string;
  meta: string;
  shortHash: string;
  activeLabel: string;
  idleLabel: string;
  statusLabel?: string;
  statusClassName?: string;
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
    items: RuntimeTimelineItem[];
    overlay?: ReactNode;
  };
  composer: ComponentPropsWithoutRef<typeof RuntimeComposer>;
};

export function RuntimeWorkspacePage({ controller }: { controller: RuntimeWorkspacePageController }) {
  const detailsSummary = controller.header.detailsSummary || [];
  const workspaceHeader = controller.header.customHeaderContent ?? (
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
  );

  return (
    <RuntimeWorkspaceShell
      {...controller.shell}
      rootProps={{
        ...controller.shell.rootProps,
        "data-runtime-workspace-page": "true",
      }}
      sessionPaneBody={(
        <RuntimeSessionList
          groups={controller.sessionList.groups}
          emptyState={controller.sessionList.emptyState}
          listClassName={controller.sessionList.listClassName}
          listProps={controller.sessionList.listProps}
          renderItem={(item) => (
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
                  <span className="runtime-session-topline">
                    <span
                      className={item.active ? "runtime-session-badge is-active" : "runtime-session-badge"}
                    >
                      {item.active ? item.activeLabel : item.idleLabel}
                    </span>
                    {item.statusLabel ? (
                      <span className={item.statusClassName}>{item.statusLabel}</span>
                    ) : null}
                  </span>
                  <span className="runtime-session-title-row">
                    <span className="runtime-session-title">{item.title}</span>
                  </span>
                  <span className="runtime-session-summary-row">
                    <span className="runtime-session-meta">{item.meta}</span>
                  </span>
                  <span className="runtime-session-bottomline">
                    <span className="runtime-session-hash">{item.shortHash}</span>
                  </span>
                </span>
              </button>
              {item.onDelete ? (
                <button
                  className={item.deleteClassName || "runtime-session-delete"}
                  type="button"
                  aria-label={item.deleteAriaLabel || item.deleteLabel}
                  disabled={item.deleting}
                  onClick={item.onDelete}
                  {...item.deleteProps}
                >
                  {item.deleteLabel}
                </button>
              ) : null}
            </div>
          )}
        />
      )}
      workspaceHeader={workspaceHeader}
      workspaceContent={(
        <RuntimeWorkspaceScreen {...controller.screen}>
          <RuntimeTimeline
            className={controller.timeline.className}
            timelineProps={controller.timeline.timelineProps}
            emptyState={controller.timeline.emptyState}
            items={controller.timeline.items}
            overlay={controller.timeline.overlay}
          />
        </RuntimeWorkspaceScreen>
      )}
      workspaceFooter={<RuntimeComposer {...controller.composer} />}
    />
  );
}

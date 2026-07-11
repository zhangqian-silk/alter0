import { memo, type ComponentPropsWithoutRef, type ReactNode } from "react";
import {
  RuntimeAttachmentGallery,
  type RuntimeAttachmentGalleryItem,
} from "./RuntimeTimelinePrimitives";
import { MessageMarkdownHTML, MessageMarkdownShell } from "./MessageMarkdownShell";

function joinClassNames(...values: Array<string | undefined>) {
  return values.filter(Boolean).join(" ");
}

export type RuntimeTimelineProcessEvent = {
  id: string;
  itemClassName?: string;
  itemProps?: ComponentPropsWithoutRef<"article">;
  toggleable?: boolean;
  toggleClassName?: string;
  toggleProps?: Omit<ComponentPropsWithoutRef<"button">, "type" | "className" | "children" | "onClick">;
  title: string;
  titleClassName?: string;
  meta?: ReactNode;
  expanded: boolean;
  onToggle: () => void;
  bodyClassName?: string;
  bodyProps?: ComponentPropsWithoutRef<"div">;
  detail?: ReactNode;
};

export type RuntimeTimelineBlock =
  | {
      type: "attachments";
      galleryId?: string;
      className?: string;
      items: RuntimeAttachmentGalleryItem[];
    }
  | {
      type: "rich-text";
      className?: string;
      html: string;
    }
  | {
      type: "markdown-shell";
      markdown: string;
      wrapperClassName?: string;
      wrapperProps?: ComponentPropsWithoutRef<"div">;
      bubbleClassName?: string;
      bubbleProps?: ComponentPropsWithoutRef<"div">;
      className?: string;
      bodyClassName?: string;
    }
  | {
      type: "prompt";
      className?: string;
      bubbleClassName?: string;
      bubbleProps?: ComponentPropsWithoutRef<"div">;
      textClassName?: string;
      timeClassName?: string;
      text: string;
      timeLabel?: string;
    }
  | {
      type: "process";
      shellClassName?: string;
      shellProps?: ComponentPropsWithoutRef<"section">;
      toggleClassName?: string;
      toggleProps?: Omit<ComponentPropsWithoutRef<"button">, "type" | "className" | "children" | "onClick">;
      title: ReactNode;
      summary?: ReactNode;
      meta?: ReactNode;
      expanded: boolean;
      onToggle: () => void;
      bodyClassName?: string;
      bodyProps?: ComponentPropsWithoutRef<"div">;
      emptyState?: ReactNode;
      events: RuntimeTimelineProcessEvent[];
    };

export type RuntimeTimelineItem = {
  id: string;
  className?: string;
  articleProps?: ComponentPropsWithoutRef<"article">;
  bubbleClassName?: string;
  bubbleProps?: ComponentPropsWithoutRef<"div">;
  blocks: RuntimeTimelineBlock[];
  footer?: ReactNode;
};

type RuntimeTimelineProps = {
  className?: string;
  timelineProps?: Omit<ComponentPropsWithoutRef<"div">, "children" | "className">;
  emptyState?: ReactNode;
  topContent?: ReactNode;
  items: RuntimeTimelineItem[];
  overlay?: ReactNode;
};

export const RuntimeTimeline = memo(function RuntimeTimeline({
  className,
  timelineProps,
  emptyState,
  topContent,
  items,
  overlay,
}: RuntimeTimelineProps) {
  return (
    <>
      <div
        className={joinClassNames("runtime-timeline", className)}
        data-runtime-timeline="true"
        {...timelineProps}
      >
        {items.length === 0 ? emptyState : (
          <>
            {topContent}
            {items.map((item) => (
              <RuntimeTimelineArticle key={item.id} item={item} />
            ))}
          </>
        )}
      </div>
      {overlay}
    </>
  );
});

function RuntimeTimelineArticle({ item }: { item: RuntimeTimelineItem }) {
  const content = (
    <>
      {item.blocks.map((block, index) => (
        <RuntimeTimelineBlockNode key={`${item.id}:${block.type}:${index}`} block={block} />
      ))}
    </>
  );

  return (
    <article className={item.className} {...item.articleProps}>
      {item.bubbleClassName ? (
        <div className={item.bubbleClassName} {...item.bubbleProps}>
          {content}
        </div>
      ) : content}
      {item.footer}
    </article>
  );
}

function RuntimeTimelineBlockNode({ block }: { block: RuntimeTimelineBlock }) {
  switch (block.type) {
    case "attachments":
      return (
        <RuntimeAttachmentGallery
          galleryId={block.galleryId}
          className={block.className}
          items={block.items}
        />
      );
    case "rich-text":
      return <MessageMarkdownHTML html={block.html} className={block.className} />;
    case "markdown-shell":
      const shell = (
        <MessageMarkdownShell
          markdown={block.markdown}
          className={block.className}
          bodyClassName={block.bodyClassName}
        />
      );
      if (!block.wrapperClassName && !block.bubbleClassName) {
        return shell;
      }
      return (
        <div className={block.wrapperClassName} {...block.wrapperProps}>
          {block.bubbleClassName ? (
            <div className={block.bubbleClassName} {...block.bubbleProps}>
              {shell}
            </div>
          ) : shell}
        </div>
      );
    case "prompt":
      const promptContent = (
        <>
          <div className={block.textClassName}>
            <span className="chatRuntime-log-text">{block.text}</span>
          </div>
          {block.timeLabel ? <span className={block.timeClassName}>{block.timeLabel}</span> : null}
        </>
      );
      return (
        <div className={block.className}>
          {block.bubbleClassName ? (
            <div className={block.bubbleClassName} {...block.bubbleProps}>
              {promptContent}
            </div>
          ) : promptContent}
        </div>
      );
    case "process":
      return (
        <section className={block.shellClassName} {...block.shellProps}>
          <button
            className={block.toggleClassName}
            type="button"
            aria-expanded={block.expanded}
            onClick={block.onToggle}
            {...block.toggleProps}
          >
            {block.title}
            {block.summary ? <span>{block.summary}</span> : null}
            {block.meta ? <span>{block.meta}</span> : null}
          </button>
          <div className={block.bodyClassName} hidden={!block.expanded} {...block.bodyProps}>
            {block.events.length ? block.events.map((event) => (
              <article key={event.id} className={event.itemClassName} {...event.itemProps}>
                {event.toggleable === false ? (
                  <div className={event.toggleClassName}>
                    {event.meta ? <span>{event.meta}</span> : null}
                    <span className={event.titleClassName}>{event.title}</span>
                  </div>
                ) : (
                  <button
                    className={event.toggleClassName}
                    type="button"
                    aria-expanded={event.expanded}
                    onClick={event.onToggle}
                    {...event.toggleProps}
                  >
                    <span className="chatRuntime-step-toggle-icon" aria-hidden="true">
                      {event.expanded ? "v" : ">"}
                    </span>
                    <span className={event.titleClassName}>{event.title}</span>
                    {event.meta ? <span>{event.meta}</span> : null}
                  </button>
                )}
                <div className={event.bodyClassName} hidden={!event.expanded} {...event.bodyProps}>
                  {event.detail}
                </div>
              </article>
            )) : block.emptyState}
          </div>
        </section>
      );
  }
}

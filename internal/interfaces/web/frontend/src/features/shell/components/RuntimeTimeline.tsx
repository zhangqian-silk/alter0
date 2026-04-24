import type { ComponentPropsWithoutRef, ReactNode } from "react";
import {
  RuntimeAttachmentGallery,
  RuntimeMarkdownHTML,
  RuntimeMarkdownShell,
  type RuntimeAttachmentGalleryItem,
} from "./RuntimeTimelinePrimitives";

function joinClassNames(...values: Array<string | undefined>) {
  return values.filter(Boolean).join(" ");
}

export type RuntimeTimelineProcessStep = {
  id: string;
  itemClassName?: string;
  itemProps?: ComponentPropsWithoutRef<"article">;
  toggleable?: boolean;
  toggleClassName?: string;
  toggleProps?: Omit<ComponentPropsWithoutRef<"button">, "type" | "className" | "children" | "onClick">;
  title: string;
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
      html: string;
      copyValue?: string;
      copyLabel?: string;
      wrapperClassName?: string;
      wrapperProps?: ComponentPropsWithoutRef<"div">;
      bubbleClassName?: string;
      bubbleProps?: ComponentPropsWithoutRef<"div">;
      className?: string;
      toolbarClassName?: string;
      copyButtonClassName?: string;
      bodyClassName?: string;
    }
  | {
      type: "prompt";
      className?: string;
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
      steps: RuntimeTimelineProcessStep[];
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
  items: RuntimeTimelineItem[];
  overlay?: ReactNode;
};

export function RuntimeTimeline({
  className,
  timelineProps,
  emptyState,
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
        {items.length === 0 ? emptyState : items.map((item) => (
          <RuntimeTimelineArticle key={item.id} item={item} />
        ))}
      </div>
      {overlay}
    </>
  );
}

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
      return <RuntimeMarkdownHTML html={block.html} className={block.className} />;
    case "markdown-shell":
      const shell = (
        <RuntimeMarkdownShell
          html={block.html}
          copyValue={block.copyValue}
          copyLabel={block.copyLabel}
          className={block.className}
          toolbarClassName={block.toolbarClassName}
          copyButtonClassName={block.copyButtonClassName}
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
      return (
        <div className={block.className}>
          <div className={block.textClassName}>
            <span className="terminal-log-text">{block.text}</span>
          </div>
          {block.timeLabel ? <span className={block.timeClassName}>{block.timeLabel}</span> : null}
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
            {block.steps.length ? block.steps.map((step) => (
              <article key={step.id} className={step.itemClassName} {...step.itemProps}>
                {step.toggleable === false ? (
                  <div className={step.toggleClassName}>
                    {step.meta ? <span>{step.meta}</span> : null}
                    <span>{step.title}</span>
                  </div>
                ) : (
                  <button
                    className={step.toggleClassName}
                    type="button"
                    aria-expanded={step.expanded}
                    onClick={step.onToggle}
                    {...step.toggleProps}
                  >
                    <span>{step.title}</span>
                    {step.meta ? <span>{step.meta}</span> : null}
                  </button>
                )}
                <div className={step.bodyClassName} hidden={!step.expanded} {...step.bodyProps}>
                  {step.detail}
                </div>
              </article>
            )) : block.emptyState}
          </div>
        </section>
      );
  }
}
